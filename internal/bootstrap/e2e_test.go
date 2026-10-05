// internal/bootstrap/e2e_test.go
//go:build e2e

package bootstrap

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/eip712"
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

// The --local cases run INSIDE the container: scripts/e2e-agent.sh copies a
// linux build of this test binary in and runs it as JUMPGATE_E2E_LOCAL_USER, a
// non-root sudoer, with JUMPGATE_E2E_AGENTS pointing at the copied agents and
// JUMPGATE_E2E_STRANGER naming a second non-root user outside group jumpgate.

// TestE2ELocalPairAndPeerGate is `jumpgate hosts add local --local` from a
// non-root sudoer: bootstrap through sudo with the caller's uid enrolled, then
// a signed intent straight over the unix socket. It proves ruling R23: the
// socket goes 0666 once a local uid is enrolled, the peer gate admits that uid,
// and it refuses a different non-root uid even with a valid signature.
func TestE2ELocalPairAndPeerGate(t *testing.T) {
	stranger := e2eEnv(t, "JUMPGATE_E2E_STRANGER")
	agents := e2eAgents(t)
	uid := os.Getuid()
	if uid == 0 {
		t.Fatal("run this as a non-root sudoer; root would pass the peer gate on uid alone")
	}
	jg, err := user.LookupGroup("jumpgate")
	if err == nil {
		gids, _ := os.Getgroups()
		for _, g := range gids {
			if strconv.Itoa(g) == jg.Gid {
				t.Fatal("the local user is in group jumpgate, so the group, not the enrolment, would admit it")
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ex := executor.Sudo(executor.NewLocal())
	defer ex.Close()

	// Before: a box paired only for remote controllers keeps the socket 0660.
	if r, err := ex.Run(ctx, "test -S "+agentclient.DefaultSocket+" && ! grep -q localUids /etc/jumpgate/policy.json && stat -c %a "+agentclient.DefaultSocket, nil); err == nil && r.ExitCode == 0 {
		if mode := strings.TrimSpace(r.Stdout); mode != "660" {
			t.Fatalf("remote-only socket is %s, want 660", mode)
		}
		t.Log("remote-only socket was 660 before the local pairing")
	}

	controller, _ := signer.GenerateKey()
	agentAddr, err := Run(ctx, Options{
		Exec: ex, Local: true, LocalUID: uid, Controller: controller.Address(), ControllerLabel: "e2e-local",
		AgentBinary: agents,
		Event:       func(step, line string) { t.Logf("[%s] %s", step, line) },
	})
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(agentclient.DefaultSocket)
	if err != nil {
		t.Fatal(err)
	}
	if mode := fi.Mode().Perm(); mode != 0o666 {
		t.Fatalf("socket mode %o with a local uid enrolled, want 666", mode)
	}

	c, err := agentclient.Dial(ctx, agentclient.Target{Local: true, Agent: agentAddr}, controller, agentclient.NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(ctx, intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("agent.info from enrolled uid %d: %+v %v", uid, res, err)
	}
	t.Logf("enrolled uid %d: agent.info ok", uid)

	// The same controller's intent, sent by a uid that is not enrolled.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, "sudo", "-n", "-u", stranger, "env",
		"JUMPGATE_E2E_STRANGER_KEY="+hex.EncodeToString(controller.Bytes()),
		"JUMPGATE_E2E_STRANGER_AGENT="+agentAddr.Hex(),
		exe, "-test.run", "^TestE2ELocalStrangerIsRefused$", "-test.v", "-test.count=1")
	out, err := cmd.CombinedOutput()
	t.Logf("as %s:\n%s", stranger, out)
	if err != nil || !strings.Contains(string(out), "--- PASS: TestE2ELocalStrangerIsRefused") {
		t.Fatalf("stranger run: %v", err)
	}
}

// TestE2ELocalStrangerIsRefused runs as the stranger, started by the test
// above. The kernel lets it connect (the socket is 0666), and then the peer
// gate closes the connection before a validly signed intent is read.
func TestE2ELocalStrangerIsRefused(t *testing.T) {
	keyHex := e2eEnv(t, "JUMPGATE_E2E_STRANGER_KEY")
	agentAddr, err := eip712.ParseAddress(e2eEnv(t, "JUMPGATE_E2E_STRANGER_AGENT"))
	if err != nil {
		t.Fatal(err)
	}
	if os.Getuid() == 0 {
		t.Fatal("the stranger must not be root")
	}
	b, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatal(err)
	}
	controller, err := signer.KeyFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := net.Dial("unix", agentclient.DefaultSocket)
	if err != nil {
		t.Fatalf("uid %d cannot even connect, so the socket mode (not the peer gate) refused it: %v", os.Getuid(), err)
	}
	conn.Close()

	c, err := agentclient.Dial(ctx, agentclient.Target{Local: true, Agent: agentAddr}, controller, agentclient.NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(ctx, intent.KindAgentInfo, struct{}{})
	if err == nil {
		t.Fatalf("uid %d was served: %+v", os.Getuid(), res)
	}
	if !errors.Is(err, agentclient.ErrUnreachable) {
		t.Fatalf("uid %d: %v; want the connection closed by the peer gate", os.Getuid(), err)
	}
	t.Logf("uid %d connected but was refused: %v", os.Getuid(), err)
}
