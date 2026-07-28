package setup

// Every command the node plan issues, broken one at a time, on the same
// contract the gateway table uses: the original cause must reach the top via
// errors.Is. A setup step that turns a dropped connection into its own tidy
// error is indistinguishable from one that worked, and this plan installs a
// chain client — "it reported success" is the expensive way to find out.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// nodeReady is a Linux root target where every probe answers the way a
// healthy, fully-provisioned node does: the disk is large, the service user
// exists, both binaries run, the units are in place and both endpoints answer.
func nodeReady() *fakeExecutor {
	return newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n"}).
		script("id -u", executor.Result{Stdout: "0\n"}).
		script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n4000000000000\n"}).
		script("id -u valve-node-app", executor.Result{Stdout: "997\n"}).
		script("stat -c %U", executor.Result{Stdout: "valve-node-app\n"}).
		script("systemctl is-active", executor.Result{Stdout: "active\nactive\n"}).
		script("eth/v1/node/syncing", executor.Result{Stdout: "200"}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("journalctl", executor.Result{Stdout: ""})
}

// shrinkHandshakeWait keeps the handshake poll out of real time.
func shrinkHandshakeWait(t *testing.T) {
	t.Helper()
	oldTimeout, oldInterval := handshakeTimeout, handshakePollInterval
	handshakeTimeout = 5 * time.Millisecond
	handshakePollInterval = time.Millisecond
	t.Cleanup(func() { handshakeTimeout, handshakePollInterval = oldTimeout, oldInterval })
}

func TestNodePlan_CommandFailuresAllSurface(t *testing.T) {
	shrinkHandshakeWait(t)
	boom := errors.New("ssh: connection lost")

	breaks := []struct {
		name    string
		cmd     string
		wantSay string
	}{
		{name: "reading the OS", cmd: "uname", wantSay: "uname"},
		{name: "reading the uid", cmd: "id -u", wantSay: "id -u"},
		{name: "measuring free space", cmd: "df -B1 --output=avail", wantSay: "df"},
		{name: "creating the service user", cmd: "useradd", wantSay: "user"},
		{name: "probing git", cmd: "command -v git", wantSay: "git"},
		{name: "building reth", cmd: "git clone --depth 1 https://github.com/valve-tech/reth.git", wantSay: "reth"},
		{name: "checking the reth binary", cmd: "test -x '/usr/local/bin/reth'", wantSay: "reth"},
		{name: "building lighthouse", cmd: "/tmp/build-lighthouse-pulse", wantSay: "lighthouse"},
		{name: "checking the JWT", cmd: "test -f '/mnt/reth/jwt.hex'", wantSay: "jwt"},
		{name: "stopping the services before chown", cmd: "systemctl stop", wantSay: "chown"},
		{name: "owning the data tree", cmd: "chown -R valve-node-app", wantSay: "chown"},
		{name: "driving systemctl", cmd: "systemctl daemon-reload", wantSay: "systemctl"},
		{name: "reading the data dir's owner", cmd: "stat -c %U", wantSay: "owner"},
		{name: "starting the services", cmd: "systemctl start", wantSay: "start"},
		{name: "probing the beacon", cmd: "eth/v1/node/syncing", wantSay: "beacon"},
		{name: "probing the exec client", cmd: "eth_syncing", wantSay: "exec"},
		{name: "reading the journal", cmd: "journalctl", wantSay: "journalctl"},
	}

	for _, tc := range breaks {
		t.Run(tc.name, func(t *testing.T) {
			e := nodeReady()
			e.errOn(tc.cmd, boom)

			steps, err := Plan(testWire())
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			err = runPlan(t, steps, e, &State{Wire: testWire()})
			if err == nil {
				t.Fatalf("breaking %q set the node up successfully", tc.cmd)
			}
			if !errors.Is(err, boom) {
				t.Errorf("the transport cause did not reach the top: %v", err)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.wantSay) {
				t.Errorf("error does not say what it was doing (want %q): %v", tc.wantSay, err)
			}
		})
	}
}

// The unit files are the whole install: a plan that cannot write them and
// reports success leaves a machine with binaries and no services.
func TestNodePlan_UnwritableUnitsSurface(t *testing.T) {
	shrinkHandshakeWait(t)
	full := errors.New("write /dev/sda1: no space left on device")

	for _, unit := range []string{"valve-node-app-exec.service", "valve-node-app-beacon.service"} {
		t.Run(unit, func(t *testing.T) {
			e := readOnlyTarget{fakeExecutor: nodeReady(), deny: unit, err: full}

			steps, err := Plan(testWire())
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			err = runPlan(t, steps, e, &State{Wire: testWire()})
			if err == nil {
				t.Fatalf("a node whose %s could not be written set up successfully", unit)
			}
			if !errors.Is(err, full) {
				t.Errorf("the write failure did not reach the top: %v", err)
			}
			if !strings.Contains(err.Error(), unit) {
				t.Errorf("error does not name the unit: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------
// Plan's own validation
// ---------------------------------------------------------------------

// Plan refuses an unknown or miscategorised client BEFORE any step runs, so
// an invalid pair fails immediately rather than partway through RunAll with
// a half-built machine.
func TestPlan_RefusesClientsItCannotWire(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*catalog.WireConfig)
		wantSay string
	}{
		{
			name:    "unknown execution client",
			mutate:  func(w *catalog.WireConfig) { w.ExecID = "not-a-client" },
			wantSay: "execution client",
		},
		{
			name:    "unknown beacon client",
			mutate:  func(w *catalog.WireConfig) { w.BeaconID = "not-a-client" },
			wantSay: "beacon client",
		},
		{
			// A beacon id in the exec slot is the mistake worth catching:
			// both are real clients, so only the KIND check rejects it.
			name:    "a beacon in the execution slot",
			mutate:  func(w *catalog.WireConfig) { w.ExecID = "lighthouse-pulse" },
			wantSay: "execution client",
		},
		{
			name:    "an execution client in the beacon slot",
			mutate:  func(w *catalog.WireConfig) { w.BeaconID = "reth" },
			wantSay: "beacon client",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := testWire()
			tc.mutate(&w)

			steps, err := Plan(w)
			if err == nil {
				t.Fatalf("Plan accepted it and produced %d steps", len(steps))
			}
			if !strings.Contains(err.Error(), tc.wantSay) {
				t.Errorf("error does not say which slot is wrong (want %q): %v", tc.wantSay, err)
			}
		})
	}
}

// humanBytes' sub-kilobyte branch, which is the one a nearly-full disk hits
// and therefore the one an operator actually reads.
func TestHumanBytes_SmallValuesStayInBytes(t *testing.T) {
	if got := humanBytes(0); got != "0 B" {
		t.Errorf("got %q, want %q", got, "0 B")
	}
	if got := humanBytes(999); got != "999 B" {
		t.Errorf("got %q, want %q", got, "999 B")
	}
	// The boundary flips to the next unit rather than reading "1000 B".
	if got := humanBytes(1000); strings.HasSuffix(got, " B") {
		t.Errorf("got %q, want it promoted to the next unit", got)
	}
}
