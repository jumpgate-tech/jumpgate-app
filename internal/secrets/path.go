// Package secrets locates and reads jumpgate's secrets.
package secrets

import (
	"path/filepath"

	"github.com/valve-tech/jumpgate/internal/config"
)

// VaultPath is the encrypted vault: ~/.jumpgate/vault.
func VaultPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vault"), nil
}
