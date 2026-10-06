package ops

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// On the local machine docker runs as argv, and nothing reaches a shell.
func TestDockerRun_UsesArgvOnAShellLessExecutor(t *testing.T) {
	f := argvfake.New()
	if _, err := DockerRun(context.Background(), f, "ps", "-a", "--filter", "name=^x$", "--format", "{{.Names}}"); err != nil {
		t.Fatal(err)
	}
	got := f.Argvs()
	if len(got) != 1 || strings.Join(got[0], "|") != "docker|ps|-a|--filter|name=^x$|--format|{{.Names}}" {
		t.Fatalf("argv %q", got)
	}
	if len(f.ShellCalls()) != 0 {
		t.Fatalf("shell used: %q", f.ShellCalls())
	}
}

func TestProbeDocker_OverArgvReadsTheEngine(t *testing.T) {
	f := argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0, build bde2b89\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|linux|x86_64|docker-desktop|Docker Desktop\n"})
	info, err := ProbeDocker(context.Background(), f)
	if err != nil || !info.DaemonReachable || info.Flavor != FlavorDockerDesktop || info.OSType != "linux" {
		t.Fatalf("got %+v, %v", info, err)
	}
	for _, a := range f.Argvs() {
		if strings.Contains(strings.Join(a, " "), "'") {
			t.Fatalf("argv %q carries shell quoting", a)
		}
	}
}

func TestProbeDocker_OverArgvAbsentIsTyped(t *testing.T) {
	f := argvfake.New().Script("docker --version", executor.Result{ExitCode: 127, Stderr: "docker: not found on PATH\n"})
	_, err := ProbeDocker(context.Background(), f)
	if !errors.Is(err, ErrDockerAbsent) {
		t.Fatalf("got %v, want ErrDockerAbsent", err)
	}
}

func TestEnginePlatform_UsesTheLocalHostsNativeArch(t *testing.T) {
	f := argvfake.New()
	f.Arch = "arm64"
	info := DockerInfo{Architecture: "x86_64", Flavor: FlavorDockerDesktop}
	if got := EnginePlatform(context.Background(), f, info); got != "linux/arm64" {
		t.Fatalf("got %q, want the host's reading on a VM-backed engine", got)
	}
}

// Docker Desktop can switch to Linux containers; Docker Engine on Windows
// Server cannot, and must not be told to look for a menu it does not have.
func TestWindowsContainersHintNamesTheRightFix(t *testing.T) {
	desktop := DockerInfo{OSType: "windows", Flavor: FlavorDockerDesktop}.WindowsContainersHint()
	server := DockerInfo{OSType: "windows", Flavor: FlavorDockerEngine}.WindowsContainersHint()
	if !strings.Contains(desktop, "Switch to Linux containers") {
		t.Errorf("Docker Desktop hint %q does not name the switch", desktop)
	}
	if strings.Contains(server, "Switch to") || !strings.Contains(server, "Windows containers only") || !strings.Contains(server, "--ssh") {
		t.Errorf("Windows Server hint %q", server)
	}
	for _, h := range []string{desktop, server} {
		if !strings.Contains(h, "Linux containers") {
			t.Errorf("hint %q lacks \"Linux containers\", which the preflight tests and the CI job look for", h)
		}
	}
}

// Docker Desktop in Windows-container mode answers `docker info` with the
// Windows daemon's own details, which say nothing about Desktop. The local
// machine recognises Desktop by its installed program instead.
func TestProbeDocker_RecognisesDockerDesktopInWindowsMode(t *testing.T) {
	old := dockerDesktopInstalled
	dockerDesktopInstalled = func() bool { return true }
	t.Cleanup(func() { dockerDesktopInstalled = old })
	f := argvfake.New().Script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|DEV-PC|Microsoft Windows 11 Pro\n"})
	info, err := ProbeDocker(context.Background(), f)
	if err != nil || !info.WindowsContainers() || info.Flavor != FlavorDockerDesktop {
		t.Fatalf("got %+v, %v", info, err)
	}
}

// The absence error names the probe that actually ran: the argv form here.
func TestProbeDocker_AbsentNamesTheArgvProbe(t *testing.T) {
	f := argvfake.New().Script("docker --version", executor.Result{ExitCode: 127})
	_, err := ProbeDocker(context.Background(), f)
	var ab *DockerAbsentError
	if !errors.As(err, &ab) || ab.Probe != strings.Join(dockerPresenceCmd.Argv, " ") {
		t.Fatalf("got %v", err)
	}
}
