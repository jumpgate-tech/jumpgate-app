//go:build linux

package jobs

import "github.com/valve-tech/jumpgate/internal/executor"

// DefaultRunner is systemd-run. Supported still checks this box boots with
// systemd: a container or a WSL distro without it is refused, not half-run.
func DefaultRunner(ex executor.Executor) Runner { return SystemdRunner{Exec: ex} }
