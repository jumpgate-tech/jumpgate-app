//go:build windows

package executor

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// I-5, end to end: a key loaded into the Windows OpenSSH agent service
// authenticates an SSH dial with no key file. CI starts the service and sets
// JUMPGATE_TEST_WIN_AGENT=1; elsewhere this skips.
func TestOpenSSHAgentPipe(t *testing.T) {
	if os.Getenv("JUMPGATE_TEST_WIN_AGENT") != "1" {
		t.Skip("set JUMPGATE_TEST_WIN_AGENT=1 with the ssh-agent service running")
	}
	t.Setenv("SSH_AUTH_SOCK", "")
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("ssh-add", keyFile).CombinedOutput(); err != nil {
		t.Fatalf("ssh-add: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("ssh-add", "-d", keyFile).Run() })
	sshPub, _ := ssh.NewPublicKey(pub)

	c := dialAgent(time.Now().Add(5 * time.Second))
	if c == nil {
		t.Fatal("dialAgent: no agent on the OpenSSH pipe")
	}
	keys, err := agent.NewClient(c).List()
	c.Close()
	found := false
	for _, k := range keys {
		found = found || string(k.Blob) == string(sshPub.Marshal())
	}
	if err != nil || !found {
		t.Fatalf("the added key is not listed: %v, %v", keys, err)
	}

	srv := &gliderssh.Server{
		PublicKeyHandler: func(_ gliderssh.Context, k gliderssh.PublicKey) bool { return gliderssh.KeysEqual(k, sshPub) },
		Handler:          func(s gliderssh.Session) { _, _ = io.WriteString(s, "ok"); _ = s.Exit(0) },
	}
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)
	srv.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	deadline := time.Now().Add(10 * time.Second)
	methods, release, err := authMethods(SSHConfig{}, deadline)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	client, err := ssh.Dial("tcp", ln.Addr().String(), &ssh.ClientConfig{
		User: "u", Auth: methods, HostKeyCallback: ssh.FixedHostKey(hostSigner.PublicKey()), Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial with the agent's key: %v", err)
	}
	defer client.Close()
	sess, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	out, err := sess.Output("anything")
	if err != nil || string(out) != "ok" {
		t.Fatalf("session = %q, %v", out, err)
	}
}
