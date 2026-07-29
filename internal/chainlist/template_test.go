package chainlist

import "testing"

// The feed writes provider slots as ${NAME}. The NAME is the identity: it is
// what the operator has to go and obtain, so it is what the app stores a key
// under and what a rejection has to say out loud.
func TestPlaceholderName(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"https://mainnet.infura.io/v3/${INFURA_API_KEY}", "INFURA_API_KEY"},
		{"wss://mainnet.infura.io/ws/v3/${INFURA_API_KEY}", "INFURA_API_KEY"},
		{"https://one.valve.city/rpc/${VALVE_API_KEY}/evm/1", "VALVE_API_KEY"},
		{"https://eth.drpc.org", ""},
		{"", ""},
		// Malformed: an opening brace with no close is not a usable name.
		{"https://x.example/${UNCLOSED", ""},
	}
	for _, tt := range tests {
		if got := PlaceholderName(tt.raw); got != tt.want {
			t.Errorf("PlaceholderName(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestResolve(t *testing.T) {
	keys := map[string]string{"INFURA_API_KEY": "abc123"}

	got, ok := Resolve("https://mainnet.infura.io/v3/${INFURA_API_KEY}", keys)
	if !ok || got != "https://mainnet.infura.io/v3/abc123" {
		t.Errorf("resolved = %q, %v", got, ok)
	}

	// No key for this placeholder: the URL is unusable and must NOT come back
	// half-substituted, which would be a live-looking URL that 401s forever.
	if got, ok := Resolve("https://x.example/${ALCHEMY_API_KEY}", keys); ok {
		t.Errorf("want unresolved, got %q", got)
	}

	// An untemplated URL resolves to itself, so callers need no special case.
	if got, ok := Resolve("https://eth.drpc.org", keys); !ok || got != "https://eth.drpc.org" {
		t.Errorf("plain URL: got %q, %v", got, ok)
	}

	// An empty stored value is not a key. Substituting it yields a URL with an
	// empty path segment that looks configured and answers nothing.
	if _, ok := Resolve("https://x.example/${K}", map[string]string{"K": ""}); ok {
		t.Error("an empty key must not resolve")
	}
}
