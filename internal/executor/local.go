package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/secretenv"
)

// ErrNoPOSIXShell reports that this machine cannot run local mode because it
// has no POSIX shell. Callers can test for it with errors.Is.
var ErrNoPOSIXShell = errors.New("local mode requires a POSIX shell")

// localShellError returns a non-nil error when goos cannot run local mode.
//
// Every command string in this codebase is POSIX shell — pipelines, `&&`,
// heredocs, `systemctl`, `journalctl` — written for the Linux node being
// provisioned. Handing those to `cmd /C` or PowerShell would not run them, it
// would fail in a hundred different confusing ways halfway through a setup,
// so there is deliberately no Windows shell port here. A Windows control
// plane's supported path is an SSH target pointing at a Linux host; local mode
// is only ever "the control plane IS the node", which a Windows box never is.
//
// Taking goos as an argument rather than reading runtime.GOOS inline keeps
// this decision unit-testable from any host OS.
func localShellError(goos string) error {
	if goos == "windows" {
		return fmt.Errorf("%w, which %s does not provide: node setup, services, logs and the VPN need a Linux machine added over SSH; the Docker gateway and devnet do run here", ErrNoPOSIXShell, goos)
	}
	return nil
}

// LocalAvailable returns nil if local mode works on this machine, or an error
// wrapping ErrNoPOSIXShell if it does not. Call it at target-construction time
// so an unsupported control plane fails immediately with an actionable
// message, rather than midway through a setup run.
func LocalAvailable() error {
	return localShellError(runtime.GOOS)
}

// local runs commands and touches files on the machine it executes on.
type local struct {
	// unsupported is non-nil when this machine has no POSIX shell. Run
	// refuses with it; RunArgv, the files and the LocalHost facts do not need
	// a shell and keep working (spec D30). RequireShell exposes it to the
	// routes that need a shell.
	unsupported error
}

// NewLocal returns an Executor that runs commands on the local machine. On a
// host with no POSIX shell, Run fails with ErrNoPOSIXShell, and RunArgv, the
// files and the LocalHost facts still work (spec D30).
func NewLocal() Executor {
	return &local{unsupported: LocalAvailable()}
}

// ShellError is nil when Run works here; see RequireShell.
func (l *local) ShellError() error { return l.unsupported }

func (l *local) Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error) {
	if l.unsupported != nil {
		return Result{}, l.unsupported
	}
	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	// A macOS app launched from Finder inherits a minimal PATH
	// (/usr/bin:/bin:/usr/sbin:/sbin) — missing Homebrew and Docker Desktop —
	// so `command -v docker` fails even when docker works in a terminal. Extend
	// PATH for local commands so tools are found the same way they are in a
	// shell. See localEnv.
	// The server's own secrets (JUMPGATE_*TOKEN*, *_SECRET*) are never handed
	// to a command run on a local target.
	c.Env = localEnv(secretenv.Environ(), runtime.GOOS, os.Getenv("HOME"))
	return l.start(ctx, c, opts)
}

// RunArgv starts argv[0] directly, with no shell, on every OS (spec D29).
// A program that is not on PATH comes back as exit 127 with a "not found"
// stderr, which is the reading `sh -c` gives, so callers branch on one shape.
// It gets the same environment Run does: the extended PATH, and none of the
// server's own secrets.
func (l *local) RunArgv(ctx context.Context, argv []string, opts *RunOpts) (Result, error) {
	if len(argv) == 0 {
		return Result{}, errors.New("executor: RunArgv: empty argv")
	}
	env := localEnv(secretenv.Environ(), runtime.GOOS, os.Getenv("HOME"))
	prog, err := lookPathIn(argv[0], envValue(env, "PATH", runtime.GOOS), runtime.GOOS, envValue(env, "PATHEXT", runtime.GOOS))
	if err != nil {
		return Result{ExitCode: 127, Stderr: argv[0] + ": not found on PATH\n"}, nil
	}
	c := exec.CommandContext(ctx, prog, argv[1:]...)
	c.Env = env
	return l.start(ctx, c, opts)
}

// The LocalHost facts (see argv.go).
func (l *local) HostGOOS() string                      { return runtime.GOOS }
func (l *local) HomeDir() (string, error)              { return os.UserHomeDir() }
func (l *local) NativeArch(ctx context.Context) string { return nativeArch(ctx, l) }
func (l *local) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}

// envValue reads key from env, ignoring case on Windows, where the
// variable is usually spelled "Path".
func envValue(env []string, key, goos string) string {
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if ok && (k == key || (goos == "windows" && strings.EqualFold(k, key))) {
			return v
		}
	}
	return ""
}

// start runs a prepared command: its own process group, stdout streamed by
// line, stderr captured, a non-zero exit reported as ExitCode, not an error.
func (l *local) start(ctx context.Context, c *exec.Cmd, opts *RunOpts) (Result, error) {
	// See proc_unix.go / proc_windows.go: on unix this runs cmd in its own
	// process group so ctx cancellation can kill every descendant it spawned,
	// not just the direct `sh` PID.
	setupProcAttrs(c)
	// Backstop: if killing the process group somehow doesn't unblock our
	// stdout read within this window (e.g. a doubly-detached daemon in a
	// different process group), exec forcibly closes the stdout pipe so Run
	// can still return instead of hanging forever.
	c.WaitDelay = 2 * time.Second

	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stderr = &stderrBuf

	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return Result{}, err
	}

	if opts != nil && opts.Stdin != nil {
		c.Stdin = opts.Stdin
	}

	if err := c.Start(); err != nil {
		return Result{}, err
	}

	var streamFn StreamFunc
	if opts != nil {
		streamFn = opts.Stream
	}
	w := &lineStreamer{buf: &stdoutBuf, fn: streamFn}

	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(w, stdoutPipe)
		copyErrCh <- err
	}()

	// It is incorrect to call Wait before all reads from the StdoutPipe have
	// completed, so wait for the copy goroutine first.
	copyErr := <-copyErrCh
	w.Flush()
	waitErr := c.Wait()

	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if copyErr != nil {
		return Result{}, copyErr
	}

	result := Result{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}

	if waitErr != nil {
		exitErr, ok := waitErr.(*exec.ExitError)
		if !ok {
			return Result{}, waitErr
		}
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, nil
}

// WriteFile writes to the machine this process runs on, so path is a LOCAL
// path and filepath (host separator) is the correct choice here — the opposite
// of the SSH executor, whose paths are always POSIX. See remotepath.go.
func (l *local) WriteFile(_ context.Context, path string, content []byte, mode fs.FileMode) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return err
	}
	// os.WriteFile only applies mode on create; an existing file keeps its
	// old permissions, and umask can still narrow them even then. Chmod
	// unconditionally so mode is authoritative in both cases.
	return os.Chmod(path, mode)
}

// ReadFile reads from the machine this process runs on; path is a LOCAL path.
func (l *local) ReadFile(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (l *local) Close() error {
	return nil
}

// localEnv returns env (normally os.Environ()) with PATH extended so local
// commands find tools a GUI-started app would miss: a Finder-launched macOS
// .app, or an Explorer-started jumpgate-tray.exe whose login-session PATH
// predates a Docker Desktop install. The extra dirs are APPENDED, so an
// existing PATH entry keeps priority and a user override still wins; a dir
// that does not exist is harmless, since a PATH lookup just skips it. Taking
// env/goos/home as arguments keeps it unit-testable from any host OS. On
// Linux it returns env unchanged.
func localEnv(env []string, goos, home string) []string {
	extra := guiPathDirs(goos, home, env)
	if len(extra) == 0 {
		return env
	}
	sep := ":"
	if goos == "windows" {
		sep = ";"
	}
	for i, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if ok && (k == "PATH" || (goos == "windows" && strings.EqualFold(k, "PATH"))) {
			env[i] = k + "=" + appendMissingDirs(v, extra, sep)
			return env
		}
	}
	// No PATH in the environment at all (unusual) — set one from the extras.
	return append(env, "PATH="+strings.Join(extra, sep))
}

// guiPathDirs lists the tool locations a GUI-started app does NOT inherit on
// PATH. On macOS: Homebrew (Apple Silicon and Intel), Docker Desktop, and the
// common Docker-alternative CLIs (OrbStack, Rancher Desktop). On Windows:
// Docker Desktop's CLI dir under ProgramFiles (read from env; nil when it is
// unset). Empty on Linux, where a local control plane runs from a normal
// shell PATH.
func guiPathDirs(goos, home string, env []string) []string {
	switch goos {
	case "darwin":
	case "windows":
		pf := envValue(env, "ProgramFiles", goos)
		if pf == "" {
			return nil
		}
		// Concatenated, not filepath.Join, so the result is the same on any
		// host OS the test runs on.
		return []string{pf + `\Docker\Docker\resources\bin`}
	default:
		return nil
	}
	dirs := []string{
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/bin",
		"/usr/local/sbin",
		"/Applications/Docker.app/Contents/Resources/bin",
	}
	if home != "" {
		dirs = append(dirs, home+"/.docker/bin", home+"/.orbstack/bin", home+"/.rd/bin")
	}
	return dirs
}

// appendMissingDirs appends each dir not already present in the
// sep-separated path, preserving order and existing precedence.
func appendMissingDirs(path string, dirs []string, sep string) string {
	have := make(map[string]bool)
	for _, p := range strings.Split(path, sep) {
		have[p] = true
	}
	out := path
	for _, d := range dirs {
		if !have[d] {
			out += sep + d
		}
	}
	return out
}
