//go:build realsecret && linux

package signer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// These tests drive the REAL secret-tool against a running Secret Service.
// Run them with scripts/test-secret-service.sh, which sets up each case (no
// D-Bus, D-Bus without a Secret Service, an unlocked gnome-keyring) in a
// debian:12 container.

func realRef(t *testing.T) string {
	t.Helper()
	ref := fmt.Sprintf("realsecret-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = exec.Command("secret-tool", "clear", "service", keychainService, "account", ref).Run()
	})
	return ref
}

func TestRealSecretRoundTrip(t *testing.T) {
	ctx := context.Background()
	ref := realRef(t)
	if DefaultStore() != StoreKeychain {
		t.Fatalf("DefaultStore = %v under a D-Bus session with a live Secret Service", DefaultStore())
	}
	_, err := Open(ctx, StoreKeychain, ref)
	if err == nil || !keychainNotFoundMsg(err) {
		t.Fatalf("Open of a missing item = %v, want a not-found error", err)
	}
	k, err := Create(ctx, StoreKeychain, ref)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Open(ctx, StoreKeychain, ref)
	if err != nil || got.Address() != k.Address() {
		t.Fatalf("Open = %v, %v", got, err)
	}
	if _, err := Create(ctx, StoreKeychain, ref); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("second Create = %v, want ErrKeyExists", err)
	}
}

func keychainNotFoundMsg(err error) bool { return strings.Contains(err.Error(), "no item") }

// TestRealSecretLockedKeyringIsNotAbsence locks the login collection with
// busctl; a locked keyring answers `lookup` with exit 1 and empty output, the
// same as a missing item, so Create must not treat that as absent.
func TestRealSecretLockedKeyringIsNotAbsence(t *testing.T) {
	if _, err := exec.LookPath("busctl"); err != nil {
		t.Skip("busctl not available")
	}
	ctx := context.Background()
	ref := realRef(t)
	k, err := Create(ctx, StoreKeychain, ref)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("busctl", "--user", "call", "org.freedesktop.secrets", "/org/freedesktop/secrets",
		"org.freedesktop.Secret.Service", "Lock", "ao", "1", "/org/freedesktop/secrets/collection/login").CombinedOutput(); err != nil {
		t.Skipf("cannot lock the keyring: %v %s", err, out)
	}
	if _, err := Create(ctx, StoreKeychain, ref); err == nil {
		t.Fatal("Create over an existing item in a locked keyring succeeded")
	}
	_, err = Open(ctx, StoreKeychain, ref)
	if err == nil || keychainNotFoundMsg(err) || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("Open in a locked keyring = %v, want a locked error, not not-found", err)
	}
	_ = k
}

// Over SSH or on a headless box there is no session bus: the default is a key
// file, and the keychain store says why it cannot be used.
func TestRealSecretServiceDefaultWithoutDBus(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	if got := DefaultStore(); got != StoreFile {
		t.Fatalf("DefaultStore() with no D-Bus = %q, want file", got)
	}
	_, err := Create(context.Background(), StoreKeychain, "controller")
	if !errors.Is(err, ErrNoKeychain) || !strings.Contains(err.Error(), "D-Bus") || !strings.Contains(err.Error(), "--store file") {
		t.Fatalf("Create with no D-Bus = %v, want ErrNoKeychain naming D-Bus and --store file", err)
	}
}

// A session bus with nothing providing org.freedesktop.secrets (a minimal
// desktop, a container): the probe gets an error and the default is a file.
// The script runs this under a bus with no activatable services.
func TestRealSecretServiceDefaultWithDBusButNoService(t *testing.T) {
	if os.Getenv("JUMPGATE_TEST_NO_SECRET_SERVICE") != "1" {
		t.Skip("needs a session bus with no Secret Service (scripts/test-secret-service.sh)")
	}
	start := time.Now()
	if got := DefaultStore(); got != StoreFile {
		t.Fatalf("DefaultStore() with no Secret Service = %q, want file", got)
	}
	if d := time.Since(start); d > probeTimeout+time.Second {
		t.Fatalf("DefaultStore took %s", d)
	}
}

// hungSecretTool puts a secret-tool first on PATH that never answers and
// leaves a child of its own behind (as a D-Bus autolaunch would), recording
// the child's pid. It returns a function that reports the child's pid.
func hungSecretTool(t *testing.T) func() int {
	t.Helper()
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	script := "#!/bin/sh\nsleep 300 &\necho $! > " + pidFile + "\nwait\n"
	if err := os.WriteFile(filepath.Join(dir, "secret-tool"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent")
	return func() int {
		b, err := os.ReadFile(pidFile)
		if err != nil {
			t.Fatalf("the fake secret-tool never started its child: %v", err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil {
			t.Fatal(err)
		}
		return pid
	}
}

// gone waits briefly for pid to die. A zombie counts as dead: whether pid 1
// reaps orphans depends on the container, not on jumpgate.
func gone(pid int) bool {
	for i := 0; i < 50; i++ {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
		if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
			// The state follows the parenthesised command name.
			if j := strings.LastIndexByte(string(b), ')'); j >= 0 && strings.HasPrefix(string(b[j+1:]), " Z") {
				return true
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// D17: a secret-tool that never answers is cut off at toolTimeout with a
// message naming --store file, and nothing it started is left running.
func TestRealSecretToolTimeoutKillsItsProcessGroup(t *testing.T) {
	childPID := hungSecretTool(t)
	old := toolTimeout
	toolTimeout = 500 * time.Millisecond
	t.Cleanup(func() { toolTimeout = old })
	start := time.Now()
	_, err := Open(context.Background(), StoreKeychain, "controller")
	if err == nil || !strings.Contains(err.Error(), "did not answer within") || !strings.Contains(err.Error(), "--store file") {
		t.Fatalf("Open with a hung secret-tool = %v, want the timeout message", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("Open returned after %s; the timeout did not stop the tool", d)
	}
	if pid := childPID(); !gone(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("secret-tool's child %d survived the timeout", pid)
	}
}

// The default-store probe gives up on a silent Secret Service after
// probeTimeout, picks the file store, and leaves nothing running.
func TestRealSecretServiceProbeTimesOut(t *testing.T) {
	childPID := hungSecretTool(t)
	start := time.Now()
	if got := DefaultStore(); got != StoreFile {
		t.Fatalf("DefaultStore() with a hung Secret Service = %q, want file", got)
	}
	if d := time.Since(start); d < probeTimeout || d > probeTimeout+3*time.Second {
		t.Fatalf("DefaultStore took %s, want about %s", d, probeTimeout)
	}
	if pid := childPID(); !gone(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("the probe's child %d survived", pid)
	}
}
