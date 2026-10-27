// Package vaulttest gives tests a vault without running the KDF in each one:
// a fixture is created once per process and copied into each test's
// directory.
package vaulttest

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// Passphrase unlocks every fixture vault.
const Passphrase = "correct horse battery staple"

var fixture = sync.OnceValues(func() ([]byte, error) {
	dir, err := os.MkdirTemp("", "vaulttest-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	p := filepath.Join(dir, "vault")
	v, err := vault.Create(p, []byte(Passphrase), vault.DefaultParams)
	if err != nil {
		return nil, err
	}
	v.Lock()
	return os.ReadFile(p)
})

// Path writes a locked fixture vault into t.TempDir() and returns its path
// and a fresh copy of the passphrase.
func Path(t testing.TB) (path string, passphrase []byte) {
	t.Helper()
	raw, err := fixture()
	if err != nil {
		t.Fatalf("vaulttest: %v", err)
	}
	path = filepath.Join(t.TempDir(), "vault")
	if err := fsperm.WriteFilePrivate(path, bytes.Clone(raw)); err != nil {
		t.Fatalf("vaulttest: %v", err)
	}
	return path, []byte(Passphrase)
}

// Unlocked returns an unlocked fixture vault.
func Unlocked(t testing.TB) *vault.Vault {
	t.Helper()
	p, pass := Path(t)
	v, err := vault.Open(p)
	if err != nil {
		t.Fatalf("vaulttest: %v", err)
	}
	if err := v.Unlock(pass); err != nil {
		t.Fatalf("vaulttest: %v", err)
	}
	t.Cleanup(v.Lock)
	return v
}
