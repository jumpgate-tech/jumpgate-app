//go:build darwin

package executor

import (
	"net"

	"golang.org/x/sys/unix"
)

func peerUIDOf(c net.Conn) (uint32, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return 0, errNotUnixConn
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, err
	}
	var x *unix.Xucred
	var serr error
	if err := raw.Control(func(fd uintptr) {
		x, serr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil {
		return 0, err
	}
	if serr != nil {
		return 0, serr
	}
	return x.Uid, nil
}
