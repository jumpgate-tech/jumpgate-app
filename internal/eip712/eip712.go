// internal/eip712/eip712.go

// Package eip712 hashes EIP-712 typed data for the fixed set of types jumpgate
// signs. It is deliberately small: the types are ours, so arrays and dynamic
// bytes, which none of them use, are refused rather than half-supported.
package eip712

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"golang.org/x/crypto/sha3"
)

// Field is one member of a struct type.
type Field struct{ Name, Type string }

// Types maps a struct type name to its fields, in declaration order.
type Types map[string][]Field

// TypedData is what a wallet's eth_signTypedData_v4 receives.
type TypedData struct {
	Types       Types
	PrimaryType string
	Domain      map[string]any
	Message     map[string]any
}

// Keccak256 hashes the concatenation of parts.
func Keccak256(parts ...[]byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	for _, p := range parts {
		h.Write(p)
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// Digest is keccak256(0x19 0x01 ‖ domainSeparator ‖ hashStruct(message)).
func Digest(td TypedData) ([32]byte, error) {
	domain, err := HashStruct(td.Types, "EIP712Domain", td.Domain)
	if err != nil {
		return [32]byte{}, fmt.Errorf("eip712: domain: %w", err)
	}
	msg, err := HashStruct(td.Types, td.PrimaryType, td.Message)
	if err != nil {
		return [32]byte{}, fmt.Errorf("eip712: message: %w", err)
	}
	return Keccak256([]byte{0x19, 0x01}, domain[:], msg[:]), nil
}

// EncodeType renders primary's signature followed by every struct type it
// references, the references sorted by name, as the EIP requires.
func EncodeType(types Types, primary string) (string, error) {
	if _, ok := types[primary]; !ok {
		return "", fmt.Errorf("eip712: unknown type %q", primary)
	}
	deps := map[string]bool{}
	if err := collectDeps(types, primary, deps); err != nil {
		return "", err
	}
	delete(deps, primary)
	names := make([]string, 0, len(deps))
	for n := range deps {
		names = append(names, n)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range append([]string{primary}, names...) {
		b.WriteString(name)
		b.WriteByte('(')
		for i, f := range types[name] {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(f.Type + " " + f.Name)
		}
		b.WriteByte(')')
	}
	return b.String(), nil
}

func collectDeps(types Types, name string, seen map[string]bool) error {
	if seen[name] {
		return nil
	}
	seen[name] = true
	for _, f := range types[name] {
		if strings.Contains(f.Type, "[") {
			return fmt.Errorf("eip712: %s.%s: array types are not supported", name, f.Name)
		}
		if _, isStruct := types[f.Type]; isStruct {
			if err := collectDeps(types, f.Type, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// HashStruct is keccak256(typeHash ‖ encodeData(data)). data must hold
// exactly the type's fields, each as the Go type listed in the package doc.
func HashStruct(types Types, primary string, data map[string]any) ([32]byte, error) {
	enc, err := EncodeType(types, primary)
	if err != nil {
		return [32]byte{}, err
	}
	fields := types[primary]
	if len(data) != len(fields) {
		return [32]byte{}, fmt.Errorf("eip712: %s has %d fields, got %d values", primary, len(fields), len(data))
	}
	typeHash := Keccak256([]byte(enc))
	buf := make([]byte, 0, 32*(len(fields)+1))
	buf = append(buf, typeHash[:]...)
	for _, f := range fields {
		v, ok := data[f.Name]
		if !ok {
			return [32]byte{}, fmt.Errorf("eip712: %s.%s is missing", primary, f.Name)
		}
		word, err := encodeValue(types, f.Type, v)
		if err != nil {
			return [32]byte{}, fmt.Errorf("eip712: %s.%s: %w", primary, f.Name, err)
		}
		buf = append(buf, word[:]...)
	}
	return Keccak256(buf), nil
}

func encodeValue(types Types, typ string, v any) ([32]byte, error) {
	var word [32]byte
	if _, isStruct := types[typ]; isStruct {
		m, ok := v.(map[string]any)
		if !ok {
			return word, fmt.Errorf("want map[string]any for %s, got %T", typ, v)
		}
		return HashStruct(types, typ, m)
	}
	switch typ {
	case "string":
		s, ok := v.(string)
		if !ok {
			return word, fmt.Errorf("want string, got %T", v)
		}
		return Keccak256([]byte(s)), nil
	case "bytes32":
		b, ok := v.([32]byte)
		if !ok {
			return word, fmt.Errorf("want [32]byte, got %T", v)
		}
		return b, nil
	case "address":
		a, ok := v.(Address)
		if !ok {
			return word, fmt.Errorf("want eip712.Address, got %T", v)
		}
		copy(word[12:], a[:])
		return word, nil
	case "bool":
		b, ok := v.(bool)
		if !ok {
			return word, fmt.Errorf("want bool, got %T", v)
		}
		if b {
			word[31] = 1
		}
		return word, nil
	case "uint8":
		n, ok := v.(uint8)
		if !ok {
			return word, fmt.Errorf("want uint8, got %T", v)
		}
		word[31] = n
		return word, nil
	case "uint64":
		n, ok := v.(uint64)
		if !ok {
			return word, fmt.Errorf("want uint64, got %T", v)
		}
		new(big.Int).SetUint64(n).FillBytes(word[:])
		return word, nil
	case "uint256":
		n, ok := v.(*big.Int)
		if !ok || n == nil {
			return word, fmt.Errorf("want *big.Int, got %T", v)
		}
		if n.Sign() < 0 || n.BitLen() > 256 {
			return word, fmt.Errorf("value %s out of uint256 range", n)
		}
		n.FillBytes(word[:])
		return word, nil
	default:
		return word, fmt.Errorf("unsupported type %q", typ)
	}
}
