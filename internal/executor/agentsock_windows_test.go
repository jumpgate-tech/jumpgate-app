//go:build windows

package executor

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/sys/windows"
)

// I-5, end to end: a key loaded into the Windows OpenSSH agent service
// authenticates an SSH dial with no key file. CI starts the service and sets
// JUMPGATE_TEST_WIN_AGENT=1; elsewhere this skips.
func TestOpenSSHAgentPipe(t *testing.T) {
	if os.Getenv("JUMPGATE_TEST_WIN_AGENT") != "1" {
		t.Skip("set JUMPGATE_TEST_WIN_AGENT=1 with the ssh-agent service running")
	}
	t.Setenv("SSH_AUTH_SOCK", "")
	// Git for Windows puts its own ssh-add first on PATH; it talks to an MSYS
	// agent, not the service's pipe.
	sshAdd := `C:\Windows\System32\OpenSSH\ssh-add.exe`
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := addCmd(sshAdd, keyFile).CombinedOutput(); err != nil {
		t.Fatalf("ssh-add: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = addCmd(sshAdd, "-d", keyFile).Run() })
	sshPub, _ := ssh.NewPublicKey(pub)

	t.Log("ssh-add done; dialling the pipe")
	c := dialAgent(time.Now().Add(5 * time.Second))
	if c == nil {
		t.Fatal("dialAgent: no agent on the OpenSSH pipe")
	}
	t.Log("dialled; listing keys")
	listed := make(chan struct{})
	var keys []*agent.Key
	go func() { keys, err = agent.NewClient(c).List(); close(listed) }()
	select {
	case <-listed:
	case <-time.After(10 * time.Second):
		t.Fatal("agent List hung on the pipe")
	}
	c.Close()
	// Sign directly, timed, to tell the agent apart from the SSH handshake.
	sc := dialAgent(time.Now().Add(8 * time.Second))
	if sc == nil {
		t.Fatal("dialAgent for Sign: no agent")
	}
	signed := make(chan error, 1)
	go func() {
		_, err := agent.NewClient(sc).Sign(sshPub, []byte("data"))
		signed <- err
	}()
	select {
	case err := <-signed:
		if err != nil {
			t.Fatalf("agent.Sign: %v", err)
		}
	case <-time.After(10 * time.Second):
		sc.Close()
		t.Fatal("agent.Sign hung on the pipe")
	}
	sc.Close()
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

	t.Log("listed; dialling ssh through the agent")
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

func sidOf(t *testing.T, k windows.WELL_KNOWN_SID_TYPE) *windows.SID {
	t.Helper()
	s, err := windows.CreateWellKnownSid(k)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The agent-owner check accepts SYSTEM and this user, and not Administrators.
func TestCheckAgentOwner(t *testing.T) {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		sid *windows.SID
		ok  bool
	}{
		"me":       {u.User.Sid, true},
		"system":   {sidOf(t, windows.WinLocalSystemSid), true},
		"admins":   {sidOf(t, windows.WinBuiltinAdministratorsSid), false},
		"everyone": {sidOf(t, windows.WinWorldSid), false},
	} {
		if err := checkAgentOwner(c.sid); (err == nil) != c.ok {
			t.Errorf("%s: checkAgentOwner(%s) = %v, want ok=%v", name, c.sid, err, c.ok)
		}
	}
}

// Ruling P36: the pipe's owner decides, and Administrators only passes when
// the serving process is SYSTEM or the user.
func TestVerifyPipeServerDecisionTable(t *testing.T) {
	u, _ := windows.GetCurrentProcessToken().GetTokenUser()
	me, system := u.User.Sid, sidOf(t, windows.WinLocalSystemSid)
	admins, everyone := sidOf(t, windows.WinBuiltinAdministratorsSid), sidOf(t, windows.WinWorldSid)
	oldO, oldS := pipeOwnerSID, pipeServerSID
	t.Cleanup(func() { pipeOwnerSID, pipeServerSID = oldO, oldS })
	bad := errors.New("denied")
	for _, c := range []struct {
		name          string
		owner, server *windows.SID // nil means the lookup fails
		ok            bool
	}{
		{"owner me", me, nil, true},
		{"owner system", system, nil, true},
		{"owner everyone", everyone, system, false}, // the fallback cannot rescue a foreign owner
		{"admins, server system", admins, system, true},
		{"admins, server me", admins, me, true},
		{"admins, server everyone", admins, everyone, false},
		{"admins, server unknown", admins, nil, false},
		{"owner unreadable, server system", nil, system, true},
		{"owner unreadable, server unknown", nil, nil, false},
		{"owner unreadable, server everyone", nil, everyone, false},
	} {
		pipeOwnerSID = func(windows.Handle) (*windows.SID, error) {
			if c.owner == nil {
				return nil, bad
			}
			return c.owner, nil
		}
		pipeServerSID = func(windows.Handle) (*windows.SID, error) {
			if c.server == nil {
				return nil, bad
			}
			return c.server, nil
		}
		if err := verifyPipeServer(0); (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

// A squatted pipe is closed and reported as no agent.
func TestDialAgentRefusesAForeignPipeServer(t *testing.T) {
	if os.Getenv("JUMPGATE_TEST_WIN_AGENT") != "1" {
		t.Skip("needs the ssh-agent service's pipe")
	}
	t.Setenv("SSH_AUTH_SOCK", "")
	old := pipeOwnerSID
	t.Cleanup(func() { pipeOwnerSID = old })
	everyone, _ := windows.CreateWellKnownSid(windows.WinWorldSid)
	pipeOwnerSID = func(windows.Handle) (*windows.SID, error) { return everyone, nil }
	if c := dialAgent(time.Now().Add(5 * time.Second)); c != nil {
		c.Close()
		t.Fatal("dialAgent accepted a pipe served by Everyone")
	}
}

// addCmd runs ssh-add without SSH_AUTH_SOCK in its environment: the test sets
// it to "", and OpenSSH reads an empty value as a socket path, not as unset.
func addCmd(name string, args ...string) *exec.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	time.AfterFunc(31*time.Second, cancel)
	cmd := exec.CommandContext(ctx, name, args...)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(e), "SSH_AUTH_SOCK=") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	return cmd
}

// Every error path of the owner decision refuses: a nil or invalid SID from
// either lookup, and a failed lookup at each stage of the fallback.
func TestVerifyPipeServerFailsClosed(t *testing.T) {
	system := sidOf(t, windows.WinLocalSystemSid)
	admins := sidOf(t, windows.WinBuiltinAdministratorsSid)
	oldO, oldS := pipeOwnerSID, pipeServerSID
	t.Cleanup(func() { pipeOwnerSID, pipeServerSID = oldO, oldS })
	boom := errors.New("boom")
	invalid := &windows.SID{} // zero revision: not a valid SID
	for _, c := range []struct {
		name                string
		owner, server       *windows.SID
		ownerErr, serverErr error
	}{
		{"owner lookup fails, server lookup fails", nil, nil, boom, boom},
		{"owner nil, server fails", nil, nil, nil, boom},
		{"owner nil, server nil", nil, nil, nil, nil},
		{"owner invalid, server invalid", invalid, invalid, nil, nil},
		{"owner invalid, server nil", invalid, nil, nil, nil},
		{"admins, server lookup (pid/open/token) fails", admins, nil, nil, boom},
		{"admins, server nil", admins, nil, nil, nil},
		{"admins, server invalid", admins, invalid, nil, nil},
		{"owner fails, server nil", nil, nil, boom, nil},
		{"owner fails, server invalid", nil, invalid, boom, nil},
	} {
		pipeOwnerSID = func(windows.Handle) (*windows.SID, error) { return c.owner, c.ownerErr }
		pipeServerSID = func(windows.Handle) (*windows.SID, error) { return c.server, c.serverErr }
		if err := verifyPipeServer(0); err == nil {
			t.Errorf("%s: accepted", c.name)
		}
	}
	// Sanity: the same seams do accept a SYSTEM owner.
	pipeOwnerSID = func(windows.Handle) (*windows.SID, error) { return system, nil }
	if err := verifyPipeServer(0); err != nil {
		t.Errorf("system owner refused: %v", err)
	}
}
