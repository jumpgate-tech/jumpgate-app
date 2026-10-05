// Package agent is jumpgate's on-box half. It accepts only typed, signed
// intents, checks them against this box's own policy and replay record, runs
// the matching operation with a local executor, and signs what it answers.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// skew is defined in replay.go.
const (
	maxLifetime = 300 * time.Second
	maxBody     = 1 << 20
)

// Config wires an Agent. Paths are /etc/jumpgate/policy.json,
// /var/lib/jumpgate/replay.json and /etc/jumpgate/node.json in production.
type Config struct {
	Key        *signer.Key
	PolicyPath string
	ReplayPath string
	NodePath   string
	Exec       executor.Executor
	Now        func() time.Time
}

// Agent handles intents. It is safe for concurrent use.
type Agent struct {
	cfg       Config
	replay    *Replay
	replayErr error

	mu   sync.Mutex
	busy map[eip712.Address]bool
}

// New builds an agent. A broken replay record does not stop construction: the
// agent still answers, with a signed replay_state rejection, so an operator
// sees why rather than a dead socket.
func New(cfg Config) *Agent {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	a := &Agent{cfg: cfg, busy: map[eip712.Address]bool{}}
	a.replay, a.replayErr = OpenReplay(cfg.ReplayPath)
	return a
}

// Address is the agent's signing address.
func (a *Agent) Address() eip712.Address { return a.cfg.Key.Address() }

// Handle runs the checks in a fixed order: parse, agent, time, payload hash,
// signatures, policy, one-at-a-time per controller, replay admission (which
// persists), and only then the operation. Everything before admission is
// free to reject without consuming the controller's sequence.
func (a *Agent) Handle(ctx context.Context, env intent.Envelope) intent.ReceiptEnvelope {
	i, err := env.Intent.Parse()
	if err != nil {
		return a.answerReject(intent.Intent{}, [32]byte{}, reject(intent.ReasonInvalidPayload, err.Error()))
	}
	reqHash, err := i.Digest()
	if err != nil {
		return a.answerReject(i, [32]byte{}, reject(intent.ReasonInvalidPayload, err.Error()))
	}
	if i.Agent != a.Address() {
		return a.answerReject(i, reqHash, reject(intent.ReasonWrongAgent, "this intent is addressed to "+i.Agent.Hex()))
	}

	// The time checks work in unix seconds as uint64, so no wire value can
	// wrap through an int64 conversion into an accepted window. Lifetime is
	// checked first; once issuedAt is within skew of now, expiry is at most
	// now+360s and expiry+skew cannot overflow.
	now := a.cfg.Now()
	nowSec, skewSec := uint64(max(now.Unix(), 0)), uint64(skew/time.Second)
	switch {
	case i.Expiry < i.IssuedAt || i.Expiry-i.IssuedAt > uint64(maxLifetime/time.Second):
		return a.answerReject(i, reqHash, reject(intent.ReasonInvalidPayload, "intent lifetime must be at most 300s"))
	case i.IssuedAt > nowSec+skewSec:
		r := reject(intent.ReasonClockSkew, fmt.Sprintf("issued at unix time %d, agent time is %s", i.IssuedAt, now.UTC().Format(time.RFC3339)))
		return a.answerReject(i, reqHash, r)
	case nowSec > i.Expiry+skewSec:
		return a.answerReject(i, reqHash, reject(intent.ReasonExpired, "intent expired at "+time.Unix(int64(i.Expiry), 0).UTC().Format(time.RFC3339)))
	}

	if intent.Hash(env.Payload) != i.PayloadHash {
		return a.answerReject(i, reqHash, reject(intent.ReasonInvalidPayload, "payload does not match its hash"))
	}

	var signers []eip712.Address
	for _, s := range env.Sigs {
		sig, err := signer.ParseSignature(s)
		if err != nil {
			return a.answerReject(i, reqHash, reject(intent.ReasonBadSignature, err.Error()))
		}
		who, err := signer.Recover(i.TypedData(), sig)
		if err != nil {
			return a.answerReject(i, reqHash, reject(intent.ReasonBadSignature, err.Error()))
		}
		signers = append(signers, who)
	}

	policy, err := LoadPolicy(a.cfg.PolicyPath)
	if err != nil {
		return a.answerReject(i, reqHash, reject(intent.ReasonUnauthorizedKind, err.Error()))
	}
	if err := policy.Authorise(i.Kind, i.Controller, signers); err != nil {
		return a.answerReject(i, reqHash, asReject(err))
	}

	if !a.claim(i.Controller) {
		return a.answerReject(i, reqHash, reject(intent.ReasonBusy, "another intent from this controller is running"))
	}
	defer a.release(i.Controller)

	replay, rerr := a.currentReplay()
	if rerr != nil {
		return a.answerReject(i, reqHash, reject(intent.ReasonReplayState, rerr.Error()))
	}
	if err := replay.Admit(i.Controller, i.Seq, i.Nonce, i.Expiry, now); err != nil {
		return a.answerReject(i, reqHash, asReject(err))
	}

	result, status, rej := a.dispatch(ctx, i, env.Payload, policy)
	if rej != nil {
		return a.answerReject(i, reqHash, rej)
	}
	return a.answer(i, reqHash, status, result)
}

// currentReplay returns the replay record. While the record is failed it is
// reloaded on each request, under the agent lock, so an operator's repair takes
// effect without a restart. A still-bad file keeps the agent failed closed.
func (a *Agent) currentReplay() (*Replay, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.replayErr != nil {
		a.replay, a.replayErr = OpenReplay(a.cfg.ReplayPath)
	}
	return a.replay, a.replayErr
}

func (a *Agent) claim(c eip712.Address) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy[c] {
		return false
	}
	a.busy[c] = true
	return true
}

func (a *Agent) release(c eip712.Address) {
	a.mu.Lock()
	delete(a.busy, c)
	a.mu.Unlock()
}

func asReject(err error) *Reject {
	if r, ok := err.(*Reject); ok {
		return r
	}
	return reject(intent.ReasonValidation, err.Error())
}

func (a *Agent) answerReject(i intent.Intent, reqHash [32]byte, r *Reject) intent.ReceiptEnvelope {
	body, _ := json.Marshal(intent.Rejection{Code: r.Code, Message: r.Message, LastSeq: r.LastSeq, AgentTime: a.cfg.Now().Unix()})
	return a.answer(i, reqHash, intent.StatusRejected, body)
}

// answer signs a receipt over the exact result bytes. Rejections are signed
// too, so a controller can prove why it was refused.
func (a *Agent) answer(i intent.Intent, reqHash [32]byte, status uint8, result []byte) intent.ReceiptEnvelope {
	rc := intent.Receipt{Agent: a.Address(), RequestHash: reqHash, Seq: i.Seq, Status: status, ResultHash: intent.Hash(result)}
	sig, _ := a.cfg.Key.SignTypedData(context.Background(), rc.TypedData())
	return intent.ReceiptEnvelope{Receipt: rc.JSON(), Result: result, Sig: sig.Hex()}
}
