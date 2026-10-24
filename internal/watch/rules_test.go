package watch

import (
	"errors"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/monitor"
)

var t0 = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func healthy() *monitor.Snapshot {
	return &monitor.Snapshot{ExecActive: true, BeaconActive: true, ExecHead: 100, RefHead: 100, ExecPeers: 10, DiskKnown: true, DiskUsedPct: 40}
}

// twice evaluates in on a fresh state, then again a poll later.
func twice(in Inputs) State {
	s := Evaluate(State{}, in, t0)
	return Evaluate(s, in, t0.Add(30*time.Second))
}

func open(s State, key string) *Alert {
	for i := range s.Open {
		if s.Open[i].Key == key {
			return &s.Open[i]
		}
	}
	return nil
}

func TestExecDownAlert(t *testing.T) {
	snap := healthy()
	snap.ExecActive = false
	in := Inputs{Snap: snap, ChainID: 369}
	if s := Evaluate(State{}, in, t0); len(s.Open) != 0 {
		t.Fatalf("one failing poll opened %+v", s.Open)
	}
	s := twice(in)
	a := open(s, "exec-down")
	if a == nil || a.Message != "The execution client on this box is not running." {
		t.Fatalf("Open = %+v", s.Open)
	}
	if !a.Since.Equal(t0.Add(30 * time.Second)) {
		t.Errorf("Since = %v", a.Since)
	}
	snap.BeaconActive = false
	if a := open(twice(in), "beacon-down"); a == nil || a.Message != "The beacon client on this box is not running." {
		t.Errorf("no beacon-down")
	}
}

func TestExecBehindAlert(t *testing.T) {
	for _, c := range []struct {
		chain     int
		lag       uint64
		wantAlert bool
	}{{369, 5, false}, {369, 6, true}, {1, 4, false}, {1, 5, true}, {369, 12, true}} {
		snap := healthy()
		snap.ExecHead = 1000
		snap.RefHead = 1000 + c.lag
		a := open(twice(Inputs{Snap: snap, ChainID: c.chain}), "exec-behind")
		if (a != nil) != c.wantAlert {
			t.Errorf("chain %d lag %d: alert %v", c.chain, c.lag, a)
		}
		if c.lag == 12 && (a == nil || a.Message != "The node is 12 blocks behind the reference.") {
			t.Errorf("message = %+v", a)
		}
	}
	// No reference head: no lag rule.
	snap := healthy()
	snap.RefHead = 0
	if a := open(twice(Inputs{Snap: snap, ChainID: 1}), "exec-behind"); a != nil {
		t.Errorf("RefHead 0 alerted")
	}
}

func TestNoPeersAlert(t *testing.T) {
	snap := healthy()
	snap.ExecPeers = 0
	if a := open(twice(Inputs{Snap: snap, ChainID: 1}), "no-peers"); a == nil {
		t.Error("no no-peers alert")
	}
}

func TestDiskAlert(t *testing.T) {
	snap := healthy()
	snap.DiskUsedPct = 93.4
	if a := open(twice(Inputs{Snap: snap, ChainID: 1}), "disk-90"); a == nil || a.Message != "Disk is 93% full." {
		t.Errorf("disk-90 = %+v", a)
	}
	snap.DiskKnown = false
	if a := open(twice(Inputs{Snap: snap, ChainID: 1}), "disk-90"); a != nil {
		t.Error("unknown disk alerted")
	}
}

func TestEndpointAlert(t *testing.T) {
	in := Inputs{
		EndpointList: []Endpoint{{Name: "gw-eu", URL: "https://gw.example/rpc", Kind: "rpc"}},
		Endpoints:    map[string]error{"gw-eu": errors.New("timeout")},
	}
	a := open(twice(in), "endpoint:gw-eu")
	if a == nil || a.Message != "gw-eu (https://gw.example/rpc) stopped answering." {
		t.Errorf("endpoint alert = %+v", a)
	}
}

func TestAlertClearsAndMovesToRecent(t *testing.T) {
	snap := healthy()
	snap.ExecPeers = 0
	s := twice(Inputs{Snap: snap, ChainID: 1})
	snap.ExecPeers = 3
	clearAt := t0.Add(time.Minute)
	s = Evaluate(s, Inputs{Snap: snap, ChainID: 1}, clearAt)
	if len(s.Open) != 0 || len(s.Recent) != 1 {
		t.Fatalf("state = %+v", s)
	}
	if r := s.Recent[0]; r.Key != "no-peers" || r.Cleared == nil || !r.Cleared.Equal(clearAt) {
		t.Errorf("Recent = %+v", r)
	}
	if s.Strikes["no-peers"] != 0 {
		t.Errorf("strikes kept: %v", s.Strikes)
	}
	// Recent is newest first and capped.
	for i := range 60 {
		snap.ExecPeers = 0
		s = Evaluate(s, Inputs{Snap: snap, ChainID: 1}, clearAt.Add(time.Duration(3*i+1)*time.Second))
		s = Evaluate(s, Inputs{Snap: snap, ChainID: 1}, clearAt.Add(time.Duration(3*i+2)*time.Second))
		snap.ExecPeers = 3
		s = Evaluate(s, Inputs{Snap: snap, ChainID: 1}, clearAt.Add(time.Duration(3*i+3)*time.Second))
	}
	if len(s.Recent) != 50 || !s.Recent[0].Cleared.After(*s.Recent[1].Cleared) {
		t.Errorf("Recent len %d", len(s.Recent))
	}
	// A single failing poll after a pass does not open.
	snap.ExecPeers = 0
	s = Evaluate(s, Inputs{Snap: snap, ChainID: 1}, clearAt.Add(time.Hour))
	if len(s.Open) != 0 {
		t.Errorf("opened on one strike")
	}
}

func TestNoWireOnlyEndpoints(t *testing.T) {
	in := Inputs{
		EndpointList: []Endpoint{{Name: "a", URL: "https://a.example", Kind: "rpc"}},
		Endpoints:    map[string]error{"a": nil},
	}
	s := twice(in)
	if len(s.Open) != 0 || len(s.Strikes) != 0 {
		t.Errorf("Snap nil evaluated node rules: %+v", s)
	}
}
