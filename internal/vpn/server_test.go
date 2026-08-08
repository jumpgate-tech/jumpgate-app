package vpn

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

const testServerPub = "kxUUFA0cGVlclB1YmxpY0tleXNlcnZlclB1YmtleT0="

// scriptHealthyServer wires a fake host where every provisioning command
// succeeds and the interface comes up listening on 51820.
func scriptHealthyServer() *fakeExecutor {
	return newFake().
		script("id -u", executor.Result{Stdout: "0\n"}).
		script("wg pubkey", executor.Result{Stdout: testServerPub + "\n"}).
		script("dump", executor.Result{ExitCode: 0, Stdout: wgDumpUp}).
		script("listen-port", executor.Result{Stdout: "51820\n"})
}

func healthyParams() ServerParams {
	return ServerParams{Iface: "jumpgate0", Address: "10.9.0.1/24", ListenPort: 51820}
}

func TestProvisionServer_HappyPath(t *testing.T) {
	f := scriptHealthyServer()
	info, err := ProvisionServer(context.Background(), f, healthyParams())
	if err != nil {
		t.Fatalf("ProvisionServer: %v", err)
	}
	if info.PublicKey != testServerPub {
		t.Errorf("PublicKey = %q, want the derived server key", info.PublicKey)
	}
	if info.ListenPort != 51820 || info.Address != "10.9.0.1/24" {
		t.Errorf("info = %+v, want port 51820 / addr 10.9.0.1/24", info)
	}
	if !strings.Contains(info.FirewallHint, "51820/udp") {
		t.Errorf("FirewallHint = %q, want it to name 51820/udp", info.FirewallHint)
	}
	if !f.called("wg-quick up 'jumpgate0'") {
		t.Errorf("wg-quick up was not invoked; calls=%v", f.calls)
	}
}

// Key custody: the server's private key is generated on the host and never
// transits this app. The proof is in the commands — the key is created with an
// idempotent guard, and the conf is built on the host by reading the key file
// ($(cat …)), so no command this app issues ever carries the key's bytes.
func TestProvisionServer_PrivateKeyNeverLeavesHost(t *testing.T) {
	f := scriptHealthyServer()
	if _, err := ProvisionServer(context.Background(), f, healthyParams()); err != nil {
		t.Fatalf("ProvisionServer: %v", err)
	}

	var genGuarded, confFromFile bool
	for _, c := range f.calls {
		if strings.Contains(c, "[ -f") && strings.Contains(c, "wg genkey") {
			genGuarded = true // idempotent: only generate if absent
		}
		if strings.Contains(c, "printf") && strings.Contains(c, "$(cat") && strings.Contains(c, ".conf") {
			confFromFile = true // conf built on host from the key FILE, not a value we hold
		}
	}
	if !genGuarded {
		t.Errorf("key generation was not guarded by an existence check; calls=%v", f.calls)
	}
	if !confFromFile {
		t.Errorf("conf was not built on the host from the key file; calls=%v", f.calls)
	}
	// The app never writes a file itself here — everything is built on the host,
	// so it cannot be the path a private key leaks through.
	if len(f.files) != 0 {
		t.Errorf("app wrote files directly (%v); the server conf must be built on the host to keep the key there", f.files)
	}
}

// The load-bearing verify: wg-quick exits 0 but the interface never came up
// (wg show fails). Provisioning must FAIL, not report a server that is not there.
func TestProvisionServer_FailsWhenInterfaceAbsentDespiteExit0(t *testing.T) {
	f := scriptHealthyServer().script("dump", executor.Result{ExitCode: 1, Stderr: "No such device"})
	if _, err := ProvisionServer(context.Background(), f, healthyParams()); err == nil {
		t.Fatalf("ProvisionServer reported success even though the interface is absent")
	}
}

// The interface is up but not listening on the requested port — a server in
// name only. Must fail.
func TestProvisionServer_FailsWhenNotListeningOnRequestedPort(t *testing.T) {
	f := scriptHealthyServer().script("listen-port", executor.Result{Stdout: "0\n"})
	_, err := ProvisionServer(context.Background(), f, healthyParams())
	if err == nil {
		t.Fatalf("expected failure when wg is not listening on the requested port")
	}
	if !strings.Contains(err.Error(), "listening on 0") {
		t.Errorf("error = %v, want it to name the wrong listen port", err)
	}
}

func TestProvisionServer_RequiresRoot(t *testing.T) {
	f := scriptHealthyServer().script("id -u", executor.Result{Stdout: "1000\n"})
	_, err := ProvisionServer(context.Background(), f, healthyParams())
	if err == nil || !strings.Contains(err.Error(), "root") {
		t.Fatalf("expected a root-required error, got %v", err)
	}
	if f.called("wg-quick up") {
		t.Errorf("brought the interface up despite not being root")
	}
}

// wireguard-tools missing and no apt to install it: an actionable error, and
// nothing is brought up.
func TestProvisionServer_FailsWhenWireGuardMissingAndNoApt(t *testing.T) {
	f := scriptHealthyServer().
		script("command -v wg >/dev/null", executor.Result{ExitCode: 1}).
		script("command -v apt-get", executor.Result{ExitCode: 1})
	_, err := ProvisionServer(context.Background(), f, healthyParams())
	if err == nil || !strings.Contains(err.Error(), "wireguard-tools") {
		t.Fatalf("expected a wireguard-tools error, got %v", err)
	}
	if f.called("wg-quick up") {
		t.Errorf("tried to bring up a server without wireguard-tools present")
	}
}

func TestStartServer_BringsUpExistingConfWithoutRewriting(t *testing.T) {
	f := newFake().script("wg show", executor.Result{ExitCode: 0, Stdout: wgDumpUp})
	if err := StartServer(context.Background(), f, "jumpgate0"); err != nil {
		t.Fatalf("StartServer: %v", err)
	}
	if !f.called("wg-quick up 'jumpgate0'") {
		t.Errorf("did not bring the interface up; calls=%v", f.calls)
	}
	// The whole point: it must NOT rewrite the conf (no printf), or it would drop
	// peers that wg-quick save persisted.
	if f.called("printf") {
		t.Errorf("StartServer rewrote the conf — that clobbers enrolled peers; calls=%v", f.calls)
	}
}

func TestStartServer_FailsWhenNoConf(t *testing.T) {
	f := newFake().script("test -f", executor.Result{ExitCode: 1})
	if err := StartServer(context.Background(), f, "jumpgate0"); err == nil {
		t.Fatalf("expected error when there is no conf to bring up")
	}
	if f.called("wg-quick up") {
		t.Errorf("tried to bring up a server with no conf")
	}
}

func TestDeprovisionServer_DownRemoveVerify(t *testing.T) {
	// After teardown, wg show reports the interface absent (exit 1).
	f := newFake().script("wg show", executor.Result{ExitCode: 1, Stderr: "No such device"})
	if err := DeprovisionServer(context.Background(), f, "jumpgate0"); err != nil {
		t.Fatalf("DeprovisionServer: %v", err)
	}
	if !f.called("wg-quick down 'jumpgate0'") {
		t.Errorf("did not bring the interface down; calls=%v", f.calls)
	}
	if !f.called("rm -f") || !f.called("jumpgate0.conf") || !f.called("jumpgate0.privatekey") {
		t.Errorf("did not remove conf+key; calls=%v", f.calls)
	}
}

// The load-bearing verify: teardown ran but wg show still reports the interface
// up → must FAIL, not silently claim the wipe worked.
func TestDeprovisionServer_FailsIfStillUp(t *testing.T) {
	f := newFake().script("wg show", executor.Result{ExitCode: 0, Stdout: wgDumpUp})
	if err := DeprovisionServer(context.Background(), f, "jumpgate0"); err == nil {
		t.Fatalf("expected error when the interface is still up after deprovision")
	}
}

func TestProvisionServer_RejectsBadParams(t *testing.T) {
	cases := map[string]ServerParams{
		"empty iface": {Iface: "", Address: "10.9.0.1/24", ListenPort: 51820},
		"bad iface":   {Iface: "has space", Address: "10.9.0.1/24", ListenPort: 51820},
		"bad address": {Iface: "jumpgate0", Address: "not-a-cidr", ListenPort: 51820},
		"bad port":    {Iface: "jumpgate0", Address: "10.9.0.1/24", ListenPort: 0},
	}
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			f := scriptHealthyServer()
			if _, err := ProvisionServer(context.Background(), f, p); err == nil {
				t.Fatalf("ProvisionServer accepted invalid params %+v", p)
			}
			if len(f.calls) != 0 {
				t.Errorf("ran commands (%v) before validating params", f.calls)
			}
		})
	}
}
