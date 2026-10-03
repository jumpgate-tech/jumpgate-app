// internal/eip712/address.go
package eip712

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Address is a 20-byte Ethereum address.
type Address [20]byte

// ParseAddress reads a 0x-prefixed, 40-hex-digit address in any case.
func ParseAddress(s string) (Address, error) {
	var a Address
	raw := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(raw) != 40 {
		return a, fmt.Errorf("eip712: address %q is not 20 bytes", s)
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return a, fmt.Errorf("eip712: address %q: %w", s, err)
	}
	copy(a[:], b)
	return a, nil
}

// IsZero reports whether a is the zero address.
func (a Address) IsZero() bool { return a == Address{} }

// Hex is the EIP-55 checksummed form, the one wallets display.
func (a Address) Hex() string {
	lower := hex.EncodeToString(a[:])
	hash := Keccak256([]byte(lower))
	out := []byte(lower)
	for i := range out {
		if out[i] >= 'a' && out[i] <= 'f' {
			nibble := hash[i/2]
			if i%2 == 0 {
				nibble >>= 4
			}
			if nibble&0x0f >= 8 {
				out[i] -= 'a' - 'A'
			}
		}
	}
	return "0x" + string(out)
}

func (a Address) String() string { return a.Hex() }
