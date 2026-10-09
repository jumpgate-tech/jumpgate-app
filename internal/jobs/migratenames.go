package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// MigrateNamesVersion is the plan version node.migrate-names runs.
const MigrateNamesVersion = "node.migrate-names/1"

// How long stop-units waits for the account's processes to exit after each
// signal, and how often it looks. Variables so tests do not wait.
var (
	stopGrace = 30 * time.Second
	killWait  = 10 * time.Second
	pollEvery = time.Second
)

// Question keys and the options the steps add to the engine's retry and abort.
const (
	qBusyUser        = "busy-user"
	qMountedData     = "mounted-data"
	optStopProcesses = "stop-processes"
	optKillProcesses = "kill-processes"
	optKeepPath      = "keep-path"
)

// Every name, unit and path this plan puts on a command line comes from these
// two sets, from catalog's marker paths, or from node.json after readNode
// validated it. The job's inputs are empty and never reach a command.
var (
	legacy  = catalog.LegacyNames
	current = catalog.CurrentNames
	unitDir = path.Dir(catalog.UnitPath(legacy.ExecUnit()))
)

func init() {
	Register(Plan{Kind: intent.KindNodeMigrateNames, Version: MigrateNamesVersion,
		Title: "Rename the node's account, units and data to " + current.Prefix,
		Validate: func(in json.RawMessage) error {
			dec := json.NewDecoder(bytes.NewReader(in))
			dec.DisallowUnknownFields()
			var p intent.MigrateNamesPayload
			return dec.Decode(&p)
		},
		Steps: []StepDef{
			{ID: "preflight", Title: "Check what to rename", Run: mnPreflight,
				AbortNote: "nothing was changed"},
			{ID: "stop-units", Title: "Stop the node services", Run: mnStopUnits,
				AbortNote: "nothing was renamed, but the node services are stopped (and any other process of the account you chose to stop): start them with `jumpgate service HOST exec start`, then beacon"},
			{ID: "rename-account", Title: "Rename the service account", Run: mnRenameAccount,
				AbortNote: "the account may already be renamed while the units still name the old one, so the node cannot start; a new node.migrate-names job finishes the rename"},
			{ID: "move-data", Title: "Move the data root", Run: mnMoveData,
				AbortNote: "the account is renamed and the data root may already be at " + current.DataRoot + ", but the units still name the old account and path, so the node cannot start; a new node.migrate-names job finishes the move"},
			{ID: "rewrite-units", Title: "Rewrite the node units", Run: mnRewriteUnits,
				AbortNote: "the units, node.json and the markers may be half switched, so the node may not start; a new node.migrate-names job finishes them"},
			{ID: "ownership", Title: "Check data ownership", Run: mnOwnership,
				AbortNote: "the names are switched and the node services are stopped; some data may not be owned by " + current.User + " yet, so a client may fail to write. Run setup, which re-owns the data, then start the services"},
			{ID: "start-units", Title: "Start the node services", Run: mnStartUnits,
				AbortNote: "the migration is done, but the node services may not be running: start them with `jumpgate service HOST exec start`, then beacon"},
			{ID: "verify", Title: "Verify", Run: mnVerify,
				AbortNote: "the migration ran but did not verify; read this job's log before you change anything"},
		}})
}

type migratePreflight struct {
	WasActive   map[string]bool `json:"wasActive"` // "exec", "beacon": running when the job began
	AlreadyDone bool            `json:"alreadyDone"`
}

type migrateMove struct {
	Moved   bool   `json:"moved"`
	DataDir string `json:"dataDir"`
	JWTPath string `json:"jwtPath"`
}

// MigrateNamesResult is the job's result; the controller copies it into its
// own record of the box (Task 12).
type MigrateNamesResult struct {
	Names   string `json:"names"`
	DataDir string `json:"dataDir"`
	JWTPath string `json:"jwtPath"`
}

// readNode reads node.json as the agent does: through the names marker
// (D-J10), so a re-pair that wrote the controller's stale wire after a
// migration does not send the job back to work already done. Every path in
// it reaches a root command line, so it is validated here, before any step
// acts on it.
func readNode(env Env) (catalog.WireConfig, error) {
	var w catalog.WireConfig
	b, err := os.ReadFile(env.NodePath)
	if errors.Is(err, fs.ErrNotExist) {
		return w, errors.New("no node is set up on this box (no node.json); there is nothing to rename")
	}
	if err != nil {
		return w, err
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return w, fmt.Errorf("node.json does not parse: %w", err)
	}
	mk, ok, err := ReadNamesMarker(env.NamesPath)
	if err != nil {
		return w, err
	}
	if ok {
		mk.Apply(&w)
	}
	return w, checkWire(w)
}

func checkWire(w catalog.WireConfig) error {
	if err := catalog.ValidateNames(w); err != nil {
		return fmt.Errorf("node.json: %w", err)
	}
	if _, ok := catalog.NetworkByChainID(w.ChainID); !ok {
		return fmt.Errorf("node.json: unknown chain id %d", w.ChainID)
	}
	if err := catalog.ValidateDataDir(w.DataDir); err != nil {
		return fmt.Errorf("node.json: %w", err)
	}
	return checkJWT(jwtOf(w))
}

// checkJWT holds the jwt path to the data dir's rules: it is chowned and
// stat'ed as root, so it must be as clean, and as far from jumpgate's own
// trees, as the data dir.
func checkJWT(p string) error {
	if err := catalog.ValidateDataDir(p); err != nil {
		return fmt.Errorf("node.json: jwt path: %w", err)
	}
	return nil
}

// jwtOf is the jwt path the units use (RenderUnits' default when unset).
func jwtOf(w catalog.WireConfig) string {
	if w.JWTPath != "" {
		return w.JWTPath
	}
	return path.Join(w.DataDir, "jwt.hex")
}

func writeNode(env Env, w catalog.WireConfig) error {
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(env.NodePath, b)
}

// run runs cmd and copies it, and what it printed, to the journal: the job's
// output is the record of every command it ran as root.
func run(ctx context.Context, env Env, cmd string) (executor.Result, error) {
	res, err := env.Exec.Run(ctx, cmd, nil)
	if env.Out != nil {
		fmt.Fprintf(env.Out, "$ %s\n%s%s", cmd, res.Stdout, res.Stderr)
		if err == nil && res.ExitCode != 0 {
			fmt.Fprintf(env.Out, "(exit %d)\n", res.ExitCode)
		}
	}
	return res, err
}

// sh runs cmd and turns a non-zero exit into an error quoting the box.
func sh(ctx context.Context, env Env, cmd string) (executor.Result, error) {
	res, err := run(ctx, env, cmd)
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%s exited %d: %s", strings.Fields(cmd)[0], res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return res, nil
}

// probe runs a check whose exit code is the answer.
func probe(ctx context.Context, env Env, cmd string) (bool, error) {
	res, err := run(ctx, env, cmd)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

func userExists(ctx context.Context, env Env, u string) (bool, error) {
	return probe(ctx, env, "id -u "+u)
}

func groupExists(ctx context.Context, env Env, g string) (bool, error) {
	return probe(ctx, env, "getent group "+g)
}

// readMarker reads a marker in catalog.MarkersDir; ok is false when there is
// none. Present but unreadable is an error, never "absent".
func readMarker(ctx context.Context, env Env, p string) (string, bool, error) {
	there, err := probe(ctx, env, "test -e "+shQuote(p))
	if err != nil || !there {
		return "", false, err
	}
	res, err := sh(ctx, env, "cat "+shQuote(p))
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(res.Stdout), true, nil
}

func alreadyDone(env Env) bool {
	var pre migratePreflight
	_ = json.Unmarshal(env.Results["preflight"], &pre)
	return pre.AlreadyDone
}

func unitState(ctx context.Context, env Env, unit string) (string, error) {
	res, err := run(ctx, env, "systemctl is-active "+unit)
	return strings.TrimSpace(res.Stdout), err
}

// running is every is-active state in which the unit's processes may still
// be there.
func running(state string) bool {
	switch state {
	case "active", "activating", "deactivating", "reloading", "refreshing":
		return true
	}
	return false
}

// leftovers lists what still says the box uses the legacy names, for a w
// that already names the current ones. Empty means the migration finished.
func leftovers(ctx context.Context, env Env, w catalog.WireConfig) ([]string, error) {
	var left []string
	if ok, err := groupExists(ctx, env, legacy.Group); err != nil {
		return nil, err
	} else if ok {
		left = append(left, "group "+legacy.Group+" still exists")
	}
	for _, u := range legacy.Units() {
		if ok, err := probe(ctx, env, "test -e "+shQuote(catalog.UnitPath(u))); err != nil {
			return nil, err
		} else if ok {
			left = append(left, catalog.UnitPath(u)+" is still there")
		}
	}
	if v, ok, err := readMarker(ctx, env, catalog.OwnerMarkerPath(w)); err != nil {
		return nil, err
	} else if ok && v == legacy.User {
		left = append(left, "the owner marker still names "+legacy.User)
	}
	if old, moved := catalog.Rebase(w.DataDir, current.DataRoot, legacy.DataRoot); moved {
		if v, ok, err := readMarker(ctx, env, catalog.SnapshotMarkerPath(w)); err != nil {
			return nil, err
		} else if ok && v == old {
			left = append(left, "the snapshot marker still names "+old)
		}
	}
	return left, nil
}

// unitsMatching runs a grep -l over the unit directory and returns the files
// it names, less the node's own units under either name set.
func unitsMatching(ctx context.Context, env Env, grep string) ([]string, error) {
	res, err := run(ctx, env, grep)
	if err != nil {
		return nil, err
	}
	switch res.ExitCode {
	case 0:
	case 1:
		return nil, nil // grep found nothing
	default:
		return nil, fmt.Errorf("searching %s exited %d: %s", unitDir, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	own := map[string]bool{}
	for _, u := range append(legacy.Units(), current.Units()...) {
		own[catalog.UnitPath(u)] = true
	}
	var out []string
	for _, p := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		if p != "" && !own[p] {
			out = append(out, p)
		}
	}
	return out, nil
}

// roots reports whether the legacy data root and anything at the current one
// exist. A dangling symlink at the new root counts: mv -T would replace it.
func roots(ctx context.Context, env Env) (oldThere, newThere bool, err error) {
	if oldThere, err = probe(ctx, env, "test -d "+shQuote(legacy.DataRoot)); err != nil {
		return
	}
	newThere, err = probe(ctx, env, "test -e "+shQuote(current.DataRoot)+" || test -L "+shQuote(current.DataRoot))
	return
}

func mnPreflight(ctx context.Context, env Env) (Outcome, error) {
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	hasLegacy, err := userExists(ctx, env, legacy.User)
	if err != nil {
		return Outcome{}, err
	}
	hasCurrent, err := userExists(ctx, env, current.User)
	if err != nil {
		return Outcome{}, err
	}
	switch {
	case hasLegacy && hasCurrent:
		return Outcome{}, fmt.Errorf("both accounts exist (%s and %s); remove the one the node does not use, then retry", legacy.User, current.User)
	case !hasLegacy && !hasCurrent:
		return Outcome{}, fmt.Errorf("neither %s nor %s exists; set the node up first", legacy.User, current.User)
	case hasLegacy && w.Names == catalog.NamesCurrent:
		return Outcome{}, fmt.Errorf("node.json names the %s account, but only %s exists; run setup to repair the box before migrating it", current.User, legacy.User)
	case hasCurrent && w.Names == catalog.NamesCurrent:
		left, err := leftovers(ctx, env, w)
		if err != nil {
			return Outcome{}, err
		}
		if len(left) == 0 {
			b, _ := json.Marshal(migratePreflight{AlreadyDone: true})
			return Outcome{Result: b}, nil
		}
		// An earlier job stopped part way; this one finishes it.
		if env.Out != nil {
			fmt.Fprintf(env.Out, "an earlier migration did not finish: %s\n", strings.Join(left, "; "))
		}
	}
	// Any other unit running as the legacy account would break when it is
	// renamed (the systemd eRPC backend, for one). Q3: those move in
	// sub-project 6; until then, refuse before touching anything.
	others, err := unitsMatching(ctx, env, "grep -l -x -r -e "+shQuote("User="+legacy.User)+" -e "+shQuote("Group="+legacy.Group)+" "+shQuote(unitDir))
	if err != nil {
		return Outcome{}, err
	}
	if len(others) > 0 {
		return Outcome{}, fmt.Errorf("other units run as %s and would break if it were renamed: %s. Remove them (jumpgate renames gateways in a later release), then retry",
			legacy.User, strings.Join(others, ", "))
	}
	if catalog.UnderRoot(w.DataDir, legacy.DataRoot) {
		newDir, _ := catalog.Rebase(w.DataDir, legacy.DataRoot, current.DataRoot)
		if err := catalog.ValidateDataDir(newDir); err != nil {
			return Outcome{}, err
		}
		// Moving the root breaks any other unit that names a path in it,
		// whatever account it runs as.
		others, err := unitsMatching(ctx, env, "grep -l -F -r -e "+shQuote(legacy.DataRoot)+" "+shQuote(unitDir))
		if err != nil {
			return Outcome{}, err
		}
		if len(others) > 0 {
			return Outcome{}, fmt.Errorf("other units use paths under %s, which this job moves to %s: %s. Remove or repoint them, then retry",
				legacy.DataRoot, current.DataRoot, strings.Join(others, ", "))
		}
		oldThere, newThere, err := roots(ctx, env)
		if err != nil {
			return Outcome{}, err
		}
		if oldThere && newThere {
			return Outcome{}, fmt.Errorf("both %s and %s exist; move %s aside, then retry", legacy.DataRoot, current.DataRoot, current.DataRoot)
		}
	}
	pre := migratePreflight{WasActive: map[string]bool{}}
	for svc, units := range map[string][]string{
		"exec":   {legacy.ExecUnit(), current.ExecUnit()},
		"beacon": {legacy.BeaconUnit(), current.BeaconUnit()},
	} {
		for _, u := range units {
			st, err := unitState(ctx, env, u)
			if err != nil {
				return Outcome{}, err
			}
			pre.WasActive[svc] = pre.WasActive[svc] || st == "active"
		}
	}
	b, _ := json.Marshal(pre)
	return Outcome{Result: b}, nil
}

// procs is every process of user, one "pid command" line each.
func procs(ctx context.Context, env Env, user string) ([]string, error) {
	res, err := run(ctx, env, "pgrep -a -u "+user)
	if err != nil {
		return nil, err
	}
	switch res.ExitCode {
	case 0:
		return strings.Split(strings.TrimSpace(res.Stdout), "\n"), nil
	case 1:
		return nil, nil // pgrep matched nothing
	}
	return nil, fmt.Errorf("pgrep exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
}

// waitGone polls user's processes until there are none or within passes.
func waitGone(ctx context.Context, env Env, user string, within time.Duration) ([]string, error) {
	deadline := time.Now().Add(within)
	for {
		left, err := procs(ctx, env, user)
		if err != nil || len(left) == 0 || !time.Now().Before(deadline) {
			return left, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollEvery):
		}
	}
}

func signal(ctx context.Context, env Env, sig, user string) error {
	res, err := run(ctx, env, "pkill -"+sig+" -u "+user)
	if err != nil {
		return err
	}
	if res.ExitCode > 1 { // 1: nothing left to signal
		return fmt.Errorf("pkill exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

func firstLines(left []string) string {
	return strings.Join(left[:min(len(left), 10)], "\n")
}

func mnStopUnits(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	// Either name set may be absent (or half there, after an earlier job), so
	// a unit that does not exist is not an error: what matters is the check
	// below that none of them still runs.
	units := append(legacy.Units(), current.Units()...)
	for _, u := range units {
		if _, err := run(ctx, env, "systemctl stop "+u); err != nil {
			return Outcome{}, err
		}
	}
	for _, u := range units {
		st, err := unitState(ctx, env, u)
		if err != nil {
			return Outcome{}, err
		}
		if running(st) {
			return Outcome{}, fmt.Errorf("%s is still %s after systemctl stop", u, st)
		}
	}
	user := legacy.User
	if ok, err := userExists(ctx, env, user); err != nil {
		return Outcome{}, err
	} else if !ok {
		user = current.User
	}
	left, err := procs(ctx, env, user)
	if err != nil || len(left) == 0 {
		return Outcome{}, err
	}
	// Review Focus: the account's other processes are signalled only on an
	// answer, and SIGKILL only on a second one given after SIGTERM failed:
	// consenting to a clean stop is not consenting to a kill.
	if a := env.Answer; a != nil && a.Key == qBusyUser {
		switch a.Option {
		case optStopProcesses:
			if err := signal(ctx, env, "TERM", user); err != nil {
				return Outcome{}, err
			}
			if left, err = waitGone(ctx, env, user, stopGrace); err != nil || len(left) == 0 {
				return Outcome{}, err
			}
			return Outcome{Ask: &intent.JobQuestion{Key: qBusyUser, Tier: intent.TierRoutine,
				Prompt: fmt.Sprintf("%d process(es) of %s are still running %s after SIGTERM:\n%s", len(left), user, stopGrace, firstLines(left)),
				Options: []intent.JobOption{
					{ID: intent.OptRetry, Label: "Check again", Detail: "give them longer, or stop them yourself first"},
					{ID: optKillProcesses, Label: "Kill them", Detail: "SIGKILL: they get no chance to save their state"},
					{ID: intent.OptAbort, Label: "Stop the migration here", Detail: "nothing has been renamed; the node services stay stopped"},
				}}}, nil
		case optKillProcesses:
			if err := signal(ctx, env, "KILL", user); err != nil {
				return Outcome{}, err
			}
			if left, err = waitGone(ctx, env, user, killWait); err != nil || len(left) == 0 {
				return Outcome{}, err
			}
			return Outcome{}, fmt.Errorf("%d process(es) still run as %s after SIGKILL: %s", len(left), user, strings.Join(left, "; "))
		}
	}
	return Outcome{Ask: &intent.JobQuestion{Key: qBusyUser, Tier: intent.TierRoutine,
		Prompt: fmt.Sprintf("%d process(es) still run as %s after the node services stopped, so the account cannot be renamed:\n%s",
			len(left), user, firstLines(left)),
		Options: []intent.JobOption{
			{ID: intent.OptRetry, Label: "Check again", Detail: "stop them yourself first"},
			{ID: optStopProcesses, Label: "Stop them", Detail: fmt.Sprintf("SIGTERM, then wait up to %s; you are asked again before any SIGKILL", stopGrace)},
			{ID: intent.OptAbort, Label: "Stop the migration here", Detail: "nothing has been renamed; the node services stay stopped"},
		}}}, nil
}

func mnRenameAccount(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	hasLegacy, err := userExists(ctx, env, legacy.User)
	if err != nil {
		return Outcome{}, err
	}
	hasCurrent, err := userExists(ctx, env, current.User)
	if err != nil {
		return Outcome{}, err
	}
	// Preflight refused a box with both accounts. One that has both now got
	// the second from something other than this job (an operator, between a
	// failed attempt and its retry): renaming onto it would fail half-way.
	switch {
	case hasLegacy && hasCurrent:
		return Outcome{}, fmt.Errorf("both accounts exist (%s and %s); remove the one the node does not use, then retry", legacy.User, current.User)
	case !hasLegacy && !hasCurrent:
		return Outcome{}, fmt.Errorf("neither %s nor %s exists", legacy.User, current.User)
	}
	hasLegacyG, err := groupExists(ctx, env, legacy.Group)
	if err != nil {
		return Outcome{}, err
	}
	hasCurrentG, err := groupExists(ctx, env, current.Group)
	if err != nil {
		return Outcome{}, err
	}
	if hasLegacyG && hasCurrentG {
		return Outcome{}, fmt.Errorf("both groups exist (%s and %s); remove the one the node does not use, then retry", legacy.Group, current.Group)
	}
	// The user first: it is the call that fails while a process still runs as
	// the account, and then nothing at all has changed. usermod -l keeps the
	// uid, so every file the account owns stays its own.
	if hasLegacy {
		if _, err := sh(ctx, env, "usermod -l "+current.User+" -d "+current.DataRoot+" "+legacy.User); err != nil {
			return Outcome{}, err
		}
	}
	if hasLegacyG {
		if _, err := sh(ctx, env, "groupmod -n "+current.Group+" "+legacy.Group); err != nil {
			return Outcome{}, err
		}
	}
	if ok, err := userExists(ctx, env, current.User); err != nil {
		return Outcome{}, err
	} else if !ok {
		return Outcome{}, fmt.Errorf("%s does not exist after the rename", current.User)
	}
	// The units name Group= as well as User=; a group named otherwise would
	// stop them starting.
	if ok, err := groupExists(ctx, env, current.Group); err != nil {
		return Outcome{}, err
	} else if !ok {
		return Outcome{}, fmt.Errorf("group %s does not exist after the rename (the account's group was not %s); create it, then retry", current.Group, legacy.Group)
	}
	return Outcome{}, nil
}

func mnMoveData(ctx context.Context, env Env) (Outcome, error) {
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	result := func(mv migrateMove) (Outcome, error) {
		b, err := json.Marshal(mv)
		return Outcome{Result: b}, err
	}
	stay := migrateMove{DataDir: w.DataDir, JWTPath: jwtOf(w)}
	if alreadyDone(env) || !catalog.UnderRoot(w.DataDir, legacy.DataRoot) {
		return result(stay) // a custom data dir stays where it is (Q2)
	}
	moved := migrateMove{Moved: true, JWTPath: jwtOf(w)}
	moved.DataDir, _ = catalog.Rebase(w.DataDir, legacy.DataRoot, current.DataRoot)
	moved.JWTPath, _ = catalog.Rebase(moved.JWTPath, legacy.DataRoot, current.DataRoot)
	if err := catalog.ValidateDataDir(moved.DataDir); err != nil {
		return Outcome{}, err
	}
	if err := checkJWT(moved.JWTPath); err != nil {
		return Outcome{}, err
	}
	oldThere, newThere, err := roots(ctx, env)
	if err != nil {
		return Outcome{}, err
	}
	switch {
	case !oldThere && newThere:
		return result(moved) // an earlier attempt moved it
	case oldThere && newThere:
		return Outcome{}, fmt.Errorf("both %s and %s exist; move %s aside, then retry", legacy.DataRoot, current.DataRoot, current.DataRoot)
	case !oldThere:
		return Outcome{}, fmt.Errorf("%s does not exist, so %s cannot be there", legacy.DataRoot, w.DataDir)
	}
	if a := env.Answer; a != nil && a.Key == qMountedData && a.Option == optKeepPath {
		return result(stay)
	}
	// Review Focus 1: a mount at or under the old root stays mounted at the
	// old path after a reboot (fstab, mount units), so it is never moved.
	res, err := sh(ctx, env, "findmnt -rn -o TARGET")
	if err != nil {
		return Outcome{}, err
	}
	var mounts []string
	for _, m := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		if catalog.UnderRoot(m, legacy.DataRoot) {
			mounts = append(mounts, m)
		}
	}
	if len(mounts) > 0 {
		return Outcome{Ask: &intent.JobQuestion{Key: qMountedData, Tier: intent.TierRoutine,
			Prompt: fmt.Sprintf("%s is, or holds, a mount point (%s). Moving it would leave /etc/fstab or its mount unit pointing at the old path.",
				legacy.DataRoot, strings.Join(mounts, ", ")),
			Options: []intent.JobOption{
				{ID: optKeepPath, Label: "Keep the data where it is", Detail: "the node runs as " + current.User + " with its data still at " + w.DataDir},
				{ID: intent.OptRetry, Label: "Check again", Detail: "after you move the mount yourself"},
			}}}, nil
	}
	// Both roots sit in /var/lib, and no mount lies at or under the old one,
	// so this is a rename(2) on one filesystem: instant, nothing copied.
	if _, err := sh(ctx, env, "mv -T "+shQuote(legacy.DataRoot)+" "+shQuote(current.DataRoot)); err != nil {
		return Outcome{}, err
	}
	return result(moved)
}

func mnRewriteUnits(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	var mv migrateMove
	if err := json.Unmarshal(env.Results["move-data"], &mv); err != nil {
		return Outcome{}, fmt.Errorf("move-data left no result: %w", err)
	}
	// The move's result must describe this node.json: its paths as they are
	// (kept, or already switched by an earlier attempt) or as move-data
	// moved them. Anything else is not this job's, and is not rendered.
	if !sameOrMoved(mv.DataDir, w.DataDir) || !sameOrMoved(mv.JWTPath, jwtOf(w)) {
		return Outcome{}, fmt.Errorf("move-data's result (%s, %s) does not match node.json (%s, %s)", mv.DataDir, mv.JWTPath, w.DataDir, jwtOf(w))
	}
	nw := w
	nw.Names, nw.DataDir, nw.JWTPath = catalog.NamesCurrent, mv.DataDir, mv.JWTPath
	if err := checkWire(nw); err != nil {
		return Outcome{}, err
	}
	execUnit, beaconUnit, err := catalog.RenderUnits(nw)
	if err != nil {
		return Outcome{}, err
	}
	if err := env.Exec.WriteFile(ctx, catalog.UnitPath(current.ExecUnit()), []byte(execUnit), 0o644); err != nil {
		return Outcome{}, err
	}
	if err := env.Exec.WriteFile(ctx, catalog.UnitPath(current.BeaconUnit()), []byte(beaconUnit), 0o644); err != nil {
		return Outcome{}, err
	}
	// Exit unchecked: the legacy units are gone after an earlier attempt.
	if _, err := run(ctx, env, "systemctl disable "+strings.Join(legacy.Units(), " ")+" 2>/dev/null"); err != nil {
		return Outcome{}, err
	}
	if _, err := sh(ctx, env, "rm -f "+shQuote(catalog.UnitPath(legacy.ExecUnit()))+" "+shQuote(catalog.UnitPath(legacy.BeaconUnit()))); err != nil {
		return Outcome{}, err
	}
	if _, err := sh(ctx, env, "systemctl daemon-reload && systemctl enable "+strings.Join(current.Units(), " ")); err != nil {
		return Outcome{}, err
	}
	// The snapshot marker, node.json and names.json switch together, after
	// the units, so from here on every reader names the new units and dir.
	if err := followSnapshotMarker(ctx, env, nw, mv.Moved); err != nil {
		return Outcome{}, err
	}
	if err := writeNode(env, nw); err != nil {
		return Outcome{}, err
	}
	mk := NamesMarker{Names: catalog.NamesCurrent}
	if mv.Moved {
		mk.DataRootFrom, mk.DataRootTo = legacy.DataRoot, current.DataRoot
	}
	return Outcome{}, writeNamesMarker(env.NamesPath, mk)
}

// sameOrMoved reports whether p is q, or q as move-data moves it.
func sameOrMoved(p, q string) bool {
	m, _ := catalog.Rebase(q, legacy.DataRoot, current.DataRoot)
	return p == q || p == m
}

// followSnapshotMarker points the snapshot marker at the data dir's new home.
// The marker names the dir the restore populated, and setup's Verify refuses
// one that names another dir; the data moved by rename, so the restore it
// vouches for moved with it. A marker naming any dir but the old one vouches
// for something this job did not move, and is left alone; so is a missing one.
func followSnapshotMarker(ctx context.Context, env Env, nw catalog.WireConfig, moved bool) error {
	if !moved {
		return nil // the data stayed: the marker already names its dir
	}
	old, _ := catalog.Rebase(nw.DataDir, current.DataRoot, legacy.DataRoot)
	v, ok, err := readMarker(ctx, env, catalog.SnapshotMarkerPath(nw))
	if err != nil || !ok || v != old {
		return err
	}
	_, err = sh(ctx, env, catalog.SnapshotMarkerCommand(nw))
	return err
}

func mnOwnership(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	if w.Names != catalog.NamesCurrent {
		return Outcome{}, errors.New("node.json still names the legacy account; rewrite-units has not finished")
	}
	owner := func(p string) (string, error) {
		res, err := sh(ctx, env, "stat -c %U "+shQuote(p))
		return strings.TrimSpace(res.Stdout), err
	}
	dirOwner, err := owner(w.DataDir)
	if err != nil {
		return Outcome{}, err
	}
	jwtOwner, err := owner(jwtOf(w))
	if err != nil {
		return Outcome{}, err
	}
	// usermod -l keeps the uid, so files the old account owned now show the
	// new name: nothing to walk. Only a tree owned by a name the uid does not
	// follow (a mismatched NFS or idmapped mount) needs chown -R (spec). The
	// node is stopped, so nothing writes behind the walk; it runs for as long
	// as it takes, and a cancelled step kills it and fails, to be retried.
	if dirOwner != current.User || jwtOwner != current.User {
		if _, err := sh(ctx, env, fmt.Sprintf("chown -R %s:%s %s %s", current.User, current.Group, shQuote(w.DataDir), shQuote(jwtOf(w)))); err != nil {
			return Outcome{}, err
		}
		_, err := sh(ctx, env, catalog.OwnerMarkerCommand(w))
		return Outcome{}, err
	}
	// The owner marker vouched that chown -R finished for the legacy account;
	// that account is this one, uid unchanged, so the walk holds for the new
	// name. No marker, or one naming another account, vouches for nothing
	// here: it is left for setup's own ownership walk, never written blind.
	v, ok, err := readMarker(ctx, env, catalog.OwnerMarkerPath(w))
	if err != nil {
		return Outcome{}, err
	}
	if ok && v == legacy.User {
		_, err = sh(ctx, env, catalog.OwnerMarkerCommand(w))
	}
	return Outcome{}, err
}

func mnStartUnits(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	var pre migratePreflight
	_ = json.Unmarshal(env.Results["preflight"], &pre)
	for _, s := range []struct{ svc, unit string }{{"exec", current.ExecUnit()}, {"beacon", current.BeaconUnit()}} {
		if !pre.WasActive[s.svc] {
			continue // it was stopped before the job: leave it so
		}
		if _, err := sh(ctx, env, "systemctl start "+s.unit); err != nil {
			return Outcome{}, err
		}
		if st, err := unitState(ctx, env, s.unit); err != nil {
			return Outcome{}, err
		} else if st != "active" {
			return Outcome{}, fmt.Errorf("%s is %s after it was started; read `journalctl -u %s`", s.unit, st, s.unit)
		}
	}
	return Outcome{}, nil
}

func mnVerify(ctx context.Context, env Env) (Outcome, error) {
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	var problems []string
	for _, c := range []struct {
		cmd     string
		want    bool
		problem string
	}{
		{"id -u " + current.User, true, current.User + " does not exist"},
		{"id -u " + legacy.User, false, legacy.User + " still exists"},
		{"getent group " + current.Group, true, "group " + current.Group + " does not exist"},
		// One unit per call: is-enabled exits 0 when any one it names is.
		{"systemctl is-enabled " + current.ExecUnit(), true, current.ExecUnit() + " is not enabled"},
		{"systemctl is-enabled " + current.BeaconUnit(), true, current.BeaconUnit() + " is not enabled"},
	} {
		ok, err := probe(ctx, env, c.cmd)
		if err != nil {
			return Outcome{}, err
		}
		if ok != c.want {
			problems = append(problems, c.problem)
		}
	}
	left, err := leftovers(ctx, env, w)
	if err != nil {
		return Outcome{}, err
	}
	problems = append(problems, left...)
	if w.Names != catalog.NamesCurrent {
		problems = append(problems, "node.json still names the legacy account")
	}
	if len(problems) > 0 {
		return Outcome{}, errors.New(strings.Join(problems, "; "))
	}
	b, err := json.Marshal(MigrateNamesResult{Names: w.Names, DataDir: w.DataDir, JWTPath: jwtOf(w)})
	return Outcome{Result: b}, err
}
