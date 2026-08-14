package buildinfo

import "testing"

// TestDefaults pins the values a plain `go build` produces. The release build
// overrides them with -ldflags -X; this guards the un-injected defaults the
// update check depends on ("dev" must stay the sentinel it keys off).
func TestDefaults(t *testing.T) {
	if got := Version(); got != "dev" {
		t.Errorf("Version() = %q, want %q for an un-injected build", got, "dev")
	}
	if got := ReleaseRepo(); got != "jumpgate-tech/jumpgate-app" {
		t.Errorf("ReleaseRepo() = %q, want %q", got, "jumpgate-tech/jumpgate-app")
	}
}
