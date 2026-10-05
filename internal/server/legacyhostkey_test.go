package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func freshHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

var someAddr = &net.TCPAddr{IP: net.ParseIP("10.0.0.5"), Port: 22}

func legacyTarget(home string, agent *config.AgentPairing) config.Target {
	return config.Target{ID: "boxa", Mode: "ssh", Agent: agent, SSH: &executor.SSHConfig{
		Host: "10.0.0.5", User: "root", HostKeyFile: filepath.Join(home, ".jumpgate", "known_hosts"),
	}}
}

// An unconfirmed, unpaired box keeps trust-on-first-use, now passed
// explicitly: the first key is recorded and accepted.
func TestLegacySSHConfigIsTOFUForAnUnconfirmedBox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg, err := legacySSHConfig(legacyTarget(home, nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HostKey == nil {
		t.Fatal("no host-key policy")
	}
	if err := cfg.HostKey("10.0.0.5:22", someAddr, freshHostKey(t)); err != nil {
		t.Fatalf("TOFU refused a first key: %v", err)
	}
	if _, err := os.Stat(cfg.HostKeyFile); err != nil {
		t.Fatalf("TOFU did not record the key: %v", err)
	}
}

// A box a person confirmed is checked Strictly on the legacy path too: a
// different key is a mismatch even though the TOFU file never saw the box.
func TestLegacySSHConfigIsStrictForAConfirmedBox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	confirmed, err := config.ConfirmedHostsFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(confirmed), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := executor.RecordHostKey(confirmed, "10.0.0.5:22", freshHostKey(t)); err != nil {
		t.Fatal(err)
	}
	cfg, err := legacySSHConfig(legacyTarget(home, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.HostKey("10.0.0.5:22", someAddr, freshHostKey(t)); !errors.Is(err, executor.ErrHostKeyMismatch) {
		t.Fatalf("a changed key on a confirmed box: err = %v, want a mismatch", err)
	}
	if _, err := os.Stat(cfg.HostKeyFile); err == nil {
		t.Fatal("the legacy path wrote the TOFU file for a confirmed box")
	}
}

// A paired box is Strict on the legacy path: an unknown key is put in front of
// a person, never recorded.
func TestLegacySSHConfigIsStrictForAPairedBox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg, err := legacySSHConfig(legacyTarget(home, &config.AgentPairing{Address: "0x0000000000000000000000000000000000000001"}))
	if err != nil {
		t.Fatal(err)
	}
	var unknown *executor.UnknownHostError
	if err := cfg.HostKey("10.0.0.5:22", someAddr, freshHostKey(t)); !errors.As(err, &unknown) {
		t.Fatalf("an unknown key on a paired box: err = %v, want UnknownHostError", err)
	}
	if _, err := os.Stat(cfg.HostKeyFile); err == nil {
		t.Fatal("the legacy path TOFU-recorded a paired box's key")
	}
}

// The policy is per hop: a confirmed jump host is Strict even in front of an
// unconfirmed box, and the target's own policy does not leak onto the jump.
func TestLegacySSHConfigDecidesEachHop(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	confirmed, _ := config.ConfirmedHostsFile()
	_ = os.MkdirAll(filepath.Dir(confirmed), 0o700)
	if err := executor.RecordHostKey(confirmed, "10.0.0.1:22", freshHostKey(t)); err != nil {
		t.Fatal(err)
	}
	tgt := legacyTarget(home, nil)
	tgt.SSH.Jump = &executor.SSHConfig{Host: "10.0.0.1", User: "ops", HostKeyFile: tgt.SSH.HostKeyFile}
	cfg, err := legacySSHConfig(tgt)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Jump == nil || cfg.Jump.HostKey == nil {
		t.Fatal("the jump hop has no policy")
	}
	if err := cfg.Jump.HostKey("10.0.0.1:22", someAddr, freshHostKey(t)); !errors.Is(err, executor.ErrHostKeyMismatch) {
		t.Fatalf("confirmed jump host: err = %v, want a mismatch", err)
	}
	if err := cfg.HostKey("10.0.0.5:22", someAddr, freshHostKey(t)); err != nil {
		t.Fatalf("unconfirmed target: TOFU refused a first key: %v", err)
	}
	if tgt.SSH.Jump.HostKey != nil || tgt.SSH.HostKey != nil {
		t.Fatal("legacySSHConfig mutated the stored target")
	}
}

// writeKnownHosts writes line to the test HOME's ~/.ssh/known_hosts.
func writeKnownHosts(t *testing.T, home, line string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "known_hosts"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Fix round 1: a box whose key is in the operator's OpenSSH known_hosts is on
// record under the Strict builder's own rules, so the legacy path is Strict
// for it too. Otherwise a box paired while known only there would fall back to
// TOFU after being removed and re-added from the web UI.
func TestLegacySSHConfigIsStrictForABoxInOpenSSHKnownHosts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	known := string(ssh.MarshalAuthorizedKey(freshHostKey(t)))
	writeKnownHosts(t, home, "10.0.0.5 "+known[:len(known)-1])
	cfg, err := legacySSHConfig(legacyTarget(home, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.HostKey("10.0.0.5:22", someAddr, freshHostKey(t)); !errors.Is(err, executor.ErrHostKeyMismatch) {
		t.Fatalf("a changed key on a box in known_hosts: err = %v, want a mismatch", err)
	}
	if _, err := os.Stat(cfg.HostKeyFile); err == nil {
		t.Fatal("the legacy path wrote the TOFU file for a box in known_hosts")
	}
}

// A @cert-authority line alone does not put a host on record (Strict's own
// rule), so such a box stays on TOFU.
func TestLegacySSHConfigIgnoresACertAuthorityLine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ca := string(ssh.MarshalAuthorizedKey(freshHostKey(t)))
	writeKnownHosts(t, home, "@cert-authority 10.0.0.5 "+ca[:len(ca)-1])
	cfg, err := legacySSHConfig(legacyTarget(home, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.HostKey("10.0.0.5:22", someAddr, freshHostKey(t)); err != nil {
		t.Fatalf("a CA-only host was not left on TOFU: %v", err)
	}
}
