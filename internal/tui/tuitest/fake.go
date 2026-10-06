package tuitest

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// Fake is a scripted backend. Every method records a call line ("Method
// arg arg") in Calls and returns the canned value, or Err[Method].
//
// It covers every Backend method.
type Fake struct {
	mu    sync.Mutex
	Calls []string
	Err   map[string]error

	ServerVersion string // non-empty: Skew reports a difference

	FleetV     api.Fleet
	FleetCh    chan apiclient.Update[api.Fleet]
	LogsCh     map[string]chan apiclient.Update[[]api.LogHit]
	TargetsV   []api.TargetView
	EndpointsV map[string]api.Endpoints
	FirewallV  map[string][]api.CheckItem
	ServiceV   api.ServiceResult
	Probes     []api.HostKeyProbe // successive ProbeHostKeys answers; the last repeats
	PairV      []api.PairEvent

	DiskV      map[string]api.DiskView
	StatusCh   map[string]chan apiclient.Update[api.NodeStatus]
	ExplainV   api.Explain
	SettingsV  api.Settings
	LastUpdate api.SettingsUpdate // what SaveSettings last received

	PrefsV      api.UIPrefs // what Prefs answers (the defaults to start)
	LastPrefs   api.UIPrefs // what SavePrefs last received
	ControllerV api.ControllerView
	CheckV      api.AgentCheck
	SSHV        map[string]api.SSHCommand
	GatewaysV   []api.GatewaySummary
}

// NewFake is a Fake with empty data and open stream channels.
func NewFake() *Fake {
	return &Fake{
		Err: map[string]error{}, FleetCh: make(chan apiclient.Update[api.Fleet], 16),
		LogsCh:     map[string]chan apiclient.Update[[]api.LogHit]{},
		EndpointsV: map[string]api.Endpoints{}, FirewallV: map[string][]api.CheckItem{},
		PrefsV: api.DefaultPrefs(), SSHV: map[string]api.SSHCommand{},
		DiskV: map[string]api.DiskView{}, StatusCh: map[string]chan apiclient.Update[api.NodeStatus]{},
	}
}

func (f *Fake) call(name string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	parts := []string{name}
	for _, a := range args {
		parts = append(parts, fmt.Sprint(a))
	}
	f.Calls = append(f.Calls, strings.Join(parts, " "))
	return f.Err[name]
}

// Called reports whether a call line starting with prefix was made.
func (f *Fake) Called(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.Calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// Logs is the logs channel for target, made on first use.
func (f *Fake) Logs(target string) chan apiclient.Update[[]api.LogHit] {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.LogsCh[target] == nil {
		f.LogsCh[target] = make(chan apiclient.Update[[]api.LogHit], 16)
	}
	return f.LogsCh[target]
}

func (f *Fake) Skew() (string, string, bool) { return f.ServerVersion, "v-test", f.ServerVersion != "" }

func (f *Fake) Fleet(context.Context) (api.Fleet, error) { return f.FleetV, f.call("Fleet") }

// WatchFleet relays FleetCh until ctx ends and then closes, as the real
// stream does, so a test's cancelled App leaves no goroutine blocked on it.
func (f *Fake) WatchFleet(ctx context.Context) <-chan apiclient.Update[api.Fleet] {
	_ = f.call("WatchFleet")
	return relay(ctx, f.FleetCh)
}

// relay forwards in to a fresh channel and closes it when ctx ends or in
// closes.
func relay[T any](ctx context.Context, in <-chan T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

func (f *Fake) Targets(context.Context) ([]api.TargetView, error) {
	return f.TargetsV, f.call("Targets")
}

func (f *Fake) Endpoints(_ context.Context, target string) (api.Endpoints, error) {
	f.mu.Lock()
	v := f.EndpointsV[target]
	f.mu.Unlock()
	return v, f.call("Endpoints", target)
}

func (f *Fake) Firewall(_ context.Context, target string) ([]api.CheckItem, error) {
	f.mu.Lock()
	v := f.FirewallV[target]
	f.mu.Unlock()
	return v, f.call("Firewall", target)
}

func (f *Fake) WatchLogs(ctx context.Context, target string, backlog int) <-chan apiclient.Update[[]api.LogHit] {
	_ = f.call("WatchLogs", target, backlog)
	return relay(ctx, f.Logs(target))
}

func (f *Fake) ServiceAction(_ context.Context, target, service, action string) (api.ServiceResult, error) {
	return f.ServiceV, f.call("ServiceAction", target, service, action)
}

func (f *Fake) ProbeHostKeys(_ context.Context, ssh api.SSHView) (api.HostKeyProbe, error) {
	err := f.call("ProbeHostKeys", ssh.Address())
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.Probes) == 0 {
		return api.HostKeyProbe{AllConfirmed: true}, err
	}
	p := f.Probes[0]
	if len(f.Probes) > 1 {
		f.Probes = f.Probes[1:]
	}
	return p, err
}

func (f *Fake) ConfirmHostKey(_ context.Context, probeID, fingerprint string) error {
	return f.call("ConfirmHostKey", probeID, fingerprint)
}

func (f *Fake) AddTarget(_ context.Context, req api.AddTarget) error {
	return f.call("AddTarget", req.ID, req.Mode)
}

func (f *Fake) Pair(_ context.Context, target string, req api.PairRequest) (<-chan api.PairEvent, error) {
	if err := f.call("Pair", target, req.Sudo); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan api.PairEvent, len(f.PairV))
	for _, ev := range f.PairV {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (f *Fake) RemoveTarget(_ context.Context, target string) error {
	return f.call("RemoveTarget", target)
}

func (f *Fake) Disk(_ context.Context, target string) (api.DiskView, error) {
	f.mu.Lock()
	v := f.DiskV[target]
	f.mu.Unlock()
	return v, f.call("Disk", target)
}

func (f *Fake) MeasureDisk(_ context.Context, target string) (api.DiskView, error) {
	f.mu.Lock()
	v := f.DiskV[target]
	f.mu.Unlock()
	return v, f.call("MeasureDisk", target)
}

// Status is the status channel for target, made on first use.
func (f *Fake) Status(target string) chan apiclient.Update[api.NodeStatus] {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.StatusCh[target] == nil {
		f.StatusCh[target] = make(chan apiclient.Update[api.NodeStatus], 16)
	}
	return f.StatusCh[target]
}

func (f *Fake) WatchStatus(ctx context.Context, target string) <-chan apiclient.Update[api.NodeStatus] {
	_ = f.call("WatchStatus", target)
	return relay(ctx, f.Status(target))
}

func (f *Fake) Explain(_ context.Context, target string, lines []string) (api.Explain, error) {
	return f.ExplainV, f.call("Explain", target, len(lines))
}

func (f *Fake) Settings(context.Context) (api.Settings, error) {
	f.mu.Lock()
	v := f.SettingsV
	f.mu.Unlock()
	return v, f.call("Settings")
}

func (f *Fake) SaveSettings(_ context.Context, u api.SettingsUpdate) (api.Settings, error) {
	f.mu.Lock()
	f.LastUpdate = u
	if f.Err["SaveSettings"] == nil { // a saved update shows in the next read, as on the server
		if u.AIProvider != nil {
			f.SettingsV.AIProvider = *u.AIProvider
			f.SettingsV.AIDisclosure = ""
		}
		if u.AIKey != nil {
			f.SettingsV.AIKeySet = *u.AIKey != ""
		}
	}
	v := f.SettingsV
	f.mu.Unlock()
	return v, f.call("SaveSettings")
}

func (f *Fake) Prefs(context.Context) (api.UIPrefs, error) {
	f.mu.Lock()
	v := f.PrefsV
	f.mu.Unlock()
	return v, f.call("Prefs")
}

func (f *Fake) SavePrefs(_ context.Context, p api.UIPrefs) (api.UIPrefs, error) {
	err := f.call("SavePrefs")
	f.mu.Lock()
	defer f.mu.Unlock()
	f.LastPrefs = p
	if err == nil {
		f.PrefsV = p
	}
	return p, err
}

func (f *Fake) Controller(context.Context) (api.ControllerView, error) {
	f.mu.Lock()
	v := f.ControllerV
	f.mu.Unlock()
	return v, f.call("Controller")
}

func (f *Fake) CheckAgent(_ context.Context, target string) (api.AgentCheck, error) {
	f.mu.Lock()
	v := f.CheckV
	f.mu.Unlock()
	return v, f.call("CheckAgent", target)
}

func (f *Fake) SSHCommand(_ context.Context, target string) (api.SSHCommand, error) {
	f.mu.Lock()
	v := f.SSHV[target]
	f.mu.Unlock()
	return v, f.call("SSHCommand", target)
}

func (f *Fake) Gateways(context.Context) ([]api.GatewaySummary, error) {
	f.mu.Lock()
	v := f.GatewaysV
	f.mu.Unlock()
	return v, f.call("Gateways")
}
