package server

import (
	"net"
	"net/http"
	"net/netip"
	"os"
)

// defaultPeerUID reads the owner of a loopback request's client socket from
// /proc/net/tcp{,6}. Loopback TCP carries no peer credentials, but both ends
// of the connection are this machine's sockets, listed there with their uid.
// Best effort: anything not found reports ok=false and the caller allows it.
var defaultPeerUID = procPeerUID

func procPeerUID(r *http.Request) (int, bool) {
	peer, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return 0, false
	}
	la, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return 0, false
	}
	local, err := netip.ParseAddrPort(la.String())
	if err != nil {
		return 0, false
	}
	for _, t := range []struct {
		path string
		v6   bool
	}{{"/proc/net/tcp", false}, {"/proc/net/tcp6", true}} {
		data, err := os.ReadFile(t.path)
		if err != nil {
			continue
		}
		if uid, ok := socketUID(data, t.v6, peer, local); ok {
			return uid, true
		}
	}
	return 0, false
}
