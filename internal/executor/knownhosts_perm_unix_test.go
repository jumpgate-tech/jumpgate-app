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
