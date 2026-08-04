// Package server implements the token-gated local HTTP server that serves
// the embedded web UI and the JSON API for valve-node-app.
package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/valve-node-app/internal/ai"
	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/chainlist"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// cookieName is the name of the cookie that carries the session token once
// it has been established via the ?token= query parameter.
const cookieName = "valve_node_token"

// Config configures a Server.
type Config struct {
	// Bind is the host:port the server listens on, e.g. "127.0.0.1:8799".
	Bind string
	// Token is the session token that authorizes API and UI requests.
	Token string
	// UI is the filesystem the static web UI is served from.
	UI fs.FS
	// NewExecutor builds the executor.Executor for a config.Target — local
	// or SSH depending on Target.Mode. Injectable for tests (a fake); nil
	// selects defaultNewExecutor, which dials the real thing.
	NewExecutor func(config.Target) (executor.Executor, error)
	// NewAIProvider builds an ai.Provider by id. Injectable for tests; nil
	// selects ai.New.
	NewAIProvider func(id, apiKey, baseURL string) (ai.Provider, error)

	// NewChainlist builds the public-endpoint discoverer. Injectable for
	// tests; nil selects chainlist.New.
	//
	// This exists for the same reason NewExecutor does. handleChainlist
	// fetches a 1.1 MB feed off the public internet and then opens a
	// connection to every endpoint it lists — so without a seam, the only
	// honest test of that route is one that talks to the real internet, which
	// is a test that fails on a plane and passes when a third-party feed
	// happens to be up.
	NewChainlist func() *chainlist.Discoverer

	// VerifyTLS runs the live HTTPS check behind GET
	// /api/gateways/{gid}/tls/verify. Injectable for tests; nil selects
	// setup.VerifyGatewayTLS.
	//
	// The real one dials the front, completes a TLS handshake, opens a
	// WebSocket and WAITS FOR A BLOCK. That is exactly what makes it worth
	// having and exactly what makes it untestable in place.
	VerifyTLS func(ctx context.Context, e executor.Executor, gatewayID string, g catalog.GatewayConfig, dialHost string) (setup.TLSVerification, error)
}

// Server is the valve-node-app local HTTP server.
type Server struct {
	cfg Config

	// cfgMu serializes read-modify-write access to the on-disk
	// internal/config file across concurrent API requests.
	cfgMu sync.Mutex

	reg *registry

	// tlsChecks is the last live HTTPS verification per gateway id, guarded by
	// tlsMu. It is a cache of an EXPENSIVE read (real connections, a real
	// subscription), kept so the RPC screen can show the last answer without
	// re-running it on every poll — see handleGatewayTLSVerify.
	tlsMu     sync.Mutex
	tlsChecks map[string]setup.TLSVerification

	// capChecks is the last capability probe per gateway id, guarded by capMu.
	// Unlike tlsChecks (which never expires — see its comment), this one
	// carries an explicit TTL (capabilitiesTTL, in capabilities.go): a
	// capability probe dials EVERY upstream on the gateway with roughly a
	// dozen calls apiece, not one front, so leaving it to go stale forever
	// would eventually show an operator a years-old verdict with nothing on
	// screen to say so. Ten minutes bounds the staleness while still keeping
	// the RPC screen's normal poll cadence from turning into sustained load
	// against every upstream a gateway fronts.
	capMu     sync.Mutex
	capChecks map[string]capabilitiesResponse

	// chainsMu guards the cached full chain catalogue (id + name for every
	// chain the feed knows) that backs the network-search picker. The feed is
	// ~1.1 MB / ~2660 chains and changes rarely, so it is fetched once and
	// reused for chainsTTL rather than pulled on every keystroke — see
	// handleChainlistAll in chainlist.go.
	chainsMu    sync.Mutex
	chainsCache []chainSummary
	chainsAt    time.Time

	newExecutor   func(config.Target) (executor.Executor, error)
	newAIProvider func(id, apiKey, baseURL string) (ai.Provider, error)
	newChainlist  func() *chainlist.Discoverer
	verifyTLS     func(ctx context.Context, e executor.Executor, gatewayID string, g catalog.GatewayConfig, dialHost string) (setup.TLSVerification, error)
}

// New constructs a Server from the given Config.
func New(cfg Config) *Server {
	s := &Server{cfg: cfg, reg: newRegistry()}
	s.newExecutor = cfg.NewExecutor
	if s.newExecutor == nil {
		s.newExecutor = defaultNewExecutor
	}
	s.newAIProvider = cfg.NewAIProvider
	if s.newAIProvider == nil {
		s.newAIProvider = ai.New
	}
	s.newChainlist = cfg.NewChainlist
	if s.newChainlist == nil {
		s.newChainlist = chainlist.New
	}
	s.verifyTLS = cfg.VerifyTLS
	if s.verifyTLS == nil {
		s.verifyTLS = setup.VerifyGatewayTLS
	}
	return s
}

// NewSessionToken returns a new random session token: 16 bytes of
// crypto/rand, hex-encoded to 32 characters.
func NewSessionToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// Handler builds the server's http.Handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}` + "\n"))
	})

	s.registerAPIRoutes(mux)

	uiHandler := http.FileServerFS(s.cfg.UI)
	mux.Handle("/", uiHandler)

	return s.authMiddleware(mux)
}

// authMiddleware enforces the session token on every request. The token may
// arrive as an Authorization: Bearer header, a valve_node_token cookie, or a
// ?token= query parameter. A valid ?token= query parameter sets the cookie
// and redirects to the same path without the query parameter.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("token"); q != "" {
			if !tokensEqual(q, s.cfg.Token) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    q,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			http.Redirect(w, r, r.URL.Path, http.StatusFound)
			return
		}

		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if tok, ok := strings.CutPrefix(authHeader, "Bearer "); ok && tokensEqual(tok, s.cfg.Token) {
				next.ServeHTTP(w, r)
				return
			}
		}

		if c, err := r.Cookie(cookieName); err == nil && tokensEqual(c.Value, s.cfg.Token) {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// tokensEqual compares a caller-supplied token against the server's real
// session token in constant time (crypto/subtle.ConstantTimeCompare), so a
// wrong guess can't be distinguished by response-time from how many leading
// bytes happened to match — an ordinary `==` string compare short-circuits
// on the first mismatched byte and leaks that timing signal. Used for both
// the Authorization header and cookie auth paths.
func tokensEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ListenAndServe runs the server until ctx is canceled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:    s.cfg.Bind,
		Handler: s.Handler(),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return httpServer.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// cookiejarNew is a small helper wrapping net/http/cookiejar.New(nil), kept
// here so tests can create a jar without importing cookiejar directly.
func cookiejarNew() (*cookiejar.Jar, error) {
	return cookiejar.New(nil)
}
