//go:build !windows

package executor

import (
	"context"
	"strings"
)

// nativeArch asks the system's own uname, at a fixed path, never one found
// on PATH. A Homebrew GNU uname is itself an x86_64 binary on an Apple
// Silicon Mac and reports x86_64; ops.unameArchProbe documents that
// measured failure. /bin/uname covers distros without a merged /usr.
func nativeArch(ctx context.Context, l *local) string {
	for _, p := range []string{"/usr/bin/uname", "/bin/uname"} {
		if res, err := l.RunArgv(ctx, []string{p, "-m"}, nil); err == nil && res.ExitCode == 0 {
			return strings.TrimSpace(res.Stdout)
		}
	}
	return ""
}
