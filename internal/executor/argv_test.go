package executor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// The shell form of an argv must stay byte-identical to the string
// ops.DockerRun built before Task 15, so an SSH target, and every test fake
// keyed on those strings, sees no change.
func TestQuoteArgvMatchesTheHistoricDockerRunString(t *testing.T) {
	got := QuoteArgv([]string{"docker", "inspect", "-f", "{{.State.Running}}", "it's"})
	want := `docker 'inspect' '-f' '{{.State.Running}}' 'it'\''s'`
	if got != want {
		t.Fatalf("QuoteArgv = %s\nwant       %s", got, want)
	}
}

type shellOnly struct{ ran string }

func (s *shellOnly) Run(_ context.Context, cmd string, _ *RunOpts) (Result, error) {
	s.ran = cmd
	return Result{}, nil
}
func (*shellOnly) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (*shellOnly) ReadFile(context.Context, string) ([]byte, error)           { return nil, nil }
func (*shellOnly) Close() error                                               { return nil }

type argvToo struct {
	shellOnly
	argv []string
}

func (a *argvToo) RunArgv(_ context.Context, argv []string, _ *RunOpts) (Result, error) {
	a.argv = argv
	return Result{}, nil
}

func TestExecPrefersArgvAndKeepsTheShellFormForOthers(t *testing.T) {
	ctx := context.Background()
	c := Command{Argv: []string{"docker", "info", "--format", "{{.OSType}}"}, Shell: "docker info --format '{{.OSType}}'"}

	sh := &shellOnly{}
	if _, err := Exec(ctx, sh, c, nil); err != nil || sh.ran != c.Shell {
		t.Fatalf("shell executor ran %q (%v), want %q", sh.ran, err, c.Shell)
	}
	av := &argvToo{}
	if _, err := Exec(ctx, av, c, nil); err != nil || strings.Join(av.argv, "|") != "docker|info|--format|{{.OSType}}" || av.ran != "" {
		t.Fatalf("argv executor got argv %q and shell %q (%v)", av.argv, av.ran, err)
	}
	sh2 := &shellOnly{}
	if _, _ = Exec(ctx, sh2, Command{Argv: []string{"docker", "ps"}}, nil); sh2.ran != "docker 'ps'" {
		t.Fatalf("no Shell given: ran %q, want the quoted argv", sh2.ran)
	}
}

// TestHelperProcess is not a test. The RunArgv tests start this test binary
// as a plain program, which works on every OS (no sh, no echo.exe).
func TestHelperProcess(t *testing.T) {
	if os.Getenv("JUMPGATE_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Println("argv:" + strings.Join(os.Args[len(os.Args)-2:], "|"))
	os.Exit(3)
}

func helperArgv(a, b string) []string {
	return []string{os.Args[0], "-test.run=^TestHelperProcess$", "--", a, b}
}

func TestLocal_RunArgvStartsTheProgramWithoutAShell(t *testing.T) {
	t.Setenv("JUMPGATE_HELPER_PROCESS", "1")
	res, err := (&local{}).RunArgv(context.Background(), helperArgv("a b", "it's"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 3 || strings.TrimSpace(res.Stdout) != "argv:a b|it's" {
		t.Fatalf("exit %d stdout %q: want exit 3 and the arguments untouched by any shell", res.ExitCode, res.Stdout)
	}
}

// A program that is not there reads the way `sh -c` reports it (exit 127),
// so ProbeDocker's absence check is one branch for both forms.
func TestLocal_RunArgvMissingProgramReadsLikeTheShell(t *testing.T) {
	res, err := (&local{}).RunArgv(context.Background(), []string{"jumpgate-no-such-program"}, nil)
	if err != nil || res.ExitCode != 127 || !strings.Contains(res.Stderr, "not found") {
		t.Fatalf("got %+v, %v; want exit 127 with a not-found stderr", res, err)
	}
}

// Windows, after Task 15: Run is refused, everything Docker needs works.
// This replaces TestLocal_UnsupportedHostFailsEveryCall (spec D30).
func TestLocal_ShellLessHostRefusesOnlyTheShell(t *testing.T) {
	t.Setenv("JUMPGATE_HELPER_PROCESS", "1")
	ctx := context.Background()
	e := &local{unsupported: localShellError("windows")}

	if _, err := e.Run(ctx, "echo hi", nil); !errors.Is(err, ErrNoPOSIXShell) {
		t.Errorf("Run: %v, want ErrNoPOSIXShell", err)
	}
	if err := RequireShell(e); !errors.Is(err, ErrNoPOSIXShell) {
		t.Errorf("RequireShell: %v, want ErrNoPOSIXShell", err)
	}
	if res, err := e.RunArgv(ctx, helperArgv("x", "y"), nil); err != nil || res.ExitCode != 3 {
		t.Errorf("RunArgv: %+v, %v; want the program to run", res, err)
	}
	p := filepath.Join(t.TempDir(), "erpc.yaml")
	if err := e.WriteFile(ctx, p, []byte("a: 1\n"), 0o644); err != nil {
		t.Errorf("WriteFile: %v", err)
	}
	if got, err := e.ReadFile(ctx, p); err != nil || string(got) != "a: 1\n" {
		t.Errorf("ReadFile: %q, %v", got, err)
	}
	if h, err := e.HomeDir(); err != nil || h == "" {
		t.Errorf("HomeDir: %q, %v", h, err)
	}
	if e.HostGOOS() != runtime.GOOS {
		t.Errorf("HostGOOS = %q", e.HostGOOS())
	}
}

// exec.Command resolves a bare name against this process's PATH, before
// c.Env applies, so the dirs localEnv adds would never be searched.
// lookPathIn takes the PATH to search.
func TestLookPathInSearchesTheGivenPathNotTheProcessPath(t *testing.T) {
	testutil.RequireUnix(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "docker")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "/nonexistent")
	if got, err := lookPathIn("docker", dir, runtime.GOOS, ""); err != nil || got != p {
		t.Fatalf("lookPathIn = %q, %v; want %q", got, err, p)
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := lookPathIn("docker", dir, runtime.GOOS, ""); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("a non-executable file: %v, want exec.ErrNotFound", err)
	}
}
