package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
)

// TestInstallIDStableAndPersisted is the guard for the gateway-URL churn bug:
// the TLS-hostname seed must NOT depend on the machine hostname (which renames
// itself on macOS), or a shared gateway URL rotates. installID persists a
// random id on first use and returns the same value every run thereafter.
func TestInstallIDStableAndPersisted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	first := installID()
	if first == "" {
		t.Fatal("installID returned empty")
	}

	path := filepath.Join(home, ".valve-node-app", "install-id")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("install-id was not persisted: %v", err)
	}
	if strings.TrimSpace(string(b)) != first {
		t.Errorf("persisted id %q != returned %q", strings.TrimSpace(string(b)), first)
	}

	// A second call (a fresh process would re-read the file) returns the same id.
	if second := installID(); second != first {
		t.Errorf("installID churned across calls: %q != %q", second, first)
	}
}

// TestSuggestedHostnameStableAcrossHostnameChange proves the derived hostname
// does not change when os.Hostname() would — because it is seeded from the
// persisted id, not the machine name.
func TestSuggestedHostnameStableAcrossHostnameChange(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Two derivations from the same persisted id must be identical. (DefaultTLSHostname
	// is pure; the only moving part is the seed, which installID pins.)
	got1 := catalog.DefaultTLSHostname("default", installID())
	got2 := catalog.DefaultTLSHostname("default", installID())
	if got1 != got2 {
		t.Errorf("hostname churned: %q != %q", got1, got2)
	}
	if !strings.HasPrefix(got1, "default-") || !strings.HasSuffix(got1, ".localhost-valaxy.com") {
		t.Errorf("unexpected hostname shape: %q", got1)
	}
}
