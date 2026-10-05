// internal/bootstrap/e2e_test.go
//go:build e2e

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// Driven by scripts/e2e-agent.sh, which starts the container and exports
// JUMPGATE_E2E_{PORT,ROOT_KEY,TRANSPORT_KEY,TRANSPORT_PUB,AGENTS}.
func e2eEnv(t *testing.T, k string) string {
	t.Helper()
	v := os.Getenv(k)
	if v == "" {
		t.Skip(k + " not set; run scripts/e2e-agent.sh")
	}
	return v
}

// e2ePort parses the container's published port.
func e2ePort(t *testing.T) int {
	t.Helper()
	port, err := strconv.Atoi(e2eEnv(t, "JUMPGATE_E2E_PORT"))
	if err != nil || port <= 0 {
		t.Fatalf("JUMPGATE_E2E_PORT is not a port: %v", err)
	}
	return port
}

// e2eAgents returns a binary lookup after checking that both agents exist, so
// a bad directory fails here and not halfway through pairing a box.
func e2eAgents(t *testing.T) func(arch string) (string, error) {
	t.Helper()
	dir := e2eEnv(t, "JUMPGATE_E2E_AGENTS")
	for _, arch := range []string{"amd64", "arm64"} {
		if _, err := os.Stat(dir + "/jumpgate-linux-" + arch); err != nil {
			t.Fatalf("JUMPGATE_E2E_AGENTS: %v", err)
		}
	}
	return func(arch string) (string, error) { return dir + "/jumpgate-linux-" + arch, nil }
}

// waitAgentSocket waits (about 10s) for the agent's socket to exist: bootstrap
// restarts the unit, and "active" does not mean it is listening yet.
func waitAgentSocket(ctx context.Context, t *testing.T, ex executor.Executor) {
	t.Helper()
	for i := 0; i < 20; i++ {
		r, err := ex.Run(ctx, "test -S "+agentclient.DefaultSocket, nil)
		if err == nil && r.ExitCode == 0 {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("agent socket %s never appeared", agentclient.DefaultSocket)
}

func TestE2EPairAndRoundTrip(t *testing.T) {
	port := e2ePort(t)
	root := executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root", KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}
	agents := e2eAgents(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	controller, _ := signer.GenerateKey()
	agentAddr, err := Run(ctx, Options{
		Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"),
		AgentBinary:  agents,
		Event:        func(step, line string) { t.Logf("[%s] %s", step, line) },
	})
	if err != nil {
		t.Fatal(err)
	}

	waitAgentSocket(ctx, t, ex)
	tunnel := agentclient.Target{Agent: agentAddr, SSH: executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "jumpgate",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}}
	c, err := agentclient.Dial(ctx, tunnel, controller, agentclient.NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(ctx, intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("agent.info over the tunnel: %+v %v", res, err)
	}

	// The tunnel user can do nothing else: no shell, no TCP forwarding.
	tun, err := executor.DialSSH(ctx, tunnel.SSH)
	if err != nil {
		t.Fatalf("the tunnel user cannot connect: %v", err)
	}
	if sess, err := tun.NewSession(); err == nil {
		out, _ := sess.CombinedOutput("id")
		sess.Close()
		if strings.Contains(string(out), "uid=") {
			t.Fatal("the tunnel user got a shell")
		}
	}
	if conn, err := tun.Dial("tcp", "127.0.0.1:22"); err == nil {
		conn.Close()
		t.Fatal("the tunnel user can forward TCP")
	}
	// PermitOpen pins streamlocal to the agent's socket; systemd's is refused.
	if conn, err := tun.Dial("unix", "/run/systemd/private"); err == nil {
		conn.Close()
		t.Fatal("the tunnel user can reach a socket other than the agent's")
	}
	tun.Close()

	// Re-running bootstrap keeps the agent's identity.
	again, err := Run(ctx, Options{Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"),
		AgentBinary:  agents})
	if err != nil || again != agentAddr {
		t.Fatalf("re-pair: %s, %v; want the same agent %s", again.Hex(), err, agentAddr.Hex())
	}
}

// The spec's acceptance: a captured intent replayed to the same agent is
// stale_seq; the same intent addressed elsewhere is wrong_agent.
func TestE2EReplayAndWrongAgent(t *testing.T) {
	port := e2ePort(t)
	agents := e2eAgents(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()})
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()
	controller, _ := signer.GenerateKey() // a fresh controller: Run enrolls it beside any earlier one
	agentAddr, err := Run(ctx, Options{Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e-replay",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"),
		AgentBinary:  agents})
	if err != nil {
		t.Fatal(err)
	}

	waitAgentSocket(ctx, t, ex)
	tunnel, err := executor.DialSSH(ctx, executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "jumpgate",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()})
	if err != nil {
		t.Fatal(err)
	}
	defer tunnel.Close()
	hc := &http.Client{Transport: &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
		return tunnel.Dial("unix", agentclient.DefaultSocket)
	}}}
	post := func(env intent.Envelope) intent.Rejection {
		t.Helper()
		b, _ := json.Marshal(env)
		res, err := hc.Post("http://agent/v1/intent", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out intent.ReceiptEnvelope
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		var rej intent.Rejection
		if out.Receipt.Status == intent.StatusRejected {
			_ = json.Unmarshal(out.Result, &rej)
		}
		return rej
	}
	sign := func(i intent.Intent) intent.Envelope {
		sig, _ := controller.SignTypedData(ctx, i.TypedData())
		return intent.Envelope{Intent: i.JSON(), Payload: []byte("{}"), Sigs: []string{sig.Hex()}}
	}

	now := uint64(time.Now().Unix())
	n1, _ := intent.NewNonce()
	first := intent.Intent{Agent: agentAddr, Controller: controller.Address(), Seq: 1, Nonce: n1,
		IssuedAt: now, Expiry: now + 120, Kind: intent.KindAgentInfo, PayloadHash: intent.Hash([]byte("{}"))}
	captured := sign(first)
	if rej := post(captured); rej.Code != "" {
		t.Fatalf("first send refused: %+v", rej)
	}
	if rej := post(captured); rej.Code != intent.ReasonStaleSeq {
		t.Fatalf("replay: %+v, want stale_seq", rej)
	}

	other, _ := signer.GenerateKey()
	n2, _ := intent.NewNonce()
	misaddressed := first
	misaddressed.Seq, misaddressed.Nonce, misaddressed.Agent = 2, n2, other.Address()
	if rej := post(sign(misaddressed)); rej.Code != intent.ReasonWrongAgent {
		t.Fatalf("misaddressed: %+v, want wrong_agent", rej)
	}
}

// Task 4's guarantee against a real sshd: a cancelled command dies on the box.
func TestE2ECancelKillsTheRemoteCommand(t *testing.T) {
	port := e2ePort(t)
	ex, err := executor.NewSSHContext(context.Background(), executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()})
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()
	ctx, cancel := context.WithCancel(context.Background())
	go ex.Run(ctx, "sleep 4242", nil)
	time.Sleep(time.Second)
	// The bracket keeps pgrep (and the sh wrapping it, whose argv holds this
	// text) from matching themselves.
	const find = "pgrep -f '[s]leep 4242' || true"
	if r, _ := ex.Run(context.Background(), find, nil); strings.TrimSpace(r.Stdout) == "" {
		t.Fatal("the remote command never started, so the test proves nothing")
	}
	cancel()
	time.Sleep(7 * time.Second)
	r, _ := ex.Run(context.Background(), find, nil)
	if strings.TrimSpace(r.Stdout) != "" {
		t.Fatalf("remote command survived cancellation: pids %s", r.Stdout)
	}
}
