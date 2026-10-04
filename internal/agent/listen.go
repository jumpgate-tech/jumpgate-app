package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// Listen opens the agent socket. A stale socket from a previous run is
// removed; any other file at the path is an error, never deleted. The socket
// gets mode (see SocketMode) and, when gid >= 0, is owned root:gid.
func Listen(path string, gid int, mode os.FileMode) (net.Listener, error) {
	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("agent: %s exists and is not a socket; refusing to remove it", path)
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, mode); err != nil {
		ln.Close()
		return nil, err
	}
	if gid >= 0 {
		if err := os.Chown(path, 0, gid); err != nil {
			ln.Close()
			return nil, err
		}
	}
	return ln, nil
}

// SocketMode is 0660 (root and group jumpgate, the SSH tunnel user) unless
// the policy enrolls a local uid, and then 0666. connect(2) needs write
// permission on the socket file, so at 0660 the kernel would refuse an
// enrolled uid outside the group before the Accept-time peer gate ever ran.
// That gate is the authority either way, and every intent still needs an
// enrolled controller's signature; remote-only boxes keep the group check
// as defence in depth.
func SocketMode(p Policy) os.FileMode {
	if len(p.LocalUIDs) > 0 {
		return 0o666
	}
	return 0o660
}

// ApplySocketMode re-chmods a live agent socket to match the policy, so an
// enroll takes effect without waiting for a restart. A missing socket (the
// agent is not running) is not an error; anything at path that is not a
// socket is refused and left alone.
func ApplySocketMode(path string, p Policy) error {
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("agent: %s is not a socket; leaving its mode alone", path)
	}
	return os.Chmod(path, SocketMode(p))
}

// JumpgateGID is the jumpgate group's id, or -1 when the group is absent.
func JumpgateGID() int {
	g, err := user.LookupGroup("jumpgate")
	if err != nil {
		return -1
	}
	n, err := strconv.Atoi(g.Gid)
	if err != nil {
		return -1
	}
	return n
}

// peerAllowed: the agent's own uid (root in production), a uid enrolled for
// local use, or any member of the jumpgate group. Signatures are still
// required for every intent; this gate only keeps strangers off the socket.
func peerAllowed(p Policy, uid int, gids []int, jumpgateGID int) bool {
	if uid == 0 || uid == os.Getuid() {
		return true
	}
	for _, u := range p.LocalUIDs {
		if u == uid {
			return true
		}
	}
	if jumpgateGID >= 0 {
		for _, g := range gids {
			if g == jumpgateGID {
				return true
			}
		}
	}
	return false
}

type peerKey struct{}

// peerGate decides whether a connection's peer may use the socket. It is a
// variable so a test can stand in for a second uid.
var peerGate = func(a *Agent, c net.Conn, jumpgateGID int) bool {
	uid, gids, err := peerCred(c)
	if err != nil {
		return false
	}
	p, _ := LoadPolicy(a.cfg.PolicyPath)
	return peerAllowed(p, uid, gids, jumpgateGID)
}

// gatedListener closes a refused peer's connection inside Accept, before
// net/http reads a byte of it, and hands only allowed connections on. A
// refusal is not an error: Accept moves on to the next connection, so one
// stranger cannot stop Serve.
type gatedListener struct {
	net.Listener
	allow func(net.Conn) bool
}

func (l gatedListener) Accept() (net.Conn, error) {
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

// Serve answers POST /v1/intent until ctx is done. Peers are gated at
// Accept; the handler checks again as defence in depth.
func Serve(ctx context.Context, a *Agent, ln net.Listener) error {
	gid := JumpgateGID()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/intent", func(w http.ResponseWriter, r *http.Request) {
		if allowed, _ := r.Context().Value(peerKey{}).(bool); !allowed {
			// Close rather than drain the body for keep-alive.
			w.Header().Set("Connection", "close")
			http.Error(w, "not allowed on this socket", http.StatusForbidden)
			return
		}
		// Read the whole capped body before decoding: a streaming decoder
		// stops at the first bad byte and would never notice the cap.
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "could not read the request", http.StatusBadRequest)
			return
		}
		var env intent.Envelope
		if err := json.Unmarshal(body, &env); err != nil {
			// An undecodable envelope still gets a signed rejection.
			env = intent.Envelope{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(a.Handle(r.Context(), env))
	})
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, peerKey{}, peerGate(a, c, gid))
		},
	}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	gated := gatedListener{Listener: ln, allow: func(c net.Conn) bool { return peerGate(a, c, gid) }}
	if err := srv.Serve(gated); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
