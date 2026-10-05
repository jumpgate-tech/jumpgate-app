package ops

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// HTTPProbe is one HTTP request a check makes FROM the target, to a service
// on the target: a gateway's eth_chainId, a devnet's block number, the
// metrics page. It has two renderings that must mean the same request: the
// curl command an SSH target runs, unchanged from before Task 15, and an
// in-process request on the local machine, which needs neither curl nor a
// shell (spec D31).
type HTTPProbe struct {
	URL string
	// Body, when set, is POSTed as application/json; otherwise the probe is a GET.
	Body string
	// Resolve, when set, is curl's --resolve value "name:port:addr": the
	// URL's name:port is connected at addr:port, so a TLS front is probed by
	// its hostname without that name resolving.
	Resolve string
	// CAFile, when set, is a PEM file on the target trusted for the server's
	// certificate. A request that fails with it is retried once with the
	// system roots, as the curl form's `|| …` does.
	CAFile string
	// MaxTime bounds the request; 0 leaves it to ctx.
	MaxTime time.Duration
}

// ProbeError is a probe that ran and got no answer: curl exited non-zero,
// or the in-process request failed. A readiness loop keeps polling on it;
// any other error from Do is the executor failing.
type ProbeError struct{ Detail string }

func (e *ProbeError) Error() string { return e.Detail }

// CurlCommand renders p as the command an SSH target runs. The byte layout
// is the one gateway.go, devnet.go and traffic.go each built by hand before.
func (p HTTPProbe) CurlCommand() string {
	base := []string{"curl -s"}
	if p.MaxTime > 0 {
		base = append(base, fmt.Sprintf("--max-time %d", int(p.MaxTime/time.Second)))
	}
	if p.Body != "" {
		base = append(base, "-X POST -H 'Content-Type: application/json' --data "+shQuote(p.Body))
	}
	if p.Resolve != "" {
		base = append(base, "--resolve "+shQuote(p.Resolve))
	}
	attempt := func(ca string) string {
		parts := append([]string(nil), base...)
		if ca != "" {
			parts = append(parts, "--cacert "+shQuote(ca))
		}
		return strings.Join(append(parts, shQuote(p.URL)), " ")
	}
	if p.CAFile == "" {
		return attempt("")
	}
	return attempt(p.CAFile) + " || " + attempt("")
}

// Do runs p on e and returns the response body, whatever the HTTP status
// (as `curl -s` does). On the local machine the request is made in process,
// dialled through executor.LocalHost. Anywhere else the curl form runs on the
// target.
func (p HTTPProbe) Do(ctx context.Context, e executor.Executor) (string, error) {
	h, ok := e.(executor.LocalHost)
	if !ok {
		res, err := e.Run(ctx, p.CurlCommand(), nil)
		if err != nil {
			return "", err
		}
		if res.ExitCode != 0 {
			return "", &ProbeError{Detail: fmt.Sprintf("curl exit %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))}
		}
		return res.Stdout, nil
	}
	var roots *x509.CertPool
	if p.CAFile != "" {
		if pemBytes, err := e.ReadFile(ctx, p.CAFile); err == nil {
			roots = x509.NewCertPool()
			if !roots.AppendCertsFromPEM(pemBytes) {
				roots = nil
			}
		}
	}
	body, err := p.local(ctx, h, roots)
	if err != nil && roots != nil {
		body, err = p.local(ctx, h, nil)
	}
	if err != nil {
		return "", &ProbeError{Detail: err.Error()}
	}
	return body, nil
}

func (p HTTPProbe) local(ctx context.Context, h executor.LocalHost, roots *x509.CertPool) (string, error) {
	from, to := "", ""
	if parts := strings.SplitN(p.Resolve, ":", 3); len(parts) == 3 {
		from = net.JoinHostPort(parts[0], parts[1])
		to = net.JoinHostPort(strings.Trim(parts[2], "[]"), parts[1])
	}
	tr := &http.Transport{
		// A probe of this machine's own port never goes through a proxy.
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if from != "" && addr == from {
				addr = to
			}
			return h.DialContext(ctx, network, addr)
		},
		TLSClientConfig:   &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
		DisableKeepAlives: true,
	}
	defer tr.CloseIdleConnections()
	method, rd := http.MethodGet, io.Reader(nil)
	if p.Body != "" {
		method, rd = http.MethodPost, strings.NewReader(p.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.URL, rd)
	if err != nil {
		return "", err
	}
	if p.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Transport: tr, Timeout: p.MaxTime}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return string(b), err
}
