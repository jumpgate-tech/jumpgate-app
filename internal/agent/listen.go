package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
// is 0660 and, when gid >= 0, owned root:gid, so only root and the jumpgate
// group (the SSH tunnel user) can connect.
func Listen(path string, gid int) (net.Listener, error) {
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
	if err := os.Chmod(path, 0o660); err != nil {
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

// Serve answers POST /v1/intent until ctx is done.
func Serve(ctx context.Context, a *Agent, ln net.Listener) error {
	gid := JumpgateGID()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/intent", func(w http.ResponseWriter, r *http.Request) {
		if allowed, _ := r.Context().Value(peerKey{}).(bool); !allowed {
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
			uid, gids, err := peerCred(c)
			if err != nil {
				return context.WithValue(ctx, peerKey{}, false)
			}
			p, _ := LoadPolicy(a.cfg.PolicyPath)
			return context.WithValue(ctx, peerKey{}, peerAllowed(p, uid, gids, gid))
		},
	}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
