// Package server implements the token-gated local HTTP server that serves
// the embedded web UI and the JSON API for jumpgate.
package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valve-tech/jumpgate/internal/ai"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/chainlist"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/setup"
	"github.com/valve-tech/jumpgate/internal/updatecheck"
)

// cookieName is the name of the cookie that carries the session token once
// it has been established via the ?token= query parameter.
const cookieName = "jumpgate_token"

// Config configures a Server.
type Config struct {
	// Bind is the host:port the server listens on, e.g. "127.0.0.1:8799".
	Bind string
	// Token is the session token that authorizes API and UI requests.
	Token string
	// UI is the filesystem the static web UI is served from.
	UI fs.FS

	// Relay is the public data-plane handler — customer RPC traffic, keyed per
	// request. It is nil when the operator sells no keys.
	//
	// It is held here as a plain handler and served on its OWN listener by
	// ListenAndServeRelay. It never joins the mux Handler() builds, because
	// that mux is wrapped in authMiddleware and its token authorizes full
	// control of the operator's servers. A customer must never be one route
	// away from that.
	Relay http.Handler
	// Keys manages customer API keys against the billing store. Nil when the
	// operator sells no keys, which makes the key routes answer 501 rather than
	// dereference a nil client.
	Keys KeyAdmin
	// RelayBind is the host:port the data plane listens on. Bind it to the
	// interface Caddy reaches, never to 0.0.0.0: Caddy is the public door and
	// the TLS terminator, and a plaintext keyed URL would expose the key on the
	// wire.
	RelayBind string
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

	// Updater reads the latest published release for the update check.
	// Injectable for tests (a fake that never touches the network); nil
	// selects a real updatecheck.Client against buildinfo.ReleaseRepo().
	Updater updateSource

	// Shutdown, if set, is called by POST /api/shutdown. jumpgate serve wires
	// it to cancel its own context, so `jumpgate stop` never has to signal a
	// pid it cannot be sure of.
	Shutdown func()

	// NewLocalExecutor builds an executor for the local machine — the box the
	// Docker readiness gate probes. Injectable for tests (a fake that scripts
	// docker probes); nil selects executor.NewLocal.
	NewLocalExecutor func() executor.Executor
}

// Server is the jumpgate local HTTP server.
type Server struct {
	cfg Config

	// critical counts destructive operations in flight (clear, wipe). Shutdown
	// waits for them; see criticalOp.
	critical       sync.WaitGroup
	criticalActive atomic.Int64

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
	// capFlights is the probe currently running per gateway id, also guarded
	// by capMu. See sharedCapabilityProbe.
	capFlights map[string]*capFlight

	// capProbe and capTimeout replace the real probe and its deadline. They
	// are nil and zero outside tests, where a real probe dials real sockets.
	capProbe   func(context.Context, config.Config, config.Gateway) capabilitiesResponse
	capTimeout time.Duration

	// chainsMu guards the cached full chain catalogue (id + name for every
	// chain the feed knows) that backs the network-search picker. The feed is
	// ~1.1 MB / ~2660 chains and changes rarely, so it is fetched once and
	// reused for chainsTTL rather than pulled on every keystroke — see
	// handleChainlistAll in chainlist.go.
	chainsMu    sync.Mutex
	chainsCache []chainSummary
	chainsAt    time.Time

	newExecutor      func(config.Target) (executor.Executor, error)
	newAIProvider    func(id, apiKey, baseURL string) (ai.Provider, error)
	newChainlist     func() *chainlist.Discoverer
	newLocalExecutor func() executor.Executor
	verifyTLS        func(ctx context.Context, e executor.Executor, gatewayID string, g catalog.GatewayConfig, dialHost string) (setup.TLSVerification, error)

	// Update-check state, guarded by updMu. updCache is the last release read
	// from GitHub, updAt when it was read, updErr the last check's error text,
	// and updHasCache whether a check has run at all. See latestRelease for the
	// cache window (updateCheckInterval).
	updater     updateSource
	updMu       sync.Mutex
	updCache    updatecheck.Release
	updAt       time.Time
	updErr      string
	updHasCache bool

	// now is the server's clock, defaulting to time.Now. A test sets it to
	// drive the update-check cache window without waiting real hours.
	now func() time.Time
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
	s.updater = cfg.Updater
	if s.updater == nil {
		s.updater = updatecheck.New(buildinfo.ReleaseRepo())
	}
	s.newLocalExecutor = cfg.NewLocalExecutor
	if s.newLocalExecutor == nil {
		s.newLocalExecutor = executor.NewLocal
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

	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, struct {
			Version string `json:"version"`
		}{Version: buildinfo.Version()})
	})

	mux.HandleFunc("POST /api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Shutdown == nil {
			writeError(w, http.StatusNotImplemented, "this server cannot be stopped over the API")
			return
		}
		w.WriteHeader(http.StatusAccepted)
		go s.cfg.Shutdown()
	})

	s.registerAPIRoutes(mux)
	s.registerKeyRoutes(mux)

	uiHandler := http.FileServerFS(s.cfg.UI)
	mux.Handle("/", uiHandler)

	return s.authMiddleware(mux)
}

// authMiddleware enforces the session token on every request. The token may
// arrive as an Authorization: Bearer header, a jumpgate_token cookie, or a
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
			// SameSite ignores the port, so a page on another local port is
			// "same-site" and the browser sends it this cookie. A cookie may
			// therefore authorise a change only when the request also comes
			// from this origin. The bearer path above needs no such check: a
			// cross-origin page cannot set Authorization without a CORS
			// preflight, and this server grants none.
			if !isSafeMethod(r.Method) && !sameOrigin(r) {
				http.Error(w, "cross-origin request refused", http.StatusForbidden)
				return
			}
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
	err := serveUntil(ctx, newHTTPServer(s.cfg.Bind, s.Handler()), shutdownGrace)
	s.waitCritical()
	return err
}

// ServeUnix serves the same authenticated handler on a unix socket (0600), for
// the CLI and TUI on this machine. The token is still required: the socket's
// permissions keep other users out, the token keeps other programs of this
// user honest about which server they talk to. Shutdown is the same as
// ListenAndServe's, including the wait for destructive operations.
//
// A stale socket from a dead server is removed; any other file at path is not.
func (s *Server) ServeUnix(ctx context.Context, path string) error {
	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("server: %s exists and is not a socket", path)
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return err
	}
	srv := newHTTPServer("", s.Handler())
	err = serveWith(ctx, srv, shutdownGrace, func() error { return srv.Serve(ln) })
	s.waitCritical()
	return err
}

// criticalTimeout bounds one destructive operation. It is generous: a clear
// deletes hundreds of gigabytes, and stopping halfway is the outcome this
// exists to prevent.
const criticalTimeout = 15 * time.Minute

// criticalOp gives a destructive, multi-step operation a context its client
// cannot cancel, and registers it so shutdown waits for it.
//
// A clear stops a unit, deletes its data and starts it again. Run on the
// request's context, a closed tab or a Ctrl-C between those steps left the
// node stopped with half its data gone. Detached, the operation either
// finishes or hits criticalTimeout; a second Ctrl-C still kills the process
// outright, which is the operator's explicit choice.
func (s *Server) criticalOp(r *http.Request) (context.Context, func()) {
	s.critical.Add(1)
	s.criticalActive.Add(1)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), criticalTimeout)
	return ctx, func() {
		cancel()
		s.criticalActive.Add(-1)
		s.critical.Done()
	}
}

// waitCritical blocks until every destructive operation has finished.
func (s *Server) waitCritical() {
	if n := s.criticalActive.Load(); n > 0 {
		log.Printf("jumpgate: waiting for %d destructive operation(s) to finish; press Ctrl-C again to force", n)
	}
	s.critical.Wait()
}

// shutdownGrace bounds how long a listener waits for its requests to finish
// once it has been told to stop, before closing their connections anyway.
const shutdownGrace = 5 * time.Second

// serveUntil runs srv until ctx is canceled, then shuts it down within grace.
//
// http.Server.Shutdown waits for every active request to return, but it does
// not cancel their contexts, and the SSE handlers (setup, monitor, logs)
// return only when their request context ends. Left alone, Shutdown and an
// open UI tab would wait on each other forever, so Ctrl-C hung. Every request
// context therefore derives from a base context that is canceled the moment
// shutdown starts. That also cancels ordinary requests still in flight, which
// is what stopping the process means anyway.
//
// A handler that ignores its context would still pin Shutdown, so it is
// bounded by grace and followed by Close. That case is still a clean stop
// from the caller's side: the process was asked to exit, and it has.
func serveUntil(ctx context.Context, srv *http.Server, grace time.Duration) error {
	return serveWith(ctx, srv, grace, srv.ListenAndServe)
}

// serveWith is serveUntil for any way of serving: serve blocks until srv stops,
// and is srv.ListenAndServe or srv.Serve on a listener the caller prepared.
func serveWith(ctx context.Context, srv *http.Server, grace time.Duration, serve func() error) error {
	base, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()
	srv.BaseContext = func(net.Listener) context.Context { return base }

	errCh := make(chan error, 1)
	go func() {
		errCh <- serve()
	}()

	select {
	case <-ctx.Done():
		cancelBase()
		sctx, cancel := context.WithTimeout(context.Background(), grace)
		defer cancel()
		if err := srv.Shutdown(sctx); err != nil {
			srv.Close()
		}
		return nil
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

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// sameOrigin reports whether a browser request came from this server's own
// origin. Fetch metadata is preferred when present; otherwise Origin must name
// this host. A request carrying neither is not from a browser page (browsers
// always send Origin on a cross-origin POST), so it is allowed.
func sameOrigin(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "none":
		// "none" is a user-initiated navigation, not a page.
	case "":
		// Older browser or not a browser: fall through to Origin.
	default:
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		// Includes "null", which sandboxed and file:// pages send.
		return false
	}
	return u.Host == r.Host
}

// Listener timeouts. Header reads are bounded so a slow-header client cannot
// pin sockets open. There is deliberately no WriteTimeout: SSE and WebSocket
// responses are long-lived, and a whole-response deadline would cut them off.
const (
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 2 * time.Minute
)

// newHTTPServer builds a listener's http.Server with the shared timeouts.
func newHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
}
