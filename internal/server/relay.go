package server

import (
	"context"
	"errors"
	"net/http"
)

// The data plane is a second listener with its own handler and its own bind.
//
// It is a separate listener rather than a route group on the existing mux
// because that is what makes the auth boundary a fact of the wiring. The
// control plane's session token authorises full control of the operator's
// servers. A customer holds a key, which authorises one chain's traffic and
// nothing else. Sharing a mux would put those two credentials one forgotten
// route apart, and a later edit could bridge them without anyone noticing.
//
// The relay handler itself lives in internal/relay. This file only wires it.

// RelayHandler returns the data-plane handler, or nil when the operator sells
// no keys. A gateway with no metering needs no data plane, and the absence must
// not be a nil dereference at startup.
func (s *Server) RelayHandler() http.Handler {
	if s.cfg.Relay == nil {
		return nil
	}
	return s.cfg.Relay
}

// ListenAndServeRelay runs the data plane until ctx is canceled. It is a
// sibling of ListenAndServe rather than part of it, so a failure on one plane
// is reported as that plane's failure.
//
// It returns nil immediately when no relay is configured, so a caller can start
// it unconditionally.
func (s *Server) ListenAndServeRelay(ctx context.Context) error {
	handler := s.RelayHandler()
	if handler == nil {
		return nil
	}
	if s.cfg.RelayBind == "" {
		return errors.New("server: a relay is configured with no bind address")
	}

	// No authMiddleware. The data plane authenticates by key, inside the relay
	// handler. Wrapping it here would let a session token buy customer traffic,
	// and worse, would suggest the session token belongs on this plane at all.
	relayServer := &http.Server{
		Addr:    s.cfg.RelayBind,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- relayServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return relayServer.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
