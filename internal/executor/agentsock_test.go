package executor

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// serveTestAgent runs an in-memory ssh-agent on a unix socket (which Windows
// supports too, as MSYS and Cygwin agents use) and returns its public key.
func serveTestAgent(t *testing.T) ssh.PublicKey {
	t.Helper()
	sock := filepath.Join(testutil.ShortTempDir(t), "agent.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyring := agent.NewKeyring()
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); _ = agent.ServeAgent(keyring, c) }()
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", sock)
	pub, _ := ssh.NewPublicKey(priv.Public())
	return pub
}

func TestDialAgentReachesAUnixSocketAgent(t *testing.T) {
	pub := serveTestAgent(t)
	if !AgentAvailable() {
		t.Fatal("AgentAvailable() = false with a live agent")
	}
	c := dialAgent(time.Now().Add(2 * time.Second))
	if c == nil {
		t.Fatal("dialAgent returned nil")
	}
	defer c.Close()
	keys, err := agent.NewClient(c).List()
	if err != nil || len(keys) != 1 || string(keys[0].Blob) != string(pub.Marshal()) {
		t.Fatalf("List = %v, %v", keys, err)
	}
}

func TestAgentAvailableIsFalseForADeadSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(testutil.ShortTempDir(t), "missing.sock"))
	if AgentAvailable() {
		t.Fatal("AgentAvailable() = true for a socket that does not exist")
	}
}
