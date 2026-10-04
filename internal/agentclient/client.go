// Package agentclient is the controller's side of the agent protocol: it
// builds and signs intents, carries them to the agent's socket, and accepts an
// answer only if the paired agent signed it and it answers this exact request.
package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

var (
	// ErrBadReceipt is a security error: the answer was not signed by the
	// paired agent, or does not answer this request. Its result is discarded.
	ErrBadReceipt = errors.New("agentclient: receipt failed verification")
	// ErrUnreachable is a transport failure, never a refusal.
	ErrUnreachable = errors.New("agentclient: could not reach the agent")
	// ErrAgentHTTP means the agent socket answered with a non-200 HTTP status
	// (peer refused, request too large, unreadable body). Such replies are
	// unsigned and are never a Rejection.
	ErrAgentHTTP = errors.New("agentclient: agent socket answered with an HTTP error")
)

// maxReply bounds how much of an agent reply is read before verification; it
// leaves room for logs.read at n=2000.
const maxReply = 16 << 20

// Target says how to reach one agent and whom to expect there.
type Target struct {
	Local  bool
	Socket string
	SSH    executor.SSHConfig
	Agent  eip712.Address
}

// SeqStore keeps the next sequence number per agent.
type SeqStore interface {
	Next(agent eip712.Address) (uint64, error)
	Set(agent eip712.Address, next uint64) error
}

type memSeqs struct {
	mu sync.Mutex
	m  map[eip712.Address]uint64
}

// NewMemorySeqStore is for tests and one-shot tools.
func NewMemorySeqStore() SeqStore { return &memSeqs{m: map[eip712.Address]uint64{}} }

func (s *memSeqs) Next(a eip712.Address) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m[a] == 0 {
		return 1, nil
	}
	return s.m[a], nil
}

func (s *memSeqs) Set(a eip712.Address, next uint64) error {
	s.mu.Lock()
	s.m[a] = next
	s.mu.Unlock()
	return nil
}

// Response is a verified answer.
type Response struct {
	Status    uint8
	Result    json.RawMessage
	Rejection *intent.Rejection
	Failure   *intent.Failure
}

// Client talks to one agent.
type Client struct {
	t      Target
	signer signer.Signer
	seqs   SeqStore
	hc     *http.Client
	closer func() error
	mu     sync.Mutex // serialises Do so a sequence is never signed twice
	now    func() time.Time
}

// Dial connects (for SSH, the connection is opened now and reused).
func Dial(ctx context.Context, t Target, s signer.Signer, seqs SeqStore) (*Client, error) {
	hc, closer, err := transport(ctx, t)
	if err != nil {
		return nil, err
	}
	return &Client{t: t, signer: s, seqs: seqs, hc: hc, closer: closer, now: time.Now}, nil
}

// Close releases the SSH connection.
func (c *Client) Close() error { return c.closer() }

// Do sends one intent and returns the verified answer. A stale_seq rejection
// resynchronises the counter from the agent's signed LastSeq and retries once.
func (c *Client) Do(ctx context.Context, kind string, payload any) (Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}
	for attempt := 0; ; attempt++ {
		res, err := c.once(ctx, kind, body)
		if err != nil {
			return res, err
		}
		if res.Rejection != nil && res.Rejection.Code == intent.ReasonStaleSeq && attempt == 0 {
			if err := c.seqs.Set(c.t.Agent, res.Rejection.LastSeq+1); err != nil {
				return res, err
			}
			continue
		}
		return res, nil
	}
}

func (c *Client) once(ctx context.Context, kind string, body []byte) (Response, error) {
	seq, err := c.seqs.Next(c.t.Agent)
	if err != nil {
		return Response{}, err
	}
	nonce, err := intent.NewNonce()
	if err != nil {
		return Response{}, err
	}
	now := uint64(c.now().Unix())
	in := intent.Intent{Agent: c.t.Agent, Controller: c.signer.Address(), Seq: seq, Nonce: nonce,
		IssuedAt: now, Expiry: now + 120, Kind: kind, PayloadHash: intent.Hash(body)}
	sig, err := c.signer.SignTypedData(ctx, in.TypedData())
	if err != nil {
		return Response{}, fmt.Errorf("agentclient: sign: %w", err)
	}
	reqHash, err := in.Digest()
	if err != nil {
		return Response{}, err
	}

	envBytes, _ := json.Marshal(intent.Envelope{Intent: in.JSON(), Payload: body, Sigs: []string{sig.Hex()}})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://agent/v1/intent", bytes.NewReader(envBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return Response{}, fmt.Errorf("%w: HTTP %d: %s", ErrAgentHTTP, resp.StatusCode, bytes.TrimSpace(snippet))
	}
	// Read one byte past the limit so truncation can never pass for a reply.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxReply+1))
	if err != nil {
		return Response{}, fmt.Errorf("%w: reading answer: %v", ErrUnreachable, err)
	}
	if len(raw) > maxReply {
		return Response{}, fmt.Errorf("%w: answer exceeds %d bytes", ErrBadReceipt, maxReply)
	}
	var out intent.ReceiptEnvelope
	if err := json.Unmarshal(raw, &out); err != nil {
		return Response{}, fmt.Errorf("%w: undecodable answer: %v", ErrBadReceipt, err)
	}

	rc, err := out.Receipt.Parse()
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrBadReceipt, err)
	}
	rsig, err := signer.ParseSignature(out.Sig)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrBadReceipt, err)
	}
	who, err := signer.Recover(rc.TypedData(), rsig)
	switch {
	case err != nil:
		return Response{}, fmt.Errorf("%w: %v", ErrBadReceipt, err)
	case who != c.t.Agent:
		return Response{}, fmt.Errorf("%w: signed by %s, expected agent %s", ErrBadReceipt, who.Hex(), c.t.Agent.Hex())
	case rc.Agent != c.t.Agent || rc.RequestHash != reqHash || rc.Seq != seq || rc.ResultHash != intent.Hash(out.Result):
		return Response{}, fmt.Errorf("%w: receipt does not answer this request", ErrBadReceipt)
	}

	// The agent saw this sequence (admitted or not); never reuse it.
	if err := c.seqs.Set(c.t.Agent, seq+1); err != nil {
		return Response{}, err
	}
	res := Response{Status: rc.Status, Result: out.Result}
	switch rc.Status {
	case intent.StatusRejected:
		res.Rejection = &intent.Rejection{}
		_ = json.Unmarshal(out.Result, res.Rejection)
	case intent.StatusFailed:
		res.Failure = &intent.Failure{}
		_ = json.Unmarshal(out.Result, res.Failure)
	}
	return res, nil
}
