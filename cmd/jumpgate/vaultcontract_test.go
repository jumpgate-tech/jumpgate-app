package main

import (
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/server"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// TestVaultContract compiles only if the vault plan exposes what the
// server-install plan consumes (docs/superpowers/plans/2026-10-17-server-install.md).
func TestVaultContract(t *testing.T) {
	_ = []any{
		vault.Create, vault.Open, vault.DefaultParams,
		(*vault.Vault).Path, (*vault.Vault).Unlock, (*vault.Vault).UnlockKEK,
		(*vault.Vault).Lock, (*vault.Vault).Locked, (*vault.Vault).AddKEKSlot,
		vault.SlotSystemdCreds, (*vault.Vault).RemoveSlots, (*vault.Vault).Rekey,
		vault.ErrWrongPassphrase, vault.ErrCorrupt, vault.ErrLocked,
		signer.StoreVault, signer.CreateVault, signer.OpenVault,
		secrets.VaultPath,
		(*server.Server).VaultState, (*server.Server).Unlock,
		(*server.Server).UnlockKEK, (*server.Server).Lock,
		api.CodeLocked, api.CodeWrongPassphrase, api.CodeVaultCorrupt,
		(*apiclient.Client).Unlock, (*apiclient.Client).Lock,
		readPassphrase, credentialKEK,
	}
	_ = server.Config{Vault: (*vault.Vault)(nil)}
}
