package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// localSeams installs a world with the given OS, euid and terminal, records
// whether the daemon was started, and restores everything afterwards.
func localSeams(t *testing.T, goos string, euid int, terminal bool) *bool {
	t.Helper()
	testutil.Home(t)
	started := false
	oldGOOS, oldEuid, oldEnsure, oldTerm := hostGOOS, geteuid, ensureRunning, stdinIsTerminal
	hostGOOS, geteuid = goos, func() int { return euid }
	stdinIsTerminal = func() bool { return terminal }
	ensureRunning = func(context.Context, string) (daemon.Info, error) {
		started = true
		return daemon.Info{}, errors.New("test: no server")
	}
	t.Cleanup(func() { hostGOOS, geteuid, ensureRunning, stdinIsTerminal = oldGOOS, oldEuid, oldEnsure, oldTerm })
	return &started
}

// The seam stands in for the OS, so this runs as "macOS" and "Windows" on
// every runner, and on the real thing in CI.
func TestHostsAddLocalIsRefusedOffLinux(t *testing.T) {
	for _, goos := range []string{"darwin", "windows"} {
		started := localSeams(t, goos, 501, true)
		if code := hostsAdd([]string{"me", "--local"}); code != 1 {
			t.Fatalf("%s: exit %d, want 1", goos, code)
		}
		if *started {
			t.Fatalf("%s: the daemon was started before refusing --local", goos)
		}
	}
}

// Review Focus 3: no terminal means no sudo prompt is possible; fail at once.
func TestHostsAddLocalNonRootNeedsATerminal(t *testing.T) {
	started := localSeams(t, "linux", 1000, false)
	if code := hostsAdd([]string{"me", "--local"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if *started {
		t.Fatal("the daemon was started without a terminal to prompt on")
	}
}

// Root needs no terminal: the server runs the steps itself, without sudo.
func TestHostsAddLocalAsRootGoesToTheServer(t *testing.T) {
	started := localSeams(t, "linux", 0, false)
	hostsAdd([]string{"me", "--local"})
	if !*started {
		t.Fatal("root pairing never reached the server")
	}
}

type cmdRecorder struct {
	mu   sync.Mutex
	cmds []string
}

func (r *cmdRecorder) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmds = append(r.cmds, cmd)
	return executor.Result{}, nil
}
func (r *cmdRecorder) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (r *cmdRecorder) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (r *cmdRecorder) Close() error                                                 { return nil }

func withController(t *testing.T) eip712.Address {
	t.Helper()
	k, _ := signer.GenerateKey()
	if _, err := config.Update(func(c *config.Config) error {
		c.Controller = &config.Controller{KeyStore: "file", KeyRef: "x", Address: k.Address().Hex()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return k.Address()
}

// stubForeground replaces the sudo checks and bootstrap; the returned
// pointers record whether sudo prompted and what bootstrap was given.
func stubForeground(t *testing.T, nonInteractive, prompt error, agent eip712.Address) (*bool, *bootstrap.Options, *bool) {
	t.Helper()
	prompted, ran := false, false
	var got bootstrap.Options
	oldN, oldP, oldB := sudoNonInteractive, sudoPrompt, runBootstrap
	sudoNonInteractive = func(context.Context) error { return nonInteractive }
	sudoPrompt = func(context.Context) error { prompted = true; return prompt }
	runBootstrap = func(_ context.Context, o bootstrap.Options) (eip712.Address, error) {
		ran, got = true, o
		return agent, nil
	}
	t.Cleanup(func() { sudoNonInteractive, sudoPrompt, runBootstrap = oldN, oldP, oldB })
	return &prompted, &got, &ran
}

func TestPairLocalForegroundRunsBootstrapThroughSudo(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	controller := withController(t)
	rec := &cmdRecorder{}
	oldL := newLocalExecutor
	newLocalExecutor = func() executor.Executor { return rec }
	t.Cleanup(func() { newLocalExecutor = oldL })
	agent := eip712.Address{0x01}
	prompted, got, _ := stubForeground(t, nil, nil, agent)

	addr, code := pairLocalForeground(context.Background(), io.Discard, config.Target{ID: "me", Mode: "local"})
	if code != 0 || addr != agent.Hex() {
		t.Fatalf("pairLocalForeground = %q, %d", addr, code)
	}
	if *prompted {
		t.Fatal("prompted for a password although sudo -n worked")
	}
	if !got.Local || got.Controller != controller || got.LocalUID != os.Getuid() {
		t.Fatalf("options: Local=%v Controller=%s LocalUID=%d", got.Local, got.Controller.Hex(), got.LocalUID)
	}
	if got.AgentBinary == nil {
		t.Fatal("no agent binary lookup")
	}
	_, _ = got.Exec.Run(context.Background(), "id -u", nil)
	if len(rec.cmds) != 1 || !strings.HasPrefix(rec.cmds[0], "sudo -n ") {
		t.Fatalf("commands %v, want one run through sudo -n", rec.cmds)
	}
}

// P6: the target's node configuration reaches bootstrap, which writes
// node.json from it, exactly as the server's pairing does.
func TestPairLocalForegroundPassesTheTargetsWire(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	withController(t)
	_, got, _ := stubForeground(t, nil, nil, eip712.Address{0x01})
	wire := &catalog.WireConfig{ChainID: 369, ExecID: "reth"}
	if _, code := pairLocalForeground(context.Background(), io.Discard, config.Target{ID: "me", Mode: "local", Wire: wire}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got.Wire != wire {
		t.Fatalf("Wire = %v, want the target's", got.Wire)
	}
}

func TestPairLocalForegroundAsksForTheSudoPassword(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	withController(t)
	prompted, _, ran := stubForeground(t, errors.New("a password is required"), nil, eip712.Address{0x01})
	var out bytes.Buffer
	if _, code := pairLocalForeground(context.Background(), &out, config.Target{ID: "me", Mode: "local"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !*prompted || !*ran || !strings.Contains(out.String(), "sudo may ask for your password") {
		t.Fatalf("prompted=%v ran=%v out=%q", *prompted, *ran, out.String())
	}
}

// captureStderr redirects os.Stderr to a file for the rest of the test and
// returns a function reading what was written.
func captureStderr(t *testing.T) func() string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = f
	t.Cleanup(func() { os.Stderr = old; f.Close() })
	return func() string {
		b, _ := os.ReadFile(f.Name())
		return string(b)
	}
}

func TestPairLocalForegroundStopsWhenSudoIsRefused(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	withController(t)
	stderr := captureStderr(t)
	_, _, ran := stubForeground(t, errors.New("a password is required"), errors.New("user is not in the sudoers file"), eip712.Address{})
	if _, code := pairLocalForeground(context.Background(), io.Discard, config.Target{ID: "me", Mode: "local"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if *ran {
		t.Fatal("bootstrap ran without root")
	}
	// hosts add saved the target already; the operator must learn it is
	// unpaired and how to finish.
	if got := stderr(); !strings.Contains(got, "saved but not paired") || !strings.Contains(got, "jumpgate hosts add me --local") {
		t.Fatalf("stderr %q does not say the target is saved but unpaired", got)
	}
}

func TestPairLocalForegroundFailureSaysHowToFinish(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	withController(t)
	stderr := captureStderr(t)
	stubForeground(t, nil, nil, eip712.Address{})
	runBootstrap = func(context.Context, bootstrap.Options) (eip712.Address, error) {
		return eip712.Address{}, errors.New("preflight: not systemd")
	}
	if _, code := pairLocalForeground(context.Background(), io.Discard, config.Target{ID: "me", Mode: "local"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if got := stderr(); !strings.Contains(got, "saved but not paired") {
		t.Fatalf("stderr %q does not say the target is saved but unpaired", got)
	}
}

func TestPairedMessageAdvisesOnRootSSHOnlyAfterSSH(t *testing.T) {
	if m := pairedMessage("0x01", true); strings.Contains(m, "PermitRootLogin") || !strings.Contains(m, "paired: agent 0x01") {
		t.Fatalf("local: %q", m)
	}
	if m := pairedMessage("0x01", false); !strings.Contains(m, "PermitRootLogin") {
		t.Fatalf("ssh: %q", m)
	}
}

func TestPairLocalForegroundNeedsAControllerKey(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	_, _, ran := stubForeground(t, nil, nil, eip712.Address{})
	if _, code := pairLocalForeground(context.Background(), io.Discard, config.Target{ID: "me", Mode: "local"}); code != exitCode("no_controller_key") {
		t.Fatalf("exit %d, want %d", code, exitCode("no_controller_key"))
	}
	if *ran {
		t.Fatal("bootstrap ran without a controller key")
	}
}

// D19: the agent runs on the Linux boxes jumpgate manages, never on Windows.
func TestAgentIsRefusedOnWindows(t *testing.T) {
	localSeams(t, "windows", 1000, true)
	if code := cmdAgent([]string{"init"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}
