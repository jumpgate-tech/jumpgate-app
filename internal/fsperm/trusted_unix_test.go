//go:build !windows

package fsperm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrustedWritableMode(t *testing.T) {
	for _, c := range []struct {
		mode      os.FileMode
		uid, euid uint32
		ok        bool
	}{
		{0o644, 0, 1000, true},
		{0o644, 1000, 1000, true},
		{0o644, 1001, 1000, false}, // someone else's file
		{0o664, 0, 1000, false},    // group writable
		{0o646, 0, 1000, false},    // world writable
	} {
		if err := trustedWritableMode("f", c.mode, c.uid, c.euid); (err == nil) != c.ok {
			t.Errorf("%v uid %d euid %d: err = %v, want ok=%v", c.mode, c.uid, c.euid, err, c.ok)
		}
	}
}

// A symlink is resolved first: the target and its real directory are judged,
// not the link's directory.
func TestCheckTrustedWritableJudgesTheSymlinkTarget(t *testing.T) {
	base := t.TempDir()
	good := filepath.Join(base, "good")
	bad := filepath.Join(base, "bad")
	for _, d := range []string{good, bad} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(bad, "known_hosts")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(good, "known_hosts")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := CheckTrustedWritable(link); err != nil {
		t.Fatalf("a link to a trusted file was refused: %v", err)
	}
	if err := os.Chmod(bad, 0o777); err != nil { // the real directory is now open
		t.Fatal(err)
	}
	if err := CheckTrustedWritable(link); err == nil {
		t.Fatal("a link into a world-writable directory was trusted")
	}
}

// The file returned is the file checked: swapping the path afterwards, between
// the check and the caller's read, changes nothing.
func TestOpenTrustedWritableReturnsTheCheckedFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(p, []byte("checked"), 0o644); err != nil {
		t.Fatal(err)
	}
	trustedCheckHook = func() {
		_ = os.Remove(p)
		if err := os.WriteFile(p, []byte("swapped"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { trustedCheckHook = nil })
	f, err := OpenTrustedWritable(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b := make([]byte, 16)
	n, _ := f.Read(b)
	if string(b[:n]) != "checked" {
		t.Fatalf("read %q from the returned file, want the checked content", b[:n])
	}
}

// A symlink is resolved once; the resolved file is then opened with O_NOFOLLOW.
func TestOpenTrustedWritableResolvesALinkOnce(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	// EvalSymlinks resolves it; the open of the result is O_NOFOLLOW and works.
	f, err := OpenTrustedWritable(link)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

// The returned file's Name is the resolved path that was checked, which a
// caller passes on (to an ssh child) instead of looking the link up again.
func TestOpenTrustedWritableNamesTheResolvedPath(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	f, err := OpenTrustedWritable(link)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if f.Name() != want {
		t.Fatalf("Name() = %q, want the resolved %q", f.Name(), want)
	}
}
