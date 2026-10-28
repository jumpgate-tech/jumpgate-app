//go:build linux

package peercred

import (
	"errors"
	"net"
	"os/user"
	"strconv"

	"golang.org/x/sys/unix"
)

func Of(c net.Conn) (Cred, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return Cred{}, errors.New("not a unix socket")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return Cred{}, err
	}
	var cred *unix.Ucred
	var serr error
	if err := raw.Control(func(fd uintptr) {
		cred, serr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return Cred{}, err
	}
	if serr != nil {
		return Cred{}, serr
	}
	return Cred{UID: int(cred.Uid), GIDs: groupsOf(int(cred.Uid), int(cred.Gid))}, nil
}

// groupsOf lists a uid's primary and supplementary groups from /etc/group
// (pure Go: the agent is built without cgo).
func groupsOf(uid, primary int) []int {
	gids := []int{primary}
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return gids
	}
	ids, err := u.GroupIds()
	if err != nil {
		return gids
	}
	for _, s := range ids {
		if n, err := strconv.Atoi(s); err == nil {
			gids = append(gids, n)
		}
	}
	return gids
}
