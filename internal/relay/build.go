package relay

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// defaultProjectID matches eRPC's own default project segment.
const defaultProjectID = "main"

// BuildOptions is the startup configuration for the data plane.
type BuildOptions struct {
	// RelayBind is the data plane's listen address. An empty value turns the
	// relay off, which is the default: a gateway that sells no keys needs none.
	RelayBind string
	// BillingSocket is the unix socket the Rust key store listens on.
	BillingSocket string
	// RelayToken is the least-privilege credential for /internal/authenticate.
	// It is NOT the admin token, which would also authorise minting keys.
	RelayToken string
	// ERPCURL is the keyless eRPC, reached over loopback.
	ERPCURL string
	// ProjectID is eRPC's project segment. Empty defaults to "main".
	ProjectID string
	// CacheTTL bounds how long a revoked key keeps working. Zero takes the
	// package default.
	CacheTTL time.Duration
	// Beacon resolves a chain's beacon client. Nil means this gateway serves no
	// beacon API, which the relay answers as a definite 501.
	Beacon func(chainID int) (*url.URL, bool)
	// PollInterval is the latency floor a synthesised subscription pays. Zero
	// takes the package default of one second.
	PollInterval time.Duration
}

// Build turns startup configuration into a data-plane handler. It returns a nil
// handler and a nil error when the relay is switched off.
//
// Every missing piece is a hard failure rather than a quiet fallback. A relay
// that disabled itself halfway would serve every customer's traffic for free,
// and nothing in the request path would report it — the operator would find out
// from a bill that never arrived.
func Build(opt BuildOptions) (http.Handler, error) {
	if opt.RelayBind == "" {
		return nil, nil
	}
	if opt.BillingSocket == "" {
		return nil, errors.New("relay: a relay bind is set with no billing socket")
	}
	if opt.RelayToken == "" {
		return nil, errors.New("relay: a relay bind is set with no relay token (JUMPGATE_RELAY_TOKEN)")
	}
	if opt.ERPCURL == "" {
		return nil, errors.New("relay: a relay bind is set with no erpc url")
	}

	erpc, err := url.Parse(opt.ERPCURL)
	if err != nil {
		return nil, fmt.Errorf("relay: erpc url %q: %w", opt.ERPCURL, err)
	}
	if erpc.Scheme == "" || erpc.Host == "" {
		return nil, fmt.Errorf("relay: erpc url %q needs a scheme and a host", opt.ERPCURL)
	}

	projectID := opt.ProjectID
	if projectID == "" {
		projectID = defaultProjectID
	}

	// The cache wraps the client so the store is asked once per key per TTL
	// rather than once per customer request.
	store := NewBillingClient(opt.BillingSocket, opt.RelayToken)
	cache := NewKeyCache(store, CacheOptions{TTL: opt.CacheTTL})

	// The caller and the streams are what let the relay TERMINATE a customer's
	// WebSocket. Without them a terminated session would have nothing to talk
	// to, so the handler answers 501 rather than proxying the upgrade — the one
	// thing this design must never do.
	caller := NewERPCCaller(erpc, projectID)

	return NewHandler(Config{
		Auth:      cache,
		ProjectID: projectID,
		ERPC:      erpc,
		Beacon:    opt.Beacon,
		Caller:    caller,
		Streams:   NewPollerStreams(caller, opt.PollInterval),
	})
}
