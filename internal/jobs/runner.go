package jobs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// Runner starts a step's process somewhere that outlives the agent.
type Runner interface {
	// Supported is nil when this box can run jobs, or an ErrUnsupported
	// that says why not.
	Supported(ctx context.Context) error
	// Start launches argv as the transient unit unit. It returns once
	// systemd accepted (and, with Type=exec, executed) it.
	Start(ctx context.Context, unit string, argv []string) error
	// Active reports whether unit is still running. An error means "cannot
	// tell", which callers treat as "still running".
	Active(ctx context.Context, unit string) (bool, error)
}

var ErrUnsupported = errors.New("jobs: durable jobs are not available on this box")

var (
	stepRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	unitRE = regexp.MustCompile(`^jumpgate-job-[0-9a-f]{16}-[a-z0-9][a-z0-9-]{0,31}-[0-9]{1,4}$`)
)

func ValidStepID(s string) bool { return stepRE.MatchString(s) }

// UnitName is the spec's jumpgate-job-<id>-<step>, where <step> is the step
// id and the attempt (D-J3), so a retry never collides with its failed
// predecessor.
func UnitName(id, step string, attempt int) string {
	return "jumpgate-job-" + id + "-" + step + "-" + strconv.Itoa(attempt)
}

// UnitPattern matches every unit of a job, for journalctl -u.
func UnitPattern(id string) string { return "jumpgate-job-" + id + "-*" }

// SystemdRunner runs steps with systemd-run. The unit is PID 1's child, not
// the agent's, so it survives an agent restart; --collect unloads it when it
// ends, and the step's result file, not the unit, says how it ended.
type SystemdRunner struct{ Exec executor.Executor }

func (r SystemdRunner) Supported(ctx context.Context) error {
	res, err := r.Exec.Run(ctx, "test -d /run/systemd/system", nil)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("%w: this box does not run systemd as its init (/run/systemd/system is absent)", ErrUnsupported)
	}
	res, err = r.Exec.Run(ctx, "command -v systemd-run", nil)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("%w: systemd-run is not installed", ErrUnsupported)
	}
	return nil
}

func (r SystemdRunner) Start(ctx context.Context, unit string, argv []string) error {
	// The unit name is the only caller-influenced text that is not quoted,
	// so it must match the one shape UnitName produces.
	if !unitRE.MatchString(unit) || len(argv) == 0 {
		return fmt.Errorf("jobs: refusing to start unit %q", unit)
	}
	const prefix = len("jumpgate-job-")
	id := unit[prefix : prefix+16]
	step := unit[prefix+17:]
	var b strings.Builder
	b.WriteString("systemd-run --unit=" + unit)
	b.WriteString(" --description=" + shQuote("jumpgate job "+id+" step "+step))
	b.WriteString(" --collect --quiet --property=Type=exec --")
	for _, a := range argv {
		b.WriteString(" " + shQuote(a))
	}
	res, err := r.Exec.Run(ctx, b.String(), nil)
	if err != nil {
		return fmt.Errorf("jobs: systemd-run %s: %w", unit, err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("jobs: systemd-run %s exited %d: %s", unit, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

func (r SystemdRunner) Active(ctx context.Context, unit string) (bool, error) {
	if !unitRE.MatchString(unit) {
		return false, fmt.Errorf("jobs: %q is not a job unit", unit)
	}
	// is-active exits non-zero for every state but active; only stdout says
	// which. A collected unit reads "inactive".
	res, err := r.Exec.Run(ctx, "systemctl is-active "+unit+".service", nil)
	if err != nil {
		return false, err
	}
	switch strings.TrimSpace(res.Stdout) {
	case "active", "activating", "deactivating", "reloading", "refreshing":
		return true, nil
	case "inactive", "failed":
		return false, nil
	}
	return false, fmt.Errorf("jobs: systemctl is-active %s: %q %s", unit, strings.TrimSpace(res.Stdout), strings.TrimSpace(res.Stderr))
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
