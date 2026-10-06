//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A token file must be private: group or other bits are refused, with the
// file named and the chmod that fixes it.
func TestTokenFileWithGroupOrOtherBitsIsRefused(t *testing.T) {
	f := filepath.Join(t.TempDir(), "admin")
	if err := os.WriteFile(f, []byte("tok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(f, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readToken(tokenEnv("JUMPGATE_ADMIN_TOKEN", f), "JUMPGATE_ADMIN_TOKEN")
	if err == nil {
		t.Fatal("a 0644 token file was accepted")
	}
	if !strings.Contains(err.Error(), f) || !strings.Contains(err.Error(), "chmod 600 "+f) {
		t.Fatalf("error %q does not name the file and the fix", err)
	}
}

func TestTokenFile0600IsAccepted(t *testing.T) {
	f := filepath.Join(t.TempDir(), "admin")
	if err := os.WriteFile(f, []byte("tok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readToken(tokenEnv("JUMPGATE_ADMIN_TOKEN", f), "JUMPGATE_ADMIN_TOKEN")
	if err != nil || got != "tok" {
		t.Fatalf("readToken = %q, %v", got, err)
	}
}

// A symlink is refused even when it points at a private file: the name the
// operator configured must be the file itself.
func TestTokenFileSymlinkIsRefused(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, []byte("tok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	_, err := readToken(tokenEnv("JUMPGATE_RELAY_TOKEN", link), "JUMPGATE_RELAY_TOKEN")
	if err == nil {
		t.Fatal("a symlinked token file was accepted")
	}
	if !strings.Contains(err.Error(), link) {
		t.Fatalf("error %q does not name the file", err)
	}
}

func TestTokenFileThatIsADirectoryIsRefused(t *testing.T) {
	d := t.TempDir()
	if err := os.Chmod(d, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(tokenEnv("JUMPGATE_ADMIN_TOKEN", d), "JUMPGATE_ADMIN_TOKEN"); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("a directory: err = %v", err)
	}
}
