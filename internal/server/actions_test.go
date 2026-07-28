package server

// The decision functions behind every button the UI offers, plus the two
// constructors the API is wired with by default.
//
// These are worth testing exhaustively rather than through a handler, because
// the rule they encode is a NEGATIVE one — an action that could only fail is
// never offered — and a handler test can only ever observe the actions that
// ARE present. Offering "start" on a stopped-but-unreadable engine is not a
// crash; it is a button that produces an error the operator cannot act on.

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/monitor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// dockerUp is an engine that is installed and answering.
func dockerUp() dockerView { return dockerView{Present: true, Reachable: true} }

// ---------------------------------------------------------------------
// availableActions
// ---------------------------------------------------------------------

func TestAvailableActions_BlockedStatesOfferNothingAndSayWhy(t *testing.T) {
	tests := []struct {
		name    string
		view    containerView
		docker  dockerView
		wantSay string
	}{
		{
			name:    "the container could not be read",
			view:    containerView{Error: "inspect failed"},
			docker:  dockerUp(),
			wantSay: "could not be read",
		},
		{
			name:    "no docker at all",
			view:    containerView{Configured: true},
			docker:  dockerView{},
			wantSay: "no docker engine",
		},
		{
			name:    "docker installed but no engine answering",
			view:    containerView{Configured: true},
			docker:  dockerView{Present: true},
			wantSay: "no engine answered",
		},
		{
			name:    "nothing configured and nothing created",
			view:    containerView{},
			docker:  dockerUp(),
			wantSay: "Nothing is configured yet",
		},
		{
			// An unrecognized state must not fall through to an action set.
			// Guessing "start" for a state the engine did not report is how a
			// button that cannot work gets drawn.
			name:    "an unusable state from the engine",
			view:    containerView{Configured: true, Status: ops.ContainerStatus{State: "who-knows"}},
			docker:  dockerUp(),
			wantSay: "did not report a usable state",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actions, why := availableActions(tc.view, tc.docker)
			if len(actions) != 0 {
				t.Errorf("offered %v for a blocked state", actions)
			}
			if !strings.Contains(why, tc.wantSay) {
				t.Errorf("explanation = %q, want it to mention %q", why, tc.wantSay)
			}
		})
	}
}

func TestAvailableActions_EachStateOffersOnlyWhatCanWork(t *testing.T) {
	tests := []struct {
		state   string
		want    []string
		notWant []string
	}{
		{state: ops.StateNotCreated, want: []string{actionCreate}, notWant: []string{actionStart, actionStop}},
		// No stop on a stopped container, and no restart — which for a
		// stopped container is a start under a name that suggests otherwise.
		{state: ops.StateStopped, want: []string{actionStart, actionRecreate, actionWipe}, notWant: []string{actionStop, actionRestart}},
		{state: ops.StateRunning, want: []string{actionStop, actionRestart, actionRecreate, actionWipe}, notWant: []string{actionStart, actionCreate}},
	}

	for _, tc := range tests {
		t.Run(tc.state, func(t *testing.T) {
			v := containerView{Configured: true, Status: ops.ContainerStatus{State: tc.state}}
			actions, why := availableActions(v, dockerUp())
			if why != "" {
				t.Errorf("a workable state came with an explanation: %q", why)
			}
			if !slices.Equal(actions, tc.want) {
				t.Errorf("actions = %v, want %v", actions, tc.want)
			}
			for _, bad := range tc.notWant {
				if slices.Contains(actions, bad) {
					t.Errorf("offered %q on a %s container", bad, tc.state)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------
// gatewayActions
// ---------------------------------------------------------------------

func TestGatewayActions_BlockedStatesOfferNothingAndSayWhy(t *testing.T) {
	tests := []struct {
		name    string
		view    gatewayView
		docker  dockerView
		wantSay string
	}{
		{
			name:    "the gateway could not be read",
			view:    gatewayView{Error: "inspect failed"},
			docker:  dockerUp(),
			wantSay: "could not be read",
		},
		{
			// The message names the MACHINE the gateway is placed on, which
			// is not necessarily this one — a gateway is a fleet-wide object.
			name:    "no docker on the machine it is placed on",
			view:    gatewayView{},
			docker:  dockerView{},
			wantSay: "machine this gateway is placed on",
		},
		{
			name:    "docker installed but no engine answering",
			view:    gatewayView{},
			docker:  dockerView{Present: true},
			wantSay: "no engine answered",
		},
		{
			name:    "an unusable state from the engine",
			view:    gatewayView{Config: oneChainGateway(), Status: ops.ContainerStatus{State: "who-knows"}},
			docker:  dockerUp(),
			wantSay: "usable state",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actions, why := gatewayActions(tc.view, tc.docker)
			if len(actions) != 0 {
				t.Errorf("offered %v for a blocked state", actions)
			}
			if !strings.Contains(why, tc.wantSay) {
				t.Errorf("explanation = %q, want it to mention %q", why, tc.wantSay)
			}
		})
	}
}

// A gateway with no chain is the one blocked state whose cause is the CONFIG
// rather than the machine, so it is called out separately: eRPC refuses a
// project with no networks, and the fix is to add a chain, not to look at
// docker.
func TestGatewayActions_NoChainsPointsAtTheConfigNotTheEngine(t *testing.T) {
	actions, why := gatewayActions(gatewayView{}, dockerUp())
	if len(actions) != 0 {
		t.Errorf("offered %v for a gateway that cannot be created", actions)
	}
	if !strings.Contains(why, "add a chain") {
		t.Errorf("explanation = %q, want it to name the fix", why)
	}

	// If the container nonetheless EXISTS, it can still be torn down — but
	// never re-created, because rendering its config would fail.
	existing := gatewayView{Status: ops.ContainerStatus{State: ops.StateRunning}}
	actions, why = gatewayActions(existing, dockerUp())
	if !slices.Equal(actions, []string{actionStop, actionWipe}) {
		t.Errorf("actions = %v, want only the teardown pair", actions)
	}
	if slices.Contains(actions, actionRecreate) || slices.Contains(actions, actionCreate) {
		t.Error("offered a create/recreate that would fail at render time")
	}
	if !strings.Contains(why, "cannot be re-created") {
		t.Errorf("explanation = %q, want it to say why", why)
	}
}

func TestGatewayActions_EachStateOffersOnlyWhatCanWork(t *testing.T) {
	tests := []struct {
		state string
		want  []string
	}{
		{state: ops.StateNotCreated, want: []string{actionCreate}},
		{state: ops.StateStopped, want: []string{actionStart, actionRecreate, actionWipe}},
		{state: ops.StateRunning, want: []string{actionStop, actionRestart, actionRecreate, actionWipe}},
	}

	for _, tc := range tests {
		t.Run(tc.state, func(t *testing.T) {
			v := gatewayView{Config: oneChainGateway(), Status: ops.ContainerStatus{State: tc.state}}
			actions, why := gatewayActions(v, dockerUp())
			if why != "" {
				t.Errorf("a workable state came with an explanation: %q", why)
			}
			if !slices.Equal(actions, tc.want) {
				t.Errorf("actions = %v, want %v", actions, tc.want)
			}
		})
	}
}

// oneChainGateway is the minimum configuration eRPC will accept.
func oneChainGateway() catalog.GatewayConfig {
	return catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{
			{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
				{ID: "n1", Endpoint: "https://rpc.example.com"},
			}},
		},
	}
}

// ---------------------------------------------------------------------
// gatewayUnprovisionable
// ---------------------------------------------------------------------

// "no chains" is misleading for a gateway that HAS chains whose every
// upstream is broken, so the dead references get named.
func TestGatewayUnprovisionable_NamesTheDeadReferences(t *testing.T) {
	empty := config.Gateway{}
	if got := gatewayUnprovisionable(empty, nil); !strings.Contains(got, "serves no chains") {
		t.Errorf("got %q, want it to say there are no chains", got)
	}

	withChains := config.Gateway{Config: oneChainGateway()}
	got := gatewayUnprovisionable(withChains, []string{"upstream n1 points at a deleted node"})
	if strings.Contains(got, "serves no chains") {
		t.Errorf("got %q — this gateway HAS chains, so that reads as the wrong problem", got)
	}
	if !strings.Contains(got, "deleted node") {
		t.Errorf("got %q, want the dead reference named", got)
	}
}

// ---------------------------------------------------------------------
// diagTriggerForSnapshot
// ---------------------------------------------------------------------

func TestDiagTriggerForSnapshot(t *testing.T) {
	healthy := monitor.Snapshot{ExecActive: true, BeaconActive: true, ExecPeers: 12, BeaconPeers: 30}
	if got := diagTriggerForSnapshot(healthy); got != "" {
		t.Errorf("a healthy snapshot triggered a diagnostic run: %q", got)
	}

	tests := []struct {
		name    string
		mutate  func(*monitor.Snapshot)
		wantSay string
	}{
		{"exec inactive", func(s *monitor.Snapshot) { s.ExecActive = false }, "execution service inactive"},
		{"beacon inactive", func(s *monitor.Snapshot) { s.BeaconActive = false }, "beacon service inactive"},
		// Zero peers is a failed connection in the making — exactly when the
		// operator wants the ladder run for them rather than after.
		{"exec has no peers", func(s *monitor.Snapshot) { s.ExecPeers = 0 }, "execution client has 0 peers"},
		{"beacon has no peers", func(s *monitor.Snapshot) { s.BeaconPeers = 0 }, "beacon client has 0 peers"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := healthy
			tc.mutate(&s)
			if got := diagTriggerForSnapshot(s); !strings.Contains(got, tc.wantSay) {
				t.Errorf("got %q, want it to mention %q", got, tc.wantSay)
			}
		})
	}
}

// ---------------------------------------------------------------------
// defaultNewExecutor
// ---------------------------------------------------------------------

// Refusing at CONSTRUCTION is the point: an ssh target with no ssh config
// that is accepted here gets persisted and then fails on every command it
// ever runs, with the error arriving far from the mistake.
func TestDefaultNewExecutor_RefusesTargetsItCannotDrive(t *testing.T) {
	tests := []struct {
		name    string
		target  config.Target
		wantSay string
	}{
		{
			name:    "ssh with no ssh config",
			target:  config.Target{ID: "box", Mode: "ssh"},
			wantSay: "requires an ssh config",
		},
		{
			name:    "a mode that does not exist",
			target:  config.Target{ID: "box", Mode: "carrier-pigeon"},
			wantSay: "box",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ex, err := defaultNewExecutor(tc.target)
			if err == nil {
				ex.Close()
				t.Fatal("built an executor for a target it cannot drive")
			}
			if !strings.Contains(err.Error(), tc.wantSay) {
				t.Errorf("error = %v, want it to mention %q", err, tc.wantSay)
			}
		})
	}
}

// Local mode is only available where there is a POSIX shell to drive. On a
// host that has one, it must actually build.
func TestDefaultNewExecutor_LocalMatchesWhetherThisHostSupportsIt(t *testing.T) {
	ex, err := defaultNewExecutor(config.Target{ID: "here", Mode: "local"})
	if err != nil {
		// Windows: refused at construction, with the target named.
		if !strings.Contains(err.Error(), "here") {
			t.Errorf("a refusal must name the target: %v", err)
		}
		return
	}
	defer ex.Close()
	if ex == nil {
		t.Fatal("no error and no executor")
	}
}

// ---------------------------------------------------------------------
// releaseSetupRun
// ---------------------------------------------------------------------

// setupRunOf reads the target's currently-claimed run.
func setupRunOf(s *Server, id string) *setupRun {
	entry := s.reg.get(id)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return entry.setup
}

// A claim whose run never started must be undone: the slot is cleared and the
// claim's context canceled, so nothing is left holding a run that will never
// produce an event.
func TestReleaseSetupRun_ClearsTheSlotAndCancelsTheClaim(t *testing.T) {
	s := New(Config{Token: NewSessionToken()})

	c, ok := s.claimSetupRun(httptest.NewRecorder(), "box")
	if !ok {
		t.Fatal("could not claim a fresh target")
	}
	if setupRunOf(s, "box") != c.run {
		t.Fatal("the claim did not take the slot")
	}

	s.releaseSetupRun("box", c)

	if got := setupRunOf(s, "box"); got != nil {
		t.Errorf("the slot still holds %p after release", got)
	}
	select {
	case <-c.ctx.Done():
	default:
		t.Error("release left the claim's context live, so its work is never told to stop")
	}
}

// Releasing a claim that is no longer the live one must NOT clear the run that
// replaced it — that would free the slot out from under a run in flight.
func TestReleaseSetupRun_AStaleReleaseLeavesTheLiveRunAlone(t *testing.T) {
	s := New(Config{Token: NewSessionToken()})

	stale, ok := s.claimSetupRun(httptest.NewRecorder(), "box")
	if !ok {
		t.Fatal("could not claim")
	}
	s.releaseSetupRun("box", stale)

	live, ok := s.claimSetupRun(httptest.NewRecorder(), "box")
	if !ok {
		t.Fatal("a released claim blocked the next attempt")
	}

	s.releaseSetupRun("box", stale) // stale: already superseded

	if setupRunOf(s, "box") != live.run {
		t.Fatal("a stale release cleared the live run's slot")
	}
	select {
	case <-live.ctx.Done():
		t.Error("a stale release canceled the live run")
	default:
	}
	s.releaseSetupRun("box", live)
}

// The conflict a caller actually meets: a run that is RUNNING is refused with
// 409, rather than a second run being started over the first.
func TestClaimSetupRun_RefusesASecondRunWhileOneIsRunning(t *testing.T) {
	s := New(Config{Token: NewSessionToken()})

	first, ok := s.claimSetupRun(httptest.NewRecorder(), "box")
	if !ok {
		t.Fatal("could not claim a fresh target")
	}
	first.run.mu.Lock()
	first.run.running = true
	first.run.mu.Unlock()

	w := httptest.NewRecorder()
	if _, ok := s.claimSetupRun(w, "box"); ok {
		t.Fatal("a second setup run was claimed while the first was still running")
	}
	if w.Code != http.StatusConflict {
		t.Errorf("got %d, want %d", w.Code, http.StatusConflict)
	}
	if !strings.Contains(w.Body.String(), "already running") {
		t.Errorf("the refusal does not say why: %s", w.Body.String())
	}
	s.releaseSetupRun("box", first)
}

// ---------------------------------------------------------------------
// ListenAndServe
// ---------------------------------------------------------------------

func TestListenAndServe_ServesUntilTheContextIsCanceled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	token := NewSessionToken()
	s := New(Config{Bind: addr, Token: token})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.ListenAndServe(ctx) }()

	// Poll until it is actually accepting, rather than sleeping a guess.
	var res *http.Response
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", "http://"+addr+"/api/targets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if res, err = http.DefaultClient.Do(req); err == nil {
			res.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("the server never accepted a connection: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET /api/targets = %d, want 200", res.StatusCode)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("a canceled context must be a clean shutdown, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ListenAndServe did not return after its context was canceled")
	}
}

// A bind address nothing can listen on comes back as an error rather than
// hanging, so a misconfigured --bind fails at startup.
func TestListenAndServe_ReportsAnUnusableBindAddress(t *testing.T) {
	s := New(Config{Bind: "127.0.0.1:-1", Token: NewSessionToken()})

	err := s.ListenAndServe(context.Background())
	if err == nil {
		t.Fatal("an unusable bind address reported a clean start")
	}
	if errors.Is(err, http.ErrServerClosed) {
		t.Errorf("a listen failure was reported as a clean close: %v", err)
	}
}
