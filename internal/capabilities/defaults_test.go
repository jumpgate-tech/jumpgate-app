package capabilities

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// A zero-valued Prober has to work: it is what `var p Prober` gives, and what
// a struct literal setting only ProbeWS gives. A concurrency of zero would
// mean "no worker may run", which deadlocks rather than failing — which is
// the whole reason these accessors exist rather than reading the fields.
func TestProber_ZeroValueFallsBackToDefaults(t *testing.T) {
	var p Prober

	if got := p.httpClient(); got != http.DefaultClient {
		t.Errorf("httpClient = %v, want http.DefaultClient", got)
	}
	if got := p.probeTimeout(); got != defaultProbeTimeout {
		t.Errorf("probeTimeout = %v, want %v", got, defaultProbeTimeout)
	}
	if got := p.concurrency(); got < 1 {
		t.Fatalf("concurrency = %d — below 1 cannot make progress", got)
	}
	if got := p.dialer(); got == nil {
		t.Error("dialer = nil, want a usable dialer")
	}
}

func TestProber_ExplicitSettingsWin(t *testing.T) {
	client := &http.Client{Timeout: time.Second}
	p := Prober{HTTPClient: client, ProbeTimeout: 4 * time.Second, Concurrency: 3}

	if p.httpClient() != client {
		t.Error("the injected http client was ignored")
	}
	if got := p.probeTimeout(); got != 4*time.Second {
		t.Errorf("probeTimeout = %v, want 4s", got)
	}
	if got := p.concurrency(); got != 3 {
		t.Errorf("concurrency = %d, want 3", got)
	}
}

func TestNewProber_IsUsableWithoutFurtherSetup(t *testing.T) {
	p := NewProber()
	if p.httpClient() == nil || p.probeTimeout() <= 0 || p.concurrency() < 1 {
		t.Fatalf("NewProber returned something unusable: %+v", p)
	}
}

// ---------------------------------------------------------------------
// the valve.city client
// ---------------------------------------------------------------------

func TestClient_ZeroValueFallsBackToDefaults(t *testing.T) {
	var c Client

	if got := c.httpClient(); got != http.DefaultClient {
		t.Errorf("httpClient = %v, want http.DefaultClient", got)
	}
	if got := c.baseURL(); got != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", got, DefaultBaseURL)
	}
}

// A trailing slash is trimmed, because the paths appended to this are written
// with a leading one — "…/api//check" is a 404 on most routers.
func TestClient_BaseURLTrimsATrailingSlash(t *testing.T) {
	c := Client{BaseURL: "https://valve.example/api/"}
	got := c.baseURL()
	if strings.HasSuffix(got, "/") {
		t.Errorf("baseURL = %q, want the trailing slash trimmed", got)
	}
	if got != "https://valve.example/api" {
		t.Errorf("baseURL = %q, want the configured base", got)
	}
}

func TestNewClient_PointsAtValveCity(t *testing.T) {
	c := NewClient()
	if c.httpClient() == nil {
		t.Error("no http client")
	}
	if c.baseURL() != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL(), DefaultBaseURL)
	}
}

// NewGatherer must come with BOTH sources wired. One nil source is not a
// crash — Gather checks — but it silently halves the answer, so the
// constructor is the place that has to get it right.
func TestNewGatherer_WiresBothSources(t *testing.T) {
	g := NewGatherer()
	if g.Client == nil {
		t.Error("no valve.city client, so the published matrix is never fetched")
	}
	if g.Prober == nil {
		t.Error("no prober, so the local endpoints are never checked")
	}
}

// ---------------------------------------------------------------------
// Label
// ---------------------------------------------------------------------

// An unknown key falls back to itself so a capability a NEWER valve.city
// knows about still renders, instead of showing a blank cell.
func TestLabel_FallsBackToTheKeyItself(t *testing.T) {
	if got := Label("a-capability-invented-tomorrow"); got != "a-capability-invented-tomorrow" {
		t.Errorf("got %q, want the key itself", got)
	}
	if got := Label(""); got != "" {
		t.Errorf("got %q, want the empty key back", got)
	}

	// A known key resolves to something friendlier than the key.
	if got := Label(KeyTrace); got == "" {
		t.Error("a known capability rendered as nothing")
	}
}

func TestHelp_IsEmptyForAnUnknownKey(t *testing.T) {
	if got := Help("a-capability-invented-tomorrow"); got != "" {
		t.Errorf("got %q, want nothing for an unknown key", got)
	}
	if got := Help(KeyTrace); got == "" {
		t.Error("a known capability has no how-it-is-tested text")
	}
}
