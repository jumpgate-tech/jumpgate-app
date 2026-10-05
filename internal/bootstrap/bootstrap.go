// internal/bootstrap/bootstrap.go

// Package bootstrap installs and pairs a jumpgate agent over a privileged
// executor. Every step is idempotent, so re-running it repairs a half-paired
// box and pairs a second controller without disturbing the first.
package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"golang.org/x/crypto/ssh"
)

const (
	BinaryPath     = "/usr/local/lib/jumpgate/jumpgate"
	UnitPath       = "/etc/systemd/system/jumpgate-agent.service"
	DropInPath     = "/etc/ssh/sshd_config.d/50-jumpgate.conf"
	NodePath       = "/etc/jumpgate/node.json"
	HomeDir        = "/var/lib/jumpgate/home"
	AuthorizedKeys = HomeDir + "/.ssh/authorized_keys"
)

// Options configures one pairing.
type Options struct {
	Exec            executor.Executor // privileged: root, or executor.Sudo(...)
	Local           bool              // pairing the machine the controller runs on
	LocalUID        int               // with Local: the controller's uid (0 enrolls none)
	AgentBinary     func(arch string) (string, error)
	Controller      eip712.Address
	ControllerLabel string
	TransportKey    string // "ssh-ed25519 AAAA… comment"; empty with Local
	Wire            *catalog.WireConfig
	Event           func(step, line string)
}

// StepError names the step that failed and, once Run returns it, what the
// earlier steps left on the box, so the operator knows what a re-run will find.
type StepError struct {
	Step string
	Err  error
	// LeftInPlace is filled in by Run.
	LeftInPlace string
}

func (e *StepError) Error() string {
	msg := "pairing step " + e.Step + ": " + e.Err.Error()
	if e.LeftInPlace != "" {
		msg += "; left in place: " + e.LeftInPlace
	}
	return msg
}

// leftInPlace describes what the steps before step leave on the box when step
// fails. Every step is idempotent, so re-running hosts add resumes from here.
func leftInPlace(step string, local bool) string {
	binary := "the agent binary " + BinaryPath
	user := "the jumpgate user and group (home " + HomeDir + ")"
	sshd := "the sshd drop-in " + DropInPath + " and the restricted transport key in " + AuthorizedKeys
	identity := "the agent key and replay record in /var/lib/jumpgate"
	policy := "the controller's entry in /etc/jumpgate/policy.json and " + NodePath
	var have []string
	switch step {
	case "preflight":
		return "nothing; no changes were made on the box"
	case "upload":
		return "the box's previous agent binary, if any; at most a staged " + BinaryPath + ".new"
	case "user":
		have = []string{binary}
	case "sshd":
		return binary + ", " + user + "; the previous sshd drop-in is restored if `sshd -t` failed, and the transport key may already be in " + AuthorizedKeys
	case "identity":
		have = []string{binary, user}
		if !local {
			have = append(have, sshd)
		}
	case "policy":
		have = []string{binary, user}
		if !local {
			have = append(have, sshd)
		}
		have = append(have, identity)
	case "service":
		have = []string{binary, user}
		if !local {
			have = append(have, sshd)
		}
		have = append(have, identity, policy, "the unit "+UnitPath)
	default:
		return ""
	}
	return strings.Join(have, ", ") + "; re-running `jumpgate hosts add` resumes from here"
}
func (e *StepError) Unwrap() error { return e.Err }

type runner struct {
	ctx context.Context
	o   Options
}

func (r runner) emit(step, line string) {
	if r.o.Event != nil {
		r.o.Event(step, line)
	}
}

// sh runs cmd and turns a non-zero exit into an error carrying stderr's tail.
func (r runner) sh(step, cmd string) (string, error) {
	res, err := r.o.Exec.Run(r.ctx, cmd, nil)
	if err != nil {
		return "", &StepError{Step: step, Err: err}
	}
	if res.ExitCode != 0 {
		return res.Stdout, &StepError{Step: step, Err: fmt.Errorf("`%s` exited %d: %s", cmd, res.ExitCode, tail(res.Stderr))}
	}
	return res.Stdout, nil
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		s = "…" + s[len(s)-400:]
	}
	return s
}

// Run performs steps 1–7 of the spec and returns the agent's address. Step 8
// (a signed round trip) is the caller's: it holds the controller's key.
func Run(ctx context.Context, o Options) (eip712.Address, error) {
	addr, err := run(ctx, o)
	var se *StepError
	if errors.As(err, &se) {
		se.LeftInPlace = leftInPlace(se.Step, o.Local)
	}
	return addr, err
}

func run(ctx context.Context, o Options) (eip712.Address, error) {
	r := runner{ctx: ctx, o: o}
	arch, err := r.preflight()
	if err != nil {
		return eip712.Address{}, err
	}
	steps := []struct {
		name string
		fn   func() error
	}{
		{"upload", func() error { return r.upload(arch) }},
		{"user", r.user},
		{"sshd", r.sshd},
	}
	for _, s := range steps {
		r.emit(s.name, "start")
		if err := s.fn(); err != nil {
			return eip712.Address{}, err
		}
	}
	addr, err := r.identity()
	if err != nil {
		return eip712.Address{}, err
	}
	if err := r.policyAndNode(); err != nil {
		return eip712.Address{}, err
	}
	if err := r.service(); err != nil {
		return eip712.Address{}, err
	}
	return addr, nil
}

func (r runner) preflight() (string, error) {
	r.emit("preflight", "start")
	osName, err := r.sh("preflight", "uname -s")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(osName) != "Linux" {
		return "", &StepError{Step: "preflight", Err: fmt.Errorf("the agent runs on Linux; this box is %s", strings.TrimSpace(osName))}
	}
	if _, err := r.sh("preflight", "command -v systemctl >/dev/null && [ -d /run/systemd/system ]"); err != nil {
		return "", &StepError{Step: "preflight", Err: fmt.Errorf("systemd is required")}
	}
	if !r.o.Local {
		// Editing sshd_config itself risks locking the operator out; a drop-in
		// is only honoured when the main file includes the directory.
		if _, err := r.sh("preflight", `grep -Eq '^[[:space:]]*Include[[:space:]]+/etc/ssh/sshd_config\.d/\*\.conf' /etc/ssh/sshd_config`); err != nil {
			return "", &StepError{Step: "preflight", Err: fmt.Errorf("/etc/ssh/sshd_config does not Include /etc/ssh/sshd_config.d/*.conf; add that line first")}
		}
	}
	m, err := r.sh("preflight", "uname -m")
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(m) {
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	}
	return "", &StepError{Step: "preflight", Err: fmt.Errorf("unsupported architecture %q", strings.TrimSpace(m))}
}

func (r runner) upload(arch string) error {
	path, err := r.o.AgentBinary(arch)
	if err != nil {
		return &StepError{Step: "upload", Err: err}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return &StepError{Step: "upload", Err: err}
	}
	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])
	if res, err := r.o.Exec.Run(r.ctx, "sha256sum "+BinaryPath+" 2>/dev/null", nil); err == nil && res.ExitCode == 0 && strings.HasPrefix(res.Stdout, want) {
		r.emit("upload", "already installed")
		return nil
	}
	tmp := BinaryPath + ".new"
	if err := r.o.Exec.WriteFile(r.ctx, tmp, content, 0o755); err != nil {
		return &StepError{Step: "upload", Err: err}
	}
	_, err = r.sh("upload", fmt.Sprintf("echo %s'  '%s | sha256sum -c - && mv -f %s %s", want, tmp, tmp, BinaryPath))
	return err
}

func (r runner) user() error {
	// Older shadow-utils' --create-home does not create parent directories.
	_, err := r.sh("user", "install -d -m 0755 "+path.Dir(HomeDir)+"; getent group jumpgate >/dev/null || groupadd --system jumpgate; "+
		"id -u jumpgate >/dev/null 2>&1 || useradd --system --gid jumpgate --home-dir "+HomeDir+
		" --create-home --shell /usr/sbin/nologin jumpgate")
	return err
}

func (r runner) sshd() error {
	if r.o.Local {
		return nil
	}
	// Validate the key before touching the box: a malformed or multi-line key
	// must never reach authorized_keys.
	key, err := canonicalTransportKey(r.o.TransportKey)
	if err != nil {
		return &StepError{Step: "sshd", Err: err}
	}
	if err := r.dropIn(); err != nil {
		return err
	}
	if _, err := r.sh("sshd", "systemctl reload ssh 2>/dev/null || systemctl reload sshd"); err != nil {
		return err
	}
	return r.authorize(key)
}

// dropIn installs the sshd drop-in, validates it, and on failure puts back
// whatever was there before: deleting a previously good drop-in would strip
// the Match restrictions while authorized_keys still allows forwarding.
func (r runner) dropIn() error {
	backup := DropInPath + ".jumpgate-bak"
	if _, err := r.sh("sshd", fmt.Sprintf("if [ -e %s ]; then cp -p %s %s; else rm -f %s; fi", DropInPath, DropInPath, backup, backup)); err != nil {
		return err
	}
	if err := r.o.Exec.WriteFile(r.ctx, DropInPath, []byte(sshdDropIn), 0o644); err != nil {
		return &StepError{Step: "sshd", Err: err}
	}
	// sshd is usually not on PATH for `sudo -n sh -c`.
	if _, err := r.sh("sshd", "PATH=$PATH:/usr/sbin:/sbin sshd -t"); err != nil {
		_, _ = r.o.Exec.Run(r.ctx, fmt.Sprintf("if [ -e %s ]; then mv -f %s %s; else rm -f %s; fi", backup, backup, DropInPath, DropInPath), nil)
		return err
	}
	_, _ = r.o.Exec.Run(r.ctx, "rm -f "+backup, nil)
	return nil
}

// authorize makes authorized_keys carry exactly one line for the transport
// key, in the current form: an old line for the same key (matched on type and
// base64 alone, whatever its options, comment or line ending) is replaced in
// place, so a re-pair upgrades boxes paired with older options. Every other
// line is kept byte for byte. An already-current line is not rewritten, and
// the file is always left owned by the tunnel user so a re-pair repairs a
// half-finished earlier attempt.
func (r runner) authorize(key string) error {
	// Decide existence explicitly: a failed read must never be mistaken for
	// "no keys", or every other controller's key would be overwritten.
	res, err := r.o.Exec.Run(r.ctx, "test -e "+AuthorizedKeys, nil)
	if err != nil {
		return &StepError{Step: "sshd", Err: err}
	}
	var existing []byte
	switch res.ExitCode {
	case 0:
		existing, err = r.o.Exec.ReadFile(r.ctx, AuthorizedKeys)
		if err != nil {
			return &StepError{Step: "sshd", Err: fmt.Errorf("authorized_keys exists but cannot be read; refusing to overwrite it: %w", err)}
		}
	case 1:
	default:
		return &StepError{Step: "sshd", Err: fmt.Errorf("`test -e %s` exited %d: %s", AuthorizedKeys, res.ExitCode, tail(res.Stderr))}
	}
	if _, err := r.sh("sshd", "install -d -m 0700 -o jumpgate -g jumpgate "+HomeDir+"/.ssh"); err != nil {
		return err
	}
	merged, changed := mergeTransportKey(string(existing), key)
	if !changed {
		r.emit("sshd", "transport key unchanged")
		_, err := r.sh("sshd", "chown jumpgate:jumpgate "+AuthorizedKeys+" && chmod 0600 "+AuthorizedKeys)
		return err
	}
	// Stage, hand to the tunnel user, then rename: the final path is never
	// root-owned and is replaced atomically, so an interruption leaves the
	// previous file intact. A failed hand-over drops the staged copy.
	tmp := AuthorizedKeys + ".new"
	if err := r.o.Exec.WriteFile(r.ctx, tmp, []byte(merged), 0o600); err != nil {
		return &StepError{Step: "sshd", Err: err}
	}
	if _, err = r.sh("sshd", fmt.Sprintf("chown jumpgate:jumpgate %s && chmod 0600 %s && mv -f %s %s", tmp, tmp, tmp, AuthorizedKeys)); err != nil {
		_, _ = r.o.Exec.Run(r.ctx, "rm -f "+tmp, nil)
		return err
	}
	return nil
}

// mergeTransportKey returns authorizedKeys with key's line in the current form
// and whether that changed anything. The first line carrying key (type and
// base64 as adjacent fields) is replaced; later lines for the same key are
// dropped; with none, the line is appended. Other lines are untouched.
func mergeTransportKey(authorizedKeys, key string) (string, bool) {
	want := transportKeyOptions + " " + key
	f := strings.Fields(key)
	typ, blob := f[0], f[1]
	lines := strings.Split(authorizedKeys, "\n")
	out := make([]string, 0, len(lines)+1)
	found, changed := false, false
	for i, l := range lines {
		if !lineHasKey(l, typ, blob) {
			out = append(out, l)
			continue
		}
		if found {
			changed = true // a duplicate of our own key
			continue
		}
		found = true
		if l != want {
			changed = true
		}
		out = append(out, want)
		if i == len(lines)-1 {
			// Our line ended the file without a newline; give it one.
			out = append(out, "")
			changed = true
		}
	}
	if found {
		return strings.Join(out, "\n"), changed
	}
	merged := strings.TrimRight(authorizedKeys, "\n")
	if merged != "" {
		merged += "\n"
	}
	return merged + want + "\n", true
}

// lineHasKey reports whether an authorized_keys line is for the key typ blob:
// the two appear as adjacent whole fields. Options precede the type, so the
// position varies; a trailing \r or blanks are only whitespace to Fields.
func lineHasKey(line, typ, blob string) bool {
	f := strings.Fields(line)
	for i := 1; i < len(f); i++ {
		if f[i] == blob && f[i-1] == typ {
			return true
		}
	}
	return false
}

// canonicalTransportKey accepts exactly one bare public key and returns its
// canonical "type base64" form. Options, a second key, or a newline would
// otherwise add an unrestricted authorized_keys line.
func canonicalTransportKey(s string) (string, error) {
	pub, _, opts, rest, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(s)))
	if err != nil {
		return "", fmt.Errorf("transport key: %w", err)
	}
	if len(opts) > 0 {
		return "", fmt.Errorf("transport key must not carry options")
	}
	if len(strings.TrimSpace(string(rest))) > 0 {
		return "", fmt.Errorf("transport key must be a single line")
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub))), nil
}

func (r runner) identity() (eip712.Address, error) {
	r.emit("identity", "start")
	out, err := r.sh("identity", BinaryPath+" agent init")
	if err != nil {
		return eip712.Address{}, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	addr, err := eip712.ParseAddress(strings.TrimSpace(lines[len(lines)-1]))
	if err != nil {
		return eip712.Address{}, &StepError{Step: "identity", Err: fmt.Errorf("agent init printed %q: %w", out, err)}
	}
	r.emit("identity", "agent address "+addr.Hex())
	return addr, nil
}

func (r runner) policyAndNode() error {
	r.emit("policy", "start")
	cmd := fmt.Sprintf("%s agent enroll --address %s --tier routine --label %s", BinaryPath, r.o.Controller.Hex(), shQuote(r.o.ControllerLabel))
	// Root needs no grant: the peer gate admits uid 0 already, and listing
	// any local uid opens the socket's mode to every local user.
	if r.o.Local && r.o.LocalUID != 0 {
		cmd += " --local-uid " + strconv.Itoa(r.o.LocalUID)
	}
	if _, err := r.sh("policy", cmd); err != nil {
		return err
	}
	if r.o.Wire == nil {
		return nil
	}
	b, err := json.Marshal(r.o.Wire)
	if err != nil {
		return &StepError{Step: "policy", Err: err}
	}
	if err := r.o.Exec.WriteFile(r.ctx, NodePath, b, 0o600); err != nil {
		return &StepError{Step: "policy", Err: err}
	}
	return nil
}

func (r runner) service() error {
	r.emit("service", "start")
	if err := r.o.Exec.WriteFile(r.ctx, UnitPath, []byte(agentUnit), 0o644); err != nil {
		return &StepError{Step: "service", Err: err}
	}
	// restart, not just enable --now: a re-pair that uploaded a new binary
	// must replace the running one.
	if _, err := r.sh("service", "systemctl daemon-reload && systemctl enable --now jumpgate-agent.service && systemctl restart jumpgate-agent.service"); err != nil {
		return err
	}
	// systemd reports "active" as soon as it forks the agent, before the agent
	// has read its key, so a crash-looping agent would pass a one-shot
	// is-active. Wait (about 10s) for the unit to be active AND its socket to
	// exist; the restart removed the old socket with the runtime directory.
	// is-active exits 3 when not active, so its output is read regardless.
	res, err := r.o.Exec.Run(r.ctx, waitListening, nil)
	if err != nil {
		return &StepError{Step: "service", Err: err}
	}
	if state := strings.TrimSpace(res.Stdout); state != "listening" {
		msg := fmt.Sprintf("jumpgate-agent.service is %q and not listening on %s", state, agentclient.DefaultSocket)
		if j, err := r.o.Exec.Run(r.ctx, "journalctl -u jumpgate-agent.service -n 5 --no-pager -o cat", nil); err == nil && strings.TrimSpace(j.Stdout) != "" {
			msg += "; journal: " + tail(j.Stdout)
		}
		return &StepError{Step: "service", Err: fmt.Errorf("%s; see journalctl -u jumpgate-agent", msg)}
	}
	return nil
}

// waitListening prints "listening" once the agent is up, else the unit's last
// state.
const waitListening = `for i in $(seq 1 20); do s=$(systemctl is-active jumpgate-agent.service); ` +
	`if [ "$s" = active ] && [ -S ` + agentclient.DefaultSocket + ` ]; then echo listening; exit 0; fi; sleep 0.5; done; echo "$s"`

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'" }
