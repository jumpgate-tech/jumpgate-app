package server

import (
	"errors"
	"os"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
)

func remoteTarget(t *testing.T) config.Target {
	t.Helper()
	agentKey, err := signer.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return config.Target{ID: "box", Mode: "ssh",
		SSH:   &executor.SSHConfig{Host: "box.example", Port: 22, User: "root"},
		Agent: &config.AgentPairing{Address: agentKey.Address().Hex(), Transport: "ssh"}}
}

func TestAgentTargetUsesTheVaultSigner(t *testing.T) {
	srv, _, _, v := refsServer(t, false, nil)
	line, err := srv.ensureTransportKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(transportKeyPath()); !os.IsNotExist(err) {
		t.Fatalf("a key file was written under the vault: %v", err)
	}
	at, err := srv.agentTarget(remoteTarget(t))
	if err != nil {
		t.Fatal(err)
	}
	if at.SSH.KeyPath != "" || at.SSH.Signer == nil {
		t.Fatalf("ssh config %+v", at.SSH)
	}
	if want, _ := secrets.EnsureTransportKey(v); want != line {
		t.Fatalf("line %q, vault %q", line, want)
	}
	v.Lock()
	if _, err := srv.agentTarget(remoteTarget(t)); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("locked: %v", err)
	}
	if _, err := srv.ensureTransportKey(); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("locked ensure: %v", err)
	}
}

func TestAgentTargetKeepsTheFileWithoutAVault(t *testing.T) {
	testutil.Home(t)
	at, err := (&Server{}).agentTarget(remoteTarget(t))
	if err != nil {
		t.Fatal(err)
	}
	if at.SSH.KeyPath != transportKeyPath() || at.SSH.Signer != nil {
		t.Fatalf("ssh config %+v", at.SSH)
	}
}
