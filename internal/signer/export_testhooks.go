package signer

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// SetToolRunnerForTest makes the key stores run fn instead of their helper
// tools (security, secret-tool, op), as if on goos with every tool present,
// until t ends. It panics outside a test binary.
func SetToolRunnerForTest(t testing.TB, goos string, fn func(ctx context.Context, stdin, name string, args ...string) (string, error)) {
	if !testing.Testing() {
		panic("signer: SetToolRunnerForTest outside a test")
	}
	t.Helper()
	oldRun, oldOS, oldLook, oldDBus := runCmd, hostOS, lookPath, dbusSession
	runCmd, hostOS = fn, goos
	lookPath = func(string) error { return nil }
	dbusSession = func() bool { return true }
	t.Cleanup(func() { runCmd, hostOS, lookPath, dbusSession = oldRun, oldOS, oldLook, oldDBus })
}

// FakeKeychainForTest is SetToolRunnerForTest with an in-memory macOS
// keychain behind `security`. It returns the commands run, one per call.
func FakeKeychainForTest(t testing.TB) (calls func() [][]string) {
	t.Helper()
	var mu sync.Mutex
	items := map[string]string{}
	var seen [][]string
	SetToolRunnerForTest(t, "darwin", func(_ context.Context, stdin, name string, args ...string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, append([]string{name}, args...))
		notFound := &cmdError{Name: name, ExitCode: 44, Err: errNotFoundForTest}
		if name != "security" || len(args) == 0 {
			return "", &cmdError{Name: name, ExitCode: 1, Err: errNotFoundForTest}
		}
		if args[0] == "-i" { // add-generic-password -a REF -s SERVICE -w SECRET
			f := strings.Fields(stdin)
			if len(f) < 7 {
				return "", &cmdError{Name: name, ExitCode: 1, Err: errNotFoundForTest}
			}
			items[f[2]] = f[len(f)-1]
			return "", nil
		}
		// find-generic-password -a REF -s SERVICE [-w]
		v, ok := items[args[2]]
		if !ok {
			return "", notFound
		}
		if args[len(args)-1] == "-w" {
			return v + "\n", nil
		}
		return "", nil
	})
	return func() [][]string {
		mu.Lock()
		defer mu.Unlock()
		return append([][]string(nil), seen...)
	}
}

type testErr string

func (e testErr) Error() string { return string(e) }

const errNotFoundForTest = testErr("exit status 44")
