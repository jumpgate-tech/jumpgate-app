//go:build unix

package signer

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Symlinks are covered on every OS by TestLoadKeyFileRefusesASymlink.
func TestLoadKeyFileRefusesNonRegularFiles(t *testing.T) {
	dir := t.TempDir()

	fifo := filepath.Join(dir, "fifo.key")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeyFile(fifo); err == nil {
		t.Fatal("LoadKeyFile accepted a FIFO")
	}

	sub := filepath.Join(dir, "dir.key")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeyFile(sub); err == nil {
		t.Fatal("LoadKeyFile accepted a directory")
	}
}
