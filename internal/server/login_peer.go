package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

// socketUID finds, in a /proc/net/tcp or /proc/net/tcp6 table, the socket
// whose local end is local and remote end is remote, and returns its owner's
// uid. For a request, local is the client's address (the server's
// RemoteAddr) and remote the server's: that row is the client's socket, not
// the server's accepted one.
func socketUID(table []byte, v6 bool, local, remote netip.AddrPort) (int, bool) {
	l, r := procAddr(local, v6), procAddr(remote, v6)
	if l == "" || r == "" {
		return 0, false
	}
	sc := bufio.NewScanner(bytes.NewReader(table))
	sc.Scan() // header
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 8 || !strings.EqualFold(f[1], l) || !strings.EqualFold(f[2], r) {
			continue
		}
		uid, err := strconv.Atoi(f[7])
		return uid, err == nil
	}
	return 0, false
}

// procAddr formats ap as /proc/net/tcp{,6} prints it: each 32-bit word of
// the address in host byte order as hex, a colon, the port as hex. In the
// tcp6 table an IPv4 address appears v4-mapped.
func procAddr(ap netip.AddrPort, v6 bool) string {
	a := ap.Addr().Unmap()
	var b []byte
	switch {
	case v6:
		b16 := a.As16()
		b = b16[:]
	case a.Is4():
		b4 := a.As4()
		b = b4[:]
	default:
		return ""
	}
	var sb strings.Builder
	for i := 0; i < len(b); i += 4 {
		fmt.Fprintf(&sb, "%08X", binary.NativeEndian.Uint32(b[i:i+4]))
	}
	fmt.Fprintf(&sb, ":%04X", ap.Port())
	return sb.String()
}

// peerVerdict is what the socket tables say about a login request's client.
type peerVerdict int

const (
	// peerUnknown: the tables cannot be read at all (they do not exist, or
	// reading them is not permitted: a restricted /proc, some sandboxes).
	// Nothing can be decided; the login is allowed with a warning.
	peerUnknown peerVerdict = iota
	// peerFound: the client's socket was found; its uid decides.
	peerFound
	// peerMissing: the tables were readable but did not list the client's
	// socket, even on a second read. A live loopback connection is always
	// listed, so this is a torn or incomplete read, which another local user
	// can provoke by churning connections, and the login is refused.
	peerMissing
	// peerNotReported: both tables were read and neither has a single data
	// row. A real kernel lists an established connection, and a torn read of
	// a populated table still yields rows, so an attacker cannot force this:
	// the environment does not report connections (WSL1, gVisor). The login
	// is allowed with a warning (ruling P37).
	peerNotReported
	// peerNotTCP: the request did not come over TCP (the owner-only unix
	// socket has no peer address), so there is no row to look up and /proc
	// is not read. handleLogin refuses such requests before asking, as a
	// login link is for a browser; were one to get here, it is refused.
	peerNotTCP
)

// procTable is the contents of /proc/net/tcp (v6 false) or tcp6 (v6 true).
type procTable struct {
	data []byte
	v6   bool
}

// peerFromTables looks up a loopback connection's client socket (local end
// client, remote end local) in the tables read returns. Each call to read is
// one snapshot, both tables read whole, and each attempt decides from its
// own snapshot alone. The client's row may sit in either table (an IPv4
// client of a dual-stack listener is in tcp, the listener's accepted socket
// in tcp6, v4-mapped), so every table is searched.
//
// read returns nil only when the tables genuinely cannot be read; that, on
// the first read, is one case that is not refused; a first snapshot whose
// two tables are both header-only (peerNotReported) is the other. A readable snapshot
// that lacks the client's row is read once more, then refused; so is a
// re-read that finds the tables gone, since the first read proved this
// system lists its sockets. Nothing is remembered between calls.
func peerFromTables(read func() []procTable, client, local netip.AddrPort) (int, peerVerdict) {
	for attempt := 0; attempt < 2; attempt++ {
		tables := read()
		if tables == nil {
			if attempt == 0 {
				return 0, peerUnknown
			}
			break
		}
		for _, t := range tables {
			if uid, ok := socketUID(t.data, t.v6, client, local); ok {
				return uid, peerFound
			}
		}
		// Only a first snapshot of both tables, each header-only, says the
		// environment lists no connections. One table alone, or emptiness
		// that appears on the re-read after rows, is refused.
		if attempt == 0 && len(tables) == 2 && !hasDataRows(tables[0].data) && !hasDataRows(tables[1].data) {
			return 0, peerNotReported
		}
	}
	return 0, peerMissing
}

// hasDataRows reports whether a /proc/net/tcp{,6} table has any non-blank
// line after its header.
func hasDataRows(table []byte) bool {
	_, rest, _ := bytes.Cut(table, []byte("\n"))
	return len(bytes.TrimSpace(rest)) > 0
}

// requestAddrs is a request's client address (RemoteAddr) and the server's
// end of the connection (its LocalAddr).
func requestAddrs(r *http.Request) (client, local netip.AddrPort, ok bool) {
	client, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return client, local, false
	}
	la, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return client, local, false
	}
	local, err = netip.ParseAddrPort(la.String())
	return client, local, err == nil
}

// peerUIDFromTables is a peer lookup that consults the socket tables read
// returns (readProcTables on Linux).
func peerUIDFromTables(read func() []procTable) func(*http.Request) (int, peerVerdict) {
	return func(r *http.Request) (int, peerVerdict) {
		client, local, ok := requestAddrs(r)
		if !ok {
			return 0, peerNotTCP
		}
		return peerFromTables(read, client, local)
	}
}

// procSnapshot reads /proc/net/tcp and tcp6 with readFile as one snapshot,
// each table whole. It is nil, "cannot be read", only when every table is
// missing or not permitted. Any other error (EMFILE from a flood of
// connections, say) is not proof that /proc cannot be read: that table is
// left out of a non-nil snapshot, so a missing row fails closed.
func procSnapshot(readFile func(path string) ([]byte, error)) []procTable {
	tables := []procTable{}
	readable := false
	for _, t := range []struct {
		path string
		v6   bool
	}{{"/proc/net/tcp", false}, {"/proc/net/tcp6", true}} {
		data, err := readFile(t.path)
		switch {
		case err == nil:
			tables = append(tables, procTable{data: data, v6: t.v6})
			readable = true
		case errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission):
		default:
			readable = true
		}
	}
	if !readable {
		return nil
	}
	return tables
}
