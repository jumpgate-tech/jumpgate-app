package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestDesktopVMKind(t *testing.T) {
	cases := []struct {
		name, procVersion, hostname string
		limaDir                     bool
		want                        string
	}{
		{"wsl2", "Linux version 5.15.153.1-microsoft-standard-WSL2 (root@x) (gcc)", "DESKTOP-ABC", false, "WSL2"},
		{"wsl mixed case", "Linux version 6.6 Microsoft", "pc", false, "WSL2"},
		{"lima hostname", "Linux version 6.8.0-ubuntu", "lima-default", false, "Lima"},
		{"lima dir", "Linux version 6.8.0-ubuntu", "ubuntu", true, "Lima"},
		{"plain linux", "Linux version 6.8.0-ubuntu", "box-a", false, ""},
		{"empty", "", "", false, ""},
	}
	for _, c := range cases {
		if got := desktopVMKind(c.procVersion, c.hostname, c.limaDir); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPreflight_WarnsOnDesktopVMButDoesNotBlock(t *testing.T) {
	e := rootUIDScript(newFakeExecutor()).
		script("uname", executor.Result{Stdout: "Linux\n"}).
		script("cat /proc/version", executor.Result{Stdout: "Linux version 5.15-microsoft-standard-WSL2\n"}).
		script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n9999999999999\n"}).
		script("ss -ltn", executor.Result{Stdout: "State  Recv-Q Send-Q Local Address:Port\n"})
	events := make(chan Event, 16)
	st := &State{Wire: testWire(), Events: events}

	if err := preflightStep().Verify(context.Background(), e, st); err != nil {
		t.Fatalf("a desktop VM must only warn, got %v", err)
	}
	close(events)
	var lines []string
	for ev := range events {
		lines = append(lines, ev.Line)
	}
	got := strings.Join(lines, "\n")
	for _, want := range []string{"WSL2", "sleep", "NAT", "TB"} {
		if !strings.Contains(got, want) {
			t.Errorf("warning should mention %q, got:\n%s", want, got)
		}
	}
}

func TestPreflight_NoWarningOnPlainLinux(t *testing.T) {
	e := rootUIDScript(newFakeExecutor()).
		script("uname", executor.Result{Stdout: "Linux\n"}).
		script("cat /proc/version", executor.Result{Stdout: "Linux version 6.8.0-ubuntu\n"}).
		script("hostname", executor.Result{Stdout: "box-a\n"}).
		script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n9999999999999\n"}).
		script("ss -ltn", executor.Result{Stdout: "State  Recv-Q Send-Q Local Address:Port\n"})
	events := make(chan Event, 16)
	if err := preflightStep().Verify(context.Background(), e, &State{Wire: testWire(), Events: events}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("plain Linux should emit no warning, got %d events", len(events))
	}
}
