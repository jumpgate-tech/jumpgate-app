# Platform Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the jumpgate controller (tray/web app, `jumpgate` CLI, local server) fully supported and CI-tested on macOS, Windows and Linux, with a double-clickable terminal launcher on each OS, while the agent stays Linux-only.

**Architecture:** Portable test helpers and a three-OS CI matrix come first, so every later change is proven on Windows. Then owner-only file protection (`internal/fsperm`, DACLs on Windows), agent binaries embedded in release builds (`internal/agentbin`), a Windows Credential Manager key store, root/foreground local pairing, Windows daemon fixes, Windows SSH-agent pipes, one-time login codes for the browser, and a terminal home screen (`runTerminalHome`, the seam the TUI later replaces). The last tasks package each OS (Windows console + GUI exes with a notification icon, a Linux tarball with `.desktop` entries on WebKitGTK 4.1, a macOS terminal app) and wire the release workflow.

**Tech Stack:** Go 1.25 (module; local toolchain 1.26), `golang.org/x/sys/windows` (already a dependency, including its `registry` subpackage), `github.com/webview/webview_go` (tray builds), GitHub Actions (ubuntu-24.04, ubuntu-24.04-arm, macos-latest/macos-14/macos-15-intel, windows-latest), goreleaser v2, Docker on colima for Linux checks.

**Spec:** `docs/superpowers/specs/2026-10-05-platform-support-design.md` (parent: `docs/superpowers/specs/2026-10-02-controller-agent-architecture-design.md`). Read the spec before Task 1; its "Decisions" table (D1–D28) is binding.

## Preconditions

- Branch: `feat/platform-support` from `main` (at or after `0bfebcc`).
- Baseline: `go test ./...` green on macOS before Task 1. Record the result.
- **Windows verification needs GitHub Actions** (spec D22). Before Task 1 Step 11, ask the user once for permission to push `feat/platform-support` and open a draft PR so CI runs; every later task's Windows check reuses that PR (`gh pr checks --watch`, or `gh run watch`). Never push without that permission.
- Docker via colima is running for the Linux checks (`colima status`). The Linux test runner is `scripts/test-linux.sh`, added in Task 1.

## Global Constraints

- No new Go module dependency. `golang.org/x/sys/windows` (and `golang.org/x/sys/windows/registry`) cover every Windows API; tools run with `go run …@version` in CI are not module dependencies.
- The agent stays Linux-only. Agent binaries are `CGO_ENABLED=0`, `linux/amd64` and `linux/arm64`, built only by `scripts/build-agents.sh`.
- Build tag for embedded agents: `embedagents`. Embedded files: `internal/agentbin/embedded/jumpgate-linux-{amd64,arm64}.gz` and `internal/agentbin/embedded/SHA256SUMS` (sums of the uncompressed binaries). The directory is gitignored except `.gitkeep`.
- Version string: the git tag, `v`-prefixed (`v0.9.0`), injected with `-X github.com/valve-tech/jumpgate/internal/buildinfo.version=<tag>` into controller and agents alike (D15).
- Owner-only means: unix `0700` dirs / `0600` files; Windows protected DACL with the current user and SYSTEM, `GENERIC_ALL` (D2, D16).
- Login codes: 128-bit random hex, 60 s lifetime, single use; route `GET /login?code=…`; mint route `POST /api/login-code` (D4).
- Key-store tool timeout 2 minutes; Secret Service probe 3 seconds (D17).
- Error codes added: `local_unsupported`, `local_needs_terminal`.
- Windows minimum: Windows 10 1803 / Server 2019 (AF_UNIX).
- Release artifact names: `jumpgate_<os>_<arch>.{tar.gz,zip}` (goreleaser, unchanged), `jumpgate-linux-{amd64,arm64}` + `agents-SHA256SUMS`, `Jumpgate-macos-{arm64,amd64}.zip`, `jumpgate-windows-amd64.zip`, `jumpgate-linux-{amd64,arm64}.tar.gz`, `checksums.txt`.
- Windows exes: `jumpgate.exe` (console subsystem, `CGO_ENABLED=0`) and `jumpgate-tray.exe` (`-tags tray`, `-H windowsgui`).
- Environment variable set by launchers: `JUMPGATE_LAUNCHER=1`.
- The terminal home's entry function is exactly `func runTerminalHome(ctx context.Context, in io.Reader, out io.Writer) int` (the TUI seam).
- Every commit message is conventional (`feat(…)`, `fix(…)`, `test(…)`, `ci: …`, `build: …`, `docs: …`) and ends with the line `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Comments follow the codebase's style: full sentences explaining *why*, at the density of the surrounding code. User-visible strings say "jumpgate" and `~/.jumpgate`.

## Review Focus

1. **An existing `~/.jumpgate` made by an older release, with a loose key file or a loose directory** (Windows profile with inherited ACLs, or a unix `0755`). Expected: the first start tightens the directories it owns, and `LoadKeyFile` refuses a loose key on every OS with a message saying how to fix it, rather than signing with a key others can read. Tests: Task 2 `TestMkdirPrivateTightensAnExistingDir`, `TestLoadKeyFileRefusesASharedKey`.
2. **A login link opened twice** (a browser prefetch, a refresh, a pasted link). Expected: the first visit signs in; the second gets a 401 page naming `jumpgate open`, and the cookie from the first stays valid. Test: Task 8 `TestLoginCodeIsSingleUse`.
3. **A non-root `jumpgate hosts add NAME --local` from a non-terminal** (cron, CI, a pipe). Expected: it fails at once with the fix named, before starting the daemon or sudo, and never hangs on a password prompt. Test: Task 5 `TestHostsAddLocalNonRootNeedsATerminal`.
4. **A stale `~/.jumpgate/agents` on a release install.** Expected: it is used (D14), and the pairing stream names the source, so a person can see why an old agent was installed. Test: Task 3 `TestPathPrefersTheDevDirAndNamesIt` plus the pair-stream line in Task 3 Step 9.
5. **The terminal home launched with stdin already at end of input** (a script, a launcher whose terminal closed). Expected: it prints the overview once and returns 0; it never spins on EOF. Test: Task 9 `TestTerminalHomeEndsOnEOF`.

## File Structure

Created:

| Path | Responsibility |
|---|---|
| `internal/testutil/testutil.go`, `loosen_unix.go`, `loosen_windows.go`, `testutil_test.go` | Portable test helpers: short temp dirs, isolated HOME, shell/unix guards, owner-only assertion, loosening a file |
| `scripts/test-linux.sh` | Run `go test` in a Linux container as a non-root user |
| `.gitattributes` | LF line endings in checkouts on every OS |
| `internal/fsperm/fsperm.go`, `fsperm_unix.go`, `fsperm_windows.go`, `fsperm_test.go`, `fsperm_windows_test.go` | Owner-only files and dirs; checks; Windows rename retry |
| `internal/agentbin/agentbin.go`, `embed_on.go`, `embed_off.go`, `agentbin_test.go`, `embed_on_test.go`, `embedded/.gitkeep` | Find, extract and verify the Linux agent for an arch |
| `internal/buildinfo/cgo_on.go`, `cgo_off.go`, `static_test.go` | `SelfIsStaticLinux()` |
| `internal/signer/wincred.go`, `wincred_windows.go`, `wincred_other.go`, `wincred_test.go`, `wincred_windows_test.go` | Windows Credential Manager store |
| `internal/signer/secretservice.go`, `secretservice_test.go` | D-Bus session detection, Secret Service probe |
| `scripts/test-secret-service.sh` | Real Secret Service test in a container |
| `internal/bootstrap/local.go`, `local_test.go` | `LocalSupported(goos)` shared by CLI and server |
| `cmd/jumpgate/pair_local.go`, `pair_local_test.go` | Foreground non-root local pairing |
| `scripts/e2e-local.sh` | Container e2e for local pairing |
| `internal/daemon/start_unix.go`, `start_windows.go`, `start_test.go`, `start_windows_test.go` | Detached start with job breakaway; working dir |
| `scripts/windows-smoke.ps1` | Windows daemon lifecycle smoke |
| `internal/executor/agentsock_unix.go`, `agentsock_windows.go`, `agentsock_test.go`, `agentsock_windows_test.go`, `knownhosts_files.go`, `knownhosts_files_test.go` | SSH agent dialling per OS; OpenSSH known_hosts list |
| `cmd/jumpgate/paths.go`, `paths_test.go` | `~` expansion |
| `internal/server/login.go`, `login_test.go` | Login codes |
| `cmd/jumpgate/opener.go`, `opener_test.go`, `opener_linux_test.go`, `cli_open.go` | Browser opener; `jumpgate open` |
| `cmd/jumpgate/home_terminal.go`, `home_terminal_test.go`, `help.go`, `console_windows.go`, `console_other.go`, `console_test.go` | Terminal home, command help, console detection |
| `cmd/jumpgate/gui.go`, `gui_windows.go`, `gui_other.go`, `gui_test.go` | GUI launch detection, GUI logging, fatal dialog, WebView2 check |
| `cmd/jumpgate/statusitem_windows.go`, `statusitem_windows_test.go`, `trayicon_windows.png` | Windows notification-area icon |
| `scripts/package-windows.sh`, `packaging/windows/README.txt`, `scripts/windows-desktop-smoke.ps1` | Windows bundle and smoke |
| `scripts/linux/pkgconfig/webkit2gtk-4.0.pc` | WebKitGTK 4.1 shim |
| `packaging/linux/jumpgate.desktop`, `jumpgate-window.desktop`, `install.sh`, `README.txt`, `scripts/package-linux.sh`, `scripts/test-linux-desktop.sh` | Linux bundle and its test |
| `packaging/macos/jumpgate-terminal`, `packaging/macos/jumpgate.command`, `scripts/test-macos-bundle.sh` | macOS terminal launcher and its test |

Modified: `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.goreleaser.yaml`, `.gitignore`, `README.md`, `scripts/build-agents.sh`, `scripts/e2e-agent.sh`, `cmd/jumpgate/{main.go,cli.go,cli_hosts.go,cli_keys.go,cli_agent.go,appinstance.go,tray.go,statusitem_other.go,statusitem_darwin.go,trayhealth.go,build-macos-app.sh}`, `internal/config/config.go`, `internal/daemon/{daemon.go,detach_unix.go→removed,detach_windows.go→removed}`, `internal/server/{server.go,controller.go,pair.go,api.go,vpn.go,vpnserver.go,gateways.go}`, `internal/signer/{store.go,keychain.go,keyfile.go,onepassword.go}`, `internal/executor/{dial.go,hostkey.go}`, `internal/setup/trust.go`, and the test files listed in Task 1.

---

### Task 1: Portable test suite and the three-OS CI matrix

Covers B-5, the CI half of D7. Nothing later is verifiable on Windows without this.

**Files:**
- Create: `internal/testutil/testutil.go`, `internal/testutil/testutil_test.go`, `scripts/test-linux.sh`, `.gitattributes`
- Modify: `.github/workflows/ci.yml`
- Modify (tests): `cmd/jumpgate/appinstance_test.go`, `cmd/jumpgate/cli_test.go`, `internal/server/pair_test.go`, `internal/server/intent_test.go`, `internal/server/unix_test.go`, `internal/agent/listen_test.go`, `internal/daemon/daemon_test.go`, `internal/relay/billing_test.go`, `internal/executor/dial_test.go`, `internal/agentclient/client_test.go`, `internal/config/{config,dir,update,persistence}_test.go`, `internal/server/{api,actions,gateways_installid}_test.go`, `internal/executor/{ssh,local,procgroup,sudo,remotepath,gaps,hostkey}_test.go`, `internal/catalog/snapshot_test.go`, `internal/setup/snapshot_test.go`, `internal/ops/{diagnose_gaps,lifecycle,docker}_test.go`, `internal/monitor/disk_test.go`, `internal/vpn/server_test.go`, `internal/signer/store_test.go`

**Interfaces:**
- Produces (package `github.com/valve-tech/jumpgate/internal/testutil`):
  - `func ShortTempDir(t testing.TB) string`
  - `func Home(t testing.TB) string`
  - `func RequirePOSIXShell(t testing.TB)`
  - `func RequireUnix(t testing.TB)`
  - `func AssertPrivate(t testing.TB, path string)` (Task 2 extends it to Windows)
- Produces: `scripts/test-linux.sh [go test args…]` (default `./...`; `AS_ROOT=1` runs as root).

- [ ] **Step 1: Write the failing tests for the helpers**

```go
// internal/testutil/testutil_test.go
package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// A unix socket path must fit in sun_path: 104 bytes on macOS, 108 on Linux
// and Windows. The run dir's socket is <dir>/.jumpgate/run/server.sock, so the
// base dir has to leave room for about 30 more bytes.
func TestShortTempDirLeavesRoomForASocket(t *testing.T) {
	d := ShortTempDir(t)
	if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
		t.Fatalf("ShortTempDir = %q, not a directory: %v", d, err)
	}
	sock := filepath.Join(d, ".jumpgate", "run", "server.sock")
	if len(sock) > 100 {
		t.Fatalf("socket path %q is %d bytes; want at most 100", sock, len(sock))
	}
}

func TestHomeSetsHomeAndUserProfile(t *testing.T) {
	h := Home(t)
	if os.Getenv("HOME") != h || os.Getenv("USERPROFILE") != h {
		t.Fatalf("HOME=%q USERPROFILE=%q, want both %q", os.Getenv("HOME"), os.Getenv("USERPROFILE"), h)
	}
	got, err := os.UserHomeDir()
	if err != nil || got != h {
		t.Fatalf("os.UserHomeDir() = %q, %v; want %q", got, err, h)
	}
}

func TestAssertPrivateAcceptsAnOwnerOnlyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	AssertPrivate(t, p)
}

func TestRequirePOSIXShellSkipsOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only meaningful on Windows")
	}
	ran := t.Run("inner", func(t *testing.T) {
		RequirePOSIXShell(t)
		t.Fatal("RequirePOSIXShell did not skip on Windows")
	})
	if !ran {
		t.Fatal("inner test failed instead of skipping")
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/testutil/`
Expected: FAIL — `undefined: ShortTempDir` (and the other helpers).

- [ ] **Step 3: Implement the helpers**

```go
// internal/testutil/testutil.go

// Package testutil holds helpers that keep tests portable across macOS, Linux
// and Windows. Only test files import it.
package testutil

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// ShortTempDir returns a fresh directory, removed when the test ends, whose
// path is short enough to hold unix sockets a few levels down. t.TempDir()
// embeds the test name under TMPDIR, which on macOS (/var/folders/…) already
// eats most of sun_path's 104 bytes, so macOS uses /tmp. Linux's TMPDIR is
// /tmp, and Windows' %TEMP% is short enough without the test name.
func ShortTempDir(t testing.TB) string {
	t.Helper()
	base := os.TempDir()
	if runtime.GOOS == "darwin" {
		base = "/tmp"
	}
	d, err := os.MkdirTemp(base, "jg")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

// Home points both HOME and USERPROFILE at a fresh short directory for the
// rest of the test. os.UserHomeDir reads USERPROFILE on Windows and HOME
// elsewhere; setting only HOME on Windows silently shares one home between
// every test in the package.
func Home(t testing.TB) string {
	t.Helper()
	h := ShortTempDir(t)
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	return h
}

// RequirePOSIXShell skips a test that executes `sh`. The controller never
// shells out locally on Windows (local mode is refused there), so a test that
// needs sh tests code Windows does not run.
func RequirePOSIXShell(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell; local commands never run on a Windows controller")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH")
	}
}

// RequireUnix skips a test of unix-only behaviour: permission bits on
// non-secret files, FIFOs, peer credentials, the agent itself.
func RequireUnix(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("unix-only behaviour")
	}
}

// AssertPrivate fails the test unless only the file's owner can read or
// change path. On unix that is the mode bits; Windows is checked through its
// DACL once internal/fsperm exists (Task 2 of the platform plan).
func AssertPrivate(t testing.TB, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Logf("AssertPrivate(%s): DACL check not implemented yet", path)
		return
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := fi.Mode().Perm()
	if perm&0o077 != 0 || perm&0o600 != 0o600 {
		t.Fatalf("%s has mode %o; want owner read/write and nothing for group or others", path, perm)
	}
}
```

- [ ] **Step 4: Run the helper tests**

Run: `go test ./internal/testutil/`
Expected: PASS (the Windows-only test skips on macOS).

- [ ] **Step 5: Add the Linux test runner and LF checkouts**

```bash
#!/usr/bin/env bash
# scripts/test-linux.sh — run Go tests in a Linux container (Docker via colima
# on macOS) as an unprivileged user, the way CI's ubuntu runner does. Root
# would pass permission tests that a normal user fails.
#
#   scripts/test-linux.sh                         # go test ./...
#   scripts/test-linux.sh ./internal/fsperm/ -v   # any go test arguments
#   AS_ROOT=1 scripts/test-linux.sh ./...         # as root (pairing as root)
#   CGO_ENABLED=0 scripts/test-linux.sh ./internal/buildinfo/   # passed through
set -euo pipefail
image="${GO_IMAGE:-golang:1.25}"
[ $# -eq 0 ] && set -- ./...
docker volume create jumpgate-gocache >/dev/null
drop='exec setpriv --reuid=1000 --regid=1000 --clear-groups "$@"'
[ "${AS_ROOT:-0}" = 1 ] && drop='exec "$@"'
envs=()
[ -n "${CGO_ENABLED:-}" ] && envs+=(-e "CGO_ENABLED=$CGO_ENABLED")
docker run --rm \
  -v "$PWD":/src -w /src \
  -v jumpgate-gocache:/cache \
  -e HOME=/tmp/home -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod -e GOFLAGS=-buildvcs=false \
  ${envs[@]+"${envs[@]}"} \
  "$image" sh -c 'mkdir -p /tmp/home /cache && chown -R 1000:1000 /tmp/home /cache && '"$drop" sh go test "$@"
```

```gitattributes
# .gitattributes — LF everywhere, so Windows checkouts (core.autocrlf=true on
# the runners) compare byte-for-byte with golden strings and testdata.
* text=auto eol=lf
*.png binary
*.ico binary
*.gz binary
```

Run: `chmod +x scripts/test-linux.sh && scripts/test-linux.sh ./internal/testutil/`
Expected: PASS inside the container.

- [ ] **Step 6: Convert the short-temp-dir and HOME sites**

Apply these exact edits (import `"github.com/valve-tech/jumpgate/internal/testutil"` in each file; drop now-unused imports):

| File | Change |
|---|---|
| `cmd/jumpgate/appinstance_test.go` `shortHome` | body becomes `t.Helper(); testutil.Home(t)` |
| `cmd/jumpgate/cli_test.go` `TestAgentEnrollLocalUIDOpensTheSocket` | first line `testutil.RequireUnix(t)`; `d, err := os.MkdirTemp("/tmp", "jgs")` + its error check + cleanup become `d := testutil.ShortTempDir(t)`; at lines ~128, ~237, ~243 leave the mode asserts (the test is unix-only now) |
| `cmd/jumpgate/cli_test.go` `TestAgentInitKeepsItsIdentity` (mode assert ~line 128) | add `testutil.RequireUnix(t)` as its first line: `agent init` is on-box agent code, which Task 5 refuses on Windows; the `0o700` assert stays |
| `internal/server/pair_test.go` `shortHome` | body becomes `t.Helper(); return testutil.ShortTempDir(t)`; every `t.Setenv("HOME", shortHome(t))` (in `pairServer` and elsewhere) becomes `testutil.Home(t)`; the assert at ~204 becomes `testutil.AssertPrivate(t, transportKeyPath())`; at ~325 `testutil.AssertPrivate(t, filepath.Dir(transportKeyPath()))` |
| `internal/server/intent_test.go` `pairedLocal` | first line `testutil.RequireUnix(t)` (it starts a real agent, which needs peer credentials); `home, _ := os.MkdirTemp(…)` + cleanup + `t.Setenv("HOME", home)` become `home := testutil.Home(t)` |
| `internal/server/unix_test.go` (3 sites) | `dir, _ := os.MkdirTemp("/tmp", "jgs"); defer os.RemoveAll(dir)` becomes `dir := testutil.ShortTempDir(t)`; the `0o600` assert becomes `testutil.AssertPrivate(t, sock)` |
| `internal/agent/listen_test.go` | add `//go:build linux \|\| darwin` as line 1 (the agent is Linux-only; macOS runs it for development); `shortDir` body becomes `t.Helper(); return testutil.ShortTempDir(t)` |
| `internal/daemon/daemon_test.go` `isolate` | body becomes `t.Helper(); testutil.Home(t)`; the `server.json` mode assert (~95) becomes `testutil.AssertPrivate(t, filepath.Join(dir, "server.json"))` using the path variable already in that test |
| `internal/relay/billing_test.go` | `dir, err := os.MkdirTemp("/tmp", "jgr")` + check + cleanup become `dir := testutil.ShortTempDir(t)` |
| `internal/executor/dial_test.go` `shortTempDir` | body becomes `t.Helper(); return testutil.ShortTempDir(t)` |
| `internal/agentclient/client_test.go` `startAgent` | first line `testutil.RequireUnix(t)` (real agent); its `MkdirTemp` + cleanup become `dir := testutil.ShortTempDir(t)` |
| `internal/agentclient/client_test.go` `fakeAgent` | `dir, err := os.MkdirTemp("/tmp", "jgf"); must(t, err); t.Cleanup(…)` becomes `dir := testutil.ShortTempDir(t)` (no RequireUnix: AF_UNIX works on Windows) |
| `internal/config/{config,dir,update,persistence}_test.go`, `internal/server/{api,actions,gateways_installid}_test.go` | every `home := t.TempDir(); t.Setenv("HOME", home)` (or `t.Setenv("HOME", dir)`) becomes `home := testutil.Home(t)` (keep the variable name the test uses); every `0o600`/`0o700` assert on a config or key file becomes `testutil.AssertPrivate(t, path)` |

Before (typical HOME site):

```go
	home := t.TempDir()
	t.Setenv("HOME", home)
```

After:

```go
	home := testutil.Home(t)
```

- [ ] **Step 7: Guard tests that need `sh` or unix-only semantics**

| File | Change |
|---|---|
| `internal/executor/local_test.go`, `internal/executor/procgroup_test.go`, `internal/executor/sudo_test.go`, `internal/executor/remotepath_test.go` | add `//go:build unix` as line 1: they drive the local executor or `sh` directly, which a Windows controller never does (`executor.LocalAvailable` refuses it) |
| `internal/executor/ssh_test.go`, `internal/executor/gaps_test.go`, `internal/executor/hostkey_test.go` | in each test whose gliderlabs handler runs `exec.Command("sh", …)` or which asserts a `0o640` mode, add `testutil.RequirePOSIXShell(t)` (handler runs sh) or `testutil.RequireUnix(t)` (mode) as its first line; `0o600` asserts on the host-key file become `testutil.AssertPrivate(t, path)` |
| `internal/catalog/snapshot_test.go`, `internal/setup/snapshot_test.go`, `internal/ops/{diagnose_gaps,lifecycle,docker}_test.go`, `internal/monitor/disk_test.go`, `internal/vpn/server_test.go` | in each test that execs `sh` (search the file for `"sh"`), add `testutil.RequirePOSIXShell(t)` as its first line |
| `internal/signer/store_test.go` (~line 545 runs `sh`) | that test gets `testutil.RequirePOSIXShell(t)`; the template-mode assertion on `f.modes` runs only when `runtime.GOOS != "windows"` (a mode means nothing there; Task 2 restricts the template with `fsperm.MakePrivate`, which `TestLoadKeyFileRefusesASharedKey`-style DACL checks cover) |

- [ ] **Step 8: Run the whole suite on macOS and in a Linux container**

Run: `go test ./... && scripts/test-linux.sh`
Expected: PASS on both. If a converted test fails, it is a conversion mistake (wrong variable kept, missing import), not a new behaviour: fix it in place.

- [ ] **Step 9: Cross-compile every package's tests for Windows locally**

Run: `GOOS=windows go vet ./... && GOOS=windows go test -c -o "$(mktemp -d)/" ./...`
Expected: no output from vet; the compile succeeds for every package.

- [ ] **Step 10: Replace `.github/workflows/ci.yml`**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: read

jobs:
  # The embedded web UI must be committed and reproducible. It is checked once,
  # on Linux; the Go jobs below use the committed dist.
  web:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "22"
          cache: npm
          cache-dependency-path: cmd/jumpgate/web/package-lock.json
      - name: Build web UI
        run: |
          cd cmd/jumpgate/web
          npm ci
          npm run build
      - name: Verify web/dist is committed and reproducible
        run: git diff --exit-code cmd/jumpgate/web/dist

  # The controller is supported on all three desktop OSes, so its tests run on
  # all three. bash on Windows is Git Bash, present on every hosted image.
  go:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-24.04, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    defaults:
      run:
        shell: bash
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: go vet
        run: go vet ./...
      - name: go test
        run: go test ./...
      - name: go build (headless, no cgo)
        env:
          CGO_ENABLED: "0"
        run: go build -o "$RUNNER_TEMP/jumpgate-ci" ./cmd/jumpgate

  # Cheap guard for targets no runner tests natively, including the linux/arm64
  # build the agent ships as.
  cross:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: vet, compile tests and build for every target
        run: |
          set -euo pipefail
          for target in linux/amd64 linux/arm64 darwin/arm64 darwin/amd64 windows/amd64; do
            echo "== $target"
            export GOOS="${target%/*}" GOARCH="${target#*/}" CGO_ENABLED=0
            go vet ./...
            go test -c -o "$(mktemp -d)/" ./...
            go build ./...
          done
      - name: build the linux/arm64 controller and agent
        env:
          GOOS: linux
          GOARCH: arm64
          CGO_ENABLED: "0"
        run: go build -o /dev/null ./cmd/jumpgate

  # The Rust billing service. It is a standalone crate under services/billing,
  # invisible to the Go tooling above, so it needs its own gate.
  billing:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: services/billing
    steps:
      - uses: actions/checkout@v4
      - name: Add rustfmt and clippy
        run: rustup component add rustfmt clippy
      - name: cargo fmt
        run: cargo fmt --all --check
      - name: cargo clippy
        run: cargo clippy --all-targets --locked -- -D warnings
      - name: cargo test
        run: cargo test --locked
```

- [ ] **Step 11: Commit, push (with the user's permission, see Preconditions) and watch Windows**

```bash
git add internal/testutil scripts/test-linux.sh .gitattributes .github/workflows/ci.yml cmd internal
git commit -m "test: make the suite portable and run it on macOS, Linux and Windows in CI

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push -u origin feat/platform-support
gh pr create --draft --title "Platform support" --body "Implements docs/superpowers/plans/2026-10-05-platform-support.md

🤖 Generated with [Claude Code](https://claude.com/claude-code)"
gh pr checks --watch
```

Expected: `web`, `cross`, `billing`, and the ubuntu and macOS `go` legs pass. The Windows `go` leg is the acceptance gate for this task.

- [ ] **Step 12: Fix every remaining Windows failure by its category, until the Windows leg is green**

Read the failures with `gh run view --log-failed`. Each falls in one of these categories; apply the matching fix, never a blanket skip of a whole package:

| Symptom in the log | Fix |
|---|---|
| expected `a/b`, got `a\b` | build the expected value with `filepath.Join`, or compare after `filepath.ToSlash` when the code under test emits POSIX paths for a box |
| `\r\n` in a compared string | the file is test input checked out before `.gitattributes`: `git add --renormalize .`; for runtime output, the code must emit `\n` (fix the code) |
| `exec: "sh": executable file not found` or a `/bin/…` path | `testutil.RequirePOSIXShell(t)` in that test |
| mode `666`/`444` where `600`/`640` expected | secret file: `testutil.AssertPrivate`; non-secret file: `testutil.RequireUnix(t)` |
| `The process cannot access the file because it is being used by another process` during cleanup | close the file, listener or server before the test returns (`t.Cleanup(func(){ ln.Close() })`, cancel the serve context and wait for it) |
| `bind: invalid argument` / path too long on a socket | use `testutil.ShortTempDir(t)` |
| the test reads another test's state under one shared HOME | `testutil.Home(t)` |
| `peer credentials unsupported` | the test starts a real agent: `testutil.RequireUnix(t)` |

Run after each fix: `go test ./<pkg>/ && GOOS=windows go vet ./<pkg>/`, then commit and push:

```bash
git commit -am "test: <package>: <what was made portable>

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected at the end: every CI job green, including `go (windows-latest)`.

---

### Task 2: `internal/fsperm` — owner-only files on every OS

Covers I-1, the socket part of I-2, M-4, M-5, M-6, M-14, and D2.

**Files:**
- Create: `internal/fsperm/fsperm.go`, `internal/fsperm/fsperm_unix.go`, `internal/fsperm/fsperm_windows.go`, `internal/fsperm/fsperm_test.go`, `internal/fsperm/fsperm_windows_test.go`, `internal/testutil/loosen_unix.go`, `internal/testutil/loosen_windows.go`
- Modify: `internal/testutil/testutil.go` (`AssertPrivate` on Windows), `internal/config/config.go` (`lockPath`, `Save`), `internal/daemon/daemon.go` (`RunDir`, `Publish`, `EnsureRunning`'s log), `internal/server/controller.go` (`ensureTransportKey`), `internal/server/server.go` (`ServeUnix`), `internal/server/gateways.go:2095-2112` (install id), `internal/executor/hostkey.go:74-95` (`RecordHostKey`), `internal/signer/keyfile.go`, `internal/signer/onepassword.go:60-70`
- Test: `internal/fsperm/*_test.go`, `internal/signer/keyfile_test.go` (new), `internal/server/unix_test.go`

**Interfaces:**
- Consumes: `testutil.ShortTempDir`, `testutil.AssertPrivate` (Task 1).
- Produces (package `github.com/valve-tech/jumpgate/internal/fsperm`):
  - `var ErrNotPrivate error`
  - `func MkdirPrivate(dir string) error`
  - `func MakePrivate(path string) error`
  - `func WriteFilePrivate(path string, data []byte) error`
  - `func CheckPrivate(path string) error`
  - `func CheckPrivateFile(f *os.File) error`
  - `func Rename(oldpath, newpath string) error`
- Produces: `func testutil.Loosen(t testing.TB, path string)`; `testutil.AssertPrivate` now checks the DACL on Windows.

- [ ] **Step 1: Write the failing cross-OS tests**

```go
// internal/fsperm/fsperm_test.go
package fsperm_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestMkdirPrivateCreatesAnOwnerOnlyDir(t *testing.T) {
	d := filepath.Join(t.TempDir(), "a", "b")
	if err := fsperm.MkdirPrivate(d); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivate(d); err != nil {
		t.Fatalf("CheckPrivate after MkdirPrivate: %v", err)
	}
	testutil.AssertPrivate(t, d)
}

// M-6: a ~/.jumpgate an older release (or a umask) left open must be
// tightened, not trusted because it already exists.
func TestMkdirPrivateTightensAnExistingDir(t *testing.T) {
	d := filepath.Join(t.TempDir(), "loose")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.Loosen(t, d)
	if err := fsperm.CheckPrivate(d); !errors.Is(err, fsperm.ErrNotPrivate) {
		t.Fatalf("CheckPrivate on a loosened dir = %v, want ErrNotPrivate", err)
	}
	if err := fsperm.MkdirPrivate(d); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivate(d); err != nil {
		t.Fatalf("still not private after MkdirPrivate: %v", err)
	}
}

func TestWriteFilePrivateWritesAnOwnerOnlyFileAtomically(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "server.json")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.WriteFilePrivate(p, []byte("new")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "new" {
		t.Fatalf("content = %q, %v; want new", b, err)
	}
	if err := fsperm.CheckPrivate(p); err != nil {
		t.Fatalf("CheckPrivate: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestCheckPrivateRefusesALoosenedFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	if err := fsperm.WriteFilePrivate(p, []byte("x")); err != nil {
		t.Fatal(err)
	}
	testutil.Loosen(t, p)
	err := fsperm.CheckPrivate(p)
	if !errors.Is(err, fsperm.ErrNotPrivate) {
		t.Fatalf("CheckPrivate = %v, want ErrNotPrivate", err)
	}
	f, ferr := os.Open(p)
	if ferr != nil {
		t.Fatal(ferr)
	}
	defer f.Close()
	if err := fsperm.CheckPrivateFile(f); !errors.Is(err, fsperm.ErrNotPrivate) {
		t.Fatalf("CheckPrivateFile = %v, want ErrNotPrivate", err)
	}
}

func TestMakePrivateFixesALoosenedFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Loosen(t, p)
	if err := fsperm.MakePrivate(p); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivate(p); err != nil {
		t.Fatalf("CheckPrivate after MakePrivate: %v", err)
	}
}

func TestRenameReplacesTheTarget(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	_ = os.WriteFile(a, []byte("a"), 0o600)
	_ = os.WriteFile(b, []byte("b"), 0o600)
	if err := fsperm.Rename(a, b); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(b); string(got) != "a" {
		t.Fatalf("b = %q, want a", got)
	}
	if runtime.GOOS == "windows" {
		t.Log("Rename retried path covered in fsperm_windows_test.go")
	}
}
```

```go
// internal/fsperm/fsperm_windows_test.go
//go:build windows

package fsperm

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// M-5: an indexer or antivirus holding the target open makes MoveFileEx fail
// with a sharing violation for a moment. Rename must wait it out.
func TestRenameRetriesASharingViolation(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	_ = os.WriteFile(a, []byte("a"), 0o600)
	_ = os.WriteFile(b, []byte("b"), 0o600)
	calls := 0
	old := moveFile
	moveFile = func(from, to string) error {
		calls++
		if calls < 3 {
			return &os.LinkError{Op: "rename", Old: from, New: to, Err: windows.ERROR_SHARING_VIOLATION}
		}
		return old(from, to)
	}
	t.Cleanup(func() { moveFile = old })
	if err := Rename(a, b); err != nil {
		t.Fatalf("Rename = %v after %d calls", err, calls)
	}
	if calls != 3 {
		t.Fatalf("moveFile called %d times, want 3", calls)
	}
}

// The server socket is an AF_UNIX reparse point; MakePrivate must set its
// DACL without following it (D2).
func TestMakePrivateOnAUnixSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "jg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "s.sock")
	ln, err := netListenUnix(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if err := MakePrivate(sock); err != nil {
		t.Fatalf("MakePrivate(socket): %v", err)
	}
	if err := CheckPrivate(sock); err != nil {
		t.Fatalf("CheckPrivate(socket): %v", err)
	}
}
```

Add to the same file the helper it uses (kept in the test so production code has no net import):

```go
func netListenUnix(path string) (interface{ Close() error }, error) {
	return net.Listen("unix", path)
}
```

and add `"net"` to its imports.

- [ ] **Step 2: Add `testutil.Loosen` and run the tests to verify they fail**

```go
// internal/testutil/loosen_unix.go
//go:build unix

package testutil

import (
	"os"
	"testing"
)

// Loosen makes path readable by other users, for tests that must see a
// refusal.
func Loosen(t testing.TB, path string) {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0o644)
	if fi.IsDir() {
		mode = 0o755
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
```

```go
// internal/testutil/loosen_windows.go
//go:build windows

package testutil

import (
	"testing"

	"golang.org/x/sys/windows"
)

// Loosen adds an allow ACE granting Everyone read access to path, on top of
// whatever DACL it has, the way a redirected or shared profile would.
func Loosen(t testing.TB, path string) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	old, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	everyone, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		t.Fatal(err)
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_READ,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
			TrusteeValue: windows.TrusteeValueFromSID(everyone),
		},
	}}, old)
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.UNPROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
}
```

Run: `go test ./internal/fsperm/`
Expected: FAIL — `undefined: fsperm.MkdirPrivate` (and the rest).

- [ ] **Step 3: Implement the portable API**

```go
// internal/fsperm/fsperm.go

// Package fsperm makes files and directories readable and writable only by
// the user who owns them, on every OS the controller runs on. On unix that is
// 0600/0700. On Windows a mode does nothing (os.Chmod only flips the
// read-only bit), so the same promise is kept with a protected DACL that
// grants the current user and SYSTEM, and nobody else.
//
// Everything the controller keeps secret goes through here: the controller
// key file, config.json (provider API keys, VPN keys), server.json and the
// server socket (the session token), the transport SSH key and the confirmed
// host keys.
package fsperm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotPrivate means someone other than the owner can read or change a file.
var ErrNotPrivate = errors.New("fsperm: other users can read or change this")

// MkdirPrivate creates dir and any missing parents, then restricts dir itself
// to its owner. An existing dir is tightened too: an older release or a loose
// umask may have left ~/.jumpgate open, and existing is not the same as safe.
// Parents are created with the default mode and left alone.
func MkdirPrivate(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return MakePrivate(dir)
}

// MakePrivate restricts an existing file, directory or socket to its owner.
// It never follows a symlink.
func MakePrivate(path string) error { return makePrivate(path) }

// CheckPrivate returns an error wrapping ErrNotPrivate, naming who has
// access, when anyone but the owner can read or change path.
func CheckPrivate(path string) error { return checkPrivate(path) }

// CheckPrivateFile is CheckPrivate on an open file, so the file checked is
// the file read.
func CheckPrivateFile(f *os.File) error { return checkPrivateFile(f) }

// WriteFilePrivate replaces path with data as an owner-only file. The data
// goes to a restricted temp file in the same directory, is synced, then
// renamed over path, so a reader sees the old file or the new one, never a
// half-written or briefly readable one.
func WriteFilePrivate(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			tmp.Close()
			os.Remove(name)
		}
	}()
	// Restrict before writing: the temp file is empty until this succeeds.
	if err := MakePrivate(name); err != nil {
		return fmt.Errorf("fsperm: restrict %s: %w", name, err)
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := Rename(name, path); err != nil {
		return err
	}
	ok = true
	return nil
}

// Rename is os.Rename. On Windows it is retried for up to about 500 ms while
// another process (an indexer, antivirus, a concurrent reader) holds the
// target open, which makes MoveFileEx fail with a sharing violation.
func Rename(oldpath, newpath string) error { return rename(oldpath, newpath) }
```

- [ ] **Step 4: Implement unix**

```go
// internal/fsperm/fsperm_unix.go
//go:build unix

package fsperm

import (
	"fmt"
	"os"
)

func makePrivate(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fsperm: %s is a symlink; refusing to change what it points to", path)
	}
	mode := os.FileMode(0o600)
	if fi.IsDir() {
		mode = 0o700
	}
	return os.Chmod(path, mode)
}

func checkPrivate(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	return checkMode(path, fi.Mode())
}

func checkPrivateFile(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	return checkMode(f.Name(), fi.Mode())
}

func checkMode(path string, m os.FileMode) error {
	if m.Perm()&0o077 != 0 {
		return fmt.Errorf("%w: %s is mode %o; run chmod %o %s", ErrNotPrivate, path, m.Perm(), privateMode(m), path)
	}
	return nil
}

func privateMode(m os.FileMode) os.FileMode {
	if m.IsDir() {
		return 0o700
	}
	return 0o600
}

func rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }
```

Run: `go test ./internal/fsperm/`
Expected: PASS on macOS.

- [ ] **Step 5: Implement Windows**

```go
// internal/fsperm/fsperm_windows.go
//go:build windows

package fsperm

import (
	"errors"
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// accessMask is every right that lets a SID read the content, change it, or
// change who may: an allow ACE with any of these for a stranger means the
// file is not private.
const accessMask = windows.GENERIC_ALL | windows.GENERIC_READ | windows.GENERIC_WRITE |
	0x1 /* FILE_READ_DATA */ | 0x2 /* FILE_WRITE_DATA */ | 0x4 /* FILE_APPEND_DATA */ |
	windows.WRITE_DAC | windows.WRITE_OWNER

// openForSecurity opens path itself, never what it points to: the flags make
// it work on directories (BACKUP_SEMANTICS) and on AF_UNIX sockets and links
// (OPEN_REPARSE_POINT), which a by-name call would fail on or follow.
func openForSecurity(path string, access uint32) (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	return windows.CreateFile(p, access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
}

func currentUser() (*windows.SID, error) {
	tu, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return tu.User.Sid.Copy()
}

func makePrivate(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fsperm: %s is a symlink; refusing to change what it points to", path)
	}
	user, err := currentUser()
	if err != nil {
		return err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}
	inherit := uint32(windows.NO_INHERITANCE)
	if fi.IsDir() {
		inherit = windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT
	}
	grant := func(sid *windows.SID, kind windows.TRUSTEE_TYPE) windows.EXPLICIT_ACCESS {
		return windows.EXPLICIT_ACCESS{
			AccessPermissions: windows.GENERIC_ALL,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       inherit,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  kind,
				TrusteeValue: windows.TrusteeValueFromSID(sid),
			},
		}
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{
		grant(user, windows.TRUSTEE_IS_USER),
		grant(system, windows.TRUSTEE_IS_WELL_KNOWN_GROUP),
	}, nil)
	if err != nil {
		return err
	}
	h, err := openForSecurity(path, windows.READ_CONTROL|windows.WRITE_DAC)
	if err != nil {
		return fmt.Errorf("fsperm: open %s: %w", path, err)
	}
	defer windows.CloseHandle(h)
	// PROTECTED drops inherited ACEs: a profile's inherited grants are
	// exactly what this must not depend on.
	return windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}

func checkPrivate(path string) error {
	h, err := openForSecurity(path, windows.READ_CONTROL)
	if err != nil {
		return fmt.Errorf("fsperm: open %s: %w", path, err)
	}
	defer windows.CloseHandle(h)
	return checkHandle(path, h)
}

func checkPrivateFile(f *os.File) error {
	return checkHandle(f.Name(), windows.Handle(f.Fd()))
}

// checkHandle accepts allow ACEs for the owner, SYSTEM and Administrators
// only (spec D16). Administrators can take ownership of any file anyway, and
// default profile ACLs include them. Inherit-only ACEs do not apply to the
// object itself and are skipped.
func checkHandle(path string, h windows.Handle) error {
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("fsperm: read the DACL of %s: %w", path, err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("fsperm: read the DACL of %s: %w", path, err)
	}
	if dacl == nil {
		return fmt.Errorf("%w: %s has no DACL, so everyone has full access", ErrNotPrivate, path)
	}
	user, err := currentUser()
	if err != nil {
		return err
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue
		}
		if uint32(ace.Mask)&accessMask == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if sid.Equals(user) || sid.IsWellKnown(windows.WinLocalSystemSid) || sid.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
			continue
		}
		who := sid.String()
		if acct, dom, _, lerr := sid.LookupAccount(""); lerr == nil {
			who = dom + `\` + acct
		}
		return fmt.Errorf("%w: %s grants access to %s; remove that entry (Properties > Security) or delete the file and let jumpgate recreate it", ErrNotPrivate, path, who)
	}
	return nil
}

// moveFile is a seam for the retry test.
var moveFile = os.Rename

func rename(oldpath, newpath string) error {
	var err error
	for i := 0; i < 10; i++ {
		err = moveFile(oldpath, newpath)
		if err == nil || !(errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED)) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}
```

Run: `GOOS=windows go vet ./internal/fsperm/ ./internal/testutil/ && GOOS=windows go test -c -o /dev/null ./internal/fsperm/`
Expected: clean.

- [ ] **Step 6: Make `testutil.AssertPrivate` check the DACL on Windows**

In `internal/testutil/testutil.go`, replace the Windows branch of `AssertPrivate`:

```go
	if runtime.GOOS == "windows" {
		if err := fsperm.CheckPrivate(path); err != nil {
			t.Fatal(err)
		}
		return
	}
```

and import `"github.com/valve-tech/jumpgate/internal/fsperm"`. Update its doc comment's last sentence to: "On Windows it checks the DACL through fsperm.CheckPrivate."

Run: `go test ./internal/testutil/ ./internal/fsperm/`
Expected: PASS.

- [ ] **Step 7: Write the failing key-file and socket tests**

```go
// internal/signer/keyfile_test.go
package signer

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// Review Focus 1: a key other users can read is refused on every OS,
// including Windows, where the mode check used to be skipped.
func TestLoadKeyFileRefusesASharedKey(t *testing.T) {
	p := filepath.Join(t.TempDir(), "controller.key")
	if _, err := GenerateKeyFile(p); err != nil {
		t.Fatal(err)
	}
	testutil.AssertPrivate(t, p)
	if _, err := LoadKeyFile(p); err != nil {
		t.Fatalf("LoadKeyFile on a private key: %v", err)
	}
	testutil.Loosen(t, p)
	if _, err := LoadKeyFile(p); !errors.Is(err, ErrKeyFilePermissions) {
		t.Fatalf("LoadKeyFile on a shared key = %v, want ErrKeyFilePermissions", err)
	}
}
```

In `internal/server/unix_test.go` `TestServeUnixAndShutdown`, the assert from Task 1 is already `testutil.AssertPrivate(t, sock)`; on Windows it now checks the socket's DACL (spec D2).

Run: `go test ./internal/signer/ -run SharedKey ./internal/server/ -run ServeUnix`
Expected on macOS: `TestLoadKeyFileRefusesASharedKey` passes already (unix checked before). The Windows leg will fail until Step 8 (Windows skipped the check).

- [ ] **Step 8: Route every secret write through fsperm**

`internal/signer/keyfile.go`:

```go
// ErrKeyFilePermissions refuses a key file other users can read: a signing
// key anyone on the machine can copy authorises nothing.
var ErrKeyFilePermissions = errors.New("signer: key file is readable by other users; restrict it to your user (chmod 600 on macOS and Linux)")

// LoadKeyFile reads a hex key written by GenerateKeyFile. It opens the path
// once, refuses symlinks and Windows reparse points, and checks the open
// handle, so the file it checks is the file it reads.
func LoadKeyFile(path string) (*Key, error) {
	// O_NOFOLLOW covers unix; Windows has no such open flag here, so the link
	// is refused before opening (M-4).
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, fmt.Errorf("signer: %s is a link, not a key file", path)
	}
	f, err := os.OpenFile(path, os.O_RDONLY|openNoFollow, 0)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("signer: %s is not a regular file", path)
	}
	if err := fsperm.CheckPrivateFile(f); err != nil {
		return nil, fmt.Errorf("%w (%v)", ErrKeyFilePermissions, err)
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxKeyFileSize))
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	b, err := hex.DecodeString(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(raw)), "0x")))
	if err != nil {
		return nil, fmt.Errorf("signer: %s is not a hex key", path)
	}
	return KeyFromBytes(b)
}

// GenerateKeyFile creates a new owner-only key at path, refusing to
// overwrite. A missing parent directory is created owner-only; an existing
// one the operator chose (their home, say) is left as it is.
func GenerateKeyFile(path string) (*Key, error) {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		if err := fsperm.MkdirPrivate(dir); err != nil {
			return nil, fmt.Errorf("signer: %w", err)
		}
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	// Restrict before the key is written: on Windows the new file inherits
	// its directory's DACL until this runs.
	if err := fsperm.MakePrivate(path); err != nil {
		f.Close()
		os.Remove(path)
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

Imports: add `"io/fs"` and `"github.com/valve-tech/jumpgate/internal/fsperm"`; remove `"runtime"`.

`internal/signer/onepassword.go` (the template file, ~line 67): replace `if err := f.Chmod(0o600); err != nil {` with `if err := fsperm.MakePrivate(f.Name()); err != nil {` (M-14), and import fsperm.

`internal/config/config.go`:

```go
func lockPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := fsperm.MkdirPrivate(dir); err != nil {
		return "", fmt.Errorf("config: create %s: %w", dir, err)
	}
	return filepath.Join(dir, lockFileName), nil
}

// Save writes c to ~/.jumpgate/config.json, creating the directory if
// needed. The write is atomic and owner-only (fsperm.WriteFilePrivate), since
// the file may contain AI provider API keys and VPN private keys.
func (c Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := fsperm.MkdirPrivate(dir); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	if err := fsperm.WriteFilePrivate(filepath.Join(dir, configFileName), data); err != nil {
		return fmt.Errorf("config: write: %w", err)
	}
	return nil
}
```

`internal/daemon/daemon.go`:

```go
// RunDir is ~/.jumpgate/run, owner-only (it holds the session token and the
// server socket).
func RunDir() (string, error) {
	base, err := config.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "run")
	if err := fsperm.MkdirPrivate(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// Publish writes server.json (owner-only: it carries the session token) once
// the listeners are up.
func (h *Holder) Publish(info Info) error {
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return fsperm.WriteFilePrivate(filepath.Join(h.dir, "server.json"), b)
}
```

In `EnsureRunning`, after opening `server.log`, add:

```go
	if err := fsperm.MakePrivate(logf.Name()); err != nil {
		logf.Close()
		return Info{}, err
	}
```

`internal/server/controller.go` `ensureTransportKey`: replace the `os.MkdirAll(dir, 0o700)` + `os.Chmod(dir, 0o700)` pair with `if err := fsperm.MkdirPrivate(dir); err != nil { return "", err }`, and `werr := tmp.Chmod(0o600)` with `werr := fsperm.MakePrivate(tmp.Name())` (update the comment above it to "Restrict before the key is written.").

`internal/server/server.go` `ServeUnix`: replace

```go
	if err := os.Chmod(path, 0o600); err != nil {
```

with

```go
	// Owner-only on every OS: on Windows a mode does nothing, so the socket
	// gets the same protected DACL as the run dir (spec D2).
	if err := fsperm.MakePrivate(path); err != nil {
```

`internal/server/gateways.go` (install id, ~2110):

```go
	if mkerr := fsperm.MkdirPrivate(dir); mkerr == nil {
		_ = fsperm.WriteFilePrivate(path, []byte(id+"\n")) // best-effort; retried next run
	}
```

`internal/executor/hostkey.go` `RecordHostKey`: keep `os.MkdirAll(dir, 0o700)` for missing parents (it may be handed any path), and after the `os.OpenFile(…, 0o600)` succeeds add:

```go
	// Owner-only on every OS; the confirmed store decides which hosts this
	// controller trusts, so another user must not be able to append to it.
	if err := fsperm.MakePrivate(hostKeyFile); err != nil {
		f.Close()
		return fmt.Errorf("restrict host key file %s: %w", hostKeyFile, err)
	}
```

Add the fsperm import to each file and remove imports left unused. `internal/filelock` is unchanged: lock files hold nothing secret and live in these private directories.

- [ ] **Step 9: Run everything on macOS and Linux**

Run: `go test ./... && scripts/test-linux.sh && GOOS=windows go vet ./...`
Expected: PASS; vet clean.

- [ ] **Step 10: Commit and verify on Windows**

```bash
git add internal/fsperm internal/testutil internal/signer internal/config internal/daemon internal/server internal/executor
git commit -m "feat(fsperm): owner-only files and DACLs for every secret the controller writes

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: all green, including `go (windows-latest)` running `TestMakePrivateOnAUnixSocket`, `TestRenameRetriesASharingViolation`, `TestLoadKeyFileRefusesASharedKey` and the socket DACL assert in `TestServeUnixAndShutdown`. If `TestMakePrivateOnAUnixSocket` fails with `ERROR_CANT_ACCESS_FILE`, the socket reparse point cannot be opened even with `FILE_FLAG_OPEN_REPARSE_POINT` on that Windows build: change `ServeUnix` to call `fsperm.MakePrivate(filepath.Dir(path))` (the run dir, whose inheritable DACL the socket gets at creation) before `net.Listen`, and make the socket test assert `fsperm.CheckPrivate` on the socket's directory instead; record that in the commit message.

---
### Task 3: Agent binaries — embedded in release builds, never the wrong build

Covers B-1, B-2, M-9, and D1, D13, D14, D27.

**Files:**
- Create: `internal/agentbin/agentbin.go`, `internal/agentbin/embed_on.go`, `internal/agentbin/agentbin_test.go`, `internal/agentbin/embed_on_test.go`, `internal/agentbin/embedded/.gitkeep`, `internal/buildinfo/cgo_on.go`, `internal/buildinfo/cgo_off.go`, `internal/buildinfo/static_test.go`
- Modify: `internal/buildinfo/buildinfo.go`, `internal/server/controller.go:124-153` (remove `agentBinary`), `internal/server/pair.go:131`, `scripts/build-agents.sh`, `.gitignore`, `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `fsperm.MkdirPrivate`, `fsperm.WriteFilePrivate` (Task 2); `config.Dir()`; `buildinfo.Version()`.
- Produces (package `github.com/valve-tech/jumpgate/internal/agentbin`):
  - `type Source string`; `const SourceDevDir Source = "dev override ~/.jumpgate/agents"`, `SourceEmbedded Source = "embedded in this build"`, `SourceSelf Source = "this binary"`
  - `func Path(arch string) (path string, src Source, err error)`
- Produces: `func buildinfo.SelfIsStaticLinux() bool`.
- Produces: `func server.agentBinaryReporting(report func(line string)) func(arch string) (string, error)` (used by the pair handler; Task 5 calls `agentbin.Path` directly from the CLI).
- Produces: `scripts/build-agents.sh [OUT_DIR]` with env `VERSION` (default `git describe --tags --always --dirty`) and `GZIP=1` (also write `.gz`).

- [ ] **Step 1: Write the failing `buildinfo` test**

```go
// internal/buildinfo/static_test.go
package buildinfo

import (
	"debug/elf"
	"os"
	"runtime"
	"testing"
)

// SelfIsStaticLinux must never claim a dynamically linked binary is static:
// a controller that believed it would upload a binary that cannot start on a
// headless box (B-2). The converse is allowed to be conservative.
func TestSelfIsStaticLinuxIsNeverWrong(t *testing.T) {
	if runtime.GOOS != "linux" {
		if SelfIsStaticLinux() {
			t.Fatal("SelfIsStaticLinux() is true off Linux")
		}
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	f, err := elf.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dynamic := false
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			dynamic = true
		}
	}
	if SelfIsStaticLinux() && dynamic {
		t.Fatal("SelfIsStaticLinux() is true for a dynamically linked binary")
	}
	if os.Getenv("CGO_ENABLED") == "0" && !SelfIsStaticLinux() {
		t.Fatal("a CGO_ENABLED=0 Linux build must report SelfIsStaticLinux() = true")
	}
}
```

Run: `go test ./internal/buildinfo/`
Expected: FAIL — `undefined: SelfIsStaticLinux`.

- [ ] **Step 2: Implement `SelfIsStaticLinux`**

```go
// internal/buildinfo/cgo_on.go
//go:build cgo

package buildinfo

// cgoEnabled is true when this binary was built with cgo, which links the
// build host's libc (and, for tray builds, GTK and WebKit).
const cgoEnabled = true
```

```go
// internal/buildinfo/cgo_off.go
//go:build !cgo

package buildinfo

const cgoEnabled = false
```

Append to `internal/buildinfo/buildinfo.go` (and add `import "runtime"`):

```go
// SelfIsStaticLinux reports whether this binary can itself run as the agent
// on a Linux box of the same arch. Only a Linux build without cgo is
// statically linked. A cgo build (every tray build, and a source build with
// Go's default CGO_ENABLED=1) links the build host's glibc and desktop
// libraries and would fail to start on a headless box, so it must never be
// uploaded as the agent.
func SelfIsStaticLinux() bool { return runtime.GOOS == "linux" && !cgoEnabled }
```

Run: `go test ./internal/buildinfo/ && CGO_ENABLED=0 scripts/test-linux.sh ./internal/buildinfo/ -v && scripts/test-linux.sh ./internal/buildinfo/ -v`
Expected: PASS all three (the container's default build has cgo on, and the test accepts the conservative `false`).

- [ ] **Step 3: Write the failing `agentbin` tests**

```go
// internal/agentbin/agentbin_test.go
package agentbin

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func gz(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sumLine(name string, b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]) + "  " + name + "\n"
}

// withSeams resets every seam for one test: no embedded agents, not a
// static Linux build, amd64.
func withSeams(t *testing.T) {
	t.Helper()
	oldFS, oldStatic, oldArch, oldExe := embeddedFS, selfIsStaticLinux, goarch, executable
	embeddedFS, selfIsStaticLinux, goarch = nil, func() bool { return false }, "amd64"
	executable = func() (string, error) { return "/self/jumpgate", nil }
	t.Cleanup(func() { embeddedFS, selfIsStaticLinux, goarch, executable = oldFS, oldStatic, oldArch, oldExe })
}

func embeddedWith(t *testing.T, arch string, content []byte, sums string) fstest.MapFS {
	t.Helper()
	return fstest.MapFS{
		"embedded/jumpgate-linux-" + arch + ".gz": {Data: gz(t, content)},
		"embedded/SHA256SUMS":                     {Data: []byte(sums)},
	}
}

func writeDevDir(t *testing.T, home, arch string, content []byte, sums string) string {
	t.Helper()
	dir := filepath.Join(home, ".jumpgate", "agents")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "jumpgate-linux-"+arch)
	if err := os.WriteFile(p, content, 0o755); err != nil {
		t.Fatal(err)
	}
	if sums != "" {
		if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(sums), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func TestPathRejectsAnUnsupportedArch(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	if _, _, err := Path("riscv64"); err == nil || !strings.Contains(err.Error(), "linux/amd64 and linux/arm64") {
		t.Fatalf("Path(riscv64) = %v, want an error naming the supported arches", err)
	}
}

// Review Focus 4 / D14: the developer override wins, and says so.
func TestPathPrefersTheDevDirAndNamesIt(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	dev := []byte("dev agent")
	want := writeDevDir(t, home, "amd64", dev, sumLine("jumpgate-linux-amd64", dev))
	embeddedFS = embeddedWith(t, "amd64", []byte("embedded agent"), sumLine("jumpgate-linux-amd64", []byte("embedded agent")))
	got, src, err := Path("amd64")
	if err != nil || got != want || src != SourceDevDir {
		t.Fatalf("Path = %q, %q, %v; want %q, %q", got, src, err, want, SourceDevDir)
	}
}

// A dev binary whose sums are missing or wrong is an error, never a reason
// to fall through to another source silently.
func TestPathRefusesAnUnverifiableDevBinary(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	writeDevDir(t, home, "amd64", []byte("dev agent"), "")
	if _, _, err := Path("amd64"); err == nil || !strings.Contains(err.Error(), "SHA256SUMS") {
		t.Fatalf("no sums: Path = %v, want an error naming SHA256SUMS", err)
	}
	writeDevDir(t, home, "amd64", []byte("dev agent"), sumLine("jumpgate-linux-amd64", []byte("something else")))
	if _, _, err := Path("amd64"); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("bad sums: Path = %v, want a mismatch error", err)
	}
}

func TestPathExtractsAndVerifiesTheEmbeddedAgent(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	content := []byte("embedded agent")
	embeddedFS = embeddedWith(t, "arm64", content, sumLine("jumpgate-linux-arm64", content))
	p, src, err := Path("arm64")
	if err != nil || src != SourceEmbedded {
		t.Fatalf("Path = %q, %q, %v; want the embedded source", p, src, err)
	}
	got, err := os.ReadFile(p)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("extracted %q, %v; want %q", got, err, content)
	}
	testutil.AssertPrivate(t, p)
	again, _, err := Path("arm64")
	if err != nil || again != p {
		t.Fatalf("second Path = %q, %v; want the cached %q", again, err, p)
	}
}

func TestPathRefusesADamagedEmbeddedAgent(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	embeddedFS = embeddedWith(t, "amd64", []byte("tampered"), sumLine("jumpgate-linux-amd64", []byte("original")))
	if _, _, err := Path("amd64"); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Fatalf("Path = %v, want a damaged-agent error", err)
	}
}

// B-2: the controller uploads itself only when it is a static Linux build of
// the very arch the box needs.
func TestPathUsesSelfOnlyForAStaticLinuxBuildOfTheSameArch(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	selfIsStaticLinux = func() bool { return true }
	if p, src, err := Path("amd64"); err != nil || src != SourceSelf || p != "/self/jumpgate" {
		t.Fatalf("same arch: Path = %q, %q, %v; want self", p, src, err)
	}
	if _, _, err := Path("arm64"); err == nil {
		t.Fatal("other arch: Path used self")
	}
	selfIsStaticLinux = func() bool { return false }
	if _, _, err := Path("amd64"); err == nil || !strings.Contains(err.Error(), "scripts/build-agents.sh") {
		t.Fatalf("cgo build: Path = %v, want the no-agent error naming scripts/build-agents.sh", err)
	}
}
```

Run: `go test ./internal/agentbin/`
Expected: FAIL — package does not exist / `undefined: Path`.

- [ ] **Step 4: Implement `agentbin`**

```go
// internal/agentbin/agentbin.go

// Package agentbin finds the Linux agent binary that pairing uploads to a
// box. A controller on any OS can pair any Linux box, so it needs the agent
// for the box's arch, built statically and from the same commit (spec D1).
package agentbin

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// Source says where an agent binary came from, for the pairing log.
type Source string

const (
	SourceDevDir   Source = "dev override ~/.jumpgate/agents"
	SourceEmbedded Source = "embedded in this build"
	SourceSelf     Source = "this binary"
)

// maxAgentSize bounds decompression of an embedded agent.
const maxAgentSize = 256 << 20

// Seams for tests; production never reassigns them. embeddedFS is set by
// embed_on.go in builds made with -tags embedagents and is nil otherwise.
var (
	embeddedFS        fs.FS
	selfIsStaticLinux = buildinfo.SelfIsStaticLinux
	goarch            = runtime.GOARCH
	executable        = os.Executable
)

// Path returns a local file holding the verified Linux agent for arch
// ("amd64" or "arm64"), and where it came from. The order is: the developer
// override in ~/.jumpgate/agents (spec D14); the agent embedded in a release
// build, extracted to ~/.jumpgate/agents-cache/<version>/ (D13); this binary,
// only when it is a static Linux build of that arch (B-2); otherwise an
// error saying how to get one.
func Path(arch string) (string, Source, error) {
	if arch != "amd64" && arch != "arm64" {
		return "", "", fmt.Errorf("agentbin: no agent for linux/%s; the jumpgate agent runs on linux/amd64 and linux/arm64", arch)
	}
	base, err := config.Dir()
	if err != nil {
		return "", "", err
	}
	dev := filepath.Join(base, "agents")
	if p, err := fromDir(dev, arch); err == nil {
		return p, SourceDevDir, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", "", err
	}
	if embeddedFS != nil {
		p, err := extractFrom(embeddedFS, arch, filepath.Join(base, "agents-cache", buildinfo.Version()))
		if err != nil {
			return "", "", err
		}
		return p, SourceEmbedded, nil
	}
	if selfIsStaticLinux() && goarch == arch {
		exe, err := executable()
		if err != nil {
			return "", "", err
		}
		return exe, SourceSelf, nil
	}
	return "", "", fmt.Errorf("agentbin: this build of jumpgate carries no agent for linux/%s. Use a release build (they embed both agents), or for development run scripts/build-agents.sh, which writes them to %s", arch, dev)
}

// fromDir returns dir's agent for arch after checking it against the
// SHA256SUMS beside it. Only a missing binary reports fs.ErrNotExist, so the
// caller falls through; anything else about a present binary is an error.
func fromDir(dir, arch string) (string, error) {
	name := "jumpgate-linux-" + arch
	path := filepath.Join(dir, name)
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err != nil {
		return "", fmt.Errorf("agentbin: %v", err)
	}
	sums, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return "", fmt.Errorf("agentbin: %s has no readable SHA256SUMS beside it (%v); rerun scripts/build-agents.sh", path, err)
	}
	if err := checkSum(sums, name, content); err != nil {
		return "", fmt.Errorf("agentbin: %s: %v", path, err)
	}
	return path, nil
}

// extractFrom decompresses the embedded agent for arch, checks it against
// the embedded sums, and caches it owner-only under cacheDir. The cache is
// rewritten if its content differs, so a damaged cache heals itself.
func extractFrom(fsys fs.FS, arch, cacheDir string) (string, error) {
	name := "jumpgate-linux-" + arch
	sums, err := fs.ReadFile(fsys, "embedded/SHA256SUMS")
	if err != nil {
		return "", fmt.Errorf("agentbin: embedded SHA256SUMS: %w", err)
	}
	f, err := fsys.Open("embedded/" + name + ".gz")
	if err != nil {
		return "", fmt.Errorf("agentbin: embedded agent for linux/%s: %w", arch, err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("agentbin: the embedded agent for linux/%s is damaged: %v", arch, err)
	}
	content, err := io.ReadAll(io.LimitReader(zr, maxAgentSize+1))
	if err != nil || len(content) > maxAgentSize {
		return "", fmt.Errorf("agentbin: the embedded agent for linux/%s is damaged: %v", arch, err)
	}
	if err := checkSum(sums, name, content); err != nil {
		return "", fmt.Errorf("agentbin: the embedded agent for linux/%s is damaged: %v", arch, err)
	}
	path := filepath.Join(cacheDir, name)
	if cached, err := os.ReadFile(path); err == nil && bytes.Equal(cached, content) {
		return path, nil
	}
	if err := fsperm.MkdirPrivate(cacheDir); err != nil {
		return "", err
	}
	if err := fsperm.WriteFilePrivate(path, content); err != nil {
		return "", err
	}
	return path, nil
}

// checkSum finds name in a sha256sum-format list and compares content to it.
func checkSum(sums []byte, name string, content []byte) error {
	got := sha256.Sum256(content)
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			if !strings.EqualFold(f[0], hex.EncodeToString(got[:])) {
				return fmt.Errorf("%s does not match its SHA256SUMS entry", name)
			}
			return nil
		}
	}
	return fmt.Errorf("%s is not listed in SHA256SUMS", name)
}
```

```go
// internal/agentbin/embed_on.go
//go:build embedagents

package agentbin

import "embed"

// Release builds run scripts/build-agents.sh into ./embedded with GZIP=1
// before compiling, so these files exist exactly when the tag is set.
//
//go:embed embedded/jumpgate-linux-amd64.gz embedded/jumpgate-linux-arm64.gz embedded/SHA256SUMS
var embeddedFiles embed.FS

func init() { embeddedFS = embeddedFiles }
```

Create `internal/agentbin/embedded/.gitkeep` (empty) and add to `.gitignore`:

```gitignore
# Agents embedded by release builds (scripts/build-agents.sh with GZIP=1).
/internal/agentbin/embedded/*
!/internal/agentbin/embedded/.gitkeep
```

Run: `go test ./internal/agentbin/ && scripts/test-linux.sh ./internal/agentbin/`
Expected: PASS.

- [ ] **Step 5: Rewrite `scripts/build-agents.sh`**

```bash
#!/usr/bin/env bash
# scripts/build-agents.sh — build the static Linux agent binaries that pairing
# uploads, plus the SHA256SUMS they are checked against. One script for every
# use, so developer, embedded and published agents are byte-identical for a
# version (spec D27):
#
#   scripts/build-agents.sh                                  # ~/.jumpgate/agents (dev override)
#   GZIP=1 scripts/build-agents.sh internal/agentbin/embedded  # for -tags embedagents
#   VERSION=v0.9.0 ...                                       # release: the git tag
set -euo pipefail
out="${1:-$HOME/.jumpgate/agents}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
mkdir -p "$out"
sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$@"; else shasum -a 256 "$@"; fi
}
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$version" \
    -o "$out/jumpgate-linux-$arch" ./cmd/jumpgate
done
( cd "$out" && sha256 jumpgate-linux-amd64 jumpgate-linux-arm64 > SHA256SUMS )
if [ "${GZIP:-0}" = 1 ]; then
  for arch in amd64 arm64; do
    gzip -9 -n -k -f "$out/jumpgate-linux-$arch"
  done
fi
echo "agents ($version) in $out"
```

Run: `GZIP=1 scripts/build-agents.sh internal/agentbin/embedded && ls internal/agentbin/embedded && git status --short internal/agentbin`
Expected: `SHA256SUMS`, both binaries and both `.gz` files exist; `git status` shows nothing under `internal/agentbin/embedded` except possibly `.gitkeep` as new.

- [ ] **Step 6: Write the embedded-build test**

```go
// internal/agentbin/embed_on_test.go
//go:build embedagents

package agentbin

import (
	"debug/elf"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// The agents a release embeds are Linux ELF files of the right machine, and
// statically linked: no PT_INTERP, so they start on any distro (D1).
func TestEmbeddedAgentsAreStaticLinuxOfTheRightArch(t *testing.T) {
	for arch, machine := range map[string]elf.Machine{"amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64} {
		testutil.Home(t)
		p, src, err := Path(arch)
		if err != nil || src != SourceEmbedded {
			t.Fatalf("%s: Path = %q, %q, %v; want the embedded agent", arch, p, src, err)
		}
		f, err := elf.Open(p)
		if err != nil {
			t.Fatalf("%s: %v", arch, err)
		}
		if f.Machine != machine {
			t.Errorf("%s: machine %v, want %v", arch, f.Machine, machine)
		}
		for _, prog := range f.Progs {
			if prog.Type == elf.PT_INTERP {
				t.Errorf("%s: dynamically linked (has PT_INTERP)", arch)
			}
		}
		f.Close()
	}
}
```

Run: `go test -tags embedagents ./internal/agentbin/ -run Embedded -v`
Expected: PASS (it uses the files Step 5 built).

- [ ] **Step 7: Use `agentbin` from the pair handler and name the source**

In `internal/server/controller.go`, delete `agentBinary` (lines 124-153) and its now-unused imports (`crypto/sha256`, `encoding/hex`, `runtime` if unused), and add:

```go
// agentBinaryReporting finds the Linux agent for an arch with agentbin.Path
// and reports its source on the pairing stream, so a stale developer
// override is visible to the person pairing (spec D14).
func agentBinaryReporting(report func(line string)) func(arch string) (string, error) {
	return func(arch string) (string, error) {
		p, src, err := agentbin.Path(arch)
		if err == nil {
			report(fmt.Sprintf("agent binary for linux/%s: %s", arch, src))
		}
		return p, err
	}
}
```

In `internal/server/pair.go`, the `bootstrap.Run` options change `AgentBinary: agentBinary,` to:

```go
		AgentBinary: agentBinaryReporting(func(line string) { send(pairEvent{Step: "upload", Line: line}) }),
```

Run: `go build ./... && go test ./internal/server/ ./internal/bootstrap/`
Expected: PASS.

- [ ] **Step 8: Add the embedded-agents CI job**

Append to `.github/workflows/ci.yml` `jobs:`:

```yaml
  # Release builds embed both agents (spec D1). Build them the way the release
  # does and prove extraction and verification work on every controller OS.
  agents:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-24.04, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    defaults:
      run:
        shell: bash
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: build the embedded agents
        run: GZIP=1 scripts/build-agents.sh internal/agentbin/embedded
      - name: test the embedded build
        run: go test -tags embedagents ./internal/agentbin/
      - name: a release-style controller builds
        env:
          CGO_ENABLED: "0"
        run: go build -tags embedagents -o "$RUNNER_TEMP/jg" ./cmd/jumpgate
```

- [ ] **Step 9: Run the full suite and the SSH e2e on Linux, then commit**

Run: `go test ./... && scripts/test-linux.sh && scripts/e2e-agent.sh`
Expected: PASS; the e2e output's pairing stream shows its upload step.

```bash
rm -f internal/agentbin/embedded/jumpgate-linux-* internal/agentbin/embedded/SHA256SUMS
git add internal/agentbin internal/buildinfo internal/server scripts/build-agents.sh .gitignore .github/workflows/ci.yml
git commit -m "feat(agentbin): embed verified static agents in release builds; never upload a cgo controller

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: the `agents` job is green on all three OSes.

---

### Task 4: Key stores — Windows Credential Manager, and a sane Linux default

Covers I-3, I-4, D3, D17.

**Files:**
- Create: `internal/signer/wincred.go`, `internal/signer/wincred_windows.go`, `internal/signer/wincred_other.go`, `internal/signer/wincred_test.go`, `internal/signer/wincred_windows_test.go`, `internal/signer/secretservice.go`, `internal/signer/secretservice_test.go`, `internal/signer/secretservice_real_test.go`, `scripts/test-secret-service.sh`
- Modify: `internal/signer/store.go` (`StoreWinCred`, `DefaultStore`, `Open`, `Create`, `runCmd` timeout), `internal/signer/keychain.go` (`ErrNoKeychain`, `keychainTool`, `keychainCreate`), `cmd/jumpgate/cli_keys.go`, `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: the existing seams `hostOS`, `lookPath`, `runCmd`, `cmdError`, `keychainNameRE`, `verifyStored`, `keyFromHex`, `ErrKeyExists`, `GenerateKey`.
- Produces: `const signer.StoreWinCred Store = "wincred"`; `DefaultStore()` returns it on Windows.
- Produces (unexported, for tests): `type credStore interface { read(target string) ([]byte, error); write(target string, blob []byte) error; del(target string) error }`, `var winCreds credStore`, `var errCredNotFound`, `func credTarget(ref string) string`, `var dbusSession func() bool`, `var toolTimeout time.Duration` (2m), `const probeTimeout = 3 * time.Second`, `func secretServiceAnswers(ctx context.Context) bool`.

- [ ] **Step 1: Write the failing store tests (all OSes, with fakes)**

```go
// internal/signer/wincred_test.go
package signer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeCreds struct {
	m       map[string][]byte
	readErr error
}

func (f *fakeCreds) read(target string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	b, ok := f.m[target]
	if !ok {
		return nil, errCredNotFound
	}
	return b, nil
}
func (f *fakeCreds) write(target string, blob []byte) error { f.m[target] = blob; return nil }
func (f *fakeCreds) del(target string) error                { delete(f.m, target); return nil }

func withCreds(t *testing.T, f *fakeCreds) {
	t.Helper()
	old := winCreds
	winCreds = f
	t.Cleanup(func() { winCreds = old })
}

func TestWinCredCreateOpenRoundTrip(t *testing.T) {
	f := &fakeCreds{m: map[string][]byte{}}
	withCreds(t, f)
	k, err := Create(context.Background(), StoreWinCred, "controller")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.m["jumpgate/controller"]; !ok {
		t.Fatalf("stored under %v, want target jumpgate/controller", f.m)
	}
	k2, err := Open(context.Background(), StoreWinCred, "controller")
	if err != nil || k2.Address() != k.Address() {
		t.Fatalf("Open = %v, %v; want the created key", k2, err)
	}
}

func TestWinCredNeverReplacesAKey(t *testing.T) {
	f := &fakeCreds{m: map[string][]byte{"jumpgate/controller": []byte("00")}}
	withCreds(t, f)
	if _, err := Create(context.Background(), StoreWinCred, "controller"); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("Create over an existing credential = %v, want ErrKeyExists", err)
	}
}

// Any read failure other than "not found" fails closed: treating it as absent
// could replace a real key.
func TestWinCredExistenceCheckFailsClosed(t *testing.T) {
	f := &fakeCreds{m: map[string][]byte{}, readErr: errors.New("access denied")}
	withCreds(t, f)
	if _, err := Create(context.Background(), StoreWinCred, "controller"); err == nil || errors.Is(err, ErrKeyExists) {
		t.Fatalf("Create = %v, want a hard error", err)
	}
	if len(f.m) != 0 {
		t.Fatal("a key was stored after a failed existence check")
	}
}

func TestWinCredRejectsUnsafeNames(t *testing.T) {
	withCreds(t, &fakeCreds{m: map[string][]byte{}})
	if _, err := Create(context.Background(), StoreWinCred, "a b"); err == nil || !strings.Contains(err.Error(), "may only contain") {
		t.Fatalf("Create(\"a b\") = %v", err)
	}
}
```

```go
// internal/signer/secretservice_test.go
package signer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func withDBus(t *testing.T, up bool) {
	t.Helper()
	old := dbusSession
	dbusSession = func() bool { return up }
	t.Cleanup(func() { dbusSession = old })
}

func TestDefaultStorePerOS(t *testing.T) {
	answers := func(stdin, name string, args []string) (string, error) { return "", nil }
	noService := func(stdin, name string, args []string) (string, error) {
		return "", &cmdError{Name: name, ExitCode: 1, Stderr: "Cannot autolaunch D-Bus without X11 $DISPLAY", Err: errors.New("exit status 1")}
	}
	cases := []struct {
		name string
		goos string
		have []string
		dbus bool
		run  func(string, string, []string) (string, error)
		want Store
	}{
		{"windows", "windows", nil, false, answers, StoreWinCred},
		{"macOS with security", "darwin", []string{"security"}, false, answers, StoreKeychain},
		{"macOS without security", "darwin", nil, false, answers, StoreFile},
		{"linux desktop", "linux", []string{"secret-tool"}, true, answers, StoreKeychain},
		{"linux over SSH, no D-Bus", "linux", []string{"secret-tool"}, false, answers, StoreFile},
		{"linux, D-Bus but no Secret Service", "linux", []string{"secret-tool"}, true, noService, StoreFile},
		{"linux without secret-tool", "linux", nil, true, answers, StoreFile},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fakeRunner{fn: c.run}
			withRunner(t, f, c.goos, c.have...)
			withDBus(t, c.dbus)
			if got := DefaultStore(); got != c.want {
				t.Fatalf("DefaultStore() = %q, want %q (calls %v)", got, c.want, f.calls)
			}
			if !c.dbus && c.goos == "linux" && len(f.calls) != 0 {
				t.Fatalf("probed the Secret Service with no D-Bus session: %v", f.calls)
			}
		})
	}
}

// I-4: over SSH there is no D-Bus session; the error must say why and name
// the file store.
func TestKeychainOnLinuxWithoutDBusNamesTheFileStore(t *testing.T) {
	withRunner(t, &fakeRunner{}, "linux", "secret-tool")
	withDBus(t, false)
	_, err := Create(context.Background(), StoreKeychain, "controller")
	if !errors.Is(err, ErrNoKeychain) || !strings.Contains(err.Error(), "D-Bus") || !strings.Contains(err.Error(), "--store file") {
		t.Fatalf("Create = %v; want ErrNoKeychain naming D-Bus and --store file", err)
	}
}

// D17: a tool that never answers (a locked keyring with nothing to prompt)
// is cut off with a message saying what to do.
func TestRunCmdTimesOut(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	old := toolTimeout
	toolTimeout = 100 * time.Millisecond
	t.Cleanup(func() { toolTimeout = old })
	start := time.Now()
	_, err := runCmd(context.Background(), "", "sleep", "5")
	if err == nil || !strings.Contains(err.Error(), "did not answer within") || !strings.Contains(err.Error(), "--store file") {
		t.Fatalf("runCmd = %v; want the timeout message", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("runCmd did not stop the tool at the timeout")
	}
}
```

Run: `go test ./internal/signer/`
Expected: FAIL — `undefined: StoreWinCred`, `winCreds`, `dbusSession`, `toolTimeout`.

- [ ] **Step 2: Implement the Credential Manager store (portable part)**

```go
// internal/signer/wincred.go
package signer

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// credStore is Windows Credential Manager, behind an interface so the store's
// rules (fail closed, never replace, verify) are tested on every OS.
type credStore interface {
	read(target string) ([]byte, error) // errCredNotFound when absent
	write(target string, blob []byte) error
	del(target string) error
}

var errCredNotFound = errors.New("signer: credential not found")

// winCreds is the real Credential Manager on Windows and a store that refuses
// everything elsewhere. Tests replace it.
var winCreds credStore = platformCreds()

// credTarget is the generic credential's target name for a ref.
func credTarget(ref string) string { return "jumpgate/" + ref }

func winCredCreate(ref string) (*Key, error) {
	if !keychainNameRE.MatchString(ref) {
		return nil, fmt.Errorf("signer: credential name %q may only contain letters, digits, '.', '_' and '-'", ref)
	}
	target := credTarget(ref)
	switch _, err := winCreds.read(target); {
	case err == nil:
		return nil, fmt.Errorf("%w: Windows credential %q", ErrKeyExists, target)
	case !errors.Is(err, errCredNotFound):
		return nil, fmt.Errorf("signer: Windows Credential Manager existence check failed: %w", err)
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	if err := winCreds.write(target, []byte(hex.EncodeToString(k.Bytes()))); err != nil {
		return nil, fmt.Errorf("signer: store in Windows Credential Manager: %w", err)
	}
	k2, err := verifyStored(k, func() (*Key, error) { return winCredRead(ref) })
	if err != nil {
		return nil, fmt.Errorf("signer: a key was written to Windows credential %q but could not be verified; inspect or remove it in Credential Manager: %w", target, err)
	}
	return k2, nil
}

func winCredRead(ref string) (*Key, error) {
	b, err := winCreds.read(credTarget(ref))
	if errors.Is(err, errCredNotFound) {
		return nil, fmt.Errorf("signer: no Windows credential %q; run `jumpgate keys init --store wincred`", credTarget(ref))
	}
	if err != nil {
		return nil, fmt.Errorf("signer: read Windows Credential Manager: %w", err)
	}
	return keyFromHex(string(b))
}
```

```go
// internal/signer/wincred_other.go
//go:build !windows

package signer

import "errors"

var errWinCredUnsupported = errors.New("signer: Windows Credential Manager exists only on Windows; use --store keychain or --store file")

type noCreds struct{}

func (noCreds) read(string) ([]byte, error) { return nil, errWinCredUnsupported }
func (noCreds) write(string, []byte) error  { return errWinCredUnsupported }
func (noCreds) del(string) error            { return errWinCredUnsupported }

func platformCreds() credStore { return noCreds{} }
```

```go
// internal/signer/wincred_windows.go
//go:build windows

package signer

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// advapi32's credential functions are not wrapped by x/sys/windows, so they
// are loaded lazily from the system directory (never from the search path).
var (
	advapi32        = windows.NewLazySystemDLL("advapi32.dll")
	procCredWriteW  = advapi32.NewProc("CredWriteW")
	procCredReadW   = advapi32.NewProc("CredReadW")
	procCredDeleteW = advapi32.NewProc("CredDeleteW")
	procCredFree    = advapi32.NewProc("CredFree")
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2 // this user, this machine; never roams to other machines
)

// credential is CREDENTIALW.
type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type winCredManager struct{}

func platformCreds() credStore { return winCredManager{} }

func (winCredManager) read(target string) ([]byte, error) {
	t, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return nil, err
	}
	var pc *credential
	r, _, callErr := procCredReadW.Call(uintptr(unsafe.Pointer(t)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&pc)))
	if r == 0 {
		if errors.Is(callErr, windows.ERROR_NOT_FOUND) {
			return nil, errCredNotFound
		}
		return nil, fmt.Errorf("CredReadW: %w", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(pc)))
	blob := unsafe.Slice(pc.CredentialBlob, pc.CredentialBlobSize)
	out := make([]byte, len(blob))
	copy(out, blob)
	return out, nil
}

func (winCredManager) write(target string, blob []byte) error {
	t, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	user, _ := windows.UTF16PtrFromString("jumpgate")
	comment, _ := windows.UTF16PtrFromString("jumpgate controller key")
	c := credential{
		Type:               credTypeGeneric,
		TargetName:         t,
		Comment:            comment,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     &blob[0],
		Persist:            credPersistLocalMachine,
		UserName:           user,
	}
	if r, _, callErr := procCredWriteW.Call(uintptr(unsafe.Pointer(&c)), 0); r == 0 {
		return fmt.Errorf("CredWriteW: %w", callErr)
	}
	return nil
}

func (winCredManager) del(target string) error {
	t, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	if r, _, callErr := procCredDeleteW.Call(uintptr(unsafe.Pointer(t)), credTypeGeneric, 0); r == 0 && !errors.Is(callErr, windows.ERROR_NOT_FOUND) {
		return fmt.Errorf("CredDeleteW: %w", callErr)
	}
	return nil
}
```

- [ ] **Step 3: Implement the Secret Service probe and the tool timeout**

```go
// internal/signer/secretservice.go
package signer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// toolTimeout bounds every key-store tool run. It is long because macOS and
// 1Password may show a prompt a person has to answer; it exists so a locked
// keyring with nothing to prompt cannot hang server start forever (spec D17).
// A var so tests can shorten it.
var toolTimeout = 2 * time.Minute

// probeTimeout bounds the Secret Service liveness probe, which never prompts.
const probeTimeout = 3 * time.Second

// dbusSession reports whether a D-Bus session bus is reachable; the Secret
// Service lives on it. Over SSH there usually is none. A seam for tests.
var dbusSession = func() bool {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return true
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		if _, err := os.Stat(filepath.Join(d, "bus")); err == nil {
			return true
		}
	}
	return false
}

// secretServiceAnswers runs a harmless search. An answer, empty or not, means
// a Secret Service is there; an error printed on stderr, or no answer within
// probeTimeout, means it is not.
func secretServiceAnswers(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	_, err := runCmd(ctx, "", "secret-tool", "search", "service", keychainService)
	if err == nil {
		return true
	}
	var ce *cmdError
	return errors.As(err, &ce) && ce.ExitCode == 1 && strings.TrimSpace(ce.Stderr) == "" && ctx.Err() == nil
}
```

In `internal/signer/store.go`:

```go
const (
	StoreFile        Store = "file"
	StoreKeychain    Store = "keychain"
	StoreWinCred     Store = "wincred"
	StoreOnePassword Store = "1password"
)
```

Replace `runCmd`'s body so every tool run is bounded:

```go
	runCmd = func(ctx context.Context, stdin, name string, args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, toolTimeout)
		defer cancel()
		c := exec.CommandContext(ctx, name, args...)
		c.Stdin = strings.NewReader(stdin)
		var out, errb bytes.Buffer
		c.Stdout, c.Stderr = &out, &errb
		if err := c.Run(); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				err = fmt.Errorf("%s did not answer within %s; the keyring may be locked with nothing to prompt you. Unlock it, or use --store file", name, toolTimeout)
			}
			ce := &cmdError{Name: name, ExitCode: -1, Stdout: out.String(), Stderr: errb.String(), Err: err}
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				ce.ExitCode = ee.ExitCode()
			}
			return "", ce
		}
		return out.String(), nil
	}
```

Replace `DefaultStore`:

```go
// DefaultStore picks the OS key store when it can actually be used here:
// Credential Manager on Windows; the macOS keychain; on Linux the Secret
// Service only when a D-Bus session exists and the service answers (a
// headless box or an SSH login has neither). Otherwise a key file.
func DefaultStore() Store {
	switch hostOS {
	case "windows":
		return StoreWinCred
	case "darwin":
		if lookPath("security") == nil {
			return StoreKeychain
		}
	case "linux":
		if keychainTool() == "secret-tool" && secretServiceAnswers(context.Background()) {
			return StoreKeychain
		}
	}
	return StoreFile
}
```

Add the `StoreWinCred` cases to `Open` (`return winCredRead(ref)`) and `Create` (`return winCredCreate(ref)`).

In `internal/signer/keychain.go`:

```go
// ErrNoKeychain means this machine has no usable keychain. It is an error
// rather than a quiet fallback to a file, so a key never lands somewhere the
// operator did not choose.
var ErrNoKeychain = errors.New("signer: no OS keychain here (macOS needs `security`; Linux needs `secret-tool` and a D-Bus session; Windows uses --store wincred); use --store file")

func keychainTool() string {
	switch hostOS {
	case "darwin":
		if lookPath("security") == nil {
			return "security"
		}
	case "linux":
		if lookPath("secret-tool") == nil && dbusSession() {
			return "secret-tool"
		}
	}
	return ""
}

// noKeychainErr explains why there is no keychain, naming the common
// headless case where secret-tool is installed but unreachable.
func noKeychainErr() error {
	if hostOS == "linux" && lookPath("secret-tool") == nil && !dbusSession() {
		return fmt.Errorf("%w: secret-tool is installed but there is no D-Bus session (usual over SSH)", ErrNoKeychain)
	}
	return ErrNoKeychain
}
```

and in `keychainCreate`, change `return nil, ErrNoKeychain` to `return nil, noKeychainErr()`. In `keychainRead`, add before the tool switch:

```go
	if keychainTool() == "" {
		return nil, noKeychainErr()
	}
```

Run: `go test ./internal/signer/ && GOOS=windows go vet ./internal/signer/`
Expected: PASS; vet clean.

- [ ] **Step 4: Wire `wincred` into `keys init`**

In `cmd/jumpgate/cli_keys.go` `keysInit`:

```go
	store := fset.String("store", string(signer.DefaultStore()), "file | keychain | wincred | 1password")
	ref := fset.String("ref", "", "key file path, keychain or Windows credential name, or op://vault/item/field")
```

and in the `*ref == ""` switch:

```go
		case signer.StoreKeychain, signer.StoreWinCred:
			*ref = "controller"
```

with the default branch's message `"unknown --store %q: want file, keychain, wincred or 1password"`.

Run: `go build ./cmd/jumpgate && go test ./cmd/jumpgate/`
Expected: PASS.

- [ ] **Step 5: Write the real-store tests**

```go
// internal/signer/wincred_windows_test.go
//go:build windows

package signer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

// Against the real Credential Manager of the CI runner's user. The ref is
// unique per run, and the credential is deleted afterwards.
func TestRealWinCred(t *testing.T) {
	ref := fmt.Sprintf("jumpgate-ci-%d-%d", os.Getpid(), time.Now().UnixNano())
	t.Cleanup(func() { _ = winCreds.del(credTarget(ref)) })
	k, err := Create(context.Background(), StoreWinCred, ref)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := Open(context.Background(), StoreWinCred, ref)
	if err != nil || k2.Address() != k.Address() {
		t.Fatalf("Open = %v, %v; want %s", k2, err, k.Address().Hex())
	}
	if _, err := Create(context.Background(), StoreWinCred, ref); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("second Create = %v, want ErrKeyExists", err)
	}
	if DefaultStore() != StoreWinCred {
		t.Fatalf("DefaultStore() = %q on Windows", DefaultStore())
	}
}
```

```go
// internal/signer/secretservice_real_test.go
//go:build realsecretservice

package signer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Run by scripts/test-secret-service.sh in a container with libsecret-tools.

func TestRealSecretServiceDefaultWithoutDBus(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	if got := DefaultStore(); got != StoreFile {
		t.Fatalf("DefaultStore() with no D-Bus = %q, want file", got)
	}
}

func TestRealSecretServiceRoundTrip(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("needs dbus-run-session")
	}
	if got := DefaultStore(); got != StoreKeychain {
		t.Fatalf("DefaultStore() under a live Secret Service = %q, want keychain", got)
	}
	ref := fmt.Sprintf("jumpgate-test-%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = exec.Command("secret-tool", "clear", "service", keychainService, "account", ref).Run() })
	k, err := Create(context.Background(), StoreKeychain, ref)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := Open(context.Background(), StoreKeychain, ref)
	if err != nil || k2.Address() != k.Address() {
		t.Fatalf("Open = %v, %v", k2, err)
	}
}
```

```bash
#!/usr/bin/env bash
# scripts/test-secret-service.sh — the Linux key-store default against a real
# Secret Service (spec D3): no D-Bus session means a key file; under
# dbus-run-session with an unlocked gnome-keyring, a keychain key round-trips.
set -euo pipefail
docker run --rm -v "$PWD":/src -w /src -e GOFLAGS=-buildvcs=false "${GO_IMAGE:-golang:1.25}" bash -euc '
  apt-get update -qq
  apt-get install -y -qq libsecret-tools gnome-keyring dbus >/dev/null
  go test -count=1 -tags realsecretservice -run TestRealSecretServiceDefaultWithoutDBus ./internal/signer/
  dbus-run-session -- bash -euc "
    printf test | gnome-keyring-daemon --unlock --components=secrets >/dev/null
    go test -count=1 -tags realsecretservice -run TestRealSecretServiceRoundTrip -v ./internal/signer/
  "
'
```

Run: `chmod +x scripts/test-secret-service.sh && scripts/test-secret-service.sh`
Expected: both tests PASS in the container.

- [ ] **Step 6: Add the Secret Service job to CI**

Append to `.github/workflows/ci.yml` `jobs:`:

```yaml
  secret-service:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
      - name: Linux key-store default against a real Secret Service
        run: scripts/test-secret-service.sh
```

- [ ] **Step 7: Update the README key-store line and commit**

In `README.md`, replace the `jumpgate keys init` line (~line 76) with:

```
jumpgate keys init                      # create the controller signing key: --store keychain (macOS, or Linux with a desktop session), wincred (Windows Credential Manager, the Windows default), file (an owner-only file; the default over SSH and on headless Linux) or 1password (--ref op://vault/item/field)
```

```bash
git add internal/signer cmd/jumpgate/cli_keys.go scripts/test-secret-service.sh .github/workflows/ci.yml README.md
git commit -m "feat(signer): Windows Credential Manager store; Secret Service only with a D-Bus session; bounded tool runs

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: `TestRealWinCred` passes on `go (windows-latest)`; `secret-service` is green.

---

### Task 5: Local pairing — root without sudo, non-root from the terminal, refused off Linux

Covers B-4, I-10, M-8, D5, D18, D19.

**Files:**
- Create: `internal/bootstrap/local.go`, `internal/bootstrap/local_test.go`, `cmd/jumpgate/pair_local.go`, `cmd/jumpgate/pair_local_test.go`, `scripts/e2e-local.sh`
- Modify: `internal/server/server.go` (`Config.Geteuid`, `Server.goos`, `Server.geteuid`), `internal/server/pair.go` (request, guards, root path, installed path), `internal/server/pair_test.go`, `cmd/jumpgate/cli_hosts.go` (`hostsAdd`, `streamPair`), `cmd/jumpgate/cli_agent.go` (`cmdAgent`), `scripts/e2e-agent.sh` (docker flags), `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `agentbin.Path` (Task 3); `bootstrap.Run`, `bootstrap.Options`; `executor.Sudo`, `executor.NewLocal`; `testutil.Home`.
- Produces: `var bootstrap.ErrLocalUnsupported error`; `func bootstrap.LocalSupported(goos string) error`.
- Produces: pair request body `{"sudo": bool, "installed": "0x…"}`; error codes `local_unsupported` (400) and `local_needs_terminal` (409).
- Produces (cmd/jumpgate, unexported): `type pairBody struct { Sudo bool; Installed string }`, `func streamPair(info daemon.Info, name string, body pairBody) int`, `func pairLocalForeground(ctx context.Context, out io.Writer) (string, int)`, seams `hostGOOS`, `geteuid`, `ensureRunning`, `sudoNonInteractive`, `sudoPrompt`, `runBootstrap`, `newLocalExecutor`.

- [ ] **Step 1: Write the failing shared guard test**

```go
// internal/bootstrap/local_test.go
package bootstrap

import (
	"errors"
	"testing"
)

func TestLocalSupportedOnlyOnLinux(t *testing.T) {
	if err := LocalSupported("linux"); err != nil {
		t.Fatalf("linux: %v", err)
	}
	for _, goos := range []string{"darwin", "windows", "freebsd"} {
		if err := LocalSupported(goos); !errors.Is(err, ErrLocalUnsupported) {
			t.Fatalf("%s: %v, want ErrLocalUnsupported", goos, err)
		}
	}
}
```

Run: `go test ./internal/bootstrap/ -run LocalSupported`
Expected: FAIL — `undefined: LocalSupported`.

- [ ] **Step 2: Implement it**

```go
// internal/bootstrap/local.go
package bootstrap

import "errors"

// ErrLocalUnsupported refuses --local off Linux: it pairs the machine the
// controller runs on, and the agent runs only on Linux (I-10).
var ErrLocalUnsupported = errors.New("--local pairs this machine, and the jumpgate agent runs only on Linux; pair a Linux box with --ssh instead")

// LocalSupported reports whether a controller on goos can pair itself.
func LocalSupported(goos string) error {
	if goos == "linux" {
		return nil
	}
	return ErrLocalUnsupported
}
```

Run: `go test ./internal/bootstrap/ -run LocalSupported`
Expected: PASS.

- [ ] **Step 3: Write the failing server tests**

Append to `internal/server/pair_test.go`:

```go
// localPairServer saves a local target and starts a server whose OS, euid
// and local executor the test chooses.
func localPairServer(t *testing.T, goos string, euid int, local executor.Executor) (*httptest.Server, string) {
	t.Helper()
	testutil.Home(t)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "local"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctrl, _ := signer.GenerateKey()
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{}, Signer: ctrl,
		NewLocalExecutor: func() executor.Executor { return local },
		Geteuid:          func() int { return euid }})
	s.goos = goos
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

// recordingExec answers every command with "Darwin", so bootstrap stops at
// its Linux preflight, and records what it was asked to run.
type recordingExec struct {
	mu   sync.Mutex
	cmds []string
}

func (r *recordingExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	r.mu.Lock()
	r.cmds = append(r.cmds, cmd)
	r.mu.Unlock()
	return executor.Result{Stdout: "Darwin\n"}, nil
}
func (r *recordingExec) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (r *recordingExec) ReadFile(context.Context, string) ([]byte, error)          { return nil, nil }
func (r *recordingExec) Close() error                                                { return nil }

func readCode(t *testing.T, res *http.Response) (int, string, string) {
	t.Helper()
	var e struct{ Code, Hint string }
	_ = json.NewDecoder(res.Body).Decode(&e)
	return res.StatusCode, e.Code, e.Hint
}

func TestPairLocalIsRefusedOffLinux(t *testing.T) {
	ts, token := localPairServer(t, "darwin", 0, &recordingExec{})
	status, code, _ := readCode(t, postPair(t, ts, token, `{}`))
	if status != http.StatusBadRequest || code != "local_unsupported" {
		t.Fatalf("got %d %q, want 400 local_unsupported", status, code)
	}
}

// D5: the detached server has no terminal, so it never tries sudo for a
// non-root local pairing; it says how to do it instead.
func TestPairLocalNonRootNeedsTheTerminal(t *testing.T) {
	rec := &recordingExec{}
	ts, token := localPairServer(t, "linux", 1000, rec)
	status, code, hint := readCode(t, postPair(t, ts, token, `{}`))
	if status != http.StatusConflict || code != "local_needs_terminal" || !strings.Contains(hint, "jumpgate hosts add") {
		t.Fatalf("got %d %q %q, want 409 local_needs_terminal with a hint", status, code, hint)
	}
	if len(rec.cmds) != 0 {
		t.Fatalf("ran commands: %v", rec.cmds)
	}
}

// B-4: as root, pairing runs the steps directly. Stock Debian with a root
// password has no sudo at all.
func TestPairLocalAsRootDoesNotUseSudo(t *testing.T) {
	rec := &recordingExec{}
	ts, token := localPairServer(t, "linux", 0, rec)
	res := postPair(t, ts, token, `{}`)
	_, _ = io.ReadAll(res.Body)
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.cmds) == 0 {
		t.Fatal("no command ran")
	}
	for _, c := range rec.cmds {
		if strings.Contains(c, "sudo") {
			t.Fatalf("root pairing used sudo: %q", c)
		}
	}
}

func TestPairInstalledOnlyAppliesToLocalTargets(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	ts, token := pairServer(t, d)
	res := postPair(t, ts, token, `{"installed":"0x0000000000000000000000000000000000000001"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
}
```

Add imports `"context"`, `"io/fs"`, `"github.com/valve-tech/jumpgate/internal/testutil"` to `pair_test.go` as needed.

Run: `go test ./internal/server/ -run 'PairLocal|PairInstalled'`
Expected: FAIL — `unknown field Geteuid`, `s.goos undefined`.

- [ ] **Step 4: Implement the server side**

In `internal/server/server.go`, add to `Config` (after `NewLocalExecutor`):

```go
	// Geteuid reports this process's effective uid; injectable for tests.
	// Nil selects os.Geteuid. Pairing this machine as root needs no sudo.
	Geteuid func() int
```

add to `Server`:

```go
	// goos is runtime.GOOS; a field so tests can stand in for another OS.
	goos    string
	geteuid func() int
```

and in `New`, beside `s.newLocalExecutor`:

```go
	s.goos = runtime.GOOS
	s.geteuid = cfg.Geteuid
	if s.geteuid == nil {
		s.geteuid = os.Geteuid
	}
```

In `internal/server/pair.go`:

```go
type pairRequest struct {
	Sudo bool `json:"sudo"`
	// Installed is set by `jumpgate hosts add --local` run as a non-root
	// user: the CLI ran the privileged steps in the foreground, where sudo
	// can prompt, and this server only verifies and records (spec D18).
	Installed string `json:"installed"`
}

const hintLocalPair = "run `jumpgate hosts add NAME --local` in a terminal (it asks for your sudo password there), or run jumpgate as root"
```

After `local := t.Mode == "local"` and the SSH-address check, insert:

```go
	var installed eip712.Address
	if req.Installed != "" {
		if !local {
			writeError(w, http.StatusBadRequest, `"installed" applies only to pairing this machine`)
			return
		}
		a, err := eip712.ParseAddress(req.Installed)
		if err != nil {
			writeError(w, http.StatusBadRequest, "installed: "+err.Error())
			return
		}
		installed = a
	}
	if local {
		if err := bootstrap.LocalSupported(s.goos); err != nil {
			writeErrorDetail(w, http.StatusBadRequest, err.Error(), "", "local_unsupported")
			return
		}
		if req.Installed == "" && s.geteuid() != 0 {
			writeErrorDetail(w, http.StatusConflict,
				"pairing this machine needs root, and the server has no terminal to ask for a sudo password on", hintLocalPair, "local_needs_terminal")
			return
		}
	}
```

Replace the executor selection:

```go
	var priv executor.Executor
	if local {
		priv = executor.Sudo(s.newLocalExecutor())
	} else {
```

with:

```go
	var priv executor.Executor
	switch {
	case local && req.Installed != "":
		// The CLI already ran the privileged steps; nothing runs as root here.
	case local:
		// Root (checked above): run the steps directly. A root controller on
		// stock Debian may have no sudo at all (B-4).
		priv = s.newLocalExecutor()
	default:
```

and close the former `else` block's `}` as the `default:` case's end (the SSH code inside is unchanged). Replace `defer priv.Close()` with:

```go
	if priv != nil {
		defer priv.Close()
	}
```

Replace the `addr, err := bootstrap.Run(…)` call and its error handling with:

```go
	var addr eip712.Address
	if req.Installed != "" {
		addr = installed
		send(pairEvent{Step: "install", Line: "done in the foreground by the CLI"})
	} else {
		addr, err = bootstrap.Run(ctx, bootstrap.Options{
			Exec: priv, Local: local, LocalUID: os.Getuid(),
			AgentBinary: agentBinaryReporting(func(line string) { send(pairEvent{Step: "upload", Line: line}) }),
			Controller:  s.cfg.Signer.Address(), ControllerLabel: controllerLabel(), TransportKey: transportKey, Wire: t.Wire,
			Event: func(step, line string) { send(pairEvent{Step: step, Line: line}) },
		})
		if err != nil {
			var se *bootstrap.StepError
			step := ""
			if errors.As(err, &se) {
				step = se.Step
			}
			send(pairEvent{Step: step, Err: err.Error(), Code: "step_failed"})
			return
		}
	}
```

Add imports `runtime` (server.go) and `eip712` (pair.go).

Run: `go test ./internal/server/`
Expected: PASS.

- [ ] **Step 5: Write the failing CLI tests**

```go
// cmd/jumpgate/pair_local_test.go
package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// localSeams installs a Linux non-root world with a terminal, records
// whether the daemon was started, and restores everything afterwards.
func localSeams(t *testing.T, goos string, euid int, terminal bool) *bool {
	t.Helper()
	testutil.Home(t)
	started := false
	oldGOOS, oldEuid, oldEnsure, oldTerm := hostGOOS, geteuid, ensureRunning, stdinIsTerminal
	hostGOOS, geteuid = goos, func() int { return euid }
	stdinIsTerminal = func() bool { return terminal }
	ensureRunning = func(context.Context, string) (daemon.Info, error) {
		started = true
		return daemon.Info{}, errors.New("test: no server")
	}
	t.Cleanup(func() { hostGOOS, geteuid, ensureRunning, stdinIsTerminal = oldGOOS, oldEuid, oldEnsure, oldTerm })
	return &started
}

func TestHostsAddLocalIsRefusedOffLinux(t *testing.T) {
	started := localSeams(t, "darwin", 501, true)
	if code := hostsAdd([]string{"me", "--local"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if *started {
		t.Fatal("the daemon was started before refusing --local")
	}
}

// Review Focus 3: no terminal means no sudo prompt is possible; fail at once.
func TestHostsAddLocalNonRootNeedsATerminal(t *testing.T) {
	started := localSeams(t, "linux", 1000, false)
	if code := hostsAdd([]string{"me", "--local"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if *started {
		t.Fatal("the daemon was started without a terminal to prompt on")
	}
}

type cmdRecorder struct {
	mu   sync.Mutex
	cmds []string
}

func (r *cmdRecorder) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmds = append(r.cmds, cmd)
	return executor.Result{}, nil
}
func (r *cmdRecorder) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (r *cmdRecorder) ReadFile(context.Context, string) ([]byte, error)          { return nil, nil }
func (r *cmdRecorder) Close() error                                                { return nil }

func withController(t *testing.T) eip712.Address {
	t.Helper()
	k, _ := signer.GenerateKey()
	if _, err := config.Update(func(c *config.Config) error {
		c.Controller = &config.Controller{KeyStore: "file", KeyRef: "x", Address: k.Address().Hex()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return k.Address()
}

func TestPairLocalForegroundRunsBootstrapThroughSudo(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	controller := withController(t)
	rec := &cmdRecorder{}
	prompted := false
	var got bootstrap.Options
	agent := eip712.Address{0x01}
	oldN, oldP, oldB, oldL := sudoNonInteractive, sudoPrompt, runBootstrap, newLocalExecutor
	sudoNonInteractive = func(context.Context) error { return nil }
	sudoPrompt = func(context.Context) error { prompted = true; return nil }
	runBootstrap = func(_ context.Context, o bootstrap.Options) (eip712.Address, error) { got = o; return agent, nil }
	newLocalExecutor = func() executor.Executor { return rec }
	t.Cleanup(func() { sudoNonInteractive, sudoPrompt, runBootstrap, newLocalExecutor = oldN, oldP, oldB, oldL })

	addr, code := pairLocalForeground(context.Background(), io.Discard)
	if code != 0 || addr != agent.Hex() {
		t.Fatalf("pairLocalForeground = %q, %d", addr, code)
	}
	if prompted {
		t.Fatal("prompted for a password although sudo -n worked")
	}
	if !got.Local || got.Controller != controller {
		t.Fatalf("options: Local=%v Controller=%s", got.Local, got.Controller.Hex())
	}
	_, _ = got.Exec.Run(context.Background(), "id -u", nil)
	if len(rec.cmds) != 1 || !strings.HasPrefix(rec.cmds[0], "sudo -n ") {
		t.Fatalf("commands %v, want one run through sudo -n", rec.cmds)
	}
}

func TestPairLocalForegroundStopsWhenSudoIsRefused(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	withController(t)
	ran := false
	oldN, oldP, oldB := sudoNonInteractive, sudoPrompt, runBootstrap
	sudoNonInteractive = func(context.Context) error { return errors.New("a password is required") }
	sudoPrompt = func(context.Context) error { return errors.New("user is not in the sudoers file") }
	runBootstrap = func(context.Context, bootstrap.Options) (eip712.Address, error) { ran = true; return eip712.Address{}, nil }
	t.Cleanup(func() { sudoNonInteractive, sudoPrompt, runBootstrap = oldN, oldP, oldB })
	if _, code := pairLocalForeground(context.Background(), io.Discard); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if ran {
		t.Fatal("bootstrap ran without root")
	}
}
```

Run: `go test ./cmd/jumpgate/ -run 'HostsAddLocal|PairLocalForeground'`
Expected: FAIL — `undefined: hostGOOS`, `pairLocalForeground`, …

- [ ] **Step 6: Implement the CLI side**

```go
// cmd/jumpgate/pair_local.go
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/valve-tech/jumpgate/internal/agentbin"
	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// Seams for tests; production never reassigns them.
var (
	hostGOOS           = runtime.GOOS
	geteuid            = os.Geteuid
	ensureRunning      = daemon.EnsureRunning
	sudoNonInteractive = func(ctx context.Context) error { return exec.CommandContext(ctx, "sudo", "-n", "true").Run() }
	sudoPrompt         = func(ctx context.Context) error {
		c := exec.CommandContext(ctx, "sudo", "-v")
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		return c.Run()
	}
	runBootstrap     = bootstrap.Run
	newLocalExecutor = executor.NewLocal
)

// pairBody is the pair endpoint's request.
type pairBody struct {
	Sudo      bool   `json:"sudo"`
	Installed string `json:"installed,omitempty"`
}

// pairLocalForeground runs the privileged pairing steps for this machine in
// this process, on the operator's terminal, so sudo can ask for a password
// (spec D5). The detached server never could: it has no terminal. sudo's
// cached credential covers the commands below because the local executor
// keeps this terminal session (Setpgid, never Setsid). It returns the new
// agent's address for the server to verify and record; the controller key
// itself never leaves the server.
func pairLocalForeground(ctx context.Context, out io.Writer) (string, int) {
	c, err := config.Load()
	if err != nil {
		return "", failed("load config: %v", err)
	}
	if c.Controller == nil {
		return "", reportServerError(os.Stderr, "pair", apiError{Error: "this controller has no signing key yet", Code: "no_controller_key"})
	}
	controller, err := eip712.ParseAddress(c.Controller.Address)
	if err != nil {
		return "", failed("controller address in config.json: %v", err)
	}
	if err := sudoNonInteractive(ctx); err != nil {
		fmt.Fprintln(out, "pairing this machine runs commands as root; sudo may ask for your password")
		if err := sudoPrompt(ctx); err != nil {
			return "", failed("sudo: %v. Pairing this machine needs root: ask an administrator for sudo rights, or run jumpgate as root", err)
		}
	}
	label, _ := os.Hostname()
	if label == "" {
		label = "controller"
	}
	addr, err := runBootstrap(ctx, bootstrap.Options{
		Exec: executor.Sudo(newLocalExecutor()), Local: true, LocalUID: os.Getuid(),
		AgentBinary: func(arch string) (string, error) {
			p, src, err := agentbin.Path(arch)
			if err == nil {
				fmt.Fprintf(out, "[upload] agent binary for linux/%s: %s\n", arch, src)
			}
			return p, err
		},
		Controller: controller, ControllerLabel: label,
		Event: func(step, line string) { fmt.Fprintf(out, "[%s] %s\n", step, line) },
	})
	if err != nil {
		return "", failed("pairing failed: %v", err)
	}
	return addr.Hex(), 0
}
```

In `cmd/jumpgate/cli_hosts.go` `hostsAdd`, directly after the flag-parse check, insert:

```go
	if *local {
		if err := bootstrap.LocalSupported(hostGOOS); err != nil {
			return failed("%v", err)
		}
		if geteuid() != 0 && !stdinIsTerminal() {
			return failed("pairing this machine as a non-root user runs sudo, which asks for your password on a terminal, and stdin is not one. Run `jumpgate hosts add %s --local` from an interactive shell, or as root", name)
		}
	}
```

replace `info, err := daemon.EnsureRunning(ctx, exe)` with `info, err := ensureRunning(ctx, exe)`, and replace the final `return streamPair(info, name, *sudo)` with:

```go
	if *local && geteuid() != 0 {
		addr, code := pairLocalForeground(ctx, os.Stdout)
		if code != 0 {
			return code
		}
		return streamPair(info, name, pairBody{Installed: addr})
	}
	return streamPair(info, name, pairBody{Sudo: *sudo})
```

Change `streamPair`'s signature and first line:

```go
func streamPair(info daemon.Info, name string, body pairBody) int {
	b, _ := json.Marshal(body)
```

Add the `bootstrap` import to `cli_hosts.go`.

In `cmd/jumpgate/cli_agent.go` `cmdAgent`, first lines:

```go
	// The agent runs on Linux boxes; macOS keeps it for its tests (spec D19).
	if hostGOOS == "windows" {
		return failed("`jumpgate agent` runs on the Linux boxes jumpgate manages, not on Windows")
	}
```

Run: `go test ./cmd/jumpgate/ ./internal/server/ ./internal/bootstrap/ && GOOS=windows go vet ./cmd/jumpgate/`
Expected: PASS; vet clean.

- [ ] **Step 7: Write the local-pairing e2e script**

First make the container flags overridable in `scripts/e2e-agent.sh`: replace

```bash
docker run -d --name "$name" --privileged -p 127.0.0.1::22 jumpgate-e2e >/dev/null
```

with

```bash
# shellcheck disable=SC2086 # E2E_DOCKER_FLAGS is a list of flags
docker run -d --name "$name" ${E2E_DOCKER_FLAGS:---privileged} -p 127.0.0.1::22 jumpgate-e2e >/dev/null
```

```bash
#!/usr/bin/env bash
# scripts/e2e-local.sh — pair the machine the controller runs on (B-4, D5):
#   (a) as root on a box without sudo, (b) as a NOPASSWD sudo user on a
#   terminal. Each must pair and then get a signed answer from the agent
#   (not_set_up, since no node is set up in the container). Needs Docker.
set -euo pipefail
base="${BASE:-debian:12}"
work="$(mktemp -d)"
name="jumpgate-e2e-local-$$"
trap 'docker rm -f "$name" >/dev/null 2>&1 || true; rm -rf "$work"' EXIT

arch="$(docker info --format '{{.Architecture}}')"
case "$arch" in x86_64|amd64) goarch=amd64 ;; aarch64|arm64) goarch=arm64 ;; *) echo "unsupported docker arch $arch" >&2; exit 1 ;; esac
# A static Linux controller uploads itself (agentbin's "this binary" source).
CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build -o "$work/jumpgate" ./cmd/jumpgate

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
# shellcheck disable=SC2086
docker run -d --name "$name" ${E2E_DOCKER_FLAGS:---privileged} jumpgate-e2e >/dev/null
for _ in $(seq 1 60); do
  state="$(docker exec "$name" systemctl is-system-running 2>/dev/null || true)"
  [ "$state" = running ] || [ "$state" = degraded ] && break
  sleep 1
done
docker cp "$work/jumpgate" "$name:/usr/local/bin/jumpgate"

expect_signed_answer() { # user home
  out="$(docker exec -u "$1" -e HOME="$2" "$name" jumpgate status me 2>&1 || true)"
  echo "$out"
  echo "$out" | grep -q 'no node is set up' || { echo "FAIL: no signed answer for $1" >&2; exit 1; }
}

echo "== (a) root, no sudo on the box"
docker exec "$name" sh -c 'mv /usr/bin/sudo /usr/bin/sudo.off'
docker exec -e HOME=/root "$name" sh -c 'jumpgate keys init --store file && jumpgate hosts add me --local'
docker exec -e HOME=/root "$name" jumpgate hosts list | grep -q 'agent 0x'
expect_signed_answer root /root
docker exec -e HOME=/root "$name" jumpgate stop
docker exec "$name" sh -c 'mv /usr/bin/sudo.off /usr/bin/sudo'

echo "== (b) NOPASSWD sudo user, on a terminal"
docker exec "$name" sh -c 'useradd -m op && echo "op ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/op && chmod 440 /etc/sudoers.d/op'
# -t gives the CLI a terminal on stdin, which non-root local pairing requires.
docker exec -t -u op -e HOME=/home/op "$name" sh -c 'jumpgate keys init --store file && jumpgate hosts add me --local'
docker exec -u op -e HOME=/home/op "$name" jumpgate hosts list | grep -q 'agent 0x'
expect_signed_answer op /home/op
docker exec -u op -e HOME=/home/op "$name" jumpgate stop

echo "e2e-local: OK"
```

Run: `chmod +x scripts/e2e-local.sh && scripts/e2e-local.sh`
Expected: `e2e-local: OK`. If case (a) fails with `sudo: not found`, the server still wraps root in sudo: recheck Step 4's `case local:` branch.

- [ ] **Step 8: Add the e2e job to CI**

Append to `.github/workflows/ci.yml` `jobs:`:

```yaml
  # Pairing against real systemd containers: over SSH (e2e-agent) and this
  # machine itself (e2e-local), on both supported distros.
  e2e:
    strategy:
      fail-fast: false
      matrix:
        base: ["debian:12", "ubuntu:24.04"]
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: pair over SSH
        env:
          BASE: ${{ matrix.base }}
        run: scripts/e2e-agent.sh
      - name: pair this machine
        env:
          BASE: ${{ matrix.base }}
        run: scripts/e2e-local.sh
```

If the container never reaches `running`/`degraded` on the runner, set `E2E_DOCKER_FLAGS: --privileged --cgroupns=host -v /sys/fs/cgroup:/sys/fs/cgroup:rw` in both steps' `env` (systemd under cgroup v2 on some runner kernels), and note it in the commit message.

- [ ] **Step 9: Commit**

```bash
git add internal/bootstrap internal/server cmd/jumpgate scripts/e2e-local.sh scripts/e2e-agent.sh .github/workflows/ci.yml
git commit -m "fix(pair): pair this machine as root without sudo, or from the terminal as a user; refuse --local off Linux

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: `e2e` green on both distros; `go (windows-latest)` runs `TestHostsAddLocalIsRefusedOffLinux` (the seam stands in for Windows' GOOS too).

Manual check, record the result in the PR description: on a Linux desktop as a normal user without NOPASSWD, `jumpgate hosts add me --local` asks for the sudo password once and pairs.

---
### Task 6: Windows daemon lifecycle and the server socket

Covers I-2 (stale socket), I-7, M-13. (M-5 landed with `fsperm.Rename` in Task 2.)

**Files:**
- Create: `internal/daemon/start_unix.go`, `internal/daemon/start_windows.go`, `internal/daemon/start_test.go`, `internal/daemon/start_windows_test.go`, `scripts/windows-smoke.ps1`
- Delete: `internal/daemon/detach_unix.go`, `internal/daemon/detach_windows.go`
- Modify: `internal/daemon/daemon.go` (`EnsureRunning`), `internal/server/server.go` (`ServeUnix`), `internal/server/unix_test.go`, `README.md`, `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `daemon.RunDir`, `testutil.Home`, `testutil.ShortTempDir`.
- Produces (unexported): `func startDetached(cmd *exec.Cmd) (*exec.Cmd, error)` per OS (returns the command actually started); `var startServer = startDetached` in `daemon.go`; Windows-only `var startProc func(*exec.Cmd) error`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/daemon/start_test.go
package daemon

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// I-7: the auto-started server must not hold the CLI's working directory
// open (on Windows that blocks deleting or renaming the folder, often the
// unzipped download). It runs from the run dir.
func TestEnsureRunningStartsTheServerInTheRunDir(t *testing.T) {
	testutil.Home(t)
	var got *exec.Cmd
	old := startServer
	startServer = func(cmd *exec.Cmd) (*exec.Cmd, error) { got = cmd; return nil, errors.New("test: not starting") }
	t.Cleanup(func() { startServer = old })

	if _, err := EnsureRunning(context.Background(), "/path/to/jumpgate"); err == nil {
		t.Fatal("EnsureRunning succeeded with a start that failed")
	}
	dir, _ := RunDir()
	if got == nil || got.Dir != dir {
		t.Fatalf("server cmd.Dir = %v, want %q", got, dir)
	}
	if !slices.Equal(got.Args[1:], []string{"serve", "--no-open"}) {
		t.Fatalf("server args %v", got.Args)
	}
}
```

```go
// internal/daemon/start_windows_test.go
//go:build windows

package daemon

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

// A job that forbids breakaway refuses CREATE_BREAKAWAY_FROM_JOB with
// ERROR_ACCESS_DENIED; the server then starts inside the job rather than not
// at all.
func TestStartDetachedRetriesWithoutBreakaway(t *testing.T) {
	var flags []uint32
	old := startProc
	startProc = func(cmd *exec.Cmd) error {
		flags = append(flags, cmd.SysProcAttr.CreationFlags)
		if len(flags) == 1 {
			return windows.ERROR_ACCESS_DENIED
		}
		return nil
	}
	t.Cleanup(func() { startProc = old })

	started, err := startDetached(exec.Command("jumpgate.exe", "serve", "--no-open"))
	if err != nil || started == nil {
		t.Fatalf("startDetached = %v, %v", started, err)
	}
	if len(flags) != 2 || flags[0]&createBreakawayFromJob == 0 || flags[1]&createBreakawayFromJob != 0 {
		t.Fatalf("creation flags %#x, want breakaway then none", flags)
	}
	for _, f := range flags {
		if f&detachedProcess == 0 || f&createNewProcessGroup == 0 {
			t.Fatalf("flags %#x lack DETACHED_PROCESS or CREATE_NEW_PROCESS_GROUP", f)
		}
	}
}
```

Append to `internal/server/unix_test.go`:

```go
// I-2: a server killed hard (Task Manager, a crash) leaves its socket file
// behind. The next server must replace it, on every OS, including Windows
// where the file is an AF_UNIX reparse point.
func TestServeUnixReplacesAStaleSocket(t *testing.T) {
	dir := testutil.ShortTempDir(t)
	sock := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	ln.Close()
	if _, err := os.Lstat(sock); err != nil {
		t.Fatalf("no stale socket left behind: %v", err)
	}

	token := NewSessionToken()
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- New(Config{Token: token, UI: fstest.MapFS{}}).ServeUnix(ctx, sock) }()
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
	var res *http.Response
	for i := 0; i < 50; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://x/api/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if res, err = client.Do(req); err == nil {
			res.Body.Close()
			break
		}
		select {
		case err := <-served:
			t.Fatalf("ServeUnix over a stale socket: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("health: %v", err)
	}
	cancel()
	<-served
}
```

Run: `go test ./internal/daemon/ -run RunDir ./internal/server/ -run StaleSocket`
Expected: FAIL — `undefined: startServer`. (`TestServeUnixReplacesAStaleSocket` passes on macOS already; it is the Windows guard.)

- [ ] **Step 2: Implement the per-OS start**

```go
// internal/daemon/start_unix.go
//go:build unix

package daemon

import (
	"os/exec"
	"syscall"
)

// startDetached starts the server in its own session, so closing the
// terminal that launched it does not deliver SIGHUP to it. A server that must
// also outlive an SSH logout under systemd-logind's KillUserProcesses=yes
// needs `loginctl enable-linger` (see README).
func startDetached(cmd *exec.Cmd) (*exec.Cmd, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd, cmd.Start()
}
```

```go
// internal/daemon/start_windows.go
//go:build windows

package daemon

import (
	"errors"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

const (
	detachedProcess        = 0x00000008
	createNewProcessGroup  = 0x00000200
	createBreakawayFromJob = 0x01000000
)

// startProc is a seam for the retry test.
var startProc = func(cmd *exec.Cmd) error { return cmd.Start() }

// startDetached starts the server with no console, in its own process group,
// and outside the caller's job object (I-7). An OpenSSH session, Windows
// Terminal, VS Code and CI all run the CLI inside a job that kills every
// member when it closes, which would take the server with it. A job that
// forbids breakaway refuses the start with ERROR_ACCESS_DENIED; the server
// then starts inside it, which is still better than not at all.
func startDetached(cmd *exec.Cmd) (*exec.Cmd, error) {
	base := uint32(detachedProcess | createNewProcessGroup)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: base | createBreakawayFromJob}
	err := startProc(cmd)
	if err == nil || !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return cmd, err
	}
	// An exec.Cmd cannot be started twice, so the retry is a fresh one.
	retry := exec.Command(cmd.Path, cmd.Args[1:]...)
	retry.Args = cmd.Args
	retry.Dir, retry.Env, retry.Stdout, retry.Stderr = cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr
	retry.SysProcAttr = &syscall.SysProcAttr{CreationFlags: base}
	return retry, startProc(retry)
}
```

In `internal/daemon/daemon.go` add `var startServer = startDetached // a seam for tests` and replace in `EnsureRunning`:

```go
	cmd := exec.Command(exe, "serve", "--no-open")
	cmd.Stdout, cmd.Stderr = logf, logf
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return Info{}, fmt.Errorf("daemon: start server: %w", err)
	}
	_ = cmd.Process.Release()
```

with:

```go
	cmd := exec.Command(exe, "serve", "--no-open")
	cmd.Stdout, cmd.Stderr = logf, logf
	// The server never pins the directory the CLI happened to start in.
	cmd.Dir = dir
	started, err := startServer(cmd)
	if err != nil {
		return Info{}, fmt.Errorf("daemon: start server: %w", err)
	}
	_ = started.Process.Release()
```

Delete `internal/daemon/detach_unix.go` and `internal/daemon/detach_windows.go`.

- [ ] **Step 3: Name the Windows minimum when the socket cannot listen**

In `internal/server/server.go` `ServeUnix`, replace

```go
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
```

with:

```go
	ln, err := net.Listen("unix", path)
	if err != nil {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("server: listen on %s: %w (the local socket needs Windows 10 version 1803 or Windows Server 2019 or later)", path, err)
		}
		return err
	}
```

Run: `go test ./internal/daemon/ ./internal/server/ && GOOS=windows go vet ./internal/daemon/ ./internal/server/`
Expected: PASS; vet clean.

- [ ] **Step 4: Write the Windows lifecycle smoke**

```powershell
# scripts/windows-smoke.ps1 — the controller's server lifecycle on real
# Windows (I-2, I-7): a CLI command auto-starts the server from inside a
# PowerShell job and the server outlives the job; a hard kill leaves a stale
# socket the next start recovers from; `jumpgate stop` ends it.
$ErrorActionPreference = 'Stop'
$root = Join-Path ($env:RUNNER_TEMP ?? $env:TEMP) 'jgsmoke'
Remove-Item -Recurse -Force $root -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $root | Out-Null
$env:USERPROFILE = $root
$env:HOME = $root
$exe = Join-Path $root 'jg.exe'
go build -o $exe ./cmd/jumpgate
if ($LASTEXITCODE) { throw 'build failed' }
$run = Join-Path $root '.jumpgate\run'

function Get-Server {
  $deadline = (Get-Date).AddSeconds(15)
  while ((Get-Date) -lt $deadline) {
    $f = Join-Path $run 'server.json'
    if (Test-Path $f) {
      $info = Get-Content $f -Raw | ConvertFrom-Json
      try {
        $h = Invoke-WebRequest -UseBasicParsing -Headers @{ Authorization = "Bearer $($info.token)" } "http://$($info.httpAddr)/api/health"
        if ($h.StatusCode -eq 200) { return $info }
      } catch { }
    }
    Start-Sleep -Milliseconds 300
  }
  Get-Content (Join-Path $run 'server.log') -ErrorAction SilentlyContinue
  throw 'no healthy server'
}

# 1. Auto-start from inside a job, then end the job.
$job = Start-Job -ScriptBlock { param($e) & $e status nosuch-target 2>&1 | Out-Null } -ArgumentList $exe
Wait-Job $job | Out-Null
Remove-Job $job
$info = Get-Server
Write-Host "server pid $($info.pid) outlived the job that started it"

# 2. Kill it hard; the stale socket must not block the next start.
Stop-Process -Id $info.pid -Force
Start-Sleep -Seconds 1
& $exe status nosuch-target 2>&1 | Out-Null
$info2 = Get-Server
if ($info2.pid -eq $info.pid) { throw 'the same pid answered after a kill' }

# 3. Stop it over the API.
& $exe stop
if ($LASTEXITCODE -ne 0) { throw "stop exited $LASTEXITCODE" }
Start-Sleep -Seconds 2
if (Get-Process -Id $info2.pid -ErrorAction SilentlyContinue) { throw 'the server is still running after stop' }
Write-Host 'windows-smoke: OK'
```

Append to `.github/workflows/ci.yml` `jobs:`:

```yaml
  windows-daemon:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: server lifecycle smoke
        shell: pwsh
        run: scripts/windows-smoke.ps1
```

- [ ] **Step 5: Document linger for Linux servers**

In `README.md`, in the section that describes `jumpgate serve` / the background server, add:

```markdown
**Linux over SSH:** the server jumpgate starts in the background survives closing the terminal. On distributions where systemd-logind kills a user's processes at logout (`KillUserProcesses=yes`), run `loginctl enable-linger $USER` once so it also survives the SSH session ending.

**Windows:** the background server needs Windows 10 version 1803 or Windows Server 2019 or later (for its local socket).
```

- [ ] **Step 6: Commit and verify on Windows**

```bash
git add internal/daemon internal/server scripts/windows-smoke.ps1 README.md .github/workflows/ci.yml
git commit -m "fix(daemon): start the server outside the caller's job and directory; recover a stale socket on Windows

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: `windows-daemon` prints `windows-smoke: OK`; `TestServeUnixReplacesAStaleSocket` and `TestStartDetachedRetriesWithoutBreakaway` pass on `go (windows-latest)`. If the stale-socket test fails on Windows with `exists and is not a socket`, this Go version does not report `ModeSocket` for the AF_UNIX reparse point: add `internal/server/socket_windows.go` with `func isSocket(path string, fi os.FileInfo) bool` that also returns true when `fi.Sys().(*syscall.Win32FileAttributeData).FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0` and the reparse tag read with `windows.FindFirstFile` (`Win32finddata.Reserved0`) equals `0x80000023` (`IO_REPARSE_TAG_AF_UNIX`), a unix twin returning `fi.Mode()&os.ModeSocket != 0`, and use it in `ServeUnix` in place of the mode check.

---

### Task 7: SSH agent over Windows named pipes; `~` in key paths; system known_hosts

Covers I-5, M-2, M-10.

**Files:**
- Create: `internal/executor/agentsock_unix.go`, `internal/executor/agentsock_windows.go`, `internal/executor/agentsock.go`, `internal/executor/agentsock_test.go`, `internal/executor/agentsock_windows_test.go`, `internal/executor/knownhosts_files.go`, `internal/executor/knownhosts_files_test.go`, `cmd/jumpgate/paths.go`, `cmd/jumpgate/paths_test.go`
- Modify: `internal/executor/dial.go:120-145` (`authMethods`), `internal/server/api.go:627`, `internal/server/controller.go` (`strictHostKey`), `cmd/jumpgate/cli_hosts.go` (`hostsAdd` key path, `strictCheck`), `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `hostGOOS` (Task 5), `testutil.ShortTempDir`, `testutil.Home`.
- Produces: `func executor.AgentAvailable() bool`; `func executor.OpenSSHKnownHosts(home string) []string`; unexported `func dialAgent(deadline time.Time) io.ReadWriteCloser` per OS, `var systemKnownHosts string`; `func expandHome(p string) (string, error)` in `cmd/jumpgate`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/executor/agentsock_test.go
package executor

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// serveTestAgent runs an in-memory ssh-agent on a unix socket (which Windows
// supports too, as MSYS and Cygwin agents use) and returns its public key.
func serveTestAgent(t *testing.T) ssh.PublicKey {
	t.Helper()
	sock := filepath.Join(testutil.ShortTempDir(t), "agent.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyring := agent.NewKeyring()
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); _ = agent.ServeAgent(keyring, c) }()
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", sock)
	pub, _ := ssh.NewPublicKey(priv.Public())
	return pub
}

func TestDialAgentReachesAUnixSocketAgent(t *testing.T) {
	pub := serveTestAgent(t)
	if !AgentAvailable() {
		t.Fatal("AgentAvailable() = false with a live agent")
	}
	c := dialAgent(time.Now().Add(2 * time.Second))
	if c == nil {
		t.Fatal("dialAgent returned nil")
	}
	defer c.Close()
	keys, err := agent.NewClient(c).List()
	if err != nil || len(keys) != 1 || string(keys[0].Blob) != string(pub.Marshal()) {
		t.Fatalf("List = %v, %v", keys, err)
	}
}

func TestAgentAvailableIsFalseForADeadSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(testutil.ShortTempDir(t), "missing.sock"))
	if AgentAvailable() {
		t.Fatal("AgentAvailable() = true for a socket that does not exist")
	}
}
```

```go
// internal/executor/knownhosts_files_test.go
package executor

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// M-10: the system-wide known_hosts counts as confirmed too, when present.
func TestOpenSSHKnownHostsIncludesTheSystemFileWhenPresent(t *testing.T) {
	home := t.TempDir()
	sys := filepath.Join(t.TempDir(), "ssh_known_hosts")
	old := systemKnownHosts
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts = old })

	user := filepath.Join(home, ".ssh", "known_hosts")
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user}) {
		t.Fatalf("without a system file: %v", got)
	}
	if err := os.WriteFile(sys, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user, sys}) {
		t.Fatalf("with a system file: %v", got)
	}
}
```

```go
// cmd/jumpgate/paths_test.go
package main

import (
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// M-2: cmd.exe and Windows PowerShell never expand ~, and bash does not for
// --key=~/x.
func TestExpandHome(t *testing.T) {
	home := testutil.Home(t)
	old := hostGOOS
	t.Cleanup(func() { hostGOOS = old })
	for _, c := range []struct{ goos, in, want string }{
		{"linux", "~/.ssh/id", filepath.Join(home, ".ssh", "id")},
		{"linux", "~", home},
		{"linux", "/abs/key", "/abs/key"},
		{"linux", "rel/~/key", "rel/~/key"},
		{"linux", "~other/key", "~other/key"},
		{"windows", `~\.ssh\id`, filepath.Join(home, ".ssh", "id")},
		{"darwin", `~\x`, `~\x`},
	} {
		hostGOOS = c.goos
		got, err := expandHome(c.in)
		if err != nil || got != c.want {
			t.Errorf("%s expandHome(%q) = %q, %v; want %q", c.goos, c.in, got, err, c.want)
		}
	}
}
```

Run: `go test ./internal/executor/ -run 'DialAgent|AgentAvailable|OpenSSHKnownHosts' ./cmd/jumpgate/ -run ExpandHome`
Expected: FAIL — undefined `AgentAvailable`, `dialAgent`, `systemKnownHosts`, `OpenSSHKnownHosts`, `expandHome`.

- [ ] **Step 2: Implement agent dialling per OS**

```go
// internal/executor/agentsock_unix.go
//go:build !windows

package executor

import (
	"io"
	"net"
	"os"
	"time"
)

// dialAgent connects to the ssh-agent named by SSH_AUTH_SOCK, or returns nil
// when none is configured or reachable: the agent is optional.
func dialAgent(deadline time.Time) io.ReadWriteCloser {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	c, err := (&net.Dialer{Deadline: deadline}).Dial("unix", sock)
	if err != nil {
		return nil
	}
	_ = c.SetDeadline(deadline) // a wedged agent cannot outlast the dial
	return c
}
```

```go
// internal/executor/agentsock_windows.go
//go:build windows

package executor

import (
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// openSSHAgentPipe is where Windows OpenSSH's agent service, and 1Password's
// agent, listen. They are named pipes, and SSH_AUTH_SOCK is normally unset.
const openSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

// dialAgent connects to the agent: SSH_AUTH_SOCK when it is set, otherwise
// the OpenSSH pipe. A \\.\pipe\ path is opened as a file, whose handle is
// the io.ReadWriter agent.NewClient needs; any other value is a unix socket
// (MSYS2, Cygwin). A pipe handle opened this way has no deadline, so a
// wedged agent is bounded only by the SSH handshake deadline around it.
func dialAgent(deadline time.Time) io.ReadWriteCloser {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		sock = openSSHAgentPipe
	}
	if !strings.HasPrefix(sock, `\\.\pipe\`) {
		c, err := (&net.Dialer{Deadline: deadline}).Dial("unix", sock)
		if err != nil {
			return nil
		}
		_ = c.SetDeadline(deadline)
		return c
	}
	f, err := os.OpenFile(sock, os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	return f
}
```

```go
// internal/executor/agentsock.go
package executor

import "time"

// AgentAvailable reports whether an ssh-agent answers here, so an SSH target
// with no key file can still authenticate. On Windows that includes the
// OpenSSH agent service's named pipe (I-5).
func AgentAvailable() bool {
	c := dialAgent(time.Now().Add(2 * time.Second))
	if c == nil {
		return false
	}
	c.Close()
	return true
}
```

In `internal/executor/dial.go` `authMethods`, change `var agentConn net.Conn` to `var agentConn io.ReadWriteCloser` and replace the `if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" { … }` block with:

```go
	if c := dialAgent(deadline); c != nil {
		agentConn = c
		agentClient = agent.NewClient(c)
	}
```

Add `"io"` to the imports and drop any now-unused ones.

In `internal/server/api.go` (~627) replace `(t.SSH.KeyPath == "" && os.Getenv("SSH_AUTH_SOCK") == "")` with `(t.SSH.KeyPath == "" && !executor.AgentAvailable())`.

- [ ] **Step 3: Implement the known_hosts list and `~` expansion**

```go
// internal/executor/knownhosts_files.go
package executor

import (
	"os"
	"path/filepath"
	"runtime"
)

// systemKnownHosts is OpenSSH's system-wide known_hosts on this OS; a var so
// tests can point it elsewhere.
var systemKnownHosts = defaultSystemKnownHosts()

func defaultSystemKnownHosts() string {
	if runtime.GOOS == "windows" {
		pd := os.Getenv("ProgramData")
		if pd == "" {
			pd = `C:\ProgramData`
		}
		return filepath.Join(pd, "ssh", "ssh_known_hosts")
	}
	return "/etc/ssh/ssh_known_hosts"
}

// OpenSSHKnownHosts lists the OpenSSH known_hosts files strict checking
// consults besides jumpgate's confirmed store: the user's, and the
// system-wide one when it exists (M-10).
func OpenSSHKnownHosts(home string) []string {
	files := []string{filepath.Join(home, ".ssh", "known_hosts")}
	if _, err := os.Stat(systemKnownHosts); err == nil {
		files = append(files, systemKnownHosts)
	}
	return files
}
```

In `internal/server/controller.go` `strictHostKey` and in `cmd/jumpgate/cli_hosts.go` `strictCheck`, replace

```go
	known := filepath.Join(home, ".ssh", "known_hosts")
	return executor.Strict(confirmed, known), executor.KnownHostKeyAlgorithms(confirmed, known), nil
```

with

```go
	known := executor.OpenSSHKnownHosts(home)
	return executor.Strict(confirmed, known...), executor.KnownHostKeyAlgorithms(confirmed, known...), nil
```

```go
// cmd/jumpgate/paths.go
package main

import (
	"os"
	"path/filepath"
	"strings"
)

// expandHome expands a leading ~ that the shell left alone: cmd.exe and
// Windows PowerShell never expand it, and bash does not in --key=~/x (M-2).
// Only "~" and "~/…" (and "~\…" on Windows) are expanded; "~user" is left as
// it is, since jumpgate cannot resolve other users' homes portably.
func expandHome(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") && !(hostGOOS == "windows" && strings.HasPrefix(p, `~\`)) {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, p[1:]), nil
}
```

In `hostsAdd`, inside `if keyPath != "" {`, before `filepath.Abs`:

```go
			expanded, err := expandHome(keyPath)
			if err != nil {
				return failed("--key: %v", err)
			}
			keyPath = expanded
```

Run: `go test ./internal/executor/ ./internal/server/ ./cmd/jumpgate/ && GOOS=windows go vet ./internal/executor/ ./cmd/jumpgate/`
Expected: PASS; vet clean.

- [ ] **Step 4: Write the Windows named-pipe test**

```go
// internal/executor/agentsock_windows_test.go
//go:build windows

package executor

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// I-5, end to end: a key loaded into the Windows OpenSSH agent service
// authenticates an SSH dial with no key file. CI starts the service and sets
// JUMPGATE_TEST_WIN_AGENT=1; elsewhere this skips.
func TestOpenSSHAgentPipe(t *testing.T) {
	if os.Getenv("JUMPGATE_TEST_WIN_AGENT") != "1" {
		t.Skip("set JUMPGATE_TEST_WIN_AGENT=1 with the ssh-agent service running")
	}
	t.Setenv("SSH_AUTH_SOCK", "")
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("ssh-add", keyFile).CombinedOutput(); err != nil {
		t.Fatalf("ssh-add: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("ssh-add", "-d", keyFile).Run() })
	sshPub, _ := ssh.NewPublicKey(pub)

	c := dialAgent(time.Now().Add(5 * time.Second))
	if c == nil {
		t.Fatal("dialAgent: no agent on the OpenSSH pipe")
	}
	keys, err := agent.NewClient(c).List()
	c.Close()
	found := false
	for _, k := range keys {
		found = found || string(k.Blob) == string(sshPub.Marshal())
	}
	if err != nil || !found {
		t.Fatalf("the added key is not listed: %v, %v", keys, err)
	}

	srv := &gliderssh.Server{
		PublicKeyHandler: func(_ gliderssh.Context, k gliderssh.PublicKey) bool { return gliderssh.KeysEqual(k, sshPub) },
		Handler:          func(s gliderssh.Session) { _, _ = io.WriteString(s, "ok"); _ = s.Exit(0) },
	}
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)
	srv.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	deadline := time.Now().Add(10 * time.Second)
	methods, release, err := authMethods(SSHConfig{}, deadline)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	client, err := ssh.Dial("tcp", ln.Addr().String(), &ssh.ClientConfig{
		User: "u", Auth: methods, HostKeyCallback: ssh.FixedHostKey(hostSigner.PublicKey()), Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial with the agent's key: %v", err)
	}
	defer client.Close()
	sess, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	out, err := sess.Output("anything")
	if err != nil || string(out) != "ok" {
		t.Fatalf("session = %q, %v", out, err)
	}
}
```

Append to `.github/workflows/ci.yml` `jobs:`:

```yaml
  windows-ssh-agent:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: start the OpenSSH agent service
        shell: pwsh
        run: |
          Set-Service ssh-agent -StartupType Manual
          Start-Service ssh-agent
      - name: dial through the named pipe
        shell: bash
        env:
          JUMPGATE_TEST_WIN_AGENT: "1"
        run: go test -count=1 -run TestOpenSSHAgentPipe -v ./internal/executor/
```

- [ ] **Step 5: Commit and verify**

```bash
git add internal/executor internal/server cmd/jumpgate .github/workflows/ci.yml
git commit -m "feat(executor): use the Windows OpenSSH agent pipe; expand ~ in --key; read the system known_hosts

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: `windows-ssh-agent` runs `TestOpenSSHAgentPipe` (not skipped) and passes; all other jobs green.

---

### Task 8: Browser handoff with a one-time login code; `jumpgate open`

Covers I-6, M-7, D4, D26.

**Files:**
- Create: `internal/server/login.go`, `internal/server/login_test.go`, `cmd/jumpgate/opener.go`, `cmd/jumpgate/opener_test.go`, `cmd/jumpgate/opener_linux_test.go`, `cmd/jumpgate/cli_open.go`
- Modify: `internal/server/server.go` (`Server` fields, `New`, `Handler`, `authMiddleware`), `cmd/jumpgate/main.go` (`runApp`, `openRunningServer`, remove old `openBrowser`), `cmd/jumpgate/cli.go` (`subcommands`)

**Interfaces:**
- Consumes: `ensureRunning`, `hostGOOS` (Task 5); `mustRequest`, `daemon.Info.Client()`.
- Produces (server): `func (s *Server) NewLoginCode() string`; routes `GET /login?code=…`, `POST /api/login-code` → `{"code": "…"}`; unexported `loginCodeTTL = 60 * time.Second`, `func (s *Server) redeemLoginCode(code string) bool`, `s.now func() time.Time`, `func sessionCookie(token string) *http.Cookie`.
- Produces (cmd/jumpgate): `func loginURL(addr, code string) string`, `func requestLoginCode(ctx context.Context, info daemon.Info) (string, error)`, `func openBrowser(url string) error`, `var startOpener func(name string, args ...string) error`, `func openerCommand(goos, url string) (string, []string)`, `func openWebApp(ctx context.Context, out io.Writer) error`, `func cmdOpen(args []string) int`.

- [ ] **Step 1: Write the failing server tests**

```go
// internal/server/login_test.go
package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func loginServer(t *testing.T) (*Server, *httptest.Server, string) {
	t.Helper()
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{"index.html": {Data: []byte("ui")}}})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return s, ts, token
}

func noRedirect() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// Review Focus 2: the first visit signs in; a second use of the same link is
// refused with a page naming `jumpgate open`; the cookie from the first stays
// valid.
func TestLoginCodeIsSingleUse(t *testing.T) {
	s, ts, token := loginServer(t)
	code := s.NewLoginCode()
	if code == token || len(code) != 32 {
		t.Fatalf("code %q: want 32 hex chars distinct from the token", code)
	}
	res, err := noRedirect().Get(ts.URL + "/login?code=" + code)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusFound || res.Header.Get("Location") != "/" {
		t.Fatalf("first use: %d %q, want 302 to /", res.StatusCode, res.Header.Get("Location"))
	}
	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == cookieName {
			cookie = c
		}
	}
	if cookie == nil || cookie.Value != token || !cookie.HttpOnly {
		t.Fatalf("session cookie %+v", cookie)
	}

	res, err = noRedirect().Get(ts.URL + "/login?code=" + code)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized || !strings.Contains(string(body), "jumpgate open") {
		t.Fatalf("second use: %d %q", res.StatusCode, body)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/health", nil)
	req.AddCookie(cookie)
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("cookie after a refused reuse: %v %v", res, err)
	}
	res.Body.Close()
}

func TestLoginCodeExpires(t *testing.T) {
	s, ts, _ := loginServer(t)
	now := time.Now()
	s.now = func() time.Time { return now }
	code := s.NewLoginCode()
	now = now.Add(loginCodeTTL + time.Second)
	res, err := noRedirect().Get(ts.URL + "/login?code=" + code)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expired code: %d, want 401", res.StatusCode)
	}
}

func TestUnknownLoginCodeIsRefused(t *testing.T) {
	_, ts, _ := loginServer(t)
	for _, q := range []string{"", "?code=", "?code=00000000000000000000000000000000"} {
		res, err := noRedirect().Get(ts.URL + "/login" + q)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("/login%s: %d, want 401", q, res.StatusCode)
		}
	}
}

// Another process (a second app launch, `jumpgate open`) mints codes over the
// authenticated API; nobody else can.
func TestLoginCodeMintNeedsTheToken(t *testing.T) {
	_, ts, token := loginServer(t)
	res, err := http.Post(ts.URL+"/api/login-code", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated mint: %d", res.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/login-code", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var body struct{ Code string }
	_ = json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || len(body.Code) != 32 {
		t.Fatalf("mint: %d %+v", res.StatusCode, body)
	}
	res, err = noRedirect().Get(ts.URL + "/login?code=" + body.Code)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("minted code: %d, want 302", res.StatusCode)
	}
}
```

Run: `go test ./internal/server/ -run LoginCode`
Expected: FAIL — `undefined: s.NewLoginCode`, `loginCodeTTL`, `s.now`.

- [ ] **Step 2: Implement login codes**

```go
// internal/server/login.go
package server

import (
	"io"
	"net/http"
	"sync"
	"time"
)

// loginCodeTTL bounds how long a one-time login link works (spec D4).
const loginCodeTTL = 60 * time.Second

// loginCodes are one-time, short-lived stand-ins for the session token in
// URLs handed to a browser opener. An opener's command line (xdg-open, open,
// rundll32, and the browser it starts) is readable by every local user
// through /proc or ps, so the long-lived token never goes there (I-6).
type loginCodes struct {
	mu sync.Mutex
	m  map[string]time.Time
}

// NewLoginCode mints a code that signs one browser in, once, within
// loginCodeTTL.
func (s *Server) NewLoginCode() string {
	code := NewSessionToken()
	now := s.now()
	s.codes.mu.Lock()
	defer s.codes.mu.Unlock()
	if s.codes.m == nil {
		s.codes.m = map[string]time.Time{}
	}
	for c, exp := range s.codes.m {
		if !now.Before(exp) {
			delete(s.codes.m, c)
		}
	}
	s.codes.m[code] = now.Add(loginCodeTTL)
	return code
}

// redeemLoginCode consumes code. It is true only for a code that exists and
// has not expired; either way the code is gone afterwards.
func (s *Server) redeemLoginCode(code string) bool {
	if code == "" {
		return false
	}
	s.codes.mu.Lock()
	defer s.codes.mu.Unlock()
	exp, ok := s.codes.m[code]
	delete(s.codes.m, code)
	return ok && s.now().Before(exp)
}

// handleLogin exchanges a login code for the session cookie.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.redeemLoginCode(r.URL.Query().Get("code")) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "This jumpgate login link has expired or was already used.\nRun `jumpgate open` for a new one.\n")
		return
	}
	http.SetCookie(w, sessionCookie(s.cfg.Token))
	http.Redirect(w, r, "/", http.StatusFound)
}

// sessionCookie carries the session token once a browser has signed in.
func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode}
}
```

In `internal/server/server.go`: add to `Server` the fields `codes loginCodes` and `now func() time.Time`; in `New` set `s.now = time.Now`. In `Handler`, after the `POST /api/shutdown` route:

```go
	mux.HandleFunc("POST /api/login-code", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"code": s.NewLoginCode()})
	})
```

In `authMiddleware`, as the first statement inside the handler func:

```go
		// A login link carries a one-time code, not the token; it is its own
		// authentication, so it is checked before the token is required.
		if r.URL.Path == "/login" && r.Method == http.MethodGet {
			s.handleLogin(w, r)
			return
		}
```

and in the `?token=` branch replace the inline `&http.Cookie{…}` with `sessionCookie(q)`.

Run: `go test ./internal/server/`
Expected: PASS.

- [ ] **Step 3: Write the failing opener tests**

```go
// cmd/jumpgate/opener_test.go
package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/server"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestOpenerCommandPerOS(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin":  "open http://x/login?code=c",
		"windows": "rundll32 url.dll,FileProtocolHandler http://x/login?code=c",
		"linux":   "xdg-open http://x/login?code=c",
	} {
		name, args := openerCommand(goos, "http://x/login?code=c")
		if got := strings.Join(append([]string{name}, args...), " "); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
}

// startUnixServer runs a real server on a unix socket and returns what
// server.json would say about it.
func startUnixServer(t *testing.T) daemon.Info {
	t.Helper()
	token := server.NewSessionToken()
	sock := filepath.Join(testutil.ShortTempDir(t), "s.sock")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = server.New(server.Config{Token: token, UI: fstest.MapFS{}}).ServeUnix(ctx, sock); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	info := daemon.Info{Socket: sock, Token: token, HTTPAddr: "127.0.0.1:8799"}
	for i := 0; i < 100; i++ {
		if _, err := requestLoginCode(context.Background(), info); err == nil {
			return info
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("server did not come up")
	return info
}

// I-6: what reaches the opener's argv is a one-time login link, never the
// session token.
func TestOpenWebAppNeverPutsTheTokenOnACommandLine(t *testing.T) {
	info := startUnixServer(t)
	var argv []string
	oldStart, oldEnsure := startOpener, ensureRunning
	startOpener = func(name string, args ...string) error { argv = append([]string{name}, args...); return nil }
	ensureRunning = func(context.Context, string) (daemon.Info, error) { return info, nil }
	t.Cleanup(func() { startOpener, ensureRunning = oldStart, oldEnsure })

	var out strings.Builder
	if err := openWebApp(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	line := strings.Join(argv, " ")
	if strings.Contains(line, info.Token) {
		t.Fatalf("the session token is on the opener's command line: %q", line)
	}
	if !strings.Contains(line, "http://127.0.0.1:8799/login?code=") {
		t.Fatalf("opener argv %q, want a login link", line)
	}
}

func TestOpenWebAppPrintsTheLinkWhenNoBrowserOpens(t *testing.T) {
	info := startUnixServer(t)
	oldStart, oldEnsure := startOpener, ensureRunning
	startOpener = func(string, ...string) error { return errors.New("exec: \"xdg-open\": executable file not found in $PATH") }
	ensureRunning = func(context.Context, string) (daemon.Info, error) { return info, nil }
	t.Cleanup(func() { startOpener, ensureRunning = oldStart, oldEnsure })
	var out strings.Builder
	if err := openWebApp(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "within 60 seconds") || !strings.Contains(out.String(), "/login?code=") {
		t.Fatalf("output %q, want the link and its lifetime", out.String())
	}
}
```

```go
// cmd/jumpgate/opener_linux_test.go
//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The real opener path on Linux: a stand-in xdg-open on PATH records the
// argv it was given, which must be exactly the login link.
func TestOpenBrowserHandsXdgOpenOnlyTheLink(t *testing.T) {
	dir := t.TempDir()
	rec := filepath.Join(dir, "argv")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + rec + "\n"
	if err := os.WriteFile(filepath.Join(dir, "xdg-open"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	old := hostGOOS
	hostGOOS = "linux"
	t.Cleanup(func() { hostGOOS = old })

	link := loginURL("127.0.0.1:8799", "0123456789abcdef0123456789abcdef")
	if err := openBrowser(link); err != nil {
		t.Fatal(err)
	}
	var got []byte
	for i := 0; i < 100 && len(got) == 0; i++ {
		got, _ = os.ReadFile(rec)
		time.Sleep(20 * time.Millisecond)
	}
	if strings.TrimSpace(string(got)) != link {
		t.Fatalf("xdg-open argv %q, want %q", got, link)
	}
}
```

Run: `go test ./cmd/jumpgate/ -run 'Opener|OpenWebApp|OpenBrowser'`
Expected: FAIL — undefined `openerCommand`, `requestLoginCode`, `startOpener`, `openWebApp`, `loginURL`.

- [ ] **Step 4: Implement the opener and `jumpgate open`**

```go
// cmd/jumpgate/opener.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/valve-tech/jumpgate/internal/daemon"
)

// startOpener runs a URL opener without waiting for it, and reaps it in the
// background so it never lingers as a zombie (M-7). A seam for tests.
var startOpener = func(name string, args ...string) error {
	c := exec.Command(name, args...)
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }()
	return nil
}

// openerCommand is the program and arguments that open url on goos.
func openerCommand(goos, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	}
	return "xdg-open", []string{url}
}

// openBrowser opens url in the default browser. url must be a one-time login
// link, never one carrying the session token: an opener's command line is
// readable by every local user (I-6).
func openBrowser(url string) error {
	name, args := openerCommand(hostGOOS, url)
	return startOpener(name, args...)
}

// loginURL is the one-time login link for a server listening on addr.
func loginURL(addr, code string) string { return "http://" + addr + "/login?code=" + code }

// requestLoginCode asks a running server, over its socket, for a login code.
func requestLoginCode(ctx context.Context, info daemon.Info) (string, error) {
	res, err := info.Client().Do(mustRequest(ctx, info, "/api/login-code", nil))
	if err != nil {
		return "", fmt.Errorf("ask the server for a login link: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ask the server for a login link: %s", res.Status)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&body); err != nil || body.Code == "" {
		return "", fmt.Errorf("ask the server for a login link: bad answer")
	}
	return body.Code, nil
}

// openWebApp starts the server if needed and opens the web app in the
// browser with a fresh login link. With no browser to open, it prints the
// link and how long it works.
func openWebApp(ctx context.Context, out io.Writer) error {
	exe, _ := os.Executable()
	info, err := ensureRunning(ctx, exe)
	if err != nil {
		return err
	}
	code, err := requestLoginCode(ctx, info)
	if err != nil {
		return err
	}
	link := loginURL(info.HTTPAddr, code)
	if err := openBrowser(link); err != nil {
		fmt.Fprintf(out, "could not open a browser (%v); open this link within 60 seconds:\n  %s\n", err, link)
		return nil
	}
	fmt.Fprintf(out, "opened the jumpgate web app (http://%s/) in your browser\n", info.HTTPAddr)
	return nil
}
```

```go
// cmd/jumpgate/cli_open.go
package main

import (
	"context"
	"os"
)

// cmdOpen is `jumpgate open`: start the server if needed and open the web
// app in the browser.
func cmdOpen(args []string) int {
	if len(args) != 0 {
		return usage("usage: jumpgate open")
	}
	if err := openWebApp(context.Background(), os.Stdout); err != nil {
		return failed("%v", err)
	}
	return 0
}
```

Add `"open": cmdOpen,` to the `subcommands` map in `cmd/jumpgate/cli.go`.

In `cmd/jumpgate/main.go`, delete the old `openBrowser` function (and the `os/exec` import if now unused). In `runApp`, replace

```go
	if !*noOpen {
		openBrowser(url)
	}
```

with

```go
	if !*noOpen {
		// Open once the listener accepts, with a one-time login link: the
		// token itself never reaches the opener's command line (spec D4).
		go func() {
			if waitReady(ctx, *bind) != nil {
				return
			}
			if err := openBrowser(loginURL(*bind, s.NewLoginCode())); err != nil {
				log.Printf("jumpgate: could not open a browser (%v); run `jumpgate open`", err)
			}
		}()
	}
```

In `openRunningServer`, replace

```go
	if !noOpen {
		openBrowser(url)
	}
```

with

```go
	if !noOpen {
		code, err := requestLoginCode(ctx, info)
		if err != nil {
			log.Printf("jumpgate: %v; run `jumpgate open`", err)
			return
		}
		if err := openBrowser(loginURL(info.HTTPAddr, code)); err != nil {
			log.Printf("jumpgate: could not open a browser (%v); run `jumpgate open`", err)
		}
	}
```

The tray window keeps `appURL(…)` with the token (spec D26: the webview runs in this process; no argv is involved).

Run: `go test ./cmd/jumpgate/ ./internal/server/ && scripts/test-linux.sh ./cmd/jumpgate/ -run OpenBrowser -v`
Expected: PASS on macOS; the Linux run executes `TestOpenBrowserHandsXdgOpenOnlyTheLink`.

- [ ] **Step 5: Commit**

```bash
git add internal/server cmd/jumpgate
git commit -m "fix(app): open the browser with a one-time login code, never the session token; add jumpgate open

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: all green.

---

### Task 9: The terminal entry point — `runTerminalHome`, the TUI seam

Covers the terminal half of D6, and D8, D9.

**Files:**
- Create: `cmd/jumpgate/home_terminal.go`, `cmd/jumpgate/home_terminal_test.go`, `cmd/jumpgate/help.go`, `cmd/jumpgate/help_test.go`, `cmd/jumpgate/console_windows.go`, `cmd/jumpgate/console_other.go`, `cmd/jumpgate/console_test.go`
- Modify: `cmd/jumpgate/main.go` (`main`, `isTerminalLaunch`, `stdoutIsTerminal`), `cmd/jumpgate/cli.go` (`subcommands`: `help`), `README.md`

**Interfaces:**
- Consumes: `openWebApp` (Task 8), `daemon.Find`, `config.Load`, `buildinfo.Version`, `stdinIsTerminal`, `inAppBundle`.
- Produces: `func runTerminalHome(ctx context.Context, in io.Reader, out io.Writer) int` (**the TUI seam, signature fixed by Global Constraints**); `func isTerminalLaunch(args []string) bool`; `var stdoutIsTerminal func() bool`; `func printHelp(w io.Writer)`; `func cmdHelp(args []string) int`; `func launchedStandalone() bool`; `func hasConsole() bool` (Windows: real; elsewhere `true`); `func pauseIfStandalone(in io.Reader, out io.Writer)`; `var termHome homeDeps` with fields `find func(context.Context) (daemon.Info, bool, error)`, `load func() (config.Config, error)`, `open func(context.Context, io.Writer) error`.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/jumpgate/home_terminal_test.go
package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

func fakeHome(t *testing.T, running bool) *int {
	t.Helper()
	opened := 0
	old := termHome
	termHome = homeDeps{
		find: func(context.Context) (daemon.Info, bool, error) {
			return daemon.Info{PID: 42, HTTPAddr: "127.0.0.1:8799"}, running, nil
		},
		load: func() (config.Config, error) {
			return config.Config{
				Controller: &config.Controller{KeyStore: "file", Address: "0xabc"},
				Targets:    []config.Target{{ID: "a", Agent: &config.AgentPairing{Address: "0x1"}}, {ID: "b"}},
			}, nil
		},
		open: func(context.Context, io.Writer) error { opened++; return nil },
	}
	t.Cleanup(func() { termHome = old })
	return &opened
}

func TestTerminalHomeShowsTheOverviewAndHelp(t *testing.T) {
	fakeHome(t, true)
	var out strings.Builder
	if code := runTerminalHome(context.Background(), strings.NewReader("h\nq\n"), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"running, pid 42", "0xabc (file)", "2 (1 paired)", "[o] open the web app", "commands", "hosts add"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestTerminalHomeOpensTheWebApp(t *testing.T) {
	opened := fakeHome(t, false)
	var out strings.Builder
	runTerminalHome(context.Background(), strings.NewReader("o\nq\n"), &out)
	if *opened != 1 {
		t.Fatalf("open called %d times, want 1", *opened)
	}
	if !strings.Contains(out.String(), "not running") {
		t.Errorf("overview should say the server is not running:\n%s", out.String())
	}
}

// Review Focus 5: input already at its end (a script, a closed terminal)
// prints the overview once and returns; it never spins.
func TestTerminalHomeEndsOnEOF(t *testing.T) {
	opened := fakeHome(t, true)
	var out strings.Builder
	if code := runTerminalHome(context.Background(), strings.NewReader(""), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if n := strings.Count(out.String(), "server:"); n != 1 {
		t.Fatalf("overview printed %d times, want 1", n)
	}
	if *opened != 0 {
		t.Fatal("opened the web app on EOF")
	}
}

func TestTerminalHomeStaysOpenUntilQuit(t *testing.T) {
	fakeHome(t, true)
	var out strings.Builder
	runTerminalHome(context.Background(), strings.NewReader("\n\nx\ns\nq\nh\n"), &out)
	if strings.Count(out.String(), "server:") != 2 {
		t.Fatalf("want the overview twice (start and s), got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "commands (run as") {
		t.Fatal("kept reading after q")
	}
	if !strings.Contains(out.String(), `unknown choice "x"`) {
		t.Fatal("an unknown choice was not reported")
	}
}

func TestIsTerminalLaunch(t *testing.T) {
	oldIn, oldOut := stdinIsTerminal, stdoutIsTerminal
	t.Cleanup(func() { stdinIsTerminal, stdoutIsTerminal = oldIn, oldOut })
	for _, c := range []struct {
		args    []string
		in, out bool
		want    bool
	}{
		{[]string{"jumpgate"}, true, true, true},
		{[]string{"jumpgate", "--bind", "x"}, true, true, false},
		{[]string{"jumpgate"}, false, true, false},
		{[]string{"jumpgate"}, true, false, false},
	} {
		stdinIsTerminal = func() bool { return c.in }
		stdoutIsTerminal = func() bool { return c.out }
		if got := isTerminalLaunch(c.args); got != c.want {
			t.Errorf("isTerminalLaunch(%v) in=%v out=%v = %v, want %v", c.args, c.in, c.out, got, c.want)
		}
	}
}
```

```go
// cmd/jumpgate/help_test.go
package main

import (
	"strings"
	"testing"
)

// Every subcommand a person runs on a controller is in the help. `agent`
// runs on the boxes and is left out on purpose.
func TestHelpListsEveryCommand(t *testing.T) {
	var b strings.Builder
	printHelp(&b)
	for name := range subcommands {
		if name == "agent" {
			continue
		}
		if !strings.Contains(b.String(), "\n  "+name) {
			t.Errorf("help does not list %q:\n%s", name, b.String())
		}
	}
}
```

```go
// cmd/jumpgate/console_test.go
package main

import "testing"

func TestLaunchedStandaloneHonoursTheLauncherVariable(t *testing.T) {
	t.Setenv("JUMPGATE_LAUNCHER", "1")
	if !launchedStandalone() {
		t.Fatal("JUMPGATE_LAUNCHER=1 not honoured")
	}
	// Under `go test` the console (if any) is shared with go and the shell,
	// so without the variable this process does not own its window.
	t.Setenv("JUMPGATE_LAUNCHER", "")
	if launchedStandalone() {
		t.Fatal("launchedStandalone() = true under go test")
	}
}
```

Run: `go test ./cmd/jumpgate/ -run 'TerminalHome|IsTerminalLaunch|HelpLists|LaunchedStandalone'`
Expected: FAIL — undefined `runTerminalHome`, `homeDeps`, `stdoutIsTerminal`, `printHelp`, `launchedStandalone`.

- [ ] **Step 2: Implement the help**

```go
// cmd/jumpgate/help.go
package main

import (
	"fmt"
	"io"
	"os"
)

// commandHelp is one line per command, in the order a new user needs them.
var commandHelp = []struct{ name, text string }{
	{"keys init", "create this controller's signing key"},
	{"keys show", "print this controller's address"},
	{"hosts add NAME --ssh USER@HOST", "pair a Linux box over SSH (--local pairs this Linux machine)"},
	{"hosts list", "list machines and their pairing"},
	{"status HOST", "sync, peers and disk of a paired box"},
	{"disk HOST", "disk usage of a paired box"},
	{"endpoints HOST", "RPC endpoints of a paired box"},
	{"firewall HOST", "firewall checklist of a paired box"},
	{"logs HOST [-n N]", "recent node logs"},
	{"service HOST exec|beacon start|stop|restart", "control a node service"},
	{"open", "open the web app in your browser"},
	{"serve", "run the server in the foreground"},
	{"stop", "stop the background server"},
	{"help", "this list"},
}

// printHelp prints the command list.
func printHelp(w io.Writer) {
	fmt.Fprintln(w, "commands (run as `jumpgate COMMAND`):")
	for _, c := range commandHelp {
		fmt.Fprintf(w, "  %-46s %s\n", c.name, c.text)
	}
}

func cmdHelp([]string) int {
	printHelp(os.Stdout)
	return 0
}
```

Add `"help": cmdHelp,` to `subcommands` in `cmd/jumpgate/cli.go`.

- [ ] **Step 3: Implement console detection**

```go
// cmd/jumpgate/console_other.go
//go:build !windows

package main

import "os"

// launchedStandalone reports whether this process owns its terminal window
// and should wait before closing it on an error. Off Windows only the
// launchers know, and they say so with JUMPGATE_LAUNCHER=1.
func launchedStandalone() bool { return os.Getenv("JUMPGATE_LAUNCHER") == "1" }

// hasConsole is always true off Windows: there is no GUI subsystem there.
func hasConsole() bool { return true }
```

```go
// cmd/jumpgate/console_windows.go
//go:build windows

package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleWindow      = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
)

// hasConsole reports whether this process has a console window. It is false
// for jumpgate-tray.exe (GUI subsystem) and for a server started detached.
func hasConsole() bool {
	h, _, _ := procGetConsoleWindow.Call()
	return h != 0
}

// launchedStandalone reports whether this process owns its console window:
// double-clicked in Explorer, the console holds only this process and closes
// the moment it exits, so an error would flash and vanish.
func launchedStandalone() bool {
	if os.Getenv("JUMPGATE_LAUNCHER") == "1" {
		return true
	}
	var pids [2]uint32
	n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return n == 1
}
```

- [ ] **Step 4: Implement the terminal home**

```go
// cmd/jumpgate/home_terminal.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

// homeDeps is what the terminal home reads and does; a seam for tests.
type homeDeps struct {
	find func(context.Context) (daemon.Info, bool, error)
	load func() (config.Config, error)
	open func(context.Context, io.Writer) error
}

var termHome = homeDeps{find: daemon.Find, load: config.Load, open: openWebApp}

// stdoutIsTerminal reports whether stdout is a character device. A variable
// so tests can stand in for a terminal.
var stdoutIsTerminal = func() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// isTerminalLaunch reports whether this is a bare `jumpgate` in a terminal,
// which opens the terminal home (spec D9). The macOS .app, the Windows GUI
// exe and a service have no terminal and still run the app.
func isTerminalLaunch(args []string) bool {
	return len(args) == 1 && stdinIsTerminal() && stdoutIsTerminal() && !inAppBundle()
}

// runTerminalHome is what a person sees when they start `jumpgate` with no
// arguments in a terminal, and what every launcher runs: the Windows console
// exe, Jumpgate Terminal.app, the Linux .desktop entry. It must not depend on
// how it was started.
//
// TUI SEAM: the TUI (sub-project 3) replaces this function's body and keeps
// its signature; main, the launchers and CI stay as they are. Until then it
// is a line-based menu (spec D8) that loops until q or end of input, so a
// double-clicked window does not flash and close.
func runTerminalHome(ctx context.Context, in io.Reader, out io.Writer) int {
	r := bufio.NewReader(in)
	printOverview(ctx, out)
	for {
		fmt.Fprint(out, "\n[o] open the web app   [s] status   [h] commands   [q] quit\n> ")
		line, err := r.ReadString('\n')
		choice := strings.ToLower(strings.TrimSpace(line))
		switch choice {
		case "o":
			if oerr := termHome.open(ctx, out); oerr != nil {
				fmt.Fprintf(out, "could not open the web app: %v\n", oerr)
			}
		case "s":
			printOverview(ctx, out)
		case "h", "?":
			printHelp(out)
		case "q", "quit", "exit":
			return 0
		case "":
		default:
			fmt.Fprintf(out, "unknown choice %q\n", choice)
		}
		if err != nil {
			fmt.Fprintln(out)
			return 0 // end of input
		}
	}
}

// printOverview is the home's status block: what is running, which key signs,
// and how many machines there are.
func printOverview(ctx context.Context, out io.Writer) {
	fmt.Fprintf(out, "jumpgate %s\n", buildinfo.Version())
	switch info, ok, err := termHome.find(ctx); {
	case err != nil:
		fmt.Fprintf(out, "  server:   unknown (%v)\n", err)
	case ok:
		fmt.Fprintf(out, "  server:   running, pid %d, http://%s/\n", info.PID, info.HTTPAddr)
	default:
		fmt.Fprintln(out, "  server:   not running (it starts when a command needs it)")
	}
	c, err := termHome.load()
	if err != nil {
		fmt.Fprintf(out, "  config:   %v\n", err)
		return
	}
	if c.Controller == nil {
		fmt.Fprintln(out, "  key:      none yet; run `jumpgate keys init`")
	} else {
		fmt.Fprintf(out, "  key:      %s (%s)\n", c.Controller.Address, c.Controller.KeyStore)
	}
	paired := 0
	for _, t := range c.Targets {
		if t.Agent != nil {
			paired++
		}
	}
	fmt.Fprintf(out, "  machines: %d (%d paired); add one with `jumpgate hosts add NAME --ssh USER@HOST`\n", len(c.Targets), paired)
}

// pauseIfStandalone waits for Enter when this process owns its window, so an
// error printed before exit can be read.
func pauseIfStandalone(in io.Reader, out io.Writer) {
	if !launchedStandalone() {
		return
	}
	fmt.Fprint(out, "press Enter to close this window")
	_, _ = bufio.NewReader(in).ReadString('\n')
}
```

- [ ] **Step 5: Wire `main`**

Replace `main` in `cmd/jumpgate/cli.go`:

```go
func main() {
	if err := migrateOnStartup(os.Args, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "jumpgate:", err)
		pauseIfStandalone(os.Stdin, os.Stderr)
		os.Exit(1)
	}
	if code, handled := dispatch(os.Args, os.Stderr); handled {
		os.Exit(code)
	}
	if isTerminalLaunch(os.Args) {
		os.Exit(runTerminalHome(context.Background(), os.Stdin, os.Stdout))
	}
	runApp()
}
```

Add `"context"` to `cli.go`'s imports, and update `dispatch`'s unknown-command message tail to `(run `jumpgate help` for the commands; `jumpgate serve` runs the server in the foreground)`.

Run: `go test ./cmd/jumpgate/ && GOOS=windows go vet ./cmd/jumpgate/ && go build -o /tmp/jg ./cmd/jumpgate && /tmp/jg help`
Expected: PASS; vet clean; the help prints.

- [ ] **Step 6: Try it in a real terminal on macOS**

Run: `/tmp/jg` in a terminal, then type `s`, Enter, `q`, Enter.
Expected: the overview, the menu, a refreshed overview, then exit. Then `printf 'q\n' | /tmp/jg` must start the app (stdin is a pipe), not the home: stop it with Ctrl-C.

- [ ] **Step 7: Document and commit**

In `README.md`, where running the app is described, add:

```markdown
Run `jumpgate` with no arguments in a terminal for the terminal home: status, a menu, and the command list (`jumpgate help`). `jumpgate open` opens the web app in your browser, starting the background server if needed; `jumpgate serve` runs the server in the foreground. The desktop launchers on each OS open a terminal running `jumpgate`.
```

```bash
git add cmd/jumpgate README.md
git commit -m "feat(cli): terminal home for bare jumpgate in a terminal; jumpgate help

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: all green.

---
### Task 10: Windows desktop build — console exe, GUI exe with a notification icon, visible errors

Covers B-3 (Windows), I-8 (Windows), I-11 (Windows), the Windows half of D6, D20, D21.

**Files:**
- Create: `cmd/jumpgate/gui.go`, `cmd/jumpgate/gui_windows.go`, `cmd/jumpgate/gui_other.go`, `cmd/jumpgate/gui_test.go`, `cmd/jumpgate/statusitem_windows.go`, `cmd/jumpgate/statusitem_windows_test.go`, `cmd/jumpgate/trayicon_windows.png`, `scripts/package-windows.sh`, `packaging/windows/README.txt`, `scripts/windows-desktop-smoke.ps1`
- Modify: `cmd/jumpgate/main.go` (`runApp`, `openRunningServer`), `cmd/jumpgate/cli.go` (`main`), `cmd/jumpgate/tray.go` (`runWindow`), `cmd/jumpgate/statusitem_other.go` (build tag, `removeStatusItem`), `cmd/jumpgate/statusitem_darwin.go` (`removeStatusItem`), `cmd/jumpgate/trayhealth.go`, `cmd/jumpgate/trayhealth_test.go`, `.gitignore`, `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `hasConsole` (Task 9), `hostGOOS` (Task 5), `daemon.RunDir`, `fsperm.MakePrivate`, `trayBuilt`, `healthKind`.
- Produces: `var fatalf func(format string, args ...any)`; `var requestQuit func()`; `func guiLaunch(args []string) bool`; `func guiSubcommandRefused(args []string) bool`; `func setupGUILogging()`; `func showErrorDialog(msg string)`; `func webview2Installed() bool`; `func removeStatusItem()` (every tray build); `func healthTooltip(k healthKind) string`; `type trayAction int` with `trayNone`, `trayOpen`, `trayQuit`; `func trayActionFor(menuID int) trayAction`.
- Produces: `scripts/package-windows.sh [OUT_DIR]` → `OUT_DIR/jumpgate-windows-amd64.zip` and the staged `OUT_DIR/jumpgate-windows-amd64/`.

- [ ] **Step 1: Write the failing portable tests**

```go
// cmd/jumpgate/gui_test.go
package main

import "testing"

// D20: the GUI exe refuses CLI subcommands (it has no console to print to),
// except serve and stop, which need none.
func TestGUISubcommandRefused(t *testing.T) {
	if hostGOOS != "windows" || !trayBuilt || hasConsole() {
		for _, args := range [][]string{{"jumpgate", "hosts", "list"}, {"jumpgate"}} {
			if guiSubcommandRefused(args) {
				t.Fatalf("refused %v outside a console-less Windows tray build", args)
			}
		}
		return
	}
	cases := map[string]bool{"hosts": true, "status": true, "serve": false, "stop": false, "--tray": false}
	for arg, want := range cases {
		if got := guiSubcommandRefused([]string{"jumpgate-tray.exe", arg}); got != want {
			t.Errorf("%s: %v, want %v", arg, got, want)
		}
	}
}

func TestGUILaunchNeedsATrayBuildAndNoArguments(t *testing.T) {
	if guiLaunch([]string{"jumpgate", "--bind", "x"}) {
		t.Fatal("guiLaunch with arguments")
	}
	if !trayBuilt && guiLaunch([]string{"jumpgate"}) {
		t.Fatal("guiLaunch in a build without the tray tag")
	}
}
```

Append to `cmd/jumpgate/trayhealth_test.go`:

```go
func TestHealthTooltip(t *testing.T) {
	for k, want := range map[healthKind]string{
		healthOff: "Jumpgate — idle", healthOK: "Jumpgate — serving",
		healthWarn: "Jumpgate — degraded", healthDown: "Jumpgate — a gateway is unavailable",
	} {
		if got := healthTooltip(k); got != want {
			t.Errorf("healthTooltip(%d) = %q, want %q", k, got, want)
		}
	}
}

func TestTrayActionFor(t *testing.T) {
	if trayActionFor(1) != trayOpen || trayActionFor(2) != trayQuit || trayActionFor(0) != trayNone || trayActionFor(99) != trayNone {
		t.Fatal("tray menu ids map wrongly")
	}
}
```

Run: `go test ./cmd/jumpgate/ -run 'GUI|HealthTooltip|TrayActionFor'`
Expected: FAIL — undefined `guiSubcommandRefused`, `guiLaunch`, `healthTooltip`, `trayActionFor`.

- [ ] **Step 2: Implement the portable GUI pieces**

Append to `cmd/jumpgate/trayhealth.go`:

```go
// healthTooltip is the status item's hover text; the same words as the macOS
// menubar's tooltips (statusitem_darwin.go).
func healthTooltip(k healthKind) string {
	switch k {
	case healthOK:
		return "Jumpgate — serving"
	case healthWarn:
		return "Jumpgate — degraded"
	case healthDown:
		return "Jumpgate — a gateway is unavailable"
	}
	return "Jumpgate — idle"
}

// trayAction is what a tray menu choice does. The menu ids are fixed here so
// the mapping is tested without a tray build.
type trayAction int

const (
	trayNone trayAction = iota
	trayOpen
	trayQuit
)

func trayActionFor(menuID int) trayAction {
	switch menuID {
	case 1:
		return trayOpen
	case 2:
		return trayQuit
	}
	return trayNone
}
```

```go
// cmd/jumpgate/gui.go
package main

import (
	"log"
	"strings"
)

// fatalf reports an error that stops the app. It is log.Fatalf, except in the
// Windows GUI build, which has no console: setupGUILogging replaces it with
// one that also writes app.log and shows a dialog (B-3).
var fatalf = log.Fatalf

// requestQuit stops the app from a tray menu. runApp sets it to the cancel
// of its context; closing the window or Ctrl-C end the app the same way.
var requestQuit = func() {}

// guiSubcommandRefused reports a CLI subcommand given to the Windows GUI exe,
// which has no console to print to or read from (spec D20). serve and stop
// need none.
func guiSubcommandRefused(args []string) bool {
	if !trayBuilt || hostGOOS != "windows" || hasConsole() || len(args) < 2 || strings.HasPrefix(args[1], "-") {
		return false
	}
	return args[1] != "serve" && args[1] != "stop"
}
```

```go
// cmd/jumpgate/gui_other.go
//go:build !windows

package main

// guiLaunch is Windows-only: elsewhere the window is chosen by --tray or the
// macOS bundle.
func guiLaunch([]string) bool { return false }

func setupGUILogging()       {}
func showErrorDialog(string) {}

// webview2Installed is a Windows question; other platforms bring their own
// web engine with the build.
func webview2Installed() bool { return true }
```

```go
// cmd/jumpgate/gui_windows.go
//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// guiLaunch reports a double-click of jumpgate-tray.exe: a tray build, no
// arguments, no console. It opens the desktop window instead of a browser
// tab and an invisible server (B-3).
func guiLaunch(args []string) bool { return trayBuilt && len(args) == 1 && !hasConsole() }

// setupGUILogging sends log output to ~/.jumpgate/run/app.log and makes
// fatalf show a dialog too: under the GUI subsystem stdout and stderr go
// nowhere, so startup failures used to be silent.
func setupGUILogging() {
	dir, err := daemon.RunDir()
	if err != nil {
		showErrorDialog("jumpgate: " + err.Error())
		return
	}
	path := filepath.Join(dir, "app.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		showErrorDialog("jumpgate: " + err.Error())
		return
	}
	_ = fsperm.MakePrivate(path)
	log.SetOutput(f)
	fatalf = func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		log.Print(msg)
		showErrorDialog(msg + "\n\nDetails are in " + path)
		os.Exit(1)
	}
}

// showErrorDialog shows msg in a modal error box.
func showErrorDialog(msg string) {
	text, _ := windows.UTF16PtrFromString(msg)
	title, _ := windows.UTF16PtrFromString("Jumpgate")
	_, _ = windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
}

// webview2Installed reports whether the Microsoft Edge WebView2 Runtime is
// present (machine-wide or per user), which the desktop window needs. The
// keys and the "pv" value are the ones Microsoft documents for detection.
func webview2Installed() bool {
	const client = `Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
	for _, k := range []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\` + client},
		{registry.LOCAL_MACHINE, `SOFTWARE\` + client},
		{registry.CURRENT_USER, `SOFTWARE\` + client},
	} {
		key, err := registry.OpenKey(k.root, k.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		v, _, err := key.GetStringValue("pv")
		key.Close()
		if err == nil && v != "" && v != "0.0.0.0" {
			return true
		}
	}
	return false
}
```

Run: `go test ./cmd/jumpgate/ && GOOS=windows go vet ./cmd/jumpgate/`
Expected: PASS; vet clean.

- [ ] **Step 3: Use them in `main` and `runApp`**

In `main` (`cmd/jumpgate/cli.go`), first statement:

```go
	if guiSubcommandRefused(os.Args) {
		showErrorDialog("jumpgate-tray.exe is the desktop app and has no console. Run commands with jumpgate.exe in a terminal, for example:\n\n    jumpgate.exe " + strings.Join(os.Args[1:], " "))
		os.Exit(exitCode("usage"))
	}
```

In `runApp` (`cmd/jumpgate/main.go`):
- directly after `flag.Parse()` and the `--version` early return, add:

```go
	gui := guiLaunch(os.Args)
	if gui {
		setupGUILogging()
	}
```

- change `windowed := *tray || inAppBundle()` to `windowed := *tray || inAppBundle() || gui`;
- after `ctx, stop := shutdownContext(…)` add `requestQuit = stop`;
- replace every `log.Fatalf(` in `runApp` and `openRunningServer` with `fatalf(`;
- in the `if windowed {` block, immediately before `runWindow(ctx, url)`, add `log.Print("jumpgate: opening the desktop window")` (the Windows smoke checks app.log for it).

In `cmd/jumpgate/tray.go` `runWindow`, first lines:

```go
	if !webview2Installed() {
		fatalf("jumpgate: the desktop window needs the Microsoft Edge WebView2 Runtime. Install it from https://go.microsoft.com/fwlink/p/?LinkId=2124703, or run `jumpgate.exe open` to use your browser")
	}
```

and right after `installStatusItem(w.Window())` add `defer removeStatusItem()`.

In `cmd/jumpgate/statusitem_other.go`, change the build line to `//go:build tray && !darwin && !windows` and add `func removeStatusItem() {}`. In `cmd/jumpgate/statusitem_darwin.go` add:

```go
// removeStatusItem is a no-op on macOS: the status item goes away with NSApp.
func removeStatusItem() {}
```

Run: `go build ./cmd/jumpgate && go vet -tags tray ./cmd/jumpgate && GOOS=windows go vet ./cmd/jumpgate`
Expected: clean (the macOS tray build still compiles).

- [ ] **Step 4: Implement the Windows notification icon**

Render the icon (macOS: `brew install librsvg` if needed):

```bash
rsvg-convert -w 32 -h 32 cmd/jumpgate/icon.svg -o cmd/jumpgate/trayicon_windows.png
```

and add to `.gitignore` after the `*.png` line: `!/cmd/jumpgate/trayicon_windows.png`.

```go
// cmd/jumpgate/statusitem_windows.go
//go:build tray && windows

// The Windows notification-area icon: left click shows the window, right
// click offers Open and Quit, and the tooltip shows gateway health. It runs
// its own hidden window and message loop on a dedicated OS thread, so the
// webview's loop on the main thread is untouched. Pure syscalls through
// x/sys/windows; no new dependency (spec D21).
package main

import (
	_ "embed"
	"fmt"
	"log"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed trayicon_windows.png
var trayIconPNG []byte

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	shell32                      = windows.NewLazySystemDLL("shell32.dll")
	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessageW         = user32.NewProc("DispatchMessageW")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	procAppendMenuW              = user32.NewProc("AppendMenuW")
	procTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	procDestroyMenu              = user32.NewProc("DestroyMenu")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
	procRegisterWindowMessageW   = user32.NewProc("RegisterWindowMessageW")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procLoadIconW                = user32.NewProc("LoadIconW")
	procShellNotifyIconW         = shell32.NewProc("Shell_NotifyIconW")
)

const (
	wmClose        = 0x0010
	wmDestroy      = 0x0002
	wmContextMenu  = 0x007B
	wmLButtonUp    = 0x0202
	wmRButtonUp    = 0x0205
	wmApp          = 0x8000
	wmTrayCallback = wmApp + 1
	nimAdd         = 0
	nimModify      = 1
	nimDelete      = 2
	nifMessage     = 0x1
	nifIcon        = 0x2
	nifTip         = 0x4
	mfString       = 0x0
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
	swRestore      = 9
	idiApplication = 32512
)

// notifyIconData is NOTIFYICONDATAW.
type notifyIconData struct {
	CbSize           uint32
	HWnd             windows.HWND
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     windows.Handle
}

// wndClassEx is WNDCLASSEXW.
type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type point struct{ X, Y int32 }

// msg is MSG.
type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

var trayIcon struct {
	mu             sync.Mutex
	hwnd           uintptr // our hidden window
	main           uintptr // the webview window
	data           notifyIconData
	added          bool
	taskbarCreated uint32
}

// installStatusItem adds the notification-area icon for the webview window.
func installStatusItem(win unsafe.Pointer) {
	trayIcon.mu.Lock()
	trayIcon.main = uintptr(win)
	trayIcon.mu.Unlock()
	ready := make(chan struct{})
	go runTrayLoop(ready)
	<-ready
}

func runTrayLoop(ready chan<- struct{}) {
	runtime.LockOSThread()
	hwnd, err := createTrayWindow()
	if err != nil {
		log.Printf("jumpgate: tray icon: %v", err)
		close(ready)
		return
	}
	trayIcon.mu.Lock()
	trayIcon.hwnd = hwnd
	trayIcon.mu.Unlock()
	addTrayIcon(hwnd)
	close(ready)
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// createTrayWindow makes a hidden top-level window to receive the icon's
// messages. Not a message-only window: those miss the TaskbarCreated
// broadcast that says Explorer restarted and the icon must be re-added.
func createTrayWindow() (uintptr, error) {
	className, _ := windows.UTF16PtrFromString("JumpgateTray")
	var inst windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &inst); err != nil {
		return 0, err
	}
	wc := wndClassEx{LpfnWndProc: windows.NewCallback(trayWndProc), HInstance: inst, LpszClassName: className}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return 0, fmt.Errorf("RegisterClassExW: %w", err)
	}
	title, _ := windows.UTF16PtrFromString("Jumpgate")
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		0, 0, 0, 0, 0, 0, 0, uintptr(inst), 0)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %w", err)
	}
	tb, _ := windows.UTF16PtrFromString("TaskbarCreated")
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(tb)))
	trayIcon.mu.Lock()
	trayIcon.taskbarCreated = uint32(r)
	trayIcon.mu.Unlock()
	return hwnd, nil
}

// loadTrayIcon builds the icon from the embedded PNG (Vista and later accept
// PNG icon data), falling back to the stock application icon.
func loadTrayIcon() windows.Handle {
	if len(trayIconPNG) > 0 {
		h, _, _ := procCreateIconFromResourceEx.Call(uintptr(unsafe.Pointer(&trayIconPNG[0])), uintptr(len(trayIconPNG)), 1, 0x00030000, 32, 32, 0)
		if h != 0 {
			return windows.Handle(h)
		}
	}
	h, _, _ := procLoadIconW.Call(0, idiApplication)
	return windows.Handle(h)
}

func setTip(d *notifyIconData, s string) {
	u, _ := windows.UTF16FromString(s)
	n := copy(d.SzTip[:len(d.SzTip)-1], u)
	d.SzTip[n] = 0
}

func addTrayIcon(hwnd uintptr) {
	trayIcon.mu.Lock()
	defer trayIcon.mu.Unlock()
	d := &trayIcon.data
	d.CbSize = uint32(unsafe.Sizeof(*d))
	d.HWnd = windows.HWND(hwnd)
	d.UID = 1
	d.UFlags = nifMessage | nifIcon | nifTip
	d.UCallbackMessage = wmTrayCallback
	if d.HIcon == 0 {
		d.HIcon = loadTrayIcon()
	}
	if d.SzTip[0] == 0 {
		setTip(d, healthTooltip(healthOff))
	}
	r, _, _ := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(d)))
	trayIcon.added = r != 0
}

// setHealth updates the tooltip. Shell_NotifyIconW may be called from any
// thread.
func setHealth(k healthKind) {
	trayIcon.mu.Lock()
	defer trayIcon.mu.Unlock()
	setTip(&trayIcon.data, healthTooltip(k))
	if trayIcon.added {
		procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&trayIcon.data)))
	}
}

// removeStatusItem deletes the icon and ends the tray thread's loop, so no
// ghost icon stays in the notification area after the app quits.
func removeStatusItem() {
	trayIcon.mu.Lock()
	defer trayIcon.mu.Unlock()
	if trayIcon.added {
		procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&trayIcon.data)))
		trayIcon.added = false
	}
	if trayIcon.hwnd != 0 {
		procPostMessageW.Call(trayIcon.hwnd, wmClose, 0, 0)
		trayIcon.hwnd = 0
	}
}

func trayWndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch uint32(message) {
	case wmTrayCallback:
		switch uint32(lParam) & 0xFFFF {
		case wmLButtonUp:
			runTrayAction(trayOpen)
		case wmRButtonUp, wmContextMenu:
			runTrayAction(trayActionFor(showTrayMenu(hwnd)))
		}
		return 0
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	trayIcon.mu.Lock()
	tb := trayIcon.taskbarCreated
	trayIcon.mu.Unlock()
	if tb != 0 && uint32(message) == tb {
		addTrayIcon(hwnd)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

// showTrayMenu shows Open / Quit at the cursor and returns the chosen id.
func showTrayMenu(hwnd uintptr) int {
	menu, _, _ := procCreatePopupMenu.Call()
	defer procDestroyMenu.Call(menu)
	open, _ := windows.UTF16PtrFromString("Open Jumpgate")
	quit, _ := windows.UTF16PtrFromString("Quit Jumpgate")
	procAppendMenuW.Call(menu, mfString, 1, uintptr(unsafe.Pointer(open)))
	procAppendMenuW.Call(menu, mfString, 2, uintptr(unsafe.Pointer(quit)))
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	// Without this the menu stays up when the user clicks elsewhere.
	procSetForegroundWindow.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	return int(cmd)
}

func runTrayAction(a trayAction) {
	switch a {
	case trayOpen:
		trayIcon.mu.Lock()
		main := trayIcon.main
		trayIcon.mu.Unlock()
		if main != 0 {
			procShowWindow.Call(main, swRestore)
			procSetForegroundWindow.Call(main)
		}
	case trayQuit:
		requestQuit()
	}
}
```

```go
// cmd/jumpgate/statusitem_windows_test.go
//go:build tray && windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

// The icon is added, its tooltip follows health, and it is removed. A CI
// session may have no notification area; then the add is refused and the
// test says so instead of failing.
func TestTrayIconLifecycle(t *testing.T) {
	installStatusItem(nil)
	trayIcon.mu.Lock()
	hwnd, added := trayIcon.hwnd, trayIcon.added
	trayIcon.mu.Unlock()
	if hwnd == 0 {
		t.Fatal("the tray window was not created")
	}
	if !added {
		removeStatusItem()
		t.Skip("Shell_NotifyIconW refused: no notification area in this session")
	}
	setHealth(healthOK)
	trayIcon.mu.Lock()
	tip := windows.UTF16ToString(trayIcon.data.SzTip[:])
	trayIcon.mu.Unlock()
	if tip != healthTooltip(healthOK) {
		t.Fatalf("tooltip %q, want %q", tip, healthTooltip(healthOK))
	}
	removeStatusItem()
	trayIcon.mu.Lock()
	defer trayIcon.mu.Unlock()
	if trayIcon.added || trayIcon.hwnd != 0 {
		t.Fatal("the icon or its window survived removeStatusItem")
	}
}
```

Run: `GOOS=windows go vet ./cmd/jumpgate/` (the tray-tagged Windows file needs cgo for webview and is checked on the Windows runner in Step 7).
Expected: clean.

- [ ] **Step 5: Write the Windows package script and README**

```bash
#!/usr/bin/env bash
# scripts/package-windows.sh — jumpgate-windows-amd64.zip (spec D6):
#   jumpgate.exe       console subsystem; double-click opens a terminal
#                      running the terminal home; also the CLI
#   jumpgate-tray.exe  GUI subsystem; the desktop window and tray icon
# Both embed the Linux agents. Runs on a Windows runner (Git Bash) with Go and
# a C compiler (the tray build uses cgo for WebView2).
set -euo pipefail
out="${1:-dist}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
ld="-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$version"
VERSION="$version" GZIP=1 scripts/build-agents.sh internal/agentbin/embedded
stage="$out/jumpgate-windows-amd64"
rm -rf "$stage"
mkdir -p "$stage"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags embedagents -trimpath -ldflags "$ld" \
  -o "$stage/jumpgate.exe" ./cmd/jumpgate
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -tags "tray embedagents" -trimpath -ldflags "$ld -H windowsgui" \
  -o "$stage/jumpgate-tray.exe" ./cmd/jumpgate
cp packaging/windows/README.txt "$stage/README.txt"
rm -f "$out/jumpgate-windows-amd64.zip"
( cd "$out" && 7z a -tzip jumpgate-windows-amd64.zip jumpgate-windows-amd64 >/dev/null )
echo "$out/jumpgate-windows-amd64.zip"
```

```text
Jumpgate for Windows
====================

jumpgate-tray.exe   The desktop app. Double-click it: a window opens and a
                    Jumpgate icon appears in the notification area (right-click
                    it to quit). Needs the Microsoft Edge WebView2 Runtime,
                    which Windows 11 includes.

jumpgate.exe        The terminal. Double-click it for the terminal home, or run
                    it from a terminal: `jumpgate.exe help` lists the commands,
                    `jumpgate.exe open` opens the web app in your browser.

Both need Windows 10 version 1803, Windows Server 2019, or later.
Errors from the desktop app are written to %USERPROFILE%\.jumpgate\run\app.log.

These files are not code-signed yet, so SmartScreen may warn the first time:
choose "More info" and "Run anyway".
```

(save as `packaging/windows/README.txt`, with CRLF line endings is not required; Notepad on Windows 10 1809+ reads LF.)

- [ ] **Step 6: Write the desktop smoke**

```powershell
# scripts/windows-desktop-smoke.ps1 STAGE_DIR — double-click jumpgate-tray.exe
# (no arguments, no console) and check it chose the desktop window, logged to
# app.log, served, and stops on `jumpgate.exe stop` (B-3).
param([Parameter(Mandatory)] [string] $Stage)
$ErrorActionPreference = 'Stop'
$root = Join-Path ($env:RUNNER_TEMP ?? $env:TEMP) 'jgdesk'
Remove-Item -Recurse -Force $root -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $root | Out-Null
$env:USERPROFILE = $root
$env:HOME = $root
$run = Join-Path $root '.jumpgate\run'

$p = Start-Process -FilePath (Join-Path $Stage 'jumpgate-tray.exe') -PassThru
$ok = $false
$deadline = (Get-Date).AddSeconds(30)
while ((Get-Date) -lt $deadline -and -not $ok) {
  $f = Join-Path $run 'server.json'
  if (Test-Path $f) {
    $info = Get-Content $f -Raw | ConvertFrom-Json
    try {
      $h = Invoke-WebRequest -UseBasicParsing -Headers @{ Authorization = "Bearer $($info.token)" } "http://$($info.httpAddr)/api/health"
      $ok = $h.StatusCode -eq 200
    } catch { }
  }
  if (-not $ok) { Start-Sleep -Milliseconds 500 }
}
$log = Join-Path $run 'app.log'
if (-not (Test-Path $log)) { throw 'the GUI exe wrote no app.log, so its errors would be invisible' }
Get-Content $log
if (-not (Select-String -Path $log -Pattern 'opening the desktop window' -Quiet)) { throw 'the GUI exe did not choose the desktop window' }
if (-not $ok) { throw 'the server never answered /api/health' }

& (Join-Path $Stage 'jumpgate.exe') stop
if ($LASTEXITCODE -ne 0) { throw "jumpgate.exe stop exited $LASTEXITCODE" }
if (-not $p.WaitForExit(15000)) { throw 'jumpgate-tray.exe kept running after stop' }
Write-Host 'windows-desktop-smoke: OK'
```

- [ ] **Step 7: Add the Windows desktop CI job**

Append to `.github/workflows/ci.yml` `jobs:`:

```yaml
  desktop-windows:
    runs-on: windows-latest
    defaults:
      run:
        shell: bash
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: WebView2 runtime
        shell: pwsh
        run: |
          $k = 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}'
          if (-not (Test-Path $k)) {
            Invoke-WebRequest https://go.microsoft.com/fwlink/p/?LinkId=2124703 -OutFile wv2.exe
            Start-Process .\wv2.exe -ArgumentList '/silent','/install' -Wait
          }
      - name: vet and test the tray build
        run: |
          go vet -tags tray ./cmd/jumpgate
          go test -tags tray ./cmd/jumpgate
      - name: package
        run: scripts/package-windows.sh dist
      - name: double-click smoke
        shell: pwsh
        run: scripts/windows-desktop-smoke.ps1 -Stage dist/jumpgate-windows-amd64
      - uses: actions/upload-artifact@v4
        with:
          name: jumpgate-windows-amd64-ci
          path: dist/jumpgate-windows-amd64.zip
```

- [ ] **Step 8: Commit and verify**

```bash
chmod +x scripts/package-windows.sh
git add cmd/jumpgate scripts/package-windows.sh scripts/windows-desktop-smoke.ps1 packaging/windows .gitignore .github/workflows/ci.yml
git commit -m "feat(windows): console jumpgate.exe and GUI jumpgate-tray.exe with a tray icon, app.log and error dialogs

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: `desktop-windows` green and its log ends with `windows-desktop-smoke: OK`. If the smoke finds `app.log` with the WebView2 message, or a window-creation error after the health check passed, the runner session cannot host the window: keep the job's assertions on app.log and health (they prove B-3's two failures are fixed) and record that the window itself is checked manually.

Manual check before release, on a real Windows 10/11 desktop: double-click `jumpgate-tray.exe` (window and tray icon appear; right-click > Quit Jumpgate quits); double-click `jumpgate.exe` (a console opens with the terminal home and stays until `q`).

---

### Task 11: Linux desktop build — WebKitGTK 4.1, `.desktop` entries, tarball

Covers B-3 (Linux), I-8 (Linux window; icon deferred by D12), I-11 (Linux), D10, D11.

**Files:**
- Create: `scripts/linux/pkgconfig/webkit2gtk-4.0.pc`, `packaging/linux/jumpgate.desktop`, `packaging/linux/jumpgate-window.desktop`, `packaging/linux/install.sh`, `packaging/linux/README.txt`, `scripts/package-linux.sh`, `scripts/test-linux-desktop.sh`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `scripts/build-agents.sh` (Task 3), the tray build, `runTerminalHome` via plain `jumpgate` (Task 9).
- Produces: `scripts/package-linux.sh [OUT_DIR]` → `OUT_DIR/jumpgate-linux-<goarch>.tar.gz` (arch of the build host) containing `jumpgate-linux-<goarch>/{jumpgate,jumpgate.desktop,jumpgate-window.desktop,jumpgate.svg,install.sh,README.txt}`.

- [ ] **Step 1: Write the desktop-bundle test script first (it fails: nothing to package yet)**

```bash
#!/usr/bin/env bash
# scripts/test-linux-desktop.sh — the Linux desktop bundle end to end, in
# containers: build on Ubuntu 24.04 against WebKitGTK 4.1 (spec D11), validate
# the .desktop files, install as a normal user, open the window under Xvfb and
# check the server answers, then check the binary's libraries resolve on
# Debian 13. Needs Docker; CI runs the same script.
set -euo pipefail
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
go_version="${GO_VERSION:-1.25.0}"

docker run --rm -v "$PWD":/src -v "$work":/out -w /src -e GOFLAGS=-buildvcs=false ubuntu:24.04 bash -euc '
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y -qq build-essential pkg-config git ca-certificates curl gzip \
    libgtk-3-dev libwebkit2gtk-4.1-dev xvfb xauth desktop-file-utils >/dev/null
  curl -fsSL "https://go.dev/dl/go'"$go_version"'.linux-$(dpkg --print-architecture).tar.gz" | tar -C /usr/local -xz
  export PATH=/usr/local/go/bin:$PATH
  git config --global --add safe.directory /src
  scripts/package-linux.sh /out
  arch=$(go env GOARCH)
  b=/out/jumpgate-linux-$arch
  desktop-file-validate "$b/jumpgate.desktop" "$b/jumpgate-window.desktop"
  grep -qx "Terminal=true" "$b/jumpgate.desktop"

  useradd -m u
  cp -r "$b" /home/u/bundle && chown -R u:u /home/u/bundle
  su u -c "/home/u/bundle/install.sh"
  test -x /home/u/.local/bin/jumpgate
  grep -q "^Exec=env JUMPGATE_LAUNCHER=1 /home/u/.local/bin/jumpgate$" /home/u/.local/share/applications/jumpgate.desktop
  grep -q "^Exec=/home/u/.local/bin/jumpgate --tray$" /home/u/.local/share/applications/jumpgate-window.desktop
  test -f /home/u/.local/share/icons/hicolor/scalable/apps/jumpgate.svg

  # The window entry: the server must come up behind the webview. WebKit'"'"'s
  # bubblewrap sandbox cannot run inside an unprivileged container.
  su u -c "cd ~ && WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1 xvfb-run -a ~/.local/bin/jumpgate --tray >/tmp/window.log 2>&1 &"
  ok=
  for _ in $(seq 1 60); do
    f=/home/u/.jumpgate/run/server.json
    if [ -f "$f" ]; then
      token=$(sed -n "s/.*\"token\": \"\(.*\)\".*/\1/p" "$f")
      addr=$(sed -n "s/.*\"httpAddr\": \"\(.*\)\".*/\1/p" "$f")
      if curl -fsS -H "Authorization: Bearer $token" "http://$addr/api/health" >/dev/null 2>&1; then ok=1; break; fi
    fi
    sleep 0.5
  done
  [ -n "$ok" ] || { cat /tmp/window.log; echo "the desktop window build never served" >&2; exit 1; }
  su u -c "~/.local/bin/jumpgate stop"
  echo "ubuntu 24.04: OK"
'

docker run --rm -v "$work":/out debian:trixie bash -euc '
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq && apt-get install -y -qq libwebkit2gtk-4.1-0 libgtk-3-0 >/dev/null
  b=$(ls -d /out/jumpgate-linux-*/ | head -1)
  if ldd "$b/jumpgate" | grep "not found"; then echo "missing libraries on Debian 13" >&2; exit 1; fi
  "$b/jumpgate" --version
  echo "debian 13: OK"
'
```

Run: `chmod +x scripts/test-linux-desktop.sh && scripts/test-linux-desktop.sh`
Expected: FAIL — `scripts/package-linux.sh: No such file or directory`.

- [ ] **Step 2: Add the pkg-config shim, the entries and the installer**

```text
# scripts/linux/pkgconfig/webkit2gtk-4.0.pc
# webview_go asks pkg-config for webkit2gtk-4.0, which Ubuntu 24.04 and
# Debian 13 no longer ship. 4.1 is the same API on libsoup 3, so this name
# simply requires it (spec D11). Only the release and CI builds put this
# directory on PKG_CONFIG_PATH.
Name: webkit2gtk-4.0
Description: Shim mapping webkit2gtk-4.0 to webkit2gtk-4.1 for webview_go
Version: 2.44.0
Requires: webkit2gtk-4.1
```

```ini
# packaging/linux/jumpgate.desktop
[Desktop Entry]
Type=Application
Name=Jumpgate
GenericName=Node fleet control
Comment=Run your node fleet from a terminal
Exec=env JUMPGATE_LAUNCHER=1 jumpgate
Icon=jumpgate
Terminal=true
Categories=Network;System;
Keywords=ethereum;pulsechain;node;validator;
```

```ini
# packaging/linux/jumpgate-window.desktop
[Desktop Entry]
Type=Application
Name=Jumpgate (window)
Comment=Open the Jumpgate panel in a desktop window
Exec=jumpgate --tray
Icon=jumpgate
Terminal=false
Categories=Network;System;
StartupWMClass=jumpgate
```

(Desktop files may not carry comment lines before `[Desktop Entry]` per the spec's validator; write the two files without the first `# packaging/…` line.)

```sh
#!/bin/sh
# install.sh — install jumpgate for this user, without root: the binary, two
# menu entries (a terminal and a window) and the icon, under $PREFIX
# (default ~/.local).
set -eu
here="$(cd "$(dirname "$0")" && pwd)"
prefix="${PREFIX:-$HOME/.local}"
bin="$prefix/bin"
apps="$prefix/share/applications"
icons="$prefix/share/icons/hicolor/scalable/apps"
mkdir -p "$bin" "$apps" "$icons"
install -m 0755 "$here/jumpgate" "$bin/jumpgate"
install -m 0644 "$here/jumpgate.svg" "$icons/jumpgate.svg"
# A desktop session often lacks ~/.local/bin on PATH, so the entries name the
# binary by its absolute path.
for f in jumpgate.desktop jumpgate-window.desktop; do
  sed -e "s|^Exec=env JUMPGATE_LAUNCHER=1 jumpgate\$|Exec=env JUMPGATE_LAUNCHER=1 $bin/jumpgate|" \
      -e "s|^Exec=jumpgate --tray\$|Exec=$bin/jumpgate --tray|" "$here/$f" > "$apps/$f"
  chmod 0644 "$apps/$f"
done
if command -v update-desktop-database >/dev/null 2>&1; then update-desktop-database "$apps" || true; fi
echo "installed $bin/jumpgate; Jumpgate and Jumpgate (window) are in your applications menu"
case ":$PATH:" in
  *":$bin:"*) ;;
  *) echo "note: $bin is not on your PATH; add it to run jumpgate from a shell" ;;
esac
```

```text
Jumpgate for Linux
==================

./install.sh   installs jumpgate for your user under ~/.local (no root), with
               two menu entries: "Jumpgate" opens a terminal running jumpgate,
               "Jumpgate (window)" opens the desktop panel.

Needs WebKitGTK 4.1 and GTK 3 for the window:
  Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0
  Fedora:        sudo dnf install webkit2gtk4.1 gtk3

The terminal entry and every command (`jumpgate help`) work without them on
a server, as does the headless jumpgate_linux_<arch>.tar.gz release archive.
```

(save as `packaging/linux/README.txt`).

- [ ] **Step 3: Write the package script**

```bash
#!/usr/bin/env bash
# scripts/package-linux.sh — jumpgate-linux-<arch>.tar.gz (spec D6, D10): the
# desktop build (webview window on WebKitGTK 4.1) with embedded agents, two
# .desktop entries, the icon and a per-user installer. Needs go, a C
# compiler, pkg-config, libgtk-3-dev and libwebkit2gtk-4.1-dev.
set -euo pipefail
out="${1:-dist}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
arch="$(go env GOARCH)"
VERSION="$version" GZIP=1 scripts/build-agents.sh internal/agentbin/embedded
stage="$out/jumpgate-linux-$arch"
rm -rf "$stage"
mkdir -p "$stage"
PKG_CONFIG_PATH="$PWD/scripts/linux/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}" \
CGO_ENABLED=1 go build -tags "tray embedagents" -trimpath \
  -ldflags "-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$version" \
  -o "$stage/jumpgate" ./cmd/jumpgate
cp packaging/linux/jumpgate.desktop packaging/linux/jumpgate-window.desktop \
   packaging/linux/install.sh packaging/linux/README.txt "$stage/"
cp cmd/jumpgate/icon.svg "$stage/jumpgate.svg"
chmod 0755 "$stage/install.sh" "$stage/jumpgate"
tar -C "$out" -czf "$out/jumpgate-linux-$arch.tar.gz" "jumpgate-linux-$arch"
echo "$out/jumpgate-linux-$arch.tar.gz"
```

Run: `chmod +x scripts/package-linux.sh packaging/linux/install.sh && scripts/test-linux-desktop.sh`
Expected: `ubuntu 24.04: OK` and `debian 13: OK`. If the build fails to compile webview's C++ against 4.1 (an undeclared WebKit symbol), vendor `webview_go` (`go mod vendor` is not allowed to change dependencies; instead copy the module to `third_party/webview_go`, change only its `#cgo linux … pkg-config:` line to `webkit2gtk-4.1` and the failing call to its 4.1 replacement, and add a `replace github.com/webview/webview_go => ./third_party/webview_go` line to `go.mod`); record the change in the commit message.

- [ ] **Step 4: Add the Linux desktop CI job**

Append to `.github/workflows/ci.yml` `jobs:`:

```yaml
  desktop-linux:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: build, install and open the Linux desktop bundle
        run: scripts/test-linux-desktop.sh
```

- [ ] **Step 5: Commit**

```bash
git add scripts/linux scripts/package-linux.sh scripts/test-linux-desktop.sh packaging/linux .github/workflows/ci.yml
git commit -m "feat(linux): desktop bundle on WebKitGTK 4.1 with terminal and window menu entries

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: `desktop-linux` green.

Manual check before release, on an Ubuntu 24.04 desktop: run `install.sh`; "Jumpgate" opens a terminal with the terminal home; "Jumpgate (window)" opens the panel.

---

### Task 12: macOS terminal launcher app

Covers the macOS half of D6, and D24.

**Files:**
- Create: `packaging/macos/jumpgate-terminal`, `packaging/macos/jumpgate.command`, `scripts/test-macos-bundle.sh`
- Modify: `cmd/jumpgate/build-macos-app.sh`, `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `scripts/build-agents.sh` (Task 3); `runTerminalHome` through plain `jumpgate` (Task 9); `JUMPGATE_LAUNCHER`.
- Produces: `cmd/jumpgate/build-macos-app.sh [OUT_DIR]` builds `OUT_DIR/Jumpgate.app` and `OUT_DIR/Jumpgate Terminal.app`, both with embedded agents. `packaging/macos/jumpgate-terminal` honours `JUMPGATE_OPEN` (path of `open`, for tests).

- [ ] **Step 1: Write the bundle test first**

```bash
#!/usr/bin/env bash
# scripts/test-macos-bundle.sh — build both macOS apps and check them without
# a GUI: plists lint, binaries run, and the terminal app hands its bundled
# jumpgate.command to `open` (spec D24). macOS only.
set -euo pipefail
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT
bash cmd/jumpgate/build-macos-app.sh "$out"
tapp="$out/Jumpgate Terminal.app"
for app in "$out/Jumpgate.app" "$tapp"; do
  plutil -lint "$app/Contents/Info.plist"
  codesign --verify "$app"
done
"$out/Jumpgate.app/Contents/MacOS/jumpgate" --version
"$tapp/Contents/Resources/jumpgate" --version
test -x "$tapp/Contents/MacOS/jumpgate-terminal"
test -x "$tapp/Contents/Resources/jumpgate.command"
grep -q '^export JUMPGATE_LAUNCHER=1$' "$tapp/Contents/Resources/jumpgate.command"
sh -n "$tapp/Contents/Resources/jumpgate.command"

rec="$out/open-argv"
fake="$out/fake-open"
printf '#!/bin/sh\nprintf "%%s\\n" "$@" > "%s"\n' "$rec" > "$fake"
chmod +x "$fake"
JUMPGATE_OPEN="$fake" "$tapp/Contents/MacOS/jumpgate-terminal"
grep -qx "$tapp/Contents/Resources/jumpgate.command" "$rec"
echo "macos bundles: OK"
```

Run: `chmod +x scripts/test-macos-bundle.sh && scripts/test-macos-bundle.sh`
Expected: FAIL — `Jumpgate Terminal.app` does not exist.

- [ ] **Step 2: Write the launcher scripts**

```sh
#!/bin/sh
# packaging/macos/jumpgate-terminal — the executable of Jumpgate Terminal.app.
# It opens the bundled jumpgate.command with whatever app handles .command
# files (Terminal.app unless the user chose another), so jumpgate runs in a
# real terminal. `open` needs no Apple Events permission, unlike scripting
# Terminal with osascript (spec D24). JUMPGATE_OPEN replaces open in tests.
here="$(cd "$(dirname "$0")/../Resources" && pwd)"
exec "${JUMPGATE_OPEN:-/usr/bin/open}" "$here/jumpgate.command"
```

```sh
#!/bin/sh
# packaging/macos/jumpgate.command — run by the terminal Jumpgate Terminal.app
# opened. JUMPGATE_LAUNCHER tells jumpgate it owns this window, so an early
# error waits for Enter instead of vanishing.
export JUMPGATE_LAUNCHER=1
exec "$(dirname "$0")/jumpgate"
```

- [ ] **Step 3: Build both apps, with embedded agents**

In `cmd/jumpgate/build-macos-app.sh`:

- before `# --- compile the tray binary`, add:

```bash
# --- the Linux agents this app uploads when pairing (spec D1) --------------
echo "--> building the embedded Linux agents"
( cd "$REPO_ROOT" && VERSION="$VERSION" GZIP=1 bash scripts/build-agents.sh internal/agentbin/embedded )
```

- change `CGO_ENABLED=1 go build -tags tray \` to `CGO_ENABLED=1 go build -tags "tray embedagents" \`;
- after the `codesign` of `Jumpgate.app`, before the final `echo "==> Built $APP"`, add:

```bash
# --- Jumpgate Terminal.app: opens the user's terminal running jumpgate ------
TAPP="$OUT_DIR/$APP_NAME Terminal.app"
echo "==> Building $APP_NAME Terminal.app"
rm -rf "$TAPP"
mkdir -p "$TAPP/Contents/MacOS" "$TAPP/Contents/Resources"
install -m 0755 "$REPO_ROOT/packaging/macos/jumpgate-terminal" "$TAPP/Contents/MacOS/jumpgate-terminal"
install -m 0755 "$REPO_ROOT/packaging/macos/jumpgate.command" "$TAPP/Contents/Resources/jumpgate.command"
# The same binary as the window app: with a terminal and no arguments it
# opens the terminal home, and it starts the server when a command needs it.
install -m 0755 "$APP/Contents/MacOS/$EXE_NAME" "$TAPP/Contents/Resources/jumpgate"
cp "$APP/Contents/Resources/AppIcon.icns" "$TAPP/Contents/Resources/AppIcon.icns"
cat >"$TAPP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>            <string>$APP_NAME Terminal</string>
	<key>CFBundleDisplayName</key>     <string>$APP_NAME Terminal</string>
	<key>CFBundleIdentifier</key>      <string>$BUNDLE_ID.terminal</string>
	<key>CFBundleExecutable</key>      <string>jumpgate-terminal</string>
	<key>CFBundleIconFile</key>        <string>AppIcon</string>
	<key>CFBundlePackageType</key>     <string>APPL</string>
	<key>CFBundleShortVersionString</key> <string>$VERSION</string>
	<key>CFBundleVersion</key>         <string>$VERSION</string>
	<key>LSMinimumSystemVersion</key>  <string>10.15</string>
	<key>LSUIElement</key>             <true/>
</dict>
</plist>
PLIST
codesign --force --deep --sign - "$TAPP" >/dev/null 2>&1 || \
	echo "    (codesign skipped — bundle still runs locally)"
echo "==> Built $TAPP"
```

(`LSUIElement` keeps the launcher, which exits at once, from flashing a Dock icon.)

Run: `scripts/test-macos-bundle.sh`
Expected: `macos bundles: OK`.

- [ ] **Step 4: Try it by hand on this Mac**

Run: `bash cmd/jumpgate/build-macos-app.sh /tmp/jgapps && open "/tmp/jgapps/Jumpgate Terminal.app"`
Expected: a Terminal window opens with the terminal home; `q` ends it.

- [ ] **Step 5: Add the macOS desktop CI job and commit**

Append to `.github/workflows/ci.yml` `jobs:`:

```yaml
  desktop-macos:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: tools
        run: brew install librsvg
      - name: vet and test the tray build
        run: |
          go vet -tags tray ./cmd/jumpgate
          go test -tags tray ./cmd/jumpgate
      - name: build and check both apps
        run: scripts/test-macos-bundle.sh
```

```bash
chmod +x packaging/macos/jumpgate-terminal packaging/macos/jumpgate.command
git add packaging/macos scripts/test-macos-bundle.sh cmd/jumpgate/build-macos-app.sh .github/workflows/ci.yml
git commit -m "feat(macos): Jumpgate Terminal.app opens the user's terminal running jumpgate; embed agents in the apps

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: `desktop-macos` green.

---

### Task 13: Local-host features on each OS; no home, no files

Covers the server part of I-12 (D23), and M-12.

> **Amended by the addendum (Tasks 15–16).** Read "Changes to Task 13" in the addendum before starting: shell-needing routes refuse through `getShellExecutor`, and `writeExecutorError` takes a fallback status.

**Files:**
- Modify: `internal/server/vpn.go` (`vpnExecutor`), `internal/server/vpnserver.go` (`hostExecutor`), `internal/server/api.go` (new `writeExecutorError`), `internal/setup/trust.go` (linux, windows), `internal/setup/trust_test.go`, `cmd/jumpgate/cli.go` (`jgFile`)
- Create: `internal/server/local_unsupported_test.go`, `cmd/jumpgate/jgfile_test.go`

**Interfaces:**
- Consumes: `executor.ErrNoPOSIXShell`, `writeErrorDetail`.
- Produces: `func writeExecutorError(w http.ResponseWriter, err error)` (409 `local_unsupported` for `ErrNoPOSIXShell`, else 500); `const hintLocalUnsupported`; `var exit = os.Exit` in `cmd/jumpgate`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/server/local_unsupported_test.go
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// I-12: on a controller with no POSIX shell (Windows), "this host" features
// answer with a clear refusal and a hint, not a 500.
func TestLocalHostFeaturesAreRefusedClearly(t *testing.T) {
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{},
		NewExecutor: func(config.Target) (executor.Executor, error) {
			return nil, fmt.Errorf("%w, which windows does not provide", executor.ErrNoPOSIXShell)
		}})
	for name, call := range map[string]func(http.ResponseWriter) bool{
		"vpn":  func(w http.ResponseWriter) bool { _, _, ok := s.vpnExecutor(w, config.Config{}, config.VPN{}); return ok },
		"host": func(w http.ResponseWriter) bool { _, ok := s.hostExecutor(w, config.Config{}, ""); return ok },
	} {
		rec := httptest.NewRecorder()
		if call(rec) {
			t.Fatalf("%s: got an executor", name)
		}
		var e struct{ Code, Hint string }
		_ = json.NewDecoder(rec.Body).Decode(&e)
		if rec.Code != http.StatusConflict || e.Code != "local_unsupported" || e.Hint == "" {
			t.Fatalf("%s: %d %+v, want 409 local_unsupported with a hint", name, rec.Code, e)
		}
	}
}
```

Append to `internal/setup/trust_test.go`:

```go
// I-12 (Linux): Fedora and Arch have update-ca-trust, not
// update-ca-certificates; the command handles both, and the manual form is a
// sudo command a person can paste.
func TestTrustStoreCommandLinuxHandlesBothTools(t *testing.T) {
	got, err := TrustStoreCommand("linux", "/var/lib/x/caddy-root.crt", "edge")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"update-ca-certificates", "update-ca-trust extract", "/etc/pki/ca-trust/source/anchors", "/usr/local/share/ca-certificates/valve-node-app-edge.crt"} {
		if !strings.Contains(got.Command, want) {
			t.Errorf("command lacks %q:\n%s", want, got.Command)
		}
	}
	if !strings.HasPrefix(got.ManualCommand, "sudo sh -c ") {
		t.Errorf("ManualCommand %q, want a sudo sh -c form", got.ManualCommand)
	}
}
```

and change the existing Windows assertion (`trust_test.go:78`) to:

```go
	if !strings.Contains(windows.Command, `certutil -addstore -f ROOT "`+path+`"`) {
```

```go
// cmd/jumpgate/jgfile_test.go
package main

import "testing"

// M-12: with no home directory jumpgate stops, rather than writing keys into
// whatever directory it was started from.
func TestJgFileStopsWithoutAHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "") // plan9
	code := -1
	old := exit
	exit = func(c int) { code = c; panic("exit") }
	t.Cleanup(func() { exit = old })
	func() {
		defer func() { _ = recover() }()
		jgFile("keys", "controller.key")
	}()
	if code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
}
```

Run: `go test ./internal/server/ -run LocalHostFeatures ./internal/setup/ -run TrustStore ./cmd/jumpgate/ -run JgFile`
Expected: FAIL — 500 instead of 409; the Linux command lacks `update-ca-trust`; Windows quoting differs; `undefined: exit`.

- [ ] **Step 2: Implement**

In `internal/server/api.go`, beside `writeError`:

```go
const hintLocalUnsupported = "this computer cannot run node commands itself; choose a Linux machine you added with `jumpgate hosts add NAME --ssh`"

// writeExecutorError reports a failure to get an executor. A machine with no
// POSIX shell (a Windows controller asked to act on "this host") is a clear
// refusal the UI can explain, not a server error (I-12).
func writeExecutorError(w http.ResponseWriter, err error) {
	if errors.Is(err, executor.ErrNoPOSIXShell) {
		writeErrorDetail(w, http.StatusConflict, err.Error(), hintLocalUnsupported, "local_unsupported")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}
```

In `vpnExecutor` (`vpn.go`) and `hostExecutor` (`vpnserver.go`), replace the `writeError(w, http.StatusInternalServerError, err.Error())` after `s.getExecutor(t)` with `writeExecutorError(w, err)`.

In `internal/setup/trust.go`, replace the `case "linux":` and `case "windows":` branches:

```go
	case "linux":
		// Debian and Ubuntu rebuild the bundle from
		// /usr/local/share/ca-certificates with update-ca-certificates; Fedora,
		// RHEL and Arch use update-ca-trust with an anchors directory. The
		// command picks whichever the box has. The file is named per gateway so
		// two gateways' roots do not clobber each other. The valve-node-app-
		// prefix is an on-box name and stays until the rename migration.
		name := "valve-node-app-" + sanitizeGatewayID(gatewayID) + ".crt"
		cmd := "if command -v update-ca-certificates >/dev/null 2>&1; then " +
			"cp " + shQuote(certPath) + " " + shQuote("/usr/local/share/ca-certificates/"+name) + " && update-ca-certificates; " +
			"elif command -v update-ca-trust >/dev/null 2>&1; then " +
			"d=/etc/pki/ca-trust/source/anchors; [ -d \"$d\" ] || d=/etc/ca-certificates/trust-source/anchors; " +
			"cp " + shQuote(certPath) + " \"$d/" + name + "\" && update-ca-trust extract; " +
			"else echo 'no CA trust tool found (update-ca-certificates or update-ca-trust)' >&2; exit 1; fi"
		return TrustStoreInstall{
			Command:   cmd,
			NeedsRoot: true,
			// For a person at a terminal when the controller is not root.
			ManualCommand: "sudo sh -c " + shQuote(cmd),
		}, nil

	case "windows":
		// certutil -addstore ROOT writes the machine root store, which requires
		// an elevated (Administrator) shell. cmd.exe does not understand POSIX
		// single quotes, so the path is double-quoted; validateCertPath already
		// forbids a double quote in it.
		return TrustStoreInstall{
			Command:   `certutil -addstore -f ROOT "` + certPath + `"`,
			NeedsRoot: true,
		}, nil
```

Update the doc comment on `ManualCommand` ("It exists for darwin …") to end: "It is also set on Linux, as the sudo form of Command. Empty on Windows, whose Command is already the run-by-hand form."

In `cmd/jumpgate/cli.go`:

```go
// exit is os.Exit; a seam for tests.
var exit = os.Exit

// jgFile is a path inside ~/.jumpgate. With no home directory there is no
// safe place for jumpgate's files, so it stops instead of writing keys into
// the working directory (M-12).
func jgFile(parts ...string) string {
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "jumpgate:", err)
		exit(1)
	}
	return filepath.Join(append([]string{dir}, parts...)...)
}
```

Run: `go test ./internal/server/ ./internal/setup/ ./cmd/jumpgate/`
Expected: PASS.

- [ ] **Step 3: Check the trust command on real distros**

The command runs on the box, so run the exact string `TrustStoreCommand` returns on Debian and Fedora:

```bash
cat > /tmp/trustcmd_test.go <<'EOF2'
package setup

import (
	"fmt"
	"os"
	"testing"
)

func TestPrintLinuxTrustCommand(t *testing.T) {
	i, err := TrustStoreCommand("linux", "/tmp/ca.crt", "edge")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(os.Stderr, i.Command)
}
EOF2
cp /tmp/trustcmd_test.go internal/setup/zz_trustcmd_test.go
cmd="$(go test ./internal/setup/ -run TestPrintLinuxTrustCommand -count=1 2>&1 >/dev/null)"
rm internal/setup/zz_trustcmd_test.go
openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=jumpgate-test -keyout /dev/null -out /tmp/jg-ca.crt -days 1 2>/dev/null
docker run --rm -v /tmp/jg-ca.crt:/tmp/ca.crt:ro debian:12 sh -c "apt-get update -qq && apt-get install -y -qq ca-certificates >/dev/null && $cmd && echo debian:12 trusted"
docker run --rm -v /tmp/jg-ca.crt:/tmp/ca.crt:ro fedora:41 sh -c "$cmd && echo fedora:41 trusted"
```

Expected: `debian:12 trusted` and `fedora:41 trusted` (Debian takes the `update-ca-certificates` branch, Fedora the `update-ca-trust` one).

- [ ] **Step 4: Commit**

```bash
git add internal/server internal/setup cmd/jumpgate
git commit -m "fix(server): refuse local-host features clearly where there is no POSIX shell; trust installs on Fedora too

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: all green.

---

### Task 14: Release workflow, docs, and full verification

Covers D1 (release), D6 (release bundles), D7 (release half), D15, D25, D28, I-9.

**Files:**
- Modify: `.goreleaser.yaml`, `.github/workflows/release.yml`, `README.md`

**Interfaces:**
- Consumes: `scripts/build-agents.sh` (Task 3), `scripts/package-windows.sh` (Task 10), `scripts/package-linux.sh` (Task 11), `cmd/jumpgate/build-macos-app.sh` (Task 12).
- Produces: the release asset set in Global Constraints.

- [ ] **Step 1: Update `.goreleaser.yaml`**

```yaml
version: 2
project_name: jumpgate

before:
  hooks:
    - go mod tidy
    # Release controllers embed both static Linux agents (spec D1), built from
    # this commit with this tag as their version (D15).
    - env GZIP=1 VERSION={{ .Tag }} bash scripts/build-agents.sh internal/agentbin/embedded

builds:
  - id: jumpgate
    main: ./cmd/jumpgate
    binary: jumpgate
    env:
      - CGO_ENABLED=0
    tags:
      - embedagents
    # The tag (v-prefixed) is the version everywhere, so the controller and
    # the agents it embeds report the same string.
    ldflags:
      - -s -w
      - -X github.com/valve-tech/jumpgate/internal/buildinfo.version={{ .Tag }}
    goos:
      - darwin
      - linux
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - id: jumpgate
    formats: [tar.gz]
    name_template: >-
      {{ .ProjectName }}_{{ .Os }}_{{ .Arch }}
    format_overrides:
      - goos: windows
        formats: [zip]

checksum:
  name_template: "checksums.txt"
  extra_files:
    - glob: ./internal/agentbin/embedded/jumpgate-linux-amd64
    - glob: ./internal/agentbin/embedded/jumpgate-linux-arm64

release:
  # The raw agents, for anyone who wants to verify or install one by hand.
  extra_files:
    - glob: ./internal/agentbin/embedded/jumpgate-linux-amd64
    - glob: ./internal/agentbin/embedded/jumpgate-linux-arm64
    - glob: ./internal/agentbin/embedded/SHA256SUMS
      name_template: agents-SHA256SUMS

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

Run (if goreleaser is installed: `brew install goreleaser`): `goreleaser release --snapshot --clean --skip=publish && ls dist`
Expected: archives for six targets; `dist/checksums.txt` lists `jumpgate-linux-amd64` and `jumpgate-linux-arm64`. If goreleaser is not installed, run `goreleaser check` in CI instead via Step 3's `workflow_dispatch`.

- [ ] **Step 2: Replace the desktop job of `.github/workflows/release.yml`**

Replace the header comment's "Two flavors" block with:

```yaml
# Two flavors, per the product decision:
#   - headless binaries (CGO off, cross-compiled, embedded agents) — goreleaser
#   - desktop bundles   (native runners, embedded agents) — the matrix below
#       macOS   -> Jumpgate-macos-<arch>.zip: Jumpgate.app (window + menubar)
#                  and Jumpgate Terminal.app (opens a terminal running jumpgate)
#       Windows -> jumpgate-windows-amd64.zip: jumpgate.exe (console) and
#                  jumpgate-tray.exe (window + notification icon)
#       Linux   -> jumpgate-linux-<arch>.tar.gz: window build on WebKitGTK
#                  4.1, .desktop entries for a terminal and a window, install.sh
```

Replace the whole `desktop:` job with:

```yaml
  desktop:
    needs: [binaries]
    # run even on workflow_dispatch, where `binaries` is skipped
    if: always()
    strategy:
      fail-fast: false
      matrix:
        include:
          - label: macos-arm64
            os: macos-14
          - label: macos-amd64
            os: macos-15-intel # macos-13 (last Intel image) retired ~Dec 2025; macos-15-intel is the x86_64 label until ~Aug 2027
          - label: windows-amd64
            os: windows-latest
          - label: linux-amd64
            os: ubuntu-24.04
          - label: linux-arm64
            os: ubuntu-24.04-arm
    runs-on: ${{ matrix.os }}
    defaults:
      run:
        shell: bash
    env:
      # The tag on a release; on a manual run, git describe inside the scripts.
      VERSION: ${{ startsWith(github.ref, 'refs/tags/v') && github.ref_name || '' }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Build both apps (macOS)
        if: runner.os == 'macOS'
        run: |
          brew install librsvg >/dev/null
          bash cmd/jumpgate/build-macos-app.sh dist
          cd dist && zip -qry "Jumpgate-${{ matrix.label }}.zip" Jumpgate.app "Jumpgate Terminal.app"

      - name: Build the desktop bundle (Linux)
        if: runner.os == 'Linux'
        run: |
          sudo apt-get update
          sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev pkg-config
          scripts/package-linux.sh dist

      - name: Build both exes (Windows)
        if: runner.os == 'Windows'
        run: scripts/package-windows.sh dist

      - name: Upload run artifact
        uses: actions/upload-artifact@v4
        with:
          name: jumpgate-${{ matrix.label }}
          path: |
            dist/*.zip
            dist/*.tar.gz
          if-no-files-found: error

      - name: Attach to Release (tags only)
        if: startsWith(github.ref, 'refs/tags/v')
        uses: softprops/action-gh-release@v2
        with:
          files: |
            dist/*.zip
            dist/*.tar.gz
```

The `binaries` and `checksums` jobs stay as they are (goreleaser's `checksums.txt` now covers the raw agents; the merge adds the desktop bundles).

- [ ] **Step 3: Run the release workflow without publishing**

```bash
git add .goreleaser.yaml .github/workflows/release.yml
git commit -m "ci(release): embed agents everywhere; ship terminal launchers and the Windows GUI exe; add linux/arm64 desktop

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push
gh workflow run release.yml --ref feat/platform-support
gh run watch "$(gh run list --workflow release.yml --branch feat/platform-support --limit 1 --json databaseId -q '.[0].databaseId')"
gh run download "$(gh run list --workflow release.yml --branch feat/platform-support --limit 1 --json databaseId -q '.[0].databaseId')" -D /tmp/jg-release
find /tmp/jg-release -type f | sort
```

Expected: five artifacts: `Jumpgate-macos-arm64.zip`, `Jumpgate-macos-amd64.zip` (each with both apps), `jumpgate-windows-amd64.zip` (both exes and README.txt), `jumpgate-linux-amd64.tar.gz`, `jumpgate-linux-arm64.tar.gz`. Check one by hand:

```bash
unzip -l /tmp/jg-release/jumpgate-windows-amd64/jumpgate-windows-amd64.zip
tar tzf /tmp/jg-release/jumpgate-linux-arm64/jumpgate-linux-arm64.tar.gz
```

Expected: `jumpgate.exe`, `jumpgate-tray.exe`, `README.txt`; and the Linux bundle's six files.

- [ ] **Step 4: Prove a macOS release controller pairs over SSH with no agents dir**

This is the B-1 acceptance check: a controller with embedded agents and no `~/.jumpgate/agents` pairs a Linux box. Run on this Mac, with colima:

```bash
work="$(mktemp -d)"
ssh-keygen -q -t ed25519 -N '' -f "$work/root"
docker build -q -t jumpgate-e2e scripts/e2e >/dev/null
docker run -d --name jg-release-check --privileged -p 127.0.0.1:2222:22 jumpgate-e2e >/dev/null
sleep 5
docker exec -i jg-release-check sh -c 'cat >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys' < "$work/root.pub"
GZIP=1 scripts/build-agents.sh internal/agentbin/embedded
go build -tags embedagents -o "$work/jumpgate" ./cmd/jumpgate
export HOME="$work/home" && mkdir -p "$HOME"
"$work/jumpgate" keys init --store file
"$work/jumpgate" hosts add e2e --ssh root@127.0.0.1:2222 --key "$work/root"   # answer yes to the fingerprint
"$work/jumpgate" stop
docker rm -f jg-release-check
```

Expected: no `~/.jumpgate/agents` exists in the fresh `HOME`; the pairing stream shows `[upload] agent binary for linux/<arch>: embedded in this build` and ends with `paired: agent 0x…`. Open a new shell afterwards (the `export HOME` above was for this check only).

- [ ] **Step 5: Update the README platform section**

Add a "Platforms" section near the top of `README.md`:

```markdown
## Platforms

| | macOS | Windows | Linux |
|---|---|---|---|
| Desktop window | Jumpgate.app (menubar) | jumpgate-tray.exe (notification icon) | "Jumpgate (window)" menu entry |
| Terminal launcher | Jumpgate Terminal.app | double-click jumpgate.exe | "Jumpgate" menu entry |
| Key store default | Keychain | Credential Manager (`wincred`) | Secret Service with a desktop session, else an owner-only file |
| Pair this machine (`--local`) | no | no | yes (root, or sudo on the terminal) |
| Minimum | macOS 10.15 | Windows 10 1803 / Server 2019 | WebKitGTK 4.1 for the window (Ubuntu 22.04+, Debian 12+, Fedora 38+) |

The agent that runs on your node boxes is Linux-only (amd64 and arm64). Every release of jumpgate carries both agents inside it, so a controller on any OS can pair any Linux box without extra downloads.

The downloads are not code-signed yet: Windows SmartScreen may warn ("More info" > "Run anyway"), and macOS may ask you to open the app from System Settings > Privacy & Security the first time.
```

and update the "Grab the archive for your platform" paragraph to name the desktop bundles (`Jumpgate-macos-<arch>.zip`, `jumpgate-windows-amd64.zip`, `jumpgate-linux-<arch>.tar.gz`) beside the headless `jumpgate_<os>_<arch>` archives.

- [ ] **Step 6: Full verification**

Run, and record each result in the PR description:

```bash
go vet ./... && go test ./...
GOOS=windows go vet ./... && GOOS=linux GOARCH=arm64 go vet ./...
scripts/test-linux.sh
scripts/e2e-agent.sh && scripts/e2e-local.sh
scripts/test-secret-service.sh
scripts/test-linux-desktop.sh
scripts/test-macos-bundle.sh
gh pr checks
```

Expected: every command passes, and every CI job is green: `web`, `go` (ubuntu-24.04, macos-latest, windows-latest), `cross`, `agents` (×3), `secret-service`, `e2e` (×2), `windows-daemon`, `windows-ssh-agent`, `desktop-windows`, `desktop-linux`, `desktop-macos`, `billing`.

- [ ] **Step 7: Commit the docs**

```bash
git add README.md
git commit -m "docs: platform support table, launchers, key stores and minimums

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Then run the manual checks listed in Tasks 5, 10 and 11 on real desktops and record them in the PR description before marking it ready for review.


---

## Addendum: Tasks 15–16

Added 2026-10-05, after the gap analysis (`.superpowers/sdd/followups/gap-analysis.md`, items W2 and W5, plus parts of W1 and W3). The spec's addendum (decisions D29–D40) is binding for these two tasks, as D1–D28 are for the rest.

**Goal:** a Windows controller runs the local Docker features (RPC gateway and devnet) with no POSIX shell, and tells a user whose engine is in Windows-container mode what to do. Every controller gets the eRPC gateway image as a prebuilt, pinned, multi-arch download instead of a 12-minute local `docker build` that needs buildx.

**Architecture:** the local executor gains a shell-free surface: `RunArgv` (start a program from an argument vector) and `LocalHost` (home directory, native CPU, dialing). Each Docker call is written once as an `executor.Command`. It runs as argv on the local machine on every OS, and as the same shell string as today over SSH. Readiness probes and the port check run in process on the local machine (`net/http`, `net.Dial`) instead of through `curl` and `ss`. File binds move to `--mount`. `Run` still refuses on Windows, and Task 13 (amended) refuses every shell-needing route with `local_unsupported`. Task 16 then pins `ghcr.io/jumpgate-tech/erpc@sha256:…` in the catalog. A workflow publishes and signs that image, and the gateway pulls it by digest. A local build is kept as a fallback.

**Order and dependencies:**
- **Task 13** is amended (see "Changes to Task 13" below) and must land before Task 15.
- **Task 15** depends on Task 1 (CI matrix, `testutil`) and on the amended Task 13. Its Windows smoke step (Step 16) also needs Task 6 (the server on Windows) merged.
- **Task 16** depends on Task 15, which converts the functions Task 16 edits. Land it after Task 14, so the two `release.yml` edits do not conflict.

**Sizes:** Task 15 is L, Task 16 is M, and the amendment to Task 13 adds S.

### Addendum preconditions

- The same as the plan's own: one confirmation from the user before pushing (D22). Every Windows check runs on GitHub Actions, or by hand where a step says so.
- **Task 16 needs two one-time actions from the user.** Ask once, before Task 16 Step 1:
  1. Allow the `erpc-image` workflow to push to `ghcr.io/jumpgate-tech/erpc`. The repo is `jumpgate-tech/jumpgate-app`, so `GITHUB_TOKEN` with `packages: write` can create the package; no PAT is needed.
  2. After the first publish, set that package's visibility to **Public** (GitHub > jumpgate-tech > Packages > erpc > Package settings). Anonymous `docker pull` needs this.
- Docker via colima on this Mac, for Task 15 Step 15 and Task 16 Step 10.

### Addendum global constraints

These add to the plan's Global Constraints; they do not replace them.

- Shell-free local surface, in package `executor`: `ArgvRunner`, `LocalHost`, `Command`, `Exec`, `QuoteArgv`, `RequireShell`. No Docker call in `internal/ops`, `internal/setup` or `internal/server` builds a `docker …` string for `e.Run` itself. Every one goes through `ops.DockerRun` or `executor.Exec`.
- `executor.QuoteArgv` is byte-identical to the string `ops.DockerRun` built before Task 15: the program bare, then every argument single-quoted with `'\''` escaping. Today's probe strings (`command -v docker`, `docker --version`, `docker info --format '…'`, `command -p uname -m`, the curl forms, `listenerProbe`, `printf '%s\n' "$HOME"`) stay byte-identical for non-argv executors. This means SSH targets and the existing test fakes see no change.
- File binds use `--mount type=bind,source=<host path>,target=<container path>,readonly`, rendered as a CSV record. Named volumes keep `-v`.
- Error code `local_unsupported` keeps its name. Its hint becomes: "node setup, services, logs, diagnostics and the VPN need a POSIX shell, which this computer does not have; add a Linux machine with `jumpgate hosts add NAME --ssh` for those. The Docker gateway and devnet do run here."
- eRPC image:
  - Repository: `ghcr.io/jumpgate-tech/erpc`.
  - Tags: `src-<40-hex ERPCSourceRef>` and `<first 8 hex>`.
  - Platforms: `linux/amd64` and `linux/arm64`.
  - Label: `org.opencontainers.image.revision=<ERPCSourceRef>`.
  - Signed with cosign keyless. The identity regexp is `^https://github.com/jumpgate-tech/jumpgate-app/\.github/workflows/erpc-image\.yml@`, and the issuer is `https://token.actions.githubusercontent.com`.
  - Pinned by its index digest in `internal/catalog/erpc.go`.
- Environment variables:
  - `JUMPGATE_DOCKER_LIVE=1` runs the live Docker tests against this machine's engine.
  - `JUMPGATE_ERPC_LOCAL_BUILD=1` skips the pull and builds the eRPC image from source.
- Timeouts: eRPC pull 10 minutes, eRPC build 30 minutes, local port probe 500 ms per address.

### Addendum Review Focus

1. **A Windows user whose Docker Desktop is in Windows-container mode presses the gateway power button.** Expected: `GET /api/docker` says not ready, with the Docker Desktop "Switch to Linux containers…" hint. The panel shows that hint and does not start provisioning; the response is never a 500. Test: Task 15 `TestDockerStatusWindowsContainers`, and the `windows-docker` CI job.
2. **Docker Engine on Windows Server, which has no Linux mode** (GitHub's runners, the QEMU VM). Expected: the hint says the engine runs Windows containers only, and names Docker Desktop or an SSH Linux machine. It does not send the user looking for a switch menu that does not exist. Test: Task 15 `TestWindowsContainersHintNamesTheRightFix`.
3. **A home directory with a comma or a space** (`C:\Users\Ann, B`). Expected: the bind still names exactly the config file, and `docker run` does not mis-parse it. Test: Task 15 `TestBindMountQuotesAPathWithACommaOrQuote`.
4. **The first gateway provision with no network.** Expected: within the pull timeout, one error naming every way out: connect, load the image by hand, or install buildx. Not a hang, and not "legacy builder deprecated". Test: Task 16 `TestEnsureImage_OfflineWithoutBuildKitNamesEveryFix`.
5. **An air-gapped load of the wrong image under the right tag** (`src-<ref>` built from another ref). Expected: jumpgate refuses that image, says why, and falls through to a local build or the offline error. Test: Task 16 `TestEnsureImage_RefusesASourceTagBuiltFromAnotherRef`.

### Addendum file structure

Created:

| Path | Responsibility |
|---|---|
| `internal/executor/argv.go`, `argv_test.go` | `ArgvRunner`, `LocalHost`, `Command`, `Exec`, `QuoteArgv` |
| `internal/executor/lookpath.go`, `lookpath_test.go`, `lookpath_windows_test.go` | `lookPathIn`: find a program on a given PATH (with PATHEXT on Windows) |
| `internal/executor/nativearch_unix.go`, `nativearch_windows.go` | The machine's real CPU, past translation |
| `internal/executor/shell.go`, `shell_test.go` | `RequireShell` (from the amended Task 13) |
| `internal/executor/argvfake/argvfake.go`, `argvfake_test.go` | A test double for a shell-less local executor, shared by the ops, setup and server tests |
| `internal/ops/httpprobe.go`, `httpprobe_test.go` | `HTTPProbe`: the curl form for SSH, in-process for local |
| `internal/ops/mount.go`, `mount_test.go` | `bindMount`: `--mount` values |
| `internal/setup/targetpath.go`, `targetpath_test.go` | `homeOn`, `joinOn`, `dirOn`: path rules of the machine acted on |
| `internal/setup/listeners.go`, `listeners_test.go` | `probeListeners`: the port check, local or shell |
| `internal/setup/shellless_test.go` | Whole gateway and devnet plans against `argvfake` |
| `internal/setup/docker_live_test.go` | `TestLocalDockerLive` (`JUMPGATE_DOCKER_LIVE=1`) |
| `scripts/windows-docker-smoke.ps1` | A real `jumpgate.exe` against the machine's Docker engine |
| `internal/catalog/erpc.go`, `erpc_test.go` | The eRPC source and image pin |
| `.github/workflows/erpc-image.yml` | Build, push, sign and verify the eRPC image |
| `scripts/erpc-pin.sh`, `scripts/erpc-verify.sh` | Read the pin from the catalog; verify the published image against it |

Modified: `internal/executor/{local.go,local_test.go,localenv_test.go}`, `internal/ops/{docker.go,lifecycle.go,docker_test.go,lifecycle_test.go}`, `internal/setup/{gateway.go,devnet.go,tls.go,traffic.go,gateway_test.go,devnet_test.go,traffic_test.go}`, `internal/server/{api.go,docker.go,containers.go,gateways.go,docker_test.go,containers_test.go}`, `.github/workflows/{ci.yml,release.yml}`, `README.md`.

### Changes to Task 13

Apply these to Task 13 before it starts. Task 13 as written refuses a shell-less local target only where `getExecutor` fails, which is target construction. Task 15 lets a Windows controller construct a local target, because it needs one to run Docker. From then on the refusal has to come from the routes that need a shell. If the guard is not in place first, Task 15 would turn every such route into a mid-stream `ErrNoPOSIXShell` on Windows.

1. **Interfaces.** Add to Task 13's Produces:
   - `func RequireShell(e Executor) error` in `internal/executor/shell.go`;
   - `func (l *local) ShellError() error`;
   - `func (s *Server) getShellExecutor(t config.Target) (executor.Executor, error)`.

   Change `writeExecutorError`'s signature to `func writeExecutorError(w http.ResponseWriter, err error, otherwise int)`. The routes it is moved into answer an unreachable SSH host with 502 today, and must keep doing so.
2. **Hint.** Replace `hintLocalUnsupported` with the text in the addendum global constraints.
3. **Files.** Add these to Task 13's Files:
   - Create: `internal/executor/shell.go`, `internal/executor/shell_test.go`.
   - Modify: `internal/server/diag.go`, and the handlers named in item 5.
4. **Step 1 gains two tests.**

```go
// internal/executor/shell_test.go
package executor

import (
	"context"
	"errors"
	"io/fs"
	"testing"
)

type plainExec struct{}

func (plainExec) Run(context.Context, string, *RunOpts) (Result, error)      { return Result{}, nil }
func (plainExec) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (plainExec) ReadFile(context.Context, string) ([]byte, error)           { return nil, nil }
func (plainExec) Close() error                                               { return nil }

// Only a local executor on a machine with no POSIX shell refuses; every other
// executor (SSH, a working local one, a test fake) can run shell commands.
func TestRequireShell(t *testing.T) {
	if err := RequireShell(plainExec{}); err != nil {
		t.Fatalf("an executor without ShellError: %v, want nil", err)
	}
	if err := RequireShell(&local{}); err != nil {
		t.Fatalf("a local executor with a shell: %v, want nil", err)
	}
	if err := RequireShell(&local{unsupported: localShellError("windows")}); !errors.Is(err, ErrNoPOSIXShell) {
		t.Fatalf("a shell-less local executor: %v, want ErrNoPOSIXShell", err)
	}
}
```

Append to `internal/server/local_unsupported_test.go`:

```go
// shellLess is "this computer" on Windows once Task 15 lands: the target can
// be added (Docker features need it), but no shell command can run.
type shellLess struct{ autoSucceedExecutor }

func (*shellLess) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, fmt.Errorf("%w, which windows does not provide", executor.ErrNoPOSIXShell)
}

func (*shellLess) ShellError() error {
	return fmt.Errorf("%w, which windows does not provide", executor.ErrNoPOSIXShell)
}

// Every route that needs a shell refuses a shell-less target up front with
// local_unsupported, instead of failing halfway through with a raw error.
func TestShellRoutesRefuseAShellLessTarget(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return &shellLess{}, nil })
	if res := a.do(t, "POST", "/api/targets", map[string]any{"id": "me", "mode": "local"}); res.StatusCode != http.StatusCreated {
		t.Fatalf("add local target: %d", res.StatusCode)
	}
	for _, r := range []struct {
		method, path string
		body         any
	}{
		{"POST", "/api/targets/me/setup", catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"}},
		{"POST", "/api/targets/me/services/execution/restart", nil},
		{"POST", "/api/targets/me/services/execution/clear", map[string]string{"confirm": "execution"}},
		{"GET", "/api/targets/me/du", nil},
		{"GET", "/api/targets/me/disk?path=/", nil},
		{"GET", "/api/targets/me/endpoints", nil},
		{"GET", "/api/targets/me/firewall", nil},
		{"GET", "/api/targets/me/diagnostics", nil},
	} {
		res := a.do(t, r.method, r.path, r.body)
		var e struct{ Code, Hint string }
		_ = json.NewDecoder(res.Body).Decode(&e)
		if res.StatusCode != http.StatusConflict || e.Code != "local_unsupported" || e.Hint == "" {
			t.Errorf("%s %s: %d %+v, want 409 local_unsupported with a hint", r.method, r.path, res.StatusCode, e)
		}
	}
}
```

(Add `context` and `catalog` to that file's imports.)

5. **Step 2 gains the implementation.**

```go
// internal/executor/shell.go
package executor

// RequireShell returns nil when e can run Run's `sh -c` commands, and an
// error wrapping ErrNoPOSIXShell when it cannot. Only a local executor on a
// machine with no POSIX shell (Windows) cannot: it still runs Docker through
// RunArgv (spec D30), so it is constructed, and the routes that need a shell
// ask this first.
func RequireShell(e Executor) error {
	if s, ok := e.(interface{ ShellError() error }); ok {
		return s.ShellError()
	}
	return nil
}
```

In `local.go`, add:

```go
// ShellError is nil when Run works here; see RequireShell.
func (l *local) ShellError() error { return l.unsupported }
```

In `api.go`, add:

```go
// getShellExecutor is getExecutor for the routes whose work is shell
// commands: node setup, services, disk, endpoints, firewall, diagnostics,
// logs, the monitor and the VPN. A target whose executor cannot run a shell
// (this computer, on Windows) is refused here with ErrNoPOSIXShell, which
// writeExecutorError turns into 409 local_unsupported. Docker routes use
// getExecutor: they run through RunArgv and need no shell.
func (s *Server) getShellExecutor(t config.Target) (executor.Executor, error) {
	ex, err := s.getExecutor(t)
	if err != nil {
		return nil, err
	}
	if err := executor.RequireShell(ex); err != nil {
		return nil, fmt.Errorf("target %q: %w", t.ID, err)
	}
	return ex, nil
}
```

Then make these switches:
- `handleStartSetup`, `handleServiceAction`, `handleServiceClear`, `handleDiskUsage`, `handleEndpoints`, `handleFirewall` and `handleDiskFree`: replace `s.getExecutor(target)` with `s.getShellExecutor(target)`. Replace the `writeError(w, http.StatusBadGateway, err.Error())` that follows it with `writeExecutorError(w, err, http.StatusBadGateway)`.
- `runDiagnostics` (`diag.go`), `bringUpOverlay` and `handleVPNServerDelete`: call `getShellExecutor`.
- `handleDiagnostics`: map `runDiagnostics`'s error through `writeExecutorError(w, err, http.StatusBadGateway)`.
- `getMonitor` and `getWatcher`: after `s.getExecutorLocked(entry, t)`, add `if err := executor.RequireShell(ex); err != nil { return nil, nil, fmt.Errorf("target %q: %w", t.ID, err) }`. Their HTTP callers map the error through `writeExecutorError(w, err, http.StatusBadGateway)`.
- `vpnExecutor` and `hostExecutor` keep Task 13's change, now spelled `writeExecutorError(w, err, http.StatusInternalServerError)`. Each also calls `executor.RequireShell` on the executor it got, before returning it.
- `writeExecutorError` writes `otherwise` instead of the fixed 500.

Leave the Docker routes on `getExecutor`: everything in `containers.go`, `gateways.go` and `docker.go`.

6. **Commit message:** `fix(server): refuse shell-needing routes clearly where there is no POSIX shell; trust installs on Fedora too`.

Until Task 15 lands, construction still refuses a Windows local target, so these guards are unreachable on Windows; the fake exercises them. That is deliberate: Task 15 only removes the construction refusal.

---

### Task 15: Local Docker features on a Windows controller, without a shell

Covers gap W5. Also covers W3's silent bind-mount crash loop, through `--mount` (D33), and makes the Windows-container message reachable (D35). Decisions: D29–D36.

**Why argv, and not the Engine API or WSL (D29).** Every Docker operation jumpgate runs is already an argument vector: `DockerRun` builds the vector and then quotes it into a string only so that `sh -c` can split it again. Running that vector directly removes the shell and changes nothing else. The same `docker` CLI keeps doing the work, with the same contexts, `DOCKER_HOST`, credential helpers, `--platform` handling and BuildKit.
- **The Engine API over `npipe:////./pipe/docker_engine` was rejected.** It would need either the Docker Go SDK (a new module dependency, which the Global Constraints forbid) or a hand-written client. That client would re-implement `docker run`'s flag semantics (`-p`, `host-gateway`, `--platform`), building from a git context, registry auth and context selection. It would also be a second Docker path that SSH targets never use.
- **A WSL-backed executor was rejected** because Docker Desktop can run on Hyper-V without any WSL distro, and paths would need translating both ways.

**Shell inventory.** Every shell construct the gateway and devnet paths use today, and what replaces it on the local machine. Over SSH, every row keeps today's string.

| # | Where | Today (shell) | Local, after Task 15 |
|---|---|---|---|
| 1 | `ops.ProbeDocker` presence | `command -v docker` | argv `docker --version`; a missing program is exit 127 from `RunArgv` (`lookPathIn` fails) |
| 2 | `ops.ProbeDocker` banner | `docker --version` | argv, same |
| 3 | `ops.ProbeDocker` info | `docker info --format '{{.ServerVersion}}\|…'` (single quotes protect `{{`, `\|`) | argv `docker info --format {{.ServerVersion}}\|…`; no quoting needed |
| 4 | `ops.EnginePlatform` host CPU | `command -p uname -m` | `LocalHost.NativeArch`: `/usr/bin/uname -m` or `/bin/uname -m` as argv on unix; `IsWow64Process2` on Windows |
| 5 | `ops.readEmulation` | `docker version --format '{{.Server.Os}}/{{.Server.Arch}}'` | argv |
| 6 | `ops.ImageExists` | `docker image inspect '<tag>' --format '{{.Id}}'` | argv |
| 7 | `ops.BuildImage` | `docker 'build' …` (QuoteArgv) | argv |
| 8 | `ops.DockerRun` (run, rm -f, stop, ps, inspect, network, exec, volume, port, start/stop/restart) | `docker 'a' 'b' …` | argv; `--filter name=^x$` and `{{json …}}` reach docker unquoted and unexpanded |
| 9 | `setup` gateway `configPath` | `printf '%s\n' "$HOME"`, then `path.Join` | `LocalHost.HomeDir()` (`os.UserHomeDir`), then `filepath.Join` (`C:\Users\…` on Windows) |
| 10 | `setup` `listenerProbe` (gateway and devnet port checks) | `{ ss -ltn \|\| netstat -an \|\| lsof … ; } \| grep -Ei '[:.]N…' \| grep -i listen` (braces, `\|\|`, pipes, redirects, a regex) | `LocalHost.DialContext` to `127.0.0.1:N` and `[::1]:N`, 500 ms each; a connection means a listener |
| 11 | gateway readiness `probeCommand` | `curl -s --max-time 10 -X POST -H '…' --data '…' [--resolve '…'] [--cacert '…'] '<url>' \|\| <same without --cacert>` | `ops.HTTPProbe.Do`: `net/http` dialled through `LocalHost`; `--resolve` becomes a dial override, `--cacert` becomes `RootCAs` read with `e.ReadFile`, and `\|\|` becomes one retry with system roots |
| 12 | devnet readiness `rpcCall` | `curl -s -X POST -H '…' --data '…' '<url>'` | `ops.HTTPProbe.Do` |
| 13 | traffic `ReadGatewaySamples` | `curl -s --max-time N '<url>'` | `ops.HTTPProbe.Do` (GET) |
| 14 | `exportRootCA` | `docker 'exec' '<c>' 'cat' '<path>'` (the shell is only transport; `cat` runs inside the Linux container) | argv; `cat` still runs in the container |
| 15 | erpc.yaml, Caddyfile and root CA files | `e.WriteFile` and `e.ReadFile` (the local executor already uses `os`; no shell) | unchanged, but no longer refused on Windows |
| 16 | systemd backend: `uname`, `id -u`, `chgrp`, `systemctl … && …` | shell | unchanged; on a shell-less executor preflight refuses first, naming the docker backend |
| 17 | trust-cert: `uname`, `id -u`, `TrustVerifyCommand`, `install.Command` | shell | unchanged; Windows is answered with the elevated-prompt command to run by hand (no execution) |
| 18 | `POST /api/docker/start` | `open -a Docker \|\| open -a OrbStack` | unchanged (macOS only); Windows auto-start is deferred (D36) |

**Files:**
- Create: `internal/executor/argv.go`, `argv_test.go`, `lookpath.go`, `lookpath_test.go`, `lookpath_windows_test.go`, `nativearch_unix.go`, `nativearch_windows.go`, `internal/executor/argvfake/argvfake.go`, `argvfake_test.go`, `internal/ops/httpprobe.go`, `httpprobe_test.go`, `internal/ops/mount.go`, `mount_test.go`, `internal/setup/targetpath.go`, `targetpath_test.go`, `listeners.go`, `listeners_test.go`, `shellless_test.go`, `docker_live_test.go`, `scripts/windows-docker-smoke.ps1`
- Modify: `internal/executor/local.go`, `local_test.go`, `localenv_test.go`, `internal/ops/docker.go`, `lifecycle.go`, `docker_test.go:33,412-426,999,1073`, `internal/setup/gateway.go`, `devnet.go`, `traffic.go`, `gateway_test.go:519`, `internal/server/api.go`, `docker.go`, `containers.go`, `gateways.go`, `docker_test.go`, `.github/workflows/ci.yml`

**Interfaces:**
- Consumes (amended Task 13): `executor.RequireShell`, `(*local).ShellError`, `writeExecutorError(w, err, otherwise)`. Consumes (Task 1): `testutil.RequireUnix`.
- Produces (package `executor`):
  - `type ArgvRunner interface { RunArgv(ctx context.Context, argv []string, opts *RunOpts) (Result, error) }`
  - `type LocalHost interface { HostGOOS() string; HomeDir() (string, error); NativeArch(ctx context.Context) string; DialContext(ctx context.Context, network, addr string) (net.Conn, error) }`
  - `type Command struct { Argv []string; Shell string }`
  - `func Exec(ctx context.Context, e Executor, c Command, opts *RunOpts) (Result, error)`
  - `func QuoteArgv(argv []string) string`
  - `*local` implements `ArgvRunner` and `LocalHost` on every OS.
- Produces (package `argvfake`): `func New() *Fake`; `(*Fake).Script(prefix string, res executor.Result) *Fake`; `(*Fake).Route(addr, to string) *Fake`; the fields `GOOS`, `Home`, `Arch`, `Files`; and `(*Fake).Argvs() [][]string`, `(*Fake).ShellCalls() []string`.
- Produces (package `ops`):
  - `type HTTPProbe struct { URL, Body, Resolve, CAFile string; MaxTime time.Duration }`
  - `func (p HTTPProbe) CurlCommand() string`
  - `func (p HTTPProbe) Do(ctx context.Context, e executor.Executor) (string, error)`
  - `type ProbeError struct{ Detail string }`
  - `func (d DockerInfo) WindowsContainersHint() string`
  - `func bindMount(src, dst string) string` (unexported; used by `ERPCRunArgs` and `CaddyRunArgs`)
- Produces (package `setup`): `homeOn`, `joinOn`, `dirOn`, `probeListeners(ctx, e, port int) (string, error)`.
- Produces (package `server`): `dockerStatusResponse.WindowsContainers bool` (`json:"windowsContainers,omitempty"`).

- [ ] **Step 1: Write the failing executor tests**

```go
// internal/executor/argv_test.go
package executor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// The shell form of an argv must stay byte-identical to the string
// ops.DockerRun built before Task 15, so an SSH target, and every test fake
// keyed on those strings, sees no change.
func TestQuoteArgvMatchesTheHistoricDockerRunString(t *testing.T) {
	got := QuoteArgv([]string{"docker", "inspect", "-f", "{{.State.Running}}", "it's"})
	want := `docker 'inspect' '-f' '{{.State.Running}}' 'it'\''s'`
	if got != want {
		t.Fatalf("QuoteArgv = %s\nwant       %s", got, want)
	}
}

type shellOnly struct{ ran string }

func (s *shellOnly) Run(_ context.Context, cmd string, _ *RunOpts) (Result, error) {
	s.ran = cmd
	return Result{}, nil
}
func (*shellOnly) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (*shellOnly) ReadFile(context.Context, string) ([]byte, error)           { return nil, nil }
func (*shellOnly) Close() error                                               { return nil }

type argvToo struct {
	shellOnly
	argv []string
}

func (a *argvToo) RunArgv(_ context.Context, argv []string, _ *RunOpts) (Result, error) {
	a.argv = argv
	return Result{}, nil
}

func TestExecPrefersArgvAndKeepsTheShellFormForOthers(t *testing.T) {
	ctx := context.Background()
	c := Command{Argv: []string{"docker", "info", "--format", "{{.OSType}}"}, Shell: "docker info --format '{{.OSType}}'"}

	sh := &shellOnly{}
	if _, err := Exec(ctx, sh, c, nil); err != nil || sh.ran != c.Shell {
		t.Fatalf("shell executor ran %q (%v), want %q", sh.ran, err, c.Shell)
	}
	av := &argvToo{}
	if _, err := Exec(ctx, av, c, nil); err != nil || strings.Join(av.argv, "|") != "docker|info|--format|{{.OSType}}" || av.ran != "" {
		t.Fatalf("argv executor got argv %q and shell %q (%v)", av.argv, av.ran, err)
	}
	sh2 := &shellOnly{}
	if _, _ = Exec(ctx, sh2, Command{Argv: []string{"docker", "ps"}}, nil); sh2.ran != "docker 'ps'" {
		t.Fatalf("no Shell given: ran %q, want the quoted argv", sh2.ran)
	}
}

// TestHelperProcess is not a test. The RunArgv tests start this test binary
// as a plain program, which works on every OS (no sh, no echo.exe).
func TestHelperProcess(t *testing.T) {
	if os.Getenv("JUMPGATE_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Println("argv:" + strings.Join(os.Args[len(os.Args)-2:], "|"))
	os.Exit(3)
}

func helperArgv(a, b string) []string {
	return []string{os.Args[0], "-test.run=^TestHelperProcess$", "--", a, b}
}

func TestLocal_RunArgvStartsTheProgramWithoutAShell(t *testing.T) {
	t.Setenv("JUMPGATE_HELPER_PROCESS", "1")
	res, err := (&local{}).RunArgv(context.Background(), helperArgv("a b", "it's"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 3 || strings.TrimSpace(res.Stdout) != "argv:a b|it's" {
		t.Fatalf("exit %d stdout %q: want exit 3 and the arguments untouched by any shell", res.ExitCode, res.Stdout)
	}
}

// A program that is not there reads the way `sh -c` reports it (exit 127),
// so ProbeDocker's absence check is one branch for both forms.
func TestLocal_RunArgvMissingProgramReadsLikeTheShell(t *testing.T) {
	res, err := (&local{}).RunArgv(context.Background(), []string{"jumpgate-no-such-program"}, nil)
	if err != nil || res.ExitCode != 127 || !strings.Contains(res.Stderr, "not found") {
		t.Fatalf("got %+v, %v; want exit 127 with a not-found stderr", res, err)
	}
}

// Windows, after Task 15: Run is refused, everything Docker needs works.
// This replaces TestLocal_UnsupportedHostFailsEveryCall (spec D30).
func TestLocal_ShellLessHostRefusesOnlyTheShell(t *testing.T) {
	t.Setenv("JUMPGATE_HELPER_PROCESS", "1")
	ctx := context.Background()
	e := &local{unsupported: localShellError("windows")}

	if _, err := e.Run(ctx, "echo hi", nil); !errors.Is(err, ErrNoPOSIXShell) {
		t.Errorf("Run: %v, want ErrNoPOSIXShell", err)
	}
	if err := RequireShell(e); !errors.Is(err, ErrNoPOSIXShell) {
		t.Errorf("RequireShell: %v, want ErrNoPOSIXShell", err)
	}
	if res, err := e.RunArgv(ctx, helperArgv("x", "y"), nil); err != nil || res.ExitCode != 3 {
		t.Errorf("RunArgv: %+v, %v; want the program to run", res, err)
	}
	p := filepath.Join(t.TempDir(), "erpc.yaml")
	if err := e.WriteFile(ctx, p, []byte("a: 1\n"), 0o644); err != nil {
		t.Errorf("WriteFile: %v", err)
	}
	if got, err := e.ReadFile(ctx, p); err != nil || string(got) != "a: 1\n" {
		t.Errorf("ReadFile: %q, %v", got, err)
	}
	if h, err := e.HomeDir(); err != nil || h == "" {
		t.Errorf("HomeDir: %q, %v", h, err)
	}
	if e.HostGOOS() != runtime.GOOS {
		t.Errorf("HostGOOS = %q", e.HostGOOS())
	}
}

// exec.Command resolves a bare name against this process's PATH, before
// c.Env applies, so the dirs localEnv adds would never be searched.
// lookPathIn takes the PATH to search.
func TestLookPathInSearchesTheGivenPathNotTheProcessPath(t *testing.T) {
	testutil.RequireUnix(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "docker")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "/nonexistent")
	if got, err := lookPathIn("docker", dir, runtime.GOOS, ""); err != nil || got != p {
		t.Fatalf("lookPathIn = %q, %v; want %q", got, err, p)
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := lookPathIn("docker", dir, runtime.GOOS, ""); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("a non-executable file: %v, want exec.ErrNotFound", err)
	}
}
```

```go
// internal/executor/lookpath_windows_test.go
package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLookPathInHonoursPATHEXT(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker.exe")
	if err := os.WriteFile(p, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := lookPathIn("docker", dir, "windows", ".COM;.EXE")
	if err != nil || !strings.EqualFold(got, p) {
		t.Fatalf("lookPathIn = %q, %v; want %q", got, err, p)
	}
}
```

Append to `internal/executor/localenv_test.go`:

```go
// An Explorer-started jumpgate-tray.exe inherits the PATH of the login
// session. If Docker Desktop was installed after login, its CLI dir is not
// on that PATH, so localEnv appends it, the same way it appends Homebrew on
// macOS. Windows spells the variable "Path" and separates entries with ";".
func TestLocalEnvAddsDockerDesktopOnWindows(t *testing.T) {
	env := localEnv([]string{`Path=C:\Windows\system32`, `ProgramFiles=C:\Program Files`}, "windows", `C:\Users\dev`)
	want := `Path=C:\Windows\system32;C:\Program Files\Docker\Docker\resources\bin`
	if env[0] != want {
		t.Fatalf("got %q\nwant %q", env[0], want)
	}
}
```

Delete `TestLocal_UnsupportedHostFailsEveryCall` from `local_test.go`; `TestLocal_ShellLessHostRefusesOnlyTheShell` replaces it.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/executor/`
Expected: FAIL to compile: `undefined: QuoteArgv`, `Command`, `Exec`, `lookPathIn`; `e.RunArgv undefined`.

- [ ] **Step 3: Implement the executor side**

```go
// internal/executor/argv.go
package executor

import (
	"context"
	"net"
	"strings"
)

// ArgvRunner is implemented by an executor that can start a program directly
// from an argument vector, with no shell in between. The local executor is
// one, on every OS: that is how a Windows controller runs docker, with no sh
// to hand a command string to (spec D29). The SSH executor is not one,
// because a remote command is a string the far side's shell parses.
type ArgvRunner interface {
	RunArgv(ctx context.Context, argv []string, opts *RunOpts) (Result, error)
}

// LocalHost is implemented by the local executor. Each method answers a
// question a caller would otherwise put to the target's shell (`printf
// "$HOME"`, `uname -m`, `ss`, `curl`), read from this process instead, so the
// answer does not depend on a shell existing.
type LocalHost interface {
	// HostGOOS is runtime.GOOS: whose path rules this target's files follow.
	HostGOOS() string
	// HomeDir is the current user's home directory (os.UserHomeDir).
	HomeDir() (string, error)
	// NativeArch is the machine's CPU as `uname -m` or GOARCH spells it
	// (ops.PlatformForArch reads both), or "" when it cannot be read.
	NativeArch(ctx context.Context) string
	// DialContext connects from this machine, as net.Dialer does.
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// Command is one program run, in both forms an executor may need.
type Command struct {
	// Argv is the program and its arguments, for an ArgvRunner.
	Argv []string
	// Shell is the `sh -c` form for every other executor. Empty means
	// QuoteArgv(Argv). It is set where an existing probe string must stay
	// byte-identical for SSH targets and the tests that know it.
	Shell string
}

// Exec runs c on e: as argv when e can start programs directly, otherwise
// as a shell string. A caller writes one call, and gets the shell-free path
// on the local machine and the unchanged shell path over SSH.
func Exec(ctx context.Context, e Executor, c Command, opts *RunOpts) (Result, error) {
	if a, ok := e.(ArgvRunner); ok && len(c.Argv) > 0 {
		return a.RunArgv(ctx, c.Argv, opts)
	}
	cmd := c.Shell
	if cmd == "" {
		cmd = QuoteArgv(c.Argv)
	}
	return e.Run(ctx, cmd, opts)
}

// QuoteArgv renders argv as a shell command: the program name as is, then
// every argument single-quoted. It is byte-identical to the string
// ops.DockerRun has always built ('\'' escaping, not ssh.go's '"'"'), so a
// shell target sees no change.
func QuoteArgv(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	parts := make([]string, 0, len(argv))
	parts = append(parts, argv[0])
	for _, a := range argv[1:] {
		parts = append(parts, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
	}
	return strings.Join(parts, " ")
}
```

```go
// internal/executor/lookpath.go
package executor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// lookPathIn is exec.LookPath against a given PATH rather than this
// process's own. exec.Command resolves a bare name with os.Getenv("PATH")
// when it is constructed, before c.Env applies. Without this, the dirs
// localEnv adds (Docker Desktop's CLI dir for an app started from Finder or
// Explorer) would never be searched.
func lookPathIn(name, pathList, goos, pathext string) (string, error) {
	if strings.ContainsAny(name, `/\`) {
		return name, nil
	}
	sep, exts := ":", []string{""}
	if goos == "windows" {
		sep, exts = ";", nil
		if filepath.Ext(name) != "" {
			exts = append(exts, "")
		}
		if pathext == "" {
			pathext = ".COM;.EXE;.BAT;.CMD"
		}
		for _, e := range strings.Split(pathext, ";") {
			if e != "" {
				exts = append(exts, strings.ToLower(e))
			}
		}
	}
	for _, dir := range strings.Split(pathList, sep) {
		if dir == "" {
			continue
		}
		for _, ext := range exts {
			p := filepath.Join(dir, name+ext)
			fi, err := os.Stat(p)
			if err != nil || !fi.Mode().IsRegular() {
				continue
			}
			if goos == "windows" || fi.Mode()&0o111 != 0 {
				return p, nil
			}
		}
	}
	return "", exec.ErrNotFound
}
```

```go
// internal/executor/nativearch_unix.go
//go:build !windows

package executor

import (
	"context"
	"strings"
)

// nativeArch asks the system's own uname, at a fixed path, never one found
// on PATH. A Homebrew GNU uname is itself an x86_64 binary on an Apple
// Silicon Mac and reports x86_64; ops.unameArchProbe documents that
// measured failure. /bin/uname covers distros without a merged /usr.
func nativeArch(ctx context.Context, l *local) string {
	for _, p := range []string{"/usr/bin/uname", "/bin/uname"} {
		if res, err := l.RunArgv(ctx, []string{p, "-m"}, nil); err == nil && res.ExitCode == 0 {
			return strings.TrimSpace(res.Stdout)
		}
	}
	return ""
}
```

```go
// internal/executor/nativearch_windows.go
package executor

import (
	"context"

	"golang.org/x/sys/windows"
)

// IMAGE_FILE_MACHINE_* values IsWow64Process2 reports.
const (
	imageFileMachineAMD64 = 0x8664
	imageFileMachineARM64 = 0xAA64
)

// nativeArch asks Windows for the machine's own architecture. An amd64
// jumpgate on Windows on Arm runs emulated, and runtime.GOARCH would say
// amd64, which is the same lie Rosetta tells on a Mac.
func nativeArch(context.Context, *local) string {
	var proc, native uint16
	if err := windows.IsWow64Process2(windows.CurrentProcess(), &proc, &native); err != nil {
		return ""
	}
	switch native {
	case imageFileMachineAMD64:
		return "amd64"
	case imageFileMachineARM64:
		return "arm64"
	}
	return ""
}
```

In `internal/executor/local.go`:

1. Move everything in `Run` after the `exec.CommandContext(…)` and `c.Env` lines into `func (l *local) start(ctx context.Context, c *exec.Cmd, opts *RunOpts) (Result, error)`. `Run` becomes the refusal check, `c := exec.CommandContext(ctx, "sh", "-c", cmd)`, `c.Env = localEnv(os.Environ(), runtime.GOOS, os.Getenv("HOME"))`, then `return l.start(ctx, c, opts)`.
2. Add:

```go
// RunArgv starts argv[0] directly, with no shell, on every OS (spec D29).
// A program that is not on PATH comes back as exit 127 with a "not found"
// stderr, which is the reading `sh -c` gives, so callers branch on one shape.
func (l *local) RunArgv(ctx context.Context, argv []string, opts *RunOpts) (Result, error) {
	if len(argv) == 0 {
		return Result{}, errors.New("executor: RunArgv: empty argv")
	}
	env := localEnv(os.Environ(), runtime.GOOS, os.Getenv("HOME"))
	prog, err := lookPathIn(argv[0], envValue(env, "PATH", runtime.GOOS), runtime.GOOS, envValue(env, "PATHEXT", runtime.GOOS))
	if err != nil {
		return Result{ExitCode: 127, Stderr: argv[0] + ": not found on PATH\n"}, nil
	}
	c := exec.CommandContext(ctx, prog, argv[1:]...)
	c.Env = env
	return l.start(ctx, c, opts)
}

// The LocalHost facts (see argv.go).
func (l *local) HostGOOS() string                       { return runtime.GOOS }
func (l *local) HomeDir() (string, error)               { return os.UserHomeDir() }
func (l *local) NativeArch(ctx context.Context) string { return nativeArch(ctx, l) }
func (l *local) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}

// envValue reads key from env, ignoring case on Windows, where the
// variable is usually spelled "Path".
func envValue(env []string, key, goos string) string {
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if ok && (k == key || (goos == "windows" && strings.EqualFold(k, key))) {
			return v
		}
	}
	return ""
}
```

3. Remove the `l.unsupported` checks from `WriteFile` and `ReadFile`. Rewrite the doc comment on the `unsupported` field: "non-nil when this machine has no POSIX shell. Run refuses with it; RunArgv, the files and the LocalHost facts do not need a shell and keep working (spec D30). RequireShell exposes it to the routes that need a shell."
4. Reword `localShellError`'s message to: `"%w, which %s does not provide: node setup, services, logs and the VPN need a Linux machine added over SSH; the Docker gateway and devnet do run here"`.
5. Update `localEnv`, `guiPathDirs` and `appendMissingDirs` for Windows:
   - Match the PATH key with `strings.EqualFold` when `goos == "windows"`.
   - Use `;` as the separator on Windows and `:` elsewhere; `appendMissingDirs` takes it as a parameter.
   - `guiPathDirs("windows", …)` returns `[]string{programFiles + `\Docker\Docker\resources\bin`}`, where `programFiles` is the `ProgramFiles` value read from `env` with `envValue`. Build it by string concatenation, so the test holds on any host OS. Return nil when `ProgramFiles` is unset.
   - `guiPathDirs` gains an `env []string` parameter, and `localEnv` passes its own.

Then create `internal/executor/argvfake/argvfake.go`:

```go
// Package argvfake is a test double for the local executor on a machine with
// no POSIX shell: a Windows controller. Run fails the way the real one does
// there and records the attempt, so a test can assert a whole plan never
// needed a shell. RunArgv answers from scripts, files live in memory, the
// LocalHost facts are fixed, and DialContext reaches only addresses a test
// routed. It is a non-test package so the ops, setup and server tests can
// share it; nothing outside tests imports it.
package argvfake

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/valve-tech/jumpgate/internal/executor"
)

type Fake struct {
	GOOS, Home, Arch string
	Files            map[string][]byte

	mu      sync.Mutex
	scripts map[string]executor.Result
	routes  map[string]string
	argvs   [][]string
	shell   []string
}

// New is a Windows controller: GOOS "windows", home C:\Users\dev, amd64.
func New() *Fake {
	return &Fake{GOOS: "windows", Home: `C:\Users\dev`, Arch: "amd64",
		Files: map[string][]byte{}, scripts: map[string]executor.Result{}, routes: map[string]string{}}
}

// Script answers every argv whose space-joined form starts with prefix; the
// longest matching prefix wins. Unscripted argvs succeed with no output.
func (f *Fake) Script(prefix string, res executor.Result) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts[prefix] = res
	return f
}

// Route makes DialContext to addr connect to "to" (an httptest listener)
// instead. Every other address refuses, as a free port does.
func (f *Fake) Route(addr, to string) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routes[addr] = to
	return f
}

func (f *Fake) Argvs() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.argvs...)
}

func (f *Fake) ShellCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.shell...)
}

func (f *Fake) shellErr() error {
	return fmt.Errorf("%w, which %s does not provide", executor.ErrNoPOSIXShell, f.GOOS)
}

func (f *Fake) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	f.mu.Lock()
	f.shell = append(f.shell, cmd)
	f.mu.Unlock()
	return executor.Result{}, f.shellErr()
}

func (f *Fake) ShellError() error { return f.shellErr() }

func (f *Fake) RunArgv(_ context.Context, argv []string, opts *executor.RunOpts) (executor.Result, error) {
	joined := strings.Join(argv, " ")
	f.mu.Lock()
	f.argvs = append(f.argvs, append([]string(nil), argv...))
	keys := make([]string, 0, len(f.scripts))
	for k := range f.scripts {
		if strings.HasPrefix(joined, k) {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	res := executor.Result{}
	if len(keys) > 0 {
		res = f.scripts[keys[0]]
	}
	f.mu.Unlock()
	if opts != nil && opts.Stream != nil {
		for _, line := range strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n") {
			if line != "" {
				opts.Stream(line)
			}
		}
	}
	return res, nil
}

func (f *Fake) WriteFile(_ context.Context, path string, content []byte, _ fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Files[path] = append([]byte(nil), content...)
	return nil
}

func (f *Fake) ReadFile(_ context.Context, path string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.Files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}
	return b, nil
}

func (f *Fake) Close() error                              { return nil }
func (f *Fake) HostGOOS() string                          { return f.GOOS }
func (f *Fake) HomeDir() (string, error)                  { return f.Home, nil }
func (f *Fake) NativeArch(context.Context) string         { return f.Arch }

func (f *Fake) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	f.mu.Lock()
	to, ok := f.routes[addr]
	f.mu.Unlock()
	if !ok {
		return nil, &net.OpError{Op: "dial", Net: network, Err: syscall.ECONNREFUSED}
	}
	var d net.Dialer
	return d.DialContext(ctx, network, to)
}

var (
	_ executor.Executor   = (*Fake)(nil)
	_ executor.ArgvRunner = (*Fake)(nil)
	_ executor.LocalHost  = (*Fake)(nil)
)
```

`argvfake_test.go` checks the longest-prefix rule and that `Run` is recorded and refused:

```go
package argvfake

import (
	"context"
	"errors"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestFakeRefusesTheShellAndAnswersArgvByLongestPrefix(t *testing.T) {
	f := New().Script("docker", executor.Result{Stdout: "short"}).Script("docker info", executor.Result{Stdout: "long"})
	if res, _ := f.RunArgv(context.Background(), []string{"docker", "info", "--format", "x"}, nil); res.Stdout != "long" {
		t.Fatalf("got %q, want the longest prefix's answer", res.Stdout)
	}
	if _, err := f.Run(context.Background(), "uname", nil); !errors.Is(err, executor.ErrNoPOSIXShell) || len(f.ShellCalls()) != 1 {
		t.Fatalf("Run: %v, calls %v", err, f.ShellCalls())
	}
}
```

- [ ] **Step 4: Run the executor tests**

Run: `go test ./internal/executor/... && GOOS=windows go vet ./internal/executor/...`
Expected: PASS, and vet clean for Windows (`nativearch_windows.go` compiles).

- [ ] **Step 5: Commit**

```bash
git add internal/executor
git commit -m "feat(executor): run programs from argv without a shell; local host facts; a shell-less test double

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 6: Write the failing ops tests**

```go
// internal/ops/argv_test.go
package ops

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// On the local machine docker runs as argv, and nothing reaches a shell.
func TestDockerRun_UsesArgvOnAShellLessExecutor(t *testing.T) {
	f := argvfake.New()
	if _, err := DockerRun(context.Background(), f, "ps", "-a", "--filter", "name=^x$", "--format", "{{.Names}}"); err != nil {
		t.Fatal(err)
	}
	got := f.Argvs()
	if len(got) != 1 || strings.Join(got[0], "|") != "docker|ps|-a|--filter|name=^x$|--format|{{.Names}}" {
		t.Fatalf("argv %q", got)
	}
	if len(f.ShellCalls()) != 0 {
		t.Fatalf("shell used: %q", f.ShellCalls())
	}
}

func TestProbeDocker_OverArgvReadsTheEngine(t *testing.T) {
	f := argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0, build bde2b89\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|linux|x86_64|docker-desktop|Docker Desktop\n"})
	info, err := ProbeDocker(context.Background(), f)
	if err != nil || !info.DaemonReachable || info.Flavor != FlavorDockerDesktop || info.OSType != "linux" {
		t.Fatalf("got %+v, %v", info, err)
	}
	for _, a := range f.Argvs() {
		if strings.Contains(strings.Join(a, " "), "'") {
			t.Fatalf("argv %q carries shell quoting", a)
		}
	}
}

func TestProbeDocker_OverArgvAbsentIsTyped(t *testing.T) {
	f := argvfake.New().Script("docker --version", executor.Result{ExitCode: 127, Stderr: "docker: not found on PATH\n"})
	_, err := ProbeDocker(context.Background(), f)
	if !errors.Is(err, ErrDockerAbsent) {
		t.Fatalf("got %v, want ErrDockerAbsent", err)
	}
}

func TestEnginePlatform_UsesTheLocalHostsNativeArch(t *testing.T) {
	f := argvfake.New()
	f.Arch = "arm64"
	info := DockerInfo{Architecture: "x86_64", Flavor: FlavorDockerDesktop}
	if got := EnginePlatform(context.Background(), f, info); got != "linux/arm64" {
		t.Fatalf("got %q, want the host's reading on a VM-backed engine", got)
	}
}

// Docker Desktop can switch to Linux containers; Docker Engine on Windows
// Server cannot, and must not be told to look for a menu it does not have.
func TestWindowsContainersHintNamesTheRightFix(t *testing.T) {
	desktop := DockerInfo{OSType: "windows", Flavor: FlavorDockerDesktop}.WindowsContainersHint()
	server := DockerInfo{OSType: "windows", Flavor: FlavorDockerEngine}.WindowsContainersHint()
	if !strings.Contains(desktop, "Switch to Linux containers") {
		t.Errorf("Docker Desktop hint %q does not name the switch", desktop)
	}
	if strings.Contains(server, "Switch to") || !strings.Contains(server, "Windows containers only") || !strings.Contains(server, "--ssh") {
		t.Errorf("Windows Server hint %q", server)
	}
	for _, h := range []string{desktop, server} {
		if !strings.Contains(h, "Linux containers") {
			t.Errorf("hint %q lacks \"Linux containers\", which the preflight tests and the CI job look for", h)
		}
	}
}

// Docker Desktop in Windows-container mode answers `docker info` with the
// Windows daemon's own details, which say nothing about Desktop. The local
// machine recognises Desktop by its installed program instead.
func TestProbeDocker_RecognisesDockerDesktopInWindowsMode(t *testing.T) {
	old := dockerDesktopInstalled
	dockerDesktopInstalled = func() bool { return true }
	t.Cleanup(func() { dockerDesktopInstalled = old })
	f := argvfake.New().Script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|DEV-PC|Microsoft Windows 11 Pro\n"})
	info, err := ProbeDocker(context.Background(), f)
	if err != nil || !info.WindowsContainers() || info.Flavor != FlavorDockerDesktop {
		t.Fatalf("got %+v, %v", info, err)
	}
}
```

```go
// internal/ops/mount_test.go
package ops

import (
	"encoding/csv"
	"strings"
	"testing"
)

// docker reads --mount as one CSV record, so a comma or quote in a path
// must be quoted. A Windows path's drive-letter colon needs nothing.
func TestBindMountQuotesAPathWithACommaOrQuote(t *testing.T) {
	for _, src := range []string{`C:\Users\dev\.valve-node-app\erpc.yaml`, `C:\Users\Ann, B\.valve-node-app\erpc.yaml`, `/home/o'neil/x.yaml`, `/Users/a "b"/x.yaml`} {
		got := bindMount(src, "/erpc.yaml")
		rec, err := csv.NewReader(strings.NewReader(got)).Read()
		if err != nil {
			t.Fatalf("%q: not one CSV record: %v", got, err)
		}
		want := []string{"type=bind", "source=" + src, "target=/erpc.yaml", "readonly"}
		if strings.Join(rec, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("%q parsed as %q, want %q", got, rec, want)
		}
	}
}

func TestERPCRunArgsMountsTheConfigFile(t *testing.T) {
	args := ERPCRunArgs(ERPCRunSpec{HostConfigPath: `C:\Users\dev\.valve-node-app\erpc.yaml`, Platform: "linux/amd64"})
	if got := valueAfter(t, args, "--mount"); got != `type=bind,source=C:\Users\dev\.valve-node-app\erpc.yaml,target=/erpc.yaml,readonly` {
		t.Fatalf("--mount %q", got)
	}
	for _, a := range args {
		if a == "-v" {
			t.Fatalf("a file bind still uses -v: %q", args)
		}
	}
}
```

```go
// internal/ops/httpprobe_test.go
package ops

import (
	"context"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// The shell forms are the strings gateway.go, devnet.go and traffic.go built
// by hand before Task 15. SSH targets and the existing fakes see them
// unchanged.
func TestHTTPProbeCurlCommandIsTheHistoricString(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`
	for _, tc := range []struct {
		p    HTTPProbe
		want string
	}{
		{HTTPProbe{URL: "http://127.0.0.1:4000/main/evm/1", Body: body, MaxTime: 10 * time.Second},
			`curl -s --max-time 10 -X POST -H 'Content-Type: application/json' --data '` + body + `' 'http://127.0.0.1:4000/main/evm/1'`},
		{HTTPProbe{URL: "http://127.0.0.1:8545", Body: body},
			`curl -s -X POST -H 'Content-Type: application/json' --data '` + body + `' 'http://127.0.0.1:8545'`},
		{HTTPProbe{URL: "http://127.0.0.1:4001/metrics", MaxTime: 5 * time.Second},
			`curl -s --max-time 5 'http://127.0.0.1:4001/metrics'`},
		{HTTPProbe{URL: "https://rpc.lan:8443/main/evm/1", Body: body, MaxTime: 10 * time.Second, Resolve: "rpc.lan:8443:127.0.0.1", CAFile: "/h/ca.crt"},
			`curl -s --max-time 10 -X POST -H 'Content-Type: application/json' --data '` + body + `' --resolve 'rpc.lan:8443:127.0.0.1' --cacert '/h/ca.crt' 'https://rpc.lan:8443/main/evm/1'` +
				` || curl -s --max-time 10 -X POST -H 'Content-Type: application/json' --data '` + body + `' --resolve 'rpc.lan:8443:127.0.0.1' 'https://rpc.lan:8443/main/evm/1'`},
	} {
		if got := tc.p.CurlCommand(); got != tc.want {
			t.Errorf("CurlCommand:\n got %s\nwant %s", got, tc.want)
		}
	}
}

// Local: in process, the name resolved to the given address, the CA read
// with e.ReadFile, and no shell.
func TestHTTPProbeDoLocalResolvesAndTrustsTheCAFile(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || string(b) != `{"q":1}` {
			http.Error(w, "bad request", 400)
			return
		}
		_, _ = w.Write([]byte(`{"result":"0x171"}`))
	}))
	defer ts.Close()
	f := argvfake.New()
	f.Files["/h/ca.crt"] = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw})
	addr := strings.TrimPrefix(ts.URL, "https://")
	// httptest's certificate is valid for example.com; resolve it to the server.
	f.Route("127.0.0.1:8443", addr)
	p := HTTPProbe{URL: "https://example.com:8443/x", Body: `{"q":1}`, Resolve: "example.com:8443:127.0.0.1", CAFile: "/h/ca.crt", MaxTime: 5 * time.Second}
	got, err := p.Do(context.Background(), f)
	if err != nil || got != `{"result":"0x171"}` {
		t.Fatalf("got %q, %v", got, err)
	}
	if len(f.ShellCalls()) != 0 {
		t.Fatalf("shell used: %q", f.ShellCalls())
	}
}

// curl's `--cacert X || plain` fallback: a CA file that does not sign the
// server's certificate is retried once against the system roots. Neither
// signs httptest's certificate here, so the result is one ProbeError naming
// the TLS failure, not a panic or a hang.
func TestHTTPProbeDoLocalFallsBackToSystemRootsThenReportsTheTLSFailure(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }))
	defer ts.Close()
	f := argvfake.New()
	other := httptest.NewTLSServer(http.NotFoundHandler())
	other.Close()
	f.Files["/h/ca.crt"] = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: other.Certificate().Raw})
	u, _ := url.Parse(ts.URL)
	f.Route(u.Host, u.Host)
	_, err := HTTPProbe{URL: ts.URL, CAFile: "/h/ca.crt", MaxTime: 5 * time.Second}.Do(context.Background(), f)
	var pe *ProbeError
	if !errors.As(err, &pe) || !strings.Contains(pe.Detail, "certificate") {
		t.Fatalf("got %v, want a ProbeError about the certificate", err)
	}
}

// A refused connection is a ProbeError, so a readiness loop keeps polling.
func TestHTTPProbeDoLocalRefusedIsAProbeError(t *testing.T) {
	_, err := HTTPProbe{URL: "http://127.0.0.1:1/"}.Do(context.Background(), argvfake.New())
	var pe *ProbeError
	if !errors.As(err, &pe) {
		t.Fatalf("got %v, want *ProbeError", err)
	}
}
```

Update the existing `-v` assertions so they expect the `--mount` form:
- `docker_test.go:33`: the expected argv slice becomes `"--mount", "type=bind,source=/var/lib/valve-node-app/369/erpc.yaml,target=/erpc.yaml,readonly"`.
- `docker_test.go:412-426`: `valueAfter(t, args, "--mount")`, equal to `bindMount(<path>, "/erpc.yaml")`.
- `docker_test.go:999`: the substring becomes `source=/home/o/.valve-node-app/erpc.yaml,target=/erpc.yaml,readonly`.
- `docker_test.go:1073`: the substring becomes `type=bind,source=/home/o/.valve-node-app/Caddyfile,target=/etc/caddy/Caddyfile,readonly`.
- `internal/setup/gateway_test.go:519`: the substring becomes `'type=bind,source=/Users/dev/.valve-node-app/erpc.yaml,target=/erpc.yaml,readonly'`.

Run: `go test ./internal/ops/`
Expected: FAIL: `undefined: bindMount`, `HTTPProbe`, `ProbeError`, `dockerDesktopInstalled`, `WindowsContainersHint`; the argv tests fail because `DockerRun` calls `Run`.

- [ ] **Step 7: Implement the ops side**

In `internal/ops/docker.go`:

```go
// The probes as Commands: argv for the local machine, and for SSH the exact
// strings the constants above have always been.
const (
	dockerInfoFormat     = "{{.ServerVersion}}|{{.OSType}}|{{.Architecture}}|{{.Name}}|{{.OperatingSystem}}"
	enginePlatformFormat = "{{.Server.Os}}/{{.Server.Arch}}"
)

var (
	// Presence: `command -v` has no argv equivalent, but `docker --version`
	// exits 127 from RunArgv when there is no docker. Callers read only the
	// exit code, on which the two forms agree.
	dockerPresenceCmd = executor.Command{Argv: []string{"docker", "--version"}, Shell: dockerPresenceProbe}
	dockerVersionCmd  = executor.Command{Argv: []string{"docker", "--version"}, Shell: dockerVersionProbe}
	dockerInfoCmd     = executor.Command{Argv: []string{"docker", "info", "--format", dockerInfoFormat}, Shell: dockerInfoProbe}
	enginePlatformCmd = executor.Command{Argv: []string{"docker", "version", "--format", enginePlatformFormat}, Shell: enginePlatformProbe}
)
```

Redefine `dockerInfoProbe` as `"docker info --format '" + dockerInfoFormat + "'"`, and `enginePlatformProbe` (in `lifecycle.go`) as `"docker version --format '" + enginePlatformFormat + "'"`. They stay byte-identical.

`ProbeDocker` changes:
- Use `executor.Exec(ctx, e, dockerPresenceCmd, nil)`, then the banner and info `Command`s, wherever it called `e.Run(ctx, <probe>, nil)`.
- `DockerAbsentError.Probe` is `"docker --version"` when `e` is an `executor.ArgvRunner`, and `dockerPresenceProbe` otherwise.
- After the flavor is detected:

```go
	// Docker Desktop in Windows-container mode answers `docker info` with
	// the Windows daemon's details, which never mention Desktop. On this
	// machine, its installed program says so instead (spec D35). The fix
	// differs: Desktop can switch to Linux containers; Docker Engine on
	// Windows Server cannot.
	if info.WindowsContainers() {
		if h, ok := e.(executor.LocalHost); ok && h.HostGOOS() == "windows" && dockerDesktopInstalled() {
			info.Flavor = FlavorDockerDesktop
		}
	}
```

with

```go
// dockerDesktopInstalled reports whether Docker Desktop is installed on this
// Windows machine. A package var so tests on any OS can answer it.
var dockerDesktopInstalled = func() bool {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(pf, "Docker", "Docker", "Docker Desktop.exe"))
	return err == nil
}

// WindowsContainersHint is what to do about an engine in Windows-container
// mode, which cannot run jumpgate's Linux images. Docker Desktop can switch;
// Docker Engine on Windows Server (Moby, what GitHub's Windows runners have)
// cannot run Linux containers at all, so telling its user to "switch" would
// send them looking for a menu that does not exist.
func (d DockerInfo) WindowsContainersHint() string {
	if d.Flavor == FlavorDockerDesktop {
		return `jumpgate's gateway and devnet are Linux images: switch Docker Desktop to Linux containers (right-click the Docker icon in the notification area, then "Switch to Linux containers…") and retry`
	}
	return "this Docker engine runs Windows containers only, and jumpgate's gateway and devnet need Linux containers: install Docker Desktop, or put them on a Linux machine added with `jumpgate hosts add NAME --ssh`"
}
```

`EnginePlatform`: replace the `e.Run(ctx, unameArchProbe, nil)` block with `host := hostArch(ctx, e)`:

```go
// hostArch is the second opinion EnginePlatform needs: on the local machine
// LocalHost.NativeArch, which reads the CPU without a shell; on a remote
// one, unameArchProbe as before.
func hostArch(ctx context.Context, e executor.Executor) string {
	if h, ok := e.(executor.LocalHost); ok {
		return PlatformForArch(h.NativeArch(ctx))
	}
	if res, err := e.Run(ctx, unameArchProbe, nil); err == nil && res.ExitCode == 0 {
		return PlatformForArch(firstNonEmptyLine(res.Stdout))
	}
	return ""
}
```

`DockerRun`:

```go
// DockerRun runs `docker args...` on e: as argv on the local machine, with
// no shell (spec D29), and over SSH as the single-quoted string it has always
// been (executor.QuoteArgv).
func DockerRun(ctx context.Context, e executor.Executor, args ...string) (executor.Result, error) {
	argv := append([]string{"docker"}, args...)
	res, err := executor.Exec(ctx, e, executor.Command{Argv: argv}, nil)
	if err != nil {
		return res, fmt.Errorf("ops: %s: %w", executor.QuoteArgv(argv), err)
	}
	return res, nil
}
```

`ImageExists` uses `executor.Exec` with `Argv: {"docker", "image", "inspect", tag, "--format", "{{.Id}}"}` and `Shell:` today's string. `BuildImage` calls `DockerRun(ctx, e, args...)` and keeps its error wording. In `lifecycle.go`, `readEmulation` uses `executor.Exec(ctx, e, enginePlatformCmd, nil)`.

```go
// internal/ops/mount.go
package ops

import (
	"encoding/csv"
	"strings"
)

// bindMount renders a read-only file bind as a --mount value (spec D33).
// --mount, not -v, for two reasons. First, a Windows source path's drive
// letter (C:\…) is a colon that -v has to guess about. Second, --mount
// refuses a source that does not exist, where -v quietly bind-mounts an
// empty directory in its place. On colima that produced the crash loop the
// gap analysis saw live ("read /erpc.yaml: is a directory"). docker parses
// the value as one CSV record, so a comma or a quote in a path is quoted.
func bindMount(src, dst string) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"type=bind", "source=" + src, "target=" + dst, "readonly"})
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n")
}
```

In `ERPCRunArgs`, replace `"-v", spec.HostConfigPath+":"+erpcContainerConfigPath+":ro"` with `"--mount", bindMount(spec.HostConfigPath, erpcContainerConfigPath)`. In `CaddyRunArgs`, make the Caddyfile and the cert and key binds `"--mount", bindMount(…)`. The data volume keeps `"-v", volume+":"+catalog.CaddyDataPath`.

```go
// internal/ops/httpprobe.go
package ops

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// HTTPProbe is one HTTP request a check makes FROM the target, to a service
// on the target: a gateway's eth_chainId, a devnet's block number, the
// metrics page. It has two renderings that must mean the same request: the
// curl command an SSH target runs, unchanged from before Task 15, and an
// in-process request on the local machine, which needs neither curl nor a
// shell (spec D31).
type HTTPProbe struct {
	URL string
	// Body, when set, is POSTed as application/json; otherwise the probe is a GET.
	Body string
	// Resolve, when set, is curl's --resolve value "name:port:addr": the
	// URL's name:port is connected at addr:port, so a TLS front is probed by
	// its hostname without that name resolving.
	Resolve string
	// CAFile, when set, is a PEM file on the target trusted for the server's
	// certificate. A request that fails with it is retried once with the
	// system roots, as the curl form's `|| …` does.
	CAFile string
	// MaxTime bounds the request; 0 leaves it to ctx.
	MaxTime time.Duration
}

// ProbeError is a probe that ran and got no answer: curl exited non-zero,
// or the in-process request failed. A readiness loop keeps polling on it;
// any other error from Do is the executor failing.
type ProbeError struct{ Detail string }

func (e *ProbeError) Error() string { return e.Detail }

// CurlCommand renders p as the command an SSH target runs. The byte layout
// is the one gateway.go, devnet.go and traffic.go each built by hand before.
func (p HTTPProbe) CurlCommand() string {
	base := []string{"curl -s"}
	if p.MaxTime > 0 {
		base = append(base, fmt.Sprintf("--max-time %d", int(p.MaxTime/time.Second)))
	}
	if p.Body != "" {
		base = append(base, "-X POST -H 'Content-Type: application/json' --data "+shQuote(p.Body))
	}
	if p.Resolve != "" {
		base = append(base, "--resolve "+shQuote(p.Resolve))
	}
	attempt := func(ca string) string {
		parts := append([]string(nil), base...)
		if ca != "" {
			parts = append(parts, "--cacert "+shQuote(ca))
		}
		return strings.Join(append(parts, shQuote(p.URL)), " ")
	}
	if p.CAFile == "" {
		return attempt("")
	}
	return attempt(p.CAFile) + " || " + attempt("")
}

// Do runs p on e and returns the response body, whatever the HTTP status
// (as `curl -s` does). On the local machine the request is made in process,
// dialled through executor.LocalHost. Anywhere else the curl form runs on the
// target.
func (p HTTPProbe) Do(ctx context.Context, e executor.Executor) (string, error) {
	h, ok := e.(executor.LocalHost)
	if !ok {
		res, err := e.Run(ctx, p.CurlCommand(), nil)
		if err != nil {
			return "", err
		}
		if res.ExitCode != 0 {
			return "", &ProbeError{Detail: fmt.Sprintf("curl exit %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))}
		}
		return res.Stdout, nil
	}
	var roots *x509.CertPool
	if p.CAFile != "" {
		if pemBytes, err := e.ReadFile(ctx, p.CAFile); err == nil {
			roots = x509.NewCertPool()
			if !roots.AppendCertsFromPEM(pemBytes) {
				roots = nil
			}
		}
	}
	body, err := p.local(ctx, h, roots)
	if err != nil && roots != nil {
		body, err = p.local(ctx, h, nil)
	}
	if err != nil {
		return "", &ProbeError{Detail: err.Error()}
	}
	return body, nil
}

func (p HTTPProbe) local(ctx context.Context, h executor.LocalHost, roots *x509.CertPool) (string, error) {
	from, to := "", ""
	if parts := strings.SplitN(p.Resolve, ":", 3); len(parts) == 3 {
		from = net.JoinHostPort(parts[0], parts[1])
		to = net.JoinHostPort(strings.Trim(parts[2], "[]"), parts[1])
	}
	tr := &http.Transport{
		// A probe of this machine's own port never goes through a proxy.
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if from != "" && addr == from {
				addr = to
			}
			return h.DialContext(ctx, network, addr)
		},
		TLSClientConfig:   &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
		DisableKeepAlives: true,
	}
	defer tr.CloseIdleConnections()
	method, rd := http.MethodGet, io.Reader(nil)
	if p.Body != "" {
		method, rd = http.MethodPost, strings.NewReader(p.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.URL, rd)
	if err != nil {
		return "", err
	}
	if p.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Transport: tr, Timeout: p.MaxTime}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return string(b), err
}
```

Run: `go test ./internal/ops/ && GOOS=windows go vet ./internal/ops/`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/ops
git commit -m "feat(ops): run docker as argv on the local machine; in-process HTTP probes; --mount file binds; a Windows-container hint per engine

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 9: Write the failing setup tests**

```go
// internal/setup/shellless_test.go
package setup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// rpcServer answers eth_chainId with chain and eth_blockNumber with 5.
func rpcServer(t *testing.T, chain string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req struct{ Method string }
		_ = json.Unmarshal(b, &req)
		result := chain
		if req.Method == "eth_blockNumber" {
			result = "0x5"
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"` + result + `"}`))
	}))
	t.Cleanup(ts.Close)
	return ts
}

// windowsDesktop is a Windows controller whose Docker Desktop runs Linux
// containers, and whose container (by name) is already running, so the
// port checks defer to it the way they do for our own live container.
func windowsDesktop() *argvfake.Fake {
	return argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0, build bde2b89\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|linux|x86_64|docker-desktop|Docker Desktop\n"}).
		Script("docker image inspect", executor.Result{Stdout: "sha256:abc\n"}).
		Script("docker inspect -f {{.State.Running}}", executor.Result{Stdout: "true\n"})
}

// The whole devnet plan runs on a machine with no shell at all.
func TestDevnetPlanNeedsNoShell(t *testing.T) {
	f := windowsDesktop()
	d := testDevnet()
	f.Route("127.0.0.1:"+itoa(d.HTTP()), strings.TrimPrefix(rpcServer(t, "0x539").URL, "http://"))
	steps, err := PlanDevnet(d)
	if err != nil {
		t.Fatal(err)
	}
	if err := RunAll(context.Background(), f, steps, &State{}); err != nil {
		t.Fatalf("devnet on a shell-less controller: %v", err)
	}
	if sh := f.ShellCalls(); len(sh) != 0 {
		t.Fatalf("the devnet plan used a shell: %q", sh)
	}
}

// The whole docker-backend gateway plan, too. The config lands under the
// Windows home, and docker is given that path in a --mount.
func TestGatewayPlanNeedsNoShell(t *testing.T) {
	f := windowsDesktop()
	g := testGateway()
	f.Route("127.0.0.1:4100", strings.TrimPrefix(rpcServer(t, "0x171").URL, "http://"))
	steps := mustPlanGateway(t, g, BackendDocker)
	if err := RunAll(context.Background(), f, steps, &State{}); err != nil {
		t.Fatalf("gateway on a shell-less controller: %v", err)
	}
	if sh := f.ShellCalls(); len(sh) != 0 {
		t.Fatalf("the gateway plan used a shell: %q", sh)
	}
	cfg := filepath.Join(f.Home, ".valve-node-app", "erpc.yaml")
	if _, ok := f.Files[cfg]; !ok {
		t.Fatalf("erpc.yaml not written at %s; files: %v", cfg, keys(f.Files))
	}
	var run []string
	for _, a := range f.Argvs() {
		if len(a) > 1 && a[0] == "docker" && a[1] == "run" {
			run = a
		}
	}
	if !strings.Contains(strings.Join(run, " "), "--mount type=bind,source="+cfg+",target=/erpc.yaml,readonly") {
		t.Fatalf("docker run %q does not bind %s", run, cfg)
	}
}

// The message the gap analysis found unreachable now reaches a Windows user.
func TestPreflight_WindowsContainerModeReachesAShellLessController(t *testing.T) {
	f := argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|WIN-RUNNER|Microsoft Windows Server 2022 Datacenter\n"})
	for name, step := range map[string]Step{
		"gateway": stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight"),
		"devnet":  stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight"),
	} {
		err := step.Verify(context.Background(), f, &State{})
		// "Linux containers" is in both hints; which one is ops' concern
		// (TestWindowsContainersHintNamesTheRightFix), and it depends on
		// whether Docker Desktop is installed on the machine running this.
		if err == nil || !strings.Contains(err.Error(), "Linux containers") {
			t.Errorf("%s: %v, want the Windows-container hint", name, err)
		}
	}
	if sh := f.ShellCalls(); len(sh) != 0 {
		t.Fatalf("preflight used a shell: %q", sh)
	}
}

func TestGatewayPreflight_SystemdOnAShellLessControllerNamesTheDockerBackend(t *testing.T) {
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendSystemd), "preflight")
	err := step.Verify(context.Background(), argvfake.New(), &State{})
	if err == nil || !strings.Contains(err.Error(), `"docker" backend`) {
		t.Fatalf("got %v, want a refusal naming the docker backend", err)
	}
}

// D34: the TLS front would mount a certificate file at its own host path
// inside a Linux container, and a Windows path cannot be one.
func TestGatewayPreflight_RefusesCertFilesOnWindows(t *testing.T) {
	g := testGateway()
	g.TLS = &catalog.GatewayTLS{Enabled: true, Hostname: "rpc.lan", CertSource: catalog.CertFiles, CertFile: `C:\certs\rpc.pem`, KeyFile: `C:\certs\rpc.key`}
	step := stepByID(t, mustPlanGateway(t, g, BackendDocker), "preflight")
	err := step.Verify(context.Background(), windowsDesktop(), &State{})
	if err == nil || !strings.Contains(err.Error(), "certificate files") {
		t.Fatalf("got %v, want the cert-files refusal", err)
	}
}
```

Add the two small helpers this file uses to it, importing `strconv` and `sort`: `func itoa(n int) string { return strconv.Itoa(n) }` and `func keys(m map[string][]byte) []string` (the sorted keys). `testDevnet`, `mustPlanDevnet`, `mustPlanGateway`, `stepByID` and `testGateway` already exist in `devnet_test.go` and `gateway_test.go`. `testDevnet()` is chain 1337 on 18545 and 18546, so the routed RPC answers `0x539`.

```go
// internal/setup/listeners_test.go
package setup

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

func TestProbeListeners_LocalDialsInsteadOfAShell(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	f := argvfake.New().Route("127.0.0.1:4000", ln.Addr().String())
	found, err := probeListeners(context.Background(), f, 4000)
	if err != nil || !strings.Contains(found, "127.0.0.1:4000") {
		t.Fatalf("busy port: %q, %v", found, err)
	}
	if found, err := probeListeners(context.Background(), f, 4001); err != nil || found != "" {
		t.Fatalf("free port: %q, %v", found, err)
	}
	if len(f.ShellCalls()) != 0 {
		t.Fatalf("shell used: %q", f.ShellCalls())
	}
}
```

```go
// internal/setup/targetpath_test.go
package setup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// Local paths follow this machine's rules (C:\Users\… on Windows); remote
// ones stay POSIX.
func TestPathsFollowTheMachineActedOn(t *testing.T) {
	f := argvfake.New()
	home, err := homeOn(context.Background(), f)
	if err != nil || home != f.Home {
		t.Fatalf("homeOn = %q, %v", home, err)
	}
	if got := joinOn(f, home, ".valve-node-app", "erpc.yaml"); got != filepath.Join(f.Home, ".valve-node-app", "erpc.yaml") {
		t.Fatalf("joinOn(local) = %q", got)
	}
	remote := newFakeExecutor().script(`printf '%s\n' "$HOME"`, executor.Result{Stdout: "/home/o\n"})
	if home, err := homeOn(context.Background(), remote); err != nil || home != "/home/o" {
		t.Fatalf("homeOn(remote) = %q, %v", home, err)
	}
	if got := joinOn(remote, "/home/o", ".valve-node-app", "erpc.yaml"); got != "/home/o/.valve-node-app/erpc.yaml" {
		t.Fatalf("joinOn(remote) = %q", got)
	}
}
```

```go
// internal/setup/docker_live_test.go
package setup

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// TestLocalDockerLive runs the devnet plan through the real local executor
// against this machine's real engine: RunArgv, the in-process probes, the
// whole path a Windows controller takes. With a Windows-container engine
// (GitHub's Windows runners, a Windows Server VM) it asserts the message
// that user sees instead. Set JUMPGATE_DOCKER_LIVE=1 to run it.
func TestLocalDockerLive(t *testing.T) {
	if os.Getenv("JUMPGATE_DOCKER_LIVE") != "1" {
		t.Skip("set JUMPGATE_DOCKER_LIVE=1 to run against this machine's Docker engine")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	e := executor.NewLocal()
	info, err := ops.ProbeDocker(ctx, e)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !info.DaemonReachable {
		t.Fatalf("no engine answered: %s", info.DaemonError)
	}
	d := catalog.DevnetConfig{ContainerName: "jumpgate-live-devnet", HTTPPort: freePort(t), WSPort: freePort(t)}
	steps, err := PlanDevnet(d)
	if err != nil {
		t.Fatal(err)
	}
	if info.WindowsContainers() {
		err := RunAll(ctx, e, steps, &State{})
		if err == nil || !strings.Contains(err.Error(), "Linux containers") {
			t.Fatalf("a Windows-container engine: %v, want the switch-to-Linux-containers refusal", err)
		}
		t.Logf("refused as a Windows user will see it: %v", err)
		return
	}
	t.Cleanup(func() { _ = ops.RemoveContainer(context.Background(), e, d.Name()) })
	if err := RunAll(ctx, e, steps, &State{}); err != nil {
		t.Fatalf("devnet through the local executor: %v", err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
```

Run: `go test ./internal/setup/`
Expected: FAIL: `undefined: probeListeners`, `homeOn`, `joinOn`. `TestGatewayPlanNeedsNoShell` and `TestDevnetPlanNeedsNoShell` fail with the shell calls listed (`printf`, `listenerProbe`, `curl`).

- [ ] **Step 10: Implement the setup side**

```go
// internal/setup/targetpath.go
package setup

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// homeOn is the home directory of the user jumpgate acts as on the target:
// read from this process on the local machine (no shell needed), and from
// the target's shell anywhere else.
func homeOn(ctx context.Context, e executor.Executor) (string, error) {
	if h, ok := e.(executor.LocalHost); ok {
		return h.HomeDir()
	}
	res, err := e.Run(ctx, `printf '%s\n' "$HOME"`, nil)
	if err != nil {
		return "", err
	}
	home := strings.TrimSpace(res.Stdout)
	if res.ExitCode != 0 || home == "" {
		return "", fmt.Errorf("$HOME is empty on the target (exit %d)", res.ExitCode)
	}
	return home, nil
}

// joinOn and dirOn apply the path rules of the machine e acts on: this
// machine's (filepath, so C:\Users\… on Windows) for the local executor;
// POSIX for a remote one, whose paths always are (executor/remotepath.go).
func joinOn(e executor.Executor, elem ...string) string {
	if _, ok := e.(executor.LocalHost); ok {
		return filepath.Join(elem...)
	}
	return path.Join(elem...)
}

func dirOn(e executor.Executor, p string) string {
	if _, ok := e.(executor.LocalHost); ok {
		return filepath.Dir(p)
	}
	return path.Dir(p)
}
```

```go
// internal/setup/listeners.go
package setup

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// localDialTimeout bounds one loopback connect in probeListeners.
var localDialTimeout = 500 * time.Millisecond

// probeListeners reports what already listens on port on the target, or ""
// when nothing does. On the local machine it connects to the loopback
// addresses directly (spec D32), because ss, netstat and lsof sit behind a
// shell. A listener on a wildcard or loopback address accepts that connect,
// and those are the only listeners a 127.0.0.1 publish collides with. For a
// wildcard publish, docker still fails loudly on a real collision.
// Anywhere else it runs listenerProbe as before.
func probeListeners(ctx context.Context, e executor.Executor, port int) (string, error) {
	h, ok := e.(executor.LocalHost)
	if !ok {
		res, err := e.Run(ctx, fmt.Sprintf(listenerProbe, port), nil)
		if err != nil {
			return "", err
		}
		if res.ExitCode == 0 {
			return strings.TrimSpace(res.Stdout), nil
		}
		return "", nil
	}
	for _, host := range []string{"127.0.0.1", "::1"} {
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		dctx, cancel := context.WithTimeout(ctx, localDialTimeout)
		c, err := h.DialContext(dctx, "tcp", addr)
		cancel()
		if err == nil {
			c.Close()
			return "something accepts connections on " + addr, nil
		}
	}
	return "", nil
}
```

`gateway.go`:
- `configPath`'s docker branch:

```go
	home, err := homeOn(ctx, e)
	if err != nil {
		return "", fmt.Errorf("gateway: could not resolve the home directory on the target: %w — the docker backend keeps erpc.yaml there because it must be a path the engine can bind-mount", err)
	}
	p.dockerConfigPath = joinOn(e, home, gatewayHomeDir, p.configFileName())
```

- `siblingPath` returns `joinOn(e, dirOn(e, cfg), name)`. The systemd branch of `configPath` keeps `path.Join`, because systemd is always a Linux target.
- `probePort`: replace the `e.Run(ctx, fmt.Sprintf(listenerProbe, port), nil)` block with `found, err := probeListeners(ctx, e, port)`, then branch on `found != ""`. The error and event wording is unchanged.
- In `preflight`'s `BackendDocker` case:

```go
		if info.WindowsContainers() {
			return fmt.Errorf("preflight: this docker engine is in Windows-container mode, and the eRPC image is a Linux image — %s", info.WindowsContainersHint())
		}
		...
		// D34: the TLS front mounts a certificate file at its own host path
		// inside a Linux container, and a Windows path (C:\…) cannot be one.
		if h, ok := e.(executor.LocalHost); ok && h.HostGOOS() == "windows" && p.fronted() && p.gw.TLS.CertSourceOrDefault() == catalog.CertFiles {
			return fmt.Errorf("preflight: a gateway on this Windows computer cannot use certificate files yet (%s): choose the internal or ACME certificate source, or host the gateway on a Linux machine", p.gw.TLS.CertFile)
		}
```

- `requireLinuxRoot`: before running `uname`, refuse a shell-less executor:

```go
	if err := executor.RequireShell(e); err != nil {
		return fmt.Errorf("preflight: a systemd gateway runs on a Linux host, and this computer has no POSIX shell (%v) — use the %q backend here, or a Linux machine added with --ssh", err, BackendDocker)
	}
```

- `gatewayCheck` and `probeCommand`: `probeCommand` becomes `probe(ctx, e, chainID) (string, ops.HTTPProbe, error)`. It returns an `ops.HTTPProbe{URL: url, Body: gatewayChainIDCall, MaxTime: 10 * time.Second}`. When fronted, it adds `Resolve: fmt.Sprintf("%s:%d:%s", tls.Hostname, tls.HTTPS(), probeHost(tls.Bind()))` and `CAFile: ca`, only when the cert source is not ACME, exactly as today. `gatewayCheck` then does:

```go
	raw, err := pr.Do(ctx, e)
	var pe *ops.ProbeError
	switch {
	case errors.As(err, &pe):
		return fmt.Errorf("gateway: eth_chainId at %s failed (%s)", url, pe.Detail)
	case err != nil:
		return fmt.Errorf("gateway: eth_chainId probe: %w", err)
	}
	raw = strings.TrimSpace(raw)
```

  The JSON handling below that is unchanged.

`devnet.go`:
- The Windows-container message becomes `"preflight: this docker engine is in Windows-container mode, and the reth image is a Linux image — " + info.WindowsContainersHint()`.
- `checkPortsFree` uses `probeListeners`.
- `rpcCall` builds `ops.HTTPProbe{URL: url, Body: body}` and maps a `*ops.ProbeError` to `fmt.Errorf("devnet: rpc at %s failed (%s)", url, pe.Detail)`.

`traffic.go`:
- `ReadGatewaySamples` builds `ops.HTTPProbe{URL: url, MaxTime: trafficScrapeTimeout * time.Second}`, as a GET. Convert the constant if it is an untyped int.
- It maps a `*ops.ProbeError` to `fmt.Errorf("traffic: %s did not answer (%s) — the gateway publishes its counters on loopback only, so this is read on the machine it runs on", url, pe.Detail)`.

`tls.go` needs no change: `exportRootCA` already uses `ops.DockerRun` and `e.WriteFile`.

Existing tests that assert `curl exit N` still pass: over a shell fake, `ProbeError.Detail` starts with `curl exit N:`.

Run: `go test ./internal/setup/ ./internal/ops/ ./internal/executor/...`
Expected: PASS.

- [ ] **Step 11: Commit**

```bash
git add internal/setup
git commit -m "feat(setup): gateway and devnet run on a controller with no shell; Windows-container mode reaches the user

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 12: Write the failing server tests**

Append to `internal/server/docker_test.go`:

```go
// A Windows controller can now probe its engine (it used to answer 500).
// An engine in Windows-container mode is up but unusable: not running, and
// with the hint, so the panel's "not running" path shows it and does not
// start provisioning (spec D35).
func TestDockerStatusWindowsContainers(t *testing.T) {
	f := argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|WIN-RUNNER|Microsoft Windows Server 2022 Datacenter\n"})
	a := newDockerTestServer(t, f)
	got := decodeJSON[dockerStatusResponse](t, a.do(t, "GET", "/api/docker", nil))
	if !got.Present || got.Running || !got.WindowsContainers || got.CanStart || !strings.Contains(got.Hint, "Linux containers") {
		t.Fatalf("got %+v, want present, not running, windowsContainers, the hint, no auto-start", got)
	}
}

// "This computer" can be added on any OS. On Windows it has no shell, and
// RequireShell is what the shell routes check (Task 13).
func TestDefaultExecutorBuildsALocalTargetEverywhere(t *testing.T) {
	ex, err := defaultNewExecutor(config.Target{ID: "me", Mode: "local"})
	if err != nil {
		t.Fatalf("local target refused on %s: %v", runtime.GOOS, err)
	}
	shellErr := executor.RequireShell(ex)
	if (runtime.GOOS == "windows") != errors.Is(shellErr, executor.ErrNoPOSIXShell) {
		t.Fatalf("RequireShell on %s = %v", runtime.GOOS, shellErr)
	}
}
```

Append to `internal/server/containers_test.go`:

```go
func TestContainerListShowsTheWindowsContainerHint(t *testing.T) {
	f := argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|WIN-RUNNER|Microsoft Windows Server 2022 Datacenter\n"})
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return f, nil })
	if res := a.do(t, "POST", "/api/targets", map[string]any{"id": "me", "mode": "local"}); res.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d", res.StatusCode)
	}
	got := decodeJSON[containersResponse](t, a.do(t, "GET", "/api/targets/me/containers", nil))
	if !strings.Contains(got.Docker.Hint, "Linux containers") {
		t.Fatalf("docker view %+v, want the Windows-container hint", got.Docker)
	}
}
```

Append to `internal/server/gateways_test.go` (create the file if it does not exist):

```go
// Windows: the trust-store command needs an elevated prompt, not sudo.
func TestTrustNeedsRootMessageOnWindows(t *testing.T) {
	if got := trustNeedsRootMessage("windows", "me"); !strings.Contains(got, "Run as administrator") {
		t.Fatalf("got %q", got)
	}
	if got := trustNeedsRootMessage("linux", "box"); !strings.Contains(got, "sudo") {
		t.Fatalf("got %q", got)
	}
}
```

Run: `go test ./internal/server/`
Expected: FAIL: `got.WindowsContainers undefined`; `defaultNewExecutor` refuses on Windows CI; `undefined: trustNeedsRootMessage`.

- [ ] **Step 13: Implement the server side**

`api.go`, in the `"local"` case of `defaultNewExecutor`:

```go
	case "local":
		// "This computer" exists on every OS. On one with no POSIX shell
		// (Windows) it runs the Docker gateway and devnet through RunArgv
		// (spec D29). The routes that need a shell refuse it through
		// getShellExecutor (Task 13), so it is not refused here any more.
		return executor.NewLocal(), nil
```

`docker.go`:

```go
	// WindowsContainers is true when an engine answered but runs Windows
	// containers, which cannot run jumpgate's Linux images. Running is then
	// false: the engine is up but unusable, and the UI's not-running path
	// shows Hint without trying to start or provision anything (spec D35).
	WindowsContainers bool `json:"windowsContainers,omitempty"`
```

and in `handleDockerStatus`'s `default:` branch:

```go
		resp.Present = info.Present
		resp.WindowsContainers = info.WindowsContainers()
		resp.Running = info.DaemonReachable && !resp.WindowsContainers
		switch {
		case resp.WindowsContainers:
			resp.Hint = info.WindowsContainersHint()
		case resp.Present && !resp.Running:
			resp.Hint = dockerStartHint
		}
	}
	resp.CanStart = runtime.GOOS == "darwin" && resp.Present && !resp.Running && !resp.WindowsContainers
```

`containers.go` `probeDockerView`: `v.Hint = info.WindowsContainersHint()` in the Windows-container case.

`gateways.go`: factor the needs-root message into a helper, and use it where `install.NeedsRoot && !targetIsRoot(…)` is handled:

```go
// trustNeedsRootMessage is what a person is told when installing the root
// needs privileges jumpgate does not have. Windows has no sudo: certutil
// -addstore ROOT needs a prompt opened with "Run as administrator".
func trustNeedsRootMessage(goos, hostID string) string {
	if goos == "windows" {
		return fmt.Sprintf("installing a root certificate needs an administrator on machine %q. Open a Command Prompt with \"Run as administrator\" and run:", hostID)
	}
	return fmt.Sprintf("installing a root certificate needs root on machine %q. Run this on it (e.g. with sudo):", hostID)
}
```

Run: `go test ./... && GOOS=windows go vet ./... && GOOS=linux GOARCH=arm64 go vet ./...`
Expected: PASS.

- [ ] **Step 14: Commit**

```bash
git add internal/server
git commit -m "feat(server): add this computer on Windows for Docker features; report Windows-container mode as not ready, with the fix

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 15: Live check on this Mac (colima)**

The macOS and Linux local executors now take the argv path too. That is deliberate (D29): the code a Windows controller runs is exercised daily on the machines developers use.

```bash
colima status
JUMPGATE_DOCKER_LIVE=1 go test -v -count=1 -run '^TestLocalDockerLive$' ./internal/setup/
```

Expected: PASS. The log shows the devnet plan's steps, and `docker ps -a | grep jumpgate-live-devnet` is empty afterwards.

- [ ] **Step 16: CI jobs and the Windows smoke script**

Add to `.github/workflows/ci.yml`, after the `e2e` job:

```yaml
  # The shell-free Docker path (Task 15) against a real engine. Ubuntu's
  # runner has Docker with Linux containers, so the devnet runs end to end
  # through RunArgv and the in-process probes.
  docker-live:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: devnet through the local executor
        env:
          JUMPGATE_DOCKER_LIVE: "1"
        run: go test -v -count=1 -run '^TestLocalDockerLive$' ./internal/setup/

  # GitHub's Windows runners have Docker Engine in Windows-container mode
  # only. That is exactly the state whose message a Windows user must see,
  # so this job proves the message reaches them: through the plan, through
  # the API, and through a real jumpgate.exe.
  windows-docker:
    runs-on: windows-latest
    defaults:
      run:
        shell: pwsh
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: the engine is in Windows-container mode
        run: |
          $os = docker info --format '{{.OSType}}'
          if ($os -ne 'windows') { throw "expected a Windows-container engine on this runner, got '$os'; the runner image changed, so revisit this job" }
      - name: the plan refuses with the hint
        env:
          JUMPGATE_DOCKER_LIVE: "1"
        run: go test -v -count=1 -run '^TestLocalDockerLive$' ./internal/setup/
      - name: a real jumpgate.exe reports it
        run: ./scripts/windows-docker-smoke.ps1 -Expect windows-containers
```

Create `scripts/windows-docker-smoke.ps1`:

```powershell
# Drives a real jumpgate.exe through the local Docker features (plan Task 15).
#
#   -Expect windows-containers  The engine runs Windows containers: GitHub's
#                               runners, or a Windows Server VM with Moby.
#                               /api/docker and the containers view must both
#                               carry the Linux-containers hint.
#   -Expect linux-containers    Docker Desktop in Linux mode, on a real
#                               Windows desktop. /api/docker must say running,
#                               with no hint. Provisioning itself is covered by
#                               TestLocalDockerLive, run on the same machine.
#
#   -Exe C:\path\jumpgate.exe   Use this binary instead of building one, for a
#                               machine without Go (the QEMU VM).
param(
  [Parameter(Mandatory)][ValidateSet('windows-containers', 'linux-containers')][string]$Expect,
  [string]$Exe
)
$ErrorActionPreference = 'Stop'
$tmp = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { $env:TEMP }
$work = Join-Path $tmp ("jg-docker-smoke-" + [guid]::NewGuid().ToString('N').Substring(0, 8))
$home_ = Join-Path $work 'home'
New-Item -ItemType Directory -Force -Path $home_ | Out-Null
if (-not $Exe) {
  $Exe = Join-Path $work 'jumpgate.exe'
  go build -o $Exe ./cmd/jumpgate
  if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
}
$env:USERPROFILE = $home_
$env:HOME = $home_
$port = 18799
$srv = Start-Process -FilePath $Exe -ArgumentList 'serve', '--bind', "127.0.0.1:$port" -PassThru -NoNewWindow `
  -RedirectStandardError (Join-Path $work 'serve.err') -RedirectStandardOutput (Join-Path $work 'serve.out')
try {
  $infoFile = Join-Path $home_ '.jumpgate\run\server.json'
  $info = $null
  for ($i = 0; $i -lt 60 -and -not $info; $i++) {
    Start-Sleep -Seconds 1
    if (Test-Path $infoFile) { $info = Get-Content $infoFile -Raw | ConvertFrom-Json }
  }
  if (-not $info) { throw "server.json never appeared: $(Get-Content (Join-Path $work 'serve.err') -Raw)" }
  $h = @{ Authorization = "Bearer $($info.token)" }
  $base = "http://127.0.0.1:$port"

  $docker = Invoke-RestMethod "$base/api/docker" -Headers $h
  Invoke-RestMethod "$base/api/targets" -Method Post -Headers $h -ContentType 'application/json' `
    -Body '{"id":"me","mode":"local"}' | Out-Null
  $list = Invoke-RestMethod "$base/api/targets/me/containers" -Headers $h

  if ($Expect -eq 'windows-containers') {
    if (-not $docker.windowsContainers -or $docker.running -or $docker.hint -notmatch 'Linux containers') {
      throw "GET /api/docker: $($docker | ConvertTo-Json -Compress)"
    }
    if ($list.docker.hint -notmatch 'Linux containers') {
      throw "containers view: $($list.docker | ConvertTo-Json -Compress)"
    }
    Write-Output "windows-container mode reaches the user: $($docker.hint)"
  } else {
    if (-not $docker.running -or $docker.hint) { throw "GET /api/docker: $($docker | ConvertTo-Json -Compress)" }
    if ($list.docker.hint) { throw "containers view has a hint: $($list.docker.hint)" }
    Write-Output 'Linux containers: ready'
  }
} finally {
  Stop-Process -Id $srv.Id -Force -ErrorAction SilentlyContinue
}
```

```bash
git add .github/workflows/ci.yml scripts/windows-docker-smoke.ps1
git commit -m "ci: run the shell-free Docker path live on Linux, and assert the Windows-container message on Windows

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: `docker-live` and `windows-docker` are green, alongside every existing job. `windows-docker` needs Task 6's Windows server fixes on the branch. If Task 6 has not merged yet, the third step fails at `server.json`: wait for Task 6, then re-run.

- [ ] **Step 17: Manual checks (record each result in the PR description)**

1. **QEMU Windows Server VM.** It has no nested virtualisation, so it can run Windows containers with process isolation, but not Linux containers. Cross-build on the Mac:

   ```bash
   GOOS=windows go build -o /tmp/jg/jumpgate.exe ./cmd/jumpgate
   GOOS=windows go test -c -o /tmp/jg/setup.test.exe ./internal/setup/
   ```

   Copy both files to `C:\jg`, then:
   - **Moby running:** run `scripts\windows-docker-smoke.ps1 -Expect windows-containers -Exe C:\jg\jumpgate.exe`. Then set `$env:JUMPGATE_DOCKER_LIVE=1` and run `C:\jg\setup.test.exe -test.run TestLocalDockerLive -test.v`. Both pass.
   - **`Stop-Service docker`:** `GET /api/docker` says present, not running, with the start hint.
   - **docker.exe renamed away:** it says not present, with the install hint.
2. **A physical Windows 11 machine with Docker Desktop (WSL 2).** A cloud VM with nested virtualisation also works.
   - **Linux mode:** `-Expect linux-containers`, and `setup.test.exe -test.run TestLocalDockerLive` provisions the devnet end to end. In the app, provision the devnet and a gateway in front of it. Check that `curl.exe -s -X POST -H "Content-Type: application/json" --data "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_chainId\",\"params\":[]}" http://127.0.0.1:4000/main/evm/1337` answers `0x539`.
   - **"Switch to Windows containers…":** `-Expect windows-containers` passes, and the hint names Docker Desktop's switch menu, because Desktop is recognised by its installed exe.

---

### Task 16: A prebuilt, pinned, multi-arch eRPC image

Covers gap W2. Also covers W1's misleading build error: BuildKit is detected before building, and the last lines of output are shown. Decisions: D37–D40.

**Files:**
- Create: `internal/catalog/erpc.go`, `internal/catalog/erpc_test.go`, `.github/workflows/erpc-image.yml`, `scripts/erpc-pin.sh`, `scripts/erpc-verify.sh`
- Modify: `internal/ops/docker.go` (pin constants become aliases; `PullImage`, `ImageLabel`, `BuildKitAvailable`, `lastLines`; `ImageBuildArgs` labels; `BuildImage` error detail), `internal/ops/docker_test.go`, `internal/setup/gateway.go` (`ensureImage`, `runDocker`), `internal/setup/gateway_test.go`, `internal/setup/docker_live_test.go`, `.github/workflows/release.yml`, `README.md`

**Interfaces:**
- Consumes (Task 15): `ops.DockerRun`, `executor.Exec`, `argvfake`, `TestLocalDockerLive`, the `docker-live` CI job.
- Produces (package `catalog`):
  - `const ERPCSourceRepo`, `ERPCSourceRef`, `ERPCImageRepo`, `ERPCImageDigest`
  - `func ERPCImageRef() string` returns `ERPCImageRepo + "@" + ERPCImageDigest`
  - `func ERPCImageSourceTag() string` returns `ERPCImageRepo + ":src-" + ERPCSourceRef`
- Produces (package `ops`):
  - `const LabelRevision = "org.opencontainers.image.revision"`
  - `func PullImage(ctx context.Context, e executor.Executor, ref, platform string) error`
  - `func ImageLabel(ctx context.Context, e executor.Executor, ref, label string) (string, bool)`
  - `func BuildKitAvailable(ctx context.Context, e executor.Executor) bool`
- Produces (package `setup`): `(*gatewayPlan).ensureImage(ctx, e, st, platform) (string, error)`, which returns the image reference to run.
- Produces (scripts): `scripts/erpc-pin.sh` prints `ref=`, `short=`, `repo=` and `digest=` lines. `scripts/erpc-verify.sh` exits 0 only when the pin matches the published, signed, two-platform image.

- [ ] **Step 1: Pin file, pin script and publishing workflow; publish the first image**

```go
// internal/catalog/erpc.go
package catalog

// The eRPC gateway image (spec D37). The fork's source is pinned by commit,
// and the image built from it is pinned by the digest of its multi-arch
// index, so every controller of a release runs byte-identical gateway code.
// The digest is compiled in, and the binary carrying it is itself covered by
// the release's checksums.txt (D38).
//
// To move to a new eRPC commit:
//  1. set ERPCSourceRef;
//  2. push: .github/workflows/erpc-image.yml publishes
//     ERPCImageRepo:src-<ref>, signs it, and fails, printing the digest;
//  3. set ERPCImageDigest to that digest and push again: the same workflow
//     verifies it and goes green.
//
// scripts/erpc-pin.sh reads these four lines with sed: keep each a one-line
// `Name = "value"`.
const (
	ERPCSourceRepo  = "https://github.com/valve-tech/erpc.git"
	ERPCSourceRef   = "a7a53ec21a7922c4c6d8582e3466331b1a7cc622"
	ERPCImageRepo   = "ghcr.io/jumpgate-tech/erpc"
	ERPCImageDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
)

// ERPCImageRef is the image a gateway runs: pulled and run by digest.
func ERPCImageRef() string { return ERPCImageRepo + "@" + ERPCImageDigest }

// ERPCImageSourceTag is the published tag for ERPCSourceRef. jumpgate accepts
// it, after a failed pull, only when its revision label names
// ERPCSourceRef; it is how an air-gapped machine is given the image (D39).
func ERPCImageSourceTag() string { return ERPCImageRepo + ":src-" + ERPCSourceRef }
```

In `internal/ops/docker.go`, `ERPCSourceRepo` and `ERPCSourceRef` become `= catalog.ERPCSourceRepo` and `= catalog.ERPCSourceRef`. Their doc comments point at `catalog/erpc.go`. `ERPCImageTag()` stays: it is now the local-build tag, `valve-node-app/erpc:<ref8>`.

```bash
# scripts/erpc-pin.sh
#!/usr/bin/env bash
# Prints the eRPC image pin from internal/catalog/erpc.go as key=value lines,
# for $GITHUB_OUTPUT and for eval. TestERPCPinScriptReadsTheCatalog keeps this
# parser and the Go constants in step.
set -euo pipefail
f="$(cd "$(dirname "$0")/.." && pwd)/internal/catalog/erpc.go"
get() { sed -n "s/^[[:space:]]*$1[[:space:]]*=[[:space:]]*\"\([^\"]*\)\".*/\1/p" "$f"; }
ref="$(get ERPCSourceRef)"; repo="$(get ERPCImageRepo)"; digest="$(get ERPCImageDigest)"
if [ -z "$ref" ] || [ -z "$repo" ] || [ -z "$digest" ]; then
  echo "erpc-pin: could not read the pin from $f" >&2
  exit 1
fi
printf 'ref=%s\nshort=%s\nrepo=%s\ndigest=%s\n' "$ref" "${ref:0:8}" "$repo" "$digest"
```

```bash
# scripts/erpc-verify.sh
#!/usr/bin/env bash
# Checks the published eRPC image against the catalog's pin: both platforms,
# the revision label, a keyless signature from this repo's erpc-image
# workflow, and finally that the pinned digest is the published one. The
# last check runs last, so a fresh publish fails here with the digest to
# commit. Needs docker (buildx), jq and cosign.
set -euo pipefail
eval "$("$(dirname "$0")/erpc-pin.sh")"
tag="$repo:src-$ref"
manifest="$(docker buildx imagetools inspect "$tag" --format '{{json .Manifest}}')"
published="$(jq -r .digest <<<"$manifest")"
platforms="$(jq -r '[.manifests[].platform | select(.os != "unknown") | "\(.os)/\(.architecture)"] | sort | join(",")' <<<"$manifest")"
if [ "$platforms" != "linux/amd64,linux/arm64" ]; then
  echo "erpc-verify: $tag has platforms '$platforms', want linux/amd64,linux/arm64" >&2; exit 1
fi
for p in linux/amd64 linux/arm64; do
  rev="$(docker buildx imagetools inspect "$tag" --format '{{json .Image}}' | jq -r --arg p "$p" '.[$p].config.Labels["org.opencontainers.image.revision"] // empty')"
  if [ "$rev" != "$ref" ]; then
    echo "erpc-verify: $tag ($p) is labelled revision '$rev', want $ref" >&2; exit 1
  fi
done
cosign verify \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp '^https://github.com/jumpgate-tech/jumpgate-app/\.github/workflows/erpc-image\.yml@' \
  "$repo@$published" >/dev/null
if [ "$digest" != "$published" ]; then
  echo "::error file=internal/catalog/erpc.go::ERPCImageDigest is $digest, but $tag is $published. Set ERPCImageDigest = \"$published\"."
  exit 1
fi
echo "erpc-verify: $repo@$published is $tag, signed, linux/amd64 + linux/arm64"
```

```yaml
# .github/workflows/erpc-image.yml
# Publishes the eRPC gateway image jumpgate pins (spec D37–D40), and verifies
# the pin against it.
#
# For the commit in internal/catalog/erpc.go: if ghcr.io/jumpgate-tech/erpc:
# src-<ref> does not exist yet (or `force` is set), build it natively for
# amd64 and arm64, merge the two into one index, tag it src-<ref> and <ref8>,
# and sign the index keylessly. Then verify (scripts/erpc-verify.sh), which
# fails with the digest to commit when the catalog does not pin it yet.
name: eRPC image

on:
  push:
    branches: [main, 'feat/**']
    paths:
      - internal/catalog/erpc.go
      - .github/workflows/erpc-image.yml
      - scripts/erpc-pin.sh
      - scripts/erpc-verify.sh
  workflow_call:
  workflow_dispatch:
    inputs:
      force:
        description: Rebuild and re-tag although src-<ref> exists (the catalog digest must then be updated)
        type: boolean
        default: false

permissions:
  contents: read
  packages: write
  id-token: write

concurrency:
  group: erpc-image
  cancel-in-progress: false

jobs:
  pin:
    runs-on: ubuntu-24.04
    outputs:
      ref: ${{ steps.pin.outputs.ref }}
      short: ${{ steps.pin.outputs.short }}
      repo: ${{ steps.pin.outputs.repo }}
      publish: ${{ steps.exists.outputs.publish }}
    steps:
      - uses: actions/checkout@v4
      - id: pin
        run: scripts/erpc-pin.sh >> "$GITHUB_OUTPUT"
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - id: exists
        run: |
          # Never overwrite a published source tag by accident: controllers
          # already pin its digest. `force` is the deliberate override.
          if docker buildx imagetools inspect "${{ steps.pin.outputs.repo }}:src-${{ steps.pin.outputs.ref }}" >/dev/null 2>&1 \
             && [ "${{ inputs.force }}" != "true" ]; then
            echo publish=false >> "$GITHUB_OUTPUT"
          else
            echo publish=true >> "$GITHUB_OUTPUT"
          fi

  build:
    needs: pin
    if: needs.pin.outputs.publish == 'true'
    strategy:
      matrix:
        include:
          - arch: amd64
            os: ubuntu-24.04
          - arch: arm64
            os: ubuntu-24.04-arm
    runs-on: ${{ matrix.os }}
    steps:
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: build and push by digest
        run: |
          docker buildx build \
            --platform linux/${{ matrix.arch }} \
            --label org.opencontainers.image.revision=${{ needs.pin.outputs.ref }} \
            --label org.opencontainers.image.source=https://github.com/valve-tech/erpc \
            --provenance=false \
            --output type=image,name=${{ needs.pin.outputs.repo }},push-by-digest=true,name-canonical=true,push=true \
            --metadata-file meta.json \
            "https://github.com/valve-tech/erpc.git#${{ needs.pin.outputs.ref }}"
          mkdir -p digests
          jq -r '."containerimage.digest"' meta.json > "digests/${{ matrix.arch }}"
      - uses: actions/upload-artifact@v4
        with:
          name: erpc-digest-${{ matrix.arch }}
          path: digests/${{ matrix.arch }}

  merge:
    needs: [pin, build]
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/download-artifact@v4
        with:
          pattern: erpc-digest-*
          merge-multiple: true
          path: digests
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: sigstore/cosign-installer@v3
      - name: one index, two tags, signed
        run: |
          repo='${{ needs.pin.outputs.repo }}'
          docker buildx imagetools create \
            -t "$repo:src-${{ needs.pin.outputs.ref }}" -t "$repo:${{ needs.pin.outputs.short }}" \
            "$repo@$(cat digests/amd64)" "$repo@$(cat digests/arm64)"
          index="$(docker buildx imagetools inspect "$repo:src-${{ needs.pin.outputs.ref }}" --format '{{json .Manifest}}' | jq -r .digest)"
          cosign sign --yes "$repo@$index"
          echo "Published \`$repo@$index\` for ${{ needs.pin.outputs.ref }}." >> "$GITHUB_STEP_SUMMARY"

  verify:
    needs: [pin, build, merge]
    if: always() && needs.pin.result == 'success' && needs.merge.result != 'failure' && needs.build.result != 'failure'
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: sigstore/cosign-installer@v3
      - run: scripts/erpc-verify.sh
```

```bash
chmod +x scripts/erpc-pin.sh scripts/erpc-verify.sh
scripts/erpc-pin.sh
git add internal/catalog/erpc.go internal/ops/docker.go scripts/erpc-pin.sh scripts/erpc-verify.sh .github/workflows/erpc-image.yml
git commit -m "ci: publish and sign a multi-arch eRPC image for the pinned source commit

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push
gh run watch "$(gh run list --workflow erpc-image.yml --branch "$(git branch --show-current)" --limit 1 --json databaseId -q '.[0].databaseId')"
```

Expected:
- `erpc-pin.sh` prints four lines, with `short=a7a53ec2`.
- The `build` jobs and `merge` succeed. `verify` fails with `ERPCImageDigest is sha256:000…, but ghcr.io/jumpgate-tech/erpc:src-a7a53ec2… is sha256:<64 hex>. Set ERPCImageDigest = "sha256:<64 hex>".`

Copy that digest. Then ask the user to set the package to Public (precondition 2), and confirm the image is public:

```bash
docker logout ghcr.io; docker buildx imagetools inspect ghcr.io/jumpgate-tech/erpc:src-a7a53ec21a7922c4c6d8582e3466331b1a7cc622
```

- [ ] **Step 2: Write the failing pin tests**

```go
// internal/catalog/erpc_test.go
package catalog

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// A malformed pin would only fail at a user's first gateway provision; catch
// it here. The all-zero digest is the "not published yet" value Step 1
// commits, and must never reach a release.
func TestERPCPinIsWellFormed(t *testing.T) {
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(ERPCSourceRef) {
		t.Errorf("ERPCSourceRef %q is not a full commit hash", ERPCSourceRef)
	}
	if !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(ERPCImageDigest) || strings.Trim(ERPCImageDigest[7:], "0") == "" {
		t.Errorf("ERPCImageDigest %q is not a published digest; see the comment in erpc.go", ERPCImageDigest)
	}
	if ERPCImageRef() != ERPCImageRepo+"@"+ERPCImageDigest || !strings.HasPrefix(ERPCImageRepo, "ghcr.io/") {
		t.Errorf("ERPCImageRef %q", ERPCImageRef())
	}
}

// CI reads the pin with scripts/erpc-pin.sh; it must read what Go compiles.
func TestERPCPinScriptReadsTheCatalog(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	out, err := exec.Command("bash", "../../scripts/erpc-pin.sh").Output()
	if err != nil {
		t.Fatal(err)
	}
	want := "ref=" + ERPCSourceRef + "\nshort=" + ERPCSourceRef[:8] + "\nrepo=" + ERPCImageRepo + "\ndigest=" + ERPCImageDigest + "\n"
	if string(out) != want {
		t.Fatalf("erpc-pin.sh printed\n%s\nwant\n%s", out, want)
	}
}
```

Run: `go test ./internal/catalog/ -run ERPCPin`
Expected: FAIL. `ERPCImageDigest "sha256:000…" is not a published digest`.

- [ ] **Step 3: Pin the digest**

Set `ERPCImageDigest` in `internal/catalog/erpc.go` to the digest Step 1 printed.

Run: `go test ./internal/catalog/ -run ERPCPin && scripts/erpc-verify.sh`
Expected: PASS, then `erpc-verify: ghcr.io/jumpgate-tech/erpc@sha256:… is …, signed, linux/amd64 + linux/arm64`. This needs `cosign`, which you can get with `brew install cosign`. Without it, push and let the workflow's `verify` job run it.

```bash
git add internal/catalog
git commit -m "feat(catalog): pin the published eRPC image by digest

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push
```

Expected: the `eRPC image` workflow skips `build` and `merge` (the tag exists), and `verify` is green.

- [ ] **Step 4: Write the failing ensureImage tests**

Append to `internal/setup/gateway_test.go`:

```go
// The ensureImage cases run on argvfake: they read the argv docker got.
func pullFake() *argvfake.Fake {
	return argvfake.New().
		Script("docker image inspect "+catalog.ERPCImageRef(), executor.Result{ExitCode: 1, Stderr: "Error: No such image\n"}).
		Script("docker image inspect "+catalog.ERPCImageSourceTag(), executor.Result{ExitCode: 1}).
		Script("docker image inspect "+ops.ERPCImageTag(), executor.Result{ExitCode: 1})
}

func argvStarting(f *argvfake.Fake, prefix string) []string {
	for _, a := range f.Argvs() {
		if strings.HasPrefix(strings.Join(a, " "), prefix) {
			return a
		}
	}
	return nil
}

func ensure(t *testing.T, f *argvfake.Fake) (string, error) {
	t.Helper()
	p := &gatewayPlan{id: "default", gw: testGateway(), backend: BackendDocker}
	return p.ensureImage(context.Background(), f, &State{}, "linux/amd64")
}

func TestEnsureImage_PinnedPresentSkipsThePull(t *testing.T) {
	f := pullFake().Script("docker image inspect "+catalog.ERPCImageRef(), executor.Result{Stdout: "sha256:abc\n"})
	ref, err := ensure(t, f)
	if err != nil || ref != catalog.ERPCImageRef() || argvStarting(f, "docker pull") != nil {
		t.Fatalf("ref %q, err %v, argvs %q", ref, err, f.Argvs())
	}
}

func TestEnsureImage_PullsByDigestForThePlatform(t *testing.T) {
	f := pullFake()
	ref, err := ensure(t, f)
	pull := argvStarting(f, "docker pull")
	if err != nil || ref != catalog.ERPCImageRef() || strings.Join(pull, " ") != "docker pull --platform linux/amd64 "+catalog.ERPCImageRef() {
		t.Fatalf("ref %q, err %v, pull %q", ref, err, pull)
	}
	if argvStarting(f, "docker build") != nil {
		t.Fatal("built although the pull succeeded")
	}
}

func TestEnsureImage_PullFailsUsesALabelledSourceTag(t *testing.T) {
	f := pullFake().
		Script("docker pull", executor.Result{ExitCode: 1, Stderr: "dial tcp: lookup ghcr.io: no such host\n"}).
		Script("docker image inspect --format {{ index .Config.Labels \"org.opencontainers.image.revision\" }} "+catalog.ERPCImageSourceTag(),
			executor.Result{Stdout: catalog.ERPCSourceRef + "\n"})
	ref, err := ensure(t, f)
	if err != nil || ref != catalog.ERPCImageSourceTag() {
		t.Fatalf("ref %q, err %v", ref, err)
	}
}

func TestEnsureImage_RefusesASourceTagBuiltFromAnotherRef(t *testing.T) {
	f := pullFake().
		Script("docker pull", executor.Result{ExitCode: 1, Stderr: "no such host\n"}).
		Script("docker image inspect --format {{ index .Config.Labels \"org.opencontainers.image.revision\" }} "+catalog.ERPCImageSourceTag(),
			executor.Result{Stdout: "1111111111111111111111111111111111111111\n"}).
		Script("docker buildx version", executor.Result{ExitCode: 1})
	ref, err := ensure(t, f)
	if err == nil || ref == catalog.ERPCImageSourceTag() {
		t.Fatalf("ref %q, err %v; want the mislabelled image refused", ref, err)
	}
}

func TestEnsureImage_PullFailsBuildsWhenBuildKitIsThere(t *testing.T) {
	f := pullFake().
		Script("docker pull", executor.Result{ExitCode: 1, Stderr: "no such host\n"}).
		Script("docker buildx version", executor.Result{Stdout: "github.com/docker/buildx v0.19.0\n"})
	ref, err := ensure(t, f)
	build := strings.Join(argvStarting(f, "docker build"), " ")
	if err != nil || ref != ops.ERPCImageTag() || !strings.Contains(build, "--label "+ops.LabelRevision+"="+catalog.ERPCSourceRef) {
		t.Fatalf("ref %q, err %v, build %q", ref, err, build)
	}
}

// No network, no BuildKit: one message naming every way out, quickly.
func TestEnsureImage_OfflineWithoutBuildKitNamesEveryFix(t *testing.T) {
	f := pullFake().
		Script("docker pull", executor.Result{ExitCode: 1, Stderr: "Error response from daemon: Get \"https://ghcr.io/v2/\": dial tcp: lookup ghcr.io: no such host\n"}).
		Script("docker buildx version", executor.Result{ExitCode: 1, Stderr: "docker: 'buildx' is not a docker command.\n"})
	_, err := ensure(t, f)
	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{"no such host", "docker load", catalog.ERPCImageSourceTag(), "buildx"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q:\n%v", want, err)
		}
	}
	if argvStarting(f, "docker build") != nil {
		t.Fatal("ran the legacy builder, whose error hides the cause")
	}
}

func TestEnsureImage_LocalBuildEnvSkipsThePull(t *testing.T) {
	t.Setenv("JUMPGATE_ERPC_LOCAL_BUILD", "1")
	f := pullFake().Script("docker buildx version", executor.Result{Stdout: "v0.19.0\n"})
	ref, err := ensure(t, f)
	if err != nil || ref != ops.ERPCImageTag() || argvStarting(f, "docker pull") != nil {
		t.Fatalf("ref %q, err %v, argvs %q", ref, err, f.Argvs())
	}
}
```

Append to `internal/ops/docker_test.go`:

```go
// The first stderr line of a failed build is usually "DEPRECATED: The legacy
// builder…" or a progress line; the cause is at the end.
func TestPullImageReportsTheLastLines(t *testing.T) {
	f := argvfake.New().Script("docker pull", executor.Result{ExitCode: 1, Stderr: "Pulling…\nwaiting\nError: denied: permission_denied\n"})
	err := PullImage(context.Background(), f, "ghcr.io/x/y@sha256:ab", "linux/amd64")
	if err == nil || !strings.Contains(err.Error(), "permission_denied") {
		t.Fatalf("got %v", err)
	}
}
```

Also update the existing `ImageBuildArgs` expectations in `docker_test.go`. The argv now carries `"--label", LabelRevision + "=" + ERPCSourceRef` between the platform and `-t`.

Run: `go test ./internal/setup/ ./internal/ops/`
Expected: FAIL. `ensureImage` returns one value, and `PullImage`, `LabelRevision`, `ImageLabel` and `BuildKitAvailable` are undefined.

- [ ] **Step 5: Implement**

In `internal/ops/docker.go`:

```go
// LabelRevision is the OCI label naming the source commit an image was built
// from. Published and locally built eRPC images both carry it, which is how
// ensureImage tells an image built from ERPCSourceRef from one that only
// carries the right tag (spec D39).
const LabelRevision = "org.opencontainers.image.revision"

// PullImage pulls ref for platform. A failure carries the last lines of
// docker's output, where the cause is; the first lines are progress.
func PullImage(ctx context.Context, e executor.Executor, ref, platform string) error {
	res, err := DockerRun(ctx, e, "pull", "--platform", resolveRunPlatform(platform), ref)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("docker pull %s failed (exit %d): %s", ref, res.ExitCode, lastLines(res.Stderr+res.Stdout, 5))
	}
	return nil
}

// ImageLabel reads one label of a local image; ok is false when the image is
// absent or has no such label.
func ImageLabel(ctx context.Context, e executor.Executor, ref, label string) (string, bool) {
	res, err := DockerRun(ctx, e, "image", "inspect", "--format", `{{ index .Config.Labels "`+label+`" }}`, ref)
	if err != nil || res.ExitCode != 0 {
		return "", false
	}
	v := strings.TrimSpace(res.Stdout)
	return v, v != "" && v != "<no value>"
}

// BuildKitAvailable reports whether the engine's CLI has buildx. The eRPC
// Dockerfile uses RUN --mount=type=cache, which the legacy builder rejects
// with a first line ("DEPRECATED: The legacy builder…") that hides the
// cause (gap W1). Checking first lets the error say what to install.
func BuildKitAvailable(ctx context.Context, e executor.Executor) bool {
	res, err := DockerRun(ctx, e, "buildx", "version")
	return err == nil && res.ExitCode == 0
}

// lastLines is the last n non-empty lines of s, joined with " | ".
func lastLines(s string, n int) string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}
```

Changes to existing functions:
- `ImageBuildArgs`: insert `"--label", LabelRevision + "=" + ERPCSourceRef` after `--platform`.
- `BuildImage`'s failure: replace `firstNonEmptyLine(res.Stderr, res.Stdout)` with `lastLines(res.Stderr+res.Stdout, 5)`.

In `internal/setup/gateway.go`:

```go
// erpcPullTimeout/erpcBuildTimeout bound ensureImage's two slow paths, so a
// machine with no network fails with a message rather than hanging. Package
// vars so tests can shrink them.
var (
	erpcPullTimeout  = 10 * time.Minute
	erpcBuildTimeout = 30 * time.Minute
)

// ensureImage makes the eRPC image present on the target and returns the
// reference to run (spec D37–D39). In order:
//  1. the pinned image (repo@digest) is already here: run it;
//  2. pull it by digest (skipped when JUMPGATE_ERPC_LOCAL_BUILD=1, for
//     working on the eRPC fork itself);
//  3. the pull failed: run an image already here that was built from
//     ERPCSourceRef. That is either the published source tag, loaded by hand
//     on an air-gapped machine and checked by its revision label, or an
//     earlier local build;
//  4. build from source, when the engine has BuildKit;
//  5. otherwise fail, naming every way out.
func (p *gatewayPlan) ensureImage(ctx context.Context, e executor.Executor, st *State, platform string) (string, error) {
	pinned := catalog.ERPCImageRef()
	if ok, err := ops.ImageExists(ctx, e, pinned); err != nil {
		return "", fmt.Errorf("run: %w", err)
	} else if ok {
		_ = emit(ctx, st, Event{StepID: "run", Line: "image " + pinned + " already present"})
		return pinned, nil
	}

	var pullErr error
	if os.Getenv("JUMPGATE_ERPC_LOCAL_BUILD") == "1" {
		pullErr = errors.New("JUMPGATE_ERPC_LOCAL_BUILD=1 is set, so the published image was not pulled")
	} else {
		_ = emit(ctx, st, Event{StepID: "run", Line: "pulling " + pinned})
		pctx, cancel := context.WithTimeout(ctx, erpcPullTimeout)
		pullErr = ops.PullImage(pctx, e, pinned, platform)
		cancel()
		if pullErr == nil {
			return pinned, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}

	src := catalog.ERPCImageSourceTag()
	if rev, ok := ops.ImageLabel(ctx, e, src, ops.LabelRevision); ok {
		if rev == catalog.ERPCSourceRef {
			_ = emit(ctx, st, Event{StepID: "run", Line: fmt.Sprintf("%v; using %s, already on this machine", pullErr, src)})
			return src, nil
		}
		_ = emit(ctx, st, Event{StepID: "run", Line: fmt.Sprintf("ignoring %s: it was built from %s, not %s", src, rev, catalog.ERPCSourceRef)})
	}
	if ok, err := ops.ImageExists(ctx, e, ops.ERPCImageTag()); err == nil && ok {
		_ = emit(ctx, st, Event{StepID: "run", Line: fmt.Sprintf("%v; using the earlier local build %s", pullErr, ops.ERPCImageTag())})
		return ops.ERPCImageTag(), nil
	}

	if !ops.BuildKitAvailable(ctx, e) {
		return "", fmt.Errorf("run: the eRPC image is not on this machine and could not be pulled: %v. Any one of these fixes it: "+
			"connect this machine to the network and retry; "+
			"or, on a connected machine, `docker pull %s`, `docker save -o erpc.tar %s`, copy erpc.tar here and `docker load -i erpc.tar`; "+
			"or install BuildKit (the docker-buildx or docker-buildx-plugin package; `brew install docker-buildx` with colima) so jumpgate can build it here",
			pullErr, src, src)
	}
	tag := ops.ERPCImageTag()
	_ = emit(ctx, st, Event{StepID: "run", Line: fmt.Sprintf("%v; building %s from %s (several minutes)", pullErr, tag, ops.ERPCBuildContext())})
	bctx, cancel := context.WithTimeout(ctx, erpcBuildTimeout)
	defer cancel()
	if _, err := ops.BuildImage(bctx, e, ops.ImageBuildArgs(ops.ImageBuildSpec{Tag: tag, Platform: platform})...); err != nil {
		return "", fmt.Errorf("run: %w", err)
	}
	return tag, nil
}
```

In `runDocker`, replace the `ensureImage` call with `image, err := p.ensureImage(ctx, e, st, platform)`, and pass `Image: image` in the `ops.ERPCRunSpec`. Delete the old body. Its comment ("builds the gateway image on the target unless it is already…") is replaced by the one above.

Run: `go test ./internal/setup/ ./internal/ops/ ./internal/catalog/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/ops internal/setup
git commit -m "feat(gateway): pull the pinned eRPC image by digest; build only as a fallback, and say what is missing when neither works

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 7: Extend the live test to the gateway**

In `TestLocalDockerLive`, after the devnet's `RunAll` succeeds, add:

```go
	// The gateway, in front of the devnet, through the pulled image. Both
	// containers join ops.NetworkName, so the upstream is the devnet's
	// container name on that network, which resolves the same way on every
	// engine (no host.docker.internal needed).
	g := catalog.GatewayConfig{
		Port:       freePort(t),
		MetricsOff: true, // the runner's 4001 may be taken; metrics are not under test
		Networks: []catalog.GatewayNetwork{{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
			{ID: "devnet", Endpoint: fmt.Sprintf("http://%s:%d", d.Name(), catalog.DevnetContainerHTTPPort)},
		}}},
	}
	gsteps, err := PlanGateway("live", g, BackendDocker)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ops.RemoveContainer(context.Background(), e, ops.ERPCContainerNameFor("live")) })
	if err := RunAll(ctx, e, gsteps, &State{}); err != nil {
		t.Fatalf("gateway through the pulled image: %v", err)
	}
	if ok, _ := ops.ImageExists(ctx, e, catalog.ERPCImageRef()); !ok {
		t.Fatalf("%s is not present after provisioning; the gateway ran something else", catalog.ERPCImageRef())
	}
```

(Add `fmt` to the imports.)

Run: `JUMPGATE_DOCKER_LIVE=1 go test -v -count=1 -run '^TestLocalDockerLive$' ./internal/setup/`
Expected: PASS on this Mac (colima), and the log shows `pulling ghcr.io/jumpgate-tech/erpc@sha256:…`. On an Intel Mac this takes seconds rather than the 12 minutes the gap analysis measured. Afterwards, `docker ps -a` shows neither live container.

- [ ] **Step 8: Release workflow and README**

In `.github/workflows/release.yml`:

```yaml
  # The eRPC image every controller of this release pins must exist, be
  # signed and match the catalog before anything ships (spec D40). On a
  # release the source tag already exists, so this only verifies.
  erpc-image:
    uses: ./.github/workflows/erpc-image.yml
    permissions:
      contents: read
      packages: write
      id-token: write
```

Add `needs: [erpc-image]` to the `binaries` job. The `desktop` job keeps `needs: [binaries]`, so the gate covers it too. Add the line "eRPC image: verified against `internal/catalog/erpc.go` by the erpc-image job" to the header comment.

In `README.md`, under the gateway section, add:

```markdown
### The gateway image

The RPC gateway runs eRPC from a prebuilt image, `ghcr.io/jumpgate-tech/erpc`, for amd64 and arm64. Each jumpgate release pins one image by its digest, and that image is signed by this repository's release workflow. Check it with:

    cosign verify --certificate-oidc-issuer https://token.actions.githubusercontent.com \
      --certificate-identity-regexp '^https://github.com/jumpgate-tech/jumpgate-app/\.github/workflows/erpc-image\.yml@' \
      ghcr.io/jumpgate-tech/erpc@<digest>

**No network on the gateway machine?** On a connected machine, run `docker pull ghcr.io/jumpgate-tech/erpc:src-<ref>` and `docker save -o erpc.tar ghcr.io/jumpgate-tech/erpc:src-<ref>`. Copy `erpc.tar` across and run `docker load -i erpc.tar`. jumpgate uses the loaded image once its revision label matches. The ref and digest for your version are in `internal/catalog/erpc.go` at that version's tag.

**Working on the eRPC fork?** `JUMPGATE_ERPC_LOCAL_BUILD=1` builds the image from source instead. This needs Docker BuildKit (`docker buildx`).
```

```bash
git add .github/workflows/release.yml README.md
git commit -m "ci(release): verify the pinned eRPC image before releasing; document offline use

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
git push && gh pr checks --watch
```

Expected: every job is green, including `docker-live`, which now pulls the image and runs the gateway on ubuntu-24.04.

- [ ] **Step 9: Run the release workflow without publishing**

```bash
gh workflow run release.yml --ref "$(git branch --show-current)"
gh run watch "$(gh run list --workflow release.yml --limit 1 --json databaseId -q '.[0].databaseId')"
```

Expected: `erpc-image / pin` sees the tag, `build` and `merge` are skipped, and `verify` is green. Then the existing jobs run as in Task 14 Step 3.

- [ ] **Step 10: Manual checks (record each in the PR description)**

1. **No network, on this Mac.** Remove the image with `docker rmi ghcr.io/jumpgate-tech/erpc@<digest>`, then turn Wi-Fi off and provision a gateway in the app. Expected: within seconds, the error from `TestEnsureImage_OfflineWithoutBuildKitNamesEveryFix` (this Mac's colima has no buildx). Turn Wi-Fi on and retry: the image pulls and the gateway answers.
2. **The air-gapped path.** On this Mac, run `docker save` on the `src-<ref>` tag. Then `docker rmi` both references, turn Wi-Fi off, run `docker load`, and provision. Expected: the stream says `using ghcr.io/jumpgate-tech/erpc:src-… already on this machine`.
3. **Windows 11 with Docker Desktop**, the machine from Task 15 Step 17. A gateway provisions by pulling, without buildx and without a 12-minute build.

