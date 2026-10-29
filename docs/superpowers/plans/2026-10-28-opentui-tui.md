# OpenTUI Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A second terminal dashboard in `tui/`, written in Bun and OpenTUI, that reaches every capability in the inventory through the existing server API. Each screen is the plainest one that works, so the user can decide what to keep. The Go TUI (`internal/tui`) stays unchanged.

**Architecture:** One Bun process. `src/api/` is a thin typed client: discovery from `system.json`/`server.json`, `fetch` over the unix socket (`fetch(url, { unix })`) with Bearer or peer-cred auth, and SSE for streams. A route table is checked against the Go `HandleFunc` patterns in CI. `src/shell/` holds the single layout (status strip, nav, main, activity, drawer) and the key router. `src/screens/` has one file per capability group. Each screen is a pure `model -> lines` view plus an `update(key|event)` function, so it can be tested without a terminal. The OpenTUI renderables are thin and get tested with `createTestRenderer`. Actions the API lacks run `jumpgate <args>` while the renderer is suspended, and the child owns the TTY.

**Tech Stack:** Bun 1.4.3 (`bun test`, `bun build --compile`), `@opentui/core` 0.6.2 (published 2026-10-12, engines `bun >=1.3.14`), `@types/bun` 1.4.3, TypeScript 5 (Bun's bundled transpiler, `tsc --noEmit` via `typescript` 5.9.3). No React, no Solid: plain `@opentui/core` renderables. Pin exact versions and commit `bun.lock`.

**Spec:** `docs/superpowers/specs/2026-10-28-opentui-inventory.md` (rows referenced as group letters A–L).

## Global Constraints

- **Secrets.** The TUI never writes a token, passphrase, API key, recovery code or VPN private key to stdout, a log, the activity pane, a file or an error string. A passphrase goes into a masked input (`InputRenderable`, every char drawn as `*`), is read once into a `Uint8Array`, and is zeroed with `.fill(0)` once the request settles, whatever the outcome. The input's value is cleared at the same moment. The TUI stores nothing on disk. A one-time secret the server returns (a new customer API key, a VPN peer config) is shown in a modal once, then dropped from state when the modal closes. `redact()` runs on every error message before it reaches the activity pane. A test (`secrets.test.ts`) greps captured frames and logs for fixture secrets.
- **Text.** User text says **gate**, never **agent**. `vocab.test.ts` scans `src/**/*.ts` string literals for `\bagent\b`. Route paths such as `/agent/upgrade` are the only allowed hits.
- **Scope.** No trend graphs or sparklines; numbers and tables only. No server logic: the TUI shows what the server returns and sends what the user chose.
- **Layout** (as approved for the Go TUI, `docs/superpowers/specs/2026-10-23-tui-dashboard-design.md`): a top status strip (server, version skew, vault state, fleet counts, pending decisions), left nav (groups), main pane, right activity log, and a bottom drawer for progress. `.` toggles the drawer, which can be minimized. The state persists in `UIPrefs.drawer` (`auto|open|min`). The drawer never opens itself. When it is minimized, the line reads `[! N needs decision]`. At 80×24 or larger the right pane folds away below 100 columns. Below 80×24 a "window too small" message replaces the layout.
- **Fail closed.** For an unknown value show `n/a`, for a stale one `~`, for an error `FAIL`. Never show 0 for unknown. Mask URLs with userinfo or keys, the same way as `internal/tui/maskurl.go`.
- **Terminal safety.** Strip control sequences from every server string before drawing it, as in `internal/tui/sanitize.go`.
- **Shell-out.** CLI-only actions run `jumpgate` from `$JUMPGATE_BIN`, falling back to `jumpgate` on `PATH`. The argv is fixed and holds no secret. The renderer is suspended for the run and resumed after it, and the exit code is shown.
- **Exit codes** follow the CLI: 0 ok, 1 failed, 2 usage, 3 unreachable, 4 security.
- Every task ends with `cd tui && bun test && bun run typecheck` green, then a commit that ends with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## File Structure

```
tui/
  package.json            exact pins; scripts: test, typecheck, build, check-routes
  bun.lock
  tsconfig.json
  bunfig.toml             [test] preload = ./test/setup.ts
  src/main.ts             arg parsing (--socket, --help, --version), start renderer
  src/api/discover.ts     find system.json / server.json, return Conn (never logs token)
  src/api/http.ts         request(): fetch over unix socket, error -> ApiError{code,message}
  src/api/sse.ts          stream(): SSE parse, reconnect with backoff
  src/api/routes.ts       ROUTES: readonly list of "METHOD /path" used by the client
  src/api/types.ts        hand-written TS mirrors of internal/api structs actually read
  src/api/client.ts       one typed function per route
  src/secret.ts           withSecret(buf, fn), redact()
  src/shellout.ts         runJumpgate(args, renderer)
  src/shell/layout.ts     layoutFor(w,h,drawer) -> rects (pure)
  src/shell/app.ts        App: renderables per region, key routing, screen switching
  src/shell/statusbar.ts  status strip text (pure)
  src/shell/activity.ts   ring buffer of activity lines (pure)
  src/shell/drawer.ts     running jobs / pending decisions text (pure)
  src/shell/confirm.ts    yes/no modal, passphrase modal, show-once modal
  src/screens/<group>.ts  one per group: connect, vault, fleet, node, jobs, keys,
                          hosts, modules, gateways, vpn, settings, setup
  test/setup.ts           fake server (Bun.serve on a temp unix socket) + fixtures
  test/fixtures/*.json    captured from the Go api types (Task 2)
  test/*.test.ts
  scripts/check-routes.ts compares ROUTES against Go HandleFunc patterns
```

---

## Phase 0: Skeleton and client

### Task 0: Scaffold `tui/`

**Files:** Create `tui/package.json`, `tui/tsconfig.json`, `tui/bunfig.toml`, `tui/src/main.ts`, `tui/test/smoke.test.ts`, `tui/.gitignore` (`node_modules/`, `dist/`).

- [ ] **Step 1:** Write `package.json`:
  ```json
  {
    "name": "jumpgate-tui", "private": true, "type": "module",
    "packageManager": "bun@1.4.3",
    "scripts": {
      "test": "bun test",
      "typecheck": "tsc --noEmit",
      "check-routes": "bun run scripts/check-routes.ts",
      "build": "bun build src/main.ts --compile --minify --outfile dist/jumpgate-tui"
    },
    "dependencies": { "@opentui/core": "0.6.2" },
    "devDependencies": { "@types/bun": "1.4.3", "typescript": "5.9.3" }
  }
  ```
- [ ] **Step 2:** Write the failing test `test/smoke.test.ts`: `bun run src/main.ts --version` exits 0 and prints `jumpgate-tui <version>`.
- [ ] **Step 3:** Implement `main.ts --version` and `--help`.
- [ ] **Step 4:** `cd tui && bun install --frozen-lockfile=false && bun test && bun run typecheck && bun run build && ./dist/jumpgate-tui --version`
- [ ] **Step 5:** Commit `feat(tui): scaffold the Bun + OpenTUI dashboard`.

### Task 1: Discovery and transport

**Files:** Create `src/api/discover.ts`, `src/api/http.ts`, `test/discover.test.ts`, `test/http.test.ts`, `test/setup.ts`.

**Behaviour** (mirrors `internal/daemon.Find`):
1. `$JUMPGATE_SYSTEM_DIR`, else `/run/jumpgate-ctl/system.json`. If it exists, the result is `Conn{socket, token: null, system: true}`. EACCES means `NotAdmin` ("you are not in jumpgate-admin").
2. Otherwise `$JUMPGATE_HOME`, else `~/.jumpgate/run/server.json`, which must be mode 0600 and owned by the user (refuse otherwise, as `tokenfile_unix.go` does). The result is `Conn{socket, token, httpAddr}`.
3. Probe `GET /api/health`. On failure, return `NotRunning`, and the connect screen offers "start server" (shell-out `jumpgate serve` detached, then poll).
4. Windows: `fetch` unix sockets are unverified, so fall back to `httpAddr` with Bearer (open question 6).

- [ ] **Step 1:** Tests: system.json wins; a server.json with mode 0644 is refused; a missing token on a non-system connection is an error; `request()` sets `Authorization: Bearer` only when a token exists; `ApiError` carries the server's `code` (`internal/api/errors.go`); and the token appears in no thrown message (`expect(String(err)).not.toContain(token)`).
- [ ] **Step 2:** `test/setup.ts`: `fakeServer(routes)` uses `Bun.serve({ unix: tmp })` and records the requests.
- [ ] **Step 3:** Implement. `request<T>(conn, method, path, body?) : Promise<T>`, `fetch("http://jumpgate"+path, { unix: conn.socket, headers })`.
- [ ] **Step 4:** `bun test test/discover.test.ts test/http.test.ts`
- [ ] **Step 5:** Commit.

### Task 2: Route table checked against Go, types and fixtures

**Files:** Create `src/api/routes.ts`, `src/api/types.ts`, `scripts/check-routes.ts`, `test/routes.test.ts`, `internal/api/tsfixtures_test.go` (Go, test-only), `tui/test/fixtures/*.json`.

- [ ] **Step 1:** `scripts/check-routes.ts` reads `../internal/server/*.go` (skipping `_test.go`) and extracts the `HandleFunc("METHOD /path"` patterns. It fails if a `ROUTES` entry has no Go route, which is a hard error. It prints Go routes the TUI does not use as a warning, and an `--strict` flag makes those an error once Phase 3 is done.
- [ ] **Step 2:** The Go test `TestWriteTSFixtures` runs under `-run TSFixtures` with `JUMPGATE_TS_FIXTURES=../../tui/test/fixtures`. It marshals one populated value of each api type the TUI reads (`Fleet`, `TargetView`, `VaultView`, `JobsInbox`, `JobDetail`, `Settings`, `UIPrefs`, `ControllerView`, `ModulesView`, `GatewaySummary`, `NodeStatus`, ...) to JSON. Without the env var it only checks that marshalling succeeds.
- [ ] **Step 3:** `test/routes.test.ts` has two checks. First, every fixture parses through a tiny structural checker (`assertShape(fixture, sample)`, keys and primitive types) against a sample object typed as the TS interface. A Go field rename then breaks a TS test. Second, `bun run check-routes` exits 0.
- [ ] **Step 4:** `go test ./internal/api -run TSFixtures` with the env var, then `cd tui && bun test`.
- [ ] **Step 5:** Commit `feat(tui): typed route table checked against the Go mux`.

### Task 3: SSE and secrets helpers

**Files:** Create `src/api/sse.ts`, `src/secret.ts`, `test/sse.test.ts`, `test/secret.test.ts`.

- [ ] **Step 1:** Tests. SSE: parse `event:`/`data:` frames across chunk splits, reconnect after a drop with 1/2/4/8 s backoff capped at 15 s, and stop on `AbortSignal`. Secrets: `withSecret(buf, fn)` zeroes `buf` on both resolve and reject. `redact()` removes `Bearer …`, 64-hex tokens, `?token=`, `?code=` and URL userinfo.
- [ ] **Step 2:** Implement. `stream<T>(conn, path, signal): AsyncIterable<{event, data:T}>`.
- [ ] **Step 3:** `bun test`. Commit.

### Task 4: Client functions for the slice

**Files:** Create `src/api/client.ts`, `test/client.test.ts`.

The slice needs: `health, version, vault, unlock, lock, fleet, watchFleet, fleetOne, targets, watchStatus, disk, endpoints, firewall, logs, watchLogs, serviceAction, jobs, watchJobs, runningJobs, job, watchJobOutput, decide, adopt, prefs, savePrefs`.

- [ ] **Step 1:** For each function, one test against the fake server for path, method and body. `unlock(conn, pass: Uint8Array)` sends `{"passphrase": "..."}`, builds the body from the bytes, and zeroes the temporary string's buffer copy. The request body is never logged. This is a JS limit: strings cannot be zeroed, so the fake server test checks that no reference escapes and the plan accepts the remaining risk.
- [ ] **Step 2:** Implement and add every path to `ROUTES`.
- [ ] **Step 3:** `bun test && bun run check-routes`. Commit.

---

## Phase 1: Shell and vertical slice (connect, unlock, fleet, node, jobs)

### Task 5: `layoutFor`

**Files:** Create `src/shell/layout.ts`, `test/layout.test.ts`.

- [ ] **Step 1:** Tests, ported from `internal/tui/layout_test.go`. At 80×24 with the drawer open, main keeps at least 8 rows and the drawer shrinks to 2 lines. Below 100 columns there is no activity pane. Below 80×24 the result is `tooSmall`. `auto` resolves to `min` when `h < 34`.
- [ ] **Step 2:** Implement as pure rects. `bun test`. Commit.

### Task 6: App shell

**Files:** Create `src/shell/app.ts`, `src/shell/statusbar.ts`, `src/shell/activity.ts`, `src/shell/drawer.ts`, `src/shell/confirm.ts`, `test/shell.test.ts`.

- [ ] **Step 1:** Tests use `createTestRenderer({ width: 100, height: 30 })` from `@opentui/core/testing` and `captureCharFrame()`. Check that the frame shows all five regions. `.` toggles the drawer and calls `savePrefs({drawer})`, and when the save is refused it flashes and re-reads. `.` typed into a focused input types a dot. Tab and arrow keys move nav. `q`/Ctrl-C quits and restores the terminal. `?` lists keys.
- [ ] **Step 2:** Implement. The status strip reads `server <ver> [skew!] | vault <state> | boxes N ok/M warn/K down | ! N decisions`. The activity pane holds the last 200 lines, each passed through `redact()`.
- [ ] **Step 3:** `confirm.ts` has three modals: `confirm(text)`, `askSecret(prompt)` and `showOnce(title, value)`. `askSecret(prompt)` is a masked input that resolves to a `Uint8Array` and clears its own value. `showOnce` draws the value, then on close blanks it and drops the reference.
- [ ] **Step 4:** `bun test`. Commit.

### Task 7: Connect screen (A)

**Files:** Create `src/screens/connect.ts`, `test/screens/connect.test.ts`.

- [ ] Tests: when the server is not running, the screen shows "No jumpgate server is running. [s] start it". `s` shells out to `jumpgate serve`, which `shellout.ts` detaches, then polls health for up to 10 s. When the user is not an admin, it names the group. On skew, the strip shows `[skew!]` and the screen shows both versions. On success, the app goes to Fleet.
- [ ] Implement `src/shellout.ts` (`runJumpgate(args, {detach?})`, renderer `suspend()`/`resume()`), with a test that uses a fake `JUMPGATE_BIN` script recording argv.
- [ ] Commit.

### Task 8: Vault unlock and lock (B, slice part)

**Files:** Create `src/screens/vault.ts`, `test/screens/vault.test.ts`.

- [ ] Tests: `u` opens `askSecret("vault passphrase")`, and the fake server receives it. Afterwards the input value is `""` and the bytes are zeroed (spy on `withSecret`). A wrong passphrase shows the server's message and backoff. `l` locks after a confirm. The frame never contains the passphrase (`secrets.test.ts` pattern).
- [ ] Implement. Commit.

### Task 9: Fleet list (D, slice part)

**Files:** Create `src/screens/fleet.ts`, `test/screens/fleet.test.ts`.

- [ ] Tests: the fleet fixture renders one row per box with columns `name  gate  sync  peers  disk  state`. A `fleet/stream` update replaces the rows, and a dropped stream marks the values `~`. Enter opens node detail. Missing fields show `n/a`.
- [ ] Implement. Commit.

### Task 10: Node detail (E, slice part)

**Files:** Create `src/screens/node.ts`, `test/screens/node.test.ts`.

- [ ] Tabs as plain text sections, switched with 1–5: status (`status/stream`), disk (`d` measure), endpoints (masked), firewall checklist, and logs (`logs/stream`, sanitized). `s`/`S`/`r` start, stop or restart a service after a confirm.
- [ ] Tests per section from fixtures. Escape sequences in log lines are stripped. A masked URL hides userinfo.
- [ ] Commit.

### Task 11: Jobs and decisions (G, C)

**Files:** Create `src/screens/jobs.ts`, `test/screens/jobs.test.ts`.

- [ ] An inbox list (`jobs/stream`). Enter shows detail: the steps, then output (`output/stream`). `a`/`x` approve or refuse a pending step after a confirm that shows the step text, and `A` adopts. The drawer shows running jobs and `[! N needs decision]`.
- [ ] Tests: decide sends `{decision:"approve"}` (exact shape from the `api.JobDecision` fixture). The minimized drawer line counts pending decisions. The drawer never opens itself on a new decision.
- [ ] Commit `feat(tui): vertical slice: connect, unlock, fleet, node, jobs`.

### Task 12: CI and build for the slice

**Files:** Modify `.gitlab-ci.yml`, `.github/workflows/ci.yml`.

- [ ] Add the GitLab job (git.craftsmanfounder.com, shared docker runners, no `tags:`):
  ```yaml
  tui:
    stage: test
    image: oven/bun:1.4.3
    cache:
      key: { files: [tui/bun.lock] }
      paths: [tui/node_modules/]
    script:
      - cd tui
      - bun install --frozen-lockfile
      - bun run typecheck
      - bun test
      - bun run check-routes
      - bun run build
      - ./dist/jumpgate-tui --version
  ```
- [ ] Add the GitHub job `tui` (ubuntu-24.04, `actions/checkout@v4`, `oven-sh/setup-bun@v2` with `bun-version: 1.4.3`), with the same script.
- [ ] Add a Go-side check to the `go-linux` job: `go test ./internal/api -run TSFixtures` (it already runs in `./...`). Then, from the fixtures directory, run `git diff --exit-code tui/test/fixtures` after regenerating with the env var, so a Go type change forces a fixture refresh.
- [ ] Push to GitLab only when the user asks. Locally run `cd tui && bun install --frozen-lockfile && bun test && bun run check-routes && bun run build`.
- [ ] Commit `ci(tui): test and build the OpenTUI dashboard`.

---

## Phase 2: The rest, group by group

Each task below is one screen file plus its test file, its client functions, and their `ROUTES` entries. The pattern is the same: fixture-driven view tests, then fake-server action tests (method, path, body), then a confirm on anything destructive.

### Task 13: Hosts (D, rest)

`src/screens/hosts.ts`. **Add box:** a form (name, user@host[:port], jump), then `hostkeys/probe`, then a fingerprint shown in full with `confirm`, then `hostkeys/confirm`, then `POST /api/targets`, then `pair` streaming into the drawer. **Add this machine** shells out to `jumpgate hosts add NAME --local` (sudo prompt on the TTY). Also **Invite** (from `pairing-info`, in the CLI's format: port the test from `cli_hosts_invite_test.go`'s expected line), **Upgrade gate** (`agent/upgrade`), **Remove** (confirm), **Forget host key** (`hostkeys` list, then `forget`), **SSH** (`fleet/{id}/ssh`, then suspend, run the argv, resume), **Rename legacy names** (confirm, then start the `node.migrate-names` job), and **orphans** (list from `/api/targets`, then `DELETE /api/orphans/{name}`).

### Task 14: Vault administration (B, rest)

Extends `src/screens/vault.ts`. **Idle lock** minutes go through `PUT /api/settings`, validated to 0–1440. **Put secret** reads `askSecret`, sends `PUT /api/vault/secrets/{relay-token|admin-token}`, then zeroes the buffer. Every shell-out action has a fixed argv, and the child reads its own passphrase: `vault init`, `vault migrate`, `vault backup FILE` (path input, not secret), `vault restore FILE`, `vault add-recovery` (the code prints on the real terminal before the TUI resumes, with a note telling the user to copy it before pressing Enter), `vault rekdf`, `vault seal --tpm2|--allow-host-key` (Linux only, otherwise hidden). Test: the argv recorded by the fake bin holds no secret.

### Task 15: Signers and keys (C)

`src/screens/keys.ts`. Shows `GET /api/controller`. **Test signing** on the selected box calls `POST /api/fleet/{id}/check`. **Create key** calls `POST /api/system/init-key` for a system service (passphrase via `askSecret`) or shells out to `jumpgate keys init` for a user server. **Restart server** calls `POST /api/restart`. **Export keystore** shells out to `jumpgate keys export --keystore-v3 FILE`.

### Task 16: Node services and containers (E, rest)

Extends `src/screens/node.ts`. Adds containers (list, status, config view and edit as raw JSON in a text area, action, provision, wipe/reset with confirm), service clear, du by path, the setup wizard (`POST setup`, then `setup/stream` into the drawer), monitor (`monitor/stream`, current numbers only), diagnostics (run and latest), and explain (`POST explain` on the selected log lines; the result is shown as text).

### Task 17: Modules (F)

`src/screens/modules.ts`. Shows the catalog. Per box, a list of modules with on/off and a raw-JSON config editor that sends `PUT /api/targets/{id}/modules/{module}` for `msgboard`, `el-bootnode`, `cl-bootnode`, `watch`, `private-send` and `checkpointz`. The watch module's alerts come from fleet. Test: the request body round-trips the `SetModule` fixture.

### Task 18: Gateways, RPC, customer keys (H)

`src/screens/gateways.ts`. Covers list, get, create (form), delete (confirm), start/stop/restart, config (raw JSON), provision and wipe (confirm), trust cert and TLS verify, capabilities, knownset, and analytics and traffic as tables. The chainlist is a picker. Also Docker status and start, and host info. **Customer keys:** list, revoke (confirm), and create, where the reply's raw key goes to `showOnce` and never to state, the activity pane or a log. Test: after `showOnce` closes, `captureCharFrame()` and the activity buffer do not contain the fixture key.

### Task 19: VPN (I)

`src/screens/vpn.ts`. Clients: list, save (a form whose private-key field is masked and zeroed), delete, up/down, status. Servers: list, create, delete, up/down, status, endpoint. Peers: add, where the returned config goes to `showOnce`, and remove.

### Task 20: Settings and prefs (J)

`src/screens/settings.ts`. Shows `GET /api/settings` as key/value lines and edits one field at a time with `PUT`. The AI provider key is write-only: it shows `set`/`not set` and is entered through `askSecret`. UI prefs cover the drawer setting. Update check reads `GET /api/update`, and `r` refreshes it.

### Task 21: Activity and alerts (K)

`src/shell/activity.ts` gets alert lines derived from fleet updates (a box going down, a stale gate, a pending decision), with the same rules as `internal/tui/alerts.go` (port its table tests).

### Task 22: Setup (L) and server control (A, rest)

`src/screens/setup.ts`. These shell out: `jumpgate setup install-service`, `jumpgate setup server [--self-only]`, and `jumpgate open` (browser login). Hidden when not on Linux where Linux-only. **Stop server** calls `POST /api/shutdown` after a confirm and then returns to the connect screen.

### Task 23: Route coverage gate

- [ ] Turn `check-routes --strict` on in both CI jobs. Each Go route is either used by `client.ts` or listed in `scripts/route-allowlist.txt` with a reason (for example, `GET /login` is browser-only, and `POST /api/login-code` is used through `jumpgate open`).
- [ ] Add `test/vocab.test.ts` and `test/secrets.test.ts` (whole-app frames across every screen with fixture secrets).
- [ ] Commit.

---

## Phase 3: Release asset (cheap path)

### Task 24: Release binaries

- [ ] **GitHub** `release.yml`: add a job `tui` with a matrix of `bun build --compile --target=bun-linux-x64|bun-linux-arm64|bun-darwin-arm64|bun-darwin-x64|bun-windows-x64`, all built on one ubuntu runner (Bun cross-compiles), producing `jumpgate-tui-<os>-<arch>[.exe]`. Upload them as an artifact, then fold them into the existing "unified checksums" and attestation steps by adding the artifact to their download lists.
- [ ] **GitLab** `release-snapshot`: build `linux-x64` only, to prove the build works.
- [ ] Check the cost first: the binary is about 60–100 MB per target because it embeds the Bun runtime and OpenTUI's native lib. If that is too large, ship Linux and macOS only (open question 2).
- [ ] Commit `ci(release): attach jumpgate-tui binaries`.

---

## Review Focus

1. A passphrase never survives the request: the input is cleared, the buffer zeroed, the value is absent from frames, the activity pane and errors (Tasks 3, 8, 14).
2. A one-time secret (customer API key, VPN peer config, AI key) appears only in `showOnce`, and only once (Tasks 18, 19, 20).
3. A shell-out passes no secret on argv and restores the terminal even when the child fails (Task 7).
4. The route check fails on a Go route rename (Task 2).
5. On the service's admin socket the TUI sends no token. A non-admin gets a clear message, not a stack trace (Task 1).

## Open Questions (for the user, most important first)

1. **Does this TUI eventually replace the Go TUI?** If yes, the Go TUI goes into maintenance now and the Bun binary needs a release path. If no, this is an experiment and Phase 3 can be skipped.
2. **Ship it in releases, and on which platforms?** Each compiled binary is roughly 60–100 MB. Options are all five targets, Linux and macOS only, or not at all until it is picked over the Go TUI.
3. **How is it launched?** As a separate `jumpgate-tui` binary, or from `jumpgate tui --next`, with `jumpgate` looking it up next to itself?
4. **What happens to the web UI?** It holds the most capability rows (gateways, VPN, containers, analytics). If the TUI covers them, is the web UI kept, frozen, or retired?
5. **Billing pricing and audit:** do you want them in the TUI? That needs new server routes, since today only the Rust admin socket has them.
6. **Windows:** is it a target for this TUI? It decides between the unix socket and TCP with Bearer for transport there.
7. **Shell-out vs new routes for vault administration:** the recommendation is shell-out. Do you want any of `vault init/backup/restore/rekdf/add-recovery` moved into the API later?
8. **Do the screens edit config as raw JSON** (gateway config, module config, container config), or does any group deserve a real form from the start?
