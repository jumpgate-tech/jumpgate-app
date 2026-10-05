//go:build realsecret && linux

package signer

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// These tests drive the REAL secret-tool against a running Secret Service
// (see the container recipe in .superpowers/sdd/followups/secret-tool-report.md).
// Run: go test -tags realsecret -run RealSecret -v ./internal/signer/

func realRef(t *testing.T) string {
	t.Helper()
	ref := fmt.Sprintf("realsecret-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = exec.Command("secret-tool", "clear", "service", keychainService, "account", ref).Run()
	})
	return ref
}

func TestRealSecretRoundTrip(t *testing.T) {
	ctx := context.Background()
	ref := realRef(t)
	if DefaultStore() != StoreKeychain {
		t.Fatalf("DefaultStore = %v with secret-tool on PATH", DefaultStore())
	}
	_, err := Open(ctx, StoreKeychain, ref)
	if err == nil || !keychainNotFoundMsg(err) {
		t.Fatalf("Open of a missing item = %v, want a not-found error", err)
	}
	k, err := Create(ctx, StoreKeychain, ref)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Open(ctx, StoreKeychain, ref)
	if err != nil || got.Address() != k.Address() {
		t.Fatalf("Open = %v, %v", got, err)
	}
	if _, err := Create(ctx, StoreKeychain, ref); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("second Create = %v, want ErrKeyExists", err)
	}
}

func keychainNotFoundMsg(err error) bool { return strings.Contains(err.Error(), "no item") }

// TestRealSecretLockedKeyringIsNotAbsence locks the login collection with
// busctl; a locked keyring answers `lookup` with exit 1 and empty output, the
// same as a missing item, so Create must not treat that as absent.
func TestRealSecretLockedKeyringIsNotAbsence(t *testing.T) {
	if _, err := exec.LookPath("busctl"); err != nil {
		t.Skip("busctl not available")
	}
	ctx := context.Background()
	ref := realRef(t)
	k, err := Create(ctx, StoreKeychain, ref)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("busctl", "--user", "call", "org.freedesktop.secrets", "/org/freedesktop/secrets",
		"org.freedesktop.Secret.Service", "Lock", "ao", "1", "/org/freedesktop/secrets/collection/login").CombinedOutput(); err != nil {
		t.Skipf("cannot lock the keyring: %v %s", err, out)
	}
	if _, err := Create(ctx, StoreKeychain, ref); err == nil {
		t.Fatal("Create over an existing item in a locked keyring succeeded")
	}
	_, err = Open(ctx, StoreKeychain, ref)
	if err == nil || keychainNotFoundMsg(err) || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("Open in a locked keyring = %v, want a locked error, not not-found", err)
	}
	_ = k
}
