package catalog

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

// parseLeaf mints a certificate and hands back the parsed x509, which is what
// CertNames takes. It exists alongside issueCert because CertNames is reached
// from the live verification with a certificate pulled off the wire, never
// from a PEM file.
func parseLeaf(t *testing.T, tmpl *x509.Certificate) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl.SerialNumber = big.NewInt(1)
	tmpl.NotBefore = time.Now().Add(-time.Hour)
	tmpl.NotAfter = time.Now().Add(24 * time.Hour)
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return leaf
}

// A mismatch message has to say which names the certificate DOES cover, or the
// operator is told their certificate is wrong without being told what it is
// for — and both SAN kinds count, because pinning a gateway to an IP is a
// normal thing to do on a private network.
func TestCertNames_ListsBothDNSNamesAndIPs(t *testing.T) {
	leaf := parseLeaf(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "ignored"},
		DNSNames:    []string{"gw.valve.city", "alt.valve.city"},
		IPAddresses: []net.IP{net.ParseIP("100.64.0.7")},
	})

	got := CertNames(leaf)
	for _, want := range []string{"gw.valve.city", "alt.valve.city", "100.64.0.7"} {
		if !strings.Contains(got, want) {
			t.Errorf("CertNames = %q, missing %q", got, want)
		}
	}
	// The common name is deliberately NOT listed when there are SANs: no
	// modern client looks at it, so showing it invites the operator to
	// "fix" a name nothing reads.
	if strings.Contains(got, "ignored") {
		t.Errorf("CertNames = %q — the common name must not be offered as a usable name when SANs exist", got)
	}
}

// A certificate with a common name and no SANs is the specific trap this
// message exists for: openssl's defaults still produce one, it looks correct
// in every text dump, and Go's own TLS stack has refused to look at the common
// name since 1.15. The operator needs to be told THAT, not "hostname mismatch".
func TestCertNames_CallsOutTheLegacyCommonNameTrap(t *testing.T) {
	leaf := parseLeaf(t, &x509.Certificate{Subject: pkix.Name{CommonName: "gw.valve.city"}})

	got := CertNames(leaf)
	if !strings.Contains(got, "gw.valve.city") {
		t.Errorf("CertNames = %q — must still name what the certificate claims", got)
	}
	if !strings.Contains(got, "common name") {
		t.Errorf("CertNames = %q — must say the name is the legacy common name, which is the whole diagnosis", got)
	}
	if !strings.Contains(got, "no modern client") {
		t.Errorf("CertNames = %q — must say why it will not work, or it reads as a working certificate", got)
	}
}

func TestCertNames_SaysSoWhenThereAreNoNamesAtAll(t *testing.T) {
	leaf := parseLeaf(t, &x509.Certificate{})
	if got := CertNames(leaf); got != "no names at all" {
		t.Errorf("CertNames = %q, want %q", got, "no names at all")
	}
}
