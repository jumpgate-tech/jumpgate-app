//go:build !linux

package jobs

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
)

// J9: on macOS (where the agent runs only for its tests) jobs are refused
// with a reason that names the OS.
func TestDefaultRunnerRefusesOffLinux(t *testing.T) {
	err := DefaultRunner(nil).Supported(context.Background())
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), runtime.GOOS) {
		t.Fatalf("Supported = %v", err)
	}
}
