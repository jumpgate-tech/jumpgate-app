// This file holds what the server needs to act as a controller of paired
// agents: the SSH transport key, how an agent is reached, and where each
// target's intent sequence is kept.
package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// jgPath is a path under the controller's state directory (~/.jumpgate).
func jgPath(parts ...string) string {
	dir, _ := config.Dir()
	return filepath.Join(append([]string{dir}, parts...)...)
}

// transportKeyPath is the controller's SSH key for the jumpgate tunnel user.
func transportKeyPath() string { return jgPath("ssh", "jumpgate_ed25519") }

// ensureTransportKey returns the controller's tunnel key as an authorized_keys
// line, creating an ed25519 key (0600) on first use. It is separate from the
// operator's login key so revoking jumpgate's access never touches theirs.
//
// A key is generated only when nothing at all is at the path. Anything else
// that cannot be read as a key (a dangling symlink, a directory, an unreadable
// or corrupt file) is an error for the operator, never silently replaced. The
// new key is written to a temp file and hard-linked into place, so a
// concurrent caller sees either no file or a whole key, and exactly one
// caller's key wins.
func ensureTransportKey() (string, error) {
	path := transportKeyPath()
	line, err := readTransportKey(path)
	if err == nil {
		return line, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if fi, lerr := os.Lstat(path); lerr == nil {
		// A concurrent caller may have linked its key in between the read
		// and this Lstat; that key is whole (it arrives by hard link), so use it.
		if line, rerr := readTransportKey(path); rerr == nil {
			return line, nil
		}
		what := "it cannot be read"
		if fi.Mode()&os.ModeSymlink != 0 {
			what = "it is a symlink to a file that does not exist"
		}
		return "", fmt.Errorf("transport key %s: %s; fix or remove it", path, what)
	} else if !os.IsNotExist(lerr) {
		return "", fmt.Errorf("transport key %s: %w", path, lerr)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	block, err := ssh.MarshalPrivateKey(priv, "jumpgate-controller")
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".jumpgate_ed25519.tmp-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	// CreateTemp already makes the file 0600; Chmod states it outright.
	werr := tmp.Chmod(0o600)
	if werr == nil {
		_, werr = tmp.Write(pem.EncodeToMemory(block))
	}
	if werr == nil {
		werr = tmp.Sync()
	}
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return "", fmt.Errorf("transport key %s: %w", path, werr)
	}
	if err := os.Link(tmp.Name(), path); err != nil {
		if !os.IsExist(err) {
			return "", fmt.Errorf("transport key %s: %w", path, err)
		}
		// Another caller linked its key first; use that one.
	}
	return readTransportKey(path)
}

// readTransportKey reads the key at path as an authorized_keys line. A
// missing file keeps its IsNotExist error so the caller can tell it apart.
func readTransportKey(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", fmt.Errorf("transport key %s: %w", path, err)
	}
	signer, err := ssh.ParsePrivateKey(b)
	if err != nil {
		return "", fmt.Errorf("transport key %s: %w", path, err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))) + " jumpgate-controller", nil
}

// strictHostKey checks a host key against keys a person confirmed: the
// confirmed-only store (never written by trust-on-first-use) and the
// operator's OpenSSH known_hosts. DialSSH hands it to the jump host too. The
// second result asks each host for the key types on record, so a host known
// by one type is not mistaken for a changed one.
func strictHostKey() (ssh.HostKeyCallback, func(string) []string, error) {
	confirmed, err := config.ConfirmedHostsFile()
	if err != nil {
		return nil, nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	known := executor.OpenSSHKnownHosts(home)
	return executor.Strict(confirmed, known...), executor.KnownHostKeyAlgorithms(confirmed, known...), nil
}

// agentTarget is how this controller reaches t's agent: directly for a local
// target, otherwise as the jumpgate tunnel user with the transport key and
// strict host-key checking.
func agentTarget(t config.Target) (agentclient.Target, error) {
	if t.Agent == nil {
		return agentclient.Target{}, fmt.Errorf("target %q is not paired", t.ID)
	}
	addr, err := eip712.ParseAddress(t.Agent.Address)
	if err != nil {
		return agentclient.Target{}, err
	}
	at := agentclient.Target{Agent: addr, Socket: t.Agent.Socket, Local: t.Agent.Transport == "local"}
	if !at.Local {
		if t.SSH == nil {
			return agentclient.Target{}, fmt.Errorf("target %q has no SSH address", t.ID)
		}
		hostKey, algos, err := strictHostKey()
		if err != nil {
			return agentclient.Target{}, err
		}
		at.SSH = executor.SSHConfig{
			Host: t.SSH.Host, Port: t.SSH.Port, User: "jumpgate", KeyPath: transportKeyPath(), Jump: t.SSH.Jump,
			HostKey: hostKey, HostKeyAlgorithms: algos,
		}
	}
	return at, nil
}

// configSeqs stores the next sequence in the target's pairing record. It only
// touches a record still paired with the agent it is asked about, so a
// re-pairing that lands mid-request is never given the old agent's counter.
type configSeqs struct{ targetID string }

func (s configSeqs) pairing(c *config.Config, agent eip712.Address) (*config.AgentPairing, error) {
	for i := range c.Targets {
		t := &c.Targets[i]
		if t.ID != s.targetID || t.Agent == nil {
			continue
		}
		if a, err := eip712.ParseAddress(t.Agent.Address); err == nil && a == agent {
			return t.Agent, nil
		}
		return nil, fmt.Errorf("target %q is now paired with a different agent", s.targetID)
	}
	return nil, fmt.Errorf("target %q is not paired", s.targetID)
}

func (s configSeqs) Next(agent eip712.Address) (uint64, error) {
	c, err := config.Load()
	if err != nil {
		return 0, err
	}
	p, err := s.pairing(&c, agent)
	if err != nil {
		return 0, err
	}
	if p.NextSeq == 0 {
		return 1, nil
	}
	return p.NextSeq, nil
}

func (s configSeqs) Set(agent eip712.Address, next uint64) error {
	_, err := config.Update(func(c *config.Config) error {
		p, err := s.pairing(c, agent)
		if err != nil {
			return err
		}
		p.NextSeq = next
		return nil
	})
	return err
}
