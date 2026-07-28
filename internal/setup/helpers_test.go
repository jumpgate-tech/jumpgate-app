package setup

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
)

// The pure helpers. Each is small enough that its whole behaviour is a table,
// and each is on a path where getting it wrong produces a plausible-looking
// wrong answer rather than a failure — a mislabelled TLS version, a Host:
// header that makes Caddy answer 404, a df line parsed off the header row.

func TestTLSVersionName(t *testing.T) {
	tests := []struct {
		in   uint16
		want string
	}{
		{tls.VersionTLS13, "TLS 1.3"},
		{tls.VersionTLS12, "TLS 1.2"},
		// Zero is "no handshake happened", which must not render as a version
		// number the operator then goes looking for.
		{0, "an unknown TLS version"},
		{tls.VersionTLS10, "TLS 0x0301"},
	}
	for _, tc := range tests {
		if got := tlsVersionName(tc.in); got != tc.want {
			t.Errorf("tlsVersionName(0x%04x) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIssuerOf(t *testing.T) {
	if got := issuerOf(nil); got != "unknown" {
		t.Errorf("no certificate = %q, want %q", got, "unknown")
	}

	named := &x509.Certificate{Issuer: pkix.Name{CommonName: "Valve Internal CA"}}
	if got := issuerOf(named); got != "Valve Internal CA" {
		t.Errorf("got %q, want the common name", got)
	}

	// A CA with no CN still has to render as something identifying, or the
	// operator is told their certificate was issued by "".
	anon := &x509.Certificate{Issuer: pkix.Name{Organization: []string{"Valve"}}}
	got := issuerOf(anon)
	if got == "" || !strings.Contains(got, "Valve") {
		t.Errorf("got %q, want the full issuer when there is no common name", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("got %q, want it left alone", got)
	}
	got := truncate("0123456789abc", 10)
	if got != "0123456789…" {
		t.Errorf("got %q, want the first 10 runes and an ellipsis", got)
	}
	// The marker matters: a silently-cut body reads as a complete one.
	if !strings.HasSuffix(got, "…") {
		t.Error("truncation left no sign it happened")
	}
}

// The Host: header must carry the NAME, not the pinned address, or Caddy
// matches no site block and answers 404 — which reads as "the gateway is
// down" rather than "we asked for the wrong vhost".
func TestTLSProbeHostHeader(t *testing.T) {
	if got := (tlsProbe{Hostname: "gw.example", Port: 443}).hostHeader(); got != "gw.example" {
		t.Errorf("got %q, want the bare name on the default port", got)
	}
	if got := (tlsProbe{Hostname: "gw.example", Port: 8443}).hostHeader(); got != "gw.example:8443" {
		t.Errorf("got %q, want name:port off the default port", got)
	}
}

// Same rule for the URL: :443 is elided, anything else is explicit.
func TestTLSProbeURL(t *testing.T) {
	p := tlsProbe{Hostname: "gw.example", Port: 443, Path: "/main"}
	if got := p.url("https"); got != "https://gw.example/main" {
		t.Errorf("got %q, want the port elided", got)
	}
	// http on 443 is still written explicitly — it is the plaintext probe,
	// and eliding the port there would aim it at port 80 instead.
	if got := p.url("http"); got != "http://gw.example:443/main" {
		t.Errorf("got %q, want the port kept for the plaintext probe", got)
	}
	q := tlsProbe{Hostname: "gw.example", Port: 8443, Path: "/main"}
	if got := q.url("https"); got != "https://gw.example:8443/main" {
		t.Errorf("got %q, want name:port", got)
	}
}

func TestTLSProbeTrustLabel(t *testing.T) {
	if got := (tlsProbe{}).trustLabel(); got != "the system trust store" {
		t.Errorf("got %q, want the default label", got)
	}
	if got := (tlsProbe{TrustSource: "/etc/caddy-root.crt"}).trustLabel(); got != "/etc/caddy-root.crt" {
		t.Errorf("got %q, want the explicit source", got)
	}
}

func TestTierName(t *testing.T) {
	if got := tierName(true); got != "archive" {
		t.Errorf("got %q, want %q", got, "archive")
	}
	if got := tierName(false); got != "full" {
		t.Errorf("got %q, want %q", got, "full")
	}
}

// parseDFAvail must skip the header, or every free-space check reads zero and
// the preflight refuses a disk that is actually empty.
func TestParseDFAvail(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want uint64
		bad  bool
	}{
		{name: "header then value", in: "Avail\n1234567890\n", want: 1234567890},
		{name: "blank lines and padding", in: "\n  Avail  \n\n   42   \n\n", want: 42},
		{name: "value only", in: "999\n", want: 999},
		{name: "no numeric line", in: "Avail\n(none)\n", bad: true},
		{name: "empty output", in: "", bad: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDFAvail(tc.in)
			if tc.bad {
				if err == nil {
					t.Fatalf("parsed %d out of %q, want an error", got, tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDFAvail(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestJWTPathFor(t *testing.T) {
	explicit := catalog.WireConfig{DataDir: "/data", JWTPath: "/secrets/jwt.hex"}
	if got := jwtPathFor(explicit); got != "/secrets/jwt.hex" {
		t.Errorf("got %q, want the configured path", got)
	}
	derived := catalog.WireConfig{DataDir: "/data"}
	if got := jwtPathFor(derived); got != "/data/jwt.hex" {
		t.Errorf("got %q, want it derived from the data dir", got)
	}
}

// ---------------------------------------------------------------------
// describeCertificate
// ---------------------------------------------------------------------

// parseTestCert turns the PEM testCert produces back into a leaf.
func parseTestCert(t *testing.T, notBefore, notAfter time.Time, names ...string) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(testCert(t, notBefore, notAfter, names...))
	if block == nil {
		t.Fatal("test certificate is not PEM")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse test certificate: %v", err)
	}
	return leaf
}

func TestDescribeCertificate_ReportsWhenItStopsWorking(t *testing.T) {
	now := time.Now()
	var v TLSVerification
	v.describeCertificate(parseTestCert(t, now.Add(-time.Hour), now.Add(720*time.Hour), "gw.example"), catalog.CertInternal)

	if v.NotBefore == nil || v.NotAfter == nil {
		t.Fatal("the validity window was not recorded")
	}
	if !strings.HasPrefix(v.ExpiresIn, "in ") {
		t.Errorf("ExpiresIn = %q, want a remaining duration", v.ExpiresIn)
	}
	// An internal-CA certificate renews itself, so a month out is not a
	// warning — warning on it would train the operator to ignore the field.
	if v.ExpiryWarning != "" {
		t.Errorf("unexpected warning for the internal CA: %q", v.ExpiryWarning)
	}
	if v.Issuer == "" {
		t.Error("no issuer recorded")
	}
}

func TestDescribeCertificate_ExpiredSaysSoAndStops(t *testing.T) {
	now := time.Now()
	var v TLSVerification
	v.describeCertificate(parseTestCert(t, now.Add(-400*24*time.Hour), now.Add(-72*time.Hour), "gw.example"), catalog.CertFiles)

	if !strings.HasPrefix(v.ExpiresIn, "expired ") || !strings.HasSuffix(v.ExpiresIn, " ago") {
		t.Errorf("ExpiresIn = %q, want it to read as already expired", v.ExpiresIn)
	}
	if v.ExpiryWarning == "" {
		t.Fatal("an expired certificate was described without a warning")
	}
}

// A file-sourced certificate nearing expiry IS a warning: nothing renews it,
// and when it lapses the endpoint silently falls back to Caddy's own
// authority and every browser that trusted it starts warning.
func TestDescribeCertificate_WarnsOnAFileNearingExpiry(t *testing.T) {
	now := time.Now()
	leaf := parseTestCert(t, now.Add(-time.Hour), now.Add(tlsExpiryWarnWithin/2), "gw.example")

	var warned TLSVerification
	warned.describeCertificate(leaf, catalog.CertFiles)
	if warned.ExpiryWarning == "" {
		t.Fatal("a cert file expiring inside the warning window produced no warning")
	}

	// The same certificate under the internal CA is not a warning, because
	// that one renews itself. Source is the whole difference.
	var quiet TLSVerification
	quiet.describeCertificate(leaf, catalog.CertInternal)
	if quiet.ExpiryWarning != "" {
		t.Errorf("warned about a self-renewing certificate: %q", quiet.ExpiryWarning)
	}
}

// Caddy's leaf has an EMPTY subject — the names live only in the SANs. The
// field has to keep meaning "which certificate is this" exactly there.
func TestDescribeCertificate_FallsBackToSANsForAnEmptySubject(t *testing.T) {
	now := time.Now()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(7),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(720 * time.Hour),
		DNSNames:     []string{"gw.example"},
		// no Subject at all
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	var v TLSVerification
	v.describeCertificate(leaf, catalog.CertInternal)
	if !strings.Contains(v.Subject, "gw.example") {
		t.Errorf("Subject = %q, want it to fall back to the SANs", v.Subject)
	}
}
