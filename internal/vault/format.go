// Package vault keeps the controller's secrets in one encrypted file.
//
// The file is the magic line "JGVAULT1\n", one line of JSON header that lists
// key slots, then an XChaCha20-Poly1305 ciphertext of the secrets. The body
// is sealed under a random 32-byte data key (DEK), and every slot wraps that
// DEK under its own key-encryption key (KEK): argon2id over a passphrase, a
// systemd credential, or an OS key store item. The whole header line is the
// body's associated data, so any change to it makes decryption fail.
package vault

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	magic   = "JGVAULT1\n"
	version = 1
	keyLen  = 32
	saltLen = 16

	kdfAlg = "argon2id"
	minM   = 65536
	maxM   = 1048576
	minT   = 3
	maxT   = 16
	minP   = 1
	maxP   = 64
)

var (
	// ErrCorrupt means the file is not a vault this build can read, or it
	// was changed: bad magic or JSON, an unknown version, KDF parameters out
	// of bounds, a truncation, or an AEAD failure on the body.
	ErrCorrupt = errors.New("vault: the vault file is corrupt or was tampered with")
	// ErrWrongPassphrase means no passphrase slot unwrapped the data key.
	ErrWrongPassphrase = errors.New("vault: wrong passphrase")
	// ErrBadName means an entry name is not allowed.
	ErrBadName = errors.New("vault: bad entry name")
)

// SlotKind names how a slot's KEK is obtained.
type SlotKind string

const (
	SlotPassphrase   SlotKind = "passphrase"    // Label "primary" or "recovery"
	SlotSystemdCreds SlotKind = "systemd-creds" // Cred "jumpgate-vault-key"
	SlotKeystore     SlotKind = "keystore"      // Store "keychain"|"wincred"|"1password", Ref
)

type kdfParams struct {
	Alg  string `json:"alg"`
	M    uint32 `json:"m"`
	T    uint32 `json:"t"`
	P    uint8  `json:"p"`
	Salt []byte `json:"salt"`
}

type slotRec struct {
	Kind    string     `json:"kind"`
	Label   string     `json:"label,omitempty"`
	KDF     *kdfParams `json:"kdf,omitempty"`
	Cred    string     `json:"cred,omitempty"`
	Store   string     `json:"store,omitempty"`
	Ref     string     `json:"ref,omitempty"`
	Nonce   []byte     `json:"nonce"`
	Wrapped []byte     `json:"wrapped"`
}

type header struct {
	V     int       `json:"v"`
	Slots []slotRec `json:"slots"`
	Nonce []byte    `json:"nonce"`
}

type entryRec struct {
	Value   []byte    `json:"value"`
	Created time.Time `json:"created"`
	From    string    `json:"from,omitempty"`
}

type body struct {
	Entries map[string]entryRec `json:"entries"`
}

func corrupt(why string) error { return fmt.Errorf("%w: %s", ErrCorrupt, why) }

// checkKDF refuses parameters outside the floor and ceiling, before any KDF
// runs: a header can be neither downgraded nor made to exhaust memory.
func checkKDF(k kdfParams) error {
	switch {
	case k.Alg != kdfAlg:
		return corrupt("unknown KDF " + k.Alg)
	case len(k.Salt) != saltLen:
		return corrupt("bad KDF salt")
	case k.M < minM || k.M > maxM, k.T < minT || k.T > maxT, k.P < minP || k.P > maxP:
		return corrupt(fmt.Sprintf("KDF parameters m=%d t=%d p=%d out of bounds", k.M, k.T, k.P))
	}
	return nil
}

func deriveKEK(pass []byte, k kdfParams) []byte {
	return argon2.IDKey(pass, k.Salt, k.T, k.M, k.P, keyLen)
}

func slotAD(s slotRec) []byte {
	return []byte("jgvault-slot\x00" + s.Kind + "\x00" + s.Label)
}

// wrapDEK seals dek under kek into s, with a fresh nonce.
func wrapDEK(kek, dek []byte, s *slotRec) error {
	aead, err := chacha20poly1305.NewX(kek)
	if err != nil {
		return fmt.Errorf("vault: %w", err)
	}
	n := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(n); err != nil {
		return fmt.Errorf("vault: %w", err)
	}
	s.Nonce = n
	s.Wrapped = aead.Seal(nil, n, dek, slotAD(*s))
	return nil
}

// unwrapDEK opens s under kek. An AEAD failure is ErrWrongPassphrase: before
// the DEK is known nothing can tell a tampered wrap from a wrong key.
func unwrapDEK(kek []byte, s slotRec) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(kek)
	if err != nil {
		return nil, ErrWrongPassphrase
	}
	dek, err := aead.Open(nil, s.Nonce, s.Wrapped, slotAD(s))
	if err != nil {
		return nil, ErrWrongPassphrase
	}
	return dek, nil
}

// seal returns the whole file for b under dek, with a fresh body nonce.
func seal(h header, dek []byte, b body) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(dek)
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	h.V = version
	h.Nonce = make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(h.Nonce); err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	line, err := json.Marshal(h)
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	if b.Entries == nil {
		b.Entries = map[string]entryRec{}
	}
	pt, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	defer clear(pt)
	out := make([]byte, 0, len(magic)+len(line)+1+len(pt)+aead.Overhead())
	out = append(out, magic...)
	out = append(out, line...)
	out = append(out, '\n')
	return aead.Seal(out, h.Nonce, pt, line), nil
}

// parse splits a vault file and checks its header. Every passphrase slot's
// KDF is bounds-checked here, so no caller reaches deriveKEK with a bad one.
func parse(raw []byte) (h header, line, ct []byte, err error) {
	if !bytes.HasPrefix(raw, []byte(magic)) {
		return header{}, nil, nil, corrupt("not a jumpgate vault")
	}
	rest := raw[len(magic):]
	i := bytes.IndexByte(rest, '\n')
	if i < 0 {
		return header{}, nil, nil, corrupt("truncated header")
	}
	line, ct = rest[:i], rest[i+1:]
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&h); err != nil || dec.More() {
		return header{}, nil, nil, corrupt("bad header")
	}
	if h.V != version {
		return header{}, nil, nil, corrupt(fmt.Sprintf("version %d", h.V))
	}
	if len(h.Nonce) != chacha20poly1305.NonceSizeX || len(h.Slots) == 0 {
		return header{}, nil, nil, corrupt("bad header")
	}
	if len(ct) < chacha20poly1305.Overhead {
		return header{}, nil, nil, corrupt("truncated body")
	}
	for _, s := range h.Slots {
		if len(s.Nonce) != chacha20poly1305.NonceSizeX || len(s.Wrapped) != keyLen+chacha20poly1305.Overhead {
			return header{}, nil, nil, corrupt("bad slot")
		}
		switch SlotKind(s.Kind) {
		case SlotPassphrase:
			if s.KDF == nil {
				return header{}, nil, nil, corrupt("passphrase slot without KDF")
			}
			if err := checkKDF(*s.KDF); err != nil {
				return header{}, nil, nil, err
			}
		case SlotSystemdCreds, SlotKeystore:
			if s.KDF != nil {
				return header{}, nil, nil, corrupt("KDF on a key slot")
			}
		default:
			return header{}, nil, nil, corrupt("unknown slot kind")
		}
	}
	return h, line, ct, nil
}

// openBody decrypts ct with dek, the header line as associated data.
func openBody(line, ct, dek []byte, nonce []byte) (body, error) {
	aead, err := chacha20poly1305.NewX(dek)
	if err != nil {
		return body{}, corrupt("bad data key")
	}
	pt, err := aead.Open(nil, nonce, ct, line)
	if err != nil {
		return body{}, corrupt("body does not decrypt")
	}
	defer clear(pt)
	var b body
	if err := json.Unmarshal(pt, &b); err != nil {
		return body{}, corrupt("bad body")
	}
	if b.Entries == nil {
		b.Entries = map[string]entryRec{}
	}
	for n := range b.Entries {
		if validName(n) != nil {
			return body{}, corrupt("bad entry name")
		}
	}
	return b, nil
}

// validName enforces ^[A-Za-z0-9][A-Za-z0-9/_.-]{0,127}$.
func validName(n string) error {
	if len(n) == 0 || len(n) > 128 {
		return fmt.Errorf("%w: %q", ErrBadName, n)
	}
	for i := 0; i < len(n); i++ {
		c := n[i]
		alnum := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
		if !alnum && (i == 0 || (c != '/' && c != '_' && c != '.' && c != '-')) {
			return fmt.Errorf("%w: %q", ErrBadName, n)
		}
	}
	return nil
}
