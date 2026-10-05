//go:build windows

package executor

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// openSSHAgentPipe is where Windows OpenSSH's agent service, and 1Password's
// agent, listen. They are named pipes, and SSH_AUTH_SOCK is normally unset.
const openSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

// dialAgent connects to the agent: SSH_AUTH_SOCK when it is set, otherwise
// the OpenSSH pipe. A \\.\pipe\ path is opened as a file, whose handle is
// the io.ReadWriter agent.NewClient needs; any other value is a unix socket
// (MSYS2, Cygwin). A pipe handle opened this way has no deadline, so a
// wedged agent is bounded only by the SSH handshake deadline around it.
func dialAgent(deadline time.Time) io.ReadWriteCloser {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		sock = openSSHAgentPipe
	}
	if !strings.HasPrefix(sock, `\\.\pipe\`) {
		c, err := (&net.Dialer{Deadline: deadline}).Dial("unix", sock)
		if err != nil {
			return nil
		}
		_ = c.SetDeadline(deadline)
		return c
	}
	f, err := os.OpenFile(sock, os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	// Whoever created the pipe receives every sign request. With the service
	// stopped, another local user could have created it first, so the server
	// must be SYSTEM or this user.
	if err := verifyPipeServer(f); err != nil {
		log.Printf("jumpgate: ssh-agent %v", err)
		f.Close()
		return nil
	}
	return f
}

// pipeServerSID is the account running the process that serves the pipe; a
// var so tests can inject it.
var pipeServerSID = func(f *os.File) (*windows.SID, error) {
	var pid uint32
	if err := windows.GetNamedPipeServerProcessId(windows.Handle(f.Fd()), &pid); err != nil {
		return nil, err
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h)
	var tok windows.Token
	if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &tok); err != nil {
		return nil, err
	}
	defer tok.Close()
	u, err := tok.GetTokenUser()
	if err != nil {
		return nil, err
	}
	return u.User.Sid, nil
}

func verifyPipeServer(f *os.File) error {
	sid, err := pipeServerSID(f)
	if err != nil {
		return fmt.Errorf("pipe owner unknown (%v); refusing", err)
	}
	return checkAgentOwner(sid)
}

// checkAgentOwner accepts LocalSystem (the Windows ssh-agent service) and the
// current user (a user-run agent such as 1Password's or Pageant's).
func checkAgentOwner(sid *windows.SID) error {
	me, err := currentUserSID()
	if err != nil {
		return err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}
	if !sidIn(sid, []*windows.SID{me, system}) {
		return fmt.Errorf("pipe is owned by %s; refusing", sid)
	}
	return nil
}
