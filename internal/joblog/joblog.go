// internal/joblog/joblog.go

// Package joblog is a durable job's history: an append-only list of entries,
// each signed by the agent over EIP-712 and chained to the one before by its
// digest. The agent writes it; any controller verifies it, so a controller
// that adopts a job reads a history it can check rather than one it must
// trust. The package is pure: no files, no clocks of its own.
package joblog

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// Types is the frozen schema. See TestSchemaIsFrozen.
var Types = eip712.Types{
	"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}},
	"JobLogEntry": {
		{Name: "agent", Type: "address"}, {Name: "job", Type: "string"}, {Name: "index", Type: "uint64"},
		{Name: "prev", Type: "bytes32"}, {Name: "kind", Type: "string"}, {Name: "bodyHash", Type: "bytes32"},
		{Name: "at", Type: "uint64"},
	},
}

// Entry is one signed entry. Body is the exact bytes hashed and signed.
type Entry struct {
	Index uint64
	Prev  [32]byte // the previous entry's digest; zero for entry 0
	Kind  string
	Body  []byte
	At    uint64 // unix seconds, the agent's clock
	Sig   signer.Signature
}

// ErrBroken is every way a log can fail verification.
var ErrBroken = errors.New("joblog: the job log does not verify")

// TypedData binds an entry to its agent and job, so an entry cannot be moved
// to another job's log or claimed by another agent.
func (e Entry) TypedData(agent eip712.Address, job string) eip712.TypedData {
	return eip712.TypedData{Types: Types, PrimaryType: "JobLogEntry",
		Domain: map[string]any{"name": "jumpgate", "version": "1"},
		Message: map[string]any{"agent": agent, "job": job, "index": e.Index, "prev": e.Prev,
			"kind": e.Kind, "bodyHash": eip712.Keccak256(e.Body), "at": e.At}}
}

// Digest is the entry's EIP-712 digest: what the agent signs, and what the
// next entry carries as Prev.
func (e Entry) Digest(agent eip712.Address, job string) ([32]byte, error) {
	return eip712.Digest(e.TypedData(agent, job))
}

// Sign makes entry index of job, chained to prev, and returns it with its digest.
func Sign(ctx context.Context, s signer.Signer, job string, index uint64, prev [32]byte, kind string, body []byte, at time.Time) (Entry, [32]byte, error) {
	e := Entry{Index: index, Prev: prev, Kind: kind, Body: body, At: uint64(max(at.Unix(), 0))}
	sig, err := s.SignTypedData(ctx, e.TypedData(s.Address(), job))
	if err != nil {
		return Entry{}, [32]byte{}, fmt.Errorf("joblog: sign: %w", err)
	}
	e.Sig = sig
	d, err := e.Digest(s.Address(), job)
	return e, d, err
}

// Verify checks entries as positions start, start+1, … of job's log, the
// first chained to prev, every one signed by agent. It returns the last
// digest, which a caller keeps to verify the next page without the first.
//
// A nil error proves only that entries are a valid prefix of a log: a log cut
// short still verifies. The caller must compare the returned head (or the
// entry count) with a length it recorded independently, such as the
// manifest's, to detect truncation.
func Verify(agent eip712.Address, job string, entries []Entry, start uint64, prev [32]byte) ([32]byte, error) {
	for k, e := range entries {
		want := start + uint64(k)
		if e.Index != want {
			return prev, fmt.Errorf("%w: position %d holds entry %d", ErrBroken, want, e.Index)
		}
		if e.Prev != prev {
			return prev, fmt.Errorf("%w: entry %d does not follow entry %d", ErrBroken, want, int64(want)-1)
		}
		who, err := signer.Recover(e.TypedData(agent, job), e.Sig)
		if err != nil || who != agent {
			return prev, fmt.Errorf("%w: entry %d is not signed by agent %s", ErrBroken, want, agent.Hex())
		}
		d, err := e.Digest(agent, job)
		if err != nil {
			return prev, fmt.Errorf("%w: entry %d: %v", ErrBroken, want, err)
		}
		prev = d
	}
	return prev, nil
}

var jobIDRE = regexp.MustCompile(`^[0-9a-f]{16}$`)

// JobIDFor is the id of the job a creating intent starts: the first 8 bytes of
// the intent's EIP-712 digest, in hex (16 characters). The id commits to the
// signed request that created the job, so a created entry can be checked
// against the job it claims to start, and the intent's random nonce keeps ids
// unique.
func JobIDFor(in intent.Intent) (string, error) {
	d, err := in.Digest()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(d[:8]), nil
}

// VerifyIntents checks every entry that records a request: the intent was
// addressed to this agent, the controller it names signed it, and it is
// about this job. Without those bindings a genuine intent signed for another
// agent or job would pass and wrongly credit its controller. A created entry
// records no payload (D1); its intent must be the one the job id is derived
// from (JobIDFor), and the signed intent's payloadHash commits to the inputs
// the agent checked at admission. A later request's payload names its job,
// and is checked against the signed payloadHash. The agent checked all this
// when it admitted the intent; a controller checks it again because the log
// is how it learns who asked for what.
//
// What it proves is narrow: each recorded request was signed by the
// controller it names, about this job, for this agent. It does not prove
// that controller was enrolled, or held the tier the request needed, when
// it asked: the log does not carry the box's policy, and the agent checks
// enrollment and tier at admission, before it logs anything. Nor does it
// guard against a compromised agent key, which signs the log itself and is
// out of scope. It does not check the agent's entry signatures either, so
// the caller must also run Verify on the same entries.
func VerifyIntents(agent eip712.Address, job string, entries []Entry) error {
	// An empty or malformed id would let an id-less payload match it.
	if !jobIDRE.MatchString(job) {
		return fmt.Errorf("%w: %q is not a job id", ErrBroken, job)
	}
	for _, e := range entries {
		if e.Kind != KindCreated && e.Kind != KindIntent {
			continue
		}
		var b IntentBody
		if err := json.Unmarshal(e.Body, &b); err != nil {
			return fmt.Errorf("%w: entry %d: %v", ErrBroken, e.Index, err)
		}
		in, err := b.Envelope.Intent.Parse()
		if err != nil {
			return fmt.Errorf("%w: entry %d: %v", ErrBroken, e.Index, err)
		}
		if in.Agent != agent {
			return fmt.Errorf("%w: entry %d: the intent was addressed to agent %s", ErrBroken, e.Index, in.Agent.Hex())
		}
		if e.Kind == KindCreated {
			if id, err := JobIDFor(in); err != nil || id != job {
				return fmt.Errorf("%w: entry %d: the creating intent is not this job's", ErrBroken, e.Index)
			}
		}
		if e.Kind == KindIntent {
			var p struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(b.Envelope.Payload, &p); err != nil || p.ID != job {
				return fmt.Errorf("%w: entry %d: the intent is not for job %s", ErrBroken, e.Index, job)
			}
		}
		// A created entry written today has no payload (D1, see IntentBody),
		// and needs none: the controller's signature below covers the intent,
		// whose payloadHash commits to the inputs. One written before carries
		// the payload, which must still be the one signed. A later request
		// always carries its payload: the id check above read it.
		if (e.Kind == KindIntent || b.Envelope.Payload != nil) && intent.Hash(b.Envelope.Payload) != in.PayloadHash {
			return fmt.Errorf("%w: entry %d: the payload is not the one signed", ErrBroken, e.Index)
		}
		named := false
		for _, s := range b.Envelope.Sigs {
			sig, err := signer.ParseSignature(s)
			if err != nil {
				continue
			}
			if who, err := signer.Recover(in.TypedData(), sig); err == nil && who == in.Controller {
				named = true
			}
		}
		if !named {
			return fmt.Errorf("%w: entry %d: not signed by controller %s", ErrBroken, e.Index, in.Controller.Hex())
		}
	}
	return nil
}

// ToJSON is the wire and file form.
func ToJSON(e Entry) intent.JobLogEntryJSON {
	return intent.JobLogEntryJSON{Index: e.Index, Prev: "0x" + hex.EncodeToString(e.Prev[:]), Kind: e.Kind, Body: e.Body, At: e.At, Sig: e.Sig.Hex()}
}

// FromJSON parses the wire and file form. It does not verify.
func FromJSON(j intent.JobLogEntryJSON) (Entry, error) {
	e := Entry{Index: j.Index, Kind: j.Kind, Body: j.Body, At: j.At}
	p, err := hex.DecodeString(strings.TrimPrefix(j.Prev, "0x"))
	if err != nil || len(p) != 32 {
		return e, fmt.Errorf("%w: entry %d: prev is not 32 bytes of hex", ErrBroken, j.Index)
	}
	copy(e.Prev[:], p)
	if e.Sig, err = signer.ParseSignature(j.Sig); err != nil {
		return e, fmt.Errorf("%w: entry %d: %v", ErrBroken, j.Index, err)
	}
	return e, nil
}
