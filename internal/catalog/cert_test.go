package catalog

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

// issueCert mints a self-signed leaf for exactly the names and validity window
// a test needs. Real certificates rather than fixture strings, because the
// whole point of this check is that it reads the same x509 fields a browser
// does — a hand-written PEM constant would only prove the parser accepts what
// the test author expected.
func issueCert(t *testing.T, notBefore, notAfter time.Time, dnsNames []string, ips []net.IP) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "valve-node-app test"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// Every trigger, one test each. These are not hypothetical failure modes:
// maximum certificate lifetimes are shrinking on a published schedule, so a
// bundled or hand-obtained certificate WILL hit the expired case, and the
// difference between falling back and refusing is the difference between a
// one-time trust prompt and a dead endpoint.
func TestCheckCertificate_Triggers(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	good := issueCert(t, now.Add(-24*time.Hour), now.Add(90*24*time.Hour), []string{"gw.example"}, nil)

	tests := []struct {
		name       string
		pem        []byte
		host       string
		wantReason string
		wantDetail string
	}{
		{
			name: "a usable certificate is not a problem",
			pem:  good, host: "gw.example",
		},
		{
			name: "missing file", pem: nil, host: "gw.example",
			wantReason: CertProblemMissing,
		},
		{
			// A DER file with a .pem extension, a key pasted where a cert
			// belongs, a truncated download.
			name: "unparseable", pem: []byte("this is not a certificate"), host: "gw.example",
			wantReason: CertProblemUnparseable,
		},
		{
			name:       "expired — the scheduled failure this whole path exists for",
			pem:        issueCert(t, now.Add(-400*24*time.Hour), now.Add(-11*24*time.Hour), []string{"gw.example"}, nil),
			host:       "gw.example",
			wantReason: CertProblemExpired,
			wantDetail: "11 days",
		},
		{
			name:       "not yet valid — almost always a wrong clock, so say so",
			pem:        issueCert(t, now.Add(48*time.Hour), now.Add(90*24*time.Hour), []string{"gw.example"}, nil),
			host:       "gw.example",
			wantReason: CertProblemNotYetValid,
			wantDetail: "clock",
		},
		{
			name:       "hostname not covered, and the message names what IS covered",
			pem:        good,
			host:       "other.example",
			wantReason: CertProblemMismatch,
			wantDetail: "gw.example",
		},
		{
			name:       "no hostname configured is a mismatch, not a pass",
			pem:        good,
			host:       "  ",
			wantReason: CertProblemMismatch,
		},
		{
			// x509's own VerifyHostname, so a wildcard behaves exactly as it
			// will in a browser rather than as a substring match would guess.
			name: "a wildcard covers one label",
			pem:  issueCert(t, now.Add(-time.Hour), now.Add(time.Hour), []string{"*.example"}, nil),
			host: "gw.example",
		},
		{
			name:       "a wildcard does not cover two labels",
			pem:        issueCert(t, now.Add(-time.Hour), now.Add(time.Hour), []string{"*.example"}, nil),
			host:       "a.gw.example",
			wantReason: CertProblemMismatch,
		},
		{
			name: "an IP SAN covers a bare address",
			pem:  issueCert(t, now.Add(-time.Hour), now.Add(time.Hour), nil, []net.IP{net.ParseIP("100.64.1.2")}),
			host: "100.64.1.2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckCertificate(tc.pem, tc.host, now)
			if tc.wantReason == "" {
				if got != nil {
					t.Fatalf("want usable, got %s: %s", got.Reason, got.Detail)
				}
				return
			}
			if got == nil {
				t.Fatalf("want %s, got a usable verdict", tc.wantReason)
			}
			if got.Reason != tc.wantReason {
				t.Fatalf("reason = %q, want %q (detail: %s)", got.Reason, tc.wantReason, got.Detail)
			}
			if tc.wantDetail != "" && !strings.Contains(got.Detail, tc.wantDetail) {
				t.Errorf("detail %q must mention %q", got.Detail, tc.wantDetail)
			}
		})
	}
}

// A certificate whose issuer this machine does not trust is exactly the
// localhost.direct / private-CA case the CertFiles source exists to support.
// Rejecting it would refuse the thing the operator asked for, so chain
// validity is deliberately not part of this check.
func TestCheckCertificate_DoesNotRequireATrustedIssuer(t *testing.T) {
	now := time.Now()
	self := issueCert(t, now.Add(-time.Hour), now.Add(time.Hour), []string{"gw.example"}, nil)
	if p := CheckCertificate(self, "gw.example", now); p != nil {
		t.Fatalf("a self-signed certificate must be usable: %s", p.Detail)
	}
}

// A combined cert+key file is a common enough mistake that reading past the
// key block beats failing on it.
func TestCheckCertificate_SkipsNonCertificateBlocks(t *testing.T) {
	now := time.Now()
	bundle := append(
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte("not really a key")}),
		issueCert(t, now.Add(-time.Hour), now.Add(time.Hour), []string{"gw.example"}, nil)...,
	)
	if p := CheckCertificate(bundle, "gw.example", now); p != nil {
		t.Fatalf("want the certificate found past the key block, got %s: %s", p.Reason, p.Detail)
	}
}
