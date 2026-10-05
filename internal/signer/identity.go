package signer

import (
	"errors"
	"fmt"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

// ErrAddressMismatch means a key store holds a key whose address is not the
// identity on record. Signing with it would act as someone else: every paired
// box would refuse it, and nothing would say why.
var ErrAddressMismatch = errors.New("signer: the key's address is not the recorded controller address")

// CheckAddress verifies that k is the identity recorded as want (any case).
func CheckAddress(k Signer, want string) error {
	rec, err := eip712.ParseAddress(want)
	if err != nil {
		return fmt.Errorf("recorded controller address %q: %w", want, err)
	}
	if got := k.Address(); got != rec {
		return fmt.Errorf("%w: the key store holds %s, config.json records %s", ErrAddressMismatch, got.Hex(), rec.Hex())
	}
	return nil
}
