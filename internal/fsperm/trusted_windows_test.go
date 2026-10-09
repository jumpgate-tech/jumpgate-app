//go:build windows

package fsperm

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenTrustedWritableOnWindows(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ssh_known_hosts")
	if err := os.WriteFile(p, []byte("checked"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := OpenTrustedWritable(p)
	if err != nil {
		t.Fatalf("owner-writable file refused: %v", err)
	}
	b := make([]byte, 16)
	n, _ := f.Read(b)
	f.Close()
	if string(b[:n]) != "checked" {
		t.Fatalf("read %q", b[:n])
	}
	if out, err := exec.Command("icacls", p, "/grant", "*S-1-1-0:(W)").CombinedOutput(); err != nil {
		t.Fatalf("icacls: %v: %s", err, out)
	}
	if _, err := OpenTrustedWritable(p); err == nil {
		t.Fatal("a file Everyone can write was trusted")
	}
}

// A symlink is resolved once and the real file judged; a file with a reparse
// point as its last component is refused if it is reached that way.
func TestOpenNoReparseRefusesAReparsePoint(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("no symlink privilege: %v", err)
	}
	if h, err := openNoReparse(link, 0x80000000, 7, 0); err == nil {
		_ = h
		t.Fatal("a symlink was opened as a file")
	}
}

// The returned file's Name is the final path of the open handle with no \\?\
// prefix for a drive path: the resolved file that was checked, in a form ssh's
// argv split accepts.
func TestOpenTrustedWritableNamesTheResolvedPath(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := OpenTrustedWritable(real)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got := f.Name()
	if strings.HasPrefix(got, `\\`) || len(got) < 3 || got[1] != ':' || got[2] != '\\' {
		t.Fatalf("Name() = %q, want a plain drive path", got)
	}
	want, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(got, want) {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}
