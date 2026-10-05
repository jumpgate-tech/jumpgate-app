package setup

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// TestLocalDockerLive runs the devnet plan through the real local executor
// against this machine's real engine: RunArgv, the in-process probes, the
// whole path a Windows controller takes. With a Windows-container engine
// (GitHub's Windows runners, a Windows Server VM) it asserts the message
// that user sees instead. Set JUMPGATE_DOCKER_LIVE=1 to run it.
func TestLocalDockerLive(t *testing.T) {
	if os.Getenv("JUMPGATE_DOCKER_LIVE") != "1" {
		t.Skip("set JUMPGATE_DOCKER_LIVE=1 to run against this machine's Docker engine")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	e := executor.NewLocal()
	info, err := ops.ProbeDocker(ctx, e)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !info.DaemonReachable {
		t.Fatalf("no engine answered: %s", info.DaemonError)
	}
	d := catalog.DevnetConfig{ContainerName: "jumpgate-live-devnet", HTTPPort: freePort(t), WSPort: freePort(t)}
	steps, err := PlanDevnet(d)
	if err != nil {
		t.Fatal(err)
	}
	if info.WindowsContainers() {
		err := RunAll(ctx, e, steps, &State{})
		if err == nil || !strings.Contains(err.Error(), "Linux containers") {
			t.Fatalf("a Windows-container engine: %v, want the switch-to-Linux-containers refusal", err)
		}
		t.Logf("refused as a Windows user will see it: %v", err)
		return
	}
	t.Cleanup(func() { _ = ops.RemoveContainer(context.Background(), e, d.Name()) })
	if err := RunAll(ctx, e, steps, &State{}); err != nil {
		t.Fatalf("devnet through the local executor: %v", err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
