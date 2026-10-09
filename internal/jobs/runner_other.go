//go:build !linux

package jobs

import (
	"context"
	"fmt"
	"runtime"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// The agent is Linux-only by design; macOS keeps it only so its tests run
// there (platform spec D19), and Windows refuses `jumpgate agent` outright.
// Off Linux every job kind is refused with this reason (D-J9).
type unsupportedRunner struct{ goos string }

func DefaultRunner(executor.Executor) Runner { return unsupportedRunner{goos: runtime.GOOS} }

func (u unsupportedRunner) Supported(context.Context) error {
	return fmt.Errorf("%w: durable jobs run only on Linux boxes with systemd; this agent runs on %s", ErrUnsupported, u.goos)
}

func (u unsupportedRunner) Start(ctx context.Context, _ string, _ []string) error {
	return u.Supported(ctx)
}
func (unsupportedRunner) Active(context.Context, string) (bool, error) { return false, nil }
