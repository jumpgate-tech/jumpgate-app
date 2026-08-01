package setup

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testChain mints a throwaway CA and a leaf for host, valid over the given
// window, and returns the PEM fullchain (leaf + CA), the leaf key PEM, and a
// root pool containing the CA — so a test can exercise verifyValveCert's ACCEPT
// path without a genuinely public certificate.
func testChain(t *testing.T, host string, notBefore, notAfter time.Time) (fullchainPEM, keyPEM []byte, roots *x509.CertPool) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ca key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "valvecert test CA"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("ca cert: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca: %v", err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("leaf key: %v", err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: host},
		DNSNames:     []string{host},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("leaf cert: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		t.Fatalf("marshal leaf key: %v", err)
	}
	fullchainPEM = append(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})...,
	)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	roots = x509.NewCertPool()
	roots.AddCert(caCert)
	return fullchainPEM, keyPEM, roots
}

const testHost = "default-abc123.localhost-valaxy.com"

func TestVerifyValveCert_AcceptsCertFromTrustedRoot(t *testing.T) {
	now := time.Now()
	chain, _, roots := testChain(t, testHost, now.Add(-time.Hour), now.Add(24*time.Hour))
	if err := verifyValveCert(chain, testHost, roots); err != nil {
		t.Fatalf("a currently-valid cert that chains to the trusted root must verify: %v", err)
	}
}

func TestVerifyValveCert_RejectsExpired(t *testing.T) {
	now := time.Now()
	chain, _, roots := testChain(t, testHost, now.Add(-48*time.Hour), now.Add(-time.Hour))
	if err := verifyValveCert(chain, testHost, roots); err == nil {
		t.Fatal("an expired cert must be rejected so the front falls back to the internal CA")
	}
}

func TestVerifyValveCert_RejectsWrongHostname(t *testing.T) {
	now := time.Now()
	chain, _, roots := testChain(t, testHost, now.Add(-time.Hour), now.Add(24*time.Hour))
	if err := verifyValveCert(chain, "someone-else.example.com", roots); err == nil {
		t.Fatal("a cert that does not cover the hostname must be rejected")
	}
}

// The production safety property: with the real system trust store (roots nil),
// a self-signed / privately-minted cert does NOT verify — so the app never
// serves something a browser would still reject; it falls back to the internal
// CA + one-click trust instead.
func TestVerifyValveCert_RejectsUntrustedAgainstSystemRoots(t *testing.T) {
	now := time.Now()
	chain, _, _ := testChain(t, testHost, now.Add(-time.Hour), now.Add(24*time.Hour))
	if err := verifyValveCert(chain, testHost, nil); err == nil {
		t.Fatal("a cert not chaining to a SYSTEM root must be rejected (nil roots = host trust store)")
	}
}

func TestParseCertChain_SplitsLeafAndIntermediates(t *testing.T) {
	now := time.Now()
	chain, _, _ := testChain(t, testHost, now.Add(-time.Hour), now.Add(24*time.Hour))
	leaf, intermediates, err := parseCertChain(chain)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if leaf == nil || leaf.Subject.CommonName != testHost {
		t.Fatalf("first CERTIFICATE block must be the leaf, got %+v", leaf)
	}
	if len(intermediates) != 1 {
		t.Fatalf("the CA that follows the leaf must land in intermediates, got %d", len(intermediates))
	}
}

func TestParseCertChain_ErrorsWhenNoCertificate(t *testing.T) {
	if _, _, err := parseCertChain([]byte("not a pem")); err == nil {
		t.Fatal("a bundle with no CERTIFICATE block must error, not return a nil leaf")
	}
}

// fetchValveCert reaches its verify step against the SYSTEM store, so a test
// cert legitimately fails there — these cases pin the pre-verify failures
// (transport, HTTP status, malformed bundle) that must each fall back cleanly.
func TestFetchValveCert_FailsCleanlyOnBadResponses(t *testing.T) {
	now := time.Now()
	chain, key, _ := testChain(t, testHost, now.Add(-time.Hour), now.Add(24*time.Hour))
	good, _ := json.Marshal(valveCertBundle{Cert: string(chain), Key: string(key)})

	cases := []struct {
		name    string
		status  int
		body    string
		wantSub string
	}{
		{"http error", http.StatusInternalServerError, "boom", "HTTP 500"},
		{"not json", http.StatusOK, "<html>nope", "decode bundle"},
		{"missing fields", http.StatusOK, `{"cert":"","key":""}`, "missing cert or key"},
		// A well-formed bundle still fails at verification against the real
		// system roots, which is the correct production outcome for a cert no
		// public CA signed.
		{"untrusted cert", http.StatusOK, string(good), "does not verify"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			t.Setenv("VALVE_CERT_URL", srv.URL)
			_, _, err := fetchValveCert(context.Background(), testHost)
			if err == nil {
				t.Fatalf("%s: expected an error so the caller falls back to the internal CA", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("%s: error %q must mention %q", tc.name, err, tc.wantSub)
			}
		})
	}
}
