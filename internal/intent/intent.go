// Package intent defines the messages a controller signs to ask an agent to do
// something, and the receipts an agent signs in answer. What is signed is the
// decision itself (a kind plus the hash of its exact payload bytes), never a
// shell command.
package intent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

// Types is the frozen EIP-712 schema. See TestSchemasAreFrozen.
var Types = eip712.Types{
	"EIP712Domain": {
		{Name: "name", Type: "string"}, {Name: "version", Type: "string"},
	},
	"Intent": {
		{Name: "agent", Type: "address"}, {Name: "controller", Type: "address"},
		{Name: "seq", Type: "uint64"}, {Name: "nonce", Type: "bytes32"},
		{Name: "issuedAt", Type: "uint64"}, {Name: "expiry", Type: "uint64"},
		{Name: "kind", Type: "string"}, {Name: "payloadHash", Type: "bytes32"},
	},
	"Receipt": {
		{Name: "agent", Type: "address"}, {Name: "requestHash", Type: "bytes32"},
		{Name: "seq", Type: "uint64"}, {Name: "status", Type: "uint8"},
		{Name: "resultHash", Type: "bytes32"},
	},
}

// domain carries no chainId: wallets tie chainId to the selected network, and
// these messages are not on-chain. A node's chain id is ordinary data.
func domain() map[string]any { return map[string]any{"name": "jumpgate", "version": "1"} }

// Receipt statuses.
const (
	StatusOK       uint8 = 0
	StatusRejected uint8 = 1 // policy, replay, expiry or validation refused it
	StatusFailed   uint8 = 2 // it ran and the operation failed
)

// Intent is one signed request.
type Intent struct {
	Agent, Controller eip712.Address
	Seq               uint64
	Nonce             [32]byte
	IssuedAt, Expiry  uint64
	Kind              string
	PayloadHash       [32]byte
}

// TypedData is what each signer signs.
func (i Intent) TypedData() eip712.TypedData {
	return eip712.TypedData{Types: Types, PrimaryType: "Intent", Domain: domain(), Message: map[string]any{
		"agent": i.Agent, "controller": i.Controller, "seq": i.Seq, "nonce": i.Nonce,
		"issuedAt": i.IssuedAt, "expiry": i.Expiry, "kind": i.Kind, "payloadHash": i.PayloadHash,
	}}
}

// Digest is the request hash a receipt answers.
func (i Intent) Digest() ([32]byte, error) { return eip712.Digest(i.TypedData()) }

// Receipt is the agent's signed answer.
type Receipt struct {
	Agent       eip712.Address
	RequestHash [32]byte
	Seq         uint64
	Status      uint8
	ResultHash  [32]byte
}

// TypedData is what the agent signs.
func (r Receipt) TypedData() eip712.TypedData {
	return eip712.TypedData{Types: Types, PrimaryType: "Receipt", Domain: domain(), Message: map[string]any{
		"agent": r.Agent, "requestHash": r.RequestHash, "seq": r.Seq, "status": r.Status, "resultHash": r.ResultHash,
	}}
}

// Digest is the hash the agent signs.
func (r Receipt) Digest() ([32]byte, error) { return eip712.Digest(r.TypedData()) }

// Hash is keccak256 of the exact bytes sent. The receiver hashes what it
// received before decoding it, so no JSON canonicalisation is ever needed.
func Hash(b []byte) [32]byte { return eip712.Keccak256(b) }

// NewNonce returns 32 random bytes.
func NewNonce() ([32]byte, error) {
	var n [32]byte
	_, err := rand.Read(n[:])
	return n, err
}

// IntentJSON is the wire form of Intent.
type IntentJSON struct {
	Agent       string `json:"agent"`
	Controller  string `json:"controller"`
	Seq         uint64 `json:"seq"`
	Nonce       string `json:"nonce"`
	IssuedAt    uint64 `json:"issuedAt"`
	Expiry      uint64 `json:"expiry"`
	Kind        string `json:"kind"`
	PayloadHash string `json:"payloadHash"`
}

// Envelope is one request on the wire. Payload is base64 in JSON.
type Envelope struct {
	Intent  IntentJSON `json:"intent"`
	Payload []byte     `json:"payload"`
	Sigs    []string   `json:"sigs"`
}

// ReceiptJSON is the wire form of Receipt.
type ReceiptJSON struct {
	Agent       string `json:"agent"`
	RequestHash string `json:"requestHash"`
	Seq         uint64 `json:"seq"`
	Status      uint8  `json:"status"`
	ResultHash  string `json:"resultHash"`
}

// ReceiptEnvelope is one answer on the wire.
type ReceiptEnvelope struct {
	Receipt ReceiptJSON `json:"receipt"`
	Result  []byte      `json:"result"`
	Sig     string      `json:"sig"`
}

func hex32(b [32]byte) string { return "0x" + hex.EncodeToString(b[:]) }

func parse32(s string) ([32]byte, error) {
	var out [32]byte
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 32 {
		return out, fmt.Errorf("intent: %q is not 32 bytes of hex", s)
	}
	copy(out[:], b)
	return out, nil
}

// JSON converts to the wire form.
func (i Intent) JSON() IntentJSON {
	return IntentJSON{Agent: i.Agent.Hex(), Controller: i.Controller.Hex(), Seq: i.Seq, Nonce: hex32(i.Nonce),
		IssuedAt: i.IssuedAt, Expiry: i.Expiry, Kind: i.Kind, PayloadHash: hex32(i.PayloadHash)}
}

// Parse validates and converts from the wire form.
func (j IntentJSON) Parse() (Intent, error) {
	var i Intent
	var err error
	if i.Agent, err = eip712.ParseAddress(j.Agent); err != nil {
		return i, err
	}
	if i.Controller, err = eip712.ParseAddress(j.Controller); err != nil {
		return i, err
	}
	if i.Nonce, err = parse32(j.Nonce); err != nil {
		return i, err
	}
	if i.PayloadHash, err = parse32(j.PayloadHash); err != nil {
		return i, err
	}
	if j.Kind == "" {
		return i, fmt.Errorf("intent: kind is empty")
	}
	i.Seq, i.IssuedAt, i.Expiry, i.Kind = j.Seq, j.IssuedAt, j.Expiry, j.Kind
	return i, nil
}

// JSON converts to the wire form.
func (r Receipt) JSON() ReceiptJSON {
	return ReceiptJSON{Agent: r.Agent.Hex(), RequestHash: hex32(r.RequestHash), Seq: r.Seq, Status: r.Status, ResultHash: hex32(r.ResultHash)}
}

// Parse validates and converts from the wire form.
func (j ReceiptJSON) Parse() (Receipt, error) {
	var r Receipt
	var err error
	if r.Agent, err = eip712.ParseAddress(j.Agent); err != nil {
		return r, err
	}
	if r.RequestHash, err = parse32(j.RequestHash); err != nil {
		return r, err
	}
	if r.ResultHash, err = parse32(j.ResultHash); err != nil {
		return r, err
	}
	r.Seq, r.Status = j.Seq, j.Status
	return r, nil
}
