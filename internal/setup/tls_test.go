package setup

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

const (
	tlsHost     = "gw.example"
	tlsCertPath = "/etc/valve/cert.pem"
	tlsKeyPath  = "/etc/valve/key.pem"
)

// testCert mints a self-signed leaf so each fallback trigger can be produced
// exactly rather than approximated.
func testCert(t *testing.T, notBefore, notAfter time.Time, dnsNames ...string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		DNSNames:     dnsNames,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func filesTLS() *catalog.GatewayTLS {
	return &catalog.GatewayTLS{
		Enabled:    true,
		Hostname:   tlsHost,
		CertSource: catalog.CertFiles,
		CertFile:   tlsCertPath,
		KeyFile:    tlsKeyPath,
	}
}

// withCertFiles puts a certificate and a key on the fake target.
func withCertFiles(t *testing.T, e *fakeExecutor, certPEM []byte) *fakeExecutor {
	t.Helper()
	ctx := context.Background()
	if certPEM != nil {
		if err := e.WriteFile(ctx, tlsCertPath, certPEM, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.WriteFile(ctx, tlsKeyPath, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	return e
}

// ---------------------------------------------------------------------
// the auto-fallback
// ---------------------------------------------------------------------

// EVERY trigger, because the whole argument for making this automatic is that
// a certificate on disk fails on a schedule nobody here controls, and the
// failure must be a one-time trust prompt rather than a dead endpoint.
func TestResolveTLSFront_FallsBackAndSaysWhy(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		cert       []byte
		host       string
		wantReason string
		wantSaid   string
	}{
		{
			name: "missing file",
			cert: nil,
			// The key is still written by withCertFiles, so this is precisely
			// "the certificate is not where you said it is".
			wantReason: catalog.CertProblemMissing,
			wantSaid:   tlsCertPath,
		},
		{
			name:       "unparseable",
			cert:       []byte("-----BEGIN CERTIFICATE-----\nnot base64\n-----END CERTIFICATE-----\n"),
			wantReason: catalog.CertProblemUnparseable,
		},
		{
			name:       "expired",
			cert:       testCert(t, now.Add(-400*24*time.Hour), now.Add(-3*24*time.Hour), tlsHost),
			wantReason: catalog.CertProblemExpired,
			wantSaid:   "3 days",
		},
		{
			name:       "hostname not covered",
			cert:       testCert(t, now.Add(-time.Hour), now.Add(720*time.Hour), "someone-else.example"),
			wantReason: catalog.CertProblemMismatch,
			wantSaid:   "someone-else.example",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := withCertFiles(t, newFakeExecutor(), tc.cert)

			front, err := resolveTLSFront(context.Background(), e, filesTLS(), "erpc", 4000)
			if err != nil {
				t.Fatalf("resolveTLSFront: %v", err)
			}
			if front.FallbackReason != tc.wantReason {
				t.Fatalf("reason = %q, want %q (said: %s)", front.FallbackReason, tc.wantReason, front.Fallback)
			}
			// HTTPS stays ON. The alternative — refusing to serve — turns a
			// scheduled, inevitable expiry into an outage.
			if front.Caddy.CertSourceOrDefault() != catalog.CertInternal {
				t.Errorf("want a fallback to the internal CA, got %q", front.Caddy.CertSource)
			}
			if front.Caddy.CertFile != "" || front.Caddy.KeyFile != "" {
				t.Error("the unusable paths must be dropped, or Caddy is handed the same broken file")
			}
			// Falling back QUIETLY would be the very behaviour this avoids.
			if front.Fallback == "" {
				t.Fatal("a fallback with no stated reason is the silent failure this exists to prevent")
			}
			if tc.wantSaid != "" && !strings.Contains(front.Fallback, tc.wantSaid) {
				t.Errorf("reason %q must mention %q", front.Fallback, tc.wantSaid)
			}
			// It must render — a fallback that cannot produce a Caddyfile is
			// not a fallback.
			if _, err := catalog.RenderCaddyfile(front.Caddy); err != nil {
				t.Errorf("the fallback config must render: %v", err)
			}
		})
	}
}

// A missing PRIVATE KEY falls back too. Caddy would otherwise fail to start
// and the gateway would have no HTTPS at all, which is the outcome the whole
// mechanism exists to avoid.
func TestResolveTLSFront_MissingKeyAlsoFallsBack(t *testing.T) {
	e := newFakeExecutor()
	if err := e.WriteFile(context.Background(), tlsCertPath, testCert(t, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), tlsHost), 0644); err != nil {
		t.Fatal(err)
	}
	front, err := resolveTLSFront(context.Background(), e, filesTLS(), "erpc", 4000)
	if err != nil {
		t.Fatalf("resolveTLSFront: %v", err)
	}
	if front.FallbackReason != catalog.CertProblemMissing || !strings.Contains(front.Fallback, tlsKeyPath) {
		t.Fatalf("want a missing-key fallback naming the key path, got %q / %q", front.FallbackReason, front.Fallback)
	}
}

// A usable certificate is used, unchanged. The fallback must not be a
// one-way door that quietly downgrades every operator who supplied a good file.
func TestResolveTLSFront_UsableCertificateIsKept(t *testing.T) {
	e := withCertFiles(t, newFakeExecutor(), testCert(t, time.Now().Add(-time.Hour), time.Now().Add(720*time.Hour), tlsHost))

	front, err := resolveTLSFront(context.Background(), e, filesTLS(), "valve-node-app-erpc", 4000)
	if err != nil {
		t.Fatalf("resolveTLSFront: %v", err)
	}
	if front.Fallback != "" {
		t.Fatalf("no fallback expected: %s", front.Fallback)
	}
	rendered, err := catalog.RenderCaddyfile(front.Caddy)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(rendered, "tls "+tlsCertPath+" "+tlsKeyPath) {
		t.Errorf("the operator's own files must be used:\n%s", rendered)
	}
	// The upstream is the eRPC CONTAINER NAME on the shared network, which is
	// what lets eRPC publish no host port at all.
	if !strings.Contains(rendered, "reverse_proxy valve-node-app-erpc:4000") {
		t.Errorf("want a container-name upstream:\n%s", rendered)
	}
}

// The internal CA needs nothing from the target and is the default, so it is
// never subject to a fallback.
func TestResolveTLSFront_InternalNeedsNoFiles(t *testing.T) {
	front, err := resolveTLSFront(context.Background(), newFakeExecutor(),
		&catalog.GatewayTLS{Enabled: true, Hostname: tlsHost}, "erpc", 4000)
	if err != nil {
		t.Fatalf("resolveTLSFront: %v", err)
	}
	if front.Fallback != "" || front.Caddy.CertSourceOrDefault() != catalog.CertInternal {
		t.Fatalf("got %+v", front)
	}
	if f, err := resolveTLSFront(context.Background(), newFakeExecutor(), nil, "erpc", 4000); err != nil || f != nil {
		t.Fatalf("no TLS means no front: %v, %v", f, err)
	}
}

// ---------------------------------------------------------------------
// provisioning a fronted gateway
// ---------------------------------------------------------------------

func frontedGateway() catalog.GatewayConfig {
	g := testGateway()
	g.TLS = &catalog.GatewayTLS{Enabled: true, Hostname: tlsHost, HTTPSPort: 8443}
	return g
}

// caddyReady is dockerReady plus the two things a TLS front adds: an internal
// CA root readable out of the container, and an https probe that answers.
func caddyReady() *fakeExecutor {
	return dockerReady().
		script("'exec' 'valve-node-app-caddy' 'cat'", executor.Result{
			Stdout: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n",
		})
}

func TestGatewayRun_FrontedGatewayPublishesOnlyCaddy(t *testing.T) {
	shrinkGatewayWait(t)
	e := caddyReady()
	steps := mustPlanGateway(t, frontedGateway(), BackendDocker)

	if err := stepByID(t, steps, "config").Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("config: %v", err)
	}
	if err := stepByID(t, steps, "run").Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	var erpcRun, caddyRun, networkEnsured string
	for _, cmd := range e.callLog() {
		switch {
		case strings.Contains(cmd, "'network'"):
			networkEnsured = cmd
		case strings.Contains(cmd, "'run' '-d' '--name' 'valve-node-app-erpc'"):
			erpcRun = cmd
		case strings.Contains(cmd, "'run' '-d' '--name' 'valve-node-app-caddy'"):
			caddyRun = cmd
		}
	}

	if networkEnsured == "" {
		t.Fatalf("the private network must be ensured before anything joins it: %#v", e.callLog())
	}
	if erpcRun == "" || caddyRun == "" {
		t.Fatalf("want both containers created: %#v", e.callLog())
	}
	// Caddy is the only RPC front door. A published eRPC port would be a
	// second, plaintext, unauthenticated way in that the operator did not ask
	// for — so the invariant is that no mapping reaches eRPC's RPC port, NOT
	// that eRPC publishes nothing at all.
	//
	// The one mapping it does get is the metrics port, pinned to loopback. That
	// is a deliberate narrowing of the older "publishes nothing" rule rather
	// than an erosion of it: without it, the recommended configuration (HTTPS
	// on) would be the single configuration whose traffic share could never be
	// read. Both halves are asserted, because a regression that widened this to
	// 0.0.0.0 or pointed it at 4000 would otherwise still pass.
	if strings.Contains(erpcRun, ":"+strconv.Itoa(ops.ERPCContainerPort)+"'") {
		t.Errorf("a fronted eRPC must publish no RPC port: %s", erpcRun)
	}
	wantMetrics := "'-p' '127.0.0.1:4001:" + strconv.Itoa(ops.ERPCContainerMetricsPort) + "'"
	if !strings.Contains(erpcRun, wantMetrics) {
		t.Errorf("a fronted eRPC must still publish its counters on loopback (%s): %s", wantMetrics, erpcRun)
	}
	if strings.Contains(erpcRun, "'-p' '0.0.0.0:4001") {
		t.Errorf("the metrics port must never be widened past loopback: %s", erpcRun)
	}
	if !strings.Contains(caddyRun, "'-p' '0.0.0.0:8443:443'") {
		t.Errorf("want the TLS front published on the configured port: %s", caddyRun)
	}
	for _, want := range []string{"'--network' '" + ops.NetworkName + "'"} {
		if !strings.Contains(erpcRun, want) || !strings.Contains(caddyRun, want) {
			t.Errorf("both containers must join %s", ops.NetworkName)
		}
	}
	// The data volume is what keeps the internal CA stable across recreates.
	if !strings.Contains(caddyRun, catalog.CaddyDataVolume+":"+catalog.CaddyDataPath) {
		t.Errorf("the CA volume is not optional: %s", caddyRun)
	}
}

// The Caddyfile is written beside erpc.yaml and carries the resolved upstream.
func TestGatewayConfigStep_WritesTheCaddyfile(t *testing.T) {
	e := caddyReady()
	step := stepByID(t, mustPlanGateway(t, frontedGateway(), BackendDocker), "config")

	if err := step.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("config: %v", err)
	}
	got, err := e.ReadFile(context.Background(), "/Users/dev/.valve-node-app/Caddyfile")
	if err != nil {
		t.Fatalf("Caddyfile not written: %v", err)
	}
	for _, want := range []string{tlsHost + " {", "tls internal", "reverse_proxy valve-node-app-erpc:4000"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("Caddyfile missing %q:\n%s", want, got)
		}
	}
	// Verify must now hold, or the run step would restart on every pass.
	if err := step.Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("verify after write: %v", err)
	}

	// erpc.yaml must have gzip off, because Caddy is in front of it.
	cfg, err := e.ReadFile(context.Background(), "/Users/dev/.valve-node-app/erpc.yaml")
	if err != nil {
		t.Fatalf("erpc.yaml: %v", err)
	}
	if !strings.Contains(string(cfg), "enableGzip: false") {
		t.Errorf("a fronted gateway must disable gzip or eth_subscribe fails:\n%s", cfg)
	}
}

// A Caddyfile from a run with a different hostname is drift, not success —
// otherwise the old TLS front keeps serving while setup reports done.
func TestGatewayConfigStep_CaddyfileDriftIsDetected(t *testing.T) {
	e := caddyReady()
	if err := e.WriteFile(context.Background(), "/Users/dev/.valve-node-app/Caddyfile", []byte("old.example {\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	step := stepByID(t, mustPlanGateway(t, frontedGateway(), BackendDocker), "config")
	err := step.Verify(context.Background(), e, &State{})
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("want drift, got %v", err)
	}
}

// The readiness probe goes through HTTPS and VERIFIES THE CHAIN. Passing -k
// would make it unable to tell a working front from one serving a certificate
// for the wrong name — the exact failure this feature introduces.
func TestGatewayCheck_FrontedProbeUsesHTTPSAndVerifies(t *testing.T) {
	e := caddyReady()
	p := &gatewayPlan{id: testGatewayID, gw: frontedGateway(), backend: BackendDocker}

	url, cmd, err := p.probeCommand(context.Background(), e, 369)
	if err != nil {
		t.Fatalf("probeCommand: %v", err)
	}
	if !strings.HasPrefix(url, "https://"+tlsHost+":8443/") {
		t.Errorf("url = %q, want the https front door", url)
	}
	if strings.Contains(cmd, " -k") || strings.Contains(cmd, "--insecure") {
		t.Errorf("the probe must verify the chain: %s", cmd)
	}
	if !strings.Contains(cmd, "--cacert") {
		t.Errorf("want the exported internal CA root used as the trust anchor: %s", cmd)
	}
	// Pinned to the published bind address on the TARGET, so a gateway that
	// works fails setup only for real reasons and not for un-pointed DNS.
	if !strings.Contains(cmd, "--resolve '"+tlsHost+":8443:127.0.0.1'") {
		t.Errorf("want DNS pinned to the target's own loopback: %s", cmd)
	}

	// An unfronted gateway is unchanged: plain http on the published port.
	plain := &gatewayPlan{id: testGatewayID, gw: testGateway(), backend: BackendDocker}
	url, cmd, err = plain.probeCommand(context.Background(), e, 369)
	if err != nil {
		t.Fatalf("probeCommand: %v", err)
	}
	if !strings.HasPrefix(url, "http://127.0.0.1:4100/") || strings.Contains(cmd, "--cacert") {
		t.Errorf("unfronted probe changed: %q / %q", url, cmd)
	}
}

// A TLS front is a container, so it cannot be put in front of a unit.
func TestPlanGateway_TLSNeedsTheContainerBackend(t *testing.T) {
	_, err := PlanGateway("default", frontedGateway(), BackendSystemd)
	if err == nil || !strings.Contains(err.Error(), BackendDocker) {
		t.Fatalf("want a refusal naming the docker backend, got %v", err)
	}
}

// Turning HTTPS off must take the front with it, or the old Caddy keeps
// serving from a Caddyfile pointing at an eRPC that has moved back to a
// published host port.
func TestGatewayRun_UnfrontedRemovesAnyStaleTLSFront(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady()
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "run")
	if err := step.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	var removed bool
	for _, cmd := range e.callLog() {
		removed = removed || strings.Contains(cmd, "'rm' '-f' 'valve-node-app-caddy'")
	}
	if !removed {
		t.Fatalf("want the stale front removed: %#v", e.callLog())
	}
}

// The internal CA's root is copied to the HOST, because it is the file the
// operator installs in their trust store — and "it is inside a container, at
// this path" is a hostile way to end a setup flow.
func TestExportRootCA(t *testing.T) {
	e := caddyReady()
	got, err := exportRootCA(context.Background(), e, "valve-node-app-caddy", "/Users/dev/.valve-node-app/caddy-root.crt")
	if err != nil || !got {
		t.Fatalf("exportRootCA = %v, %v", got, err)
	}
	b, err := e.ReadFile(context.Background(), "/Users/dev/.valve-node-app/caddy-root.crt")
	if err != nil || !strings.Contains(string(b), "BEGIN CERTIFICATE") {
		t.Fatalf("root not written: %q, %v", b, err)
	}

	// Caddy writes the root during startup, so a probe that finds nothing yet
	// is a retry, not a failure.
	early := newFakeExecutor().script("'exec'", executor.Result{ExitCode: 1, Stderr: "No such file"})
	if got, err := exportRootCA(context.Background(), early, "c", "/tmp/root.crt"); got || err != nil {
		t.Fatalf("want a soft no, got %v / %v", got, err)
	}
}
