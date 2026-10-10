// internal/eip712/eip712_test.go
package eip712

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
)

// The Mail example from the EIP-712 specification (its Example.js). If any
// constant below ever disagrees with this code, re-check it against
// https://eips.ethereum.org/EIPS/eip-712 before touching the encoder.
func mailTypedData(t *testing.T) TypedData {
	t.Helper()
	addr := func(s string) Address {
		a, err := ParseAddress(s)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	return TypedData{
		Types: Types{
			"EIP712Domain": {{"name", "string"}, {"version", "string"}, {"chainId", "uint256"}, {"verifyingContract", "address"}},
			"Person":       {{"name", "string"}, {"wallet", "address"}},
			"Mail":         {{"from", "Person"}, {"to", "Person"}, {"contents", "string"}},
		},
		PrimaryType: "Mail",
		Domain: map[string]any{
			"name": "Ether Mail", "version": "1", "chainId": big.NewInt(1),
			"verifyingContract": addr("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"),
		},
		Message: map[string]any{
			"from":     map[string]any{"name": "Cow", "wallet": addr("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826")},
			"to":       map[string]any{"name": "Bob", "wallet": addr("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB")},
			"contents": "Hello, Bob!",
		},
	}
}

func h32(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 32 {
		t.Fatalf("bad hex %q", s)
	}
	var out [32]byte
	copy(out[:], b)
	return out
}

func TestEncodeTypeOrdersDependencies(t *testing.T) {
	got, err := EncodeType(mailTypedData(t).Types, "Mail")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Mail(Person from,Person to,string contents)Person(string name,address wallet)"; got != want {
		t.Fatalf("EncodeType = %q, want %q", got, want)
	}
}

func TestMailVectors(t *testing.T) {
	td := mailTypedData(t)

	typeHash := Keccak256([]byte("Mail(Person from,Person to,string contents)Person(string name,address wallet)"))
	if want := h32(t, "0xa0cedeb2dc280ba39b857546d74f5549c3a1d7bdc2dd96bf881f76108e23dac2"); typeHash != want {
		t.Fatalf("Mail typeHash = %x", typeHash)
	}

	domain, err := HashStruct(td.Types, "EIP712Domain", td.Domain)
	if err != nil {
		t.Fatal(err)
	}
	if want := h32(t, "0xf2cee375fa42b42143804025fc449deafd50cc031ca257e0b194a650a912090f"); domain != want {
		t.Errorf("domainSeparator = %x", domain)
	}

	msg, err := HashStruct(td.Types, "Mail", td.Message)
	if err != nil {
		t.Fatal(err)
	}
	if want := h32(t, "0xc52c0ee5d84264471806290a3f2c4cecfc5490626bf912d01f240d7a274b371e"); msg != want {
		t.Errorf("hashStruct(message) = %x", msg)
	}

	digest, err := Digest(td)
	if err != nil {
		t.Fatal(err)
	}
	if want := h32(t, "0xbe609aee343fb3c4b28e1df9e632fca64fcfaede20f02e86244efddf30957bd2"); digest != want {
		t.Errorf("digest = %x", digest)
	}
}

func TestHashStructRejectsWrongShapes(t *testing.T) {
	types := Types{"T": {{"n", "uint64"}, {"s", "string"}}}
	cases := map[string]map[string]any{
		"missing field": {"n": uint64(1)},
		"wrong go type": {"n": 1, "s": "x"},
		"extra field":   {"n": uint64(1), "s": "x", "z": "y"},
	}
	for name, msg := range cases {
		if _, err := HashStruct(types, "T", msg); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := HashStruct(Types{"T": {{"xs", "uint64[]"}}}, "T", map[string]any{"xs": []uint64{1}}); err == nil {
		t.Error("arrays must be rejected, not mis-encoded")
	}
	if _, err := HashStruct(Types{"T": {{"n", "uint8"}}}, "T", map[string]any{"n": uint64(300)}); err == nil {
		t.Error("a uint64 value must not be accepted for a uint8 field")
	}
}

func TestAddressChecksum(t *testing.T) {
	a, err := ParseAddress("0xcd2a3d9f938e13cd947ec05abc7fe734df8dd826")
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Hex(); got != "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826" {
		t.Fatalf("Hex() = %s", got)
	}
	if _, err := ParseAddress("0x1234"); err == nil {
		t.Error("short address accepted")
	}
}

// A value of the wrong Go type for its field fails the hash rather than
// being coerced: a signature over a silently mis-encoded message would
// verify a request nobody meant.
func TestEveryFieldTypeRefusesAWrongGoType(t *testing.T) {
	types := Types{"Inner": {{"n", "uint64"}}}
	for typ, wrong := range map[string]any{
		"string":  []byte("s"),
		"bytes32": []byte{1},
		"address": "0x0000000000000000000000000000000000000001",
		"bool":    1,
		"uint8":   uint64(1),
		"uint64":  uint32(1),
		"uint256": uint64(1),
		"Inner":   struct{ N uint64 }{1},
		"int256":  big.NewInt(1), // a type this package does not encode at all
	} {
		ts := Types{"T": {{"v", typ}}, "Inner": types["Inner"]}
		if _, err := HashStruct(ts, "T", map[string]any{"v": wrong}); err == nil {
			t.Errorf("%s accepted a %T", typ, wrong)
		}
	}
}

// uint256 is the only width a value can overflow: negative numbers, values
// past 2^256 and a nil pointer are refused; the largest value is encoded.
func TestUint256Range(t *testing.T) {
	ts := Types{"T": {{"v", "uint256"}}}
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	for name, v := range map[string]*big.Int{
		"negative":  big.NewInt(-1),
		"2^256":     new(big.Int).Lsh(big.NewInt(1), 256),
		"nil value": nil,
	} {
		if _, err := HashStruct(ts, "T", map[string]any{"v": v}); err == nil {
			t.Errorf("%s was encoded", name)
		}
	}
	if _, err := HashStruct(ts, "T", map[string]any{"v": max}); err != nil {
		t.Fatalf("2^256-1: %v", err)
	}
}

// Words are left-padded big-endian, as the EIP's encodeData requires: the
// same small number as uint8, uint64 and uint256 hashes the same, true is 1,
// and an address sits in the low 20 bytes.
func TestSmallValuesEncodeToTheSameWord(t *testing.T) {
	word := func(typ string, v any) [32]byte {
		t.Helper()
		w, err := encodeValue(Types{}, typ, v)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	var seven [32]byte
	seven[31] = 7
	for typ, v := range map[string]any{"uint8": uint8(7), "uint64": uint64(7), "uint256": big.NewInt(7)} {
		if got := word(typ, v); got != seven {
			t.Errorf("%s 7 = %x", typ, got)
		}
	}
	var one, zero [32]byte
	one[31] = 1
	if word("bool", true) != one || word("bool", false) != zero {
		t.Error("bool is not 0 or 1 in the last byte")
	}
	var a Address
	a[0], a[19] = 0xAB, 0xCD
	if w := word("address", a); w[12] != 0xAB || w[31] != 0xCD || w[11] != 0 {
		t.Errorf("address word %x", w)
	}
}

// Digest names which half failed: the domain or the message.
func TestDigestNamesTheHalfThatFailed(t *testing.T) {
	td := mailTypedData(t)
	td.Domain = map[string]any{}
	if _, err := Digest(td); err == nil || !strings.Contains(err.Error(), "domain") {
		t.Errorf("bad domain: %v", err)
	}
	td = mailTypedData(t)
	td.Message = map[string]any{}
	if _, err := Digest(td); err == nil || !strings.Contains(err.Error(), "message") {
		t.Errorf("bad message: %v", err)
	}
	if _, err := EncodeType(Types{}, "Missing"); err == nil {
		t.Error("an unknown primary type was encoded")
	}
}
