# The vault

The vault is one encrypted file, `~/.jumpgate/vault`, that can hold the controller key and the other secrets jumpgate keeps: the relay and admin tokens, the AI key and VPN configs. A passphrase unlocks it. A lost passphrase cannot be recovered: jumpgate keeps no copy, so add a recovery code.

While the vault is locked the server cannot sign, so commands to your boxes fail as locked until you unlock. The boxes keep running.

## Create and unlock

```
jumpgate vault init                      # create the vault; asks for a new passphrase
jumpgate vault add-recovery              # a recovery code that unlocks it if the passphrase is lost
jumpgate keys init --store vault         # keep the controller key in it
jumpgate unlock                          # the server takes the passphrase and unlocks
jumpgate lock                            # the server forgets the vault key
```

The passphrase is typed on the terminal, or sent as one line on standard input with `--stdin`; it is never taken from a command-line argument or the environment. On macOS and Windows, `vault init` also adds a slot that your login's key store unlocks, so the server unlocks at start with no prompt; `--keystore none` leaves that out.

In the TUI the Vault screen (`6`) does the same with keys: `u` unlocks, `ctrl+k` locks (it asks first when jobs are running), `I` creates the vault, `ctrl+n` adds a recovery code. `U` unlocks from any screen while the vault is locked, and `ctrl+k` locks it from any screen. `x` lists what you can do in the vault's current state.

## Plain secrets

An older setup may keep secrets as plain text in its config. `jumpgate vault migrate` creates the vault if there is none and moves them in; it lists what moved and what is still plain. The Vault screen has the same step on `m`.

## Back up and restore

```
jumpgate vault backup FILE               # copy the vault out; never overwrites a file
jumpgate vault restore FILE              # put it back; with --replace, over an existing vault
jumpgate vault rekdf                     # raise the passphrase's argon2id cost
```

A backup is useless without the passphrase. A TPM or key-store slot does not travel with it. Restore needs the server stopped (`jumpgate stop`), checks the file before it touches anything, and with `--replace` keeps the old vault as `vault.bak-TIME` beside it.

In the TUI, `ctrl+w` backs up and `ctrl+r` restores (it asks you to type `restore`). Making the passphrase harder to crack at the price of a slower unlock is in the `x` menu: the same as `jumpgate vault rekdf`, which raises the key-derivation (argon2id) cost.

## Tokens for the relay

`jumpgate vault put relay-token` (or `admin-token`) stores a token in the unlocked vault, read from the terminal or one line of `--stdin`. A controller that serves the relay itself finds it when `JUMPGATE_RELAY_TOKEN=vault:relay-token` is set. A running relay keeps its old token until jumpgate restarts.

## Unlock at boot, on a Linux server

A service started on a server comes back locked after a restart. To let it unlock on its own, seal the vault key with systemd:

```
jumpgate vault seal --tpm2               # bind it to this machine's TPM2
jumpgate vault seal --allow-host-key     # or to the host key, with no TPM2
```

The host key protects only against copying the vault file alone, so prefer the TPM2. On the Vault screen `ctrl+s` seals with the TPM2 and the host key is in the `x` menu. Without a seal you unlock the service by hand after each restart.

## Idle lock

The vault can lock itself after some idle minutes, from the Vault screen (`t`) or in Settings. Plans in progress keep it unlocked. 0 means never.

## What a lock clears

Locking overwrites the vault's data key and its decrypted entries and makes the server forget the controller signer. It does not remove every copy of a secret from memory: a value already turned into text, a running tunnel's config or an open connection's key can stay until the process ends or the memory is reused. Nothing is locked into RAM, so encrypt swap if that matters.
