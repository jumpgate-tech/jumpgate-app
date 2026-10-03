//go:build darwin

package agent

import (
	"errors"
	"net"

	"golang.org/x/sys/unix"
)

// macOS is a development platform for the agent; production is Linux.
func peerCred(c net.Conn) (int, []int, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return 0, nil, errors.New("not a unix socket")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, nil, err
	}
	var x *unix.Xucred
	var serr error
	if err := raw.Control(func(fd uintptr) {
		x, serr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil {
		return 0, nil, err
	}
	if serr != nil {
		return 0, nil, serr
	}
	n := int(x.Ngroups)
	if n < 0 || n > len(x.Groups) {
		return 0, nil, errors.New("peer credentials report an impossible group count")
	}
	gids := make([]int, 0, n)
	for _, g := range x.Groups[:n] {
		gids = append(gids, int(g))
	}
	return int(x.Uid), gids, nil
}
