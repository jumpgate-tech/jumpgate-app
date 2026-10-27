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
	if s.cfg.Relay != nil {
		return s.cfg.Relay
	}
	s.relayMu.RLock()
	defer s.relayMu.RUnlock()
	return s.relayH
}

// keyStore is Config.Keys, or the key admin AttachRelay installed; nil
// when the operator sells no keys.
func (s *Server) keyStore() KeyAdmin {
	if s.cfg.Keys != nil {
		return s.cfg.Keys
	}
	s.relayMu.RLock()
	defer s.relayMu.RUnlock()
	return s.relayKeys
}

// AttachRelay installs a relay handler and key admin built after start,
// once the vault that holds their tokens is unlocked. It may be called once,
// and only when Config.Relay and Config.Keys are nil. The caller then runs
// ListenAndServeRelay.
func (s *Server) AttachRelay(h http.Handler, keys KeyAdmin) error {
	s.relayMu.Lock()
	defer s.relayMu.Unlock()
	if s.cfg.Relay != nil || s.cfg.Keys != nil {
		return errors.New("server: a relay was configured at start")
	}
	if s.relayAttached {
		return errors.New("server: a relay is already attached")
	}
	s.relayAttached, s.relayH, s.relayKeys = true, h, keys
	return nil
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
	return ServeRelay(ctx, s.cfg.RelayBind, handler)
}

// ServeRelay runs a relay handler on its own listener until ctx is canceled,
// with the same header and idle timeouts and graceful shutdown as the control
// plane. It needs no Server: `jumpgate relay` serves the data plane with it
// and never builds a controller, so it holds no config, key or executor.
func ServeRelay(ctx context.Context, bind string, handler http.Handler) error {
	return serveUntil(ctx, newHTTPServer(bind, handler), shutdownGrace)
}
