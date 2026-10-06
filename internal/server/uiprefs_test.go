// internal/server/uiprefs_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
)

func getJSON(t *testing.T, url, token string, out any) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	_ = json.NewDecoder(res.Body).Decode(out)
	return res
}

func putJSON(t *testing.T, url, token string, in, out any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(string(b)))
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	_ = json.NewDecoder(res.Body).Decode(out)
	return res
}

func TestUIPrefsRoundTrip(t *testing.T) {
	s, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	var p api.UIPrefs
	getJSON(t, ts.URL+"/api/ui-prefs", token, &p)
	if !slices.Equal(p.FleetColumns, api.DefaultPrefs().FleetColumns) {
		t.Fatalf("defaults %+v", p)
	}
	var e api.Error
	bad := api.DefaultPrefs()
	bad.FleetColumns = []string{"nope"}
	if res := putJSON(t, ts.URL+"/api/ui-prefs", token, bad, &e); res.StatusCode != http.StatusBadRequest || e.Code != api.CodeInvalidPrefs {
		t.Fatalf("invalid prefs: %d %+v", res.StatusCode, e)
	}
	good := api.DefaultPrefs()
	good.FleetColumns = []string{"host", "disk", "jobs"}
	good.RefreshSeconds = 30
	putJSON(t, ts.URL+"/api/ui-prefs", token, good, &p)
	getJSON(t, ts.URL+"/api/ui-prefs", token, &p)
	if !slices.Equal(p.FleetColumns, good.FleetColumns) || p.RefreshSeconds != 30 {
		t.Fatalf("stored %+v", p)
	}
	s.fleet.round(context.Background())
	if f := s.fleet.snapshot(); f.IntervalSeconds != 30 {
		t.Fatalf("fleet interval %d, want the preference", f.IntervalSeconds)
	}
}

func controllerServer(t *testing.T, recorded string, sig signer.Signer) (*httptest.Server, string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	if recorded != "" {
		if _, err := config.Update(func(c *config.Config) error {
			c.Controller = &config.Controller{KeyStore: "file", KeyRef: "x", Address: recorded}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	token := NewSessionToken()
	ts := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, Signer: sig}).Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func TestControllerView(t *testing.T) {
	k, _ := signer.GenerateKey()
	other, _ := signer.GenerateKey()
	for name, c := range map[string]struct {
		recorded string
		sig      signer.Signer
		state    string
	}{
		"missing":  {"", nil, "missing"},
		"unopened": {k.Address().Hex(), nil, "unopened"},
		"mismatch": {other.Address().Hex(), k, "mismatch"},
		"ok":       {k.Address().Hex(), k, "ok"},
	} {
		ts, token := controllerServer(t, c.recorded, c.sig)
		var v api.ControllerView
		getJSON(t, ts.URL+"/api/controller", token, &v)
		if v.State != c.state {
			t.Errorf("%s: state %q (%+v)", name, v.State, v)
		}
	}
}

func TestCheckSendsASignedAgentInfo(t *testing.T) {
	ts, token := pairedBox(t, nopExec{}, true)
	var c api.AgentCheck
	if res := postJSON(t, ts.URL+"/api/fleet/box/check", token, nil, &c); res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if !strings.HasPrefix(c.Agent, "0x") || !c.SetUp || c.ChainID != 369 || c.Signers != 1 {
		t.Fatalf("check %+v", c)
	}
}

func TestSSHCommand(t *testing.T) {
	ts, token := contractServer(t,
		config.Target{ID: "far", Mode: "ssh", SSH: &executor.SSHConfig{Host: "10.0.0.5", Port: 2222, User: "root", KeyPath: "/k/id"}},
		config.Target{ID: "here", Mode: "local"},
	)
	var c api.SSHCommand
	getJSON(t, ts.URL+"/api/fleet/far/ssh", token, &c)
	joined := strings.Join(c.Argv, " ")
	for _, want := range []string{"ssh ", "-p 2222", "-i /k/id", "-l root", "ProxyJump=none", "StrictHostKeyChecking=yes", "UserKnownHostsFile=", "confirmed_hosts", " 10.0.0.5"} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv %q lacks %q", joined, want)
		}
	}
	res, e := do(t, ts, token, "GET", "/api/fleet/here/ssh", "")
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeNoSSH {
		t.Fatalf("local target: %d %+v", res.StatusCode, e)
	}
}

// A stored block that is not valid never crashes a screen: it reads as the
// defaults. Unknown fields in a stored or sent block are ignored.
func TestLoadPrefsFallsBackAndIgnoresUnknownFields(t *testing.T) {
	def := api.DefaultPrefs()
	for name, raw := range map[string]string{
		"garbage":      `not json`,
		"wrong type":   `{"refreshSeconds":"fast"}`,
		"bad refresh":  `{"refreshSeconds":7}`,
		"bad column":   `{"fleetColumns":["nope"]}`,
		"null":         `null`,
		"empty object": `{}`,
	} {
		p := loadPrefs(config.Config{UI: []byte(raw)})
		if !slices.Equal(p.FleetColumns, def.FleetColumns) || p.RefreshSeconds != def.RefreshSeconds {
			t.Errorf("%s: %+v", name, p)
		}
	}
	p := loadPrefs(config.Config{UI: []byte(`{"refreshSeconds":30,"fromTheFuture":{"x":1}}`)})
	if p.RefreshSeconds != 30 {
		t.Fatalf("unknown field broke the block: %+v", p)
	}
	if loadPrefs(config.Config{}).RefreshSeconds != 15 {
		t.Fatal("no block is the defaults")
	}
}

func TestPutUIPrefsRejectsBadValuesAndKeepsTheStoredBlock(t *testing.T) {
	_, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	for name, body := range map[string]string{
		"not json":  `{`,
		"refresh":   `{"refreshSeconds":1}`,
		"units":     `{"units":"TB"}`,
		"glyphs":    `{"glyphs":"emoji"}`,
		"no host":   `{"fleetColumns":["disk"]}`,
		"chain":     `{"chains":[0]}`,
		"dup tab":   `{"hostTabs":["logs","logs"]}`,
		"many":      `{"chains":[` + strings.Repeat("1,", 100) + `1]}`,
		"bad tab":   `{"hostTabs":["nope"]}`,
		"empty col": `{"fleetColumns":[]}`,
	} {
		res, e := do(t, ts, token, "PUT", "/api/ui-prefs", body)
		if res.StatusCode != http.StatusBadRequest || e.Code != api.CodeInvalidPrefs {
			t.Errorf("%s: %d %+v", name, res.StatusCode, e)
		}
	}
	// Unknown fields on PUT are ignored.
	res, _ := do(t, ts, token, "PUT", "/api/ui-prefs", `{"refreshSeconds":60,"extra":true}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("unknown field: %d", res.StatusCode)
	}
	var p api.UIPrefs
	getJSON(t, ts.URL+"/api/ui-prefs", token, &p)
	if p.RefreshSeconds != 60 {
		t.Fatalf("stored %+v", p)
	}
}

// A preference change restarts the poller's wait with the new interval and
// leaves no goroutine or timer behind once the poller stops.
func TestPrefsChangeRetimesTheRunningPoller(t *testing.T) {
	s, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	_ = fp
	s.fleet.want()
	deadline := time.Now().Add(3 * time.Second)
	for s.fleet.snapshot().IntervalSeconds == 0 || len(s.fleet.snapshot().Rows) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("poller never ran a round")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := s.fleet.snapshot().IntervalSeconds; got != 15 {
		t.Fatalf("interval %d before the change", got)
	}
	good := api.DefaultPrefs()
	good.RefreshSeconds = 60
	putJSON(t, ts.URL+"/api/ui-prefs", token, good, &api.UIPrefs{})
	for {
		if s.fleet.snapshot().IntervalSeconds == 60 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("interval %d, the running poller did not follow the preference", s.fleet.snapshot().IntervalSeconds)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if s.fleet.maxActiveLoops() != 1 {
		t.Fatalf("%d loops ran at once", s.fleet.maxActiveLoops())
	}
	s.fleet.stop()
	requireNoFleetGoroutines(t)
}

// A signer is loaded but the controller was never recorded: not a mismatch.
func TestControllerViewUnrecorded(t *testing.T) {
	k, _ := signer.GenerateKey()
	ts, token := controllerServer(t, "", k)
	var v api.ControllerView
	getJSON(t, ts.URL+"/api/controller", token, &v)
	if v.State != "unrecorded" || v.Address != k.Address().Hex() || v.Recorded != "" || !strings.Contains(v.Reason, "restore config.json") || strings.Contains(v.Reason, "to record it") {
		t.Fatalf("%+v", v)
	}
}
