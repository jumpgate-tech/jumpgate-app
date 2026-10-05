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
