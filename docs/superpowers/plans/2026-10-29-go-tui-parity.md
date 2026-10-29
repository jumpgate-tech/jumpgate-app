# Go TUI Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Go dashboard (`internal/tui`, launched by `jumpgate tui`) the one terminal UI: bring over the UX of the OpenTUI experiment (`tui/`, branch `feat/opentui-ux` merged into main), add every capability of the inventory it lacks as the plainest screen, port the live e2e to Go, then delete `tui/`.

**Architecture:** The existing App (layout, focus regions, screens, modals, `Backend`, `tuitest.Fake`) stays. New API calls go through one generic `Backend.Do(ctx, method, path, in, out)` (already on `apiclient.Client`), so each new screen adds no interface churn; `tuitest.Fake` answers `Do` from a route map. CLI-only actions run `a.o.Self <args>` (the running binary from `os.Executable()`) through `tea.ExecProcess`, so the passphrase prompt, sudo and printed one-time secrets stay on the real terminal and never pass through the TUI; on return the screen re-reads `/api/vault` (or what changed). New screens are added to the nav as groups: Fleet, Jobs, Signers, Gateways, VPN, Vault, Server, Settings.

**Tech Stack:** Go, charm.land bubbletea/v2, lipgloss/v2, bubbles/v2, `x/exp/teatest/v2`, `internal/tui/tuitest` (golden helper with `-update`, fake backend).

**Spec:** `docs/superpowers/specs/2026-10-28-opentui-inventory.md` (rows by group A–L). Requirements mined from `tui/src` and `tui/test`.

## Global Constraints

- User text says **gate**, never **agent** (`vocab_test.go`). No trend graphs or sparklines: numbers and tables.
- Fail closed: unknown `n/a`, stale `~`, error `FAIL`. URLs through `maskURL`; server strings through `sanitizeLine`.
- Secrets: a passphrase or secret value is typed into the existing masked field and zeroed after the request; a one-time secret the server returns (customer API key, VPN peer config) is shown once in a modal and dropped on close; never logged to the activity pane. The typed confirm (`confirm.go`) stays the only way to confirm a destructive action; a mouse click never confirms it.
- Every fact is shown once: no pane's first line repeats its title, no pane repeats the status strip (test in Task 1).
- Minimum 80×24, ASCII twin for every glyph, `NO_COLOR` honoured.
- `go build -tags notui ./cmd/jumpgate` links no charm package.
- One commit per task, no attribution lines. Each task ends with `gofmt -l internal cmd` empty, `go vet ./...` and `go test ./internal/tui/... ./internal/api/... ./cmd/...` green.

## Gap analysis (inventory against `internal/tui` at 2ef0375)

Present in Go: A1 attach, A2 health/skew, A5 restart; B1 state, B2 unlock, B3 lock, B4 idle lock; C1 controller, C2 keys init (shell-out), C6 test signing, C7/C8 decide and adopt; D1 fleet, D2 box detail, D3–D6 add over SSH and pair, D7 `--local` (shell-out), D10 remove, D13 SSH, D15 migrate-names; E1 status, E2 disk, E3 measure, E5 endpoints, E6 firewall, E7 logs, E8 service actions, E16 explain; F1–F5 modules; G1–G3 jobs, running, detail, G8 gate check; H1 gateway list; J1 settings, J2 prefs; K1 activity, K2 alerts.

Missing (each gets the plainest screen):

| Grp | Missing | How |
|---|---|---|
| A | start server if absent, stop, open in browser, update check, server shape warnings (J4) | Server screen: `s` re-execs `jumpgate serve` detached through `daemon.EnsureRunning`-like start, `x` `POST /api/shutdown`, `o` `POST /api/login-code` then the opener, `GET /api/update` |
| B | put secret, init, migrate, backup, restore, add-recovery, rekdf, seal | Vault screen: put secret via masked field `PUT /api/vault/secrets/{name}`; the rest re-exec `jumpgate vault <cmd>` with `tea.ExecProcess`; path prompts for backup/restore |
| C | export keystore v3, create key on the service | Signers: `e` re-execs `keys export --keystore-v3 FILE`; `K` on a service re-execs `keys init` (which calls `init-key`) |
| D | pairing info, invite one-liner, upgrade gate, recorded host keys, forget host key, orphan cleanup | Invite strip (Task 2), Hosts: `u` upgrade, `k` host keys list with `f` forget, orphans list with typed-confirm delete |
| E | clear service state, containers list/config/action/provision/wipe/reset, du, setup wizard stream, monitor numbers, diagnostics, generic intent | Box tabs: Containers, Diagnostics; Storage gains du; Services gains `c` clear; setup wizard stream view; monitor as numbers |
| F | gateway modules | Gateways detail: modules list, toggle |
| G | job output (live), start job, generic intent | Job detail streams output; `n` starts a job by kind; intent runner on the box |
| H | create, delete, start/stop/restart, config, provision/wipe, trust cert / TLS verify, capabilities, known set, analytics, traffic, chain list, docker readiness/start, local host, customer keys | Gateways screen with detail tabs; key create shows the raw key once |
| I | VPN clients and servers, up/down/status, endpoint, peers | VPN screen |
| J | (theme: web only, n/a) | none |
| L | install service, setup server | Server screen re-execs `jumpgate setup install-service` / `setup server` (sudo on the real terminal) |
| — | setup guide checklist | Task 2 |
| — | billing pricing/audit | left out: no API route (inventory says add one only if wanted) |
| — | `serve`/`relay`/`agent *` process modes, gate health watch, gate setup on the box | out of scope (not controller actions) |

---

### Task 1: UX foundations

**Files:** `internal/tui/frame.go`, `focus.go`, `keys.go`, `app.go`, `mouse.go` (new), `dup_test.go` (new), `mouse_test.go` (new), goldens.

- [ ] Write `dup_test.go`: for every screen at 120×40 and 80×24, a pane's first body line must not equal its title (case-insensitive, trimmed), and no pane body line may repeat a status-strip segment (vault state, fleet counts, server version). Run: fails on today's goldens where it should.
- [ ] Focused pane: distinct border (already info colour + `>`), Tab / Shift-Tab cycle nav → main → log, Esc returns to the nav (closing the detail first).
- [ ] Active nav item and selected row drawn with `th.Sel` (reverse).
- [ ] Main title is a breadcrumb: `Fleet › box › tab` (`>` in ASCII).
- [ ] Hints bar lists only the focused pane's keys.
- [ ] Mouse (`mouse.go`): click nav item/row selects, double-click (≤400 ms, same cell) opens, click on `[k]` in the hints or a pane presses `k`, wheel scrolls the pane under the pointer, click on the drawer line toggles it, click on a strip segment opens its screen. A click inside a typed-confirm modal does nothing.
- [ ] Modal key flush: keys arriving within 150 ms of a modal opening are dropped.
- [ ] Tests: `mouse_test.go`, `focus` cases, regenerate goldens (`go test ./internal/tui -update`). Commit `feat(tui): every fact once, focus, reverse selection, breadcrumb, mouse, modal flush`.

### Task 2: Setup guide and invite strip

**Files:** `guide.go`, `guide_test.go`, `invite.go`, `clipboard.go`, `clipboard_test.go`, `bins.go`.

- [ ] Port `tui/src/shell/guide.ts`: 6 required steps (server running, vault created and unlocked, controller key, recovery code, a box paired, the box's gate healthy) plus optional ones (second box, a module on, a gateway); facts from live state only.
- [ ] Strip shows `setup N/6` until 6/6; clicking it or `g` opens the guide; it opens by itself on first run (no box and vault not unlocked), once per session.
- [ ] Invite on one unbordered line under the panes: `curl … | sh` from `GET /api/pairing-info`; `c` copies it with OSC 52 plus `pbcopy`/`wl-copy`/`xclip` found by absolute path on PATH (relative entries dropped).
- [ ] `bins.go`: resolve `jumpgate` (`JUMPGATE_BIN`, `os.Executable()`) and `ssh` absolute; refuse relative.
- [ ] Commit `feat(tui): setup guide, one-line invite with c to copy`.

### Task 3: Generic backend call, shell-out helper, Server screen (A, L)

- [ ] `Backend.Do`; `tuitest.Fake.Routes map[string]any` keyed `"METHOD /path"`, recording calls.
- [ ] `runSelf(args...)`: `tea.ExecProcess(a.o.Command(a.o.Self, args...))`, flash the exit code, re-read the vault.
- [ ] Server screen: version, skew, update check, shape warnings, start/stop, open in browser, install service, setup server.
- [ ] Commit `feat(tui): server screen: stop, open, update, install, setup`.

### Task 4: Vault screen (B)

- [ ] State, idle lock, put secret (masked), and re-exec init, migrate, backup FILE, restore FILE, add-recovery, rekdf, seal. Commit.

### Task 5: Signers and hosts gaps (C, D)

- [ ] Export keystore v3 (re-exec), upgrade gate, recorded host keys and forget, orphans. Commit.

### Task 6: Box gaps (E)

- [ ] Clear service state, containers (list, config, actions, provision, wipe/reset typed-confirm), du, diagnostics, monitor numbers, setup wizard stream, generic intent. Commit.

### Task 7: Jobs gaps (G) and gateway modules (F)

- [ ] Live job output, start job. Commit.

### Task 8: Gateways (H)

- [ ] Detail tabs: overview/config, capabilities, known set, analytics table, traffic, modules, keys (create shows the raw key once), chain list, docker, host; actions start/stop/restart/provision/wipe/delete/trust-cert/tls verify. Commit.

### Task 9: VPN (I)

- [ ] Clients and servers: list, save, delete, up/down/status, endpoint, peers add/remove (peer config shown once). Commit.

### Task 10: Port the live e2e

- [ ] `internal/tui/live_test.go` (build tag `live`): drives `tui.Run` in teatest against the real server `scripts/e2e-tui.sh` starts; same coverage as `tui/test/live.test.ts` (pair both boxes, restart a service, decide a job, turn a module on, invite, upgrade a gate, lock and unlock); frames written to `$JUMPGATE_LIVE_FRAMES`.
- [ ] `scripts/e2e-tui.sh` runs `go test -tags live -run TestLive ./internal/tui` instead of Bun. Commit.

### Task 11: Retire `tui/`

- [ ] Delete `tui/`, the `tui` jobs in `.gitlab-ci.yml` and `.github/workflows/ci.yml`, the TS fixture step and its Go writer if only `tui/` used it, Bun docs; keep Go fixture tests that guard web API shapes.
- [ ] Mark `docs/superpowers/plans/2026-10-28-opentui-tui.md` and `docs/superpowers/specs/2026-10-28-opentui-inventory.md` superseded (one line each). Commit.

### Task 12: Verify

- [ ] gofmt, `go vet ./...`, `go test ./...`, `go test -race ./internal/tui/...`, cross-builds (windows, darwin, linux × amd64, arm64), `notui` build, `scripts/e2e-tui.sh`, `scripts/e2e-agent.sh` (Debian 12).
