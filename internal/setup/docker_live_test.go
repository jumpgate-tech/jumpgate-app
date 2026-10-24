package setup

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/monitor"
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

// TestPrivateSendLive provisions a chain-1 Docker gateway with private send
// next to one public read upstream, then sends eth_sendRawTransaction with a
// payload that cannot decode as a transaction, so nothing can ever be mined.
// The error must come from a private route (eRPC names upstream 1-mod-*),
// never the public one, and a read must still be served. It builds the eRPC
// image when absent, so it also needs JUMPGATE_DOCKER_LIVE_GATEWAY=1.
func TestPrivateSendLive(t *testing.T) {
	if os.Getenv("JUMPGATE_DOCKER_LIVE") != "1" || os.Getenv("JUMPGATE_DOCKER_LIVE_GATEWAY") != "1" {
		t.Skip("set JUMPGATE_DOCKER_LIVE=1 and JUMPGATE_DOCKER_LIVE_GATEWAY=1 to provision a private-send gateway")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	e := executor.NewLocal()
	const id = "jumpgate-live-private-send"
	g := catalog.GatewayConfig{
		Port: freePort(t), MetricsPort: freePort(t),
		Modules: []catalog.GatewayModule{{ID: "private-send"}},
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "1-public", Endpoint: "https://ethereum-rpc.publicnode.com"},
		}}},
	}
	t.Cleanup(func() { _ = ops.RemoveContainer(context.Background(), e, ops.ERPCContainerNameFor(id)) })
	provisionLiveGateway(ctx, t, e, id, g)

	call := func(method, params string) string {
		t.Helper()
		body := `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`
		res, err := http.Post("http://127.0.0.1:"+strconv.Itoa(g.Port)+g.PathForArch("evm", 1), "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		return string(raw)
	}
	if got := call("eth_sendRawTransaction", `["0x00"]`); !strings.Contains(got, "1-mod-") || strings.Contains(got, "1-public") {
		t.Fatalf("the send was not answered by a private route only: %s", got)
	}
	if got := call("eth_blockNumber", `[]`); !strings.Contains(got, `"result":"0x`) {
		t.Fatalf("read not served: %s", got)
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

// TestMsgboardLive checks the D9 detection against real reth images. The
// negative half always runs: upstream reth refuses the msgboard namespace at
// startup, and the line it logs is the one the modules step quotes. The
// positive half needs a valve-tech/reth image with the board, named by
// JUMPGATE_MSGBOARD_IMAGE: msgboard_status answers and decodes, and a --dev
// node (no peers) is not accepting.
func TestMsgboardLive(t *testing.T) {
	if os.Getenv("JUMPGATE_DOCKER_LIVE") != "1" {
		t.Skip("set JUMPGATE_DOCKER_LIVE=1 to run against this machine's Docker engine")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	t.Run("upstream refuses the namespace", func(t *testing.T) {
		out, err := exec.CommandContext(ctx, "docker", "run", "--rm", "ghcr.io/paradigmxyz/reth:latest",
			"node", "--dev", "--http", "--http.api", "eth,net,web3,msgboard").CombinedOutput()
		if err == nil {
			t.Fatalf("upstream reth started with the msgboard namespace:\n%s", out)
		}
		line := lastUnknownModuleLine(string(out))
		if line == "" {
			t.Fatalf("no Unknown RPC module line in the log:\n%s", out)
		}
		t.Logf("quoted line: %s", line)
	})

	t.Run("the fork serves msgboard_status", func(t *testing.T) {
		image := os.Getenv("JUMPGATE_MSGBOARD_IMAGE")
		if image == "" {
			t.Skip("set JUMPGATE_MSGBOARD_IMAGE to a valve-tech/reth image with the message board")
		}
		port := freePort(t)
		name := "jumpgate-live-msgboard-" + strconv.Itoa(port)
		run := exec.CommandContext(ctx, "docker", "run", "-d", "--rm", "--name", name,
			"-p", "127.0.0.1:"+strconv.Itoa(port)+":8545", image,
			"node", "--dev", "--http", "--http.addr", "0.0.0.0", "--http.api", "eth,net,web3,msgboard")
		if out, err := run.CombinedOutput(); err != nil {
			t.Fatalf("docker run %s: %v\n%s", image, err, out)
		}
		t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", name).Run() })

		body := `{"jsonrpc":"2.0","id":1,"method":"msgboard_status","params":[]}`
		deadline := time.Now().Add(2 * time.Minute)
		for {
			res, err := http.Post("http://127.0.0.1:"+strconv.Itoa(port), "application/json", strings.NewReader(body))
			if err == nil {
				raw, _ := io.ReadAll(res.Body)
				res.Body.Close()
				st, perr := monitor.ParseModuleStatus("msgboard", string(raw))
				if perr != nil {
					t.Fatalf("msgboard_status %s: %v", raw, perr)
				}
				if st.Accepting {
					t.Fatalf("a --dev node with no peers is accepting: %+v", st)
				}
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("msgboard_status never answered: %v", err)
			}
			time.Sleep(time.Second)
		}
	})
}
