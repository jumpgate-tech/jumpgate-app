package vpn

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ServerParams describes a WireGuard server to bring up on a host — the
// "provision on a device" half of the easy button (the other half is applying a
// bring-your-own client .conf; see WgQuick + StaticProvider).
type ServerParams struct {
	Iface      string // interface name, e.g. "jumpgate0"
	Address    string // the server's own overlay address WITH mask, e.g. "10.9.0.1/24"
	ListenPort int    // UDP port peers dial, e.g. 51820
}

// ServerInfo is the result of provisioning: everything needed to enroll a
// device and NOTHING secret. The server's private key is generated on the host
// and never leaves it — only the PUBLIC key travels back here (see
// ProvisionServer's key handling).
type ServerInfo struct {
	Iface      string
	PublicKey  string
	ListenPort int
	Address    string // overlay address (with mask) the server holds

	// FirewallHint is the command an operator must run to admit peers. This app
	// never mutates the firewall itself — it reports the fix and lets the
	// operator apply it (the same stance internal/ops FirewallChecklist takes:
	// no `ufw allow` / `iptables -A` ever appears in a command this app runs) —
	// so opening the UDP port is handed back rather than done.
	FirewallHint string
}

func serverConfPath(iface string) string { return "/etc/wireguard/" + iface + ".conf" }
func serverKeyPath(iface string) string  { return "/etc/wireguard/" + iface + ".privatekey" }

func (p ServerParams) validate() error {
	if !validIfaceName(p.Iface) {
		return fmt.Errorf("vpn: server interface name %q is invalid — up to 15 of letters, digits, dot, dash or underscore", p.Iface)
	}
	if _, _, err := net.ParseCIDR(strings.TrimSpace(p.Address)); err != nil {
		return fmt.Errorf("vpn: server address %q must be a CIDR like 10.9.0.1/24: %w", p.Address, err)
	}
	if p.ListenPort < 1 || p.ListenPort > 65535 {
		return fmt.Errorf("vpn: server listen port %d is out of range (1-65535)", p.ListenPort)
	}
	return nil
}

// ProvisionServer sets up (or repairs) a WireGuard server on the host behind
// exec, which may be local or SSH — the same executor seam every other host
// operation uses, so "provision on this laptop" and "provision on a fleet box
// over SSH" are one code path.
//
// It is idempotent: the server's key is generated only if absent, so
// re-provisioning keeps the server's identity (and therefore every peer config
// already handed out stays valid). The private key is generated ON THE HOST and
// never transits this app — only the public key is read back.
//
// It VERIFIES rather than trusts: a wg-quick that exits 0 can still have failed,
// and an interface that is up but not listening is not a server — so success is
// reported only after `wg show` confirms both the interface and the listen port
// (this app's recurring failure mode is success reported over a broken state).
//
// It does NOT touch the firewall; the UDP port an operator must open comes back
// in ServerInfo.FirewallHint.
func ProvisionServer(ctx context.Context, exec executor.Executor, p ServerParams) (ServerInfo, error) {
	if err := p.validate(); err != nil {
		return ServerInfo{}, err
	}
	if err := requireRoot(ctx, exec); err != nil {
		return ServerInfo{}, err
	}
	if err := ensureWireGuard(ctx, exec); err != nil {
		return ServerInfo{}, err
	}

	pub, err := ensureServerKey(ctx, exec, p.Iface)
	if err != nil {
		return ServerInfo{}, err
	}
	if err := writeServerConf(ctx, exec, p); err != nil {
		return ServerInfo{}, err
	}

	// (Re)bring up: down first, ignoring "not a WireGuard interface", so a
	// re-provision with an edited conf actually takes effect, then up.
	_ = (WgQuick{Exec: exec, Iface: p.Iface}).Down(ctx)
	res, err := exec.Run(ctx, "wg-quick up "+shellArg(p.Iface), nil)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("vpn: wg-quick up: %w", err)
	}
	if res.ExitCode != 0 {
		return ServerInfo{}, fmt.Errorf("vpn: wg-quick up exited %d: %s", res.ExitCode, firstLine(res.Stderr))
	}

	// VERIFY: interface present.
	st, err := (WgQuick{Exec: exec, Iface: p.Iface}).Status(ctx)
	if err != nil {
		return ServerInfo{}, err
	}
	if !st.Up {
		return ServerInfo{}, fmt.Errorf("vpn: wg-quick up reported success but interface %q is not present", p.Iface)
	}
	// VERIFY: actually listening on the requested port. An interface that came
	// up but is not listening would accept no peers — a server in name only.
	listen, err := serverListenPort(ctx, exec, p.Iface)
	if err != nil {
		return ServerInfo{}, err
	}
	if listen != p.ListenPort {
		return ServerInfo{}, fmt.Errorf("vpn: interface %q is up but listening on %d, not the requested %d", p.Iface, listen, p.ListenPort)
	}

	return ServerInfo{
		Iface:        p.Iface,
		PublicKey:    pub,
		ListenPort:   p.ListenPort,
		Address:      strings.TrimSpace(p.Address),
		FirewallHint: fmt.Sprintf("ufw allow %d/udp", p.ListenPort),
	}, nil
}

// DeprovisionServer is the exact reverse of ProvisionServer: it brings the
// interface down and removes the server's on-host config and private key, then
// VERIFIES the interface is gone. This is the teardown behind a "wipe" — after
// it, the host is as it was before provisioning (no WireGuard left for this
// iface).
//
// It is idempotent: wiping a server that is already down/absent is not an error,
// because the end state (gone) is exactly what was asked for. That is the
// counterpart to "disconnect" (just `wg-quick down`, which leaves the conf and
// key in place so a reconnect brings the same identity back up) — deprovision
// removes them, so a later provision on the same iface mints a NEW key.
func DeprovisionServer(ctx context.Context, exec executor.Executor, iface string) error {
	if !validIfaceName(iface) {
		return fmt.Errorf("vpn: interface name %q is invalid", iface)
	}
	// Bring it down (Down already treats "not a WireGuard interface" as success).
	if err := (WgQuick{Exec: exec, Iface: iface}).Down(ctx); err != nil {
		return err
	}
	// Remove the conf and the key. rm -f so an already-absent file is fine.
	rm := "rm -f " + shellArg(serverConfPath(iface)) + " " + shellArg(serverKeyPath(iface))
	if res, err := exec.Run(ctx, rm, nil); err != nil {
		return fmt.Errorf("vpn: removing server files: %w", err)
	} else if res.ExitCode != 0 {
		return fmt.Errorf("vpn: removing server files exited %d: %s", res.ExitCode, firstLine(res.Stderr))
	}
	// VERIFY the interface is really gone — a wipe that left the tunnel up is the
	// worst outcome (this app reports success only over a confirmed end state).
	st, err := (WgQuick{Exec: exec, Iface: iface}).Status(ctx)
	if err != nil {
		return err
	}
	if st.Up {
		return fmt.Errorf("vpn: deprovision ran but interface %q is still up", iface)
	}
	return nil
}

// StartServer brings an already-provisioned server back UP from its existing
// on-host conf — the reverse of a disconnect (Down). Crucially, unlike
// ProvisionServer it does NOT rewrite the conf, so peers enrolled since
// provisioning (which AddPeer persisted with `wg-quick save`) survive a
// disconnect/reconnect. Reconnecting through ProvisionServer instead would
// overwrite the conf with the peerless template and silently drop every device.
//
// It refuses when there is no conf to bring up (a wiped server) rather than
// quietly minting a fresh, peerless, new-keyed server in its place. Verifies the
// interface actually came up.
func StartServer(ctx context.Context, exec executor.Executor, iface string) error {
	if !validIfaceName(iface) {
		return fmt.Errorf("vpn: interface name %q is invalid", iface)
	}
	if res, err := exec.Run(ctx, "test -f "+shellArg(serverConfPath(iface)), nil); err != nil {
		return fmt.Errorf("vpn: checking for server config: %w", err)
	} else if res.ExitCode != 0 {
		return fmt.Errorf("vpn: no config for %q on the host — nothing to bring up (re-provision instead)", iface)
	}
	// Down first (idempotent) in case it is half-up, then up from the existing conf.
	_ = (WgQuick{Exec: exec, Iface: iface}).Down(ctx)
	res, err := exec.Run(ctx, "wg-quick up "+shellArg(iface), nil)
	if err != nil {
		return fmt.Errorf("vpn: wg-quick up: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("vpn: wg-quick up exited %d: %s", res.ExitCode, firstLine(res.Stderr))
	}
	st, err := (WgQuick{Exec: exec, Iface: iface}).Status(ctx)
	if err != nil {
		return err
	}
	if !st.Up {
		return fmt.Errorf("vpn: wg-quick up reported success but interface %q is not present", iface)
	}
	return nil
}

// requireRoot fails early, with an actionable message, when the target is not
// root — the same stance setup's preflight takes, and for the same reasons:
// writing /etc/wireguard and running wg-quick both need it.
func requireRoot(ctx context.Context, exec executor.Executor) error {
	res, err := exec.Run(ctx, "id -u", nil)
	if err != nil {
		return fmt.Errorf("vpn: id -u: %w", err)
	}
	if uid := strings.TrimSpace(res.Stdout); uid != "0" {
		return fmt.Errorf("vpn: provisioning a WireGuard server needs root on the target (to write /etc/wireguard and run wg-quick) — SSH as root, or run in local mode as root; id -u reported %q", uid)
	}
	return nil
}

// ensureWireGuard makes sure `wg` and `wg-quick` are present, installing
// wireguard-tools on apt-based hosts as a best effort. On a host with neither
// the tools nor apt it returns an actionable error rather than guessing at a
// package manager — installing is additive and needed for the feature (unlike
// the firewall, which this app never touches), but it is not worth a distro
// guessing game.
func ensureWireGuard(ctx context.Context, exec executor.Executor) error {
	present := func() bool {
		res, err := exec.Run(ctx, "command -v wg >/dev/null 2>&1 && command -v wg-quick >/dev/null 2>&1", nil)
		return err == nil && res.ExitCode == 0
	}
	if present() {
		return nil
	}
	if res, err := exec.Run(ctx, "command -v apt-get >/dev/null 2>&1", nil); err == nil && res.ExitCode == 0 {
		install := "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y wireguard-tools"
		if res, err := exec.Run(ctx, install, nil); err != nil {
			return fmt.Errorf("vpn: installing wireguard-tools: %w", err)
		} else if res.ExitCode != 0 {
			return fmt.Errorf("vpn: installing wireguard-tools failed (exit %d): %s", res.ExitCode, firstLine(res.Stderr))
		}
	}
	if !present() {
		return fmt.Errorf("vpn: wireguard-tools is not installed on the target and could not be installed automatically — install it (for example `apt-get install wireguard-tools` or `dnf install wireguard-tools`) and try again")
	}
	return nil
}

// ensureServerKey generates the server's private key ON THE HOST if it is not
// already there, then returns its PUBLIC key. The private key is written to a
// 0600 file (umask 077) and is never read back into this app — only `wg pubkey`
// reads it, on the host, to derive the public half. The `[ -f ] ||` guard makes
// this idempotent: re-provisioning keeps the existing identity.
func ensureServerKey(ctx context.Context, exec executor.Executor, iface string) (string, error) {
	key := serverKeyPath(iface)
	gen := fmt.Sprintf("umask 077; mkdir -p /etc/wireguard; [ -f %s ] || wg genkey > %s", shellArg(key), shellArg(key))
	if res, err := exec.Run(ctx, gen, nil); err != nil {
		return "", fmt.Errorf("vpn: generating server key: %w", err)
	} else if res.ExitCode != 0 {
		return "", fmt.Errorf("vpn: generating server key failed (exit %d): %s", res.ExitCode, firstLine(res.Stderr))
	}
	res, err := exec.Run(ctx, "wg pubkey < "+shellArg(key), nil)
	if err != nil {
		return "", fmt.Errorf("vpn: deriving server public key: %w", err)
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("vpn: deriving server public key failed (exit %d): %s", res.ExitCode, firstLine(res.Stderr))
	}
	pub := strings.TrimSpace(res.Stdout)
	if pub == "" {
		return "", fmt.Errorf("vpn: server public key came back empty")
	}
	return pub, nil
}

// writeServerConf writes /etc/wireguard/<iface>.conf ON THE HOST, reading the
// private key from its on-host file via $(cat …) so the key never transits this
// app. The Address and ListenPort are our own validated params (a CIDR and an
// int), so they are safe to interpolate into the format string; only the key —
// the one untrusted-to-this-app value — comes from the shell, as printf's
// argument, never the format.
func writeServerConf(ctx context.Context, exec executor.Executor, p ServerParams) error {
	conf := serverConfPath(p.Iface)
	key := serverKeyPath(p.Iface)
	build := fmt.Sprintf(
		"umask 077; printf '[Interface]\\nPrivateKey = %%s\\nAddress = %s\\nListenPort = %d\\n' \"$(cat %s)\" > %s",
		strings.TrimSpace(p.Address), p.ListenPort, shellArg(key), shellArg(conf))
	if res, err := exec.Run(ctx, build, nil); err != nil {
		return fmt.Errorf("vpn: writing %s: %w", conf, err)
	} else if res.ExitCode != 0 {
		return fmt.Errorf("vpn: writing %s failed (exit %d): %s", conf, res.ExitCode, firstLine(res.Stderr))
	}
	return nil
}

// serverListenPort reads the port wg is actually listening on, the load-bearing
// half of "the server really came up".
func serverListenPort(ctx context.Context, exec executor.Executor, iface string) (int, error) {
	res, err := exec.Run(ctx, "wg show "+shellArg(iface)+" listen-port", nil)
	if err != nil {
		return 0, fmt.Errorf("vpn: wg show listen-port: %w", err)
	}
	if res.ExitCode != 0 {
		return 0, fmt.Errorf("vpn: wg show listen-port exited %d: %s", res.ExitCode, firstLine(res.Stderr))
	}
	port, err := strconv.Atoi(strings.TrimSpace(res.Stdout))
	if err != nil {
		return 0, fmt.Errorf("vpn: wg reported a non-numeric listen port %q", strings.TrimSpace(res.Stdout))
	}
	return port, nil
}

// validIfaceName accepts what the kernel will: at most 15 characters
// (IFNAMSIZ-1) of a safe alphabet.
func validIfaceName(s string) bool {
	if s == "" || len(s) > 15 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
