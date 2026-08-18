package relay

import (
	"strings"
	"testing"
)

// Build turns startup configuration into a data-plane handler. The interesting
// case is the half-configured one: a relay that quietly disabled itself would
// serve every customer's traffic for free and report nothing.

func TestBuildDisabledWhenNoBind(t *testing.T) {
	h, err := Build(BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if h != nil {
		t.Error("handler is non-nil with no relay bind — the relay must be off by default")
	}
}

func TestBuildSucceedsWhenFullyConfigured(t *testing.T) {
	h, err := Build(BuildOptions{
		RelayBind:     "127.0.0.1:8790",
		BillingSocket: "/run/jumpgate/billing.sock",
		RelayToken:    "relay-token",
		ERPCURL:       "http://127.0.0.1:4000",
		ProjectID:     "main",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if h == nil {
		t.Fatal("handler is nil with a complete configuration")
	}
}

// Each missing piece is a hard failure. Serving unmetered traffic is worse than
// not serving it: the operator sells access and would be giving it away, and
// nothing in the request path would report the mistake.
func TestBuildRefusesAHalfConfiguredRelay(t *testing.T) {
	full := BuildOptions{
		RelayBind:     "127.0.0.1:8790",
		BillingSocket: "/run/jumpgate/billing.sock",
		RelayToken:    "relay-token",
		ERPCURL:       "http://127.0.0.1:4000",
		ProjectID:     "main",
	}

	tests := []struct {
		name    string
		mutate  func(*BuildOptions)
		wantMsg string
	}{
		{"no billing socket", func(o *BuildOptions) { o.BillingSocket = "" }, "socket"},
		{"no relay token", func(o *BuildOptions) { o.RelayToken = "" }, "token"},
		{"no eRPC url", func(o *BuildOptions) { o.ERPCURL = "" }, "erpc"},
	}
	for _, tt := range tests {
		opt := full
		tt.mutate(&opt)

		h, err := Build(opt)
		if err == nil {
			t.Errorf("%s: err = nil, want a refusal", tt.name)
			continue
		}
		if h != nil {
			t.Errorf("%s: handler is non-nil despite the error", tt.name)
		}
		if !strings.Contains(strings.ToLower(err.Error()), tt.wantMsg) {
			t.Errorf("%s: err = %q, want it to name the missing piece (%q)", tt.name, err, tt.wantMsg)
		}
	}
}

// An unusable eRPC URL is a startup failure, not a run-time surprise on the
// first customer request.
func TestBuildRejectsAMalformedERPCURL(t *testing.T) {
	_, err := Build(BuildOptions{
		RelayBind:     "127.0.0.1:8790",
		BillingSocket: "/run/jumpgate/billing.sock",
		RelayToken:    "relay-token",
		ERPCURL:       "://not a url",
		ProjectID:     "main",
	})
	if err == nil {
		t.Fatal("err = nil, want a refusal for a malformed URL")
	}
}

// The project id defaults rather than failing. eRPC's own default is "main",
// and an operator who never set one should not be blocked by it.
func TestBuildDefaultsTheProjectID(t *testing.T) {
	h, err := Build(BuildOptions{
		RelayBind:     "127.0.0.1:8790",
		BillingSocket: "/run/jumpgate/billing.sock",
		RelayToken:    "relay-token",
		ERPCURL:       "http://127.0.0.1:4000",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if h == nil {
		t.Fatal("handler is nil")
	}
}

func TestBuildAdminDisabledWithoutASocket(t *testing.T) {
	c, err := BuildAdmin("", "admin-token")
	if err != nil {
		t.Fatalf("BuildAdmin: %v", err)
	}
	if c != nil {
		t.Error("client is non-nil with no billing socket")
	}
}

// A socket with no admin token is a misconfiguration, not a quiet disable. An
// operator who configured a store expects to manage keys with it.
func TestBuildAdminRefusesASocketWithoutAToken(t *testing.T) {
	c, err := BuildAdmin("/run/jumpgate/billing.sock", "")
	if err == nil {
		t.Fatal("err = nil, want a refusal")
	}
	if c != nil {
		t.Error("client is non-nil despite the error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "token") {
		t.Errorf("err = %q, want it to name the missing token", err)
	}
}

func TestBuildAdminSucceeds(t *testing.T) {
	c, err := BuildAdmin("/run/jumpgate/billing.sock", "admin-token")
	if err != nil {
		t.Fatalf("BuildAdmin: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil with a complete configuration")
	}
}
