//go:build !windows

package executor

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
	"time"
)

// dialAgent connects to the ssh-agent named by SSH_AUTH_SOCK, or returns nil
// when none is configured, reachable or run by someone else: the agent is
// optional.
func dialAgent(deadline time.Time) io.ReadWriteCloser {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	c, err := (&net.Dialer{Deadline: deadline}).Dial("unix", sock)
	if err != nil {
		return nil
	}
	// Check the peer of the connected socket, never the path: a path can be
	// swapped between a check and the connect, the connection cannot.
	if err := checkAgentPeer(c, uint32(os.Geteuid())); err != nil {
		log.Printf("jumpgate: ssh-agent at %s: %v", sock, err)
		c.Close()
		return nil
	}
	_ = c.SetDeadline(deadline) // a wedged agent cannot outlast the dial
	return c
}

// peerUID reports the uid of the process at the other end of c; a var so
// tests can inject it.
var peerUID = peerUIDOf

// checkAgentPeer accepts c only when its peer runs as euid. Root counts only
// when jumpgate itself is root: a root-run agent is not trusted by a user, and
// under sudo the user's agent is refused (use the key file, or run as the user).
func checkAgentPeer(c net.Conn, euid uint32) error {
	uid, err := peerUID(c)
	if err != nil {
		return fmt.Errorf("cannot identify the agent (%v); refusing", err)
	}
	if uid != euid {
		if euid == 0 {
			// sudo keeps the invoking user's SSH_AUTH_SOCK, but root does not
			// trust an agent a user runs: a user could point it anywhere.
			return fmt.Errorf("is run by %s, not root; running under sudo? the agent belongs to %s; refusing", uidName(uid), uidName(uid))
		}
		return fmt.Errorf("is run by %s, not you; refusing", uidName(uid))
	}
	return nil
}

func uidName(uid uint32) string {
	if u, err := user.LookupId(strconv.Itoa(int(uid))); err == nil {
		return fmt.Sprintf("%s (uid %d)", u.Username, uid)
	}
	return fmt.Sprintf("uid %d", uid)
}

var errNotUnixConn = errors.New("not a unix socket")
