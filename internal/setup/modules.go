package setup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/monitor"
)

// gateBinary is where pairing installs the box's gate. A box module with no
// client of its own (the health watch) runs this binary.
const gateBinary = "/usr/local/lib/jumpgate/jumpgate"

// Package vars so tests can stand in a registry; the catalog registers no
// real modules until each module's own task lands.
var (
	moduleByID         = catalog.ModuleByID
	renderBoxModule    = catalog.RenderBoxModule
	validateBoxModules = catalog.ValidateBoxModules

	// probeBoxModule reports whether a box module is up, through the same
	// probe status uses. A reading the shell could not take is an error.
	probeBoxModule = func(ctx context.Context, e executor.Executor, b catalog.BoxModule) (bool, error) {
		st := monitor.ProbeBoxModule(ctx, e, b)
		if !st.Active && st.Error != "" {
			return false, errors.New(st.Error)
		}
		return st.Active, nil
	}
)

// moduleHelp is one enabled node module's binary probe: Flag must appear in
// `<bin> node --help`, and Support names the feature in the D9 error
// ("message board").
type moduleHelp struct {
	Flag    string
	Support string
}

// nodeModuleHelp collects the binary probes of w's enabled node modules.
func nodeModuleHelp(w catalog.WireConfig) []moduleHelp {
	var out []moduleHelp
	for _, nm := range w.Modules {
		m, ok := moduleByID(nm.ID)
		if !ok || m.Probe.HelpContains == "" {
			continue
		}
		out = append(out, moduleHelp{Flag: m.Probe.HelpContains, Support: strings.ToLower(m.Name)})
	}
	return out
}

// missingHelpFlag runs `<dest> node --help` once and returns the first probe
// whose flag it does not list, or nil. With no probes it runs nothing, so a
// box with no module on reads no help at all.
func missingHelpFlag(ctx context.Context, e executor.Executor, dest string, probes []moduleHelp) (*moduleHelp, error) {
	if len(probes) == 0 {
		return nil, nil
	}
	res, err := e.Run(ctx, shQuote(dest)+" node --help 2>&1", nil)
	if err != nil {
		return nil, err
	}
	for i := range probes {
		if !strings.Contains(res.Stdout, probes[i].Flag) {
			return &probes[i], nil
		}
	}
	return nil, nil
}

// checkHelpAfterInstall fails with the D9 text when the binary just built
// still lacks a module's flag. Install runs before wire, so the unit is
// never rewritten for a binary that would reject it.
func checkHelpAfterInstall(ctx context.Context, e executor.Executor, client catalog.Client, dest string, probes []moduleHelp) error {
	missing, err := missingHelpFlag(ctx, e, dest, probes)
	if err != nil {
		return fmt.Errorf("install: probe %s: %w", client.ID, err)
	}
	if missing == nil {
		return nil
	}
	prefix := missing.Flag
	if i := strings.Index(prefix, "."); i >= 0 {
		prefix = prefix[:i]
	}
	repo := strings.TrimPrefix(strings.TrimPrefix(client.Repo, "https://"), "github.com/")
	return fmt.Errorf("%s at %s has no %s support (`%s node --help` lists no `%s.*` flags), even after a rebuild from %s main.",
		client.ID, dest, missing.Support, client.ID, prefix, repo)
}

// ---------------------------------------------------------------------
// modules: the live probe (D9)
// ---------------------------------------------------------------------

// modulesStep checks, after handshake, that every enabled node module's
// StatusMethod answers. It is a no-op without node modules. Verify is the
// same probe, so a re-run is cheap.
//
// docs:covers msgboard-troubleshoot
func modulesStep() Step {
	return Step{
		ID:    "modules",
		Title: "Check node modules",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			opts := streamOpts(ctx, st, "modules")
			deadline := time.Now().Add(handshakeTimeout)
			for {
				err, final := modulesCheck(ctx, e, st.Wire, opts)
				if err == nil || final {
					return err
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("modules: timed out after %s: %w", handshakeTimeout, err)
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(handshakePollInterval):
				}
			}
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			err, _ := modulesCheck(ctx, e, st.Wire, nil)
			return err
		},
	}
}

// modulesCheck probes each module once. final is true for an answer that
// waiting will not change (method not found).
func modulesCheck(ctx context.Context, e executor.Executor, w catalog.WireConfig, opts *executor.RunOpts) (err error, final bool) {
	for _, nm := range w.Modules {
		m, ok := moduleByID(nm.ID)
		if !ok || m.Probe.StatusMethod == "" {
			continue
		}
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":%q,"params":[]}`, m.Probe.StatusMethod)
		res, err := e.Run(ctx, "curl -s -X POST -H 'Content-Type: application/json' --data "+shQuote(body)+
			" http://"+w.RPCBind()+":"+strconv.Itoa(w.ExecHTTP()), opts)
		if err != nil {
			return fmt.Errorf("modules: %s probe: %w", m.ID, err), false
		}
		var reply struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code int `json:"code"`
			} `json:"error"`
		}
		if res.ExitCode != 0 || json.Unmarshal([]byte(res.Stdout), &reply) != nil {
			return fmt.Errorf("modules: %s did not answer %s yet", w.ExecID, m.Probe.StatusMethod), false
		}
		if reply.Error != nil && reply.Error.Code == -32601 {
			msg := fmt.Sprintf("%s is running but not serving %s. The unit's --http.api may have been edited.", w.ExecID, m.ID)
			if line := unknownModuleLine(ctx, e, w); line != "" {
				msg += "\n" + line
			}
			return fmt.Errorf("%s", msg), true
		}
		if reply.Error != nil || len(reply.Result) == 0 {
			return fmt.Errorf("modules: %s returned no result for %s yet", w.ExecID, m.Probe.StatusMethod), false
		}
	}
	return nil, false
}

// unknownModuleLine is the exec journal's last "Unknown RPC module" line, if
// any: the binary's own words for a namespace it refused.
func unknownModuleLine(ctx context.Context, e executor.Executor, w catalog.WireConfig) string {
	res, err := e.Run(ctx, "journalctl -u "+shQuote(w.NodeNames().ExecUnit())+" --since -10m --no-pager", nil)
	if err != nil {
		return ""
	}
	found := ""
	for _, line := range strings.Split(res.Stdout, "\n") {
		if strings.Contains(line, "Unknown RPC module") {
			found = strings.TrimSpace(line)
		}
	}
	return found
}

// ---------------------------------------------------------------------
// PlanBoxModule
// ---------------------------------------------------------------------

// PlanBoxModule installs (enable) or removes (disable) one box module on a
// box. w is the box's node, or nil when it has none; it is only used to
// check ports. Disabling keeps the module's data dir.
//
// docs:covers box-modules
func PlanBoxModule(b catalog.BoxModule, w *catalog.WireConfig, enable bool) ([]Step, error) {
	p := &boxPlan{b: b}
	// Even a disable builds commands from b: its id must be a registered box
	// module (a chain it no longer applies to may still be turned off).
	m, ok := moduleByID(b.ID)
	if !ok || m.Placement != catalog.PlaceBox {
		return nil, fmt.Errorf("setup: %s is not a known box module", b.ID)
	}
	if !enable {
		return []Step{p.stopStep(), p.removeStep()}, nil
	}
	u, err := renderBoxModule(b)
	if err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	if err := validateBoxModules([]catalog.BoxModule{b}, w); err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	p.m, p.u = m, u

	toolchain := Step{ID: "toolchain", Title: "Toolchains (none needed)",
		Verify: func(context.Context, executor.Executor, *State) error { return nil }}
	install := p.gateStep()
	if u.Client != "" {
		client, ok := catalog.ClientByID(u.Client)
		if !ok {
			return nil, fmt.Errorf("setup: %s needs client %q, which the catalog does not know", b.ID, u.Client)
		}
		toolchain = toolchainStep(neededToolchains(client))
		install = installStep("install", "Install "+client.ID, client)
	}
	return []Step{
		p.preflightStep(),
		currentAccountStep(),
		toolchain,
		install,
		p.writeFilesStep(),
		p.unitStep(),
		p.startStep(),
		p.probeStep(),
	}, nil
}

type boxPlan struct {
	b           catalog.BoxModule
	m           catalog.Module
	u           catalog.BoxUnit
	unitChanged bool // set by unit, read by start: a rewritten unit needs a restart
}

func (p *boxPlan) unitPath() string { return catalog.UnitPath(p.b.UnitName()) }

// currentAccountStep is accountStep for the CurrentNames account, whatever
// names the box's node (if any) uses: box modules always run as
// jumpgate-node (D-B).
func currentAccountStep() Step {
	s := accountStep(catalog.CurrentNames)
	withCurrent := func(f func(context.Context, executor.Executor, *State) error) func(context.Context, executor.Executor, *State) error {
		return func(ctx context.Context, e executor.Executor, st *State) error {
			cp := *st
			cp.Wire = catalog.WireConfig{Names: catalog.CurrentNames.ID}
			return f(ctx, e, &cp)
		}
	}
	s.Run, s.Verify = withCurrent(s.Run), withCurrent(s.Verify)
	return s
}

func (p *boxPlan) preflightStep() Step {
	return Step{
		ID:    "preflight",
		Title: "Preflight checks",
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			res, err := e.Run(ctx, "uname", nil)
			if err != nil {
				return fmt.Errorf("preflight: uname: %w", err)
			}
			if osName := strings.TrimSpace(res.Stdout); osName != "Linux" {
				return fmt.Errorf("preflight: box modules run on a Linux host, but this one reports %q", osName)
			}
			res, err = e.Run(ctx, "id -u", nil)
			if err != nil {
				return fmt.Errorf("preflight: id -u: %w", err)
			}
			if uid := strings.TrimSpace(res.Stdout); uid != "0" {
				return fmt.Errorf("preflight: installing a module needs root on the target; id -u reported %q", uid)
			}
			if len(p.u.Ports) == 0 {
				return nil
			}
			// The module's own running unit holds its ports on a re-run.
			if up, err := probeBoxModule(ctx, e, p.b); err == nil && up {
				return nil
			}
			res, err = e.Run(ctx, "ss -lntu", nil)
			if err != nil {
				return fmt.Errorf("preflight: ss: %w", err)
			}
			for _, pu := range p.u.Ports {
				if strings.Contains(res.Stdout, ":"+strconv.Itoa(pu.Port)+" ") {
					return fmt.Errorf("preflight: port %d/%s is already in use (from `ss -lntu`)", pu.Port, pu.Proto)
				}
			}
			return nil
		},
	}
}

// gateStep is install for a module that runs the box's gate binary: nothing
// to install, only a check that pairing put the gate there.
func (p *boxPlan) gateStep() Step {
	return Step{
		ID:    "install",
		Title: "Check the box's gate",
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			res, err := e.Run(ctx, "test -x "+gateBinary, nil)
			if err != nil {
				return fmt.Errorf("install: probe the box's gate: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("The %s runs the box's gate. Pair this box first.", strings.ToLower(p.m.Name))
			}
			return nil
		},
	}
}

func (p *boxPlan) sortedFiles() []string {
	out := make([]string, 0, len(p.u.Files))
	for f := range p.u.Files {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

func (p *boxPlan) writeFilesStep() Step {
	n := catalog.CurrentNames
	return Step{
		ID:    "write-files",
		Title: "Write " + p.b.Instance() + " files",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			opts := streamOpts(ctx, st, "write-files")
			data, conf := p.b.DataDir(), p.b.ConfigDir()
			// The data dir is the service's own: root creates the leaf and
			// hands it over, and never operates inside it after. Every file
			// setup writes goes in the root-owned config dir, outside any
			// path the service can write; each is staged under a random
			// name in a root-only dir, given its owner and mode there, then
			// renamed into place. The symlink refusals are defence in depth.
			res, err := e.Run(ctx, fmt.Sprintf("%s && mkdir -p %s && chown -h %s:%s %s && install -d -o root -g root -m 0755 %s %s && install -d -o root -g root -m 0700 %s",
				refuseSymlinks(n.DataRoot+"/modules", data, catalog.BoxModulesConfigRoot, conf, stagingDir),
				shQuote(data), n.User, n.Group, shQuote(data),
				shQuote(catalog.BoxModulesConfigRoot), shQuote(conf), shQuote(stagingDir)), opts)
			if err != nil {
				return fmt.Errorf("write-files: mkdir %s: %w", conf, err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("write-files: mkdir %s failed (exit %d): %s", conf, res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			for _, f := range p.sortedFiles() {
				if path.Dir(f) != conf {
					return fmt.Errorf("write-files: %s is not in %s", f, conf)
				}
				staged, err := stagingPath()
				if err != nil {
					return fmt.Errorf("write-files: %w", err)
				}
				if err := e.WriteFile(ctx, staged, []byte(p.u.Files[f]), 0600); err != nil {
					return fmt.Errorf("write-files: stage %s: %w", f, err)
				}
				// module.json is world-readable so any prober can list the
				// box's modules; config the unit reads is readable by the
				// service's group only.
				group, mode := n.Group, "0640"
				if f == p.b.SidecarPath() {
					group, mode = "root", "0644"
				}
				q := shQuote(staged)
				cmd := fmt.Sprintf("%s && chown root:%s %s && chmod %s %s && mv -T %s %s; rc=$?; rm -f %s; exit $rc",
					refuseSymlinks(f), group, q, mode, q, q, shQuote(f), q)
				res, err := e.Run(ctx, cmd, opts)
				if err != nil {
					return fmt.Errorf("write-files: install %s: %w", f, err)
				}
				if res.ExitCode != 0 {
					return fmt.Errorf("write-files: install %s failed (exit %d): %s", f, res.ExitCode, strings.TrimSpace(res.Stderr))
				}
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			for _, f := range p.sortedFiles() {
				got, err := e.ReadFile(ctx, f)
				if err != nil || string(got) != p.u.Files[f] {
					return fmt.Errorf("write-files: %s is not written yet", f)
				}
			}
			return nil
		},
	}
}

// stagingDir is root-only (0700): files bound for a service-owned dir wait
// here under a random name, out of the service account's reach.
const stagingDir = "/var/lib/jumpgate/staging"

func stagingPath() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("staging name: %w", err)
	}
	return stagingDir + "/" + hex.EncodeToString(b[:]), nil
}

// refuseSymlinks is a shell guard that fails when any of paths is a symlink.
func refuseSymlinks(paths ...string) string {
	var tests []string
	for _, p := range paths {
		tests = append(tests, "[ -L "+shQuote(p)+" ]")
	}
	return fmt.Sprintf(`if %s; then echo "refusing: a module path is a symlink" >&2; false; fi`, strings.Join(tests, " || "))
}

func (p *boxPlan) unitStep() Step {
	return Step{
		ID:    "unit",
		Title: "Write " + p.b.UnitName(),
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			if old, err := e.ReadFile(ctx, p.unitPath()); err == nil && string(old) != p.u.Unit {
				p.unitChanged = true
			}
			if err := e.WriteFile(ctx, p.unitPath(), []byte(p.u.Unit), 0644); err != nil {
				return fmt.Errorf("unit: write %s: %w", p.unitPath(), err)
			}
			res, err := e.Run(ctx, "systemctl daemon-reload", streamOpts(ctx, st, "unit"))
			if err != nil {
				return fmt.Errorf("unit: daemon-reload: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("unit: daemon-reload failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			got, err := e.ReadFile(ctx, p.unitPath())
			if err != nil || string(got) != p.u.Unit {
				return fmt.Errorf("unit: %s does not match the module's config yet", p.unitPath())
			}
			return nil
		},
	}
}

func (p *boxPlan) startStep() Step {
	unit := shQuote(p.b.UnitName())
	return Step{
		ID:    "start",
		Title: "Start " + unit,
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			cmd := "systemctl enable --now " + unit
			if p.unitChanged {
				cmd += " && systemctl restart " + unit
			}
			res, err := e.Run(ctx, cmd, streamOpts(ctx, st, "start"))
			if err != nil {
				return fmt.Errorf("start: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("start: %s failed (exit %d): %s", cmd, res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			p.unitChanged = false
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			if p.unitChanged {
				return fmt.Errorf("start: %s changed and needs a restart", unit)
			}
			res, err := e.Run(ctx, "systemctl is-enabled "+unit, nil)
			if err != nil {
				return fmt.Errorf("start: verify: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("start: %s is not enabled yet", unit)
			}
			res, err = e.Run(ctx, "systemctl is-active "+unit, nil)
			if err != nil {
				return fmt.Errorf("start: verify: %w", err)
			}
			if strings.TrimSpace(res.Stdout) != "active" {
				return fmt.Errorf("start: %s is not active yet", unit)
			}
			return nil
		},
	}
}

func (p *boxPlan) probeStep() Step {
	check := func(ctx context.Context, e executor.Executor) error {
		up, err := probeBoxModule(ctx, e, p.b)
		if err != nil {
			return fmt.Errorf("probe: %s: %w", p.b.Instance(), err)
		}
		if !up {
			return fmt.Errorf("probe: %s is not active", p.b.Instance())
		}
		return nil
	}
	return Step{
		ID:    "probe",
		Title: "Check " + p.b.Instance(),
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			deadline := time.Now().Add(handshakeTimeout)
			for {
				err := check(ctx, e)
				if err == nil {
					return nil
				}
				if time.Now().After(deadline) {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(handshakePollInterval):
				}
			}
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error { return check(ctx, e) },
	}
}

// unitPresent reports whether the module's unit file is on the box.
func (p *boxPlan) unitPresent(ctx context.Context, e executor.Executor) (bool, error) {
	res, err := e.Run(ctx, "test -f "+shQuote(p.unitPath()), nil)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

func (p *boxPlan) stopStep() Step {
	unit := shQuote(p.b.UnitName())
	return Step{
		ID:    "stop",
		Title: "Stop " + unit,
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			res, err := e.Run(ctx, "systemctl disable --now "+unit, streamOpts(ctx, st, "stop"))
			if err != nil {
				return fmt.Errorf("stop: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("stop: systemctl disable --now %s failed (exit %d): %s", unit, res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			present, err := p.unitPresent(ctx, e)
			if err != nil {
				return fmt.Errorf("stop: verify: %w", err)
			}
			if !present {
				return nil
			}
			res, err := e.Run(ctx, "systemctl is-active "+unit, nil)
			if err != nil {
				return fmt.Errorf("stop: verify: %w", err)
			}
			if strings.TrimSpace(res.Stdout) == "active" {
				return fmt.Errorf("stop: %s is still running", unit)
			}
			res, err = e.Run(ctx, "systemctl is-enabled "+unit, nil)
			if err != nil {
				return fmt.Errorf("stop: verify: %w", err)
			}
			if res.ExitCode == 0 {
				return fmt.Errorf("stop: %s is still enabled", unit)
			}
			return nil
		},
	}
}

func (p *boxPlan) removeStep() Step {
	return Step{
		ID:    "remove",
		Title: "Remove " + p.b.UnitName() + " (data kept)",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			// The config dir goes too: its module.json would keep the module
			// listed, and its config.json holds the webhook token.
			conf := p.b.ConfigDir()
			cmd := "rm -f " + shQuote(p.unitPath()) + " && systemctl daemon-reload && " +
				refuseSymlinks(conf) + " && rm -rf -- " + shQuote(conf)
			res, err := e.Run(ctx, cmd, streamOpts(ctx, st, "remove"))
			if err != nil {
				return fmt.Errorf("remove: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("remove: %s failed (exit %d): %s", cmd, res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			present, err := p.unitPresent(ctx, e)
			if err != nil {
				return fmt.Errorf("remove: verify: %w", err)
			}
			if present {
				return fmt.Errorf("remove: %s is still on the box", p.unitPath())
			}
			res, err := e.Run(ctx, "test -e "+shQuote(p.b.SidecarPath()), nil)
			if err != nil {
				return fmt.Errorf("remove: verify: %w", err)
			}
			if res.ExitCode == 0 {
				return fmt.Errorf("remove: %s is still on the box", p.b.SidecarPath())
			}
			return nil
		},
	}
}
