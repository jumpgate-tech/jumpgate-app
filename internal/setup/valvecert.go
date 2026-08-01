package setup

// The gateway's HTTPS front serves one of two kinds of certificate. Caddy's own
// internal CA (catalog.CertInternal) is self-signed, so a browser or wallet
// warns until its root is trusted — the one-click "trust on this machine" flow
// exists for exactly that case, and remains the fallback.
//
// The better default, when it can be reached, is a GENUINELY publicly-trusted
// certificate for the loopback domain *.localhost-valaxy.com. The name resolves
// to 127.0.0.1, but a public CA (Let's Encrypt via DNS-01) will still issue for
// it: DNS validation proves control of the zone, not of an address. Valve mints
// that wildcard centrally — the zone credentials never leave valve's own
// infrastructure — and serves the current cert+key from an endpoint every
// install can fetch, so no user ever installs anything and every browser trusts
// it out of the box.
//
// The private key is, by construction, shared across installs and therefore
// effectively public. That is an acceptable bargain ONLY because the name it
// certifies resolves to loopback: to abuse the key an attacker must already be
// able to run code on — or MITM the loopback of — the victim's own machine, at
// which point a leaked cert key is the least of anyone's problems.
//
// This file only FETCHES and VALIDATES the bundle. Whether to use it, and the
// fall back to the internal CA when it cannot be had, is decided by the TLS
// setup step (which writes these PEMs to the target and sets CertSource=files).

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// defaultValveCertURL is where the current *.localhost-valaxy.com bundle is
// served. Overridable with VALVE_CERT_URL for a self-hosted mirror or tests.
const defaultValveCertURL = "https://one.valve.city/tls/localhost-valaxy.json"

// valveCertMaxBytes bounds the response read — a PEM cert+key pair is a few KiB;
// 1 MiB is a generous ceiling that still refuses a misconfigured endpoint
// streaming something enormous.
const valveCertMaxBytes = 1 << 20

// valveCertBundle is the JSON the endpoint returns: a PEM fullchain and its
// matching PEM private key.
type valveCertBundle struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

// valveCertURL resolves the bundle endpoint; the env override wins so a test or
// an air-gapped mirror can point it elsewhere without a rebuild.
func valveCertURL() string {
	if u := strings.TrimSpace(os.Getenv("VALVE_CERT_URL")); u != "" {
		return u
	}
	return defaultValveCertURL
}

// fetchValveCert fetches the publicly-trusted wildcard bundle and returns its
// cert and key PEMs — but ONLY if the certificate genuinely verifies against
// the system roots for hostname and is currently within its validity window.
//
// A bundle that does not verify is returned as an error, not a cert: switching
// the front to "files" is only ever an improvement if the file is actually
// trusted, so the caller must fall back to the internal CA rather than serve
// something a browser would reject anyway. The check here is the same one a
// browser runs, so "verifies here" means "no warning there".
func fetchValveCert(ctx context.Context, hostname string) (certPEM, keyPEM []byte, err error) {
	url := valveCertURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("valve-cert: build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("valve-cert: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("valve-cert: %s returned HTTP %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, valveCertMaxBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("valve-cert: read body: %w", err)
	}
	var b valveCertBundle
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, nil, fmt.Errorf("valve-cert: decode bundle from %s: %w", url, err)
	}
	cert := []byte(strings.TrimSpace(b.Cert))
	key := []byte(strings.TrimSpace(b.Key))
	if len(cert) == 0 || len(key) == 0 {
		return nil, nil, errors.New("valve-cert: bundle is missing cert or key")
	}
	// nil roots → the host's own trust store (production). Tests pass a pool.
	if err := verifyValveCert(cert, hostname, nil); err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// verifyValveCert parses the fullchain and checks the leaf is currently valid
// and chains to a trusted root for hostname. roots nil means the host's own
// trust store — the same authorities a browser trusts — so a pass there is a
// promise the front will not warn; a test passes its own pool to exercise the
// accept path without a real public cert.
func verifyValveCert(fullchainPEM []byte, hostname string, roots *x509.CertPool) error {
	leaf, intermediates, err := parseCertChain(fullchainPEM)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	for _, ic := range intermediates {
		pool.AddCert(ic)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{
		DNSName:       hostname,
		Intermediates: pool,
		Roots:         roots, // nil → host system trust store (what a browser uses)
		// CurrentTime zero → time.Now(), so an expired cert fails here.
	}); err != nil {
		return fmt.Errorf("valve-cert: fetched certificate does not verify for %s, falling back to the internal CA: %w", hostname, err)
	}
	return nil
}

// parseCertChain splits a PEM fullchain into its leaf (the first CERTIFICATE
// block) and any intermediates that follow. Non-CERTIFICATE blocks are skipped
// so a bundle that accidentally carries the key in the same file is tolerated.
func parseCertChain(pemBytes []byte) (leaf *x509.Certificate, intermediates []*x509.Certificate, err error) {
	rest := pemBytes
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, perr := x509.ParseCertificate(block.Bytes)
		if perr != nil {
			return nil, nil, fmt.Errorf("valve-cert: parse certificate: %w", perr)
		}
		if leaf == nil {
			leaf = c
		} else {
			intermediates = append(intermediates, c)
		}
	}
	if leaf == nil {
		return nil, nil, errors.New("valve-cert: no certificate found in bundle")
	}
	return leaf, intermediates, nil
}
