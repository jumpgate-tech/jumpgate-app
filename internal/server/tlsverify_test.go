package server

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// GET /api/gateways/{gid}/tls/verify sat at 31% for the same reason
// /api/chainlist did: the real check dials the front, completes a TLS
// handshake, opens a WebSocket and WAITS FOR A BLOCK. That is what makes it
// worth having and what makes it untestable in place, so Config.VerifyTLS is
// the seam — the wiring around the check is what these test, and the check
// itself is internal/setup's own business.

// frontedGateway is a gateway with HTTPS on, which is the precondition the
// route checks before it does anything else.
func frontedGateway() catalog.GatewayConfig {
	return catalog.GatewayConfig{
		Port: 4100,
		TLS: &catalog.GatewayTLS{
			Enabled:  true,
			Hostname: "gw.localhost-valaxy.com",
		},
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "public-369-1", Endpoint: "https://rpc.pulsechain.com"},
		}}},
	}
}

func tlsVerifyServer(t *testing.T, verify func(context.Context, executor.Executor, string, catalog.GatewayConfig, string) (setup.TLSVerification, error)) *apiTestServer {
	t.Helper()
	a := newAPITestServerCfg(t,
		func(config.Target) (executor.Executor, error) { return readyExecutor(), nil },
		func(c *Config) { c.VerifyTLS = verify },
	)
	addTarget(t, a)
	return a
}

// The happy path, and the two things about it worth pinning: the result
// reaches the wire, and it is CACHED — the RPC screen shows the last answer
// beside the front on every poll, and re-running a check that waits for a
// block on each of those polls would make the screen unusable.
func TestHandleGatewayTLSVerify_ResultIsReturnedAndRemembered(t *testing.T) {
	at := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	var calls int
	a := tlsVerifyServer(t, func(_ context.Context, _ executor.Executor, gid string, g catalog.GatewayConfig, dialHost string) (setup.TLSVerification, error) {
		calls++
		if gid != "default" {
			t.Errorf("gateway id passed to the check: %q", gid)
		}
		if !g.Fronted() {
			t.Error("the check must be handed the RESOLVED config, front and all")
		}
		// The name is pinned to an address exactly as the provisioner's curl
		// --resolve pins it: DNS is not what is under test.
		if dialHost == "" {
			t.Error("the check must be told what address to pin the hostname to")
		}
		return setup.TLSVerification{At: at, URL: "https://gw.localhost-valaxy.com", Hostname: g.TLS.Hostname, Address: dialHost, ChainID: 369}, nil
	})
	addGateway(t, a, "default", "local", frontedGateway())

	res := a.do(t, "GET", "/api/gateways/default/tls/verify", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
	body := decode[setup.TLSVerification](t, res)
	if body.ChainID != 369 || body.Hostname != "gw.localhost-valaxy.com" {
		t.Errorf("verification: %+v", body)
	}

	// The gateway view now carries it, without the check running again.
	view := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	if view.TLS.Verification == nil {
		t.Fatalf("the gateway view must carry the last verification: %+v", view.TLS)
	}
	if !view.TLS.Verification.At.Equal(at) {
		t.Errorf("cached verification: got %v, want the one just run (%v)", view.TLS.Verification.At, at)
	}
	if calls != 1 {
		t.Errorf("the check ran %d times — showing the last answer must not re-run it, it waits for a block", calls)
	}
}

// A gateway with no HTTPS front has nothing to verify. That is a 400 with a
// typed code and an instruction, not a failed check: the answer is a setting
// the operator has not turned on, and saying "verification failed" would send
// them looking for a broken certificate.
func TestHandleGatewayTLSVerify_NoFrontIs400WithTheFix(t *testing.T) {
	var called bool
	a := tlsVerifyServer(t, func(context.Context, executor.Executor, string, catalog.GatewayConfig, string) (setup.TLSVerification, error) {
		called = true
		return setup.TLSVerification{}, nil
	})
	cfg := frontedGateway()
	cfg.TLS = nil
	addGateway(t, a, "default", "local", cfg)

	res := a.do(t, "GET", "/api/gateways/default/tls/verify", nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	body := decode[errorDetail](t, res)
	if body.Code != codeNotConfigured {
		t.Errorf("code: got %q, want %q", body.Code, codeNotConfigured)
	}
	if body.Hint == "" {
		t.Error("the hint must say what to turn on — the fix is a setting, not a repair")
	}
	if called {
		t.Error("a gateway with no front must not be dialed at all")
	}
}

// A failure from the check is classified rather than flattened: a container
// that was never created is a 409 the operator fixes by creating it, and a
// docker engine that is not there is a 502 with the reason. Both would
// otherwise arrive as an indistinguishable 502 with a bare message.
func TestHandleGatewayTLSVerify_FailuresKeepTheirClassification(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		want     int
		wantCode string
	}{
		{
			name:     "the front was never created",
			err:      &ops.ServiceNotCreatedError{ID: "caddy:default", ContainerName: "valve-node-app-caddy", Action: "start"},
			want:     http.StatusConflict,
			wantCode: codeNotCreated,
		},
		{
			name: "anything else is upstream of this app",
			err:  errors.New("the handshake fell over"),
			want: http.StatusBadGateway,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := tlsVerifyServer(t, func(context.Context, executor.Executor, string, catalog.GatewayConfig, string) (setup.TLSVerification, error) {
				return setup.TLSVerification{}, tc.err
			})
			addGateway(t, a, "default", "local", frontedGateway())

			res := a.do(t, "GET", "/api/gateways/default/tls/verify", nil)
			if res.StatusCode != tc.want {
				t.Fatalf("got %d, want %d", res.StatusCode, tc.want)
			}
			body := decode[errorDetail](t, res)
			if tc.wantCode != "" && body.Code != tc.wantCode {
				t.Errorf("code: got %q, want %q", body.Code, tc.wantCode)
			}
			if body.Error == "" {
				t.Error("the reason must reach the operator")
			}

			// A failed check must NOT be cached: the screen showing "verified"
			// after a check that failed is the worst possible outcome here.
			view := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
			if view.TLS.Verification != nil {
				t.Errorf("a failed check was cached as a result: %+v", view.TLS.Verification)
			}
		})
	}
}

func TestHandleGatewayTLSVerify_UnknownGatewayIs404(t *testing.T) {
	a := tlsVerifyServer(t, func(context.Context, executor.Executor, string, catalog.GatewayConfig, string) (setup.TLSVerification, error) {
		t.Fatal("must not run for a gateway that does not exist")
		return setup.TLSVerification{}, nil
	})
	res := a.do(t, "GET", "/api/gateways/nope/tls/verify", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusNotFound || body.Code != codeGatewayNotFound {
		t.Fatalf("got %d/%q", res.StatusCode, body.Code)
	}
}

// tlsDialHost pins the name to an address, and for a machine reached over SSH
// that address is the host this app reaches it on — the one address known to
// route there from here. A loopback bind on that machine would be OUR
// loopback, which is the same mistake reachableAcrossMachines exists to catch.
func TestTLSDialHost_SSHTargetIsPinnedToTheHostWeReachItOn(t *testing.T) {
	tls := &catalog.GatewayTLS{Enabled: true, Hostname: "gw.example", BindAddr: "0.0.0.0"}

	local := config.Target{ID: "here", Mode: "local"}
	if got := tlsDialHost(local, tls); got != "127.0.0.1" {
		t.Errorf("local: got %q, want loopback — a wildcard bind names every interface but is not itself a destination", got)
	}

	ssh := config.Target{ID: "boxa", Mode: "ssh", SSH: &executor.SSHConfig{Host: "100.64.0.7"}}
	if got := tlsDialHost(ssh, tls); got != "100.64.0.7" {
		t.Errorf("ssh: got %q, want the host this app reaches that machine on", got)
	}
}
