package agent

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// skew is how far a controller's clock may differ from the agent's. It is
// defined here only; agent.go uses it.
const skew = 60 * time.Second

// ErrReplayState means the replay record is missing or unreadable. The agent
// refuses every intent until an operator resets it, because starting from an
// empty record would make every captured intent replayable.
var ErrReplayState = errors.New("agent: replay state is missing or corrupt; run `jumpgate agent reset-replay` after checking why")

// Replay is /var/lib/jumpgate/replay.json.
type Replay struct {
	path string
	mu   sync.Mutex
	st   replayFile
}

type replayFile struct {
	Controllers map[string]uint64 `json:"controllers"` // address hex → last accepted seq
	Nonces      map[string]uint64 `json:"nonces"`      // nonce hex → unix time it may be forgotten
}

// InitReplay creates an empty record if none exists. It never overwrites.
func InitReplay(path string) error {
	b, _ := json.Marshal(replayFile{Controllers: map[string]uint64{}, Nonces: map[string]uint64{}})
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return writeAtomic(path, b)
}

// OpenReplay loads the record, failing closed.
func OpenReplay(path string) (*Replay, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReplayState, err)
	}
	var st replayFile
	if err := json.Unmarshal(b, &st); err != nil || st.Controllers == nil || st.Nonces == nil {
		return nil, fmt.Errorf("%w: %s does not parse", ErrReplayState, path)
	}
	return &Replay{path: path, st: st}, nil
}

// LastSeq is the last sequence accepted from controller.
func (r *Replay) LastSeq(controller eip712.Address) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.st.Controllers[controller.Hex()]
}

// Admit accepts (controller, seq, nonce) once, and persists the acceptance
// before returning. The caller executes only after Admit succeeds: a crash
// after this point loses one intent, which is safe, where executing first
// would let the same intent run twice.
func (r *Replay) Admit(controller eip712.Address, seq uint64, nonce [32]byte, expiry uint64, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := controller.Hex()
	last := r.st.Controllers[key]
	if seq <= last {
		return &Reject{Code: intent.ReasonStaleSeq, Message: fmt.Sprintf("sequence %d is not above %d", seq, last), LastSeq: last}
	}
	nk := hex.EncodeToString(nonce[:])
	if until, seen := r.st.Nonces[nk]; seen && uint64(now.Unix()) <= until {
		return reject(intent.ReasonReplayedNonce, "this nonce was already used")
	}

	next := replayFile{Controllers: map[string]uint64{}, Nonces: map[string]uint64{}}
	for k, v := range r.st.Controllers {
		next.Controllers[k] = v
	}
	for k, v := range r.st.Nonces {
		if v >= uint64(now.Unix()) { // prune expired nonces
			next.Nonces[k] = v
		}
	}
	next.Controllers[key] = seq
	next.Nonces[nk] = expiry + uint64(skew/time.Second)

	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := writeAtomic(r.path, b); err != nil {
		return fmt.Errorf("agent: persist replay state: %w", err)
	}
	r.st = next
	return nil
}
