//go:build windows

package executor

import (
	"os/exec"
	"testing"
)

// makeWorldWritable grants Everyone (S-1-1-0) write on path.
func makeWorldWritable(t *testing.T, path string) {
	t.Helper()
	if out, err := exec.Command("icacls", path, "/grant", "*S-1-1-0:(W)").CombinedOutput(); err != nil {
		t.Fatalf("icacls: %v: %s", err, out)
	}
}
