// internal/signer/keychain.go
package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
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
	if keychainExists(ctx, tool, ref) {
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
	return verifyStored(k, func() (*Key, error) { return keychainRead(ctx, ref) })
}

// keychainExists reports whether an item is already stored. A missing item
// makes both tools exit non-zero.
func keychainExists(ctx context.Context, tool, ref string) bool {
	var err error
	if tool == "security" {
		_, err = runCmd(ctx, "", "security", "find-generic-password", "-a", ref, "-s", keychainService)
	} else {
		_, err = runCmd(ctx, "", "secret-tool", "lookup", "service", keychainService, "account", ref)
	}
	return err == nil
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
		return nil, keychainErr("read", err)
	}
	return keyFromHex(out)
}
