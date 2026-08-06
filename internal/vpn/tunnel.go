package vpn

import (
	"context"
	"time"
)

// State is a point-in-time read of a tunnel.
type State struct {
	Up            bool      // interface exists and is configured
	Provider      string    // which provider produced the config
	Interface     string    // OS interface name, e.g. "jumpgate0"
	Addresses     []string  // overlay CIDRs (the interface's own addresses)
	Peers         int       // configured peer count
	LastHandshake time.Time // most recent successful handshake across peers; zero = never
}

// Handshaked reports whether at least one peer has completed a handshake — the
// real "the tunnel is actually carrying traffic" signal, distinct from "the
// interface came up". Used to verify a bring-up rather than trusting an exit
// code (a wg-quick that exits 0 can still never reach its endpoint).
func (s State) Handshaked() bool { return !s.LastHandshake.IsZero() }

// Tunnel is a single WireGuard overlay we can bring up, tear down, and inspect.
type Tunnel interface {
	// Up applies the provider's config and brings the interface up, returning
	// the resulting State. Implementations MUST verify the interface actually
	// came up rather than trusting the launch command's exit code.
	Up(ctx context.Context) (State, error)
	Down(ctx context.Context) error
	Status(ctx context.Context) (State, error)
}

// Provider is a source of a WireGuard Config — the seam that makes VPNs
// swappable. Bring-your-own is a StaticProvider wrapping a pasted `.conf`; a
// ProtonVPN or Mullvad adapter would implement this by fetching/generating a
// config, and everything downstream (the wg-quick backend, the overlay
// grading) stays identical.
type Provider interface {
	Name() string
	Config(ctx context.Context) (Config, error)
}

// StaticProvider is the generic bring-your-own provider: a config the operator
// already has (a Proton/Mullvad/self-hosted `.conf` they pasted or a path we
// read). It is the reference implementation of Provider and the one that
// proves the design is genuinely provider-neutral.
type StaticProvider struct {
	ProviderName string
	Conf         Config
}

// NewStaticProvider parses conf text into a StaticProvider, validating it up
// front so a bad config fails at configuration time, not at tunnel bring-up.
// name is a label ("bring-your-own", "proton", "mullvad", ...) used only for
// display/telemetry.
func NewStaticProvider(name, confText string) (StaticProvider, error) {
	cfg, err := ParseConfig(confText)
	if err != nil {
		return StaticProvider{}, err
	}
	if err := cfg.Validate(); err != nil {
		return StaticProvider{}, err
	}
	if name == "" {
		name = "bring-your-own"
	}
	return StaticProvider{ProviderName: name, Conf: cfg}, nil
}

func (s StaticProvider) Name() string { return s.ProviderName }

func (s StaticProvider) Config(context.Context) (Config, error) { return s.Conf, nil }
