package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/ai"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// pairedOpts varies pairedBoxWith's box and server.
type pairedOpts struct {
	setUp  bool                 // write a node.json on the box
	server func(*Config)        // adjust the server's Config (signer, executor seam)
	target func(*config.Target) // adjust the saved target after it is paired
	// onServer sees the server, for tests that look at its registry.
	onServer func(*Server)
	// ai, when set, is the provider the server's NewAIProvider seam hands
	// out, so no test dials a real one (Ruling T11).
	ai *fakeAIProvider
}

// requireAgentPeer skips a test that needs the agent to answer: it runs on
// Linux (and on macOS for tests) and refuses every peer elsewhere.
func requireAgentPeer(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("the agent refuses every peer on " + runtime.GOOS)
	}
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
	// The agent persists replay and policy records (fsyncing directories) and
	// refuses every peer outside Linux and macOS, as newRig in internal/agent
	// does: skip explicitly rather than pass without running.
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("the agent cannot run on " + runtime.GOOS)
	}
	home := shortTempDir(t, "jgn")
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
	if err := p.Save(filepath.Join(home, "policy.json")); err != nil {
		t.Fatal(err)
	}
	if err := agent.InitReplay(filepath.Join(home, "replay.json")); err != nil {
		t.Fatal(err)
	}
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
	if o.ai != nil {
		cfg.NewAIProvider = func(id, _, _ string) (ai.Provider, error) { o.ai.id = id; return o.ai, nil }
	}
	if o.server != nil {
		o.server(&cfg)
	}
	srv := New(cfg)
	if o.onServer != nil {
		o.onServer(srv)
	}
	// Cleanups run last first: the server closes (its handlers have all
	// returned), then its log followers stop, before the agent goes away
	// and before any test restores the timing globals (Ruling T8).
	t.Cleanup(srv.stopLogFollowers)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func TestServiceActionOnAPairedBoxUsesTheAgent(t *testing.T) {
	requireAgentPeer(t)
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
	requireAgentPeer(t)
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
	requireAgentPeer(t)
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

// shortTempDir is a temp directory whose paths fit a Unix socket's sun_path
// limit (104 bytes on macOS); Windows has no /tmp.
func shortTempDir(t *testing.T, prefix string) string {
	t.Helper()
	base := "/tmp"
	if runtime.GOOS == "windows" {
		base = ""
	}
	dir, err := os.MkdirTemp(base, prefix)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// fakeAgentSocket serves h on a Unix socket in place of an agent, for answers
// a real agent never gives (an HTTP error, silence). Register it before the
// server so its cleanup runs after the server has closed.
func fakeAgentSocket(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	sock := filepath.Join(shortTempDir(t, "jgf"), "f.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return sock
}

var wiredForLegacy = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/mnt/reth"}

// A paired box answers every node route with the agent's own error, whatever
// went wrong with the agent, and never opens the legacy executor, even with a
// wire on record that the executor could use.
func TestPairedFailuresNeverFallBackToSSH(t *testing.T) {
	old := agentStatusInterval
	agentStatusInterval = time.Hour
	t.Cleanup(func() { agentStatusInterval = old })
	other, _ := signer.GenerateKey()
	cases := []struct {
		name      string
		needsPeer bool // the real agent must answer
		setUp     bool
		target    func(t *testing.T, tg *config.Target)
		status    int
		code      api.Code
	}{
		{name: "unreachable", setUp: true, status: http.StatusGatewayTimeout, code: api.CodeUnreachable,
			target: func(t *testing.T, tg *config.Target) {
				tg.Agent.Socket = filepath.Join(filepath.Dir(tg.Agent.Socket), "gone.sock")
			}},
		{name: "agent_http", setUp: true, status: http.StatusBadGateway, code: api.CodeAgentHTTP,
			target: func(t *testing.T, tg *config.Target) {
				tg.Agent.Socket = fakeAgentSocket(t, func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "peer refused", http.StatusForbidden)
				})
			}},
		{name: "rejected", needsPeer: true, setUp: false, status: http.StatusConflict, code: api.CodeRejected},
		{name: "bad_receipt", needsPeer: true, setUp: true, status: http.StatusBadGateway, code: api.CodeBadReceipt,
			target: func(t *testing.T, tg *config.Target) { tg.Agent.Address = other.Address().Hex() }},
	}
	routes := []struct{ method, path string }{
		{"GET", "/du"}, {"GET", "/endpoints"}, {"GET", "/firewall"}, {"GET", "/logs"},
		{"POST", "/services/exec/restart"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.needsPeer {
				requireAgentPeer(t)
			}
			var legacy countingExec
			ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{setUp: c.setUp,
				server: func(cfg *Config) { cfg.NewExecutor = legacy.newExecutor },
				target: func(tg *config.Target) {
					tg.Wire = wiredForLegacy
					if c.target != nil {
						c.target(t, tg)
					}
				}})
			for _, r := range routes {
				res, e := do(t, ts, token, r.method, "/api/targets/box"+r.path, "")
				if res.StatusCode != c.status || e.Code != c.code || res.Header.Get("X-Jumpgate-Via") != "agent" {
					t.Errorf("%s %s: got %d %+v via %q", r.method, r.path, res.StatusCode, e, res.Header.Get("X-Jumpgate-Via"))
				}
			}
			// The logs stream opens with its (empty) reset, then the error.
			for path, n := range map[string]int{"/monitor/stream": 1, "/logs/stream?backlog=10": 2} {
				frames, h := readFrames(t, ts, token, "/api/targets/box"+path, n)
				last := frames[len(frames)-1]
				if h.Get("X-Jumpgate-Via") != "agent" || !strings.HasPrefix(last, "event: error\n") || !strings.Contains(last, `"code":"`+string(c.code)+`"`) {
					t.Errorf("%s: via %q frames %q", path, h.Get("X-Jumpgate-Via"), frames)
				}
			}
			if n := legacy.n.Load(); n != 0 {
				t.Fatalf("the legacy executor was opened %d times for a paired box", n)
			}
		})
	}
}

// A slow agent cannot hold a stream silent past the client's 45 s idle
// timeout: the stream keeps pinging while the poll waits, and a poll that
// outlives agentPollTimeout is reported as an "unreachable" error event.
func TestSlowAgentStreamStillPingsAndReportsTheTimeout(t *testing.T) {
	oldStatus, oldLogs, oldPing, oldPoll := agentStatusInterval, agentLogsSnapshotInterval, ssePingInterval, agentPollTimeout
	agentStatusInterval, agentLogsSnapshotInterval, ssePingInterval, agentPollTimeout = time.Hour, time.Hour, 20*time.Millisecond, 300*time.Millisecond
	t.Cleanup(func() {
		agentStatusInterval, agentLogsSnapshotInterval, ssePingInterval, agentPollTimeout = oldStatus, oldLogs, oldPing, oldPoll
	})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	sock := fakeAgentSocket(t, func(w http.ResponseWriter, r *http.Request) {
		select { // never answers on its own
		case <-r.Context().Done():
		case <-release:
		}
	})
	ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{setUp: true,
		target: func(tg *config.Target) { tg.Agent.Socket = sock }})
	for _, path := range []string{"/api/targets/box/monitor/stream", "/api/targets/box/logs/stream?backlog=5"} {
		r, stop := openSSE(t, &apiTestServer{ts: ts, token: token}, path)
		pinged := false
		ok := readLines(r, 5*time.Second, func(l string) bool {
			if l == ": ping\n" {
				pinged = true
			}
			return pinged && strings.HasPrefix(l, "data: ") && strings.Contains(l, `"code":"unreachable"`)
		})
		stop()
		if !ok {
			t.Errorf("%s: want pings and then an unreachable error event (pinged %v)", path, pinged)
		}
	}
}

// failSigner holds the controller's address but cannot sign, a local failure.
type failSigner struct{ signer.Signer }

func (failSigner) SignTypedData(context.Context, eip712.TypedData) (signer.Signature, error) {
	return signer.Signature{}, errors.New("keychain locked mid-request")
}

// A failure on this machine (signing, the sequence store) is the server's
// own, 500 internal, not the box's.
func TestLocalIntentFailureIsInternal(t *testing.T) {
	ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{setUp: true,
		server: func(c *Config) { c.Signer = failSigner{c.Signer} }})
	for _, r := range []struct{ method, path string }{{"GET", "/api/targets/box/du"}, {"POST", "/api/targets/box/intent/status.read"}} {
		res, e := do(t, ts, token, r.method, r.path, "")
		if res.StatusCode != http.StatusInternalServerError || e.Code != api.CodeInternal || !strings.Contains(e.Message, "keychain locked") {
			t.Errorf("%s: got %d %+v", r.path, res.StatusCode, e)
		}
	}
}

// Only pairing records an agent: a POST /api/targets carrying an agent block
// saves an unpaired target, whose node operations go to the legacy executor.
func TestAddTargetIgnoresAClientAgentBlock(t *testing.T) {
	a := newAPITestServer(t)
	evil, _ := signer.GenerateKey()
	body := map[string]any{"id": "box", "mode": "local",
		"agent":   map[string]any{"address": evil.Address().Hex(), "transport": "local", "socket": "/tmp/evil.sock", "nextSeq": 7},
		"gateway": map[string]any{"BindAddr": "0.0.0.0", "Port": 4000},
		"devnet":  map[string]any{"chainId": 1337},
	}
	res := a.do(t, "POST", "/api/targets", body)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d", res.StatusCode)
	}
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	tg, _ := findTarget(c, "box")
	if tg.Agent != nil || tg.Devnet != nil || tg.LegacyGateway != nil || tg.Wire != nil || len(c.Gateways) != 0 {
		t.Fatalf("server-owned fields taken from the client: %+v (gateways %d)", tg, len(c.Gateways))
	}
	res2, e := do(t, a.ts, a.token, "GET", "/api/targets/box/du", "")
	if res2.Header.Get("X-Jumpgate-Via") != "ssh" || e.Code != api.CodeTargetNotSetUp {
		t.Fatalf("node op on the added target: %d %+v via %q", res2.StatusCode, e, res2.Header.Get("X-Jumpgate-Via"))
	}
}

// failingExec fails every command, so the agent signs a failure receipt.
type failingExec struct{ nopExec }

func (failingExec) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, errors.New("du: cannot access")
}

func TestAgentFailureIsAgentFailed(t *testing.T) {
	requireAgentPeer(t)
	ts, token := pairedBox(t, failingExec{}, true)
	res, e := do(t, ts, token, "GET", "/api/targets/box/du", "")
	if res.StatusCode != http.StatusBadGateway || e.Code != api.CodeAgentFailed || !strings.Contains(e.Message, "cannot access") {
		t.Fatalf("got %d %+v", res.StatusCode, e)
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
	requireAgentPeer(t)
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
