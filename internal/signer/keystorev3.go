package signer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

// Web3 Secret Storage (keystore v3) limits on what DecryptKeystoreV3 will
// run: a hostile file must not make scrypt exhaust memory.
const (
	maxScryptN  = 1 << 20
	maxScryptRP = 64
)

type v3CipherParams struct {
	IV string `json:"iv"`
}

type v3KDFParams struct {
	DKLen int    `json:"dklen"`
	N     int    `json:"n"`
	R     int    `json:"r"`
	P     int    `json:"p"`
	Salt  string `json:"salt"`
}

type v3Crypto struct {
	Cipher       string          `json:"cipher"`
	CipherText   string          `json:"ciphertext"`
	CipherParams v3CipherParams  `json:"cipherparams"`
	KDF          string          `json:"kdf"`
	KDFParams    json.RawMessage `json:"kdfparams"`
	MAC          string          `json:"mac"`
}

type v3File struct {
	Address string   `json:"address,omitempty"`
	Crypto  v3Crypto `json:"crypto"`
	ID      string   `json:"id"`
	Version int      `json:"version"`
}

func v3Keys(pass []byte, salt []byte, n, r, p int) ([]byte, error) {
	return scrypt.Key(pass, salt, n, r, p, 32)
}

func v3MAC(dk, ct []byte) [32]byte { return eip712.Keccak256(dk[16:32], ct) }

func aesCTR(key, iv, in []byte) ([]byte, error) {
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(in))
	cipher.NewCTR(b, iv).XORKeyStream(out, in)
	return out, nil
}

// EncryptKeystoreV3 writes k as an Ethereum keystore v3 file: scrypt
// (n=scryptN, r=8, p=1, dklen=32), aes-128-ctr, and the keccak256 MAC. It
// clears passphrase.
func EncryptKeystoreV3(k *Key, passphrase []byte, scryptN int) ([]byte, error) {
	defer clear(passphrase)
	if len(passphrase) == 0 {
		return nil, errors.New("signer: empty keystore passphrase")
	}
	if scryptN < 2 || scryptN > maxScryptN || scryptN&(scryptN-1) != 0 {
		return nil, fmt.Errorf("signer: scrypt n %d is not a power of two up to %d", scryptN, maxScryptN)
	}
	salt, iv, id := make([]byte, 32), make([]byte, 16), make([]byte, 16)
	for _, b := range [][]byte{salt, iv, id} {
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
	}
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	dk, err := v3Keys(passphrase, salt, scryptN, 8, 1)
	if err != nil {
		return nil, err
	}
	defer clear(dk)
	priv := k.Bytes()
	defer clear(priv)
	ct, err := aesCTR(dk[:16], iv, priv)
	if err != nil {
		return nil, err
	}
	mac := v3MAC(dk, ct)
	kp, _ := json.Marshal(v3KDFParams{DKLen: 32, N: scryptN, R: 8, P: 1, Salt: hex.EncodeToString(salt)})
	h := hex.EncodeToString(id)
	f := v3File{
		Address: strings.ToLower(strings.TrimPrefix(k.Address().Hex(), "0x")),
		Crypto: v3Crypto{
			Cipher: "aes-128-ctr", CipherText: hex.EncodeToString(ct),
			CipherParams: v3CipherParams{IV: hex.EncodeToString(iv)},
			KDF:          "scrypt", KDFParams: kp, MAC: hex.EncodeToString(mac[:]),
		},
		ID:      h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:],
		Version: 3,
	}
	return json.MarshalIndent(f, "", "  ")
}

// DecryptKeystoreV3 reads a keystore v3 file made with scrypt and
// aes-128-ctr; pbkdf2 and other ciphers are refused by name. It clears
// passphrase.
func DecryptKeystoreV3(b, passphrase []byte) (*Key, error) {
	defer clear(passphrase)
	var f v3File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("signer: not a keystore v3 file: %w", err)
	}
	if f.Version != 3 {
		return nil, fmt.Errorf("signer: keystore version %d, want 3", f.Version)
	}
	c := f.Crypto
	if c.KDF != "scrypt" {
		return nil, fmt.Errorf("signer: keystore kdf %q is not supported; only scrypt is", c.KDF)
	}
	if c.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("signer: keystore cipher %q is not supported; only aes-128-ctr is", c.Cipher)
	}
	var kp v3KDFParams
	if err := json.Unmarshal(c.KDFParams, &kp); err != nil {
		return nil, fmt.Errorf("signer: keystore kdfparams: %w", err)
	}
	if kp.DKLen != 32 || kp.N < 2 || kp.N > maxScryptN || kp.N&(kp.N-1) != 0 ||
		kp.R < 1 || kp.P < 1 || kp.R*kp.P > maxScryptRP {
		return nil, fmt.Errorf("signer: keystore scrypt parameters n=%d r=%d p=%d dklen=%d are out of bounds", kp.N, kp.R, kp.P, kp.DKLen)
	}
	salt, err1 := hex.DecodeString(kp.Salt)
	iv, err2 := hex.DecodeString(c.CipherParams.IV)
	ct, err3 := hex.DecodeString(c.CipherText)
	mac, err4 := hex.DecodeString(c.MAC)
	if err := errors.Join(err1, err2, err3, err4); err != nil || len(iv) != 16 || len(ct) != 32 || len(mac) != 32 {
		return nil, errors.New("signer: keystore fields are not well-formed hex")
	}
	dk, err := v3Keys(passphrase, salt, kp.N, kp.R, kp.P)
	if err != nil {
		return nil, err
	}
	defer clear(dk)
	want := v3MAC(dk, ct)
	if subtle.ConstantTimeCompare(want[:], mac) != 1 {
		return nil, errors.New("signer: keystore mac mismatch: wrong passphrase or a damaged file")
	}
	priv, err := aesCTR(dk[:16], iv, ct)
	if err != nil {
		return nil, err
	}
	defer clear(priv)
	k, err := KeyFromBytes(priv)
	if err != nil {
		return nil, err
	}
	if f.Address != "" && !strings.EqualFold(strings.TrimPrefix(f.Address, "0x"), strings.TrimPrefix(k.Address().Hex(), "0x")) {
		return nil, errors.New("signer: the keystore's address does not match its key")
	}
	return k, nil
}
