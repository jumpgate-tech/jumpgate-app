package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// pulsechainOnly is the smallest config a gateway can be provisioned with:
// one chain, one endpoint that needs no resolution.
func pulsechainOnly(port int) catalog.GatewayConfig {
	return catalog.GatewayConfig{
		Port: port,
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
		}}},
	}
}

// ---------------------------------------------------------------------
// POST /api/gateways/{gid}/{action}
// ---------------------------------------------------------------------

// The action has to land on the gateway's OWN container. Two gateways on one
// machine have distinct containers by design, so an action that resolved the
// name from anything but the gateway's id would stop the wrong one — and both
// would report success.
func TestGatewayAction_ActsOnThatGatewaysOwnContainer(t *testing.T) {
	for _, action := range []string{actionStart, actionStop, actionRestart} {
		t.Run(action, func(t *testing.T) {
			f := newFleet()
			a := newAPITestServerWithExecutor(t, f.factory)
			addTarget(t, a)
			res2 := a.do(t, "POST", "/api/targets", map[string]any{"id": "second", "mode": "local"})
			res2.Body.Close()
			addGateway(t, a, "default", "local", pulsechainOnly(4100))
			addGateway(t, a, "edge", "second", pulsechainOnly(4200))

			res := a.do(t, "POST", "/api/gateways/edge/"+action, nil)
			body := decode[actionResponse](t, res)
			if res.StatusCode != http.StatusOK {
				t.Fatalf("got %d, want 200", res.StatusCode)
			}
			if body.Status.ContainerName != "valve-node-app-erpc-edge" {
				t.Errorf("status names %q, want the edge gateway's container", body.Status.ContainerName)
			}
			if !f.ran(t, "second", "'"+action+"'", "'valve-node-app-erpc-edge'") {
				t.Errorf("no %s reached the edge container; ran: %q", action, f.commands(t, "second"))
			}
			// The default gateway is a bystander and must not be touched.
			if f.ran(t, "local", "'"+action+"'", "'valve-node-app-erpc'") {
				t.Errorf("%s also hit the default gateway's container", action)
			}
		})
	}
}

// Creating and destroying are not actions on this route — they are their own
// endpoints, because one is a plan with a progress stream and the other is
// irreversible. The refusal names both, so a caller that guessed is told where
// to go rather than just told no.
func TestGatewayAction_RefusesAnythingThatIsNotStartStopRestart(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addGateway(t, a, "default", "local", pulsechainOnly(4100))

	// "wipe" and "provision" are deliberately absent: they are their own
	// routes, and the more specific pattern wins in the mux, so they never
	// reach this handler at all.
	for _, action := range []string{"create", "recreate", "delete", "rm", "bounce"} {
		res := a.do(t, "POST", "/api/gateways/default/"+action, nil)
		body := decode[errorDetail](t, res)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("action %q: got %d, want 400", action, res.StatusCode)
			continue
		}
		for _, want := range []string{"provision", "wipe"} {
			if !strings.Contains(body.Error, want) {
				t.Errorf("action %q: message does not point at %q: %q", action, want, body.Error)
			}
		}
	}
	// Nothing was run against the machine for any of them.
	for _, cmd := range f.commands(t, "local") {
		if strings.Contains(cmd, "'start'") || strings.Contains(cmd, "'stop'") || strings.Contains(cmd, "'restart'") {
			t.Errorf("a rejected action still reached the machine: %q", cmd)
		}
	}
}

// A gateway names its host. If that host is deregistered the gateway record
// survives, and every action on it must fail with the reason rather than fall
// back to some other machine — which is how an action gets applied to the
// wrong box.
func TestGatewayAction_SaysWhenThePlacementMachineIsGone(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addSSHTarget(t, a, "boxa")
	addGateway(t, a, "edge", "boxa", pulsechainOnly(4200))

	res := a.do(t, "DELETE", "/api/targets/boxa", nil)
	res.Body.Close()

	res = a.do(t, "POST", "/api/gateways/edge/restart", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want 404", res.StatusCode)
	}
	if !strings.Contains(body.Error, "boxa") {
		t.Errorf("message must name the machine that is gone: %q", body.Error)
	}
	if f.ran(t, "local", "'restart'", "'valve-node-app-erpc-edge'") {
		t.Error("the action fell through to the local machine — that restarts a container on the wrong box")
	}
}

// ---------------------------------------------------------------------
// POST /api/gateways/{gid}/wipe
// ---------------------------------------------------------------------

// The typed token is the gateway's own id, and typing it must actually destroy
// the container — a confirmation gate that guards nothing is worse than none,
// because it teaches the operator the word is a formality.
func TestGatewayWipe_TheTypedIDDestroysThatGatewaysContainer(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addGateway(t, a, "edge", "local", pulsechainOnly(4200))

	res := a.do(t, "POST", "/api/gateways/edge/wipe", map[string]string{"Confirm": "edge"})
	body := decode[wipeResponse](t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d (%s), want 200", res.StatusCode, body.Error)
	}
	if !body.Report.ContainerRemoved {
		t.Error("the report does not say the container was removed")
	}
	if !f.ran(t, "local", "'rm'", "'valve-node-app-erpc-edge'") {
		t.Errorf("the container was never removed; ran: %q", f.commands(t, "local"))
	}
	// A wipe forgets the container, not the configuration: the gateway must
	// still be there to re-create.
	res = a.do(t, "GET", "/api/gateways/edge", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("after wipe the gateway itself is gone (%d) — wipe destroys a container, delete forgets a config", res.StatusCode)
	}
}

// Confirming with ANOTHER gateway's id must not wipe either of them. The gate
// is per-gateway, so a stale confirmation dialog cannot destroy the one that
// happens to be open now. The two gateways are on two machines — one managed
// eRPC per device — so the mismatch is checked across machines too.
func TestGatewayWipe_AnotherGatewaysIDIsNotAConfirmation(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	res2 := a.do(t, "POST", "/api/targets", map[string]any{"id": "second", "mode": "local"})
	res2.Body.Close()
	addGateway(t, a, "default", "local", pulsechainOnly(4100))
	addGateway(t, a, "edge", "second", pulsechainOnly(4200))

	res := a.do(t, "POST", "/api/gateways/edge/wipe", map[string]string{"Confirm": "default"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if f.ran(t, "second", "'rm'", "'valve-node-app-erpc-edge'") {
		t.Error("a mismatched confirmation still removed the edge gateway's container")
	}
	if f.ran(t, "local", "'rm'", "'valve-node-app-erpc'") {
		t.Error("a mismatched confirmation still removed the default gateway's container")
	}
}

// ---------------------------------------------------------------------
// POST /api/gateways/{gid}/provision
// ---------------------------------------------------------------------

// The run slot is per MACHINE, and a gateway is provisioned on the machine it
// is PLACED on — not on whatever target the caller happens to be looking at.
// The response says which stream to follow so the UI does not have to know the
// placement rule, and following the wrong one shows a blank progress panel
// while the work happens elsewhere.
func TestGatewayProvision_RunsOnThePlacementMachineAndSaysWhichStreamToFollow(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addSSHTarget(t, a, "boxa")
	addGateway(t, a, "edge", "boxa", pulsechainOnly(4200))

	res := a.do(t, "POST", "/api/gateways/edge/provision", nil)
	body := decode[map[string]string](t, res)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("got %d, want 202", res.StatusCode)
	}
	if body["targetId"] != "boxa" {
		t.Errorf("targetId: got %q, want boxa — the machine the gateway is placed on", body["targetId"])
	}
	if body["status"] != "started" {
		t.Errorf("status: got %q", body["status"])
	}
}

// gateOnListenerProbe is the command a gatedExecutor blocks on: the gateway
// preflight's port check. It is the right choke point because ONLY a
// provisioning run issues it — rendering a gateway view probes docker and
// inspects containers, but never looks for listeners — so the gate stops a run
// mid-flight without also freezing the ordinary reads the test needs to make.
const gateOnListenerProbe = "ss -ltn"

// gatedExecutor holds the provisioning run at its port check until the test
// releases it, so the run is GENUINELY still in flight when the second request
// arrives.
//
// Forcing the overlap rather than observing it is the point. A version of this
// test that just fires two requests and hopes passes on a fast machine and
// fails on a slow one — it measures scheduling, not the run slot.
type gatedExecutor struct {
	*scriptedExecutor
	gate    chan struct{}
	once    sync.Once
	reached chan struct{} // closed the first time the run blocks
}

func newGatedExecutor(gate chan struct{}) *gatedExecutor {
	return &gatedExecutor{
		// The container is reported STOPPED on purpose: preflight skips the
		// port check entirely when this gateway's own container already holds
		// the port, and a skipped check is a gate that never closes.
		scriptedExecutor: fleetExecutor("false|0|img|sha256:abc\n"),
		gate:             gate,
		reached:          make(chan struct{}),
	}
}

func (g *gatedExecutor) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	if strings.Contains(cmd, gateOnListenerProbe) {
		g.once.Do(func() { close(g.reached) })
		select {
		case <-g.gate:
		case <-ctx.Done():
			return executor.Result{}, ctx.Err()
		}
	}
	return g.scriptedExecutor.Run(ctx, cmd, opts)
}

// awaitInFlight blocks until the run has actually reached the gate, which is
// the moment it is provably holding the slot. Without it the test would race
// the goroutine launchSetupRun starts, and would be asserting on timing again.
func (g *gatedExecutor) awaitInFlight(t *testing.T) {
	t.Helper()
	select {
	case <-g.reached:
	case <-time.After(10 * time.Second):
		t.Fatal("the provisioning run never reached its port check, so nothing was ever holding the run slot")
	}
}

// A machine hosts one gateway now, so a second provision run on that machine
// can only be a double-submitted provision of the SAME gateway — not a second
// one, which the create-time guard already refuses. Keep the gated-executor
// setup: the slot still matters, and this is what now exercises it.
func TestGatewayProvision_RefusesASecondRunWhileOneIsInFlight(t *testing.T) {
	gate := make(chan struct{})
	// Released before the httptest server is torn down (cleanups run in
	// reverse order of registration), so the blocked run finishes rather than
	// leaking a goroutine into the rest of the package's tests.
	t.Cleanup(func() { close(gate) })

	gx := newGatedExecutor(gate)
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return gx, nil
	})
	addTarget(t, a)
	addGateway(t, a, "default", "local", pulsechainOnly(4100))

	first := a.do(t, "POST", "/api/gateways/default/provision", nil)
	first.Body.Close()
	if first.StatusCode != http.StatusAccepted {
		t.Fatalf("first provision: got %d, want 202", first.StatusCode)
	}
	gx.awaitInFlight(t)

	// A machine hosts one gateway, so a second run on that machine is the same
	// gateway again — a double-submitted provision, which the slot must still
	// refuse rather than running twice over one container.
	second := a.do(t, "POST", "/api/gateways/default/provision", nil)
	second.Body.Close()
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("second provision of the same gateway: got %d, want 409 — the first run still holds the slot", second.StatusCode)
	}
}

// The slot is per MACHINE, not global: a gateway provision on one box must not
// block one on another, or a fleet serializes on its slowest member.
func TestGatewayProvision_ARunOnOneMachineDoesNotBlockAnother(t *testing.T) {
	gate := make(chan struct{})
	t.Cleanup(func() { close(gate) })

	gx := newGatedExecutor(gate)
	a := newAPITestServerWithExecutor(t, func(tg config.Target) (executor.Executor, error) {
		if tg.ID == "local" {
			return gx, nil
		}
		return fleetExecutor("false|0|img|sha256:abc\n"), nil
	})
	addTarget(t, a)
	addSSHTarget(t, a, "boxa")
	addGateway(t, a, "default", "local", pulsechainOnly(4100))
	addGateway(t, a, "edge", "boxa", pulsechainOnly(4200))

	first := a.do(t, "POST", "/api/gateways/default/provision", nil)
	first.Body.Close()
	if first.StatusCode != http.StatusAccepted {
		t.Fatalf("first provision: got %d, want 202", first.StatusCode)
	}
	gx.awaitInFlight(t)

	second := a.do(t, "POST", "/api/gateways/edge/provision", nil)
	second.Body.Close()
	if second.StatusCode != http.StatusAccepted {
		t.Fatalf("provision on the other machine: got %d, want 202 — one box's run must not hold another's slot", second.StatusCode)
	}
}

// ---------------------------------------------------------------------
// GET /api/gateways/{gid}/tls/verify
// ---------------------------------------------------------------------

// There is nothing to verify on a gateway that serves no HTTPS, and the answer
// must say what to turn on rather than reporting a failed check — a red cross
// against a feature that was never enabled reads as broken.
func TestGatewayTLSVerify_SaysWhatToTurnOnWhenThereIsNoFront(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addGateway(t, a, "default", "local", pulsechainOnly(4100))

	res := a.do(t, "GET", "/api/gateways/default/tls/verify", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if body.Code != codeNotConfigured {
		t.Errorf("code: got %q, want %q", body.Code, codeNotConfigured)
	}
	if !strings.Contains(body.Hint, "Serve HTTPS") {
		t.Errorf("hint does not name the setting to turn on: %q", body.Hint)
	}
}

// ---------------------------------------------------------------------
// the TLS view — re-resolved on every read
// ---------------------------------------------------------------------

// certExecutor is a fleet executor whose ReadFile serves a specific
// certificate and key, so a test can put a REAL certificate (valid, expired,
// wrong name) on the target and see what the read reports.
type certExecutor struct {
	*scriptedExecutor
	files map[string][]byte
}

func (c *certExecutor) ReadFile(_ context.Context, path string) ([]byte, error) {
	if b, ok := c.files[path]; ok {
		return b, nil
	}
	return nil, &noSuchFileError{path: path}
}

type noSuchFileError struct{ path string }

func (e *noSuchFileError) Error() string { return "no such file: " + e.path }

// issueLeaf mints a self-signed certificate for hostname over a validity
// window. Real x509 rather than a fixture, because the resolution being tested
// reads the same fields a browser does.
func issueLeaf(t *testing.T, hostname string, notBefore, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: hostname},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		DNSNames:     []string{hostname},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

const (
	testCertPath = "/etc/valve/gw.crt"
	testKeyPath  = "/etc/valve/gw.key"
)

// tlsGatewayServer builds a server whose local machine has the given files on
// disk, and a gateway configured to serve HTTPS from testCertPath.
func tlsGatewayServer(t *testing.T, files map[string][]byte) *apiTestServer {
	t.Helper()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return &certExecutor{scriptedExecutor: fleetExecutor("true|0|img|sha256:abc\n"), files: files}, nil
	})
	addTarget(t, a)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		TLS: &catalog.GatewayTLS{
			Enabled:    true,
			Hostname:   "gw.valve.city",
			CertSource: catalog.CertFiles,
			CertFile:   testCertPath,
			KeyFile:    testKeyPath,
		},
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
		}}},
	})
	return a
}

// A certificate that was fine when the gateway was provisioned expires on a
// wall clock, not on a provisioning run. Re-running the resolution on every
// READ is what makes that visible; reading stored state would report the
// provisioning-day answer forever, and the operator would find out from a
// browser instead.
func TestGatewayTLSView_AnExpiredCertificateShowsAsFallenBackOnRead(t *testing.T) {
	past := time.Now().Add(-90 * 24 * time.Hour)
	a := tlsGatewayServer(t, map[string][]byte{
		testCertPath: issueLeaf(t, "gw.valve.city", past, past.Add(24*time.Hour)),
		testKeyPath:  []byte("key"),
	})

	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	tls := got.TLS
	if !tls.Enabled {
		t.Fatal("HTTPS is configured but the view reports it off")
	}
	// What the operator ASKED for is unchanged — the stored settings are not
	// rewritten, so the fallback lifts by itself once the file is renewed.
	if tls.CertSource != catalog.CertFiles {
		t.Errorf("certSource: got %q, want the configured %q", tls.CertSource, catalog.CertFiles)
	}
	if tls.EffectiveCertSource != catalog.CertInternal {
		t.Errorf("effectiveCertSource: got %q, want %q — an expired file cannot be served", tls.EffectiveCertSource, catalog.CertInternal)
	}
	if tls.FallbackReason != catalog.CertProblemExpired {
		t.Errorf("fallbackReason: got %q, want %q", tls.FallbackReason, catalog.CertProblemExpired)
	}
	if !strings.Contains(tls.Fallback, testCertPath) {
		t.Errorf("the reason must name the file to fix: %q", tls.Fallback)
	}
	// And the root the operator has to install is named, or the browser
	// warning it causes is a mystery.
	if tls.RootCAPath == "" {
		t.Error("no rootCaPath, so the one-time browser warning has no stated fix")
	}
}

// A certificate for the WRONG name is the failure this whole path introduces:
// everything about the file is valid, and it still cannot serve this gateway.
func TestGatewayTLSView_ACertificateForAnotherNameFallsBackAndSaysWhichNames(t *testing.T) {
	now := time.Now()
	a := tlsGatewayServer(t, map[string][]byte{
		testCertPath: issueLeaf(t, "other.example", now.Add(-time.Hour), now.Add(90*24*time.Hour)),
		testKeyPath:  []byte("key"),
	})

	tls := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil)).TLS
	if tls.FallbackReason != catalog.CertProblemMismatch {
		t.Fatalf("fallbackReason: got %q, want %q", tls.FallbackReason, catalog.CertProblemMismatch)
	}
	if !strings.Contains(tls.Fallback, "other.example") {
		t.Errorf("the reason must say which names the certificate DOES cover, or there is nothing to act on: %q", tls.Fallback)
	}
}

// The other half: a good certificate must NOT fall back. Without this, a
// resolution that always fell back would pass every test above.
func TestGatewayTLSView_AValidCertificateIsServedAsConfigured(t *testing.T) {
	now := time.Now()
	a := tlsGatewayServer(t, map[string][]byte{
		testCertPath: issueLeaf(t, "gw.valve.city", now.Add(-time.Hour), now.Add(90*24*time.Hour)),
		testKeyPath:  []byte("key"),
	})

	tls := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil)).TLS
	if tls.EffectiveCertSource != catalog.CertFiles {
		t.Errorf("effectiveCertSource: got %q, want %q", tls.EffectiveCertSource, catalog.CertFiles)
	}
	if tls.Fallback != "" || tls.FallbackReason != "" {
		t.Errorf("a usable certificate fell back anyway: %q / %q", tls.FallbackReason, tls.Fallback)
	}
	// A fronted gateway publishes no plaintext port, so the URL an operator
	// copies has to be the https one.
	if !strings.HasPrefix(tls.URL, "https://gw.valve.city") {
		t.Errorf("tls url: got %q", tls.URL)
	}
}

// The front is its own container and can be down while the gateway behind it
// is up — which is a dead endpoint that looks healthy if the two states are
// collapsed into one. They are reported separately.
func TestGatewayTLSView_ReportsTheFrontsOwnContainerSeparately(t *testing.T) {
	now := time.Now()
	a := tlsGatewayServer(t, map[string][]byte{
		testCertPath: issueLeaf(t, "gw.valve.city", now.Add(-time.Hour), now.Add(90*24*time.Hour)),
		testKeyPath:  []byte("key"),
	})

	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	if got.TLS.ContainerName == got.ContainerName {
		t.Fatalf("the front and the gateway report the same container %q — they are two containers", got.ContainerName)
	}
	if got.TLS.ContainerName != ops.CaddyContainerNameFor("default") {
		t.Errorf("front container: got %q, want %q", got.TLS.ContainerName, ops.CaddyContainerNameFor("default"))
	}
}

// A gateway with no HTTPS still has to offer a name to turn it on with,
// otherwise the setting opens on a blank field and the operator invents one
// that does not resolve.
func TestGatewayTLSView_OffersAHostnameEvenWithHTTPSOff(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addGateway(t, a, "default", "local", pulsechainOnly(4100))

	tls := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil)).TLS
	if tls.Enabled {
		t.Fatal("HTTPS reported on for a gateway that has none")
	}
	if tls.SuggestedHostname == "" {
		t.Fatal("no suggested hostname, so the setting opens on a blank field")
	}
	if !strings.HasSuffix(tls.SuggestedHostname, catalog.DefaultTLSDomain) {
		t.Errorf("suggested hostname %q is not under the domain whose wildcard resolves to loopback (%s)",
			tls.SuggestedHostname, catalog.DefaultTLSDomain)
	}
}

// ---------------------------------------------------------------------
// POST /api/gateways/{gid}/trust-cert
// ---------------------------------------------------------------------

// internalTLSGateway is the smallest fronted gateway: HTTPS on, served by
// Caddy's own internal CA — the case trust-cert exists for.
func internalTLSGateway(port int) catalog.GatewayConfig {
	cfg := pulsechainOnly(port)
	cfg.TLS = &catalog.GatewayTLS{
		Enabled:    true,
		Hostname:   "default-abc.localhost-valaxy.com",
		CertSource: catalog.CertInternal,
	}
	return cfg
}

// trustExecutor is a fleet executor whose ReadFile serves a certificate for
// the exported-root path, so the trust-cert handler's "is this actually the
// root we exported?" read has something to find.
type trustExecutor struct {
	*scriptedExecutor
	cert []byte
}

func (e *trustExecutor) ReadFile(_ context.Context, path string) ([]byte, error) {
	if strings.HasSuffix(path, ".crt") {
		return e.cert, nil
	}
	return nil, &noSuchFileError{path: path}
}

func (e *trustExecutor) ran(sub string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, c := range e.calls {
		if strings.Contains(c, sub) {
			return true
		}
	}
	return false
}

// The one-click trust install must run the OS's trust-store command, on the
// gateway's own machine, against THIS gateway's exported internal root — the
// derived rootCAPath — and report structured success.
func TestGatewayTrustCert_InstallsTheGatewaysOwnInternalRoot(t *testing.T) {
	root := issueLeaf(t, "root", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
	var ex *trustExecutor
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		if ex == nil {
			ex = &trustExecutor{scriptedExecutor: fleetExecutor("true|0|img|sha256:abc\n"), cert: root}
			// Root, so the linux branch runs its command rather than returning
			// the sudo fallback — keeping the run-path assertion OS-independent.
			ex.script("id -u", executor.Result{Stdout: "0\n"})
			// This test is the fresh-install path: the root is NOT trusted yet, so
			// the darwin verify probe fails (exit 1) and the install runs. On a
			// darwin test host a default exit 0 would otherwise short-circuit it.
			ex.script("verify-cert", executor.Result{ExitCode: 1})
		}
		return ex, nil
	})
	addTarget(t, a)
	addGateway(t, a, "default", "local", internalTLSGateway(4100))

	res := a.do(t, "POST", "/api/gateways/default/trust-cert", nil)
	body := decode[trustCertResult](t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200 (%s)", res.StatusCode, body.Message)
	}
	if !body.OK {
		t.Fatalf("trust reported failure: %s", body.Message)
	}
	// It installs the gateway's OWN exported root (rootCAPath ends in
	// caddy-root.crt), never a path from the request.
	if !strings.Contains(body.RanCommand, "caddy-root.crt") {
		t.Errorf("command does not install the gateway's own root: %q", body.RanCommand)
	}
	// And it ran on the machine the gateway is placed on.
	if ex == nil || !ex.ran("caddy-root.crt") {
		t.Error("no trust command reached the gateway's machine")
	}
}

// The truthful retry. On darwin a detached launch (over SSH, a background
// service, nohup) has no GUI session, so the osascript install fails "no user
// interaction was possible". The operator then trusts the root by hand and it
// IS trusted — but a naive retry re-runs osascript, fails the same GUI prompt
// again, and falsely reports failure. So trust-cert probes FIRST: when the root
// already verifies, it reports success WITHOUT running the install.
func TestGatewayTrustCert_AlreadyTrustedShortCircuitsWithoutInstalling(t *testing.T) {
	root := issueLeaf(t, "root", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
	var ex *trustExecutor
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		if ex == nil {
			ex = &trustExecutor{scriptedExecutor: fleetExecutor("true|0|img|sha256:abc\n"), cert: root}
			// Force the darwin path regardless of the test host's own OS: an SSH
			// target is asked with uname, and this one answers Darwin.
			ex.script("uname", executor.Result{Stdout: "Darwin\n"})
			// The root already verifies (exit 0): the operator trusted it by hand
			// after a detached-launch osascript failure.
			ex.script("verify-cert", executor.Result{ExitCode: 0})
		}
		return ex, nil
	})
	addSSHTarget(t, a, "box")
	addGateway(t, a, "default", "box", internalTLSGateway(4100))

	res := a.do(t, "POST", "/api/gateways/default/trust-cert", nil)
	body := decode[trustCertResult](t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200 (%s)", res.StatusCode, body.Message)
	}
	if !body.OK {
		t.Fatalf("an already-trusted root must report success, got failure: %s", body.Message)
	}
	// The whole point: the retry did NOT re-run the osascript install, which would
	// fail the same GUI prompt again and falsely report failure.
	if ex.ran("add-trusted-cert") || ex.ran("osascript") {
		t.Error("the install ran though the root was already trusted — a retry would falsely report failure")
	}
	// And it proved the claim by verifying, not by asserting trust blind.
	if !ex.ran("verify-cert") {
		t.Error("success was reported without actually verifying the root is trusted")
	}
}

// SECURITY: a gateway not served by Caddy's own CA has no internal root of
// ours to install. Trust-cert must refuse it rather than install some other
// file as a root authority. tlsGatewayServer serves a valid files certificate,
// so the effective source is "files".
func TestGatewayTrustCert_RefusesAnythingButItsOwnInternalRoot(t *testing.T) {
	now := time.Now()
	a := tlsGatewayServer(t, map[string][]byte{
		testCertPath: issueLeaf(t, "gw.valve.city", now.Add(-time.Hour), now.Add(24*time.Hour)),
		testKeyPath:  []byte("key"),
	})

	res := a.do(t, "POST", "/api/gateways/default/trust-cert", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if !strings.Contains(body.Error, catalog.CertFiles) {
		t.Errorf("refusal should name the cert source that is not ours: %q", body.Error)
	}
}

// A gateway with no HTTPS front has no certificate to trust; the refusal points
// at the setting to turn on rather than failing obscurely.
func TestGatewayTrustCert_RefusesWhenNotFronted(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addGateway(t, a, "default", "local", pulsechainOnly(4100))

	res := a.do(t, "POST", "/api/gateways/default/trust-cert", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if body.Code != codeNotConfigured {
		t.Errorf("code: got %q, want %q", body.Code, codeNotConfigured)
	}
}

// The root is written by the HTTPS front at provision time. If it is not there
// yet, the operator is told to create the gateway first — not handed a command
// that would install nothing (or worse, whatever else is at that path).
func TestGatewayTrustCert_SaysWhenTheRootIsNotExportedYet(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addGateway(t, a, "default", "local", internalTLSGateway(4100))

	res := a.do(t, "POST", "/api/gateways/default/trust-cert", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if !strings.Contains(body.Error, "not been exported") {
		t.Errorf("message should say the root is not exported yet: %q", body.Error)
	}
}
