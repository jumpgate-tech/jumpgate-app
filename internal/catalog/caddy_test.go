package catalog

import (
	"regexp"
	"strings"
	"testing"
)

// The rendered shape here was verified by running it: Caddy served
// https://default-1a2b3c.localhost-valaxy.com, proxied to an eRPC container reached BY NAME
// on a private docker network with no published ports, and carried both
// JSON-RPC and eth_subscribe over wss.

func TestRenderCaddyfile_Internal(t *testing.T) {
	got, err := RenderCaddyfile(CaddyConfig{
		Hostname:     "default-1a2b3c.localhost-valaxy.com",
		UpstreamHost: "valve-node-app-erpc",
		UpstreamPort: 4000,
	})
	if err != nil {
		t.Fatalf("RenderCaddyfile: %v", err)
	}
	for _, want := range []string{
		"default-1a2b3c.localhost-valaxy.com {",
		"tls internal",
		"reverse_proxy valve-node-app-erpc:4000",
		// Without this Caddy also binds :80 purely to redirect, colliding
		// with whatever else wants that port on the operator's machine.
		"auto_https disable_redirects",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	// The upstream is addressed plainly: eRPC routes by URL path
	// (/<project>/evm/<chainId>), so nothing may rewrite the path.
	if strings.Contains(got, "rewrite") || strings.Contains(got, "strip_prefix") {
		t.Errorf("the path must reach eRPC untouched:\n%s", got)
	}
}

func TestRenderCaddyfile_Files(t *testing.T) {
	// The `tailscale cert` path: a genuine publicly-trusted certificate that
	// Tailscale issues and renews, so there is nothing to install and nothing
	// for us to run.
	got, err := RenderCaddyfile(CaddyConfig{
		Hostname:     "box.tail1234.ts.net",
		CertSource:   CertFiles,
		CertFile:     "/var/lib/valve-node-app/tls/cert.pem",
		KeyFile:      "/var/lib/valve-node-app/tls/key.pem",
		UpstreamHost: "erpc",
		UpstreamPort: 4000,
	})
	if err != nil {
		t.Fatalf("RenderCaddyfile: %v", err)
	}
	if !strings.Contains(got, "tls /var/lib/valve-node-app/tls/cert.pem /var/lib/valve-node-app/tls/key.pem") {
		t.Errorf("cert/key not applied:\n%s", got)
	}
	if strings.Contains(got, "tls internal") {
		t.Errorf("must not fall back to the internal CA when files were given:\n%s", got)
	}
}

// The Public tier: the operator's own real domain and a Let's Encrypt cert
// that stock Caddy obtains itself over HTTP-01 / TLS-ALPN. Two shapes matter:
// the site address is the real domain, and the global auto_https block is GONE
// (that block is what stops Caddy binding :80, which HTTP-01 needs).
func TestRenderCaddyfile_ACME(t *testing.T) {
	got, err := RenderCaddyfile(CaddyConfig{
		Hostname:     "rpc.your-company.com",
		CertSource:   CertACME,
		UpstreamHost: "valve-node-app-erpc",
		UpstreamPort: 4000,
	})
	if err != nil {
		t.Fatalf("RenderCaddyfile: %v", err)
	}
	for _, want := range []string{
		"rpc.your-company.com {",
		"reverse_proxy valve-node-app-erpc:4000",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	// The global block MUST be omitted for a public front: it is what keeps
	// Caddy off :80, and HTTP-01 needs :80.
	if strings.Contains(got, "auto_https disable_redirects") {
		t.Errorf("a public front must bind :80 for HTTP-01; the global block must be omitted:\n%s", got)
	}
	// No email given: Caddy runs auto-HTTPS, so no tls directive at all.
	if strings.Contains(got, "tls ") {
		t.Errorf("no email was set, so there must be no tls directive:\n%s", got)
	}
	// The path must reach eRPC untouched.
	if strings.Contains(got, "rewrite") || strings.Contains(got, "strip_prefix") {
		t.Errorf("the path must reach eRPC untouched:\n%s", got)
	}
}

// With an email the operator gets Let's Encrypt expiry notices; render `tls
// <email>`. Only then — an empty email must not produce a bare `tls`.
func TestRenderCaddyfile_ACMEWithEmail(t *testing.T) {
	got, err := RenderCaddyfile(CaddyConfig{
		Hostname:     "rpc.your-company.com",
		CertSource:   CertACME,
		ACMEEmail:    "ops@your-company.com",
		UpstreamHost: "erpc",
		UpstreamPort: 4000,
	})
	if err != nil {
		t.Fatalf("RenderCaddyfile: %v", err)
	}
	if !strings.Contains(got, "tls ops@your-company.com") {
		t.Errorf("want the operator's email in the tls directive:\n%s", got)
	}
	if strings.Contains(got, "auto_https disable_redirects") {
		t.Errorf("a public front must not carry the global block:\n%s", got)
	}
}

// The Public tier accepts a real public domain and rejects the three shapes
// that could never get a public certificate: a loopback name under our own
// wildcard, a bare IP, and a single-label host.
func TestCaddyConfig_ValidateACME(t *testing.T) {
	base := CaddyConfig{CertSource: CertACME, UpstreamHost: "erpc", UpstreamPort: 4000}

	if err := (func() error { c := base; c.Hostname = "rpc.your-company.com"; return c.Validate() })(); err != nil {
		t.Errorf("a public FQDN must be accepted: %v", err)
	}
	// An email that looks like one is fine; one that does not is rejected.
	if err := (func() error { c := base; c.Hostname = "rpc.x.com"; c.ACMEEmail = "ops@x.com"; return c.Validate() })(); err != nil {
		t.Errorf("a valid email must be accepted: %v", err)
	}

	bad := []struct {
		name string
		host string
		mail string
		want string
	}{
		{"loopback wildcard", "gw-abc.localhost-valaxy.com", "", "loopback"},
		{"bare ipv4", "203.0.113.10", "", "IP"},
		{"single label", "localhost", "", "single label"},
		{"bad email", "rpc.x.com", "not-an-email", "email"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			c.Hostname = tc.host
			c.ACMEEmail = tc.mail
			err := c.Validate()
			if err == nil {
				t.Fatalf("want a rejection for %q", tc.host)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestCaddyConfig_URL(t *testing.T) {
	base := CaddyConfig{Hostname: "h.example", UpstreamHost: "erpc", UpstreamPort: 4000}
	if got := base.URL(); got != "https://h.example" {
		t.Errorf("default port should be omitted, got %q", got)
	}
	custom := base
	custom.HTTPSPort = 8443
	if got := custom.URL(); got != "https://h.example:8443" {
		t.Errorf("got %q", got)
	}
}

func TestRenderCaddyfile_Rejects(t *testing.T) {
	valid := CaddyConfig{Hostname: "h.example", UpstreamHost: "erpc", UpstreamPort: 4000}

	tests := []struct {
		name string
		mut  func(*CaddyConfig)
		want string
	}{
		{"no hostname", func(c *CaddyConfig) { c.Hostname = " " }, "hostname is required"},
		{
			// Pasting a URL here is the likely operator mistake, and Caddy's
			// own failure would not name the cause.
			name: "hostname is a URL",
			mut:  func(c *CaddyConfig) { c.Hostname = "https://h.example/" },
			want: "must be a bare host name",
		},
		{"no upstream host", func(c *CaddyConfig) { c.UpstreamHost = "" }, "upstream host is required"},
		{"bad upstream port", func(c *CaddyConfig) { c.UpstreamPort = 0 }, "upstream port"},
		{
			name: "files without files",
			mut:  func(c *CaddyConfig) { c.CertSource = CertFiles },
			want: "needs both certFile and keyFile",
		},
		{
			// ACME is implemented now, but the default hostname resolves to
			// loopback, which no public CA can validate — so acme on it fails.
			name: "acme on a loopback name",
			mut:  func(c *CaddyConfig) { c.CertSource = CertACME; c.Hostname = "gw.localhost-valaxy.com" },
			want: "loopback",
		},
		{"unknown source", func(c *CaddyConfig) { c.CertSource = "magic" }, "unknown cert source"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := valid
			tc.mut(&c)
			_, err := RenderCaddyfile(c)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

// The default hostname has to be usable without the operator owning anything:
// a name under a wildcard that already resolves to loopback, valid as a DNS
// name, stable for one install and different across installs.
func TestDefaultTLSHostname(t *testing.T) {
	got := DefaultTLSHostname("default", "machine-a")
	if !strings.HasSuffix(got, "."+DefaultTLSDomain) {
		t.Fatalf("hostname %q is not under %s", got, DefaultTLSDomain)
	}
	if !strings.HasPrefix(got, "default-") {
		t.Errorf("hostname %q should start with the gateway id", got)
	}
	// One label under the wildcard, so a single *.<domain> record covers it.
	if label := strings.TrimSuffix(got, "."+DefaultTLSDomain); strings.Contains(label, ".") {
		t.Errorf("hostname %q uses more than one label under the wildcard", got)
	}
	if again := DefaultTLSHostname("default", "machine-a"); again != got {
		t.Errorf("not stable for one install: %q then %q", got, again)
	}
	if other := DefaultTLSHostname("default", "machine-b"); other == got {
		t.Errorf("two installs got the same name %q — two machines would serve different certificates for it", got)
	}
	// The per-install tag must carry enough entropy not to collide at scale:
	// 8 bytes -> 16 hex chars (64 bits). Guards against shrinking it back to the
	// old 24-bit tag, which collided by the birthday bound at ~4k installs.
	tag := strings.TrimSuffix(strings.TrimPrefix(got, "default-"), "."+DefaultTLSDomain)
	if len(tag) < 16 {
		t.Errorf("per-install tag %q is %d chars; want >= 16 hex (64 bits)", tag, len(tag))
	}

	// A gateway id may contain dots and underscores; a DNS label may not.
	if got := DefaultTLSHostname("edge_1.eu", "seed"); strings.Contains(strings.TrimSuffix(got, "."+DefaultTLSDomain), ".") ||
		strings.Contains(got, "_") {
		t.Errorf("id was not reduced to a DNS label: %q", got)
	}
	// And an id with nothing usable in it still yields a name.
	if got := DefaultTLSHostname("...", "seed"); !strings.HasPrefix(got, "gateway-") {
		t.Errorf("want a fallback label, got %q", got)
	}
}

// Security fix: the rendered Caddyfile must redact a customer key on sight,
// so the day someone adds a debug log directive they cannot write a raw key
// to disk. See docs/superpowers/specs/2026-08-15-relay-keyed-access-design.md
// section 10.
func TestRenderCaddyfile_RedactsKeyInLog(t *testing.T) {
	got, err := RenderCaddyfile(CaddyConfig{
		Hostname:     "default-1a2b3c.localhost-valaxy.com",
		UpstreamHost: "valve-node-app-erpc",
		UpstreamPort: 4000,
	})
	if err != nil {
		t.Fatalf("RenderCaddyfile: %v", err)
	}
	for _, want := range []string{
		"format filter",
		"wrap console",
		"request>uri regexp",
		`"jg_REDACTED"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q from the log redaction block:\n%s", want, got)
		}
	}
}

// The anti-drift guard for CHANGE 2: this compiles the exact pattern the
// template renders and checks it against a sample key. A future typo in the
// character class then fails the build instead of silently logging keys.
func TestKeyRedactionPattern_MatchesASampleKey(t *testing.T) {
	re, err := regexp.Compile(keyRedactionPattern)
	if err != nil {
		t.Fatalf("keyRedactionPattern does not compile: %v", err)
	}
	// A key is "jg_" plus base58: this sample deliberately avoids 0, O, I
	// and l, none of which are valid base58 characters.
	sample := "jg_1a2B3c4D5e6F7g8H9j"
	if !re.MatchString(sample) {
		t.Errorf("pattern %q must match a sample key %q", keyRedactionPattern, sample)
	}
}

// A non-metered gateway keeps pointing straight at eRPC, exactly as today.
func TestRenderCaddyfile_NotMetered(t *testing.T) {
	got, err := RenderCaddyfile(CaddyConfig{
		Hostname:     "default-1a2b3c.localhost-valaxy.com",
		UpstreamHost: "valve-node-app-erpc",
		UpstreamPort: 4000,
	})
	if err != nil {
		t.Fatalf("RenderCaddyfile: %v", err)
	}
	if !strings.Contains(got, "reverse_proxy valve-node-app-erpc:4000") {
		t.Errorf("a non-metered gateway must proxy straight to eRPC:\n%s", got)
	}
}

// A metered gateway's single reverse_proxy targets the relay, not eRPC. One
// line is enough: the relay itself serves /rpc, /beacon and /health, so a
// matcher per category would be redundant.
func TestRenderCaddyfile_Metered(t *testing.T) {
	got, err := RenderCaddyfile(CaddyConfig{
		Hostname:     "default-1a2b3c.localhost-valaxy.com",
		UpstreamHost: "valve-node-app-erpc",
		UpstreamPort: 4000,
		Metered:      true,
		RelayHost:    "valve-node-app-relay",
		RelayPort:    8090,
	})
	if err != nil {
		t.Fatalf("RenderCaddyfile: %v", err)
	}
	if !strings.Contains(got, "reverse_proxy valve-node-app-relay:8090") {
		t.Errorf("a metered gateway must proxy to the relay:\n%s", got)
	}
	if strings.Contains(got, "valve-node-app-erpc:4000") {
		t.Errorf("a metered gateway must not also proxy to eRPC directly:\n%s", got)
	}
	// Exactly one reverse_proxy line: the relay owns all three categories, so
	// a matcher per category would be redundant.
	if n := strings.Count(got, "reverse_proxy"); n != 1 {
		t.Errorf("want exactly one reverse_proxy directive, got %d:\n%s", n, got)
	}
}

// Caddy must rewrite nothing for a metered gateway: the relay owns every
// path strip, so there must be no uri strip_prefix and no handle_path.
func TestRenderCaddyfile_MeteredHasNoRewritingDirective(t *testing.T) {
	got, err := RenderCaddyfile(CaddyConfig{
		Hostname:     "default-1a2b3c.localhost-valaxy.com",
		UpstreamHost: "valve-node-app-erpc",
		UpstreamPort: 4000,
		Metered:      true,
		RelayHost:    "valve-node-app-relay",
		RelayPort:    8090,
	})
	if err != nil {
		t.Fatalf("RenderCaddyfile: %v", err)
	}
	for _, unwanted := range []string{"rewrite", "strip_prefix", "handle_path"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("a metered gateway must not rewrite the path, found %q:\n%s", unwanted, got)
		}
	}
}

// A metered gateway needs a relay upstream. An empty one must be a
// validation error, the same as an empty eRPC upstream is today.
func TestRenderCaddyfile_MeteredRejectsEmptyRelay(t *testing.T) {
	base := CaddyConfig{Hostname: "h.example", UpstreamHost: "erpc", UpstreamPort: 4000, Metered: true}

	t.Run("no relay host", func(t *testing.T) {
		c := base
		c.RelayPort = 8090
		_, err := RenderCaddyfile(c)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "relay host") {
			t.Errorf("error = %q, want it to mention the relay host", err)
		}
	})

	t.Run("bad relay port", func(t *testing.T) {
		c := base
		c.RelayHost = "relay"
		_, err := RenderCaddyfile(c)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "relay port") {
			t.Errorf("error = %q, want it to mention the relay port", err)
		}
	})
}
