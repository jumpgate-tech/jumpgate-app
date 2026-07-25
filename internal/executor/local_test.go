package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestLocalShellError covers the control-plane-OS decision without needing to
// build for that OS: local mode runs POSIX shell command strings, so it is
// supported exactly where a POSIX shell is. Windows control planes drive
// Linux nodes over SSH instead.
func TestLocalShellError(t *testing.T) {
	tests := []struct {
		name          string
		goos          string
		wantErr       bool
		wantSubstring string
	}{
		{name: "linux", goos: "linux"},
		{name: "darwin", goos: "darwin"},
		{name: "freebsd", goos: "freebsd"},
		{
			name:          "windows has no POSIX shell",
			goos:          "windows",
			wantErr:       true,
			wantSubstring: "SSH target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := localShellError(tt.goos)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("localShellError(%q) = %v, want nil", tt.goos, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("localShellError(%q) = nil, want an error", tt.goos)
			}
			if !errors.Is(err, ErrNoPOSIXShell) {
				t.Errorf("localShellError(%q) = %v, want it to wrap ErrNoPOSIXShell", tt.goos, err)
			}
			if !strings.Contains(err.Error(), tt.wantSubstring) {
				t.Errorf("localShellError(%q) = %q, want it to mention %q so the operator knows what to do instead", tt.goos, err, tt.wantSubstring)
			}
		})
	}
}

// TestLocalAvailable_MatchesHost pins LocalAvailable to the host it runs on:
// it must agree with localShellError for runtime.GOOS, and on this (POSIX)
// machine that means local mode is available.
func TestLocalAvailable_MatchesHost(t *testing.T) {
	got, want := LocalAvailable(), localShellError(runtime.GOOS)
	if (got == nil) != (want == nil) {
		t.Fatalf("LocalAvailable() = %v, want %v for GOOS=%s", got, want, runtime.GOOS)
	}
	if runtime.GOOS != "windows" && got != nil {
		t.Errorf("LocalAvailable() = %v on %s, want nil", got, runtime.GOOS)
	}
}

// TestLocal_UnsupportedHostFailsEveryCall asserts the refusal is total. A
// local executor on a shell-less host must not half-work: writing a systemd
// unit onto the control plane's own C: drive while no command can ever run is
// worse than an outright error.
func TestLocal_UnsupportedHostFailsEveryCall(t *testing.T) {
	e := &local{unsupported: localShellError("windows")}

	if _, err := e.Run(context.Background(), "echo hello", nil); !errors.Is(err, ErrNoPOSIXShell) {
		t.Errorf("Run error = %v, want ErrNoPOSIXShell", err)
	}
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := e.WriteFile(context.Background(), path, []byte("x"), 0o600); !errors.Is(err, ErrNoPOSIXShell) {
		t.Errorf("WriteFile error = %v, want ErrNoPOSIXShell", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Errorf("WriteFile created %s on an unsupported host; it must not touch the filesystem", path)
	}
	if _, err := e.ReadFile(context.Background(), path); !errors.Is(err, ErrNoPOSIXShell) {
		t.Errorf("ReadFile error = %v, want ErrNoPOSIXShell", err)
	}
	if err := e.Close(); err != nil {
		t.Errorf("Close error = %v, want nil (closing a never-opened executor is a no-op)", err)
	}
}

func TestLocal_Run_CapturesStdout(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "echo hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Stdout != "hello\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "hello\n")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestLocal_Run_CapturesStderrSeparately(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "echo out; echo err 1>&2", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Stdout != "out\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "out\n")
	}
	if res.Stderr != "err\n" {
		t.Errorf("Stderr = %q, want %q", res.Stderr, "err\n")
	}
}

func TestLocal_Run_NonZeroExitIsNotAnError(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "exit 3", nil)
	if err != nil {
		t.Fatalf("Run returned error for non-zero exit: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}
}

func TestLocal_Run_StreamReceivesLinesInOrder(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	var lines []string
	opts := &RunOpts{Stream: func(line string) {
		lines = append(lines, line)
	}}

	res, err := e.Run(context.Background(), `printf 'a\nb\n'`, opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Errorf("streamed lines = %v, want [a b]", lines)
	}
	if res.Stdout != "a\nb\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "a\nb\n")
	}
}

func TestLocal_WriteFile_ReadFile_RoundTrips(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "file.txt")
	content := []byte("hello world\n")

	if err := e.WriteFile(context.Background(), path, content, 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := e.ReadFile(context.Background(), path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadFile content = %q, want %q", got, content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want %v", info.Mode().Perm(), os.FileMode(0o640))
	}
}

func TestLocal_WriteFile_OverwriteAppliesNewMode(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	if err := e.WriteFile(context.Background(), path, []byte("first"), 0o600); err != nil {
		t.Fatalf("WriteFile (create): %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat after create: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode after create = %v, want 0600", info.Mode().Perm())
	}

	if err := e.WriteFile(context.Background(), path, []byte("second"), 0o644); err != nil {
		t.Fatalf("WriteFile (overwrite): %v", err)
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatalf("Stat after overwrite: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode after overwrite = %v, want 0644 (WriteFile must chmod on overwrite, not just on create)", info.Mode().Perm())
	}
}

func TestLocal_Run_CtxCancelKillsFast(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := e.Run(ctx, "sleep 5", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error from ctx cancel, got nil")
	}
	if elapsed >= time.Second {
		t.Errorf("Run took %v, want < 1s", elapsed)
	}
}

// TestLocal_Run_CtxCancelKillsProcessGroup exercises a shell command that
// backgrounds an orphaned child ("sleep 30 &") in addition to its foreground
// child. If Run only kills the direct `sh` PID, the backgrounded sleep keeps
// the stdout pipe's write end open and Run blocks reading it until the
// orphan naturally exits (30s here) or the process's own WaitDelay elapses.
// Killing the whole process group must make Run return well under 2s.
func TestLocal_Run_CtxCancelKillsProcessGroup(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := e.Run(ctx, "sleep 30 & sleep 30", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error from ctx cancel, got nil")
	}
	if elapsed >= 2*time.Second {
		t.Errorf("Run took %v, want < 2s (orphaned backgrounded child must be killed too)", elapsed)
	}
}

// largeLineCmd is a portable (no coreutils/GNU-isms) shell pipeline that
// prints exactly 2MB of 'x' characters with no trailing newline, as a
// single unbroken line — well past the historical 1MB scanner buffer cap.
const largeLineCmd = "head -c 2097152 /dev/zero | tr '\\0' 'x'"

func TestLocal_Run_LargeSingleLineStdout_NoStream(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	const size = 2 * 1024 * 1024

	res, err := e.Run(context.Background(), largeLineCmd, nil)
	if err != nil {
		t.Fatalf("Run returned error for a 2MB single line: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if len(res.Stdout) != size {
		t.Errorf("len(Stdout) = %d, want %d", len(res.Stdout), size)
	}
	if strings.Count(res.Stdout, "x") != size {
		t.Errorf("Stdout content is not all 'x'")
	}
}

func TestLocal_Run_LargeSingleLineStdout_WithStream(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	const size = 2 * 1024 * 1024
	opts := &RunOpts{Stream: func(line string) {}}

	res, err := e.Run(context.Background(), largeLineCmd, opts)
	if err != nil {
		t.Fatalf("Run returned error for a 2MB single line with Stream set: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if len(res.Stdout) != size {
		t.Errorf("len(Stdout) = %d, want %d", len(res.Stdout), size)
	}
}
