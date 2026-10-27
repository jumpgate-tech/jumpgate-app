//go:build !linux && !darwin

package peercred

import (
	"errors"
	"net"
)

// Jumpgate runs on Linux. Elsewhere every peer is refused.
func Of(c net.Conn) (Cred, error) {
	if _, ok := c.(*net.UnixConn); !ok {
		return Cred{}, errors.New("not a unix socket")
	}
	return Cred{}, errors.New("peer credentials unsupported here")
}
