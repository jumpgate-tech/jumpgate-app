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
	// BeaconEndpoints is the per-chain beacon pool. A chain absent here has no
	// consensus layer and answers a definite 501.
	BeaconEndpoints map[int][]*url.URL
	// Chains is what this gateway serves, for the health rollup.
	Chains []int
	// Credits meters spend. Nil leaves metering OFF, which is how a gateway
	// behaves before an operator switches billing on.
	Credits CreditStore
	// CreditBlock is how many credits to lease at a time. Zero takes the default.
	CreditBlock int64
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

	cfg := Config{
		Auth:      cache,
		ProjectID: projectID,
		ERPC:      erpc,
		Beacon:    opt.Beacon,
		Caller:    caller,
		Streams:   NewPollerStreams(caller, opt.PollInterval),
	}

	// The beacon pool round-robins a chain's consensus upstreams and drops the
	// ones that report themselves down. Its Next satisfies the Beacon hook, so
	// an explicit hook still wins when a caller supplies one.
	var pool *BeaconPool
	if len(opt.BeaconEndpoints) > 0 {
		pool = NewBeaconPool(opt.BeaconEndpoints)
		if cfg.Beacon == nil {
			cfg.Beacon = pool.Next
		}
	}
	cfg.Health = NewHealthProbe(caller, pool, opt.Chains)

	// Metering stays off unless a ledger is supplied. Serving unmetered is the
	// status quo; switching billing on must be deliberate.
	if opt.Credits != nil {
		cfg.Credits = NewCreditLease(opt.Credits, CreditOptions{BlockSize: opt.CreditBlock})
	}

	return NewHandler(cfg)
}

// BuildBeaconPool exposes the pool a caller must Run so it keeps re-probing. A
// pool that never re-probed would shrink to nothing over a long uptime.
func BuildBeaconPool(endpoints map[int][]*url.URL) *BeaconPool {
	if len(endpoints) == 0 {
		return nil
	}
	return NewBeaconPool(endpoints)
}

// BuildAdmin builds the operator's key-management client. It returns nil when
// no billing socket is configured, so a gateway that sells no keys simply has
// no key store.
//
// The caller must assign the result to a KeyAdmin interface ONLY when it is
// non-nil. A nil *AdminClient stored in a non-nil interface would pass every
// `== nil` check and then panic on the first call — the classic Go typed-nil
// trap, and here it would turn a clean 501 into a crash on an operator's click.
func BuildAdmin(socketPath, adminToken string) (*AdminClient, error) {
	if socketPath == "" {
		return nil, nil
	}
	if adminToken == "" {
		return nil, errors.New("relay: a billing socket is set with no admin token (JUMPGATE_ADMIN_TOKEN)")
	}
	return NewAdminClient(socketPath, adminToken), nil
}
