# Foundations (sub-project 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `jumpgate` with a signed-intent agent: pair a box over SSH, then read status, disk, endpoints, firewall and logs and start/stop/restart services through EIP-712-signed intents that the box checks against its own policy — on remote boxes and on the local machine.

**Architecture:** One binary, `cmd/jumpgate`, plays controller and agent. The controller is the existing HTTP server (now also on a unix socket, single-instance, detachable) and holds the signing key; CLI subcommands talk to it. The agent is `jumpgate agent run`, a root systemd service on a unix socket that only accepts typed intents, verifies signatures, replay and policy, then runs the existing `internal/ops` / `internal/monitor` code with a local executor. Remote controllers reach the socket through an SSH `direct-streamlocal` channel as an unprivileged `jumpgate` user.

**Tech Stack:** Go 1.25 module (local toolchain 1.26), `golang.org/x/crypto` (`ssh`, `ssh/agent`, `ssh/knownhosts`, `sha3`), `github.com/decred/dcrd/dcrec/secp256k1/v4` (new), `github.com/gliderlabs/ssh` (tests), systemd, OpenSSH.

**Spec:** `docs/superpowers/specs/2026-10-02-foundations-design.md` (parent: `docs/superpowers/specs/2026-10-02-controller-agent-architecture-design.md`). Read both before Task 0.

## Preconditions

- **Do not start Task 0 until these branches are merged to `main`:** `fix/review-subproject-0`, `fix/relay-streams`, `fix/relay-limits`, `fix/billing-hardening`, `fix/server-hardening`, `fix/vpn-reprovision`, `fix/small-correctness`. Task 0 rewrites every import path; starting earlier guarantees conflicts with all of them. Sub-project 0's Origin check (`fix/review-subproject-0`) is a hard dependency of Task 16.
- Branch for this plan: `feat/foundations` from `main` after those merges.
- Baseline before Task 0: `go test ./...` fully green (the server-hardening branch fixes the one pre-existing failure). Record the result.

## Global Constraints

- New dependency allowed: `github.com/decred/dcrd/dcrec/secp256k1/v4` only. No go-ethereum, no keyring library, no TUI library in this sub-project.
- The agent binary is built `CGO_ENABLED=0` for `linux/amd64` and `linux/arm64`; nothing in `internal/agent`, `internal/intent`, `internal/eip712`, `internal/signer` (except build-tagged keychain code) may use cgo.
- EIP-712 domain is exactly `{name: "jumpgate", version: "1"}` — no `chainId`, no `verifyingContract`.
- Intent lifetime: client sets `expiry = issuedAt + 120`; agent rejects `expiry − issuedAt > 300`; clock-skew allowance 60 s each side.
- Agent request body cap 1 MiB; `logs.read` `n` default 200, max 2000.
- On-box paths: socket `/run/jumpgate/agent.sock` (`root:jumpgate`, `0660`); state `/var/lib/jumpgate/`; config `/etc/jumpgate/` (`0700` dir, `0600` files); binary `/usr/local/lib/jumpgate/jumpgate`; unit `jumpgate-agent.service`; sshd drop-in `/etc/ssh/sshd_config.d/50-jumpgate.conf`.
- Controller paths: `~/.jumpgate/` (`0700`), `~/.jumpgate/config.json` (`0600`), `~/.jumpgate/run/` (`0700`), `~/.jumpgate/known_hosts`, `~/.jumpgate/ssh/jumpgate_ed25519`, `~/.jumpgate/keys/controller.key`, `~/.jumpgate/agents/`.
- **On-box names are NOT renamed in this sub-project.** Units (`valve-node-app-exec.service` …), `/var/lib/valve-node-app`, container names, the `valve-node-app` service user and `VALVE_*` env placeholders stay as they are; their migration is a durable-job intent in sub-project 2 (see Task 19's spec update).
- Every commit message ends with the line `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Comments follow the codebase's style: full sentences explaining *why*, at the density of the surrounding code.

## Review Focus

1. **A truncated or corrupt `/var/lib/jumpgate/replay.json`** (agent crashed mid-write, disk full). Expected: the agent refuses every intent with `replay_state` and logs how to reset; it never silently starts from empty state, which would reopen replay of captured intents. Test lives in Task 11.
2. **Two controllers paired to the same box acting concurrently.** Expected: each has its own sequence; one controller's traffic never makes the other's intents `stale_seq` or `busy`. Test lives in Task 11 (`TestControllersAreIndependent`); the busy rule in Task 12 is keyed by controller for the same reason.
3. **A target whose setup never ran (no `node.json` / no Wire).** Expected: `status.read`, `disk.read`, `endpoints.read`, `firewall.read`, `logs.read`, `service.action` are rejected with `not_set_up`, never probed against an empty `DataDir`. `agent.info` still works. Test lives in Task 12.
4. **Re-pairing a box, or pairing a second controller.** Expected: `authorized_keys` and `policy.json` gain an entry; existing entries survive; the agent key is not regenerated. Test lives in Task 14.
5. **A stale `server.json` whose pid now belongs to an unrelated process.** Expected: `jumpgate` never signals that pid and never trusts the file; it starts a fresh server. Test lives in Task 15.

## File Structure

Created:

| Path | Responsibility |
|---|---|
| `internal/eip712/eip712.go` | Typed-data model, `encodeType`, `HashStruct`, `Digest` |
| `internal/eip712/eip712_test.go` | Spec vectors, error cases |
| `internal/signer/signer.go` | `Signer` interface, `Address`, `Signature`, `Recover`, `Verify` |
| `internal/signer/key.go` | secp256k1 private key wrapper, address derivation, signing |
| `internal/signer/keyfile.go` | `KeyFile` signer and `GenerateKeyFile` |
| `internal/signer/keychain.go` | `Keychain` store (macOS `security -i`, Linux `secret-tool`) |
| `internal/signer/onepassword.go` | `OnePassword` store (`op`) |
| `internal/signer/store.go` | `Open(store, ref)` / `Create(store, ref)` dispatch |
| `internal/intent/intent.go` | `Intent`, `Receipt` messages, EIP-712 schemas, envelope types |
| `internal/intent/kinds.go` | Kind constants, payload/result structs, reason codes |
| `internal/agent/policy.go` | `Policy` load/save, requirement check |
| `internal/agent/replay.go` | Persisted last-seq + nonce window |
| `internal/agent/agent.go` | `Agent`: verify → persist → dispatch → sign receipt |
| `internal/agent/dispatch.go` | Kind → `ops`/`monitor`/journal calls |
| `internal/agent/listen.go` | Unix listener, `SO_PEERCRED` gate, HTTP handler |
| `internal/agent/peercred_linux.go`, `peercred_darwin.go`, `peercred_other.go` | Peer uid/gid lookup |
| `internal/agent/errors.go` | `Reject` |
| `internal/agentclient/client.go` | Build/sign intents, verify receipts, seq resync |
| `internal/agentclient/dial.go` | Local socket, SSH streamlocal, jump chain |
| `internal/bootstrap/bootstrap.go` | Steps 1–8 of pairing |
| `internal/executor/sudo.go` | `sudo -n` executor wrapper (beside `writeFileCmd`, which it reuses) |
| `internal/executor/dial.go`, `hostkey_decider.go`, `procgroup.go` | Handshake deadline + ssh-agent + jump hosts; strict host keys; remote process groups |
| `internal/filelock/` | Cross-process advisory locks (config, server single-instance) |
| `internal/server/controller.go` | Controller key, transport key, agent binary lookup, seq store, agent target |
| `internal/daemon/daemon.go` | Lock, discovery file, find-or-start, stop |
| `internal/server/intent.go` | `POST /api/targets/{id}/intent/{kind}` |
| `internal/server/pair.go` | `POST /api/targets/{id}/pair` (SSE) |
| `cmd/jumpgate/cli.go` | Subcommand dispatch |
| `cmd/jumpgate/cli_hosts.go`, `cli_intents.go`, `cli_keys.go`, `cli_agent.go`, `cli_serve.go` | Subcommands |
| `cmd/jumpgate/web/embed.go` | Embedded UI shared by every entry point |
| `scripts/build-agents.sh` | Cross-build agent binaries + `SHA256SUMS` |
| `scripts/e2e-agent.sh`, `internal/bootstrap/e2e_test.go` | Container end-to-end (build tag `e2e`) |

Modified: `go.mod`; `internal/config/config.go` (dir, lock, `Update`, `Target.Agent`, `Controller`); `internal/executor/{executor,ssh,hostkey,local}.go`; `internal/catalog/units.go` (+ `datadir.go`); `internal/setup/steps.go`; `internal/ops/ops.go`; `internal/monitor/monitor.go` (`PollOnce`); `internal/server/{server,api}.go`; `cmd/valve-node-app` → `cmd/jumpgate` (moved).

---

### Task 0: Retire the valve-node-app name on the controller side

**Files:**
- Move: `cmd/valve-node-app/` → `cmd/jumpgate/` (`git mv`)
- Modify: `go.mod` (module path), every `.go` import, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.goreleaser.yaml`, `.gitignore`, `cmd/jumpgate/build-macos-app.sh`, `README.md`
- Modify: `internal/config/config.go` (`Dir`, migration), `internal/server/server.go` (`cookieName`), tray flag `__VALVE_TRAY__` in `cmd/jumpgate/tray.go` and `cmd/jumpgate/web/src/**`
- Test: `internal/config/dir_test.go` (new)

**Interfaces:**
- Produces: module path `github.com/valve-tech/jumpgate`; `config.Dir()` returns `~/.jumpgate`; `config.MigrateLegacyDir() (moved bool, err error)`; cookie name `jumpgate_token`; tray global `window.__JUMPGATE_TRAY__`.
- Out of scope (stay unchanged): every on-box name listed in Global Constraints, `VALVE_API_KEY`/`VALVE_KEY`/`VALVE_CERT_URL` (rendered into on-box units and eRPC config), and `valve-node-app.explain-consent` (a localStorage key; renaming it silently re-asks every user for consent — leave it).

- [ ] **Step 1: Write the failing test for the config directory and its migration**

```go
// internal/config/dir_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirIsJumpgate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".jumpgate"); got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
}

// An existing install keeps its targets, keys and known_hosts: the old
// directory is moved once, and a pointer file is left so a reader of the old
// path learns where it went.
func TestMigrateLegacyDirMovesOnce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	legacy := filepath.Join(home, ".valve-node-app")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.json"), []byte(`{"targets":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	moved, err := MigrateLegacyDir()
	if err != nil || !moved {
		t.Fatalf("MigrateLegacyDir() = %v, %v; want true, nil", moved, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".jumpgate", "config.json")); err != nil {
		t.Fatalf("config not moved: %v", err)
	}
	pointer, err := os.ReadFile(filepath.Join(legacy, "MOVED"))
	if err != nil || len(pointer) == 0 {
		t.Fatalf("no pointer file left behind: %v", err)
	}

	moved, err = MigrateLegacyDir()
	if err != nil || moved {
		t.Fatalf("second MigrateLegacyDir() = %v, %v; want false, nil", moved, err)
	}
}

// Both directories existing means someone already started fresh; merging
// two configs silently would lose one of them, so it is an error to resolve
// by hand.
func TestMigrateLegacyDirRefusesWhenBothExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, d := range []string{".valve-node-app", ".jumpgate"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, d, "config.json"), []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := MigrateLegacyDir(); err == nil {
		t.Fatal("want an error when both directories hold a config")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/config/ -run 'Dir|MigrateLegacy' -v`
Expected: FAIL — `Dir() = ".../.valve-node-app"` and `undefined: MigrateLegacyDir`.

- [ ] **Step 3: Implement `Dir` and `MigrateLegacyDir`**

In `internal/config/config.go`, replace `Dir`:

```go
// dirName is the controller's state directory under $HOME.
const dirName = ".jumpgate"

// legacyDirName is where releases before the rename kept the same state.
const legacyDirName = ".valve-node-app"

// Dir returns the directory jumpgate's local state lives in (~/.jumpgate),
// without creating it.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve home directory: %w", err)
	}
	return filepath.Join(home, dirName), nil
}

// MigrateLegacyDir moves ~/.valve-node-app to ~/.jumpgate once, and leaves a
// MOVED pointer file in the old place. It is a rename, not a copy, so secrets
// (provider keys, VPN private keys) never exist twice on disk. It reports
// whether it moved anything. Both directories holding a config is an error:
// merging two configs silently would lose one of them.
func MigrateLegacyDir() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("config: resolve home directory: %w", err)
	}
	legacy := filepath.Join(home, legacyDirName)
	current := filepath.Join(home, dirName)

	if _, err := os.Stat(filepath.Join(legacy, configFileName)); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("config: inspect %s: %w", legacy, err)
	}
	if _, err := os.Stat(filepath.Join(current, configFileName)); err == nil {
		return false, fmt.Errorf("config: both %s and %s hold a config.json; keep one and remove the other", legacy, current)
	}
	// An empty ~/.jumpgate (created by something that never wrote a config)
	// would make the rename fail; remove it only if it is empty.
	_ = os.Remove(current)
	if err := os.Rename(legacy, current); err != nil {
		return false, fmt.Errorf("config: move %s to %s: %w", legacy, current, err)
	}
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		return true, fmt.Errorf("config: recreate %s for the pointer file: %w", legacy, err)
	}
	note := []byte("jumpgate moved this directory to " + current + "\n")
	if err := os.WriteFile(filepath.Join(legacy, "MOVED"), note, 0o600); err != nil {
		return true, fmt.Errorf("config: write pointer file: %w", err)
	}
	return true, nil
}
```

Update the doc comments on `Load` and `Save` from `~/.valve-node-app/config.json` to `~/.jumpgate/config.json`. Note `MigrateLegacyDir` must leave a directory containing only `MOVED`, so its own check (`config.json` absent) makes the second call a no-op.

- [ ] **Step 4: Run the config tests**

Run: `go test ./internal/config/ -v -run 'Dir|MigrateLegacy'`
Expected: PASS (3 tests).

- [ ] **Step 5: Move the command and rewrite the module path**

```bash
git mv cmd/valve-node-app cmd/jumpgate
go mod edit -module github.com/valve-tech/jumpgate
git grep -l 'github.com/valve-tech/valve-node-app' -- '*.go' | xargs sed -i '' 's#github.com/valve-tech/valve-node-app#github.com/valve-tech/jumpgate#g'
sed -i '' 's#github.com/valve-tech/valve-node-app#github.com/valve-tech/jumpgate#g' .goreleaser.yaml internal/buildinfo/buildinfo.go
sed -i '' 's#cmd/valve-node-app#cmd/jumpgate#g' .github/workflows/ci.yml .github/workflows/release.yml .goreleaser.yaml .gitignore cmd/jumpgate/build-macos-app.sh
sed -i '' 's#^/valve-node-app$#/jumpgate#' .gitignore
sed -i '' 's#EXE_NAME="valve-node-app"#EXE_NAME="jumpgate"#' cmd/jumpgate/build-macos-app.sh
```

(`sed -i ''` is BSD/macOS syntax; on Linux use `sed -i`.)

- [ ] **Step 6: Rename the controller-side identifiers**

- `internal/server/server.go`: `const cookieName = "jumpgate_token"`. An existing browser session simply re-authenticates through the printed `?token=` URL; no migration is needed.
- Tray flag: replace `__VALVE_TRAY__` with `__JUMPGATE_TRAY__` in `cmd/jumpgate/tray.go` and every file under `cmd/jumpgate/web/src` (`git grep -l __VALVE_TRAY__`).
- `cmd/jumpgate/main.go`: log prefixes `valve-node-app:` → `jumpgate:`; the package comment's first line → `// Command jumpgate sets up and monitors …`; call `config.MigrateLegacyDir()` before `config.Load()` and print one line when it moved something:

```go
	if moved, err := config.MigrateLegacyDir(); err != nil {
		log.Fatalf("jumpgate: %v", err)
	} else if moved {
		fmt.Fprintln(os.Stderr, "jumpgate: moved ~/.valve-node-app to ~/.jumpgate")
	}
```

- `internal/server/api.go` default host-key file (`api.go:598-609`): it derives from `config.Dir()`, so it follows automatically — verify with `git grep -n known_hosts internal/server`.
- Rebuild the embedded UI so `dist` matches the source: `cd cmd/jumpgate/web && npm ci && npm run build` (CI checks `git diff --exit-code cmd/jumpgate/web/dist`). If `npm` is unavailable, stop and report it — do not hand-edit `dist`.
- `README.md`: title and prose `valve-node-app` → `jumpgate`; keep one sentence noting the old name for people searching for it.

- [ ] **Step 7: Verify nothing controller-side still carries the old name**

Run:
```bash
git grep -n 'valve-node-app\|valve_node\|__VALVE_TRAY__' -- ':!docs' ':!*.md' ':!*package-lock.json' ':!cmd/jumpgate/web/dist'
```
Expected: only on-box names — unit names, `/var/lib/valve-node-app`, container names, `catalog.ServiceUser`, `valve-node-app.explain-consent`, `.local/share/ca-certificates/valve-node-app-*`, test fixtures asserting those. Any other hit is a miss: fix it.

- [ ] **Step 8: Full verification**

Run: `gofmt -l . ; go vet ./... ; go test ./... ; go build ./cmd/jumpgate`
Expected: no gofmt output, vet clean, all tests pass, binary builds.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "chore: rename the controller to jumpgate

Module path, cmd/jumpgate, ~/.jumpgate (moved once from ~/.valve-node-app
with a pointer file), session cookie and tray flag. On-box names (units,
/var/lib/valve-node-app, containers, the service user) are untouched: renaming
them stops a live node and is a durable-job migration in sub-project 2.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 1: Validate `DataDir` everywhere it is used

**Files:**
- Create: `internal/catalog/datadir.go`, `internal/catalog/datadir_test.go`
- Modify: `internal/setup/steps.go:36` (`Plan`), `internal/ops/ops.go:169` (`clearPaths`), `internal/server/api.go:778-783` (`handleStartSetup`)
- Test: `internal/setup/steps_test.go`, `internal/ops/ops_test.go`, `internal/server/api_test.go`

**Interfaces:**
- Produces: `func catalog.ValidateDataDir(p string) error` — used again by the agent in Task 12 when it loads `node.json`.

- [ ] **Step 1: Write the failing table test**

```go
// internal/catalog/datadir_test.go
package catalog

import "testing"

func TestValidateDataDir(t *testing.T) {
	ok := []string{
		"/var/lib/valve-node-app/369",
		"/mnt/nvme0/reth",
		"/data/eth",
		"/opt/jumpgate/mainnet",
	}
	bad := []string{
		"",                  // empty
		"data",              // relative: rm -rf would run relative to the login dir
		"./data",            // relative
		"/",                 // root
		"/var",              // system tree: chown -R would re-own it
		"/var/lib",          // system tree
		"/etc",              // system tree
		"/home",             // system tree
		"/var/lib/x/../..",  // not clean
		"/var/lib/x/",       // not clean (trailing slash)
		"/data/with space",  // breaks ExecStart argument splitting
		"/data/new\nline",   // injects a unit directive
		"/data/tab\there",   // control character
		"/a",                // fewer than two components
	}
	for _, p := range ok {
		if err := ValidateDataDir(p); err != nil {
			t.Errorf("ValidateDataDir(%q) = %v, want nil", p, err)
		}
	}
	for _, p := range bad {
		if err := ValidateDataDir(p); err == nil {
			t.Errorf("ValidateDataDir(%q) = nil, want an error", p)
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/catalog/ -run TestValidateDataDir`
Expected: FAIL — `undefined: ValidateDataDir`.

- [ ] **Step 3: Implement**

```go
// internal/catalog/datadir.go
package catalog

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// systemTrees are directories a DataDir may live under but never BE. Setup
// runs `chown -R` on DataDir as root and clear runs `rm -rf` inside it, so a
// DataDir of /var would re-own the system tree to the service user.
var systemTrees = map[string]bool{
	"/bin": true, "/boot": true, "/dev": true, "/etc": true, "/home": true,
	"/lib": true, "/lib64": true, "/opt": true, "/proc": true, "/root": true,
	"/run": true, "/sbin": true, "/srv": true, "/sys": true, "/tmp": true,
	"/usr": true, "/var": true, "/var/lib": true,
}

// ValidateDataDir checks a node data directory before anything acts on it.
//
// It must be absolute (a relative path resolves against the SSH login
// directory), already clean (so what is checked is what is used), at least
// two components deep, free of whitespace and control characters (the path is
// written unquoted into a systemd unit, where a newline injects a directive
// that root then runs), and not itself a system tree.
func ValidateDataDir(p string) error {
	if p == "" {
		return fmt.Errorf("data directory is empty")
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("data directory %q must be an absolute path", p)
	}
	if path.Clean(p) != p {
		return fmt.Errorf("data directory %q is not a clean path (use %q)", p, path.Clean(p))
	}
	for _, r := range p {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("data directory %q contains whitespace or a control character", p)
		}
	}
	if strings.Count(p, "/") < 2 {
		return fmt.Errorf("data directory %q must be at least two levels deep", p)
	}
	if systemTrees[p] {
		return fmt.Errorf("data directory %q is a system directory; use a directory inside it", p)
	}
	return nil
}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./internal/catalog/ -run TestValidateDataDir -v`
Expected: PASS.

- [ ] **Step 5: Write failing tests for the three call sites**

Append to `internal/setup/steps_test.go`:

```go
func TestPlanRejectsAnUnsafeDataDir(t *testing.T) {
	w := validWire(t) // existing helper; if none exists, copy the WireConfig literal used by TestPlan* in this file
	w.DataDir = "/var"
	if _, err := Plan(w); err == nil {
		t.Fatal("Plan accepted DataDir /var")
	}
}
```

Append to `internal/ops/ops_test.go`:

```go
// The review's case: a relative DataDir passed the old containment check, so
// rm -rf ran relative to the SSH login directory.
func TestClearPathsRejectsARelativeDataDir(t *testing.T) {
	if _, err := clearPaths("data", []string{"reth"}); err == nil {
		t.Fatal("clearPaths accepted a relative DataDir")
	}
}
```

Append to `internal/server/api_test.go` (use the file's existing helpers for posting to `/api/targets/{id}/setup`; mirror the nearest existing setup test):

```go
func TestStartSetupRejectsAnUnsafeDataDir(t *testing.T) {
	// Arrange a target exactly as the existing TestStartSetup* tests do, then
	// post a wire whose DataDir is "/etc".
	// Assert: 400, and the body names the data directory.
}
```

Fill the server test body by copying the closest existing `TestStartSetup` arrangement in `api_test.go` verbatim and changing only `DataDir` and the assertion; the assertion is `res.StatusCode == 400 && strings.Contains(body, "data directory")`.

- [ ] **Step 6: Run them to verify they fail**

Run: `go test ./internal/setup/ ./internal/ops/ ./internal/server/ -run 'UnsafeDataDir|RelativeDataDir'`
Expected: 3 FAILs.

- [ ] **Step 7: Wire the validator in**

`internal/setup/steps.go`, first lines of `Plan`:

```go
	if err := catalog.ValidateDataDir(w.DataDir); err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
```

`internal/ops/ops.go`, first lines of `clearPaths` (keep the existing checks after it; they still guard the subdirs):

```go
	if err := catalog.ValidateDataDir(dataDir); err != nil {
		return nil, fmt.Errorf("refusing to clear: %w", err)
	}
```

`internal/server/api.go`, after the `DataDir`/`JWTPath` defaults in `handleStartSetup`:

```go
	if err := catalog.ValidateDataDir(wire.DataDir); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
```

- [ ] **Step 8: Run the whole suite**

Run: `go test ./...`
Expected: PASS. If an existing test fixture uses a DataDir the validator rejects (for example a relative `"data"` or `"/a"`), change the fixture to an absolute two-level path such as `/data/test` — the fixture was describing a configuration production now refuses. List each fixture you changed in the commit body.

- [ ] **Step 9: Commit**

```bash
git add internal/catalog/datadir.go internal/catalog/datadir_test.go internal/setup internal/ops internal/server
git commit -m "fix(catalog): validate DataDir before setup, clear, or the setup API use it

A DataDir of /var was chown -R'd to the service user, a newline injected a
directive into a root-written unit, and a relative path passed clearPaths so
rm -rf ran relative to the SSH login directory.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: File lock for `config.json` (and a shared lock helper)

**Files:**
- Create: `internal/filelock/filelock.go`, `internal/filelock/filelock_unix.go`, `internal/filelock/filelock_windows.go`, `internal/filelock/filelock_test.go`
- Modify: `internal/config/config.go` (`Load` takes a shared lock; new `Update`), `internal/server/api.go:346-369` (`loadConfig`, `updateConfig`)
- Test: `internal/config/update_test.go`

**Interfaces:**
- Produces: `filelock.Lock(path string, exclusive bool) (*filelock.Handle, error)`, `filelock.TryLock(path string) (*filelock.Handle, error)` (exclusive, non-blocking, `filelock.ErrLocked` when held), `(*Handle).Unlock() error`; `config.Update(fn func(*Config) error) (Config, error)`. Task 15 uses `TryLock`.

- [ ] **Step 1: Write the failing lock test**

```go
// internal/filelock/filelock_test.go
package filelock

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestTryLockIsExclusive(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.lock")
	h, err := TryLock(p)
	if err != nil {
		t.Fatalf("first TryLock: %v", err)
	}
	if _, err := TryLock(p); !errors.Is(err, ErrLocked) {
		t.Fatalf("second TryLock = %v, want ErrLocked", err)
	}
	if err := h.Unlock(); err != nil {
		t.Fatal(err)
	}
	h2, err := TryLock(p)
	if err != nil {
		t.Fatalf("TryLock after Unlock: %v", err)
	}
	h2.Unlock()
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/filelock/`
Expected: FAIL — package has no non-test Go files / undefined `TryLock`.

- [ ] **Step 3: Implement**

```go
// internal/filelock/filelock.go

// Package filelock takes advisory locks on files, so two processes (the web
// app and the TUI's server, or two servers) never interleave a
// read-modify-write of the same state.
package filelock

import (
	"errors"
	"os"
)

// ErrLocked reports that TryLock found the lock held.
var ErrLocked = errors.New("filelock: already locked")

// Handle is a held lock. The lock lives as long as the open file.
type Handle struct{ f *os.File }

// Lock blocks until it holds the lock at path, creating the file 0600.
func Lock(path string, exclusive bool) (*Handle, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(f, exclusive, true); err != nil {
		f.Close()
		return nil, err
	}
	return &Handle{f: f}, nil
}

// TryLock takes an exclusive lock without waiting, or returns ErrLocked.
func TryLock(path string) (*Handle, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(f, true, false); err != nil {
		f.Close()
		return nil, err
	}
	return &Handle{f: f}, nil
}

// File exposes the locked file, for a holder that writes its pid into it.
func (h *Handle) File() *os.File { return h.f }

// Unlock releases the lock.
func (h *Handle) Unlock() error {
	if err := unlockFile(h.f); err != nil {
		h.f.Close()
		return err
	}
	return h.f.Close()
}
```

```go
// internal/filelock/filelock_unix.go
//go:build unix

package filelock

import (
	"errors"
	"os"
	"syscall"
)

func lockFile(f *os.File, exclusive, wait bool) error {
	how := syscall.LOCK_SH
	if exclusive {
		how = syscall.LOCK_EX
	}
	if !wait {
		how |= syscall.LOCK_NB
	}
	err := syscall.Flock(int(f.Fd()), how)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return ErrLocked
	}
	return err
}

func unlockFile(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
```

```go
// internal/filelock/filelock_windows.go
//go:build windows

package filelock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func lockFile(f *os.File, exclusive, wait bool) error {
	var flags uint32
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	if !wait {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	ol := new(windows.Overlapped)
	err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, ol)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return ErrLocked
	}
	return err
}

func unlockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
}
```

`golang.org/x/sys` is already an indirect dependency; `go mod tidy` promotes it to direct.

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./internal/filelock/ -v && GOOS=windows go vet ./internal/filelock/`
Expected: PASS, and the Windows build vets cleanly.

- [ ] **Step 5: Write the failing `config.Update` test**

```go
// internal/config/update_test.go
package config

import (
	"fmt"
	"sync"
	"testing"
)

// Two processes editing config.json must not lose each other's edits. Each
// goroutine here takes the lock through its own file descriptor, which is how
// a second process would, so this is the cross-process case in miniature.
func TestUpdateSerialisesConcurrentEdits(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := Update(func(c *Config) error {
				c.Targets = append(c.Targets, Target{ID: fmt.Sprintf("t%d", i), Mode: "local"})
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Targets) != n {
		t.Fatalf("%d targets saved, want %d: concurrent edits were lost", len(c.Targets), n)
	}
}

func TestUpdateDoesNotSaveWhenFnFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, err := Update(func(c *Config) error {
		c.Targets = append(c.Targets, Target{ID: "x", Mode: "local"})
		return fmt.Errorf("nope")
	})
	if err == nil {
		t.Fatal("want fn's error")
	}
	c, _ := Load()
	if len(c.Targets) != 0 {
		t.Fatal("a failed Update was saved")
	}
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `go test ./internal/config/ -run Update`
Expected: FAIL — `undefined: Update`.

- [ ] **Step 7: Implement `Update` and the shared lock on `Load`**

In `internal/config/config.go`:

```go
// lockFileName sits beside config.json. The lock is on a separate file
// because Save replaces config.json by rename, and a lock on the old inode
// would not exclude a writer that opened the new one.
const lockFileName = "config.json.lock"

func lockPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("config: create %s: %w", dir, err)
	}
	return filepath.Join(dir, lockFileName), nil
}

// Update loads the config, applies fn, and saves it, holding an exclusive
// lock across all three so another process's edit can never be lost between
// this one's read and write. If fn fails nothing is saved.
func Update(fn func(*Config) error) (Config, error) {
	lp, err := lockPath()
	if err != nil {
		return Config{}, err
	}
	h, err := filelock.Lock(lp, true)
	if err != nil {
		return Config{}, fmt.Errorf("config: lock: %w", err)
	}
	defer h.Unlock()

	c, err := load()
	if err != nil {
		return Config{}, err
	}
	if err := fn(&c); err != nil {
		return Config{}, err
	}
	if err := c.Save(); err != nil {
		return Config{}, err
	}
	return c, nil
}
```

Rename the current `Load` body to an unexported `load()`, and make `Load` take a shared lock around it:

```go
func Load() (Config, error) {
	lp, err := lockPath()
	if err != nil {
		return Config{}, err
	}
	h, err := filelock.Lock(lp, false)
	if err != nil {
		return Config{}, fmt.Errorf("config: lock: %w", err)
	}
	defer h.Unlock()
	return load()
}
```

Note: `lockPath` creates `~/.jumpgate` (0700) on first `Load`, where before only `Save` created it. That is harmless and keeps the lock usable before any save.

In `internal/server/api.go`, make `updateConfig` delegate (keep `cfgMu`, which still orders requests inside one process):

```go
func (s *Server) updateConfig(fn func(c *config.Config) error) (config.Config, error) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	return config.Update(fn)
}
```

- [ ] **Step 8: Run the tests**

Run: `go test -race ./internal/config/ ./internal/server/ && go test ./...`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/filelock internal/config internal/server/api.go go.mod go.sum
git commit -m "feat(config): lock config.json across processes

The server serialised its own edits with an in-process mutex only, so the web
app and a second process (the TUI's server, a CLI) could each read, modify and
save, losing one edit. Update now holds an exclusive file lock across load,
apply and save; Load takes a shared one.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Remote `WriteFile` sends content on stdin

**Files:**
- Modify: `internal/executor/executor.go` (`RunOpts.Stdin`), `internal/executor/ssh.go:143-170`, `internal/executor/local.go` (honour `Stdin`)
- Test: `internal/executor/ssh_test.go`, `internal/executor/remotepath_test.go` (update the existing `writeFileCmd` assertions)

**Interfaces:**
- Produces: `RunOpts{Stream StreamFunc; Stdin io.Reader}`; `writeFileCmd(remotePath string, mode fs.FileMode) string` (content no longer a parameter).

- [ ] **Step 1: Write the failing tests**

Append to `internal/executor/ssh_test.go`:

```go
// The content must never appear in the remote command line: any local user on
// the target can read every process's arguments from /proc.
func TestWriteFileCmdCarriesNoContent(t *testing.T) {
	cmd := writeFileCmd("/etc/wireguard/jumpgate0.conf", 0o600)
	for _, frag := range []string{"base64", "printf"} {
		if strings.Contains(cmd, frag) {
			t.Errorf("command still embeds content via %s: %s", frag, cmd)
		}
	}
	if !strings.Contains(cmd, "umask 077") {
		t.Errorf("temp file is not created under umask 077: %s", cmd)
	}
}

// Parent directories keep the caller's umask. Only the temp file is created
// 077, so a new directory a service must traverse is not made 0700 root.
func TestWriteFileDoesNotTightenNewParentDirs(t *testing.T) {
	cmd := writeFileCmd("/var/lib/x/y/file", 0o644)
	mk := strings.Index(cmd, "mkdir -p")
	um := strings.Index(cmd, "umask 077")
	if mk < 0 || um < 0 || mk > um {
		t.Fatalf("mkdir -p must run before umask 077 takes effect: %s", cmd)
	}
}

func TestSSH_WriteFile_LargeContentAndMode(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	ex, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "big.bin")
	content := bytes.Repeat([]byte("0123456789abcdef"), 64*1024) // 1 MiB: far past ARG_MAX for argv
	if err := ex.WriteFile(context.Background(), target, content, 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("content mismatch (err %v, %d bytes)", err, len(got))
	}
	fi, _ := os.Stat(target)
	if fi.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}
```

(Add `"bytes"` to the test imports.) In `internal/executor/remotepath_test.go`, change every `writeFileCmd(path, content, mode)` call to `writeFileCmd(path, mode)` and drop assertions about the base64 payload; keep the assertions about the POSIX parent directory.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/executor/ -run 'WriteFile'`
Expected: compile failure (`too few arguments`) — then, after Step 3's signature change only, the content and umask assertions FAIL. Do the signature change first, watch the behavioural failures, then finish Step 3.

- [ ] **Step 3: Implement**

`internal/executor/executor.go`:

```go
// RunOpts controls Run behavior. Every field may be zero.
type RunOpts struct {
	Stream StreamFunc
	// Stdin, if set, is copied to the command's standard input. WriteFile
	// uses it so file content never appears in a command line.
	Stdin io.Reader
}
```

`internal/executor/ssh.go`:

```go
// WriteFile writes content to path on the remote host. The content travels on
// the session's stdin, never in the command line, where any local user on the
// target could read it from /proc. It is written to a temp file created under
// umask 077, chmod'ed, then renamed into place, so the file is never readable
// wider than its final mode and a reader never sees it half-written.
func (s *sshExecutor) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	res, err := s.Run(ctx, writeFileCmd(path, mode), &RunOpts{Stdin: bytes.NewReader(content)})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("write remote file %s: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	return nil
}

// writeFileCmd builds the POSIX shell line WriteFile runs. mkdir -p runs
// first, under the caller's umask, so a new parent directory a service must
// traverse is not created 0700; only the temp file is created under 077.
// remoteDir (not filepath.Dir) keeps this correct on a Windows control plane.
func writeFileCmd(remotePath string, mode fs.FileMode) string {
	q := shQuote(remotePath)
	return fmt.Sprintf(
		"mkdir -p %s && (umask 077 && tmp=$(mktemp %s.XXXXXX) && cat > \"$tmp\" && chmod %o \"$tmp\" && mv -f \"$tmp\" %s || { rm -f \"$tmp\"; exit 1; })",
		shQuote(remoteDir(remotePath)), shQuote(remotePath), mode.Perm(), q,
	)
}
```

In `sshExecutor.Run`, after creating the session and before `Start`:

```go
	if opts != nil && opts.Stdin != nil {
		session.Stdin = opts.Stdin
	}
```

In `local.Run`, before `c.Start()`:

```go
	if opts != nil && opts.Stdin != nil {
		c.Stdin = opts.Stdin
	}
```

Remove the now-unused `encoding/base64` import from `ssh.go` only if `ReadFile` no longer uses it (it does — keep it).

- [ ] **Step 4: Run the executor tests**

Run: `go test -race ./internal/executor/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/executor
git commit -m "fix(executor): send remote file content on stdin, under umask 077

WriteFile put the whole payload (WireGuard private keys, eRPC provider keys)
in the sh -c command line, readable from /proc by any user on the target,
created the file world-readable before chmod, and failed past ARG_MAX. Content
now streams on stdin into a 077 temp file that is chmod'ed and renamed into
place. Parent directories keep the normal umask.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Cancelling a remote command kills it on the box

**Files:**
- Create: `internal/executor/procgroup.go`, `internal/executor/procgroup_test.go`
- Modify: `internal/executor/ssh.go` (`Run`)
- Test: `internal/executor/ssh_test.go` (new `startDetachedSSHD` helper)

**Interfaces:**
- Produces: `wrapInProcessGroup(cmd string) string`; the first stdout line of every SSH `Run` is consumed by the executor and never reaches `Result.Stdout` or `Stream`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/executor/procgroup_test.go
package executor

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// The wrapper must preserve the command's stdout and exit status exactly,
// apart from one leading marker line the executor consumes.
func TestWrapInProcessGroupPreservesOutputAndStatus(t *testing.T) {
	out, err := exec.Command("sh", "-c", wrapInProcessGroup("echo one; echo two; exit 7")).Output()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 7 {
		t.Fatalf("exit = %v, want status 7", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], pgidMarker) || lines[1] != "one" || lines[2] != "two" {
		t.Fatalf("output = %q", out)
	}
}

// Stdin still reaches the command, so WriteFile works through the wrapper.
func TestWrapInProcessGroupPassesStdin(t *testing.T) {
	c := exec.Command("sh", "-c", wrapInProcessGroup("cat"))
	c.Stdin = strings.NewReader("hello")
	var out bytes.Buffer
	c.Stdout = &out
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out.String(), "hello") {
		t.Fatalf("stdin did not reach the command: %q", out.String())
	}
}
```

Append to `internal/executor/ssh_test.go` a second server that, like OpenSSH, does **not** kill a command when its session closes, and the cancel test:

```go
// startDetachedSSHD behaves like a real sshd in the one way that matters here:
// closing a session does not kill the command it started.
func startDetachedSSHD(t *testing.T) (testSSHD, string) {
	t.Helper()
	d, keyPath := startTestSSHDWith(t, func(s gliderssh.Session) {
		c := exec.Command("sh", "-c", s.RawCommand())
		c.Stdout, c.Stderr, c.Stdin = s, s.Stderr(), s
		_ = c.Start()
		done := make(chan error, 1)
		go func() { done <- c.Wait() }()
		select {
		case err := <-done:
			code := 0
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			}
			_ = s.Exit(code)
		case <-s.Context().Done():
			// Session gone; leave the process running, as sshd would.
		}
	})
	return d, keyPath
}

func TestSSH_CancelKillsTheRemoteProcessGroup(t *testing.T) {
	d, keyPath := startDetachedSSHD(t)
	ex, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	go func() {
		_, _ = ex.Run(ctx, "sh -c 'echo $$ > "+pidFile+"; echo up; exec sleep 60'", &RunOpts{Stream: func(l string) {
			if l == "up" {
				close(started)
			}
		}})
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("command never started")
	}
	cancel()

	raw, _ := os.ReadFile(pidFile)
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return // gone
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("remote process %d survived cancellation", pid)
}
```

Refactor `startTestSSHD` into `startTestSSHDWith(t, handler)` plus the existing default handler so both helpers share setup; add `"syscall"` and `"time"` to the imports. Mark the cancel test `//go:build unix` by putting it in `ssh_cancel_unix_test.go` if the package builds on Windows.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/executor/ -run 'ProcessGroup|Cancel' -v`
Expected: FAIL — `undefined: wrapInProcessGroup`, then (once it exists as a pass-through) `remote process … survived cancellation`.

- [ ] **Step 3: Implement**

```go
// internal/executor/procgroup.go
package executor

import (
	"strconv"
	"strings"
)

// pgidMarker prefixes the first stdout line of every wrapped command.
const pgidMarker = "jumpgate-pgid:"

// wrapInProcessGroup runs cmd as a background job of a shell with job control
// on (set -m), which places the job in a process group of its own whose id is
// the job's pid. The wrapper prints that id first and then waits, exiting with
// the job's status.
//
// Closing an SSH session does not stop the command it started, so without a
// group id there is no way to stop a cancelled command short of guessing at
// pids. setsid(1) is not an option: it is missing on macOS and forks away from
// the session when the shell is already a group leader.
func wrapInProcessGroup(cmd string) string {
	return "set -m; ( " + cmd + " ) & jg_pg=$!; printf '" + pgidMarker + "%s\\n' \"$jg_pg\"; wait \"$jg_pg\""
}

// parsePgidLine reads the marker line. ok is false for any other line.
func parsePgidLine(line string) (int, bool) {
	rest, found := strings.CutPrefix(line, pgidMarker)
	if !found {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(rest))
	return n, err == nil && n > 1
}

// killGroupCmd stops a process group: TERM, then up to five seconds for it to
// exit, then KILL.
func killGroupCmd(pgid int) string {
	g := strconv.Itoa(pgid)
	return "kill -TERM -- -" + g + " 2>/dev/null; i=0; while [ $i -lt 5 ] && kill -0 -- -" + g +
		" 2>/dev/null; do sleep 1; i=$((i+1)); done; kill -KILL -- -" + g + " 2>/dev/null; exit 0"
}
```

In `sshExecutor.Run`:

1. Start `wrapInProcessGroup(cmd)` instead of `cmd`.
2. Give `lineStreamer` a one-shot interceptor: the first complete line, if `parsePgidLine` accepts it, is stored (under a mutex or an atomic) and not written to the buffer or passed to `fn`. Add the field to `lineStreamer`:

```go
	// onFirst, if set, sees the first complete line and reports whether to
	// swallow it. It is cleared after the first line.
	onFirst func(line string) bool
```

and in the method that emits a complete line, before buffering it:

```go
	if l.onFirst != nil {
		swallow := l.onFirst(line)
		l.onFirst = nil
		if swallow {
			return
		}
	}
```

3. Replace the cancel goroutine with:

```go
	go func() {
		select {
		case <-ctx.Done():
			_ = session.Signal(ssh.SIGTERM) // honoured by OpenSSH >= 7.9; harmless otherwise
			if pg := pgid.Load(); pg > 1 {
				s.killGroup(int(pg))
			}
			_ = session.Close()
		case <-done:
		}
	}()
```

with `pgid` an `atomic.Int64` set by `onFirst`, and:

```go
// killGroup stops a cancelled command's process group on a fresh session. It
// gets its own short budget because the caller's context is already done.
func (s *sshExecutor) killGroup(pgid int) {
	kctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sess, err := s.client.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()
	done := make(chan struct{})
	go func() { _ = sess.Run(killGroupCmd(pgid)); close(done) }()
	select {
	case <-done:
	case <-kctx.Done():
	}
}
```

- [ ] **Step 4: Run the executor tests**

Run: `go test -race ./internal/executor/ && go test ./...`
Expected: PASS. Every existing SSH test still sees exactly the command's own output.

- [ ] **Step 5: Commit**

```bash
git add internal/executor
git commit -m "fix(executor): stop a cancelled remote command on the box

Cancelling only closed the SSH session, and sshd does not kill a command when
its session closes, so a cancelled setup left cargo builds, downloads or rm -rf
running and a retry ran a second copy beside them. Commands now run in their
own process group, whose id the executor reads from a marker line and kills
(TERM, then KILL after 5s) on a fresh session.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: SSH dialling: handshake deadline, ssh-agent, jump hosts

**Files:**
- Modify: `internal/executor/executor.go` (`SSHConfig`), `internal/executor/ssh.go` (`NewSSH`)
- Create: `internal/executor/dial.go`, `internal/executor/dial_test.go`

**Interfaces:**
- Produces:
  - `SSHConfig{Host, User, KeyPath, HostKeyFile string; Port int; Jump *SSHConfig; HostKey ssh.HostKeyCallback /* json:"-" */}`
  - `func DialSSH(ctx context.Context, cfg SSHConfig) (*ssh.Client, error)` — used by Task 13.
  - `func NewSSHContext(ctx context.Context, cfg SSHConfig) (Executor, error)`; `NewSSH(cfg)` becomes `NewSSHContext(context.Background(), cfg)`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/executor/dial_test.go
package executor

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// A host that accepts TCP and never speaks SSH must not hang the caller.
func TestDialSSHHasAHandshakeDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close() // hold it open, say nothing
		}
	}()

	old := handshakeTimeout
	handshakeTimeout = 300 * time.Millisecond
	defer func() { handshakeTimeout = old }()

	addr := ln.Addr().(*net.TCPAddr)
	start := time.Now()
	_, err = DialSSH(context.Background(), SSHConfig{
		Host: "127.0.0.1", Port: addr.Port, User: "x",
		KeyPath: writeTempKey(t), HostKey: ssh.InsecureIgnoreHostKey(),
	})
	if err == nil {
		t.Fatal("handshake against a silent server succeeded")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("took %v; the handshake deadline did not apply", time.Since(start))
	}
}

// With SSH_AUTH_SOCK set and no key file, the agent's keys authenticate. This
// is how passphrase-protected keys are supported.
func TestDialSSHUsesTheSSHAgent(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyring := agent.NewKeyring()
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(shortTempDir(t), "agent.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(keyring, c)
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", sock)

	signer, _ := ssh.NewSignerFromKey(priv)
	d := startTestSSHDAcceptingOnly(t, signer.PublicKey())
	client, err := DialSSH(context.Background(), SSHConfig{
		Host: d.host, Port: d.port, User: "x", HostKey: ssh.FixedHostKey(d.hostKey),
	})
	if err != nil {
		t.Fatalf("DialSSH via agent: %v", err)
	}
	client.Close()
}

// A jump host is dialled first and the target is reached through it.
func TestDialSSHThroughAJumpHost(t *testing.T) {
	jump, keyPath := startTestSSHDWithForwarding(t) // gliderssh with LocalPortForwardingCallback returning true and DirectTCPIPHandler set
	target, _ := startTestSSHD(t)
	client, err := DialSSH(context.Background(), SSHConfig{
		Host: target.host, Port: target.port, User: "x", KeyPath: keyPath,
		HostKey: ssh.FixedHostKey(target.hostKey),
		Jump: &SSHConfig{Host: jump.host, Port: jump.port, User: "x", KeyPath: keyPath,
			HostKey: ssh.FixedHostKey(jump.hostKey)},
	})
	if err != nil {
		t.Fatalf("DialSSH via jump: %v", err)
	}
	client.Close()
}

func shortTempDir(t *testing.T) string {
	t.Helper()
	// unix socket paths are capped near 104 bytes on macOS; t.TempDir() is too long.
	d, err := os.MkdirTemp("/tmp", "jg")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}
```

Add the helpers next to `startTestSSHD` in `ssh_test.go`: `writeTempKey(t) string` (wraps the existing `writePrivateKey` with a fresh key), `startTestSSHDAcceptingOnly(t, ssh.PublicKey) testSSHD` (PublicKeyHandler compares `key.Marshal()` with the allowed key), and `startTestSSHDWithForwarding(t) (testSSHD, string)` (sets `LocalPortForwardingCallback: func(gliderssh.Context, string, uint32) bool { return true }` and `ChannelHandlers: map[string]gliderssh.ChannelHandler{"session": gliderssh.DefaultSessionHandler, "direct-tcpip": gliderssh.DirectTCPIPHandler}`).

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/executor/ -run DialSSH -v`
Expected: FAIL — `undefined: DialSSH`, `undefined: handshakeTimeout`, unknown fields `Jump`, `HostKey`.

- [ ] **Step 3: Implement**

`internal/executor/executor.go` — extend `SSHConfig`:

```go
type SSHConfig struct {
	Host        string
	User        string
	KeyPath     string // optional when an ssh-agent is available
	HostKeyFile string
	Port        int // 0 => 22
	// Jump, if set, is dialled first and this host is reached through it
	// (ProxyJump). It may itself have a Jump.
	Jump *SSHConfig `json:"jump,omitempty"`
	// HostKey overrides host-key checking. Nil keeps trust-on-first-use
	// against HostKeyFile, which is what the web UI relies on until it moves
	// onto agents. It is never persisted.
	HostKey ssh.HostKeyCallback `json:"-"`
}
```

```go
// internal/executor/dial.go
package executor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// handshakeTimeout bounds TCP connect plus the whole SSH handshake. A host
// that accepts TCP and then stalls in key exchange would otherwise hang the
// caller forever: ssh.ClientConfig.Timeout covers only the connect. A var so
// tests can shorten it.
var handshakeTimeout = 10 * time.Second

// DialSSH connects to cfg, through cfg.Jump if set.
func DialSSH(ctx context.Context, cfg SSHConfig) (*ssh.Client, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))

	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	hostKey := cfg.HostKey
	if hostKey == nil {
		hostKey = tofuHostKeyCallback(cfg.HostKeyFile)
	}
	clientCfg := &ssh.ClientConfig{User: cfg.User, Auth: auth, HostKeyCallback: hostKey, Timeout: handshakeTimeout}

	var conn net.Conn
	if cfg.Jump != nil {
		jump, err := DialSSH(ctx, *cfg.Jump)
		if err != nil {
			return nil, fmt.Errorf("jump host %s: %w", cfg.Jump.Host, err)
		}
		conn, err = jump.DialContext(ctx, "tcp", addr)
		if err != nil {
			jump.Close()
			return nil, fmt.Errorf("via jump host %s: %w", cfg.Jump.Host, err)
		}
	} else {
		d := net.Dialer{Timeout: handshakeTimeout}
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
	}

	deadline := time.Now().Add(handshakeTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, clientCfg)
	if err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

// authMethods offers ssh-agent keys first, then KeyPath. Passphrase-protected
// key files are supported only through an agent.
func authMethods(cfg SSHConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if c, err := net.Dial("unix", sock); err == nil {
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(c).Signers))
		}
	}
	if cfg.KeyPath != "" {
		keyBytes, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key %s: %w", cfg.KeyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			if len(methods) == 0 {
				return nil, fmt.Errorf("private key %s is passphrase-protected; load it into ssh-agent (ssh-add %s)", cfg.KeyPath, cfg.KeyPath)
			}
		} else if err != nil {
			return nil, fmt.Errorf("parse private key %s: %w", cfg.KeyPath, err)
		} else {
			methods = append(methods, ssh.PublicKeys(signer))
		}
	}
	if len(methods) == 0 {
		return nil, errors.New("no SSH credentials: set a key path or run an ssh-agent")
	}
	return methods, nil
}
```

`internal/executor/ssh.go`:

```go
// NewSSH dials cfg with no caller deadline beyond the handshake timeout.
func NewSSH(cfg SSHConfig) (Executor, error) { return NewSSHContext(context.Background(), cfg) }

// NewSSHContext dials cfg and returns an executor over the connection.
func NewSSHContext(ctx context.Context, cfg SSHConfig) (Executor, error) {
	client, err := DialSSH(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &sshExecutor{client: client}, nil
}
```

Delete the old body of `NewSSH` (key reading and `ssh.Dial` now live in `dial.go`).

- [ ] **Step 4: Run the tests**

Run: `go test -race ./internal/executor/ && go test ./...`
Expected: PASS, including the existing TOFU tests (nil `HostKey` still means TOFU).

- [ ] **Step 5: Commit**

```bash
git add internal/executor
git commit -m "feat(executor): bound the SSH handshake, use ssh-agent, support jump hosts

A host that accepted TCP and stalled in key exchange hung the caller: the
client timeout only covers the connect. The whole handshake now has a 10s
deadline. ssh-agent keys are offered first, which is also how passphrase
keys work, and SSHConfig.Jump reaches a host through a bastion.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Host-key decisions: strict checking, OpenSSH known_hosts, capture-and-confirm

**Files:**
- Create: `internal/executor/hostkey_decider.go`, `internal/executor/hostkey_decider_test.go`
- Modify: `internal/executor/hostkey.go` (export `RecordHostKey`)

**Interfaces:**
- Produces:
  - `var ErrUnknownHost = errors.New(...)`; `type UnknownHostError struct{ Host, Fingerprint string; Key ssh.PublicKey }` (implements `error`, `Unwrap() error { return ErrUnknownHost }`)
  - `func Strict(jumpgateFile string, opensshFiles ...string) ssh.HostKeyCallback` — accept only a key already recorded in `jumpgateFile` or in an OpenSSH known_hosts file; unknown → `*UnknownHostError`; mismatch anywhere → error.
  - `func CaptureHostKey(ctx context.Context, cfg SSHConfig) (ssh.PublicKey, error)` — learns a host's key without authenticating.
  - `func RecordHostKey(file, hostport string, key ssh.PublicKey) error`
  - `func Fingerprint(key ssh.PublicKey) string` — `SHA256:…`

- [ ] **Step 1: Write the failing tests**

```go
// internal/executor/hostkey_decider_test.go
package executor

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func newHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	k, _ := ssh.NewPublicKey(pub)
	return k
}

var addr = &net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 22}

func TestStrictRefusesAnUnknownHostWithItsFingerprint(t *testing.T) {
	cb := Strict(filepath.Join(t.TempDir(), "known_hosts"))
	key := newHostKey(t)
	err := cb("203.0.113.7:22", addr, key)
	var unknown *UnknownHostError
	if !errors.As(err, &unknown) || !errors.Is(err, ErrUnknownHost) {
		t.Fatalf("err = %v, want *UnknownHostError", err)
	}
	if unknown.Fingerprint != Fingerprint(key) {
		t.Errorf("fingerprint = %q, want %q", unknown.Fingerprint, Fingerprint(key))
	}
}

func TestStrictAcceptsARecordedKeyAndRefusesAChangedOne(t *testing.T) {
	file := filepath.Join(t.TempDir(), "known_hosts")
	key := newHostKey(t)
	if err := RecordHostKey(file, "203.0.113.7:22", key); err != nil {
		t.Fatal(err)
	}
	cb := Strict(file)
	if err := cb("203.0.113.7:22", addr, key); err != nil {
		t.Fatalf("recorded key refused: %v", err)
	}
	if err := cb("203.0.113.7:22", addr, newHostKey(t)); err == nil || errors.Is(err, ErrUnknownHost) {
		t.Fatalf("changed key: err = %v, want a mismatch error", err)
	}
}

// An operator who already ssh'd to the box has verified it; jumpgate trusts
// their OpenSSH known_hosts rather than asking again.
func TestStrictTrustsOpenSSHKnownHosts(t *testing.T) {
	dir := t.TempDir()
	openssh := filepath.Join(dir, "ssh_known_hosts")
	key := newHostKey(t)
	line := knownhosts.Line([]string{knownhosts.Normalize("203.0.113.7:22")}, key)
	if err := os.WriteFile(openssh, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cb := Strict(filepath.Join(dir, "jg_known_hosts"), openssh)
	if err := cb("203.0.113.7:22", addr, key); err != nil {
		t.Fatalf("key in OpenSSH known_hosts refused: %v", err)
	}
	if err := cb("203.0.113.7:22", addr, newHostKey(t)); err == nil || errors.Is(err, ErrUnknownHost) {
		t.Fatalf("mismatch against OpenSSH known_hosts: err = %v, want a mismatch error", err)
	}
}

func TestCaptureHostKeyNeedsNoCredentials(t *testing.T) {
	d, _ := startTestSSHD(t)
	key, err := CaptureHostKey(context.Background(), SSHConfig{Host: d.host, Port: d.port, User: "nobody"})
	if err != nil {
		t.Fatalf("CaptureHostKey: %v", err)
	}
	if string(key.Marshal()) != string(d.hostKey.Marshal()) {
		t.Fatal("captured the wrong key")
	}
}
```

Rename the package-level `addr` if it collides with an existing identifier in the test package.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/executor/ -run 'Strict|Capture' -v`
Expected: FAIL — undefined `Strict`, `UnknownHostError`, `RecordHostKey`, `CaptureHostKey`, `Fingerprint`.

- [ ] **Step 3: Implement**

In `internal/executor/hostkey.go`, rename `appendHostKey` to exported `RecordHostKey` (keep its body; update its one caller in `tofuHostKeyCallback`).

```go
// internal/executor/hostkey_decider.go
package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ErrUnknownHost marks a host whose key nobody has confirmed yet.
var ErrUnknownHost = errors.New("unknown SSH host")

// UnknownHostError carries what an operator needs to confirm a host: its
// address and key fingerprint.
type UnknownHostError struct {
	Host        string
	Fingerprint string
	Key         ssh.PublicKey
}

func (e *UnknownHostError) Error() string {
	return fmt.Sprintf("%s presents an unconfirmed host key %s", e.Host, e.Fingerprint)
}

func (e *UnknownHostError) Unwrap() error { return ErrUnknownHost }

// Fingerprint is OpenSSH's SHA256 form, the one `ssh-keygen -lf` prints, so
// an operator can compare it with the console of the box.
func Fingerprint(key ssh.PublicKey) string { return ssh.FingerprintSHA256(key) }

// Strict accepts a host only if its key is already recorded, in jumpgate's own
// file or in one of the operator's OpenSSH known_hosts files. Unlike
// trust-on-first-use it never records anything itself: an unknown host is an
// *UnknownHostError the caller must put in front of a person.
func Strict(jumpgateFile string, opensshFiles ...string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		known, err := lookupHostKey(jumpgateFile, hostname)
		if err != nil {
			return err
		}
		if known != nil {
			if !bytes.Equal(known.Marshal(), key.Marshal()) {
				return fmt.Errorf("host key mismatch for %s: presented %s does not match %s on record in %s (possible man-in-the-middle, or the host was rebuilt)",
					hostname, Fingerprint(key), Fingerprint(known), jumpgateFile)
			}
			return nil
		}
		var present []string
		for _, f := range opensshFiles {
			if _, err := os.Stat(f); err == nil {
				present = append(present, f)
			}
		}
		if len(present) > 0 {
			cb, err := knownhosts.New(present...)
			if err != nil {
				return fmt.Errorf("read known_hosts: %w", err)
			}
			err = cb(hostname, remote, key)
			var keyErr *knownhosts.KeyError
			switch {
			case err == nil:
				return nil
			case errors.As(err, &keyErr) && len(keyErr.Want) > 0:
				return fmt.Errorf("host key mismatch for %s against your OpenSSH known_hosts: %w", hostname, err)
			case errors.As(err, &keyErr):
				// Not listed there either: fall through to unknown.
			default:
				return err
			}
		}
		return &UnknownHostError{Host: hostname, Fingerprint: Fingerprint(key), Key: key}
	}
}

// errCaptured stops the handshake once the key is in hand.
var errCaptured = errors.New("host key captured")

// CaptureHostKey connects far enough to read the host's key and then hangs up,
// before any authentication. It is how the CLI shows a fingerprint to confirm.
func CaptureHostKey(ctx context.Context, cfg SSHConfig) (ssh.PublicKey, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	var got ssh.PublicKey
	probe := cfg
	probe.KeyPath = ""
	probe.HostKey = func(_ string, _ net.Addr, key ssh.PublicKey) error {
		got = key
		return errCaptured
	}
	d := net.Dialer{Timeout: handshakeTimeout}
	var conn net.Conn
	var err error
	if cfg.Jump != nil {
		jump, jerr := DialSSH(ctx, *cfg.Jump)
		if jerr != nil {
			return nil, fmt.Errorf("jump host %s: %w", cfg.Jump.Host, jerr)
		}
		defer jump.Close()
		conn, err = jump.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = d.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	_, _, _, err = ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User:            cfg.User,
		HostKeyCallback: probe.HostKey,
	})
	if got == nil {
		return nil, fmt.Errorf("no host key from %s: %w", addr, err)
	}
	return got, nil
}
```

(Add `"time"` to the imports.)

- [ ] **Step 4: Run the tests**

Run: `go test -race ./internal/executor/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/executor
git commit -m "feat(executor): strict host-key checking and capture-to-confirm

Trust-on-first-use pinned whatever key answered first, including a
man-in-the-middle's, with no fingerprint ever shown. Strict accepts only keys
already recorded by jumpgate or in the operator's OpenSSH known_hosts and
returns the fingerprint of anything else; CaptureHostKey reads a host's key
without authenticating so the CLI can ask a person. TOFU remains the default
for the web UI until it moves onto agents.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 7: `internal/eip712` — typed-data hashing

**Files:**
- Create: `internal/eip712/eip712.go`, `internal/eip712/address.go`, `internal/eip712/eip712_test.go`

**Interfaces:**
- Produces:
  - `type Address [20]byte`; `ParseAddress(s string) (Address, error)`; `(Address) Hex() string` (EIP-55 checksummed); `(Address) IsZero() bool`
  - `type Field struct{ Name, Type string }`; `type Types map[string][]Field`
  - `type TypedData struct{ Types Types; PrimaryType string; Domain map[string]any; Message map[string]any }`
  - `Keccak256(parts ...[]byte) [32]byte`
  - `EncodeType(types Types, primary string) (string, error)`; `HashStruct(types Types, primary string, data map[string]any) ([32]byte, error)`; `Digest(td TypedData) ([32]byte, error)`
  - Supported field types: `address` (`Address`), `bool` (`bool`), `string` (`string`), `bytes32` (`[32]byte`), `uint8` (`uint8`), `uint64` (`uint64`), `uint256` (`*big.Int`), and any struct type named in `Types` (`map[string]any`). Arrays are rejected.

- [ ] **Step 1: Write the failing tests (EIP-712 specification vectors)**

```go
// internal/eip712/eip712_test.go
package eip712

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
)

// The Mail example from the EIP-712 specification (its Example.js). If any
// constant below ever disagrees with this code, re-check it against
// https://eips.ethereum.org/EIPS/eip-712 before touching the encoder.
func mailTypedData(t *testing.T) TypedData {
	t.Helper()
	addr := func(s string) Address {
		a, err := ParseAddress(s)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	return TypedData{
		Types: Types{
			"EIP712Domain": {{"name", "string"}, {"version", "string"}, {"chainId", "uint256"}, {"verifyingContract", "address"}},
			"Person":       {{"name", "string"}, {"wallet", "address"}},
			"Mail":         {{"from", "Person"}, {"to", "Person"}, {"contents", "string"}},
		},
		PrimaryType: "Mail",
		Domain: map[string]any{
			"name": "Ether Mail", "version": "1", "chainId": big.NewInt(1),
			"verifyingContract": addr("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"),
		},
		Message: map[string]any{
			"from":     map[string]any{"name": "Cow", "wallet": addr("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826")},
			"to":       map[string]any{"name": "Bob", "wallet": addr("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB")},
			"contents": "Hello, Bob!",
		},
	}
}

func h32(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 32 {
		t.Fatalf("bad hex %q", s)
	}
	var out [32]byte
	copy(out[:], b)
	return out
}

func TestEncodeTypeOrdersDependencies(t *testing.T) {
	got, err := EncodeType(mailTypedData(t).Types, "Mail")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Mail(Person from,Person to,string contents)Person(string name,address wallet)"; got != want {
		t.Fatalf("EncodeType = %q, want %q", got, want)
	}
}

func TestMailVectors(t *testing.T) {
	td := mailTypedData(t)

	typeHash := Keccak256([]byte("Mail(Person from,Person to,string contents)Person(string name,address wallet)"))
	if want := h32(t, "0xa0cedeb2dc280ba39b857546d74f5549c3a1d7bdc2dd96bf881f76108e23dac2"); typeHash != want {
		t.Fatalf("Mail typeHash = %x", typeHash)
	}

	domain, err := HashStruct(td.Types, "EIP712Domain", td.Domain)
	if err != nil {
		t.Fatal(err)
	}
	if want := h32(t, "0xf2cee375fa42b42143804025fc449deafd50cc031ca257e0b194a650a912090f"); domain != want {
		t.Errorf("domainSeparator = %x", domain)
	}

	msg, err := HashStruct(td.Types, "Mail", td.Message)
	if err != nil {
		t.Fatal(err)
	}
	if want := h32(t, "0xc52c0ee5d84264471806290a3f2c4cecfc5490626bf912d01f240d7a274b371e"); msg != want {
		t.Errorf("hashStruct(message) = %x", msg)
	}

	digest, err := Digest(td)
	if err != nil {
		t.Fatal(err)
	}
	if want := h32(t, "0xbe609aee343fb3c4b28e1df9e632fca64fcfaede20f02e86244efddf30957bd2"); digest != want {
		t.Errorf("digest = %x", digest)
	}
}

func TestHashStructRejectsWrongShapes(t *testing.T) {
	types := Types{"T": {{"n", "uint64"}, {"s", "string"}}}
	cases := map[string]map[string]any{
		"missing field": {"n": uint64(1)},
		"wrong go type": {"n": 1, "s": "x"},
		"extra field":   {"n": uint64(1), "s": "x", "z": "y"},
	}
	for name, msg := range cases {
		if _, err := HashStruct(types, "T", msg); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := HashStruct(Types{"T": {{"xs", "uint64[]"}}}, "T", map[string]any{"xs": []uint64{1}}); err == nil {
		t.Error("arrays must be rejected, not mis-encoded")
	}
	if _, err := HashStruct(Types{"T": {{"n", "uint8"}}}, "T", map[string]any{"n": uint64(300)}); err == nil {
		t.Error("a uint64 value must not be accepted for a uint8 field")
	}
}

func TestAddressChecksum(t *testing.T) {
	a, err := ParseAddress("0xcd2a3d9f938e13cd947ec05abc7fe734df8dd826")
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Hex(); got != "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826" {
		t.Fatalf("Hex() = %s", got)
	}
	if _, err := ParseAddress("0x1234"); err == nil {
		t.Error("short address accepted")
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/eip712/`
Expected: FAIL — package has no non-test files.

- [ ] **Step 3: Implement**

```go
// internal/eip712/address.go
package eip712

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Address is a 20-byte Ethereum address.
type Address [20]byte

// ParseAddress reads a 0x-prefixed, 40-hex-digit address in any case.
func ParseAddress(s string) (Address, error) {
	var a Address
	raw := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(raw) != 40 {
		return a, fmt.Errorf("eip712: address %q is not 20 bytes", s)
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return a, fmt.Errorf("eip712: address %q: %w", s, err)
	}
	copy(a[:], b)
	return a, nil
}

// IsZero reports whether a is the zero address.
func (a Address) IsZero() bool { return a == Address{} }

// Hex is the EIP-55 checksummed form, the one wallets display.
func (a Address) Hex() string {
	lower := hex.EncodeToString(a[:])
	hash := Keccak256([]byte(lower))
	out := []byte(lower)
	for i := range out {
		if out[i] >= 'a' && out[i] <= 'f' {
			nibble := hash[i/2]
			if i%2 == 0 {
				nibble >>= 4
			}
			if nibble&0x0f >= 8 {
				out[i] -= 'a' - 'A'
			}
		}
	}
	return "0x" + string(out)
}

func (a Address) String() string { return a.Hex() }
```

```go
// internal/eip712/eip712.go

// Package eip712 hashes EIP-712 typed data for the fixed set of types jumpgate
// signs. It is deliberately small: the types are ours, so arrays and dynamic
// bytes, which none of them use, are refused rather than half-supported.
package eip712

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"golang.org/x/crypto/sha3"
)

// Field is one member of a struct type.
type Field struct{ Name, Type string }

// Types maps a struct type name to its fields, in declaration order.
type Types map[string][]Field

// TypedData is what a wallet's eth_signTypedData_v4 receives.
type TypedData struct {
	Types       Types
	PrimaryType string
	Domain      map[string]any
	Message     map[string]any
}

// Keccak256 hashes the concatenation of parts.
func Keccak256(parts ...[]byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	for _, p := range parts {
		h.Write(p)
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// Digest is keccak256(0x19 0x01 ‖ domainSeparator ‖ hashStruct(message)).
func Digest(td TypedData) ([32]byte, error) {
	domain, err := HashStruct(td.Types, "EIP712Domain", td.Domain)
	if err != nil {
		return [32]byte{}, fmt.Errorf("eip712: domain: %w", err)
	}
	msg, err := HashStruct(td.Types, td.PrimaryType, td.Message)
	if err != nil {
		return [32]byte{}, fmt.Errorf("eip712: message: %w", err)
	}
	return Keccak256([]byte{0x19, 0x01}, domain[:], msg[:]), nil
}

// EncodeType renders primary's signature followed by every struct type it
// references, the references sorted by name, as the EIP requires.
func EncodeType(types Types, primary string) (string, error) {
	if _, ok := types[primary]; !ok {
		return "", fmt.Errorf("eip712: unknown type %q", primary)
	}
	deps := map[string]bool{}
	if err := collectDeps(types, primary, deps); err != nil {
		return "", err
	}
	delete(deps, primary)
	names := make([]string, 0, len(deps))
	for n := range deps {
		names = append(names, n)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range append([]string{primary}, names...) {
		b.WriteString(name)
		b.WriteByte('(')
		for i, f := range types[name] {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(f.Type + " " + f.Name)
		}
		b.WriteByte(')')
	}
	return b.String(), nil
}

func collectDeps(types Types, name string, seen map[string]bool) error {
	if seen[name] {
		return nil
	}
	seen[name] = true
	for _, f := range types[name] {
		if strings.Contains(f.Type, "[") {
			return fmt.Errorf("eip712: %s.%s: array types are not supported", name, f.Name)
		}
		if _, isStruct := types[f.Type]; isStruct {
			if err := collectDeps(types, f.Type, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// HashStruct is keccak256(typeHash ‖ encodeData(data)). data must hold
// exactly the type's fields, each as the Go type listed in the package doc.
func HashStruct(types Types, primary string, data map[string]any) ([32]byte, error) {
	enc, err := EncodeType(types, primary)
	if err != nil {
		return [32]byte{}, err
	}
	fields := types[primary]
	if len(data) != len(fields) {
		return [32]byte{}, fmt.Errorf("eip712: %s has %d fields, got %d values", primary, len(fields), len(data))
	}
	typeHash := Keccak256([]byte(enc))
	buf := make([]byte, 0, 32*(len(fields)+1))
	buf = append(buf, typeHash[:]...)
	for _, f := range fields {
		v, ok := data[f.Name]
		if !ok {
			return [32]byte{}, fmt.Errorf("eip712: %s.%s is missing", primary, f.Name)
		}
		word, err := encodeValue(types, f.Type, v)
		if err != nil {
			return [32]byte{}, fmt.Errorf("eip712: %s.%s: %w", primary, f.Name, err)
		}
		buf = append(buf, word[:]...)
	}
	return Keccak256(buf), nil
}

func encodeValue(types Types, typ string, v any) ([32]byte, error) {
	var word [32]byte
	if _, isStruct := types[typ]; isStruct {
		m, ok := v.(map[string]any)
		if !ok {
			return word, fmt.Errorf("want map[string]any for %s, got %T", typ, v)
		}
		return HashStruct(types, typ, m)
	}
	switch typ {
	case "string":
		s, ok := v.(string)
		if !ok {
			return word, fmt.Errorf("want string, got %T", v)
		}
		return Keccak256([]byte(s)), nil
	case "bytes32":
		b, ok := v.([32]byte)
		if !ok {
			return word, fmt.Errorf("want [32]byte, got %T", v)
		}
		return b, nil
	case "address":
		a, ok := v.(Address)
		if !ok {
			return word, fmt.Errorf("want eip712.Address, got %T", v)
		}
		copy(word[12:], a[:])
		return word, nil
	case "bool":
		b, ok := v.(bool)
		if !ok {
			return word, fmt.Errorf("want bool, got %T", v)
		}
		if b {
			word[31] = 1
		}
		return word, nil
	case "uint8":
		n, ok := v.(uint8)
		if !ok {
			return word, fmt.Errorf("want uint8, got %T", v)
		}
		word[31] = n
		return word, nil
	case "uint64":
		n, ok := v.(uint64)
		if !ok {
			return word, fmt.Errorf("want uint64, got %T", v)
		}
		new(big.Int).SetUint64(n).FillBytes(word[:])
		return word, nil
	case "uint256":
		n, ok := v.(*big.Int)
		if !ok || n == nil {
			return word, fmt.Errorf("want *big.Int, got %T", v)
		}
		if n.Sign() < 0 || n.BitLen() > 256 {
			return word, fmt.Errorf("value %s out of uint256 range", n)
		}
		n.FillBytes(word[:])
		return word, nil
	default:
		return word, fmt.Errorf("unsupported type %q", typ)
	}
}
```

- [ ] **Step 4: Run the tests**

Run: `go test -v ./internal/eip712/`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/eip712
git commit -m "feat(eip712): typed-data hashing for jumpgate's signed messages

Small encoder for the fixed types jumpgate signs, checked against the
specification's Mail vectors. Arrays and dynamic bytes are refused rather
than half-supported.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: `internal/signer` — keys, signing, recovery, key files

**Files:**
- Create: `internal/signer/signer.go`, `internal/signer/key.go`, `internal/signer/keyfile.go`, `internal/signer/signer_test.go`
- Modify: `go.mod`, `go.sum` (`go get github.com/decred/dcrd/dcrec/secp256k1/v4`)

**Interfaces:**
- Consumes: `eip712.Address`, `eip712.TypedData`, `eip712.Digest`, `eip712.Keccak256` (Task 7).
- Produces:
  - `type Signature [65]byte` (r‖s‖v, v ∈ {27,28}); `(Signature) Hex() string`; `ParseSignature(s string) (Signature, error)`
  - `type Signer interface { Address() eip712.Address; SignTypedData(ctx context.Context, td eip712.TypedData) (Signature, error) }`
  - `type Key struct{…}` implementing `Signer`; `GenerateKey() (*Key, error)`; `KeyFromBytes([]byte) (*Key, error)`; `(*Key) Bytes() []byte`; `(*Key) SignDigest([32]byte) Signature`
  - `RecoverDigest(d [32]byte, sig Signature) (eip712.Address, error)`; `Recover(td eip712.TypedData, sig Signature) (eip712.Address, error)`; `var ErrHighS, ErrBadV`
  - `LoadKeyFile(path string) (*Key, error)`; `GenerateKeyFile(path string) (*Key, error)`; `var ErrKeyFilePermissions`

- [ ] **Step 1: Add the dependency**

Run: `go get github.com/decred/dcrd/dcrec/secp256k1/v4@latest && go mod tidy`
Expected: `go.mod` lists the module as a direct requirement.

- [ ] **Step 2: Write the failing tests**

```go
// internal/signer/signer_test.go
package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

func cowKey(t *testing.T) *Key {
	t.Helper()
	k, err := KeyFromBytes(func() []byte { h := eip712.Keccak256([]byte("cow")); return h[:] }())
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func mail(t *testing.T) eip712.TypedData {
	t.Helper()
	a := func(s string) eip712.Address { x, _ := eip712.ParseAddress(s); return x }
	return eip712.TypedData{
		Types: eip712.Types{
			"EIP712Domain": {{"name", "string"}, {"version", "string"}, {"chainId", "uint256"}, {"verifyingContract", "address"}},
			"Person":       {{"name", "string"}, {"wallet", "address"}},
			"Mail":         {{"from", "Person"}, {"to", "Person"}, {"contents", "string"}},
		},
		PrimaryType: "Mail",
		Domain: map[string]any{"name": "Ether Mail", "version": "1", "chainId": big.NewInt(1),
			"verifyingContract": a("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC")},
		Message: map[string]any{
			"from":     map[string]any{"name": "Cow", "wallet": a("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826")},
			"to":       map[string]any{"name": "Bob", "wallet": a("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB")},
			"contents": "Hello, Bob!",
		},
	}
}

// The EIP-712 specification signs its Mail example with keccak256("cow").
// RFC 6979 signing is deterministic, so the signature must match exactly.
func TestSignsTheSpecificationVector(t *testing.T) {
	k := cowKey(t)
	if got := k.Address().Hex(); got != "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826" {
		t.Fatalf("address = %s", got)
	}
	sig, err := k.SignTypedData(context.Background(), mail(t))
	if err != nil {
		t.Fatal(err)
	}
	wantR := "4355c47d63924e8a72e509b65029052eb6c299d53a04e167c5775fd466751c9d"
	wantS := "07299936d304c153f6443dfa05f40ff007d72911b6f72307f996231605b91562"
	if hex.EncodeToString(sig[:32]) != wantR || hex.EncodeToString(sig[32:64]) != wantS || sig[64] != 28 {
		t.Fatalf("signature = %s", sig.Hex())
	}
}

func TestRecoverRoundTrip(t *testing.T) {
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	td := mail(t)
	sig, _ := k.SignTypedData(context.Background(), td)
	got, err := Recover(td, sig)
	if err != nil || got != k.Address() {
		t.Fatalf("Recover = %s, %v; want %s", got.Hex(), err, k.Address().Hex())
	}
	// A different message recovers a different address, never the signer.
	td.Message["contents"] = "Hello, Eve!"
	if got, _ := Recover(td, sig); got == k.Address() {
		t.Fatal("a signature verified for a message it did not sign")
	}
}

// Ethereum accepts only low-s signatures; the high-s twin of a valid signature
// is the classic malleability, and must not verify.
func TestRecoverRejectsHighS(t *testing.T) {
	k := cowKey(t)
	td := mail(t)
	sig, _ := k.SignTypedData(context.Background(), td)

	n, _ := new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	s := new(big.Int).SetBytes(sig[32:64])
	high := new(big.Int).Sub(n, s)
	var twin Signature
	copy(twin[:32], sig[:32])
	high.FillBytes(twin[32:64])
	twin[64] = 27 + (28 - sig[64]) // the twin's recovery id flips
	if _, err := Recover(td, twin); !errors.Is(err, ErrHighS) {
		t.Fatalf("high-s twin: err = %v, want ErrHighS", err)
	}
}

func TestRecoverRejectsABadV(t *testing.T) {
	sig, _ := cowKey(t).SignTypedData(context.Background(), mail(t))
	sig[64] = 1
	if _, err := Recover(mail(t), sig); !errors.Is(err, ErrBadV) {
		t.Fatalf("err = %v, want ErrBadV", err)
	}
}

func TestKeyFileRoundTripAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys", "controller.key")
	k, err := GenerateKeyFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateKeyFile(path); err == nil {
		t.Fatal("GenerateKeyFile overwrote an existing key")
	}
	loaded, err := LoadKeyFile(path)
	if err != nil || loaded.Address() != k.Address() {
		t.Fatalf("LoadKeyFile = %v, %v", loaded, err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeyFile(path); !errors.Is(err, ErrKeyFilePermissions) {
		t.Fatalf("group/world-readable key: err = %v, want ErrKeyFilePermissions", err)
	}
}
```

- [ ] **Step 3: Run them to verify they fail**

Run: `go test ./internal/signer/`
Expected: FAIL — package has no non-test files.

- [ ] **Step 4: Implement**

```go
// internal/signer/signer.go

// Package signer signs and verifies jumpgate's EIP-712 messages with
// secp256k1 keys, the same scheme every Ethereum wallet uses, so a hardware
// wallet can later sign the same messages a key file does.
package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

// Signature is r ‖ s ‖ v with v ∈ {27, 28}, the layout wallets return.
type Signature [65]byte

// Signer is anything that can sign typed data for one address.
type Signer interface {
	Address() eip712.Address
	SignTypedData(ctx context.Context, td eip712.TypedData) (Signature, error)
}

var (
	// ErrHighS rejects the malleable twin of a valid signature.
	ErrHighS = errors.New("signer: signature s is in the upper half of the curve order")
	// ErrBadV rejects a recovery byte other than 27 or 28.
	ErrBadV = errors.New("signer: signature v must be 27 or 28")
)

// Hex is the 0x-prefixed 130-digit form.
func (s Signature) Hex() string { return "0x" + hex.EncodeToString(s[:]) }

// ParseSignature reads the Hex form.
func ParseSignature(s string) (Signature, error) {
	var sig Signature
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 65 {
		return sig, fmt.Errorf("signer: signature %q is not 65 bytes of hex", s)
	}
	copy(sig[:], b)
	return sig, nil
}

// RecoverDigest returns the address that produced sig over d.
func RecoverDigest(d [32]byte, sig Signature) (eip712.Address, error) {
	var zero eip712.Address
	if sig[64] != 27 && sig[64] != 28 {
		return zero, ErrBadV
	}
	var s secp256k1.ModNScalar
	if overflow := s.SetByteSlice(sig[32:64]); overflow || s.IsOverHalfOrder() {
		return zero, ErrHighS
	}
	compact := make([]byte, 65)
	compact[0] = sig[64] // 27 + recovery id, uncompressed: decred's compact header
	copy(compact[1:], sig[:64])
	pub, _, err := ecdsa.RecoverCompact(compact, d[:])
	if err != nil {
		return zero, fmt.Errorf("signer: recover: %w", err)
	}
	return addressOf(pub), nil
}

// Recover returns the address that signed td.
func Recover(td eip712.TypedData, sig Signature) (eip712.Address, error) {
	d, err := eip712.Digest(td)
	if err != nil {
		return eip712.Address{}, err
	}
	return RecoverDigest(d, sig)
}

func addressOf(pub *secp256k1.PublicKey) eip712.Address {
	uncompressed := pub.SerializeUncompressed() // 0x04 ‖ X ‖ Y
	h := eip712.Keccak256(uncompressed[1:])
	var a eip712.Address
	copy(a[:], h[12:])
	return a
}
```

```go
// internal/signer/key.go
package signer

import (
	"context"
	"errors"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

// Key is a secp256k1 private key held in memory.
type Key struct {
	priv *secp256k1.PrivateKey
	addr eip712.Address
}

// GenerateKey makes a new random key.
func GenerateKey() (*Key, error) {
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	return newKey(priv), nil
}

// KeyFromBytes loads a 32-byte scalar, rejecting zero and values at or above
// the curve order.
func KeyFromBytes(b []byte) (*Key, error) {
	if len(b) != 32 {
		return nil, errors.New("signer: a private key is 32 bytes")
	}
	var s secp256k1.ModNScalar
	if overflow := s.SetByteSlice(b); overflow || s.IsZero() {
		return nil, errors.New("signer: private key out of range")
	}
	return newKey(secp256k1.NewPrivateKey(&s)), nil
}

func newKey(priv *secp256k1.PrivateKey) *Key {
	return &Key{priv: priv, addr: addressOf(priv.PubKey())}
}

// Bytes is the 32-byte scalar. Handle with care.
func (k *Key) Bytes() []byte { return k.priv.Serialize() }

// Address is the key's Ethereum address.
func (k *Key) Address() eip712.Address { return k.addr }

// SignDigest signs d deterministically (RFC 6979) with a low s.
func (k *Key) SignDigest(d [32]byte) Signature {
	compact := ecdsa.SignCompact(k.priv, d[:], false) // [27+recid] ‖ r ‖ s
	var sig Signature
	copy(sig[:64], compact[1:])
	sig[64] = compact[0]
	return sig
}

// SignTypedData signs td's EIP-712 digest.
func (k *Key) SignTypedData(_ context.Context, td eip712.TypedData) (Signature, error) {
	d, err := eip712.Digest(td)
	if err != nil {
		return Signature{}, err
	}
	return k.SignDigest(d), nil
}
```

```go
// internal/signer/keyfile.go
package signer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrKeyFilePermissions refuses a key file other users can read: a signing
// key anyone on the machine can copy authorises nothing.
var ErrKeyFilePermissions = errors.New("signer: key file is readable by other users; chmod 600 it")

// LoadKeyFile reads a hex key written by GenerateKeyFile.
func LoadKeyFile(path string) (*Key, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w (%s is %o)", ErrKeyFilePermissions, path, fi.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	b, err := hex.DecodeString(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(raw)), "0x")))
	if err != nil {
		return nil, fmt.Errorf("signer: %s is not a hex key", path)
	}
	return KeyFromBytes(b)
}

// GenerateKeyFile creates a new key at path, 0600, refusing to overwrite.
func GenerateKeyFile(path string) (*Key, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	if _, err := f.WriteString(hex.EncodeToString(k.Bytes()) + "\n"); err != nil {
		f.Close()
		os.Remove(path)
		return nil, fmt.Errorf("signer: %w", err)
	}
	return k, f.Close()
}
```

- [ ] **Step 5: Run the tests**

Run: `go test -race -v ./internal/signer/ && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./internal/signer/`
Expected: PASS; the cgo-free cross build succeeds.

- [ ] **Step 6: Commit**

```bash
git add internal/signer go.mod go.sum
git commit -m "feat(signer): secp256k1 EIP-712 signing, recovery and key files

Signs and recovers jumpgate's typed messages exactly as Ethereum wallets do,
proven against the EIP-712 specification's signature vector. Recovery rejects
high-s twins and bad recovery bytes; key files refuse to load when other users
can read them.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: Key stores — OS keychain and 1Password

**Files:**
- Create: `internal/signer/store.go`, `internal/signer/keychain.go`, `internal/signer/onepassword.go`, `internal/signer/store_test.go`

**Interfaces:**
- Consumes: `Key`, `KeyFromBytes`, `GenerateKey`, `LoadKeyFile`, `GenerateKeyFile` (Task 8).
- Produces:
  - `type Store string` with `StoreFile = "file"`, `StoreKeychain = "keychain"`, `StoreOnePassword = "1password"`
  - `Open(ctx context.Context, store Store, ref string) (*Key, error)`; `Create(ctx context.Context, store Store, ref string) (*Key, error)`
  - `DefaultStore() Store` — keychain when `security` (darwin) or `secret-tool` (linux) is on PATH, else file
  - `var ErrNoKeychain`
  - Ref meaning: file → path; keychain → item name (service `jumpgate`, account = ref); 1password → `op://<vault>/<item>/<field>`

- [ ] **Step 1: Write the failing tests**

```go
// internal/signer/store_test.go
package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
)

// fakeRunner records commands. The property under test is that a key's hex
// never appears in any argument: every process's argv is readable by every
// local user.
type fakeRunner struct {
	calls  [][]string
	stdins []string
	files  map[string]string // template files read during a call
	out    string
}

func (f *fakeRunner) run(_ context.Context, stdin string, name string, args ...string) (string, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	f.stdins = append(f.stdins, stdin)
	for i, a := range args {
		if a == "--template" && i+1 < len(args) {
			b, _ := os.ReadFile(args[i+1])
			if f.files == nil {
				f.files = map[string]string{}
			}
			f.files[args[i+1]] = string(b)
		}
	}
	return f.out, nil
}

func withRunner(t *testing.T, f *fakeRunner, goos string, have ...string) {
	t.Helper()
	oldRun, oldGOOS, oldLook := runCmd, hostOS, lookPath
	runCmd, hostOS = f.run, goos
	lookPath = func(name string) error {
		for _, h := range have {
			if h == name {
				return nil
			}
		}
		return errors.New("not found")
	}
	t.Cleanup(func() { runCmd, hostOS, lookPath = oldRun, oldGOOS, oldLook })
}

func assertNoSecretInArgv(t *testing.T, f *fakeRunner, secret string) {
	t.Helper()
	for _, c := range f.calls {
		if strings.Contains(strings.Join(c, " "), secret) {
			t.Fatalf("key material in argv: %v", c)
		}
	}
}

func TestKeychainMacOSKeepsTheKeyOffArgv(t *testing.T) {
	f := &fakeRunner{}
	withRunner(t, f, "darwin", "security")
	k, err := Create(context.Background(), StoreKeychain, "controller")
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(k.Bytes())
	assertNoSecretInArgv(t, f, secret)
	if !strings.Contains(f.stdins[0], secret) {
		t.Fatal("the key was not handed to `security -i` on stdin")
	}

	f.out = secret + "\n"
	got, err := Open(context.Background(), StoreKeychain, "controller")
	if err != nil || got.Address() != k.Address() {
		t.Fatalf("Open = %v, %v", got, err)
	}
}

func TestKeychainLinuxUsesSecretToolStdin(t *testing.T) {
	f := &fakeRunner{}
	withRunner(t, f, "linux", "secret-tool")
	k, err := Create(context.Background(), StoreKeychain, "controller")
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(k.Bytes())
	assertNoSecretInArgv(t, f, secret)
	if f.calls[0][0] != "secret-tool" || f.calls[0][1] != "store" || f.stdins[0] != secret {
		t.Fatalf("store call = %v stdin %q", f.calls[0], f.stdins[0])
	}
}

func TestKeychainMissingIsAnErrorNotAFallback(t *testing.T) {
	withRunner(t, &fakeRunner{}, "linux")
	if _, err := Create(context.Background(), StoreKeychain, "controller"); !errors.Is(err, ErrNoKeychain) {
		t.Fatalf("err = %v, want ErrNoKeychain", err)
	}
}

func TestOnePasswordCreatesFromATemplateFileItThenDeletes(t *testing.T) {
	f := &fakeRunner{}
	withRunner(t, f, "darwin", "op")
	k, err := Create(context.Background(), StoreOnePassword, "op://Private/jumpgate-controller/credential")
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(k.Bytes())
	assertNoSecretInArgv(t, f, secret)
	var tmpl string
	for path, body := range f.files {
		tmpl = body
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("template %s was left on disk", path)
		}
	}
	if !strings.Contains(tmpl, secret) || !strings.Contains(tmpl, "jumpgate-controller") {
		t.Fatalf("template = %q", tmpl)
	}

	f.out = secret
	got, err := Open(context.Background(), StoreOnePassword, "op://Private/jumpgate-controller/credential")
	if err != nil || got.Address() != k.Address() {
		t.Fatalf("Open = %v, %v", got, err)
	}
	last := f.calls[len(f.calls)-1]
	if strings.Join(last, " ") != "op read op://Private/jumpgate-controller/credential" {
		t.Fatalf("read call = %v", last)
	}
}

func TestOnePasswordRejectsAMalformedRef(t *testing.T) {
	withRunner(t, &fakeRunner{}, "darwin", "op")
	if _, err := Create(context.Background(), StoreOnePassword, "Private/jumpgate"); err == nil {
		t.Fatal("want an error for a ref without op://vault/item/field")
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/signer/ -run 'Keychain|OnePassword'`
Expected: FAIL — undefined `Create`, `Open`, `StoreKeychain`, `runCmd`, `hostOS`, `lookPath`.

- [ ] **Step 3: Implement**

```go
// internal/signer/store.go
package signer

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Store names where a controller key lives.
type Store string

const (
	StoreFile        Store = "file"
	StoreKeychain    Store = "keychain"
	StoreOnePassword Store = "1password"
)

// Seams for tests: which OS we are on, whether a tool exists, and how a tool
// runs. Production never reassigns them.
var (
	hostOS   = runtime.GOOS
	lookPath = func(name string) error { _, err := exec.LookPath(name); return err }
	runCmd   = func(ctx context.Context, stdin, name string, args ...string) (string, error) {
		c := exec.CommandContext(ctx, name, args...)
		c.Stdin = strings.NewReader(stdin)
		var out, errb bytes.Buffer
		c.Stdout, c.Stderr = &out, &errb
		if err := c.Run(); err != nil {
			return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(errb.String()))
		}
		return out.String(), nil
	}
)

// DefaultStore prefers the OS keychain and falls back to a key file only when
// no keychain tool exists, as on a headless box.
func DefaultStore() Store {
	if keychainTool() != "" {
		return StoreKeychain
	}
	return StoreFile
}

// Open loads an existing key.
func Open(ctx context.Context, store Store, ref string) (*Key, error) {
	switch store {
	case StoreFile:
		return LoadKeyFile(ref)
	case StoreKeychain:
		return keychainRead(ctx, ref)
	case StoreOnePassword:
		return onePasswordRead(ctx, ref)
	}
	return nil, fmt.Errorf("signer: unknown key store %q", store)
}

// Create makes a new key in store.
func Create(ctx context.Context, store Store, ref string) (*Key, error) {
	switch store {
	case StoreFile:
		return GenerateKeyFile(ref)
	case StoreKeychain:
		return keychainCreate(ctx, ref)
	case StoreOnePassword:
		return onePasswordCreate(ctx, ref)
	}
	return nil, fmt.Errorf("signer: unknown key store %q", store)
}

func keyFromHex(s string) (*Key, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(s), "0x"))
	if err != nil {
		return nil, fmt.Errorf("signer: stored key is not hex")
	}
	return KeyFromBytes(b)
}
```

```go
// internal/signer/keychain.go
package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ErrNoKeychain means this machine has no keychain tool. It is an error rather
// than a quiet fallback to a file, so a key never lands somewhere the operator
// did not choose.
var ErrNoKeychain = errors.New("signer: no OS keychain here (need `security` on macOS or `secret-tool` on Linux); use --store file")

const keychainService = "jumpgate"

func keychainTool() string {
	switch hostOS {
	case "darwin":
		if lookPath("security") == nil {
			return "security"
		}
	case "linux":
		if lookPath("secret-tool") == nil {
			return "secret-tool"
		}
	}
	return ""
}

// keychainCreate stores a new key. The key's hex is passed on stdin only:
// `security -i` reads commands from stdin and secret-tool reads the secret
// from it, so it never appears in any process's argv.
func keychainCreate(ctx context.Context, ref string) (*Key, error) {
	tool := keychainTool()
	if tool == "" {
		return nil, ErrNoKeychain
	}
	if strings.ContainsAny(ref, " \t\n\"'") {
		return nil, fmt.Errorf("signer: keychain item name %q must not contain spaces or quotes", ref)
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	secret := hex.EncodeToString(k.Bytes())
	switch tool {
	case "security":
		cmd := fmt.Sprintf("add-generic-password -U -a %s -s %s -w %s\n", ref, keychainService, secret)
		_, err = runCmd(ctx, cmd, "security", "-i")
	case "secret-tool":
		_, err = runCmd(ctx, secret, "secret-tool", "store", "--label=jumpgate controller key", "service", keychainService, "account", ref)
	}
	if err != nil {
		return nil, fmt.Errorf("signer: keychain store: %w", err)
	}
	return k, nil
}

func keychainRead(ctx context.Context, ref string) (*Key, error) {
	var out string
	var err error
	switch keychainTool() {
	case "security":
		out, err = runCmd(ctx, "", "security", "find-generic-password", "-a", ref, "-s", keychainService, "-w")
	case "secret-tool":
		out, err = runCmd(ctx, "", "secret-tool", "lookup", "service", keychainService, "account", ref)
	default:
		return nil, ErrNoKeychain
	}
	if err != nil {
		return nil, fmt.Errorf("signer: keychain read: %w", err)
	}
	return keyFromHex(out)
}
```

```go
// internal/signer/onepassword.go
package signer

import (
	"context"
	"encoding/hex"
	"encoding/json"
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
	if _, err := runCmd(ctx, "", "op", "item", "create", "--vault", vault, "--template", path); err != nil {
		return nil, fmt.Errorf("signer: 1Password store: %w", err)
	}
	return k, nil
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
```

Before relying on the `op` flags, check them against the installed CLI: `op item create --help` must list `--template` and `--vault`. If they differ, adjust the arguments and the test together and note it in the commit.

- [ ] **Step 4: Run the tests**

Run: `go test -race ./internal/signer/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/signer
git commit -m "feat(signer): keep controller keys in the OS keychain or 1Password

Keychain on macOS through \`security -i\` and on Linux through secret-tool;
1Password through op with a 0600 template file removed after use. The key's
hex never appears in a process argument list. No keychain tool is an error,
not a silent fall back to a file.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: `internal/intent` — the signed message types

**Files:**
- Create: `internal/intent/intent.go`, `internal/intent/kinds.go`, `internal/intent/intent_test.go`

**Interfaces:**
- Consumes: `eip712.*` (Task 7), `signer.Signature`, `signer.Signer`, `signer.Recover` (Task 8).
- Produces:
  - `var Types eip712.Types` (`EIP712Domain(string name,string version)`, `Intent`, `Receipt`); `var Domain = map[string]any{"name":"jumpgate","version":"1"}`
  - `type Intent struct{ Agent, Controller eip712.Address; Seq uint64; Nonce [32]byte; IssuedAt, Expiry uint64; Kind string; PayloadHash [32]byte }` with `TypedData()`, `Digest() ([32]byte, error)`
  - `type Receipt struct{ Agent eip712.Address; RequestHash [32]byte; Seq uint64; Status uint8; ResultHash [32]byte }` with `TypedData()`, `Digest()`
  - `const StatusOK, StatusRejected, StatusFailed uint8 = 0, 1, 2`
  - `Hash(b []byte) [32]byte` (Keccak256 of the exact bytes)
  - Wire: `type Envelope struct{ Intent IntentJSON; Payload []byte; Sigs []string }`, `type ReceiptEnvelope struct{ Receipt ReceiptJSON; Result []byte; Sig string }`, `(Intent) JSON() IntentJSON`, `(IntentJSON) Parse() (Intent, error)`, same pair for Receipt
  - `NewNonce() ([32]byte, error)`
  - Kinds, payloads, results and reason codes in `kinds.go` (listed in Step 3)

- [ ] **Step 1: Write the failing tests**

```go
// internal/intent/intent_test.go
package intent

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// The schemas are frozen. Changing a field name, type or order changes every
// digest and silently invalidates every signature in flight; this test is
// there so that can only happen on purpose.
func TestSchemasAreFrozen(t *testing.T) {
	cases := map[string]string{
		"Intent":       "Intent(address agent,address controller,uint64 seq,bytes32 nonce,uint64 issuedAt,uint64 expiry,string kind,bytes32 payloadHash)",
		"Receipt":      "Receipt(address agent,bytes32 requestHash,uint64 seq,uint8 status,bytes32 resultHash)",
		"EIP712Domain": "EIP712Domain(string name,string version)",
	}
	for primary, want := range cases {
		got, err := eip712.EncodeType(Types, primary)
		if err != nil || got != want {
			t.Errorf("EncodeType(%s) = %q, %v; want %q", primary, got, err, want)
		}
	}
}

func sample() Intent {
	var i Intent
	i.Agent[19] = 0xA
	i.Controller[19] = 0xC
	i.Seq = 7
	i.Nonce[0] = 0x11
	i.IssuedAt = 1_800_000_000
	i.Expiry = 1_800_000_120
	i.Kind = KindStatusRead
	i.PayloadHash = Hash([]byte("{}"))
	return i
}

// Recompute the intent's struct hash by hand from the ABI rules, independent of
// the eip712 package, so an encoder bug cannot hide behind its own tests.
func TestIntentHashMatchesAHandEncoding(t *testing.T) {
	i := sample()
	word := func(b []byte) []byte { w := make([]byte, 32); copy(w[32-len(b):], b); return w }
	u64 := func(n uint64) []byte { b := make([]byte, 8); binary.BigEndian.PutUint64(b, n); return word(b) }
	typeHash := eip712.Keccak256([]byte("Intent(address agent,address controller,uint64 seq,bytes32 nonce,uint64 issuedAt,uint64 expiry,string kind,bytes32 payloadHash)"))
	kind := eip712.Keccak256([]byte(i.Kind))
	var buf []byte
	buf = append(buf, typeHash[:]...)
	buf = append(buf, word(i.Agent[:])...)
	buf = append(buf, word(i.Controller[:])...)
	buf = append(buf, u64(i.Seq)...)
	buf = append(buf, i.Nonce[:]...)
	buf = append(buf, u64(i.IssuedAt)...)
	buf = append(buf, u64(i.Expiry)...)
	buf = append(buf, kind[:]...)
	buf = append(buf, i.PayloadHash[:]...)
	want := eip712.Keccak256(buf)

	td := i.TypedData()
	got, err := eip712.HashStruct(td.Types, "Intent", td.Message)
	if err != nil || got != want {
		t.Fatalf("hashStruct = %x, %v; want %x", got, err, want)
	}
}

func TestEnvelopeRoundTripsAndStillVerifies(t *testing.T) {
	k, _ := signer.GenerateKey()
	i := sample()
	i.Controller = k.Address()
	sig, err := k.SignTypedData(context.Background(), i.TypedData())
	if err != nil {
		t.Fatal(err)
	}
	env := Envelope{Intent: i.JSON(), Payload: []byte("{}"), Sigs: []string{sig.Hex()}}
	raw, _ := json.Marshal(env)

	var back Envelope
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	parsed, err := back.Intent.Parse()
	if err != nil || parsed != i {
		t.Fatalf("Parse = %+v, %v; want %+v", parsed, err, i)
	}
	s, _ := signer.ParseSignature(back.Sigs[0])
	who, err := signer.Recover(parsed.TypedData(), s)
	if err != nil || who != k.Address() {
		t.Fatalf("recovered %s, %v", who.Hex(), err)
	}
}

func TestParseRejectsMalformedFields(t *testing.T) {
	good := sample().JSON()
	bad := []func(j *IntentJSON){
		func(j *IntentJSON) { j.Agent = "0x12" },
		func(j *IntentJSON) { j.Nonce = "0xzz" },
		func(j *IntentJSON) { j.PayloadHash = "" },
		func(j *IntentJSON) { j.Kind = "" },
	}
	for n, mutate := range bad {
		j := good
		mutate(&j)
		if _, err := j.Parse(); err == nil {
			t.Errorf("case %d parsed", n)
		}
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/intent/`
Expected: FAIL — package has no non-test files.

- [ ] **Step 3: Implement**

```go
// internal/intent/intent.go

// Package intent defines the messages a controller signs to ask an agent to do
// something, and the receipts an agent signs in answer. What is signed is the
// decision itself (a kind plus the hash of its exact payload bytes), never a
// shell command.
package intent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/valve-tech/jumpgate/internal/eip712"
)

// Types is the frozen EIP-712 schema. See TestSchemasAreFrozen.
var Types = eip712.Types{
	"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}},
	"Intent": {
		{Name: "agent", Type: "address"}, {Name: "controller", Type: "address"},
		{Name: "seq", Type: "uint64"}, {Name: "nonce", Type: "bytes32"},
		{Name: "issuedAt", Type: "uint64"}, {Name: "expiry", Type: "uint64"},
		{Name: "kind", Type: "string"}, {Name: "payloadHash", Type: "bytes32"},
	},
	"Receipt": {
		{Name: "agent", Type: "address"}, {Name: "requestHash", Type: "bytes32"},
		{Name: "seq", Type: "uint64"}, {Name: "status", Type: "uint8"},
		{Name: "resultHash", Type: "bytes32"},
	},
}

// Domain carries no chainId: wallets tie chainId to the selected network, and
// these messages are not on-chain. A node's chain id is ordinary data.
func domain() map[string]any { return map[string]any{"name": "jumpgate", "version": "1"} }

// Receipt statuses.
const (
	StatusOK       uint8 = 0
	StatusRejected uint8 = 1 // policy, replay, expiry or validation refused it
	StatusFailed   uint8 = 2 // it ran and the operation failed
)

// Intent is one signed request.
type Intent struct {
	Agent, Controller eip712.Address
	Seq               uint64
	Nonce             [32]byte
	IssuedAt, Expiry  uint64
	Kind              string
	PayloadHash       [32]byte
}

// TypedData is what each signer signs.
func (i Intent) TypedData() eip712.TypedData {
	return eip712.TypedData{Types: Types, PrimaryType: "Intent", Domain: domain(), Message: map[string]any{
		"agent": i.Agent, "controller": i.Controller, "seq": i.Seq, "nonce": i.Nonce,
		"issuedAt": i.IssuedAt, "expiry": i.Expiry, "kind": i.Kind, "payloadHash": i.PayloadHash,
	}}
}

// Digest is the request hash a receipt answers.
func (i Intent) Digest() ([32]byte, error) { return eip712.Digest(i.TypedData()) }

// Receipt is the agent's signed answer.
type Receipt struct {
	Agent       eip712.Address
	RequestHash [32]byte
	Seq         uint64
	Status      uint8
	ResultHash  [32]byte
}

func (r Receipt) TypedData() eip712.TypedData {
	return eip712.TypedData{Types: Types, PrimaryType: "Receipt", Domain: domain(), Message: map[string]any{
		"agent": r.Agent, "requestHash": r.RequestHash, "seq": r.Seq, "status": r.Status, "resultHash": r.ResultHash,
	}}
}

func (r Receipt) Digest() ([32]byte, error) { return eip712.Digest(r.TypedData()) }

// Hash is keccak256 of the exact bytes sent. The receiver hashes what it
// received before decoding it, so no JSON canonicalisation is ever needed.
func Hash(b []byte) [32]byte { return eip712.Keccak256(b) }

// NewNonce returns 32 random bytes.
func NewNonce() ([32]byte, error) {
	var n [32]byte
	_, err := rand.Read(n[:])
	return n, err
}

// IntentJSON is the wire form of Intent.
type IntentJSON struct {
	Agent       string `json:"agent"`
	Controller  string `json:"controller"`
	Seq         uint64 `json:"seq"`
	Nonce       string `json:"nonce"`
	IssuedAt    uint64 `json:"issuedAt"`
	Expiry      uint64 `json:"expiry"`
	Kind        string `json:"kind"`
	PayloadHash string `json:"payloadHash"`
}

// Envelope is one request on the wire. Payload is base64 in JSON.
type Envelope struct {
	Intent  IntentJSON `json:"intent"`
	Payload []byte     `json:"payload"`
	Sigs    []string   `json:"sigs"`
}

// ReceiptJSON is the wire form of Receipt.
type ReceiptJSON struct {
	Agent       string `json:"agent"`
	RequestHash string `json:"requestHash"`
	Seq         uint64 `json:"seq"`
	Status      uint8  `json:"status"`
	ResultHash  string `json:"resultHash"`
}

// ReceiptEnvelope is one answer on the wire.
type ReceiptEnvelope struct {
	Receipt ReceiptJSON `json:"receipt"`
	Result  []byte      `json:"result"`
	Sig     string      `json:"sig"`
}

func hex32(b [32]byte) string { return "0x" + hex.EncodeToString(b[:]) }

func parse32(s string) ([32]byte, error) {
	var out [32]byte
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 32 {
		return out, fmt.Errorf("intent: %q is not 32 bytes of hex", s)
	}
	copy(out[:], b)
	return out, nil
}

func (i Intent) JSON() IntentJSON {
	return IntentJSON{Agent: i.Agent.Hex(), Controller: i.Controller.Hex(), Seq: i.Seq, Nonce: hex32(i.Nonce),
		IssuedAt: i.IssuedAt, Expiry: i.Expiry, Kind: i.Kind, PayloadHash: hex32(i.PayloadHash)}
}

func (j IntentJSON) Parse() (Intent, error) {
	var i Intent
	var err error
	if i.Agent, err = eip712.ParseAddress(j.Agent); err != nil {
		return i, err
	}
	if i.Controller, err = eip712.ParseAddress(j.Controller); err != nil {
		return i, err
	}
	if i.Nonce, err = parse32(j.Nonce); err != nil {
		return i, err
	}
	if i.PayloadHash, err = parse32(j.PayloadHash); err != nil {
		return i, err
	}
	if j.Kind == "" {
		return i, fmt.Errorf("intent: kind is empty")
	}
	i.Seq, i.IssuedAt, i.Expiry, i.Kind = j.Seq, j.IssuedAt, j.Expiry, j.Kind
	return i, nil
}

func (r Receipt) JSON() ReceiptJSON {
	return ReceiptJSON{Agent: r.Agent.Hex(), RequestHash: hex32(r.RequestHash), Seq: r.Seq, Status: r.Status, ResultHash: hex32(r.ResultHash)}
}

func (j ReceiptJSON) Parse() (Receipt, error) {
	var r Receipt
	var err error
	if r.Agent, err = eip712.ParseAddress(j.Agent); err != nil {
		return r, err
	}
	if r.RequestHash, err = parse32(j.RequestHash); err != nil {
		return r, err
	}
	if r.ResultHash, err = parse32(j.ResultHash); err != nil {
		return r, err
	}
	r.Seq, r.Status = j.Seq, j.Status
	return r, nil
}
```

```go
// internal/intent/kinds.go
package intent

// Kinds in this sub-project. All are routine tier.
const (
	KindAgentInfo     = "agent.info"
	KindStatusRead    = "status.read"
	KindDiskRead      = "disk.read"
	KindEndpointsRead = "endpoints.read"
	KindFirewallRead  = "firewall.read"
	KindLogsRead      = "logs.read"
	KindServiceAction = "service.action"
)

// Reason codes carried by a rejected receipt's result. They are stable: the
// CLI maps each to a one-line remedy.
const (
	ReasonWrongAgent       = "wrong_agent"
	ReasonExpired          = "expired"
	ReasonClockSkew        = "clock_skew"
	ReasonBadSignature     = "bad_signature"
	ReasonUnauthorizedKind = "unauthorized_kind"
	ReasonUnknownKind      = "unknown_kind"
	ReasonStaleSeq         = "stale_seq"
	ReasonReplayedNonce    = "replayed_nonce"
	ReasonBusy             = "busy"
	ReasonInvalidPayload   = "invalid_payload"
	ReasonValidation       = "validation"
	ReasonNotSetUp         = "not_set_up"
	ReasonReplayState      = "replay_state"
)

// Rejection is the result of a rejected intent. LastSeq lets a controller
// whose counter fell behind resynchronise; AgentTime lets it report skew.
type Rejection struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	LastSeq   uint64 `json:"lastSeq,omitempty"`
	AgentTime int64  `json:"agentTime,omitempty"`
}

// Failure is the result of an intent that ran and failed.
type Failure struct {
	Message string `json:"message"`
}

// Payloads.
type (
	LogsReadPayload      struct{ N int `json:"n"` }
	EndpointsReadPayload struct{ SSHLogin string `json:"sshLogin"` }
	ServiceActionPayload struct {
		Service string `json:"service"` // "exec" | "beacon"
		Action  string `json:"action"`  // "start" | "stop" | "restart"
	}
)

// AgentInfo answers agent.info.
type AgentInfo struct {
	Version      string   `json:"version"`
	Address      string   `json:"address"`
	LastSeq      uint64   `json:"lastSeq"` // for the asking controller
	PlanVersions []string `json:"planVersions"`
	Signers      int      `json:"signers"`
	SetUp        bool     `json:"setUp"`
}

// Logs limits.
const (
	LogsDefaultN = 200
	LogsMaxN     = 2000
)
```

Results of the other kinds are the existing types, JSON-encoded: `status.read` → `monitor.Snapshot`; `disk.read` → `DiskResult{Usage ops.DU; FreeBytes uint64}` (define `DiskResult` in `internal/agent`, Task 12); `endpoints.read` → `ops.EndpointInfo`; `firewall.read` → `[]ops.CheckItem`; `logs.read` → `[]logwatch.Hit`; `service.action` → `struct{ Active bool }`.

- [ ] **Step 4: Run the tests**

Run: `go test -race -v ./internal/intent/`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/intent
git commit -m "feat(intent): signed intent and receipt types with frozen schemas

EIP-712 Intent and Receipt messages, their wire envelopes, the routine kinds of
sub-project 1 and the stable rejection codes. The struct hash is checked against
a hand ABI encoding and the schemas are frozen by test.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 11: Agent policy and replay state

**Files:**
- Create: `internal/agent/policy.go`, `internal/agent/replay.go`, `internal/agent/errors.go`, `internal/agent/policy_test.go`, `internal/agent/replay_test.go`

**Interfaces:**
- Consumes: `eip712.Address`, `intent.Kind*`, `intent.Reason*` (Tasks 7, 10).
- Produces:
  - `type Tier string` (`TierRoutine = "routine"`, `TierApproval = "approval"`); `type SignerEntry struct{ Address string; Tier Tier; Label string }` (JSON `address`, `tier`, `label`)
  - `type Policy struct{ Signers []SignerEntry; LocalUIDs []int; Kinds map[string][]Tier }` (JSON `signers`, `localUids`, `kinds`)
  - `LoadPolicy(path string) (Policy, error)`; `(Policy) Save(path string) error`; `(*Policy) AddSigner(SignerEntry) (added bool)`; `(*Policy) AddLocalUID(int) bool`; `(Policy) Authorise(kind string, controller eip712.Address, signers []eip712.Address) error`
  - `type Reject struct{ Code, Message string; LastSeq uint64 }` implementing `error`
  - `OpenReplay(path string) (*Replay, error)`; `InitReplay(path string) error` (creates `{}` only if absent); `(*Replay) Admit(controller eip712.Address, seq uint64, nonce [32]byte, expiry uint64, now time.Time) error`; `(*Replay) LastSeq(eip712.Address) uint64`; `var ErrReplayState`

- [ ] **Step 1: Write the failing tests**

```go
// internal/agent/policy_test.go
package agent

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func addr(b byte) eip712.Address { var a eip712.Address; a[19] = b; return a }

func TestAuthoriseNeedsTheControllerToBeAnEnrolledSigner(t *testing.T) {
	p := Policy{Signers: []SignerEntry{{Address: addr(1).Hex(), Tier: TierRoutine, Label: "laptop"}}}

	if err := p.Authorise(intent.KindStatusRead, addr(1), []eip712.Address{addr(1)}); err != nil {
		t.Fatalf("enrolled controller refused: %v", err)
	}
	// Signed by someone else while claiming to be the controller.
	var r *Reject
	if err := p.Authorise(intent.KindStatusRead, addr(1), []eip712.Address{addr(2)}); !errors.As(err, &r) || r.Code != intent.ReasonUnauthorizedKind {
		t.Fatalf("foreign signature: err = %v", err)
	}
	// An unenrolled controller signing for itself.
	if err := p.Authorise(intent.KindStatusRead, addr(2), []eip712.Address{addr(2)}); !errors.As(err, &r) {
		t.Fatalf("unenrolled controller: err = %v", err)
	}
}

func TestAuthoriseUnknownKind(t *testing.T) {
	p := Policy{Signers: []SignerEntry{{Address: addr(1).Hex(), Tier: TierRoutine}}}
	var r *Reject
	if err := p.Authorise("wipe.everything", addr(1), []eip712.Address{addr(1)}); !errors.As(err, &r) || r.Code != intent.ReasonUnknownKind {
		t.Fatalf("err = %v, want unknown_kind", err)
	}
}

// An approval-tier requirement is met only by a distinct signer holding that
// tier; the routine controller's own signature cannot count twice.
func TestAuthoriseApprovalTierNeedsASecondSigner(t *testing.T) {
	p := Policy{
		Signers: []SignerEntry{{Address: addr(1).Hex(), Tier: TierRoutine}, {Address: addr(9).Hex(), Tier: TierApproval}},
		Kinds:   map[string][]Tier{intent.KindServiceAction: {TierRoutine, TierApproval}},
	}
	if err := p.Authorise(intent.KindServiceAction, addr(1), []eip712.Address{addr(1)}); err == nil {
		t.Fatal("approval-tier kind passed with only the routine signature")
	}
	if err := p.Authorise(intent.KindServiceAction, addr(1), []eip712.Address{addr(1), addr(9)}); err != nil {
		t.Fatalf("both tiers present: %v", err)
	}
}

func TestPolicyAddSignerIsIdempotentAndSaveRoundTrips(t *testing.T) {
	var p Policy
	if !p.AddSigner(SignerEntry{Address: addr(1).Hex(), Tier: TierRoutine, Label: "a"}) {
		t.Fatal("first add reported no change")
	}
	if p.AddSigner(SignerEntry{Address: addr(1).Hex(), Tier: TierRoutine, Label: "a"}) {
		t.Fatal("second add of the same signer reported a change")
	}
	p.AddSigner(SignerEntry{Address: addr(2).Hex(), Tier: TierRoutine, Label: "b"})
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	back, err := LoadPolicy(path)
	if err != nil || len(back.Signers) != 2 {
		t.Fatalf("LoadPolicy = %+v, %v", back, err)
	}
}
```

```go
// internal/agent/replay_test.go
package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
)

var t0 = time.Unix(1_800_000_000, 0)

func nonce(b byte) [32]byte { var n [32]byte; n[0] = b; return n }

func newReplay(t *testing.T) (*Replay, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "replay.json")
	if err := InitReplay(path); err != nil {
		t.Fatal(err)
	}
	r, err := OpenReplay(path)
	if err != nil {
		t.Fatal(err)
	}
	return r, path
}

func TestAdmitEnforcesIncreasingSeqAndPersistsIt(t *testing.T) {
	r, path := newReplay(t)
	exp := uint64(t0.Unix() + 120)
	if err := r.Admit(addr(1), 5, nonce(1), exp, t0); err != nil {
		t.Fatal(err)
	}
	var rej *Reject
	if err := r.Admit(addr(1), 5, nonce(2), exp, t0); !errors.As(err, &rej) || rej.Code != intent.ReasonStaleSeq || rej.LastSeq != 5 {
		t.Fatalf("repeat seq: err = %v", err)
	}
	// A fresh process reading the same file must see the persisted seq.
	again, err := OpenReplay(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.LastSeq(addr(1)) != 5 {
		t.Fatalf("persisted LastSeq = %d, want 5", again.LastSeq(addr(1)))
	}
}

func TestAdmitRejectsAReplayedNonceWithinItsWindow(t *testing.T) {
	r, _ := newReplay(t)
	exp := uint64(t0.Unix() + 120)
	_ = r.Admit(addr(1), 1, nonce(7), exp, t0)
	var rej *Reject
	if err := r.Admit(addr(1), 2, nonce(7), exp, t0); !errors.As(err, &rej) || rej.Code != intent.ReasonReplayedNonce {
		t.Fatalf("err = %v, want replayed_nonce", err)
	}
	// Once the window has passed, the nonce is forgotten (seq still protects).
	if err := r.Admit(addr(1), 3, nonce(7), uint64(t0.Unix()+1000), t0.Add(10*time.Minute)); err != nil {
		t.Fatalf("after the window: %v", err)
	}
}

// Review Focus 2: each controller has its own sequence.
func TestControllersAreIndependent(t *testing.T) {
	r, _ := newReplay(t)
	exp := uint64(t0.Unix() + 120)
	if err := r.Admit(addr(1), 50, nonce(1), exp, t0); err != nil {
		t.Fatal(err)
	}
	if err := r.Admit(addr(2), 1, nonce(2), exp, t0); err != nil {
		t.Fatalf("second controller blocked by the first's sequence: %v", err)
	}
}

// Review Focus 1: a corrupt or missing state file fails closed. Starting from
// empty would let any captured intent be replayed.
func TestOpenReplayFailsClosed(t *testing.T) {
	dir := t.TempDir()
	corrupt := filepath.Join(dir, "replay.json")
	if err := os.WriteFile(corrupt, []byte(`{"controllers":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReplay(corrupt); !errors.Is(err, ErrReplayState) {
		t.Fatalf("corrupt file: err = %v, want ErrReplayState", err)
	}
	if _, err := OpenReplay(filepath.Join(dir, "missing.json")); !errors.Is(err, ErrReplayState) {
		t.Fatalf("missing file: err = %v, want ErrReplayState", err)
	}
	if err := InitReplay(corrupt); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReplay(corrupt); !errors.Is(err, ErrReplayState) {
		t.Fatal("InitReplay overwrote an existing (corrupt) file; it must only create")
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/agent/`
Expected: FAIL — package has no non-test files.

- [ ] **Step 3: Implement**

```go
// internal/agent/errors.go
package agent

// Reject is a refusal the agent signs and returns. Code is one of the
// intent.Reason* constants.
type Reject struct {
	Code    string
	Message string
	LastSeq uint64
}

func (r *Reject) Error() string { return r.Code + ": " + r.Message }

func reject(code, msg string) *Reject { return &Reject{Code: code, Message: msg} }
```

```go
// internal/agent/policy.go
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// Tier is how much a signer is trusted.
type Tier string

const (
	TierRoutine  Tier = "routine"
	TierApproval Tier = "approval"
)

// SignerEntry is one enrolled signer.
type SignerEntry struct {
	Address string `json:"address"`
	Tier    Tier   `json:"tier"`
	Label   string `json:"label"`
}

// Policy is /etc/jumpgate/policy.json: who may ask this box for what.
type Policy struct {
	Signers []SignerEntry `json:"signers"`
	// LocalUIDs may connect to the socket directly. A controller on the box
	// itself is enrolled by uid so it works without a re-login for a new group.
	LocalUIDs []int `json:"localUids,omitempty"`
	// Kinds overrides DefaultRequirements per kind.
	Kinds map[string][]Tier `json:"kinds,omitempty"`
}

// DefaultRequirements are sub-project 1's kinds, all routine. A kind absent
// from both this table and Policy.Kinds is unknown and refused.
var DefaultRequirements = map[string][]Tier{
	intent.KindAgentInfo:     {TierRoutine},
	intent.KindStatusRead:    {TierRoutine},
	intent.KindDiskRead:      {TierRoutine},
	intent.KindEndpointsRead: {TierRoutine},
	intent.KindFirewallRead:  {TierRoutine},
	intent.KindLogsRead:      {TierRoutine},
	intent.KindServiceAction: {TierRoutine},
}

// LoadPolicy reads the policy file, refusing one other users can write or read.
func LoadPolicy(path string) (Policy, error) {
	var p Policy
	fi, err := os.Stat(path)
	if err != nil {
		return p, fmt.Errorf("agent: policy: %w", err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return p, fmt.Errorf("agent: policy %s is mode %o; it must be 0600", path, fi.Mode().Perm())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return p, fmt.Errorf("agent: policy: %w", err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("agent: policy %s: %w", path, err)
	}
	return p, nil
}

// Save writes the policy atomically, 0600.
func (p Policy) Save(path string) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

// AddSigner enrolls e unless its address is already enrolled. It never
// changes an existing entry's tier: upgrading a signer is a separate,
// approval-tier decision.
func (p *Policy) AddSigner(e SignerEntry) bool {
	for _, s := range p.Signers {
		if strings.EqualFold(s.Address, e.Address) {
			return false
		}
	}
	p.Signers = append(p.Signers, e)
	return true
}

// AddLocalUID allows a local uid to use the socket.
func (p *Policy) AddLocalUID(uid int) bool {
	for _, u := range p.LocalUIDs {
		if u == uid {
			return false
		}
	}
	p.LocalUIDs = append(p.LocalUIDs, uid)
	return true
}

func (p Policy) tierOf(a eip712.Address) (Tier, bool) {
	for _, s := range p.Signers {
		if parsed, err := eip712.ParseAddress(s.Address); err == nil && parsed == a {
			return s.Tier, true
		}
	}
	return "", false
}

// Authorise checks that controller is an enrolled routine signer that signed,
// and that every tier the kind requires is covered by a distinct signer.
func (p Policy) Authorise(kind string, controller eip712.Address, signers []eip712.Address) error {
	req, ok := p.Kinds[kind]
	if !ok {
		req, ok = DefaultRequirements[kind]
	}
	if !ok {
		return reject(intent.ReasonUnknownKind, fmt.Sprintf("this agent does not know %q", kind))
	}
	if tier, enrolled := p.tierOf(controller); !enrolled || tier != TierRoutine {
		return reject(intent.ReasonUnauthorizedKind, fmt.Sprintf("controller %s is not enrolled on this box", controller.Hex()))
	}
	used := map[eip712.Address]bool{}
	signed := map[eip712.Address]bool{}
	for _, s := range signers {
		signed[s] = true
	}
	if !signed[controller] {
		return reject(intent.ReasonUnauthorizedKind, "the intent is not signed by the controller it names")
	}
	for _, need := range req {
		found := false
		// Prefer the controller for the routine slot so an approval signer
		// stays free for the approval slot.
		if need == TierRoutine && !used[controller] {
			used[controller], found = true, true
		}
		for _, s := range signers {
			if found {
				break
			}
			if tier, ok := p.tierOf(s); ok && tier == need && !used[s] {
				used[s], found = true, true
			}
		}
		if !found {
			return reject(intent.ReasonUnauthorizedKind, fmt.Sprintf("%s needs a %s signature", kind, need))
		}
	}
	return nil
}

// writeAtomic writes b to path through a fsynced temp file and rename.
func writeAtomic(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}
```

```go
// internal/agent/replay.go
package agent

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// ErrReplayState means the replay record is missing or unreadable. The agent
// refuses every intent until an operator resets it, because starting from an
// empty record would make every captured intent replayable.
var ErrReplayState = errors.New("agent: replay state is missing or corrupt; run `jumpgate agent reset-replay` after checking why")

// Replay is /var/lib/jumpgate/replay.json.
type Replay struct {
	path string
	mu   sync.Mutex
	st   replayFile
}

type replayFile struct {
	Controllers map[string]uint64 `json:"controllers"` // address hex → last accepted seq
	Nonces      map[string]uint64 `json:"nonces"`      // nonce hex → unix time it may be forgotten
}

// InitReplay creates an empty record if none exists. It never overwrites.
func InitReplay(path string) error {
	b, _ := json.Marshal(replayFile{Controllers: map[string]uint64{}, Nonces: map[string]uint64{}})
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return writeAtomic(path, b)
}

// OpenReplay loads the record, failing closed.
func OpenReplay(path string) (*Replay, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReplayState, err)
	}
	var st replayFile
	if err := json.Unmarshal(b, &st); err != nil || st.Controllers == nil || st.Nonces == nil {
		return nil, fmt.Errorf("%w: %s does not parse", ErrReplayState, path)
	}
	return &Replay{path: path, st: st}, nil
}

// LastSeq is the last sequence accepted from controller.
func (r *Replay) LastSeq(controller eip712.Address) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.st.Controllers[controller.Hex()]
}

// Admit accepts (controller, seq, nonce) once, and persists the acceptance
// before returning. The caller executes only after Admit succeeds: a crash
// after this point loses one intent, which is safe, where executing first
// would let the same intent run twice.
func (r *Replay) Admit(controller eip712.Address, seq uint64, nonce [32]byte, expiry uint64, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := controller.Hex()
	last := r.st.Controllers[key]
	if seq <= last {
		return &Reject{Code: intent.ReasonStaleSeq, Message: fmt.Sprintf("sequence %d is not above %d", seq, last), LastSeq: last}
	}
	nk := hex.EncodeToString(nonce[:])
	if until, seen := r.st.Nonces[nk]; seen && uint64(now.Unix()) <= until {
		return reject(intent.ReasonReplayedNonce, "this nonce was already used")
	}

	next := replayFile{Controllers: map[string]uint64{}, Nonces: map[string]uint64{}}
	for k, v := range r.st.Controllers {
		next.Controllers[k] = v
	}
	for k, v := range r.st.Nonces {
		if v >= uint64(now.Unix()) { // prune expired nonces
			next.Nonces[k] = v
		}
	}
	next.Controllers[key] = seq
	next.Nonces[nk] = expiry + uint64(skew/time.Second)

	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := writeAtomic(r.path, b); err != nil {
		return fmt.Errorf("agent: persist replay state: %w", err)
	}
	r.st = next
	return nil
}
```

`skew` is defined in Task 12 (`const skew = 60 * time.Second`); for this task, add it now at the top of `replay.go` and move it when Task 12 needs it in `agent.go` — keep exactly one definition.

- [ ] **Step 4: Run the tests**

Run: `go test -race -v ./internal/agent/`
Expected: PASS (8 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/agent
git commit -m "feat(agent): policy and fail-closed replay state

The policy says which signers may ask for which kinds; the controller named in
an intent must itself be an enrolled routine signer and must have signed it.
Replay state persists the last sequence per controller and recent nonces with
fsync and rename before anything runs, and a missing or corrupt record refuses
every intent instead of starting empty.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 12: The agent: verify, dispatch, sign; socket and peer credentials

**Files:**
- Create: `internal/agent/agent.go`, `internal/agent/dispatch.go`, `internal/agent/listen.go`, `internal/agent/peercred_linux.go`, `internal/agent/peercred_darwin.go`, `internal/agent/peercred_other.go`, `internal/agent/agent_test.go`, `internal/agent/listen_test.go`
- Modify: `internal/monitor/monitor.go` (add `PollOnce`), `internal/ops/ops.go` (add `NodeUnits`)

**Interfaces:**
- Consumes: Tasks 8, 10, 11; `catalog.ValidateDataDir` (Task 1); `ops.DiskUsage`, `ops.FreeBytesAt`, `ops.Endpoints`, `ops.FirewallChecklist`, `ops.ParseOverlayCIDRs`, `ops.ServiceAction`; `logwatch.Classify`; `monitor.New`; `buildinfo.Version`.
- Produces:
  - `type Config struct{ Key *signer.Key; PolicyPath, ReplayPath, NodePath string; Exec executor.Executor; Now func() time.Time }`
  - `New(cfg Config) *Agent`; `(*Agent) Address() eip712.Address`; `(*Agent) Handle(ctx context.Context, env intent.Envelope) intent.ReceiptEnvelope`
  - `Listen(path string, gid int) (net.Listener, error)` (removes a stale *socket* only, chmods 0660, chowns `0:gid` when `gid >= 0`)
  - `Serve(ctx context.Context, a *Agent, ln net.Listener) error` — HTTP on `POST /v1/intent`, 1 MiB body cap, peer-credential gate
  - `type DiskResult struct{ Usage ops.DU `json:"usage"`; FreeBytes uint64 `json:"freeBytes"` }`; `type FirewallReadPayload struct{ OverlayCIDRs []string `json:"overlayCidrs"` }`
  - `const skew = 60 * time.Second`, `maxLifetime = 300 * time.Second`, `maxBody = 1 << 20`
  - `func (m *Monitor) PollOnce(ctx context.Context) Snapshot` in `internal/monitor`; `func NodeUnits() []string` in `internal/ops`

- [ ] **Step 1: Write the failing agent tests**

```go
// internal/agent/agent_test.go
package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// fakeExec answers every command with a fixed result and records them.
type fakeExec struct {
	cmds []string
	res  executor.Result
}

func (f *fakeExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	f.cmds = append(f.cmds, cmd)
	return f.res, nil
}
func (f *fakeExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (f *fakeExec) ReadFile(context.Context, string) ([]byte, error)              { return nil, nil }
func (f *fakeExec) Close() error                                                  { return nil }

type rig struct {
	a          *Agent
	agentKey   *signer.Key
	controller *signer.Key
	exec       *fakeExec
	now        time.Time
	seq        uint64
	dir        string
}

func newRig(t *testing.T, setUp bool) *rig {
	t.Helper()
	dir := t.TempDir()
	r := &rig{dir: dir, exec: &fakeExec{res: executor.Result{Stdout: "active\nactive\n"}}, now: time.Unix(1_800_000_000, 0)}
	r.agentKey, _ = signer.GenerateKey()
	r.controller, _ = signer.GenerateKey()
	p := Policy{Signers: []SignerEntry{{Address: r.controller.Address().Hex(), Tier: TierRoutine, Label: "test"}}}
	if err := p.Save(filepath.Join(dir, "policy.json")); err != nil {
		t.Fatal(err)
	}
	if err := InitReplay(filepath.Join(dir, "replay.json")); err != nil {
		t.Fatal(err)
	}
	if setUp {
		w := catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse", DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
		b, _ := json.Marshal(w)
		if err := os.WriteFile(filepath.Join(dir, "node.json"), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r.a = New(Config{
		Key: r.agentKey, Exec: r.exec, Now: func() time.Time { return r.now },
		PolicyPath: filepath.Join(dir, "policy.json"), ReplayPath: filepath.Join(dir, "replay.json"), NodePath: filepath.Join(dir, "node.json"),
	})
	return r
}

// send builds, signs and handles one intent; mutate may alter it before signing.
func (r *rig) send(t *testing.T, kind string, payload any, mutate func(*intent.Intent)) (intent.Receipt, []byte) {
	t.Helper()
	body, _ := json.Marshal(payload)
	r.seq++
	n, _ := intent.NewNonce()
	i := intent.Intent{Agent: r.agentKey.Address(), Controller: r.controller.Address(), Seq: r.seq, Nonce: n,
		IssuedAt: uint64(r.now.Unix()), Expiry: uint64(r.now.Unix() + 120), Kind: kind, PayloadHash: intent.Hash(body)}
	if mutate != nil {
		mutate(&i)
	}
	sig, _ := r.controller.SignTypedData(context.Background(), i.TypedData())
	out := r.a.Handle(context.Background(), intent.Envelope{Intent: i.JSON(), Payload: body, Sigs: []string{sig.Hex()}})

	rc, err := out.Receipt.Parse()
	if err != nil {
		t.Fatal(err)
	}
	s, _ := signer.ParseSignature(out.Sig)
	who, err := signer.Recover(rc.TypedData(), s)
	if err != nil || who != r.agentKey.Address() {
		t.Fatalf("receipt not signed by the agent: %v", err)
	}
	if rc.ResultHash != intent.Hash(out.Result) {
		t.Fatal("receipt does not cover its result")
	}
	return rc, out.Result
}

func rejection(t *testing.T, rc intent.Receipt, result []byte) intent.Rejection {
	t.Helper()
	if rc.Status != intent.StatusRejected {
		t.Fatalf("status = %d, want rejected; result %s", rc.Status, result)
	}
	var rej intent.Rejection
	_ = json.Unmarshal(result, &rej)
	return rej
}

func TestAgentInfoAnswersEvenBeforeSetup(t *testing.T) {
	r := newRig(t, false)
	rc, res := r.send(t, intent.KindAgentInfo, struct{}{}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	var info intent.AgentInfo
	_ = json.Unmarshal(res, &info)
	if info.Address != r.agentKey.Address().Hex() || info.SetUp || info.LastSeq != 1 {
		t.Fatalf("info = %+v", info)
	}
}

// Review Focus 3.
func TestNodeKindsBeforeSetupAreNotSetUp(t *testing.T) {
	r := newRig(t, false)
	for _, k := range []string{intent.KindStatusRead, intent.KindDiskRead, intent.KindFirewallRead, intent.KindLogsRead} {
		rc, res := r.send(t, k, struct{}{}, nil)
		if got := rejection(t, rc, res).Code; got != intent.ReasonNotSetUp {
			t.Errorf("%s: code %s, want not_set_up", k, got)
		}
	}
	if len(r.exec.cmds) != 0 {
		t.Fatalf("probed an unset-up box: %v", r.exec.cmds)
	}
}

func TestRejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*intent.Intent)
		want   string
	}{
		{"wrong agent", func(i *intent.Intent) { i.Agent = eip712.Address{1} }, intent.ReasonWrongAgent},
		{"expired", func(i *intent.Intent) { i.IssuedAt -= 1000; i.Expiry -= 1000 }, intent.ReasonExpired},
		{"from the future", func(i *intent.Intent) { i.IssuedAt += 600; i.Expiry += 600 }, intent.ReasonClockSkew},
		{"too long-lived", func(i *intent.Intent) { i.Expiry = i.IssuedAt + 3600 }, intent.ReasonInvalidPayload},
		{"payload swapped", func(i *intent.Intent) { i.PayloadHash = intent.Hash([]byte(`{"n":1}`)) }, intent.ReasonInvalidPayload},
	}
	for _, c := range cases {
		r := newRig(t, true)
		rc, res := r.send(t, intent.KindStatusRead, struct{}{}, c.mutate)
		if got := rejection(t, rc, res).Code; got != c.want {
			t.Errorf("%s: code %s, want %s", c.name, got, c.want)
		}
	}
}

func TestAStrangerIsUnauthorised(t *testing.T) {
	r := newRig(t, true)
	r.controller, _ = signer.GenerateKey() // not enrolled
	rc, res := r.send(t, intent.KindStatusRead, struct{}{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonUnauthorizedKind {
		t.Fatalf("code %s", got)
	}
}

// The spec's acceptance: a captured intent replayed to the same agent is
// rejected. A rejection must not consume the sequence either.
func TestReplayIsRejectedAndRejectionsDoNotConsumeSeq(t *testing.T) {
	r := newRig(t, true)
	body := []byte("{}")
	n, _ := intent.NewNonce()
	i := intent.Intent{Agent: r.agentKey.Address(), Controller: r.controller.Address(), Seq: 1, Nonce: n,
		IssuedAt: uint64(r.now.Unix()), Expiry: uint64(r.now.Unix() + 120), Kind: intent.KindStatusRead, PayloadHash: intent.Hash(body)}
	sig, _ := r.controller.SignTypedData(context.Background(), i.TypedData())
	env := intent.Envelope{Intent: i.JSON(), Payload: body, Sigs: []string{sig.Hex()}}

	first := r.a.Handle(context.Background(), env)
	if first.Receipt.Status != intent.StatusOK {
		t.Fatalf("first: %s", first.Result)
	}
	again := r.a.Handle(context.Background(), env)
	var rej intent.Rejection
	_ = json.Unmarshal(again.Result, &rej)
	if rej.Code != intent.ReasonStaleSeq || rej.LastSeq != 1 {
		t.Fatalf("replay: %+v", rej)
	}
}

func TestRejectionBeforeAdmitDoesNotAdvanceSeq(t *testing.T) {
	r := newRig(t, true)
	r.send(t, intent.KindStatusRead, struct{}{}, func(i *intent.Intent) { i.Agent = eip712.Address{1} }) // seq 1, rejected
	r.seq = 0
	rc, res := r.send(t, intent.KindStatusRead, struct{}{}, nil) // seq 1 again
	if rc.Status != intent.StatusOK {
		t.Fatalf("seq 1 was consumed by a rejected intent: %s", res)
	}
}

func TestServiceActionRunsOpsAndValidatesInput(t *testing.T) {
	r := newRig(t, true)
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "beacon", Action: "restart"}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	rc, res = r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "beacon; rm -rf /", Action: "restart"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonInvalidPayload {
		t.Fatalf("hostile service name: code %s", got)
	}
}

func TestLogsReadClampsN(t *testing.T) {
	r := newRig(t, true)
	r.send(t, intent.KindLogsRead, intent.LogsReadPayload{N: 1_000_000}, nil)
	for _, c := range r.exec.cmds {
		if !containsAll(c, "journalctl", "-n 2000") {
			t.Fatalf("journalctl not clamped to 2000: %s", c)
		}
	}
}

func containsAll(s string, subs ...string) bool {
	for _, x := range subs {
		if !strings.Contains(s, x) {
			return false
		}
	}
	return true
}

// Review Focus 1, end to end: a broken replay record refuses everything.
func TestCorruptReplayStateRefusesEverything(t *testing.T) {
	r := newRig(t, true)
	if err := os.WriteFile(filepath.Join(r.dir, "replay.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.a = New(Config{Key: r.agentKey, Exec: r.exec, Now: func() time.Time { return r.now },
		PolicyPath: filepath.Join(r.dir, "policy.json"), ReplayPath: filepath.Join(r.dir, "replay.json"), NodePath: filepath.Join(r.dir, "node.json")})
	rc, res := r.send(t, intent.KindStatusRead, struct{}{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonReplayState {
		t.Fatalf("code %s, want replay_state", got)
	}
}
```

Add `"strings"` to the imports.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/agent/ -run 'Agent|Node|Rejection|Stranger|Replay|Service|Logs|Corrupt'`
Expected: FAIL — undefined `New`, `Config`.

- [ ] **Step 3: Add the two small helpers the agent needs**

`internal/monitor/monitor.go`:

```go
// PollOnce takes one snapshot without starting the polling loop. The agent
// answers status.read with it; head lag is computed by the controller, which
// holds the reference RPC.
func (m *Monitor) PollOnce(ctx context.Context) Snapshot { return m.poll(ctx) }
```

`internal/ops/ops.go`:

```go
// NodeUnits are the node's two systemd units, for callers that tail their logs.
func NodeUnits() []string { return []string{execUnitName, beaconUnitName} }
```

- [ ] **Step 4: Implement the agent**

```go
// internal/agent/agent.go

// Package agent is jumpgate's on-box half. It accepts only typed, signed
// intents, checks them against this box's own policy and replay record, runs
// the matching operation with a local executor, and signs what it answers.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

const (
	skew        = 60 * time.Second
	maxLifetime = 300 * time.Second
	maxBody     = 1 << 20
)

// Config wires an Agent. Paths are /etc/jumpgate/policy.json,
// /var/lib/jumpgate/replay.json and /etc/jumpgate/node.json in production.
type Config struct {
	Key        *signer.Key
	PolicyPath string
	ReplayPath string
	NodePath   string
	Exec       executor.Executor
	Now        func() time.Time
}

// Agent handles intents. It is safe for concurrent use.
type Agent struct {
	cfg       Config
	replay    *Replay
	replayErr error

	mu   sync.Mutex
	busy map[eip712.Address]bool
}

// New builds an agent. A broken replay record does not stop construction: the
// agent still answers, with a signed replay_state rejection, so an operator
// sees why rather than a dead socket.
func New(cfg Config) *Agent {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	a := &Agent{cfg: cfg, busy: map[eip712.Address]bool{}}
	a.replay, a.replayErr = OpenReplay(cfg.ReplayPath)
	return a
}

// Address is the agent's signing address.
func (a *Agent) Address() eip712.Address { return a.cfg.Key.Address() }

// Handle runs the checks in a fixed order: parse, agent, time, payload hash,
// signatures, policy, one-at-a-time per controller, replay admission (which
// persists), and only then the operation. Everything before admission is
// free to reject without consuming the controller's sequence.
func (a *Agent) Handle(ctx context.Context, env intent.Envelope) intent.ReceiptEnvelope {
	i, err := env.Intent.Parse()
	if err != nil {
		return a.answerReject(intent.Intent{}, [32]byte{}, reject(intent.ReasonInvalidPayload, err.Error()))
	}
	reqHash, err := i.Digest()
	if err != nil {
		return a.answerReject(i, [32]byte{}, reject(intent.ReasonInvalidPayload, err.Error()))
	}
	if i.Agent != a.Address() {
		return a.answerReject(i, reqHash, reject(intent.ReasonWrongAgent, "this intent is addressed to "+i.Agent.Hex()))
	}

	now := a.cfg.Now()
	issued, expiry := time.Unix(int64(i.IssuedAt), 0), time.Unix(int64(i.Expiry), 0)
	switch {
	case i.Expiry < i.IssuedAt || expiry.Sub(issued) > maxLifetime:
		return a.answerReject(i, reqHash, reject(intent.ReasonInvalidPayload, "intent lifetime must be at most 300s"))
	case now.Before(issued.Add(-skew)):
		r := reject(intent.ReasonClockSkew, fmt.Sprintf("issued at %s, agent time is %s", issued.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339)))
		return a.answerReject(i, reqHash, r)
	case now.After(expiry.Add(skew)):
		return a.answerReject(i, reqHash, reject(intent.ReasonExpired, "intent expired at "+expiry.UTC().Format(time.RFC3339)))
	}

	if intent.Hash(env.Payload) != i.PayloadHash {
		return a.answerReject(i, reqHash, reject(intent.ReasonInvalidPayload, "payload does not match its hash"))
	}

	var signers []eip712.Address
	for _, s := range env.Sigs {
		sig, err := signer.ParseSignature(s)
		if err != nil {
			return a.answerReject(i, reqHash, reject(intent.ReasonBadSignature, err.Error()))
		}
		who, err := signer.Recover(i.TypedData(), sig)
		if err != nil {
			return a.answerReject(i, reqHash, reject(intent.ReasonBadSignature, err.Error()))
		}
		signers = append(signers, who)
	}

	policy, err := LoadPolicy(a.cfg.PolicyPath)
	if err != nil {
		return a.answerReject(i, reqHash, reject(intent.ReasonUnauthorizedKind, err.Error()))
	}
	if err := policy.Authorise(i.Kind, i.Controller, signers); err != nil {
		return a.answerReject(i, reqHash, asReject(err))
	}

	if !a.claim(i.Controller) {
		return a.answerReject(i, reqHash, reject(intent.ReasonBusy, "another intent from this controller is running"))
	}
	defer a.release(i.Controller)

	if a.replayErr != nil {
		return a.answerReject(i, reqHash, reject(intent.ReasonReplayState, a.replayErr.Error()))
	}
	if err := a.replay.Admit(i.Controller, i.Seq, i.Nonce, i.Expiry, now); err != nil {
		return a.answerReject(i, reqHash, asReject(err))
	}

	result, status, rej := a.dispatch(ctx, i, env.Payload, policy)
	if rej != nil {
		return a.answerReject(i, reqHash, rej)
	}
	return a.answer(i, reqHash, status, result)
}

func (a *Agent) claim(c eip712.Address) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy[c] {
		return false
	}
	a.busy[c] = true
	return true
}

func (a *Agent) release(c eip712.Address) {
	a.mu.Lock()
	delete(a.busy, c)
	a.mu.Unlock()
}

func asReject(err error) *Reject {
	if r, ok := err.(*Reject); ok {
		return r
	}
	return reject(intent.ReasonValidation, err.Error())
}

func (a *Agent) answerReject(i intent.Intent, reqHash [32]byte, r *Reject) intent.ReceiptEnvelope {
	body, _ := json.Marshal(intent.Rejection{Code: r.Code, Message: r.Message, LastSeq: r.LastSeq, AgentTime: a.cfg.Now().Unix()})
	return a.answer(i, reqHash, intent.StatusRejected, body)
}

// answer signs a receipt over the exact result bytes. Rejections are signed
// too, so a controller can prove why it was refused.
func (a *Agent) answer(i intent.Intent, reqHash [32]byte, status uint8, result []byte) intent.ReceiptEnvelope {
	rc := intent.Receipt{Agent: a.Address(), RequestHash: reqHash, Seq: i.Seq, Status: status, ResultHash: intent.Hash(result)}
	sig, _ := a.cfg.Key.SignTypedData(context.Background(), rc.TypedData())
	return intent.ReceiptEnvelope{Receipt: rc.JSON(), Result: result, Sig: sig.Hex()}
}
```

```go
// internal/agent/dispatch.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/logwatch"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// DiskResult answers disk.read.
type DiskResult struct {
	Usage     ops.DU `json:"usage"`
	FreeBytes uint64 `json:"freeBytes"`
}

// FirewallReadPayload carries the controller's trusted overlay ranges, which
// live in the controller's config, not on the box.
type FirewallReadPayload struct {
	OverlayCIDRs []string `json:"overlayCidrs"`
}

// PlanVersions lists the plan versions this agent can run. Sub-project 1 runs
// no plans; sub-project 2 adds the first.
var PlanVersions = []string{}

func (a *Agent) loadNode() (catalog.WireConfig, *Reject) {
	var w catalog.WireConfig
	b, err := os.ReadFile(a.cfg.NodePath)
	if errors.Is(err, os.ErrNotExist) {
		return w, reject(intent.ReasonNotSetUp, "this box has no node set up yet")
	}
	if err != nil {
		return w, reject(intent.ReasonValidation, err.Error())
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return w, reject(intent.ReasonValidation, "node.json does not parse: "+err.Error())
	}
	if err := catalog.ValidateDataDir(w.DataDir); err != nil {
		return w, reject(intent.ReasonValidation, err.Error())
	}
	return w, nil
}

func decode[T any](payload []byte) (T, *Reject) {
	var v T
	if err := json.Unmarshal(payload, &v); err != nil {
		return v, reject(intent.ReasonInvalidPayload, err.Error())
	}
	return v, nil
}

// dispatch runs one admitted intent. An operation error is a signed failure
// receipt (StatusFailed), distinct from a rejection.
func (a *Agent) dispatch(ctx context.Context, i intent.Intent, payload []byte, p Policy) ([]byte, uint8, *Reject) {
	if i.Kind == intent.KindAgentInfo {
		_, notSetUp := a.loadNode()
		return mustJSON(intent.AgentInfo{
			Version: buildinfo.Version(), Address: a.Address().Hex(), LastSeq: a.replay.LastSeq(i.Controller),
			PlanVersions: PlanVersions, Signers: len(p.Signers), SetUp: notSetUp == nil,
		}), intent.StatusOK, nil
	}

	w, rej := a.loadNode()
	if rej != nil {
		return nil, 0, rej
	}
	ex := a.cfg.Exec

	switch i.Kind {
	case intent.KindStatusRead:
		snap := monitor.New(monitor.Config{Exec: ex, Wire: w}).PollOnce(ctx)
		return mustJSON(snap), intent.StatusOK, nil

	case intent.KindDiskRead:
		du, err := ops.DiskUsage(ctx, ex, w)
		if err != nil {
			return failed(err)
		}
		free, err := ops.FreeBytesAt(ctx, ex, w.DataDir)
		if err != nil {
			return failed(err)
		}
		return mustJSON(DiskResult{Usage: du, FreeBytes: free}), intent.StatusOK, nil

	case intent.KindEndpointsRead:
		pl, rej := decode[intent.EndpointsReadPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		info, err := ops.Endpoints(ctx, ex, w, pl.SSHLogin != "", pl.SSHLogin)
		if err != nil {
			return failed(err)
		}
		return mustJSON(info), intent.StatusOK, nil

	case intent.KindFirewallRead:
		pl, rej := decode[FirewallReadPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		items, err := ops.FirewallChecklist(ctx, ex, w, ops.ParseOverlayCIDRs(pl.OverlayCIDRs)...)
		if err != nil {
			return failed(err)
		}
		return mustJSON(items), intent.StatusOK, nil

	case intent.KindLogsRead:
		pl, rej := decode[intent.LogsReadPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		n := pl.N
		if n <= 0 {
			n = intent.LogsDefaultN
		}
		if n > intent.LogsMaxN {
			n = intent.LogsMaxN
		}
		var hits []logwatch.Hit
		for _, unit := range ops.NodeUnits() {
			res, err := ex.Run(ctx, "journalctl -u "+unit+" -n "+strconv.Itoa(n)+" --no-pager -o cat", nil)
			if err != nil {
				return failed(err)
			}
			now := a.cfg.Now()
			for _, line := range strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n") {
				if line == "" {
					continue
				}
				if h, ok := logwatch.Classify(unit, line, now); ok {
					hits = append(hits, h)
				} else {
					hits = append(hits, logwatch.Hit{Unit: unit, Line: line, At: now, Severity: "info"})
				}
			}
		}
		return mustJSON(hits), intent.StatusOK, nil

	case intent.KindServiceAction:
		pl, rej := decode[intent.ServiceActionPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		if (pl.Service != "exec" && pl.Service != "beacon") || (pl.Action != "start" && pl.Action != "stop" && pl.Action != "restart") {
			return nil, 0, reject(intent.ReasonInvalidPayload, fmt.Sprintf("service %q / action %q", pl.Service, pl.Action))
		}
		active, err := ops.ServiceAction(ctx, ex, pl.Service, pl.Action)
		if err != nil {
			return failed(err)
		}
		return mustJSON(struct {
			Active bool `json:"active"`
		}{active}), intent.StatusOK, nil
	}
	return nil, 0, reject(intent.ReasonUnknownKind, i.Kind)
}

func failed(err error) ([]byte, uint8, *Reject) {
	return mustJSON(intent.Failure{Message: err.Error()}), intent.StatusFailed, nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(intent.Failure{Message: "encode result: " + err.Error()})
	}
	return b
}
```

`ops.NodeUnits()` returns fixed unit names (no user input), so building the journalctl line by concatenation is safe; keep it that way.

- [ ] **Step 5: Run the agent tests**

Run: `go test -race ./internal/agent/`
Expected: PASS. If `TestLogsReadClampsN` fails because `fakeExec.res.Stdout` produces no hits, that is fine; the test only inspects the commands.

- [ ] **Step 6: Write the failing listener tests**

```go
// internal/agent/listen_test.go
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/intent"
)

func shortDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "jga")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

func unixClient(sock string) *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
}

func TestServeAnswersOverTheSocket(t *testing.T) {
	r := newRig(t, true)
	sock := filepath.Join(shortDir(t), "agent.sock")
	ln, err := Listen(sock, -1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Serve(ctx, r.a, ln)

	fi, _ := os.Stat(sock)
	if fi.Mode().Perm() != 0o660 {
		t.Errorf("socket mode %o, want 660", fi.Mode().Perm())
	}

	body, _ := json.Marshal(intent.Envelope{})
	res, err := unixClient(sock).Post("http://agent/v1/intent", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out intent.ReceiptEnvelope
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil || out.Receipt.Status != intent.StatusRejected {
		t.Fatalf("status %d, %v", out.Receipt.Status, err)
	}

	big := bytes.Repeat([]byte("a"), maxBody+1)
	res2, err := unixClient(sock).Post("http://agent/v1/intent", "application/json", bytes.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	if res2.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body: %d, want 413", res2.StatusCode)
	}
}

// Listen replaces a stale socket but never deletes anything else.
func TestListenRefusesToDeleteANonSocket(t *testing.T) {
	path := filepath.Join(shortDir(t), "agent.sock")
	if err := os.WriteFile(path, []byte("precious"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(path, -1); err == nil {
		t.Fatal("Listen removed a regular file")
	}
}

func TestPeerGateAllowsOwnUIDAndEnrolledUIDsOnly(t *testing.T) {
	p := Policy{LocalUIDs: []int{4242}}
	if !peerAllowed(p, os.Getuid(), nil, -1) {
		t.Error("the agent's own uid was refused")
	}
	if !peerAllowed(p, 4242, nil, -1) {
		t.Error("an enrolled local uid was refused")
	}
	if peerAllowed(p, 777, nil, -1) {
		t.Error("an unknown uid was allowed")
	}
	if !peerAllowed(p, 777, []int{55}, 55) {
		t.Error("a member of the jumpgate group was refused")
	}
}
```

- [ ] **Step 7: Run them to verify they fail**

Run: `go test ./internal/agent/ -run 'Serve|Listen|PeerGate'`
Expected: FAIL — undefined `Listen`, `Serve`, `peerAllowed`.

- [ ] **Step 8: Implement the listener**

```go
// internal/agent/listen.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// Listen opens the agent socket. A stale socket from a previous run is
// removed; any other file at the path is an error, never deleted. The socket
// is 0660 and, when gid >= 0, owned root:gid, so only root and the jumpgate
// group (the SSH tunnel user) can connect.
func Listen(path string, gid int) (net.Listener, error) {
	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("agent: %s exists and is not a socket; refusing to remove it", path)
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o660); err != nil {
		ln.Close()
		return nil, err
	}
	if gid >= 0 {
		if err := os.Chown(path, 0, gid); err != nil {
			ln.Close()
			return nil, err
		}
	}
	return ln, nil
}

// JumpgateGID is the jumpgate group's id, or -1 when the group is absent.
func JumpgateGID() int {
	g, err := user.LookupGroup("jumpgate")
	if err != nil {
		return -1
	}
	n, err := strconv.Atoi(g.Gid)
	if err != nil {
		return -1
	}
	return n
}

// peerAllowed: the agent's own uid (root in production), a uid enrolled for
// local use, or any member of the jumpgate group. Signatures are still
// required for every intent; this gate only keeps strangers off the socket.
func peerAllowed(p Policy, uid int, gids []int, jumpgateGID int) bool {
	if uid == 0 || uid == os.Getuid() {
		return true
	}
	for _, u := range p.LocalUIDs {
		if u == uid {
			return true
		}
	}
	if jumpgateGID >= 0 {
		for _, g := range gids {
			if g == jumpgateGID {
				return true
			}
		}
	}
	return false
}

type peerKey struct{}

// Serve answers POST /v1/intent until ctx is done.
func Serve(ctx context.Context, a *Agent, ln net.Listener) error {
	gid := JumpgateGID()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/intent", func(w http.ResponseWriter, r *http.Request) {
		if allowed, _ := r.Context().Value(peerKey{}).(bool); !allowed {
			http.Error(w, "not allowed on this socket", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		var env intent.Envelope
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
				return
			}
			// An undecodable envelope still gets a signed rejection.
			env = intent.Envelope{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(a.Handle(r.Context(), env))
	})
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			uid, gids, err := peerCred(c)
			if err != nil {
				return context.WithValue(ctx, peerKey{}, false)
			}
			p, _ := LoadPolicy(a.cfg.PolicyPath)
			return context.WithValue(ctx, peerKey{}, peerAllowed(p, uid, gids, gid))
		},
	}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
```

`TestServeAnswersOverTheSocket` connects as the test's own uid, which `peerAllowed` admits. Note `Handle` on an empty envelope returns a rejection whose `Intent` is zero — that is expected.

```go
// internal/agent/peercred_linux.go
//go:build linux

package agent

import (
	"errors"
	"net"
	"os/user"
	"strconv"

	"golang.org/x/sys/unix"
)

func peerCred(c net.Conn) (int, []int, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return 0, nil, errors.New("not a unix socket")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, nil, err
	}
	var cred *unix.Ucred
	var serr error
	if err := raw.Control(func(fd uintptr) {
		cred, serr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return 0, nil, err
	}
	if serr != nil {
		return 0, nil, serr
	}
	return int(cred.Uid), groupsOf(int(cred.Uid), int(cred.Gid)), nil
}

// groupsOf lists a uid's primary and supplementary groups from /etc/group
// (pure Go: the agent is built without cgo).
func groupsOf(uid, primary int) []int {
	gids := []int{primary}
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return gids
	}
	ids, err := u.GroupIds()
	if err != nil {
		return gids
	}
	for _, s := range ids {
		if n, err := strconv.Atoi(s); err == nil {
			gids = append(gids, n)
		}
	}
	return gids
}
```

```go
// internal/agent/peercred_darwin.go
//go:build darwin

package agent

import (
	"errors"
	"net"

	"golang.org/x/sys/unix"
)

// macOS is a development platform for the agent; production is Linux.
func peerCred(c net.Conn) (int, []int, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return 0, nil, errors.New("not a unix socket")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, nil, err
	}
	var x *unix.Xucred
	var serr error
	if err := raw.Control(func(fd uintptr) {
		x, serr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil {
		return 0, nil, err
	}
	if serr != nil {
		return 0, nil, serr
	}
	gids := make([]int, 0, x.Ngroups)
	for _, g := range x.Groups[:x.Ngroups] {
		gids = append(gids, int(g))
	}
	return int(x.Uid), gids, nil
}
```

```go
// internal/agent/peercred_other.go
//go:build !linux && !darwin

package agent

import (
	"errors"
	"net"
)

// The agent runs on Linux. Elsewhere every peer is refused.
func peerCred(net.Conn) (int, []int, error) { return 0, nil, errors.New("peer credentials unsupported here") }
```

- [ ] **Step 9: Run all agent tests and cross-builds**

Run: `go test -race ./internal/agent/ ./internal/monitor/ ./internal/ops/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./internal/agent/ && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./internal/agent/`
Expected: PASS; both Linux cross-builds succeed without cgo.

- [ ] **Step 10: Commit**

```bash
git add internal/agent internal/monitor/monitor.go internal/ops/ops.go go.mod go.sum
git commit -m "feat(agent): verify, admit and answer signed intents on a unix socket

The agent checks addressee, lifetime, clock skew, payload hash, signatures,
policy and replay in a fixed order, persists admission before running
anything, dispatches the sub-project 1 kinds to the existing ops and monitor
code with a local executor, and signs every answer, rejections included. Kinds
that need a node are refused with not_set_up before setup has run. The socket
is 0660 and admits root, enrolled local uids and the jumpgate group.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 13: `internal/agentclient` — the controller's side of the wire

**Files:**
- Create: `internal/agentclient/client.go`, `internal/agentclient/dial.go`, `internal/agentclient/client_test.go`

**Interfaces:**
- Consumes: `agent.New`, `agent.Listen`, `agent.Serve` (tests), `executor.DialSSH`, `executor.SSHConfig` (Task 5), `intent.*`, `signer.*`.
- Produces:
  - `const DefaultSocket = "/run/jumpgate/agent.sock"`
  - `type Target struct{ Local bool; Socket string; SSH executor.SSHConfig; Agent eip712.Address }`
  - `type SeqStore interface{ Next(agent eip712.Address) (uint64, error); Set(agent eip712.Address, next uint64) error }`; `NewMemorySeqStore() SeqStore`
  - `Dial(ctx context.Context, t Target, s signer.Signer, seqs SeqStore) (*Client, error)`; `(*Client) Do(ctx context.Context, kind string, payload any) (Response, error)`; `(*Client) Close() error`
  - `type Response struct{ Status uint8; Result json.RawMessage; Rejection *intent.Rejection; Failure *intent.Failure }`
  - `var ErrBadReceipt`, `var ErrUnreachable`

- [ ] **Step 1: Write the failing tests**

```go
// internal/agentclient/client_test.go
package agentclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

type stubExec struct{}

func (stubExec) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) { return executor.Result{}, nil }
func (stubExec) WriteFile(context.Context, string, []byte, os.FileMode) error             { return nil }
func (stubExec) ReadFile(context.Context, string) ([]byte, error)                         { return nil, nil }
func (stubExec) Close() error                                                             { return nil }

// startAgent runs a real agent on a temp socket and returns its socket, the
// agent's address and the enrolled controller key.
func startAgent(t *testing.T) (sock string, agentAddr eip712.Address, controller *signer.Key, a *agent.Agent) {
	t.Helper()
	dir, _ := os.MkdirTemp("/tmp", "jgc")
	t.Cleanup(func() { os.RemoveAll(dir) })
	agentKey, _ := signer.GenerateKey()
	controller, _ = signer.GenerateKey()
	p := agent.Policy{Signers: []agent.SignerEntry{{Address: controller.Address().Hex(), Tier: agent.TierRoutine}}}
	_ = p.Save(filepath.Join(dir, "policy.json"))
	_ = agent.InitReplay(filepath.Join(dir, "replay.json"))
	a = agent.New(agent.Config{Key: agentKey, Exec: stubExec{}, PolicyPath: filepath.Join(dir, "policy.json"),
		ReplayPath: filepath.Join(dir, "replay.json"), NodePath: filepath.Join(dir, "node.json")})
	sock = filepath.Join(dir, "a.sock")
	ln, err := agent.Listen(sock, -1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go agent.Serve(ctx, a, ln)
	return sock, agentKey.Address(), controller, a
}

func TestDoLocalAgentInfo(t *testing.T) {
	sock, agentAddr, controller, _ := startAgent(t)
	c, err := Dial(context.Background(), Target{Local: true, Socket: sock, Agent: agentAddr}, controller, NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("Do = %+v, %v", res, err)
	}
	var info intent.AgentInfo
	_ = json.Unmarshal(res.Result, &info)
	if info.Address != agentAddr.Hex() {
		t.Fatalf("info = %+v", info)
	}
}

// A receipt signed by anyone other than the paired agent is a security error,
// never a result.
func TestDoRejectsAReceiptFromTheWrongAgent(t *testing.T) {
	sock, _, controller, _ := startAgent(t)
	other, _ := signer.GenerateKey()
	c, _ := Dial(context.Background(), Target{Local: true, Socket: sock, Agent: other.Address()}, controller, NewMemorySeqStore())
	defer c.Close()
	_, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if !errors.Is(err, ErrBadReceipt) {
		t.Fatalf("err = %v, want ErrBadReceipt", err)
	}
}

// A controller whose counter fell behind (restored config, second machine)
// resynchronises from the signed stale_seq rejection and retries once.
func TestDoResyncsAStaleSequence(t *testing.T) {
	sock, agentAddr, controller, _ := startAgent(t)
	seqs := NewMemorySeqStore()
	c, _ := Dial(context.Background(), Target{Local: true, Socket: sock, Agent: agentAddr}, controller, seqs)
	defer c.Close()
	for i := 0; i < 3; i++ {
		if _, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{}); err != nil {
			t.Fatal(err)
		}
	}
	_ = seqs.Set(agentAddr, 1) // forget
	res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("after resync: %+v, %v", res, err)
	}
}

func TestDoSeparatesUnreachableFromRefused(t *testing.T) {
	k, _ := signer.GenerateKey()
	c, err := Dial(context.Background(), Target{Local: true, Socket: "/tmp/does-not-exist.sock", Agent: k.Address()}, k, NewMemorySeqStore())
	if err == nil {
		_, err = c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	}
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
}

// The remote path: SSH as the tunnel user, then a direct-streamlocal channel
// to the agent's socket, exactly as OpenSSH provides it.
func TestDoOverSSHStreamlocal(t *testing.T) {
	sock, agentAddr, controller, _ := startAgent(t)
	host, port, hostKey, keyPath := startStreamlocalSSHD(t)
	target := Target{Socket: sock, Agent: agentAddr, SSH: executor.SSHConfig{
		Host: host, Port: port, User: "jumpgate", KeyPath: keyPath, HostKey: ssh.FixedHostKey(hostKey),
	}}
	c, err := Dial(context.Background(), target, controller, NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("Do over SSH = %+v, %v", res, err)
	}
}

// startStreamlocalSSHD is a gliderlabs server whose only capability is the
// direct-streamlocal@openssh.com channel, registered by hand.
func startStreamlocalSSHD(t *testing.T) (string, int, ssh.PublicKey, string) {
	t.Helper()
	// Generate host and client keys exactly as internal/executor/ssh_test.go's
	// startTestSSHD does (copy writePrivateKey from there into this file).
	hostSigner, clientKeyPath := testKeys(t)
	srv := &gliderssh.Server{
		PublicKeyHandler: func(gliderssh.Context, gliderssh.PublicKey) bool { return true },
		ChannelHandlers: map[string]gliderssh.ChannelHandler{
			"direct-streamlocal@openssh.com": func(_ *gliderssh.Server, _ *ssh.ServerConn, newChan ssh.NewChannel, _ gliderssh.Context) {
				var req struct {
					SocketPath string
					Reserved0  string
					Reserved1  uint32
				}
				if err := ssh.Unmarshal(newChan.ExtraData(), &req); err != nil {
					newChan.Reject(ssh.ConnectionFailed, "bad request")
					return
				}
				conn, err := net.Dial("unix", req.SocketPath)
				if err != nil {
					newChan.Reject(ssh.ConnectionFailed, err.Error())
					return
				}
				ch, reqs, err := newChan.Accept()
				if err != nil {
					conn.Close()
					return
				}
				go ssh.DiscardRequests(reqs)
				go func() { io.Copy(ch, conn); ch.CloseWrite() }()
				go func() { io.Copy(conn, ch); conn.Close() }()
			},
		},
	}
	srv.AddHostKey(hostSigner)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	a := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", a.Port, hostSigner.PublicKey(), clientKeyPath
}
```

Write `testKeys(t) (ssh.Signer, string)` in this test file: an ed25519 host signer plus a client private key written to a temp file, copying `writePrivateKey` from `internal/executor/ssh_test.go`. Drop `"time"` from the imports if nothing else uses it.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/agentclient/`
Expected: FAIL — package has no non-test files.

- [ ] **Step 3: Implement**

```go
// internal/agentclient/dial.go
package agentclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// DefaultSocket is where the agent listens.
const DefaultSocket = "/run/jumpgate/agent.sock"

// transport returns an HTTP client whose every connection is to the agent
// socket — local, or through the SSH connection as a direct-streamlocal
// channel — and a closer for the SSH connection.
func transport(ctx context.Context, t Target) (*http.Client, func() error, error) {
	sock := t.Socket
	if sock == "" {
		sock = DefaultSocket
	}
	var dial func(ctx context.Context) (net.Conn, error)
	closer := func() error { return nil }

	if t.Local {
		dial = func(ctx context.Context) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", sock)
		}
	} else {
		client, err := executor.DialSSH(ctx, t.SSH)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: ssh %s: %v", ErrUnreachable, t.SSH.Host, err)
		}
		closer = client.Close
		dial = func(context.Context) (net.Conn, error) { return dialUnix(client, sock) }
	}
	hc := &http.Client{
		Timeout: 2 * time.Minute,
		Transport: &http.Transport{
			DialContext:     func(ctx context.Context, _, _ string) (net.Conn, error) { return dial(ctx) },
			MaxIdleConns:    2,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return hc, closer, nil
}

// dialUnix opens a direct-streamlocal channel. x/crypto's Client.Dial
// supports "unix" for exactly this.
func dialUnix(c *ssh.Client, path string) (net.Conn, error) { return c.Dial("unix", path) }
```

```go
// internal/agentclient/client.go

// Package agentclient is the controller's side of the agent protocol: it
// builds and signs intents, carries them to the agent's socket, and accepts an
// answer only if the paired agent signed it and it answers this exact request.
package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

var (
	// ErrBadReceipt is a security error: the answer was not signed by the
	// paired agent, or does not answer this request. Its result is discarded.
	ErrBadReceipt = errors.New("agentclient: receipt failed verification")
	// ErrUnreachable is a transport failure, never a refusal.
	ErrUnreachable = errors.New("agentclient: could not reach the agent")
)

// Target says how to reach one agent and whom to expect there.
type Target struct {
	Local  bool
	Socket string
	SSH    executor.SSHConfig
	Agent  eip712.Address
}

// SeqStore keeps the next sequence number per agent.
type SeqStore interface {
	Next(agent eip712.Address) (uint64, error)
	Set(agent eip712.Address, next uint64) error
}

type memSeqs struct {
	mu sync.Mutex
	m  map[eip712.Address]uint64
}

// NewMemorySeqStore is for tests and one-shot tools.
func NewMemorySeqStore() SeqStore { return &memSeqs{m: map[eip712.Address]uint64{}} }

func (s *memSeqs) Next(a eip712.Address) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m[a] == 0 {
		return 1, nil
	}
	return s.m[a], nil
}

func (s *memSeqs) Set(a eip712.Address, next uint64) error {
	s.mu.Lock()
	s.m[a] = next
	s.mu.Unlock()
	return nil
}

// Response is a verified answer.
type Response struct {
	Status    uint8
	Result    json.RawMessage
	Rejection *intent.Rejection
	Failure   *intent.Failure
}

// Client talks to one agent.
type Client struct {
	t      Target
	signer signer.Signer
	seqs   SeqStore
	hc     *http.Client
	closer func() error
	now    func() time.Time
}

// Dial connects (for SSH, the connection is opened now and reused).
func Dial(ctx context.Context, t Target, s signer.Signer, seqs SeqStore) (*Client, error) {
	hc, closer, err := transport(ctx, t)
	if err != nil {
		return nil, err
	}
	return &Client{t: t, signer: s, seqs: seqs, hc: hc, closer: closer, now: time.Now}, nil
}

// Close releases the SSH connection.
func (c *Client) Close() error { return c.closer() }

// Do sends one intent and returns the verified answer. A stale_seq rejection
// resynchronises the counter from the agent's signed LastSeq and retries once.
func (c *Client) Do(ctx context.Context, kind string, payload any) (Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}
	for attempt := 0; ; attempt++ {
		res, err := c.once(ctx, kind, body)
		if err != nil {
			return res, err
		}
		if res.Rejection != nil && res.Rejection.Code == intent.ReasonStaleSeq && attempt == 0 {
			if err := c.seqs.Set(c.t.Agent, res.Rejection.LastSeq+1); err != nil {
				return res, err
			}
			continue
		}
		return res, nil
	}
}

func (c *Client) once(ctx context.Context, kind string, body []byte) (Response, error) {
	seq, err := c.seqs.Next(c.t.Agent)
	if err != nil {
		return Response{}, err
	}
	nonce, err := intent.NewNonce()
	if err != nil {
		return Response{}, err
	}
	now := uint64(c.now().Unix())
	in := intent.Intent{Agent: c.t.Agent, Controller: c.signer.Address(), Seq: seq, Nonce: nonce,
		IssuedAt: now, Expiry: now + 120, Kind: kind, PayloadHash: intent.Hash(body)}
	sig, err := c.signer.SignTypedData(ctx, in.TypedData())
	if err != nil {
		return Response{}, fmt.Errorf("agentclient: sign: %w", err)
	}
	reqHash, err := in.Digest()
	if err != nil {
		return Response{}, err
	}

	envBytes, _ := json.Marshal(intent.Envelope{Intent: in.JSON(), Payload: body, Sigs: []string{sig.Hex()}})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://agent/v1/intent", bytes.NewReader(envBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("%w: agent socket answered HTTP %d", ErrUnreachable, resp.StatusCode)
	}
	var out intent.ReceiptEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Response{}, fmt.Errorf("%w: undecodable answer: %v", ErrBadReceipt, err)
	}

	rc, err := out.Receipt.Parse()
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrBadReceipt, err)
	}
	rsig, err := signer.ParseSignature(out.Sig)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrBadReceipt, err)
	}
	who, err := signer.Recover(rc.TypedData(), rsig)
	switch {
	case err != nil:
		return Response{}, fmt.Errorf("%w: %v", ErrBadReceipt, err)
	case who != c.t.Agent:
		return Response{}, fmt.Errorf("%w: signed by %s, expected agent %s", ErrBadReceipt, who.Hex(), c.t.Agent.Hex())
	case rc.Agent != c.t.Agent || rc.RequestHash != reqHash || rc.Seq != seq || rc.ResultHash != intent.Hash(out.Result):
		return Response{}, fmt.Errorf("%w: receipt does not answer this request", ErrBadReceipt)
	}

	// The agent saw this sequence (admitted or not); never reuse it.
	if err := c.seqs.Set(c.t.Agent, seq+1); err != nil {
		return Response{}, err
	}
	res := Response{Status: rc.Status, Result: out.Result}
	switch rc.Status {
	case intent.StatusRejected:
		res.Rejection = &intent.Rejection{}
		_ = json.Unmarshal(out.Result, res.Rejection)
	case intent.StatusFailed:
		res.Failure = &intent.Failure{}
		_ = json.Unmarshal(out.Result, res.Failure)
	}
	return res, nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test -race -v ./internal/agentclient/`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/agentclient
git commit -m "feat(agentclient): sign intents and accept only verified receipts

Reaches an agent over its local socket or through an SSH direct-streamlocal
channel, optionally via a jump host. A receipt counts only if the paired agent
signed it and it names this request's hash and sequence; anything else is
ErrBadReceipt and its result is discarded. Transport failures are a separate
ErrUnreachable, and a stale sequence resynchronises once from the agent's
signed answer.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 14: Bootstrap — install and pair an agent

**Files:**
- Create: `internal/executor/sudo.go`, `internal/executor/sudo_test.go`, `internal/bootstrap/bootstrap.go`, `internal/bootstrap/unit.go`, `internal/bootstrap/bootstrap_test.go`
- Create: `cmd/jumpgate/cli_agent.go` is Task 17; this task only shells out to `jumpgate agent init` / `agent enroll`, whose behaviour Task 17 implements and tests.

**Interfaces:**
- Consumes: `executor.Executor`, `RunOpts.Stdin` (Task 3); `catalog.WireConfig`; `eip712.Address`.
- Produces:
  - `executor.Sudo(e Executor) Executor` — every command runs as `sudo -n sh -c '<cmd>'`; `WriteFile`/`ReadFile` go through the same wrapper with stdin passthrough
  - `bootstrap.Options{ Exec executor.Executor; Local bool; LocalUID int; AgentBinary func(arch string) (string, error); Controller eip712.Address; ControllerLabel string; TransportKey string; Wire *catalog.WireConfig; Event func(step, line string) }`
  - `bootstrap.Run(ctx context.Context, o Options) (agent eip712.Address, err error)`
  - `bootstrap.StepError{ Step string; Err error }`
  - Constants: `BinaryPath = "/usr/local/lib/jumpgate/jumpgate"`, `UnitPath = "/etc/systemd/system/jumpgate-agent.service"`, `DropInPath = "/etc/ssh/sshd_config.d/50-jumpgate.conf"`, `NodePath = "/etc/jumpgate/node.json"`, `AuthorizedKeys = "/var/lib/jumpgate/home/.ssh/authorized_keys"`

- [ ] **Step 1: Write the failing sudo test**

```go
// internal/executor/sudo_test.go
package executor

import (
	"context"
	"io"
	"strings"
	"testing"
)

type recordExec struct {
	cmds   []string
	stdins []string
}

func (r *recordExec) Run(_ context.Context, cmd string, o *RunOpts) (Result, error) {
	r.cmds = append(r.cmds, cmd)
	if o != nil && o.Stdin != nil {
		b, _ := io.ReadAll(o.Stdin)
		r.stdins = append(r.stdins, string(b))
	}
	return Result{}, nil
}
func (r *recordExec) WriteFile(context.Context, string, []byte, fsFileMode) error { return nil }
func (r *recordExec) ReadFile(context.Context, string) ([]byte, error)          { return nil, nil }
func (r *recordExec) Close() error                                              { return nil }

func TestSudoWrapsCommandsAndKeepsStdin(t *testing.T) {
	inner := &recordExec{}
	s := Sudo(inner)
	_, _ = s.Run(context.Background(), "echo 'hi'", nil)
	if inner.cmds[0] != `sudo -n sh -c 'echo '"'"'hi'"'"''` {
		t.Fatalf("cmd = %s", inner.cmds[0])
	}
	_ = s.WriteFile(context.Background(), "/etc/x", []byte("secret"), 0o600)
	if !strings.HasPrefix(inner.cmds[1], "sudo -n sh -c ") || strings.Contains(inner.cmds[1], "secret") || inner.stdins[0] != "secret" {
		t.Fatalf("WriteFile via sudo: cmd %s stdin %q", inner.cmds[1], inner.stdins)
	}
}
```

(`fsFileMode` stands for `fs.FileMode`; import `io/fs` and write `fs.FileMode`.)

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/executor/ -run Sudo`
Expected: FAIL — undefined `Sudo`.

- [ ] **Step 3: Implement `Sudo`**

```go
// internal/executor/sudo.go
package executor

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"strings"
)

type sudoExec struct{ inner Executor }

// Sudo runs every command as root through `sudo -n`, for pairing a box whose
// login user is not root. -n makes a password prompt an immediate failure
// instead of a hang: bootstrap needs passwordless sudo, and says so.
func Sudo(e Executor) Executor { return sudoExec{inner: e} }

func (s sudoExec) Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error) {
	return s.inner.Run(ctx, "sudo -n sh -c "+shQuote(cmd), opts)
}

func (s sudoExec) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	res, err := s.Run(ctx, writeFileCmd(path, mode), &RunOpts{Stdin: bytes.NewReader(content)})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("write %s as root: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	return nil
}

func (s sudoExec) ReadFile(ctx context.Context, path string) ([]byte, error) {
	res, err := s.Run(ctx, readFileCmd(path), nil)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("read %s as root: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	return base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(res.Stdout), "\n", ""))
}

func (s sudoExec) Close() error { return s.inner.Close() }
```

Run: `go test ./internal/executor/ -run Sudo` → PASS.

- [ ] **Step 4: Write the failing bootstrap tests**

```go
// internal/bootstrap/bootstrap_test.go
package bootstrap

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// fakeBox is a scripted target: each rule maps a command substring to a
// result, and written files are kept in memory.
type fakeBox struct {
	rules  []rule
	cmds   []string
	files  map[string]string
	stdins []string
}

type rule struct {
	match string
	res   executor.Result
}

func (b *fakeBox) Run(_ context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	b.cmds = append(b.cmds, cmd)
	if o != nil && o.Stdin != nil {
		x, _ := io.ReadAll(o.Stdin)
		b.stdins = append(b.stdins, string(x))
	}
	for _, r := range b.rules {
		if strings.Contains(cmd, r.match) {
			return r.res, nil
		}
	}
	return executor.Result{}, nil
}

func (b *fakeBox) WriteFile(_ context.Context, path string, content []byte, _ fs.FileMode) error {
	if b.files == nil {
		b.files = map[string]string{}
	}
	b.files[path] = string(content)
	return nil
}

func (b *fakeBox) ReadFile(_ context.Context, path string) ([]byte, error) {
	if c, ok := b.files[path]; ok {
		return []byte(c), nil
	}
	return nil, os.ErrNotExist
}

func (b *fakeBox) Close() error { return nil }

const agentAddrHex = "0x00000000000000000000000000000000000000aa"

func freshBox() *fakeBox {
	return &fakeBox{rules: []rule{
		{"uname -s", executor.Result{Stdout: "Linux\n"}},
		{"uname -m", executor.Result{Stdout: "x86_64\n"}},
		{"Include", executor.Result{}},                                // drop-in include present
		{"sha256sum " + BinaryPath + " ", executor.Result{ExitCode: 1}}, // not installed yet
		{"agent init", executor.Result{Stdout: agentAddrHex + "\n"}},
		{"is-active", executor.Result{Stdout: "active\n"}},
	}}
}

func opts(t *testing.T, box *fakeBox) Options {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "jumpgate-linux-amd64")
	_ = os.WriteFile(bin, []byte("binary"), 0o755)
	var ctrl eip712.Address
	ctrl[19] = 0xC
	return Options{
		Exec: box, AgentBinary: func(arch string) (string, error) {
			if arch != "amd64" {
				return "", errors.New("wrong arch " + arch)
			}
			return bin, nil
		},
		Controller: ctrl, ControllerLabel: "laptop", TransportKey: "ssh-ed25519 AAAATEST jumpgate-controller",
	}
}

func TestRunPairsAFreshBox(t *testing.T) {
	box := freshBox()
	got, err := Run(context.Background(), opts(t, box))
	if err != nil {
		t.Fatal(err)
	}
	if got.Hex() != eip712.Address{0xaa}.Hex() && !strings.EqualFold(got.Hex(), agentAddrHex) {
		t.Fatalf("agent address = %s", got.Hex())
	}
	all := strings.Join(box.cmds, "\n")
	for _, want := range []string{"groupadd --system jumpgate", "useradd --system", "sshd -t", "agent enroll --address 0x", "systemctl enable --now jumpgate-agent.service"} {
		if !strings.Contains(all, want) {
			t.Errorf("never ran %q", want)
		}
	}
	if !strings.Contains(box.files[AuthorizedKeys], `restrict,port-forwarding,command="/bin/false" ssh-ed25519 AAAATEST`) {
		t.Errorf("authorized_keys = %q", box.files[AuthorizedKeys])
	}
	if !strings.Contains(box.files[DropInPath], "AllowStreamLocalForwarding local") {
		t.Errorf("drop-in = %q", box.files[DropInPath])
	}
}

// Review Focus 4: re-pairing or pairing a second controller appends; it never
// drops an existing key.
func TestRunKeepsExistingAuthorizedKeys(t *testing.T) {
	box := freshBox()
	box.files = map[string]string{AuthorizedKeys: `restrict,port-forwarding,command="/bin/false" ssh-ed25519 AAAAOTHER other-controller` + "\n"}
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	ak := box.files[AuthorizedKeys]
	if !strings.Contains(ak, "AAAAOTHER") || !strings.Contains(ak, "AAAATEST") {
		t.Fatalf("authorized_keys = %q", ak)
	}
	// And pairing the same controller again adds nothing.
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	if strings.Count(box.files[AuthorizedKeys], "AAAATEST") != 1 {
		t.Fatalf("duplicated key: %q", box.files[AuthorizedKeys])
	}
}

func TestRunRemovesABadSSHDDropIn(t *testing.T) {
	box := freshBox()
	box.rules = append([]rule{{"sshd -t", executor.Result{ExitCode: 255, Stderr: "bad config"}}}, box.rules...)
	_, err := Run(context.Background(), opts(t, box))
	var se *StepError
	if !errors.As(err, &se) || se.Step != "sshd" {
		t.Fatalf("err = %v, want a StepError for sshd", err)
	}
	if !strings.Contains(strings.Join(box.cmds, "\n"), "rm -f "+DropInPath) {
		t.Fatal("a drop-in that failed sshd -t was left in place")
	}
}

func TestRunRefusesWithoutTheSSHDInclude(t *testing.T) {
	box := freshBox()
	box.rules = append([]rule{{"Include", executor.Result{ExitCode: 1}}}, box.rules...)
	_, err := Run(context.Background(), opts(t, box))
	var se *StepError
	if !errors.As(err, &se) || se.Step != "preflight" {
		t.Fatalf("err = %v, want preflight", err)
	}
}

func TestRunVerifiesTheUploadedBinary(t *testing.T) {
	box := freshBox()
	box.rules = append([]rule{{"sha256sum -c", executor.Result{ExitCode: 1, Stderr: "FAILED"}}}, box.rules...)
	_, err := Run(context.Background(), opts(t, box))
	var se *StepError
	if !errors.As(err, &se) || se.Step != "upload" {
		t.Fatalf("err = %v, want upload", err)
	}
}

func TestLocalSkipsSSHDAndEnrollsTheUID(t *testing.T) {
	box := freshBox()
	o := opts(t, box)
	o.Local, o.LocalUID, o.TransportKey = true, 501, ""
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(box.cmds, "\n")
	if strings.Contains(all, "sshd -t") {
		t.Error("local pairing touched sshd")
	}
	if !strings.Contains(all, "--local-uid 501") {
		t.Error("local uid not enrolled")
	}
}
```

Fix the address assertion in `TestRunPairsAFreshBox` to the single clear form when you write it: `if !strings.EqualFold(got.Hex(), agentAddrHex) { … }`.

- [ ] **Step 5: Run them to verify they fail**

Run: `go test ./internal/bootstrap/`
Expected: FAIL — package has no non-test files.

- [ ] **Step 6: Implement**

```go
// internal/bootstrap/unit.go
package bootstrap

// agentUnit runs the agent as root with the hardening that does not stop it
// from managing units and data directories. ProtectSystem=strict is
// deliberately absent: the agent's job is to change the system.
const agentUnit = `[Unit]
Description=jumpgate agent
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/lib/jumpgate/jumpgate agent run
Restart=on-failure
RestartSec=5
User=root
RuntimeDirectory=jumpgate
RuntimeDirectoryMode=0755
StateDirectory=jumpgate
LoadCredential=agent.key:/var/lib/jumpgate/agent.key
NoNewPrivileges=yes
PrivateTmp=yes
ProtectHome=read-only
ProtectKernelTunables=yes
ProtectControlGroups=yes
RestrictSUIDSGID=yes

[Install]
WantedBy=multi-user.target
`

// sshdDropIn confines the tunnel user to reaching unix sockets.
const sshdDropIn = `# Managed by jumpgate. The jumpgate user may only open unix-socket channels.
Match User jumpgate
    AllowTcpForwarding no
    AllowStreamLocalForwarding local
    PermitTTY no
    X11Forwarding no
    AllowAgentForwarding no
`
```

```go
// internal/bootstrap/bootstrap.go

// Package bootstrap installs and pairs a jumpgate agent over a privileged
// executor. Every step is idempotent, so re-running it repairs a half-paired
// box and pairs a second controller without disturbing the first.
package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

const (
	BinaryPath     = "/usr/local/lib/jumpgate/jumpgate"
	UnitPath       = "/etc/systemd/system/jumpgate-agent.service"
	DropInPath     = "/etc/ssh/sshd_config.d/50-jumpgate.conf"
	NodePath       = "/etc/jumpgate/node.json"
	HomeDir        = "/var/lib/jumpgate/home"
	AuthorizedKeys = HomeDir + "/.ssh/authorized_keys"
)

// Options configures one pairing.
type Options struct {
	Exec            executor.Executor // privileged: root, or executor.Sudo(...)
	Local           bool              // pairing the machine the controller runs on
	LocalUID        int               // with Local: the controller's uid
	AgentBinary     func(arch string) (string, error)
	Controller      eip712.Address
	ControllerLabel string
	TransportKey    string // "ssh-ed25519 AAAA… comment"; empty with Local
	Wire            *catalog.WireConfig
	Event           func(step, line string)
}

// StepError names the step that failed, so the operator knows what was left
// in place.
type StepError struct {
	Step string
	Err  error
}

func (e *StepError) Error() string { return "pairing step " + e.Step + ": " + e.Err.Error() }
func (e *StepError) Unwrap() error { return e.Err }

type runner struct {
	ctx context.Context
	o   Options
}

func (r runner) emit(step, line string) {
	if r.o.Event != nil {
		r.o.Event(step, line)
	}
}

// sh runs cmd and turns a non-zero exit into an error carrying stderr's tail.
func (r runner) sh(step, cmd string) (string, error) {
	res, err := r.o.Exec.Run(r.ctx, cmd, nil)
	if err != nil {
		return "", &StepError{Step: step, Err: err}
	}
	if res.ExitCode != 0 {
		return res.Stdout, &StepError{Step: step, Err: fmt.Errorf("`%s` exited %d: %s", cmd, res.ExitCode, tail(res.Stderr))}
	}
	return res.Stdout, nil
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		s = "…" + s[len(s)-400:]
	}
	return s
}

// Run performs steps 1–7 of the spec and returns the agent's address. Step 8
// (a signed round trip) is the caller's: it holds the controller's key.
func Run(ctx context.Context, o Options) (eip712.Address, error) {
	r := runner{ctx: ctx, o: o}
	arch, err := r.preflight()
	if err != nil {
		return eip712.Address{}, err
	}
	steps := []struct {
		name string
		fn   func() error
	}{
		{"upload", func() error { return r.upload(arch) }},
		{"user", r.user},
		{"sshd", r.sshd},
	}
	for _, s := range steps {
		r.emit(s.name, "start")
		if err := s.fn(); err != nil {
			return eip712.Address{}, err
		}
	}
	addr, err := r.identity()
	if err != nil {
		return eip712.Address{}, err
	}
	if err := r.policyAndNode(); err != nil {
		return eip712.Address{}, err
	}
	if err := r.service(); err != nil {
		return eip712.Address{}, err
	}
	return addr, nil
}

func (r runner) preflight() (string, error) {
	r.emit("preflight", "start")
	osName, err := r.sh("preflight", "uname -s")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(osName) != "Linux" {
		return "", &StepError{Step: "preflight", Err: fmt.Errorf("the agent runs on Linux; this box is %s", strings.TrimSpace(osName))}
	}
	if _, err := r.sh("preflight", "command -v systemctl >/dev/null && [ -d /run/systemd/system ]"); err != nil {
		return "", &StepError{Step: "preflight", Err: fmt.Errorf("systemd is required")}
	}
	if !r.o.Local {
		// Editing sshd_config itself risks locking the operator out; a drop-in
		// is only honoured when the main file includes the directory.
		if _, err := r.sh("preflight", `grep -Eq '^[[:space:]]*Include[[:space:]]+/etc/ssh/sshd_config\.d/\*\.conf' /etc/ssh/sshd_config`); err != nil {
			return "", &StepError{Step: "preflight", Err: fmt.Errorf("/etc/ssh/sshd_config does not Include /etc/ssh/sshd_config.d/*.conf; add that line first")}
		}
	}
	m, err := r.sh("preflight", "uname -m")
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(m) {
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	}
	return "", &StepError{Step: "preflight", Err: fmt.Errorf("unsupported architecture %q", strings.TrimSpace(m))}
}

func (r runner) upload(arch string) error {
	path, err := r.o.AgentBinary(arch)
	if err != nil {
		return &StepError{Step: "upload", Err: err}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return &StepError{Step: "upload", Err: err}
	}
	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])
	if res, err := r.o.Exec.Run(r.ctx, "sha256sum "+BinaryPath+" 2>/dev/null", nil); err == nil && res.ExitCode == 0 && strings.HasPrefix(res.Stdout, want) {
		r.emit("upload", "already installed")
		return nil
	}
	tmp := BinaryPath + ".new"
	if err := r.o.Exec.WriteFile(r.ctx, tmp, content, 0o755); err != nil {
		return &StepError{Step: "upload", Err: err}
	}
	_, err = r.sh("upload", fmt.Sprintf("echo %s'  '%s | sha256sum -c - && mv -f %s %s", want, tmp, tmp, BinaryPath))
	return err
}

func (r runner) user() error {
	_, err := r.sh("user", "getent group jumpgate >/dev/null || groupadd --system jumpgate; "+
		"id -u jumpgate >/dev/null 2>&1 || useradd --system --gid jumpgate --home-dir "+HomeDir+
		" --create-home --shell /usr/sbin/nologin jumpgate")
	return err
}

func (r runner) sshd() error {
	if r.o.Local {
		return nil
	}
	if err := r.o.Exec.WriteFile(r.ctx, DropInPath, []byte(sshdDropIn), 0o644); err != nil {
		return &StepError{Step: "sshd", Err: err}
	}
	if _, err := r.sh("sshd", "sshd -t"); err != nil {
		_, _ = r.o.Exec.Run(r.ctx, "rm -f "+DropInPath, nil)
		return err
	}
	if _, err := r.sh("sshd", "systemctl reload ssh 2>/dev/null || systemctl reload sshd"); err != nil {
		return err
	}

	line := `restrict,port-forwarding,command="/bin/false" ` + strings.TrimSpace(r.o.TransportKey)
	keyField := strings.Fields(r.o.TransportKey)
	existing, _ := r.o.Exec.ReadFile(r.ctx, AuthorizedKeys)
	if len(keyField) >= 2 && strings.Contains(string(existing), keyField[1]) {
		return nil
	}
	merged := strings.TrimRight(string(existing), "\n")
	if merged != "" {
		merged += "\n"
	}
	merged += line + "\n"
	if _, err := r.sh("sshd", "install -d -m 0700 -o jumpgate -g jumpgate "+HomeDir+"/.ssh"); err != nil {
		return err
	}
	if err := r.o.Exec.WriteFile(r.ctx, AuthorizedKeys, []byte(merged), 0o600); err != nil {
		return &StepError{Step: "sshd", Err: err}
	}
	_, err := r.sh("sshd", "chown jumpgate:jumpgate "+AuthorizedKeys)
	return err
}

func (r runner) identity() (eip712.Address, error) {
	r.emit("identity", "start")
	out, err := r.sh("identity", BinaryPath+" agent init")
	if err != nil {
		return eip712.Address{}, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	addr, err := eip712.ParseAddress(strings.TrimSpace(lines[len(lines)-1]))
	if err != nil {
		return eip712.Address{}, &StepError{Step: "identity", Err: fmt.Errorf("agent init printed %q: %w", out, err)}
	}
	r.emit("identity", "agent address "+addr.Hex())
	return addr, nil
}

func (r runner) policyAndNode() error {
	r.emit("policy", "start")
	cmd := fmt.Sprintf("%s agent enroll --address %s --tier routine --label %s", BinaryPath, r.o.Controller.Hex(), shQuote(r.o.ControllerLabel))
	if r.o.Local {
		cmd += " --local-uid " + strconv.Itoa(r.o.LocalUID)
	}
	if _, err := r.sh("policy", cmd); err != nil {
		return err
	}
	if r.o.Wire == nil {
		return nil
	}
	b, err := json.Marshal(r.o.Wire)
	if err != nil {
		return &StepError{Step: "policy", Err: err}
	}
	if err := r.o.Exec.WriteFile(r.ctx, NodePath, b, 0o600); err != nil {
		return &StepError{Step: "policy", Err: err}
	}
	return nil
}

func (r runner) service() error {
	r.emit("service", "start")
	if err := r.o.Exec.WriteFile(r.ctx, UnitPath, []byte(agentUnit), 0o644); err != nil {
		return &StepError{Step: "service", Err: err}
	}
	// restart, not just enable --now: a re-pair that uploaded a new binary
	// must replace the running one.
	if _, err := r.sh("service", "systemctl daemon-reload && systemctl enable --now jumpgate-agent.service && systemctl restart jumpgate-agent.service"); err != nil {
		return err
	}
	out, err := r.sh("service", "systemctl is-active jumpgate-agent.service")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "active" {
		return &StepError{Step: "service", Err: fmt.Errorf("jumpgate-agent.service is %s; see journalctl -u jumpgate-agent", strings.TrimSpace(out))}
	}
	return nil
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'" }
```

The fake box's `ReadFile` returns `os.ErrNotExist` for a missing `authorized_keys`; `sshd()` treats any read error as "no existing keys", which is correct for a fresh box.

- [ ] **Step 7: Run the tests**

Run: `go test -race -v ./internal/bootstrap/ ./internal/executor/`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/bootstrap internal/executor/sudo.go internal/executor/sudo_test.go
git commit -m "feat(bootstrap): install and pair an agent idempotently

Preflight (Linux, systemd, sshd Include), checksum-verified binary upload, the
unprivileged jumpgate tunnel user, an sshd drop-in validated by sshd -t and
removed if it fails, a forced-command authorized_keys entry appended without
disturbing others, agent identity and enrollment, and the hardened unit.
executor.Sudo runs all of it through sudo -n for non-root logins.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 15: One controller server per user — lock, discovery, detach, unix socket

**Files:**
- Create: `internal/daemon/daemon.go`, `internal/daemon/detach_unix.go`, `internal/daemon/detach_windows.go`, `internal/daemon/daemon_test.go`
- Create: `cmd/jumpgate/web/embed.go` (package `web`); Modify: `cmd/jumpgate/main.go` (use it)
- Modify: `internal/server/server.go` (`ServeUnix`, `Config.Shutdown`, `POST /api/shutdown`)
- Test: `internal/server/unix_test.go`

**Interfaces:**
- Consumes: `filelock.TryLock`, `filelock.ErrLocked` (Task 2); `config.Dir` (Task 0).
- Produces:
  - `daemon.Info{PID int; Socket, HTTPAddr, Token, Version string; StartedAt time.Time}` (JSON camelCase)
  - `daemon.Acquire() (*Holder, error)` (`ErrAlreadyRunning` when held); `(*Holder) Publish(Info) error`; `(*Holder) Release()`
  - `daemon.Find(ctx context.Context) (Info, bool, error)` — running only if the lock is held **and** `GET /api/health` answers on the socket with the token
  - `daemon.EnsureRunning(ctx context.Context, exe string) (Info, error)` — starts `exe serve` detached and waits up to 10 s
  - `daemon.Stop(ctx context.Context) error` — `POST /api/shutdown`; never signals a pid
  - `(Info) Client() *http.Client`; `const BaseURL = "http://jumpgate"`
  - `daemon.RunDir() (string, error)` → `~/.jumpgate/run` (0700)
  - `server.Config.Shutdown func()`; `(*Server) ServeUnix(ctx context.Context, path string) error`
  - `web.FS` (`embed.FS`, rooted so `fs.Sub(web.FS, "dist")` is the UI)

- [ ] **Step 1: Write the failing daemon tests**

```go
// internal/daemon/daemon_test.go
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func isolate(t *testing.T) {
	t.Helper()
	// unix socket paths must stay short, so HOME itself is short here.
	home, err := os.MkdirTemp("/tmp", "jgh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
}

func TestAcquireIsSingleInstance(t *testing.T) {
	isolate(t)
	h, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second Acquire = %v, want ErrAlreadyRunning", err)
	}
	h.Release()
	h2, err := Acquire()
	if err != nil {
		t.Fatalf("Acquire after Release: %v", err)
	}
	h2.Release()
}

// Review Focus 5: a server.json left by a crashed server, naming a pid that
// now belongs to something else, is never trusted and never signalled.
func TestFindIgnoresAStaleDiscoveryFile(t *testing.T) {
	isolate(t)
	dir, _ := RunDir()
	stale := Info{PID: os.Getpid(), Socket: filepath.Join(dir, "server.sock"), Token: "x"}
	b, _ := json.Marshal(stale)
	if err := os.WriteFile(filepath.Join(dir, "server.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	_, running, err := Find(context.Background())
	if err != nil || running {
		t.Fatalf("Find = running %v, err %v; want not running", running, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "server.json")); !os.IsNotExist(err) {
		t.Fatal("stale server.json was left in place")
	}
}

// A live server: lock held, socket answering /api/health with the token.
func TestFindSeesALiveServer(t *testing.T) {
	isolate(t)
	h, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	dir, _ := RunDir()
	sock := filepath.Join(dir, "server.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" && r.Header.Get("Authorization") == "Bearer tok" {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		http.Error(w, "no", http.StatusUnauthorized)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if err := h.Publish(Info{PID: os.Getpid(), Socket: sock, Token: "tok", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	info, running, err := Find(context.Background())
	if err != nil || !running || info.Token != "tok" {
		t.Fatalf("Find = %+v, %v, %v", info, running, err)
	}
	fi, _ := os.Stat(filepath.Join(dir, "server.json"))
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("server.json mode %o, want 600 (it holds the token)", fi.Mode().Perm())
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/daemon/`
Expected: FAIL — package has no non-test files.

- [ ] **Step 3: Implement**

```go
// internal/daemon/daemon.go

// Package daemon keeps one controller server per user and lets every jumpgate
// command find it. The server outlives the command that started it, so a TUI
// or CLI can quit and come back to the same server and the same jobs.
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/filelock"
)

// ErrAlreadyRunning means another server holds the lock.
var ErrAlreadyRunning = errors.New("daemon: a jumpgate server is already running for this user")

// BaseURL is the authority used for requests over the unix socket; the
// transport ignores it.
const BaseURL = "http://jumpgate"

// Info is server.json.
type Info struct {
	PID       int       `json:"pid"`
	Socket    string    `json:"socket"`
	HTTPAddr  string    `json:"httpAddr"`
	Token     string    `json:"token"`
	Version   string    `json:"version"`
	StartedAt time.Time `json:"startedAt"`
}

// RunDir is ~/.jumpgate/run, created 0700.
func RunDir() (string, error) {
	base, err := config.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "run")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// Holder owns the single-instance lock for the server's lifetime.
type Holder struct {
	lock *filelock.Handle
	dir  string
}

// Acquire takes the lock or reports ErrAlreadyRunning.
func Acquire() (*Holder, error) {
	dir, err := RunDir()
	if err != nil {
		return nil, err
	}
	h, err := filelock.TryLock(filepath.Join(dir, "server.lock"))
	if errors.Is(err, filelock.ErrLocked) {
		return nil, ErrAlreadyRunning
	}
	if err != nil {
		return nil, err
	}
	return &Holder{lock: h, dir: dir}, nil
}

// Publish writes server.json (0600: it carries the session token) once the
// listeners are up.
func (h *Holder) Publish(info Info) error {
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(h.dir, "server.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(h.dir, "server.json"))
}

// Release removes server.json and drops the lock.
func (h *Holder) Release() {
	_ = os.Remove(filepath.Join(h.dir, "server.json"))
	_ = h.lock.Unlock()
}

// Client talks HTTP to the server over its unix socket.
func (i Info) Client() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", i.Socket)
	}}}
}

// Find reports a running server. Neither the file nor its pid is trusted: a
// server is running only if the lock is held and its socket answers
// /api/health with the token. A file failing either test is stale and removed.
func Find(ctx context.Context) (Info, bool, error) {
	dir, err := RunDir()
	if err != nil {
		return Info{}, false, err
	}
	path := filepath.Join(dir, "server.json")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Info{}, false, nil
	}
	if err != nil {
		return Info{}, false, err
	}
	var info Info
	if err := json.Unmarshal(b, &info); err != nil {
		_ = os.Remove(path)
		return Info{}, false, nil
	}

	if h, err := filelock.TryLock(filepath.Join(dir, "server.lock")); err == nil {
		// Nobody holds the lock: the file is left over from a dead server.
		_ = h.Unlock()
		_ = os.Remove(path)
		return Info{}, false, nil
	}

	hctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(hctx, http.MethodGet, BaseURL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)
	res, err := info.Client().Do(req)
	if err != nil {
		return info, false, nil
	}
	res.Body.Close()
	return info, res.StatusCode == http.StatusOK, nil
}

// EnsureRunning returns the running server, starting `exe serve` detached if
// there is none.
func EnsureRunning(ctx context.Context, exe string) (Info, error) {
	if info, ok, err := Find(ctx); err != nil || ok {
		return info, err
	}
	dir, err := RunDir()
	if err != nil {
		return Info{}, err
	}
	logf, err := os.OpenFile(filepath.Join(dir, "server.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return Info{}, err
	}
	defer logf.Close()
	cmd := exec.Command(exe, "serve", "--no-open")
	cmd.Stdout, cmd.Stderr = logf, logf
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return Info{}, fmt.Errorf("daemon: start server: %w", err)
	}
	_ = cmd.Process.Release()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if info, ok, err := Find(ctx); err == nil && ok {
			return info, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return Info{}, fmt.Errorf("daemon: the server did not come up within 10s; see %s", filepath.Join(dir, "server.log"))
}

// Stop asks the server to shut down over its authenticated API. It never
// signals a pid: a pid from a file may by now belong to another process.
func Stop(ctx context.Context) error {
	info, ok, err := Find(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("daemon: no jumpgate server is running")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+"/api/shutdown", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)
	res, err := info.Client().Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("daemon: shutdown answered %d", res.StatusCode)
	}
	return nil
}
```

```go
// internal/daemon/detach_unix.go
//go:build unix

package daemon

import (
	"os/exec"
	"syscall"
)

// detach starts the server in its own session, so closing the terminal that
// launched it does not deliver SIGHUP to it.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
```

```go
// internal/daemon/detach_windows.go
//go:build windows

package daemon

import (
	"os/exec"
	"syscall"
)

const (
	detachedProcess       = 0x00000008
	createNewProcessGroup = 0x00000200
)

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcessGroup}
}
```

- [ ] **Step 4: Run the daemon tests**

Run: `go test -race -v ./internal/daemon/ && GOOS=windows go vet ./internal/daemon/`
Expected: PASS; Windows vets cleanly.

- [ ] **Step 5: Write the failing server tests**

```go
// internal/server/unix_test.go
package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func TestServeUnixAndShutdown(t *testing.T) {
	dir, _ := os.MkdirTemp("/tmp", "jgs")
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "s.sock")
	token := NewSessionToken()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopped := make(chan struct{})
	s := New(Config{Token: token, UI: fstest.MapFS{}, Shutdown: func() { close(stopped) }})
	go s.ServeUnix(ctx, sock)

	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
	var res *http.Response
	var err error
	for i := 0; i < 50; i++ { // wait for the listener
		req, _ := http.NewRequest(http.MethodGet, "http://x/api/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if res, err = client.Do(req); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("health over unix socket: %v", err)
	}
	fi, _ := os.Stat(sock)
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("socket mode %o, want 600", fi.Mode().Perm())
	}

	req, _ := http.NewRequest(http.MethodPost, "http://x/api/shutdown", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = client.Do(req)
	if err != nil || res.StatusCode != http.StatusAccepted {
		t.Fatalf("shutdown: %v %v", res, err)
	}
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown hook not called")
	}
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `go test ./internal/server/ -run ServeUnix`
Expected: FAIL — unknown field `Shutdown`, undefined `ServeUnix`.

- [ ] **Step 7: Implement in `internal/server/server.go`**

Add to `Config`:

```go
	// Shutdown, if set, is called by POST /api/shutdown. jumpgate serve wires
	// it to cancel its own context, so `jumpgate stop` never has to signal a
	// pid it cannot be sure of.
	Shutdown func()
```

Register in `Handler()` (inside the authenticated mux):

```go
	mux.HandleFunc("POST /api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Shutdown == nil {
			writeError(w, http.StatusNotImplemented, "this server cannot be stopped over the API")
			return
		}
		w.WriteHeader(http.StatusAccepted)
		go s.cfg.Shutdown()
	})
```

Add:

```go
// ServeUnix serves the same authenticated handler on a unix socket (0600), for
// the CLI and TUI on this machine. The token is still required: the socket's
// permissions keep other users out, the token keeps other programs of this
// user honest about which server they talk to.
func (s *Server) ServeUnix(ctx context.Context, path string) error {
	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("server: %s exists and is not a socket", path)
		}
		_ = os.Remove(path)
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return err
	}
	srv := newHTTPServer("", s.Handler())
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
```

Note: `authMiddleware`'s Origin check (sub-project 0) only applies to cookie auth; the CLI uses Bearer, so it is unaffected.

- [ ] **Step 8: Move the embedded UI into its own package**

```go
// cmd/jumpgate/web/embed.go

// Package web embeds the built UI so every entry point (the tray app, jumpgate
// serve) serves the same files.
package web

import "embed"

// FS holds dist/; use fs.Sub(FS, "dist").
//
//go:embed all:dist
var FS embed.FS
```

In `cmd/jumpgate/main.go`, delete the `//go:embed` block and `embeddedUI`, import `github.com/valve-tech/jumpgate/cmd/jumpgate/web`, and replace `fs.Sub(embeddedUI, "web/dist")` with `fs.Sub(web.FS, "dist")`. The `web/` directory already holds `package.json` etc.; a `.go` file beside them is fine.

- [ ] **Step 9: Run tests and build**

Run: `go test -race ./internal/server/ ./internal/daemon/ && go build ./cmd/jumpgate`
Expected: PASS; builds.

- [ ] **Step 10: Commit**

```bash
git add internal/daemon internal/server cmd/jumpgate
git commit -m "feat(daemon): one controller server per user, discoverable and stoppable

A lock file makes the server single-instance; server.json (0600) says where its
unix socket is. A server counts as running only if the lock is held and its
socket answers health with the token, so a stale file is never trusted and no
pid is ever signalled: jumpgate stop calls the authenticated /api/shutdown.
The server also serves its API on that socket, and the embedded UI moves to
its own package so jumpgate serve can serve it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 16: Server endpoints — pair a target and run intents

**Files:**
- Modify: `internal/config/config.go` (`Target.Agent`, `Config.Controller`)
- Create: `internal/server/controller.go` (key, transport key, agent binary, seq store, agent target), `internal/server/intent.go`, `internal/server/pair.go`, `internal/server/intent_test.go`, `internal/server/pair_test.go`
- Modify: `internal/server/server.go` (`Config.Signer`), `internal/server/api.go` (route registration; relax `KeyPath` requirement)
- Modify: `internal/monitor/monitor.go` (export `FetchRefHead`)

**Interfaces:**
- Consumes: Tasks 5, 6, 8, 9, 10, 13, 14; `config.Update` (Task 2).
- Produces:
  - `config.AgentPairing{Address string; PairedAt time.Time; Transport string /* "ssh"|"local" */; NextSeq uint64}`; `Target.Agent *AgentPairing` (`json:"agent,omitempty"`)
  - `config.Controller{KeyStore string; KeyRef string; Address string}`; `Config.Controller *Controller` (`json:"controller,omitempty"`)
  - `server.Config.Signer signer.Signer`
  - `POST /api/targets/{id}/intent/{kind}` → `200 {"status":0|1|2,"result":…,"rejection":{…},"failure":{…}}`; `409 not_paired`; `502 bad_receipt`; `503 no_controller_key`; `504 unreachable`
  - `POST /api/targets/{id}/pair` body `{"sudo":bool}` → SSE events `{"step","line"}`, final `{"done":true,"agent":"0x…"}` or `{"step","err","code"}`; `409 unknown_host` (JSON, before streaming) with `{"fingerprint","host"}`
  - `monitor.FetchRefHead(ctx context.Context, hc *http.Client, url string) uint64`
  - Server helpers: `transportKeyPath() string` (`~/.jumpgate/ssh/jumpgate_ed25519`), `ensureTransportKey() (authorizedKeysLine string, err error)`, `agentBinary(arch string) (string, error)`, `knownHostsFile() string` (`~/.jumpgate/known_hosts`)

- [ ] **Step 1: Write the failing intent-endpoint tests**

```go
// internal/server/intent_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

type nopExec struct{}

func (nopExec) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) { return executor.Result{}, nil }
func (nopExec) WriteFile(context.Context, string, []byte, os.FileMode) error             { return nil }
func (nopExec) ReadFile(context.Context, string) ([]byte, error)                         { return nil, nil }
func (nopExec) Close() error                                                             { return nil }

// pairedLocal starts a real agent on a temp socket and saves a target that is
// already paired with it, so the endpoint can be exercised without bootstrap.
func pairedLocal(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	home, _ := os.MkdirTemp("/tmp", "jgi")
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)

	agentKey, _ := signer.GenerateKey()
	controller, _ := signer.GenerateKey()
	p := agent.Policy{Signers: []agent.SignerEntry{{Address: controller.Address().Hex(), Tier: agent.TierRoutine}}}
	_ = p.Save(filepath.Join(home, "policy.json"))
	_ = agent.InitReplay(filepath.Join(home, "replay.json"))
	a := agent.New(agent.Config{Key: agentKey, Exec: nopExec{}, PolicyPath: filepath.Join(home, "policy.json"),
		ReplayPath: filepath.Join(home, "replay.json"), NodePath: filepath.Join(home, "node.json")})
	sock := filepath.Join(home, "a.sock")
	ln, _ := agent.Listen(sock, -1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go agent.Serve(ctx, a, ln)

	_, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "local",
			Agent: &config.AgentPairing{Address: agentKey.Address().Hex(), Transport: "local", PairedAt: time.Now(), Socket: sock}})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	token := NewSessionToken()
	ts := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, Signer: controller}).Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func postIntent(t *testing.T, ts *httptest.Server, token, path, body string) (*http.Response, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res, out
}

func TestIntentEndpointRunsAVerifiedIntent(t *testing.T) {
	ts, token := pairedLocal(t)
	res, out := postIntent(t, ts, token, "/api/targets/box/intent/agent.info", `{}`)
	if res.StatusCode != http.StatusOK || out["status"].(float64) != float64(intent.StatusOK) {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	// The sequence advanced in config, so the next call does not collide.
	c, _ := config.Load()
	if c.Targets[0].Agent.NextSeq != 2 {
		t.Fatalf("NextSeq = %d, want 2", c.Targets[0].Agent.NextSeq)
	}
	if res, _ := postIntent(t, ts, token, "/api/targets/box/intent/agent.info", `{}`); res.StatusCode != http.StatusOK {
		t.Fatalf("second call: %d", res.StatusCode)
	}
}

func TestIntentEndpointRefusesUnpairedAndKeylessServers(t *testing.T) {
	ts, token := pairedLocal(t)
	_, _ = config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "raw", Mode: "local"})
		return nil
	})
	if res, out := postIntent(t, ts, token, "/api/targets/raw/intent/agent.info", `{}`); res.StatusCode != http.StatusConflict || out["code"] != "not_paired" {
		t.Fatalf("unpaired: %d %v", res.StatusCode, out)
	}
	keyless := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}}).Handler())
	defer keyless.Close()
	if res, out := postIntent(t, keyless, token, "/api/targets/box/intent/agent.info", `{}`); res.StatusCode != http.StatusServiceUnavailable || out["code"] != "no_controller_key" {
		t.Fatalf("keyless: %d %v", res.StatusCode, out)
	}
}
```

`AgentPairing` needs a `Socket string` (`json:"socket,omitempty"`) override for this test and for local development; production leaves it empty (default `/run/jumpgate/agent.sock`). Add it to the Interfaces list above when you implement.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/server/ -run IntentEndpoint`
Expected: FAIL — unknown fields `Agent`, `Signer`, `AgentPairing`.

- [ ] **Step 3: Implement config additions**

In `internal/config/config.go`:

```go
// AgentPairing records that a target runs a paired jumpgate agent.
type AgentPairing struct {
	Address   string    `json:"address"`   // the agent's signing address, fixed at pairing
	PairedAt  time.Time `json:"pairedAt"`
	Transport string    `json:"transport"` // "ssh" | "local"
	NextSeq   uint64    `json:"nextSeq"`   // next intent sequence for this controller
	Socket    string    `json:"socket,omitempty"`
}

// Controller is this machine's signing identity. The key itself is never here:
// KeyRef names where it lives (a path, a keychain item, an op:// reference).
type Controller struct {
	KeyStore string `json:"keyStore"` // "file" | "keychain" | "1password"
	KeyRef   string `json:"keyRef"`
	Address  string `json:"address"`
}
```

Add `Agent *AgentPairing `json:"agent,omitempty"`` to `Target` and `Controller *Controller `json:"controller,omitempty"`` to `Config`.

- [ ] **Step 4: Implement the controller helpers and the intent endpoint**

```go
// internal/server/controller.go
package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func jgPath(parts ...string) string {
	dir, _ := config.Dir()
	return filepath.Join(append([]string{dir}, parts...)...)
}

func knownHostsFile() string   { return jgPath("known_hosts") }
func transportKeyPath() string { return jgPath("ssh", "jumpgate_ed25519") }

// ensureTransportKey returns the controller's tunnel key as an authorized_keys
// line, creating an ed25519 key (0600) on first use. It is separate from the
// operator's login key so revoking jumpgate's access never touches theirs.
func ensureTransportKey() (string, error) {
	path := transportKeyPath()
	if b, err := os.ReadFile(path); err == nil {
		signer, err := ssh.ParsePrivateKey(b)
		if err != nil {
			return "", fmt.Errorf("transport key %s: %w", path, err)
		}
		return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))) + " jumpgate-controller", nil
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	block, err := ssh.MarshalPrivateKey(priv, "jumpgate-controller")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return "", err
	}
	return ensureTransportKey()
}

// agentBinary finds the Linux agent for arch: this binary when it already is
// one, otherwise ~/.jumpgate/agents/jumpgate-linux-<arch>, checked against the
// SHA256SUMS written beside it by scripts/build-agents.sh.
func agentBinary(arch string) (string, error) {
	if runtime.GOOS == "linux" && runtime.GOARCH == arch {
		return os.Executable()
	}
	dir := jgPath("agents")
	name := "jumpgate-linux-" + arch
	path := filepath.Join(dir, name)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("no agent binary for linux/%s at %s; run scripts/build-agents.sh", arch, path)
	}
	sums, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return "", fmt.Errorf("no SHA256SUMS beside %s", path)
	}
	got := sha256.Sum256(content)
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			if f[0] != hex.EncodeToString(got[:]) {
				return "", fmt.Errorf("%s does not match its SHA256SUMS entry", path)
			}
			return path, nil
		}
	}
	return "", fmt.Errorf("%s is not listed in SHA256SUMS", name)
}

// agentTarget is how this controller reaches t's agent: directly for a local
// target, otherwise as the jumpgate tunnel user with strict host-key checking.
func agentTarget(t config.Target) (agentclient.Target, error) {
	if t.Agent == nil {
		return agentclient.Target{}, fmt.Errorf("target %q is not paired", t.ID)
	}
	addr, err := eip712.ParseAddress(t.Agent.Address)
	if err != nil {
		return agentclient.Target{}, err
	}
	at := agentclient.Target{Agent: addr, Socket: t.Agent.Socket, Local: t.Agent.Transport == "local"}
	if !at.Local {
		if t.SSH == nil {
			return agentclient.Target{}, fmt.Errorf("target %q has no SSH address", t.ID)
		}
		home, _ := os.UserHomeDir()
		at.SSH = executor.SSHConfig{
			Host: t.SSH.Host, Port: t.SSH.Port, User: "jumpgate", KeyPath: transportKeyPath(), Jump: t.SSH.Jump,
			HostKey: executor.Strict(knownHostsFile(), filepath.Join(home, ".ssh", "known_hosts")),
		}
	}
	return at, nil
}

// configSeqs stores the next sequence in the target's pairing record.
type configSeqs struct{ targetID string }

func (s configSeqs) Next(eip712.Address) (uint64, error) {
	c, err := config.Load()
	if err != nil {
		return 0, err
	}
	for _, t := range c.Targets {
		if t.ID == s.targetID && t.Agent != nil {
			if t.Agent.NextSeq == 0 {
				return 1, nil
			}
			return t.Agent.NextSeq, nil
		}
	}
	return 0, fmt.Errorf("target %q is not paired", s.targetID)
}

func (s configSeqs) Set(_ eip712.Address, next uint64) error {
	_, err := config.Update(func(c *config.Config) error {
		for i := range c.Targets {
			if c.Targets[i].ID == s.targetID && c.Targets[i].Agent != nil {
				c.Targets[i].Agent.NextSeq = next
				return nil
			}
		}
		return fmt.Errorf("target %q is not paired", s.targetID)
	})
	return err
}
```

`ssh.MarshalPrivateKey` exists in `golang.org/x/crypto/ssh` since v0.17; the repo pins v0.54, so it is available.

```go
// internal/server/intent.go
package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/monitor"
)

type intentReply struct {
	Status    uint8             `json:"status"`
	Result    json.RawMessage   `json:"result,omitempty"`
	Rejection *intent.Rejection `json:"rejection,omitempty"`
	Failure   *intent.Failure   `json:"failure,omitempty"`
	RefHead   uint64            `json:"refHead,omitempty"` // status.read only
}

// handleIntent signs one intent with the controller key, sends it to the
// target's agent and returns the verified answer. Keys are only ever loaded
// by the server process; this route is why it must sit behind the Origin
// check, since it makes the server sign.
func (s *Server) handleIntent(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Signer == nil {
		writeErrorDetail(w, http.StatusServiceUnavailable, "this server has no controller key", "run `jumpgate keys init`, then restart the server", "no_controller_key")
		return
	}
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	t, ok := findTarget(cfg, r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "no such target")
		return
	}
	if t.Agent == nil {
		writeErrorDetail(w, http.StatusConflict, "this target has no paired agent", "run `jumpgate hosts add`", "not_paired")
		return
	}
	at, err := agentTarget(t)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	kind := r.PathValue("kind")
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the body")
		return
	}
	var payload any = json.RawMessage(body)
	switch kind {
	case intent.KindFirewallRead:
		payload = map[string]any{"overlayCidrs": cfg.TrustedOverlayCIDRs()}
	case intent.KindEndpointsRead:
		login := ""
		if t.SSH != nil {
			login = t.SSH.User + "@" + t.SSH.Host
		}
		payload = intent.EndpointsReadPayload{SSHLogin: login}
	}

	client, err := agentclient.Dial(r.Context(), at, s.cfg.Signer, configSeqs{targetID: t.ID})
	if err != nil {
		writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), "check that the box is up and reachable over SSH", "unreachable")
		return
	}
	defer client.Close()
	res, err := client.Do(r.Context(), kind, payload)
	switch {
	case errors.Is(err, agentclient.ErrBadReceipt):
		writeErrorDetail(w, http.StatusBadGateway, err.Error(), "the answer was not signed by this box's paired agent; do not trust this box until you re-pair it", "bad_receipt")
		return
	case errors.Is(err, agentclient.ErrUnreachable):
		writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), "check that the box is up and reachable over SSH", "unreachable")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	reply := intentReply{Status: res.Status, Result: res.Result, Rejection: res.Rejection, Failure: res.Failure}
	if kind == intent.KindStatusRead && res.Status == intent.StatusOK && cfg.RefRPCBase != "" && t.Wire != nil {
		reply.RefHead = monitor.FetchRefHead(r.Context(), http.DefaultClient, refRPCURL(cfg.RefRPCBase, t.Wire.ChainID))
	}
	writeJSON(w, http.StatusOK, reply)
}
```

Add a `refRPCURL(base string, chainID int) string` helper in `api.go` returning `fmt.Sprintf("%s/evm/%d", base, chainID)` and make `getMonitor` use it (it currently formats the same string inline). In `internal/monitor/monitor.go`, extract the body of `(*Monitor).fetchRefHead` into an exported `FetchRefHead(ctx, hc, url)` and have the method call it.

Register in `registerAPIRoutes`:

```go
	mux.HandleFunc("POST /api/targets/{id}/intent/{kind}", s.handleIntent)
	mux.HandleFunc("POST /api/targets/{id}/pair", s.handlePair)
```

Add `Signer signer.Signer` to `server.Config` with a comment: nil means the intent and pair routes answer 503.

- [ ] **Step 5: Run the intent tests**

Run: `go test -race ./internal/server/ -run IntentEndpoint`
Expected: PASS.

- [ ] **Step 6: Write the failing pair test**

```go
// internal/server/pair_test.go
package server

import (
	"bufio"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"net/http/httptest"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// A host whose key nobody confirmed is refused before anything runs on it,
// with the fingerprint the CLI shows the operator.
func TestPairRefusesAnUnconfirmedHost(t *testing.T) {
	d, keyPath := startPairTestSSHD(t) // copy of executor's startTestSSHD: host key, client key path
	t.Setenv("HOME", shortHome(t))
	_, _ = config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "ssh", SSH: &executorSSH{Host: d.host, Port: d.port, User: "root", KeyPath: keyPath}})
		return nil
	})
	ctrl, _ := signer.GenerateKey()
	token := NewSessionToken()
	ts := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, Signer: ctrl}).Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/targets/box/pair", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, want 409", res.StatusCode)
	}
	line, _ := bufio.NewReader(res.Body).ReadString('\n')
	if !strings.Contains(line, "unknown_host") || !strings.Contains(line, "SHA256:") {
		t.Fatalf("body %q", line)
	}
}
```

(`executorSSH` is `executor.SSHConfig`; `shortHome` is `os.MkdirTemp("/tmp", "jgp")` with cleanup. Copy `startTestSSHD`/`writePrivateKey` from `internal/executor/ssh_test.go` into this file as `startPairTestSSHD`.) A full pairing is covered by the container end-to-end test in Task 18, because it needs systemd, sshd and root.

- [ ] **Step 7: Run it to verify it fails**

Run: `go test ./internal/server/ -run Pair`
Expected: FAIL — `handlePair` undefined (or 404).

- [ ] **Step 8: Implement `handlePair`**

```go
// internal/server/pair.go
package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

type pairRequest struct {
	Sudo bool `json:"sudo"`
}

// handlePair installs and pairs the target's agent, streaming each step, then
// proves the pairing with a signed agent.info round trip before recording it.
// The host key must already be confirmed (the CLI does that with the
// operator); an unconfirmed host is refused before anything runs on it.
func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Signer == nil {
		writeErrorDetail(w, http.StatusServiceUnavailable, "this server has no controller key", "run `jumpgate keys init`, then restart the server", "no_controller_key")
		return
	}
	var req pairRequest
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req)
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	t, ok := findTarget(cfg, r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "no such target")
		return
	}

	// Pairing runs root commands on the box, so it must not interleave with a
	// setup run on the same target: claim the same per-target slot setup uses.
	release, ok := s.claimTargetSlot(t.ID)
	if !ok {
		writeErrorDetail(w, http.StatusConflict, "a setup or pairing run is already in progress on this target", "", "busy")
		return
	}
	defer release()

	ctx := context.WithoutCancel(r.Context()) // a browser disconnect must not abort a half-done pairing
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	var priv executor.Executor
	local := t.Mode == "local"
	if local {
		priv = executor.Sudo(executor.NewLocal())
	} else {
		home, _ := os.UserHomeDir()
		login := *t.SSH
		login.HostKey = executor.Strict(knownHostsFile(), filepath.Join(home, ".ssh", "known_hosts"))
		ex, err := executor.NewSSHContext(ctx, login)
		var unknown *executor.UnknownHostError
		if errors.As(err, &unknown) {
			writeJSON(w, http.StatusConflict, map[string]string{"code": "unknown_host", "host": unknown.Host, "fingerprint": unknown.Fingerprint})
			return
		}
		if err != nil {
			writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), "check the address, user and key", "unreachable")
			return
		}
		defer ex.Close()
		priv = ex
		if req.Sudo || t.SSH.User != "root" {
			priv = executor.Sudo(ex)
		}
	}

	sseHeaders(w)
	flusher, _ := w.(http.Flusher)
	send := func(v any) {
		writeSSEEvent(w, v)
		if flusher != nil {
			flusher.Flush()
		}
	}

	transportKey := ""
	if !local {
		if transportKey, err = ensureTransportKey(); err != nil {
			send(map[string]string{"step": "transport-key", "err": err.Error()})
			return
		}
	}
	addr, err := bootstrap.Run(ctx, bootstrap.Options{
		Exec: priv, Local: local, LocalUID: os.Getuid(), AgentBinary: agentBinary,
		Controller: s.cfg.Signer.Address(), ControllerLabel: hostLabel(), TransportKey: transportKey, Wire: t.Wire,
		Event: func(step, line string) { send(map[string]string{"step": step, "line": line}) },
	})
	if err != nil {
		var se *bootstrap.StepError
		step := ""
		if errors.As(err, &se) {
			step = se.Step
		}
		send(map[string]string{"step": step, "err": err.Error()})
		return
	}

	transport := "ssh"
	if local {
		transport = "local"
	}
	t.Agent = &config.AgentPairing{Address: addr.Hex(), Transport: transport, NextSeq: 1}
	at, err := agentTarget(t)
	if err == nil {
		var client *agentclient.Client
		if client, err = agentclient.Dial(ctx, at, s.cfg.Signer, agentclient.NewMemorySeqStore()); err == nil {
			defer client.Close()
			var res agentclient.Response
			res, err = client.Do(ctx, intent.KindAgentInfo, struct{}{})
			if err == nil && res.Status != intent.StatusOK {
				err = errors.New("agent refused agent.info: " + string(res.Result))
			}
		}
	}
	if err != nil {
		send(map[string]string{"step": "verify", "err": err.Error()})
		return
	}

	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Targets {
			if c.Targets[i].ID == t.ID {
				c.Targets[i].Agent = &config.AgentPairing{Address: addr.Hex(), Transport: transport, PairedAt: time.Now().UTC(), NextSeq: 2}
				return nil
			}
		}
		return errors.New("target disappeared during pairing")
	}); err != nil {
		send(map[string]string{"step": "record", "err": err.Error()})
		return
	}
	send(map[string]any{"done": true, "agent": addr.Hex()})
}

func hostLabel() string {
	h, err := os.Hostname()
	if err != nil {
		return "controller"
	}
	return h
}
```

Two notes for the implementer:
- `claimTargetSlot(id) (release func(), ok bool)`: after `fix/server-hardening` is merged, `api.go` has the per-target setup slot (`claimSetupRun`) that destructive routes use. Wrap it so pairing claims the same slot; read that code and reuse it rather than adding a second mechanism. Add `"io"` to the imports.
- The verification round trip uses a memory seq store and seq 1; recording `NextSeq: 2` afterwards keeps the persisted counter ahead of it.

In `handleAddTarget` (`api.go` around line 594), relax the SSH validation so `KeyPath` may be empty when an ssh-agent is available:

```go
		if t.SSH == nil || t.SSH.Host == "" || t.SSH.User == "" || (t.SSH.KeyPath == "" && os.Getenv("SSH_AUTH_SOCK") == "") {
```

and update its error message to "ssh targets need host, user, and a key path or a running ssh-agent".

- [ ] **Step 9: Run the server tests**

Run: `go test -race ./internal/server/ ./internal/monitor/ ./internal/config/`
Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/server internal/config internal/monitor
git commit -m "feat(server): pair targets and run signed intents through the API

POST /api/targets/{id}/pair refuses an unconfirmed host key with its
fingerprint, runs bootstrap in the target's setup slot on a context a client
disconnect cannot cancel, streams each step, and records the pairing only after
a signed agent.info round trip. POST /api/targets/{id}/intent/{kind} signs with
the controller key, keeps the sequence in config, and separates unreachable,
refused and bad-receipt answers.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 17: The `jumpgate` command line

**Files:**
- Create: `cmd/jumpgate/cli.go`, `cmd/jumpgate/cli_serve.go`, `cmd/jumpgate/cli_keys.go`, `cmd/jumpgate/cli_agent.go`, `cmd/jumpgate/cli_hosts.go`, `cmd/jumpgate/cli_intents.go`, `cmd/jumpgate/cli_test.go`
- Modify: `cmd/jumpgate/main.go` (rename `main` → `runApp`; new `main` dispatches)

**Interfaces:**
- Consumes: everything above.
- Produces: subcommands
  - `jumpgate` (no args) — unchanged web app behaviour (`runApp`)
  - `jumpgate serve [--bind ADDR] [--no-open]` — `daemon.Acquire`, load controller key if configured, HTTP + unix socket, `Publish`, run until signal or `/api/shutdown`
  - `jumpgate stop`
  - `jumpgate keys init [--store file|keychain|1password] [--ref REF]`, `jumpgate keys show`
  - `jumpgate agent init [--state-dir D] [--config-dir D]`, `agent enroll --address A --tier routine --label L [--local-uid N] [--config-dir D]`, `agent run`, `agent reset-replay --yes`
  - `jumpgate hosts add NAME (--ssh USER@HOST[:PORT] [--key PATH] [--jump USER@HOST[:PORT]] [--sudo] | --local)`, `jumpgate hosts list`
  - `jumpgate status|disk|endpoints|firewall HOST`, `jumpgate logs HOST [-n N]`, `jumpgate service HOST exec|beacon start|stop|restart`
  - Exit codes: `0` ok, `1` refused or failed, `2` usage, `3` unreachable, `4` security (bad receipt, host-key mismatch)

- [ ] **Step 1: Write the failing tests**

```go
// cmd/jumpgate/cli_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func TestParseSSHTarget(t *testing.T) {
	cases := map[string]struct {
		user, host string
		port       int
		ok         bool
	}{
		"root@203.0.113.7":        {"root", "203.0.113.7", 0, true},
		"ops@box.example.com:2222": {"ops", "box.example.com", 2222, true},
		"root@[2001:db8::1]:22":   {"root", "2001:db8::1", 22, true},
		"203.0.113.7":             {"", "", 0, false}, // user is required
		"root@":                   {"", "", 0, false},
		"root@host:notaport":      {"", "", 0, false},
	}
	for in, want := range cases {
		user, host, port, err := parseSSHTarget(in)
		if (err == nil) != want.ok || (want.ok && (user != want.user || host != want.host || port != want.port)) {
			t.Errorf("parseSSHTarget(%q) = %q %q %d %v", in, user, host, port, err)
		}
	}
}

// Every rejection code the agent can send has a one-line remedy; a new code
// without one fails here instead of reaching an operator as a bare string.
func TestEveryReasonHasARemedy(t *testing.T) {
	for _, code := range []string{
		intent.ReasonWrongAgent, intent.ReasonExpired, intent.ReasonClockSkew, intent.ReasonBadSignature,
		intent.ReasonUnauthorizedKind, intent.ReasonUnknownKind, intent.ReasonStaleSeq, intent.ReasonReplayedNonce,
		intent.ReasonBusy, intent.ReasonInvalidPayload, intent.ReasonValidation, intent.ReasonNotSetUp, intent.ReasonReplayState,
	} {
		if remedies[code] == "" {
			t.Errorf("no remedy for %s", code)
		}
	}
}

func TestExitCodes(t *testing.T) {
	cases := map[string]int{"ok": 0, "refused": 1, "failed": 1, "usage": 2, "unreachable": 3, "bad_receipt": 4, "host_key": 4}
	for outcome, want := range cases {
		if got := exitCode(outcome); got != want {
			t.Errorf("exitCode(%s) = %d, want %d", outcome, got, want)
		}
	}
}

// agent init is idempotent: the box keeps its identity across re-pairing.
func TestAgentInitKeepsItsIdentity(t *testing.T) {
	state, conf := t.TempDir(), t.TempDir()
	var out1, out2 strings.Builder
	if err := agentInit(&out1, state, conf); err != nil {
		t.Fatal(err)
	}
	if err := agentInit(&out2, state, conf); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out1.String()) != strings.TrimSpace(out2.String()) {
		t.Fatalf("identity changed: %q vs %q", out1.String(), out2.String())
	}
	if _, err := os.Stat(filepath.Join(state, "replay.json")); err != nil {
		t.Fatal("agent init did not create replay.json")
	}
	if fi, _ := os.Stat(conf); fi.Mode().Perm() != 0o700 {
		t.Errorf("config dir mode %o, want 700", fi.Mode().Perm())
	}
}

func TestAgentEnrollIsIdempotent(t *testing.T) {
	conf := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := agentEnroll(conf, "0x00000000000000000000000000000000000000cc", "routine", "laptop", 501); err != nil {
			t.Fatal(err)
		}
	}
	p, err := agent.LoadPolicy(filepath.Join(conf, "policy.json"))
	if err != nil || len(p.Signers) != 1 || len(p.LocalUIDs) != 1 {
		t.Fatalf("policy = %+v, %v", p, err)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/jumpgate/ -run 'ParseSSH|Remedy|ExitCodes|AgentInit|AgentEnroll'`
Expected: FAIL — undefined `parseSSHTarget`, `remedies`, `exitCode`, `agentInit`, `agentEnroll`.

- [ ] **Step 3: Implement dispatch and the shared helpers**

```go
// cmd/jumpgate/cli.go
package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// subcommands maps a first argument to its handler. Anything else falls
// through to runApp, so `jumpgate` and `jumpgate --bind …` behave as before.
var subcommands = map[string]func(args []string) int{
	"serve": cmdServe, "stop": cmdStop, "keys": cmdKeys, "agent": cmdAgent, "hosts": cmdHosts,
	"status": cmdIntent(intent.KindStatusRead), "disk": cmdIntent(intent.KindDiskRead),
	"endpoints": cmdIntent(intent.KindEndpointsRead), "firewall": cmdIntent(intent.KindFirewallRead),
	"logs": cmdLogs, "service": cmdService,
}

func main() {
	if len(os.Args) > 1 {
		if fn, ok := subcommands[os.Args[1]]; ok {
			os.Exit(fn(os.Args[2:]))
		}
	}
	runApp()
}

// remedies turns each rejection code into one line an operator can act on.
var remedies = map[string]string{
	intent.ReasonWrongAgent:       "this box's agent identity changed since pairing; re-pair with `jumpgate hosts add`",
	intent.ReasonExpired:          "the intent expired before the box checked it; check both clocks (enable NTP)",
	intent.ReasonClockSkew:        "the clocks disagree by more than 60s; enable NTP on this machine and the box",
	intent.ReasonBadSignature:     "the box could not verify this controller's signature; re-pair with `jumpgate hosts add`",
	intent.ReasonUnauthorizedKind: "this controller is not enrolled on the box for that; pair it with `jumpgate hosts add`",
	intent.ReasonUnknownKind:      "the box runs an older agent; re-run `jumpgate hosts add` to upgrade it",
	intent.ReasonStaleSeq:         "another process is using this controller key against this box",
	intent.ReasonReplayedNonce:    "the same intent was sent twice; retry the command",
	intent.ReasonBusy:             "another command from this controller is still running on the box",
	intent.ReasonInvalidPayload:   "the request was malformed; this is a jumpgate bug, please report it",
	intent.ReasonValidation:       "the box's node configuration is invalid; check /etc/jumpgate/node.json",
	intent.ReasonNotSetUp:         "no node is set up on this box yet; run setup first",
	intent.ReasonReplayState:      "the box's replay record is damaged; on the box, inspect /var/lib/jumpgate/replay.json then run `sudo jumpgate agent reset-replay --yes`",
}

func exitCode(outcome string) int {
	switch outcome {
	case "ok":
		return 0
	case "refused", "failed":
		return 1
	case "usage":
		return 2
	case "unreachable":
		return 3
	case "bad_receipt", "host_key":
		return 4
	}
	return 1
}

// parseSSHTarget reads user@host[:port], with IPv6 hosts in brackets.
func parseSSHTarget(s string) (user, host string, port int, err error) {
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return "", "", 0, fmt.Errorf("want user@host[:port], got %q", s)
	}
	user, rest := s[:at], s[at+1:]
	if h, p, splitErr := net.SplitHostPort(rest); splitErr == nil {
		n, convErr := strconv.Atoi(p)
		if convErr != nil || n <= 0 || n > 65535 {
			return "", "", 0, fmt.Errorf("bad port in %q", s)
		}
		return user, h, n, nil
	}
	if strings.Contains(rest, ":") && !strings.HasPrefix(rest, "[") {
		return "", "", 0, fmt.Errorf("bad port in %q", s)
	}
	return user, strings.Trim(rest, "[]"), 0, nil
}

func fail(format string, a ...any) int {
	fmt.Fprintf(os.Stderr, "jumpgate: "+format+"\n", a...)
	return exitCode("usage")
}
```

In `cmd/jumpgate/main.go`, rename `func main()` to `func runApp()`; it keeps reading `flag` from `os.Args[1:]` exactly as before.

- [ ] **Step 4: Implement `agent` subcommands**

```go
// cmd/jumpgate/cli_agent.go
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
)

const (
	defaultStateDir  = "/var/lib/jumpgate"
	defaultConfigDir = "/etc/jumpgate"
)

func cmdAgent(args []string) int {
	if len(args) == 0 {
		return fail("usage: jumpgate agent init|enroll|run|reset-replay")
	}
	fs := flag.NewFlagSet("agent "+args[0], flag.ContinueOnError)
	state := fs.String("state-dir", defaultStateDir, "agent state directory")
	conf := fs.String("config-dir", defaultConfigDir, "agent config directory")
	address := fs.String("address", "", "signer address to enroll")
	tier := fs.String("tier", "routine", "signer tier")
	label := fs.String("label", "", "signer label")
	localUID := fs.Int("local-uid", -1, "also allow this local uid on the socket")
	yes := fs.Bool("yes", false, "confirm reset-replay")
	if err := fs.Parse(args[1:]); err != nil {
		return exitCode("usage")
	}
	var err error
	switch args[0] {
	case "init":
		err = agentInit(os.Stdout, *state, *conf)
	case "enroll":
		err = agentEnroll(*conf, *address, *tier, *label, *localUID)
	case "run":
		err = agentRun(*state, *conf)
	case "reset-replay":
		if !*yes {
			return fail("reset-replay reopens a replay window for intents captured before now; re-run with --yes once you know why the record broke")
		}
		path := filepath.Join(*state, "replay.json")
		_ = os.Rename(path, path+".broken-"+time.Now().UTC().Format("20060102T150405"))
		err = agent.InitReplay(path)
	default:
		return fail("unknown agent subcommand %q", args[0])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "jumpgate agent:", err)
		return 1
	}
	return 0
}

// agentInit creates the agent key and replay record if absent and prints the
// agent's address. It never replaces an existing key: the box keeps its
// identity across re-pairing.
func agentInit(out io.Writer, stateDir, configDir string) error {
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(configDir, 0o700); err != nil {
		return err
	}
	keyPath := filepath.Join(stateDir, "agent.key")
	k, err := signer.LoadKeyFile(keyPath)
	if errors.Is(err, fs.ErrNotExist) {
		k, err = signer.GenerateKeyFile(keyPath)
	}
	if err != nil {
		return err
	}
	if err := agent.InitReplay(filepath.Join(stateDir, "replay.json")); err != nil {
		return err
	}
	fmt.Fprintln(out, k.Address().Hex())
	return nil
}

func agentEnroll(configDir, address, tier, label string, localUID int) error {
	a, err := eip712.ParseAddress(address)
	if err != nil {
		return err
	}
	if tier != string(agent.TierRoutine) && tier != string(agent.TierApproval) {
		return fmt.Errorf("tier must be routine or approval")
	}
	path := filepath.Join(configDir, "policy.json")
	p, err := agent.LoadPolicy(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	p.AddSigner(agent.SignerEntry{Address: a.Hex(), Tier: agent.Tier(tier), Label: label})
	if localUID >= 0 {
		p.AddLocalUID(localUID)
	}
	return p.Save(path)
}

// agentRun serves until SIGTERM. systemd hands the key in through
// LoadCredential; outside systemd it is read from the state directory.
func agentRun(stateDir, configDir string) error {
	keyPath := filepath.Join(stateDir, "agent.key")
	if d := os.Getenv("CREDENTIALS_DIRECTORY"); d != "" {
		keyPath = filepath.Join(d, "agent.key")
	}
	k, err := signer.LoadKeyFile(keyPath)
	if err != nil {
		return err
	}
	a := agent.New(agent.Config{
		Key: k, Exec: executor.NewLocal(),
		PolicyPath: filepath.Join(configDir, "policy.json"),
		ReplayPath: filepath.Join(stateDir, "replay.json"),
		NodePath:   filepath.Join(configDir, "node.json"),
	})
	ln, err := agent.Listen(agentclient.DefaultSocket, agent.JumpgateGID())
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stderr, "jumpgate agent %s listening on %s\n", a.Address().Hex(), agentclient.DefaultSocket)
	return agent.Serve(ctx, a, ln)
}
```

(Add `"errors"` and `"io/fs"` to the imports. `signer.LoadKeyFile` and `agent.LoadPolicy` wrap the stat error with `%w`, so `errors.Is(err, fs.ErrNotExist)` sees through them.)

- [ ] **Step 5: Implement `serve`, `stop` and `keys`**

```go
// cmd/jumpgate/cli_serve.go
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/valve-tech/jumpgate/cmd/jumpgate/web"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/server"
	"github.com/valve-tech/jumpgate/internal/signer"
)

func cmdServe(args []string) int {
	fset := flag.NewFlagSet("serve", flag.ContinueOnError)
	bind := fset.String("bind", "127.0.0.1:8799", bindFlagUsage)
	_ = fset.Bool("no-open", true, "accepted for symmetry; serve never opens a browser")
	if err := fset.Parse(args); err != nil {
		return exitCode("usage")
	}
	if _, err := config.MigrateLegacyDir(); err != nil {
		return fail("%v", err)
	}
	holder, err := daemon.Acquire()
	if err != nil {
		return fail("%v", err)
	}
	defer holder.Release()

	cfg, err := config.Load()
	if err != nil {
		return fail("load config: %v", err)
	}
	var sgn signer.Signer
	if cfg.Controller != nil {
		k, err := signer.Open(context.Background(), signer.Store(cfg.Controller.KeyStore), cfg.Controller.KeyRef)
		if err != nil {
			return fail("load controller key: %v", err)
		}
		sgn = k
	}
	ui, err := fs.Sub(web.FS, "dist")
	if err != nil {
		return fail("embedded UI: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	token := server.NewSessionToken()
	s := server.New(server.Config{Bind: *bind, Token: token, UI: ui, Signer: sgn, Shutdown: stop})

	dir, _ := daemon.RunDir()
	sock := filepath.Join(dir, "server.sock")
	errs := make(chan error, 2)
	go func() { errs <- s.ListenAndServe(ctx) }()
	go func() { errs <- s.ServeUnix(ctx, sock) }()
	time.Sleep(100 * time.Millisecond) // let the listeners bind before advertising them
	if err := holder.Publish(daemon.Info{PID: os.Getpid(), Socket: sock, HTTPAddr: *bind, Token: token, Version: buildinfo.Version(), StartedAt: time.Now().UTC()}); err != nil {
		return fail("publish: %v", err)
	}
	fmt.Fprintf(os.Stderr, "jumpgate server on %s and %s\n", *bind, sock)
	select {
	case <-ctx.Done():
		return 0
	case err := <-errs:
		if err != nil {
			return fail("%v", err)
		}
		return 0
	}
}

func cmdStop([]string) int {
	if err := daemon.Stop(context.Background()); err != nil {
		return fail("%v", err)
	}
	return 0
}
```

Replace the `time.Sleep` with a readiness check if it proves flaky: poll `daemon.Find` is not possible before `Publish`, so instead dial `sock` until it accepts (max 2 s) before publishing.

```go
// cmd/jumpgate/cli_keys.go
package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/signer"
)

func cmdKeys(args []string) int {
	if len(args) == 0 {
		return fail("usage: jumpgate keys init|show")
	}
	switch args[0] {
	case "show":
		c, err := config.Load()
		if err != nil || c.Controller == nil {
			return fail("no controller key; run `jumpgate keys init`")
		}
		fmt.Printf("%s (%s: %s)\n", c.Controller.Address, c.Controller.KeyStore, c.Controller.KeyRef)
		return 0
	case "init":
		fset := flag.NewFlagSet("keys init", flag.ContinueOnError)
		store := fset.String("store", string(signer.DefaultStore()), "file | keychain | 1password")
		ref := fset.String("ref", "", "key file path, keychain item, or op://vault/item/field")
		if err := fset.Parse(args[1:]); err != nil {
			return exitCode("usage")
		}
		if c, _ := config.Load(); c.Controller != nil {
			return fail("a controller key already exists (%s); jumpgate never replaces one silently", c.Controller.Address)
		}
		if *ref == "" {
			switch signer.Store(*store) {
			case signer.StoreFile:
				*ref = jgFile("keys", "controller.key")
			case signer.StoreKeychain:
				*ref = "controller"
			default:
				return fail("--ref op://<vault>/<item>/<field> is required for 1password")
			}
		}
		k, err := signer.Create(context.Background(), signer.Store(*store), *ref)
		if err != nil {
			return fail("%v", err)
		}
		if _, err := config.Update(func(c *config.Config) error {
			c.Controller = &config.Controller{KeyStore: *store, KeyRef: *ref, Address: k.Address().Hex()}
			return nil
		}); err != nil {
			return fail("%v", err)
		}
		fmt.Println(k.Address().Hex())
		fmt.Println("restart the server to use it: jumpgate stop")
		return 0
	}
	return fail("unknown keys subcommand %q", args[0])
}
```

Add `jgFile(parts ...string) string` to `cli.go`: `filepath.Join(config.Dir(), parts...)` (ignore the error branch by falling back to `"."`).

- [ ] **Step 6: Implement `hosts` and the intent commands**

```go
// cmd/jumpgate/cli_hosts.go
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func cmdHosts(args []string) int {
	if len(args) == 0 {
		return fail("usage: jumpgate hosts add|list")
	}
	switch args[0] {
	case "list":
		c, err := config.Load()
		if err != nil {
			return fail("%v", err)
		}
		for _, t := range c.Targets {
			paired := "not paired"
			if t.Agent != nil {
				paired = "agent " + t.Agent.Address
			}
			fmt.Printf("%-16s %-6s %s\n", t.ID, t.Mode, paired)
		}
		return 0
	case "add":
		return hostsAdd(args[1:])
	}
	return fail("unknown hosts subcommand %q", args[0])
}

func hostsAdd(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fail("usage: jumpgate hosts add NAME (--ssh USER@HOST[:PORT] [--key PATH] [--jump USER@HOST[:PORT]] [--sudo] | --local)")
	}
	name := args[0]
	fset := flag.NewFlagSet("hosts add", flag.ContinueOnError)
	sshArg := fset.String("ssh", "", "user@host[:port]")
	key := fset.String("key", "", "private key (optional with ssh-agent)")
	jumpArg := fset.String("jump", "", "jump host user@host[:port]")
	sudo := fset.Bool("sudo", false, "run setup commands through sudo -n")
	local := fset.Bool("local", false, "pair this machine")
	if err := fset.Parse(args[1:]); err != nil || (*local == (*sshArg != "")) {
		return fail("give exactly one of --ssh or --local")
	}

	ctx := context.Background()
	exe, _ := os.Executable()
	info, err := daemon.EnsureRunning(ctx, exe)
	if err != nil {
		return fail("%v", err)
	}

	target := map[string]any{"id": name, "mode": "local"}
	if !*local {
		cfg, err := sshConfigFrom(*sshArg, *key, *jumpArg)
		if err != nil {
			return fail("%v", err)
		}
		if code := confirmHostKeys(ctx, cfg); code != 0 {
			return code
		}
		target = map[string]any{"id": name, "mode": "ssh", "ssh": cfg}
	}
	if code := post(info, "/api/targets", target, nil); code != 0 {
		return code
	}
	return streamPair(info, name, *sudo)
}

func sshConfigFrom(login, key, jump string) (executor.SSHConfig, error) {
	user, host, port, err := parseSSHTarget(login)
	if err != nil {
		return executor.SSHConfig{}, err
	}
	cfg := executor.SSHConfig{Host: host, Port: port, User: user, KeyPath: key, HostKeyFile: jgFile("known_hosts")}
	if jump != "" {
		ju, jh, jp, err := parseSSHTarget(jump)
		if err != nil {
			return executor.SSHConfig{}, fmt.Errorf("--jump: %w", err)
		}
		cfg.Jump = &executor.SSHConfig{Host: jh, Port: jp, User: ju, KeyPath: key, HostKeyFile: jgFile("known_hosts")}
	}
	return cfg, nil
}

// confirmHostKeys shows the operator every unconfirmed host key on the path
// (jump host first) and records it only on an explicit "yes". A key that
// contradicts one already on record is a hard stop.
func confirmHostKeys(ctx context.Context, cfg executor.SSHConfig) int {
	hops := []executor.SSHConfig{}
	if cfg.Jump != nil {
		hops = append(hops, *cfg.Jump)
	}
	hops = append(hops, cfg)
	home, _ := os.UserHomeDir()
	for i, hop := range hops {
		probe := hop
		if i > 0 && cfg.Jump != nil {
			probe.Jump = cfg.Jump
		}
		key, err := executor.CaptureHostKey(ctx, probe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "jumpgate: cannot reach %s: %v\n", hop.Host, err)
			return exitCode("unreachable")
		}
		port := hop.Port
		if port == 0 {
			port = 22
		}
		hostport := net.JoinHostPort(hop.Host, strconv.Itoa(port))
		check := executor.Strict(jgFile("known_hosts"), filepath.Join(home, ".ssh", "known_hosts"))
		err = check(hostport, nil, key)
		var unknown *executor.UnknownHostError
		switch {
		case err == nil:
			continue
		case errors.As(err, &unknown):
			fmt.Printf("%s presents host key %s\nCompare it with the box's console (`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub`).\nType yes to trust it: ", hostport, unknown.Fingerprint)
			answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			if strings.TrimSpace(answer) != "yes" {
				return fail("not trusted; nothing was changed")
			}
			if err := executor.RecordHostKey(jgFile("known_hosts"), hostport, key); err != nil {
				return fail("%v", err)
			}
		default:
			fmt.Fprintf(os.Stderr, "jumpgate: %v\n", err)
			return exitCode("host_key")
		}
	}
	return 0
}

func post(info daemon.Info, path string, body any, out any) int {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, daemon.BaseURL+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+info.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := info.Client().Do(req)
	if err != nil {
		return fail("server: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		var e map[string]any
		_ = json.NewDecoder(res.Body).Decode(&e)
		return fail("%s: %v", path, e)
	}
	if out != nil {
		_ = json.NewDecoder(res.Body).Decode(out)
	}
	return 0
}

// streamPair prints each pairing event as it arrives.
func streamPair(info daemon.Info, name string, sudo bool) int {
	b, _ := json.Marshal(map[string]bool{"sudo": sudo})
	req, _ := http.NewRequest(http.MethodPost, daemon.BaseURL+"/api/targets/"+name+"/pair", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+info.Token)
	res, err := info.Client().Do(req)
	if err != nil {
		return fail("server: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		var e map[string]any
		_ = json.NewDecoder(res.Body).Decode(&e)
		if e["code"] == "unknown_host" {
			fmt.Fprintf(os.Stderr, "jumpgate: %v presents an unconfirmed key %v\n", e["host"], e["fingerprint"])
			return exitCode("host_key")
		}
		return fail("pair: %v", e)
	}
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		line, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line), &ev) != nil {
			continue
		}
		switch {
		case ev["done"] == true:
			fmt.Printf("paired: agent %v\nRecommended now: disable root SSH login on the box (PermitRootLogin no). Keep console access as the way back in.\n", ev["agent"])
			return 0
		case ev["err"] != nil:
			fmt.Fprintf(os.Stderr, "jumpgate: pairing failed at %v: %v\n", ev["step"], ev["err"])
			return 1
		default:
			fmt.Printf("[%v] %v\n", ev["step"], ev["line"])
		}
	}
	return fail("the server closed the stream before pairing finished")
}
```

```go
// cmd/jumpgate/cli_intents.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func cmdIntent(kind string) func([]string) int {
	return func(args []string) int {
		if len(args) != 1 {
			return fail("usage: jumpgate %s HOST", kind)
		}
		return runIntent(args[0], kind, struct{}{})
	}
}

func cmdLogs(args []string) int {
	fset := flag.NewFlagSet("logs", flag.ContinueOnError)
	n := fset.Int("n", intent.LogsDefaultN, "lines per unit (max 2000)")
	if len(args) == 0 {
		return fail("usage: jumpgate logs HOST [-n N]")
	}
	if err := fset.Parse(args[1:]); err != nil {
		return exitCode("usage")
	}
	return runIntent(args[0], intent.KindLogsRead, intent.LogsReadPayload{N: *n})
}

func cmdService(args []string) int {
	if len(args) != 3 {
		return fail("usage: jumpgate service HOST exec|beacon start|stop|restart")
	}
	return runIntent(args[0], intent.KindServiceAction, intent.ServiceActionPayload{Service: args[1], Action: args[2]})
}

type reply struct {
	Status    uint8             `json:"status"`
	Result    json.RawMessage   `json:"result"`
	Rejection *intent.Rejection `json:"rejection"`
	Failure   *intent.Failure   `json:"failure"`
	RefHead   uint64            `json:"refHead"`
	Code      string            `json:"code"`
	Error     string            `json:"error"`
	Hint      string            `json:"hint"`
}

func runIntent(host, kind string, payload any) int {
	ctx := context.Background()
	exe, _ := os.Executable()
	info, err := daemon.EnsureRunning(ctx, exe)
	if err != nil {
		return fail("%v", err)
	}
	var r reply
	b, _ := json.Marshal(payload)
	req := mustRequest(ctx, info, "/api/targets/"+host+"/intent/"+kind, b)
	res, err := info.Client().Do(req)
	if err != nil {
		return fail("server: %v", err)
	}
	defer res.Body.Close()
	_ = json.NewDecoder(res.Body).Decode(&r)

	switch {
	case r.Code == "unreachable":
		fmt.Fprintf(os.Stderr, "jumpgate: could not reach the agent on %s: %s\n", host, r.Error)
		return exitCode("unreachable")
	case r.Code == "bad_receipt":
		fmt.Fprintf(os.Stderr, "jumpgate: SECURITY: %s\n%s\n", r.Error, r.Hint)
		return exitCode("bad_receipt")
	case r.Code != "":
		fmt.Fprintf(os.Stderr, "jumpgate: %s (%s)\n", r.Error, r.Hint)
		return exitCode("usage")
	case r.Rejection != nil:
		fmt.Fprintf(os.Stderr, "jumpgate: %s refused (%s): %s\n  → %s\n", host, r.Rejection.Code, r.Rejection.Message, remedies[r.Rejection.Code])
		if r.Rejection.Code == intent.ReasonClockSkew && r.Rejection.AgentTime != 0 {
			fmt.Fprintf(os.Stderr, "  this machine: %s, the box: %s\n", time.Now().UTC().Format(time.RFC3339), time.Unix(r.Rejection.AgentTime, 0).UTC().Format(time.RFC3339))
		}
		return exitCode("refused")
	case r.Failure != nil:
		fmt.Fprintf(os.Stderr, "jumpgate: %s ran it and it failed: %s\n", host, r.Failure.Message)
		return exitCode("failed")
	}
	var pretty any
	_ = json.Unmarshal(r.Result, &pretty)
	if r.RefHead != 0 {
		pretty = map[string]any{"node": pretty, "refHead": r.RefHead}
	}
	out, _ := json.MarshalIndent(pretty, "", "  ")
	fmt.Println(string(out))
	return 0
}
```

Add `mustRequest(ctx, info, path, body) *http.Request` in `cli.go`: a POST to `daemon.BaseURL+path` with the Bearer token and JSON content type. The server's `writeErrorDetail` JSON field names must match `reply.Code`, `reply.Error` and `reply.Hint` — check `writeErrorDetail` in `internal/server/containers.go` and use its actual field names in the `reply` struct tags.

- [ ] **Step 7: Run the tests and build**

Run: `go test -race ./cmd/jumpgate/ && go build ./cmd/jumpgate && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /dev/null ./cmd/jumpgate`
Expected: PASS; the native build and the cgo-free Linux arm64 agent build both succeed.

- [ ] **Step 8: Smoke test locally**

```bash
./jumpgate keys init --store file
./jumpgate serve &   # or rely on EnsureRunning
./jumpgate hosts list
./jumpgate stop
```
Expected: an address is printed; `hosts list` prints the configured targets; `stop` exits 0 and `~/.jumpgate/run/server.json` is gone.

- [ ] **Step 9: Commit**

```bash
git add cmd/jumpgate
git commit -m "feat(cli): jumpgate serve, stop, keys, agent, hosts and intent commands

The bare command keeps the web app. New subcommands run the controller server,
manage the controller key, run and enroll the agent on a box, pair hosts with
an explicit host-key confirmation, and send signed intents, printing a remedy
for every rejection code and exiting 0/1/2/3/4 for ok, refused, usage,
unreachable and security failures.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 18: Agent builds and the container end-to-end test

**Files:**
- Create: `scripts/build-agents.sh`, `scripts/e2e/Dockerfile`, `scripts/e2e-agent.sh`, `internal/bootstrap/e2e_test.go` (build tag `e2e`)

**Interfaces:**
- Consumes: the `jumpgate` binary (Task 17), `bootstrap.Run`, `agentclient`, `executor`.
- Produces: `~/.jumpgate/agents/jumpgate-linux-{amd64,arm64}` + `SHA256SUMS`; `make`-free entry points `scripts/build-agents.sh` and `scripts/e2e-agent.sh`.

- [ ] **Step 1: Write the build script**

```bash
#!/usr/bin/env bash
# scripts/build-agents.sh — cross-build the Linux agent binaries that pairing
# uploads, plus the SHA256SUMS the server checks them against.
set -euo pipefail
out="${1:-$HOME/.jumpgate/agents}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
mkdir -p "$out"
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$version" \
    -o "$out/jumpgate-linux-$arch" ./cmd/jumpgate
done
( cd "$out" && shasum -a 256 jumpgate-linux-amd64 jumpgate-linux-arm64 > SHA256SUMS )
echo "agents in $out"
```

Run: `chmod +x scripts/build-agents.sh && scripts/build-agents.sh "$(mktemp -d)"`
Expected: two binaries and a `SHA256SUMS` with two lines.

- [ ] **Step 2: Write the test box image**

```dockerfile
# scripts/e2e/Dockerfile — a systemd + sshd box to pair against.
ARG BASE=debian:12
FROM ${BASE}
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      systemd systemd-sysv openssh-server sudo procps ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && systemctl enable ssh \
    && mkdir -p /root/.ssh && chmod 700 /root/.ssh
STOPSIGNAL SIGRTMIN+3
CMD ["/lib/systemd/systemd"]
```

- [ ] **Step 3: Write the failing end-to-end test**

```go
// internal/bootstrap/e2e_test.go
//go:build e2e

package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// Driven by scripts/e2e-agent.sh, which starts the container and exports
// JUMPGATE_E2E_{PORT,ROOT_KEY,TRANSPORT_KEY,TRANSPORT_PUB,AGENTS}.
func e2eEnv(t *testing.T, k string) string {
	v := os.Getenv(k)
	if v == "" {
		t.Skip(k + " not set; run scripts/e2e-agent.sh")
	}
	return v
}

func TestE2EPairAndRoundTrip(t *testing.T) {
	port, _ := strconv.Atoi(e2eEnv(t, "JUMPGATE_E2E_PORT"))
	root := executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root", KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	controller, _ := signer.GenerateKey()
	agentAddr, err := Run(ctx, Options{
		Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"),
		AgentBinary: func(arch string) (string, error) { return e2eEnv(t, "JUMPGATE_E2E_AGENTS") + "/jumpgate-linux-" + arch, nil },
		Event:       func(step, line string) { t.Logf("[%s] %s", step, line) },
	})
	if err != nil {
		t.Fatal(err)
	}

	tunnel := agentclient.Target{Agent: agentAddr, SSH: executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "jumpgate",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}}
	c, err := agentclient.Dial(ctx, tunnel, controller, agentclient.NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(ctx, intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("agent.info over the tunnel: %+v %v", res, err)
	}

	// The tunnel user can do nothing else: no shell, no TCP forwarding.
	tun, err := executor.NewSSHContext(ctx, tunnel.SSH)
	if err == nil {
		r, _ := tun.Run(ctx, "id", nil)
		if strings.Contains(r.Stdout, "uid=") {
			t.Fatal("the tunnel user got a shell")
		}
		tun.Close()
	}

	// Re-running bootstrap keeps the agent's identity.
	again, err := Run(ctx, Options{Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"),
		AgentBinary:  func(arch string) (string, error) { return e2eEnv(t, "JUMPGATE_E2E_AGENTS") + "/jumpgate-linux-" + arch, nil }})
	if err != nil || again != agentAddr {
		t.Fatalf("re-pair: %s, %v; want the same agent %s", again.Hex(), err, agentAddr.Hex())
	}
}

// The spec's acceptance: a captured intent replayed to the same agent is
// stale_seq; the same intent addressed elsewhere is wrong_agent.
func TestE2EReplayAndWrongAgent(t *testing.T) {
	port, _ := strconv.Atoi(e2eEnv(t, "JUMPGATE_E2E_PORT"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()})
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()
	controller, _ := signer.GenerateKey() // a fresh controller: Run enrolls it beside any earlier one
	agentAddr, err := Run(ctx, Options{Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e-replay",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"),
		AgentBinary:  func(arch string) (string, error) { return e2eEnv(t, "JUMPGATE_E2E_AGENTS") + "/jumpgate-linux-" + arch, nil }})
	if err != nil {
		t.Fatal(err)
	}

	tunnel, err := executor.DialSSH(ctx, executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "jumpgate",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()})
	if err != nil {
		t.Fatal(err)
	}
	defer tunnel.Close()
	hc := &http.Client{Transport: &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
		return tunnel.Dial("unix", agentclient.DefaultSocket)
	}}}
	post := func(env intent.Envelope) intent.Rejection {
		t.Helper()
		b, _ := json.Marshal(env)
		res, err := hc.Post("http://agent/v1/intent", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out intent.ReceiptEnvelope
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		var rej intent.Rejection
		if out.Receipt.Status == intent.StatusRejected {
			_ = json.Unmarshal(out.Result, &rej)
		}
		return rej
	}
	sign := func(i intent.Intent) intent.Envelope {
		sig, _ := controller.SignTypedData(ctx, i.TypedData())
		return intent.Envelope{Intent: i.JSON(), Payload: []byte("{}"), Sigs: []string{sig.Hex()}}
	}

	now := uint64(time.Now().Unix())
	n1, _ := intent.NewNonce()
	first := intent.Intent{Agent: agentAddr, Controller: controller.Address(), Seq: 1, Nonce: n1,
		IssuedAt: now, Expiry: now + 120, Kind: intent.KindAgentInfo, PayloadHash: intent.Hash([]byte("{}"))}
	captured := sign(first)
	if rej := post(captured); rej.Code != "" {
		t.Fatalf("first send refused: %+v", rej)
	}
	if rej := post(captured); rej.Code != intent.ReasonStaleSeq {
		t.Fatalf("replay: %+v, want stale_seq", rej)
	}

	other, _ := signer.GenerateKey()
	n2, _ := intent.NewNonce()
	misaddressed := first
	misaddressed.Seq, misaddressed.Nonce, misaddressed.Agent = 2, n2, other.Address()
	if rej := post(sign(misaddressed)); rej.Code != intent.ReasonWrongAgent {
		t.Fatalf("misaddressed: %+v, want wrong_agent", rej)
	}
}

// Task 4's guarantee against a real sshd: a cancelled command dies on the box.
func TestE2ECancelKillsTheRemoteCommand(t *testing.T) {
	port, _ := strconv.Atoi(e2eEnv(t, "JUMPGATE_E2E_PORT"))
	ex, err := executor.NewSSHContext(context.Background(), executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()})
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()
	ctx, cancel := context.WithCancel(context.Background())
	go ex.Run(ctx, "sleep 4242", nil)
	time.Sleep(time.Second)
	cancel()
	time.Sleep(7 * time.Second)
	r, _ := ex.Run(context.Background(), "pgrep -f 'sleep 4242' || true", nil)
	if strings.TrimSpace(r.Stdout) != "" {
		t.Fatalf("remote command survived cancellation: pids %s", r.Stdout)
	}
}

```

Imports for this file: `bytes`, `context`, `encoding/json`, `net`, `net/http`, `os`, `strconv`, `strings`, `testing`, `time`, `golang.org/x/crypto/ssh`, and the `agentclient`, `executor`, `intent`, `signer` packages. Every test runs against the same container, so they may run in any order: each pairs (idempotently) before it starts.

- [ ] **Step 4: Write the runner script**

```bash
#!/usr/bin/env bash
# scripts/e2e-agent.sh — pair a real systemd+sshd container and run the e2e
# tests against it. Needs Docker. BASE=ubuntu:24.04 scripts/e2e-agent.sh for
# the second distro.
set -euo pipefail
base="${BASE:-debian:12}"
work="$(mktemp -d)"
trap 'docker rm -f jumpgate-e2e >/dev/null 2>&1 || true; rm -rf "$work"' EXIT

scripts/build-agents.sh "$work/agents"
ssh-keygen -q -t ed25519 -N '' -f "$work/root"
ssh-keygen -q -t ed25519 -N '' -C jumpgate-controller -f "$work/transport"

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
docker run -d --name jumpgate-e2e --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw -p 127.0.0.1::22 jumpgate-e2e >/dev/null
for _ in $(seq 1 30); do docker exec jumpgate-e2e systemctl is-active ssh >/dev/null 2>&1 && break; sleep 1; done
docker exec -i jumpgate-e2e sh -c 'cat >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys' < "$work/root.pub"
port="$(docker port jumpgate-e2e 22/tcp | head -1 | sed 's/.*://')"

JUMPGATE_E2E_PORT="$port" \
JUMPGATE_E2E_ROOT_KEY="$work/root" \
JUMPGATE_E2E_TRANSPORT_KEY="$work/transport" \
JUMPGATE_E2E_TRANSPORT_PUB="$(cat "$work/transport.pub")" \
JUMPGATE_E2E_AGENTS="$work/agents" \
  go test -tags e2e -count=1 -v ./internal/bootstrap/ -run E2E
```

- [ ] **Step 5: Run it on both distributions**

Run: `chmod +x scripts/e2e-agent.sh && scripts/e2e-agent.sh && BASE=ubuntu:24.04 scripts/e2e-agent.sh`
Expected: all `TestE2E*` PASS on both. On Apple Silicon, the container is arm64 and the arm64 agent is uploaded; on x86_64, amd64. If Docker is unavailable, stop and report it — do not mark this task done without one passing run.

- [ ] **Step 6: Commit**

```bash
git add scripts internal/bootstrap/e2e_test.go
git commit -m "test(bootstrap): pair a real systemd box end to end

scripts/build-agents.sh cross-builds the cgo-free Linux agents with
SHA256SUMS; scripts/e2e-agent.sh pairs a Debian 12 or Ubuntu 24.04 container
and checks the signed round trip over the tunnel user, that the tunnel user
gets no shell, that re-pairing keeps the agent's identity, that replays and
misaddressed intents are refused, and that a cancelled command dies on the box.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 19: Spec and docs reconcile, full verification

**Files:**
- Modify: `docs/superpowers/specs/2026-10-02-foundations-design.md`, `docs/superpowers/specs/2026-10-02-controller-agent-architecture-design.md`, `README.md`

- [ ] **Step 1: Bring the specs in line with what was built**

In the foundations spec:
- "Step 0" and the parent's "Naming": on-box names (units, `/var/lib/valve-node-app`, containers, `valve-node-app` user, `VALVE_*`) are migrated by a **sub-project 2 durable-job intent** (`node.migrate-names`), not by pairing; pairing in this sub-project leaves them untouched.
- Shared-code change 6: the deciders are `executor.Strict` (jumpgate's file plus OpenSSH known_hosts, never records) and `executor.CaptureHostKey` + `RecordHostKey` (the CLI's confirm flow), with TOFU unchanged as the default.
- Components table: `executor.Sudo` lives in `internal/executor/sudo.go`; add `internal/filelock`; `jumpgate stop` uses `POST /api/shutdown`, not a signal; status head lag adds `refHead` to the API reply.
- Bootstrap: authorized_keys lives at `/var/lib/jumpgate/home/.ssh/authorized_keys`; enrollment and identity are `jumpgate agent enroll` / `agent init` on the box; local pairing enrolls the controller's uid in `policy.json` `localUids` instead of adding it to the group (no re-login needed).

In the parent spec's build order, sub-project 2's row gains "on-box rename migration (`node.migrate-names`)".

- [ ] **Step 2: README**

Add a short "Command line" section after v0.3 listing `jumpgate keys init`, `jumpgate hosts add`, `jumpgate status|disk|endpoints|firewall|logs|service`, `jumpgate serve` / `stop`, one sentence on what pairing installs, and the recommendation to disable root SSH afterwards.

- [ ] **Step 3: Full verification**

Run:
```bash
gofmt -l . ; go vet ./... ; go test -race ./... ; \
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/jumpgate ; \
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /dev/null ./cmd/jumpgate ; \
GOOS=windows go build -o /dev/null ./cmd/jumpgate ; \
scripts/e2e-agent.sh
```
Expected: no gofmt output; vet clean; every test passes; all three cross builds succeed; e2e passes. Then walk the spec's "Done when" list and confirm each item against a command you ran, quoting its output in the final report.

- [ ] **Step 4: Commit**

```bash
git add docs README.md
git commit -m "docs: reconcile the foundations spec with the build, document the CLI

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
