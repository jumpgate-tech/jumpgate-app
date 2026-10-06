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

// TestLocalDockerLiveGateway puts a docker-backend gateway in front of a
// devnet through the real local executor, and reads its counters, so the
// gateway's whole shell-free path runs: --mount of erpc.yaml under the home
// directory, the in-process readiness probe and the metrics scrape. It builds
// the eRPC image when it is absent (BuildKit needed, minutes), so it has its
// own switch: JUMPGATE_DOCKER_LIVE_GATEWAY=1. On a VM-backed engine the home
// directory must be one the VM shares (colima shares /Users, not /tmp).
func TestLocalDockerLiveGateway(t *testing.T) {
	if os.Getenv("JUMPGATE_DOCKER_LIVE_GATEWAY") != "1" {
		t.Skip("set JUMPGATE_DOCKER_LIVE_GATEWAY=1 to provision a gateway on this machine's Docker engine")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	e := executor.NewLocal()
	d := catalog.DevnetConfig{ContainerName: "jumpgate-live-devnet", HTTPPort: freePort(t), WSPort: freePort(t)}
	t.Cleanup(func() { _ = ops.RemoveContainer(context.Background(), e, d.Name()) })
	steps, err := PlanDevnet(d)
	if err != nil {
		t.Fatal(err)
	}
	if err := RunAll(ctx, e, steps, &State{}); err != nil {
		t.Fatalf("devnet: %v", err)
	}

	const id = "jumpgate-live"
	g := catalog.GatewayForDevnet(d)
	g.Port = freePort(t)
	g.MetricsPort = freePort(t)
	t.Cleanup(func() { _ = ops.RemoveContainer(context.Background(), e, ops.ERPCContainerNameFor(id)) })
	provisionLiveGateway(ctx, t, e, id, g)
	samples, err := ReadGatewaySamples(ctx, e, g)
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	t.Logf("gateway answered eth_chainId %#x through the in-process probe; %d metric samples", d.ChainIDOrDefault(), len(samples))

	// The TLS front on Caddy's internal CA: the root comes out through
	// `docker exec … cat` (argv), and readiness is the in-process HTTPS probe
	// with Resolve and CAFile, verifying the chain against that root.
	g.TLS = &catalog.GatewayTLS{Enabled: true, Hostname: "rpc.jumpgate.test", HTTPSPort: freePort(t), BindAddr: "127.0.0.1"}
	// Caddy's data volume holds a CA that devices may trust, so it is removed
	// only when this test created it. Cleanups run last-in first-out: the
	// container goes first, then the volume it held.
	if res, err := ops.DockerRun(ctx, e, "volume", "inspect", catalog.CaddyDataVolume); err == nil && res.ExitCode != 0 {
		t.Cleanup(func() { _, _ = ops.DockerRun(context.Background(), e, "volume", "rm", catalog.CaddyDataVolume) })
	}
	t.Cleanup(func() { _ = ops.RemoveContainer(context.Background(), e, ops.CaddyContainerNameFor(id)) })
	provisionLiveGateway(ctx, t, e, id, g)
	t.Logf("TLS front answered at https://rpc.jumpgate.test:%d, verified against the exported root", g.TLS.HTTPSPort)
}

func provisionLiveGateway(ctx context.Context, t *testing.T, e executor.Executor, id string, g catalog.GatewayConfig) {
	t.Helper()
	gsteps, err := PlanGateway(id, g, BackendDocker)
	if err != nil {
		t.Fatal(err)
	}
	events, drained := make(chan Event, 1024), make(chan struct{})
	go func() {
		defer close(drained)
		for ev := range events {
			if ev.Line != "" {
				t.Logf("%s: %s", ev.StepID, ev.Line)
			}
		}
	}()
	err = RunAll(ctx, e, gsteps, &State{Events: events})
	close(events)
	<-drained
	if err != nil {
		t.Fatalf("gateway: %v", err)
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
