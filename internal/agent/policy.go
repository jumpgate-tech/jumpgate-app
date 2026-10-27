package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/peercred"
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
	// Scope is "" (any peer) or ScopeLocal: the signer is admitted only
	// from a peer on the box itself (root or an enrolled local uid), never
	// through the SSH tunnel (D11, a --self-only controller).
	Scope string `json:"scope,omitempty"`
}

// ScopeLocal limits a signer to peers on the box itself.
const ScopeLocal = "local"

// Policy is /etc/jumpgate/policy.json: who may ask this box for what.
type Policy struct {
	Signers []SignerEntry `json:"signers"`
	// LocalUIDs may connect to the socket directly. A controller on the box
	// itself is enrolled by uid so it works without a re-login for a new group.
	LocalUIDs []int `json:"localUids,omitempty"`
	// Kinds overrides DefaultRequirements per kind.
	Kinds map[string][]Tier `json:"kinds,omitempty"`
}

// DefaultRequirements are sub-project 1's kinds and sub-project 2's job kinds, all routine. A kind absent
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
	intent.KindModulesRead:   {TierRoutine},
	// agent.upgrade is routine, like re-pairing a pushed box; a box can
	// require approval for it in policy.json.
	intent.KindAgentUpgrade: {TierRoutine},

	intent.KindJobList:          {TierRoutine},
	intent.KindJobRead:          {TierRoutine},
	intent.KindJobOutput:        {TierRoutine},
	intent.KindJobLease:         {TierRoutine},
	intent.KindJobAdvance:       {TierRoutine},
	intent.KindJobDecide:        {TierRoutine},
	intent.KindNodeMigrateNames: {TierRoutine},
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
	return fsatomic.WriteFile(path, b)
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

// NarrowScope sets address's scope to ScopeLocal. It only ever narrows: no
// call widens a scoped signer back to any peer. It reports a change.
func (p *Policy) NarrowScope(address string) bool {
	for i, s := range p.Signers {
		if strings.EqualFold(s.Address, address) && s.Scope != ScopeLocal {
			p.Signers[i].Scope = ScopeLocal
			return true
		}
	}
	return false
}

// CheckScope admits controller's request from peer. An unscoped signer is
// admitted from any peer; a ScopeLocal one only from root or a uid in
// LocalUIDs, which the tunnel user never is.
func (p Policy) CheckScope(controller eip712.Address, peer peercred.Cred, havePeer bool) error {
	scope := ""
	for _, s := range p.Signers {
		if parsed, err := eip712.ParseAddress(s.Address); err == nil && parsed == controller {
			scope = s.Scope
			break
		}
	}
	if scope == "" {
		return nil
	}
	if havePeer {
		if peer.UID == 0 {
			return nil
		}
		for _, u := range p.LocalUIDs {
			if u == peer.UID {
				return nil
			}
		}
	}
	return reject(intent.ReasonScopeLocal, fmt.Sprintf("controller %s is enrolled for local use only and cannot reach this box through a tunnel", controller.Hex()))
}

// routineSigners counts the controllers enrolled at the routine tier.
func (p Policy) routineSigners() int {
	n := 0
	for _, s := range p.Signers {
		if s.Tier == TierRoutine {
			n++
		}
	}
	return n
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
// and that every tier the kind requires, plus each of extra, is covered by a
// distinct signer. extra is what this one request needs beyond its kind: a
// decision on a question of the approval tier needs an approval signature
// whatever the policy asks of job.decide (Q5). A job kind with neither a
// default requirement nor a policy entry stays unknown: registering a plan
// does not authorise anyone to start it.
func (p Policy) Authorise(kind string, controller eip712.Address, signers []eip712.Address, extra ...Tier) error {
	req, ok := p.Kinds[kind]
	if !ok {
		req, ok = DefaultRequirements[kind]
	}
	if !ok {
		return reject(intent.ReasonUnknownKind, fmt.Sprintf("this agent does not know %q", kind))
	}
	req = append(append([]Tier{}, req...), extra...)
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
