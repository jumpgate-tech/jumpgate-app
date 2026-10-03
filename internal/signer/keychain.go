// internal/signer/keychain.go
package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrNoKeychain means this machine has no keychain tool. It is an error rather
// than a quiet fallback to a file, so a key never lands somewhere the operator
// did not choose.
var ErrNoKeychain = errors.New("signer: no OS keychain here (need `security` on macOS or `secret-tool` on Linux); use --store file")

const keychainService = "jumpgate"

// keychainNameRE is deliberately narrow: `security -i` parses the line itself,
// so anything with quoting, escaping or line-break meaning could alter the command.
var keychainNameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func keychainTool() string {
	switch hostOS {
	case "darwin":
		if lookPath("security") == nil {
			return "security"
		}
	case "linux":
		if lookPath("secret-tool") == nil {
			return "secret-tool"
		}
	}
	return ""
}

// keychainCreate stores a new key. The key's hex is passed on stdin only:
// `security -i` reads commands from stdin and secret-tool reads the secret
// from it, so it never appears in any process's argv.
func keychainCreate(ctx context.Context, ref string) (*Key, error) {
	tool := keychainTool()
	if tool == "" {
		return nil, ErrNoKeychain
	}
	if !keychainNameRE.MatchString(ref) {
		return nil, fmt.Errorf("signer: keychain item name %q may only contain letters, digits, '.', '_' and '-'", ref)
	}
	exists, err := keychainExists(ctx, tool, ref)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: keychain item %q", ErrKeyExists, ref)
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	secret := hex.EncodeToString(k.Bytes())
	switch tool {
	case "security":
		cmd := fmt.Sprintf("add-generic-password -a %s -s %s -w %s\n", ref, keychainService, secret)
		_, err = runCmd(ctx, cmd, "security", "-i")
	case "secret-tool":
		_, err = runCmd(ctx, secret, "secret-tool", "store", "--label=jumpgate controller key", "service", keychainService, "account", ref)
	}
	if err != nil {
		return nil, keychainErr("store", err)
	}
	k2, err := verifyStored(k, func() (*Key, error) { return keychainRead(ctx, ref) })
	if err != nil {
		return nil, fmt.Errorf("signer: a key was written to keychain item %q (service %s) but could not be verified; inspect or remove that item by hand: %w", ref, keychainService, err)
	}
	return k2, nil
}

// keychainNotFound recognises each tool's "no such item" result and nothing
// else. Assumptions: `security` exits 44 (errSecItemNotFound); `secret-tool
// lookup` exits 1 with empty stdout and empty stderr (a missing Secret Service
// also exits 1, but prints an error, so it is not mistaken for absence).
func keychainNotFound(tool string, err error) bool {
	var ce *cmdError
	if !errors.As(err, &ce) {
		return false
	}
	if tool == "security" {
		return ce.ExitCode == 44
	}
	return ce.ExitCode == 1 && strings.TrimSpace(ce.Stdout) == "" && strings.TrimSpace(ce.Stderr) == ""
}

// keychainExists reports whether an item is stored. It fails closed: only a
// recognised not-found means absent, any other failure is an error, because
// treating it as absent could replace a real key.
func keychainExists(ctx context.Context, tool, ref string) (bool, error) {
	var err error
	if tool == "security" {
		_, err = runCmd(ctx, "", "security", "find-generic-password", "-a", ref, "-s", keychainService)
	} else {
		_, err = runCmd(ctx, "", "secret-tool", "lookup", "service", keychainService, "account", ref)
	}
	switch {
	case err == nil:
		return true, nil
	case keychainNotFound(tool, err):
		return false, nil
	}
	return false, keychainErr("existence check failed", err)
}

// keychainErr wraps a tool failure. On Linux the usual cause is a headless box
// with no Secret Service, so the error names the file store.
func keychainErr(op string, err error) error {
	if hostOS == "linux" {
		return fmt.Errorf("signer: keychain %s: %w (is a Secret Service running? on a headless box use --store file)", op, err)
	}
	return fmt.Errorf("signer: keychain %s: %w", op, err)
}

func keychainRead(ctx context.Context, ref string) (*Key, error) {
	var out string
	var err error
	switch keychainTool() {
	case "security":
		out, err = runCmd(ctx, "", "security", "find-generic-password", "-a", ref, "-s", keychainService, "-w")
	case "secret-tool":
		out, err = runCmd(ctx, "", "secret-tool", "lookup", "service", keychainService, "account", ref)
	default:
		return nil, ErrNoKeychain
	}
	if err != nil {
		if keychainNotFound(keychainTool(), err) {
			return nil, fmt.Errorf("signer: keychain read: no item %q: %w", ref, err)
		}
		return nil, keychainErr("read", err)
	}
	return keyFromHex(out)
}
