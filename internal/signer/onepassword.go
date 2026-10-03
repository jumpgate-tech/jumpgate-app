// internal/signer/onepassword.go
package signer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	// Fail closed: only op's "isn't an item" (assumed message text) means absent.
	// Any other failure could hide an existing item, and creating then would
	// leave a duplicate that makes `op read` ambiguous.
	_, getErr := runCmd(ctx, "", "op", "item", "get", item, "--vault", vault)
	var ce *cmdError
	switch {
	case getErr == nil:
		return nil, fmt.Errorf("%w: 1Password item %q in vault %q", ErrKeyExists, item, vault)
	case errors.As(getErr, &ce) && strings.Contains(ce.Stderr, "isn't an item"):
	default:
		return nil, fmt.Errorf("signer: 1Password existence check failed: %w", getErr)
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
	out, err := runCmd(ctx, "", "op", "item", "create", "--vault", vault, "--template", path, "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("signer: 1Password store failed (an item %q may have been created in vault %q; check and remove it by hand): %w", item, vault, err)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &created); err != nil || created.ID == "" {
		return nil, fmt.Errorf("signer: a 1Password item %q was created in vault %q but its id could not be read, so it was not removed; delete it by hand", item, vault)
	}
	k2, err := verifyStored(k, func() (*Key, error) { return onePasswordRead(ctx, ref) })
	if err != nil {
		// Roll back exactly the item just created, by id.
		if _, delErr := runCmd(ctx, "", "op", "item", "delete", created.ID, "--vault", vault); delErr != nil {
			return nil, fmt.Errorf("signer: 1Password item %s was created in vault %q but could not be verified, and removing it failed (%v); delete item %s by hand: %w", created.ID, vault, delErr, created.ID, err)
		}
		return nil, fmt.Errorf("signer: 1Password item %s could not be verified and was removed: %w", created.ID, err)
	}
	return k2, nil
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
