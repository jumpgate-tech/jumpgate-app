//go:build !windows

package executor

import (
	"os"
	"testing"
)

func makeWorldWritable(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
}

func TestTrustedUnixFileDecision(t *testing.T) {
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
		if err := trustedUnixFile("f", c.mode, c.uid, c.euid); (err == nil) != c.ok {
			t.Errorf("%v uid %d euid %d: err = %v, want ok=%v", c.mode, c.uid, c.euid, err, c.ok)
		}
	}
}
