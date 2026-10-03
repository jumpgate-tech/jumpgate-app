//go:build unix

package signer

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestLoadKeyFileRefusesSymlinksAndNonRegularFiles(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.key")
	if _, err := GenerateKeyFile(real); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "link.key")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeyFile(link); err == nil {
		t.Fatal("LoadKeyFile followed a symlink")
	}

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
