package setup

// "Is HTTPS actually serving?", asked of a RUNNING gateway rather than of a
// provisioning run.
//
// The provisioner already proves the front works once, at the moment it is
// created (gatewayCheck / probeCommand). That is not the same question. A TLS
// front decays on a clock nobody here controls — certificates expire, a name
// stops covering the host it is served for, an upstream is re-pointed — so the
// only useful answer is one the operator can ask for again tomorrow and see.
//
// FIVE assertions, because each one catches a failure the others report as
// healthy. They are deliberately not collapsed into a single "curl succeeded":
//
//  1. HANDSHAKE. Something on that port speaks TLS at all. A plaintext server
//     on the TLS port is a real outcome — it is what a mis-rendered Caddyfile
//     or a port collision with something else produces.
//  2. HOSTNAME. The certificate presented COVERS the configured name. A valid
//     certificate for the wrong name is the most common real failure and it
//     looks perfect from every angle except the one that matters: a browser
//     refuses it outright.
//  3. CHAIN. It verifies against the root we EXPECT — Caddy's exported root
//     for the internal CA, the system trust store for a publicly-trusted one.
//     Nothing here ever accepts an unverified chain as a pass: the verification
//     is relocated (see inspect) so a failure can be attributed to the right
//     assertion, never skipped.
//  4. RPC THROUGH IT. eth_chainId over https returns the CORRECT chain id for
//     the gateway's first network. Terminating TLS and actually reaching eRPC
//     are two different things, and a front pointed at a dead upstream passes
//     the first three assertions.
//  5. SUBSCRIPTIONS. eth_subscribe over wss:// delivers, or is reported as
//     unavailable in as many words. This is load-bearing and was MEASURED, not
//     assumed: eRPC infers WebSocket capability from the upstream SCHEME alone,
//     so an http:// upstream serves eth_chainId over wss perfectly and refuses
//     eth_subscribe. A check that stopped at assertion 4 would call that
//     healthy, and the operator would find out from a dApp that never receives
//     a block.
//
// Plus the certificate's NotAfter and the time left on it — an expiry is only
// useful BEFORE it breaks — and one cheap negative: plain http:// on the TLS
// port must not serve RPC.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/wsrpc"
)

// The assertion ids, stable so a UI can branch on them without matching prose.
const (
	TLSAssertHandshake = "handshake"
	TLSAssertHostname  = "hostname"
	TLSAssertChain     = "chain"
	TLSAssertRPC       = "rpc"
	TLSAssertSubscribe = "subscribe"
	TLSAssertPlaintext = "plaintext-refused"
)

// The per-assertion outcomes.
//
// TLSStatusUnavailable is NOT a fourth spelling of "fail": it is the answer
// assertion 5 needs. A gateway whose upstream is http:// serves HTTPS
// perfectly and cannot serve subscriptions, and calling that a failure of the
// HTTPS front would be wrong in the other direction. It is a capability that
// is missing, reported as missing.
const (
	TLSStatusPass        = "pass"
	TLSStatusFail        = "fail"
	TLSStatusSkip        = "skip"
	TLSStatusUnavailable = "unavailable"
)

// TLSAssertion is one thing that was checked and what it found.
type TLSAssertion struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Why says what this assertion catches that the others do not. It is on
	// the wire rather than in the UI because the reason a check exists is the
	// part that stops it from being deleted as redundant.
	Why    string `json:"why"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// TLSVerification is one whole run.
type TLSVerification struct {
	At       time.Time `json:"at"`
	URL      string    `json:"url"`
	Hostname string    `json:"hostname"`
	// Address is what was actually dialed. The name is PINNED to it exactly as
	// the provisioner's curl --resolve pins it: DNS is not what is under test,
	// and a gateway that serves perfectly must not be reported as broken
	// because the operator has not pointed a name at their machine yet.
	Address string `json:"address"`
	ChainID int    `json:"chainId,omitempty"`
	Path    string `json:"path,omitempty"`

	// CertSource is the EFFECTIVE source (after any fallback), and TrustSource
	// names what the chain was verified against, so a pass says what it means.
	CertSource  string `json:"certSource,omitempty"`
	TrustSource string `json:"trustSource,omitempty"`

	// The certificate actually presented on the wire, not the one on disk.
	Subject   string     `json:"subject,omitempty"`
	Issuer    string     `json:"issuer,omitempty"`
	NotBefore *time.Time `json:"notBefore,omitempty"`
	NotAfter  *time.Time `json:"notAfter,omitempty"`
	// ExpiresIn is the time left, rendered the way catalog renders every other
	// certificate duration.
	ExpiresIn string `json:"expiresIn,omitempty"`
	// ExpiryWarning is set only when the expiry is a thing the operator has to
	// act on. Caddy's internal CA issues 12-hour leaves and renews them in
	// process, so warning about those would be crying wolf every hour of every
	// day; a certificate FILE that runs out is nobody's job but theirs.
	ExpiryWarning string `json:"expiryWarning,omitempty"`

	Assertions []TLSAssertion `json:"assertions"`

	// OK is the HTTPS verdict: every assertion that bears on "is this serving
	// HTTPS correctly" passed. Assertion 5 is deliberately excluded — see
	// TLSStatusUnavailable — and reported separately.
	OK              bool   `json:"ok"`
	SubscriptionsOK bool   `json:"subscriptionsOk"`
	Summary         string `json:"summary"`
}

// ErrNoTLSFront is "this gateway has no HTTPS front", which is a question that
// cannot be answered rather than an answer of "no".
var ErrNoTLSFront = errors.New("tls: this gateway does not serve HTTPS, so there is no TLS front to verify")

const (
	// tlsVerifyDialTimeout bounds each individual connection.
	tlsVerifyDialTimeout = 10 * time.Second
	// tlsSubscribeWait is how long a newHeads notification is waited for after
	// the subscription is accepted. It is short on purpose: the subscription id
	// already proves eRPC accepted the subscribe on a WS-capable upstream, and
	// whether a block happens to be produced in the next few seconds is the
	// chain's business, not the gateway's.
	tlsSubscribeWait = 12 * time.Second
	// tlsExpiryWarnWithin is when a certificate FILE's expiry starts being the
	// operator's problem today rather than eventually.
	tlsExpiryWarnWithin = 14 * 24 * time.Hour
)

// gatewaySubscribeCall is the probe body assertion 5 posts over wss://.
const gatewaySubscribeCall = `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`

// VerifyGatewayTLS runs the whole check against a gateway's live HTTPS front.
//
// dialHost is the address the hostname is pinned to — the target's own
// loopback for a local machine, its SSH host for a remote one. The executor is
// used only to READ (the internal CA's root, a configured certificate file):
// nothing is written and no container is touched, so this is safe to run
// against a gateway that is serving traffic.
func VerifyGatewayTLS(ctx context.Context, e executor.Executor, gatewayID string, g catalog.GatewayConfig, dialHost string) (TLSVerification, error) {
	if !g.Fronted() {
		return TLSVerification{}, ErrNoTLSFront
	}

	p := &gatewayPlan{id: gatewayID, gw: g, backend: BackendDocker}
	front, err := p.front(ctx, e)
	if err != nil {
		return TLSVerification{}, err
	}

	probe := tlsProbe{
		Hostname:   front.Caddy.Hostname,
		Port:       g.TLS.HTTPS(),
		Address:    net.JoinHostPort(strings.Trim(dialHost, "[]"), fmt.Sprint(g.TLS.HTTPS())),
		CertSource: front.Caddy.CertSourceOrDefault(),
	}
	if len(g.Networks) > 0 {
		probe.ChainID = g.Networks[0].ChainID
		probe.Path = g.PathFor(probe.ChainID)
	}
	probe.Roots, probe.TrustSource, probe.RootErr = p.expectedRoots(ctx, e, front)

	return verifyTLSEndpoint(ctx, probe), nil
}

// expectedRoots resolves WHAT the chain is supposed to verify against, which
// is the whole difference between this check and `curl -k`.
//
//   - internal CA: the root Caddy generated, exported to the target at
//     provision time. That file is the one an operator installs in a trust
//     store, so verifying against it is verifying against exactly what their
//     browser will use.
//   - a certificate file: the system trust store PLUS the configured
//     certificate itself, mirroring what the provisioner's curl does with
//     --cacert. A `tailscale cert` is publicly trusted and needs the first; a
//     self-signed file is its own root and needs the second. Trying to make the
//     operator declare which they have would only be a way to get it wrong.
func (p *gatewayPlan) expectedRoots(ctx context.Context, e executor.Executor, front *tlsFront) (*x509.CertPool, string, string) {
	if front.Caddy.CertSourceOrDefault() == catalog.CertInternal {
		path, err := p.rootCAPath(ctx, e)
		if err != nil {
			return nil, "", err.Error()
		}
		pem, err := e.ReadFile(ctx, path)
		if err != nil {
			return nil, "", fmt.Sprintf("the internal CA's root at %s could not be read on the target (%v), so there is nothing to verify the chain against — re-create the TLS front to export it again", path, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, "", fmt.Sprintf("the file at %s is not a PEM certificate, so it cannot be used as the expected root", path)
		}
		return pool, "Caddy's own certificate authority, as exported to " + path, ""
	}

	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	trust := "the system trust store"
	if certPath := strings.TrimSpace(front.Caddy.CertFile); certPath != "" {
		if pem, err := e.ReadFile(ctx, certPath); err == nil && pool.AppendCertsFromPEM(pem) {
			trust += ", plus the configured certificate " + certPath + " as its own root"
		}
	}
	return pool, trust, ""
}

// tlsProbe is one verification's inputs, with no dependency on a gateway, a
// target or an executor — which is what makes the assertions testable against
// an ordinary httptest TLS server.
type tlsProbe struct {
	Hostname    string
	Port        int
	Address     string
	Roots       *x509.CertPool
	TrustSource string
	// RootErr is why the expected root could not be loaded. The chain
	// assertion fails with this rather than quietly falling back to the system
	// store, which would turn "we cannot check" into a wrong answer.
	RootErr    string
	Path       string
	ChainID    int
	CertSource string
}

func (p tlsProbe) url(scheme string) string {
	if p.Port == 443 && scheme == "https" {
		return "https://" + p.Hostname + p.Path
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, p.Hostname, p.Port, p.Path)
}

// verifyTLSEndpoint runs the assertions in order and stops depending on
// nothing: an assertion that cannot be reached is reported as skipped, with
// the reason, rather than left out.
func verifyTLSEndpoint(ctx context.Context, p tlsProbe) TLSVerification {
	v := TLSVerification{
		At:          time.Now(),
		URL:         p.url("https"),
		Hostname:    p.Hostname,
		Address:     p.Address,
		ChainID:     p.ChainID,
		Path:        p.Path,
		CertSource:  p.CertSource,
		TrustSource: p.TrustSource,
	}

	in := p.inspect(ctx)
	v.add(p.handshakeAssertion(in))
	if in.Leaf != nil {
		v.describeCertificate(in.Leaf, p.CertSource)
	}
	v.add(p.hostnameAssertion(in))
	v.add(p.chainAssertion(in))

	trusted := in.Leaf != nil && in.HostnameErr == nil && in.ChainErr == nil && p.RootErr == ""
	v.add(p.rpcAssertion(ctx, trusted))
	v.add(p.subscribeAssertion(ctx, trusted))
	v.add(p.plaintextAssertion(ctx))

	v.finish()
	return v
}

func (v *TLSVerification) add(a TLSAssertion) { v.Assertions = append(v.Assertions, a) }

// finish computes the two verdicts and the sentence that goes above them.
func (v *TLSVerification) finish() {
	v.OK, v.SubscriptionsOK = true, false
	var failed []string
	for _, a := range v.Assertions {
		if a.ID == TLSAssertSubscribe {
			v.SubscriptionsOK = a.Status == TLSStatusPass
			continue
		}
		switch a.Status {
		case TLSStatusFail:
			v.OK = false
			failed = append(failed, a.Title)
		case TLSStatusSkip:
			v.OK = false
		}
	}
	switch {
	case !v.OK && len(failed) > 0:
		v.Summary = fmt.Sprintf("%s is NOT serving HTTPS correctly: %s.", v.URL, strings.Join(failed, "; "))
	case !v.OK:
		v.Summary = fmt.Sprintf("%s could not be fully verified — see the assertions below.", v.URL)
	case v.SubscriptionsOK:
		v.Summary = fmt.Sprintf("%s is serving HTTPS, verified end to end: the chain verifies against %s, chain %d answers through it, and subscriptions deliver.", v.URL, v.TrustSource, v.ChainID)
	default:
		v.Summary = fmt.Sprintf("%s is serving HTTPS — the chain verifies against %s and chain %d answers through it — but SUBSCRIPTIONS ARE NOT AVAILABLE on it, so a dApp waiting on eth_subscribe will wait forever.", v.URL, v.TrustSource, v.ChainID)
	}
}

// describeCertificate reports the certificate actually presented, including
// the one fact nobody sees until it is too late: when it stops working.
func (v *TLSVerification) describeCertificate(leaf *x509.Certificate, certSource string) {
	nb, na := leaf.NotBefore, leaf.NotAfter
	// A modern leaf often has an EMPTY subject — Caddy's does — because the
	// names live in the SANs and nowhere else. Falling back to them keeps the
	// field meaning "which certificate is this", rather than being blank
	// exactly where the answer matters.
	v.Subject, v.Issuer = leaf.Subject.String(), leaf.Issuer.String()
	if strings.TrimSpace(v.Subject) == "" {
		v.Subject = catalog.CertNames(leaf)
	}
	v.NotBefore, v.NotAfter = &nb, &na

	left := time.Until(na)
	if left < 0 {
		v.ExpiresIn = "expired " + catalog.RoundedDuration(left) + " ago"
		v.ExpiryWarning = fmt.Sprintf("the certificate being served expired on %s", na.UTC().Format(time.RFC3339))
		return
	}
	v.ExpiresIn = "in " + catalog.RoundedDuration(left)
	if certSource == catalog.CertFiles && left < tlsExpiryWarnWithin {
		v.ExpiryWarning = fmt.Sprintf("the certificate file expires on %s, %s from now — renew it before then, or this endpoint falls back to Caddy's own authority and every browser that trusted it starts warning",
			na.UTC().Format(time.RFC3339), catalog.RoundedDuration(left))
	}
}

// ---------------------------------------------------------------------
// assertion 1-3: the handshake and the certificate on it
// ---------------------------------------------------------------------

// tlsInspection is one handshake's findings.
type tlsInspection struct {
	Leaf         *x509.Certificate
	Version      uint16
	Cipher       uint16
	HandshakeErr error
	HostnameErr  error
	ChainErr     error
	// NotTLS is set when the port answered but not with TLS — the plaintext
	// case, which is a different failure from "nothing is listening".
	NotTLS bool
}

// inspect completes ONE handshake and records why it would or would not have
// been trusted.
//
// The verification is RELOCATED, not skipped. Go's own verifier collapses
// "wrong name" and "unknown authority" into a single handshake error, and
// those are two different operator problems with two different fixes — so
// VerifyPeerCertificate below does the same x509 work Go would have done, in
// the same order, and records each verdict separately instead of returning at
// the first one. Nothing is trusted as a result: this connection carries no
// request and is closed immediately, and assertions 4 and 5 open a SECOND
// connection with ordinary, unmodified verification. A pass here is a pass
// because leaf.Verify said so.
func (p tlsProbe) inspect(ctx context.Context) tlsInspection {
	var in tlsInspection

	conn, err := (&net.Dialer{Timeout: tlsVerifyDialTimeout}).DialContext(ctx, "tcp", p.Address)
	if err != nil {
		in.HandshakeErr = err
		return in
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(tlsVerifyDialTimeout))

	c := tls.Client(conn, &tls.Config{
		ServerName: p.Hostname,
		MinVersion: tls.VersionTLS12,
		// See the doc comment: this hands verification to the callback below,
		// which performs it in full and attributes the result.
		InsecureSkipVerify: true, //nolint:gosec
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			certs := make([]*x509.Certificate, 0, len(raw))
			for _, der := range raw {
				cert, parseErr := x509.ParseCertificate(der)
				if parseErr != nil {
					in.ChainErr = fmt.Errorf("the server presented a certificate that could not be parsed: %w", parseErr)
					return nil
				}
				certs = append(certs, cert)
			}
			if len(certs) == 0 {
				in.ChainErr = errors.New("the server presented no certificate at all")
				return nil
			}
			in.Leaf = certs[0]

			intermediates := x509.NewCertPool()
			for _, c := range certs[1:] {
				intermediates.AddCert(c)
			}
			if p.RootErr != "" {
				in.ChainErr = errors.New(p.RootErr)
			} else {
				_, in.ChainErr = certs[0].Verify(x509.VerifyOptions{
					Roots:         p.Roots,
					Intermediates: intermediates,
					KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				})
			}
			in.HostnameErr = certs[0].VerifyHostname(p.Hostname)
			return nil
		},
	})

	if err := c.HandshakeContext(ctx); err != nil {
		in.HandshakeErr = err
		var rec tls.RecordHeaderError
		if errors.As(err, &rec) {
			in.NotTLS = true
		}
		return in
	}
	st := c.ConnectionState()
	in.Version, in.Cipher = st.Version, st.CipherSuite
	_ = c.Close()
	return in
}

func (p tlsProbe) handshakeAssertion(in tlsInspection) TLSAssertion {
	a := TLSAssertion{
		ID:    TLSAssertHandshake,
		Title: "A TLS handshake completes on " + p.Address,
		Why:   "proves something on that port speaks TLS at all — a plaintext server there is what a mis-rendered config or a port collision produces, and it fails in a way no browser explains.",
	}
	switch {
	case in.NotTLS:
		a.Status, a.Detail = TLSStatusFail, fmt.Sprintf("%s answered, but not with TLS — it looks like a plaintext server on the HTTPS port (%v)", p.Address, in.HandshakeErr)
	case in.HandshakeErr != nil:
		a.Status, a.Detail = TLSStatusFail, fmt.Sprintf("no TLS handshake with %s: %v", p.Address, in.HandshakeErr)
	default:
		a.Status, a.Detail = TLSStatusPass, fmt.Sprintf("%s negotiated %s with %s", p.Address, tlsVersionName(in.Version), tls.CipherSuiteName(in.Cipher))
	}
	return a
}

func (p tlsProbe) hostnameAssertion(in tlsInspection) TLSAssertion {
	a := TLSAssertion{
		ID:    TLSAssertHostname,
		Title: fmt.Sprintf("The certificate covers %q", p.Hostname),
		Why:   "a valid certificate for the WRONG name is the most common real failure: everything looks fine until a browser refuses the connection outright.",
	}
	switch {
	case in.Leaf == nil:
		a.Status, a.Detail = TLSStatusSkip, "no certificate was presented, so there is nothing to check the name against"
	case in.HostnameErr != nil:
		a.Status, a.Detail = TLSStatusFail, fmt.Sprintf("the certificate served does not cover %q — it covers %s", p.Hostname, catalog.CertNames(in.Leaf))
	default:
		a.Status, a.Detail = TLSStatusPass, fmt.Sprintf("the certificate served covers %s", catalog.CertNames(in.Leaf))
	}
	return a
}

func (p tlsProbe) chainAssertion(in tlsInspection) TLSAssertion {
	a := TLSAssertion{
		ID:    TLSAssertChain,
		Title: "The chain verifies against " + p.trustLabel(),
		Why:   "verification is the only thing that distinguishes a working front from one serving anything at all — a check that passed -k would prove nothing.",
	}
	switch {
	case in.Leaf == nil && in.HandshakeErr != nil:
		a.Status, a.Detail = TLSStatusSkip, "the handshake did not get far enough to present a certificate"
	case in.ChainErr != nil:
		a.Status, a.Detail = TLSStatusFail, fmt.Sprintf("the certificate served does not verify against %s: %v", p.trustLabel(), in.ChainErr)
	default:
		a.Status, a.Detail = TLSStatusPass, fmt.Sprintf("verified against %s (issuer %s)", p.trustLabel(), issuerOf(in.Leaf))
	}
	return a
}

func (p tlsProbe) trustLabel() string {
	if p.TrustSource != "" {
		return p.TrustSource
	}
	return "the system trust store"
}

func issuerOf(leaf *x509.Certificate) string {
	if leaf == nil {
		return "unknown"
	}
	if leaf.Issuer.CommonName != "" {
		return leaf.Issuer.CommonName
	}
	return leaf.Issuer.String()
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case 0:
		return "an unknown TLS version"
	default:
		return fmt.Sprintf("TLS 0x%04x", v)
	}
}

// ---------------------------------------------------------------------
// assertion 4: RPC through it
// ---------------------------------------------------------------------

// verifiedTLSConfig is the ordinary, fully-verifying configuration every
// request-carrying connection below uses. There is no skip flag on this path
// and there must never be one: assertions 4 and 5 are claims about traffic
// that crossed a VERIFIED connection.
func (p tlsProbe) verifiedTLSConfig() *tls.Config {
	return &tls.Config{
		ServerName: p.Hostname,
		RootCAs:    p.Roots,
		MinVersion: tls.VersionTLS12,
	}
}

// pinnedTransport dials p.Address whatever host is in the URL, which is what
// lets the request carry the real hostname (and so the real SNI and the real
// certificate check) without depending on DNS.
func (p tlsProbe) pinnedTransport(useTLS bool) *http.Transport {
	t := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: tlsVerifyDialTimeout}).DialContext(ctx, network, p.Address)
		},
		DisableKeepAlives: true,
	}
	if useTLS {
		t.TLSClientConfig = p.verifiedTLSConfig()
	}
	return t
}

func (p tlsProbe) rpcAssertion(ctx context.Context, trusted bool) TLSAssertion {
	a := TLSAssertion{
		ID:    TLSAssertRPC,
		Title: fmt.Sprintf("eth_chainId over https returns %d", p.ChainID),
		Why:   "terminating TLS and actually reaching eRPC are different things — a front pointed at a dead upstream passes every certificate check there is.",
	}
	if p.ChainID == 0 {
		a.Status, a.Detail = TLSStatusSkip, "this gateway serves no chains, so there is no path to call through"
		return a
	}
	if !trusted {
		a.Status, a.Detail = TLSStatusSkip, "the certificate did not verify, so a request through it would prove nothing about a connection anyone else can make"
		return a
	}

	client := &http.Client{Transport: p.pinnedTransport(true), Timeout: tlsVerifyDialTimeout}
	defer client.CloseIdleConnections()

	got, err := postChainID(ctx, client, p.url("https"))
	switch {
	case err != nil:
		a.Status, a.Detail = TLSStatusFail, fmt.Sprintf("%s: %v", p.url("https"), err)
	case got != p.ChainID:
		a.Status, a.Detail = TLSStatusFail, fmt.Sprintf("%s answered with chain %d, but this path is configured for chain %d — the front is reaching the wrong upstream", p.url("https"), got, p.ChainID)
	default:
		a.Status, a.Detail = TLSStatusPass, fmt.Sprintf("%s answered eth_chainId with %d over a verified connection", p.url("https"), got)
	}
	return a
}

// postChainID performs the one call and reads its answer, giving each way it
// can go wrong its own sentence — "not JSON" is what an empty body from
// something that is not eRPC looks like, and it is worth saying so.
func postChainID(ctx context.Context, client *http.Client, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(gatewayChainIDCall))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return 0, err
	}

	var body struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	text := strings.TrimSpace(string(raw))
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		return 0, fmt.Errorf("HTTP %d, and the answer was not JSON: %q", resp.StatusCode, truncate(text, 200))
	}
	if body.Error != nil {
		return 0, fmt.Errorf("eth_chainId was answered with an error: %s", body.Error.Message)
	}
	return parseHexChainID(body.Result)
}

// ---------------------------------------------------------------------
// assertion 5: subscriptions over wss://
// ---------------------------------------------------------------------

// subscribeAssertion opens a real WebSocket over the same verified TLS and
// asks for newHeads.
//
// The WebSocket client is hand-rolled (a masked text frame and a frame reader,
// nothing more) rather than pulling in a dependency: this app has two direct
// dependencies and adding a third for one probe would be the more expensive
// choice by a distance.
func (p tlsProbe) subscribeAssertion(ctx context.Context, trusted bool) TLSAssertion {
	a := TLSAssertion{
		ID:    TLSAssertSubscribe,
		Title: "eth_subscribe over wss:// delivers",
		Why:   "MEASURED: eRPC infers WebSocket capability from the upstream SCHEME alone, so an http:// upstream serves eth_chainId over wss perfectly and refuses every eth_subscribe. A check that stopped at eth_chainId would call that healthy.",
	}
	if p.ChainID == 0 {
		a.Status, a.Detail = TLSStatusSkip, "this gateway serves no chains, so there is no path to subscribe on"
		return a
	}
	if !trusted {
		a.Status, a.Detail = TLSStatusSkip, "the certificate did not verify, so nothing is proven by talking through it"
		return a
	}

	status, detail := p.subscribe(ctx)
	a.Status, a.Detail = status, detail
	return a
}

func (p tlsProbe) subscribe(ctx context.Context) (status, detail string) {
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: tlsVerifyDialTimeout},
		Config:    p.verifiedTLSConfig(),
	}
	conn, err := dialer.DialContext(ctx, "tcp", p.Address)
	if err != nil {
		return TLSStatusFail, fmt.Sprintf("the wss:// connection to %s could not be opened: %v", p.Address, err)
	}
	defer conn.Close()

	// The handshake, the request and the first answer share one deadline, as
	// they did when this dialled and framed by hand. wsrpc.Handshake puts the
	// context's deadline onto the socket, so the bound is stated once, here,
	// instead of as a SetDeadline sitting apart from the thing it bounds.
	hctx, cancel := context.WithTimeout(ctx, tlsVerifyDialTimeout)
	defer cancel()

	ws, err := wsrpc.Handshake(hctx, conn, p.hostHeader(), p.Path, nil)
	if err != nil {
		return TLSStatusUnavailable, fmt.Sprintf("the WebSocket upgrade on %s was refused: %v", p.url("wss"), err)
	}
	if err := ws.WriteText([]byte(gatewaySubscribeCall)); err != nil {
		return TLSStatusFail, fmt.Sprintf("the subscription request could not be sent: %v", err)
	}

	payload, err := ws.ReadMessage()
	if err != nil {
		return TLSStatusFail, fmt.Sprintf("no answer to eth_subscribe on %s: %v", p.url("wss"), err)
	}
	var reply struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &reply); err != nil {
		return TLSStatusFail, fmt.Sprintf("eth_subscribe was answered with something that is not JSON: %q", truncate(string(payload), 200))
	}
	if reply.Error != nil {
		return TLSStatusUnavailable, fmt.Sprintf(
			"HTTPS works, but eth_subscribe is refused on %s: %s. eRPC decides WebSocket capability from the upstream's SCHEME, so an http:// upstream produces exactly this while every ordinary call keeps working — point the upstream at ws:// and re-create the gateway.",
			p.url("wss"), reply.Error.Message)
	}
	if strings.TrimSpace(reply.Result) == "" {
		return TLSStatusFail, fmt.Sprintf("eth_subscribe returned no subscription id: %q", truncate(string(payload), 200))
	}

	// A subscription id already proves the upstream is WS-capable. Waiting for
	// a head is the stronger claim, so it is made only when one arrives, and
	// its absence is reported as what it is: the chain not producing a block.
	_ = ws.SetDeadline(time.Now().Add(tlsSubscribeWait))
	for {
		msg, err := ws.ReadMessage()
		if err != nil {
			return TLSStatusPass, fmt.Sprintf("eth_subscribe was accepted on %s (subscription %s), but no newHeads arrived within %s — the subscription is live, the chain simply produced no block",
				p.url("wss"), reply.Result, tlsSubscribeWait)
		}
		var note struct {
			Method string `json:"method"`
		}
		if json.Unmarshal(msg, &note) == nil && note.Method == "eth_subscription" {
			return TLSStatusPass, fmt.Sprintf("eth_subscribe on %s delivered a newHeads notification (subscription %s) over a verified connection", p.url("wss"), reply.Result)
		}
	}
}

// hostHeader is the Host: the upgrade request carries. It must be the NAME,
// not the pinned address, or Caddy matches no site block and answers 404.
func (p tlsProbe) hostHeader() string {
	if p.Port == 443 {
		return p.Hostname
	}
	return fmt.Sprintf("%s:%d", p.Hostname, p.Port)
}

// ---------------------------------------------------------------------
// the negative: plaintext must not serve
// ---------------------------------------------------------------------

// plaintextAssertion asserts the cheap negative. It is worth having because
// "HTTPS works" and "only HTTPS works" are different claims, and only the
// second one means an operator can point a browser at this endpoint and know
// no call of theirs is going out in the clear.
//
// A redirect, a 400, or a refused connection all count as not serving. The
// only failure is an RPC ANSWER: that would mean the same port serves
// unencrypted requests.
func (p tlsProbe) plaintextAssertion(ctx context.Context) TLSAssertion {
	a := TLSAssertion{
		ID:    TLSAssertPlaintext,
		Title: fmt.Sprintf("Plain http:// on port %d does NOT serve RPC", p.Port),
		Why:   "\"HTTPS works\" and \"only HTTPS works\" are different claims; a port that answers both is one typo away from sending every call in the clear.",
	}
	if p.ChainID == 0 {
		a.Status, a.Detail = TLSStatusSkip, "this gateway serves no chains, so there is no path to try"
		return a
	}

	client := &http.Client{
		Transport: p.pinnedTransport(false),
		Timeout:   tlsVerifyDialTimeout,
		// A redirect to https is a legitimate "not serving", and it is more
		// useful reported as itself than followed into a success.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	defer client.CloseIdleConnections()

	got, err := postChainID(ctx, client, p.url("http"))
	switch {
	case err != nil:
		a.Status, a.Detail = TLSStatusPass, fmt.Sprintf("%s did not serve RPC: %v", p.url("http"), err)
	default:
		a.Status, a.Detail = TLSStatusFail, fmt.Sprintf("%s answered eth_chainId with %d over plaintext — this port serves unencrypted RPC", p.url("http"), got)
	}
	return a
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
