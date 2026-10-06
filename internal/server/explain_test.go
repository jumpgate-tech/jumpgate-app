package server

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestExplainRedactsForRemoteProviders(t *testing.T) {
	a := newAPITestServer(t)
	addTarget(t, a)
	setProvider(t, a, "gemini")
	res := a.do(t, "POST", "/api/targets/local/explain", map[string]any{"lines": []string{"peer 203.0.113.7 timed out"}})
	out := decode[api.Explain](t, res)
	want := "peer <ip-1> timed out"
	if !out.Redacted || len(out.SentExcerpt) != 1 || out.SentExcerpt[0] != want ||
		!slices.Equal(a.fakeAI.lastReq.Lines, []string{want}) {
		t.Fatalf("response %+v, provider got %q", out, a.fakeAI.lastReq.Lines)
	}
}

func TestExplainSendsLocalProvidersTheRawLines(t *testing.T) {
	a := newAPITestServer(t)
	addTarget(t, a)
	res := a.do(t, "PUT", "/api/settings", map[string]any{"aiProvider": "ollama"})
	res.Body.Close()
	res = a.do(t, "POST", "/api/targets/local/explain", map[string]any{"lines": []string{"peer 203.0.113.7 timed out"}})
	out := decode[api.Explain](t, res)
	if out.Redacted || a.fakeAI.lastReq.Lines[0] != "peer 203.0.113.7 timed out" {
		t.Fatalf("response %+v", out)
	}
}

// A line with nothing to hide is sent as it is and is not reported redacted.
func TestExplainReportsRedactedOnlyWhenALineChanged(t *testing.T) {
	a := newAPITestServer(t)
	addTarget(t, a)
	setProvider(t, a, "groq")
	res := a.do(t, "POST", "/api/targets/local/explain", map[string]any{"lines": []string{"reth::cli started"}})
	if out := decode[api.Explain](t, res); out.Redacted {
		t.Fatalf("%+v", out)
	}
}

// Redact caps its output, so it can return fewer lines than it got. The
// excerpt is what was sent, however many lines that is.
func TestExplainSentExcerptIsWhatTheProviderGot(t *testing.T) {
	a := newAPITestServer(t)
	addTarget(t, a)
	setProvider(t, a, "groq")
	in := make([]string, 2500)
	for i := range in {
		in[i] = "line from 203.0.113.7"
	}
	res := a.do(t, "POST", "/api/targets/local/explain", map[string]any{"lines": in})
	out := decode[api.Explain](t, res)
	if !out.Redacted || len(out.SentExcerpt) >= len(in) || !slices.Equal(out.SentExcerpt, a.fakeAI.lastReq.Lines) {
		t.Fatalf("sent %d lines, provider got %d", len(out.SentExcerpt), len(a.fakeAI.lastReq.Lines))
	}
	for _, l := range a.fakeAI.lastReq.Lines {
		if strings.Contains(l, "203.0.113.7") {
			t.Fatalf("an address reached the provider: %q", l)
		}
	}
}

func TestExplainWithoutAProviderIsAIUnconfigured(t *testing.T) {
	ts, token := contractServer(t, config.Target{ID: "local", Mode: "local"})
	res, e := do(t, ts, token, "POST", "/api/targets/local/explain", "{}")
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeAIUnconfigured {
		t.Fatalf("%d %+v", res.StatusCode, e)
	}
}

func TestSettingsCarryTheDisclosureAndNeverTheKey(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "PUT", "/api/settings", map[string]any{"aiProvider": "groq", "aiKey": "sk-secret-key"})
	res.Body.Close()
	res = a.do(t, "GET", "/api/settings", nil)
	defer res.Body.Close()
	var raw map[string]any
	_ = json.NewDecoder(res.Body).Decode(&raw)
	b, _ := json.Marshal(raw)
	if strings.Contains(string(b), "sk-secret-key") {
		t.Fatalf("the key came back: %s", b)
	}
	var s api.Settings
	_ = json.Unmarshal(b, &s)
	if s.AIDisclosure == "" || s.AIProvider != "groq" || !s.AIKeySet {
		t.Fatalf("settings %+v", s)
	}
}

// agentErrorExec answers the agent's journal with an error line that names a
// peer address, so a default explain has something to read and to redact.
type agentErrorExec struct{ recordingExec }

func (e *agentErrorExec) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	if strings.Contains(cmd, "-o cat") && strings.Contains(cmd, "-exec") {
		_, _ = e.recordingExec.Run(ctx, cmd, o)
		return executor.Result{Stdout: "ERROR peer 203.0.113.7 dropped us\nINFO all fine\n"}, nil
	}
	return e.recordingExec.Run(ctx, cmd, o)
}

// A paired box's default lines come from its agent, not from root SSH, and
// are redacted before the provider sees them.
func TestExplainReadsAPairedBoxThroughItsAgent(t *testing.T) {
	requireAgentPeer(t)
	rec := &agentErrorExec{}
	fake := &fakeAIProvider{text: "because"}
	ts, token := pairedBoxWith(t, rec, pairedOpts{setUp: true, ai: fake})
	if _, err := config.Update(func(c *config.Config) error { c.AIProvider = "gemini"; c.AIKey = "k"; return nil }); err != nil {
		t.Fatal(err)
	}
	res, e := do(t, ts, token, "POST", "/api/targets/box/explain", "{}")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", res.StatusCode, e)
	}
	if !rec.saw("journalctl", "-o cat") {
		t.Fatalf("the agent never read the journal; commands %q", rec.cmds)
	}
	if len(fake.lastReq.Lines) != 1 || fake.lastReq.Lines[0] != "ERROR peer <ip-1> dropped us" {
		t.Fatalf("the provider got %q", fake.lastReq.Lines)
	}
}

// recordingExec is journalExec that remembers what it ran.
type recordingExec struct {
	journalExec
	mu   sync.Mutex
	cmds []string
}

func (r *recordingExec) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	r.mu.Lock()
	r.cmds = append(r.cmds, cmd)
	r.mu.Unlock()
	return r.journalExec.Run(ctx, cmd, o)
}

func (r *recordingExec) saw(parts ...string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.cmds {
		all := true
		for _, p := range parts {
			all = all && strings.Contains(c, p)
		}
		if all {
			return true
		}
	}
	return false
}
