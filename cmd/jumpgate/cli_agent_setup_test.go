package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// stubAgentSetup sets the euid and replaces bootstrap and the local
// executor; the returned pointers record whether bootstrap ran and with what.
func stubAgentSetup(t *testing.T, euid int) (*bool, *bootstrap.Options, *cmdRecorder) {
	t.Helper()
	ran := false
	var got bootstrap.Options
	rec := &cmdRecorder{}
	oldE, oldB, oldL := geteuid, runBootstrap, newLocalExecutor
	geteuid = func() int { return euid }
	runBootstrap = func(_ context.Context, o bootstrap.Options) (eip712.Address, error) {
		ran, got = true, o
		return eip712.Address{0x02}, nil
	}
	newLocalExecutor = func() executor.Executor { return rec }
	t.Cleanup(func() { geteuid, runBootstrap, newLocalExecutor = oldE, oldB, oldL })
	return &ran, &got, rec
}

const testTransportKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJumpgateTestKeyOnlyForTests jumpgate-controller"

func TestAgentSetupRunsBootstrapWithTheTransportKey(t *testing.T) {
	ran, got, rec := stubAgentSetup(t, 0)
	controller := eip712.Address{0x01}
	agentAddr := eip712.Address{0x02}.Hex()
	var out strings.Builder
	captureStderr(t)
	if code := agentSetup(context.Background(), &out, controller.Hex(), testTransportKey, ""); code != 0 {
		t.Fatalf("exit %d, out %q", code, out.String())
	}
	if !*ran {
		t.Fatal("bootstrap never ran")
	}
	if got.Local || got.TransportKey != testTransportKey || got.Controller != controller || got.ControllerLabel != "pull" {
		t.Fatalf("bootstrap options %+v", *got)
	}
	if e, ok := got.Exec.(*cmdRecorder); !ok || e != rec {
		t.Fatalf("Exec is %T, want the plain local executor (no sudo wrapper)", got.Exec)
	}
	if got.AgentBinary == nil {
		t.Fatal("no agent binary source")
	}
	if !strings.Contains(out.String(), "gate address "+agentAddr) {
		t.Fatalf("stdout lacks the gate address: %q", out.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if last := lines[len(lines)-1]; !strings.HasPrefix(last, "on the controller, run: jumpgate hosts add") ||
		!strings.Contains(last, "--installed "+agentAddr) {
		t.Fatalf("last line %q", last)
	}
}

func TestAgentSetupNeedsRoot(t *testing.T) {
	ran, _, _ := stubAgentSetup(t, 1000)
	captureStderr(t)
	if code := agentSetup(context.Background(), io.Discard, eip712.Address{0x01}.Hex(), testTransportKey, ""); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if *ran {
		t.Fatal("bootstrap ran without root")
	}
}

func TestAgentSetupRefusesABadAddress(t *testing.T) {
	ran, _, _ := stubAgentSetup(t, 0)
	captureStderr(t)
	old := hostGOOS
	t.Cleanup(func() { hostGOOS = old })
	hostGOOS = "linux"
	if code := cmdAgent([]string{"setup", "--controller", "nope", "--transport-key", testTransportKey}); code != exitCode("usage") {
		t.Fatalf("exit %d, want usage", code)
	}
	if code := cmdAgent([]string{"setup", "--controller", eip712.Address{0x01}.Hex()}); code != exitCode("usage") {
		t.Fatalf("no --transport-key: exit %d, want usage", code)
	}
	if *ran {
		t.Fatal("bootstrap ran on bad arguments")
	}
}

func TestSelfBinaryRefusesAnotherArch(t *testing.T) {
	if _, err := selfBinary("not-" + "an-arch"); err == nil {
		t.Fatal("selfBinary served another arch")
	}
}
