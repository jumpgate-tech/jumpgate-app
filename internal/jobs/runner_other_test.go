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

// Off Linux nothing can be started either, and nothing reads as running.
func TestDefaultRunnerOffLinuxStartsNothing(t *testing.T) {
	r := DefaultRunner(nil)
	if err := r.Start(context.Background(), "jumpgate-job-0123456789abcdef-one-0", []string{"/bin/true"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Start = %v", err)
	}
	if on, err := r.Active(context.Background(), "u"); on || err != nil {
		t.Fatalf("Active = %v, %v", on, err)
	}
	if on, err := r.AnyActive(context.Background(), "0123456789abcdef"); on || err != nil {
		t.Fatalf("AnyActive = %v, %v", on, err)
	}
}
