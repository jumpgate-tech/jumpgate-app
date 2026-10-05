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
