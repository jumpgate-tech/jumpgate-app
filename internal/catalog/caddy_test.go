package catalog

import (
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
			// Named but unimplemented on purpose: DNS-01 is the only usable
			// challenge for a loopback name, and that needs zone credentials.
			name: "acme is not implemented",
			mut:  func(c *CaddyConfig) { c.CertSource = CertACME },
			want: "not implemented",
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
