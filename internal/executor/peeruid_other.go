//go:build !windows && !linux && !darwin

package executor

import (
	"errors"
	"net"
)

// Other unixes are not supported targets; failing closed means no agent.
func peerUIDOf(net.Conn) (uint32, error) {
	return 0, errors.New("peer credentials are not implemented on this OS")
}
