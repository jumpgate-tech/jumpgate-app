// Package peercred reads the kernel's credentials for the peer of a unix
// socket connection. The gate's socket and the system-mode server's admin
// socket both admit peers by these, never by anything the peer sends.
package peercred

import (
	"os/user"
	"strconv"
)

// Cred is a socket peer's uid and groups. GIDs holds the primary group plus
// the supplementary groups from /etc/group on Linux, and the xucred groups
// on darwin.
type Cred struct {
	UID  int
	GIDs []int
}

// InGroup reports whether gid is one of the peer's groups.
func (c Cred) InGroup(gid int) bool {
	for _, g := range c.GIDs {
		if g == gid {
			return true
		}
	}
	return false
}

// GroupID is the id of the named group, and false when the group is absent.
func GroupID(name string) (int, bool) {
	g, err := user.LookupGroup(name)
	if err != nil {
		return -1, false
	}
	n, err := strconv.Atoi(g.Gid)
	if err != nil {
		return -1, false
	}
	return n, true
}
