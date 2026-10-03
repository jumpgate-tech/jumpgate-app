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
