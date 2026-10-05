package setup

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// homeOn is the home directory of the user jumpgate acts as on the target:
// read from this process on the local machine (no shell needed), and from
// the target's shell anywhere else.
func homeOn(ctx context.Context, e executor.Executor) (string, error) {
	if h, ok := e.(executor.LocalHost); ok {
		return h.HomeDir()
	}
	res, err := e.Run(ctx, `printf '%s\n' "$HOME"`, nil)
	if err != nil {
		return "", err
	}
	home := strings.TrimSpace(res.Stdout)
	if res.ExitCode != 0 || home == "" {
		return "", fmt.Errorf("$HOME is empty on the target (exit %d)", res.ExitCode)
	}
	return home, nil
}

// joinOn and dirOn apply the path rules of the machine e acts on: this
// machine's (filepath, so C:\Users\… on Windows) for the local executor;
// POSIX for a remote one, whose paths always are (executor/remotepath.go).
func joinOn(e executor.Executor, elem ...string) string {
	if _, ok := e.(executor.LocalHost); ok {
		return filepath.Join(elem...)
	}
	return path.Join(elem...)
}

func dirOn(e executor.Executor, p string) string {
	if _, ok := e.(executor.LocalHost); ok {
		return filepath.Dir(p)
	}
	return path.Dir(p)
}
