// internal/signer/onepassword.go
package signer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// parseOPRef splits op://vault/item/field.
func parseOPRef(ref string) (vault, item, field string, err error) {
	rest, ok := strings.CutPrefix(ref, "op://")
	parts := strings.Split(rest, "/")
	if !ok || len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("signer: 1Password ref %q must be op://<vault>/<item>/<field>", ref)
	}
	return parts[0], parts[1], parts[2], nil
}

// onePasswordCreate stores a new key as a 1Password item. `op item create`
// takes the item from a template file, which is created 0600, holds the key
// for the one call, and is removed before returning, so the key is never an
// argument. Requires a signed-in `op` CLI (v2).
func onePasswordCreate(ctx context.Context, ref string) (*Key, error) {
	vault, item, field, err := parseOPRef(ref)
	if err != nil {
		return nil, err
	}
	if lookPath("op") != nil {
		return nil, fmt.Errorf("signer: the 1Password CLI `op` is not installed")
	}
	if _, err := runCmd(ctx, "", "op", "item", "get", item, "--vault", vault); err == nil {
		return nil, fmt.Errorf("%w: 1Password item %q in vault %q", ErrKeyExists, item, vault)
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	tmpl, err := json.Marshal(map[string]any{
		"title":    item,
		"category": "API_CREDENTIAL",
		"fields": []map[string]string{{
			"id": field, "label": field, "type": "CONCEALED", "value": hex.EncodeToString(k.Bytes()),
		}},
	})
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp("", "jumpgate-op-*.json")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	defer os.Remove(path)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Write(tmpl); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if _, err := runCmd(ctx, "", "op", "item", "create", "--vault", vault, "--template", path); err != nil {
		return nil, fmt.Errorf("signer: 1Password store: %w", err)
	}
	return verifyStored(k, func() (*Key, error) { return onePasswordRead(ctx, ref) })
}

func onePasswordRead(ctx context.Context, ref string) (*Key, error) {
	if _, _, _, err := parseOPRef(ref); err != nil {
		return nil, err
	}
	out, err := runCmd(ctx, "", "op", "read", ref)
	if err != nil {
		return nil, fmt.Errorf("signer: 1Password read: %w", err)
	}
	return keyFromHex(out)
}
