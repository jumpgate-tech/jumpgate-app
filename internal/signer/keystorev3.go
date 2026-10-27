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
// run, checked before any derivation: scrypt takes about 128*N*r bytes, so
// N*r is capped at maxScryptNR (2^18 at r=8, geth's standard cost: 256 MiB),
// and p, which multiplies the time, at maxScryptP.
const (
	maxScryptN  = 1 << 18
	maxScryptNR = 1 << 18 * 8
	maxScryptP  = 16
)

// scryptParamsOK reports whether n, r, p, dklen are within those limits: n a
// power of two above 1, r and p at least 1, dklen 32.
func scryptParamsOK(n, r, p, dklen int) bool {
	return dklen == 32 && n > 1 && n&(n-1) == 0 && n <= maxScryptNR &&
		r >= 1 && r <= maxScryptNR/n && p >= 1 && p <= maxScryptP
}

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
	if !scryptParamsOK(kp.N, kp.R, kp.P, kp.DKLen) {
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
