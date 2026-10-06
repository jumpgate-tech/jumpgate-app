// Package tui is jumpgate's terminal UI. It holds no keys and does no work:
// it renders the local server's state and sends it requests through Backend,
// so it can quit and reattach freely (umbrella spec, "The TUI").
package tui

import (
	"context"
	"os/exec"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// Backend is the server as the TUI uses it; *apiclient.Client is the real one.
//
// It lists the client methods that exist so far. The status stream, disk,
// explain, SSH command, controller, agent check, gateways, prefs and settings
// methods join it with the tasks that add them to apiclient.
type Backend interface {
	Skew() (server, mine string, differs bool)
	Fleet(ctx context.Context) (api.Fleet, error)
	WatchFleet(ctx context.Context) <-chan apiclient.Update[api.Fleet]
	Targets(ctx context.Context) ([]api.TargetView, error)
	Endpoints(ctx context.Context, target string) (api.Endpoints, error)
	Firewall(ctx context.Context, target string) ([]api.CheckItem, error)
	WatchLogs(ctx context.Context, target string, backlog int) <-chan apiclient.Update[[]api.LogHit]
	ServiceAction(ctx context.Context, target, service, action string) (api.ServiceResult, error)
	ProbeHostKeys(ctx context.Context, ssh api.SSHView) (api.HostKeyProbe, error)
	ConfirmHostKey(ctx context.Context, probeID, fingerprint string) error
	AddTarget(ctx context.Context, req api.AddTarget) error
	Pair(ctx context.Context, target string, req api.PairRequest) (<-chan api.PairEvent, error)
	RemoveTarget(ctx context.Context, target string) error
	Disk(ctx context.Context, target string) (api.DiskView, error)
	MeasureDisk(ctx context.Context, target string) (api.DiskView, error)
	WatchStatus(ctx context.Context, target string) <-chan apiclient.Update[api.NodeStatus]
	Explain(ctx context.Context, target string, lines []string) (api.Explain, error)
	Settings(ctx context.Context) (api.Settings, error)
	SaveSettings(ctx context.Context, u api.SettingsUpdate) (api.Settings, error)
}

var _ Backend = (*apiclient.Client)(nil)

// Options are what cmd/jumpgate hands the TUI.
type Options struct {
	Backend Backend
	// Self is this jumpgate executable. Pairing this machine as a non-root
	// user and `keys init` run as foreground children of the TUI (spec D17).
	Self    string
	Command func(name string, args ...string) *exec.Cmd // nil: exec.Command
	// OpenWeb opens the web app (platform Task 8); nil hides the action.
	OpenWeb func(ctx context.Context) error
	// Restart stops the server and returns a backend for a fresh one (spec
	// D30); nil hides the action.
	Restart  func(ctx context.Context) (Backend, error)
	GOOS     string              // "" : runtime.GOOS
	Getenv   func(string) string // nil: os.Getenv
	Hostname string              // the default name when pairing this machine
	Now      func() time.Time    // nil: time.Now
}

// uiPrefs holds the display choices the shell reads. It stands in for
// api.UIPrefs, which the server's prefs route brings (Task 12); the field
// names match it, so the swap is a type change. Until then nothing fetches
// prefs: the TUI starts on the defaults.
type uiPrefs struct {
	Units  string // "GB" | "GiB"
	Mouse  bool
	Glyphs string // "auto" | "unicode" | "ascii"
}

// defaultPrefs mirrors api.DefaultPrefs for the fields above.
func defaultPrefs() uiPrefs { return uiPrefs{Units: "GB", Glyphs: "auto"} }

// withDefaults fills every zero-valued field from defaultPrefs.
func (p uiPrefs) withDefaults() uiPrefs {
	d := defaultPrefs()
	if p.Units == "" {
		p.Units = d.Units
	}
	if p.Glyphs == "" {
		p.Glyphs = d.Glyphs
	}
	return p
}
