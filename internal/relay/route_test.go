package relay

import (
	"errors"
	"testing"
)

// The relay's grammar is /<category>/<key>/<arch>/<chainId>[/...]. Category and
// arch are different dimensions: category names the kind of API, arch names the
// chain family, and every category carries an arch slot. These tests pin the
// grammar before any handler exists, because every later stage reads the parse
// result and a wrong parse reaches an upstream with a key still attached.

func TestParseRPCPath(t *testing.T) {
	got, err := ParseRoute("/rpc/jg_abc123/evm/369")
	if err != nil {
		t.Fatalf("ParseRoute: %v", err)
	}
	if got.Category != CategoryRPC {
		t.Errorf("category = %q, want %q", got.Category, CategoryRPC)
	}
	if got.Key != "jg_abc123" {
		t.Errorf("key = %q, want jg_abc123", got.Key)
	}
	if got.Arch != "evm" {
		t.Errorf("arch = %q, want evm", got.Arch)
	}
	if got.ChainID != 369 {
		t.Errorf("chainID = %d, want 369", got.ChainID)
	}
	if got.Rest != "" {
		t.Errorf("rest = %q, want empty for rpc", got.Rest)
	}
}

// /rpc is fixed depth. eRPC addresses a chain by the whole path, so an extra
// segment is not a sub-resource — it is a malformed request that must never
// reach eRPC.
func TestParseRPCRejectsExtraSegments(t *testing.T) {
	_, err := ParseRoute("/rpc/jg_abc123/evm/369/extra")
	if !errors.Is(err, ErrBadShape) {
		t.Fatalf("err = %v, want ErrBadShape", err)
	}
}

// /beacon is prefix plus tree. The beacon API is a REST tree under /eth/..., so
// a remainder is the normal case and the relay carries it through.
func TestParseBeaconKeepsRemainder(t *testing.T) {
	got, err := ParseRoute("/beacon/jg_abc123/evm/369/eth/v1/beacon/genesis")
	if err != nil {
		t.Fatalf("ParseRoute: %v", err)
	}
	if got.Category != CategoryBeacon {
		t.Errorf("category = %q, want %q", got.Category, CategoryBeacon)
	}
	if got.ChainID != 369 {
		t.Errorf("chainID = %d, want 369", got.ChainID)
	}
	if got.Rest != "/eth/v1/beacon/genesis" {
		t.Errorf("rest = %q, want /eth/v1/beacon/genesis", got.Rest)
	}
}

// A beacon call with no remainder is still a valid route. It addresses the
// beacon client's root, and the upstream decides what that means.
func TestParseBeaconWithoutRemainder(t *testing.T) {
	got, err := ParseRoute("/beacon/jg_abc123/evm/369")
	if err != nil {
		t.Fatalf("ParseRoute: %v", err)
	}
	if got.Rest != "" {
		t.Errorf("rest = %q, want empty", got.Rest)
	}
}

// Health filters the category x arch x chain matrix. Each level pins one more
// dimension, so the depth is the answer shape.
func TestParseHealthDepths(t *testing.T) {
	tests := []struct {
		path     string
		wantKey  string
		wantSel  string
		wantChan int
		wantDep  HealthDepth
	}{
		{"/health", "", "", 0, HealthLiveness},
		{"/health/jg_abc123", "jg_abc123", "", 0, HealthAll},
		{"/health/jg_abc123/evm", "jg_abc123", "evm", 0, HealthSelector},
		{"/health/jg_abc123/beacon", "jg_abc123", "beacon", 0, HealthSelector},
		{"/health/jg_abc123/evm/1", "jg_abc123", "evm", 1, HealthCell},
	}
	for _, tt := range tests {
		got, err := ParseRoute(tt.path)
		if err != nil {
			t.Errorf("ParseRoute(%q): %v", tt.path, err)
			continue
		}
		if got.Category != CategoryHealth {
			t.Errorf("ParseRoute(%q) category = %q, want health", tt.path, got.Category)
		}
		if got.Key != tt.wantKey {
			t.Errorf("ParseRoute(%q) key = %q, want %q", tt.path, got.Key, tt.wantKey)
		}
		if got.Sel != tt.wantSel {
			t.Errorf("ParseRoute(%q) sel = %q, want %q", tt.path, got.Sel, tt.wantSel)
		}
		if got.ChainID != tt.wantChan {
			t.Errorf("ParseRoute(%q) chainID = %d, want %d", tt.path, got.ChainID, tt.wantChan)
		}
		if got.Depth != tt.wantDep {
			t.Errorf("ParseRoute(%q) depth = %v, want %v", tt.path, got.Depth, tt.wantDep)
		}
	}
}

// The unkeyed liveness level carries no key, so nothing downstream may treat it
// as authenticated.
func TestParseHealthLivenessNeedsNoKey(t *testing.T) {
	got, err := ParseRoute("/health")
	if err != nil {
		t.Fatalf("ParseRoute: %v", err)
	}
	if got.Keyed() {
		t.Error("Keyed() = true for /health, want false")
	}
}

func TestParseRouteRejects(t *testing.T) {
	tests := []struct {
		name string
		path string
		want error
	}{
		{"unknown category", "/nope/jg_abc/evm/1", ErrNotFound},
		{"bare slash", "/", ErrNotFound},
		{"empty", "", ErrNotFound},
		{"rpc missing key", "/rpc", ErrBadShape},
		{"rpc missing arch", "/rpc/jg_abc", ErrBadShape},
		{"rpc missing chain", "/rpc/jg_abc/evm", ErrBadShape},
		{"rpc empty key", "/rpc//evm/1", ErrBadShape},
		{"rpc non-numeric chain", "/rpc/jg_abc/evm/mainnet", ErrBadShape},
		{"rpc negative chain", "/rpc/jg_abc/evm/-1", ErrBadShape},
		{"beacon missing chain", "/beacon/jg_abc/evm", ErrBadShape},
		{"health too deep", "/health/jg_abc/evm/1/extra", ErrBadShape},
		{"health non-numeric chain", "/health/jg_abc/evm/mainnet", ErrBadShape},
	}
	for _, tt := range tests {
		_, err := ParseRoute(tt.path)
		if !errors.Is(err, tt.want) {
			t.Errorf("ParseRoute(%q) err = %v, want %v", tt.path, err, tt.want)
		}
	}
}

// v1 serves evm only. An arch the relay understands but does not yet serve is a
// 501, which is a different fact from a malformed path and must stay different.
func TestParseRouteUnsupportedArch(t *testing.T) {
	for _, path := range []string{"/rpc/jg_abc/svm/1", "/beacon/jg_abc/btc/1"} {
		_, err := ParseRoute(path)
		if !errors.Is(err, ErrUnsupportedArch) {
			t.Errorf("ParseRoute(%q) err = %v, want ErrUnsupportedArch", path, err)
		}
	}
}

// An arch nobody has ever defined is a bad shape, not an unimplemented feature.
func TestParseRouteUnknownArchIsBadShape(t *testing.T) {
	_, err := ParseRoute("/rpc/jg_abc/quantum/1")
	if !errors.Is(err, ErrBadShape) {
		t.Fatalf("err = %v, want ErrBadShape", err)
	}
}

// A trailing slash is the same route. Callers append one by habit and a 400
// there would be a support ticket, not a defence.
func TestParseRouteToleratesTrailingSlash(t *testing.T) {
	got, err := ParseRoute("/rpc/jg_abc123/evm/369/")
	if err != nil {
		t.Fatalf("ParseRoute: %v", err)
	}
	if got.ChainID != 369 {
		t.Errorf("chainID = %d, want 369", got.ChainID)
	}
}

// A url-encoded segment decodes before it is read. A key that survives encoding
// unchanged must compare equal to the raw one, or a customer's working URL
// stops working the moment a client library escapes it.
func TestParseRouteDecodesEscapedSegments(t *testing.T) {
	got, err := ParseRoute("/rpc/jg%5Fabc123/evm/369")
	if err != nil {
		t.Fatalf("ParseRoute: %v", err)
	}
	if got.Key != "jg_abc123" {
		t.Errorf("key = %q, want jg_abc123", got.Key)
	}
}

// An encoded slash must not smuggle an extra segment past the depth check.
func TestParseRouteRejectsEncodedSlashInKey(t *testing.T) {
	_, err := ParseRoute("/rpc/jg%2Fabc/evm/1")
	if !errors.Is(err, ErrBadShape) {
		t.Fatalf("err = %v, want ErrBadShape", err)
	}
}

// The forward path is what eRPC actually serves, built from the parse result so
// the relay and the keyless path cannot drift.
func TestRouteUpstreamPathForRPC(t *testing.T) {
	got, err := ParseRoute("/rpc/jg_abc123/evm/369")
	if err != nil {
		t.Fatalf("ParseRoute: %v", err)
	}
	if want := "/main/evm/369"; got.UpstreamPath("main") != want {
		t.Errorf("UpstreamPath = %q, want %q", got.UpstreamPath("main"), want)
	}
}

// The beacon client sees its own native tree, with the whole relay prefix gone.
func TestRouteUpstreamPathForBeacon(t *testing.T) {
	got, err := ParseRoute("/beacon/jg_abc123/evm/369/eth/v1/beacon/genesis")
	if err != nil {
		t.Fatalf("ParseRoute: %v", err)
	}
	if want := "/eth/v1/beacon/genesis"; got.UpstreamPath("main") != want {
		t.Errorf("UpstreamPath = %q, want %q", got.UpstreamPath("main"), want)
	}
}

// The load-bearing property of the whole slice: no upstream path may contain
// the key, whatever the category.
func TestUpstreamPathNeverCarriesTheKey(t *testing.T) {
	paths := []string{
		"/rpc/jg_secret/evm/369",
		"/beacon/jg_secret/evm/369/eth/v1/beacon/genesis",
		"/beacon/jg_secret/evm/369",
	}
	for _, p := range paths {
		got, err := ParseRoute(p)
		if err != nil {
			t.Errorf("ParseRoute(%q): %v", p, err)
			continue
		}
		up := got.UpstreamPath("main")
		if contains(up, "jg_secret") {
			t.Errorf("ParseRoute(%q).UpstreamPath = %q, still carries the key", p, up)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
