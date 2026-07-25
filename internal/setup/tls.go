package setup

// The HTTPS front for a gateway: resolving which certificate will actually be
// used, rendering the Caddyfile, and getting Caddy's own root certificate off
// the container so the operator (and this app's readiness probe) can verify the
// chain properly instead of skipping verification.
//
// WHY the certificate decision lives HERE rather than in catalog: it depends on
// bytes that exist only ON THE TARGET, which may be an SSH host. catalog owns
// the pure verdict (CheckCertificate); this file owns reading the file and
// acting on the answer.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// caddyRootCertPath is where Caddy keeps the root of the CA it issues from
// when `tls internal` is in effect, inside the container.
//
// It is a path into catalog.CaddyDataPath, which is why that volume is not
// optional: recreate the container without it and this file is a DIFFERENT
// root, so every device that installed the old one starts rejecting the
// gateway. Measured — see catalog.CaddyDataVolume.
const caddyRootCertPath = catalog.CaddyDataPath + "/caddy/pki/authorities/local/root.crt"

// tlsFront is the RESOLVED front for one gateway: what will actually be
// rendered and run, after the certificate on disk has been looked at.
type tlsFront struct {
	// Caddy is the effective configuration. Its CertSource may differ from the
	// one the operator stored — that is the whole point of Fallback.
	Caddy catalog.CaddyConfig

	// Fallback is why the stored cert source was not used, and is empty when it
	// was. It is a sentence written for an operator, because the situation it
	// describes ("your certificate expired eleven days ago, so this is now
	// serving a self-signed one and your browser will say so") is not
	// self-explanatory from the outside.
	Fallback string

	// FallbackReason is the stable catalog.CertProblem* identifier behind
	// Fallback, so a UI can branch without matching on prose.
	FallbackReason string
}

// resolveTLSFront decides what the TLS front will actually serve.
//
// The AUTO-FALLBACK, and why it is automatic rather than an error: a
// certificate on disk is the one source that expires on a schedule nobody here
// controls (catalog/cert.go sets out the full argument). Refusing to start
// would turn a scheduled, inevitable event into a dead endpoint. Falling back
// to Caddy's internal CA turns it into a one-time browser trust prompt, which
// is recoverable by the person looking at it. What must NOT happen is falling
// back quietly, so the reason travels with the decision all the way to the API.
//
// The fallback is triggered by every condition CheckCertificate reports:
// missing, unparseable, expired, not yet valid, or not covering the configured
// hostname. The key file is checked for presence only — a key that does not
// match its certificate is a Caddy startup failure with a clear message of its
// own, and re-deriving that check here would only add a way to disagree with it.
func resolveTLSFront(ctx context.Context, e executor.Executor, tls *catalog.GatewayTLS, upstreamHost string, upstreamPort int) (*tlsFront, error) {
	if !tls.On() {
		return nil, nil
	}
	if err := tls.ValidateSettings(); err != nil {
		return nil, err
	}

	front := &tlsFront{Caddy: tls.Caddy(upstreamHost, upstreamPort)}
	if front.Caddy.CertSourceOrDefault() != catalog.CertFiles {
		return front, nil
	}

	problem := inspectCertFiles(ctx, e, tls)
	if problem == nil {
		return front, nil
	}

	// Fall back, and say so in full. The stored settings are NOT rewritten:
	// the operator asked for those files, the files may well be renewed
	// tomorrow, and silently editing their configuration to match a transient
	// failure would mean the fallback never lifted.
	front.Caddy.CertSource = catalog.CertInternal
	front.Caddy.CertFile, front.Caddy.KeyFile = "", ""
	front.FallbackReason = problem.Reason
	front.Fallback = fmt.Sprintf(
		"The certificate at %s could not be used: %s. HTTPS is still on, but it is now served with Caddy's own "+
			"certificate authority instead — browsers will warn once until you install that authority's root, and "+
			"the alternative was no HTTPS endpoint at all. Fix or renew the file and re-create the gateway to go back to it.",
		tls.CertFile, problem.Detail)
	return front, nil
}

// inspectCertFiles reads the certificate and key off the target and returns
// the first reason they cannot be used, or nil.
//
// A read failure is reported as "missing" rather than propagated: from the
// operator's point of view a path that cannot be read and a path that is not
// there are the same problem with the same fix, and the distinction is not
// worth a second failure mode that behaves differently.
func inspectCertFiles(ctx context.Context, e executor.Executor, tls *catalog.GatewayTLS) *catalog.CertProblem {
	certPEM, err := e.ReadFile(ctx, tls.CertFile)
	if err != nil {
		return &catalog.CertProblem{
			Reason: catalog.CertProblemMissing,
			Detail: fmt.Sprintf("the certificate file %s could not be read on the target (%v)", tls.CertFile, err),
		}
	}
	if _, err := e.ReadFile(ctx, tls.KeyFile); err != nil {
		return &catalog.CertProblem{
			Reason: catalog.CertProblemMissing,
			Detail: fmt.Sprintf("the private key %s could not be read on the target (%v)", tls.KeyFile, err),
		}
	}
	return catalog.CheckCertificate(certPEM, tls.Hostname, time.Now())
}

// TLSState is the read-only answer to "what certificate is this gateway
// actually serving, and why is it not the configured one".
type TLSState struct {
	// CertSource is the EFFECTIVE source, after any fallback.
	CertSource string
	// Fallback and FallbackReason are empty when the configured source is in
	// use; see tlsFront.
	Fallback       string
	FallbackReason string
}

// GatewayTLSState resolves a gateway's TLS front for DISPLAY, and returns the
// path the internal CA's root is (or would be) exported to on the target.
//
// It is the same resolution the provisioner performs, deliberately re-run on
// every read rather than cached: a certificate's validity is a function of the
// wall clock, so a gateway provisioned successfully in March can be serving a
// fallback certificate in October with nothing having been changed. Reading
// stored state would report the March answer forever.
//
// Nothing is written and no container is touched.
func GatewayTLSState(ctx context.Context, e executor.Executor, gatewayID string, g catalog.GatewayConfig) (TLSState, string, error) {
	p := &gatewayPlan{id: gatewayID, gw: g, backend: BackendDocker}
	front, err := p.front(ctx, e)
	if err != nil {
		return TLSState{}, "", err
	}
	if front == nil {
		return TLSState{}, "", nil
	}
	path, err := p.rootCAPath(ctx, e)
	if err != nil {
		return TLSState{}, "", err
	}
	return TLSState{
		CertSource:     front.Caddy.CertSourceOrDefault(),
		Fallback:       front.Fallback,
		FallbackReason: front.FallbackReason,
	}, path, nil
}

// ---------------------------------------------------------------------
// the internal CA's root
// ---------------------------------------------------------------------

// exportRootCA copies Caddy's internal root certificate out of the container
// to hostPath, and reports whether it got one.
//
// TWO reasons this is worth doing rather than telling the operator to run
// `docker cp` themselves:
//
//  1. It is the file they have to install in their machine's trust store, and
//     "the thing you need is inside a container, at this path" is a needlessly
//     hostile way to end a setup flow.
//  2. It is what lets this app's OWN readiness probe verify the chain, rather
//     than passing curl -k. A probe that skips verification cannot tell a
//     working TLS front from one serving a certificate for the wrong name, and
//     the wrong name is exactly the failure this feature introduces.
//
// It is best-effort by design. Caddy writes the root at startup, so a probe
// milliseconds after `docker run` can legitimately find nothing yet; the caller
// retries, and a gateway whose root could not be exported still works, it just
// cannot be verified by us.
func exportRootCA(ctx context.Context, e executor.Executor, containerName, hostPath string) (bool, error) {
	res, err := ops.DockerRun(ctx, e, "exec", containerName, "cat", caddyRootCertPath)
	if err != nil {
		return false, err
	}
	if res.ExitCode != 0 || !strings.Contains(res.Stdout, "BEGIN CERTIFICATE") {
		return false, nil
	}
	if err := e.WriteFile(ctx, hostPath, []byte(res.Stdout), 0644); err != nil {
		return false, fmt.Errorf("tls: write the internal CA root to %s: %w", hostPath, err)
	}
	return true, nil
}
