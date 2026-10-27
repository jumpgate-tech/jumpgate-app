package executor

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// A host that accepts TCP and never speaks SSH must not hang the caller.
func TestDialSSHHasAHandshakeDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close() // hold it open, say nothing
		}
	}()

	old := handshakeTimeout
	handshakeTimeout = 300 * time.Millisecond
	defer func() { handshakeTimeout = old }()

	addr := ln.Addr().(*net.TCPAddr)
	start := time.Now()
	_, err = DialSSH(context.Background(), SSHConfig{
		Host: "127.0.0.1", Port: addr.Port, User: "x",
		KeyPath: writeTempKey(t), HostKey: ssh.InsecureIgnoreHostKey(),
	})
	if err == nil {
		t.Fatal("handshake against a silent server succeeded")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("took %v; the handshake deadline did not apply", time.Since(start))
	}
}

// With SSH_AUTH_SOCK set and no key file, the agent's keys authenticate. This
// is how passphrase-protected keys are supported.
func TestDialSSHUsesTheSSHAgent(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyring := agent.NewKeyring()
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(shortTempDir(t), "agent.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(keyring, c)
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", sock)

	signer, _ := ssh.NewSignerFromKey(priv)
	d := startTestSSHDAcceptingOnly(t, signer.PublicKey())
	client, err := DialSSH(context.Background(), SSHConfig{
		Host: d.host, Port: d.port, User: "x", HostKey: ssh.FixedHostKey(d.hostKey),
	})
	if err != nil {
		t.Fatalf("DialSSH via agent: %v", err)
	}
	client.Close()
}

// A jump host is dialled first and the target is reached through it.
func TestDialSSHThroughAJumpHost(t *testing.T) {
	jump, keyPath := startTestSSHDWithForwarding(t)
	target, _ := startTestSSHD(t)
	client, err := DialSSH(context.Background(), SSHConfig{
		Host: target.host, Port: target.port, User: "x", KeyPath: keyPath,
		HostKey: ssh.FixedHostKey(target.hostKey),
		Jump: &SSHConfig{Host: jump.host, Port: jump.port, User: "x", KeyPath: keyPath,
			HostKey: ssh.FixedHostKey(jump.hostKey)},
	})
	if err != nil {
		t.Fatalf("DialSSH via jump: %v", err)
	}
	client.Close()
}

func shortTempDir(t *testing.T) string {
	t.Helper()
	return testutil.ShortTempDir(t)
}

func shortHandshakeTimeout(t *testing.T) {
	t.Helper()
	old := handshakeTimeout
	handshakeTimeout = 300 * time.Millisecond
	t.Cleanup(func() { handshakeTimeout = old })
}

// listenAndStall accepts TCP connections and never speaks.
func listenAndStall(t *testing.T, network, addr string) net.Listener {
	t.Helper()
	ln, err := net.Listen(network, addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { c.Close() })
		}
	}()
	return ln
}

// A target behind a jump host that accepts the forwarded channel and then
// stalls must still hit the handshake deadline: the forwarded channel's
// SetDeadline is unsupported, so the deadline cannot rely on it.
func TestDialSSHHandshakeDeadlineAppliesThroughAJumpHost(t *testing.T) {
	shortHandshakeTimeout(t)
	jump, keyPath := startTestSSHDWithForwarding(t)
	silent := listenAndStall(t, "tcp", "127.0.0.1:0").Addr().(*net.TCPAddr)

	start := time.Now()
	_, err := DialSSH(context.Background(), SSHConfig{
		Host: "127.0.0.1", Port: silent.Port, User: "x", KeyPath: keyPath,
		HostKey: ssh.InsecureIgnoreHostKey(),
		Jump: &SSHConfig{Host: jump.host, Port: jump.port, User: "x", KeyPath: keyPath,
			HostKey: ssh.FixedHostKey(jump.hostKey)},
	})
	if err == nil {
		t.Fatal("handshake against a silent target succeeded")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("took %v; the handshake deadline did not apply through the jump host", time.Since(start))
	}
}

// Cancelling the context aborts a stalled handshake on the direct path.
func TestDialSSHContextCancelAbortsTheHandshake(t *testing.T) {
	silent := listenAndStall(t, "tcp", "127.0.0.1:0").Addr().(*net.TCPAddr)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(200 * time.Millisecond); cancel() }()
	start := time.Now()
	_, err := DialSSH(ctx, SSHConfig{
		Host: "127.0.0.1", Port: silent.Port, User: "x",
		KeyPath: writeTempKey(t), HostKey: ssh.InsecureIgnoreHostKey(),
	})
	if err == nil {
		t.Fatal("handshake succeeded")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("took %v; cancel did not abort the handshake", time.Since(start))
	}
}

// An ssh-agent that accepts and never answers must not hold DialSSH past the
// handshake deadline.
func TestDialSSHBoundsAWedgedAgent(t *testing.T) {
	shortHandshakeTimeout(t)
	sock := filepath.Join(shortTempDir(t), "agent.sock")
	listenAndStall(t, "unix", sock)
	t.Setenv("SSH_AUTH_SOCK", sock)

	d := startTestSSHDAcceptingOnly(t, mustPublicKey(t))
	start := time.Now()
	_, err := DialSSH(context.Background(), SSHConfig{
		Host: d.host, Port: d.port, User: "x", HostKey: ssh.FixedHostKey(d.hostKey),
	})
	if err == nil {
		t.Fatal("DialSSH succeeded with a wedged agent")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("took %v; a wedged agent was not bounded", time.Since(start))
	}
}

func mustPublicKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// A configured in-memory signer is offered, and KeyPath is never read.
func TestAuthMethodsPrefersAConfiguredSigner(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	methods, release, err := authMethods(SSHConfig{Signer: s, KeyPath: "/nonexistent"}, time.Now().Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	_, hpriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hpriv)
	want := string(s.PublicKey().Marshal())
	scfg := &ssh.ServerConfig{PublicKeyCallback: func(_ ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
		if string(k.Marshal()) == want {
			return nil, nil
		}
		return nil, errors.New("unknown key")
	}}
	scfg.AddHostKey(hostSigner)
	// A loopback socket, not net.Pipe: both ends write their version line
	// at once, and an unbuffered pipe deadlocks on that.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		sc, err := ln.Accept()
		if err != nil {
			return
		}
		defer sc.Close()
		if conn, _, _, err := ssh.NewServerConn(sc, scfg); err == nil {
			conn.Close()
		}
	}()
	cc, err := net.DialTimeout("tcp", ln.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer cc.Close()
	conn, _, _, err := ssh.NewClientConn(cc, "pipe", &ssh.ClientConfig{
		User: "jumpgate", Auth: methods, HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("the configured signer was not offered: %v", err)
	}
	conn.Close()
}
