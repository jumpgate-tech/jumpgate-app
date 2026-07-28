package setup

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/metrics"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// The lifecycle CONSTRUCTORS, as distinct from the plans they share bodies
// with. PlanDevnet/PlanGateway are covered elsewhere; these are the other
// entry point — the one ops calls to start a service outside a setup run,
// with a nil State and no event stream. That nil is the whole risk: every
// step body reached this way runs on a code path the plan tests never take.

// ---------------------------------------------------------------------
// DevnetService
// ---------------------------------------------------------------------

func TestDevnetService_DescribesTheContainerItOwns(t *testing.T) {
	d := testDevnet()
	front := ops.DockerService{ID: "erpc:default", ContainerName: "valve-node-app-erpc"}
	svc := DevnetService(d, front)

	if svc.ID != "devnet" {
		t.Errorf("ID = %q, want %q", svc.ID, "devnet")
	}
	if svc.ContainerName != d.Name() {
		t.Errorf("ContainerName = %q, want %q", svc.ContainerName, d.Name())
	}
	// Declaring a volume here would make WipeService try to `docker volume rm`
	// something that does not exist. reth --dev keeps the chain in the
	// container's own writable layer, so `docker rm -f -v` is the whole wipe.
	if len(svc.Volumes) != 0 {
		t.Errorf("Volumes = %v, want none — a --dev chain has no named volume", svc.Volumes)
	}
	if len(svc.FrontedBy) != 1 || svc.FrontedBy[0].ID != front.ID {
		t.Errorf("FrontedBy = %+v, want the gateway in front of it", svc.FrontedBy)
	}
}

func TestDevnetService_CreateStartsTheChain(t *testing.T) {
	shrinkDevnetWait(t)
	d := testDevnet()
	e := liveDocker{devnetReady(d)}

	if err := DevnetService(d).Create(context.Background(), e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if run := lastCallWithPrefix(e.fakeExecutor, "docker 'run'"); run == "" {
		t.Fatal("Create ran no container")
	} else if !strings.Contains(run, catalog.DefaultDevnetImage) {
		t.Errorf("run did not use the devnet image: %s", run)
	}
}

// Validate runs BEFORE anything is created, which is the point: an
// unsatisfiable chain id produces a container that exits immediately, and the
// readiness poll then reports that as a timeout — a thoroughly misleading way
// to learn about a typo.
func TestDevnetService_CreateRefusesAConfigItCannotServe(t *testing.T) {
	d := testDevnet()
	d.ChainID = 999 // reth --dev's genesis is fixed at 1337

	e := liveDocker{devnetReady(d)}
	err := DevnetService(d).Create(context.Background(), e)
	if err == nil {
		t.Fatal("Create accepted a chain id reth --dev cannot serve")
	}
	if !strings.Contains(err.Error(), "999") {
		t.Errorf("error does not name the rejected id: %v", err)
	}
	if run := lastCallWithPrefix(e.fakeExecutor, "docker 'run'"); run != "" {
		t.Errorf("a refused config still created a container: %s", run)
	}
}

// ---------------------------------------------------------------------
// GatewayService
// ---------------------------------------------------------------------

func TestGatewayService_CreateWritesConfigBeforeRunning(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady()

	if err := GatewayService(testGatewayID, testGateway()).Create(context.Background(), e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Order is the assertion, not merely presence. A gateway container started
	// against a missing erpc.yaml comes up and serves nothing, which reads as
	// "the gateway is running" everywhere a container state is displayed.
	wroteAt, ranAt := -1, -1
	for i, c := range e.callLog() {
		if wroteAt < 0 && strings.HasPrefix(c, "WriteFile") && strings.Contains(c, "erpc.yaml") {
			wroteAt = i
		}
		if ranAt < 0 && strings.HasPrefix(c, "docker 'run'") {
			ranAt = i
		}
	}
	if wroteAt < 0 {
		t.Fatal("Create never wrote erpc.yaml")
	}
	if ranAt < 0 {
		t.Fatal("Create never ran the container")
	}
	if wroteAt > ranAt {
		t.Errorf("config written at %d, container run at %d — a gateway started without its config serves nothing", wroteAt, ranAt)
	}
}

// Rendering happens up front so an unusable config fails before an image is
// built or a container created.
func TestGatewayService_CreateRefusesAnUnrenderableConfig(t *testing.T) {
	e := dockerReady()
	empty := catalog.GatewayConfig{Port: 4100} // no networks

	err := GatewayService(testGatewayID, empty).Create(context.Background(), e)
	if err == nil {
		t.Fatal("Create accepted a gateway with no networks")
	}
	if !strings.Contains(err.Error(), "networks") {
		t.Errorf("error does not say what is wrong: %v", err)
	}
	for _, c := range e.callLog() {
		if strings.HasPrefix(c, "docker 'run'") || strings.HasPrefix(c, "WriteFile") {
			t.Errorf("a refused config still touched the target: %s", c)
		}
	}
}

func TestGatewayService_CreateSurfacesAFailedRun(t *testing.T) {
	shrinkGatewayWait(t)
	boom := errors.New("no space left on device")
	e := dockerReady()
	e.errOn("docker 'run'", boom)

	err := GatewayService(testGatewayID, testGateway()).Create(context.Background(), e)
	if err == nil {
		t.Fatal("Create reported success after the container failed to start")
	}
	if !errors.Is(err, boom) {
		t.Errorf("error lost the cause: %v", err)
	}
}

// A non-default gateway id must reach the container name, or two gateways on
// one machine fight over one container.
func TestGatewayService_ScopesToItsGatewayID(t *testing.T) {
	svc := GatewayService("secondary", testGateway())
	if !strings.Contains(svc.ID, "secondary") {
		t.Errorf("ID = %q, want it scoped to the gateway id", svc.ID)
	}
	if svc.ContainerName == GatewayService(testGatewayID, testGateway()).ContainerName {
		t.Error("two gateway ids produced one container name")
	}
}

// ---------------------------------------------------------------------
// GatewayTLSState
// ---------------------------------------------------------------------

// The read is a re-resolution, not a cache: a certificate's validity is a
// function of the wall clock, so state stored at provision time reports the
// provision-day answer forever.
func TestGatewayTLSState_ReportsTheResolvedFront(t *testing.T) {
	e := caddyReady()

	state, rootPath, err := GatewayTLSState(context.Background(), e, testGatewayID, frontedGateway())
	if err != nil {
		t.Fatalf("GatewayTLSState: %v", err)
	}
	if state.CertSource != catalog.CertInternal {
		t.Errorf("CertSource = %q, want %q", state.CertSource, catalog.CertInternal)
	}
	if state.Fallback != "" {
		t.Errorf("an internal-CA front needs no fallback, got %q", state.Fallback)
	}
	if !strings.HasSuffix(rootPath, "caddy-root.crt") {
		t.Errorf("root path = %q, want the file the operator installs", rootPath)
	}
	// Nothing is written and no container is touched.
	for _, c := range e.callLog() {
		if strings.HasPrefix(c, "WriteFile") || strings.HasPrefix(c, "docker 'run'") {
			t.Errorf("a read modified the target: %s", c)
		}
	}
}

func TestGatewayTLSState_NoFrontIsNotAnError(t *testing.T) {
	state, rootPath, err := GatewayTLSState(context.Background(), dockerReady(), testGatewayID, testGateway())
	if err != nil {
		t.Fatalf("a gateway without TLS is a legal gateway: %v", err)
	}
	if state != (TLSState{}) || rootPath != "" {
		t.Errorf("got %+v / %q, want the zero state", state, rootPath)
	}
}

// A fallback is reported rather than hidden — falling back quietly is the
// silent failure the whole TLS path is written to avoid.
func TestGatewayTLSState_ReportsAFallbackAndItsReason(t *testing.T) {
	e := withCertFiles(t, caddyReady(), nil) // cert path named, file absent

	g := frontedGateway()
	g.TLS.CertSource = catalog.CertFiles
	g.TLS.CertFile, g.TLS.KeyFile = tlsCertPath, tlsKeyPath

	state, _, err := GatewayTLSState(context.Background(), e, testGatewayID, g)
	if err != nil {
		t.Fatalf("GatewayTLSState: %v", err)
	}
	if state.Fallback == "" {
		t.Fatal("fell back to the internal CA without saying so")
	}
	if state.FallbackReason != catalog.CertProblemMissing {
		t.Errorf("FallbackReason = %q, want %q", state.FallbackReason, catalog.CertProblemMissing)
	}
	if state.CertSource != catalog.CertInternal {
		t.Errorf("CertSource = %q, want the source actually in force", state.CertSource)
	}
}

// ---------------------------------------------------------------------
// ReadGatewayAnalytics
// ---------------------------------------------------------------------

// One scrape, folded a second way. The dump carries every family either view
// needs, so this must not curl the gateway twice.
func TestReadGatewayAnalytics_FoldsOneScrape(t *testing.T) {
	g := trafficGateway()
	e := newFakeExecutor().script("curl", executor.Result{Stdout: erpcFixture(t)})

	got, err := ReadGatewayAnalytics(context.Background(), e, g)
	if err != nil {
		t.Fatalf("ReadGatewayAnalytics: %v", err)
	}

	var curls int
	for _, c := range e.callLog() {
		if strings.Contains(c, "curl") {
			curls++
		}
	}
	if curls != 1 {
		t.Errorf("scraped %d times, want 1 — both halves come from one dump", curls)
	}
	if len(got.Networks) == 0 {
		t.Fatal("folded the real fixture into no networks at all")
	}

	// The fold must be the same one metrics performs on the same samples,
	// not a second implementation that drifts.
	samples, err := ReadGatewaySamples(context.Background(), e, g)
	if err != nil {
		t.Fatalf("ReadGatewaySamples: %v", err)
	}
	want := metrics.AnalyticsFromSamples(samples, g.ProjectIDOrDefault())
	if len(got.Networks) != len(want.Networks) {
		t.Errorf("networks = %d, want %d", len(got.Networks), len(want.Networks))
	}
}

func TestReadGatewayAnalytics_SurfacesTheScrapeFailure(t *testing.T) {
	g := trafficGateway()
	g.MetricsOff = true

	e := newFakeExecutor()
	if _, err := ReadGatewayAnalytics(context.Background(), e, g); !errors.Is(err, ErrMetricsOff) {
		t.Fatalf("got %v, want it to wrap ErrMetricsOff", err)
	}
	// Same contract as the traffic fold: a disabled gateway costs no round
	// trip, rather than failing one silently.
	if calls := e.callLog(); len(calls) != 0 {
		t.Fatalf("a disabled gateway must never be dialed, but the executor saw: %#v", calls)
	}
}

// A scrape that fails must not fold into an empty-but-healthy-looking page.
func TestReadGatewayAnalytics_SurfacesAnUnreachableGateway(t *testing.T) {
	e := newFakeExecutor().script("curl", executor.Result{ExitCode: 7, Stderr: "connection refused"})

	_, err := ReadGatewayAnalytics(context.Background(), e, trafficGateway())
	if err == nil {
		t.Fatal("a refused scrape folded into a successful read")
	}
	if !strings.Contains(err.Error(), "loopback") {
		t.Errorf("error does not explain where the counters live: %v", err)
	}
}
