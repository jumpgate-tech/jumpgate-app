//go:build livebox

// Live-box integration test for the full WireGuard server lifecycle. It is
// excluded from the normal build and CI by the `livebox` tag — it needs a real
// throwaway Linux host reachable over SSH with root and wireguard-tools, and it
// mutates that host (brings interfaces up, writes /etc/wireguard, tears them
// down). Run it deliberately against a disposable box:
//
//	JUMPGATE_LIVEBOX_HOST=203.0.113.7 \
//	JUMPGATE_LIVEBOX_USER=root \
//	JUMPGATE_LIVEBOX_KEY=~/.ssh/redacted-key-name \
//	JUMPGATE_LIVEBOX_PORT=2222 \
//	go test -tags livebox -run TestLiveBox -v ./internal/vpn/
//
// With the env unset the test SKIPS, so `go test -tags livebox ./...` on a
// machine without a box is a no-op rather than a failure.
//
// What it proves is the exact round trip the Private access UI drives, checked
// against ground truth on the host (`wg show`, file presence) at every step —
// not the app's own success reports, which this codebase has repeatedly seen
// lie (see the verify-by-running discipline): provision -> enroll -> disconnect
// (peer kept) -> reconnect (SAME peer survives) -> wipe (interface + conf + key
// all gone). The peer-survival check is the regression guard for the bug where
// reconnecting by re-provisioning silently dropped every enrolled device.
package vpn

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// liveExecutor dials the box from the JUMPGATE_LIVEBOX_* env, or skips the test
// if the host is not configured. The host key is trusted on first use into a
// throwaway known_hosts under t.TempDir, so the test needs no prior SSH state.
func liveExecutor(t *testing.T) executor.Executor {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("JUMPGATE_LIVEBOX_HOST"))
	if host == "" {
		t.Skip("JUMPGATE_LIVEBOX_HOST unset — skipping live-box integration test")
	}
	user := strings.TrimSpace(os.Getenv("JUMPGATE_LIVEBOX_USER"))
	if user == "" {
		user = "root"
	}
	key := strings.TrimSpace(os.Getenv("JUMPGATE_LIVEBOX_KEY"))
	if key == "" {
		t.Fatal("JUMPGATE_LIVEBOX_KEY must name the SSH private key for the box")
	}
	if strings.HasPrefix(key, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("resolve home for key path: %v", err)
		}
		key = filepath.Join(home, key[2:])
	}
	port := 22
	if p := strings.TrimSpace(os.Getenv("JUMPGATE_LIVEBOX_PORT")); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			t.Fatalf("JUMPGATE_LIVEBOX_PORT %q is not a number: %v", p, err)
		}
		port = n
	}
	ex, err := executor.NewSSH(executor.SSHConfig{
		Host:        host,
		User:        user,
		KeyPath:     key,
		HostKeyFile: filepath.Join(t.TempDir(), "known_hosts"),
		Port:        port,
	})
	if err != nil {
		t.Fatalf("dial live box %s@%s:%d: %v", user, host, port, err)
	}
	return ex
}

// remoteFileExists reports whether `path` exists on the host (via `test -f`),
// the ground-truth check for conf/key presence after disconnect vs. wipe.
func remoteFileExists(ctx context.Context, t *testing.T, ex executor.Executor, path string) bool {
	t.Helper()
	res, err := ex.Run(ctx, "test -f "+shellArg(path), nil)
	if err != nil {
		t.Fatalf("test -f %s: %v", path, err)
	}
	return res.ExitCode == 0
}

func TestLiveBox_ServerLifecycle_ProvisionEnrollDisconnectReconnectWipe(t *testing.T) {
	ex := liveExecutor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const iface = "jgtest0" // deliberately not jumpgate0, so a stray real server is never touched
	params := ServerParams{Iface: iface, Address: "10.77.0.1/24", ListenPort: 51999}

	// Safety net: whatever happens, leave the box clean. DeprovisionServer is
	// idempotent enough that a second teardown on an already-clean box is fine.
	defer func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		_ = DeprovisionServer(cctx, ex, iface)
	}()

	wg := WgQuick{Exec: ex, Iface: iface}
	confPath := serverConfPath(iface)
	keyPath := serverKeyPath(iface)

	// --- 1. Provision: interface up, listening, public key handed back -----
	info, err := ProvisionServer(ctx, ex, params)
	if err != nil {
		t.Fatalf("ProvisionServer: %v", err)
	}
	if info.PublicKey == "" {
		t.Error("provision returned an empty public key")
	}
	if info.ListenPort != params.ListenPort {
		t.Errorf("listen port = %d, want %d", info.ListenPort, params.ListenPort)
	}
	if st, err := wg.Status(ctx); err != nil {
		t.Fatalf("status after provision: %v", err)
	} else if !st.Up {
		t.Fatal("interface is not up after provision (host ground truth)")
	}
	if !remoteFileExists(ctx, t, ex, confPath) {
		t.Errorf("conf %s absent after provision", confPath)
	}

	// --- 2. Enroll a device: peer count goes to 1 on the host --------------
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	peerIP, err := NextPeerIP(params.Address, nil)
	if err != nil {
		t.Fatalf("NextPeerIP: %v", err)
	}
	if err := AddPeer(ctx, ex, AddPeerParams{
		Iface:         iface,
		PeerPublicKey: key.PublicKey,
		AllowedIP:     peerIP,
	}); err != nil {
		t.Fatalf("AddPeer: %v", err)
	}
	if st, err := wg.Status(ctx); err != nil {
		t.Fatalf("status after enroll: %v", err)
	} else if st.Peers != 1 {
		t.Fatalf("peer count = %d after enroll, want 1", st.Peers)
	}

	// --- 3. Disconnect: interface down, but conf + peer KEPT ---------------
	if err := wg.Down(ctx); err != nil {
		t.Fatalf("Down (disconnect): %v", err)
	}
	if st, err := wg.Status(ctx); err != nil {
		t.Fatalf("status after disconnect: %v", err)
	} else if st.Up {
		t.Fatal("interface still up after disconnect")
	}
	if !remoteFileExists(ctx, t, ex, confPath) {
		t.Error("conf was removed by disconnect — disconnect must be reversible, not a wipe")
	}

	// --- 4. Reconnect: interface up, SAME peer survives --------------------
	// This is the regression guard: reconnecting must NOT re-provision (which
	// would rewrite the conf from a peerless template and drop the device).
	if err := StartServer(ctx, ex, iface); err != nil {
		t.Fatalf("StartServer (reconnect): %v", err)
	}
	if st, err := wg.Status(ctx); err != nil {
		t.Fatalf("status after reconnect: %v", err)
	} else if !st.Up {
		t.Fatal("interface is not up after reconnect")
	} else if st.Peers != 1 {
		t.Fatalf("peer count = %d after reconnect, want 1 — the enrolled device was dropped", st.Peers)
	}

	// --- 5. Wipe: interface gone AND conf + key deleted from the host ------
	if err := DeprovisionServer(ctx, ex, iface); err != nil {
		t.Fatalf("DeprovisionServer (wipe): %v", err)
	}
	if st, err := wg.Status(ctx); err != nil {
		t.Fatalf("status after wipe: %v", err)
	} else if st.Up {
		t.Fatal("interface still up after wipe")
	}
	if remoteFileExists(ctx, t, ex, confPath) {
		t.Errorf("conf %s still present after wipe", confPath)
	}
	if remoteFileExists(ctx, t, ex, keyPath) {
		t.Errorf("private key %s still present after wipe", keyPath)
	}
}
