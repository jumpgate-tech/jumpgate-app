package signer

import (
	"context"
	"errors"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

// Key is a secp256k1 private key held in memory.
type Key struct {
	priv *secp256k1.PrivateKey
	addr eip712.Address
}

// GenerateKey makes a new random key.
func GenerateKey() (*Key, error) {
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	return newKey(priv), nil
}

// KeyFromBytes loads a 32-byte scalar, rejecting zero and values at or above
// the curve order.
func KeyFromBytes(b []byte) (*Key, error) {
	if len(b) != 32 {
		return nil, errors.New("signer: a private key is 32 bytes")
	}
	var s secp256k1.ModNScalar
	if overflow := s.SetByteSlice(b); overflow || s.IsZero() {
		return nil, errors.New("signer: private key out of range")
	}
	return newKey(secp256k1.NewPrivateKey(&s)), nil
}

func newKey(priv *secp256k1.PrivateKey) *Key {
	return &Key{priv: priv, addr: addressOf(priv.PubKey())}
}

// Bytes is the 32-byte scalar. Handle with care.
func (k *Key) Bytes() []byte { return k.priv.Serialize() }

// Address is the key's Ethereum address.
func (k *Key) Address() eip712.Address { return k.addr }

// SignDigest signs d deterministically (RFC 6979) with a low s.
func (k *Key) SignDigest(d [32]byte) Signature {
	compact := ecdsa.SignCompact(k.priv, d[:], false) // [27+recid] ‖ r ‖ s
	var sig Signature
	copy(sig[:64], compact[1:])
	sig[64] = compact[0]
	return sig
}

// SignTypedData signs td's EIP-712 digest.
func (k *Key) SignTypedData(_ context.Context, td eip712.TypedData) (Signature, error) {
	d, err := eip712.Digest(td)
	if err != nil {
		return Signature{}, err
	}
	return k.SignDigest(d), nil
}
