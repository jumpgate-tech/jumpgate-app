package server

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/valve-tech/jumpgate/internal/peercred"
)

// peerCredKey carries a system-socket connection's peer credentials in its
// request contexts.
type peerCredKey struct{}

// admitPeer is the system socket's gate: root, the server's own uid, or a
// member of jumpgate-admin.
func admitPeer(c peercred.Cred, self, adminGID int) bool {
	if c.UID == 0 || c.UID == self {
		return true
	}
	return adminGID >= 0 && c.InGroup(adminGID)
}

// peerFrom is the peer credentials of a request that arrived on the system
// socket; false for any other request (TCP carries none).
func peerFrom(ctx context.Context) (peercred.Cred, bool) {
	c, ok := ctx.Value(peerCredKey{}).(peercred.Cred)
	return c, ok
}

// peerGatedListener closes a refused peer's connection inside Accept, before
// net/http reads a byte of it. A refusal is not an error, so one stranger
// cannot stop Serve.
type peerGatedListener struct {
	net.Listener
	allow func(net.Conn) bool
}

func (l peerGatedListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if l.allow(c) {
			return c, nil
		}
		c.Close()
	}
}

// ServeUnixSystem serves the handler on the jumpgate service's admin socket:
// 0660, group jumpgate-admin (Config.AdminGID). A peer is admitted by its
// kernel credentials (admitPeer), checked at Accept, and needs no token; the
// TCP listener still does. A stale socket is replaced, any other file is not.
func (s *Server) ServeUnixSystem(ctx context.Context, path string) error {
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
	if err := os.Chmod(path, 0o660); err != nil {
		ln.Close()
		return err
	}
	if err := os.Chown(path, -1, s.cfg.AdminGID); err != nil {
		ln.Close()
		return err
	}
	admit := func(c net.Conn) (peercred.Cred, bool) {
		cred, err := peercred.Of(c)
		if err != nil {
			return peercred.Cred{}, false
		}
		return cred, admitPeer(cred, s.selfUID, s.cfg.AdminGID)
	}
	srv := newHTTPServer("", s.Handler())
	srv.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		if cred, ok := admit(c); ok {
			return context.WithValue(ctx, peerCredKey{}, cred)
		}
		return ctx
	}
	gated := peerGatedListener{Listener: ln, allow: func(c net.Conn) bool { _, ok := admit(c); return ok }}
	go s.jobs.sweep(ctx) // Q4, as in ListenAndServe
	err = serveWith(ctx, srv, shutdownGrace, func() error { return srv.Serve(gated) })
	s.fleet.stopWithin(fleetStopWait)
	s.jobs.stopWithin(fleetStopWait)
	s.waitCritical()
	return err
}
