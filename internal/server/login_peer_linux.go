package server

import (
	"errors"
	"io"
	"io/fs"
	"os"
)

// defaultPeerUID reads the owner of a loopback request's client socket from
// /proc/net/tcp{,6}. Loopback TCP carries no peer credentials, but both ends
// of the connection are this machine's sockets, listed there with their uid.
// It fails closed where it can: see peerFromTables.
var defaultPeerUID = peerUIDFromTables(readProcTables)

// readProcTables reads /proc/net/tcp and tcp6 as one snapshot, each read
// whole into memory before any parsing. It is nil, "cannot be read", only
// when every table is missing or not permitted. Any other error (EMFILE from
// a flood of connections, say) is not proof that /proc cannot be read: that
// table is left out of a non-nil snapshot, so a missing row fails closed.
func readProcTables() []procTable {
	tables := []procTable{}
	readable := false
	for _, t := range []struct {
		path string
		v6   bool
	}{{"/proc/net/tcp", false}, {"/proc/net/tcp6", true}} {
		data, err := readProcFile(t.path)
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

// procReadBuf is the starting buffer for a /proc table: room for thousands
// of rows (about 150 bytes each), so a busy table is read with large reads
// into one buffer rather than grown through many small ones. The kernel
// still produces the table a page or so per read(2), which is why a missing
// row is re-read and then refused rather than trusted (peerFromTables).
const procReadBuf = 1 << 20

// readProcFile reads path to EOF. /proc files report size 0, so the size
// cannot be known up front; the buffer grows as needed.
func readProcFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, 0, procReadBuf)
	for {
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}
		n, err := f.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		if err == io.EOF {
			return buf, nil
		}
		if err != nil {
			return nil, err
		}
	}
}
