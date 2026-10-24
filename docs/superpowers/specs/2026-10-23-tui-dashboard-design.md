# TUI fleet dashboard: design

Date: 2026-10-23. Status: draft for review.
Builds on: `2026-10-05-tui-design.md` (the TUI spec, decisions D1–D32) and
`2026-10-02-controller-agent-architecture-design.md` (controller/gate protocol).

## Why

The user, on the frames rendered from the goldens onto the landing page:

> "if this is the tui, as shown, then we have some serious work to do. it
> should be structured more as a dashboard. where you can see helpful pieces up
> top, a main section in the middle, side bars, and a bottom footer with
> activity bars or progressions. we are giving people a dashboard to manage a
> fleet of servers."

Today the TUI is a tab bar (`1 Fleet … 6 Settings`), one full-width screen, and
a one-line status bar (`testdata/fleet_80x24.golden`: three table rows, then 17
blank lines). It reads as a list, not a console for a fleet. This spec turns it
into a dashboard: a KPI strip, a fleet tree, a main view, a context sidebar and
an activity footer, while keeping every v1 decision that still applies.

## Decisions kept from the TUI spec

- **Fail closed.** A nil section is "unavailable", never 0 (`api.FleetRow`
  comment). A tile without data shows `n/a`, a stale one `~`; URLs go through
  `maskURL`, which over-masks when unsure. Aggregates never hide unknowns:
  `2/3 synced · 1 n/a`, not `2/2`.
- **16 ANSI colours only, no background detection (D7)**; colour is never the
  only signal; `NO_COLOR` honoured.
- **Glyph sets `unicode`/`ascii` (D8)**; every new glyph has an ASCII twin.
- **Minimum 80×24 (D10)**; below it, the existing size message.
- **Goldens are ANSI-stripped `View()` content under the ASCII profile (D11)**;
  `teatest` for flows only.
- **Poller costs (D12, D31)**: fleet stream 15 s, status 5 s for the open box,
  disk 5 min, firewall 10 min. The dashboard adds no new polling of boxes.
- **D1**: the TUI calls one route per operation; the server picks gate or SSH.
- Mouse off by default (D9); typed confirmations unchanged (D21).

## Vocabulary

User-facing text says **gate** for the per-box program: link column values
`gate` / `ssh` / `down`, "gate v0.5.1", "gate stale", "pair a gate". Code
names (`Agent`, `agent.info`, `api.Link`) stay. Golden text changes from
`agent` to `gate` accordingly (`box-a · agent · …` becomes `box-a · gate · …`).

## 1. Layout

Five regions:

| Region | Content |
|---|---|
| **Header** (1 row) | `jumpgate`, server link state, version-skew banner, clock / last update |
| **KPI strip** (3 rows boxed, 1 row compact) | 7 tiles: boxes, synced, head lag, peers, disk headroom, gateways, alerts |
| **Left sidebar** | Views list (Fleet, Jobs, Signers, Gateways, Settings) and the fleet tree: box → nodes (exec/beacon), gateways, modules |
| **Main** | The selected view: fleet table, box detail (existing tabs), node detail, jobs inbox, signers, gateways, settings |
| **Right sidebar** | Context for the selection: health checklist, recent alerts for it, quick actions with keys |
| **Footer** (3–5 rows) | Running jobs as progress bars, syncing nodes as progress bars, event ticker, key hints |

### 120×40 (full dashboard)

```
 jumpgate  server live · v0.5.1                                                          ! 2 alerts · updated 3s ago
+- Boxes ------+- Synced ------+- Head lag ----+- Peers -------+- Disk -------+- Gateways ----+- Alerts ------------+
| 2 up 1 down  | 1/2 nodes     | max 2 blk     | 51 exec       | min 39% free | 2 live        | 1 FAIL  1 warn      |
| ##.  of 3    | 1 syncing     | box-b n/a     | 72 beacon     | box-a 61%    | 0 failing     | box-b down 4m       |
+--------------+---------------+---------------+---------------+--------------+---------------+---------------------+
+- Fleet ---------------+- Fleet ---------------------------------------------------+- box-a -----------------------+
| > Fleet           [1] | HOST      LINK     NET        SYNC     LAG  PEERS  DISK   | health                        |
|   Jobs  1 needs   [2] |>box-a     gate     PulseChain synced   2    48/72  61%  w | ok   gate answering  3s       |
|   Signers         [3] | box-b     down 4m  Ethereum   syncing~ n/a  3/9~   n/a    | ok   exec synced  21,345,678  |
|   Gateways        [4] | laptop*   local*   ?          n/a      n/a  n/a    n/a    | ok   beacon synced  9,876,543 |
|   Settings        [5] |                                                           | warn firewall: 8545 open      |
| ------------------- | |                                                           | ok   disk 61% used, ~140 d    |
| v box-a      synced   |                                                           |                               |
|   + exec  reth  ok    |                                                           | alerts                        |
|   + beacon lh   ok    |                                                           | warn 12:01 firewall 8545 open |
|   + gw  rpc-main live |                                                           |                               |
|   + mod msgboard  on  |                                                           | modules                       |
| > box-b      down 4m  |                                                           | msgboard  on  public read     |
| > laptop*    local    |                                                           |                               |
|                       |                                                           | actions                       |
|                       |                                                           | enter open     x actions      |
|                       |                                                           | l logs         S ssh shell    |
|                       |                                                           | m measure disk r restart...   |
|                       |                                                           | t test gate                   |
|                       |                                                           |                               |
|                       |                                                           |                               |
|                       |                                                           |                               |
|                       |                                                           |                               |
|                       |                                                           |                               |
|                       |                                                           |                               |
|                       | 3 boxes · sort: host · ~ stale · * this machine           |                               |
+-----------------------+-----------------------------------------------------------+-------------------------------+
+- Activity -------------------------------------------------------------------------------------------------------+
| job  box-a  migrate-names   step 2/3 [##########..........]  ! needs decision                                    |
| sync box-b  exec            21.1M/21.3M [##################..] 99.1%  ~ stale 4m                                 |
| 12:04 box-b gate down · 12:03 box-a disk measured · 12:01 box-a firewall warn · 11:58 laptop paired              |
+------------------------------------------------------------------------------------------------------------------+
 tab focus  1-5 views  / filter  : palette  x actions  enter open  esc back  ? help  q quit
```

Widths at 120: left 25, right 33, main the rest (62). Heights at 40: header 1,
KPI 4, footer 5 + hints 1, body 29.

### 80×24

Sidebars fold away; the KPI strip is one line; the footer is two lines (one
activity line that rotates job → sync → events, plus hints).

```
 jumpgate  server live                                      ! 2 alerts · 3s ago
 boxes 2/3 up | synced 1/2 | lag 2 | peers 51 | disk 39% | gw 2 | ! 1 FAIL 1 warn
 [Fleet] Jobs 1  Signers  Gateways  Settings                        tab: tree/ctx
 HOST         LINK     NET        SYNC      LAG   PEERS   DISK           FW
>box-a        gate     PulseCha.. synced    2     48/72   [####|...] 61% warn
 box-b        down 4m  Ethereum   syncing~  n/a   3/9~    n/a            n/a
 laptop*      local*   ?          n/a       n/a   n/a     n/a            n/a















 job box-a migrate-names 2/3 [######....] ! needs decision      (1/3 activity)
 tab focus  1-5 views  x actions  enter open  ? help  q quit
```

At 80×24 `[` toggles the tree and `]` the context panel as overlays over the
main view (drawn over the right or left 34 columns, `esc` closes), so nothing
is lost, only folded.

### Narrow fallback (80–99 columns, any height ≥ 24; and heights 24–29)

Same as 80×24 with more table rows. Extra height first grows the main view,
then (≥ 30 rows) restores the 3-row KPI tiles, then (≥ 34 rows) the 4-line
footer.

## 2. Responsive rules

Evaluated in this order on every `WindowSizeMsg`; one pure function
`layoutFor(w, h) layout` returns region rectangles, unit-tested on its own.

| Condition | Effect |
|---|---|
| `w < 80` or `h < 24` | size message (unchanged) |
| `w ≥ 100` | left sidebar shown (width 24, tree truncates names with `..`) |
| `w ≥ 120` | right sidebar shown (width 32–40, 27 % of width) |
| `w < 100` | both sidebars are overlays (`[` / `]`), views move to a one-line tab row |
| `100 ≤ w < 120` | right sidebar is an overlay; left stays |
| `h ≥ 30` | KPI tiles boxed (4 rows) instead of one line |
| `h ≥ 34` | footer at full size (job bars + sync bars + ticker + hints) |
| `h < 34` | footer is 2 lines: one rotating activity line (every 5 s, pauses while focused) + hints |
| `w ≥ 160` | KPI tiles gain a sparkline row (head-lag and peers history from the client's ring) |

The main view always keeps at least 50 columns and 8 rows; if a rule would
break that, the region that rule added folds instead. The fleet table drops
columns right to left by priority (FW, PEERS, NET) as it does today.

## 3. Data and refresh

No new box polling. The dashboard composes streams the TUI already has, plus
one new server aggregate in phase 4.

| Tile / region | Source | Cadence | Missing / stale |
|---|---|---|---|
| Boxes up/down | `WatchFleet` rows: `Reachable`, `Link`, `LastSeen` | 15 s stream | row with `CheckedAt == nil` counts as `loading`, shown separately |
| Nodes synced | `Status.Overall` per row | 15 s | nil `Status` → `n/a` count; `Stale.Status` → `~` |
| Head lag | max of `Status` lag fields | 15 s | lag unknown when public head unknown → `n/a`, never 0 |
| Peers | sum of exec / beacon `Peers` | 15 s | partial sums marked `~` and `· 1 n/a` |
| Disk headroom | min free from `Disk` / `DiskUsedPct` | poller 5 min | nil → `n/a`; names the worst box |
| Gateways | `Gateways()` `Status.State`, `Warnings` | on enter and every 30 s | request counts: **no API yet** (open question 1); tile shows `live/failing` until then |
| Alerts | derived client-side (below) | each fleet update | none → `0`, never hidden |
| Fleet tree | fleet rows + gateways (`RoutesTo`) + `ListModules` per expanded box | modules fetched on expand, then 60 s | `GateStale` → `gate stale` on the module line |
| Right sidebar | selected row, plus `WatchStatus` (5 s) only while a box is selected for > 1 s | 5 s | each line carries its age |
| Job bars | `WatchJobs` inbox: `Step`/`len(Steps)`, `NeedsDecision` | stream | `JobsSummary == nil` → no bar, footer says `jobs n/a` for that box |
| Sync bars | `Status` head vs reference head for syncing nodes | 15 s | no reference head → indeterminate bar `[<=>  ]`, no percent |
| Event ticker | client-side diff of successive fleet/jobs/module snapshots | each update | ring of 200, session-only |

**Alerts (v1, client-derived).** A pure function `alerts(fleet, jobs,
gateways) []alert` with severity `FAIL` / `warn`: box down, status error,
section stale > 2 intervals, firewall warn/fail, disk fit verdict bad, job
needs decision, gateway warnings, module error. Same inputs give the same list,
so it is golden-testable. A server-side alert feed is open question 2.

**Staleness.** Header shows `updated Ns ago` from `Fleet.At`; it turns
yellow + `~` after 2 × `IntervalSeconds`, red + `stale` after 4 × or when the
stream is reconnecting (D4). Every tile inherits the worst age of its inputs.

**Fail-closed display.** Tiles render three states: value, `n/a` (no data),
`~value` (stale). An error section shows `FAIL` with the code, not a number.
Masked fields stay masked in the sidebar and ticker (gateway URLs, enodes).

## 4. Navigation

**Focus regions:** tree, main, context, activity. `tab` / `shift+tab` cycle
the visible ones; the focused region's border is bold/cyan (in ASCII: `=`
top border, `>` in its title). Default focus: main.

| Key | Action |
|---|---|
| `1`–`5` | Fleet, Jobs, Signers, Gateways, Settings in main (Hosts folds into Fleet, below) |
| `↑↓ j k` | move in the focused region |
| `enter` | tree/table: open box detail in main; activity: open the job |
| `esc` | back (detail → fleet); closes an overlay |
| `← → h l` | tabs inside box detail (Overview, Storage, Endpoints, Security, Logs, Services, Modules) |
| `[` / `]` | toggle tree / context (overlay when narrow) |
| `space` | expand/collapse a box in the tree |
| `/` `:` `x` `?` `q` | unchanged |
| context actions | single letters listed in the right sidebar, same bindings as today (`l`, `S`, `m`, `r`, `t`) |

Tree selection and table selection are one selection: moving in either moves
both, and the right sidebar follows it.

**Screen mapping:**

| Today | Dashboard |
|---|---|
| Fleet | main "Fleet" (table); KPIs and tree around it |
| Hosts | merged into Fleet: `a` add, `p` re-pair, `d` remove, `L` pair this machine act on the selection; the address/pairing columns become a toggleable table mode (`v`) |
| Host detail + tabs | main "box detail"; tab row inside main |
| Modules (`host_modules.go`) | box-detail tab and tree children; module modal unchanged |
| Node (exec/beacon) | new main "node detail": the Overview card for one client + its logs |
| Gateways | main view; tree children under their placement box |
| Jobs | main view; also footer bars, `enter` from footer opens the job |
| Signers, Settings | main views, unchanged content |
| Help, confirm, palette | modals, unchanged |

## 5. Visual language

- **Borders:** rounded box drawing in unicode, `+-|` in ASCII, titles in the
  top border (`+- Fleet ---`). Unfocused borders bright-black; focused cyan +
  bold. Inner separators single-line.
- **Colour roles** (16 ANSI): green ok/synced/live, yellow warn/syncing/stale,
  red FAIL/down, cyan focus/selection/links, bright-black labels and ages,
  default foreground values. Each paired with a word (`ok`, `warn`, `FAIL`,
  `n/a`, `~`). Light and dark terminals work because only ANSI indices are
  used and dim text is bright-black, never a fixed grey; values never rely on
  bold alone.
- **Bars:** `[████│░░░]` / ASCII `[####|...]`; job bars `[##########....]`;
  indeterminate `[<=>   ]` animated by the 1 s tick only when focused (golden:
  frame 0). Sparklines `▁▂▃▄▅▆▇█` / `_.-=#`. Alert glyph `⚑` / `!`.
- **Vocabulary:** gate, box, node (exec/beacon), gateway, module, job.

## 6. Testing

- **Goldens** (ASCII profile, ANSI stripped, fixed clock and fake backend):
  `dash_120x40`, `dash_100x30`, `dash_80x24`, `dash_80x24_tree_overlay`,
  `dash_80x24_ctx_overlay`, `dash_160x48` (sparklines), `dash_box_detail_120x40`,
  `dash_node_detail_120x40`, `dash_jobs_120x40`, `dash_degraded_120x40`
  (every section nil/stale/error: proves `n/a`/`~`/`FAIL`, no zeros),
  `dash_unicode_120x40`. Existing screen goldens are renamed into the frame
  as each screen moves (phase plan).
- **Unit:** `layoutFor` table test over widths 80..200 × heights 24..60
  (main ≥ 50×8, regions never overlap, sum = w); `alerts()`; KPI aggregation
  with nil sections; `gate` vocabulary test (no user-facing `agent`).
- **teatest flows:** tab cycles focus; tree `enter` opens detail and context
  follows; resize 120→80 folds sidebars and keeps the selection; `]` overlay
  opens/closes; a job in the fake advances and the footer bar moves; stream
  drop shows stale then recovers.
- **Docs images:** `../landing/scripts/docs-images.mjs` `GOLDENS` map points
  at `dash_120x40` (hero) and `dash_80x24`; regenerate landing SVGs.

## 7. Migration (each phase ships)

1. **Frame.** `layoutFor`, header, footer hints, region borders; existing
   screens render inside main unchanged; tab bar becomes the views list in
   the left sidebar at ≥ 100. `agent` → `gate` in user text. Goldens updated.
2. **KPI strip + alerts.** Pure aggregation over the fleet stream already held
   in `App`; `alerts()`; header staleness.
3. **Tree + context sidebar.** Tree from fleet + gateways + modules
   (reuse `moduleCards`/`moduleStatus` from `host_modules.go`); right sidebar
   reuses `host_overview` cards, `host_security` checklist lines, and the `x`
   actions menu bindings (`actions.go`). Hosts merges into Fleet.
4. **Activity footer.** Job bars from `WatchJobs` (reuse `jobs.go` step
   formatting), sync bars, event ticker from snapshot diffs.
5. **Node detail view**, sparklines at ≥ 160, gateway request counts once the
   server exposes them.

Reuse: `theme.go`/`palette.go` glyph sets and styles, `format.go` (ages,
units, bars), `maskurl.go`, `sanitize.go`, `keys.go` bindings, `help.go`
(help generated from region bindings), `tuitest` golden helper, the fake
backend in `backend.go`.

## 8. Open questions

1. **Gateway requests.** `api.GatewaySummary` has no request counters. Add
   per-gateway request/error counts (and a rate) to `/api/gateways`, or drop
   the tile to `live/failing`?
2. **Alerts source.** Keep alerts client-derived (session-only, no history),
   or add a server alert feed with ack/mute so the web app and TUI agree?
3. **Hosts merge.** Is folding Hosts into Fleet acceptable, or keep a sixth
   view for address/pairing work?
4. **Sync progress.** Needs a reference head per chain; when absent the bar is
   indeterminate. Should the server expose estimated time-to-sync?
5. **Event history.** Ticker events are session-only; should the server keep
   a short event log so a fresh TUI shows the last hour?
6. **Large fleets.** Tree and KPIs at 50+ boxes: group by network, or
   collapse healthy boxes by default?
7. **Minimum size.** Keep 80×24 as the floor (D10), or allow a degraded 60-col
   "KPI + table only" mode for phone SSH clients?
