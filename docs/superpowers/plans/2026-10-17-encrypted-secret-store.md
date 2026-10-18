# Encrypted Secret Store Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every secret the controller keeps (controller key, SSH transport key, AI key, provider keys, WireGuard client configs, and optionally the relay and admin tokens) is encrypted at rest in one local vault, `~/.jumpgate/vault`. The server starts `locked` and signs nothing until someone unlocks it. Plain files remain only behind an explicit `--insecure-plain`.

**Architecture:** `internal/vault` is a leaf package that holds the file format and nothing else: a `JGVAULT1` magic line, a one-line JSON header of key slots, and an XChaCha20-Poly1305 body. The body is sealed under a random data key (DEK), and each slot wraps that DEK under its own KEK. The KEK comes from argon2id over a passphrase, from a systemd credential, or from an OS key store item. `internal/secrets` is the glue between the vault and the rest of the controller: entry names, reference resolution, the transport key, and the one-transaction migration of plain secrets. `config.json` keeps `vault:NAME` references instead of values. `signer.StoreVault` becomes the fallback store. The server gains a `locked` state: box routes answer 503 `locked`, plans wait at their next step boundary, and unlocking runs migration, opens the key, and resumes jobs. The CLI, the TUI and the web UI each get an unlock surface. Vault administration (`vault init|migrate|backup|restore|add-recovery|rekdf|seal`, `keys export --keystore-v3`) runs in the CLI process under the vault's file lock, and a running server reloads the file whenever it changes.

**Tech Stack:** Go (`go 1.26.0` directive). `golang.org/x/crypto` v0.56.0 (`argon2`, `chacha20poly1305`, `hkdf`, `scrypt`, `ssh`) and `github.com/charmbracelet/x/term` v0.2.2 (`ReadPassword`), both already in `go.sum`, so no new module. `golang.org/x/sys/unix` for `prctl`. React + vitest in `cmd/jumpgate/web`. Bubble Tea v2 TUI. The Docker systemd harness in `scripts/e2e/`.

**Spec:** `docs/superpowers/specs/2026-10-17-encrypted-secret-store-design.md` (V1 to V10, error table, testing, rollout). Sibling: `docs/superpowers/specs/2026-10-17-server-install-design.md` D3 and D4. That spec's plan runs **after** this one and consumes the API in Task 3's Interfaces block and in Task 7's server methods.

## Scope boundary with the server-install plan

This plan covers the vault and the backend choice. The server-install plan owns `jumpgate.service` (including `LimitCORE=0` and the `LoadCredentialEncrypted=` line), the `jumpgate-ctl`/`jumpgate-admin` users, `serve --system`, `system.json`, peer-credential attach, `jumpgate setup server`, `install.sh` and the packages. This plan provides what those pieces call: `vault.Open`/`Unlock`/`Lock`/`Get`/`Put`, `secrets.VaultPath`, `(*server.Server).Unlock`/`Lock`, the `locked` state, `jumpgate unlock|lock`, `jumpgate vault seal`, and the reading of `$CREDENTIALS_DIRECTORY/jumpgate-vault-key` at start.

**Resolved with the server-install plan (name alignment).** That plan's Task 0 pins exactly the API in Task 3, Task 5, Task 7 and Task 11 above, so this plan needs no added task. Its system-mode seal draws the 32-byte KEK server-side and calls `AddKEKSlot(Slot{Kind: SlotSystemdCreds, Cred: "jumpgate-vault-key"}, kek)` (no secret-returning slot call is wanted here), it reads the lock state through `(*Server).VaultState().State` (there is no `LockState()`), and it reuses Task 11's `vault-auto-unlock` and Task 19's idle auto-lock instead of building its own. `vault seal` takes one mode flag, `--tpm2` or `--allow-host-key`.

## Open questions

| # | Question (spec) | Status / default implemented | Owner |
|---|---|---|---|
| Q1 | With an OS store, do rows 2 to 5 go into that store item by item, or into a vault whose DEK the OS store holds? | **Unresolved; default implemented: the vault, with a `keystore` slot.** The OS store holds one item (a secp256k1 key made by `signer.Create`), and the slot's KEK is HKDF-SHA256 of that item's scalar. That gives one code path everywhere, no new per-item store code, and an unlock at server start with no prompt. The controller key itself stays in the OS store as today. | Task 16 |
| Q2 | Scale argon2id parameters on small arm64 boxes? | **Unresolved; default: fixed** `m=65536 t=3 p=4`, the floor too, so a vault opens on any controller. `vault rekdf` raises them by hand. | Task 1 |
| Q3 | Does the desktop app hold the DEK across a window close while the tray runs? | **Unresolved; default: hold it.** The server process holds the DEK, and the window is only a client; `jumpgate lock` and the web UI's Lock button drop it. | Task 14 |
| Q4 | `LoadCredentialEncrypted=` for the agent key and WireGuard server keys | **Resolved:** later. Agents hold nothing that can sign for the fleet. Not in this plan. | — |
| Q5 | Idle auto-lock (server-install spec Q2) | **Resolved:** a setting, off by default, because durable jobs need the key unlocked. | Task 19 |

## Decisions made in this plan

| # | Decision | Why | Cost if wrong |
|---|---|---|---|
| S1 | The KDF block lives **in each passphrase slot** (`slots[i].kdf`). The spec shows it at the top level. | Otherwise the recovery slot and the primary slot share parameters, and `rekdf` with one passphrase would break the other slot. | One field moves; the format is version 1 and has never shipped. |
| S2 | **Every** slot kind carries `nonce` and `wrapped`, which hold the DEK wrapped under that slot's KEK. A `systemd-creds` credential holds a random 32-byte KEK, never the DEK. | Unlocking is the same for every slot kind, and a wrong credential is detected at unwrap time instead of looking like a corrupt body. | None. |
| S3 | The AEAD associated data is the whole header line (V2). So `rekdf`, `add-recovery` and `seal` **re-seal the body with the same DEK** under a fresh nonce. The spec's test wording "body ciphertext is unchanged" is read as "DEK and entries unchanged". | It keeps V2's rule that any header change makes decryption fail. | None for security; a write is one more encryption. |
| S4 | A changed byte **inside the slot being used** reads as `wrong_passphrase`. Any other header change, any body change, a truncation, or a swapped slot reads as `vault_corrupt`. | Before the DEK is known, nothing can tell a tampered wrap from a wrong passphrase. | A tamper inside the slot is reported as a wrong passphrase; the file still fails closed. |
| S5 | HTTP answers: 503 `locked` (as the sibling's error table says), 403 `wrong_passphrase`, 500 `vault_corrupt`, 500 `vault_unsafe` (integrity refusal), 409 `no_vault`. Exit classes: `locked` and `no_vault` 2, `wrong_passphrase` 1, `vault_corrupt` and `vault_unsafe` 4. | 401 would make `apiclient` re-read the session token; 4 marks a security failure, as `host_key` does. | Registry entries in one file. |
| S6 | `api.UnlockRequest.Passphrase` is `[]byte`, so it travels base64 in JSON. | Both Go ends keep the passphrase in a slice they can clear, never in a `string`. | The web UI base64-encodes (`btoa` of UTF-8). |
| S7 | Vault administration runs in the CLI process on the file, under an exclusive `filelock` on `<vault>.lock`. A running server reloads the file whenever its size, mtime or inode changes, and decrypts it with the DEK it holds. A reload that the held DEK cannot decrypt locks the server (`vault_corrupt`). | `keys init --store vault`, `rekdf` and `seal` work with or without a running server, and never lose each other's entries. | A `vault restore` under a running server locks it; `restore` refuses while a server answers. |
| S8 | The server opens the vault **lazily**: a vault that `vault init` creates after start is found on the next `/api/vault` request or unlock. | `vault migrate` works against an already-running server with no restart. | One `Lstat` per vault request while no vault exists. |
| S9 | Idle means: no authenticated POST, PUT or DELETE for N minutes **and** no job driver running. Reads and streams are not activity. | An open TUI streams forever; an idle person does not write. | One timestamp per mutating request. |
| S10 | When run as root, a vault write keeps the existing file's uid and gid (`fchown` of the temp file before the rename). | `sudo jumpgate vault seal` must not leave the service user unable to read its own vault. | Unix only; Windows keeps the DACL from `fsperm`. |

## Preconditions

- Branch `feat/encrypted-secret-store` from `main` at or after `bed7a63`. Baseline: `go test ./...` and `cd cmd/jumpgate/web && npm test` are green. Record both.
- Docker is available for Task 20. Without it, Task 20's run step is done in CI.
- Never push without the user's permission.

## Global Constraints

- Module `github.com/valve-tech/jumpgate`. The product, binary and commands are `jumpgate`. No new Go module, and the `go` directive stays `go 1.26.0`. `github.com/charmbracelet/x/term` moves from indirect to direct (`go mod tidy`), at v0.2.2.
- Vault path: `filepath.Join(config.Dir(), "vault")`, so `~/.jumpgate/vault` (and `/var/lib/jumpgate-ctl/.jumpgate/vault` for the service, through `HOME`, with no code). Lock file: `<vault>.lock`.
- File: `JGVAULT1\n`, then one header line of JSON ending in `\n`, then the ciphertext. Header `"v":1`. The whole header line, without its `\n`, is the body's AEAD associated data.
- Crypto: DEK 32 bytes from `crypto/rand`. AEAD `chacha20poly1305.NewX` (24-byte nonce, fresh on every write and every wrap). KDF `argon2.IDKey` with salt 16 bytes, key 32 bytes, and defaults `m=65536` (KiB), `t=3`, `p=4`. Floor `m>=65536`, `t>=3`, `p>=1`. Ceiling `m<=1048576`, `t<=16`, `p<=64`. Outside those bounds the vault is `vault_corrupt`, checked before any KDF runs.
- Size: a vault file larger than 1 MiB (`1<<20`) is refused on read, and a write that would exceed it fails with `ErrTooLarge`.
- Entry names match `^[A-Za-z0-9][A-Za-z0-9/_.-]{0,127}$`. The fixed names are `controller-key`, `transport-key`, `ai-key`, `provider/<PLACEHOLDER>`, `vpn/<id>`, `relay-token` and `admin-token`.
- `config.json` references: `vault:<name>`. With `"secretStore": "vault"`, `config.Save` refuses any plain value in `aiKey`, `providerKeys` or `vpns[].config`.
- Integrity (V1): the file is opened `O_NOFOLLOW` (Windows: `FILE_FLAG_OPEN_REPARSE_POINT`), then the handle is checked: a regular file, `fsperm.CheckPrivateFile`, owned by this user. The vault is never auto-tightened; `config.TightenState` must not touch it. Writes go through `fsperm.WriteFilePrivate`, then `fsatomic.SyncDir`.
- A passphrase is never on argv or in the environment. `secretenv` gains no name. It travels only in a request body over the unix socket or the session-authenticated loopback listener, and is never logged. Every function that takes a passphrase `[]byte` clears it before returning.
- Error codes (`internal/api/errors.go`, registry): `locked`, `wrong_passphrase`, `vault_corrupt`, `vault_unsafe`, `no_vault`, with the statuses and exit classes in S5.
- Startup warning, verbatim, whenever a plain secret remains or `secretStore` is `"file"`: ``secrets are stored unencrypted; run `jumpgate vault migrate` ``.
- Tests isolate `HOME` with `testutil.Home(t)`, and gate permission and owner tests on `testutil.RequireUnix(t)`. Tests that need a vault use `vaulttest` (Task 3) and never run their own KDF more than once.
- Every commit step adds exact paths, never `-A` or `.`. The one exception is the rebuilt `cmd/jumpgate/web/dist`, whose file names are content hashes.

## Review Focus

1. **Two writers on one vault.** `jumpgate keys init --store vault` runs while the server holds the vault unlocked. The server's next write must keep the CLI's `controller-key`, never write back its stale body. Test: Task 2 `TestPutKeepsAnotherWritersEntry`.
2. **Root rewrites a service's vault.** `sudo jumpgate vault seal` on a vault owned by `jumpgate-ctl` must leave it owned by `jumpgate-ctl`, or the service can no longer open it at boot. Test: Task 3 `TestRootWriteKeepsTheOwner`.
3. **Passphrases from pipes.** `printf 'pw\n' | jumpgate unlock --stdin`, Windows `echo pw|` (`\r\n`), and an empty line. Exactly one trailing `\n` or `\r\n` is stripped, inner whitespace is kept, and an empty passphrase is refused before any request is sent. Test: Task 5 `TestReadPassphraseFromStdin`.
4. **Settings saved while locked.** The web UI saves a new provider key while the vault is locked. The answer is 503 `locked`, and nothing plain lands in `config.json`. Tests: Task 8 `TestSettingsWhileLockedWritesNothing` and Task 4 `TestSaveRefusesPlainSecretsUnderTheVault`.
5. **Core dumps off breaks `/proc` reads.** `PR_SET_DUMPABLE=0` makes `/proc/self/*` root-owned. The login-link peer check reads `/proc/net/tcp` and must keep working. Test: Task 11 `TestLoginPeerLookupWorksWhenNotDumpable` (Linux).

---

### Task 1: Vault format: header, slots, KDF bounds, AEAD

**Files:**
- Create: `internal/vault/format.go`
- Test: `internal/vault/format_test.go`

**Interfaces:**
- Consumes: nothing (leaf package).
- Produces (unexported, package `vault`):
  - `type kdfParams struct{ Alg string "alg"; M uint32 "m"; T uint32 "t"; P uint8 "p"; Salt []byte "salt" }`
  - `type slotRec struct{ Kind, Label string; KDF *kdfParams; Cred, Store, Ref string; Nonce, Wrapped []byte }` (JSON names `kind`, `label`, `kdf`, `cred`, `store`, `ref`, `nonce`, `wrapped`; `omitempty` on the optional ones)
  - `type header struct{ V int "v"; Slots []slotRec "slots"; Nonce []byte "nonce" }`
  - `type entryRec struct{ Value []byte "value"; Created time.Time "created"; From string "from,omitempty" }`
  - `type body struct{ Entries map[string]entryRec "entries" }`
  - `func checkKDF(k kdfParams) error`: `ErrCorrupt` outside the floor and ceiling.
  - `func deriveKEK(pass []byte, k kdfParams) []byte`
  - `func wrapDEK(kek, dek []byte, s *slotRec) error`: fills `s.Nonce` and `s.Wrapped`, with AD `"jgvault-slot\x00"+s.Kind+"\x00"+s.Label`.
  - `func unwrapDEK(kek []byte, s slotRec) ([]byte, error)`: `ErrWrongPassphrase` on AEAD failure.
  - `func seal(h header, dek []byte, b body) ([]byte, error)`: the whole file, with a fresh `h.Nonce`.
  - `func parse(raw []byte) (h header, line, ct []byte, err error)`: `ErrCorrupt` on bad magic, a missing newline, bad JSON, `v != 1`, or a KDF out of bounds.
  - `func openBody(line, ct, dek []byte, nonce []byte) (body, error)`: `ErrCorrupt`.
  - Exported errors: `ErrCorrupt`, `ErrWrongPassphrase`, `ErrBadName`; `func validName(n string) error`.

- [ ] **Step 1: Write the failing tests**

```go
func TestSealParseRoundTrip(t *testing.T)            // seal → parse → unwrap with the passphrase → openBody gives the same entries
func TestEveryWriteDrawsAFreshNonce(t *testing.T)    // two seals of one body: header nonces differ, ciphertexts differ
func TestWrongPassphraseIsNotCorrupt(t *testing.T)   // errors.Is(err, ErrWrongPassphrase) && !errors.Is(err, ErrCorrupt)
func TestTamperingIsCorrupt(t *testing.T) {
	// Subtests, each must be ErrCorrupt and return no body:
	// "header nonce bit", "ciphertext bit", "tag bit" (last byte), "truncated" (drop 1 byte),
	// "truncated header" (cut inside the JSON), "magic", "swapped slot" (slot from another vault
	// made with the SAME passphrase), "version 2".
}
func TestKDFBoundsAreEnforced(t *testing.T) {
	// m=65535, t=2, p=0, m=1048577, t=17, p=65: each ErrCorrupt from parse, and parse returns
	// within 50ms for m=1048577 (no KDF ran).
}
func TestEntryNamesAreValidated(t *testing.T) // ok: "provider/VALVE_API_KEY", "vpn/proton-1"; bad: "", "/x", "a b", 129 chars, "x\x00"
```

Use `kdfParams{Alg: "argon2id", M: 65536, T: 3, P: 4}` with a fixed salt; derive the KEK once in a package-level `sync.Once` helper so the file runs one KDF.

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/vault/ -run 'TestSealParseRoundTrip|TestEveryWriteDrawsAFreshNonce|TestWrongPassphraseIsNotCorrupt|TestTamperingIsCorrupt|TestKDFBoundsAreEnforced|TestEntryNamesAreValidated' -v`
Expected: build failure, `undefined: seal` (and the other names).

- [ ] **Step 3: Implement `internal/vault/format.go`**

The header is `json.Marshal(h)` on one line; the file is `"JGVAULT1\n" + line + "\n" + ct`. `parse` runs `checkKDF` on every passphrase slot before returning, so no caller can reach `deriveKEK` with an out-of-bounds header. Use `chacha20poly1305.NewX` for both the wrap and the body.

- [ ] **Step 4: Run them to see them pass**

Run: the command from Step 2.
Expected: `ok  	github.com/valve-tech/jumpgate/internal/vault`

- [ ] **Step 5: Commit**

```bash
git add internal/vault/format.go internal/vault/format_test.go
git commit -m "feat(vault): file format with key slots, argon2id bounds and XChaCha20-Poly1305"
```

### Task 2: Vault file: create, open, unlock, lock, get, put

**Files:**
- Create: `internal/vault/vault.go`, `internal/vault/open_unix.go`, `internal/vault/open_windows.go`, `internal/vault/open_other.go`
- Test: `internal/vault/vault_test.go`, `internal/vault/vault_unix_test.go`

**Interfaces:**
- Consumes: Task 1's codec; `fsperm.CheckPrivateFile`, `fsperm.CreatePrivate`, `fsperm.WriteFilePrivate`, `fsatomic.SyncDir`, `filelock.Lock(path string, exclusive bool) (*filelock.Handle, error)`.
- Produces: the core of the public API listed in full in Task 3. This task adds `Create`, `Open`, `Path`, `Locked`, `Unlock`, `Lock`, `Get`, `Entry`, `Names`, `Put`, `PutAll`, `Delete`, `Item`, `Entry`, `Params`, `DefaultParams`, `MaxSize`, and the errors `ErrNoVault`, `ErrExists`, `ErrLocked`, `ErrUnsafeFile`, `ErrNotFound` and `ErrTooLarge`. It also adds `func openNoFollow(path string) (*os.File, error)`, unexported and per OS, modelled on `signer/keyfile_open_*.go`.

- [ ] **Step 1: Write the failing tests**

```go
func TestCreateThenOpenUnlockGet(t *testing.T)      // Create(p, pass, DefaultParams); Put("ai-key", []byte("k"), ""); Open(p).Locked()==true; Unlock(pass2); Get == "k"
func TestCreateRefusesAnExistingFile(t *testing.T)  // second Create: errors.Is(err, ErrExists); the file is unchanged
func TestPassphraseSlicesAreCleared(t *testing.T)   // after Create and Unlock, every byte of the slices passed is 0
func TestLockedVaultRefusesReadsAndWrites(t *testing.T) // Get/Put/Names on a locked vault: ErrLocked
func TestLockDropsValues(t *testing.T)              // a []byte returned by Get stays valid (copy), but v.Lock(); v.Get → ErrLocked
func TestOpenMissingIsErrNoVault(t *testing.T)      // errors.Is(err, ErrNoVault) && errors.Is(err, fs.ErrNotExist)
func TestOpenRefusesOversizeFile(t *testing.T)      // a 1<<20+1 byte file: ErrCorrupt; PutAll beyond MaxSize: ErrTooLarge
func TestPutKeepsAnotherWritersEntry(t *testing.T) {
	// Review Focus 1: a := unlocked; b := Open+Unlock same path; b.Put("controller-key", x, "");
	// a.Put("ai-key", y, ""); fresh Open+Unlock sees BOTH entries.
}
func TestReloadWithAnotherDEKLocks(t *testing.T)    // replace the file with a different vault: a.Get → ErrCorrupt and a.Locked()
func TestWriteVerifiesWhatItWrote(t *testing.T)     // with the seam writeHook corrupting the temp bytes, Put fails and Get still returns the old value
```

`vault_unix_test.go` (`testutil.RequireUnix`):

```go
func TestOpenRefusesASymlink(t *testing.T)      // ErrUnsafeFile, message contains "link"
func TestOpenRefusesGroupReadable(t *testing.T) // chmod 0640 → ErrUnsafeFile; mode is still 0640 afterwards (never tightened)
func TestVaultIsPrivateFromCreation(t *testing.T) // testutil.AssertPrivate(t, path) after Create and after Put
func TestOpenRefusesAnotherOwner(t *testing.T)  // skipped unless os.Geteuid()==0: chown 65534 → ErrUnsafeFile
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/vault/ -run 'TestCreate|TestPassphrase|TestLocked|TestLock|TestOpen|TestPut|TestReload|TestWrite|TestVault' -v`
Expected: build failure, `undefined: Create`.

- [ ] **Step 3: Implement `internal/vault/vault.go` and the `open_*.go` files**

- **Exact signatures:**
  - `func Create(path string, passphrase []byte, p Params) (*Vault, error)`
  - `func Open(path string) (*Vault, error)`
  - `func (v *Vault) Unlock(passphrase []byte) error`
  - `func (v *Vault) Lock()`
  - `func (v *Vault) Get(name string) ([]byte, error)`
  - `func (v *Vault) Put(name string, value []byte, from string) error`
  - `func (v *Vault) PutAll(items []Item) error`
  - `func (v *Vault) Delete(name string) error`
- **Concurrency:** `Vault` is safe for concurrent use under one mutex.
- **Writes:** every write takes the exclusive lock on `path+".lock"`, reloads the file if its `os.Stat` size or mtime, or `os.SameFile`, differs from the last read, applies the change, seals, writes with `fsperm.WriteFilePrivate`, runs `fsatomic.SyncDir`, then reads the file back and compares the decrypted body. Any mismatch is an error, and the in-memory body is left as it was before the write.
- **Unlock on an unlocked vault:** it checks the passphrase and keeps the DEK; a wrong one is `ErrWrongPassphrase` and does not lock.
- **`Lock`:** it `clear`s the DEK and every entry's value.
- **Test seam:** `var writeHook func([]byte) []byte`, nil in production.

- [ ] **Step 4: Run them to see them pass**

Run: the command from Step 2, then `GOOS=windows go vet ./internal/vault/`.
Expected: `ok  	github.com/valve-tech/jumpgate/internal/vault`; vet prints nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/vault/vault.go internal/vault/open_unix.go internal/vault/open_windows.go internal/vault/open_other.go internal/vault/vault_test.go internal/vault/vault_unix_test.go
git commit -m "feat(vault): open, unlock, lock, get and put with integrity checks and reload on change"
```

### Task 3: Slots, rotation and owner preservation: the final vault API

**Files:**
- Create: `internal/vault/slots.go`, `internal/vault/wordlist.go`, `internal/vault/wordlist_en.txt` (the BIP-39 English list, 2048 lines), `internal/vault/owner_unix.go`, `internal/vault/owner_other.go`, `internal/vault/nodump_linux.go`, `internal/vault/nodump_other.go`, `internal/vault/vaulttest/vaulttest.go`
- Test: `internal/vault/slots_test.go`, `internal/vault/owner_unix_test.go`

**Interfaces:**
- Consumes: Task 2.
- Produces: **the complete public API, which the server-install plan consumes.**

```go
// import "github.com/valve-tech/jumpgate/internal/vault"
package vault

type Vault struct{ /* unexported; safe for concurrent use */ }

type Params struct {
	Memory  uint32 // KiB
	Time    uint32
	Threads uint8
}

var DefaultParams = Params{Memory: 65536, Time: 3, Threads: 4}

const MaxSize = 1 << 20

type SlotKind string

const (
	SlotPassphrase   SlotKind = "passphrase"    // Label "primary" or "recovery"
	SlotSystemdCreds SlotKind = "systemd-creds" // Cred "jumpgate-vault-key"
	SlotKeystore     SlotKind = "keystore"      // Store "keychain"|"wincred"|"1password", Ref
)

type Slot struct{ Kind SlotKind; Label, Cred, Store, Ref string } // what a header says; no key material
type Item struct{ Name string; Value []byte; From string }
type Entry struct{ Value []byte; Created time.Time; From string }

var (
	ErrNoVault, ErrExists, ErrLocked, ErrWrongPassphrase, ErrSlotRefused,
	ErrCorrupt, ErrUnsafeFile, ErrNotFound, ErrTooLarge, ErrBadName error
)

func Create(path string, passphrase []byte, p Params) (*Vault, error) // new file, primary slot, returned unlocked; clears passphrase
func Open(path string) (*Vault, error)                                // integrity + header checks; returned locked
func (v *Vault) Path() string
func (v *Vault) Locked() bool
func (v *Vault) Slots() []Slot
func (v *Vault) Unlock(passphrase []byte) error            // tries every passphrase slot; clears passphrase
func (v *Vault) UnlockKEK(kind SlotKind, kek []byte) error // tries every slot of kind; ErrSlotRefused; clears kek
func (v *Vault) Lock()
func (v *Vault) Get(name string) ([]byte, error)           // a copy; ErrLocked, ErrNotFound
func (v *Vault) Entry(name string) (Entry, error)
func (v *Vault) Names() ([]string, error)                  // sorted
func (v *Vault) Put(name string, value []byte, from string) error
func (v *Vault) PutAll(items []Item) error                 // one write, verified
func (v *Vault) Delete(name string) error                  // ErrNotFound when absent
func (v *Vault) AddRecovery() (code string, err error)     // 24 words; adds or replaces the "recovery" slot
func (v *Vault) Rekdf(passphrase, newPassphrase []byte, p Params) error // nil newPassphrase keeps it; replaces the primary slot
func (v *Vault) AddKEKSlot(s Slot, kek []byte) error       // replaces a slot with the same Kind and Cred/Ref; clears kek
func (v *Vault) RemoveSlots(kind SlotKind) error           // refuses to remove the last passphrase slot
func DisableCoreDumps() error                              // Linux: prctl(PR_SET_DUMPABLE, 0); elsewhere nil
```

```go
// import "github.com/valve-tech/jumpgate/internal/vault/vaulttest"
func Path(t testing.TB) (path string, passphrase []byte) // a locked vault copied from a once-per-process fixture into t.TempDir()
func Unlocked(t testing.TB) *vault.Vault
const Passphrase = "correct horse battery staple"
```

- [ ] **Step 1: Write the failing tests**

```go
func TestRecoveryCodeUnlocks(t *testing.T)            // code has 24 space-separated words, each in the embedded list; Lock; Unlock([]byte(code)) ok
func TestWordListIsBIP39English(t *testing.T)         // 2048 unique lines; sha256 of the file == the published BIP-39 english.txt hash
func TestRekdfKeepsTheDEKAndDropsTheOldSlot(t *testing.T) {
	// copy the old file aside; Rekdf(pass, nil, Params{Memory: 131072, Time: 4, Threads: 4});
	// v.dek unchanged (internal test); every entry unchanged; the old file's primary slot,
	// spliced into the new header, does not unwrap the new body: ErrCorrupt.
}
func TestRekdfCanChangeThePassphrase(t *testing.T)    // old passphrase: ErrWrongPassphrase; new one: ok
func TestKEKSlotUnlocks(t *testing.T)                 // AddKEKSlot(Slot{Kind: SlotSystemdCreds, Cred: "jumpgate-vault-key"}, kek); Lock; UnlockKEK ok; another kek: ErrSlotRefused
func TestLastPassphraseSlotCannotBeRemoved(t *testing.T)
func TestSlotsAreReadableWhileLocked(t *testing.T)    // Open(...).Slots() lists kinds and labels, no Nonce/Wrapped fields exist on Slot
```

`owner_unix_test.go`:

```go
func TestRootWriteKeepsTheOwner(t *testing.T) {
	// Review Focus 2: keepOwner(fi os.FileInfo, euid int) (uid, gid int, chown bool)
	// euid 0, file uid 990 gid 990 → (990, 990, true); euid 501, file 501 → chown false.
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/vault/... -run 'TestRecovery|TestWordList|TestRekdf|TestKEKSlot|TestLastPassphrase|TestSlotsAre|TestRootWrite' -v`
Expected: build failure, `undefined: (*Vault).AddRecovery` (and others).

- [ ] **Step 3: Implement**

- **Recovery code:** 24 words, each drawn with `rand.Int(rand.Reader, big.NewInt(2048))`. The code goes through argon2id like any passphrase.
- **Word list:** embedded with `//go:embed wordlist_en.txt`.
- **Owner preservation:** the write path calls `keepOwner` and `f.Chown` on the temp file before the rename (unix), as S10 requires.
- **`DisableCoreDumps`:** uses `unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)`.
- **`vaulttest`:** builds its fixture once with `Create` and `DefaultParams`, then copies the bytes into each test's directory with mode 0600.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/vault/... -v`
Expected: `ok  	github.com/valve-tech/jumpgate/internal/vault` and `ok  	github.com/valve-tech/jumpgate/internal/vault/vaulttest` (or `[no test files]`).

- [ ] **Step 5: Commit**

```bash
git add internal/vault/slots.go internal/vault/wordlist.go internal/vault/wordlist_en.txt internal/vault/owner_unix.go internal/vault/owner_other.go internal/vault/nodump_linux.go internal/vault/nodump_other.go internal/vault/vaulttest/vaulttest.go internal/vault/slots_test.go internal/vault/owner_unix_test.go
git commit -m "feat(vault): recovery, rekdf and KEK slots; keep the owner on root writes; test fixtures"
```

### Task 4: Secret references in config.json

**Files:**
- Modify: `internal/config/config.go` (the `Config` struct near line 238, the `Controller.KeyStore` comment at line 79, `Save` at line 955, `notice` at line 581)
- Test: `internal/config/secretref_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces:
  - `Config.SecretStore string` with JSON name `secretStore,omitempty`; `const SecretStoreVault = "vault"`, `const SecretStoreFile = "file"`.
  - `Config.VaultIdleLockMinutes int` with JSON name `vaultIdleLockMinutes,omitempty`. Task 19 reads it.
  - `const SecretRefPrefix = "vault:"`
  - `func SecretRef(name string) string`
  - `func IsSecretRef(s string) bool`
  - `func SecretRefName(s string) (name string, ok bool)`
  - `func (c *Config) AddNotice(msg string)`: the exported `notice`, deduplicated.
  - `func (c Config) PlainSecrets() []string`: the field paths still holding plain values, for example `aiKey`, `providerKeys.INFURA_API_KEY` and `vpns.proton.config`.
  - `Save` returns an error naming the first plain field when `SecretStore == SecretStoreVault`.

- [ ] **Step 1: Write the failing tests**

```go
func TestSecretRefRoundTrip(t *testing.T)            // SecretRef("provider/X")=="vault:provider/X"; SecretRefName ok; IsSecretRef("vaultx")==false
func TestPlainSecretsListsEveryPlainField(t *testing.T) // aiKey literal, one provider literal + one ref, a vpn literal → 3 paths, sorted
func TestSaveRefusesPlainSecretsUnderTheVault(t *testing.T) {
	// Review Focus 4 (half): testutil.Home; Config{SecretStore: "vault", AIKey: "sk-live"}.Save()
	// → error containing "aiKey"; no config.json written. With AIKey "vault:ai-key": ok.
}
func TestSecretStoreSurvivesLoad(t *testing.T)       // Update sets SecretStore and VaultIdleLockMinutes=15; Load returns them
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/config/ -run 'TestSecretRef|TestPlainSecrets|TestSaveRefuses|TestSecretStore' -v`
Expected: build failure, `undefined: SecretRef`.

- [ ] **Step 3: Implement in `internal/config/config.go`.** Update the `VPN.Config`, `AIKey` and `ProviderKeys` comments to say each holds a value or a `vault:` reference.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/config/ -v`
Expected: `ok  	github.com/valve-tech/jumpgate/internal/config`

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/secretref_test.go
git commit -m "feat(config): vault references for secret fields; refuse plain values under the vault"
```

### Task 5: `signer.StoreVault`, the passphrase prompt, and `keys init|show` on the vault

**Files:**
- Create: `internal/signer/vault.go`, `internal/secrets/path.go`, `cmd/jumpgate/passphrase.go`
- Modify: `internal/signer/store.go:20-26,115-145`, `cmd/jumpgate/cli_keys.go`, `go.mod`, `go.sum`
- Test: `internal/signer/vault_test.go`, `cmd/jumpgate/passphrase_test.go`, `cmd/jumpgate/cli_keys_vault_test.go`

**Interfaces:**
- Consumes: Task 3 (`vault.Open`, `Create`, `Unlock`, `Get`, `Put`); Task 4 (`config.SecretStoreVault`).
- Produces:
  - `signer.StoreVault Store = "vault"`
  - `var signer.ErrVaultStore`: the error `Open` and `Create` return for `StoreVault`. Its message says the key opens through an unlocked vault.
  - `func signer.OpenVault(v *vault.Vault, ref string) (*Key, error)`
  - `func signer.CreateVault(v *vault.Vault, ref string) (*Key, error)`: hex value, `ErrKeyExists` when present, read back through `verifyStored`.
  - `func secrets.VaultPath() (string, error)`: `filepath.Join(config.Dir(), "vault")`.
  - In `cmd/jumpgate`: `func readPassphrase(prompt string, fromStdin, confirm bool) ([]byte, error)`, with seams `var openTTY func() (io.ReadWriteCloser, uintptr, error)` and `var stdinReader io.Reader = os.Stdin`.
  - `keys init --store vault`. The default ref is `controller-key`. It records `Controller{KeyStore: "vault", KeyRef: "controller-key"}` and `SecretStore: "vault"`, and prints ``run `jumpgate unlock` so the server uses it``.

- [ ] **Step 1: Write the failing tests**

```go
// internal/signer/vault_test.go
func TestCreateVaultStoresAHexKeyThatOpens(t *testing.T) // v := vaulttest.Unlocked(t); k := CreateVault(v,"controller-key"); OpenVault → same Address; v.Get is 64 hex chars
func TestCreateVaultNeverReplacesAKey(t *testing.T)      // second CreateVault: errors.Is(err, ErrKeyExists)
func TestOpenAndCreateRefuseTheVaultStore(t *testing.T)  // Open(ctx, StoreVault, "x") and Create: errors.Is(err, ErrVaultStore)

// cmd/jumpgate/passphrase_test.go
func TestReadPassphraseFromStdin(t *testing.T) {
	// Review Focus 3: "pw\n"→"pw"; "pw\r\n"→"pw"; " p w \n"→" p w "; "pw"→"pw"; "\n"→ error "empty passphrase";
	// a 1025-byte line → error "too long".
}
func TestReadPassphraseConfirmMismatch(t *testing.T)     // fake tty answers "a","b" → error "passphrases do not match"; nothing returned
func TestReadPassphraseNeedsATerminal(t *testing.T)      // openTTY fails, fromStdin false → error naming --stdin

// cmd/jumpgate/cli_keys_vault_test.go
func TestKeysInitVaultCreatesTheVaultAndRecordsTheKey(t *testing.T) {
	// testutil.Home; fake tty answers vaulttest.Passphrase twice; keysInit([]string{"--store","vault"})==0;
	// config: Controller.KeyStore "vault", KeyRef "controller-key", SecretStore "vault";
	// the vault opens with the passphrase and signer.OpenVault gives Controller.Address.
}
func TestKeysShowOpensAVaultKey(t *testing.T)            // keysShow prompts once (fake tty) and prints the real address
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/signer/ ./cmd/jumpgate/ -run 'TestCreateVault|TestOpenAndCreateRefuse|TestReadPassphrase|TestKeysInitVault|TestKeysShowOpensAVault' -v`
Expected: build failure, `undefined: StoreVault`.

- [ ] **Step 3: Implement**

- **The prompt:** `readPassphrase` opens `/dev/tty` (`CONIN$` on Windows) and reads with `term.ReadPassword` from `github.com/charmbracelet/x/term`. `--stdin` reads one line of at most 1024 bytes. With `confirm`, it reads twice and compares with `subtle.ConstantTimeCompare`.
- **`keys init --store vault`:** it creates the vault when `secrets.VaultPath()` is missing and prints, before asking, `a lost passphrase cannot be recovered: jumpgate keeps no copy`. Otherwise it opens and unlocks the existing vault.
- **`keys show`:** for `StoreVault` it unlocks offline with one prompt.
- **Dependency:** `go mod tidy` moves `charmbracelet/x/term` to a direct require.

- [ ] **Step 4: Run them to see them pass**

Run: the command from Step 2, then `go build ./...`.
Expected: `ok` for both packages; the build prints nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/signer/vault.go internal/signer/store.go internal/signer/vault_test.go internal/secrets/path.go cmd/jumpgate/passphrase.go cmd/jumpgate/passphrase_test.go cmd/jumpgate/cli_keys.go cmd/jumpgate/cli_keys_vault_test.go go.mod go.sum
git commit -m "feat(signer): vault key store, terminal passphrase prompt, keys init --store vault"
```

### Task 6: Backend choice: the vault replaces the file fallback; `--insecure-plain`

**Files:**
- Modify: `internal/signer/store.go:96-114` (`DefaultStore`), `internal/signer/keychain.go:16,141,147`, `internal/signer/store.go:70,72`, `internal/signer/wincred_other.go:7`, `internal/signer/realsecret_test.go:90,107,185`, `internal/signer/secretservice_test.go` (expected fallback), `cmd/jumpgate/cli_keys.go` (`keysInit`), `cmd/jumpgate/appinstance.go:83-100` (`loadServerConfig`), `cmd/jumpgate/help.go`
- Test: `internal/signer/store_test.go`, `cmd/jumpgate/cli_keys_vault_test.go`, `cmd/jumpgate/buildserver_test.go`

**Interfaces:**
- Consumes: Task 4 (`PlainSecrets`, `SecretStoreFile`), Task 5.
- Produces:
  - `DefaultStore()` returns `StoreVault` where it returned `StoreFile`.
  - `keys init --store file` without `--insecure-plain` exits 2 with `--store file keeps the key unencrypted on disk; add --insecure-plain to confirm, or use --store vault`. With the flag, it records `SecretStore: "file"`.
  - `func plainSecretsWarning(cfg config.Config) string` in `cmd/jumpgate/appinstance.go`. It returns the Global Constraints warning when `cfg.SecretStore == "file"`, when `cfg.PlainSecrets()` is non-empty, or when the controller key is `StoreFile`; otherwise it returns `""`. `loadServerConfig` prints it on every start.

- [ ] **Step 1: Write the failing tests**

```go
func TestDefaultStoreFallsBackToTheVault(t *testing.T)  // hostOS "linux", lookPath fails → StoreVault; "darwin" without security → StoreVault
func TestStoreHintsNameTheVault(t *testing.T)           // ErrNoKeychain, ErrKeychainLocked, toolTimeoutErr("security") contain "--store vault", not "--store file"
func TestKeysInitFileNeedsInsecurePlain(t *testing.T)   // exit 2 and the message above; with --insecure-plain: exit 0, SecretStore "file"
func TestServerStartWarnsAboutPlainSecrets(t *testing.T) // cfg with AIKey "sk" → loadServerConfig's stderr contains the exact warning; SecretStore "vault" with refs only → no warning
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/signer/ ./cmd/jumpgate/ -run 'TestDefaultStoreFallsBack|TestStoreHints|TestKeysInitFileNeeds|TestServerStartWarns' -v`
Expected: FAIL. `DefaultStore() = "file", want "vault"`, and `undefined: plainSecretsWarning`.

- [ ] **Step 3: Implement.** Change `DefaultStore`'s doc comment to say "Otherwise the encrypted vault". Change `keys init`'s `--store` usage to `vault | keychain | wincred | 1password | file (file needs --insecure-plain)`. Update the expected values in the existing tests named in Files.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/signer/ ./cmd/jumpgate/ -v`
Expected: `ok` for both packages.

- [ ] **Step 5: Commit**

```bash
git add internal/signer/store.go internal/signer/keychain.go internal/signer/wincred_other.go internal/signer/realsecret_test.go internal/signer/secretservice_test.go internal/signer/store_test.go cmd/jumpgate/cli_keys.go cmd/jumpgate/appinstance.go cmd/jumpgate/help.go cmd/jumpgate/cli_keys_vault_test.go cmd/jumpgate/buildserver_test.go
git commit -m "feat(signer): the vault is the fallback key store; plain files need --insecure-plain"
```

### Task 7: The server's `locked` state, vault routes and error codes

**Files:**
- Create: `internal/server/vault.go`, `internal/api/vault.go`
- Modify: `internal/api/errors.go` (codes and registry), `internal/api/fleet.go` (`Fleet.Locked`), `internal/api/misc.go` (`ControllerView` doc: state `locked`), `internal/server/server.go:113-123` (`Config`) and `Handler`, `internal/server/intent.go:24-27,45`, `internal/server/pair.go:36,163,221`, `internal/server/jobs.go:120-125`, `internal/server/node.go:46-74,112-122,346-349`, `internal/server/uiprefs.go:60-90`, `internal/server/fleet.go` (sets `Locked`)
- Test: `internal/server/vault_test.go`

**Interfaces:**
- Consumes: Task 3 (`vault.Vault`, errors), Task 5 (`signer.OpenVault`, `secrets.VaultPath`).
- Produces:
  - **API types:**
    - `api.VaultView{State string "state"; Slots []string "slots"; Plain []string "plain,omitempty"; Reason string "reason,omitempty"}`, where `State` is `none`, `locked`, `unlocked` or `unusable`.
    - `api.UnlockRequest{Passphrase []byte "passphrase"}`
  - **Error codes:** `api.CodeLocked`, `api.CodeWrongPassphrase`, `api.CodeVaultCorrupt`, `api.CodeVaultUnsafe`, `api.CodeNoVault` (S5). The hints:
    - `locked`: ``unlock the vault: `jumpgate unlock` or the unlock prompt in the TUI or web app; plans wait at their next step until then``
    - `vault_corrupt`: ``the vault file was changed or damaged and jumpgate will not fall back to plain files; restore a backup: `jumpgate stop`, then `jumpgate vault restore FILE` ``
    - `no_vault`: ``create one with `jumpgate vault migrate` (moves existing secrets in) or `jumpgate vault init` ``
  - **`server.Config` fields:**
    - `Vault *vault.Vault`: nil means look for the file lazily (S8).
    - `VaultPath string`: `""` means `secrets.VaultPath()`.
    - `Unlocked func(v *vault.Vault) (signer.Signer, error)`: run after every unlock; it returns the controller signer, or nil when the key is not in the vault.
    - `AfterUnlock func(ctx context.Context)`
  - **Methods** (the server-install plan uses the first three):
    - `func (s *Server) Unlock(passphrase []byte) (api.VaultView, error)`
    - `func (s *Server) UnlockKEK(kind vault.SlotKind, kek []byte) (api.VaultView, error)`
    - `func (s *Server) Lock() api.VaultView`
    - `func (s *Server) VaultState() api.VaultView`
    - `func (s *Server) signer() (signer.Signer, error)`, unexported. It returns `vault.ErrLocked` while a vault-held key is locked.
  - **Routes:** `GET /api/vault`, `POST /api/vault/unlock` (body at most 4 KiB, via `http.MaxBytesReader`), and `POST /api/vault/lock`.

- [ ] **Step 1: Write the failing tests**

```go
func TestVaultRoutesUnlockAndLock(t *testing.T)    // GET → "locked"; POST unlock (base64 passphrase) → 200 "unlocked"; POST lock → "locked"
func TestUnlockAnswersWrongPassphrase(t *testing.T) // 403, Code wrong_passphrase; state still "locked"
func TestUnlockAnswersVaultCorrupt(t *testing.T)   // flip a ciphertext byte → 500 vault_corrupt; no fallback
func TestUnlockWithNoVault(t *testing.T)           // 409 no_vault; after a vault file appears, GET /api/vault → "locked" (lazy open)
func TestUnsafeVaultIsReportedNotTightened(t *testing.T) // chmod 0640 → GET state "unusable", Reason has "chmod"; mode still 0640
func TestLockedServerRefusesSignedRoutes(t *testing.T) {
	// a paired box rig, Controller.KeyStore "vault", locked. Each 503 Code "locked":
	// POST /api/targets/box/intent/<intent.KindStatusRead>, POST /api/targets/box/pair,
	// POST /api/jobs/box, POST /api/fleet/box/check.
}
func TestLockedServerServesViews(t *testing.T) {
	// 200 while locked: GET /api/targets, /api/settings, /api/fleet, /api/update,
	// /api/controller (state "locked"), /api/vault, /api/gateways, /api/jobs, /api/ui-prefs.
}
func TestFleetRowsSayLocked(t *testing.T)          // fleet row Error.Code == "locked", Reachable false; Fleet.Locked true
func TestUnlockResumesOwnJobs(t *testing.T) {
	// newBoxRig with a vault key; start the test job; Lock(); advance past a step: job stays JobWaiting
	// at the boundary (the driver backs off, no advance intent signed); Unlock → the job advances.
}
func TestLockDropsTheSigner(t *testing.T)          // after Lock, s.signer() is vault.ErrLocked; Config.Unlocked ran exactly once per unlock
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/server/ -run 'TestVaultRoutes|TestUnlockAnswers|TestUnlockWithNoVault|TestUnsafeVault|TestLockedServer|TestFleetRowsSayLocked|TestUnlockResumesOwnJobs|TestLockDropsTheSigner' -v`
Expected: build failure, `undefined: api.CodeLocked`.

- [ ] **Step 3: Implement**

- **Signer state:** `internal/server/vault.go` holds `type keyState struct{ mu sync.RWMutex; v *vault.Vault; sgn signer.Signer; fromVault bool; err error }`. It is seeded from `Config.Signer`/`SignerErr`, and every `s.cfg.Signer` read in the files listed above changes to `s.signer()`.
- **Errors:** `classifyNodeError` maps `vault.ErrLocked` to `503 CodeLocked`.
- **After an unlock:** call `Config.Unlocked`, then `go s.jobs.sweep(ctx)`, then `AutostartOverlays` (only the first time this process unlocks), then `Config.AfterUnlock`, then a fleet reload.
- **Passphrase handling:** the handler `clear`s the request's `Passphrase` after `Unlock`, and logs nothing from the body.
- **`errcodes_test.go`:** it already refuses string literals, so the new codes go only through `api.Code*`.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/server/ ./internal/api/ -v`
Expected: `ok` for both packages; every existing test still passes.

- [ ] **Step 5: Commit**

```bash
git add internal/server/vault.go internal/server/vault_test.go internal/api/vault.go internal/api/errors.go internal/api/fleet.go internal/api/misc.go internal/server/server.go internal/server/intent.go internal/server/pair.go internal/server/jobs.go internal/server/node.go internal/server/uiprefs.go internal/server/fleet.go
git commit -m "feat(server): locked state; unlock and lock routes; box routes answer 503 locked"
```

### Task 8: The server reads and writes secrets through references

**Files:**
- Create: `internal/secrets/names.go`
- Modify: `internal/server/api.go:1759` (explain), `internal/server/api.go:2095-2155` (`handlePutSettings`), `internal/server/chainlist.go:235` (`providerKeys`), `internal/server/gateways.go:439`, `internal/server/vpn.go:105-135,240-250,330-380,410-420,471` (`vpnViewFrom`, `bringUpOverlay`, save, delete, up), `internal/server/vault.go`
- Test: `internal/secrets/names_test.go`, `internal/server/vaultrefs_test.go`

**Interfaces:**
- Consumes: Task 4 refs, Task 7 (`s` state).
- Produces:
  - **`internal/secrets/names.go`:**
    - `const NameControllerKey = "controller-key"`
    - `const NameTransportKey = "transport-key"`
    - `const NameAIKey = "ai-key"`
    - `const NameRelayToken = "relay-token"`
    - `const NameAdminToken = "admin-token"`
    - `func ProviderName(placeholder string) string`, which returns `"provider/"+placeholder`
    - `func VPNName(id string) string`, which returns `"vpn/"+id`
    - `func Resolve(v *vault.Vault, s string) (string, error)`: a literal is returned as is; for a reference it calls `v.Get`; a nil or locked vault gives `vault.ErrLocked`.
  - **Server:**
    - `func (s *Server) secret(s string) (string, error)`
    - `func (s *Server) putSecret(name, value string) (stored string, err error)`: it returns the reference under `secretStore: "vault"` and the literal otherwise.
    - `func (s *Server) dropSecret(stored string) error`
    - `providerKeys` becomes `func (s *Server) providerKeys(cfg config.Config) (map[string]string, error)`.
    - Locked behaviour: chainlist uses the defaults alone; knownset and gateway provision answer 503 `locked`.

- [ ] **Step 1: Write the failing tests**

```go
func TestResolvePassesLiteralsAndReadsRefs(t *testing.T)   // secrets.Resolve(nil,"sk")=="sk"; Resolve(nil,"vault:ai-key") → ErrLocked; unlocked → value
func TestSettingsWriteGoesIntoTheVault(t *testing.T)      // SecretStore vault, unlocked; PUT /api/settings {aiKey:"sk-1", providerKeys:{INFURA_API_KEY:"i"}} → config.json has "vault:ai-key" and "vault:provider/INFURA_API_KEY"; vault holds the values; GET settings: aiKeySet true
func TestSettingsWhileLockedWritesNothing(t *testing.T)   // Review Focus 4: locked; PUT /api/settings with aiKey → 503 locked; config.json bytes unchanged
func TestExplainWhileLockedIsLocked(t *testing.T)         // POST /api/targets/{id}/explain → 503 locked
func TestVPNStoredAsARef(t *testing.T)                    // POST /api/vpns {config:<valid conf>} → Config "vault:vpn/<id>"; DELETE removes vault entry
func TestVPNViewWhileLocked(t *testing.T)                 // GET /api/vpns → 200, view Error "locked: unlock to read", Valid false; POST /api/vpns/{id}/up → 503 locked
func TestGatewayProvisionWhileLocked(t *testing.T)        // POST /api/gateways/{gid}/provision with a ref provider key → 503 locked; GET /api/chainlist → 200
func TestPlainStoreStillWritesLiterals(t *testing.T)      // SecretStore "" → PUT settings stores "sk-1" as today
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/secrets/ ./internal/server/ -run 'TestResolve|TestSettingsWrite|TestSettingsWhileLocked|TestExplainWhileLocked|TestVPN|TestGatewayProvisionWhileLocked|TestPlainStoreStill' -v`
Expected: build failure, `undefined: secrets.Resolve`.

- [ ] **Step 3: Implement.** A request that changes a secret fails with `locked` **before** `updateConfig` runs. The vault write happens first, then the config reference is written, so a failure leaves at most an unreferenced vault entry and never a reference to nothing.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/secrets/ ./internal/server/ -v`
Expected: `ok` for both packages.

- [ ] **Step 5: Commit**

```bash
git add internal/secrets/names.go internal/secrets/names_test.go internal/server/api.go internal/server/chainlist.go internal/server/gateways.go internal/server/vpn.go internal/server/vault.go internal/server/vaultrefs_test.go
git commit -m "feat(server): secret fields resolve through vault references; locked refuses secret use"
```

### Task 9: The transport key comes from the vault as an `ssh.Signer`

**Files:**
- Create: `internal/secrets/transport.go`
- Modify: `internal/executor/executor.go:45-65` (`SSHConfig.Signer`), `internal/executor/dial.go:129-185` (`authMethods`), `internal/server/controller.go:30-160` (`ensureTransportKey`, `agentTarget` → methods), `internal/server/intent.go:63`, `internal/server/jobs.go:499`, `internal/server/pair.go:150,216`, `internal/server/node.go:115`
- Test: `internal/secrets/transport_test.go`, `internal/executor/dial_test.go`, `internal/server/controller_test.go`

**Interfaces:**
- Consumes: Task 8 names; Task 3.
- Produces:
  - `executor.SSHConfig.Signer ssh.Signer` with tag `json:"-"`. When set, `authMethods` offers it first, in place of reading `KeyPath`.
  - `func secrets.EnsureTransportKey(v *vault.Vault) (authorizedLine string, err error)`: it creates an ed25519 key in `transport-key` (OpenSSH PEM, comment `jumpgate-controller`) when absent.
  - `func secrets.TransportSigner(v *vault.Vault) (ssh.Signer, error)`
  - `func (s *Server) ensureTransportKey() (string, error)` and `func (s *Server) agentTarget(t config.Target) (agentclient.Target, error)`. Under `secretStore: "vault"` they use the vault and leave `KeyPath` empty; otherwise they keep today's file path.

- [ ] **Step 1: Write the failing tests**

```go
func TestEnsureTransportKeyIsStableInTheVault(t *testing.T) // two calls → same line, ends " jumpgate-controller"; TransportSigner's public key matches
func TestAuthMethodsPrefersAConfiguredSigner(t *testing.T)  // SSHConfig{Signer: s, KeyPath: "/nonexistent"} → no read error; offers s
func TestAgentTargetUsesTheVaultSigner(t *testing.T)        // SecretStore vault: at.SSH.KeyPath=="" && at.SSH.Signer!=nil; locked → errors.Is(err, vault.ErrLocked)
func TestAgentTargetKeepsTheFileWithoutAVault(t *testing.T) // SecretStore "": KeyPath == transportKeyPath()
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/secrets/ ./internal/executor/ ./internal/server/ -run 'TestEnsureTransportKeyIsStable|TestAuthMethodsPrefers|TestAgentTargetUses|TestAgentTargetKeeps' -v`
Expected: build failure, `undefined: secrets.EnsureTransportKey`.

- [ ] **Step 3: Implement.** Generate the key with `ed25519.GenerateKey` and `ssh.MarshalPrivateKey(priv, "jumpgate-controller")`, exactly as `ensureTransportKey` does today.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/secrets/ ./internal/executor/ ./internal/server/ -v`
Expected: `ok` for all three packages.

- [ ] **Step 5: Commit**

```bash
git add internal/secrets/transport.go internal/secrets/transport_test.go internal/executor/executor.go internal/executor/dial.go internal/executor/dial_test.go internal/server/controller.go internal/server/controller_test.go internal/server/intent.go internal/server/jobs.go internal/server/pair.go internal/server/node.go
git commit -m "feat(executor): dial with an in-memory ssh.Signer; the transport key lives in the vault"
```

### Task 10: Migration on unlock, as one transaction

**Files:**
- Create: `internal/secrets/migrate.go`
- Test: `internal/secrets/migrate_test.go`

**Interfaces:**
- Consumes: Tasks 3, 4 and 8; `signer.LoadKeyFile`; `fsatomic.SyncDir`.
- Produces:
  - `type MigrateReport struct{ Moved, Removed, Conflicts []string }`
  - `func Migrate(v *vault.Vault) (MigrateReport, error)`
  - The migration notice text, added with `AddNotice` whenever `Moved` or `Removed` is non-empty: `secrets moved into the vault and their plain copies were overwritten and removed; overwriting is best effort (journaling, copy-on-write filesystems and SSDs may keep old blocks), so treat a key that was ever stored plain on a disk that may have been imaged as exposed and rotate it`
  - Test seam: `var migrateCrash func(step int) error` (unexported).

- [ ] **Step 1: Write the failing tests**

```go
func TestMigrateMovesEveryPlainSecret(t *testing.T) {
	// testutil.Home; plain keys/controller.key (KeyStore "file"), ssh/jumpgate_ed25519, aiKey,
	// providerKeys{INFURA_API_KEY}, vpns[proton].config. Migrate(unlocked vault):
	// both key files gone; config: Controller{KeyStore:"vault",KeyRef:"controller-key"}, aiKey "vault:ai-key",
	// providerKeys "vault:provider/INFURA_API_KEY", vpn "vault:vpn/proton", SecretStore "vault";
	// vault entries equal the old bytes and Entry.From names each origin path or field; one notice.
}
func TestMigrateResumesAfterACrashAtEachStep(t *testing.T) {
	// for step := 1..4: migrateCrash returns an error after that step; Migrate fails; a second
	// Migrate (no crash) ends in the same state as TestMigrateMovesEveryPlainSecret, and no
	// secret exists both plain and in the vault with a reference pointing at it.
}
func TestMigrateKeepsAConflictingPlainCopy(t *testing.T) {
	// vault already has ai-key "old"; config aiKey "new": aiKey stays "new", vault keeps "old",
	// SecretStore is NOT set to "vault", Conflicts == ["aiKey"], a notice names aiKey and no value.
}
func TestMigrateOverwritesBeforeUnlinking(t *testing.T) // seam wipeHook sees the file all zeros, same length, before Remove
func TestMigrateIsANoOpWhenNothingIsPlain(t *testing.T) // empty report, config.json bytes unchanged
func TestMigrateLeavesOSStoreKeysAlone(t *testing.T)    // KeyStore "keychain": Controller unchanged; rows 2-5 still move
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/secrets/ -run 'TestMigrate' -v`
Expected: build failure, `undefined: Migrate`.

- [ ] **Step 3: Implement `Migrate`**

The steps follow V6 exactly:

1. Collect the plain values, and skip any whose name the vault already holds with an equal value.
2. Run one `PutAll`, which verifies by reading the file back.
3. Run one `config.Update` that swaps in the references for values now equal in the vault. Set `SecretStore` only when `Conflicts` is empty.
4. For each plain key file whose bytes equal the vault's copy: open it `O_WRONLY|O_NOFOLLOW`, write zeros over its length, `Sync`, `Remove`, then `fsatomic.SyncDir`.

The `From` field holds the origin path for a file, or the config field path for a config value. A file that matches the vault on a later run only finishes step 4.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/secrets/ -v`
Expected: `ok  	github.com/valve-tech/jumpgate/internal/secrets`

- [ ] **Step 5: Commit**

```bash
git add internal/secrets/migrate.go internal/secrets/migrate_test.go
git commit -m "feat(secrets): migrate plain secrets into the vault in one resumable transaction"
```

### Task 11: The composition root: open the vault at start, unlock hooks, core dumps

**Files:**
- Modify: `cmd/jumpgate/buildserver.go:200-260` (`buildServer`), `cmd/jumpgate/appinstance.go:104-120` (`openControllerKey`), `internal/config/config.go` (`TightenState` doc only)
- Test: `cmd/jumpgate/buildserver_test.go`, `internal/server/login_linux_test.go`, `internal/config/tighten_test.go`

**Interfaces:**
- Consumes: Tasks 7, 9 and 10; `vault.DisableCoreDumps`.
- Produces:
  - `func openControllerKey(cfg config.Config, v *vault.Vault) (signer.Signer, error)`: `StoreVault` with a locked or nil vault gives `vault.ErrLocked`.
  - `func credentialKEK(getenv func(string) string) ([]byte, bool)`: it reads `$CREDENTIALS_DIRECTORY/jumpgate-vault-key` (32 bytes, through `readPrivateFile`).
  - In `buildServer`:
    - `server.Config{Vault, Unlocked: func(v) { secrets.Migrate(v); log the report; reload config; openControllerKey(cfg, v) }}`.
    - A background task `vault-auto-unlock`: first the credential, then each `keystore` slot (Task 16). A failure logs `vault stays locked: <reason>; the passphrase still works`.
    - `vault.DisableCoreDumps()` whenever a vault exists or `SecretStore` is `vault`, before the first unlock.

- [ ] **Step 1: Write the failing tests**

```go
func TestBuildServerStartsLockedWithAVaultKey(t *testing.T) // vaulttest vault holding controller-key; Controller KeyStore vault → builtServer's /api/controller state "locked"; signerErr is vault.ErrLocked
func TestUnlockRunsMigrationThenOpensTheKey(t *testing.T)    // plain aiKey + vault; s.Unlock(pass) → config aiKey "vault:ai-key"; /api/controller state "ok"
func TestCredentialUnlocksAtStart(t *testing.T)             // vault with a systemd-creds slot; CREDENTIALS_DIRECTORY with the KEK file → state "unlocked" after the task runs
func TestBadCredentialStaysLocked(t *testing.T)             // wrong KEK → "locked", log line contains "the passphrase still works"
func TestTightenStateLeavesTheVaultAlone(t *testing.T)      // vault chmod 0640 → TightenState does not list or change it
func TestLoginPeerLookupWorksWhenNotDumpable(t *testing.T)  // Review Focus 5 (linux): DisableCoreDumps(); the existing peer-uid lookup finds this process's loopback connection; restore with prctl(PR_SET_DUMPABLE,1)
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./cmd/jumpgate/ ./internal/server/ ./internal/config/ -run 'TestBuildServerStartsLocked|TestUnlockRunsMigration|TestCredentialUnlocks|TestBadCredential|TestTightenStateLeavesTheVault|TestLoginPeerLookupWorksWhenNotDumpable' -v`
Expected: build failure, `too many arguments in call to openControllerKey`.

- [ ] **Step 3: Implement.** The doc comment of `buildServer` now says that a vault key starts `locked`, distinct from `no_controller_key`.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./cmd/jumpgate/ ./internal/server/ ./internal/config/ -v`
Expected: `ok` for all three packages.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/buildserver.go cmd/jumpgate/appinstance.go cmd/jumpgate/buildserver_test.go internal/server/login_linux_test.go internal/config/config.go internal/config/tighten_test.go
git commit -m "feat(serve): start locked with a vault; unlock migrates and opens the key; core dumps off"
```

### Task 12: CLI `unlock`, `lock` and `home`; no passphrase on argv or in the environment

**Files:**
- Create: `cmd/jumpgate/cli_unlock.go`
- Modify: `cmd/jumpgate/cli.go:23-28` (subcommands), `cmd/jumpgate/help.go`, `cmd/jumpgate/home_terminal.go:110-140`, `internal/apiclient/methods.go`
- Test: `cmd/jumpgate/cli_unlock_test.go`, `cmd/jumpgate/unlock_linux_test.go`, `cmd/jumpgate/home_terminal_test.go`

**Interfaces:**
- Consumes: Task 7 routes and types, Task 5 `readPassphrase`.
- Produces:
  - `func (c *Client) Vault(ctx context.Context) (api.VaultView, error)`
  - `func (c *Client) Unlock(ctx context.Context, passphrase []byte) (api.VaultView, error)`: clears `passphrase`.
  - `func (c *Client) Lock(ctx context.Context) (api.VaultView, error)`
  - Commands: `jumpgate unlock [--stdin]` and `jumpgate lock`. Errors print through `reportServerError`, so `wrong_passphrase` exits 1, `vault_corrupt` 4, and `no_vault` 2.
  - The `home` line, while locked: ``  vault:    locked: jobs wait for unlock; run `jumpgate unlock` ``.
  - Help rows: `{"unlock", "unlock the vault so the server can sign"}` and `{"lock", "drop the vault key from the server's memory"}`.

- [ ] **Step 1: Write the failing tests**

```go
func TestUnlockSendsThePassphraseOnce(t *testing.T)   // in-process server (connect seam); stdin "pw\n" → exit 0, prints "unlocked"; the server saw []byte("pw")
func TestUnlockWrongPassphraseExits1(t *testing.T)
func TestUnlockRefusesArgv(t *testing.T)              // `jumpgate unlock pw` → exit 2, "the passphrase is never taken from the command line; use --stdin or a terminal"
func TestHomeShowsLocked(t *testing.T)                // fake server state locked → the exact home line
func TestUnlockPassphraseNeverInArgvOrEnviron(t *testing.T) {
	// linux only: go build the binary into t.TempDir(); HOME temp with a vaulttest vault and a vault controller key;
	// start `jumpgate serve --no-open`; run `jumpgate unlock --stdin` with a random 24-char passphrase on stdin;
	// while it runs, sample every 1ms /proc/<pid>/cmdline and /proc/<pid>/environ of every process whose
	// PPid chain reaches this test's pid; after it exits, sample serve once more. The passphrase never appears.
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./cmd/jumpgate/ -run 'TestUnlock|TestHomeShowsLocked' -v`
Expected: build failure, `undefined: cmdUnlock`.

- [ ] **Step 3: Implement `cmd/jumpgate/cli_unlock.go` and the client methods.** `unlock` refuses any positional argument.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./cmd/jumpgate/ ./internal/apiclient/ -v`
Expected: `ok` for both packages. On macOS the linux test reports `--- SKIP`.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/cli_unlock.go cmd/jumpgate/cli_unlock_test.go cmd/jumpgate/unlock_linux_test.go cmd/jumpgate/cli.go cmd/jumpgate/help.go cmd/jumpgate/home_terminal.go cmd/jumpgate/home_terminal_test.go internal/apiclient/methods.go
git commit -m "feat(cli): jumpgate unlock and lock; home shows the vault state"
```

### Task 13: TUI: `LOCKED` badge, unlock prompt, locked rows

**Files:**
- Create: `internal/tui/unlock.go`
- Modify: `internal/tui/backend.go` (three methods), `internal/tui/tuitest/fake.go`, `internal/tui/app.go` (`topBar`, key handling), `internal/tui/keys.go` (`Unlock` `U`, `Lock` `L`), `internal/tui/fleet.go` (row text), `internal/tui/help.go`
- Test: `internal/tui/unlock_test.go`

**Interfaces:**
- Consumes: Task 12's client methods (the same signatures go on `Backend`) and `api.Fleet.Locked`.
- Produces: `tuitest.Fake` fields `VaultV api.VaultView`, `UnlockErr error` and `Unlocks [][]byte` (copies of what was sent).

- [ ] **Step 1: Write the failing tests**

```go
func TestTopBarShowsLocked(t *testing.T)           // fleetMsg with Locked true → frame's first line contains "LOCKED: jobs wait for unlock"
func TestUnlockPromptHidesInput(t *testing.T)      // press U, type "secret" → frame shows "••••••" (ascii glyphs: "******"), never "secret"
func TestUnlockPromptSendsAndClears(t *testing.T)  // enter → f.Unlocks[0]=="secret"; the modal's buffer is all zero; modal closes
func TestUnlockPromptShowsWrongPassphrase(t *testing.T) // UnlockErr wrong_passphrase → modal stays open, says "wrong passphrase", field empty
func TestLockedRowsSayUnlockToRead(t *testing.T)   // a row with Error.Code "locked" renders "locked: unlock to read", not "unreachable"
func TestLockKeyLocks(t *testing.T)                // press L → f.VaultV.State "locked" requested via Lock
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/tui/ -run 'TestTopBarShowsLocked|TestUnlockPrompt|TestLockedRows|TestLockKey' -v`
Expected: build failure, `undefined: tuitest.Fake.VaultV` (or `Backend` not satisfied).

- [ ] **Step 3: Implement.** The unlock modal implements `modal` and `pasteTaker`, and holds its input in a `[]byte` that it clears on close. The custody line that sibling D10 shows once per unlock belongs to the server-install plan; leave a `custodyLine string` field on the modal, empty here.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/tui/... -v`
Expected: `ok` for each package; the ASCII sweep (`TestASCIIFrameHasOnlyASCII`) still passes.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/unlock.go internal/tui/unlock_test.go internal/tui/backend.go internal/tui/tuitest/fake.go internal/tui/app.go internal/tui/keys.go internal/tui/fleet.go internal/tui/help.go
git commit -m "feat(tui): LOCKED badge, hidden unlock prompt, locked box rows"
```

### Task 14: Web UI and desktop: lock banner, unlock dialog, plain-secrets notice

**Files:**
- Create: `cmd/jumpgate/web/src/components/VaultBanner.tsx`, `cmd/jumpgate/web/src/components/UnlockDialog.tsx`, `cmd/jumpgate/web/src/hooks/vault.ts`
- Modify: `cmd/jumpgate/web/src/api.ts` (`getVault`, `unlockVault`, `lockVault`), `cmd/jumpgate/web/src/App.tsx:81` (beside `<UpdateBanner />`), `cmd/jumpgate/web/dist/` (rebuilt)
- Test: `cmd/jumpgate/web/src/components/VaultBanner.test.tsx`, `cmd/jumpgate/web/src/components/UnlockDialog.test.tsx`

**Interfaces:**
- Consumes: `GET /api/vault`, `POST /api/vault/unlock` with body `{"passphrase": base64(utf8)}`, and `POST /api/vault/lock`.
- Produces:
  - `useVault()`: polls every 5 s, in the style of `hooks/update`.
  - `VaultBanner`: locked shows `Locked: box actions and plans wait until you unlock.` with Unlock and Lock buttons. `plain` non-empty shows `Secrets are stored unencrypted. Run jumpgate vault migrate.`
  - `UnlockDialog`: opens once per page load when the state is `locked` (the desktop prompt at launch, V5), and can be dismissed. It uses `type="password"` and clears the field after a send.

- [ ] **Step 1: Write the failing tests**

```tsx
it("shows the locked banner and opens the dialog")            // state locked → banner text; dialog open once; dismiss → closed, not reopened on next poll
it("sends the passphrase base64 and clears the field")        // fetch body {"passphrase":"cMOkc3M="} for "päss"; input value "" afterwards
it("shows wrong passphrase without closing")                  // 403 wrong_passphrase → "Wrong passphrase", dialog open
it("shows the plain-secrets notice")                          // state none, plain ["aiKey"] → the exact copy
it("lock button calls lock")                                  // unlocked → Lock → POST /api/vault/lock
```

- [ ] **Step 2: Run them to see them fail**

Run: `cd cmd/jumpgate/web && npx vitest run src/components/VaultBanner.test.tsx src/components/UnlockDialog.test.tsx`
Expected: FAIL, `Failed to resolve import "./VaultBanner"`.

- [ ] **Step 3: Implement the components, the hook and the `api.ts` functions.**

- [ ] **Step 4: Run them to see them pass, then rebuild**

Run: `cd cmd/jumpgate/web && npm test && npm run build && cd ../../.. && go test ./cmd/jumpgate/ -run TestEmbedded -v`
Expected: vitest reports every test passed; the build writes `dist/`; the Go test is `ok` (or `no tests to run`).

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/web/src/components/VaultBanner.tsx cmd/jumpgate/web/src/components/UnlockDialog.tsx cmd/jumpgate/web/src/hooks/vault.ts cmd/jumpgate/web/src/components/VaultBanner.test.tsx cmd/jumpgate/web/src/components/UnlockDialog.test.tsx cmd/jumpgate/web/src/api.ts cmd/jumpgate/web/src/App.tsx
git add cmd/jumpgate/web/dist
git commit -m "feat(web): vault banner and unlock dialog; desktop prompts once at launch"
```

### Task 15: `jumpgate vault init|migrate|backup|restore|add-recovery|rekdf`

**Files:**
- Create: `cmd/jumpgate/cli_vault.go`
- Modify: `cmd/jumpgate/cli.go` (subcommands), `cmd/jumpgate/help.go`
- Test: `cmd/jumpgate/cli_vault_test.go`

**Interfaces:**
- Consumes: Task 3 (`Create`, `AddRecovery`, `Rekdf`), Task 5 (`readPassphrase`, `secrets.VaultPath`), Task 12 (`apiclient.Unlock`), `daemon.Find`.
- Produces:
  - `func cmdVault(args []string) int`, with each subcommand below.
  - `init [--stdin]`: prints `a lost passphrase cannot be recovered: jumpgate keeps no copy`, asks twice (with `--stdin`, reads one line once), and creates the vault. An existing vault exits 1 (`a vault already exists at PATH`).
  - `migrate [--stdin]`: runs `init` when needed, then `apiclient.Unlock`. The server's unlock runs `Migrate`. It prints the report and the exposure notice.
  - `backup FILE`: copies the file byte for byte through `fsperm.CreatePrivate` (refusing an existing FILE), under the shared lock, and prints `the backup is useless without the passphrase; TPM and keystore slots do not travel`.
  - `restore FILE`: refuses while `daemon.Find` answers (``run `jumpgate stop` first``). It refuses when a vault exists without `--replace`, `vault.Open`s FILE first (integrity and header), then copies it into place.
  - `add-recovery`: unlocks with a prompt, prints the code once between two lines of `-`, and says to store it offline.
  - `rekdf [--memory KiB] [--time N] [--threads N] [--new-passphrase]`: flags default to `DefaultParams`. A value below the floor exits 2.

- [ ] **Step 1: Write the failing tests**

```go
func TestVaultInitAsksTwiceAndWarns(t *testing.T)     // fake tty "a","a" → exit 0, stdout has the lost-passphrase line; vault opens with "a"
func TestVaultMigrateMovesSecrets(t *testing.T)       // plain aiKey; in-process server; migrate --stdin → config ref, vault value
func TestVaultBackupIsAByteCopy(t *testing.T)         // bytes equal; backup is private; existing FILE → exit 1
func TestVaultRestoreRefusesARunningServer(t *testing.T)
func TestVaultRestoreChecksTheFileFirst(t *testing.T) // corrupt FILE → exit 4, the current vault untouched
func TestVaultAddRecoveryPrintsOnce(t *testing.T)     // 24 words on one line; Unlock(code) works
func TestVaultRekdfRefusesBelowTheFloor(t *testing.T) // --memory 1024 → exit 2
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./cmd/jumpgate/ -run 'TestVault' -v`
Expected: build failure, `undefined: cmdVault`.

- [ ] **Step 3: Implement `cmd/jumpgate/cli_vault.go`.**

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./cmd/jumpgate/ -v`
Expected: `ok  	github.com/valve-tech/jumpgate/cmd/jumpgate`

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/cli_vault.go cmd/jumpgate/cli_vault_test.go cmd/jumpgate/cli.go cmd/jumpgate/help.go
git commit -m "feat(cli): vault init, migrate, backup, restore, add-recovery and rekdf"
```

### Task 16: `vault seal` (systemd-creds) and the keystore slot for OS-store users

**Files:**
- Modify: `cmd/jumpgate/cli_vault.go` (`seal`, and `init --keystore`), `cmd/jumpgate/buildserver.go` (the keystore half of `vault-auto-unlock`)
- Create: `internal/secrets/keystore.go`
- Test: `cmd/jumpgate/cli_vault_seal_test.go`, `internal/secrets/keystore_test.go`

**Interfaces:**
- Consumes: Task 3 (`AddKEKSlot`, `UnlockKEK`, `Slots`), Task 11 (`credentialKEK`), `signer.Create`/`Open`.
- Produces:
  - `jumpgate vault seal (--tpm2 | --allow-host-key) [--out PATH]`. The default `--out` is `/etc/credstore.encrypted/jumpgate-vault-key`.
    - It unlocks offline with a prompt and draws a 32-byte KEK.
    - It runs `systemd-creds encrypt --with-key=tpm2|host --name=jumpgate-vault-key - OUT` with the KEK on stdin (never argv), through a seam `var runSystemdCreds func(ctx, stdin []byte, args ...string) error`.
    - It then runs `AddKEKSlot(Slot{Kind: SlotSystemdCreds, Cred: "jumpgate-vault-key"}, kek)`.
    - `--tpm2` first runs `systemd-creds has-tpm2`. On failure it exits 1 with `no usable TPM2 here; --allow-host-key seals with the host key instead, which only protects against copying the vault file alone`.
    - `--allow-host-key` prints that sentence too.
  - `func secrets.KeystoreKEK(ctx context.Context, store signer.Store, ref string) ([]byte, error)`: HKDF-SHA256 with info `jumpgate vault keystore kek v1` over the key's `Bytes()`.
  - `func secrets.AddKeystoreSlot(ctx context.Context, v *vault.Vault, store signer.Store, ref string) error`: `signer.Create` (or `Open` when it already exists), then `AddKEKSlot(Slot{Kind: SlotKeystore, Store, Ref}, kek)`.
  - `vault init [--keystore auto|none]`: `auto` adds a keystore slot with ref `jumpgate-vault` when `signer.DefaultStore()` is `keychain` or `wincred` (Q1 default). The server's auto-unlock tries each keystore slot after the credential.

- [ ] **Step 1: Write the failing tests**

```go
func TestSealPipesTheKEKOnStdin(t *testing.T)     // fake runSystemdCreds: args contain "--with-key=host" and OUT, never the KEK hex/base64; stdin is 32 bytes; slot added; UnlockKEK(stdin bytes) works
func TestSealTPM2WithoutATPMRefuses(t *testing.T) // has-tpm2 fails → exit 1 with the sentence; no slot added
func TestSealNeedsAMode(t *testing.T)             // neither flag → exit 2
func TestKeystoreSlotUnlocksWithoutAPrompt(t *testing.T) // signer fakes (hostOS darwin, runCmd fake security); AddKeystoreSlot; Lock; KeystoreKEK → UnlockKEK ok
func TestVaultInitAddsAKeystoreSlotOnMacOS(t *testing.T) // DefaultStore keychain (fake) → Slots has passphrase/primary and keystore
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./cmd/jumpgate/ ./internal/secrets/ -run 'TestSeal|TestKeystoreSlot|TestVaultInitAddsAKeystore' -v`
Expected: build failure, `undefined: secrets.KeystoreKEK`.

- [ ] **Step 3: Implement.** The signer test seams (`hostOS`, `runCmd`, `lookPath`) are unexported, so `internal/secrets` tests drive the keystore through an exported test-only hook: `signer.SetToolRunnerForTest(t, fn)`, added in `internal/signer/export_testhooks.go` and guarded by `testing.Testing()`.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./cmd/jumpgate/ ./internal/secrets/ ./internal/signer/ -v`
Expected: `ok` for all three packages.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/cli_vault.go cmd/jumpgate/cli_vault_seal_test.go cmd/jumpgate/buildserver.go internal/secrets/keystore.go internal/secrets/keystore_test.go internal/signer/export_testhooks.go
git commit -m "feat(vault): seal with systemd-creds; an OS key store slot unlocks without a prompt"
```

### Task 17: `jumpgate keys export --keystore-v3 FILE`

**Files:**
- Create: `internal/signer/keystorev3.go`
- Modify: `cmd/jumpgate/cli_keys.go` (`export` subcommand and usage)
- Test: `internal/signer/keystorev3_test.go`, `cmd/jumpgate/cli_keys_vault_test.go`

**Interfaces:**
- Consumes: Task 5 (`OpenVault`, `readPassphrase`), `eip712.Keccak256`, `golang.org/x/crypto/scrypt`.
- Produces:
  - `func signer.EncryptKeystoreV3(k *Key, passphrase []byte, scryptN int) ([]byte, error)`: v3 JSON with cipher `aes-128-ctr`, kdf `scrypt` (`n`=scryptN, `r`=8, `p`=1, `dklen`=32), mac `keccak256(dk[16:32] || ciphertext)`, a random uuid v4 `id`, and `address` in lowercase hex without `0x`. It clears `passphrase`.
  - `func signer.DecryptKeystoreV3(b, passphrase []byte) (*Key, error)`
  - `keys export --keystore-v3 FILE`:
    - It works only for `KeyStore "vault"`; any other store exits 2 with `only a vault key is exported; the other stores already hold it where a wallet or tool can reach it`.
    - The vault passphrase comes from `/dev/tty`, then the keystore passphrase twice from `/dev/tty`. There is no `--stdin`.
    - It writes with `scryptN=262144` through `fsperm.CreatePrivate` (refusing an existing FILE), reads the file back with `DecryptKeystoreV3`, and checks the address.

- [ ] **Step 1: Write the failing tests**

```go
func TestKeystoreV3KnownVector(t *testing.T) {
	// The Web3 Secret Storage Definition scrypt test vector: password "testpassword",
	// salt ab0c7876052600dd703518d6fc3fe8984592145b591fc8fb5c6d43190334ba19,
	// iv 83dbcc02d8ccb40e466191a123791e0e, n 262144 r 8 p 1,
	// ciphertext d172bf743a674da9cdad04534d56926ef8358534d458fffccd4e6ad2fbde479c,
	// mac 2103ac29920d71da29f15d75b4a16dbe95cfd7ff8faea1056c33131d846e3097
	// → key 7a28b5ba57c53603b0b07b56bba752f7784bf506fa95edc395f5cf6c7514fe9d.
	// Check the values against the published page when writing the test.
}
func TestKeystoreV3RoundTrip(t *testing.T)       // scryptN 1<<12 in tests; Encrypt → Decrypt same Address; wrong passphrase → error mentioning "mac"
func TestKeysExportRefusesOtherStores(t *testing.T)
func TestKeysExportWritesAPrivateFile(t *testing.T) // fake tty; file private; DecryptKeystoreV3 gives Controller.Address; existing FILE → exit 1
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/signer/ ./cmd/jumpgate/ -run 'TestKeystoreV3|TestKeysExport' -v`
Expected: build failure, `undefined: EncryptKeystoreV3`.

- [ ] **Step 3: Implement.** `DecryptKeystoreV3` accepts only `scrypt` and `aes-128-ctr`; it refuses pbkdf2 with an error that names it.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/signer/ ./cmd/jumpgate/ -v`
Expected: `ok` for both packages.

- [ ] **Step 5: Commit**

```bash
git add internal/signer/keystorev3.go internal/signer/keystorev3_test.go cmd/jumpgate/cli_keys.go cmd/jumpgate/cli_keys_vault_test.go
git commit -m "feat(keys): export the controller key as an Ethereum keystore v3 file"
```

### Task 18: Relay and admin tokens from the vault (V9)

**Files:**
- Modify: `cmd/jumpgate/buildserver.go:50-90,200-260` (`readToken` callers, deferred relay), `cmd/jumpgate/cli_relay.go` (refuse references), `internal/server/server.go` (`AttachRelay`), `internal/server/vault.go` (`PUT /api/vault/secrets/{name}`), `cmd/jumpgate/cli_vault.go` (`put`)
- Test: `cmd/jumpgate/buildserver_test.go`, `cmd/jumpgate/cli_relay_test.go`, `internal/server/vault_test.go`

**Interfaces:**
- Consumes: Tasks 7, 8 and 11.
- Produces:
  - `JUMPGATE_RELAY_TOKEN=vault:relay-token` and `JUMPGATE_ADMIN_TOKEN=vault:admin-token` are accepted by `serve` and the app. `relay.Build`/`BuildAdmin` then run inside `AfterUnlock`, followed by `s.AttachRelay`, the `relay-listener` and the `relay-runtime`. The start log says `relay waits for the vault to be unlocked`.
  - `func (s *Server) AttachRelay(h http.Handler, keys KeyAdmin) error`: it may be called once, and only when `Config.Relay` was nil.
  - `jumpgate relay` with a `vault:` value exits 2 with `the standalone relay never opens controller state; use JUMPGATE_RELAY_TOKEN_FILE or systemd LoadCredential=`.
  - `PUT /api/vault/secrets/{name}` with body `{"value": base64}`. It accepts only `relay-token` and `admin-token` (any other name is 400), and answers 503 `locked` while locked.
  - `jumpgate vault put relay-token|admin-token [--stdin]`.

- [ ] **Step 1: Write the failing tests**

```go
func TestRelayWithAVaultTokenStartsAfterUnlock(t *testing.T) // relay-bind set, token "vault:relay-token"; before unlock: no listener; Unlock → listener answers
func TestStandaloneRelayRefusesVaultRefs(t *testing.T)
func TestVaultPutAcceptsOnlyTokens(t *testing.T)            // PUT .../relay-token → 204; PUT .../ai-key → 400; locked → 503 locked
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./cmd/jumpgate/ ./internal/server/ -run 'TestRelayWithAVaultToken|TestStandaloneRelayRefuses|TestVaultPutAccepts' -v`
Expected: build failure, `undefined: (*Server).AttachRelay`.

- [ ] **Step 3: Implement.**

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./cmd/jumpgate/ ./internal/server/ -v`
Expected: `ok` for both packages.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/buildserver.go cmd/jumpgate/buildserver_test.go cmd/jumpgate/cli_relay.go cmd/jumpgate/cli_relay_test.go internal/server/server.go internal/server/vault.go internal/server/vault_test.go cmd/jumpgate/cli_vault.go
git commit -m "feat(relay): in-process relay and admin tokens from the vault, started after unlock"
```

### Task 19: Idle auto-lock setting, off by default

**Files:**
- Modify: `internal/api/misc.go:20-34` (`Settings.VaultIdleLockMinutes int "vaultIdleLockMinutes"`, `SettingsUpdate.VaultIdleLockMinutes *int`), `internal/server/api.go` (`settingsRequest`, `settingsResponseFrom`, `handlePutSettings`), `internal/server/server.go` (`authMiddleware` marks activity), `internal/server/vault.go` (the idle loop), `internal/server/jobs.go` (`func (h *jobsHub) driverCount() int`), `internal/tui/settings.go`, `cmd/jumpgate/web/src/screens/Settings/Settings.tsx`, `cmd/jumpgate/web/dist/`
- Test: `internal/server/vaultidle_test.go`, `internal/tui/settings_test.go`, `cmd/jumpgate/web/src/screens/Settings/Settings.test.tsx`

**Interfaces:**
- Consumes: Task 4 `Config.VaultIdleLockMinutes`, Task 7.
- Produces:
  - **Setting:** 0 (the default) is off. Allowed values are 0 to 1440; anything else is 400 `minutes must be between 0 (off) and 1440`.
  - **Idle loop:** it ticks every 30 s (`var vaultIdleTick`, shortened in tests). It locks when the setting is above 0, the vault is unlocked, no POST, PUT or DELETE has arrived for N minutes, and `driverCount() == 0` (S9). The log line is `vault locked after N minutes idle`.
  - **Copy:** the TUI and web Settings explain `Locks the vault after this many idle minutes. Plans in progress keep it unlocked. 0 means never.`

- [ ] **Step 1: Write the failing tests**

```go
func TestIdleLockIsOffByDefault(t *testing.T)        // unlocked; clock +24h; still unlocked
func TestIdleLockLocksAfterNMinutes(t *testing.T)    // setting 5; no writes for 5m → locked; log line
func TestIdleLockWaitsForRunningPlans(t *testing.T)  // a driven job → never locks; job ends → locks on next tick
func TestReadsDoNotCountAsActivity(t *testing.T)     // GET /api/fleet every minute → still locks; one PUT resets the clock
func TestIdleLockMinutesAreBounded(t *testing.T)     // 1441 → 400
```

Add the TUI row test (`TestSettingsShowsIdleLock`) and the web Settings test (`it("saves the idle lock minutes")`) following each file's existing pattern.

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/server/ -run 'TestIdleLock|TestReadsDoNotCount' -v`
Expected: build failure, `undefined: vaultIdleTick`.

- [ ] **Step 3: Implement.** The server takes the clock from its existing `Now`-style seam, or adds `Config.now func() time.Time` (unexported, tests only) if there is none.

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/server/ ./internal/tui/ -v && cd cmd/jumpgate/web && npm test && npm run build`
Expected: `ok` for both Go packages; vitest passes; the build writes `dist/`.

- [ ] **Step 5: Commit**

```bash
git add internal/api/misc.go internal/server/api.go internal/server/server.go internal/server/vault.go internal/server/jobs.go internal/server/vaultidle_test.go internal/tui/settings.go internal/tui/settings_test.go cmd/jumpgate/web/src/screens/Settings/Settings.tsx cmd/jumpgate/web/src/screens/Settings/Settings.test.tsx
git add cmd/jumpgate/web/dist
git commit -m "feat(server): optional idle auto-lock that never interrupts a running plan"
```

### Task 20: End to end: a plan waits while locked; systemd-creds unlock at start

**Files:**
- Create: `internal/server/e2e_vault_test.go` (`//go:build e2e`), `scripts/e2e-vault.sh`
- Modify: `scripts/e2e-agent.sh` (run the new server e2e after the bootstrap ones)
- Test: the two above.

**Interfaces:**
- Consumes: everything above; `bootstrap.Run`, `agentclient`, the `JUMPGATE_E2E_*` environment `scripts/e2e-agent.sh` already exports.
- Produces: nothing new for other tasks.

- [ ] **Step 1: Write the tests**

`internal/server/e2e_vault_test.go`:

```go
// TestE2EPlanWaitsAtTheStepBoundaryWhileLocked: pair the container box (bootstrap.Run with the
// E2E transport key), build a Server with a vaulttest vault holding controller-key and the
// transport key, Unlock, start node.migrate-names (the seeded legacy node from e2e_jobs_test.go's
// recipe, copied), Lock() while a step runs. Assert the step finishes on the box, the job sits
// JobWaiting at the next boundary for 3 × jobsDriveEvery, and no job.advance reached the agent
// (its log length is unchanged). Unlock: the job advances within 30 s.
func TestE2EPlanWaitsAtTheStepBoundaryWhileLocked(t *testing.T)
```

`scripts/e2e-vault.sh`:

1. Build `jumpgate` for the container's arch and start the `scripts/e2e` image.
2. As `alice`, run `jumpgate vault init --stdin --keystore none` with the passphrase piped in.
3. As root, run `jumpgate vault seal --allow-host-key --out /etc/credstore.encrypted/jumpgate-vault-key` with `HOME=/home/alice`, then check that the vault is still owned by `alice` (S10).
4. Run `systemd-run --uid=alice -E HOME=/home/alice -p LoadCredentialEncrypted=jumpgate-vault-key --unit=jg-vault-e2e /usr/local/bin/jumpgate serve --no-open`.
5. Poll `GET /api/vault` (as alice, token from `server.json`) until `"state":"unlocked"`, then print `PASS systemd-creds unlock at start`.
6. Remove the credential file, restart the unit, and expect `"locked"`. Then print `PASS locked without the credential`.

The script does no TPM run: that one is manual, as the spec says.

- [ ] **Step 2: Run them**

Run: `scripts/e2e-agent.sh` (it now ends with `go test -tags e2e -count=1 -v ./internal/server/ -run E2E`), then `scripts/e2e-vault.sh`.
Expected: `--- PASS: TestE2EPlanWaitsAtTheStepBoundaryWhileLocked`, then both `PASS` lines; each script exits 0.

- [ ] **Step 3: Fix whatever the real box shows, in the task that owns the code, with a unit test first.**

- [ ] **Step 4: Run the whole suite**

Run: `go test ./... && (cd cmd/jumpgate/web && npm test) && GOOS=windows go vet ./... && GOOS=linux GOARCH=arm64 go build ./...`
Expected: every package `ok`, vitest passes, vet and the build print nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/server/e2e_vault_test.go scripts/e2e-vault.sh scripts/e2e-agent.sh
git commit -m "test(e2e): a plan waits while locked; systemd-creds unlocks the service at start"
```

---

## Self-review

**Spec coverage:**

| Spec item | Task(s) |
|---|---|
| Inventory rows 1 to 5 into the vault | 5, 8, 9, 10 |
| Row 6, optional | 18 |
| Rows 7 to 12 stay out of the vault | Global Constraints; 11 (`TightenState`) |
| V1: path, `O_NOFOLLOW`, private, owner, atomic, 1 MiB | 2, Global Constraints |
| V2: format, argon2id and XChaCha, AD, nonces, floor, rotation, body names | 1, 3, S1 to S4 |
| V3: `DefaultStore` → vault | 6 |
| V3: other secrets follow (Q1 default) | 16 |
| V3: `--insecure-plain` and the warning | 6 |
| V3: references | 4, 8 |
| V4: unlock surfaces | 12, 13, 14, 7, 11 |
| V4: no argv or env | 12 |
| V4: clearing | 2, 7, 12, 13 |
| V4: core dumps | 3, 11 |
| V4: error codes | 7 |
| V5: the three columns | 7, 8, 18 |
| V5: TUI badge and rows | 13 |
| V5: desktop prompt | 14 |
| V6: migration | 10, 11, 15 |
| V7: init warning, backup, recovery, restore | 15 |
| V8: rotation | needs no code (docs only); the export path is 17 |
| V9 | 18 |
| V10 | 17 |
| Error-handling table | 2, 7, 11 |
| Testing: round trip, wrong passphrase, tampering, migration and crash, locked columns, plan wait, rekdf, integrity, argv/env, systemd-creds | 1, 2, 3, 7, 10, 12, 20 |
| Resolved Q4 and Q5 | Q4 needs no code; Q5 is 19 |

**Type consistency:** these names are used identically across tasks:

- `vault.Open`, `Create`, `(*Vault).Unlock`, `UnlockKEK`, `Lock`, `Get`, `Put(name, value, from)`, `PutAll([]Item)`, `Slots`, `AddKEKSlot`
- `signer.StoreVault`, `OpenVault`, `CreateVault`
- `secrets.VaultPath`, `Resolve`, `Migrate`, `EnsureTransportKey`, `TransportSigner`, `KeystoreKEK`, `AddKeystoreSlot`
- `config.SecretStoreVault`, `SecretRef`, `IsSecretRef`, `PlainSecrets`, `AddNotice`
- `server.Config.Vault`, `Unlocked`, `AfterUnlock`, and `(*Server).Unlock`, `UnlockKEK`, `Lock`, `VaultState`, `AttachRelay`
- `api.VaultView`, `UnlockRequest`, `CodeLocked`

`openControllerKey(cfg, v)` is introduced in Task 11 and used only there and in Task 16's auto-unlock.

**Proportion:** most of the length is test names and assertions, which carry the spec's values. Function bodies appear nowhere except the exact copy the spec or this plan fixes.
