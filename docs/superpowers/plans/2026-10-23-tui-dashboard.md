# TUI Fleet Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the tab-bar TUI into a fleet dashboard (header, KPI strip, views + fleet tree, main, context sidebar, minimizable activity drawer), one shippable phase at a time.

**Architecture:** A pure `layoutFor(w, h, drawer)` returns region rectangles; `App.render` draws each region from data the App already holds, and existing screens render unchanged inside the main region. Every phase keeps all screens working and updates the goldens it changes.

**Tech Stack:** Go, charm.land bubbletea/v2, lipgloss/v2, bubbles/v2, `x/exp/teatest/v2`, the `internal/tui/tuitest` golden helper and fake backend.

**Spec:** `docs/superpowers/specs/2026-10-23-tui-dashboard-design.md` (decisions U1–U4 and §1–§7).

## Global Constraints

- Fail closed: nil section → `n/a`, stale → `~`, error → `FAIL`; never 0 for unknown. URLs via `maskURL`.
- 16 ANSI colours, colour never the only signal, `NO_COLOR` honoured; every glyph has an ASCII twin; ASCII frames contain only 0x20–0x7e and `\n` (`requireASCII`).
- Minimum 80×24; below it the existing `tooSmall` message, unchanged.
- Goldens: ANSI-stripped `View()` under the ASCII profile, fixed clock (`testNow`) and `tuitest.Fake`; `-update` regenerates; teatest for flows only.
- No new box polling.
- User text says **gate**, never **agent** (code names unchanged).
- `.` toggles the drawer; state persists in `UIPrefs.Drawer` (`auto`|`open`|`min`, default `auto`); `auto` = minimized when `h < 34`.
- The drawer never opens itself; a pending decision is highlighted on the minimized line as `[! N needs decision]`.
- `go build -tags notui ./cmd/jumpgate` links no `charm.land`/`charmbracelet` package (`scripts/build-agents.sh` check).
- Touch `internal/tui/host_modules.go` minimally (another branch edits it).

## Review Focus

1. Window exactly 80×24 and 100×24 with drawer open: body keeps ≥ 8 rows; the drawer shrinks to its 2-line form rather than squeezing main. Test in Task 1 (`TestLayoutForMainFloor`).
2. Typing `.` in a text field (filter, host-add form, AI key) types a dot, never toggles the drawer. Test in Task 4 (`TestDrawerKeyIgnoredWhileCapturing`).
3. A save of the drawer state refused by the server: the flash says so and the state comes back from the re-read, as other prefs do. Test in Task 4 (`TestDrawerSaveRefused`).
4. An older server/stored prefs block without `drawer`: `WithDefaults` gives `auto`, `Validate` accepts it. Test in Task 3.
5. Modals (help, confirm, palette) still own the keyboard and draw inside main, with header and drawer around them. Test in Task 5 (`help_80x24` golden regenerated, `TestModalInsideFrame`).

---

## File Structure (phase 1)

- Create `internal/tui/layout.go` — `drawerState`, `rect`, `layout`, `layoutFor`. Pure.
- Create `internal/tui/layout_test.go`.
- Create `internal/tui/frame.go` — `header()`, `viewsList()`, `tabRow()`, `boxed()`, `drawerView()`, `drawerLine()`; replaces `topBar`/`statusBar` in `app.go`.
- Create `internal/tui/frame_test.go` — dashboard goldens and drawer flows.
- Modify `internal/api/prefs.go` — `Drawer` field, default, validation.
- Modify `internal/tui/app.go` — render via layout; `.` key; remove `topBar`/`statusBar`.
- Modify `internal/tui/keys.go` — `globalKeys.Drawer`.
- Modify `internal/tui/settings.go` — a "activity drawer" item cycling auto/open/min.
- Modify `internal/tui/host.go` — host list moves into the left sidebar.
- Modify user-text `agent` sites: `app.go` (`linkWord`), `hostadd.go:450`, `hosts.go:222`, `signers.go:123,241`, fleet column header for `agent` if user-visible.

---

## Phase 1: Frame and activity drawer

### Task 1: `layoutFor`

**Files:** Create `internal/tui/layout.go`, `internal/tui/layout_test.go`

**Interfaces:**
- Produces:
  ```go
  type drawerState int // drawerAuto, drawerOpen, drawerMin
  func drawerFromPref(s string) drawerState // "open"/"min", anything else auto
  func (d drawerState) pref() string        // "auto"|"open"|"min"
  type rect struct{ X, Y, W, H int }
  type layout struct {
      Header rect     // always 1 row at Y=0
      Tabs   rect     // H=1 when w < 100 (one-line view tabs), else zero
      Left   rect     // views list + host list, W=24 when w >= 100, else zero
      Main   rect
      Drawer rect     // boxed activity rows; H=0 when minimized
      Hints  rect     // 1 row, last line (minimized: the status line)
      DrawerOpen bool // resolved: auto → h >= 34
      DrawerCompact bool // open but h < 34: 1 activity line, no box
  }
  func layoutFor(w, h int, d drawerState) layout
  ```
  Skew banner is not a region: `render` takes one row from Main for it.

- [ ] **Step 1: Write failing tests**

```go
func TestLayoutForMainFloor(t *testing.T) // for w in 80..200, h in 24..60, d in all three:
//   Main.W >= 50, Main.H >= 8; Left.W+Main.W == w; rows Header+Tabs+Main+Drawer+Hints == h;
//   rects don't overlap.
func TestLayoutForBreakpoints(t *testing.T) // table:
//   (80,24,auto): Left.W==0, Tabs.H==1, DrawerOpen==false, Drawer.H==0, Hints.H==1
//   (80,24,open): DrawerOpen, DrawerCompact, Drawer.H==1
//   (100,30,auto): Left.W==24, Tabs.H==0, DrawerOpen==false
//   (120,40,auto): DrawerOpen, !DrawerCompact, Drawer.H==5 (box: 2 borders + 3 lines)
//   (120,40,min): Drawer.H==0
func TestDrawerPrefRoundTrip(t *testing.T) // pref() of drawerFromPref(x)==x for auto/open/min; "" and "bogus" → auto
```

- [ ] **Step 2:** `go test ./internal/tui -run 'TestLayoutFor|TestDrawerPref'` — FAIL (undefined).
- [ ] **Step 3:** Implement. Open drawer at `h >= 34`: 5 rows (box); at `h < 34`: 1 row, unboxed.
- [ ] **Step 4:** Tests pass.
- [ ] **Step 5:** Commit `feat(tui): layoutFor, the dashboard's regions as a pure function`.

### Task 2: "gate" in user text

**Files:** Modify `app.go` (`linkWord`), `hostadd.go`, `hosts.go`, `signers.go`, `fleet.go` header for the `agent` column (column ID unchanged; header text `GATE`), check `internal/api` column titles. Test: `internal/tui/vocab_test.go`.

- [ ] **Step 1: Failing test** `TestNoAgentInUserText`: render frames from the existing golden setups (fleet, hosts, signers, host overview, hostadd confirm) under ascii and assert `!regexp.MustCompile(`(?i)\bagent\b`).MatchString(frame)`, except the literal `ssh-agent`. Also `linkWord(api.FleetRow{Link: api.LinkAgent, Reachable: true}) == "gate"`.
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Change the strings (`paired: gate …`, `pairing this machine needs Linux: a gate runs only on Linux. Pair a Linux box with a.`, `gate %s`). Regenerate affected goldens with `-update`, inspect the diff is only agent→gate.
- [ ] **Step 4:** `go test ./internal/tui/...` — PASS.
- [ ] **Step 5:** Commit `feat(tui): say gate, not agent, in user-facing text`.

### Task 3: `UIPrefs.Drawer`

**Files:** Modify `internal/api/prefs.go`, its test; `internal/tui/settings.go` and `settings_test.go`.

**Interfaces:** Produces `UIPrefs.Drawer string \`json:"drawer"\`` (`"auto"|"open"|"min"`); `DefaultPrefs().Drawer == "auto"`.

- [ ] **Step 1: Failing tests** — `TestPrefsDrawerDefault`: `UIPrefs{}.WithDefaults().Drawer == "auto"`; `TestPrefsDrawerValidate`: `"open"`,`"min"`,`"auto"` ok, `"x"` → error containing `drawer "x"`. TUI: `TestSettingsDrawerCycles`: on the Settings screen, the item labelled `activity drawer` shows `auto`; space → save with `open`, then `min`, then `auto` (Fake records saved prefs).
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Implement default, validation (`drawer %q (want auto, open or min)`), settings item in the display section.
- [ ] **Step 4:** `go test ./internal/api/... ./internal/tui/...` PASS (regenerate `settings.golden`).
- [ ] **Step 5:** Commit `feat: the activity drawer's state is a saved display setting`.

### Task 4: Drawer key and lines

**Files:** Create `internal/tui/frame.go` (drawer parts), `internal/tui/frame_test.go`; modify `keys.go`, `app.go`.

**Interfaces:**
- Consumes: `layoutFor`, `drawerFromPref`, `savePrefs(a, p)`.
- Produces:
  ```go
  globalKeys.Drawer = key.NewBinding(key.WithKeys("."), key.WithHelp(".", "activity drawer"))
  func (a *App) drawerOpen() bool             // layoutFor(a.w, a.h, drawerFromPref(a.prefs.Drawer)).DrawerOpen
  func (a *App) drawerLine(w int) string      // minimized line
  func (a *App) drawerView(r rect, compact bool) string // open content
  func (a *App) hints() string                // key hints for the current screen
  ```
- `.` sets `Drawer` to the opposite of the *resolved* state (`open`/`min`) and calls `savePrefs`.
- Minimized line, left to right, joined by two spaces: `jobs N running` (sum of `Jobs.Running`; `jobs n/a` when no row has `Jobs`), the highlight `[! N needs decision]` only when `needCount() > 0` (Warn style), `! alerts n/a` until phase 2, the flash if any (replaces counts), then ` | ` and `hints()`; truncated to `w`.
- Open content (phase 1): one line per box with a `Jobs` section: `jobs  <box>  N running · N needs decision`; then the old status line content (`<sel> | <link> | server live | updated 5s ago`); boxed with title `Activity` when not compact; compact = the first job line or the status line, then hints line.

- [ ] **Step 1: Failing tests (teatest/tuitest)**
  - `TestDrawerToggleSaves`: 120×40, auto → open; press `.` → frame lacks `+- Activity`, Fake's last saved prefs `.Drawer == "min"`; press `.` → `"open"`.
  - `TestDrawerAutoMinimizedAt80x24`: frame's last line starts with ` jobs`, contains `. drawer`.
  - `TestDrawerHighlightsDecisionNoOpen`: 80×24 auto, fleet with `Jobs{Running:1, NeedsDecision:1}` → last line contains `[! 1 needs decision]`, frame has no `Activity`.
  - `TestDrawerKeyIgnoredWhileCapturing`: open fleet filter `/`, type `.`; prefs not saved, filter text contains `.`.
  - `TestDrawerSaveRefused`: Fake SavePrefs returns error → flash `could not save the display setting`, then re-read state applies.
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Implement; add `Drawer` to help via `globalKeys` (automatic from `keyList`).
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(tui): a minimizable activity drawer, toggled with .`.

### Task 5: The frame

**Files:** Modify `frame.go`, `app.go` (`render`), `host.go` (`sidebarView` → part of left sidebar); goldens.

**Interfaces:** Produces `func (a *App) header(w int) string`, `func (a *App) viewsList(r rect) string`, `func (a *App) tabRow(w int) string`, `func (a *App) boxed(title string, focused bool, body string, w, h int) string`.

- Header: ` jumpgate  server <conn>[ (err)] · <ver if known>` left; right `! N alerts`-free in phase 1: `updated <age> ago` (or `connecting` before the first fleet); age styled warn+`~` after 2×`IntervalSeconds`, `stale` after 4× or `Retrying`.
- Left (w ≥ 100): boxed `Views`, lines `> Fleet  [1]` … for the six current screens (Hosts stays until phase 3), active one with `gl.Sel`; when a host is open, a separator then the host list (old `sidebarView`).
- Below 100: `tabRow` (`[Fleet] Hosts Signers Gateways Jobs Settings`), and `[` keeps the host-list overlay as today.
- Main: boxed with the screen's title (`Fleet`, or the host id for detail); the screen's `view(a, w-2, h-2)`; modal views also render inside main.
- Borders: unicode rounded via `lipgloss.RoundedBorder()`, ascii `+-|` (custom `lipgloss.Border`); titles in the top border; colours from `Theme`.

- [ ] **Step 1: Failing goldens + tests** in `frame_test.go` using the fleet fixture from `fleet_test.go` (box-a/box-b/laptop with box-a `Jobs{Running:1,NeedsDecision:1}`):
  `dash_120x40` (drawer open), `dash_120x40_drawer_min`, `dash_100x30`, `dash_80x24`, `dash_80x24_drawer_open`, `dash_unicode_120x40`; each ascii one through `requireASCII`; `TestFrameLinesFitWindow` (every golden size: exactly h lines, each ≤ w cells); `TestModalInsideFrame` (`?` at 120×40: header and drawer still present).
- [ ] **Step 2:** Run — FAIL (no goldens / old frame).
- [ ] **Step 3:** Implement `render` from `layoutFor`; delete `topBar`/`statusBar` (move the `statusBar` tests in `app_test.go` to `drawerView`/header).
- [ ] **Step 4:** `go test ./internal/tui/... -update`, review every changed golden for: frame intact, no lost content, `gate` wording; then `go test ./...` PASS.
- [ ] **Step 5:** Commit `feat(tui): the dashboard frame: header, views sidebar, boxed main, drawer`.

### Task 6: Phase-1 verification

- [ ] `gofmt -l .` empty; `go vet ./...` and `go vet -tags notui ./...` clean; `go test ./...` PASS; `go list -tags notui -deps ./cmd/jumpgate | grep -E 'charm|charmbracelet'` empty.
- [ ] Whole-branch review; fix findings; commit.

---

## Phase 2: KPI strip + alerts (U2)

**Files:** Create `internal/tui/alerts.go`, `internal/tui/alerts_test.go`, `internal/tui/kpi.go`, `internal/tui/kpi_test.go`; modify `layout.go`, `layout_test.go`, `frame.go`, `frame_test.go`, `app.go` (gateways feed); goldens. `host_modules.go` is not touched.

### Task 7: `alerts()` behind `alertSource`

**Files:** Create `internal/tui/alerts.go`, `internal/tui/alerts_test.go`.

**Interfaces:**
- Produces:
  ```go
  type alertSev int // sevWarn, sevFail; String() "warn" | "FAIL"
  type alert struct{ Sev alertSev; Box, Text string } // Text sanitized, URLs masked
  // alertInput is what the TUI holds; Has flags say a reading exists.
  type alertInput struct {
      Fleet    api.Fleet; FleetHas bool
      Gateways []api.GatewaySummary; GatewaysHas bool
      Now      time.Time
  }
  type alertSource interface{ alerts(in alertInput) (list []alert, known bool) }
  type derivedAlerts struct{} // the v1 source: alerts(in), known == in.FleetHas
  func alerts(in alertInput) []alert // pure; FAIL first, then box, then text
  ```
- Rules (spec §3), one alert each:
  - box down (`!Reachable && LastSeen != nil`): FAIL `down <age>`; a down box gets no other alert (its error and stale sections are why it is down).
  - row `Error` (not `target_not_set_up`, which is a state, not a failure): FAIL `<code>: <message>`; `StatusError`/`DiskError`/`FirewallError`: FAIL `<section>: <message>`.
  - stale: `Stale.Status/Disk/Firewall`, or `Status.At` older than 2 × `IntervalSeconds`: warn `<section> stale`.
  - firewall `Grade` `fail` → FAIL, `warn` → warn (`firewall fail|warn`).
  - disk `Fit` `short` → FAIL `disk short`, `tight` → warn `disk tight`.
  - `Jobs.NeedsDecision > 0`: warn `N needs decision`.
  - gateway: each warning → warn (Box = label or id, `maskText`); a known state other than `running` → FAIL `gateway <state>`.
  - module errors: the App holds no module readings in phase 2 (they live on the host screen); the rule lands with the tree in phase 3.
- `App.alertSrc alertSource` defaults to `derivedAlerts{}`; the health-watch feed replaces it later and keeps this one as the fallback (U2).

- [ ] **Step 1: Failing tests**
  - `TestAlertsSampleFleet`: `alerts` of `dashFleet()` = `[FAIL box-b "down 4m", warn box-a "1 needs decision", warn box-a "firewall warn"]` (order FAIL first, box, text); the laptop (not set up) gives none.
  - `TestAlertsEachRule`: table, one row per rule above → exactly one alert with the given Sev and Text.
  - `TestAlertsStaleTwoIntervals`: `Status.At` at 31 s with interval 15 → warn `status stale`; at 29 s → none.
  - `TestAlertsGatewayWarningMasked`: a warning containing `https://u:p@h/key` → text has no `u:p` and no `key`.
  - `TestAlertsUnknownWithoutFleet`: `derivedAlerts{}.alerts(alertInput{})` → `known == false`.
  - `TestAlertsDeterministic`: same input twice (rows shuffled) → equal lists.
- [ ] **Step 2:** `go test ./internal/tui -run TestAlerts` — FAIL (undefined).
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** PASS.

### Task 8: KPI tiles

**Files:** Create `internal/tui/kpi.go`, `internal/tui/kpi_test.go`.

**Interfaces:**
- Consumes: `alertInput`, `alerts`.
- Produces:
  ```go
  type tile struct {
      Title string    // Boxes, Synced, Head lag, Peers, Disk, Gateways, Alerts
      Lines [2]string // the boxed tile's two rows
      Short string    // the compact strip's cell, e.g. "synced ~1/3"
  }
  func kpis(a *App, in alertInput, al []alert, alertsKnown bool) []tile // 7 tiles, fixed order
  func (a *App) kpiView(r rect) string // 1 row: Shorts joined by " | " (then two spaces, then one, before anything is cut); 4 rows: boxed tiles side by side, Gateways 27 wide so its hint fits and Alerts 18, the rest sharing what is left
  ```
- Values: a value, `n/a` (no data), `~value` (some input stale: `Stale.*` or the stream retrying), `FAIL <code>` (no value and the section errored). Never 0 for unknown; aggregates name unknowns (`1/3 synced · 1 n/a`).
  - Boxes: `N up M down`, loading (`CheckedAt == nil`, never seen) counted as `n/a`; `n/a` before the first fleet.
  - Synced: rows with `Status.Overall == synced` / rows, `· N syncing`, `· N n/a` for nil `Status`.
  - Head lag: max `HeadLag` over rows with one, `· N n/a` for the others; none → `n/a`.
  - Peers: exec/beacon sums over rows with `Status`, `~` if any of them is stale, `· N n/a`.
  - Disk: min free (`100 − DiskUsedPct`) and the box with it; none → `n/a`, or `FAIL <code>` from `DiskError`.
  - Gateways: `N live M failing` (`running` = live, a known other state = failing) and `n/a  no request counts yet` (U1); before the first `Gateways()` reply `n/a`.
  - Alerts: `N FAIL M warn` and the first alert; `0` when the list is known and empty; `n/a` when unknown.

- [ ] **Step 1: Failing tests**
  - `TestKPIsSampleFleet`: on `dashFleet()` + one running gateway, Short / Lines:
    Boxes `boxes 1/3 up` / `1 up 1 down`, `1 n/a of 3`;
    Synced `synced ~1/3` / `~1/3 synced`, `1 sync 1 n/a`;
    Head lag `lag 2` / `max 2 blk`, `2 n/a`;
    Peers `peers ~51/81` / `~51 exec`, `~81 bcn 1 n/a`;
    Disk `disk 39%` / `min 39% free`, `on box-a`;
    Gateways `gw 1/1 live` / `1 live 0 failing`, `n/a no request counts yet`;
    Alerts `! 1 FAIL 2 warn` / `1 FAIL 2 warn`, `box-b down 4m`.
    Without a gateways reply: `gw n/a` / `n/a`, `n/a no request counts yet`.
  - `TestKPIsUnknownNeverZero`: fleet of three rows with every section nil (and no gateways reply) → every tile's Short and Lines contain `n/a` and none matches `\b0\b` except a known count.
  - `TestKPIsFailFromSectionError`: one row, nil `Disk`, `DiskError{Code: "disk_probe"}` → Disk Short `disk FAIL disk_probe`.
  - `TestKPIGatewaysHint`: Gateways tile Lines[1] contains `no request counts yet`.
  - `TestKPIViewFits`: `kpiView` at w 80..200 with H 1 and 4 → exactly H lines, each ≤ w cells; ascii passes `requireASCII`.
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** PASS.

### Task 9: The KPI region, alert counts and the gateways feed

**Files:** Modify `layout.go`, `layout_test.go`, `frame.go`, `frame_test.go`, `app.go`; goldens.

**Interfaces:**
- `layout.KPI rect`: Y=1, H=1 when `h < 30`, H=4 (boxed) at `h >= 30`; Main starts below it. Main floor (≥ 50×8) still holds at every size.
- `func (a *App) alertCount() string` — `! N alerts` (`! alerts n/a` when unknown).
- Header right: `! N alerts · <age>` (alerts in Bad/Warn colour when N > 0).
- Drawer line: `! alerts n/a` becomes `alertCount()`.
- App keeps `gws []api.GatewaySummary`, `gwsHas bool` from every `gatewaysMsg` (an error keeps the last list); `Init` adds `loadGateways(a)` and a 30 s `gwTickMsg` that reloads. No new box polling: `/api/gateways` reads the server's store.

- [ ] **Step 1: Failing tests**
  - `TestLayoutForKPI`: (80,24) KPI.H == 1, Main.Y == 2; (100,30) KPI.H == 4, Main.Y == 5; `TestLayoutForMainFloor` adds KPI.H to the row sum and the overlap check.
  - `TestHeaderAlertCount`: dash fleet → header contains `! 3 alerts`, line ≤ w at 80 and 120; before the fleet `! alerts n/a` is not in the header (right says `connecting`).
  - `TestDrawerLineAlertCount`: dash fleet at 80×24 → last line contains `! 3 alerts`; `TestDrawerLineFailsClosed` keeps `! alerts n/a` before the fleet.
  - `TestGatewaysFeed`: `Init` calls `Gateways`; a `gatewaysMsg` sets `a.gws`; `gwTickMsg` returns a reload command; an error reply keeps the old list.
  - Goldens: regenerate `dash_*`; add `dash_degraded_120x40` (three rows: status nil, disk nil with `DiskError`, firewall nil, stale flags, one down, no gateways reply) — `requireASCII`, and `TestDegradedNoZeroForUnknown`: the KPI rows of that frame contain `n/a` per tile and no bare `0`.
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./internal/tui/... -update`, review every golden diff; `go test ./...` PASS.
- [ ] **Step 5:** Commit `feat(tui): the KPI strip and client-derived alerts`.

### Task 10: Phase-2 verification

- [ ] `gofmt -l .` empty; `go vet ./...` and `go vet -tags notui ./...` clean; `go test ./...` PASS; `go list -tags notui -deps ./cmd/jumpgate | grep -E 'charm|charmbracelet'` empty.

## Phase 3: Tree + activity-log sidebar, Hosts into Fleet (U3, U6)

**Files:** Create `internal/tui/actlog.go`, `actlog_test.go`, `tree.go`, `tree_test.go`, `dash_test.go` (phase-3 flows); modify `layout.go`, `layout_test.go`, `alerts.go`, `alerts_test.go`, `frame.go`, `frame_test.go`, `app.go`, `keys.go`, `fleet.go`, `hosts.go`, `host.go` (the `[` host list goes: the tree replaces it), `palette.go`, `theme.go` (tree and severity glyphs), the tests that open screens by number; goldens. `host_modules.go` is not touched: the tree builds a throwaway `hostScreen{modCat, modTarget, modView}` and calls `moduleCards`/`moduleStatus` on it, read-only.

### Review Focus (phase 3)

6. `tab`/`shift+tab` now cycle focus regions, so they leave `navKeys.Left/Right`; box-detail tabs stay on `← → h l`, and a screen with a text field (host-add form, filters) still gets `tab` while it captures. Test in Task 15 (`TestTabCyclesFocus`, `TestHostTabsStillOnArrows`).
7. The log is built only from snapshots the App already holds; nothing is added to any tick except the tree's 60 s module refresh, which asks only for *expanded* boxes. Test in Task 13 (`TestTreeModulesOnlyWhenExpanded`).
8. The first reading of each kind logs nothing (no flood of "synced" at start-up); an alert whose text carries an age (`down 4m` → `down 5m`) is the same alert, not a clear and a raise. Test in Task 12 (`TestDiffEventsFirstReadingSilent`, `TestDiffEventsAlertAgeIsStable`).
9. Every Hosts test still passes, now opened from Fleet with `v`; `d`/`p` from the plain table act on the table selection. Test in Task 14.
10. Main keeps ≥ 50×8 with the right sidebar; overlays never change the layout, they only draw over main. Test in Task 11 and `TestFrameLinesFitWindow` with overlays.

### Task 11: The right region in `layoutFor`

**Files:** Modify `layout.go`, `layout_test.go`.

**Interfaces:** Produces `layout.Right rect` — at `w >= 120` (`rightMinWidth`), `W = clamp(w*27/100, 32, 40)`, `X = w - W`, same Y/H as Main; else zero. Main is `w - Left.W - Right.W`; if that would drop below 50 the Right region folds.

- [ ] **Step 1: Failing tests**
  - `TestLayoutForMainFloor`: `Left.W+Main.W+Right.W == w`; Right in the overlap check.
  - `TestLayoutForRight`: (119,40) Right.W 0; (120,40) Right {88,5,32,30}, Main.W 64; (148,40) 39; (160,40) 40; (200,60) 40; (100,30) 0.
- [ ] **Step 2:** `go test ./internal/tui -run TestLayoutFor` — FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(tui): a right region in layoutFor, 27% of the width from 120 columns`.

### Task 12: The activity log (U6)

**Files:** Create `actlog.go`, `actlog_test.go`; modify `alerts.go`, `alerts_test.go`, `theme.go`.

**Interfaces:**
- Produces:
  ```go
  type logSev int // logInfo, logOK, logWarn, logFail
  type logEvent struct{ At time.Time; Sev logSev; Box, Node, Text string } // Text sanitized + masked
  type snapshot struct {
      Fleet api.Fleet; FleetHas bool
      Jobs []api.JobView; JobsHas bool       // the inbox, while the Jobs view watches it
      Gateways []api.GatewaySummary; GatewaysHas bool
      Targets []api.TargetView; TargetsHas bool
      Modules map[string]api.ModulesView     // expanded boxes only (Task 13)
      Now time.Time
  }
  func diffEvents(prev, next snapshot) []logEvent // pure; At = next.Now
  type actLog struct{ ev []logEvent }             // newest last, at most logKeep = 200
  func (l *actLog) add(evs ...logEvent)
  func (l *actLog) scoped(box, node string) []logEvent // newest first; box "" = fleet
  func alertKey(x alert) string // box + text with numbers folded: "down 4m" == "down 5m"
  ```
- Kinds (one event each): gate/box up and down (`Reachable` flips); box added/removed (fleet rows); paired (`Agent` nil → set in the targets: the signer approval this data shows); exec/beacon sync state change (`exec synced (was syncing)`), `exec stalled at N` (syncing, head unchanged, reading newer); job started, step k/n, needs decision, decision answered (the inbox when held, else the fleet `Jobs.NeedsDecision` count); alert raised (its own Sev) / cleared (ok), keyed by `alertKey`; gateway state change; module on/off/error and `gate stale` per expanded box.
- `alerts()` gains: disk used ≥ 95 % FAIL / ≥ 85 % warn (`disk 96% used`), exec peers 0 with a reading (warn `exec peers 0`), head lag > 64 blk (warn `head lag N blk`). The sample fleet gives no new alert.
- Glyphs: `Glyphs.Info/OK/Warn/Fail` severity marks (`·`/`-`, `✓`/`+`, `▲`/`!`, `✗`/`x`), `Open/Closed` tree marks (`▾`/`-`, `▸`/`+`).

- [ ] **Step 1: Failing tests**
  - `TestDiffEventsKinds`: table, one prev/next pair per kind above → exactly the listed event (Sev, Box, Node, Text).
  - `TestDiffEventsFirstReadingSilent`: `diffEvents(snapshot{}, next)` with every kind present → none.
  - `TestDiffEventsAlertAgeIsStable`: box-b down at 4 m then 5 m → no event.
  - `TestDiffEventsMasked`: a gateway state carrying `https://u:p@h/key` → no `u:p`.
  - `TestActLogRingCap200`: 250 adds → 200 kept, the oldest 50 gone.
  - `TestActLogScope`: events on box-a/exec, box-a, box-b → `scoped("", "")` all three newest first; `scoped("box-a","")` two; `scoped("box-a","exec")` one.
  - `TestAlertsDiskPeersLag`: 84 % none, 85 % warn, 95 % FAIL; peers 0 warn; lag 65 warn, 64 none.
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(tui): the activity log, from diffs of the snapshots the TUI holds`.

### Task 13: The fleet tree

**Files:** Create `tree.go`, `tree_test.go`; modify `app.go`.

**Interfaces:**
- Produces:
  ```go
  type treeNode struct{ Kind, Box, Node, Label, State string } // Kind: fleet|box|exec|beacon|gw|mod|note
  func (a *App) treeNodes() []treeNode   // fleet root, then each fleet row (table order); children when expanded
  func (a *App) treeView(w, h int) string
  func (a *App) toggleExpand(box string) tea.Cmd // space: expands, and fetches modules the first time
  type treeModsMsg struct{ box string; cat []api.CatalogModule; target *api.TargetView; view api.ModulesView; err error }
  type treeTickMsg struct{} // every 60 s: re-reads the modules of expanded boxes only
  ```
- App: `expanded map[string]bool`, `mods map[string]treeMods`, `treeCur int`, `selNode string`; `a.sel` stays the one selection (tree and table).
- Children of an expanded box: `exec <state> <peers>p`, `beacon <state> <peers>p` (n/a without Status), `gw <label> <live|state>` for each gateway with `RoutesTo(box)`, `mod <name> <on|off>` from `moduleCards` (`gate stale` when `GateStale`), `modules loading…` before the first reply, `modules n/a` on error.

- [ ] **Step 1: Failing tests**
  - `TestTreeNodesFromFleet`: dash fleet → fleet, box-a, box-b, laptop; collapsed.
  - `TestTreeExpandShowsNodesGatewaysModules`: expand box-a with a gateway routing to it and one routing elsewhere, Fake `ModulesV`/`CatalogV` → exec, beacon, `gw rpc-main live`, `mod msgboard on`; the other gateway absent.
  - `TestTreeModulesOnlyWhenExpanded`: no `ListModules` call before space; one after; `treeTickMsg` re-reads box-a only, and returns the next tick.
  - `TestTreeGateStale`: `GateStale` → the module line says `gate stale`.
  - `TestTreeViewASCII`: the tree view under ascii passes `requireASCII`, names truncated with `~` at width 22.
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(tui): the fleet tree: boxes, nodes, gateways and modules`.

### Task 14: Hosts folds into Fleet; views renumber

**Files:** Modify `app.go` (`screenNames` = Fleet, Jobs, Signers, Gateways, Settings), `keys.go` (`1-5`), `fleet.go` (owns a `hostsScreen`; `v` toggles table mode; `a p d L` act on `a.sel`), `hosts.go` (keys reach it through Fleet), `palette.go` (`hosts` opens Fleet in host mode, `add` starts the flow there), `frame.go` hints, tests (`hostsApp` presses `v`; `5`→`2`, `6`→`5`).

**Interfaces:** `fleetScreen.hosts *hostsScreen`, `fleetScreen.hostMode bool`; `func (s *fleetScreen) selectID(a *App, id string)`; `func hostsOf(a *App) *hostsScreen`. `Init` reads the targets once (the server's store; no box is polled).

- [ ] **Step 1: Failing tests**
  - `TestViewsRenumbered`: `2` Jobs, `3` Signers, `4` Gateways, `5` Settings; `6` does nothing.
  - `TestFleetHostModeV`: `v` shows the address/pairing table (`root@10.0.0.5:22`, `not paired`), `v` again the fleet table.
  - `TestFleetRemoveActsOnSelection`: plain table, `j` (box-b), `d` → the confirmation names box-b.
  - `TestFleetPairActsOnSelection`: `p` on box-b starts the re-pair flow for box-b.
  - Every `hosts_test.go` test, now through `hostsApp` = Fleet + `v`.
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Implement; regenerate `hosts.golden`.
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(tui): Hosts folds into Fleet; views are 1-5`.

### Task 15: Sidebars, overlays and focus in the frame

**Files:** Modify `frame.go`, `app.go`, `keys.go`, `host.go`; create `dash_test.go`; goldens.

**Interfaces:**
- `type focusRegion int // focusMain, focusTree, focusLog`; `App.focus`, `App.treeOver`, `App.logOver`.
- `globalKeys.Focus` (`tab`, `shift+tab` back), `globalKeys.Tree` (`[`), `globalKeys.Log` (`]`), `navKeys.Expand` (`space`).
- `func (a *App) logView(w, h int) string` — boxed `Log: fleet` / `Log: box-a` / `Log: box-a exec`, lines `<glyph> <age> <box> <text>`, newest first; `no activity yet` when empty.
- `func overlay(base, panel string, x int) string` — splices panel lines over base from column x (ANSI-aware `ansi.Cut`).
- Left (≥ 100): Views, then the tree. Right (≥ 120): the log. Below 120 `]` draws the log over main's right 34 columns; below 100 `[` the tree over its left 34; `esc` closes an overlay first; where a region shows, `[`/`]` focus it.
- Focused box: info colour, and in ASCII a `=` top border and `>` before the title.
- Focus on the tree: `↑↓ j k` move it (and the table cursor), `space` expands, `enter` opens the box; on the log: `↑↓` scroll.

- [ ] **Step 1: Failing tests**
  - `TestTabCyclesFocus`: 120×40 main → tree → log → main; `shift+tab` back; at 80×24 with no overlay `tab` stays on main.
  - `TestHostTabsStillOnArrows`: box detail, `right` moves the tab, `tab` moves focus, not the tab.
  - `TestTreeEnterOpensBoxAndLogFollows`: focus tree, `j` (box-a), `enter` → detail box-a, log title `Log: box-a`; `j` in the tree → the table cursor follows.
  - `TestResizeFoldsSidebarsKeepsSelection`: 120×40 select box-b, resize 80×24 → no `Views`, no `Log:`, `>box-b` still selected, focus back on main.
  - `TestLogOverlayToggles`: 80×24 `]` → `Log:` in the frame, `esc` → gone; at 120 `]` focuses the log.
  - `TestTreeOverlayToggles`: 80×24 `[` → the tree drawn; `esc` closes.
  - Goldens: `dash_*` regenerated; add `dash_80x24_tree_overlay`, `dash_80x24_ctx_overlay`, `dash_box_detail_120x40`; all ascii through `requireASCII`; `TestFrameLinesFitWindow` covers the overlay cases.
  - `TestNarrowFramesKeepTheirLastRows` still passes (help and settings at 80×24).
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Implement; drop `App.sidebar`, `sidebarView`, `hostList`.
- [ ] **Step 4:** `go test ./internal/tui/... -update`, review every golden; `go test ./...` PASS.
- [ ] **Step 5:** Commit `feat(tui): tree and activity-log sidebars, overlays and focus regions`.

### Task 16: Phase-3 verification

- [ ] `gofmt -l .` empty; `go vet ./...` and `go vet -tags notui ./...` clean; `go test ./...` PASS; `go list -tags notui -deps ./cmd/jumpgate | grep -E 'charm|charmbracelet'` empty.

## Width fix (after phase 3)

At 120×40 the fleet table lost DISK and FW (main was 62 inside its box). `layoutFor` now keeps a boxed main of `mainFleetW = 82` (the default columns' 80 cells + 2 borders) from 100 columns: left `clamp(w-82, 18, 22)`; right docked only at `w >= 140` (`rightMinWidth`), `clamp(w/5, 28, 40)`, folding if main would drop below 82; below 140 `]` overlays the log. The views list pads names to the sidebar width. Tests: `TestLayoutForRight` (120 → 0, 139 → 0, 140 → 28, 160 → 32, 200 → 40), `TestLayoutForMainFitsFleet` (w 100–200: main ≥ 82), `TestMainFleetWMatchesDefaultColumns`, `TestFleetFrameKeepsDefaultColumns` (100×30, 120×40, 140×40, 160×48: DISK, FW and `warn`); the phase-3 docked-log tests move to 140×40, plus `]` at 120 opens the overlay. Goldens: `dash_*` regenerated, new `dash_140x40`, `dash_160x48`.

## Phase 4: Activity drawer bars (U7)

`func drawerBars(...) []bar` pure: running job step k/n (`WatchJobs`, reuse `jobs.go` step format), EL sync % (head vs highest), CL sync %, snapshot download/extract if exposed, fleet rollout n/m boxes, disk fill per box only when ≥ 85 %. No bar without a denominator. Order needs decision → running jobs → syncing → disk; cap 3 bars at h < 34 (80×24), 6 at ≥ 34 (120×40), then `+N more`. Tests: ordering, caps and `+N more`, no-denominator exclusion, disk 84 % vs 85 %. Goldens `dash_jobs_120x40`, `dash_80x24_drawer_open`; teatest: job advances, bar moves; stream drop → stale → recovers.

## Phase 5: Node detail (U5)

Node detail main view with current values and the delta from the previous reading (`head lag 2 blk (was 40)`); no sparklines or trend graphs. Golden `dash_node_detail_120x40`. Landing `docs-images.mjs` points at `dash_120x40` / `dash_80x24`.

## Phase 6: Gateway request counts (U1)

Server: request/error counters + rate per gateway in `api.GatewaySummary` and `/api/gateways`; TUI tile shows them and drops the hint.
