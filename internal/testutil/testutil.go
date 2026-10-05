// Package testutil holds helpers that keep tests portable across macOS, Linux
// and Windows. Only test files import it.
package testutil

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// ShortTempDir returns a fresh directory, removed when the test ends, whose
// path is short enough to hold unix sockets a few levels down. t.TempDir()
// embeds the test name under TMPDIR, which on macOS (/var/folders/…) already
// eats most of sun_path's 104 bytes, so macOS uses /tmp. Linux's TMPDIR is
// /tmp, and Windows' %TEMP% is short enough without the test name.
func ShortTempDir(t testing.TB) string {
	t.Helper()
	base := os.TempDir()
	if runtime.GOOS == "darwin" {
		base = "/tmp"
	}
	d, err := os.MkdirTemp(base, "jg")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

// Home points both HOME and USERPROFILE at a fresh short directory for the
// rest of the test. os.UserHomeDir reads USERPROFILE on Windows and HOME
// elsewhere; setting only HOME on Windows silently shares one home between
// every test in the package.
func Home(t testing.TB) string {
	t.Helper()
	h := ShortTempDir(t)
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	return h
}

// RequirePOSIXShell skips a test that executes `sh`. The controller never
// shells out locally on Windows (local mode is refused there), so a test that
// needs sh tests code Windows does not run.
func RequirePOSIXShell(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell; local commands never run on a Windows controller")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH")
	}
}

// RequireUnix skips a test of unix-only behaviour: permission bits on
// non-secret files, FIFOs, peer credentials, the agent itself.
func RequireUnix(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("unix-only behaviour")
	}
}

// AssertPrivate fails the test unless only the file's owner can read or
// change path. On unix that is the mode bits; Windows is checked through its
// DACL once internal/fsperm exists (Task 2 of the platform plan).
func AssertPrivate(t testing.TB, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Logf("AssertPrivate(%s): DACL check not implemented yet", path)
		return
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := fi.Mode().Perm()
	if perm&0o077 != 0 || perm&0o600 != 0o600 {
		t.Fatalf("%s has mode %o; want owner read/write and nothing for group or others", path, perm)
	}
}
