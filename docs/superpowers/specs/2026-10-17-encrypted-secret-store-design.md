# Encrypted secret store: one vault for every secret jumpgate keeps

Date: 2026-10-17
Status: draft for review
Sibling: `2026-10-17-server-install-design.md` (D3, D4: the controller
server's key lives here)
Written against: `8fef986`

## Goal

Whenever no OS keychain, Secret Service or 1Password holds jumpgate's
secrets, they are encrypted at rest in one local vault. That covers servers,
Linux desktops without a Secret Service, and anyone who declines those
stores. Plain owner-only files remain only as an explicit, warned opt-out.

## Inventory: what jumpgate stores today

| # | Secret | Where (code) | Into the vault? |
|---|---|---|---|
| 1 | Controller signing key (hex) | `~/.jumpgate/keys/controller.key`, store `file` (`signer/keyfile.go`); else keychain, wincred or 1Password (`signer/store.go`) | **yes** (new store `vault`) |
| 2 | SSH transport key | `~/.jumpgate/ssh/jumpgate_ed25519` (`server/controller.go` `transportKeyPath`), read by `executor/dial.go` via `KeyPath` | **yes** |
| 3 | AI provider key | `config.json` `aiKey` | **yes** |
| 4 | RPC provider API keys | `config.json` `providerKeys` (and the legacy `valveKeys`, which migration collapses into it) | **yes** |
| 5 | WireGuard client configs, including `PrivateKey` | `config.json` `vpns[].config` | **yes** |
| 6 | Relay and admin tokens | `JUMPGATE_RELAY_TOKEN` / `JUMPGATE_ADMIN_TOKEN`, or a `*_FILE` checked by `readPrivateFile` (`buildserver.go`) | optional, for in-process `serve` only (V9) |
| 7 | Session token | random per start (`NewSessionToken`), in `run/server.json` 0600 | no: ephemeral; not written at all in system mode |
| 8 | Login codes and redirect files | one-time and short-lived (`server/login.go`) | no |
| 9 | Issued RPC API keys | billing service stores **hashes** (`services/billing/src/keys.rs`) | no: nothing secret at rest |
| 10 | VPN device private keys | generated at enrollment and delivered once; `VPNPeer` keeps only the public key | no: never stored |
| 11 | WireGuard server keys, agent key | on boxes: `/etc/wireguard/*.privatekey`, `/var/lib/jumpgate/agent.key` (root, `LoadCredential`) | no: box-side, unattended; see open questions |
| 12 | `install-id`, `known_hosts`, `confirmed_hosts` | `~/.jumpgate` | no: integrity-sensitive, not secret; they stay owner-only files |

The secrets that go into the vault are rows 1 to 5 (and optionally 6).
`TightenState` keeps auditing the files that remain (rows 7, 8 and 12).

## Decisions

**V1. One vault file per controller.**
- **Location:** `~/.jumpgate/vault` (for the service,
  `/var/lib/jumpgate-ctl/.jumpgate/vault`).
- **File checks:** the same integrity rules as the key file. It is opened
  with `O_NOFOLLOW`, then the handle is checked: a regular file, owner-only
  (`fsperm.CheckPrivateFile`), owned by this user.
- **Writes:** atomic (`fsatomic`) and private from creation.
- **Size:** bounded at 1 MiB.

**V2. Format: a key-slot header and an AEAD body.**

```
JGVAULT1\n
{"v":1,"kdf":{"alg":"argon2id","m":65536,"t":3,"p":4,"salt":"…"},
 "slots":[{"kind":"passphrase","nonce":"…","wrapped":"…"},
          {"kind":"systemd-creds","cred":"jumpgate-vault-key"}],
 "nonce":"…"}\n
<XChaCha20-Poly1305 ciphertext of the secrets JSON>
```

- **Data key:** a random 32-byte key (DEK) encrypts the body. Each slot wraps
  that DEK. The passphrase slot derives a KEK with argon2id and wraps the DEK
  with XChaCha20-Poly1305.
- **Dependencies:** `golang.org/x/crypto/argon2` and
  `golang.org/x/crypto/chacha20poly1305`, both in `x/crypto` v0.56.0, which
  is already a dependency. No new module.
- **Associated data:** the whole header line is the AEAD's associated data,
  so any change to the parameters, slots or nonce makes decryption fail.
- **Nonces:** every write draws a fresh 24-byte nonce.
- **Parameter floor:** parameters below m=64 MiB, t=3 are refused on open, so
  a header can't be downgraded.
- **Rotation is cheap:** changing the passphrase or the KDF parameters
  re-wraps the DEK. Adding a TPM slot adds a slot. No secret is re-entered.
- **Body:** a JSON map from names (`controller-key`, `transport-key`,
  `ai-key`, `provider/<NAME>`, `vpn/<id>`, `relay-token`, `admin-token`) to
  values. Each value also records when it was created and where it was
  migrated from.

**V3. Backend choice: the vault replaces "file" as the fallback, never plain
files silently.**
- **Default store:** `signer.DefaultStore()` keeps its order: wincred, the
  macOS keychain, then Secret Service when it answers. The final fallback
  becomes `StoreVault` instead of `StoreFile`.
- **Other secrets:** follow the controller key's store. With keychain, Secret
  Service or 1Password, rows 2 to 5 move there under one item each. With the
  vault, they move into the vault.
- **Plain files:** `--store file` requires `--insecure-plain`. Every server
  start then logs "secrets are stored unencrypted; run `jumpgate vault
  migrate`".
- **`config.json`:** stops carrying secret values. Each field becomes a
  reference (`"aiKey": "vault:ai-key"`) or is absent. The API, as today,
  reports only that a value is set.

**V4. Unlock once per process start; the DEK lives in memory only.**
- **Locking:** the server opens the vault at start and stays **locked** until
  an unlock.
- **Unlocking:**
  - the CLI with `jumpgate unlock` (reading `/dev/tty`, or stdin with
    `--stdin`);
  - the TUI prompt (hidden input);
  - the desktop window's unlock dialog;
  - the web UI form (session-authenticated);
  - or a `systemd-creds` slot at service start (`LoadCredentialEncrypted=`,
    TPM2 by default; see the sibling spec's D4).
- **Passphrase handling:** never on argv or in the environment. `secretenv`
  gains no new pass-through name, and the passphrase travels only in a
  request body over the unix socket or the session-authenticated loopback
  listener. It is never logged.
- **Clearing:** after the KDF, the server clears the passphrase buffer.
  `jumpgate lock` drops the DEK and every decrypted value.
- **Core dumps:** best effort. Go cannot promise to wipe memory, so core dumps
  are off (`PR_SET_DUMPABLE`, and `LimitCORE=0` in the unit).
- **Wrong passphrase:** `wrong_passphrase`. A tampered file is
  `vault_corrupt`, a different code. Neither says which byte failed.

**V5. Behaviour while locked.**

| Works | Waits | Refused with `locked` |
|---|---|---|
| server, TUI and web shell; config views (targets, pairings, gateways, settings without secret values); update check; `jumpgate home` | plans at their next step boundary (agents finish the current step, durable jobs) | every signed intent (status, logs, jobs, service actions), pairing, gateway renders that need provider keys, VPN up, AI calls, in-process relay start |

- **TUI:** shows a `LOCKED` badge in the header and an unlock action. Box rows
  show "locked: unlock to read" rather than "unreachable".
- **Desktop app:** prompts once at launch and can be dismissed. It then
  stays locked, with a banner.

**V6. Migration happens on the first unlock, as one transaction. Afterwards
the secret exists exactly once.** This follows the v0.5.0 rule in
`MigrateLegacyDir` ("a rename, not a copy, so secrets never exist twice").
Crossing into encryption needs a copy, so the copy is kept to one step:
1. Read each plain secret (rows 1 to 5) and build the vault in memory.
2. Write the vault, fsync it, then re-open it and decrypt-verify every value.
   This is the existing `verifyStored` rule: never trust a write that
   "succeeded".
3. In one `config.Update`, replace the secret fields with references and add
   `"secretStore": "vault"`.
4. For each plain key file: overwrite it with zeros, fsync, then unlink.
   The overwrite is best effort and is documented as such: journaling,
   copy-on-write filesystems and SSDs may keep old blocks. The migration
   notice says to treat a key that was ever plain on a disk that may have been
   imaged as exposed, and to rotate it (V8).
5. Record the migrated paths in the vault body. A crash between steps 2 and
   4 resumes on the next unlock. Re-reading a plain file whose value already
   matches the vault only finishes its removal.

A secret whose plain copy differs from the vault copy is never overwritten.
Both are kept, and a notice in `config.Notices` names the conflict.

**V7. Passphrase loss, backup and recovery.**
- **A lost passphrase means the secrets are gone.** No recovery key is held
  anywhere. `vault init` says so and asks for the passphrase twice.
- **Backup:** `jumpgate vault backup FILE` copies the encrypted vault as is.
  The copy is useless without the passphrase. TPM slots do not travel: they
  are bound to the machine.
- **Optional recovery slot:** `jumpgate vault add-recovery` adds a second
  passphrase slot, printed once as a generated 24-word code, for offline
  storage.
- **Restoring:** `jumpgate vault restore FILE` on a new controller server,
  then unlock.

**V8. Rotating a lost or exposed controller key.**
- **Now:** run `keys init` on a new vault, then re-pair every agent. Pairing
  already enrolls an extra signer without disturbing the first (`AddSigner`).
  Then remove the old address from each box's `policy.json` as root.
- **Later:** the parent spec's approval-tier "rotate or revoke a controller
  key" intent replaces the manual removal.

**V9. Relay and admin tokens.** The standalone `jumpgate relay` must run
unattended and never opens controller state, so it keeps `*_FILE` or systemd
`LoadCredential=`. For the in-process relay in `serve`, `vault:relay-token`
and `vault:admin-token` are accepted. That relay starts only once the vault is
unlocked.

**V10. Interop export.** `jumpgate keys export --keystore-v3 FILE` writes the
controller key as an Ethereum keystore v3 file (scrypt), so it can be held in
a wallet for break-glass. That keystore has its own passphrase, which is asked
on `/dev/tty`. This is the only path by which the raw key leaves the vault.

## Error handling

| Situation | Behaviour |
|---|---|
| Wrong passphrase | `wrong_passphrase`, no partial unlock; the TUI allows a retry |
| Tampered header or body, truncated file | `vault_corrupt`; fail closed; never fall back to plain files; point at `vault restore` |
| Symlink, wrong owner, group or other bits | refused, with the same fix text as `readPrivateFile`; never auto-tightened (unlike `TightenState`), since a widened vault means someone changed it |
| KDF parameters below the floor | refused as `vault_corrupt` |
| Plain and vault values conflict during migration | both kept; notice (V6) |
| A `systemd-creds` slot fails (TPM changed) | stays locked; the passphrase still works |

## Testing

- **Round trip:** write, read and rewrite. The nonce changes on every write.
  Every row in the inventory round-trips.
- **Wrong passphrase:** `wrong_passphrase`, distinct from corrupt.
- **Tampering:** a flipped bit in the header, the ciphertext or the tag; a
  truncated file; a swapped slot; a downgraded KDF. Each one is
  `vault_corrupt`, and nothing is decrypted.
- **Migration:** plain key file plus `aiKey`, `providerKeys` and `vpns` move
  in. The plain files are gone and `config.json` holds references. A crash
  injected after each of steps 1 to 4 resumes correctly. A conflicting plain
  copy is kept and noticed.
- **Locked state:** with the server locked, every route in V5's three columns
  answers as listed. A plan waits at its step boundary in the e2e harness
  (`scripts/e2e-agent.sh` job tests).
- **Parameter upgrade:** `vault rekdf` re-wraps the DEK. The body ciphertext
  is unchanged, and the old passphrase slot no longer opens it.
- **Integrity:** a symlinked vault, a group-readable vault, and a vault owned
  by another uid are each refused.
- **No secret on argv or in the environment:** a test drives `jumpgate
  unlock --stdin` and asserts that `/proc/<pid>/cmdline` and `environ` of
  every child hold no passphrase. This extends `secretenv`'s tests.
- **systemd-creds:** in the Docker harness with `--with-key=host` (no TPM in
  CI), an unlock at service start; a TPM run is manual.

## Rollout order

1. `internal/vault` (format, KDF, slots) with unit tests.
2. `signer.StoreVault` and the `locked` server state; `unlock`/`lock`.
3. Secret references in `config.json`; the executor takes the transport key
   as an `ssh.Signer` from the vault instead of `KeyPath`.
4. Migration on first unlock.
5. `DefaultStore` falls back to the vault; `--insecure-plain`.
6. `vault init|backup|restore|add-recovery|rekdf|seal`, and `keys export
   --keystore-v3`.

## Open questions

1. With an OS store, should rows 2 to 5 also go into that store, one item
   each? Or should they always go into a vault whose DEK the OS store holds?
   The second is one prompt, and the code path is the same everywhere.
2. Should argon2id parameters scale on small arm64 boxes (64 MiB at t=3), or
   stay fixed for portability between machines?
3. Should the desktop app hold the DEK across a window close while the tray
   is running, or lock on close?

### Resolved

4. **Should the on-box agent key and WireGuard server keys get
   `LoadCredentialEncrypted=` where a TPM exists?** Resolved 2026-10-17:
   later, not in this work. Agents hold nothing that can sign for the fleet.
5. **Should the controller lock itself after a period of no use?** Resolved
   2026-10-17: idle auto-lock is a setting, off by default, because durable
   jobs need the key unlocked to reach their next step.
