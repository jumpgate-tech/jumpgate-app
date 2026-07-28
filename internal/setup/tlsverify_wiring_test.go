package setup

// The other half of the live-HTTPS verification: not the assertions themselves
// (tlsverify_test.go runs those against a real TLS server) but the WIRING —
// how a stored gateway config plus a target become the probe that gets run.
//
// It is worth its own tests because every input the probe carries is derived,
// and a derivation that is quietly wrong produces a verification that passes
// while checking the wrong thing: the wrong port, the wrong chain path, or —
// worst — the system trust store instead of the CA the operator installed,
// which turns "we cannot check this" into a confident wrong answer.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// rootPEM re-encodes a pool's single CA as PEM, which is what the target holds
// at the exported root path.
func rootPEM(t *testing.T, cert tls.Certificate) []byte {
	t.Helper()
	// The stub's chain is [leaf, ca]; the CA is what gets exported.
	if len(cert.Certificate) < 2 {
		t.Fatal("test certificate has no CA in its chain")
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[1]})
}

// serveGateway stands up a real TLS server for g and returns a GatewayConfig
// pointed at it, plus the port it landed on.
func serveGateway(t *testing.T, stub gatewayStub, cert tls.Certificate) (catalog.GatewayConfig, string) {
	t.Helper()
	srv := httptest.NewUnstartedServer(stub)
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	host, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	if err != nil {
		t.Fatalf("split %q: %v", srv.URL, err)
	}
	return catalog.GatewayConfig{
		Networks: []catalog.GatewayNetwork{{ChainID: stub.chainID, Upstreams: []catalog.GatewayUpstream{
			{ID: "up", Endpoint: "https://rpc.example"},
		}}},
		TLS: &catalog.GatewayTLS{
			Enabled:   true,
			Hostname:  verifyHost,
			HTTPSPort: atoi(t, port),
		},
	}, host
}

// tlsFakeExecutor is a fakeExecutor whose ReadFile serves the files a target
// would hold — principally the exported internal-CA root.
type tlsFakeExecutor struct {
	*fakeExecutor
	files   map[string][]byte
	readErr error
}

func (f *tlsFakeExecutor) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	for name, b := range f.files {
		if strings.HasSuffix(path, name) {
			return b, nil
		}
	}
	return f.fakeExecutor.ReadFile(ctx, path)
}

func newTLSExecutor(files map[string][]byte) *tlsFakeExecutor {
	e := newFakeExecutor()
	// The docker backend keeps erpc.yaml — and the exported root beside it —
	// under the target's $HOME, so the path resolution needs one to answer.
	e.script(`"$HOME"`, executor.Result{Stdout: "/home/ops\n"})
	return &tlsFakeExecutor{fakeExecutor: e, files: files}
}

// ---------------------------------------------------------------------
// there has to be something to verify
// ---------------------------------------------------------------------

// A gateway with no HTTPS front returns the typed error, so the caller can
// tell "not configured" from "configured and broken" — they need opposite
// things said to them.
func TestVerifyGatewayTLS_RefusesAGatewayWithNoFront(t *testing.T) {
	e := newTLSExecutor(nil)

	for name, g := range map[string]catalog.GatewayConfig{
		"no TLS at all": {Networks: []catalog.GatewayNetwork{{ChainID: 1337}}},
		"TLS off":       {TLS: &catalog.GatewayTLS{Enabled: false, Hostname: verifyHost}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := VerifyGatewayTLS(context.Background(), e, "default", g, "127.0.0.1")
			if !errors.Is(err, ErrNoTLSFront) {
				t.Fatalf("got %v, want ErrNoTLSFront", err)
			}
		})
	}
}

// ---------------------------------------------------------------------
// the whole path, against a real front
// ---------------------------------------------------------------------

// End to end: a stored config plus a target's exported root produce a probe
// that verifies a genuinely-serving front. This is the case that proves the
// derivations are right — every assertion passing means the port, the chain
// path and the trust root were all resolved correctly, because getting any one
// of them wrong fails a different assertion.
func TestVerifyGatewayTLS_VerifiesARealFrontAgainstTheExportedRoot(t *testing.T) {
	_, cert := testCA(t, verifyHost)
	g, dialHost := serveGateway(t, gatewayStub{chainID: 1337}, cert)
	e := newTLSExecutor(map[string][]byte{"caddy-root.crt": rootPEM(t, cert)})

	got, err := VerifyGatewayTLS(context.Background(), e, "default", g, dialHost)
	if err != nil {
		t.Fatalf("VerifyGatewayTLS: %v", err)
	}

	// The trust source must name the exported file. "the system trust store"
	// here would mean the check passed for the wrong reason.
	if !strings.Contains(got.TrustSource, "caddy-root.crt") {
		t.Errorf("trust source: got %q, want the root exported to the target", got.TrustSource)
	}
	// The reported URL is what the operator copies out of this screen, so it
	// has to carry the port the front actually publishes. It is derived
	// separately from the address that gets dialled, which means a verification
	// can pass while displaying a URL that goes nowhere.
	if !strings.Contains(got.URL, fmt.Sprint(g.TLS.HTTPSPort)) {
		t.Errorf("reported URL %q does not carry the published port %d", got.URL, g.TLS.HTTPSPort)
	}
	for _, id := range []string{TLSAssertHandshake, TLSAssertChain, TLSAssertHostname, TLSAssertRPC} {
		wantStatus(t, got, id, "pass")
	}
}

// The chain assertion must verify against the root the OPERATOR installed, not
// against the system store. If the exported root cannot be read, the honest
// answer is "we could not check" — silently falling back to the system store
// would report a self-signed front as untrusted-but-fine, or worse, pass a
// front that no browser will accept.
func TestVerifyGatewayTLS_SaysItCouldNotCheckWhenTheRootIsMissing(t *testing.T) {
	_, cert := testCA(t, verifyHost)
	g, dialHost := serveGateway(t, gatewayStub{chainID: 1337}, cert)

	e := newTLSExecutor(nil)
	e.readErr = errors.New("no such file")

	got, err := VerifyGatewayTLS(context.Background(), e, "default", g, dialHost)
	if err != nil {
		t.Fatalf("VerifyGatewayTLS: %v", err)
	}

	chain := assertionOf(t, got, TLSAssertChain)
	if chain.Status == "pass" {
		t.Fatal("the chain assertion passed without a root to verify against — that is a wrong answer, not a missing one")
	}
	if !strings.Contains(chain.Detail, "no such file") {
		t.Errorf("the detail must carry why the root could not be read: %q", chain.Detail)
	}
	// The handshake still happened, so the operator learns the front is up
	// even though the chain could not be judged.
	wantStatus(t, got, TLSAssertHandshake, "pass")
}

// A file that is not a PEM certificate is a different failure from an absent
// one, and it has a different fix — the message must not say "missing".
func TestVerifyGatewayTLS_SaysWhenTheExportedRootIsNotACertificate(t *testing.T) {
	_, cert := testCA(t, verifyHost)
	g, dialHost := serveGateway(t, gatewayStub{chainID: 1337}, cert)
	e := newTLSExecutor(map[string][]byte{"caddy-root.crt": []byte("this is not a certificate\n")})

	got, err := VerifyGatewayTLS(context.Background(), e, "default", g, dialHost)
	if err != nil {
		t.Fatalf("VerifyGatewayTLS: %v", err)
	}
	chain := assertionOf(t, got, TLSAssertChain)
	if chain.Status == "pass" {
		t.Fatal("the chain assertion passed against a root that is not a certificate")
	}
	if !strings.Contains(chain.Detail, "not a PEM certificate") {
		t.Errorf("detail %q does not distinguish an unparseable root from a missing one", chain.Detail)
	}
}

// The path comes from the gateway's FIRST network, and it is what the probe
// posts eth_chainId to. Deriving it wrongly means verifying a path eRPC does
// not serve, which fails an otherwise healthy gateway.
func TestVerifyGatewayTLS_ProbesTheChainPathTheGatewayActuallyServes(t *testing.T) {
	_, cert := testCA(t, verifyHost)
	g, dialHost := serveGateway(t, gatewayStub{chainID: 1337}, cert)
	g.ProjectID = "fleet"
	e := newTLSExecutor(map[string][]byte{"caddy-root.crt": rootPEM(t, cert)})

	got, err := VerifyGatewayTLS(context.Background(), e, "default", g, dialHost)
	if err != nil {
		t.Fatalf("VerifyGatewayTLS: %v", err)
	}
	want := g.PathFor(1337)
	if !strings.Contains(got.URL, want) {
		t.Errorf("probed URL %q does not carry the gateway's own path %q", got.URL, want)
	}
	if !strings.Contains(got.URL, "fleet") {
		t.Errorf("probed URL %q ignores the project id, so it is not the path eRPC serves", got.URL)
	}
	wantStatus(t, got, TLSAssertRPC, "pass")
}

// A gateway serving a different chain than the one configured is the failure
// that looks like success everywhere else: the front is up, the certificate is
// good, and every call returns data for the wrong network.
func TestVerifyGatewayTLS_CatchesAFrontServingTheWrongChain(t *testing.T) {
	_, cert := testCA(t, verifyHost)
	// The front answers 1, the config says 1337.
	g, dialHost := serveGateway(t, gatewayStub{chainID: 1}, cert)
	g.Networks[0].ChainID = 1337
	e := newTLSExecutor(map[string][]byte{"caddy-root.crt": rootPEM(t, cert)})

	got, err := VerifyGatewayTLS(context.Background(), e, "default", g, dialHost)
	if err != nil {
		t.Fatalf("VerifyGatewayTLS: %v", err)
	}
	// TLSAssertRPC is the BLOCKCHAIN chain id, not the certificate chain: the
	// certificate here is perfectly good, which is exactly what makes this
	// failure invisible without the eth_chainId probe.
	if a := assertionOf(t, got, TLSAssertRPC); a.Status == "pass" {
		t.Fatalf("a front answering chain 1 passed for a gateway configured as 1337: %+v", a)
	}
	if !strings.Contains(assertionOf(t, got, TLSAssertRPC).Detail, "1337") {
		t.Errorf("the detail must name the chain that was expected: %q", assertionOf(t, got, TLSAssertRPC).Detail)
	}
	// The certificate chain is fine, and must still say so.
	wantStatus(t, got, TLSAssertChain, "pass")
}

// A certificate FILE the target cannot read makes the front fall back to the
// internal CA. The verification has to follow that fallback rather than
// checking the source the operator configured — otherwise it reports a
// mismatch against a certificate that is not being served.
func TestVerifyGatewayTLS_FollowsTheFallbackRatherThanTheConfiguredSource(t *testing.T) {
	_, cert := testCA(t, verifyHost)
	g, dialHost := serveGateway(t, gatewayStub{chainID: 1337}, cert)
	g.TLS.CertSource = catalog.CertFiles
	g.TLS.CertFile = "/etc/valve/gone.crt"
	g.TLS.KeyFile = "/etc/valve/gone.key"

	// The configured files are absent; the exported internal root is present.
	e := newTLSExecutor(map[string][]byte{"caddy-root.crt": rootPEM(t, cert)})

	got, err := VerifyGatewayTLS(context.Background(), e, "default", g, dialHost)
	if err != nil {
		t.Fatalf("VerifyGatewayTLS: %v", err)
	}
	// Having fallen back to the internal CA, the check must verify against the
	// exported root — and does, because the chain assertion passes.
	if !strings.Contains(got.TrustSource, "caddy-root.crt") {
		t.Errorf("trust source: got %q, want the internal CA's exported root after the fallback", got.TrustSource)
	}
	wantStatus(t, got, TLSAssertChain, "pass")
}

// ---------------------------------------------------------------------
// expectedRoots on the cert-files path
// ---------------------------------------------------------------------

// With a configured certificate file, the expected roots are the system store
// PLUS that certificate — a `tailscale cert` is publicly trusted and needs the
// first, a self-signed file is its own root and needs the second, and making
// the operator declare which they have would only be a way to get it wrong.
func TestExpectedRoots_CertFilesTrustsTheSystemStoreAndTheConfiguredFile(t *testing.T) {
	_, cert := testCA(t, verifyHost)
	leaf := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	e := newTLSExecutor(map[string][]byte{"gw.crt": leaf})

	g := catalog.GatewayConfig{TLS: &catalog.GatewayTLS{
		Enabled:    true,
		Hostname:   verifyHost,
		CertSource: catalog.CertFiles,
		CertFile:   "/etc/valve/gw.crt",
		KeyFile:    "/etc/valve/gw.key",
	}}
	p := &gatewayPlan{id: "default", gw: g, backend: BackendDocker}
	front := &tlsFront{Caddy: g.TLS.Caddy("valve-node-app-erpc", 4000)}

	pool, trust, rootErr := p.expectedRoots(context.Background(), e, front)
	if rootErr != "" {
		t.Fatalf("rootErr: %s", rootErr)
	}
	if pool == nil {
		t.Fatal("no pool")
	}
	if !strings.Contains(trust, "system trust store") {
		t.Errorf("trust source %q does not mention the system store", trust)
	}
	if !strings.Contains(trust, "/etc/valve/gw.crt") {
		t.Errorf("trust source %q does not mention the configured certificate", trust)
	}
}

// A configured certificate the target cannot read must not fail the whole
// resolution: the system store alone is still a legitimate answer for a
// publicly-trusted certificate, and the file's absence is already reported by
// the front's own fallback.
func TestExpectedRoots_CertFilesStillTrustsTheSystemStoreWhenTheFileIsGone(t *testing.T) {
	e := newTLSExecutor(nil)
	e.readErr = errors.New("no such file")

	g := catalog.GatewayConfig{TLS: &catalog.GatewayTLS{
		Enabled:    true,
		Hostname:   verifyHost,
		CertSource: catalog.CertFiles,
		CertFile:   "/etc/valve/gone.crt",
	}}
	p := &gatewayPlan{id: "default", gw: g, backend: BackendDocker}
	front := &tlsFront{Caddy: g.TLS.Caddy("valve-node-app-erpc", 4000)}

	pool, trust, rootErr := p.expectedRoots(context.Background(), e, front)
	if rootErr != "" {
		t.Fatalf("an unreadable certificate file failed the whole resolution: %s", rootErr)
	}
	if pool == nil {
		t.Fatal("no pool")
	}
	if strings.Contains(trust, "gone.crt") {
		t.Errorf("trust source %q claims a file it could not read", trust)
	}
}

// systemPoolIsUsable guards the assumption the two tests above rest on.
func TestExpectedRoots_TheSystemPoolIsAvailableInThisEnvironment(t *testing.T) {
	if _, err := x509.SystemCertPool(); err != nil {
		t.Skipf("no system cert pool here: %v", err)
	}
}
