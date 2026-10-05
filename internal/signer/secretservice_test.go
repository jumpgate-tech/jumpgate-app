package signer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func withDBus(t *testing.T, up bool) {
	t.Helper()
	old := dbusSession
	dbusSession = func() bool { return up }
	t.Cleanup(func() { dbusSession = old })
}

func TestDefaultStorePerOS(t *testing.T) {
	answers := func(stdin, name string, args []string) (string, error) { return "", nil }
	noService := func(stdin, name string, args []string) (string, error) {
		return "", &cmdError{Name: name, ExitCode: 1, Stderr: "Cannot autolaunch D-Bus without X11 $DISPLAY", Err: errors.New("exit status 1")}
	}
	timedOut := func(stdin, name string, args []string) (string, error) {
		return "", &cmdError{Name: name, ExitCode: -1, Err: context.DeadlineExceeded}
	}
	cases := []struct {
		name string
		goos string
		have []string
		dbus bool
		run  func(string, string, []string) (string, error)
		want Store
	}{
		{"windows", "windows", nil, false, answers, StoreWinCred},
		{"macOS with security", "darwin", []string{"security"}, false, answers, StoreKeychain},
		{"macOS without security", "darwin", nil, false, answers, StoreFile},
		{"linux desktop", "linux", []string{"secret-tool"}, true, answers, StoreKeychain},
		{"linux over SSH, no D-Bus", "linux", []string{"secret-tool"}, false, answers, StoreFile},
		{"linux, D-Bus but no Secret Service", "linux", []string{"secret-tool"}, true, noService, StoreFile},
		{"linux, Secret Service does not answer", "linux", []string{"secret-tool"}, true, timedOut, StoreFile},
		{"linux without secret-tool", "linux", nil, true, answers, StoreFile},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fakeRunner{fn: c.run}
			withRunner(t, f, c.goos, c.have...)
			withDBus(t, c.dbus)
			if got := DefaultStore(); got != c.want {
				t.Fatalf("DefaultStore() = %q, want %q (calls %v)", got, c.want, f.calls)
			}
			if !c.dbus && c.goos == "linux" && len(f.calls) != 0 {
				t.Fatalf("probed the Secret Service with no D-Bus session: %v", f.calls)
			}
		})
	}
}

// The probe must not read jumpgate's own items: an unlocked `secret-tool
// search` prints each matching item's secret.
func TestSecretServiceProbeMatchesNoJumpgateItem(t *testing.T) {
	f := &fakeRunner{}
	withRunner(t, f, "linux", "secret-tool")
	if !secretServiceAnswers(context.Background()) {
		t.Fatal("probe did not accept an answer")
	}
	if len(f.calls) != 1 {
		t.Fatalf("probe calls = %v, want one", f.calls)
	}
	got := strings.Join(f.calls[0], " ")
	if got != "secret-tool search service "+probeService || probeService == keychainService {
		t.Fatalf("probe ran %q; it must search a service no jumpgate key uses", got)
	}
}

// Older secret-tool builds report "nothing matched" as exit 1 with no output;
// that is still an answer from a live service.
func TestSecretServiceProbeAcceptsAnEmptyExitOne(t *testing.T) {
	f := &fakeRunner{fn: func(string, string, []string) (string, error) {
		return "", &cmdError{Name: "secret-tool", ExitCode: 1, Err: errors.New("exit status 1")}
	}}
	withRunner(t, f, "linux", "secret-tool")
	if !secretServiceAnswers(context.Background()) {
		t.Fatal("an empty exit 1 was not taken as an answer")
	}
}

// I-4: over SSH there is no D-Bus session; the error must say why and name
// the file store.
func TestKeychainOnLinuxWithoutDBusNamesTheFileStore(t *testing.T) {
	f := &fakeRunner{}
	withRunner(t, f, "linux", "secret-tool")
	withDBus(t, false)
	_, err := Create(context.Background(), StoreKeychain, "controller")
	if !errors.Is(err, ErrNoKeychain) || !strings.Contains(err.Error(), "D-Bus") || !strings.Contains(err.Error(), "--store file") {
		t.Fatalf("Create = %v; want ErrNoKeychain naming D-Bus and --store file", err)
	}
	_, err = Open(context.Background(), StoreKeychain, "controller")
	if !errors.Is(err, ErrNoKeychain) || !strings.Contains(err.Error(), "D-Bus") {
		t.Fatalf("Open = %v; want ErrNoKeychain naming D-Bus", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("ran secret-tool with no D-Bus session: %v", f.calls)
	}
}

// D17: a tool that never answers (a locked keyring with nothing to prompt)
// is cut off with a message saying what to do.
func TestRunCmdTimesOut(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	old := toolTimeout
	toolTimeout = 100 * time.Millisecond
	t.Cleanup(func() { toolTimeout = old })
	start := time.Now()
	_, err := runCmd(context.Background(), "", "sleep", "5")
	if err == nil || !strings.Contains(err.Error(), "did not answer within") || !strings.Contains(err.Error(), "--store file") {
		t.Fatalf("runCmd = %v; want the timeout message", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("runCmd did not stop the tool at the timeout")
	}
}

// A caller's own deadline (the probe's) is not the tool timeout, and must not
// be reported as one.
func TestRunCmdCallerDeadlineIsNotTheToolTimeout(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := runCmd(ctx, "", "sleep", "5")
	if err == nil || strings.Contains(err.Error(), "did not answer within") {
		t.Fatalf("runCmd = %v; want a plain cancellation", err)
	}
}
