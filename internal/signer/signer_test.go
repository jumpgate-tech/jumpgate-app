package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func cowKey(t *testing.T) *Key {
	t.Helper()
	k, err := KeyFromBytes(func() []byte { h := eip712.Keccak256([]byte("cow")); return h[:] }())
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func mail(t *testing.T) eip712.TypedData {
	t.Helper()
	a := func(s string) eip712.Address { x, _ := eip712.ParseAddress(s); return x }
	return eip712.TypedData{
		Types: eip712.Types{
			"EIP712Domain": {eip712.Field{Name: "name", Type: "string"}, eip712.Field{Name: "version", Type: "string"}, eip712.Field{Name: "chainId", Type: "uint256"}, eip712.Field{Name: "verifyingContract", Type: "address"}},
			"Person":       {eip712.Field{Name: "name", Type: "string"}, eip712.Field{Name: "wallet", Type: "address"}},
			"Mail":         {eip712.Field{Name: "from", Type: "Person"}, eip712.Field{Name: "to", Type: "Person"}, eip712.Field{Name: "contents", Type: "string"}},
		},
		PrimaryType: "Mail",
		Domain: map[string]any{"name": "Ether Mail", "version": "1", "chainId": big.NewInt(1),
			"verifyingContract": a("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC")},
		Message: map[string]any{
			"from":     map[string]any{"name": "Cow", "wallet": a("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826")},
			"to":       map[string]any{"name": "Bob", "wallet": a("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB")},
			"contents": "Hello, Bob!",
		},
	}
}

// The EIP-712 specification signs its Mail example with keccak256("cow").
// RFC 6979 signing is deterministic, so the signature must match exactly.
func TestSignsTheSpecificationVector(t *testing.T) {
	k := cowKey(t)
	if got := k.Address().Hex(); got != "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826" {
		t.Fatalf("address = %s", got)
	}
	sig, err := k.SignTypedData(context.Background(), mail(t))
	if err != nil {
		t.Fatal(err)
	}
	wantR := "4355c47d63924e8a72e509b65029052eb6c299d53a04e167c5775fd466751c9d"
	wantS := "07299936d304c153f6443dfa05f40ff007d72911b6f72307f996231605b91562"
	if hex.EncodeToString(sig[:32]) != wantR || hex.EncodeToString(sig[32:64]) != wantS || sig[64] != 28 {
		t.Fatalf("signature = %s", sig.Hex())
	}
}

func TestRecoverRoundTrip(t *testing.T) {
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	td := mail(t)
	sig, _ := k.SignTypedData(context.Background(), td)
	got, err := Recover(td, sig)
	if err != nil || got != k.Address() {
		t.Fatalf("Recover = %s, %v; want %s", got.Hex(), err, k.Address().Hex())
	}
	// A different message recovers a different address, never the signer.
	td.Message["contents"] = "Hello, Eve!"
	if got, _ := Recover(td, sig); got == k.Address() {
		t.Fatal("a signature verified for a message it did not sign")
	}
}

// Ethereum accepts only low-s signatures; the high-s twin of a valid signature
// is the classic malleability, and must not verify.
func TestRecoverRejectsHighS(t *testing.T) {
	k := cowKey(t)
	td := mail(t)
	sig, _ := k.SignTypedData(context.Background(), td)

	n, _ := new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	s := new(big.Int).SetBytes(sig[32:64])
	high := new(big.Int).Sub(n, s)
	var twin Signature
	copy(twin[:32], sig[:32])
	high.FillBytes(twin[32:64])
	twin[64] = 27 + (28 - sig[64]) // the twin's recovery id flips
	if _, err := Recover(td, twin); !errors.Is(err, ErrHighS) {
		t.Fatalf("high-s twin: err = %v, want ErrHighS", err)
	}
}

func TestRecoverRejectsABadV(t *testing.T) {
	sig, _ := cowKey(t).SignTypedData(context.Background(), mail(t))
	sig[64] = 1
	if _, err := Recover(mail(t), sig); !errors.Is(err, ErrBadV) {
		t.Fatalf("err = %v, want ErrBadV", err)
	}
}

// Review Focus 1: a key other users can read is refused on every OS,
// including Windows, where the mode check used to be skipped.
func TestLoadKeyFileRefusesASharedKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys", "controller.key")
	k, err := GenerateKeyFile(path)
	if err != nil {
		t.Fatal(err)
	}
	testutil.AssertPrivate(t, path)
	if _, err := GenerateKeyFile(path); err == nil {
		t.Fatal("GenerateKeyFile overwrote an existing key")
	}
	loaded, err := LoadKeyFile(path)
	if err != nil || loaded.Address() != k.Address() {
		t.Fatalf("LoadKeyFile = %v, %v", loaded, err)
	}
	testutil.Loosen(t, path)
	if _, err := LoadKeyFile(path); !errors.Is(err, ErrKeyFilePermissions) {
		t.Fatalf("LoadKeyFile on a shared key = %v, want ErrKeyFilePermissions", err)
	}
}

// M-4: a planted link is refused, not followed. Windows has no O_NOFOLLOW,
// so there the refusal is the Lstat before the open. Creating a symlink on
// Windows needs a privilege (or developer mode) the runner may lack; then
// there is nothing to test.
func TestLoadKeyFileRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.key")
	if _, err := GenerateKeyFile(real); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.key")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	if _, err := LoadKeyFile(link); err == nil || !strings.Contains(err.Error(), "is a link") {
		t.Fatalf("LoadKeyFile(symlink) = %v, want a refusal of the link", err)
	}
}
