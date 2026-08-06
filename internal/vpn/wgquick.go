package vpn

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// WgQuick brings a tunnel up on a host via `wg-quick`, run through an
// executor.Executor — so it works identically whether the gateway host is
// local or reached over SSH, the same seam every other host operation in this
// app uses. This is the first backend; a userspace `wireguard-go` backend
// (no root, cross-platform desktop) can implement Tunnel later without any
// change to callers or providers.
type WgQuick struct {
	Exec     executor.Executor
	Iface    string // interface name, e.g. "jumpgate0"
	Provider Provider
}

// confPath is where wg-quick expects the config for this interface.
func (w WgQuick) confPath() string { return "/etc/wireguard/" + w.Iface + ".conf" }

// Up writes the provider's config and brings the interface up, then VERIFIES
// the interface actually exists by reading its status back — a wg-quick that
// exits 0 can still have failed to configure, so the exit code alone is not
// trusted (this app's recurring failure mode is success reported over a broken
// state).
func (w WgQuick) Up(ctx context.Context) (State, error) {
	if w.Iface == "" {
		return State{}, fmt.Errorf("vpn: WgQuick.Iface is empty")
	}
	if w.Provider == nil {
		return State{}, fmt.Errorf("vpn: WgQuick.Provider is nil")
	}
	cfg, err := w.Provider.Config(ctx)
	if err != nil {
		return State{}, fmt.Errorf("vpn: provider %q: %w", w.Provider.Name(), err)
	}
	if err := cfg.Validate(); err != nil {
		return State{}, err
	}
	if err := w.Exec.WriteFile(ctx, w.confPath(), []byte(cfg.Render()), 0o600); err != nil {
		return State{}, fmt.Errorf("vpn: writing %s: %w", w.confPath(), err)
	}
	res, err := w.Exec.Run(ctx, "wg-quick up "+shellArg(w.Iface), nil)
	if err != nil {
		return State{}, fmt.Errorf("vpn: wg-quick up: %w", err)
	}
	if res.ExitCode != 0 {
		return State{}, fmt.Errorf("vpn: wg-quick up exited %d: %s", res.ExitCode, firstLine(res.Stderr))
	}
	st, err := w.Status(ctx)
	if err != nil {
		return State{}, err
	}
	if !st.Up {
		// wg-quick claimed success but the interface isn't there.
		return State{}, fmt.Errorf("vpn: wg-quick up reported success but interface %q is not present", w.Iface)
	}
	st.Provider = w.Provider.Name()
	st.Addresses = cfg.OverlayCIDRs()
	return st, nil
}

// Down tears the interface down. A "not running" teardown is not an error.
func (w WgQuick) Down(ctx context.Context) error {
	res, err := w.Exec.Run(ctx, "wg-quick down "+shellArg(w.Iface), nil)
	if err != nil {
		return fmt.Errorf("vpn: wg-quick down: %w", err)
	}
	if res.ExitCode != 0 && !strings.Contains(res.Stderr, "is not a WireGuard interface") {
		return fmt.Errorf("vpn: wg-quick down exited %d: %s", res.ExitCode, firstLine(res.Stderr))
	}
	return nil
}

// Status reads the live interface via `wg show <iface> dump`. A non-zero exit
// (interface absent) is reported as State{Up:false}, not an error — "is it up?"
// is a normal question with a normal negative answer.
func (w WgQuick) Status(ctx context.Context) (State, error) {
	res, err := w.Exec.Run(ctx, "wg show "+shellArg(w.Iface)+" dump", nil)
	if err != nil {
		return State{}, fmt.Errorf("vpn: wg show: %w", err)
	}
	if res.ExitCode != 0 {
		return State{Up: false, Interface: w.Iface}, nil
	}
	st := parseWgDump(res.Stdout)
	st.Interface = w.Iface
	return st, nil
}

// parseWgDump parses `wg show <iface> dump`. The first line is the interface
// (private-key, public-key, listen-port, fwmark); each subsequent line is a
// peer (public-key, preshared-key, endpoint, allowed-ips, latest-handshake,
// rx, tx, keepalive). latest-handshake is a unix timestamp; 0 means never.
func parseWgDump(dump string) State {
	st := State{}
	lines := strings.Split(strings.TrimRight(dump, "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return st
	}
	// A parseable interface line means the interface exists.
	st.Up = true
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue
		}
		st.Peers++
		if ts, err := strconv.ParseInt(strings.TrimSpace(fields[4]), 10, 64); err == nil && ts > 0 {
			hs := time.Unix(ts, 0)
			if hs.After(st.LastHandshake) {
				st.LastHandshake = hs
			}
		}
	}
	return st
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// shellArg quotes an interface name for `sh -c`. Interface names are tightly
// constrained, but the config is never trusted blindly.
func shellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
