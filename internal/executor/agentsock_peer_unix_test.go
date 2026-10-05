//go:build !windows

package executor

import (
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// The decision reads the connected socket's peer (checkAgentPeer takes a
// net.Conn, not a path), so there is no path to swap after the check.
func TestCheckAgentPeerDecision(t *testing.T) {
	old := peerUID
	t.Cleanup(func() { peerUID = old })
	for _, c := range []struct {
		name      string
		peer, me  uint32
		wantRefus string
	}{
		{"same uid", 1000, 1000, ""},
		{"other uid", 1001, 1000, "uid 1001, not you"},
		{"root peer, non-root me", 0, 1000, "uid 0, not you"},
		{"root me, user peer", 1000, 0, "uid 1000, not you"},
		{"root both", 0, 0, ""},
	} {
		peerUID = func(net.Conn) (uint32, error) { return c.peer, nil }
		err := checkAgentPeer(nil, c.me)
		if c.wantRefus == "" && err != nil || c.wantRefus != "" && (err == nil || !strings.Contains(err.Error(), c.wantRefus)) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	peerUID = func(net.Conn) (uint32, error) { return 0, errors.New("boom") }
	if checkAgentPeer(nil, 1000) == nil {
		t.Error("an unidentifiable peer was accepted")
	}
}

// Real credentials: a listener in this process passes; with the euid seam
// shifted it is refused, and dialAgent hands back no agent.
func TestRealSocketPeerCredentials(t *testing.T) {
	serveTestAgent(t)
	c := dialAgent(deadlineSoon())
	if c == nil {
		t.Fatal("an agent run by this user was refused")
	}
	c.Close()
	uc, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		t.Fatal(err)
	}
	defer uc.Close()
	if err := checkAgentPeer(uc, uint32(os.Geteuid())+1); err == nil {
		t.Fatal("a peer with a different uid was accepted")
	}
}

func deadlineSoon() time.Time { return time.Now().Add(2 * time.Second) }
