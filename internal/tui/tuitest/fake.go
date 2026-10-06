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
// It covers the apiclient methods that exist today (Tasks 2–9). The status
// stream, disk, explain, SSH command, controller, agent check, gateways,
// prefs and settings methods are added with the tasks that add them to
// apiclient (10–12), together with their canned fields.
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
}

// NewFake is a Fake with empty data and open stream channels.
func NewFake() *Fake {
	return &Fake{
		Err: map[string]error{}, FleetCh: make(chan apiclient.Update[api.Fleet], 16),
		LogsCh:     map[string]chan apiclient.Update[[]api.LogHit]{},
		EndpointsV: map[string]api.Endpoints{}, FirewallV: map[string][]api.CheckItem{},
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
