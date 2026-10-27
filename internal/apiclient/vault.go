package apiclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/valve-tech/jumpgate/internal/api"
)

// Vault reads the vault's state.
func (c *Client) Vault(ctx context.Context) (api.VaultView, error) {
	var v api.VaultView
	err := c.Do(ctx, http.MethodGet, "/api/vault", nil, &v)
	return v, err
}

// Unlock sends passphrase to the server, which opens the vault with it. The
// passphrase and the encoded request body are cleared before it returns.
func (c *Client) Unlock(ctx context.Context, passphrase []byte) (api.VaultView, error) {
	defer clear(passphrase)
	const pre, post = `{"passphrase":"`, `"}`
	body := make([]byte, len(pre)+base64.StdEncoding.EncodedLen(len(passphrase))+len(post))
	defer clear(body)
	n := copy(body, pre)
	base64.StdEncoding.Encode(body[n:], passphrase)
	copy(body[len(body)-len(post):], post)
	var v api.VaultView
	err := c.Do(ctx, http.MethodPost, "/api/vault/unlock", rawJSON(body), &v)
	return v, err
}

// PutVaultSecret stores value in the server's unlocked vault as name
// (relay-token or admin-token). The value and the encoded body are cleared
// before it returns.
func (c *Client) PutVaultSecret(ctx context.Context, name string, value []byte) error {
	defer clear(value)
	const pre, post = `{"value":"`, `"}`
	body := make([]byte, len(pre)+base64.StdEncoding.EncodedLen(len(value))+len(post))
	defer clear(body)
	n := copy(body, pre)
	base64.StdEncoding.Encode(body[n:], value)
	copy(body[len(body)-len(post):], post)
	return c.Do(ctx, http.MethodPut, "/api/vault/secrets/"+url.PathEscape(name), rawJSON(body), nil)
}

// SystemInitKey asks the jumpgate service to create its controller key
// (POST /api/system/init-key). req.Passphrase and the encoded body are
// cleared before it returns.
func (c *Client) SystemInitKey(ctx context.Context, req api.SystemInitKey) error {
	defer clear(req.Passphrase)
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	defer clear(body)
	return c.Do(ctx, http.MethodPost, "/api/system/init-key", rawJSON(body), nil)
}

// SystemSeal has the jumpgate service add a systemd-creds slot to its vault
// (POST /api/system/seal) and returns the slot's 32-byte key, which the
// caller clears. passphrase and the encoded body are cleared.
func (c *Client) SystemSeal(ctx context.Context, passphrase []byte) ([]byte, error) {
	defer clear(passphrase)
	const pre, post = `{"passphrase":"`, `"}`
	body := make([]byte, len(pre)+base64.StdEncoding.EncodedLen(len(passphrase))+len(post))
	defer clear(body)
	n := copy(body, pre)
	base64.StdEncoding.Encode(body[n:], passphrase)
	copy(body[len(body)-len(post):], post)
	var v api.SystemSealed
	if err := c.Do(ctx, http.MethodPost, "/api/system/seal", rawJSON(body), &v); err != nil {
		clear(v.KEK)
		return nil, err
	}
	return v.KEK, nil
}

// Lock drops the vault key from the server's memory.
func (c *Client) Lock(ctx context.Context) (api.VaultView, error) {
	var v api.VaultView
	err := c.Do(ctx, http.MethodPost, "/api/vault/lock", nil, &v)
	return v, err
}
