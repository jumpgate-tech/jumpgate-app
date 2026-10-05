//go:build windows

package executor

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// openSSHAgentPipe is where Windows OpenSSH's agent service, and 1Password's
// agent, listen. They are named pipes, and SSH_AUTH_SOCK is normally unset.
const openSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

// dialAgent connects to the agent: SSH_AUTH_SOCK when it is set, otherwise
// the OpenSSH pipe. A \\.\pipe\ path is a named pipe, opened for overlapped
// I/O so the dial deadline can cut off a wedged agent; any other value is a
// unix socket (MSYS2, Cygwin). Either way the server's identity is checked on
// the open connection before anything is sent to it.
func dialAgent(deadline time.Time) io.ReadWriteCloser {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		sock = openSSHAgentPipe
	}
	if !strings.HasPrefix(sock, `\\.\pipe\`) {
		return dialAgentSocket(sock, deadline)
	}
	p, err := openPipe(sock)
	if err != nil {
		return nil
	}
	// Whoever created the pipe receives every sign request. With the service
	// stopped, another local user could have created it first.
	if err := verifyPipeServer(p.h); err != nil {
		log.Printf("jumpgate: ssh-agent %v", err)
		p.Close()
		return nil
	}
	_ = p.SetDeadline(deadline)
	return p
}

func dialAgentSocket(sock string, deadline time.Time) io.ReadWriteCloser {
	c, err := (&net.Dialer{Deadline: deadline}).Dial("unix", sock)
	if err != nil {
		return nil
	}
	if err := verifySocketPeer(c); err != nil {
		log.Printf("jumpgate: ssh-agent at %s: %v", sock, err)
		c.Close()
		return nil
	}
	_ = c.SetDeadline(deadline)
	return c
}

// pipeOwnerSID is the owner of the pipe object, which needs only READ_CONTROL
// (part of GENERIC_READ): a non-admin can read it. Only SYSTEM,
// Administrators or the creating user can be recorded as owner, and the
// ssh-agent service's pipe is owned by SYSTEM or Administrators. A var so
// tests can inject it.
var pipeOwnerSID = func(h windows.Handle) (*windows.SID, error) {
	sd, err := windows.GetSecurityInfo(h, windows.SE_KERNEL_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return nil, err
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return nil, fmt.Errorf("no owner (%v)", err)
	}
	return owner, nil
}

// pipeServerSID is the fallback: the account of the process serving the pipe.
// It can need rights a non-admin lacks on a SYSTEM process.
var pipeServerSID = func(h windows.Handle) (*windows.SID, error) {
	var pid uint32
	if err := windows.GetNamedPipeServerProcessId(h, &pid); err != nil {
		return nil, err
	}
	return processUserSID(pid)
}

func processUserSID(pid uint32) (*windows.SID, error) {
	ph, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(ph)
	var tok windows.Token
	if err := windows.OpenProcessToken(ph, windows.TOKEN_QUERY, &tok); err != nil {
		return nil, err
	}
	defer tok.Close()
	u, err := tok.GetTokenUser()
	if err != nil {
		return nil, err
	}
	return u.User.Sid, nil
}

func verifyPipeServer(h windows.Handle) error {
	sid, err := pipeOwnerSID(h)
	if err != nil {
		if sid, err = pipeServerSID(h); err != nil {
			return fmt.Errorf("pipe owner unknown (%v); refusing", err)
		}
	}
	return checkAgentOwner(sid)
}

// siocAfUnixGetPeerPid is SIO_AF_UNIX_GETPEERPID, _WSAIOR(IOC_VENDOR, 256)
// (Windows 10 1809+).
const siocAfUnixGetPeerPid = 0x58000100

// socketPeerSID is the account of the process at the other end of an AF_UNIX
// socket; a var so tests can inject it.
var socketPeerSID = func(c net.Conn) (*windows.SID, error) {
	sc, ok := c.(interface {
		SyscallConn() (syscall.RawConn, error)
	})
	if !ok {
		return nil, fmt.Errorf("not a socket")
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return nil, err
	}
	var pid, ret uint32
	var ierr error
	if err := raw.Control(func(fd uintptr) {
		ierr = windows.WSAIoctl(windows.Handle(fd), siocAfUnixGetPeerPid, nil, 0,
			(*byte)(unsafe.Pointer(&pid)), 4, &ret, nil, 0)
	}); err != nil {
		return nil, err
	}
	if ierr != nil {
		return nil, ierr
	}
	return processUserSID(pid)
}

func verifySocketPeer(c net.Conn) error {
	sid, err := socketPeerSID(c)
	if err != nil {
		return fmt.Errorf("cannot identify the agent (%v); refusing", err)
	}
	return checkAgentOwner(sid)
}

// checkAgentOwner accepts LocalSystem and Administrators (the Windows
// ssh-agent service) and the current user (a user-run agent such as
// 1Password's or Pageant's).
func checkAgentOwner(sid *windows.SID) error {
	me, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	if sid.Equals(me.User.Sid) || sid.IsWellKnown(windows.WinLocalSystemSid) || sid.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
		return nil
	}
	return fmt.Errorf("pipe or socket is owned by %s; refusing", sid)
}
