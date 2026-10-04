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
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
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
	LocalUID        int               // with Local: the controller's uid
	AgentBinary     func(arch string) (string, error)
	Controller      eip712.Address
	ControllerLabel string
	TransportKey    string // "ssh-ed25519 AAAA… comment"; empty with Local
	Wire            *catalog.WireConfig
	Event           func(step, line string)
}

// StepError names the step that failed, so the operator knows what was left
// in place.
type StepError struct {
	Step string
	Err  error
}

func (e *StepError) Error() string { return "pairing step " + e.Step + ": " + e.Err.Error() }
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
	_, err := r.sh("user", "getent group jumpgate >/dev/null || groupadd --system jumpgate; "+
		"id -u jumpgate >/dev/null 2>&1 || useradd --system --gid jumpgate --home-dir "+HomeDir+
		" --create-home --shell /usr/sbin/nologin jumpgate")
	return err
}

func (r runner) sshd() error {
	if r.o.Local {
		return nil
	}
	if err := r.o.Exec.WriteFile(r.ctx, DropInPath, []byte(sshdDropIn), 0o644); err != nil {
		return &StepError{Step: "sshd", Err: err}
	}
	if _, err := r.sh("sshd", "PATH=$PATH:/usr/sbin:/sbin sshd -t"); err != nil {
		_, _ = r.o.Exec.Run(r.ctx, "rm -f "+DropInPath, nil)
		return err
	}
	if _, err := r.sh("sshd", "systemctl reload ssh 2>/dev/null || systemctl reload sshd"); err != nil {
		return err
	}

	if strings.TrimSpace(r.o.TransportKey) == "" {
		return &StepError{Step: "sshd", Err: fmt.Errorf("no transport key to authorize")}
	}
	line := `restrict,port-forwarding,command="/bin/false" ` + strings.TrimSpace(r.o.TransportKey)
	keyField := strings.Fields(r.o.TransportKey)
	existing, _ := r.o.Exec.ReadFile(r.ctx, AuthorizedKeys)
	if len(keyField) >= 2 && hasKeyBlob(string(existing), keyField[1]) {
		return nil
	}
	merged := strings.TrimRight(string(existing), "\n")
	if merged != "" {
		merged += "\n"
	}
	merged += line + "\n"
	if _, err := r.sh("sshd", "install -d -m 0700 -o jumpgate -g jumpgate "+HomeDir+"/.ssh"); err != nil {
		return err
	}
	if err := r.o.Exec.WriteFile(r.ctx, AuthorizedKeys, []byte(merged), 0o600); err != nil {
		return &StepError{Step: "sshd", Err: err}
	}
	_, err := r.sh("sshd", "chown jumpgate:jumpgate "+AuthorizedKeys)
	return err
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
	if r.o.Local {
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
	out, err := r.sh("service", "systemctl is-active jumpgate-agent.service")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "active" {
		return &StepError{Step: "service", Err: fmt.Errorf("jumpgate-agent.service is %s; see journalctl -u jumpgate-agent", strings.TrimSpace(out))}
	}
	return nil
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'" }

// hasKeyBlob reports whether any authorized_keys line carries blob as a whole
// field. Options precede the key type, so the blob's position varies.
func hasKeyBlob(authorizedKeys, blob string) bool {
	for _, l := range strings.Split(authorizedKeys, "\n") {
		for _, f := range strings.Fields(l) {
			if f == blob {
				return true
			}
		}
	}
	return false
}
