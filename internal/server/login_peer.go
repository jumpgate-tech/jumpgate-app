package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
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
	// peerUnknown: the tables could not be read, or they do not list even
	// the server's own end of the connection (WSL1, gVisor, a restricted
	// /proc). Nothing can be decided; the login is allowed with a warning.
	peerUnknown peerVerdict = iota
	// peerFound: the client's socket was found; its uid decides.
	peerFound
	// peerMissing: the server's own end is listed but the client's is not,
	// even on a second read. A live loopback connection has both ends in
	// the tables, so this is a torn read an attacker can provoke by churning
	// connections, and the login is refused.
	peerMissing
)

// procTable is the contents of /proc/net/tcp (v6 false) or tcp6 (v6 true).
type procTable struct {
	data []byte
	v6   bool
}

// peerFromTables looks up a loopback connection in the socket tables that
// read returns: the client's socket (local end client, remote end local) and
// the server's accepted one (the two swapped). Either may sit in either
// table: a dual-stack listener's accepted socket is listed in tcp6,
// v4-mapped, while an IPv4 client's is in tcp. When only the server's row
// turns up, the tables are read once more before the client is declared
// missing; read returning no tables at all means they are unreadable.
func peerFromTables(read func() []procTable, client, local netip.AddrPort) (int, peerVerdict) {
	for attempt := 0; attempt < 2; attempt++ {
		tables := read()
		serverSeen := false
		for _, t := range tables {
			if uid, ok := socketUID(t.data, t.v6, client, local); ok {
				return uid, peerFound
			}
			if _, ok := socketUID(t.data, t.v6, local, client); ok {
				serverSeen = true
			}
		}
		// Only a first read that lacks the server's row is inconclusive. A
		// second read happens only once the tables proved they list this
		// connection, so a miss of both rows then is as torn as a miss of one.
		if !serverSeen && attempt == 0 {
			return 0, peerUnknown
		}
	}
	return 0, peerMissing
}
