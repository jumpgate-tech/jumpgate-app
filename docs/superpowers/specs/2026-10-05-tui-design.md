# The `jumpgate` TUI, and the server contract it needs

Date: 2026-10-05. Status: written without a live review (the user was not
available); every open question is decided under "Decisions". Parent:
`2026-10-02-controller-agent-architecture-design.md` (sub-project 3, "TUI
core"), pulled forward ahead of sub-project 2 (durable jobs). Builds on
`2026-10-02-foundations-design.md` (merged) and plugs into the seam built by
`2026-10-05-platform-support-design.md` (in progress on `feat/platform-support`).
Inputs: the justification reviews in `.superpowers/sdd/justify/` (`F-whole.md`,
`C-surface.md`, `E-seams.md`), cited below by their review item numbers #7, #8,
#9 and #19.

## Goal

An operator can run their fleet from `jumpgate` in a terminal, without the web
UI, and the web UI keeps working beside it. Concretely, the TUI does what the
web UI does for node operations:

- see every box at once (sync, disk, peers, health), with columns they choose;
- open one box: live status, disk usage with a measurement on demand and a
  trend, RPC endpoints, firewall checklist, live logs with a filter and an AI
  explanation, start/stop/restart of the node services;
- open a shell on a box over SSH;
- add a box, confirm its SSH host key, pair it; pair this machine when it is
  itself a node;
- see which key signs, on which boxes, and test it;
- choose what is shown (columns, tabs, chains, units, refresh), stored by the
  server so every front end and every session sees the same choice.

It works at 80×24, is keyboard-first with the mouse optional, and stays
readable with no colour.

## Shape of the work

Three phases, in this order. Phase A starts on `main` now and does not wait for
the platform work.

| Phase | What | Branch | Can start |
|---|---|---|---|
| A | The pre-TUI server contract: one error contract, shared wire types and Go client, streams with reconnect, `agent` on targets, node operations through intents for paired boxes, host-key probe/confirm, server-owned verdicts, `/api/fleet`, live agent logs, AI hygiene, TUI support routes | `feat/tui-contract` from `main` | now |
| B | TUI v1: every screen below, against the Phase A client, tested with a fake | `feat/tui` from `main` after A merges | after A |
| C | Integration: bare `jumpgate` in a terminal opens the TUI (replacing `runTerminalHome`), `jumpgate home` keeps the plain screen, agents built without the TUI, end-to-end test, CI on three OSes | `feat/tui` | after platform Task 9 is on `main` |

## Out of scope

- Durable jobs (sub-project 2). The TUI ships a jobs-inbox placeholder and a
  reserved `jobs` field on fleet rows; the inbox becomes real with sub-project 2.
- The approval tier and the browser signing page. Every intent stays routine
  tier; destructive actions get the TUI's typed confirmation now and an
  approval signature when that tier exists.
- Setup over intents (sub-project 5). The TUI does not run the setup wizard; a
  box that is not set up says so and names the web app.
- Gateway, VPN and devnet operations. The TUI shows gateways read-only; it does
  not create, provision, wipe or reconfigure them (they are not on the agent
  path yet).
- Moving the web UI onto the new server-owned verdicts. The server computes
  them in this sub-project; the web UI's copies are deleted when it next
  changes those screens (sub-project 6).
- Moving `AIKey` out of `config.json` into a key store (D15).

## Users and the box they are on

- **U1 laptop operator.** macOS, Windows or Linux controller; boxes over SSH,
  some paired, some SSH-only. Runs `jumpgate` in a terminal or double-clicks
  the terminal bundle (platform Tasks 10–12).
- **U1 on the box itself.** A Linux machine that is both controller and node:
  its own target is `mode: local`; once paired with `--local` it is reached over
  the local agent socket. The fleet marks it "this machine".
- **U1 on a jump host.** SSH session into a Linux jump host, no browser. The
  TUI is the only full front end there.

## Phase A: the server contract

### A1. One error contract (#8)

Every error response from `/api/*`, including the auth middleware, is JSON:

```json
{"error": "human sentence", "hint": "one actionable line", "code": "snake_case"}
```

plus, where they apply, `reason` (the agent's signed rejection code, with
`code: "rejected"`), `host` and `fingerprint` (with `code: "unknown_host"`).

- The codes live in one registry, `internal/api` (`type Code string`, one
  constant each, with a default hint and a CLI exit class). The server never
  writes a code as a string literal; an AST test enforces it.
- `writeError(w, status, msg)` keeps its signature and derives a code from the
  status (`bad_request`, `not_found`, `conflict`, `upstream_failed`, …) for
  the long tail. The cases a client branches on get specific codes:
  `target_not_found` (both "target not found" and "no such target"),
  `target_exists` (adding a name that exists: the CLI and TUI re-pair it),
  `target_not_set_up`, `unreachable` (also for legacy SSH dial failures, which
  today are uncoded 502s), `host_key`, `unknown_host`, and the existing agent
  codes.
- The legacy kebab-case codes (`docker-absent`, `service-not-created`,
  `gateway-not-found`, `vpn-not-found`, `vpn-server-not-found`,
  `not-configured`, `docker-unreachable`) become snake_case. The web UI stores
  `code` but never branches on these strings, so no aliases (D25).
- `authMiddleware` answers 401 `unauthorized` and 403 `forbidden` as JSON.
- Rejection hints move to the server: an intent reply whose receipt is a
  rejection carries `hint`, from `api.RejectionHint(reason)`. The CLI's
  `remedies` table is deleted; the CLI shows the server's hint, and falls back
  to `api.HintFor`/`api.RejectionHint` only when talking to an older server.
- The pairing stream's events use `error` (not `err`) for their message.

### A2. Shared wire types and one Go client (#8, F "share one client")

- `internal/api` holds the request and response types both sides use: errors,
  `IntentReply`, `TargetView`, `NodeStatus`, `DiskView`, `Fleet`/`FleetRow`,
  `HostKeyProbe`, `PairEvent`, `UIPrefs`, `ControllerView`, `SSHCommand`,
  `LogHit`, `Explain`, `GatewaySummary`. It imports only `catalog` and
  `intent` (both leaves), never `server`, `executor` or `monitor`.
- `internal/apiclient` is the one Go client of the local server, used by the CLI
  and the TUI: discovery and auto-start (`daemon.Find`/`EnsureRunning`), typed
  methods per route, error decoding into `*api.Error`, the SSE reader (A4), and
  a version-skew check against `server.json`'s `version`.
- The CLI moves onto it: intents, `hosts list` (served from `GET /api/targets`
  instead of reading `config.json`), pairing, host keys. The CLI keeps its exit
  codes, now derived from `api.Code.Exit()`. `keys init` stays local: it
  creates the key, which the server must not.

### A3. Socket and auth

Unchanged in mechanism, now written down as the contract the TUI relies on:

- The CLI and the TUI talk HTTP over `~/.jumpgate/run/server.sock`: `0600` on
  unix, an owner-only DACL on Windows (platform Task 2). They never use the TCP
  listener, which exists for the browser.
- Every request carries `Authorization: Bearer <token>`, the token read from
  `server.json` (owner-only). The socket's permissions are the first lock and
  the token the second; neither alone is enough (D5).
- Streams use the same socket and the same header. `server.json`'s `version`
  is compared with the client's own; a difference is shown, never acted on
  silently (D30).

### A4. Streams: SSE with a reconnect story (D3, D4)

- All streams stay Server-Sent Events. The server writes a `: ping` comment
  every 15 s on every stream it opens; a client that hears nothing for 45 s
  treats the stream as dead.
- State streams (fleet, node status) send the full current value on connect
  and on every change, so a reconnect needs no event ids.
- The logs stream takes `?backlog=N` (max 2000): on connect it first sends
  `event: reset` with the last N lines, then live lines. A client replaces its
  buffer on `reset`, so reconnects never duplicate or skip silently. Without
  `backlog` the stream behaves as today (the web UI is unchanged).
- The client reconnects with backoff 250 ms × 2ⁿ, capped at 8 s, ±20 % jitter,
  reset after 30 s healthy. 4xx answers other than 408/429 end the stream with
  the decoded `*api.Error` (a removed target does not retry forever); 5xx,
  transport errors and silence retry. Each state change is delivered to the
  consumer (`connecting`, `live`, `retrying` with the last error and attempt
  number, `failed`), so a front end can grey its data and say why.

### A5. Targets carry `agent` and `link`; legacy node routes frozen (#7)

- `GET /api/targets` returns `api.TargetView`: today's fields with today's
  JSON names (the web UI keeps working), plus `agent` (`address`, `transport`,
  `pairedAt`, or `null`), `link` (`agent`, `agent_local`, `ssh_only`,
  `local_only`) and `thisMachine`.
- The set of `/api/targets/{id}/…` routes is frozen by a test that lists them.
  A new node operation is a new intent kind, not a new route.

### A6. One route per node operation; the server picks the transport (#7, D1, D2)

For a paired target, the existing node routes go through the agent with signed
intents; for an SSH-only target they keep the legacy executor:

| Route | Paired box | SSH-only box |
|---|---|---|
| `GET …/monitor/stream` | `status.read` every 5 s | `monitor.Monitor` (as today) |
| `GET …/du` | `disk.read` | `ops.DiskUsage` |
| `GET …/endpoints` | `endpoints.read` | `ops.Endpoints` |
| `GET …/firewall` | `firewall.read` | `ops.FirewallChecklist` |
| `GET …/logs`, `…/logs/stream` | `logs.read` / `logs.since` (A9) | `logwatch` |
| `POST …/services/{svc}/{action}` | `service.action` | `ops.ServiceAction` |

- Every such response carries `X-Jumpgate-Via: agent` or `ssh`.
- A signed rejection is `409 {code: "rejected", reason, hint}`; a signed
  failure is `502 {code: "agent_failed"}`; transport and host-key errors use the
  agent codes from A1.
- A paired box needs no controller-side `wire` for these routes: the agent has
  its own `node.json` and answers `not_set_up` if it has none.
- Routes with no intent yet (`…/disk?path=` for the wizard,
  `…/services/{svc}/clear`, diagnostics, setup, containers) stay on the legacy
  executor for every target, as the umbrella migration plan says.
- `/intent/{kind}` remains for the CLI and scripts; both it and the node routes
  share one `sendIntent` (one seq, one dial, one serialisation per target).

### A7. Host-key probe and confirm (#9, D16)

- `POST /api/hostkeys/probe` takes an SSH address (`api.SSHView`, with an
  optional jump) and returns one entry per hop, jump first: `hostPort`,
  `fingerprint` (OpenSSH SHA256 form), `keyType`, `state` (`confirmed`,
  `unknown`, `mismatch`, `unreachable`) and, for `unknown`, a `probeId`. It
  stops at the first hop that is not `confirmed`, because the next hop is only
  reached through a confirmed one.
- The server keeps the captured key for 5 minutes under the `probeId`.
  `POST /api/hostkeys/confirm {probeId, fingerprint}` records that exact key in
  `~/.jumpgate/confirmed_hosts` only if `fingerprint` matches it
  (`fingerprint_mismatch` otherwise, `probe_expired` after 5 minutes or a second
  use). The person compares what they see; the server records what it saw.
- The host-key policy (confirmed store plus OpenSSH known_hosts) is the one
  `fix/review-hotfixes` makes explicit; the server's `strictHostKey` is the only
  copy.
- The CLI's `confirmHostKeys` uses these routes and keeps its own consent rule
  (a person at a TTY types `yes`). The TUI's rule is the same: typed `yes`.

### A8. Server-owned verdicts (#9)

The server computes what both front ends display, from the raw readings:

- `api.NodeStatus` from a status reading: per client active/syncing/head or
  slot, peers; `refHead`; `headLag` (absent when the reference is unknown);
  `overall` (`synced`, `syncing`, `stopped`, `no_data`, `unavailable`) by the
  web dashboard's rule (both stopped → stopped; either syncing → syncing).
- `api.DiskView` from a disk reading: client bytes, free bytes, the expected
  sizes (always labelled `estimate`), and a fit verdict `ok`/`tight`/`short`
  against `catalog.FitMargin` (1.1, today's preflight margin), plus the trend
  (A10).
- `api.FirewallSummary` from a checklist: counts by status and a grade `ok`,
  `warn`, `fail`, `unknown`.
- `catalog.FitMargin` and `catalog.DefaultDataDir` become the single Go source
  (moved from `server`), ready for sub-project 2's rename.

### A9. Live logs from agents, and the agent's chain (#19)

- A new routine-tier read intent `logs.since {cursor, n}` returns
  `{hits, cursor}`: `journalctl -u <exec> -u <beacon> --after-cursor <cursor>
  -n <n> -o json`, each line classified by `logwatch` with its real journal
  timestamp. An empty cursor returns the last `n` lines and a cursor.
- For a paired target, `…/logs/stream` is served by a per-target follower that
  polls `logs.since` every 3 s while at least one client listens, keeps a
  1000-line ring (for `backlog`), and stops 30 s after the last client leaves.
- An agent that answers `unknown_kind` (older than this sub-project) gets
  `logs.read` snapshots every 10 s instead, sent as `reset` events; the client
  shows "snapshot mode: re-pair to upgrade the agent" (D14).
- `agent.info` also reports the box's `chainId` from its `node.json` (0 when not
  set up), so the fleet can name a paired box's network without a
  controller-side `wire`.

### A10. `/api/fleet` and the per-box views

- `GET /api/fleet` returns `api.Fleet{at, interval, rows}`, one row per target:
  `id`, `link`, `thisMachine`, `reachable`, `lastSeen`, `network`, `chainId`,
  `status` (`api.NodeStatus` or null), `disk` (`api.DiskView` or null),
  `firewall` (`api.FirewallSummary` or null), `jobs` (null until sub-project 2),
  `error` (the last probe's `api.Error`, or null), and per-section `stale`
  flags.
- `GET /api/fleet/stream` sends the whole `Fleet` on connect and then as boxes
  finish their probes, debounced to about 250 ms (boxes that finish together
  share a frame; a slow client gets only the latest).
- A server-side poller refreshes rows while anyone watches: status every
  `interval` (15 s default, from the UI prefs), disk every 5 min, firewall every
  10 min; four targets in parallel; each probe bounded to 20 s. It stops 2
  minutes after the last fleet request or stream ends, so an idle server sends
  no intents (D12).
- A failed probe never zeroes a row: the last values stay, with their own age,
  `stale` set once older than 3 × the probe's interval, and `error` set.
  "Unavailable" is shown as such, never as 0.
- Per box: `GET /api/fleet/{id}` (one row), `GET /api/fleet/{id}/disk`
  (`DiskView` with history), `POST /api/fleet/{id}/disk/measure` (take a
  reading now), `GET /api/fleet/{id}/status/stream` (`NodeStatus` events, 5 s).
  These are views over A6's operations, not new node operations, so the freeze
  does not apply.
- Disk trend: every disk reading is kept in a per-target ring of 288 samples in
  server memory. From at least two samples 6 hours apart the view adds
  `growthPerDay` and `daysToFull`, both labelled estimates (D13).

### A11. AI hygiene and the agent log source (#19)

- For remote providers (Gemini, Groq) the server redacts each line before it
  leaves: IPv4 and IPv6 addresses, `enode://` URLs, `enr:` records and libp2p
  peer ids become stable placeholders (`<ip-1>`, `<enode-1>`, …). Ollama is
  local and gets the lines unredacted.
- The explain response says what went out (`sentExcerpt`, after redaction) and
  whether it was redacted. The settings response carries a one-line
  `aiDisclosure` that both front ends show next to the provider.
- For a paired target, explain reads its default lines from the agent
  (`logs.read`, n 400, error and critical, last 40), not from root SSH.

### A12. Routes the TUI screens need

- `GET /api/ui-prefs`, `PUT /api/ui-prefs`: the display choices (D20), stored
  in `config.json` under `ui`, validated against registries in `internal/api`.
- `GET /api/controller`: the controller key's recorded address, store, the
  address the server actually signs with, and a state (`ok`, `missing`,
  `unopened` with the reason, `mismatch`). Uses the key-identity check from
  `fix/review-hotfixes` when it is merged.
- `POST /api/targets/{id}/check` is not added (freeze). Instead
  `POST /api/fleet/{id}/check` sends a signed `agent.info` and returns
  `api.AgentCheck{agent, version, chainId, setUp, elapsedMs}`: "test signing"
  for one box.
- `GET /api/fleet/{id}/ssh` returns the `argv` of a system `ssh` command for an
  interactive shell on an SSH target (D18). The CLI gains `jumpgate ssh HOST`,
  which runs it.

## Phase B: the TUI

### Principles (from the umbrella spec)

- The TUI holds no keys and does no work. It renders server state and sends
  requests through `internal/apiclient`; it can quit and reattach freely.
- Every value shows its age; stale values are greyed and marked; a failed probe
  reads "unavailable", never 0; expected sizes always carry "estimate".
- Destructive actions need typed confirmation (D21).
- Keyboard first; mouse optional (D9); readable without colour (D7, D8).

### Layout (80×24 and up)

```
 jumpgate  [1 Fleet]  2 Hosts  3 Signers  4 Gateways  5 Jobs  6 Settings  ? help
 HOST         LINK       NET       SYNC      LAG    PEERS   DISK            FW
▸box-a        agent      PulseCha… synced    2      48/72   [███░│░░░] 61%  warn
 box-b        down 4m    Ethereum  syncing~  n/a    3/9     n/a             n/a
 laptop*      local*     ?         n/a       n/a    n/a     n/a             n/a
 …                                                       (22 rows of main view)
 3 boxes · sort: host · ~ stale  * this machine
 box-a · agent · server live · jobs ⚑ –                                  5s ago
```

- Row 1: screen tabs (the active one bracketed, so it reads without colour);
  row 24: status bar with the selected host, its link, the server connection
  state, the jobs flag and the data age. A version-skew banner, when shown,
  takes the first row of the main view.
- Host detail shows a host list on the left only at 100 columns or more;
  narrower, `[` toggles it over the main view.
- Below 80×24 the TUI shows only "terminal too small (W×H); jumpgate needs
  80×24" and keeps running; it recovers on resize.
- Text that does not fit is cut with `…`, never wrapped into the next column.

### Screens

1. **Fleet.** One row per box: host, link (with last-seen age when not
   reachable), network, sync, head lag, peers (EL/CL), disk bar (used, expected
   tick, percent), firewall grade; optional columns EL, CL, jobs, age, agent
   address. Columns come from the prefs; `c` jumps to them in Settings. `/`
   filters by text, `s` cycles the sort (host, sync, disk, lag), `enter` opens
   the box. Rows whose network is not in the prefs' chain filter are hidden,
   with a count of hidden rows.
2. **Host detail**, tabs (visible set from prefs): **Overview** (live status
   stream, EL and CL cards, head lag, peers, ages), **Storage** (disk bars per
   client and free, expected sizes as estimates, fit verdict, a trend sparkline
   with growth and days-to-full estimates, `m` measures now), **Endpoints**
   (exec and beacon URLs, reachability, chain-id match, the SSH tunnel command,
   and the gateways placed on this box, read-only), **Security** (firewall
   checklist with why and fix), **Logs** (live, `/` text filter, `!` severity
   floor, `f` follow, `e` explain), **Services** (exec and beacon state;
   `s` start, `t` stop and `r` restart with typed confirmation; `S` opens an SSH
   shell). Diagnostics and Jobs tabs are not in v1.
3. **Hosts.** Every target with link, address and pairing. `a` adds a box:
   name, `user@host[:port]`, key path, optional jump, sudo; then the host-key
   flow (each unknown hop shows its fingerprint and the command to read it on
   the box's console; typed `yes` confirms; a mismatch stops with a security
   error); then the pairing stream. `L` pairs this machine (Linux only; a
   non-root user is handed to the foreground CLI so `sudo` can prompt, D17).
   `p` re-pairs. `d` removes (typed name; it forgets the box, D19).
4. **Signers.** This controller's key: address, store, state. Every paired box
   with its agent address, transport and the tier this controller holds
   (routine in v1). `t` sends a signed `agent.info` to the selected box and
   shows the verified result. `K` runs `jumpgate keys init` in the foreground
   when there is no key; `R` restarts the server so it loads a new key.
5. **Gateways.** Read-only list from `/api/gateways`: label, host, state, base
   URL, networks and their URLs, warnings. No actions in v1.
6. **Jobs inbox.** Placeholder: "Durable jobs arrive with sub-project 2". The
   status bar's ⚑ reads `–`.
7. **Settings.** Toggles, saved to the server on change: fleet columns, host
   tabs, chain filter, units (GB or GiB), compact rows, show estimates, bell on
   ⚑, mouse, glyphs (auto, unicode, ascii), refresh interval; AI provider and
   key with the disclosure line; an "Open the web app" item (platform Task 8's
   one-time login link).
8. **Help** (`?`): global keys and the current screen's keys, generated from
   the same key bindings the screens use.

### Keys

| Key | Action |
|---|---|
| `1`…`6` | Fleet, Hosts, Signers, Gateways, Jobs, Settings |
| `↑↓` / `j k` | move; `←→` / `h l`, `tab` / `shift+tab` switch tabs |
| `enter` / `esc` | open / back |
| `/` | filter (Fleet, Hosts, Logs) |
| `x` | actions menu for the selected box |
| `:` | command palette: `:host NAME`, `:logs NAME`, `:ssh NAME`, `:measure NAME`, `:restart exec|beacon NAME`, `:add`, `:fleet`, `:help`, `:quit` |
| `?` | help |
| `q` | quit from a top-level screen; `ctrl+c` always quits |

Palette commands that are destructive go through the same typed confirmation.

### Colour, glyphs and terminals (D7, D8)

- Semantic colours use the 16 ANSI colours only (red, yellow, green, cyan,
  bright black for dim), so the user's terminal theme decides how they look and
  nothing needs truecolor. Bubble Tea's colour-profile detection honours
  `NO_COLOR` and dumb terminals.
- Colour is never the only signal: every state is also a word or a glyph
  (`ok`, `warn`, `FAIL`, `n/a`, `stale`).
- Glyph sets: `unicode` (box drawing, `█░` bars, `▁▂▃▄▅▆▇█` sparklines, `⚑`)
  and `ascii` (`+-|`, `#.`, `_.-=#`, `!`). `auto` picks `ascii` on a Windows
  console that is not Windows Terminal (no `WT_SESSION`), on `TERM=linux`, and
  when `JUMPGATE_ASCII=1`; otherwise `unicode`.
- macOS Terminal.app (256 colours, no truecolor) and Windows Terminal need no
  special case beyond the above.

## Phase C: integration

- `runTerminalHome(ctx, in, out) int` keeps its signature (platform Task 9's
  contract) and runs the TUI. Its current line-based body becomes
  `runPlainHome`, reachable as `jumpgate home`, and used automatically when
  `JUMPGATE_PLAIN=1` or `TERM=dumb` (D22).
- The launchers (platform Tasks 10–12) are unchanged: they start plain
  `jumpgate`, which now opens the TUI.
- `jumpgate tui` opens the TUI explicitly (from scripts, or with other
  arguments later).
- Agent binaries are built with `-tags notui`, so the agent links no Bubble Tea
  code (D23).

## Dependencies (new Go modules)

Pinned exactly; each justified. No other new module.

| Module | Version | Why |
|---|---|---|
| `charm.land/bubbletea/v2` | `v2.0.9` | The program loop, input (including the Windows console API), alt screen, `ExecProcess` for the SSH shell and the foreground CLI. v2 over v1: the maintained line, key events that are the same on Windows and unix, colour-profile handling built in. v2.0.9, not v2.0.10: v2.0.10 requires Go 1.26 and the module is Go 1.25. |
| `charm.land/lipgloss/v2` | `v2.0.6` | Styles, widths and joins; the layout maths for 80×24. |
| `charm.land/bubbles/v2` | `v2.2.1` | `textinput` (forms, filter, palette, typed confirmation), `viewport` (logs), `key` and `help` (bindings and the help screen from one source). |
| `github.com/charmbracelet/x/exp/teatest/v2` | `v2.0.0-20261004011457-ad85c59fdf4e` | Test only: drives the real program loop for flow tests. Pseudo-version because the module has no tags; pinned to the commit. |
| `github.com/charmbracelet/x/ansi` | `v0.11.8` | Already in the graph through lipgloss; used directly to strip ANSI from frames in golden tests. |
| `github.com/charmbracelet/colorprofile` | `v0.4.3` | Already in the graph through bubbletea; used directly to force the ASCII profile in tests. |

Transitive additions (through the above): `github.com/charmbracelet/ultraviolet`,
`x/term`, `x/termios`, `x/windows`, `github.com/clipperhouse/*`,
`github.com/rivo/uniseg`, `github.com/mattn/go-runewidth`,
`github.com/lucasb-eyer/go-colorful`, `github.com/muesli/cancelreader`,
`github.com/xo/terminfo`, `golang.org/x/sync`, `github.com/aymanbagabas/go-udiff`
(test). All pure Go; no cgo.

## Testing

- **Server (Phase A):** handler tests with `httptest`, the existing in-process
  agent (`pairedLocal`), and fake probers; pure table tests for verdicts,
  redaction, backoff and the SSE parser; an AST test for error codes; a
  source-scan test for the route freeze.
- **Client:** `apiclient` against `httptest` servers and a real unix socket;
  the reconnect loop with a server that drops, goes silent, and returns 404.
- **TUI (Phase B):** model and update unit tests that feed messages and assert
  state; golden frames of `View()` with ANSI stripped, at 80×24 and 120×40,
  under the ASCII profile (identical on every OS); a handful of `teatest` flows
  against the real program loop; one fake backend (`internal/tui/tuitest`).
  Goldens are updated with `go test ./internal/tui/ -update`.
- **End to end (Phase C):** the TUI against a real `server.Server` with an
  in-process agent, driven by `teatest`.
- **CI:** everything runs in the three-OS `go` job from platform Task 1
  (ubuntu-24.04, macos-latest, windows-latest). Golden files are LF on every
  OS (`.gitattributes`).

## Coordination with the platform and hotfix work

`feat/platform-support` (platform Tasks 1–14) and `fix/review-hotfixes` run in
parallel. Phase A starts on `main` and touches neither's files in a way that
blocks them; where both edit a file, the second to merge rebases.

| TUI task (plan numbering) | Needs | Why |
|---|---|---|
| 1 Error contract | hotfix "key identity"/`SignerErr` (soft) | Both edit `writeNoControllerKey`; keep the hotfix's reason text under code `no_controller_key` |
| 2 Shared client | hotfix "version skew" (soft) | If the hotfix adds a skew check to `daemon`, the client calls it instead of its own |
| 2 Shared client | platform T5, T8 (soft) | They add CLI code over `mustRequest`/`call`/`streamPair`; those keep their signatures |
| 5 Node ops via intents | platform T13 (soft) | T13's `writeExecutorError` folds into this task's `writeDialError` |
| 6 Host keys | hotfix "explicit host-key policy" (soft), platform T7 (soft) | One strict builder; `OpenSSHKnownHosts` adds the system file |
| 12 Support routes | hotfix "key identity" (soft), platform T7 (soft) | Controller state `mismatch`; known_hosts files for `ssh` |
| 19 Hosts screen | platform T5 (hard for `L`) | Local pairing: root without sudo, non-root in the foreground |
| 21 Settings | platform T8 (hard for "Open the web app") | `openWebApp` and login codes |
| 22 Replace the seam | platform T9 (hard), hotfix "one buildServer" (hard), platform T3 (soft) | The seam itself; an auto-started server must be the full server; `build-agents.sh` gains `-tags notui` |
| 23 End to end + CI | platform T1 (hard), T10–T12 (verify) | The three-OS matrix; the launchers start the TUI unchanged |

"Soft" means the task can land first and the later branch adapts (named in the
plan step); "hard" means the task waits for it.

## Decisions

Each line: the decision, one line of why, and what it costs if wrong.

| # | Decision | Why | Cost if wrong |
|---|---|---|---|
| D1 | The TUI calls one route per node operation and the server picks agent or SSH; the TUI never calls `/intent/{kind}` directly. | F's "scope v1 to the agent path" and the user's "SSH-only boxes too" meet here: one client code path, transport is the server's business. | If SSH-only support is dropped, the routing layer (~200 lines) is dead weight. |
| D2 | SSH-only boxes are fully shown and operable in TUI v1 through the frozen legacy routes. | The user asked for multi-box including unpaired boxes; the umbrella migration plan shows them as `SSH-only`. | The TUI depends on the legacy executor until sub-project 6 retires it; no TUI change then. |
| D3 | Streams stay SSE; no WebSocket. | Every server stream is SSE already; one-way is enough; works over the unix socket with net/http; no new dependency. | Bidirectional needs later (job decisions) would add a second transport; decisions can be plain POSTs. |
| D4 | Reconnect: 250 ms doubling to 8 s, ±20 % jitter, reset after 30 s healthy; 15 s server pings, 45 s silence timeout; 4xx except 408/429 is final. | Fast first retry after a server restart, bounded load, a removed target does not retry forever. | A wedged-but-open connection is noticed after 45 s instead of sooner. |
| D5 | Keep bearer-token auth on the socket; no peer-credential auth, no TCP for the TUI. | The socket is owner-only on every OS; the token is a second lock that costs nothing and is already implemented. | None found; peer-cred can be added without a client change. |
| D6 | Bubble Tea v2 line (`charm.land/*/v2`), bubbletea pinned at v2.0.9. | v2 is maintained and has uniform key events on Windows; v2.0.10 needs Go 1.26. | Moving to 2.0.10+ later is a pin bump with the module's Go directive. |
| D7 | Only the 16 ANSI colours, no background detection, no truecolor. | Terminal themes stay in charge; nothing to downsample; macOS Terminal and Windows consoles render it. | Less visual polish than an adaptive palette. |
| D8 | Glyph sets `unicode`/`ascii`, `auto` choosing ascii on legacy Windows console, `TERM=linux` or `JUMPGATE_ASCII=1`. | Legacy conhost fonts lack `⚑` and block elements; Windows Terminal sets `WT_SESSION`. | A legacy-console user with a good font sees ascii until they pick unicode in Settings. |
| D9 | Mouse off by default; a Settings toggle turns it on. | Mouse capture breaks native text selection, and operators copy fingerprints, tunnel commands and addresses. | Mouse users enable it once. |
| D10 | Minimum 80×24; the host sidebar only at ≥100 columns. | 80×24 is the default terminal on every OS and in SSH sessions. | Smaller terminals get a message instead of a cramped view. |
| D11 | Goldens are `View()` content, ANSI-stripped, ASCII profile, at fixed sizes, with an in-repo helper; `teatest` only for flows. | Renderer byte streams contain cursor moves that differ by timing; content frames are deterministic on every OS. | A rendering bug that only shows in the byte stream is caught by the end-to-end flow, not a golden. |
| D12 | The fleet poller runs only while watched and stops 2 minutes after the last watcher; status 15 s, disk 5 min, firewall 10 min. | An idle server should not sign intents forever; disk and firewall probes are heavy. | The first fleet view after idle shows rows "loading" for one poll. |
| D13 | Disk history is an in-memory ring of 288 samples per target; trend estimates need two samples 6 h apart. | Cheap, no new file format; the umbrella's honest-data rule says trends are estimates anyway. | The trend resets when the server restarts. |
| D14 | Live agent logs use a new routine read intent `logs.since` with journald cursors; older agents get 10 s `logs.read` snapshots. | Tail windows without cursors cannot be diffed (repeated lines, no timestamps); the fallback keeps old agents useful. | One more intent kind on the agent; old agents show snapshot mode until re-paired. |
| D15 | Redact IPs, enodes, ENRs and peer ids for remote AI providers; not for Ollama; `AIKey` stays in `config.json` for now. | Closes the third-party leak (#19) without the Windows key-store dependency (platform T4). | The provider key is readable by anyone who can read the owner-only `config.json`. |
| D16 | Host-key probes hold the captured key server-side (5 min, single use); confirm needs the matching fingerprint. | What the person compared is exactly what is recorded; no second capture to race. | A person who waits over 5 minutes probes again. |
| D17 | Non-root local pairing and `keys init` run as foreground children (`tea.ExecProcess`) of the TUI. | `sudo` and keychain prompts need the terminal; the server never prompts (platform D18). | The TUI screen is replaced by the CLI's output for the duration. |
| D18 | The SSH shell is the system `ssh` with server-built argv: `-p`, `-i`, `-J user@jump:port`, `StrictHostKeyChecking=yes`, and `UserKnownHostsFile` set to the confirmed store plus OpenSSH known_hosts. | No terminal emulation in-process; the same keys a person confirmed protect the shell. | A jump host that needs its own key file not in ssh-agent fails with ssh's error; those users run ssh themselves. |
| D19 | Removing a host forgets it on the controller only; on-box revocation waits for the approval tier. | Revoking a controller is approval-tier by the umbrella policy. | The box still lists this controller until someone revokes it there. |
| D20 | UI prefs live in `config.json` under `ui`, served by `/api/ui-prefs`, validated against registries in `internal/api`. | The umbrella says toggles are stored by the server; one place for every session. | A second front end with different needs adds keys to the same block. |
| D21 | Typed confirmation: the service name for stop and restart, the host name for remove, `yes` for a host key; start needs none. Surrounding spaces are trimmed; nothing else is forgiven. | Matches the legacy `clear` route's rule and the CLI's host-key rule. | One more step for a routine restart. |
| D22 | The plain fallback is `jumpgate home` (and `JUMPGATE_PLAIN=1`, `TERM=dumb`), not a `--plain` flag. | A leading `-` routes to the web app in `dispatch`; a subcommand needs no change there. | Anyone expecting `--plain` reads the help. |
| D23 | Agent builds use `-tags notui`. | The agent runs as root on every box; it should not link a terminal UI it never runs. | Two tiny build-tagged files in `cmd/jumpgate`. |
| D24 | Server-owned verdicts land now; the web UI keeps its copies until it next changes those screens. | Rebuilding the committed web dist for no visible change is churn; sub-project 6 rewrites those screens. | Two copies of the fit maths until then. |
| D25 | Kebab-case codes become snake_case with no aliases. | The web UI stores `code` but never compares it to these strings. | A third-party script matching `docker-absent` breaks (none known). |
| D26 | `writeError` derives a code from the status; specific codes only where a client branches; an AST test forbids literal codes. | 185 call sites; a registry enforced by the compiler and a test, not by review. | A few responses carry a generic code where a specific one would help; each is a one-line change. |
| D27 | Jobs inbox is a static placeholder; fleet rows carry `jobs: null`; the ⚑ count reads `–`. | Durable jobs are sub-project 2; the TUI is stateless, so adding the screen later is additive. | None. |
| D28 | Gateways are read-only in v1, from the legacy `/api/gateways`. | Gateway operations are not on the agent path; F scopes TUI v1 away from mirroring them. | Gateway operators still use the web UI for changes. |
| D29 | Rejection hints are the server's (`api.RejectionHint`); the CLI's table is deleted. | One text per code; the CLI's copies had already drifted. | An old server sends no hint; the client falls back to the same package. |
| D30 | Version skew shows a banner; `R` restarts the server on request; never automatically. | A running server may hold work (pairing, relay); restarting it is the person's call. | A person who ignores the banner may hit 404s from an older server, each explained by the banner. |
| D31 | Host overview uses the 5 s status stream; the fleet uses the 15 s fleet stream. | The open box gets fresher data; the fleet stays cheap. | One extra signed intent every 5 s per open host detail. |
| D32 | Signers v1 shows the controller key, per-box enrollment and test signing; hardware wallets and approval keys come with the approval tier. | Only the routine tier exists. | The screen grows when the approval tier lands. |

## Done when

- Phase A: every `/api/*` error is `{error,hint,code}` with a registered code;
  the CLI runs on `internal/apiclient`; `GET /api/targets` shows `agent` and
  `link`; a paired box's status, disk, endpoints, firewall, logs and service
  actions work with root SSH disabled; host keys can be confirmed over the API;
  `/api/fleet` and its stream serve every box; agent logs stream live; remote AI
  providers never receive an IP address.
- Phase B: every screen above works against the fake and against a real server
  at 80×24, with `NO_COLOR=1`, and with ascii glyphs.
- Phase C: bare `jumpgate` in a terminal opens the TUI on all three OSes;
  `jumpgate home` shows the plain screen; the launcher bundles open the TUI; the
  `go` CI job is green on ubuntu-24.04, macos-latest and windows-latest.
