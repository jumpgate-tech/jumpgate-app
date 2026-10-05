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
	"time"
)

// Store names where a controller key lives.
type Store string

const (
	StoreFile        Store = "file"
	StoreKeychain    Store = "keychain"
	StoreWinCred     Store = "wincred"
	StoreOnePassword Store = "1password"
)

// Seams for tests: which OS we are on, whether a tool exists, and how a tool
// runs. Production never reassigns them.
var (
	hostOS   = runtime.GOOS
	lookPath = func(name string) error { _, err := exec.LookPath(name); return err }
	runCmd   = func(ctx context.Context, stdin, name string, args ...string) (string, error) {
		ctx, cancel := context.WithTimeoutCause(ctx, toolTimeout, errToolTimeout)
		defer cancel()
		c := exec.CommandContext(ctx, name, args...)
		isolateTool(c, name)
		// Once the tool is killed, stop waiting for anything it left holding
		// its output pipes.
		c.WaitDelay = 2 * time.Second
		c.Stdin = strings.NewReader(stdin)
		var out, errb bytes.Buffer
		c.Stdout, c.Stderr = &out, &errb
		if err := c.Run(); err != nil {
			if errors.Is(context.Cause(ctx), errToolTimeout) {
				err = fmt.Errorf("%s did not answer within %s; the keyring may be locked with nothing to prompt you. Unlock it, or use --store file", name, toolTimeout)
			}
			ce := &cmdError{Name: name, ExitCode: -1, Stdout: out.String(), Stderr: errb.String(), Err: err}
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				ce.ExitCode = ee.ExitCode()
			}
			return "", ce
		}
		return out.String(), nil
	}
)

// errToolTimeout marks a tool run cut off by toolTimeout, as opposed to the
// caller's own deadline.
var errToolTimeout = errors.New("key-store tool timed out")

// cmdError is a failed tool run. Stdout is kept only so callers can tell an
// empty result from a non-empty one; Error never prints it, because a tool's
// stdout can be key material. Callers read the exit code and streams to
// tell "not found" apart from any other failure.
type cmdError struct {
	Name           string
	ExitCode       int
	Stdout, Stderr string
	Err            error
}

func (e *cmdError) Error() string {
	return fmt.Sprintf("%s: %v: %s", e.Name, e.Err, strings.TrimSpace(e.Stderr))
}

func (e *cmdError) Unwrap() error { return e.Err }

// DefaultStore picks the OS key store when it can actually be used here:
// Credential Manager on Windows; the macOS keychain; on Linux the Secret
// Service only when a D-Bus session exists and the service answers (a
// headless box or an SSH login has neither). Otherwise a key file.
func DefaultStore() Store {
	switch hostOS {
	case "windows":
		return StoreWinCred
	case "darwin":
		if lookPath("security") == nil {
			return StoreKeychain
		}
	case "linux":
		if keychainTool() == "secret-tool" && secretServiceAnswers(context.Background()) {
			return StoreKeychain
		}
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
	case StoreWinCred:
		return winCredRead(ref)
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
	case StoreWinCred:
		return winCredCreate(ref)
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
