package signer

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// The scrypt test vector of the Web3 Secret Storage Definition
// (ethereum.org / the ethereum wiki). Note its kdfparams are r=1, p=8.
const v3Vector = `{
  "crypto": {
    "cipher": "aes-128-ctr",
    "cipherparams": {"iv": "83dbcc02d8ccb40e466191a123791e0e"},
    "ciphertext": "d172bf743a674da9cdad04534d56926ef8358534d458fffccd4e6ad2fbde479c",
    "kdf": "scrypt",
    "kdfparams": {"dklen": 32, "n": 262144, "r": 1, "p": 8, "salt": "ab0c7876052600dd703518d6fc3fe8984592145b591fc8fb5c6d43190334ba19"},
    "mac": "2103ac29920d71da29f15d75b4a16dbe95cfd7ff8faea1056c33131d846e3097"
  },
  "id": "3198bc9c-6672-5ab3-d995-4942343ae5b6",
  "version": 3
}`

// scrypt needs about 128*N*r bytes: a crafted file asking for more than
// N=2^18 at r=8 (256 MiB), or with malformed parameters, is refused before
// any derivation, quickly.
func TestKeystoreV3CapsScryptMemory(t *testing.T) {
	cases := map[string][2]string{
		"n=2^19 r=8":  {`"n": 262144, "r": 1`, `"n": 524288, "r": 8`},
		"n=2^18 r=16": {`"r": 1`, `"r": 16`},
		"n=2^30 r=1":  {`"n": 262144`, `"n": 1073741824`},
		"n not pow2":  {`"n": 262144`, `"n": 262143`},
		"n=1":         {`"n": 262144`, `"n": 1`},
		"r=0":         {`"r": 1`, `"r": 0`},
		"negative r":  {`"r": 1`, `"r": -8`},
		"p=0":         {`"p": 8`, `"p": 0`},
		"p absurd":    {`"p": 8`, `"p": 1000000`},
		"dklen 64":    {`"dklen": 32`, `"dklen": 64`},
	}
	for name, c := range cases {
		f := strings.Replace(v3Vector, c[0], c[1], 1)
		if f == v3Vector {
			t.Fatalf("%s: the vector has no %s", name, c[0])
		}
		start := time.Now()
		_, err := DecryptKeystoreV3([]byte(f), []byte("x"))
		if err == nil || !strings.Contains(err.Error(), "out of bounds") {
			t.Errorf("%s: %v", name, err)
		}
		if d := time.Since(start); d > 100*time.Millisecond {
			t.Errorf("%s: took %v", name, d)
		}
	}
}

func TestKeystoreV3KnownVector(t *testing.T) {
	k, err := DecryptKeystoreV3([]byte(v3Vector), []byte("testpassword"))
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(k.Bytes()); got != "7a28b5ba57c53603b0b07b56bba752f7784bf506fa95edc395f5cf6c7514fe9d" {
		t.Fatalf("key %s", got)
	}
}

func TestKeystoreV3RoundTrip(t *testing.T) {
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncryptKeystoreV3(k, []byte("pw"), 1<<12)
	if err != nil {
		t.Fatal(err)
	}
	var f map[string]any
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	if f["address"] != strings.ToLower(strings.TrimPrefix(k.Address().Hex(), "0x")) || f["version"] != float64(3) {
		t.Fatalf("file %s", b)
	}
	if id, _ := f["id"].(string); len(id) != 36 || id[14] != '4' {
		t.Fatalf("id %q is not a uuid v4", id)
	}
	if strings.Contains(string(b), hex.EncodeToString(k.Bytes())) {
		t.Fatal("the plain key is in the file")
	}
	got, err := DecryptKeystoreV3(b, []byte("pw"))
	if err != nil || got.Address() != k.Address() {
		t.Fatalf("decrypt = %v, %v", got, err)
	}
	if _, err := DecryptKeystoreV3(b, []byte("wrong")); err == nil || !strings.Contains(err.Error(), "mac") {
		t.Fatalf("wrong passphrase: %v", err)
	}
}

func TestKeystoreV3RefusesPBKDF2AndHugeScrypt(t *testing.T) {
	pb := strings.Replace(v3Vector, `"kdf": "scrypt"`, `"kdf": "pbkdf2"`, 1)
	if _, err := DecryptKeystoreV3([]byte(pb), []byte("x")); err == nil || !strings.Contains(err.Error(), "pbkdf2") {
		t.Fatalf("pbkdf2: %v", err)
	}
	huge := strings.Replace(v3Vector, `"n": 262144`, `"n": 1073741824`, 1)
	if _, err := DecryptKeystoreV3([]byte(huge), []byte("x")); err == nil || !strings.Contains(err.Error(), "out of bounds") {
		t.Fatalf("huge n: %v", err)
	}
}
