package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// pairedOpts varies pairedBoxWith's box and server.
type pairedOpts struct {
	setUp  bool                 // write a node.json on the box
	server func(*Config)        // adjust the server's Config (signer, executor seam)
	target func(*config.Target) // adjust the saved target after it is paired
}

// pairedBox is pairedLocal with a node set up on the box (when setUp) and the
// agent's commands answered by ex. The controller's target has no wire: a
// paired box's agent owns its own node.json.
func pairedBox(t *testing.T, ex executor.Executor, setUp bool) (*httptest.Server, string) {
	t.Helper()
	return pairedBoxWith(t, ex, pairedOpts{setUp: setUp})
}

func pairedBoxWith(t *testing.T, ex executor.Executor, o pairedOpts) (*httptest.Server, string) {
	t.Helper()
	home, _ := os.MkdirTemp("/tmp", "jgn")
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if o.setUp {
		b, _ := json.Marshal(catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/jumpgate-node/369"})
		if err := os.WriteFile(filepath.Join(home, "node.json"), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	agentKey, _ := signer.GenerateKey()
	controller, _ := signer.GenerateKey()
	p := agent.Policy{Signers: []agent.SignerEntry{{Address: controller.Address().Hex(), Tier: agent.TierRoutine}}}
	_ = p.Save(filepath.Join(home, "policy.json"))
	_ = agent.InitReplay(filepath.Join(home, "replay.json"))
	a := agent.New(agent.Config{Key: agentKey, Exec: ex, PolicyPath: filepath.Join(home, "policy.json"),
		ReplayPath: filepath.Join(home, "replay.json"), NodePath: filepath.Join(home, "node.json")})
	sock := filepath.Join(home, "a.sock")
	ln, err := agent.Listen(sock, -1, 0o660)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go agent.Serve(ctx, a, ln)
	if _, err := config.Update(func(c *config.Config) error {
		tg := config.Target{ID: "box", Mode: "local",
			Agent: &config.AgentPairing{Address: agentKey.Address().Hex(), Transport: "local", PairedAt: time.Now(), Socket: sock}}
		if o.target != nil {
			o.target(&tg)
		}
		c.Targets = append(c.Targets, tg)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	token := NewSessionToken()
	cfg := Config{Token: token, UI: fstest.MapFS{}, Signer: controller}
	if o.server != nil {
		o.server(&cfg)
	}
	ts := httptest.NewServer(New(cfg).Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func TestServiceActionOnAPairedBoxUsesTheAgent(t *testing.T) {
	ts, token := pairedBox(t, nopExec{}, true)
	res, e := do(t, ts, token, "POST", "/api/targets/box/services/exec/restart", "")
	if res.StatusCode != http.StatusOK || res.Header.Get("X-Jumpgate-Via") != "agent" {
		t.Fatalf("status %d via %q err %+v", res.StatusCode, res.Header.Get("X-Jumpgate-Via"), e)
	}
	c, _ := config.Load()
	if c.Targets[0].Agent.NextSeq == 0 {
		t.Fatalf("no signed intent was sent (next seq %d)", c.Targets[0].Agent.NextSeq)
	}
}

func TestAgentRejectionIsRejectedWithItsHint(t *testing.T) {
	ts, token := pairedBox(t, nopExec{}, false)
	res, e := do(t, ts, token, "GET", "/api/targets/box/du", "")
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeRejected || e.Reason != intent.ReasonNotSetUp || e.Hint != api.RejectionHint(intent.ReasonNotSetUp) {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
}

// TestSSHOnlyBoxUsesTheExecutor: an unpaired target keeps the legacy executor
// (seedWired saves a local one; "ssh" names the legacy transport either way).
func TestSSHOnlyBoxUsesTheExecutor(t *testing.T) {
	a := newAPITestServer(t)
	seedWired(t, "legacy")
	res := a.do(t, "POST", "/api/targets/legacy/services/exec/restart", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("X-Jumpgate-Via") != "ssh" {
		t.Fatalf("status %d via %q", res.StatusCode, res.Header.Get("X-Jumpgate-Via"))
	}
}

// Every request/response node route names its transport on an unpaired box.
func TestSSHOnlyNodeRoutesSayVia(t *testing.T) {
	a := newAPITestServer(t)
	seedWired(t, "legacy")
	for _, path := range []string{"/du", "/endpoints", "/firewall", "/logs"} {
		res := a.do(t, "GET", "/api/targets/legacy"+path, nil)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || res.Header.Get("X-Jumpgate-Via") != "ssh" {
			t.Errorf("%s: status %d via %q", path, res.StatusCode, res.Header.Get("X-Jumpgate-Via"))
		}
	}
}

// An unpaired box with no wire is still "not set up", and still says ssh.
func TestSSHOnlyBoxWithoutWireIsNotSetUp(t *testing.T) {
	a := newAPITestServer(t)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "bare", Mode: "local"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/du", "/endpoints", "/firewall", "/logs", "/monitor/stream", "/logs/stream"} {
		res, e := do(t, a.ts, a.token, "GET", "/api/targets/bare"+path, "")
		if res.StatusCode != http.StatusConflict || e.Code != api.CodeTargetNotSetUp {
			t.Errorf("%s: got %d %+v", path, res.StatusCode, e)
		}
	}
}

// Each paired route reaches the agent and answers in the route's own shape.
func TestPairedNodeRoutesAnswerThroughTheAgent(t *testing.T) {
	ts, token := pairedBox(t, &autoSucceedExecutor{}, true)
	for _, path := range []string{"/du", "/endpoints", "/firewall", "/logs?n=5"} {
		res, e := do(t, ts, token, "GET", "/api/targets/box"+path, "")
		if res.StatusCode != http.StatusOK || res.Header.Get("X-Jumpgate-Via") != "agent" {
			t.Errorf("%s: status %d via %q err %+v", path, res.StatusCode, res.Header.Get("X-Jumpgate-Via"), e)
		}
	}
}

// countingExec fails a test that reaches the legacy executor: a paired box's
// node operations must never fall back to root SSH.
type countingExec struct{ n atomic.Int32 }

func (c *countingExec) newExecutor(config.Target) (executor.Executor, error) {
	c.n.Add(1)
	return &autoSucceedExecutor{}, nil
}

// A paired box whose agent cannot be reached answers "unreachable" on every
// node route and never opens the legacy executor, even with a wire on record.
func TestUnreachableAgentNeverFallsBackToSSH(t *testing.T) {
	old := agentStatusInterval
	agentStatusInterval = time.Hour
	t.Cleanup(func() { agentStatusInterval = old })
	var legacy countingExec
	ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{setUp: true,
		server: func(c *Config) { c.NewExecutor = legacy.newExecutor },
		target: func(tg *config.Target) {
			tg.Agent.Socket = filepath.Join(filepath.Dir(tg.Agent.Socket), "gone.sock")
			tg.Wire = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/mnt/reth"}
		}})
	routes := []struct{ method, path string }{
		{"GET", "/du"}, {"GET", "/endpoints"}, {"GET", "/firewall"}, {"GET", "/logs"},
		{"POST", "/services/exec/restart"},
	}
	for _, r := range routes {
		res, e := do(t, ts, token, r.method, "/api/targets/box"+r.path, "")
		if res.StatusCode != http.StatusGatewayTimeout || e.Code != api.CodeUnreachable || res.Header.Get("X-Jumpgate-Via") != "agent" {
			t.Errorf("%s %s: got %d %+v via %q", r.method, r.path, res.StatusCode, e, res.Header.Get("X-Jumpgate-Via"))
		}
	}
	// The logs stream opens with its snapshot-mode note, then the error.
	for path, n := range map[string]int{"/monitor/stream": 1, "/logs/stream?backlog=10": 2} {
		frames, h := readFrames(t, ts, token, "/api/targets/box"+path, n)
		last := frames[len(frames)-1]
		if h.Get("X-Jumpgate-Via") != "agent" || !strings.HasPrefix(last, "event: error\n") || !strings.Contains(last, string(api.CodeUnreachable)) {
			t.Errorf("%s: via %q frames %q", path, h.Get("X-Jumpgate-Via"), frames)
		}
	}
	if n := legacy.n.Load(); n != 0 {
		t.Fatalf("the legacy executor was opened %d times for a paired box", n)
	}
}

// failingExec fails every command, so the agent signs a failure receipt.
type failingExec struct{ nopExec }

func (failingExec) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, errors.New("du: cannot access")
}

func TestAgentFailureIsAgentFailed(t *testing.T) {
	ts, token := pairedBox(t, failingExec{}, true)
	res, e := do(t, ts, token, "GET", "/api/targets/box/du", "")
	if res.StatusCode != http.StatusBadGateway || e.Code != api.CodeAgentFailed || !strings.Contains(e.Message, "cannot access") {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
}

// An answer not signed by the paired agent is a security error, and the
// legacy executor is not tried instead.
func TestForeignReceiptIsBadReceiptWithoutFallback(t *testing.T) {
	var legacy countingExec
	other, _ := signer.GenerateKey()
	ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{setUp: true,
		server: func(c *Config) { c.NewExecutor = legacy.newExecutor },
		target: func(tg *config.Target) {
			tg.Agent.Address = other.Address().Hex()
			tg.Wire = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/mnt/reth"}
		}})
	res, e := do(t, ts, token, "GET", "/api/targets/box/firewall", "")
	if res.StatusCode != http.StatusBadGateway || e.Code != api.CodeBadReceipt {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
	if n := legacy.n.Load(); n != 0 {
		t.Fatalf("the legacy executor was opened %d times for a paired box", n)
	}
}

// Ruling T5a: a node route on a server whose controller key would not open
// keeps the reason the key failed, exactly as /intent does.
func TestPairedNodeRouteWithoutAControllerKey(t *testing.T) {
	cases := []struct {
		name     string
		signErr  error
		code     api.Code
		inMsg    string
		hintFrom string
	}{
		{"none", nil, api.CodeNoControllerKey, "no controller key", ""},
		{"locked", errors.New("keychain is locked"), api.CodeNoControllerKey, "keychain is locked", "fix the key store"},
		{"mismatch", fmt.Errorf("%w: holds 0x1, records 0x2", signer.ErrAddressMismatch), api.CodeControllerKeyMismatch, "holds 0x1", "restore the original key"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{setUp: true,
				server: func(cfg *Config) { cfg.Signer, cfg.SignerErr = nil, c.signErr }})
			for _, path := range []string{"/api/targets/box/du", "/api/targets/box/intent/status.read"} {
				method := "GET"
				if strings.Contains(path, "/intent/") {
					method = "POST"
				}
				res, e := do(t, ts, token, method, path, "")
				if res.StatusCode != http.StatusServiceUnavailable || e.Code != c.code || !strings.Contains(e.Message, c.inMsg) || !strings.Contains(e.Hint, c.hintFrom) {
					t.Errorf("%s: got %d %+v", path, res.StatusCode, e)
				}
				if c.hintFrom == "" && e.Hint != api.HintFor(api.CodeNoControllerKey) {
					t.Errorf("%s: hint %q, want the registry's", path, e.Hint)
				}
			}
		})
	}
}

func readFrames(t *testing.T, ts *httptest.Server, token, path string, n int) ([]string, http.Header) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var frames []string
	var cur strings.Builder
	r := bufio.NewReader(res.Body)
	for len(frames) < n {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("after %d frames: %v", len(frames), err)
		}
		if line == "\n" {
			if f := cur.String(); !strings.HasPrefix(f, ": ") {
				frames = append(frames, f)
			}
			cur.Reset()
			continue
		}
		cur.WriteString(line)
	}
	return frames, res.Header
}

func TestPairedMonitorStreamPollsTheAgent(t *testing.T) {
	old := agentStatusInterval
	agentStatusInterval = 20 * time.Millisecond
	t.Cleanup(func() { agentStatusInterval = old })
	ts, token := pairedBox(t, nopExec{}, true)
	frames, h := readFrames(t, ts, token, "/api/targets/box/monitor/stream", 2)
	if h.Get("X-Jumpgate-Via") != "agent" {
		t.Fatalf("via %q", h.Get("X-Jumpgate-Via"))
	}
	for _, f := range frames {
		if !strings.HasPrefix(f, "data: {") || !strings.Contains(f, `"execHead"`) {
			t.Fatalf("frame %q is not a snapshot", f)
		}
	}
}

func TestPairedLogsStreamSendsSnapshots(t *testing.T) {
	ts, token := pairedBox(t, nopExec{}, true)
	frames, _ := readFrames(t, ts, token, "/api/targets/box/logs/stream?backlog=10", 2)
	if !strings.HasPrefix(frames[0], "event: note\n") || !strings.HasPrefix(frames[1], "event: reset\n") {
		t.Fatalf("frames %q", frames)
	}
}

// Task 3's keep-alive holds on the paired streams too: a quiet stream pings.
func TestPairedStreamsPingWhileQuiet(t *testing.T) {
	oldStatus, oldLogs, oldPing := agentStatusInterval, agentLogsSnapshotInterval, ssePingInterval
	agentStatusInterval, agentLogsSnapshotInterval, ssePingInterval = time.Hour, time.Hour, 20*time.Millisecond
	t.Cleanup(func() { agentStatusInterval, agentLogsSnapshotInterval, ssePingInterval = oldStatus, oldLogs, oldPing })
	ts, token := pairedBox(t, nopExec{}, true)
	for _, path := range []string{"/api/targets/box/monitor/stream", "/api/targets/box/logs/stream?backlog=5"} {
		r, stop := openSSE(t, &apiTestServer{ts: ts, token: token}, path)
		ok := readLines(r, streamWait, func(l string) bool { return l == ": ping\n" })
		stop()
		if !ok {
			t.Errorf("%s: no ping comment", path)
		}
	}
}
