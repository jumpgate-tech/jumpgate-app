# Security

## The local server

- It binds to `127.0.0.1` by default and needs a session token for every request, so nothing else on your machine can drive it without that token.
- The token never appears on a command line or in the terminal.
- Login links work once, within 60 seconds, and only over loopback.
- On a Linux server, the jumpgate service is reached through a socket that only members of the `jumpgate-admin` group can open.

## The controller key and the vault

Every command to a box is signed by your controller key, kept in the macOS Keychain, Windows Credential Manager, Secret Service, 1Password, the encrypted vault or an owner-only file. `jumpgate keys show` prints the key's address and flags it if it differs from the one on record. A key in the vault is only usable while the vault is unlocked; see [The vault](vault.md). Jumpgate never replaces an existing controller key silently, because paired boxes trust its address.

## Pairing and host keys

- You confirm each box's host key by its fingerprint. Confirmed keys are kept in `~/.jumpgate/confirmed_hosts`, and `jumpgate hosts forget-key NAME` forgets one.
- A box that is paired, confirmed or listed in your `~/.ssh/known_hosts` is checked strictly: a changed key is refused, never learned again.
- After pairing, turn off root SSH login on the box (`PermitRootLogin no`).

## Signed intents and receipts

The gate on each box checks every signed intent against its policy before anything runs, and answers with a receipt signed by its own key, which the controller verifies. Durable jobs add a signed log that any controller can verify.

A security failure always exits with code `4`: a host key nobody confirmed, a host key that changed, a confirmed fingerprint that is not the key the box presented, a reply the paired gate did not sign, or a controller key that is not the one on record.

## On the box

- The node services run as an unprivileged system account, `jumpgate-node`, under hardened systemd units (`NoNewPrivileges`, `ProtectSystem=strict` with the data directory carved out, private `/tmp` and devices). Box modules run under the same account.
- The firewall checklist only suggests commands. Jumpgate never runs them for you.
- The node's RPC binds to loopback by default. You can bind it to a trusted overlay address such as a Tailscale IP. The RPC is unauthenticated, so never bind it to a public address. Checkpointz refuses a public address outright.

## Tokens for the relay

The relay and admin tokens never come from a flag. Put each in its own file, readable only by you (`chmod 600`), and point to it with `JUMPGATE_RELAY_TOKEN_FILE` and `JUMPGATE_ADMIN_TOKEN_FILE`. (`JUMPGATE_RELAY_TOKEN` and `JUMPGATE_ADMIN_TOKEN` also work, but not together with the file form.) The server and the relay refuse a token file that is a symlink or not a regular file, grants any permission to group or others, or is owned by another user (unless they run as root). Windows makes equivalent checks.

## Unsigned downloads

The downloads are not code-signed yet. Check each one against `checksums.txt`, or with `gh attestation verify <file> --repo jumpgate-tech/jumpgate-app`.
