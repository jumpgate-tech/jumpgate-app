package catalog

// Certificate inspection for the CertFiles path.
//
// WHY THIS EXISTS AT ALL, and why the answer is a fallback rather than an
// error: a certificate on disk is the one cert source that DIES ON A TIMER
// nobody here controls.
//
//   - A bundled certificate is public by definition — its private key ships
//     with the software — so it is revocable, and the well-known ones
//     periodically are.
//   - Maximum certificate lifetimes are shrinking on a published schedule:
//     ~200 days now, ~100 by 2027, ~47 by 2029. Any certificate an operator
//     obtained by hand, or that arrived bundled, WILL expire, and sooner every
//     year.
//   - `tailscale cert` output is renewed by Tailscale, but only while the
//     machine is on the tailnet and the operator remembers to re-run it.
//
// So the failure is not hypothetical, it is scheduled. The question is only
// what it looks like when it happens. Serving an expired certificate fails
// CLOSED and silently: browsers refuse the connection outright, wallets show a
// network error, and nothing in the failure names the certificate. Falling
// back to Caddy's internal CA fails OPEN with a visible, recoverable prompt —
// a one-time "this certificate is not trusted" the operator can click through
// or install the root for. A one-time trust prompt is strictly better than a
// dead endpoint, which is why this file exists and why it is automatic.
//
// The check is PURE — bytes in, verdict out — so every trigger below is
// testable without a filesystem, a clock, or a container.

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// certRenewalWindow is how close to expiry a certificate is still accepted.
//
// It is zero on purpose: this check runs at provision time, and refusing a
// certificate that is valid right now would take a working endpoint down
// early. Expiry while the gateway is up is caught on the next provision or
// status read, which is also when the operator is in a position to act. A
// warning window would be a nicer product; it is not the difference between a
// dead endpoint and a trust prompt, which is what this whole path is for.
const certRenewalWindow = 0

// CertProblem is why a certificate on disk cannot be used, in a form a caller
// can both branch on (Reason) and show verbatim (Detail).
type CertProblem struct {
	// Reason is a stable identifier: "missing", "unparseable", "expired",
	// "not-yet-valid", "hostname-mismatch".
	Reason string
	// Detail is the operator-facing sentence, naming the specific fact — the
	// expiry date, the names the certificate actually covers — because "the
	// certificate is not usable" tells nobody what to fix.
	Detail string
}

func (p *CertProblem) Error() string { return p.Detail }

// The Reason values, named so callers and tests do not spell them twice.
const (
	CertProblemMissing     = "missing"
	CertProblemUnparseable = "unparseable"
	CertProblemExpired     = "expired"
	CertProblemNotYetValid = "not-yet-valid"
	CertProblemMismatch    = "hostname-mismatch"
)

// CheckCertificate reports whether certPEM can serve hostname at time now.
//
// It returns a *CertProblem for every reason a certificate is unusable, and
// nil when it is fine. The four triggers, each measured against the real
// x509 fields rather than inferred:
//
//   - EMPTY / unreadable input → missing. The caller passes nil when the file
//     could not be read at all, so "the path is wrong" and "the file is empty"
//     produce one answer.
//   - No CERTIFICATE block, or a block x509 cannot parse → unparseable. A DER
//     file saved with a .pem extension, a key pasted where a cert belongs, a
//     truncated download.
//   - NotAfter before now → expired. This is the scheduled failure this whole
//     file exists for.
//   - NotBefore after now → not-yet-valid. Rare, and almost always a badly
//     wrong system clock, which is worth naming as such.
//   - hostname not covered by the leaf's DNS names / IP addresses →
//     hostname-mismatch. Checked with x509's own VerifyHostname, so wildcards
//     behave exactly as they will in a browser rather than as a substring
//     match here would guess.
//
// Chain validity is deliberately NOT checked. A certificate whose issuer this
// machine does not trust is precisely the localhost.direct / private-CA case
// this feature is meant to support, and rejecting it would refuse the thing
// the operator asked for.
func CheckCertificate(certPEM []byte, hostname string, now time.Time) *CertProblem {
	if len(certPEM) == 0 {
		return &CertProblem{
			Reason: CertProblemMissing,
			Detail: "the certificate file is missing or empty on the target",
		}
	}

	leaf, err := parseLeafCertificate(certPEM)
	if err != nil {
		return &CertProblem{
			Reason: CertProblemUnparseable,
			Detail: fmt.Sprintf("the certificate file could not be parsed as PEM-encoded X.509: %v", err),
		}
	}

	if !leaf.NotAfter.IsZero() && now.Add(certRenewalWindow).After(leaf.NotAfter) {
		return &CertProblem{
			Reason: CertProblemExpired,
			Detail: fmt.Sprintf("the certificate expired on %s (%s ago)",
				leaf.NotAfter.UTC().Format(time.RFC3339), roundedSince(now.Sub(leaf.NotAfter))),
		}
	}
	if !leaf.NotBefore.IsZero() && now.Before(leaf.NotBefore) {
		return &CertProblem{
			Reason: CertProblemNotYetValid,
			Detail: fmt.Sprintf("the certificate is not valid until %s — check this machine's clock",
				leaf.NotBefore.UTC().Format(time.RFC3339)),
		}
	}

	host := strings.TrimSpace(hostname)
	if host == "" {
		return &CertProblem{
			Reason: CertProblemMismatch,
			Detail: "no hostname is configured, so there is nothing the certificate can be checked against",
		}
	}
	if err := leaf.VerifyHostname(host); err != nil {
		return &CertProblem{
			Reason: CertProblemMismatch,
			Detail: fmt.Sprintf("the certificate does not cover %q (it covers %s)", host, certNames(leaf)),
		}
	}
	return nil
}

// parseLeafCertificate returns the FIRST certificate in a PEM bundle, which is
// the leaf by convention and by every tool that writes one. Non-CERTIFICATE
// blocks are skipped rather than rejected: a combined file with the key in it
// is a common enough mistake that reading past it beats failing on it.
func parseLeafCertificate(certPEM []byte) (*x509.Certificate, error) {
	rest := certPEM
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, fmt.Errorf("no CERTIFICATE block found")
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		return x509.ParseCertificate(block.Bytes)
	}
}

// certNames lists what a certificate actually covers, so a mismatch says which
// name to use instead of only which one failed.
func certNames(c *x509.Certificate) string {
	var names []string
	names = append(names, c.DNSNames...)
	for _, ip := range c.IPAddresses {
		names = append(names, ip.String())
	}
	if len(names) == 0 {
		if c.Subject.CommonName != "" {
			return "only the legacy common name " + c.Subject.CommonName + ", with no subject alternative names — no modern client will accept it"
		}
		return "no names at all"
	}
	return strings.Join(names, ", ")
}

// roundedSince renders a duration at a granularity worth reading. An expiry is
// interesting in days, or in hours when it just happened; nobody needs the
// nanoseconds Go prints by default.
func roundedSince(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	if d >= 48*time.Hour {
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
	return d.Round(time.Hour).String()
}
