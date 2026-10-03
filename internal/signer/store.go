// internal/signer/store.go
package signer

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Store names where a controller key lives.
type Store string

const (
	StoreFile        Store = "file"
	StoreKeychain    Store = "keychain"
	StoreOnePassword Store = "1password"
)

// Seams for tests: which OS we are on, whether a tool exists, and how a tool
// runs. Production never reassigns them.
var (
	hostOS   = runtime.GOOS
	lookPath = func(name string) error { _, err := exec.LookPath(name); return err }
	runCmd   = func(ctx context.Context, stdin, name string, args ...string) (string, error) {
		c := exec.CommandContext(ctx, name, args...)
		c.Stdin = strings.NewReader(stdin)
		var out, errb bytes.Buffer
		c.Stdout, c.Stderr = &out, &errb
		if err := c.Run(); err != nil {
			return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(errb.String()))
		}
		return out.String(), nil
	}
)

// DefaultStore prefers the OS keychain and falls back to a key file only when
// no keychain tool exists, as on a headless box.
func DefaultStore() Store {
	if keychainTool() != "" {
		return StoreKeychain
	}
	return StoreFile
}

// Open loads an existing key.
func Open(ctx context.Context, store Store, ref string) (*Key, error) {
	switch store {
	case StoreFile:
		return LoadKeyFile(ref)
	case StoreKeychain:
		return keychainRead(ctx, ref)
	case StoreOnePassword:
		return onePasswordRead(ctx, ref)
	}
	return nil, fmt.Errorf("signer: unknown key store %q", store)
}

// Create makes a new key in store.
func Create(ctx context.Context, store Store, ref string) (*Key, error) {
	switch store {
	case StoreFile:
		return GenerateKeyFile(ref)
	case StoreKeychain:
		return keychainCreate(ctx, ref)
	case StoreOnePassword:
		return onePasswordCreate(ctx, ref)
	}
	return nil, fmt.Errorf("signer: unknown key store %q", store)
}

// ErrKeyExists means Create refused to replace a key that is already stored:
// paired agents trust that key's address.
var ErrKeyExists = errors.New("signer: a key already exists there; refusing to replace it")

// verifyStored reads a freshly written key back the way Open would and insists
// it is the key just generated, so a store command that exited 0 without
// storing anything cannot hand back a key that does not exist anywhere.
func verifyStored(k *Key, read func() (*Key, error)) (*Key, error) {
	got, err := read()
	if err != nil {
		return nil, fmt.Errorf("signer: key was not readable after storing it: %w", err)
	}
	if got.Address() != k.Address() {
		return nil, fmt.Errorf("signer: the stored key does not match the generated one; not using it")
	}
	return k, nil
}

func keyFromHex(s string) (*Key, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(s), "0x"))
	if err != nil {
		return nil, fmt.Errorf("signer: stored key is not hex")
	}
	return KeyFromBytes(b)
}
