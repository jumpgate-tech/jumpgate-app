//go:build windows

package signer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

// Against the real Credential Manager of the CI runner's user, so it runs only
// where CI sets JUMPGATE_TEST_REAL_WINCRED=1: `go test` on a developer's
// machine never writes their Credential Manager. The ref is unique per run,
// and the credential is deleted afterwards.
func TestRealWinCred(t *testing.T) {
	if os.Getenv("JUMPGATE_TEST_REAL_WINCRED") != "1" {
		t.Skip("writes the real Credential Manager; set JUMPGATE_TEST_REAL_WINCRED=1 (CI only)")
	}
	ref := fmt.Sprintf("jumpgate-ci-%d-%d", os.Getpid(), time.Now().UnixNano())
	target := credTarget(ref)
	t.Cleanup(func() {
		if err := winCreds.del(target); err != nil {
			t.Errorf("remove %s: %v", target, err)
		}
	})
	if _, err := winCreds.read(target); !errors.Is(err, errCredNotFound) {
		t.Fatalf("read of a missing credential = %v, want errCredNotFound", err)
	}
	k, err := Create(context.Background(), StoreWinCred, ref)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := Open(context.Background(), StoreWinCred, ref)
	if err != nil || k2.Address() != k.Address() {
		t.Fatalf("Open = %v, %v; want %s", k2, err, k.Address().Hex())
	}
	if _, err := Create(context.Background(), StoreWinCred, ref); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("second Create = %v, want ErrKeyExists", err)
	}
	if k3, err := Open(context.Background(), StoreWinCred, ref); err != nil || k3.Address() != k.Address() {
		t.Fatalf("the refused Create changed the stored key: %v, %v", k3, err)
	}
	if DefaultStore() != StoreWinCred {
		t.Fatalf("DefaultStore() = %q on Windows", DefaultStore())
	}
	if err := winCreds.del(target); err != nil {
		t.Fatal(err)
	}
	if _, err := winCreds.read(target); !errors.Is(err, errCredNotFound) {
		t.Fatalf("read after delete = %v, want errCredNotFound", err)
	}
}
