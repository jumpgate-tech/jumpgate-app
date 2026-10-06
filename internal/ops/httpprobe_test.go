package ops

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// The shell forms are the strings gateway.go, devnet.go and traffic.go built
// by hand before Task 15. SSH targets and the existing fakes see them
// unchanged.
func TestHTTPProbeCurlCommandIsTheHistoricString(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`
	for _, tc := range []struct {
		p    HTTPProbe
		want string
	}{
		{HTTPProbe{URL: "http://127.0.0.1:4000/main/evm/1", Body: body, MaxTime: 10 * time.Second},
			`curl -s --max-time 10 -X POST -H 'Content-Type: application/json' --data '` + body + `' 'http://127.0.0.1:4000/main/evm/1'`},
		{HTTPProbe{URL: "http://127.0.0.1:8545", Body: body},
			`curl -s -X POST -H 'Content-Type: application/json' --data '` + body + `' 'http://127.0.0.1:8545'`},
		{HTTPProbe{URL: "http://127.0.0.1:4001/metrics", MaxTime: 5 * time.Second},
			`curl -s --max-time 5 'http://127.0.0.1:4001/metrics'`},
		{HTTPProbe{URL: "https://rpc.lan:8443/main/evm/1", Body: body, MaxTime: 10 * time.Second, Resolve: "rpc.lan:8443:127.0.0.1", CAFile: "/h/ca.crt"},
			`curl -s --max-time 10 -X POST -H 'Content-Type: application/json' --data '` + body + `' --resolve 'rpc.lan:8443:127.0.0.1' --cacert '/h/ca.crt' 'https://rpc.lan:8443/main/evm/1'` +
				` || curl -s --max-time 10 -X POST -H 'Content-Type: application/json' --data '` + body + `' --resolve 'rpc.lan:8443:127.0.0.1' 'https://rpc.lan:8443/main/evm/1'`},
	} {
		if got := tc.p.CurlCommand(); got != tc.want {
			t.Errorf("CurlCommand:\n got %s\nwant %s", got, tc.want)
		}
	}
}

// Local: in process, the name resolved to the given address, the CA read
// with e.ReadFile, and no shell.
func TestHTTPProbeDoLocalResolvesAndTrustsTheCAFile(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || string(b) != `{"q":1}` {
			http.Error(w, "bad request", 400)
			return
		}
		_, _ = w.Write([]byte(`{"result":"0x171"}`))
	}))
	defer ts.Close()
	f := argvfake.New()
	f.Files["/h/ca.crt"] = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw})
	addr := strings.TrimPrefix(ts.URL, "https://")
	// httptest's certificate is valid for example.com; resolve it to the server.
	f.Route("127.0.0.1:8443", addr)
	p := HTTPProbe{URL: "https://example.com:8443/x", Body: `{"q":1}`, Resolve: "example.com:8443:127.0.0.1", CAFile: "/h/ca.crt", MaxTime: 5 * time.Second}
	got, err := p.Do(context.Background(), f)
	if err != nil || got != `{"result":"0x171"}` {
		t.Fatalf("got %q, %v", got, err)
	}
	if len(f.ShellCalls()) != 0 {
		t.Fatalf("shell used: %q", f.ShellCalls())
	}
}

// curl's `--cacert X || plain` fallback: a CA file that does not sign the
// server's certificate is retried once against the system roots. Neither
// signs httptest's certificate here, so the result is one ProbeError naming
// the TLS failure, not a panic or a hang.
func TestHTTPProbeDoLocalFallsBackToSystemRootsThenReportsTheTLSFailure(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }))
	defer ts.Close()
	f := argvfake.New()
	// Every httptest server shares one built-in certificate, so the CA that
	// does not sign it is made here.
	f.Files["/h/ca.crt"] = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: unrelatedCA(t)})
	u, _ := url.Parse(ts.URL)
	f.Route(u.Host, u.Host)
	_, err := HTTPProbe{URL: ts.URL, CAFile: "/h/ca.crt", MaxTime: 5 * time.Second}.Do(context.Background(), f)
	var pe *ProbeError
	if !errors.As(err, &pe) || !strings.Contains(pe.Detail, "certificate") {
		t.Fatalf("got %v, want a ProbeError about the certificate", err)
	}
}

// A refused connection is a ProbeError, so a readiness loop keeps polling.
func TestHTTPProbeDoLocalRefusedIsAProbeError(t *testing.T) {
	_, err := HTTPProbe{URL: "http://127.0.0.1:1/"}.Do(context.Background(), argvfake.New())
	var pe *ProbeError
	if !errors.As(err, &pe) {
		t.Fatalf("got %v, want *ProbeError", err)
	}
}

// unrelatedCA is a self-signed CA certificate (DER) that signs nothing the
// tests serve.
func unrelatedCA(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "unrelated test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// A redirect is answered as is, like `curl -s` without -L: a probe of a
// gateway must read what the port itself serves, not wherever it points.
func TestHTTPProbeDoLocalDoesNotFollowRedirects(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/elsewhere" {
			_, _ = w.Write([]byte("followed"))
			return
		}
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	}))
	defer ts.Close()
	f := argvfake.New()
	u, _ := url.Parse(ts.URL)
	f.Route(u.Host, u.Host)
	got, err := HTTPProbe{URL: ts.URL + "/", MaxTime: 5 * time.Second}.Do(context.Background(), f)
	if err != nil || strings.Contains(got, "followed") {
		t.Fatalf("got %q, %v; want the redirect response itself", got, err)
	}
}
