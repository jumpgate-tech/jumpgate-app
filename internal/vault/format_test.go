package vault

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testPass = []byte("correct horse battery staple")
	testKDF  = kdfParams{Alg: "argon2id", M: 65536, T: 3, P: 4, Salt: bytes.Repeat([]byte{7}, 16)}

	kekOnce sync.Once
	testKEK []byte
)

// fixtureKEK runs the KDF once for the whole file.
func fixtureKEK(t *testing.T) []byte {
	t.Helper()
	kekOnce.Do(func() { testKEK = deriveKEK(testPass, testKDF) })
	return bytes.Clone(testKEK)
}

func newDEK(t *testing.T) []byte {
	t.Helper()
	dek := make([]byte, keyLen)
	if _, err := rand.Read(dek); err != nil {
		t.Fatal(err)
	}
	return dek
}

// sealed builds a one-slot vault file under the fixture passphrase.
func sealed(t *testing.T, dek []byte, b body) []byte {
	t.Helper()
	k := testKDF
	s := slotRec{Kind: string(SlotPassphrase), Label: "primary", KDF: &k}
	if err := wrapDEK(fixtureKEK(t), dek, &s); err != nil {
		t.Fatal(err)
	}
	raw, err := seal(header{V: 1, Slots: []slotRec{s}}, dek, b)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func sampleBody() body {
	return body{Entries: map[string]entryRec{
		"ai-key":         {Value: []byte("sk-123"), Created: time.Unix(1700000000, 0).UTC()},
		"controller-key": {Value: []byte{0, 1, 2, 255}, Created: time.Unix(1700000001, 0).UTC(), From: "file"},
	}}
}

// decode runs the whole read path: parse, unwrap the first slot, open the body.
func decode(t *testing.T, raw []byte, kek []byte) (body, error) {
	t.Helper()
	h, line, ct, err := parse(raw)
	if err != nil {
		return body{}, err
	}
	dek, err := unwrapDEK(kek, h.Slots[0])
	if err != nil {
		return body{}, err
	}
	return openBody(line, ct, dek, h.Nonce)
}

func TestSealParseRoundTrip(t *testing.T) {
	raw := sealed(t, newDEK(t), sampleBody())
	if !bytes.HasPrefix(raw, []byte("JGVAULT1\n")) {
		t.Fatalf("missing magic: %q", raw[:16])
	}
	b, err := decode(t, raw, fixtureKEK(t))
	if err != nil {
		t.Fatal(err)
	}
	want := sampleBody()
	if len(b.Entries) != len(want.Entries) {
		t.Fatalf("entries = %d, want %d", len(b.Entries), len(want.Entries))
	}
	for n, e := range want.Entries {
		g := b.Entries[n]
		if !bytes.Equal(g.Value, e.Value) || !g.Created.Equal(e.Created) || g.From != e.From {
			t.Fatalf("entry %s = %+v, want %+v", n, g, e)
		}
	}
}

func TestEveryWriteDrawsAFreshNonce(t *testing.T) {
	dek := newDEK(t)
	a, b := sealed(t, dek, sampleBody()), sealed(t, dek, sampleBody())
	ha, _, cta, err := parse(a)
	if err != nil {
		t.Fatal(err)
	}
	hb, _, ctb, err := parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ha.Nonce, hb.Nonce) {
		t.Fatal("header nonces repeat")
	}
	if bytes.Equal(ha.Slots[0].Nonce, hb.Slots[0].Nonce) {
		t.Fatal("slot nonces repeat")
	}
	if bytes.Equal(cta, ctb) {
		t.Fatal("ciphertexts repeat")
	}
}

func TestWrongPassphraseIsNotCorrupt(t *testing.T) {
	raw := sealed(t, newDEK(t), sampleBody())
	_, err := decode(t, raw, deriveKEK([]byte("wrong"), testKDF))
	if !errors.Is(err, ErrWrongPassphrase) || errors.Is(err, ErrCorrupt) {
		t.Fatalf("err = %v, want ErrWrongPassphrase only", err)
	}
}

// rewriteHeader re-encodes raw with its header changed by fn and the body kept.
func rewriteHeader(t *testing.T, raw []byte, fn func(*header)) []byte {
	t.Helper()
	h, _, ct, err := parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	fn(&h)
	line, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	out := append([]byte(magic), line...)
	out = append(out, '\n')
	return append(out, ct...)
}

func TestTamperingIsCorrupt(t *testing.T) {
	dek := newDEK(t)
	raw := sealed(t, dek, sampleBody())
	ctStart := len(magic) + bytes.IndexByte(raw[len(magic):], '\n') + 1
	other := sealed(t, newDEK(t), sampleBody()) // same passphrase, another DEK
	ho, _, _, err := parse(other)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func() []byte{
		"header nonce bit": func() []byte {
			return rewriteHeader(t, raw, func(h *header) { h.Nonce[0] ^= 1 })
		},
		"ciphertext bit": func() []byte { c := bytes.Clone(raw); c[ctStart] ^= 1; return c },
		"tag bit":        func() []byte { c := bytes.Clone(raw); c[len(c)-1] ^= 0x80; return c },
		"truncated":      func() []byte { return bytes.Clone(raw[:len(raw)-1]) },
		"truncated header": func() []byte {
			return bytes.Clone(raw[:len(magic)+10])
		},
		"magic":        func() []byte { c := bytes.Clone(raw); c[0] = 'X'; return c },
		"swapped slot": func() []byte { return rewriteHeader(t, raw, func(h *header) { h.Slots[0] = ho.Slots[0] }) },
		"version 2":    func() []byte { return rewriteHeader(t, raw, func(h *header) { h.V = 2 }) },
		"empty":        func() []byte { return nil },
		"no body":      func() []byte { return bytes.Clone(raw[:ctStart]) },
	}
	for name, mk := range cases {
		t.Run(name, func(t *testing.T) {
			b, err := decode(t, mk(), fixtureKEK(t))
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("err = %v, want ErrCorrupt", err)
			}
			if b.Entries != nil {
				t.Fatal("a body came back from a tampered file")
			}
		})
	}
}

func TestKDFBoundsAreEnforced(t *testing.T) {
	raw := sealed(t, newDEK(t), sampleBody())
	cases := map[string]func(*kdfParams){
		"m=65535":   func(k *kdfParams) { k.M = 65535 },
		"t=2":       func(k *kdfParams) { k.T = 2 },
		"p=0":       func(k *kdfParams) { k.P = 0 },
		"m=1048577": func(k *kdfParams) { k.M = 1048577 },
		"t=17":      func(k *kdfParams) { k.T = 17 },
		"p=65":      func(k *kdfParams) { k.P = 65 },
		"alg":       func(k *kdfParams) { k.Alg = "scrypt" },
		"salt":      func(k *kdfParams) { k.Salt = k.Salt[:8] },
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			bad := rewriteHeader(t, raw, func(h *header) {
				k := *h.Slots[0].KDF
				mut(&k)
				h.Slots[0].KDF = &k
			})
			start := time.Now()
			_, _, _, err := parse(bad)
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("err = %v, want ErrCorrupt", err)
			}
			if d := time.Since(start); d > 50*time.Millisecond {
				t.Fatalf("parse took %v; a KDF must not run", d)
			}
		})
	}
	if checkKDF(testKDF) != nil {
		t.Fatal("the defaults are refused")
	}
}

func TestEntryNamesAreValidated(t *testing.T) {
	for _, n := range []string{"provider/VALVE_API_KEY", "vpn/proton-1", "ai-key", "a", strings.Repeat("a", 128)} {
		if err := validName(n); err != nil {
			t.Errorf("validName(%q) = %v", n, err)
		}
	}
	for _, n := range []string{"", "/x", "a b", strings.Repeat("a", 129), "x\x00", "-x", "é"} {
		if err := validName(n); !errors.Is(err, ErrBadName) {
			t.Errorf("validName(%q) = %v, want ErrBadName", n, err)
		}
	}
}
