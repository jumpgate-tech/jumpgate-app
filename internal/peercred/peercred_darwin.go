//go:build darwin

package peercred

import (
	"errors"
	"net"

	"golang.org/x/sys/unix"
)

// macOS is a development platform; production is Linux.
func Of(c net.Conn) (Cred, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return Cred{}, errors.New("not a unix socket")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return Cred{}, err
	}
	var x *unix.Xucred
	var serr error
	if err := raw.Control(func(fd uintptr) {
		x, serr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil {
		return Cred{}, err
	}
	if serr != nil {
		return Cred{}, serr
	}
	n := int(x.Ngroups)
	if n < 0 || n > len(x.Groups) {
		return Cred{}, errors.New("peer credentials report an impossible group count")
	}
	gids := make([]int, 0, n)
	for _, g := range x.Groups[:n] {
		gids = append(gids, int(g))
	}
	return Cred{UID: int(x.Uid), GIDs: gids}, nil
}
