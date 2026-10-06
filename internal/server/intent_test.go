package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

type nopExec struct{}

func (nopExec) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, nil
}
func (nopExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (nopExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (nopExec) Close() error                                                 { return nil }

// pairedLocal starts a real agent on a temp socket and saves a target that is
// already paired with it, so the endpoint can be exercised without bootstrap.
func pairedLocal(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	testutil.RequireUnix(t) // starts a real agent, which needs peer credentials
	home := testutil.Home(t)

	agentKey, _ := signer.GenerateKey()
	controller, _ := signer.GenerateKey()
	p := agent.Policy{Signers: []agent.SignerEntry{{Address: controller.Address().Hex(), Tier: agent.TierRoutine}}}
	_ = p.Save(filepath.Join(home, "policy.json"))
	_ = agent.InitReplay(filepath.Join(home, "replay.json"))
	a := agent.New(agent.Config{Key: agentKey, Exec: nopExec{}, PolicyPath: filepath.Join(home, "policy.json"),
		ReplayPath: filepath.Join(home, "replay.json"), NodePath: filepath.Join(home, "node.json")})
	sock := filepath.Join(home, "a.sock")
	ln, _ := agent.Listen(sock, -1, 0o660)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go agent.Serve(ctx, a, ln)

	_, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "local",
			Agent: &config.AgentPairing{Address: agentKey.Address().Hex(), Transport: "local", PairedAt: time.Now(), Socket: sock}})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	token := NewSessionToken()
	ts := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, Signer: controller}).Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func postIntent(t *testing.T, ts *httptest.Server, token, path, body string) (*http.Response, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res, out
}

func nextSeq(t *testing.T, id string) uint64 {
	t.Helper()
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	tg, ok := findTarget(c, id)
	if !ok || tg.Agent == nil {
		t.Fatalf("target %q is not paired in config", id)
	}
	return tg.Agent.NextSeq
}

// addPairedTarget stores one more paired local target, cloned from box with
// the changes edit makes.
func addPairedTarget(t *testing.T, id string, edit func(*config.AgentPairing)) {
	t.Helper()
	_, err := config.Update(func(c *config.Config) error {
		box, _ := findTarget(*c, "box")
		ap := *box.Agent
		edit(&ap)
		c.Targets = append(c.Targets, config.Target{ID: id, Mode: "local", Agent: &ap})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntentEndpointRunsAVerifiedIntent(t *testing.T) {
	ts, token := pairedLocal(t)
	res, out := postIntent(t, ts, token, "/api/targets/box/intent/agent.info", `{}`)
	if res.StatusCode != http.StatusOK || out["status"].(float64) != float64(intent.StatusOK) {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	// The sequence advanced in config, so the next call does not collide.
	c, _ := config.Load()
	if c.Targets[0].Agent.NextSeq != 2 {
		t.Fatalf("NextSeq = %d, want 2", c.Targets[0].Agent.NextSeq)
	}
	if res, _ := postIntent(t, ts, token, "/api/targets/box/intent/agent.info", `{}`); res.StatusCode != http.StatusOK {
		t.Fatalf("second call: %d", res.StatusCode)
	}
}

func TestIntentEndpointRefusesUnpairedAndKeylessServers(t *testing.T) {
	ts, token := pairedLocal(t)
	_, _ = config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "raw", Mode: "local"})
		return nil
	})
	if res, out := postIntent(t, ts, token, "/api/targets/raw/intent/agent.info", `{}`); res.StatusCode != http.StatusConflict || out["code"] != "not_paired" || out["hint"] != api.HintFor(api.CodeNotPaired) {
		t.Fatalf("unpaired: %d %v", res.StatusCode, out)
	}
	keyless := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}}).Handler())
	defer keyless.Close()
	if res, out := postIntent(t, keyless, token, "/api/targets/box/intent/agent.info", `{}`); res.StatusCode != http.StatusServiceUnavailable || out["code"] != "no_controller_key" || out["hint"] != api.HintFor(api.CodeNoControllerKey) {
		t.Fatalf("keyless: %d %v", res.StatusCode, out)
	}
}

// A key that would not open is reported with its reason, so an operator with a
// locked keychain is not told to run `keys init` over a key that exists.
func TestIntentEndpointReportsWhyTheKeyIsMissing(t *testing.T) {
	ts, token := pairedLocal(t)
	ts.Close()
	broken := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, SignerErr: errors.New("keychain is locked")}).Handler())
	defer broken.Close()
	res, out := postIntent(t, broken, token, "/api/targets/box/intent/agent.info", `{}`)
	if res.StatusCode != http.StatusServiceUnavailable || out["code"] != "no_controller_key" {
		t.Fatalf("got %d %v", res.StatusCode, out)
	}
	if msg, _ := out["error"].(string); !strings.Contains(msg, "keychain is locked") {
		t.Fatalf("error %q does not say why the key is missing", out["error"])
	}
}

// A controller key that is not the recorded identity has its own code, so the
// operator is not sent to re-pair boxes that are fine.
func TestIntentEndpointReportsAControllerKeyMismatch(t *testing.T) {
	ts, token := pairedLocal(t)
	ts.Close()
	mismatch := fmt.Errorf("open controller key: %w", signer.ErrAddressMismatch)
	broken := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, SignerErr: mismatch}).Handler())
	defer broken.Close()
	res, out := postIntent(t, broken, token, "/api/targets/box/intent/agent.info", `{}`)
	if res.StatusCode != http.StatusServiceUnavailable || out["code"] != "controller_key_mismatch" || out["hint"] == "" {
		t.Fatalf("got %d %v", res.StatusCode, out)
	}
}

// Concurrent intents to one target are serialised, so no two of them sign the
// same sequence and every one is admitted.
func TestIntentEndpointSerialisesConcurrentIntentsToOneTarget(t *testing.T) {
	ts, token := pairedLocal(t)
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/targets/box/intent/agent.info", strings.NewReader(`{}`))
			req.Header.Set("Authorization", "Bearer "+token)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				errs <- err.Error()
				return
			}
			defer res.Body.Close()
			var out map[string]any
			_ = json.NewDecoder(res.Body).Decode(&out)
			if res.StatusCode != http.StatusOK || out["status"] != float64(intent.StatusOK) {
				errs <- strings.TrimSpace(func() string { b, _ := json.Marshal(out); return string(b) }())
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("concurrent intent failed: %s", e)
	}
	if got := nextSeq(t, "box"); got != n+1 {
		t.Errorf("NextSeq = %d, want %d", got, n+1)
	}
}

// An answer signed by anything but the paired agent is a security error, not
// a result.
func TestIntentEndpointReportsABadReceipt(t *testing.T) {
	ts, token := pairedLocal(t)
	impostor, _ := signer.GenerateKey()
	addPairedTarget(t, "swapped", func(ap *config.AgentPairing) { ap.Address = impostor.Address().Hex() })
	res, out := postIntent(t, ts, token, "/api/targets/swapped/intent/agent.info", `{}`)
	if res.StatusCode != http.StatusBadGateway || out["code"] != "bad_receipt" || out["hint"] == "" {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
}

func TestIntentEndpointReportsAnUnreachableAgent(t *testing.T) {
	ts, token := pairedLocal(t)
	addPairedTarget(t, "gone", func(ap *config.AgentPairing) { ap.Socket = filepath.Join(os.Getenv("HOME"), "nobody.sock") })
	res, out := postIntent(t, ts, token, "/api/targets/gone/intent/agent.info", `{}`)
	if res.StatusCode != http.StatusGatewayTimeout || out["code"] != "unreachable" {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
}

// An HTTP refusal from the socket (here 403, a peer not allowed on it) is
// neither a transport failure nor a signed rejection.
func TestIntentEndpointReportsAnAgentHTTPRefusal(t *testing.T) {
	ts, token := pairedLocal(t)
	sock := filepath.Join(os.Getenv("HOME"), "refuse.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	refuser := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not allowed on this socket", http.StatusForbidden)
	})}
	go refuser.Serve(ln)
	t.Cleanup(func() { refuser.Close() })
	addPairedTarget(t, "refusing", func(ap *config.AgentPairing) { ap.Socket = sock })

	res, out := postIntent(t, ts, token, "/api/targets/refusing/intent/agent.info", `{}`)
	if res.StatusCode != http.StatusBadGateway || out["code"] != "agent_http" {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	if hint, _ := out["hint"].(string); !strings.Contains(hint, "allowed") || !strings.Contains(hint, "too large") {
		t.Errorf("hint does not name the likely causes: %q", hint)
	}
}

// The intent route makes the server sign, so a cookie-authorised POST from
// another origin must never reach the agent.
func TestIntentEndpointRefusesACrossOriginCookiePost(t *testing.T) {
	ts, token := pairedLocal(t)
	res := cookieReq(t, http.MethodPost, ts.URL+"/api/targets/box/intent/agent.info", token,
		map[string]string{"Origin": "http://127.0.0.1:3000"})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin cookie POST = %d, want 403", res.StatusCode)
	}
	if got := nextSeq(t, "box"); got != 0 {
		t.Fatalf("NextSeq = %d: the refused request reached the agent", got)
	}
}

// A remote agent is reached as the jumpgate tunnel user with the transport
// key, and its host key is checked strictly against the confirmed store: a
// host only trust-on-first-use has seen is refused.
func TestAgentTargetIsStrictAndUsesTheTunnelUser(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	testutil.Home(t)
	agentKey, _ := signer.GenerateKey()
	jump := &executor.SSHConfig{Host: "bastion.example", User: "ops"}
	tg := config.Target{ID: "box", Mode: "ssh",
		SSH:   &executor.SSHConfig{Host: d.host, Port: d.port, User: "root", KeyPath: "/k", Jump: jump},
		Agent: &config.AgentPairing{Address: agentKey.Address().Hex(), Transport: "ssh"}}

	at, err := agentTarget(tg)
	if err != nil {
		t.Fatal(err)
	}
	if at.Local || at.SSH.User != "jumpgate" || at.SSH.KeyPath != transportKeyPath() || at.SSH.Jump != jump || at.Agent != agentKey.Address() {
		t.Fatalf("agent target %+v", at)
	}
	if at.SSH.HostKey == nil {
		t.Fatal("no strict host-key callback")
	}

	// TOFU has recorded the key; Strict must still call the host unknown.
	tofu := filepath.Join(os.Getenv("HOME"), ".jumpgate", "known_hosts")
	hostname := net.JoinHostPort(d.host, strconv.Itoa(d.port))
	if err := executor.RecordHostKey(tofu, hostname, d.hostKey); err != nil {
		t.Fatal(err)
	}
	var unknown *executor.UnknownHostError
	if err := at.SSH.HostKey(hostname, nil, d.hostKey); !errors.As(err, &unknown) {
		t.Fatalf("TOFU-only host: %v, want UnknownHostError", err)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, hostname, d.hostKey); err != nil {
		t.Fatal(err)
	}
	if err := at.SSH.HostKey(hostname, nil, d.hostKey); err != nil {
		t.Fatalf("confirmed host refused: %v", err)
	}
}

// The server sends the remedy for a rejection, so no client needs its own table.
func TestIntentRejectionCarriesAHint(t *testing.T) {
	ts, token := pairedLocal(t) // no node.json: the agent rejects with not_set_up
	res, out := postIntent(t, ts, token, "/api/targets/box/intent/status.read", `{}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	rej, _ := out["rejection"].(map[string]any)
	if rej["code"] != intent.ReasonNotSetUp || out["hint"] != api.RejectionHint(intent.ReasonNotSetUp) {
		t.Fatalf("reply %v", out)
	}
}
