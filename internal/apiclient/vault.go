package apiclient

import (
	"context"
	"encoding/base64"
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

// Lock drops the vault key from the server's memory.
func (c *Client) Lock(ctx context.Context) (api.VaultView, error) {
	var v api.VaultView
	err := c.Do(ctx, http.MethodPost, "/api/vault/lock", nil, &v)
	return v, err
}
