// internal/signer/store.go
package signer

import (
	"bytes"
	"context"
	"encoding/hex"
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

func keyFromHex(s string) (*Key, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(s), "0x"))
	if err != nil {
		return nil, fmt.Errorf("signer: stored key is not hex")
	}
	return KeyFromBytes(b)
}
