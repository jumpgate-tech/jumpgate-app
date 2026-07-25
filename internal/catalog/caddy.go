package catalog

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// TLS for the gateway, terminated by Caddy in front of eRPC.
//
// WHY a second container rather than eRPC's own TLS: eRPC does terminate TLS
// (server.tls with certFile/keyFile), but it is bring-your-own-cert with no
// ACME client, and Go's ListenAndServeTLS reads the certificate ONCE at
// startup. Caddy obtains certificates itself and reloads them in process. That
// difference is not academic — Caddy's internal CA issues 12-hour certificates,
// which under eRPC would mean restarting the gateway twice a day.
//
// WHY this matters at all: an https:// page cannot call an http:// RPC
// endpoint. Chrome and Firefox exempt http://localhost; Safari does not, and
// EVERY browser blocks a non-loopback origin. So a gateway on a LAN or
// Tailscale address is unusable from a browser dApp without this.

// CertSource is how a gateway's certificate is obtained.
const (
	// CertInternal uses Caddy's own CA. No domain, no network, no service to
	// run — at the cost of one trust-store install per machine. The only
	// option that works offline, and the sensible default.
	CertInternal = "internal"

	// CertFiles uses a certificate already on disk. This is the `tailscale
	// cert` path: Tailscale issues a genuine publicly-trusted certificate for
	// a <host>.<tailnet>.ts.net name and owns its renewal, so there is
	// nothing to install and nothing for us to run.
	CertFiles = "files"

	// CertACME obtains a publicly-trusted certificate directly. Deliberately
	// NOT implemented yet: for a name that resolves to loopback the only
	// usable challenge is DNS-01, which needs credentials for the zone, and
	// handing zone-write credentials to every install is worse than the
	// trust-store click it would save. Left as a named constant so the
	// decision is visible rather than forgotten.
	CertACME = "acme"
)

// DefaultCaddyImage is pinned for the same reason every other image ref here
// is: an operator's TLS front should not change version under them on a
// restart.
const DefaultCaddyImage = "caddy:2-alpine"

// CaddyDataVolume is the named volume mounted at CaddyDataPath, and it is NOT
// optional.
//
// Caddy keeps its internal CA under /data. Without a persistent volume that CA
// is regenerated every time the container is recreated — measured: recreating
// the container produced a root with a different SHA-256 fingerprint, and
// every previously-issued certificate stopped verifying.
//
// The consequence is worse than it first sounds. With CertInternal the
// operator installs that root into their machine's trust store ONCE. If the CA
// changes underneath them, every browser starts rejecting the gateway and the
// only fix is finding and installing the new root again — so an ordinary
// "recreate the TLS front" would silently break HTTPS for every device that
// trusted it.
//
// Two rules follow. This volume must always be declared, and a wipe of the
// TLS front must NOT remove it by default: wiping chain data is routine,
// whereas invalidating a trust-store install is not.
const (
	CaddyDataVolume = "valve-node-app-caddy-data"
	CaddyDataPath   = "/data"
)

// CaddyConfig describes the TLS front for one gateway.
type CaddyConfig struct {
	// Hostname is the name served, and the name the certificate is for. It
	// must resolve to the machine the browser is on — a loopback name like
	// localhost.example.com pointing at 127.0.0.1, a LAN address, or a
	// tailnet name.
	Hostname string

	// CertSource is one of the Cert* constants ("" → CertInternal).
	CertSource string

	// CertFile and KeyFile are used only when CertSource is CertFiles. They
	// are paths ON THE TARGET, mounted into the container.
	CertFile string
	KeyFile  string

	// UpstreamHost and UpstreamPort address the gateway being fronted. On a
	// shared docker network this is the eRPC container's NAME, which is the
	// point of the private network: the gateway then needs no published port
	// at all, and host.docker.internal stops being involved.
	UpstreamHost string
	UpstreamPort int

	// HTTPSPort is the host port Caddy publishes (0 → 443).
	HTTPSPort int

	// ImageRef overrides the Caddy image ("" → caddy:2-alpine).
	ImageRef string
}

// GatewayTLS is the OPERATOR-FACING half of a gateway's TLS front: what is
// stored, edited and round-tripped through the API. CaddyConfig is the
// rendering half, and the difference matters — CaddyConfig names an upstream
// container and port, which are derived at provision time and are not the
// operator's to set.
type GatewayTLS struct {
	// Enabled is the switch. A GatewayTLS with Enabled false is kept rather
	// than deleted so that turning TLS off and on again does not lose the
	// hostname and cert paths the operator typed.
	Enabled bool

	// Hostname is the name served and the name the certificate must cover.
	Hostname string

	// CertSource is one of the Cert* constants ("" → CertInternal).
	CertSource string

	// CertFile and KeyFile are paths ON THE TARGET, used only with CertFiles.
	CertFile string
	KeyFile  string

	// HTTPSPort is the host port the front publishes (0 → 443).
	HTTPSPort int

	// BindAddr is the HOST address that port binds to ("" → 0.0.0.0).
	//
	// It defaults WIDE, which is the opposite of every other bind in this app,
	// and deliberately so: the entire reason this feature exists is that a
	// browser on another device cannot call an http:// endpoint from an
	// https:// page. A TLS front bound to loopback serves only the machine it
	// runs on, which is the one machine that never needed TLS in the first
	// place.
	BindAddr string

	// ImageRef overrides the Caddy image ("" → DefaultCaddyImage).
	ImageRef string
}

// On reports whether this gateway is actually fronted. A nil pointer and a
// disabled struct are the same answer, so callers never have to check both.
func (t *GatewayTLS) On() bool { return t != nil && t.Enabled }

// CertSourceOrDefault resolves the cert source ("" → internal).
func (t *GatewayTLS) CertSourceOrDefault() string {
	if t == nil || t.CertSource == "" {
		return CertInternal
	}
	return t.CertSource
}

// HTTPS resolves the published TLS port (0 → 443).
func (t *GatewayTLS) HTTPS() int {
	if t == nil || t.HTTPSPort == 0 {
		return 443
	}
	return t.HTTPSPort
}

// Bind resolves the host bind address ("" → 0.0.0.0); see BindAddr.
func (t *GatewayTLS) Bind() string {
	if t == nil || strings.TrimSpace(t.BindAddr) == "" {
		return "0.0.0.0"
	}
	return t.BindAddr
}

// URL is the https:// base a caller dials. Empty when TLS is off, so a caller
// can use it as both the value and the test.
func (t *GatewayTLS) URL() string {
	if !t.On() {
		return ""
	}
	return t.Caddy("", 0).URL()
}

// Caddy projects the stored settings onto a renderable CaddyConfig, filling in
// the upstream the caller resolved. certSource is passed separately rather
// than read from t because the EFFECTIVE source may differ from the stored one
// — a missing or expired file falls back to the internal CA (see
// CheckCertificate), and the thing that gets rendered must be the effective
// one.
func (t *GatewayTLS) Caddy(upstreamHost string, upstreamPort int) CaddyConfig {
	c := CaddyConfig{
		UpstreamHost: upstreamHost,
		UpstreamPort: upstreamPort,
	}
	if t == nil {
		return c
	}
	c.Hostname = strings.TrimSpace(t.Hostname)
	c.CertSource = t.CertSourceOrDefault()
	c.CertFile = strings.TrimSpace(t.CertFile)
	c.KeyFile = strings.TrimSpace(t.KeyFile)
	c.HTTPSPort = t.HTTPSPort
	c.ImageRef = t.ImageRef
	return c
}

// ValidateSettings checks what can be checked without touching the target: the
// certificate FILES are validated separately, on the machine that holds them,
// by CheckCertificate.
func (t *GatewayTLS) ValidateSettings() error {
	if !t.On() {
		return nil
	}
	// Reuse the renderer's own rules for the hostname and cert source, with a
	// stand-in upstream, so the API cannot accept something the renderer will
	// later refuse.
	probe := t.Caddy("upstream", 4000)
	if err := probe.Validate(); err != nil {
		return err
	}
	if p := t.HTTPS(); p < 1 || p > 65535 {
		return fmt.Errorf("catalog: caddy: https port %d is out of range", p)
	}
	return nil
}

// caddyfileTemplate renders the Caddyfile.
//
// auto_https disable_redirects: Caddy would otherwise also bind :80 to redirect
// http→https. A gateway is called by programs, not typed into an address bar,
// and binding a second privileged port purely for a redirect is a needless
// collision with whatever else wants :80 on the operator's machine.
//
// The proxy is deliberately plain reverse_proxy with no header rewriting: eRPC
// addresses chains by URL path (/<project>/evm/<chainId>), so the path must
// reach it untouched, and WebSocket upgrades pass through natively.
const caddyfileTemplate = `{
	auto_https disable_redirects
}
{{.Hostname}} {
{{.TLSDirective}}
	reverse_proxy {{.Upstream}}
}
`

var caddyfileTmpl = template.Must(template.New("caddyfile").Parse(caddyfileTemplate))

type caddyVars struct {
	Hostname     string
	TLSDirective string
	Upstream     string
}

// CertSourceOrDefault resolves the cert source ("" → internal).
func (c CaddyConfig) CertSourceOrDefault() string {
	if c.CertSource == "" {
		return CertInternal
	}
	return c.CertSource
}

// Image resolves the Caddy image ref.
func (c CaddyConfig) Image() string {
	if c.ImageRef == "" {
		return DefaultCaddyImage
	}
	return c.ImageRef
}

// HTTPS resolves the published TLS port (0 → 443).
func (c CaddyConfig) HTTPS() int {
	if c.HTTPSPort == 0 {
		return 443
	}
	return c.HTTPSPort
}

// URL is the base https:// URL the gateway is reachable at. The port is
// omitted when it is the default, so the common case reads as a plain name.
func (c CaddyConfig) URL() string {
	if c.HTTPS() == 443 {
		return "https://" + c.Hostname
	}
	return fmt.Sprintf("https://%s:%d", c.Hostname, c.HTTPS())
}

// Validate rejects a config that would render a Caddyfile Caddy cannot serve.
func (c CaddyConfig) Validate() error {
	if strings.TrimSpace(c.Hostname) == "" {
		return fmt.Errorf("catalog: caddy: hostname is required")
	}
	// A scheme or path here means the operator pasted a URL. Caddy would
	// treat it as a site address and fail in a way that does not name the
	// cause, so say it plainly.
	if strings.Contains(c.Hostname, "://") || strings.Contains(c.Hostname, "/") {
		return fmt.Errorf("catalog: caddy: hostname %q must be a bare host name, not a URL", c.Hostname)
	}
	if strings.TrimSpace(c.UpstreamHost) == "" {
		return fmt.Errorf("catalog: caddy: upstream host is required")
	}
	if c.UpstreamPort <= 0 {
		return fmt.Errorf("catalog: caddy: upstream port %d is invalid", c.UpstreamPort)
	}
	switch c.CertSourceOrDefault() {
	case CertInternal:
	case CertFiles:
		if c.CertFile == "" || c.KeyFile == "" {
			return fmt.Errorf("catalog: caddy: cert source %q needs both certFile and keyFile", CertFiles)
		}
	case CertACME:
		return fmt.Errorf("catalog: caddy: cert source %q is not implemented", CertACME)
	default:
		return fmt.Errorf("catalog: caddy: unknown cert source %q", c.CertSource)
	}
	return nil
}

// RenderCaddyfile renders c. Pure string rendering; the caller writes it.
func RenderCaddyfile(c CaddyConfig) (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}

	var tlsDirective string
	switch c.CertSourceOrDefault() {
	case CertInternal:
		tlsDirective = "\ttls internal"
	case CertFiles:
		tlsDirective = fmt.Sprintf("\ttls %s %s", c.CertFile, c.KeyFile)
	}

	var buf bytes.Buffer
	if err := caddyfileTmpl.Execute(&buf, caddyVars{
		Hostname:     c.Hostname,
		TLSDirective: tlsDirective,
		Upstream:     fmt.Sprintf("%s:%d", c.UpstreamHost, c.UpstreamPort),
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// MustDisableGzipBehindProxy records a measured constraint that the gateway
// renderer has to honour once a gateway is fronted: erpc.yaml must set
// `server.enableGzip: false`.
//
// eRPC's WebSocket upgrade FAILS when the client advertises gzip. The upgrade
// needs http.Hijacker and eRPC's gzip response-writer wrapper does not
// implement it, so the handshake returns HTTP 500 with "websocket: response
// does not implement http.Hijacker". Every reverse proxy adds Accept-Encoding:
// gzip to proxied requests, so this breaks WebSocket behind ANY proxy, not
// just Caddy. Measured directly against the image built from ERPCSourceRef: an
// identical upgrade succeeds bare and fails with the header present, and
// setting enableGzip:false fixes both the direct and the proxied case.
//
// It is a workaround and it costs response compression for ordinary RPC. The
// real fix belongs in eRPC — skip the gzip wrapper on upgrade requests, or
// make the wrapper implement Hijacker — and is worth reporting upstream.
// Until then a fronted gateway must not offer gzip: silently losing
// eth_subscribe is far worse than losing compression.
const MustDisableGzipBehindProxy = true
