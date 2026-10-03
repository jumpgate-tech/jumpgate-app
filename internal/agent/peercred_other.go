//go:build !linux && !darwin

package agent

import (
	"errors"
	"net"
)

// The agent runs on Linux. Elsewhere every peer is refused.
func peerCred(net.Conn) (int, []int, error) {
	return 0, nil, errors.New("peer credentials unsupported here")
}
