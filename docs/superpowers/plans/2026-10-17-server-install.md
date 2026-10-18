# Server Install Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Jumpgate installs on a Linux server with one line or from apt/dnf, runs there as `jumpgate.service` with its controller key encrypted in the vault, is driven over SSH through a group-gated unix socket, and pairs agent-only boxes by push or by pull.

**Architecture:** `jumpgate serve --system` is the existing server with a second socket mode: `/run/jumpgate-ctl/server.sock` admits peers by SO_PEERCRED (the server's uid, root, or a `jumpgate-admin` member) instead of a token, and a token-free `system.json` lets `daemon.Find` locate it before any per-user server. Every privileged step (self-pair, push, pull) runs in the operator's own process and hands the server an agent address to verify (`PairRequest.Installed`), so the service never reads an operator's SSH key and never needs root. Packaging is goreleaser `nfpms` plus a signed apt/dnf repo synced to R2, and a POSIX `install.sh` that verifies checksums and provenance.

**Tech Stack:** Go 1.26.0 (module `github.com/valve-tech/jumpgate`, CGO off), `golang.org/x/sys/unix` v0.47.0, systemd (sysusers, `systemd-creds`), POSIX sh, goreleaser v2 `nfpms`, aptly, createrepo_c, rpmsign, GnuPG, GitHub Actions (release only), Cloudflare R2, Docker e2e harness (`scripts/e2e/Dockerfile`).

**Spec:** `docs/superpowers/specs/2026-10-17-server-install-design.md` (approved; open questions resolved 2026-10-17 except Q6). Sibling: `docs/superpowers/specs/2026-10-17-encrypted-secret-store-design.md`.

**Runs after:** `docs/superpowers/plans/2026-10-17-encrypted-secret-store.md`. Task 0 pins the vault API this plan consumes before anything else is built.

## Global Constraints

- Go 1.26.0, module path `github.com/valve-tech/jumpgate`, `CGO_ENABLED=0`. No new Go modules: `x/sys` v0.47.0 and `x/crypto` v0.56.0 are already dependencies.
- Product name is jumpgate. New commands: `jumpgate setup server`, `jumpgate setup install-service`, `jumpgate agent setup`, `jumpgate hosts invite`, `jumpgate hosts upgrade`, `jumpgate hosts add … --installed ADDR`. `jumpgate service HOST …` is taken; never reuse `service` for the controller unit.
- Names (D6): service user and group `jumpgate-ctl`; operator group `jumpgate-admin`; bootstrap's tunnel user and group `jumpgate` stay as they are. The controller never runs as, or gates on, `jumpgate`.
- Paths: unit `jumpgate.service` (package: `/usr/lib/systemd/system/jumpgate.service`; install.sh: `/etc/systemd/system/jumpgate.service`); sysusers `/usr/lib/sysusers.d/jumpgate.conf` (package) or `/etc/sysusers.d/jumpgate.conf` (install.sh); state `/var/lib/jumpgate-ctl` (`HOME`, so `config.Dir()` is `/var/lib/jumpgate-ctl/.jumpgate`); runtime `/run/jumpgate-ctl/` 0750 `jumpgate-ctl:jumpgate-admin`; socket `/run/jumpgate-ctl/server.sock` 0660 `jumpgate-ctl:jumpgate-admin`; discovery `/run/jumpgate-ctl/system.json` 0644 with **no token**; package binary `/usr/bin/jumpgate`; install.sh binary `/usr/local/bin/jumpgate`; agent binary `/usr/local/lib/jumpgate/jumpgate` (never owned by the package).
- The passphrase is never accepted on argv or in the environment; it travels only in a request body over the unix socket or the session-authenticated loopback listener, and is never logged.
- `serve --system` is Linux only. Per-user mode is unchanged on every OS.
- Exact copy, D7: `a jumpgate service runs on this machine; ask an administrator to run ` + "`sudo usermod -aG jumpgate-admin $USER`" + `, then log in again`.
- Exact copy, D10 (`api.CustodyWarning`, Task 7): `root on this controller server, with the key unlocked, can sign routine intents for every box that trusts it, including agent.upgrade, which replaces a box's agent. Encryption protects the key at rest, in backups and against disk theft; it does not protect it from live root.`
- install.sh: POSIX sh; the whole body is `main() { … }` and the last line is `main "$@"`; downloads use `curl -fsSL --proto '=https' --tlsv1.2` (or `wget --https-only --secure-protocol=TLSv1_2`); the only test knob is `JUMPGATE_INSTALL_BASE_URL`; release base `https://github.com/jumpgate-tech/jumpgate-app/releases`.
- Provenance: `gh attestation verify <file> --repo jumpgate-tech/jumpgate-app`.
- Repo hosting: `pkg.jumpgate.app`, R2 bucket `jumpgate-pkg` in Cloudflare account `4d950372162cef595726db63c21035ab` (valaxyblackdev). Signing secrets `PKG_SIGNING_KEY`, `PKG_SIGNING_PASSPHRASE` on GitHub environment `packages`; public key at `packaging/repo/jumpgate-archive-keyring.asc`; apt `signed-by=/usr/share/keyrings/jumpgate-archive-keyring.gpg`. Creating the bucket, domain, token, environment and key needs the user's OK (Task 22 gate).
- Git: commit on the working branch only; dev and CI live on git.craftsmanfounder.com; never push to GitHub without the user's OK; never use gitlab.com.
- Every new API error code goes into `internal/api/errors.go`'s `registry` with a hint (`TestNoLiteralErrorCodes` enforces no literal codes); every new rejection reason goes into `rejectionHints`. New codes: `key_path_refused`, `service_no_login`, `installed_mismatch`, `controller_exists`, `self_only`; new reason: `scope_local`.
- Linux-only tests run with `scripts/test-linux.sh <pkgs> -run <regex>` (uid 1000 in a container); `AS_ROOT=1` only where a test says so.
- Open question Q6 stays open; this plan publishes one multi-arch rpm repo at `/rpm/` (cheap to split later).

## Review Focus

1. **An operator just added to `jumpgate-admin` who has not logged in again.** The kernel refuses `connect()` (EACCES) because the session lacks the group. Expect the D7 hint, which ends "then log in again", and no per-user server. Test: Task 3 `TestFindSystemNotAdminGivesTheUsermodHint`.
2. **A login code minted by alice presented through bob's tunnel.** Expect 403 without the code being consumed, so alice's browser still signs in; root may redeem. Test: Task 4 `TestLoginCodeRedeemsOnlyForItsUID`.
3. **A passphrase piped to `--stdin` ending in `\n` or `\r\n` (`echo pw |`, a Windows-edited file).** Expect exactly one trailing line ending stripped, so unlock matches what was typed. Test: Task 9 `TestSetupServerReadsOneLineFromStdin`.
4. **An `agent.upgrade` that is cut short, has the wrong hash, or arrives while a job runs.** Expect the old binary intact, no `.new` file left behind, no restart, and a signed failure or rejection. Tests: Task 17 `TestUpgradeRefusesAHashMismatch`, `TestUpgradeRefusesAShortBody`, `TestUpgradeRefusesWhileAJobIsActive`.
5. **A `jumpgate-admin` member steering the service into reading a file by path** (an SSH `keyPath`, a jump-host key, a `--ref` for the file store). The service can read files the member cannot (its own state). Expect every client-chosen path to be refused in system mode. Tests: Task 5 `TestSystemModeRefusesAClientKeyPath`, `TestSystemModeProbeRefusesAJumpKey`; Task 9 `TestSystemInitKeyRefusesAClientChosenFileRef`.

## Interfaces: Consumes (from the encrypted-secret-store plan)

These are the names the vault plan (`2026-10-17-encrypted-secret-store.md`) actually produces: Task 3's Interfaces block, Task 5 (`signer`, `secrets.VaultPath`, `readPassphrase`), Task 7 (server methods, routes, codes), Task 11 (`credentialKEK`, auto-unlock), Task 12 (client) and Task 16 (`vault seal`). Task 0 compiles a contract test against them. Earlier drafts of this plan assumed other names; the mapping is below.

**Choice for what the vault plan does not produce:** a systemd-creds slot whose secret the vault generates (`AddCredSlot` returning a secret) does not exist. The vault plan's primitive is `AddKEKSlot(Slot, kek)`: the caller draws the 32-byte KEK. So this plan's Task 11 draws the KEK inside `POST /api/system/seal` and calls `AddKEKSlot`. No task is added to the vault plan. The vault plan has no `Init` that returns an unlocked vault under that name (it is `Create`), no package-level `Path` (it is `secrets.VaultPath()`), and no `LockState()` (it is `VaultState().State`).

| Assumed in earlier drafts | Real name (vault plan) |
|---|---|
| `vault.Path()` | `secrets.VaultPath() (string, error)` (Task 5); `(*vault.Vault).Path()` also exists |
| `vault.Init(path, pass)` | `vault.Create(path, pass, vault.DefaultParams) (*Vault, error)`, returned unlocked, clears `pass` |
| `(*Vault).Locked()` | same (`Locked() bool`) |
| `(*Vault).AddCredSlot(cred) (secret, err)` | `(*Vault).AddKEKSlot(vault.Slot{Kind: vault.SlotSystemdCreds, Cred: cred}, kek []byte) error`; the caller draws `kek` |
| `(*Vault).UnlockWithCred(cred, secret)` | `(*Vault).UnlockKEK(vault.SlotSystemdCreds, kek) error` |
| `vault.ErrWrongPassphrase`, `ErrCorrupt`, `ErrLocked` | same |
| `signer.CreateInVault(v, name)` | `signer.CreateVault(v, ref) (*Key, error)`; `signer.OpenVault(v, ref)` opens it |
| `signer.StoreVault` | same |
| `server.Config.Vault` | same (`Vault *vault.Vault`; nil means look lazily) |
| `(*Server).LockState()` string | `(*Server).VaultState() api.VaultView`; compare `.State`: `none`, `locked`, `unlocked` or `unusable` |
| `(*Server).Lock()` | `(*Server).Lock() api.VaultView` (also `Unlock(pass) (api.VaultView, error)`, `UnlockKEK(kind, kek)`) |
| `api.CodeLocked`, `CodeWrongPassphrase`, `CodeVaultCorrupt` | same (also `CodeVaultUnsafe`, `CodeNoVault`) |
| `apiclient.Unlock(ctx, pass) error`, `POST /api/unlock` | `(*Client).Unlock(ctx, pass []byte) (api.VaultView, error)`, `POST /api/vault/unlock` |
| `apiclient.Lock(ctx) error`, `POST /api/lock` | `(*Client).Lock(ctx) (api.VaultView, error)`, `POST /api/vault/lock`; also `(*Client).Vault(ctx)` and `GET /api/vault` |
| `readPassphrase(in, fromStdin, prompt)` | `readPassphrase(prompt string, fromStdin, confirm bool) ([]byte, error)`; the `--stdin` source is the seam `stdinReader` |
| unlock at start from `$CREDENTIALS_DIRECTORY` (`unlockFromCredentials`) | `credentialKEK(getenv)` and the `vault-auto-unlock` task in `buildServer` (vault Task 11) |
| `vault seal --tpm2 [--allow-host-key]` | `vault seal (--tpm2 \| --allow-host-key) [--out PATH]` (vault Task 16): one mode flag, not both |

```go
// package github.com/valve-tech/jumpgate/internal/vault
func Create(path string, passphrase []byte, p Params) (*Vault, error)
func Open(path string) (*Vault, error)
func (v *Vault) Path() string
func (v *Vault) Locked() bool
func (v *Vault) Unlock(passphrase []byte) error
func (v *Vault) UnlockKEK(kind SlotKind, kek []byte) error
func (v *Vault) Lock()
func (v *Vault) AddKEKSlot(s Slot, kek []byte) error
var DefaultParams Params
var ErrWrongPassphrase, ErrCorrupt, ErrLocked error
const SlotSystemdCreds SlotKind = "systemd-creds"

// package signer
const StoreVault Store = "vault"
func CreateVault(v *vault.Vault, ref string) (*Key, error) // ref "controller-key"
func OpenVault(v *vault.Vault, ref string) (*Key, error)

// package secrets
func VaultPath() (string, error)

// package server
type Config struct{ /* … */ Vault *vault.Vault }
func (s *Server) VaultState() api.VaultView // .State: "none" | "locked" | "unlocked" | "unusable"
func (s *Server) Unlock(passphrase []byte) (api.VaultView, error)
func (s *Server) UnlockKEK(kind vault.SlotKind, kek []byte) (api.VaultView, error)
func (s *Server) Lock() api.VaultView // drops the DEK and the signer

// package api
const CodeLocked Code = "locked"; CodeWrongPassphrase Code = "wrong_passphrase"; CodeVaultCorrupt Code = "vault_corrupt"

// package apiclient
func (c *Client) Unlock(ctx context.Context, passphrase []byte) (api.VaultView, error) // POST /api/vault/unlock
func (c *Client) Lock(ctx context.Context) (api.VaultView, error)                      // POST /api/vault/lock

// package main (cmd/jumpgate)
func readPassphrase(prompt string, fromStdin, confirm bool) ([]byte, error) // /dev/tty hidden, or one stdin line
func credentialKEK(getenv func(string) string) ([]byte, bool)               // $CREDENTIALS_DIRECTORY/jumpgate-vault-key
// CLI: `jumpgate unlock [--stdin]`, `jumpgate lock`, `jumpgate vault seal`, `jumpgate keys init --store vault`
// TUI: the unlock prompt and the LOCKED header badge; `jumpgate home` LOCKED line
```

The vault plan's unlock route opens the controller key named in `config.json` once the vault is unlocked (`server.Config.Unlocked`, wired in vault Task 11). This plan relies on that in Task 9: it creates the key, locks, then unlocks through the normal path.

---

### Task 0: Pin the vault API

**Files:**
- Create: `cmd/jumpgate/vaultcontract_test.go`

**Interfaces:**
- Consumes: every name in "Interfaces: Consumes" above.
- Produces: nothing at runtime. If anything is renamed, the rename is applied to this plan document.

- [ ] **Step 1: Write the contract test**

```go
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
		vault.SlotSystemdCreds,
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
```

- [ ] **Step 2: Run it**

Run: `go test ./cmd/jumpgate/ -run TestVaultContract -count=1`
Expected: PASS. A compile error such as `undefined: vault.AddKEKSlot` means the vault plan has drifted from the table above. Find its name (`grep -rn "func (v \*Vault)" internal/vault/`), replace the name everywhere in this plan document and in the test, and re-run until it passes. A missing capability is a blocker: stop and report it, and do not invent it here. This plan draws the systemd-creds KEK itself (Task 11), so a credential-slot primitive that returns a secret is not needed.

- [ ] **Step 3: Commit**

```bash
git add cmd/jumpgate/vaultcontract_test.go docs/superpowers/plans/2026-10-17-server-install.md
git commit -m "test: pin the vault API the server-install plan consumes"
```

---

### Task 1: `internal/peercred`, shared by the agent and the server

**Files:**
- Create: `internal/peercred/peercred.go`, `internal/peercred/peercred_linux.go`, `internal/peercred/peercred_darwin.go`, `internal/peercred/peercred_other.go`, `internal/peercred/peercred_test.go`
- Delete: `internal/agent/peercred_linux.go`, `internal/agent/peercred_darwin.go`, `internal/agent/peercred_other.go`
- Modify: `internal/agent/listen.go` (`JumpgateGID`, `peerGate`)

**Interfaces:**
- Produces:
  ```go
  type Cred struct{ UID int; GIDs []int } // GIDs: primary + /etc/group supplementary (Linux), xucred groups (darwin)
  func Of(c net.Conn) (Cred, error)        // error "not a unix socket" for any other conn; unsupported OS errors
  func (c Cred) InGroup(gid int) bool
  func GroupID(name string) (int, bool)    // os/user.LookupGroup; false when absent
  ```

- [ ] **Step 1: Write the failing tests** in `internal/peercred/peercred_test.go`

```go
func TestOfReportsThisProcess(t *testing.T) // listen unix in testutil.ShortTempDir, dial, Accept; c, err := Of(accepted); err == nil; c.UID == os.Getuid(); c.InGroup(os.Getgid())
func TestOfRefusesANonUnixConn(t *testing.T) // a, _ := net.Pipe(); _, err := Of(a); err != nil && strings.Contains(err.Error(), "not a unix socket")
func TestInGroup(t *testing.T)               // Cred{GIDs: []int{5, 7}}.InGroup(7) == true; InGroup(8) == false
func TestGroupIDOfAMissingGroup(t *testing.T) // _, ok := GroupID("jumpgate-no-such-group"); !ok
```

The first test is skipped on Windows (`runtime.GOOS == "windows"`).

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/peercred/ -count=1`
Expected: FAIL, `undefined: Of`.

- [ ] **Step 3: Move the code.** Move `peerCred` and `groupsOf` from `internal/agent/peercred_*.go` into `peercred.Of` with the same build tags and the same bodies, returning `Cred`. In `listen.go`, `JumpgateGID` becomes `gid, ok := peercred.GroupID("jumpgate"); if !ok { return -1 }`, and `peerGate` calls `peercred.Of(c)`.

- [ ] **Step 4: Run them to verify they pass, here and on Linux**

Run: `go test ./internal/peercred/ ./internal/agent/ -count=1 && scripts/test-linux.sh ./internal/peercred/ ./internal/agent/`
Expected: PASS (`ok` for both packages, both times).

- [ ] **Step 5: Commit**

```bash
git add internal/peercred internal/agent
git commit -m "refactor: move peer credentials into internal/peercred"
```

---

### Task 2: `jumpgate serve --system`: the admin socket and `system.json`

**Files:**
- Create: `internal/server/peerauth.go`, `internal/server/peerauth_test.go`, `internal/daemon/system.go`, `internal/daemon/system_test.go`, `cmd/jumpgate/serve_system.go`, `cmd/jumpgate/serve_system_linux.go`, `cmd/jumpgate/serve_system_other.go`, `cmd/jumpgate/serve_system_test.go`
- Modify: `internal/server/server.go` (`Config`, `authMiddleware`), `cmd/jumpgate/buildserver.go` (`serverOptions.System`, `newServeFlagSet`), `cmd/jumpgate/cli_serve.go` (`cmdServe` branch)

**Interfaces:**
- Consumes: `peercred.Of`, `peercred.GroupID` (Task 1); `(*server.Server).VaultState()` (vault Task 7).
- Produces:
  ```go
  // internal/daemon
  var SystemDir = "/run/jumpgate-ctl" // tests override
  type SystemInfo struct {
  	PID int `json:"pid"`; Socket string `json:"socket"`; HTTPAddr string `json:"httpAddr"`
  	Version string `json:"version"`; StartedAt time.Time `json:"startedAt"`
  	UID int `json:"uid"`; Lock string `json:"lock"` // no token field, ever
  }
  func PublishSystem(dir string, info SystemInfo) error // temp file created 0644 in dir, fsync, rename to system.json
  func RemoveSystem(dir string)
  // internal/server
  type Config struct{ /* … */ System bool; AdminGID int } // AdminGID: jumpgate-admin's gid
  func (s *Server) ServeUnixSystem(ctx context.Context, path string) error
  func admitPeer(c peercred.Cred, self, adminGID int) bool // uid 0, uid self, or InGroup(adminGID)
  func peerFrom(ctx context.Context) (peercred.Cred, bool)
  // cmd/jumpgate
  type serverOptions struct{ /* … */ System bool } // flag --system on `serve` only
  func systemStartWarnings(cfg config.Config) []string
  ```

- [ ] **Step 1: Write the failing tests**

`internal/server/peerauth_test.go`:
```go
func TestAdmitPeer(t *testing.T) // table: {Cred{UID:0}, true}, {Cred{UID:self}, true}, {Cred{UID:4000, GIDs:[]int{admin}}, true}, {Cred{UID:4000, GIDs:[]int{9}}, false}
func TestSystemSocketAdmitsByPeerNotToken(t *testing.T) // Config{System: true, AdminGID: os.Getgid(), Token: "tok"}; ServeUnixSystem on ShortTempDir/s.sock; GET /api/health with NO Authorization header -> 200
func TestSystemSocketModeAndGroup(t *testing.T) // after listen: mode&0o777 == 0o660; stat gid == AdminGID
func TestTCPListenerStillNeedsTheToken(t *testing.T) // httptest.NewServer(s.Handler()) with System true; GET /api/health without token -> 401
```

`internal/daemon/system_test.go`:
```go
func TestPublishSystemIsWorldReadableAndTokenFree(t *testing.T) // PublishSystem(dir, SystemInfo{PID: 1, UID: 999, Lock: "locked"}); mode 0o644; decoded map has no "token" key; "lock" == "locked"
```

`cmd/jumpgate/serve_system_test.go`:
```go
func TestServeSystemIsRefusedOffLinux(t *testing.T) // hostGOOS = "darwin"; cmdServe([]string{"--system"}) == exitCode("failed"); stderr contains "Linux"
func TestServeSystemWarnsAboutAPlainKey(t *testing.T) // systemStartWarnings(config.Config{Controller: &config.Controller{KeyStore: "file"}}) has one entry containing "stored unencrypted"
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/server/ ./internal/daemon/ ./cmd/jumpgate/ -run 'AdmitPeer|SystemSocket|TCPListenerStill|PublishSystem|ServeSystem' -count=1`
Expected: FAIL, `undefined: admitPeer` (and the same for the other new names).

- [ ] **Step 3: Implement**
  - `ServeUnixSystem` follows `ServeUnix` (same stale-socket rule) but `chmod 0660`, `chown(-1, AdminGID)`, and wraps the listener in a gated listener (the agent's `gatedListener` pattern) that closes a peer for whom `admitPeer` is false. `ConnContext` stores the `peercred.Cred` under an unexported key.
  - In `authMiddleware`, right after the `/login` branch: `if s.cfg.System { if c, ok := peerFrom(r.Context()); ok && admitPeer(c, s.selfUID, s.cfg.AdminGID) { next.ServeHTTP(w, r); return } }`. TCP requests carry no cred, so they fall through to the token and cookie checks unchanged.
  - `cmdServe` with `--system` (in `serve_system.go`, run only when `opts.System`):
    1. Off Linux → `failed("`serve --system` runs under systemd on Linux")`.
    2. `daemon.Acquire()` as today, for the single-instance lock.
    3. Resolve `jumpgate-admin` with `peercred.GroupID`; when absent → fail with "group jumpgate-admin is missing; run `sudo systemd-sysusers`".
    4. `chown(-1, gid)` and `chmod 0750` on `daemon.SystemDir`.
    5. `unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)` (`serve_system_linux.go`).
    6. `buildServer` with `Config.System=true`, `AdminGID=gid`, `LoginDir=""`.
    7. Log each `systemStartWarnings` line.
    8. Serve TCP plus `ServeUnixSystem(SystemDir/server.sock)` through `serveBoth`.
    9. Publish with `daemon.PublishSystem`, never `holder.Publish`, so there is no `server.json` in system mode. Republish when `VaultState().State` changes: a 2 s ticker compares the last published value.
    10. `RemoveSystem` on exit.

- [ ] **Step 4: Run them to verify they pass, here and on Linux**

Run: `go test ./internal/server/ ./internal/daemon/ ./cmd/jumpgate/ -count=1 && scripts/test-linux.sh ./internal/server/ ./internal/daemon/ ./cmd/jumpgate/ -run 'AdmitPeer|SystemSocket|TCPListenerStill|PublishSystem|ServeSystem'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/server internal/daemon cmd/jumpgate
git commit -m "feat(server): serve --system with a peer-credential admin socket"
```

---

### Task 3: Discovery: the service first, never a second controller

**Files:**
- Modify: `internal/daemon/system.go`, `internal/daemon/daemon.go` (`Info`, `Find`, `EnsureRunning`, `Stop`), `internal/apiclient/client.go` (`sendVia` bearer header)
- Test: `internal/daemon/system_test.go`, `internal/apiclient/client_test.go`

**Interfaces:**
- Consumes: `daemon.SystemInfo`, `SystemDir` (Task 2).
- Produces:
  ```go
  type Info struct{ /* existing… */ System bool `json:"-"`; ServiceUID int `json:"-"`; Lock string `json:"-"` }
  var ErrNotAdmin = errors.New("a jumpgate service runs on this machine; ask an administrator to run `sudo usermod -aG jumpgate-admin $USER`, then log in again")
  var ErrServiceManaged = errors.New("daemon: the jumpgate service is managed by systemd; use `sudo systemctl stop jumpgate`")
  var SystemUnitPaths = []string{"/etc/systemd/system/jumpgate.service", "/usr/lib/systemd/system/jumpgate.service"}
  func FindSystem(ctx context.Context) (Info, bool, error)
  ```
- `FindSystem` works in this order:
  1. A missing `system.json` → `(Info{}, false, nil)`.
  2. A file not owned (Lstat uid) by root or by its own `uid` field is ignored.
  3. Dial the socket and `GET /api/health` without a token.
  4. EACCES → `ErrNotAdmin`.
  5. ENOENT or ECONNREFUSED → not running.
- `Find` returns `FindSystem`'s answer when it is ok or an error, else the per-user lookup.
- `EnsureRunning` never starts `serve` when `FindSystem` errs, or when any `SystemUnitPaths` file exists. It returns "the jumpgate service is installed but not running; run `sudo systemctl start jumpgate`, then see `journalctl -u jumpgate`".
- `Stop` on a system `Info` → `ErrServiceManaged`.

- [ ] **Step 1: Write the failing tests**

```go
func TestFindPrefersTheSystemServer(t *testing.T) // SystemDir = temp; a live per-user server (as TestFindSeesALiveServer) AND a system socket answering /api/health with no token; info.System == true, info.Socket == system socket
func TestFindSystemNotAdminGivesTheUsermodHint(t *testing.T) // skip if os.Geteuid() == 0; system socket chmod 0o000; _, _, err := Find(ctx); errors.Is(err, ErrNotAdmin); err.Error() contains "usermod -aG jumpgate-admin" and "log in again"
func TestFindSystemIgnoresAFileOwnedBySomeoneElse(t *testing.T) // skip if root; system.json written by us with "uid": os.Getuid()+1; ok == false, err == nil
func TestEnsureRunningNeverStartsAUserServerBesideTheService(t *testing.T) // SystemUnitPaths = []string{tempfile}; startServer seam fails the test if called; err contains "systemctl start jumpgate"
func TestEnsureRunningNeverStartsAUserServerForANonAdmin(t *testing.T) // chmod-0 system socket; startServer not called; errors.Is(err, ErrNotAdmin); no server.json under HOME
func TestStopRefusesTheService(t *testing.T) // live system socket; errors.Is(Stop(ctx), ErrServiceManaged)
```
`internal/apiclient/client_test.go`:
```go
func TestNoBearerHeaderWithoutAToken(t *testing.T) // New(daemon.Info{Socket: s}) against a socket handler recording r.Header; Authorization == ""
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/daemon/ ./internal/apiclient/ -run 'FindPrefers|FindSystem|NeverStarts|StopRefuses|NoBearer' -count=1`
Expected: FAIL, `undefined: FindSystem`.

- [ ] **Step 3: Implement** the Produces block. Set the `Authorization` header in `sendVia` only when the token is non-empty. `rediscover` already goes through `daemon.Find`, so it needs no change.

- [ ] **Step 4: Run them to verify they pass, here and on Linux**

Run: `go test ./internal/daemon/ ./internal/apiclient/ -count=1 && scripts/test-linux.sh ./internal/daemon/ ./internal/apiclient/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/daemon internal/apiclient
git commit -m "feat(daemon): find the jumpgate service first and never start a second controller"
```

---

### Task 4: Login links bound to the requesting uid (D8)

**Files:**
- Modify: `internal/server/login.go` (`loginCode.uid`, `addLoginCode`, `redeem`, `handleLogin`), `internal/server/server.go` (`POST /api/login-code` handler), `cmd/jumpgate/opener.go` (`requestLoginLink` becomes a package var seam; system mode prints)
- Test: `internal/server/login_test.go`, `cmd/jumpgate/opener_test.go`

**Interfaces:**
- Consumes: `peerFrom` (Task 2), `daemon.Info.System` (Task 3).
- Produces:
  ```go
  func (s *Server) addLoginCode(file string, uid int) string // uid -1 means "the server's own uid"
  func (s *Server) NewLoginLinkFor(uid int) (LoginLink, error)
  func (s *Server) redeemFor(code string, uid int) (ok, reused, foreign bool) // foreign: code exists but belongs to another uid; NOT consumed
  ```
- Over the system socket, `POST /api/login-code` mints for `peerFrom(ctx).UID`. Everywhere else it mints for -1.
- `handleLogin`'s `peerFound` branch becomes `redeemFor(code, uid)`, with root (uid 0) admitted for any code. `foreign` answers 403 with the existing text "this jumpgate login link belongs to another user".
- `jumpgate open` against `Info.System` always prints. It adds `on your own machine, forward the port first: ssh -L <port>:127.0.0.1:<port> <this server>`, with the port taken from `HTTPAddr`.

- [ ] **Step 1: Write the failing tests**

```go
func TestLoginCodeRedeemsOnlyForItsUID(t *testing.T) // code := s.addLoginCode("", 2001); peerUID seam -> (2002, peerFound): 403 "belongs to another user"; then seam -> (2001, peerFound): 200 with the session cookie (the code survived)
func TestRootRedeemsAnyUsersCode(t *testing.T)       // code for 2001; seam -> (0, peerFound): 200
func TestPerUserCodesStillBelongToTheServer(t *testing.T) // NewLoginCode(); seam -> (s.selfUID, peerFound): 200; another uid: 403
func TestLoginCodeMintRecordsTheSocketPeer(t *testing.T) // System server via ServeUnixSystem; POST /api/login-code over the socket; s.codes.list[0].uid == os.Getuid()
```
`cmd/jumpgate/opener_test.go`:
```go
func TestOpenPrintsAndExplainsTheTunnelInSystemMode(t *testing.T) // ensureRunning seam -> daemon.Info{System: true, HTTPAddr: "127.0.0.1:8799"}; requestLoginLink seam -> LoginLink{Code: "c"}; output contains "ssh -L 8799:127.0.0.1:8799" and "/login?code=c"; no opener invoked
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/server/ ./cmd/jumpgate/ -run 'LoginCodeRedeemsOnly|RootRedeems|PerUserCodesStill|MintRecords|OpenPrintsAndExplains' -count=1`
Expected: FAIL (`addLoginCode` takes one argument; `redeemFor` is undefined).

- [ ] **Step 3: Implement** the Produces block. `NewLoginCode` and `NewLoginLink` keep their signatures and call through with uid -1.

- [ ] **Step 4: Run them to verify they pass, here and on Linux**

Run: `go test ./internal/server/ ./cmd/jumpgate/ -count=1 && scripts/test-linux.sh ./internal/server/ -run 'Login'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/server cmd/jumpgate
git commit -m "feat(server): bind system-mode login links to the requesting uid"
```

---

### Task 5: The service never reads operator keys; `GET /api/pairing-info`

D5 asks whether any executor path reads `KeyPath` server-side. Three do: `handleAddTarget` dials with it, the host-key probe authenticates jump hops with it, and `handlePair` logs in with it. Under `ProtectHome=yes` they cannot work, and for a `jumpgate-admin` member they would be a confused deputy (Review Focus 5). In system mode, the service reaches boxes only through agent tunnels with its own transport key.

**Files:**
- Create: `internal/server/system.go`, `internal/server/system_test.go`
- Modify: `internal/server/api.go` (`handleAddTarget`; register routes), `internal/server/hostkeys.go` (probe), `internal/server/server.go` (`defaultNewExecutor` wrapper in `New`), `internal/api/wire.go` (`PairingInfo`), `internal/api/errors.go` (codes), `internal/apiclient/methods.go`

**Interfaces:**
- Produces:
  ```go
  // internal/api
  type PairingInfo struct {
  	Controller   string              `json:"controller"`   // config.Controller.Address
  	TransportKey string              `json:"transportKey"` // authorized_keys line, public
  	System       bool                `json:"system"`
  	ServiceUID   int                 `json:"serviceUid"`   // os.Getuid() of the server in system mode, else -1
  	Scope        string              `json:"scope,omitempty"` // "self" under --self-only (Task 19)
  	Wire         *catalog.WireConfig `json:"wire,omitempty"`  // the named target's, when ?target= is given
  }
  const CodeKeyPathRefused Code = "key_path_refused"  // hint: "the jumpgate service never reads your SSH keys; leave out --key: `jumpgate hosts add` runs the root steps in your own shell"
  const CodeServiceNoLogin Code = "service_no_login"  // hint: "the jumpgate service reaches boxes only through their agents; pair the box, or use `jumpgate ssh NAME` for a shell"
  // internal/apiclient
  func (c *Client) PairingInfo(ctx context.Context, target string) (api.PairingInfo, error) // GET /api/pairing-info?target=
  ```
- `GET /api/pairing-info` works in both modes. With no controller key it answers the existing 503 `no_controller_key`. It gets the transport key from `ensureTransportKey()`, or from whatever the vault plan replaced it with; when that needs an unlock, the answer is the vault plan's 503 `locked`.
- In system mode:
  - `POST /api/targets` with any hop `KeyPath != ""` → 400 `key_path_refused`. A valid SSH target is saved without the reachability dial: the CLI confirms host keys first.
  - `POST /api/hostkeys/probe` with a jump hop → 409 `ssh_jump_unsupported`.
  - `newExecutor` for an `ssh` target returns an error that maps to `service_no_login`. Local targets are unaffected.

- [ ] **Step 1: Write the failing tests** in `internal/server/system_test.go`

```go
func TestSystemModeRefusesAClientKeyPath(t *testing.T) // System server; POST /api/targets {"id":"b","mode":"ssh","ssh":{"Host":"h","User":"root","KeyPath":"/var/lib/jumpgate-ctl/.jumpgate/ssh/jumpgate_ed25519"}} -> 400, code key_path_refused
func TestSystemModeAddTargetDoesNotDial(t *testing.T) // NewExecutor seam calls t.Fatal; same body without KeyPath -> 2xx, target saved
func TestSystemModeProbeRefusesAJumpKey(t *testing.T) // probe with ssh.jump set -> 409 ssh_jump_unsupported; probes seam never called
func TestSystemModeSSHExecutorIsRefused(t *testing.T) // GET /api/targets/{id}/du on an ssh target -> code service_no_login
func TestPairingInfoReportsTheServiceUID(t *testing.T) // System server with a controller: ServiceUID == os.Getuid(), System == true, TransportKey starts "ssh-ed25519 ", Controller == recorded address
func TestPairingInfoOutsideSystemMode(t *testing.T) // per-user: System == false, ServiceUID == -1
func TestPairingInfoWithoutAKey(t *testing.T) // no controller -> 503 no_controller_key
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/server/ -run 'SystemMode|PairingInfo' -count=1`
Expected: FAIL (404 for `/api/pairing-info`; `CodeKeyPathRefused` undefined).

- [ ] **Step 3: Implement** the Produces block, then add the two codes to `registry` with the hints above.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./internal/server/ ./internal/api/ ./internal/apiclient/ -count=1`
Expected: PASS, including `TestNoLiteralErrorCodes`.

- [ ] **Step 5: Commit**

```bash
git add internal/server internal/api internal/apiclient
git commit -m "feat(server): system mode never reads operator SSH keys; add pairing-info"
```

---

### Task 6: Self-pair as the service uid; `--installed` for SSH targets

**Files:**
- Modify: `cmd/jumpgate/pair_local.go` (`pairLocalForeground` takes facts), `cmd/jumpgate/cli_hosts.go` (`hostsAdd`: system branch, `--installed`), `internal/server/pair.go` (`Installed` for ssh targets), `internal/api/errors.go` (`installed_mismatch`)
- Test: `cmd/jumpgate/pair_local_test.go`, `internal/server/pair_test.go`

**Interfaces:**
- Consumes: `apiclient.(*Client).PairingInfo` (Task 5); `daemon.Info.System` (Task 3).
- Produces:
  ```go
  type pairingFacts struct {
  	TargetID string; Controller eip712.Address; LocalUID int; Wire *catalog.WireConfig; Scope string
  }
  func factsFromConfig(t config.Target) (pairingFacts, int)                                      // per-user: today's behaviour, LocalUID os.Getuid()
  func factsFromServer(ctx context.Context, c *apiclient.Client, id string) (pairingFacts, int) // system: PairingInfo, LocalUID ServiceUID
  func pairLocalForeground(ctx context.Context, out io.Writer, f pairingFacts) (string, int)   // as root: no sudo, plain newLocalExecutor()
  const CodeInstalledMismatch api.Code = "installed_mismatch" // hint: "the agent answering on this host is not the one `jumpgate agent setup` printed; check the address and the host"
  ```
- `hosts add NAME --local` against a system server always pairs in the foreground (as root too) with `factsFromServer`.
- `hosts add NAME --ssh jumpgate@HOST --installed 0xA…` saves and confirms the target as today, then sends `PairRequest{Installed: addr}` without running bootstrap. Combining `--installed` with `--local` is a usage error.
- `handlePair` accepts `Installed` for ssh targets and opens no login session. When verification gets a signed `wrong_agent` rejection or `ErrBadReceipt` for an `Installed` pairing, it sends `installed_mismatch`.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/jumpgate/pair_local_test.go
func TestSystemSelfPairEnrollsTheServiceUID(t *testing.T) // fake server: /api/pairing-info -> {system:true, serviceUid:991, controller:addr}; stubForeground captures Options; Options.LocalUID == 991; Options.Local == true
func TestSystemSelfPairAsRootSkipsSudo(t *testing.T)     // geteuid seam 0; sudoNonInteractive seam fails the test if called; Options.Exec is not a sudo wrapper
func TestHostsAddInstalledNeedsSSH(t *testing.T)         // cmdHosts([]string{"add","b","--local","--installed","0x…"}) == exitCode("usage")
// internal/server/pair_test.go
func TestPairInstalledOverSSHRunsNoLoginSession(t *testing.T) // startPairTestSSHD handler counts exec sessions; ssh target + {"installed": addr}; sessions == 0; stream has step "install" line "done in the foreground by the CLI"
func TestPairInstalledMismatchIsNamed(t *testing.T)           // agent answering with another key (as TestVerifyPairingRefusesAnotherAgentsKey); final event code == installed_mismatch
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/jumpgate/ ./internal/server/ -run 'SystemSelfPair|HostsAddInstalled|PairInstalled' -count=1`
Expected: FAIL (`pairLocalForeground` signature; 400 `"installed" applies only to pairing this machine`).

- [ ] **Step 3: Implement** the Produces block. Delete the `!local` refusal in `handlePair`'s `Installed` parse. Update the `hostsAdd` usage text to `… | --local | --ssh jumpgate@HOST --installed ADDR`.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./cmd/jumpgate/ ./internal/server/ ./internal/api/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate internal/server internal/api
git commit -m "feat(pair): self-pair as the service uid; accept installed agents over ssh"
```

---

### Task 7: Pushing from a system controller runs in the CLI

**Files:**
- Create: `cmd/jumpgate/pair_remote.go`, `cmd/jumpgate/pair_remote_test.go`, `internal/api/custody.go`
- Modify: `cmd/jumpgate/cli_hosts.go` (system branch of the `--ssh` path)

**Interfaces:**
- Consumes: `PairingInfo` (Task 5); `PairRequest.Installed` for ssh (Task 6); `apiclient.(*Client).RecordedHostKeys`; `bootstrap.Run`; `agentbin.Reporting`.
- Produces:
  ```go
  const api.CustodyWarning = "root on this controller server, with the key unlocked, can sign routine intents for every box that trusts it, including agent.upgrade, which replaces a box's agent. Encryption protects the key at rest, in backups and against disk theft; it does not protect it from live root."
  var newSSHExecutor = executor.NewSSHContext // seam
  func pairRemoteForeground(ctx context.Context, out io.Writer, c *apiclient.Client, name string, login executor.SSHConfig, sudo bool) (string, int)
  ```
- Against a system server, `hosts add NAME --ssh USER@HOST [--key K] [--jump …]` works like this:
  1. Host keys are confirmed through the server as today.
  2. `AddTarget` is sent with `KeyPath` cleared on every hop. `--jump` is refused with `ssh_jump_unsupported` before anything is sent.
  3. `login.HostKey` is a callback that accepts only the keys `RecordedHostKeys(host:port)` lists as confirmed.
  4. Bootstrap runs in this process over `newSSHExecutor` (sudo-wrapped when `sudo` or the user is not root) with `TransportKey` and `Controller` from `PairingInfo`.
  5. `streamPair(…Installed: addr…)`.
- `api.CustodyWarning` is printed once, before step 4.

- [ ] **Step 1: Write the failing tests** in `pair_remote_test.go` (fake API through the `connect` seam, as in `cli_client_test.go`)

```go
func TestSystemPushRunsBootstrapInTheCLI(t *testing.T) // runBootstrap stub captures Options: Local == false, TransportKey == served key, Controller == served address; the pair POST body has "installed" == stub address
func TestSystemPushNeverSendsTheKeyPath(t *testing.T)  // --key /home/alice/.ssh/id; recorded POST /api/targets body: every hop's keyPath == ""; newSSHExecutor got KeyPath "/home/alice/.ssh/id"
func TestSystemPushTrustsOnlyRecordedHostKeys(t *testing.T) // RecordedHostKeys -> none confirmed: fails before newSSHExecutor, message contains "confirm"
func TestSystemPushPrintsTheCustodyWarning(t *testing.T)    // stdout contains api.CustodyWarning exactly once
func TestSystemPushRefusesAJumpHost(t *testing.T)           // --jump x@y -> failure naming ssh_jump_unsupported; no POST /api/targets
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/jumpgate/ -run SystemPush -count=1`
Expected: FAIL, `undefined: pairRemoteForeground`.

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./cmd/jumpgate/ ./internal/api/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate internal/api
git commit -m "feat(hosts): push from a system controller runs bootstrap in the operator's shell"
```

---

### Task 8: `jumpgate.service`, sysusers, `jumpgate setup install-service`

**Files:**
- Create: `internal/sysunit/sysunit.go`, `internal/sysunit/sysunit_test.go`, `packaging/linux/server/jumpgate.service`, `packaging/linux/server/jumpgate.sysusers`, `cmd/jumpgate/cli_setup.go`, `cmd/jumpgate/cli_setup_test.go`
- Modify: `cmd/jumpgate/cli.go` (`subcommands`: add `"setup": cmdSetup`), `cmd/jumpgate/help.go` (list `setup`)

**Interfaces:**
- Produces:
  ```go
  // internal/sysunit
  const PackageUnitPath = "/usr/lib/systemd/system/jumpgate.service"; AdminUnitPath = "/etc/systemd/system/jumpgate.service"
  const PackageSysusersPath = "/usr/lib/sysusers.d/jumpgate.conf"; AdminSysusersPath = "/etc/sysusers.d/jumpgate.conf"
  const SealDropInPath = "/etc/systemd/system/jumpgate.service.d/50-vault-seal.conf"; SealCredPath = "/etc/jumpgate-ctl/vault-key.cred"
  const Sysusers = "g jumpgate-admin -\nu jumpgate-ctl - \"jumpgate controller\" /var/lib/jumpgate-ctl /usr/sbin/nologin\n"
  func Unit(bin string) string
  // cmd/jumpgate
  func cmdSetup(args []string) int // install-service | server (Task 9)
  var runPrivileged = func() executor.Executor { return executor.NewLocal() } // seam
  ```
- `Unit(bin)` is D5's `[Service]` block with `ExecStart=<bin> serve --no-open --system`. It also has:
  - `[Unit] Description=jumpgate controller`, `After=network-online.target`, `Wants=network-online.target`;
  - `SupplementaryGroups=jumpgate-admin`, needed so the non-root service may `chgrp` its socket and runtime directory to that group (a non-member cannot);
  - `RuntimeDirectoryMode=0750` and `StateDirectoryMode=0700` on their own lines, because systemd does not allow a comment after a value;
  - `[Install] WantedBy=multi-user.target`.
- `jumpgate setup install-service [--bin PATH]`:
  - It refuses off Linux and as non-root.
  - It refuses when `PackageUnitPath` exists ("the jumpgate package owns the service; use apt or dnf").
  - It writes `AdminUnitPath` (0644) with `Unit(bin)`, where `bin` defaults to the resolved `os.Executable()`, and `AdminSysusersPath` (0644) with `Sysusers`.
  - It then runs `systemd-sysusers /etc/sysusers.d/jumpgate.conf`, then `systemctl daemon-reload`. It never enables the unit.

- [ ] **Step 1: Write the failing tests**

```go
// internal/sysunit/sysunit_test.go
func TestUnitCarriesTheSpecsHardening(t *testing.T) // Unit("/usr/bin/jumpgate") contains each line: "ExecStart=/usr/bin/jumpgate serve --no-open --system", "User=jumpgate-ctl", "Group=jumpgate-ctl", "SupplementaryGroups=jumpgate-admin", "Environment=HOME=/var/lib/jumpgate-ctl", "StateDirectory=jumpgate-ctl", "RuntimeDirectory=jumpgate-ctl", "RuntimeDirectoryMode=0750", "UMask=0077", "LimitCORE=0", "NoNewPrivileges=yes", "ProtectSystem=strict", "ProtectHome=yes", "PrivateTmp=yes", "Restart=on-failure"; no line contains "#" after "="
func TestPackagingFilesMatchSysunit(t *testing.T)   // os.ReadFile("../../packaging/linux/server/jumpgate.service") == Unit("/usr/bin/jumpgate"); jumpgate.sysusers == Sysusers
// cmd/jumpgate/cli_setup_test.go
func TestInstallServiceWritesAndReloads(t *testing.T) // geteuid seam 0, hostGOOS linux, runPrivileged -> cmdRecorder; WriteFile paths == [AdminUnitPath, AdminSysusersPath]; commands in order "systemd-sysusers /etc/sysusers.d/jumpgate.conf", "systemctl daemon-reload"; none contains "enable"
func TestInstallServiceNeedsRoot(t *testing.T)        // geteuid seam 1000 -> exitCode("failed"), nothing written
func TestInstallServiceDefersToThePackage(t *testing.T) // a stat seam reports PackageUnitPath present -> failed, message contains "apt or dnf"
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/sysunit/ ./cmd/jumpgate/ -run 'Unit|PackagingFiles|InstallService' -count=1`
Expected: FAIL, `no Go files in internal/sysunit`.

- [ ] **Step 3: Implement** the Produces block. Then generate the two packaging files from `Unit("/usr/bin/jumpgate")` and `Sysusers`, and commit them.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./internal/sysunit/ ./cmd/jumpgate/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/sysunit packaging/linux/server cmd/jumpgate
git commit -m "feat(setup): jumpgate.service, sysusers and setup install-service"
```

---

### Task 9: `jumpgate setup server`: key in the vault, unlock, pair self

**Files:**
- Modify: `cmd/jumpgate/cli_setup.go` (`setupServer`), `internal/server/system.go` (`POST /api/system/init-key`), `internal/config/config.go` (`Controller.Scope`), `internal/api/wire.go` (`SystemInitKey`), `internal/api/errors.go` (`controller_exists`), `internal/apiclient/methods.go`
- Test: `cmd/jumpgate/cli_setup_test.go`, `internal/server/system_test.go`

**Interfaces:**
- Consumes: `secrets.VaultPath`, `vault.Create`, `signer.CreateVault`, `(*Server).VaultState`, `apiclient.(*Client).Unlock`, `readPassphrase` (vault plan); `daemon.FindSystem` (Task 3); `factsFromServer`, `pairLocalForeground` (Task 6); `api.CustodyWarning` (Task 7).
- Produces:
  ```go
  type config.Controller struct{ /* existing… */ Scope string `json:"scope,omitempty"` } // "" or "self"
  type api.SystemInitKey struct {
  	Store string `json:"store"` // "vault" (default) | "1password" | "file"
  	Ref string `json:"ref,omitempty"`; Passphrase string `json:"passphrase,omitempty"`
  	Scope string `json:"scope,omitempty"`; InsecurePlain bool `json:"insecurePlain,omitempty"`
  }
  const api.CodeControllerExists Code = "controller_exists" // hint: "this controller already has a key; run `jumpgate unlock`"
  func (c *apiclient.Client) SystemInitKey(ctx context.Context, req api.SystemInitKey) error
  var reexecSudo = func(args []string) int // seam: execs `sudo -- <exe> setup server <args>`, stdin passed through
  ```
- `POST /api/system/init-key` is system mode and socket peers only (else 404). With a recorded controller it answers 409 `controller_exists`.
  - `vault`: `path, _ := secrets.VaultPath()`, then `vault.Create(path, pass, vault.DefaultParams)` (it clears `pass`, so pass it a copy when the same bytes are needed again), then `signer.CreateVault(v, "controller-key")`, then record `Controller{KeyStore: "vault", KeyRef: "controller-key", Address, Scope}` and `SecretStore: "vault"`, then `v.Lock()` on that handle. The server finds the new file lazily (vault S8), so its `VaultState().State` reads `locked` on the next request. The passphrase bytes are zeroed afterwards.
  - `1password`: `signer.Create(ctx, signer.StoreOnePassword, Ref)`.
  - `file`: needs `InsecurePlain` (else 400 naming `--insecure-plain-key`). The ref is always `<config.Dir()>/keys/controller.key`, and a client `Ref` is refused (400), per Review Focus 5.
- `jumpgate setup server [--name NAME] [--store vault|1password|file] [--ref REF] [--insecure-plain-key] [--self-only] [--stdin]` runs these steps in order. Each step is skipped when it is already done.
  1. Linux only. When not root, call `reexecSudo`, after printing "setup server runs as root through sudo".
  2. Refuse when neither `sysunit.PackageUnitPath` nor `AdminUnitPath` exists: "install the jumpgate package, or run install.sh --controller, first".
  3. `systemctl enable --now jumpgate.service`.
  4. When `$SUDO_USER` is set and not a member: `usermod -aG jumpgate-admin $SUDO_USER`.
  5. Wait up to 10 s for `daemon.FindSystem`.
  6. With no controller: read the passphrase (`--stdin`: `readPassphrase(prompt, true, false)`, one line with exactly one trailing `\n` or `\r\n` stripped; otherwise `readPassphrase("passphrase: ", false, true)`, which reads `/dev/tty` twice and requires a match). Print `api.CustodyWarning`. Then `SystemInitKey`.
  7. When `VaultState().State` is `locked`: `Unlock` with the same bytes (a copy, since `Unlock` clears them). A `wrong_passphrase` exits 1 and names `jumpgate unlock`.
  8. When target `NAME` (default `os.Hostname()`) is missing or unpaired: `AddTarget{ID: NAME, Mode: "local"}`, then `pairLocalForeground(factsFromServer…)`, then `streamPair(Installed)`.
  9. Print: "log out and back in so the jumpgate-admin group applies to you, then run `jumpgate` here or `ssh -t <host> jumpgate` from your machine".

- [ ] **Step 1: Write the failing tests**

```go
// cmd/jumpgate/cli_setup_test.go (geteuid seam 0, runPrivileged recorder, fake API via connect seam, findSystem seam)
func TestSetupServerRunsTheStepsInOrder(t *testing.T) // recorder+API log == ["systemctl enable --now jumpgate.service", "usermod -aG jumpgate-admin alice", "POST /api/system/init-key", "POST /api/vault/unlock", "POST /api/targets", "POST /api/targets/box/pair"]; pair body has "installed"
func TestSetupServerReadsOneLineFromStdin(t *testing.T) // stdin "pw\r\n" -> init-key passphrase "pw"; stdin "pw\n\n" -> "pw"
func TestSetupServerRefusesPlainWithoutTheFlag(t *testing.T) // --store file -> exitCode("usage"), recorder empty
func TestSetupServerSkipsDoneSteps(t *testing.T) // API: controller present, vault state "unlocked", target paired -> no init-key, unlock or pair calls
func TestSetupServerPrintsTheCustodyWarning(t *testing.T) // stdout contains api.CustodyWarning
func TestSetupServerReexecsThroughSudo(t *testing.T) // geteuid seam 1000 -> reexecSudo called with ["--stdin"]; nothing else runs
// internal/server/system_test.go
func TestSystemInitKeyCreatesAVaultKeyAndLocks(t *testing.T) // after: config Controller.KeyStore == "vault", Address != "", VaultState().State == "locked"
func TestSystemInitKeyRefusesASecondKey(t *testing.T) // 409 controller_exists
func TestSystemInitKeyRefusesPlainWithoutTheFlag(t *testing.T) // {"store":"file"} -> 400 mentioning --insecure-plain-key
func TestSystemInitKeyRefusesAClientChosenFileRef(t *testing.T) // {"store":"file","insecurePlain":true,"ref":"/etc/shadow"} -> 400
func TestSystemInitKeyRecordsScopeSelf(t *testing.T) // {"scope":"self"} -> Controller.Scope == "self"
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/jumpgate/ ./internal/server/ -run 'SetupServer|SystemInitKey' -count=1`
Expected: FAIL (`unknown setup subcommand "server"`; 404 for init-key).

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./cmd/jumpgate/ ./internal/server/ ./internal/config/ ./internal/api/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate internal/server internal/config internal/api internal/apiclient
git commit -m "feat(setup): setup server creates the vault key, unlocks and pairs the controller"
```

---

### Task 10: Restarting the service from the TUI (D15, client side)

**Files:**
- Modify: `internal/server/server.go` (`Config.Restart`, `POST /api/restart`), `cmd/jumpgate/serve_system.go` (exit 75), `internal/apiclient/client.go` (`Restart`), `internal/tui/app.go` (banner), `internal/tui/backend.go` / `tui.Options` (`SystemService bool`), `cmd/jumpgate/tui_on.go`
- Test: `internal/server/system_test.go`, `cmd/jumpgate/serve_system_test.go`, `internal/apiclient/client_test.go`, `internal/tui/restart_test.go`

**Interfaces:**
- Produces:
  ```go
  type server.Config struct{ /* … */ Restart func() } // set only by serve --system
  const exitRestart = 75                                // serve --system exits 75 after /api/restart; Restart=on-failure brings it back
  func systemExitCode(restartRequested bool, err error) int // 75 | 1 | 0
  type tui.Options struct{ /* … */ SystemService bool }
  ```
- `POST /api/restart` answers 202 and calls `Restart` (404 when nil).
- `apiclient.Restart` on a `System` client posts `/api/restart`, then waits up to `restartWait` for `daemonFind` to report a different PID, and reconnects with `Connect(ctx, Options{})`.
- The TUI banner with `SystemService`: `server %s, this jumpgate %s: press R to restart now (you will need to unlock)`.

- [ ] **Step 1: Write the failing tests**

```go
func TestRestartRouteOnlyWhenConfigured(t *testing.T) // Config.Restart nil -> 404; set -> 202 and called once
func TestServeSystemExitsSeventyFiveOnRestart(t *testing.T) // systemExitCode(true, nil) == 75; (false, nil) == 0; (false, errors.New("x")) == 1
func TestRestartTheServiceWaitsForANewPID(t *testing.T) // daemonFind seam: PID 10 twice, then 11; Restart returns a client whose Info().PID == 11; POST /api/restart seen; daemonStop seam never called
func TestSkewBannerForTheServiceMentionsUnlock(t *testing.T) // tuitest.Fake skew differs; Options{SystemService: true, Restart: f}; view contains "you will need to unlock"
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/server/ ./cmd/jumpgate/ ./internal/apiclient/ ./internal/tui/ -run 'RestartRoute|SeventyFive|RestartTheService|SkewBannerForTheService' -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./internal/server/ ./cmd/jumpgate/ ./internal/apiclient/ ./internal/tui/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/server cmd/jumpgate internal/apiclient internal/tui
git commit -m "feat(tui): restart the jumpgate service, warning that it comes back locked"
```

---

### Task 11: Unattended unlock: sealing the service's vault with `systemd-creds`

**Files:**
- Create: `cmd/jumpgate/seal_system.go`, `cmd/jumpgate/seal_system_test.go`
- Modify: the vault plan's `vault seal` command in `cmd/jumpgate/cli_vault.go` (add the system branch), `internal/server/system.go` (`POST /api/system/seal`), `cmd/jumpgate/serve_system.go` (builds the server through `buildServer`, so vault Task 11's `vault-auto-unlock` reads `$CREDENTIALS_DIRECTORY` at start)
- Test: `internal/server/system_test.go`

**Interfaces:**
- Consumes: `(*vault.Vault).AddKEKSlot`, `(*Server).UnlockKEK`, `credentialKEK`, `vault seal` (vault Tasks 3, 7, 11, 16); `sysunit.SealDropInPath`, `SealCredPath` (Task 8); `peerFrom` (Task 2).
- Produces:
  ```go
  const sealCred = "jumpgate-vault-key" // the slot's Cred name
  func sealSystem(ctx context.Context, out io.Writer, c *apiclient.Client, tpm2, allowHostKey bool) int
  func sealKEK() ([]byte, error) // 32 bytes from crypto/rand; the server draws it, the vault plan has no secret-returning slot call
  ```
- `jumpgate vault seal --tpm2` or `--allow-host-key` (one mode flag, as vault Task 16 defines) against a system server:
  1. It must be root (re-exec through `sudo`).
  2. `systemd-creds has-tpm2` must succeed. Otherwise it refuses: "this machine has no TPM2; --allow-host-key seals with the host key instead, which protects only against copying the vault file alone".
  3. `POST /api/system/seal`, which is root peers only (403 for others) and needs the vault unlocked (the vault plan's 503 `locked`). The handler draws `kek := sealKEK()`, calls `v.AddKEKSlot(vault.Slot{Kind: vault.SlotSystemdCreds, Cred: sealCred}, copyOf(kek))` (it clears its argument), and returns `kek`.
  4. Pipe the secret to `systemd-creds encrypt --with-key=tpm2` (or `--with-key=host` for `--allow-host-key`) `--name=jumpgate-vault-key - /etc/jumpgate-ctl/vault-key.cred`, written root 0600 in a 0700 directory.
  5. Write `SealDropInPath` with `[Service]\nLoadCredentialEncrypted=jumpgate-vault-key:/etc/jumpgate-ctl/vault-key.cred\n`.
  6. `systemctl daemon-reload`, then zero the secret.
- At `serve --system` start, vault Task 11's `vault-auto-unlock` task reads `credentialKEK(os.Getenv)` and calls `(*Server).UnlockKEK(vault.SlotSystemdCreds, kek)`. On failure the server stays locked and logs `vault stays locked: <reason>; the passphrase still works` (the vault plan's line). This plan adds no second unlock path.

- [ ] **Step 1: Write the failing tests**

```go
func TestSealRefusesWithoutATPM(t *testing.T) // recorder: "systemd-creds has-tpm2" exits 1; exit failed; stderr contains "--allow-host-key"; no POST /api/system/seal
func TestSealWritesTheCredentialAndTheDropIn(t *testing.T) // encrypt argv contains "--with-key=tpm2" and "--name=jumpgate-vault-key"; its stdin == served secret; WriteFile(SealDropInPath) content == the drop-in above; last command "systemctl daemon-reload"
func TestSealWithTheHostKeyNeedsTheFlag(t *testing.T) // no TPM + --allow-host-key -> "--with-key=host"
func TestSealRouteIsForRootOnly(t *testing.T) // peer uid 1000 -> 403; uid 0 -> 200
func TestStartUnlocksFromTheCredentialsDirectory(t *testing.T) // real vault: Create, AddKEKSlot(Slot{Kind: SlotSystemdCreds, Cred: sealCred}, kek), Lock; write kek to dir/jumpgate-vault-key; the system-mode server built by serve_system.go has VaultState().State == "unlocked" once vault-auto-unlock ran; a wrong kek -> still "locked" and the log has "the passphrase still works"
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/jumpgate/ ./internal/server/ -run 'Seal|UnlocksFromTheCredentials' -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./cmd/jumpgate/ ./internal/server/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate internal/server
git commit -m "feat(vault): seal the service's vault with systemd-creds for unattended unlock"
```

---

### Task 12: Idle auto-lock (resolved Q2): delivered by the vault plan

**Removed from this plan.** The encrypted-secret-store plan's Task 19 already builds the idle auto-lock: setting `Config.VaultIdleLockMinutes` / `Settings.VaultIdleLockMinutes` (0 to 1440, off by default), an idle loop that locks when no POST, PUT or DELETE has arrived for N minutes and `driverCount() == 0`, the TUI and web Settings rows, and the log line `vault locked after N minutes idle`. A second implementation here (`AutoLockMinutes`, `idleFor`, `anyDriving`) would add a duplicate setting. This plan consumes Task 19 as it is; one difference is accepted: reads and streams are not activity, as in the vault plan's S9.

Nothing to implement, test or commit in this task. The task number stays so other references keep their meaning.

---

### Task 13: `install.sh` for the CLI and `--controller` roles

**Files:**
- Create: `packaging/install/install.sh`, `scripts/e2e-install.sh`, `scripts/e2e/relsrv/main.go`
- Modify: `scripts/e2e/Dockerfile` (add `curl` and `dpkg-dev`)

**Interfaces:**
- Consumes: `jumpgate setup install-service` (Task 8); `api.CustodyWarning` text (Global Constraints).
- Produces:
  - **Command line:** `install.sh [--controller | --agent] [--version vX.Y.Z] [--yes] [--prefix DIR] [--require-provenance] [--self-only]`, plus for `--agent`: `--controller ADDR --transport-key KEY`.
    - Because `--agent` changes the meaning of `--controller`, the script scans for `--agent` before parsing. With `--agent`, `--controller` takes a value; without it, `--controller` is the role flag.
    - `--version 0.6.0` is normalised to `v0.6.0`.
  - **Download layout:** `BASE=${JUMPGATE_INSTALL_BASE_URL:-https://github.com/jumpgate-tech/jumpgate-app/releases}`. Files come from `$BASE/download/$tag/`. `latest` resolves to the tag at the end of `curl -fsSLI -o /dev/null -w '%{url_effective}' "$BASE/latest"` (`…/tag/vX.Y.Z`), and the script prints `jumpgate <tag>` before downloading.
  - **Roles:**
    - With no role on a tty: ask on `/dev/tty`: `1) CLI only (default)  2) controller server  3) agent-only box`.
    - With no tty: install the CLI and print both role commands.
    - CLI role: `/usr/local/bin/jumpgate`, or `DIR/bin/jumpgate` with `--prefix` (no root).
    - `--controller` (root): `/usr/local/bin/jumpgate`, then `/usr/local/bin/jumpgate setup install-service`. The closing text is the custody warning plus "next: `jumpgate setup server`" (with ` --self-only` appended when given).
  - **Temp files:** `mktemp -d "${TMPDIR:-/tmp}/jumpgate-install.XXXXXX"`, removed by `trap … EXIT INT TERM`.
  - **Verification:**
    - Checksum: `grep " $file\$" checksums.txt | sha256sum -c -` (or `shasum -a 256 -c -`).
    - Provenance, when `gh auth status` succeeds: `gh attestation verify "$file" --repo jumpgate-tech/jumpgate-app`.
    - Otherwise print the D13 line verbatim; with `--require-provenance` that case fails.
  - **Refusals:**
    - `uname -s` other than Linux, or `uname -m` other than `x86_64|amd64|aarch64|arm64`: stop, naming the value and "supported: Linux on amd64 or arm64".
    - `/usr/bin/jumpgate` owned by `dpkg -S` or `rpm -qf`: refuse with "use apt upgrade jumpgate or dnf upgrade jumpgate".
  - **Root:**
    - As non-root, print each command prefixed `sudo`, then ask `[y/N]` on `/dev/tty`.
    - `--yes` skips the question.
    - With no tty and no `--yes`: print the commands and exit 1 before any `sudo`.
  - **`relsrv`:** `go run ./scripts/e2e/relsrv -dir D -cert C -key K -addr :8443` serves `D/download/<tag>/<file>` over TLS and redirects `/latest` (302) to `/tag/<newest tag dir>`.

- [ ] **Step 1: Write the failing e2e** (`scripts/e2e-install.sh`, bash, `set -euo pipefail`, a container per case as in `e2e-local.sh`)
  - **Fixtures:** a release dir with tags `v0.0.1` and `v0.0.2`. Each tag holds `jumpgate_linux_<arch>.tar.gz` containing a stub `jumpgate` that prints its tag, plus `checksums.txt`. A real `v0.0.2-real` tag is built with `go build -ldflags "-X …buildinfo.version=v0.0.2"` for `--controller`. A self-signed CA is copied into the container's trust store, and `JUMPGATE_INSTALL_BASE_URL=https://relsrv:8443`.
  - **Cases:**
    1. `head -c 200 install.sh | sh`: exit status 0 and nothing installed (`! test -e /usr/local/bin/jumpgate`).
    2. A tampered archive: nonzero, output contains `checksum`, and `ls /tmp | grep -c jumpgate-install` == 0.
    3. `--require-provenance` without `gh`: nonzero, output contains `provenance`.
    4. `--version v0.0.1 --yes` as root: `jumpgate` prints `v0.0.1`. With no `--version`, the output contains `jumpgate v0.0.2` and the binary prints `v0.0.2`.
    5. As alice, with no tty and no `--yes`: exit 1, output contains `sudo install`, and no `/usr/local/bin/jumpgate`.
    6. As bob, `--prefix /home/bob/.local`: `/home/bob/.local/bin/jumpgate` exists.
    7. A PATH shim `uname` printing `mips64` for `-m`: nonzero, output contains `mips64` and `amd64 or arm64`.
    8. A dummy `.deb` owning `/usr/bin/jumpgate` (`dpkg-deb --build`, `dpkg -i`): nonzero, output contains `apt upgrade`.
    9. `--controller --yes` as root with the real tag: `/etc/systemd/system/jumpgate.service` exists, `getent group jumpgate-admin` succeeds, `systemctl is-enabled jumpgate` is not `enabled`, and the output contains `can sign routine intents for every box`.

- [ ] **Step 2: Run it to verify it fails**

Run: `scripts/e2e-install.sh`
Expected: FAIL at case 1, `packaging/install/install.sh: No such file`.

- [ ] **Step 3: Write `install.sh` and `relsrv`** to the contract above.

- [ ] **Step 4: Lint and run**

Run: `sh -n packaging/install/install.sh && docker run --rm -v "$PWD":/m koalaman/shellcheck:v0.10.0 -s sh /m/packaging/install/install.sh && scripts/e2e-install.sh && BASE=ubuntu:24.04 scripts/e2e-install.sh`
Expected: no shellcheck findings; every case prints `ok`; exit 0 on both distros.

- [ ] **Step 5: Commit**

```bash
git add packaging/install scripts/e2e-install.sh scripts/e2e
git commit -m "feat(install): install.sh for the CLI and controller roles, verified"
```

---

### Task 14: deb and rpm packages (nfpm), and a postinst that never re-locks the fleet

**Files:**
- Create: `packaging/linux/server/postinst.sh`, `packaging/packaging_test.go`
- Modify: `.goreleaser.yaml`

**Interfaces:**
- Consumes: `packaging/linux/server/jumpgate.service` and `jumpgate.sysusers` (Task 8); `sysunit.SealDropInPath` (Task 8, as a literal in shell).
- Produces:
  - In `.goreleaser.yaml`:
    ```yaml
    nfpms:
      - id: jumpgate
        package_name: jumpgate
        ids: [jumpgate]
        formats: [deb, rpm]
        bindir: /usr/bin
        homepage: https://jumpgate.app
        description: jumpgate controller and CLI
        license: see LICENSE
        maintainer: jumpgate-tech <https://github.com/jumpgate-tech/jumpgate-app>
        file_name_template: "{{ .PackageName }}_{{ .Version }}_{{ .Arch }}"
        contents:
          - {src: packaging/linux/server/jumpgate.service, dst: /usr/lib/systemd/system/jumpgate.service, file_info: {mode: 0644}}
          - {src: packaging/linux/server/jumpgate.sysusers, dst: /usr/lib/sysusers.d/jumpgate.conf, file_info: {mode: 0644}}
        scripts:
          postinstall: packaging/linux/server/postinst.sh
    ```
  - `postinst.sh` (POSIX):
    - When `[ -d /run/systemd/system ]`: `systemd-sysusers /usr/lib/sysusers.d/jumpgate.conf`, then `systemctl daemon-reload`.
    - Then only `[ -f /etc/systemd/system/jumpgate.service.d/50-vault-seal.conf ] && systemctl try-restart jumpgate.service`.
    - Never `enable`, never a plain `restart`.
  - The maintainer string is a placeholder for the user to confirm at review.

- [ ] **Step 1: Write the failing test** in `packaging/packaging_test.go` (`package packaging`; a test-only package)

```go
func TestNFPMShipsTheServiceFiles(t *testing.T) // .goreleaser.yaml text contains "nfpms:", "formats: [deb, rpm]", "dst: /usr/lib/systemd/system/jumpgate.service", "dst: /usr/lib/sysusers.d/jumpgate.conf", "postinstall: packaging/linux/server/postinst.sh"; does NOT contain "/usr/local/lib/jumpgate"
func TestPostinstNeverEnablesOrRestartsAManualService(t *testing.T) // postinst.sh: no "enable"; every "restart" is "try-restart" on a line guarded by "50-vault-seal.conf"; `sh -n` exits 0
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./packaging/ -count=1`
Expected: FAIL, `.goreleaser.yaml` has no `nfpms:`.

- [ ] **Step 3: Add** the `nfpms:` block and `postinst.sh`.

- [ ] **Step 4: Verify the package contents**

Run: `go test ./packaging/ -count=1 && goreleaser check && goreleaser release --snapshot --clean --skip=publish && docker run --rm -v "$PWD/dist":/d debian:12 sh -c 'dpkg-deb -c /d/jumpgate_*_amd64.deb; dpkg-deb -I /d/jumpgate_*_amd64.deb postinst'`
Expected:
- `ok`;
- `goreleaser check` reports no errors;
- the listing shows `./usr/bin/jumpgate`, `./usr/lib/systemd/system/jumpgate.service` with mode `-rw-r--r--`, and `./usr/lib/sysusers.d/jumpgate.conf`;
- the postinst body matches `postinst.sh`;
- an `rpm` and an `arm64` deb exist in `dist/`.

- [ ] **Step 5: Commit**

```bash
git add .goreleaser.yaml packaging/linux/server/postinst.sh packaging/packaging_test.go
git commit -m "feat(release): deb and rpm packages for the controller service"
```

---

### Task 15: The signed apt and dnf repository, built and tested with a throwaway key

**Files:**
- Create: `scripts/build-pkg-repo.sh`, `packaging/repo/jumpgate.repo`, `packaging/repo/jumpgate.sources`, `scripts/e2e-repo.sh`, `scripts/e2e/Dockerfile.fedora`

**Interfaces:**
- Consumes: the `.deb` and `.rpm` files from Task 14.
- Produces:
  - **Command:** `scripts/build-pkg-repo.sh --in DIR --out DIR --key-id ID`. `GNUPGHOME` is from the environment, and the passphrase comes from the file named by `PKG_SIGNING_PASSPHRASE_FILE`, never from argv.
  - **Output:**
    - `OUT/apt/`: aptly `publish repo -distribution=stable -component=main -architectures=amd64,arm64`, giving `dists/stable/{InRelease,Release,Release.gpg}` and `pool/`;
    - `OUT/rpm/`: `rpmsign --addsign` on each rpm, `createrepo_c`, and `repodata/repomd.xml.asc` from `gpg --detach-sign --armor`; one multi-arch repo (Q6 open);
    - `OUT/jumpgate.asc`: the public key;
    - `OUT/jumpgate.repo`: copied from `packaging/repo/`.
  - `jumpgate.repo` contains:
    ```
    [jumpgate]
    name=jumpgate
    baseurl=https://pkg.jumpgate.app/rpm
    enabled=1
    gpgcheck=1
    repo_gpgcheck=1
    gpgkey=https://pkg.jumpgate.app/jumpgate.asc
    ```
  - `jumpgate.sources` is a deb822 file: `Types: deb`, `URIs: https://pkg.jumpgate.app/apt`, `Suites: stable`, `Components: main`, `Signed-By: /usr/share/keyrings/jumpgate-archive-keyring.gpg`.
  - **Base URL in tests:** `scripts/e2e-repo.sh` rewrites `pkg.jumpgate.app` to its local HTTP server's address.

- [ ] **Step 1: Write the failing e2e** (`scripts/e2e-repo.sh`)
  - **Fixtures:** build snapshots `v0.0.1` and `v0.0.2` (`GORELEASER_CURRENT_TAG=vX goreleaser release --snapshot --clean --skip=publish`, saving each `dist/*.deb *.rpm`). Generate a throwaway key inside a builder container (`gpg --batch --passphrase-file … --quick-gen-key 'jumpgate e2e <e2e@invalid>' ed25519 sign 1d`), build the repo, and serve it with `python3 -m http.server` on a docker network.
  - **Cases:**
    1. `debian:12` and `ubuntu:24.04` (systemd containers from `scripts/e2e/Dockerfile`): install the keyring (`gpg --dearmor`) and `jumpgate.sources`, then `apt-get update && apt-get install -y jumpgate`. Expect `/usr/bin/jumpgate` and `getent passwd jumpgate-ctl`.
    2. Upgrade with no restart: start the service (`systemctl start jumpgate`, which is locked with no vault), record `systemctl show -p MainPID jumpgate`, publish `v0.0.2`, run `apt-get install --only-upgrade jumpgate`. Expect the same MainPID.
    3. Tampered `InRelease` (one byte flipped): `apt-get update` exits nonzero and its output contains `NOSPLIT`, `BADSIG` or `not signed`.
    4. `fedora:41`: `dnf install -y jumpgate` with `repo_gpgcheck=1` succeeds. A tampered `repomd.xml` makes `dnf makecache` fail.

- [ ] **Step 2: Run it to verify it fails**

Run: `scripts/e2e-repo.sh`
Expected: FAIL, `scripts/build-pkg-repo.sh: No such file`.

- [ ] **Step 3: Write** `build-pkg-repo.sh`, the two repo files and `Dockerfile.fedora`.

- [ ] **Step 4: Lint and run**

Run: `docker run --rm -v "$PWD":/m koalaman/shellcheck:v0.10.0 /m/scripts/build-pkg-repo.sh /m/scripts/e2e-repo.sh && scripts/e2e-repo.sh`
Expected: no findings; every case prints `ok`; exit 0.

- [ ] **Step 5: Commit**

```bash
git add scripts/build-pkg-repo.sh scripts/e2e-repo.sh scripts/e2e/Dockerfile.fedora packaging/repo
git commit -m "feat(release): build and test the signed apt and dnf repository"
```

---

### Task 16: Pull pairing: `agent setup`, `install.sh --agent`, `hosts invite`

**Files:**
- Modify: `cmd/jumpgate/cli_agent.go` (`setup` subcommand), `cmd/jumpgate/cli_hosts.go` (`invite`), `packaging/install/install.sh` (`--agent`), `scripts/e2e-install.sh` (an agent case)
- Test: `cmd/jumpgate/cli_agent_test.go`, `cmd/jumpgate/cli_hosts_invite_test.go`

**Interfaces:**
- Consumes: `runBootstrap`, `newLocalExecutor` seams; `PairingInfo` (Task 5); `daemon.Info.Version`.
- Produces:
  ```go
  func agentSetup(ctx context.Context, out io.Writer, controller, transportKey, label string) int
  func hostsInvite(ctx context.Context, out io.Writer, name string) int
  var selfBinary = func(arch string) ([]byte, error) // os.Executable() bytes; error unless arch == runtime.GOARCH
  ```
- `jumpgate agent setup --controller ADDR --transport-key KEY [--label L]`:
  1. It must be root (`geteuid`).
  2. `runBootstrap(Options{Exec: newLocalExecutor(), Local: false, AgentBinary: selfBinary, Controller, ControllerLabel: L (default "pull"), TransportKey: KEY})`.
  3. On success it prints `agent address 0x…`, then `on the controller, run: jumpgate hosts add NAME --ssh jumpgate@<hostname> --installed 0x…`.
- `install.sh --agent` (root):
  1. It requires `--controller` and `--transport-key`.
  2. It downloads `jumpgate-linux-<arch>` and `checksums.txt` and verifies them like the archive.
  3. `install -D -m 0755` to `/usr/local/lib/jumpgate/jumpgate`.
  4. It runs `/usr/local/lib/jumpgate/jumpgate agent setup --controller … --transport-key …`.
- `jumpgate hosts invite NAME` prints this (single quotes in the key are escaped as `'"'"'`):
  ```
  curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent \
    --version <server version> --controller <address> --transport-key '<key line>'
  ```
  Then it prints `then: jumpgate hosts add NAME --ssh jumpgate@HOST --installed <address printed on the box>`.
  - It refuses a server version that is not `vX.Y.Z` ("this controller is a development build; an invite needs a released version").
  - It refuses `PairingInfo.Scope == "self"` with `self_only` (Task 19's code).

- [ ] **Step 1: Write the failing tests**

```go
func TestAgentSetupRunsBootstrapWithTheTransportKey(t *testing.T) // geteuid 0; runBootstrap stub: Local false, TransportKey == "ssh-ed25519 AAAA… jumpgate-controller", Controller parsed, Exec not a sudo wrapper; stdout last line starts "on the controller, run: jumpgate hosts add"
func TestAgentSetupNeedsRoot(t *testing.T) // geteuid 1000 -> exit 1; runBootstrap not called
func TestAgentSetupRefusesABadAddress(t *testing.T) // --controller nope -> usage
func TestHostsInvitePrintsAPinnedOneLiner(t *testing.T) // fake server: version v0.6.0, pairing-info; stdout contains "--version v0.6.0", "--controller 0x", "--transport-key 'ssh-ed25519 "
func TestHostsInviteRefusesADevBuild(t *testing.T) // version "dev" -> failed, stderr contains "released version"
func TestHostsInviteRefusesSelfOnly(t *testing.T) // scope "self" -> failed naming self_only
```
Also add a case to `scripts/e2e-install.sh`: `--agent` without `--transport-key` → nonzero, output contains `--transport-key`.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/jumpgate/ -run 'AgentSetup|HostsInvite' -count=1`
Expected: FAIL (`unknown agent subcommand "setup"`).

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./cmd/jumpgate/ -count=1 && scripts/e2e-install.sh`
Expected: PASS; every e2e case prints `ok`.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate packaging/install scripts/e2e-install.sh
git commit -m "feat(hosts): pull pairing with hosts invite, install.sh --agent and agent setup"
```

---

### Task 17: `agent.upgrade` on the box (resolved Q3)

Both pull and push boxes upgrade the same way, over the pairing: a signed intent through the tunnel, with the binary streamed in the request body. The tier is routine, so it matches re-pairing a pushed box, which needs no approval either. A box can still raise it with `policy.json` `"kinds": {"agent.upgrade": ["routine", "approval"]}`. The custody warning names it.

**Files:**
- Create: `internal/agent/upgrade.go`, `internal/agent/upgrade_test.go`
- Modify: `internal/intent/kinds.go`, `internal/agent/policy.go` (`DefaultRequirements`), `internal/agent/listen.go` (`Serve` route), `internal/agent/dispatch.go`, `internal/agent/agent.go` (`Config.BinaryPath`, `Config.Restart`), `cmd/jumpgate/cli_agent.go` (`agentRun` wiring), `internal/intent/kinds.go` `AgentInfo.Arch`

**Interfaces:**
- Produces:
  ```go
  const intent.KindAgentUpgrade = "agent.upgrade"
  type intent.AgentUpgradePayload struct{ Arch, Version, SHA256 string; Size int64 } // json: arch, version, sha256, size
  type intent.AgentUpgradeResult struct{ Upgraded bool; From, To string }            // json: upgraded, from, to
  type intent.AgentInfo struct{ /* … */ Arch string `json:"arch"` }                 // runtime.GOARCH
  const UpgradeEnvelopeHeader = "Jumpgate-Envelope" // base64url(json(intent.Envelope)), at most 64 KiB
  const maxUpgrade = 256 << 20
  type agent.Config struct{ /* … */ BinaryPath string; Restart func() } // agentRun: "/usr/local/lib/jumpgate/jumpgate"; `systemctl --no-block restart jumpgate-agent.service`
  ```
- `POST /v1/upgrade` goes through the same peer gate and then `a.Handle(withUpgradeBody(ctx, r.Body), env)`, so admission (signature, policy, replay) happens before a byte of the body is read.
- `dispatch` for `agent.upgrade` checks in this order:
  1. No upload body in ctx → `invalid_payload` "agent.upgrade is sent to /v1/upgrade with the binary".
  2. `Arch != runtime.GOARCH` → `validation`.
  3. `Size` must be in 1..`maxUpgrade`, else `invalid_payload`.
  4. `JobsUnavailable != nil`, or `Jobs.Active()` → `job_active`.
  5. Read the body.
- The body must be exactly `Size` bytes. It is streamed into `BinaryPath+".new"` (any stale `.new` is removed first; mode 0755) while hashing.
  - A short body or a hash mismatch: remove `.new` and answer `StatusFailed` naming which.
  - The same hash as the current binary: remove `.new` and answer `Upgraded:false`.
  - Otherwise fsync, rename, and answer `Upgraded:true`. The route handler calls `Config.Restart` only after the receipt is written.

- [ ] **Step 1: Write the failing tests** in `upgrade_test.go` (use `newRig`; a temp `BinaryPath` holding "old"; a counting `Restart` seam)

```go
func TestUpgradeInstallsAVerifiedBinary(t *testing.T) // body "new" with matching sha/size -> result Upgraded true; file == "new", mode 0o755; restart count 1
func TestUpgradeRefusesAHashMismatch(t *testing.T)    // wrong sha -> StatusFailed; file == "old"; no BinaryPath+".new"; restart 0
func TestUpgradeRefusesAShortBody(t *testing.T)       // Size 10, body 3 bytes -> StatusFailed; file == "old"; no .new; restart 0
func TestUpgradeRefusesWhileAJobIsActive(t *testing.T) // jobs manager with an active job -> rejection job_active; body never read (a reader that fails the test on Read)
func TestUpgradeRefusesOnTheIntentRoute(t *testing.T) // via a.Handle with no upload body -> invalid_payload
func TestUpgradeOfTheSameBinaryDoesNotRestart(t *testing.T) // body "old" -> Upgraded false; restart 0
func TestUpgradeRefusesAnotherArch(t *testing.T)       // Arch "riscv64" -> validation
func TestUpgradePolicyCanRequireApproval(t *testing.T) // Policy.Kinds["agent.upgrade"] = [routine, approval], routine sig only -> unauthorized_kind
func TestUpgradeEnvelopeHeaderIsCapped(t *testing.T)   // Serve over a unix socket (peerGate seam true); a 65 KiB header -> HTTP 400, nothing written
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/agent/ -run Upgrade -count=1`
Expected: FAIL, `undefined: intent.KindAgentUpgrade`.

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass, here and on Linux**

Run: `go test ./internal/agent/ ./internal/intent/ -count=1 && scripts/test-linux.sh ./internal/agent/ -run Upgrade`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/agent internal/intent cmd/jumpgate/cli_agent.go
git commit -m "feat(agent): agent.upgrade replaces the agent binary over the pairing"
```

---

### Task 18: `jumpgate hosts upgrade`, from the controller

**Files:**
- Create: `internal/server/upgrade.go`, `internal/server/upgrade_test.go`
- Modify: `internal/agentclient/client.go` (`Upload`; factor `once`), `internal/api/wire.go` (`AgentUpgradeView`), `internal/apiclient/methods.go`, `cmd/jumpgate/cli_hosts.go` (`upgrade`), `internal/api/errors.go` (hints)
- Test: `internal/agentclient/client_test.go`, `cmd/jumpgate/cli_hosts_upgrade_test.go`, `internal/api/errors_test.go`

**Interfaces:**
- Consumes: `intent.KindAgentUpgrade`, `AgentUpgradePayload`, `AgentUpgradeResult`, `AgentInfo.Arch`, `UpgradeEnvelopeHeader` (Task 17); `agentbin.Load`.
- Produces:
  ```go
  func (c *agentclient.Client) Upload(ctx context.Context, kind string, payload any, blob []byte) (Response, error) // POST /v1/upgrade; stale_seq retried once with the same blob
  type api.AgentUpgradeView struct{ Upgraded bool; From, To string } // json: upgraded, from, to
  func (c *apiclient.Client) UpgradeAgent(ctx context.Context, target string) (api.AgentUpgradeView, error) // POST /api/targets/{id}/agent/upgrade
  ```
- The server route:
  1. The target must be paired.
  2. It takes the target slot (`claimTargetOp`) and runs as a `criticalOp`.
  3. `agent.info` gives `Arch`, then `s.loadAgent(arch)` (seam; default `agentbin.Load` bytes).
  4. `Upload` the agent with `Version: buildinfo.Version()`.
  5. When `Upgraded`, it polls `agent.info` up to 30 s until `Version == To`, else 504 `timeout`.
- `jumpgate hosts upgrade NAME` prints `NAME: agent v0.6.0 -> v0.6.1` or `NAME: agent already v0.6.1`.
- In `rejectionHints`, the hints for `unknown_kind` and `unknown_plan` and `CodeUnknownPlan`'s registry hint say "run `jumpgate hosts upgrade NAME`" instead of "re-pair … to upgrade".

- [ ] **Step 1: Write the failing tests**

```go
func TestUploadSignsLikeDoAndSendsTheBlob(t *testing.T) // fake agent socket: header decodes to an Envelope signed by the controller with kind agent.upgrade; body sha == payload sha; signed receipt verified
func TestUploadRetriesStaleSeqWithTheSameBlob(t *testing.T) // first answer stale_seq LastSeq 9; second request has seq 10 and identical body
func TestAgentUpgradeSendsTheAgentForTheBoxArch(t *testing.T) // agent.info Arch arm64 -> loadAgent seam called with "arm64"; response {upgraded:true, from, to}
func TestAgentUpgradeRefusesAnUnpairedTarget(t *testing.T) // -> code not_paired
func TestHostsUpgradePrintsFromAndTo(t *testing.T) // fake API -> "box: agent v0.6.0 -> v0.6.1"
func TestUpgradeHintsPointAtHostsUpgrade(t *testing.T) // RejectionHint(intent.ReasonUnknownKind) contains "jumpgate hosts upgrade"
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/agentclient/ ./internal/server/ ./cmd/jumpgate/ ./internal/api/ -run 'Upload|AgentUpgrade|HostsUpgrade|UpgradeHints' -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./internal/agentclient/ ./internal/server/ ./cmd/jumpgate/ ./internal/api/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/agentclient internal/server internal/api internal/apiclient cmd/jumpgate
git commit -m "feat(hosts): hosts upgrade sends the controller's agent over the pairing"
```

---

### Task 19: Self-only controllers (D11)

**Files:**
- Modify: `internal/agent/policy.go` (`SignerEntry.Scope`, `CheckScope`), `internal/agent/agent.go` (`Handle`), `internal/agent/listen.go` (`ConnContext` stores the cred), `cmd/jumpgate/cli_agent.go` (`enroll --scope`), `internal/intent/kinds.go` (`ReasonScopeLocal`), `internal/api/errors.go` (`self_only`, hint for `scope_local`), `internal/bootstrap/bootstrap.go` (`Options.Scope`), `cmd/jumpgate/pair_local.go` (pass `Scope`), `internal/server/pair.go`, `internal/server/system.go` (`PairingInfo.Scope`)
- Test: `internal/agent/policy_test.go`, `cmd/jumpgate/cli_agent_test.go`, `internal/bootstrap/bootstrap_test.go`, `internal/server/pair_test.go`

**Interfaces:**
- Consumes: `peercred.Cred` (Task 1); `config.Controller.Scope` (Task 9).
- Produces:
  ```go
  type agent.SignerEntry struct{ /* … */ Scope string `json:"scope,omitempty"` } // "" | "local"
  func (p Policy) CheckScope(controller eip712.Address, peer peercred.Cred, havePeer bool) error
  const intent.ReasonScopeLocal = "scope_local" // hint: "this controller is enrolled on the box for local use only (--self-only); it cannot reach the box through a tunnel"
  func agentEnroll(configDir, socketPath, address, tier, label string, localUID int, scope string) error
  type bootstrap.Options struct{ /* … */ Scope string } // "local" appends " --scope local" to the enroll command
  const api.CodeSelfOnly Code = "self_only" // hint: "this controller was set up with --self-only and pairs only this machine; to pair other boxes, set up a controller without --self-only"
  ```
- `CheckScope` passes for an unscoped signer. A `local` one passes only when `havePeer && (peer.UID == 0 || peer.UID ∈ LocalUIDs)`.
- `Handle` calls it right after `Authorise`, with the cred `Serve` stored in ctx.
- `enroll --scope local` on an existing unscoped entry narrows it to `local`. An enroll without `--scope` never clears a scope.
- `handlePair` refuses a non-local target when `Controller.Scope == "self"`: 409 `self_only`.

- [ ] **Step 1: Write the failing tests**

```go
func TestCheckScopeLocalAdmitsRootAndLocalUIDs(t *testing.T) // Policy{LocalUIDs: [991], Signers: [{addr, routine, Scope:"local"}]}; Cred{UID:0} ok; Cred{UID:991} ok
func TestCheckScopeLocalRefusesTheTunnelUser(t *testing.T)   // Cred{UID:997, GIDs:[jumpgateGID]} -> reject scope_local
func TestCheckScopeLocalRefusesWithoutAPeer(t *testing.T)    // havePeer false -> scope_local
func TestUnscopedSignerIgnoresThePeer(t *testing.T)          // Scope "" with Cred{UID:997} -> nil
func TestEnrollScopeLocalNarrowsButNeverWidens(t *testing.T) // enroll unscoped, then --scope local -> "local"; then enroll without --scope -> still "local"
func TestBootstrapPassesScopeLocal(t *testing.T)             // Options{Local: true, LocalUID: 991, Scope: "local"} -> enroll command contains "--scope local"
func TestPairRefusesARemoteBoxUnderSelfOnly(t *testing.T)    // config Controller.Scope "self"; ssh target -> 409 self_only, message contains "--self-only"
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/agent/ ./cmd/jumpgate/ ./internal/bootstrap/ ./internal/server/ -run 'CheckScope|UnscopedSigner|EnrollScope|PassesScope|SelfOnly' -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass, here and on Linux**

Run: `go test ./internal/... ./cmd/jumpgate/ -count=1 && scripts/test-linux.sh ./internal/agent/ ./internal/bootstrap/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal cmd/jumpgate
git commit -m "feat(agent): self-only controllers are refused through a tunnel"
```

---

### Task 20: "trusted by N controllers" in the TUI host view (D2)

**Files:**
- Modify: `internal/intent/kinds.go` (`AgentInfo.Controllers`), `internal/agent/dispatch.go`, `internal/api/misc.go` (`AgentCheck.Controllers`), `internal/server/uiprefs.go` (fleet check), `internal/tui/host.go`, `internal/tui/host_security.go`
- Test: `internal/agent/agent_test.go`, `internal/tui/host_test.go`

**Interfaces:**
- Produces:
  ```go
  type intent.AgentInfo struct{ /* … */ Controllers int `json:"controllers"` } // routine signers in policy
  type api.AgentCheck struct{ /* … */ Controllers int `json:"controllers"` }
  ```
- The host screen asks `CheckAgent` when its Security tab opens. Above the FIREWALL heading it shows ` trusted by N controllers` when N > 1, and nothing otherwise.

- [ ] **Step 1: Write the failing tests**

```go
func TestAgentInfoCountsRoutineSigners(t *testing.T) // two routine + one approval enrolled -> AgentInfo.Controllers == 2
func TestSecurityTabShowsTrustedByWhenMoreThanOne(t *testing.T) // tuitest.Fake CheckAgent -> Controllers 2: view contains "trusted by 2 controllers"; Controllers 1: not contains "trusted by"
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/agent/ ./internal/tui/ -run 'CountsRoutineSigners|TrustedBy' -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement** the Produces block.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./internal/agent/ ./internal/tui/ ./internal/server/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal
git commit -m "feat(tui): show how many controllers a box trusts"
```

---

### Task 21: End-to-end: the service, attach, unlock, tunnel login, self-only, pull and upgrade

**Files:**
- Create: `scripts/e2e-service.sh`
- Modify: `internal/bootstrap/e2e_test.go` (`TestE2EScopeLocalRefusesAGroupPeer`), `scripts/e2e-agent.sh` (run it as user `jumpgate`)

**Interfaces:**
- Consumes: everything above. The controller under test is built as goreleaser builds it: `env GZIP=1 VERSION=v0.0.0-e2e bash scripts/build-agents.sh internal/agentbin/embedded && CGO_ENABLED=0 GOOS=linux go build -tags embedagents -ldflags "-X github.com/valve-tech/jumpgate/internal/buildinfo.version=v0.0.0-e2e" ./cmd/jumpgate`.

- [ ] **Step 1: Write the e2e** (bash, `set -euo pipefail`, containers A and B from `scripts/e2e/Dockerfile` on one docker network, `KEEP=1` as in `e2e-local.sh`). Each check prints `ok <name>` or exits 1.
  - **Setup in A:** `jumpgate setup install-service`, then as alice `printf 'pw\n' | jumpgate setup server --stdin --name a`.
  - **Attach and admission:**
    1. `status-answers`: as alice, `sg jumpgate-admin -c 'jumpgate status a'` exits 0.
    2. `bob-refused`: as bob, `jumpgate hosts list` exits nonzero with `usermod -aG jumpgate-admin`, and `/home/bob/.jumpgate/run` does not exist.
  - **Locking:**
    1. `restart-locks`: after `systemctl restart jumpgate`, `jumpgate status a` fails with code `locked`.
    2. `unlock`: after `printf 'pw\n' | jumpgate unlock --stdin` it answers again.
    3. `seal-host-key`: `sudo jumpgate vault seal --allow-host-key`, then `systemctl restart jumpgate`, then `status` answers with no unlock.
  - **Tunnel login:**
    1. `login-tunnel`: from B, `ssh -L 8799:127.0.0.1:8799 alice@A` (alice's key installed in setup), with the code from `jumpgate open --print` run as alice in A. `curl -s -o /dev/null -w '%{http_code}' 'http://127.0.0.1:8799/login?code=…'` == 200.
    2. `login-foreign-tunnel`: the same code shape through bob's tunnel == 403.
  - **Pull, upgrade and invite:**
    1. `pull-pair`: in B, `install.sh --agent` from the Task 13 `relsrv` with the invite's `--controller` and `--transport-key`. In A, `hosts add b --ssh jumpgate@B --installed <printed>`, then `status b` returns.
    2. `pull-mismatch`: `--installed 0x000…01` fails with `installed_mismatch`.
    3. `upgrade`: rebuild the controller with version `v0.0.1-e2e` and restart A (unlock), then `jumpgate hosts upgrade b` prints `-> v0.0.1-e2e`, and B's `/usr/local/lib/jumpgate/jumpgate` sha256 matches the new embedded agent.
  - **Self-only:** `self-only`: a fresh A2 with `setup server --self-only`. `hosts add x --ssh root@B` fails with `self_only`, and `hosts invite x` fails with `self_only`.
  - **Scope with a group peer:** `TestE2EScopeLocalRefusesAGroupPeer` runs in two phases inside the box:
    1. As alice: pair locally with `Scope: "local"`, writing the controller key and the agent address under `/opt/jumpgate-e2e` (0644, test only).
    2. As user `jumpgate`: a signed `status.read` gets the rejection `scope_local`. The `jumpgate` uid is exactly what a tunnelled connection arrives as.

- [ ] **Step 2: Run it to verify the new pieces fail before the last fixes land**

Run: `scripts/e2e-service.sh`
Expected: at least one `FAIL <name>` if any earlier task is incomplete. Otherwise every line is `ok …`; go straight to Step 4.

- [ ] **Step 3: Fix what fails** in the owning task's files, adding a unit test there that reproduces it.

- [ ] **Step 4: Run both distros and the agent e2e**

Run: `scripts/e2e-service.sh && BASE=ubuntu:24.04 scripts/e2e-service.sh && scripts/e2e-agent.sh && BASE=ubuntu:24.04 scripts/e2e-agent.sh`
Expected: every check `ok`; exit 0 four times.

- [ ] **Step 5: Commit**

```bash
git add scripts/e2e-service.sh scripts/e2e-agent.sh internal/bootstrap/e2e_test.go
git commit -m "test(e2e): service, attach, unlock, tunnel login, pull, upgrade and self-only"
```

---

### Task 22: Release wiring, then the hosting gate (MANUAL)

**Files:**
- Modify: `.goreleaser.yaml` (`release.extra_files` and `checksum.extra_files`: `./packaging/install/install.sh`), `.github/workflows/release.yml` (new `packages` job)
- Test: `packaging/packaging_test.go`

**Interfaces:**
- Consumes: `scripts/build-pkg-repo.sh` (Task 15).
- Produces:
  - **`packages` job:**
    - `needs: [provenance]`, `if: startsWith(github.ref, 'refs/tags/v') && vars.PKG_REPO_ENABLED == 'true'`, `environment: packages`, `runs-on: ubuntu-24.04`, `permissions: contents: read`.
    - Steps:
      1. Download `*.deb` and `*.rpm` from the release with `gh release download`.
      2. `apt-get install -y aptly createrepo-c rpm gnupg`.
      3. `export GNUPGHOME=$(mktemp -d /dev/shm/gnupg.XXXXXX)`, then import `PKG_SIGNING_KEY` from the environment through stdin, and write `PKG_SIGNING_PASSPHRASE` to a 0600 file in the same tmpfs.
      4. Run `scripts/build-pkg-repo.sh`.
      5. Sync to R2 with `aws s3 sync --endpoint-url https://4d950372162cef595726db63c21035ab.r2.cloudflarestorage.com` and secrets `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY`. Order: `apt/pool/` and `rpm/*.rpm` first, then `apt/dists/`, `rpm/repodata/`, `jumpgate.asc`, `jumpgate.repo`, so no client ever sees an index naming a package that is not there yet.
      6. An `if: always()` step that runs `rm -rf "$GNUPGHOME"`.
    - Actions are pinned by commit SHA, as the rest of `release.yml` is.
  - `install.sh` becomes a release asset, so the `checksums` and `provenance` jobs cover it.

- [ ] **Step 1: Write the failing test** (append to `packaging/packaging_test.go`)

```go
func TestInstallScriptIsAReleaseAsset(t *testing.T) // .goreleaser.yaml contains "./packaging/install/install.sh" under both release.extra_files and checksum.extra_files
func TestPackagesJobIsGatedAndScoped(t *testing.T)  // release.yml contains "  packages:", "needs: [provenance]", "environment: packages", "vars.PKG_REPO_ENABLED == 'true'", "4d950372162cef595726db63c21035ab.r2.cloudflarestorage.com", "rm -rf \"$GNUPGHOME\""; no "PKG_SIGNING_KEY" appears on a `run:` command line as an argument
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./packaging/ -run 'InstallScriptIsAReleaseAsset|PackagesJob' -count=1`
Expected: FAIL.

- [ ] **Step 3: Add** the goreleaser lines and the `packages` job.

- [ ] **Step 4: Verify**

Run: `go test ./packaging/ -count=1 && goreleaser check && docker run --rm -v "$PWD":/repo -w /repo rhysd/actionlint:1.7.7 .github/workflows/release.yml`
Expected: PASS; no goreleaser errors; no actionlint findings.

- [ ] **Step 5: Commit**

```bash
git add .goreleaser.yaml .github/workflows/release.yml packaging/packaging_test.go
git commit -m "feat(release): publish install.sh and a gated packages job for pkg.jumpgate.app"
```

- [ ] **Step 6: MANUAL GATE: stop and ask the user before any of the following.** Each item creates public or billable infrastructure. List them, wait for an explicit OK, then do only what was approved:
  1. Create R2 bucket `jumpgate-pkg` in Cloudflare account `4d950372162cef595726db63c21035ab` (valaxyblackdev). Check `wrangler whoami` first and set `CLOUDFLARE_ACCOUNT_ID`.
  2. Attach the custom domain `pkg.jumpgate.app` to the bucket (the jumpgate.app zone is in the same account).
  3. Create an R2 API token scoped to `jumpgate-pkg` only (object read and write).
  4. Generate the repo signing key offline (`gpg --quick-gen-key 'jumpgate archive <…>' ed25519 sign 2y`, with a passphrase). Commit its public half as `packaging/repo/jumpgate-archive-keyring.asc`, and record the fingerprint for the docs.
  5. Create the GitHub environment `packages` on `jumpgate-tech/jumpgate-app`, with a required maintainer reviewer and the secrets `PKG_SIGNING_KEY`, `PKG_SIGNING_PASSPHRASE`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`. Then set the variable `PKG_REPO_ENABLED=true`. Use `GH_TOKEN=$(gh auth token -u jumpgatedev)`.
  6. Copy `packaging/install/install.sh` into the landing repo (`~/Documents/jumpgate-tech/github/landing`) as `/install.sh` and deploy with `npx wrangler@4 pages deploy … --project-name jumpgate --branch main`.
  7. Push to GitHub and tag a release only with the user's say-so.

  After approval, verify:
  - `curl -fsSL https://pkg.jumpgate.app/jumpgate.asc | gpg --show-keys` shows the recorded fingerprint;
  - the `sha256sum` of `curl -fsSL https://jumpgate.app/install.sh` equals the release asset's `checksums.txt` line;
  - on a clean debian:12, `apt install jumpgate` with `jumpgate.sources` succeeds.

---

## Docs pathway (for the docs-system plan)

The steps and scenes below belong to the docs-system plan. Each lands with the code it covers (spec §6, rollout 8). This plan writes none of them.

| Step id | Scene | Lands after |
|---|---|---|
| `server-choose-role` | none (hand-made diagram with sidecar) | Task 13 |
| `server-install` | `tui install-controller-done` (terminal transcript) | Tasks 13 and 22; the apt and dnf variant pins the fingerprint from the Task 22 gate |
| `server-key-vault` | `tui setup-server-passphrase` (shows `api.CustodyWarning`) | Task 9 |
| `server-start-service` | `tui service-locked-banner` | Tasks 9 and 10 |
| `server-unlock` | `tui unlock-prompt`; covers the optional `vault seal --tpm2` and the idle auto-lock setting (vault Task 19) | Task 11 and vault Task 19 |
| `server-pair-self` | `tui hosts-self-paired` | Task 9 |
| `server-tui-over-ssh` | `tui fleet` | Task 3 |
| `server-web-tunnel` (optional) | `app login-via-tunnel` | Task 4 |
| `server-add-agent-box` | `tui hosts-invite`; also covers `hosts add --installed` and `hosts upgrade` | Tasks 16 to 18 |
| `server-backup-recover` | none | vault plan |

Steps `server-key-vault`, `server-pair-self` and `server-add-agent-box` `require` the existing `keys-init` and `pair-box` steps where those apply.

---

## Self-review

1. **Spec coverage.**

   | Spec item | Task(s) |
   |---|---|
   | D1 roles | 13, 16 |
   | D2 | 20 |
   | D3 vault default, plain refused without `--insecure-plain-key`, start warning | 9, 2 |
   | D4 locked state and unlock (vault plan) | 0 |
   | D4 TPM seal | 11 |
   | D4 "after a reboot" banner and home line (vault plan) | consumed |
   | D5 unit | 8 |
   | D5 `prctl` | 2 |
   | D5 ProtectHome check | 5, 7 |
   | D6 | 8 |
   | D7 | 2, 3 |
   | D8 | 4 |
   | D9 | 6, 9 |
   | D10 warnings: install.sh | 13 |
   | D10 warnings: setup | 9 |
   | D10 warnings: host-add | 7 |
   | D10 warnings: TUI unlock screen (one line in the vault plan's prompt) | not separately pinned: see Gaps |
   | D11 | 19, 9 |
   | D12 | 6, 16 |
   | D12 upgrades, revised by Q3 | 17, 18 |
   | D13 | 13, 16 |
   | D14 | 14, 15, 22 |
   | D15 | 14 (postinst), 10 (restart offer) |
   | D16 | needs no code; Task 21's self-pair proves the paths coexist |
   | Error table | 3, 6, 11, 13, 14, 19 |
   | Testing section | 13, 15, 21 and each task's unit tests |
   | Resolved Q1 (no fallback) | 3 |
   | Resolved Q2 | vault Task 19 (Task 12 here is a stub) |
   | Resolved Q3 | 17, 18 |
   | Resolved Q4, Q5 | nothing to build |
   | Resolved Q7 | 22 |

   **Gaps:** the D10 line in the TUI unlock prompt lives in the vault plan's screen. It is not pinned here. When Task 0 runs, add `api.CustodyWarning` to that prompt in system mode, with a test that the view contains it.

2. **Step scan.** Every code step names its signature and its file. Bodies appear only for exact copy: the unit lines, `Sysusers`, the drop-in, the `.repo` file and the nfpm block. Test steps carry their assertions.

3. **Type consistency.** These names match across tasks:
   - `pairingFacts`, `factsFromServer`, `pairLocalForeground(ctx, out, f)` (Tasks 6, 7, 9);
   - `api.PairingInfo` and `apiclient.PairingInfo` (Tasks 5, 6, 7, 16, 19);
   - `peercred.Cred`, `peercred.Of`, `peercred.GroupID` (Tasks 1, 2, 19);
   - `daemon.Info.System`, `ServiceUID`, `Lock` (Tasks 3, 4, 10);
   - `sysunit.SealDropInPath` and `SealCredPath` (Tasks 8, 11, 14);
   - `intent.KindAgentUpgrade` and `AgentInfo.Arch` (Tasks 17, 18);
   - `api.CustodyWarning` (Tasks 7, 9, 13);
   - `config.Controller.Scope` (Tasks 9, 19).

   Task 5 originally named the route `/api/system/pairing`. It became `/api/pairing-info` (both modes) because `hosts invite` needs it per-user too, and every reference now uses `PairingInfo`.

4. **Review Focus.** Five lines, each pinned to a named test in its owning task.

5. **Proportion.** The plan is longer than the spec, because the spec spans seven subsystems and 22 tasks. Code blocks are signatures, test names and assertions. The only full bodies are the fixed-copy artifacts above.
