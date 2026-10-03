// Package signer signs and verifies jumpgate's EIP-712 messages with
// secp256k1 keys, the same scheme every Ethereum wallet uses, so a hardware
// wallet can later sign the same messages a key file does.
package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

// Signature is r ‖ s ‖ v with v ∈ {27, 28}, the layout wallets return.
type Signature [65]byte

// Signer is anything that can sign typed data for one address.
type Signer interface {
	Address() eip712.Address
	SignTypedData(ctx context.Context, td eip712.TypedData) (Signature, error)
}

var (
	// ErrHighS rejects the malleable twin of a valid signature.
	ErrHighS = errors.New("signer: signature s is in the upper half of the curve order")
	// ErrBadV rejects a recovery byte other than 27 or 28.
	ErrBadV = errors.New("signer: signature v must be 27 or 28")
)

// Hex is the 0x-prefixed 130-digit form.
func (s Signature) Hex() string { return "0x" + hex.EncodeToString(s[:]) }

// ParseSignature reads the Hex form.
func ParseSignature(s string) (Signature, error) {
	var sig Signature
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 65 {
		return sig, fmt.Errorf("signer: signature %q is not 65 bytes of hex", s)
	}
	copy(sig[:], b)
	return sig, nil
}

// RecoverDigest returns the address that produced sig over d.
func RecoverDigest(d [32]byte, sig Signature) (eip712.Address, error) {
	var zero eip712.Address
	if sig[64] != 27 && sig[64] != 28 {
		return zero, ErrBadV
	}
	var s secp256k1.ModNScalar
	if overflow := s.SetByteSlice(sig[32:64]); overflow || s.IsOverHalfOrder() {
		return zero, ErrHighS
	}
	compact := make([]byte, 65)
	compact[0] = sig[64] // 27 + recovery id, uncompressed: decred's compact header
	copy(compact[1:], sig[:64])
	pub, _, err := ecdsa.RecoverCompact(compact, d[:])
	if err != nil {
		return zero, fmt.Errorf("signer: recover: %w", err)
	}
	return addressOf(pub), nil
}

// Recover returns the address that signed td.
func Recover(td eip712.TypedData, sig Signature) (eip712.Address, error) {
	d, err := eip712.Digest(td)
	if err != nil {
		return eip712.Address{}, err
	}
	return RecoverDigest(d, sig)
}

func addressOf(pub *secp256k1.PublicKey) eip712.Address {
	uncompressed := pub.SerializeUncompressed() // 0x04 ‖ X ‖ Y
	h := eip712.Keccak256(uncompressed[1:])
	var a eip712.Address
	copy(a[:], h[12:])
	return a
}
