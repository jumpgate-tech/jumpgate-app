package main

import (
	"context"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/sysunit"
)

// privRecorder records the commands and file writes of a privileged setup.
type privRecorder struct {
	mu     sync.Mutex
	cmds   []string
	writes map[string][]byte
	order  []string
}

func (r *privRecorder) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmds = append(r.cmds, cmd)
	return executor.Result{}, nil
}
func (r *privRecorder) WriteFile(_ context.Context, p string, b []byte, _ fs.FileMode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.writes == nil {
		r.writes = map[string][]byte{}
	}
	r.writes[p] = b
	r.order = append(r.order, p)
	return nil
}
func (r *privRecorder) ReadFile(context.Context, string) ([]byte, error) { return nil, nil }
func (r *privRecorder) Close() error                                     { return nil }

// setupSeams stands in for the OS, euid, the privileged executor and the
// package's unit file.
func setupSeams(t *testing.T, goos string, euid int, packaged bool) *privRecorder {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the service is Linux-only, and its paths are not absolute on Windows")
	}
	rec := &privRecorder{}
	oldGOOS, oldEuid, oldPriv, oldExists := hostGOOS, geteuid, runPrivileged, pathExists
	hostGOOS, geteuid = goos, func() int { return euid }
	runPrivileged = func() executor.Executor { return rec }
	pathExists = func(p string) bool { return packaged && p == sysunit.PackageUnitPath }
	oldStat := statOwner
	statOwner = fakeTree(nil)
	t.Cleanup(func() {
		hostGOOS, geteuid, runPrivileged, pathExists, statOwner = oldGOOS, oldEuid, oldPriv, oldExists, oldStat
	})
	return rec
}

type fakeStat struct {
	uid  int
	mode os.FileMode
}

// fakeTree is a filesystem where every path is a root-owned 0755 directory,
// a path ending in "jumpgate" is a root-owned 0755 file, and over replaces
// either.
func fakeTree(over map[string]fakeStat) func(string) (int, os.FileMode, error) {
	return func(p string) (int, os.FileMode, error) {
		if s, ok := over[p]; ok {
			return s.uid, s.mode, nil
		}
		if strings.HasSuffix(p, "/jumpgate") {
			return 0, 0o755, nil
		}
		return 0, os.ModeDir | 0o755, nil
	}
}

func TestInstallServiceRefusesAnUnsafeBinary(t *testing.T) {
	for name, over := range map[string]map[string]fakeStat{
		"binary owned by a user":     {"/home/al/bin/jumpgate": {1000, 0o755}},
		"binary group-writable":      {"/home/al/bin/jumpgate": {0, 0o775}},
		"binary is a symlink":        {"/home/al/bin/jumpgate": {0, os.ModeSymlink | 0o777}},
		"parent owned by a user":     {"/home/al/bin": {1000, os.ModeDir | 0o755}},
		"grandparent world-writable": {"/home/al": {0, os.ModeDir | os.ModeSticky | 0o777}},
		"parent is a symlink":        {"/home": {0, os.ModeSymlink | 0o777}},
	} {
		rec := setupSeams(t, "linux", 0, false)
		statOwner = fakeTree(over)
		_, stderr := captureStdio(t)
		if code := cmdSetup([]string{"install-service", "--bin", "/home/al/bin/jumpgate"}); code != exitCode("failed") {
			t.Fatalf("%s: exit %d, want failed", name, code)
		}
		if len(rec.cmds) != 0 || len(rec.writes) != 0 {
			t.Fatalf("%s: did something: %v", name, rec.cmds)
		}
		if !strings.Contains(stderr(), "refusing to point jumpgate.service") {
			t.Fatalf("%s: stderr %q", name, stderr())
		}
	}
}

// installedAt is the command that put a staged file at dst, and the staged
// file's content.
func installedAt(t *testing.T, rec *privRecorder, dst string) (int, []byte) {
	t.Helper()
	for i, c := range rec.cmds {
		if strings.HasPrefix(c, "install ") && strings.HasSuffix(c, " '"+dst+"'") {
			for staged, b := range rec.writes {
				if strings.Contains(c, "'"+staged+"'") {
					return i, b
				}
			}
		}
	}
	t.Fatalf("nothing installed at %s: %v", dst, rec.cmds)
	return 0, nil
}

func TestInstallServiceWritesAndReloads(t *testing.T) {
	rec := setupSeams(t, "linux", 0, false)
	captureStdio(t)
	if code := cmdSetup([]string{"install-service", "--bin", "/usr/local/bin/jumpgate"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	for p := range rec.writes {
		if !strings.HasPrefix(p, "/var/lib/jumpgate/staging/") {
			t.Fatalf("wrote %s outside the root-only staging dir", p)
		}
	}
	iu, unit := installedAt(t, rec, sysunit.AdminUnitPath)
	is, users := installedAt(t, rec, sysunit.AdminSysusersPath)
	if string(unit) != sysunit.Unit("/usr/local/bin/jumpgate") || string(users) != sysunit.Sysusers || iu > is {
		t.Fatalf("unit/sysusers content or order wrong: %v", rec.cmds)
	}
	var tail []string
	for _, c := range rec.cmds {
		if strings.Contains(c, "enable") {
			t.Fatalf("enabled the unit: %q", c)
		}
		if strings.HasPrefix(c, "systemd-sysusers") || strings.HasPrefix(c, "systemctl") {
			tail = append(tail, c)
		}
	}
	if len(tail) != 2 || tail[0] != "systemd-sysusers /etc/sysusers.d/jumpgate.conf" || tail[1] != "systemctl daemon-reload" {
		t.Fatalf("commands %v", tail)
	}
}

func TestInstallServiceNeedsRoot(t *testing.T) {
	rec := setupSeams(t, "linux", 1000, false)
	captureStdio(t)
	if code := cmdSetup([]string{"install-service"}); code != exitCode("failed") {
		t.Fatalf("exit %d, want failed", code)
	}
	if len(rec.writes) != 0 || len(rec.cmds) != 0 {
		t.Fatalf("did something as non-root: %v %v", rec.order, rec.cmds)
	}
}

func TestInstallServiceIsLinuxOnly(t *testing.T) {
	rec := setupSeams(t, "darwin", 0, false)
	captureStdio(t)
	if code := cmdSetup([]string{"install-service"}); code != exitCode("failed") || len(rec.cmds) != 0 {
		t.Fatalf("exit %d cmds %v", code, rec.cmds)
	}
}

func TestInstallServiceDefersToThePackage(t *testing.T) {
	rec := setupSeams(t, "linux", 0, true)
	_, stderr := captureStdio(t)
	if code := cmdSetup([]string{"install-service"}); code != exitCode("failed") {
		t.Fatalf("exit %d, want failed", code)
	}
	if !strings.Contains(stderr(), "apt or dnf") || len(rec.cmds) != 0 {
		t.Fatalf("stderr %q cmds %v", stderr(), rec.cmds)
	}
}
