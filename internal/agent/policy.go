package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// Tier is how much a signer is trusted.
type Tier string

const (
	TierRoutine  Tier = "routine"
	TierApproval Tier = "approval"
)

// SignerEntry is one enrolled signer.
type SignerEntry struct {
	Address string `json:"address"`
	Tier    Tier   `json:"tier"`
	Label   string `json:"label"`
}

// Policy is /etc/jumpgate/policy.json: who may ask this box for what.
type Policy struct {
	Signers []SignerEntry `json:"signers"`
	// LocalUIDs may connect to the socket directly. A controller on the box
	// itself is enrolled by uid so it works without a re-login for a new group.
	LocalUIDs []int `json:"localUids,omitempty"`
	// Kinds overrides DefaultRequirements per kind.
	Kinds map[string][]Tier `json:"kinds,omitempty"`
}

// DefaultRequirements are sub-project 1's kinds, all routine. A kind absent
// from both this table and Policy.Kinds is unknown and refused.
var DefaultRequirements = map[string][]Tier{
	intent.KindAgentInfo:     {TierRoutine},
	intent.KindStatusRead:    {TierRoutine},
	intent.KindDiskRead:      {TierRoutine},
	intent.KindEndpointsRead: {TierRoutine},
	intent.KindFirewallRead:  {TierRoutine},
	intent.KindLogsRead:      {TierRoutine},
	intent.KindLogsSince:     {TierRoutine},
	intent.KindServiceAction: {TierRoutine},
}

// LoadPolicy reads the policy file, refusing one other users can write or read.
func LoadPolicy(path string) (Policy, error) {
	var p Policy
	fi, err := os.Stat(path)
	if err != nil {
		return p, fmt.Errorf("agent: policy: %w", err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return p, fmt.Errorf("agent: policy %s is mode %o; it must be 0600", path, fi.Mode().Perm())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return p, fmt.Errorf("agent: policy: %w", err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("agent: policy %s: %w", path, err)
	}
	return p, nil
}

// Save writes the policy atomically, 0600.
func (p Policy) Save(path string) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

// AddSigner enrolls e unless its address is already enrolled. It never
// changes an existing entry's tier: upgrading a signer is a separate,
// approval-tier decision.
func (p *Policy) AddSigner(e SignerEntry) bool {
	for _, s := range p.Signers {
		if strings.EqualFold(s.Address, e.Address) {
			return false
		}
	}
	p.Signers = append(p.Signers, e)
	return true
}

// AddLocalUID allows a local uid to use the socket.
func (p *Policy) AddLocalUID(uid int) bool {
	for _, u := range p.LocalUIDs {
		if u == uid {
			return false
		}
	}
	p.LocalUIDs = append(p.LocalUIDs, uid)
	return true
}

func (p Policy) tierOf(a eip712.Address) (Tier, bool) {
	for _, s := range p.Signers {
		if parsed, err := eip712.ParseAddress(s.Address); err == nil && parsed == a {
			return s.Tier, true
		}
	}
	return "", false
}

// Authorise checks that controller is an enrolled routine signer that signed,
// and that every tier the kind requires is covered by a distinct signer.
func (p Policy) Authorise(kind string, controller eip712.Address, signers []eip712.Address) error {
	req, ok := p.Kinds[kind]
	if !ok {
		req, ok = DefaultRequirements[kind]
	}
	if !ok {
		return reject(intent.ReasonUnknownKind, fmt.Sprintf("this agent does not know %q", kind))
	}
	if tier, enrolled := p.tierOf(controller); !enrolled || tier != TierRoutine {
		return reject(intent.ReasonUnauthorizedKind, fmt.Sprintf("controller %s is not enrolled on this box", controller.Hex()))
	}
	used := map[eip712.Address]bool{}
	signed := map[eip712.Address]bool{}
	for _, s := range signers {
		signed[s] = true
	}
	if !signed[controller] {
		return reject(intent.ReasonUnauthorizedKind, "the intent is not signed by the controller it names")
	}
	for _, need := range req {
		found := false
		// Prefer the controller for the routine slot so an approval signer
		// stays free for the approval slot.
		if need == TierRoutine && !used[controller] {
			used[controller], found = true, true
		}
		for _, s := range signers {
			if found {
				break
			}
			if tier, ok := p.tierOf(s); ok && tier == need && !used[s] {
				used[s], found = true, true
			}
		}
		if !found {
			return reject(intent.ReasonUnauthorizedKind, fmt.Sprintf("%s needs a %s signature", kind, need))
		}
	}
	return nil
}

// writeAtomic writes b to path through a fsynced temp file and rename.
func writeAtomic(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return syncDir(dir)
}

// syncDir is a variable so a test can make the directory fsync fail.
var syncDir = func(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("agent: open %s to sync it: %w", dir, err)
	}
	if err := d.Sync(); err != nil {
		d.Close()
		return fmt.Errorf("agent: sync directory %s: %w", dir, err)
	}
	if err := d.Close(); err != nil {
		return fmt.Errorf("agent: close directory %s: %w", dir, err)
	}
	return nil
}
