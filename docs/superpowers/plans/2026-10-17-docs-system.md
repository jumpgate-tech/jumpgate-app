# Docs System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One docs source in `docs/`, made of atomic steps ordered into pathways, checked against the code by `go test ./...`. It renders to jumpgate.app/docs and to the TUI's Help view, and every step can show its previous steps.

**Architecture:** `docs` is a Go package. It loads `steps/**/*.md` and `pathways/*.yaml` into one `Model`, parses the step dialect into blocks, resolves typed references against `Tables`, checks the dependency graph, and computes the recap. `docs/covers` hashes the Go declarations marked `// docs:covers`. The tables come from the code: the CLI registry (snapshotted to `docs/reference/cli.json`, because `docs` cannot import package `main`), `tui.Keymap()`, `tui.Commands()`, the network catalog, `api.ExitClasses()`, `envvars.All()` and the scene registries. `internal/docstables.Live()` assembles them. `cmd/docsgen` renders the model to static HTML (`internal/docsite`), TUI scenes to SVG (`internal/tui/scenes`, `internal/ansisvg`) and fixtures for Playwright. The TUI renders the same model in the v0.5.2 Help view.

**Tech Stack:** Go 1.26 (`go 1.26.0` directive), stdlib `go/parser`, `go/printer`, `go/format`, `html/template` and `crypto/sha256`. One new module, `go.yaml.in/yaml/v3 v3.0.4`. Bubble Tea v2 and lipgloss v2 are already in `go.mod`. `@playwright/test` 1.64.1 in `mcr.microsoft.com/playwright:v1.64.1-noble`, and Pagefind 1.5.2 via `npx`. GitLab CI on git.craftsmanfounder.com, plus the GitHub release workflow.

**Spec:** `docs/superpowers/specs/2026-10-17-docs-system-design.md`. Research: `.claude/research/docs-system.md`. Executors read both before Task 1.

## Preconditions

1. **`feat/tui-guide` (v0.5.2) is merged to `main`.** Every path in this plan is on post-merge `main`: `docs/guide/guide.go` (`guide.Guide`, `Topics() []Topic`), `internal/tui/reference.go` (`Keymap() []KeyDoc`, `Commands() []string`, `keymaps`), `internal/tui/markdown.go` (`renderMarkdown`, `inline`, `asciiText`), `internal/tui/guide.go` (`guideModal`, `guideKeys`, `newGuide`), and `internal/tui/guide_test.go`. If it is not merged, stop and ask.
2. Work on branch `feat/docs-system` from that `main`. Baseline: `go test ./...` green. Record the result.
3. Phase 4's preview job needs the GitLab CI variables `CLOUDFLARE_API_TOKEN` (Pages edit, masked) and `CLOUDFLARE_ACCOUNT_ID` on project 71. Use the valaxyblackdev Cloudflare account, which owns the `jumpgate` Pages project, and set them yourself. Phase 6's release job uses `GITHUB_TOKEN` only.
4. Task 22 runs in the landing repo (`~/Documents/jumpgate-tech/github/landing`). Another agent is editing it now, so start Task 22 only once its working tree is clean and that work is merged. Until then, read the landing repo only (with `git -C … show main:<path>`).
5. Never push without the user's permission.

## Phases

Each phase ends green and is reviewed on its own. The order is the spec's rollout.

| Phase | Tasks | Ships |
|---|---|---|
| 1. CLI registry, exit and env tables | 1–3 | `commands` table, `api.ExitClasses`, `internal/envvars`, `docs/reference/cli.json`, key ids, network slugs |
| 2. The `docs` package and the content | 4–9 | loader, graph and recap, dialect and references, `docs:covers` hashing, live check, every v0.5.2 and landing page as steps |
| 3. TUI Help on the step model | 10–12 | reference pages, Help navigation, recap pane, `docs/guide` removed |
| 4. The website | 13–16 | `internal/docsite`, recap HTML, landing URL redirects, `cmd/docsgen`, Pagefind, `docs-site` and preview jobs |
| 5. Scenes | 17–20 | TUI scenes, ANSI→SVG, Playwright app scenes and site checks, desktop scenes on the Windows VM and macOS runners |
| 6. Publishing | 21–22 | `docs-site.tar.gz` on the GitHub release; the landing deploy pulls `/docs` and the interim renderer is retired |

## Global Constraints

- Module `github.com/valve-tech/jumpgate`. The product, binary and new commands are `jumpgate` / `docsgen`. Nothing new says "valve-node-app".
- `go 1.26.0` stays. The one new module is `go.yaml.in/yaml/v3 v3.0.4`. The `notui` build must still link no `charm.land/` or `github.com/charmbracelet/` package (the existing `go-linux` check).
- Content layout, verbatim: `docs/steps/<area>/<id>.md`, `docs/pathways/<id>.yaml`, `docs/images/<name>.(png|svg)`, `docs/images/<name>.yaml`.
- Step frontmatter fields: `id`, `title`, `summary` (one line), `recap` (2–3 lines), `image`, `requires`, `verified`. Every field is required except `image`, `requires` and `verified`. `verified` becomes required once any code is marked as covering the step.
- Pathway fields: `id`, `title`, `audience` (`newcomer` | `operator`), `after`, `steps`.
- Step bodies use no raw HTML. Only headings, paragraphs, lists, fenced code, inline code, emphasis and links are allowed.
- Covered item: a func or method marked `// docs:covers <step-id>`. Its hash is SHA-256 over the declaration's source after `gofmt` normalisation, with comments stripped, truncated to **12 hex characters**.
- References, verbatim: `{{cmd "hosts add"}}`, `{{flag "hosts add" "--jump"}}`, `{{key fleet.sort}}`, `{{palette "logs"}}`, `{{network pls}}`, `{{client lighthouse}}`, `{{exit 4}}`, `{{env JUMPGATE_ASCII}}`, `{{step pair-box}}`, `{{shot tui fleet}}`, `{{shot app machines}}`, `{{shot desktop win-smartscreen}}`. This plan adds `{{image <name>}}` (D4).
- Every failure names `file:line`. A hash mismatch prints the covered item, the old and the new hash, and the exact frontmatter line to change.
- Website: URLs are `/docs/<pathway>/<step>/`. `rel=canonical` points at the first pathway the step is in. The sidebar is *Get started* (newcomer), then *Operate* (operator), then *Reference*. It uses the landing palette. The recap needs no JavaScript (`<details>`). The optional script uses try/catch around storage.
- TUI: everything fits 80x24 and works under the ascii glyphs and NO_COLOR. Style is never the only signal.
- Generated images are never committed. `out/` is gitignored.
- GitLab is git.craftsmanfounder.com, never gitlab.com. GitHub carries the public release.
- Every commit message ends with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Commit by exact paths, never `-A`.

## Decisions made in this plan

The spec leaves these open, and each one is decided here. A reviewer can object to any of them.

| # | Decision | Why |
|---|---|---|
| D1 | The CLI table stays in `cmd/jumpgate` (`commands []command`). `TestCLIReferenceIsCurrent` writes it to `docs/reference/cli.json` under `-update` and fails on any diff. `docs` embeds that file. | Package `docs`, `cmd/docsgen` and the TUI cannot import `main`. The handlers depend on `main`-only code (`serverOptions`, `readToken`), so moving them out is a large refactor. A checked snapshot keeps "the build fails" true. |
| D2 | `go.yaml.in/yaml/v3` parses frontmatter and pathways through `yaml.Node`. | Its node line numbers give `file:line` for every field. A hand-written parser would have to handle `recap: \|` block scalars and flow maps. |
| D3 | `verified` must be a one-line flow map with keys sorted, e.g. `verified: { cmd/jumpgate.hostsAdd: 81c0aa7713fe }`. `image` must be quoted: `image: "{{shot tui fleet}}"`. An unquoted `{{` is a YAML flow map, and the loader reports it with the quoted fix. | Without this, "the exact frontmatter line to change" is not one line, and the spec's own example does not parse as YAML. |
| D4 | Hand-made images are referenced as `{{image <name>}}` (in the body or in `image:`). A sidecar's `covers` may name a Go declaration (`cmd/jumpgate.hostsAdd`) or a repo file path (`cmd/jumpgate/build-macos-app.sh`). A file path is hashed whole: SHA-256 of the bytes with CRLF turned into LF, 12 hex characters. | The spec has no reference for hand-made images, and macOS packaging is shell, not Go. |
| D5 | Covered-item ids are `<module-relative dir>.<Func>` or `<dir>.<Type>.<Method>` (the receiver without `*`), e.g. `cmd/jumpgate.hostsAdd` or `internal/tui.App.View`. | This matches the spec's examples. The spec's `internal/setup.Pair` and `cmd/jumpgate.buildMacApp` do not exist, so the content uses real functions. |
| D6 | Pathway order (sidebar, recap grouping, canonical): topological over `after`, ties broken by `id`. | The spec does not fix an order across pathways. |
| D7 | Recap groups are ordered by pathway order, with the current pathway last and open, and the others collapsed. A required step that is in several pathways is grouped under the current pathway when it appears there earlier; otherwise under the first pathway in order that is reachable through `after` and contains it. A step reached by two paths appears once. | This fills the gaps in spec §3 "Recap". |
| D8 | Every command that is not hidden must be named by `{{cmd}}` in at least one step. Flags are covered by the generated commands page. | "Referenced by a step or a reference page" is always true, because the reference page lists every command. |
| D9 | `catalog.Network` gains `Slug` (`eth`, `pls`, `pls-v4`) and `catalog.Client` gains `Name` (display name). `tui.KeyDoc` gains `ID` (`<keymap>.<field>` lower-case, e.g. `fleet.sort`). | `{{network pls}}`, `{{client lighthouse}}` and `{{key fleet.sort}}` need ids that the tables do not have. |
| D10 | The TUI gets a scene's frame through `tui.Options.Scene` (nil: caption only). `cmd/jumpgate` sets it to `scenes.Text`. | `internal/tui/scenes` imports `internal/tui`, so the TUI cannot import it. |
| D11 | Hand-made image sidecars are not embedded in the binary. The TUI shows `{{image x}}` as `[image x: jumpgate.app/docs/<pathway>/<step>/]`. | `go:embed` fails on an empty pattern, and no hand-made image exists yet. The website and the checks read the repo tree. |
| D12 | The tag-time docs site is built on GitHub (the release lives there). Desktop scenes come from `windows-latest` and `macos-latest` running the same scripts. GitLab's `docs-shots-desktop-*` jobs on the Windows VM runner (`windows` tag) and the macOS runner (`macos` tag) are the gate on tags and on packaging changes. | The spec allows either runner set. Releases are created by `.github/workflows/release.yml`. |
| D13 | The palette form is `help [PAGE]`, where PAGE is a step, pathway or reference page id. It replaces `help [TOPIC]`. | Topics no longer exist. |
| D14 | `*.png` stays gitignored, with the exception `!/docs/images/*.png` for hand-made images only. | Generated images are never committed. |
| D15 | *Run jumpgate on a server* (spec §5) is not created here. It belongs to the server-install plan, because the feature does not exist yet. | A pathway about code that does not exist would have nothing to cover. |

## Review Focus

1. **`{{` inside code.** A shell or Go-template example in a fenced block or in backticks (e.g. `` `{{.Name}}` ``) is shown literally, not parsed as a reference and not reported as an unknown one. Test: Task 6, `TestRefsIgnoredInCode`.
2. **A Windows checkout (CRLF).** On the Windows runner, step files and Go sources may arrive with CRLF. The frontmatter must still parse with the same line numbers, and a declaration must hash to the same 12 characters as with LF. Tests: Task 4, `TestLoadCRLF`; Task 7, `TestHashIgnoresCRLFAndComments`.
3. **HTML escaping on the website.** Prose like `a < b & c`, or a summary with `"` or `<`, must render as text in the body, `<title>`, `<meta name=description>` and the recap. Raw HTML is banned, but these characters are not. Test: Task 13, `TestEscapesProseAndFrontmatter`.
4. **A blank desktop capture.** A locked screen or session 0 on the VM gives a PNG of one colour, or no PNG. That must fail the job, not ship a black image. Test: Task 20, `TestShotsCheckRejectsBlankAndMissing`.
5. **Id collisions in URLs.** A pathway id `reference`, `img` or `pagefind`, a step id equal to its own pathway's id, or a redirect source that is also a live page would overwrite a page or route it wrongly. Those must fail the build with `file:line`. Tests: Task 5, `TestReservedIDs`; Task 15, `TestRedirectNeverShadowsAPage`.

---

## Phase 1: CLI registry, exit and env tables

### Task 1: One command table drives dispatch, help and flags

**Files:**
- Create: `cmd/jumpgate/commands.go`
- Modify: `cmd/jumpgate/cli.go` (`subcommands`, `dispatch`), `cmd/jumpgate/help.go` (remove `commandHelp`; `printHelp` reads `commands`), `cmd/jumpgate/cli_hosts.go` (`cmdHosts`, `hostsAdd`, `hostsForgetKey`), `cmd/jumpgate/cli_keys.go` (`cmdKeys`, `keysShow`, `keysInit`), `cmd/jumpgate/cli_intents.go` (`cmdLogs`), `cmd/jumpgate/cli_relay.go`, `cmd/jumpgate/buildserver.go` (`newServeFlagSet`), `cmd/jumpgate/cli_agent.go`, `cmd/jumpgate/cli_jobs.go`
- Test: `cmd/jumpgate/commands_test.go`, `cmd/jumpgate/help_test.go`

**Interfaces:**
- Produces:
  ```go
  type command struct {
      Name    string              // "hosts add": the words that select it
      Usage   string              // "jumpgate hosts add NAME (--ssh USER@HOST[:PORT] [--key PATH] [--jump USER@HOST[:PORT]] [--sudo] | --local)"
      Summary string              // the line `jumpgate help` prints under Usage
      Hidden  bool                // left out of help and docs (agent)
      Flags   func(*flag.FlagSet) // registers the flags; nil for none
      Run     func(args []string) int
  }
  var commands []command // help order: the order of today's commandHelp
  func lookup(args []string) (c *command, rest []string, ok bool) // longest Name whose words prefix args
  func (c *command) flagSet() *flag.FlagSet                       // NewFlagSet(c.Name, ContinueOnError) + c.Flags
  func (c *command) usage() int                                   // prints "jumpgate: usage: "+c.Usage, returns exitCode("usage")
  ```
  Each handler that has flags gets a typed constructor that the registry's `Flags` also calls, so there is one source: `hostsAddFlags(fs) *hostsAddOpts`, `hostsForgetKeyFlags(fs) *bool`, `logsFlags(fs) *int`, `keysShowFlags`, `keysInitFlags`, `relay` (`addRelayFlags` with an empty getenv), `serve` (`newServeFlagSet` + `addServerFlags` with an empty getenv), `agent` (its subcommand flags).

- [ ] **Step 1: Write the failing tests**

```go
// commands_test.go
func TestLookupTakesTheLongestName(t *testing.T) {
	c, rest, ok := lookup([]string{"hosts", "add", "box-a", "--ssh", "root@h"})
	if !ok || c.Name != "hosts add" || !slices.Equal(rest, []string{"box-a", "--ssh", "root@h"}) {
		t.Fatalf("%v %q %v", c, rest, ok)
	}
	if c, rest, _ := lookup([]string{"jobs", "box-a"}); c.Name != "jobs" || rest[0] != "box-a" {
		t.Fatalf("jobs HOST: %v", c)
	}
	if c, _, _ := lookup([]string{"jobs", "show", "0123456789abcdef"}); c.Name != "jobs show" {
		t.Fatalf("jobs show: %v", c)
	}
	if _, _, ok := lookup([]string{"nope"}); ok {
		t.Fatal("nope found")
	}
}

// Every visible command has a usage that starts with "jumpgate "+Name, a
// summary, and names each of its flags in the usage line.
func TestEveryCommandIsDescribed(t *testing.T) {
	for _, c := range commands {
		if c.Run == nil || c.Summary == "" || !strings.HasPrefix(c.Usage, "jumpgate "+c.Name) {
			t.Errorf("%q: %+v", c.Name, c)
		}
		c.flagSet().VisitAll(func(f *flag.Flag) {
			if f.Usage == "" || (!c.Hidden && !strings.Contains(c.Usage, "-"+f.Name)) {
				t.Errorf("%s: flag -%s is not in the usage %q", c.Name, f.Name, c.Usage)
			}
		})
	}
}

// A bare group word prints the group's commands and exits 2.
func TestBareGroupPrintsItsCommands(t *testing.T) {
	var b strings.Builder
	code, handled := dispatch([]string{"jumpgate", "hosts"}, &b)
	if !handled || code != 2 || !strings.Contains(b.String(), "jumpgate hosts add NAME") || !strings.Contains(b.String(), "jumpgate hosts list") {
		t.Fatalf("%d %v\n%s", code, handled, b.String())
	}
}
```

```go
// help_test.go: TestHelpListsEveryCommand iterates commands (not subcommands) and skips Hidden.
// New:
func TestHelpFitsEightyColumns(t *testing.T) {
	var b strings.Builder
	printHelp(&b)
	for i, l := range strings.Split(b.String(), "\n") {
		if utf8.RuneCountInString(l) > 80 {
			t.Errorf("line %d is %d columns: %q", i+1, utf8.RuneCountInString(l), l)
		}
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/jumpgate -run 'TestLookupTakesTheLongestName|TestEveryCommandIsDescribed|TestBareGroupPrintsItsCommands|TestHelpFitsEightyColumns|TestHelpListsEveryCommand' -v`
Expected: FAIL, `[build failed]`, `undefined: lookup` and `undefined: commands`

- [ ] **Step 3: Implement the table, `lookup`, `flagSet`, `usage` and the flag constructors**

`subcommands` becomes the set of first words of `commands`, and `dispatch` calls `lookup`. A first word with no leaf match prints the usages of every command in its group and returns 2. Delete `cmdHosts`'s and `cmdKeys`'s verb switches, because the table selects the leaf. `cmdJobs` keeps its token validation and moves to the `jobs`, `jobs show`, `jobs decide`, `jobs adopt` and `jobs output` rows. Each `usageText` constant is replaced by `c.usage()`. `printHelp` prints `Usage` without the `jumpgate ` prefix, wrapped at 78 columns with a 6-space continuation, then `Summary` on its own 6-space line. `agent` is `Hidden`. The unknown-command message keeps listing `subcommands`' keys, sorted.

- [ ] **Step 4: Run the package's tests**

Run: `go test ./cmd/jumpgate/`
Expected: `ok  github.com/valve-tech/jumpgate/cmd/jumpgate`. Every existing CLI test still passes: same exit codes, same messages except usage lines, which now come from the table.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/commands.go cmd/jumpgate/commands_test.go cmd/jumpgate/cli.go cmd/jumpgate/help.go cmd/jumpgate/help_test.go cmd/jumpgate/cli_hosts.go cmd/jumpgate/cli_keys.go cmd/jumpgate/cli_intents.go cmd/jumpgate/cli_relay.go cmd/jumpgate/buildserver.go cmd/jumpgate/cli_agent.go cmd/jumpgate/cli_jobs.go
git commit -m "refactor(cli): one command table drives dispatch, help and flags"
```

### Task 2: Exit-code and environment-variable tables

**Files:**
- Create: `internal/envvars/envvars.go`, `internal/envvars/envvars_test.go`
- Modify: `internal/api/errors.go` (after the `Exit*` constants), `internal/api/errors_test.go`, `cmd/jumpgate/buildserver.go:42-48` and `secretEnvNames`, `cmd/jumpgate/console_other.go:10`, `cmd/jumpgate/console_windows.go:29`, `cmd/jumpgate/home_terminal.go:91`, `internal/tui/theme.go:64`, `internal/agentbin/agentbin.go:40`

**Interfaces:**
- Produces:
  ```go
  // internal/api
  type ExitClass struct{ Code int; Name, Meaning string }
  func ExitClasses() []ExitClass // 0 ok, 1 failed, 2 usage, 3 unreachable, 4 security, in that order
  // internal/envvars
  type Var struct{ Name, Meaning string }
  const (ASCII = "JUMPGATE_ASCII"; Plain = "JUMPGATE_PLAIN"; Launcher = "JUMPGATE_LAUNCHER"
         RelayBind = "JUMPGATE_RELAY_BIND"; BillingSocket = "JUMPGATE_BILLING_SOCKET"; ERPCURL = "JUMPGATE_ERPC_URL"
         ERPCProject = "JUMPGATE_ERPC_PROJECT"; Meter = "JUMPGATE_METER"; RelayToken = "JUMPGATE_RELAY_TOKEN"
         RelayTokenFile = "JUMPGATE_RELAY_TOKEN_FILE"; AdminToken = "JUMPGATE_ADMIN_TOKEN"
         AdminTokenFile = "JUMPGATE_ADMIN_TOKEN_FILE"; DevAgents = "JUMPGATE_DEV_AGENTS")
  func All() []Var // sorted by Name
  ```

- [ ] **Step 1: Write the failing tests**

```go
// internal/api/errors_test.go
func TestExitClassesMatchTheConstants(t *testing.T) {
	want := []int{ExitOK, ExitFailed, ExitUsage, ExitUnreachable, ExitSecurity}
	got := ExitClasses()
	if len(got) != len(want) {
		t.Fatalf("%d classes", len(got))
	}
	for i, c := range got {
		if c.Code != want[i] || c.Name == "" || c.Meaning == "" {
			t.Errorf("%+v", c)
		}
	}
	for code, info := range registry { // every code's exit class is a listed one
		if !slices.ContainsFunc(got, func(c ExitClass) bool { return c.Code == info.exit }) {
			t.Errorf("%s exits %d, not a class", code, info.exit)
		}
	}
}
```

```go
// internal/envvars/envvars_test.go
// No package spells a JUMPGATE_ variable outside this table (tests excepted,
// and JUMPGATE_TEST_* which only tests read).
func TestNoVariableOutsideTheTable(t *testing.T) {
	known := map[string]bool{}
	for _, v := range All() {
		known[v.Name] = true
		if v.Meaning == "" {
			t.Errorf("%s has no meaning", v.Name)
		}
	}
	lit := regexp.MustCompile(`"(JUMPGATE_[A-Z_]+)"`)
	root := moduleRoot(t) // walks up to go.mod
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if d.IsDir() && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" || d.Name() == "testdata") {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") || strings.Contains(p, "internal/envvars/") {
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, m := range lit.FindAllStringSubmatch(string(b), -1) {
			if !strings.HasPrefix(m[1], "JUMPGATE_TEST_") {
				t.Errorf("%s spells %s; use envvars", p, m[1])
			}
		}
		return nil
	})
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/api ./internal/envvars -run 'TestExitClassesMatchTheConstants|TestNoVariableOutsideTheTable' -v`
Expected: FAIL. `undefined: ExitClasses`, and `no Go files in .../internal/envvars`.

- [ ] **Step 3: Implement `ExitClasses` and `envvars`, and switch every caller to the constants**

Take the `Meaning` strings from the README's exit-code paragraph and from each variable's current flag help (`buildserver.go`). For `JUMPGATE_ASCII`: "1 draws the TUI with ASCII glyphs only".

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/api ./internal/envvars ./cmd/jumpgate ./internal/tui ./internal/agentbin && go build -tags notui -o /dev/null ./cmd/jumpgate`
Expected: all `ok`, and the build succeeds.

- [ ] **Step 5: Commit**

```bash
git add internal/envvars/envvars.go internal/envvars/envvars_test.go internal/api/errors.go internal/api/errors_test.go cmd/jumpgate/buildserver.go cmd/jumpgate/console_other.go cmd/jumpgate/console_windows.go cmd/jumpgate/home_terminal.go internal/tui/theme.go internal/agentbin/agentbin.go
git commit -m "feat: exit-code and environment-variable tables"
```

### Task 3: Tables the docs can reach: CLI snapshot, key ids, network slugs

**Files:**
- Create: `cmd/jumpgate/reference_test.go`, `docs/reference/cli.json` (written by the test)
- Modify: `internal/tui/reference.go` (`KeyDoc.ID`, `keymaps` gets an `id`), `internal/tui/guide_test.go` (`TestKeymapHasEveryScreen`), `internal/catalog/catalog.go` (`Network.Slug`, `Client.Name`), `internal/catalog/networks.go`, `internal/catalog/catalog_test.go`

**Interfaces:**
- Produces:
  - `docs/reference/cli.json`: `[{"name","usage","summary","hidden","flags":[{"name","default","usage"}]}]` in `commands` order, flags sorted by name, two-space indent, trailing newline. Task 6's `docs.Command` and `docs.Flag` use these JSON tags.
  - `tui.KeyDoc{ID, Where, Label, Help string; Keys []string}`. `ID` is `<keymap id>.<lower-case field>`. The keymap ids are `global nav fleet hosts host svc log signer job gateway guide`, plus `settings.<n>` for Settings' extra keys, named by their help (`settings.toggle`, …).
  - `catalog.Network.Slug`: `eth` (1), `pls` (369), `pls-v4` (943). `catalog.Client.Name`: the display name, e.g. `Lighthouse`, `Lighthouse (PulseChain)`, `Reth`, `Geth`, `go-pulse`, `Erigon (PulseChain)`, `Prysm (PulseChain)`.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/jumpgate/reference_test.go
func init() { // go test -update, unless a linked package already defines it
	if flag.Lookup("update") == nil {
		flag.Bool("update", false, "rewrite docs/reference/cli.json")
	}
}

// The docs read the command table from docs/reference/cli.json. It is the
// table itself, re-derived here; run with -update after changing a command.
func TestCLIReferenceIsCurrent(t *testing.T) {
	got := cliReferenceJSON(t) // []cliDoc from commands, json.MarshalIndent(…, "", "  ") + "\n"
	path := filepath.Join("..", "..", "docs", "reference", "cli.json")
	if flag.Lookup("update").Value.String() == "true" {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, got, 0o644)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(want, got) {
		t.Fatalf("docs/reference/cli.json is stale: run go test ./cmd/jumpgate -run TestCLIReferenceIsCurrent -update and commit it")
	}
}
```

```go
// internal/tui/guide_test.go
func TestKeymapIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range Keymap() {
		if !regexp.MustCompile(`^[a-z]+\.[a-z0-9-]+$`).MatchString(d.ID) || seen[d.ID] {
			t.Errorf("id %q", d.ID)
		}
		seen[d.ID] = true
	}
	if !seen["fleet.sort"] || !seen["guide.open"] {
		t.Fatal("fleet.sort or guide.open missing")
	}
}
```

```go
// internal/catalog/catalog_test.go
func TestSlugsAndNames(t *testing.T) {
	want := map[int]string{1: "eth", 369: "pls", 943: "pls-v4"}
	for _, n := range Networks() {
		if n.Slug != want[n.ChainID] {
			t.Errorf("%d: slug %q", n.ChainID, n.Slug)
		}
		for _, id := range append(slices.Clone(n.ExecClients), n.BeaconClients...) {
			if c, _ := ClientByID(id); c.Name == "" {
				t.Errorf("client %s has no name", id)
			}
		}
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/jumpgate ./internal/tui ./internal/catalog -run 'TestCLIReferenceIsCurrent|TestKeymapIDsAreUnique|TestSlugsAndNames' -v`
Expected: FAIL. `undefined: cliReferenceJSON`, `d.ID undefined`, `n.Slug undefined`.

- [ ] **Step 3: Implement `cliReferenceJSON` (test file only, D1), `KeyDoc.ID`, `Slug` and `Name`**

`cliReferenceJSON` visits `c.flagSet()` for each command and records `f.Name`, `f.DefValue` and `f.Usage`. In `Keymap`, take the field name with `reflect.Type.Field(i).Name`, so `keyList` gains a sibling `keyFields(km any) []struct{Field string; B key.Binding}`.

- [ ] **Step 4: Write the snapshot and run the tests**

Run: `go test ./cmd/jumpgate -run TestCLIReferenceIsCurrent -update && go test ./cmd/jumpgate ./internal/tui ./internal/catalog`
Expected: `docs/reference/cli.json` exists. All three packages are `ok`.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/reference_test.go docs/reference/cli.json internal/tui/reference.go internal/tui/guide_test.go internal/catalog/catalog.go internal/catalog/networks.go internal/catalog/catalog_test.go
git commit -m "feat(docs): CLI snapshot, key ids and network slugs for references"
```

---

## Phase 2: The `docs` package and the content

### Task 4: Loader: steps, pathways and sidecars with file:line

**Files:**
- Create: `docs/docs.go`, `docs/load.go`, `docs/load_test.go`, `docs/testdata/tree/` (fixture: `steps/a/one.md`, `steps/a/two.md`, `steps/b/three.md`, `pathways/first.yaml`, `pathways/second.yaml`, `images/diagram.yaml`)
- Modify: `go.mod`, `go.sum` (`go get go.yaml.in/yaml/v3@v3.0.4`)

**Interfaces:**
- Produces:
  ```go
  package docs
  type Audience string
  const (Newcomer Audience = "newcomer"; Operator Audience = "operator")
  type Step struct {
      ID, Title, Summary, Recap, Image string
      Requires []string
      Verified map[string]string
      Area     string         // the directory under steps/
      Body     string
      File     string         // "steps/boxes/pair-box.md"
      Line     map[string]int // frontmatter key -> 1-based line in File
      BodyLine int            // line in File where Body starts
  }
  type Pathway struct {
      ID, Title string
      Audience  Audience
      After, Steps []string
      File string
      Line map[string]int
  }
  type Image struct {
      Name, File, Alt string // File: "images/diagram.png", next to the sidecar
      Covers   []string
      Verified map[string]string
      Line     map[string]int
  }
  type Model struct {
      Steps    map[string]*Step
      Pathways map[string]*Pathway
      Images   map[string]*Image
  }
  type Problem struct{ File string; Line int; Msg string }
  func (p Problem) String() string // "docs/<File>:<Line>: <Msg>"
  func LoadFS(fsys fs.FS) (*Model, []Problem)
  ```

- [ ] **Step 1: Write the failing tests**

```go
func TestLoadFixture(t *testing.T) {
	m, probs := LoadFS(os.DirFS("testdata/tree"))
	if len(probs) != 0 {
		t.Fatal(probs)
	}
	s := m.Steps["two"]
	if s.Title == "" || s.Area != "a" || s.File != "steps/a/two.md" || !slices.Equal(s.Requires, []string{"one"}) {
		t.Fatalf("%+v", s)
	}
	if s.Line["verified"] == 0 || s.BodyLine <= s.Line["verified"] {
		t.Fatalf("lines %v body %d", s.Line, s.BodyLine)
	}
	if p := m.Pathways["second"]; p.Audience != Operator || !slices.Equal(p.After, []string{"first"}) {
		t.Fatalf("%+v", p)
	}
}

func TestLoadReportsFieldProblems(t *testing.T) {
	cases := map[string]string{ // file body -> substring of the problem
		"---\nid: x\ntitle: X\nrecap: r\n---\nb\n":                       "steps/z/x.md:1: summary is required",
		"---\nid: y\ntitle: Y\nsummary: s\nrecap: r\n---\nb\n":            "id y does not match the file name x",
		"---\nid: x\ntitle: X\nsummary: \"a\\nb\"\nrecap: r\n---\nb\n":    "summary must be one line",
		"---\nid: x\ntitle: X\nsummary: s\nrecap: r\nimage: {{shot tui f}}\n---\nb\n": `quote it: image: "{{shot tui f}}"`,
		"---\nid: x\ntitle: X\nsummary: s\nrecap: r\nverified:\n  a.b: 0123456789ab\n---\nb\n": "steps/z/x.md:6: verified must be one line",
		"---\nid: x\ntitle: X\nsummary: s\nrecap: r\nsurprise: 1\n---\nb\n": "steps/z/x.md:6: unknown field surprise",
		"id: x\n": "steps/z/x.md:1: no frontmatter",
	}
	for src, want := range cases {
		_, probs := LoadFS(fstest.MapFS{"steps/z/x.md": {Data: []byte(src)}})
		if !slices.ContainsFunc(probs, func(p Problem) bool { return strings.Contains(p.String(), want) }) {
			t.Errorf("%q: want %q in %v", src, want, probs)
		}
	}
}

// Review Focus 2: a CRLF checkout loads the same, with the same lines.
func TestLoadCRLF(t *testing.T) {
	lf, _ := os.ReadFile("testdata/tree/steps/a/two.md")
	crlf := bytes.ReplaceAll(lf, []byte("\n"), []byte("\r\n"))
	a, _ := LoadFS(fstest.MapFS{"steps/a/two.md": {Data: lf}})
	b, probs := LoadFS(fstest.MapFS{"steps/a/two.md": {Data: crlf}})
	if len(probs) != 0 || !reflect.DeepEqual(a.Steps["two"], b.Steps["two"]) {
		t.Fatalf("%v\n%+v\n%+v", probs, a.Steps["two"], b.Steps["two"])
	}
}
```

Recap length rule: a `recap` of more than 3 lines is a problem (`recap is 2–3 lines`). One test case in the table covers it.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./docs -run 'TestLoad' -v`
Expected: FAIL, `[build failed]`, `undefined: LoadFS`

- [ ] **Step 3: Implement `LoadFS`**

Turn CRLF into LF first. Split the file at the leading `---\n … \n---\n`. Decode the frontmatter into a `yaml.Node`, and read each key's line from `Node.Line` (plus 1 for the opening `---`). `verified` must be a `FlowStyle` mapping on a single line. Pathways are `pathways/*.yaml`, and their `id` must equal the file name. Sidecars are `images/*.yaml`, with `File` set to the sibling `.png` or `.svg`. A sidecar without that file is a problem. Problems come out sorted by file and then line.

- [ ] **Step 4: Run the tests**

Run: `go test ./docs -run 'TestLoad' -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum docs/docs.go docs/load.go docs/load_test.go docs/testdata/tree
git commit -m "feat(docs): load steps, pathways and image sidecars with file:line"
```

### Task 5: Pathway graph, order, canonical and recap

**Files:**
- Create: `docs/graph.go`, `docs/graph_test.go`, `docs/testdata/graph/` (fixtures: `diamond/`, `cycle/`, `orphan/`, `outoforder/`, `cross/`)

**Interfaces:**
- Consumes: Task 4's `Model`, `Step`, `Pathway`, `Problem`, `LoadFS`.
- Produces:
  ```go
  func (m *Model) CheckGraph() []Problem
  func (m *Model) Order() []*Pathway                 // D6
  func (m *Model) Canonical(stepID string) *Pathway  // first pathway in Order containing it
  func (p *Pathway) Neighbours(stepID string) (prev, next string)
  type Section struct{ Name string; Pathways []*Pathway } // "Get started" (newcomer), "Operate" (operator)
  func (m *Model) Sections() []Section
  type RecapGroup struct{ Pathway *Pathway; Open bool; Steps []*Step }
  func (m *Model) Recap(pathwayID, stepID string) []RecapGroup // D7
  var Reserved = []string{"reference", "img", "pagefind", "assets", "search"} // ids no pathway may take
  ```

- [ ] **Step 1: Write the failing tests**

```go
func load(t *testing.T, dir string) *Model {
	t.Helper()
	m, probs := LoadFS(os.DirFS(filepath.Join("testdata", dir)))
	if len(probs) != 0 {
		t.Fatal(probs)
	}
	return m
}

func TestGraphProblems(t *testing.T) {
	for dir, want := range map[string]string{
		"graph/cycle":      "pathways/p.yaml:1: requires cycle: a -> b -> a",
		"graph/orphan":     "steps/x/lost.md:1: lost is in no pathway",
		"graph/outoforder": "steps/x/b.md:5: b requires c, which is neither earlier in p nor in a pathway p comes after",
		"graph/aftercycle": "pathways/p.yaml:4: after cycle: p -> q -> p",
		"graph/unknown":    "pathways/p.yaml:5: unknown step nope",
	} {
		m, _ := LoadFS(os.DirFS(filepath.Join("testdata", dir)))
		if !slices.ContainsFunc(m.CheckGraph(), func(p Problem) bool { return strings.Contains(p.String(), want) }) {
			t.Errorf("%s: want %q in %v", dir, want, m.CheckGraph())
		}
	}
}

// Review Focus 5.
func TestReservedIDs(t *testing.T) {
	m, _ := LoadFS(os.DirFS("testdata/graph/reserved")) // pathway id "reference", step "p" in pathway "p"
	probs := fmt.Sprint(m.CheckGraph())
	if !strings.Contains(probs, "pathways/reference.yaml:1: reference is a reserved id") ||
		!strings.Contains(probs, "steps/x/p.md:1: step p has the id of its pathway") {
		t.Fatal(probs)
	}
}

// cross: install [desktop-app, verify]; first-box (after install) [box-ready, keys-init, pair-box];
// pair-box requires keys-init, box-ready and verify; keys-init requires desktop-app; box-ready requires desktop-app.
func TestRecapCrossesPathwaysOnce(t *testing.T) {
	m := load(t, "graph/cross")
	got := m.Recap("first-box", "pair-box")
	if len(got) != 2 || got[0].Pathway.ID != "install" || got[0].Open || got[1].Pathway.ID != "first-box" || !got[1].Open {
		t.Fatalf("groups %+v", got)
	}
	ids := func(g RecapGroup) (s []string) {
		for _, st := range g.Steps {
			s = append(s, st.ID)
		}
		return
	}
	if !slices.Equal(ids(got[0]), []string{"desktop-app", "verify"}) || !slices.Equal(ids(got[1]), []string{"box-ready", "keys-init"}) {
		t.Fatalf("%v %v", ids(got[0]), ids(got[1])) // desktop-app once, though two steps need it
	}
	if m.Canonical("verify").ID != "install" {
		t.Fatal("canonical")
	}
	if prev, next := m.Pathways["first-box"].Neighbours("keys-init"); prev != "box-ready" || next != "pair-box" {
		t.Fatal(prev, next)
	}
	if s := m.Sections(); s[0].Name != "Get started" || s[1].Name != "Operate" {
		t.Fatal(s)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./docs -run 'TestGraphProblems|TestReservedIDs|TestRecapCrossesPathwaysOnce' -v`
Expected: FAIL, `[build failed]`, `m.CheckGraph undefined`

- [ ] **Step 3: Implement the graph**

Detect cycles with a three-colour DFS, reporting the path in `a -> b -> a` form at the first file in the cycle. `Order` is Kahn's algorithm over `after` with a sorted ready set. `Recap` computes the transitive closure of `requires` from the step, ordered topologically. Ties are broken by (rank of the item's group pathway in `Order`, index in that pathway). Groups follow D7.

- [ ] **Step 4: Run the tests**

Run: `go test ./docs -v`
Expected: PASS (every test in `docs`)

- [ ] **Step 5: Commit**

```bash
git add docs/graph.go docs/graph_test.go docs/testdata/graph
git commit -m "feat(docs): pathway graph checks, order, canonical and recap"
```

### Task 6: Step dialect and typed references

**Files:**
- Create: `docs/markdown.go`, `docs/markdown_test.go`, `docs/refs.go`, `docs/refs_test.go`

**Interfaces:**
- Consumes: Task 4's `Model`, `Step`, `Image`, `Problem`; Task 3's `cli.json` shape.
- Produces:
  ```go
  type BlockKind int
  const (Heading BlockKind = iota; Paragraph; Bullets; Numbered; Code)
  type Block struct{ Kind BlockKind; Level int; Text []Inline; Items [][]Inline; Code string; Line int }
  type InlineKind int
  const (Text InlineKind = iota; CodeSpan; Emph; Strong; Link; RefSpan)
  type Inline struct{ Kind InlineKind; Text, URL string; Ref *Ref }
  func Parse(src, file string, firstLine int) ([]Block, []Problem)

  type Ref struct{ Kind string; Args []string; File string; Line int } // Kind: cmd flag key palette network client exit env step shot image
  func ParseRef(inner, file string, line int) (Ref, error)

  type Command struct{ Name string `json:"name"`; Usage string `json:"usage"`; Summary string `json:"summary"`; Hidden bool `json:"hidden"`; Flags []Flag `json:"flags"` }
  type Flag struct{ Name string `json:"name"`; Default string `json:"default"`; Usage string `json:"usage"` }
  type Key struct{ ID, Where, Label, Help string }
  type Named struct{ ID, Name string }
  type Exit struct{ Code int; Name, Meaning string }
  type EnvVar struct{ Name, Meaning string }
  type Scene struct{ Kind, Name, Caption string } // Kind: tui app desktop
  type Tables struct {
      Commands []Command; Keys []Key; Palette []string; Networks, Clients []Named
      Exits []Exit; Env []EnvVar; Scenes []Scene
  }
  type Resolved struct {
      Ref    Ref
      Text   string // inline form: "jumpgate hosts add", "--jump", "s", ":logs NAME", "PulseChain", "4", "JUMPGATE_ASCII", the step's title
      Detail string // usage, flag help, key help, meaning; "" for step, shot, image
      StepID string // step refs
      Scene  *Scene // shot refs
      Image  *Image // image refs
  }
  func (m *Model) Resolve(r Ref, t Tables) (Resolved, error)
  func (m *Model) Blocks(s *Step) []Block // parsed once, cached
  func (m *Model) CheckRefs(t Tables) []Problem // dialect and references in every body, and every step's image
  ```

- [ ] **Step 1: Write the failing tests**

```go
// markdown_test.go
func TestParseDialect(t *testing.T) {
	bs, probs := Parse("# T\n\nA *b* **c** `d` [e](f) {{step x}}.\n\n- one\n- two\n\n1. n\n\n```\ncode {{not}}\n```\n", "steps/a/x.md", 7)
	if len(probs) != 0 || len(bs) != 5 || bs[0].Kind != Heading || bs[0].Line != 7 || bs[4].Code != "code {{not}}" {
		t.Fatalf("%v %+v", probs, bs)
	}
	kinds := []InlineKind{}
	for _, in := range bs[1].Text {
		kinds = append(kinds, in.Kind)
	}
	if !slices.Contains(kinds, RefSpan) || !slices.Contains(kinds, Link) || !slices.Contains(kinds, Strong) {
		t.Fatal(kinds)
	}
}

func TestParseRejectsOutsideTheDialect(t *testing.T) {
	for src, want := range map[string]string{
		"a <b>c</b>\n":     "steps/a/x.md:3: raw HTML",
		"| a | b |\n":      "steps/a/x.md:3: tables are not in the step dialect",
		"![alt](x.png)\n":  "steps/a/x.md:3: use {{image NAME}} or {{shot …}}",
		"> quote\n":        "steps/a/x.md:3: block quotes are not in the step dialect",
		"{{cmd \"hosts\n": "steps/a/x.md:3: unclosed {{",
	} {
		if _, probs := Parse(src, "steps/a/x.md", 3); !strings.Contains(fmt.Sprint(probs), want) {
			t.Errorf("%q: want %q, got %v", src, want, probs)
		}
	}
}

// Review Focus 1.
func TestRefsIgnoredInCode(t *testing.T) {
	bs, probs := Parse("Use `{{.Name}}` here.\n\n```\nhelm --set x={{ .Values.y }}\n```\n", "steps/a/x.md", 1)
	if len(probs) != 0 {
		t.Fatal(probs)
	}
	for _, in := range bs[0].Text {
		if in.Kind == RefSpan {
			t.Fatal("a reference inside backticks")
		}
	}
}
```

```go
// refs_test.go: fixture tables
var tables = Tables{
	Commands: []Command{{Name: "hosts add", Usage: "jumpgate hosts add NAME …", Summary: "pair a box", Flags: []Flag{{Name: "jump", Usage: "jump host user@host[:port]"}}}},
	Keys:     []Key{{ID: "fleet.sort", Where: "Fleet", Label: "s", Help: "sort: host, sync, disk, lag"}},
	Palette:  []string{"logs NAME", "help [PAGE]"},
	Networks: []Named{{ID: "pls", Name: "PulseChain"}}, Clients: []Named{{ID: "lighthouse", Name: "Lighthouse"}},
	Exits:    []Exit{{Code: 4, Name: "security", Meaning: "a host key or receipt did not verify"}},
	Env:      []EnvVar{{Name: "JUMPGATE_ASCII", Meaning: "1 draws ASCII only"}},
	Scenes:   []Scene{{Kind: "tui", Name: "fleet", Caption: "The Fleet screen"}},
}

func TestResolveEveryKind(t *testing.T) {
	m := load(t, "tree")
	for inner, want := range map[string]string{
		`cmd "hosts add"`: "jumpgate hosts add", `flag "hosts add" "--jump"`: "--jump", "key fleet.sort": "s",
		`palette "logs"`: ":logs NAME", "network pls": "PulseChain", "client lighthouse": "Lighthouse",
		"exit 4": "4", "env JUMPGATE_ASCII": "JUMPGATE_ASCII", "step one": m.Steps["one"].Title,
		"shot tui fleet": "The Fleet screen", "image diagram": m.Images["diagram"].Alt,
	} {
		r, err := ParseRef(inner, "steps/a/x.md", 9)
		if err != nil {
			t.Fatal(err)
		}
		got, err := m.Resolve(r, tables)
		if err != nil || got.Text != want {
			t.Errorf("%s: %q %v", inner, got.Text, err)
		}
	}
}

func TestUnknownRefsNameFileAndLine(t *testing.T) {
	m, _ := LoadFS(fstest.MapFS{"steps/a/x.md": {Data: []byte("---\nid: x\ntitle: X\nsummary: s\nrecap: r\nimage: \"{{shot app nope}}\"\n---\nRun {{cmd \"hosts frob\"}}.\n\nPress {{key fleet.nope}} and {{flag \"hosts add\" \"--nope\"}}.\n")}})
	got := fmt.Sprint(m.CheckRefs(tables))
	for _, want := range []string{
		`steps/a/x.md:8: unknown command "hosts frob"`, `steps/a/x.md:10: unknown key fleet.nope`,
		`steps/a/x.md:10: hosts add has no flag --nope`, `steps/a/x.md:6: unknown scene app nope`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("want %q in %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./docs -run 'TestParse|TestRefs|TestResolve|TestUnknownRefs' -v`
Expected: FAIL, `[build failed]`, `undefined: Parse`

- [ ] **Step 3: Implement `Parse`, `ParseRef`, `Resolve` and `CheckRefs`**

The block rules match `internal/tui/markdown.go`: a `#`–`######` heading; a paragraph of joined lines; `- `/`* ` and `1. ` lists, where indented continuation lines join the item; and ```` ``` ```` fences, whose content is verbatim. Inline scanning goes left to right in this order: code spans (opaque), then `{{…}}`, then `[text](url)`, then `**`, then `*`/`_` only at word boundaries (snake_case stays text, as `TestRenderMarkdown` requires). `ParseRef` splits `inner` into words, with double-quoted arguments. `{{flag "hosts add" "--jump"}}` matches `Flag.Name` with the leading dashes removed. `{{palette "logs"}}` matches the form whose first word is `logs` and renders `:` plus the form. `{{exit N}}` takes an integer.

- [ ] **Step 4: Run the tests**

Run: `go test ./docs -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docs/markdown.go docs/markdown_test.go docs/refs.go docs/refs_test.go
git commit -m "feat(docs): the step dialect and typed references"
```

### Task 7: `docs:covers` hashing and the verified check

**Files:**
- Create: `docs/covers/covers.go`, `docs/covers/covers_test.go`, `docs/covers/testdata/mod/` (a tiny module: `go.mod`, `a/a.go` with one marked func and one marked method, `.hidden/a.go` and `web/node_modules/x/x.go` that must be skipped), `docs/verified.go`, `docs/verified_test.go`

**Interfaces:**
- Consumes: Task 4's `Model`, `Step`, `Image`, `Problem`.
- Produces:
  ```go
  // package docs
  type Covered struct{ ID, Step, File string; Line int; Hash string } // one row per (item, step)
  func (m *Model) CheckVerified(items []Covered, hash func(id string) (string, error)) []Problem
  // package covers (github.com/valve-tech/jumpgate/docs/covers)
  func Scan(root string) ([]docs.Covered, []docs.Problem) // every func/method with "// docs:covers <id>" in its doc comment
  func Hash(root, id string) (string, error)             // a Go declaration id (D5) or a repo file path (D4)
  func ModuleRoot(dir string) (string, error)            // walks up to go.mod
  ```

- [ ] **Step 1: Write the failing tests**

```go
// covers_test.go
func TestScanFindsMarkedDeclarations(t *testing.T) {
	items, probs := Scan("testdata/mod")
	if len(probs) != 0 || len(items) != 2 {
		t.Fatalf("%v %+v", probs, items) // .hidden and node_modules are skipped
	}
	if items[0].ID != "a.Pair" || items[0].Step != "pair-box" || items[0].File != "a/a.go" || items[0].Line == 0 || len(items[0].Hash) != 12 {
		t.Fatalf("%+v", items[0])
	}
	if items[1].ID != "a.Box.Start" {
		t.Fatalf("method id %q", items[1].ID)
	}
}

// Review Focus 2: comments, CRLF and blank lines do not change a hash; code does.
func TestHashIgnoresCRLFAndComments(t *testing.T) {
	base := "package a\n\n// docs:covers pair-box\nfunc Pair() int {\n\t// why\n\treturn 1\n}\n"
	same := []string{
		strings.ReplaceAll(base, "\n", "\r\n"),
		strings.Replace(base, "// why", "// a different reason", 1),
		strings.Replace(base, "return 1", "return 1 // inline", 1),
		strings.Replace(base, "func Pair() int {", "func Pair()   int {", 1),
	}
	h := hashSrc(t, base, "a.Pair")
	for _, s := range same {
		if got := hashSrc(t, s, "a.Pair"); got != h {
			t.Errorf("hash changed: %q", s)
		}
	}
	if hashSrc(t, strings.Replace(base, "return 1", "return 2", 1), "a.Pair") == h {
		t.Fatal("a code change kept the hash")
	}
}

func TestHashOfAFilePath(t *testing.T) {
	h, err := Hash("testdata/mod", "a/a.go")
	if err != nil || len(h) != 12 {
		t.Fatal(h, err)
	}
}
```

```go
// verified_test.go: the error text is the spec's (§4 "Errors")
func TestStaleHashPrintsTheLineToChange(t *testing.T) {
	m, _ := LoadFS(fstest.MapFS{"steps/b/pair-box.md": {Data: []byte(
		"---\nid: pair-box\ntitle: Pair a box\nsummary: s\nrecap: |\n  one\n  two\nverified: { a.Pair: 000000000000, a.Box.Start: 111111111111 }\n---\nbody\n")}})
	items := []Covered{{ID: "a.Pair", Step: "pair-box", File: "a/a.go", Line: 4, Hash: "3f9a0c1d2e4b"}, {ID: "a.Box.Start", Step: "pair-box", File: "a/a.go", Line: 9, Hash: "111111111111"}}
	got := fmt.Sprint(m.CheckVerified(items, nil))
	want := "steps/b/pair-box.md:8: a.Pair changed (verified 000000000000, now 3f9a0c1d2e4b; a/a.go:4). Re-read the step, then change line 8 to:\n    verified: { a.Box.Start: 111111111111, a.Pair: 3f9a0c1d2e4b }"
	if !strings.Contains(got, want) {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestMissingAndStrayVerified(t *testing.T) {
	m, _ := LoadFS(fstest.MapFS{
		"steps/b/x.md": {Data: []byte("---\nid: x\ntitle: X\nsummary: s\nrecap: r\n---\nbody\n")},
		"steps/b/y.md": {Data: []byte("---\nid: y\ntitle: Y\nsummary: s\nrecap: r\nverified: { a.Gone: 123456789abc }\n---\nbody\n")},
	})
	got := fmt.Sprint(m.CheckVerified([]Covered{{ID: "a.Pair", Step: "x", File: "a/a.go", Line: 4, Hash: "3f9a0c1d2e4b"}, {ID: "a.Pair", Step: "nope", File: "a/a.go", Line: 4}}, nil))
	for _, w := range []string{
		"steps/b/x.md:1: a.Pair covers x (a/a.go:4) but x has no verified; add:\n    verified: { a.Pair: 3f9a0c1d2e4b }",
		"steps/b/y.md:6: a.Gone no longer covers y; remove it",
		"a/a.go:4: docs:covers names unknown step nope",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("want %q in\n%s", w, got)
		}
	}
}

func TestImageSidecarHashes(t *testing.T) {
	m := load(t, "tree") // images/diagram.yaml: covers [a.Pair], verified { a.Pair: 000000000000 }
	got := fmt.Sprint(m.CheckVerified(nil, func(id string) (string, error) { return "3f9a0c1d2e4b", nil }))
	if !strings.Contains(got, "images/diagram.yaml:3: a.Pair changed (verified 000000000000, now 3f9a0c1d2e4b)") {
		t.Fatal(got)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./docs/... -run 'TestScan|TestHash|TestStale|TestMissing|TestImageSidecar' -v`
Expected: FAIL, `[build failed]`, `undefined: Scan` and `m.CheckVerified undefined`

- [ ] **Step 3: Implement `Scan`, `Hash`, `ModuleRoot` and `CheckVerified`**

The hash algorithm, which the tests do not fully determine:

```text
parse the file with go/parser.ParseComments (CRLF -> LF first)
find the *ast.FuncDecl by name (and receiver type name for methods)
copy it; set Doc = nil
print the copy alone with go/printer (no CommentedNode, so body comments drop)
go/format.Source on "package p\n" + printed; take the bytes after the package line
hex(sha256(bytes))[:12]
```

`Scan` walks `root` and skips directories that start with `.` (this repo has `.claude/worktrees/` copies of itself), `node_modules`, `testdata` and `vendor`. It also skips `_test.go` files. The id's directory is the file's path relative to `root`. The marker line is `// docs:covers <id>` inside the doc comment. Several markers mean several rows. In the problem message, the "change line N to" text lists the whole map with keys sorted (D3).

- [ ] **Step 4: Run the tests**

Run: `go test ./docs/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docs/covers docs/verified.go docs/verified_test.go
git commit -m "feat(docs): docs:covers hashing and the verified check"
```

### Task 8: Live tables, the embedded model and the first pathway

**Files:**
- Create: `docs/embed.go`, `docs/check_test.go` (package `docs_test`), `internal/docstables/docstables.go`, `docs/shots/app.json` (`[]`), `docs/shots/desktop.json` (`[]`), `docs/pathways/install.yaml`, `docs/steps/install/*.md` (six steps, see the Task 9 table)
- Modify: `internal/tui/reference.go` (add `DocTables`)

**Interfaces:**
- Consumes: Tasks 2–7.
- Produces:
  ```go
  // docs
  func Embedded() *Model   // //go:embed steps pathways reference shots; loads once; panics on a load problem (the tests catch it)
  func CLI() []Command     // reference/cli.json
  func Shots() []Scene     // shots/app.json and shots/desktop.json, Kind "app" / "desktop"
  // internal/tui
  func DocTables() docs.Tables // CLI(), Keymap() as Key, Commands(), catalog (Slug/Name, client Name), api.ExitClasses(), envvars.All(); Scenes: docs.Shots()
  // internal/docstables
  func Live() docs.Tables // tui.DocTables() plus TUI scenes (Task 17 adds them)
  ```

- [ ] **Step 1: Write the failing test**

```go
// docs/check_test.go
package docs_test

// The repo's docs pass every check against the live code. This is the merge gate (spec §4).
func TestDocsCheck(t *testing.T) {
	root, err := covers.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	m, probs := docs.LoadFS(os.DirFS("."))
	probs = append(probs, m.CheckGraph()...)
	probs = append(probs, m.CheckRefs(docstables.Live())...)
	items, p := covers.Scan(root)
	probs = append(append(probs, p...), m.CheckVerified(items, func(id string) (string, error) { return covers.Hash(root, id) })...)
	for _, p := range probs {
		t.Error(p)
	}
	if e := docs.Embedded(); len(e.Steps) != len(m.Steps) || len(e.Pathways) != len(m.Pathways) {
		t.Fatalf("embedded %d/%d, tree %d/%d", len(e.Steps), len(e.Pathways), len(m.Steps), len(m.Pathways))
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./docs -run TestDocsCheck -v`
Expected: FAIL, `[build failed]`, `undefined: docs.Embedded` and `no required module provides package .../internal/docstables`

- [ ] **Step 3: Implement `embed.go`, `DocTables` and `Live`, and write the `install` pathway**

`install.yaml`: `id: install`, `title: Install`, `audience: newcomer`, `after: []`, and its six steps in the Task 9 table's order. Take each body from post-merge `docs/guide/install.md`, one `##` section per step. Take `summary` and `recap` from the landing frontmatter (`git -C ~/Documents/jumpgate-tech/github/landing show main:docs/src/01-install.md`). Leave `image` out until Phase 5. Rewrite command mentions as `{{cmd …}}`.

- [ ] **Step 4: Run the test**

Run: `go test ./docs/... ./internal/tui ./internal/docstables`
Expected: all `ok`

- [ ] **Step 5: Commit**

```bash
git add docs/embed.go docs/check_test.go internal/docstables internal/tui/reference.go docs/shots docs/pathways/install.yaml docs/steps/install
git commit -m "feat(docs): live tables, the embedded model and the Install pathway"
```

### Task 9: Migrate the guide and the landing pages into steps and pathways

**Files:**
- Create: `docs/pathways/{first-box,one-endpoint,durable-jobs,migrate-legacy-names,recover-a-box}.yaml`, `docs/steps/<area>/<id>.md` for every row below
- Modify: `docs/check_test.go` (add `TestEveryCommandHasAStep`); add `// docs:covers` markers to `cmd/jumpgate/cli_keys.go` `keysInit` (keys-init), `cmd/jumpgate/cli_hosts.go` `hostsAdd` and `confirmHostKeys` (pair-box), `hostsForgetKey` (forget-host-key), `cmd/jumpgate/cli_jobs.go` `cmdMigrateNames` (migrate-names)

**Interfaces:**
- Consumes: Task 8's check. The bodies are the post-merge `docs/guide/*.md` sections. `summary`, `recap` and `requires` come from the landing frontmatter (`main:docs/src/0N-*.md`). The landing's `section/slug` requires map to the new ids in this table.
- Produces: the step and pathway ids below. Task 15's redirects use the "Old landing URL" column.

| Pathway (audience, after) | Step id (area) | From (guide section / old landing URL under `/docs/`) |
|---|---|---|
| install (newcomer, —) | desktop-app (install) | install "Desktop app" / `install/desktop-app/` |
| | terminal-only (install) | "Terminal only" / `install/terminal-only-tui-and-cli/` |
| | unsigned-downloads (install) | "The downloads are not code-signed yet" + security "Unsigned downloads" / `install/the-downloads-are-not-code-signed-yet/`, `security/unsigned-downloads/` |
| | verify-a-download (install) | `install/verify-a-download/` |
| | minimum-versions (install) | `install/minimum-versions/` |
| | build-from-source (install) | `install/build-from-source/` |
| first-box (newcomer, install) | box-ready (boxes) | `getting-started/what-a-box-needs/` |
| | keys-init (boxes) | `getting-started/1-create-the-controller-key/` + security "The controller key" / `security/the-controller-key/` |
| | pair-box (boxes) | `getting-started/2-pair-a-box/` |
| | host-keys (security) | `security/pairing-and-host-keys/` |
| | close-root-ssh (boxes) | `getting-started/3-turn-off-root-ssh-login/` |
| | watch-it (boxes) | `getting-started/4-watch-it/` |
| | one-server (boxes) | `getting-started/one-server-for-everything/` |
| | local-server (security) | `security/the-local-server/` |
| | signed-intents (security) | `security/signed-intents-and-receipts/` |
| | tui-screens (tui) | `tui/screens/` (Keys and Commands lists become `{{key}}`/`{{palette}}` refs plus links to the reference pages) |
| | the-guide (tui) | `tui/the-guide/` (rewritten for steps, pathways and `r`) |
| | tui-terminals (tui) | `tui/terminals/` |
| | desktop-node (boxes) | `getting-started/a-node-on-your-desktop/` |
| one-endpoint (newcomer, first-box) | open-web-app (rpc) | commands "Opening the web app" / `commands/opening-the-web-app/` |
| | endpoints (rpc) | new: `{{cmd "endpoints"}}`, the box's RPC endpoints |
| | relay (rpc) | `commands/serving-metered-rpc-jumpgate-relay/` |
| | relay-tokens (security) | `security/tokens-for-the-relay/` |
| durable-jobs (operator, first-box) | how-a-job-behaves (jobs) | `jobs/how-a-job-behaves/` |
| | who-drives (jobs) | `jobs/who-drives/` |
| | job-commands (jobs) | `jobs/commands/` |
| | stuck-job (jobs) | `jobs/when-a-job-is-stuck/` |
| migrate-legacy-names (operator, durable-jobs) | upgrade-controller (upgrading) | `upgrading/the-controller/` |
| | upgrade-agent (upgrading) | `upgrading/the-agent-on-a-box/` |
| | legacy-names (upgrading) | `upgrading/moving-off-the-legacy-names/` |
| | migrate-names (jobs) | `jobs/node-migrate-names/` |
| recover-a-box (operator, first-box) | on-the-box (security) | `security/on-the-box/` |
| | forget-host-key (security) | new: from host-keys, `{{cmd "hosts forget-key"}}`, `{{exit 4}}` |
| | re-pair (boxes) | new: pair again after a rebuild, `{{step pair-box}}` |

`tui/keys/`, `tui/commands/`, `commands/` and `commands/exit-codes/` become reference pages (Task 10). Their redirects are in Task 15.

- [ ] **Step 1: Write the failing test**

```go
// D8: every visible command is taught by some step.
func TestEveryCommandHasAStep(t *testing.T) {
	m := docs.Embedded()
	named := map[string]bool{}
	for _, s := range m.Steps {
		walkRefs(m.Blocks(s), func(r docs.Ref) {
			if r.Kind == "cmd" {
				named[r.Args[0]] = true
			}
		})
	}
	for _, c := range docs.CLI() {
		if !c.Hidden && !named[c.Name] {
			t.Errorf("no step names {{cmd %q}}", c.Name)
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./docs -run 'TestEveryCommandHasAStep' -v`
Expected: FAIL, `no step names {{cmd "keys show"}}` (and the other commands that only the Install pathway lacks)

- [ ] **Step 3: Write the pathways and steps, add the markers, and fill `verified`**

Each step's body is its guide section, with commands, keys, palette forms, networks, exit codes and environment variables rewritten as references. For each new marker, run `go test ./docs -run TestDocsCheck`, re-read the step against the function, and paste the printed `verified:` line.

- [ ] **Step 4: Run the checks**

Run: `go test ./docs/... ./cmd/jumpgate ./internal/tui`
Expected: all `ok`. `TestDocsCheck` and `TestEveryCommandHasAStep` pass.

- [ ] **Step 5: Commit**

```bash
git add docs/pathways docs/steps docs/check_test.go cmd/jumpgate/cli_keys.go cmd/jumpgate/cli_hosts.go cmd/jumpgate/cli_jobs.go
git commit -m "docs: the guide and the landing pages as steps and pathways"
```

---

## Phase 3: TUI Help on the step model

### Task 10: Generated reference pages

**Files:**
- Create: `docs/reference.go`, `docs/reference_test.go`

**Interfaces:**
- Consumes: Task 6's `Tables`, `Block`, `Parse`.
- Produces:
  ```go
  type Page struct{ ID, Title string; Blocks []Block } // ids: commands keys palette networks exit-codes env
  func ReferencePages(t Tables) []Page                 // in that order; built as dialect markdown, then Parse
  ```

- [ ] **Step 1: Write the failing test**

```go
func TestReferencePagesListEveryRow(t *testing.T) {
	pages := ReferencePages(tables) // the refs_test fixture
	if ids := pageIDs(pages); !slices.Equal(ids, []string{"commands", "keys", "palette", "networks", "exit-codes", "env"}) {
		t.Fatal(ids)
	}
	text := plainText(pages) // concatenated Inline texts and code
	for _, want := range []string{"jumpgate hosts add NAME …", "--jump", "jump host user@host[:port]", "sort: host, sync, disk, lag",
		":logs NAME", "PulseChain", "Lighthouse", "security", "JUMPGATE_ASCII"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, c := range tables.Commands { // hidden commands stay out
		if c.Hidden && strings.Contains(text, c.Name) {
			t.Errorf("hidden %s listed", c.Name)
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./docs -run TestReferencePagesListEveryRow -v`
Expected: FAIL, `undefined: ReferencePages`

- [ ] **Step 3: Implement `ReferencePages`**

Each page is a heading per group (commands by first word, keys by `Where`). A row is a bullet: `` `usage` `` and then the summary. Flags are a nested list with their default. Tables are not in the dialect, so none are used.

- [ ] **Step 4: Run the tests**

Run: `go test ./docs -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docs/reference.go docs/reference_test.go
git commit -m "feat(docs): generated reference pages"
```

### Task 11: The Help view on steps and pathways

**Files:**
- Modify: `internal/tui/guide.go` (the model becomes sections → pathway → step, plus reference pages), `internal/tui/markdown.go` (`renderBlocks(bs []docs.Block, res func(docs.Ref) docs.Resolved, w int, th Theme, gl Glyphs) []string`; `renderMarkdown` becomes `renderBlocks(docs.Parse(src))`), `internal/tui/palette.go` (`help [PAGE]`, D13), `internal/tui/backend.go` (`Options.Scene`), `internal/tui/guide_test.go`, `internal/tui/ascii_test.go`, `internal/tui/testdata/guide_80x24.golden`, `cmd/jumpgate/tui_on.go`
- Create: `internal/tui/testdata/guide_pathway_80x24.golden`, `internal/tui/testdata/guide_step_80x24.golden`
- Delete: `internal/tui/testdata/guide_jobs_80x24.golden`

**Interfaces:**
- Consumes: `docs.Embedded()`, `Model.Sections()`, `Pathway.Neighbours`, `Model.Canonical`, `Model.Blocks`, `Model.Resolve`, `docs.ReferencePages`, `tui.DocTables()`.
- Produces: `var guideSource *docs.Model = docs.Embedded()` (replacing `guide.Guide`); `newGuide(page string) (*guideModal, bool)`, where page is a step, pathway or reference id; `Options.Scene func(name string, ascii bool) (frame string, ok bool)` (D10, nil until Task 18). A ref renders in the TUI as `Resolved.Text` in the Key style. `{{shot tui x}}` renders as `[frame: <caption> - v to view]`. App and desktop shots render as `[image: <caption> - jumpgate.app/docs/<pathway>/<step>/]` (spec §3).

- [ ] **Step 1: Write the failing tests**

```go
// ? then H lists Get started, Operate and Reference, each pathway's title, and every reference page.
func TestGuideListsSectionsAndPathways(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), "?", "H")
	fr := tuitest.Frame(m)
	for _, s := range guideSource.Sections() {
		if !strings.Contains(fr, s.Name) {
			t.Errorf("no %s", s.Name)
		}
		for _, p := range s.Pathways {
			if !strings.Contains(fr, p.Title) {
				t.Errorf("no %s", p.Title)
			}
		}
	}
	if !strings.Contains(fr, "Reference") {
		t.Error("no Reference")
	}
	tuitest.Golden(t, "guide_80x24", fr)
	m = press(t, m, "enter") // the first pathway: its numbered steps
	tuitest.Golden(t, "guide_pathway_80x24", tuitest.Frame(m))
	m = press(t, m, "enter")
	g := m.(*App).modal.(*guideModal)
	if g.step().ID != guideSource.Order()[0].Steps[0] {
		t.Fatalf("opened %s", g.step().ID)
	}
	tuitest.Golden(t, "guide_step_80x24", tuitest.Frame(m))
}

func TestHelpCommandOpensAPage(t *testing.T) {
	m := press(t, typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":"), "help pair-box"), "enter")
	g := m.(*App).modal.(*guideModal)
	if g.step().ID != "pair-box" || g.pathway().ID != guideSource.Canonical("pair-box").ID {
		t.Fatalf("%v", g)
	}
	m = press(t, m, "l")
	if _, next := g.pathway().Neighbours("pair-box"); g.step().ID != next {
		t.Fatal("l is not the next step")
	}
	m = press(t, typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":"), "help nope"), "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, `no docs page "nope"`) {
		t.Fatal(fr)
	}
}

// The v0.5.2 fit test, on every step and reference page, both glyph sets.
func TestEveryPageFits80x24(t *testing.T) { /* TestGuidePagesFit80x24's loop over pageIDs(): steps ∪ ReferencePages, via ":help ID" */ }
```

`expand` replaces `PAGE` with `pair-box`, so `TestEveryListedCommandRuns` covers `help pair-box`. Remove `guideTokens`, `TestGuideKeysAreBound` and `TestGuideCommandsRun`, because typed refs now enforce those (`TestDocsCheck`). In `ascii_test.go`, the "guide page" path becomes `"?", "H", "enter", "enter"`.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/tui -run 'TestGuideListsSectionsAndPathways|TestHelpCommandOpensAPage|TestEveryPageFits80x24' -v`
Expected: FAIL, `g.step undefined`

- [ ] **Step 3: Implement the navigation and `renderBlocks`, and update the goldens**

The list view is one scrolling list: section headings (Title style and the section name in capitals, so they read without colour), pathway rows with `:help <id>` dimmed, and then the reference pages. Enter on a pathway shows its steps as `1.`…`n.`. Enter on a step opens it. `←/→` move to the previous or next step in the pathway, and esc goes up one level. The page head reads `Guide · <pathway> · <n> of <len>`. Keep the cache key from v0.5.2 and add the pathway to it.

Run: `go test ./internal/tui -run 'TestGuideListsSectionsAndPathways' -update`, then review the three goldens by eye.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/tui ./cmd/jumpgate`
Expected: `ok` for both

- [ ] **Step 5: Commit**

```bash
git add internal/tui/guide.go internal/tui/markdown.go internal/tui/palette.go internal/tui/backend.go internal/tui/guide_test.go internal/tui/ascii_test.go internal/tui/testdata/guide_80x24.golden internal/tui/testdata/guide_pathway_80x24.golden internal/tui/testdata/guide_step_80x24.golden cmd/jumpgate/tui_on.go
git rm internal/tui/testdata/guide_jobs_80x24.golden
git commit -m "feat(tui): Help on steps and pathways"
```

### Task 12: The recap pane, and `docs/guide` removed

**Files:**
- Modify: `internal/tui/guide.go` (`guideKeys.Recap`, `guideKeys.View`, the recap pane), `internal/tui/guide_test.go`, `internal/tui/ascii_test.go`, `docs/steps/tui/the-guide.md` (names `{{key guide.recap}}`)
- Create: `internal/tui/testdata/guide_recap_80x24.golden`, `internal/tui/testdata/guide_recap_ascii.golden`
- Delete: `docs/guide/` (all 7 `.md`, `guide.go`, `guide_test.go`)

**Interfaces:**
- Consumes: `Model.Recap(pathwayID, stepID) []RecapGroup`.
- Produces: `guideKeys.Recap` (`r`, "show previous steps") and `guideKeys.View` (`v`, "view the frame"; used in Task 18).

- [ ] **Step 1: Write the failing tests**

```go
// r on a step lists Model.Recap in the same order: the current pathway's group open, the others collapsed.
func TestRecapPaneMatchesTheModel(t *testing.T) {
	m := press(t, typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":"), "help pair-box"), "enter", "r")
	fr := tuitest.Frame(m)
	groups := guideSource.Recap(guideSource.Canonical("pair-box").ID, "pair-box")
	last := -1
	for _, g := range groups {
		i := strings.Index(fr, g.Pathway.Title)
		if i < last {
			t.Fatalf("group %s out of order:\n%s", g.Pathway.ID, fr)
		}
		last = i
		for _, s := range g.Steps {
			if g.Open != strings.Contains(fr, s.Title) {
				t.Errorf("%s: open=%v but shown=%v", s.ID, g.Open, !g.Open)
			}
		}
	}
	tuitest.Golden(t, "guide_recap_80x24", fr)
	m = press(t, m, "enter") // on the first open item: its summary and recap appear
	if first := groups[len(groups)-1].Steps[0]; !strings.Contains(tuitest.Frame(m), strings.SplitN(first.Recap, "\n", 2)[0]) {
		t.Fatal("enter did not expand the recap")
	}
	if g := press(t, m, "r").(*App).modal.(*guideModal); g.recapOpen {
		t.Fatal("r did not close the pane")
	}
}

// Open and closed are marked by a glyph, not colour alone (NO_COLOR), and ascii stays ASCII.
func TestRecapPaneWithoutColour(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
	fr := tuitest.Frame(press(t, typed(t, press(t, a, ":"), "help pair-box"), "enter", "r"))
	if !strings.Contains(fr, "[-]") || !strings.Contains(fr, "[+]") {
		t.Fatal(fr)
	}
	requireASCII(t, fr)
	tuitest.Golden(t, "guide_recap_ascii", fr)
}
```

The step with no `requires` shows `No previous steps.` (one table case). `ascii_test.go` gains a `"guide recap"` entry.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/tui -run 'TestRecapPane' -v`
Expected: FAIL, `g.recapOpen undefined`

- [ ] **Step 3: Implement the pane and delete `docs/guide`**

The pane replaces the page body. Group rows read `▾ <pathway title>` or `▸ …` (ascii `[-]` / `[+]`). Item rows show the title, then the summary dimmed. Enter on a group toggles it. Enter on an item shows its recap lines indented, plus `[frame: …]` or `[image: …]` for its `image`. `↑/↓` move. The footer names `r back to the page`.

Run: `grep -rn 'docs/guide' --include='*.go' . | grep -v '^./.claude'`
Expected: no output

- [ ] **Step 4: Run every test**

Run: `go test ./... && go build -tags notui -o /dev/null ./cmd/jumpgate`
Expected: all `ok`, and the build succeeds

- [ ] **Step 5: Commit**

```bash
git add internal/tui/guide.go internal/tui/guide_test.go internal/tui/ascii_test.go internal/tui/testdata/guide_recap_80x24.golden internal/tui/testdata/guide_recap_ascii.golden docs/steps/tui/the-guide.md
git rm -r docs/guide
git commit -m "feat(tui): show previous steps; the step model replaces docs/guide"
```

---

## Phase 4: The website

### Task 13: `internal/docsite`: pages, sidebar, canonical and prev/next

**Files:**
- Create: `internal/docsite/site.go`, `internal/docsite/html.go` (blocks → HTML), `internal/docsite/templates/{layout,step,pathway,index,reference}.html`, `internal/docsite/assets/site.css`, `internal/docsite/site_test.go`, `internal/docsite/testdata/golden/` (written under `-update`)
- Modify: `.gitignore` (add `/out/`)

**Interfaces:**
- Consumes: `docs.Model`, `Order`, `Sections`, `Canonical`, `Neighbours`, `Blocks`, `Resolve`, `ReferencePages`, `docs.Tables`; Task 5's fixture `docs/testdata/graph/cross`.
- Produces:
  ```go
  type Options struct {
      Model  *docs.Model
      Tables docs.Tables
      Base   string            // "/docs"
      Shots  func(s docs.Scene) (src string, ok bool) // the image URL for a scene; nil: captions only
  }
  func Pages(o Options) (map[string][]byte, error) // "install/desktop-app/index.html" -> bytes, plus "assets/site.css"
  func Write(o Options, dir string) error          // Pages to disk
  ```
  The CSS tokens are copied verbatim from the landing `index.html` `:root`: `--bg:#0e0e0d; --bg-2:#151514; --panel:#1b1b19; --panel-2:#232320; --line:#33322e; --line-2:#4a4843; --ink:#f3f1ec; --ink-2:#c9c6bd; --ink-3:#a19d93; --accent:#2fd884; --accent-hi:#5fe6a2; --accent-ink:#03170b; --mono:"JetBrains Mono", ui-monospace, Menlo, Consolas, monospace; --sans:"Archivo", system-ui, sans-serif; --gut:clamp(16px, 4vw, 48px)`.

- [ ] **Step 1: Write the failing tests**

```go
func build(t *testing.T) map[string][]byte {
	m, _ := docs.LoadFS(os.DirFS("../../docs/testdata/graph/cross"))
	pages, err := Pages(Options{Model: m, Tables: fixtureTables, Base: "/docs"})
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

// A step in two pathways has a page in each, both canonical to the first.
func TestPagesAndCanonical(t *testing.T) {
	p := build(t)
	for _, path := range []string{"index.html", "install/index.html", "install/desktop-app/index.html", "first-box/pair-box/index.html", "reference/commands/index.html"} {
		if p[path] == nil {
			t.Errorf("no %s", path)
		}
	}
	if !bytes.Contains(p["first-box/verify/index.html"], []byte(`<link rel="canonical" href="https://jumpgate.app/docs/install/verify/">`)) {
		t.Error("canonical")
	}
	if !bytes.Contains(p["first-box/keys-init/index.html"], []byte(`href="/docs/first-box/box-ready/" rel="prev"`)) {
		t.Error("prev")
	}
	golden(t, p, "first-box/pair-box/index.html", "index.html") // -update writes testdata/golden/…
}

func TestSidebarSections(t *testing.T) {
	html := string(build(t)["install/index.html"])
	gs, op, ref := strings.Index(html, ">Get started<"), strings.Index(html, ">Operate<"), strings.Index(html, ">Reference<")
	if gs < 0 || ref < 0 || !(gs < ref) || (op >= 0 && !(gs < op && op < ref)) {
		t.Fatalf("%d %d %d", gs, op, ref)
	}
}

// Review Focus 3.
func TestEscapesProseAndFrontmatter(t *testing.T) {
	m, _ := docs.LoadFS(fstest.MapFS{
		"steps/a/x.md":    {Data: []byte("---\nid: x\ntitle: \"A <b> & \\\"c\\\"\"\nsummary: \"1 < 2 & 3\"\nrecap: r\n---\nIf a < b & c > d, `<tag>`.\n")},
		"pathways/p.yaml": {Data: []byte("id: p\ntitle: P\naudience: newcomer\nafter: []\nsteps: [x]\n")},
	})
	page := string(must(Pages(Options{Model: m, Base: "/docs"}))["p/x/index.html"])
	for _, bad := range []string{"<b>", "<tag>", "1 < 2"} {
		if strings.Contains(page, bad) {
			t.Errorf("unescaped %q", bad)
		}
	}
	for _, good := range []string{"<title>A &lt;b&gt; &amp; &#34;c&#34; · P · Jumpgate docs</title>", `content="1 &lt; 2 &amp; 3"`, "a &lt; b &amp; c &gt; d", "<code>&lt;tag&gt;</code>"} {
		if !strings.Contains(page, good) {
			t.Errorf("missing %q", good)
		}
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/docsite -v`
Expected: FAIL, `no Go files` / `undefined: Pages`

- [ ] **Step 3: Implement `Pages` and `Write`**

Use `html/template` throughout. Block HTML is built from `template.HTML` that comes only from `html.go`, which escapes every text node. Refs render as `<code>` for cmd, flag, key, palette, env and exit. A key gets `title="<help>"` (the tooltip). A step ref is a link. A shot or image ref is a `<figure>` with `loading="lazy"`. The layout is mobile-first: the sidebar collapses under 760px into a `<details>`, and `pre` scrolls on its own (`overflow-x:auto`), so the page never does.

Run: `go test ./internal/docsite -update`, then review the golden HTML.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/docsite -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/docsite .gitignore
git commit -m "feat(docsite): static pages, sidebar, canonical and prev/next"
```

### Task 14: The recap on the website

**Files:**
- Modify: `internal/docsite/templates/step.html`, `internal/docsite/site.go`, `internal/docsite/site_test.go`
- Create: `internal/docsite/assets/recap.js`, `internal/docsite/testdata/golden/first-box/pair-box/index.html` (updated)

**Interfaces:**
- Consumes: `Model.Recap`.
- Produces: in each step page, `<details class="recap"><summary>Show previous steps</summary>…</details>`. Inside it there is one `<details class="recap-group" data-pathway="<id>" [open]>` per group and one `<li id="recap-<step>">` per item (title link, summary, recap). An item with `image` adds `<details class="recap-img"><summary>Show image</summary><img loading="lazy" …></details>`. `recap.js` adds an "Expand all" button, and remembers which groups are open in `localStorage` under `jg-docs-recap`, inside try/catch.

- [ ] **Step 1: Write the failing test**

```go
func TestRecapOnTheStepPage(t *testing.T) {
	html := string(build(t)["first-box/pair-box/index.html"])
	install, current := strings.Index(html, `data-pathway="install"`), strings.Index(html, `data-pathway="first-box" open`)
	if install < 0 || current < 0 || install > current || strings.Contains(html, `data-pathway="install" open`) {
		t.Fatalf("groups: %d %d", install, current)
	}
	ord := []string{`id="recap-desktop-app"`, `id="recap-verify"`, `id="recap-box-ready"`, `id="recap-keys-init"`}
	for i := 1; i < len(ord); i++ {
		if strings.Index(html, ord[i-1]) > strings.Index(html, ord[i]) {
			t.Errorf("%s after %s", ord[i-1], ord[i])
		}
	}
	if strings.Count(html, `id="recap-desktop-app"`) != 1 || !strings.Contains(html, "<summary>Show previous steps</summary>") {
		t.Error("recap")
	}
	if !strings.Contains(html, `<script src="/docs/assets/recap.js" defer></script>`) {
		t.Error("script")
	}
	golden(t, build(t), "first-box/pair-box/index.html")
}
```

`TestRecapScriptGuardsStorage` reads `assets/recap.js` and asserts that every `localStorage` access sits inside a `try {`. It checks this by counting `localStorage` occurrences and `try {` blocks, both of which must be at least 1, with no `localStorage` before the first `try {`.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/docsite -run 'TestRecap' -v`
Expected: FAIL, `groups: -1 -1`

- [ ] **Step 3: Implement the recap markup and `recap.js`**

- [ ] **Step 4: Run the tests and update the golden**

Run: `go test ./internal/docsite -update && go test ./internal/docsite -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/docsite
git commit -m "feat(docsite): show previous steps"
```

### Task 15: Landing URLs keep working: redirects and `pages.json`

**Files:**
- Create: `docs/redirects.yaml`, `internal/docsite/redirects.go`, `internal/docsite/redirects_test.go`
- Modify: `internal/docsite/site.go` (`Pages` emits the redirect stubs and `pages.json`), `internal/docsite/templates/index.html` (the old single-page anchor script)

**Interfaces:**
- Consumes: Task 9's table ("Old landing URL" → pathway/step).
- Produces: `docs/redirects.yaml`, a map from the old path to the new path, both under `/docs/`. It holds every URL in the landing `docs/pages.json` (45 entries at `main` today; re-read it at execution time) whose new URL differs. The section indexes are `getting-started/` → `first-box/`, `tui/` → `first-box/tui-screens/`, `tui/keys/` → `reference/keys/`, `tui/commands/` → `reference/palette/`, `commands/` → `reference/commands/`, `commands/exit-codes/` → `reference/exit-codes/`, `jobs/` → `durable-jobs/`, `upgrading/` → `migrate-legacy-names/`, and `security/` → `first-box/local-server/`. Also produces `type Redirects map[string]string`, `func LoadRedirects(fsys fs.FS) (Redirects, []docs.Problem)`, and stub pages (`<meta http-equiv="refresh" content="0; url=/docs/<new>">`, `rel=canonical`, and a plain link). The output's `pages.json` lists every live page URL, sorted.

- [ ] **Step 1: Write the failing tests**

```go
// Every URL the interim docs published still answers: a page or a redirect to one.
func TestLandingURLsResolve(t *testing.T) {
	m, _ := docs.LoadFS(os.DirFS("../../docs"))
	pages := must(Pages(Options{Model: m, Tables: docstables.Live(), Base: "/docs"}))
	old := landingURLs(t) // testdata/landing-pages.json: a copy of landing main:docs/pages.json
	for _, u := range old {
		p := strings.TrimPrefix(u, "/docs/") + "index.html"
		body, ok := pages[p]
		if !ok {
			t.Errorf("%s: no page and no redirect", u)
			continue
		}
		if to := refreshTarget(body); to != "" && pages[strings.TrimPrefix(to, "/docs/")+"index.html"] == nil {
			t.Errorf("%s redirects to %s, which does not exist", u, to)
		}
	}
}

// Review Focus 5.
func TestRedirectNeverShadowsAPage(t *testing.T) {
	m, _ := docs.LoadFS(os.DirFS("../../docs/testdata/graph/cross"))
	_, probs := redirectsFor(m, fstest.MapFS{"redirects.yaml": {Data: []byte("install/desktop-app/: first-box/box-ready/\nold/: nowhere/\n")}})
	got := fmt.Sprint(probs)
	if !strings.Contains(got, "redirects.yaml:1: install/desktop-app/ is a live page") || !strings.Contains(got, "redirects.yaml:2: old/ points at nowhere/, which is not a page") {
		t.Fatal(got)
	}
}
```

`TestOldAnchorsMap` asserts that `index.html` contains the JSON map from each old single-page anchor (`install-desktop-app` → `/docs/install/desktop-app/`, built as the landing renderer built it: section slug, then `-` and the part slug) to its new URL.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/docsite -run 'TestLandingURLsResolve|TestRedirectNeverShadowsAPage|TestOldAnchorsMap' -v`
Expected: FAIL, `undefined: landingURLs`

- [ ] **Step 3: Copy `pages.json` into testdata, and write `redirects.yaml`, `LoadRedirects` and the stubs**

Run: `git -C ~/Documents/jumpgate-tech/github/landing show main:docs/pages.json > internal/docsite/testdata/landing-pages.json`

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/docsite ./docs/...`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add docs/redirects.yaml internal/docsite
git commit -m "feat(docsite): every interim docs URL redirects to its step"
```

### Task 16: `cmd/docsgen`, Pagefind, the `docs-site` job and branch previews

**Files:**
- Create: `cmd/docsgen/main.go`, `cmd/docsgen/main_test.go`
- Modify: `internal/docsite/templates/layout.html` (search box), `.gitlab-ci.yml` (jobs `docs-site`, `docs-preview`)

**Interfaces:**
- Consumes: `docs.LoadFS`, `docstables.Live`, all the checks, `docsite.Write`, `covers`.
- Produces: `go run ./cmd/docsgen [-src docs] [-out out/site/docs] [-shots out/shots] [-optional desktop]`. It runs every check from `TestDocsCheck` first, prints the problems and exits 1. Otherwise it writes the site. The subcommands `fixtures` and `shots-check` are added in Tasks 19–20. Exit codes: 0 ok, 1 problems, 2 usage.

- [ ] **Step 1: Write the failing test**

```go
func TestRunBuildsTheRepoDocs(t *testing.T) {
	out := t.TempDir()
	var stderr strings.Builder
	if code := run([]string{"-src", "../../docs", "-out", out}, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	for _, p := range []string{"index.html", "first-box/pair-box/index.html", "reference/keys/index.html", "pages.json", "assets/site.css"} {
		if _, err := os.Stat(filepath.Join(out, p)); err != nil {
			t.Error(err)
		}
	}
}

func TestRunFailsWithFileLine(t *testing.T) {
	src := t.TempDir()
	copyTree(t, "../../docs/testdata/tree", src)
	os.WriteFile(filepath.Join(src, "steps/a/one.md"), []byte("---\nid: one\ntitle: One\nsummary: s\nrecap: r\n---\n{{cmd \"frob\"}}\n"), 0o644)
	var stderr strings.Builder
	if code := run([]string{"-src", src, "-out", t.TempDir()}, &stderr); code != 1 || !strings.Contains(stderr.String(), `steps/a/one.md:7: unknown command "frob"`) {
		t.Fatalf("%d %s", code, stderr.String())
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd/docsgen -v`
Expected: FAIL, `no Go files in .../cmd/docsgen`

- [ ] **Step 3: Implement `run`, the search box and the CI jobs**

The search box is `<div id="search"></div>`. It loads `/docs/pagefind/pagefind-ui.js` and `.css` lazily on focus. Without Pagefind (a local build), it hides itself.

```yaml
docs-site:
  stage: test
  image: golang:1.26-bookworm
  cache: { key: { files: [go.sum] }, paths: [.cache/go-mod/, .cache/go-build/] }
  script:
    - go run ./cmd/docsgen -out out/site/docs -optional app,desktop
    - apt-get update -qq && apt-get install -y -qq --no-install-recommends nodejs npm >/dev/null
    - npx -y pagefind@1.5.2 --site out/site/docs
  artifacts: { paths: [out/site/], expire_in: 2 weeks }

docs-preview:
  stage: test
  image: node:22-bookworm
  needs: [docs-site]
  rules:
    - if: '$CI_COMMIT_BRANCH && $CI_COMMIT_BRANCH != $CI_DEFAULT_BRANCH && $CLOUDFLARE_API_TOKEN'
  script:
    - npx -y wrangler@4 pages deploy out/site --project-name jumpgate --branch "docs-$CI_COMMIT_REF_SLUG"
```

Task 19 removes `app` from `-optional` once its job feeds the shots.

- [ ] **Step 4: Run the tests, then a local build with search**

Run: `go test ./cmd/docsgen -v && go run ./cmd/docsgen -out out/site/docs && npx -y pagefind@1.5.2 --site out/site/docs`
Expected: PASS. Pagefind prints `Indexed … pages`. Opening `out/site/docs/index.html` in a browser shows the sidebar.

- [ ] **Step 5: Commit**

```bash
git add cmd/docsgen internal/docsite/templates/layout.html .gitlab-ci.yml
git commit -m "feat(docsgen): build the site, Pagefind, docs-site and preview jobs"
```

---

## Phase 5: Scenes

### Task 17: TUI scenes as named fixtures

**Files:**
- Create: `internal/tui/scenes/scenes.go`, `internal/tui/scenes/scenes_test.go`
- Modify: `internal/tui/tuitest/fake.go` (a `PrefsV api.UIPrefs` field returned by the prefs call, if it is absent), `internal/docstables/docstables.go` (adds the TUI scenes), `cmd/jumpgate/tui_on.go` (`Options.Scene = scenes.Text`)

**Interfaces:**
- Consumes: `tui.New(tui.Options)`, `tuitest.NewFake`, `tuitest.Send`, `tuitest.Settle`, `tuitest.Key`, `tuitest.Type`.
- Produces:
  ```go
  type Scene struct{ Name, Caption string; Keys []string; Typed string }
  func All() []Scene                                      // fleet help jobs signers hostadd-confirm settings guide-recap
  func Render(name string, ascii bool) (string, error)    // 80x24 frame WITH SGR codes; clock 2026-10-05T12:00:00Z, UTC, GOOS linux, Hostname laptop, empty env
  func Text(name string, ascii bool) (string, bool)       // ansi.Strip(Render) with trailing spaces trimmed; tui.Options.Scene
  func Docs() []docs.Scene                                // Kind "tui"
  ```

- [ ] **Step 1: Write the failing tests**

```go
// A scene is the golden machinery's frame: the fleet scene's text is the fleet golden.
func TestFleetSceneIsTheGolden(t *testing.T) {
	want, _ := os.ReadFile("../testdata/fleet_80x24.golden")
	got, ok := Text("fleet", false)
	if !ok || got != strings.TrimRight(string(want), "\n") {
		t.Fatalf("got\n%s", got)
	}
}

func TestScenesAreDeterministicAndColoured(t *testing.T) {
	for _, s := range All() {
		a, err := Render(s.Name, false)
		b, _ := Render(s.Name, false)
		if err != nil || a != b || !strings.Contains(a, "\x1b[") || len(strings.Split(a, "\n")) != 24 {
			t.Errorf("%s: err %v, stable %v", s.Name, err, a == b)
		}
		txt, _ := Text(s.Name, true)
		for _, r := range txt {
			if r > 127 {
				t.Errorf("%s ascii: %q", s.Name, r)
				break
			}
		}
	}
	if _, err := Render("nope", false); err == nil {
		t.Fatal("unknown scene rendered")
	}
}
```

If the `fleet_80x24` golden is drawn with a different fake than `NewFake()`'s defaults, the scene uses the same setup as `TestFleetFrame` (or the equivalent golden test). Read that test first.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/tui/scenes -v`
Expected: FAIL, `no Go files`

- [ ] **Step 3: Implement `scenes`**

Each scene sends a `tea.WindowSizeMsg{80,24}`, then `Init`'s commands through `Settle` (prefs come from the fake with `Glyphs` set), then its `Keys` and `Typed`. Content is `m.View().Content`.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/tui/... ./internal/docstables ./docs/... ./cmd/jumpgate`
Expected: all `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/tui/scenes internal/tui/tuitest/fake.go internal/docstables/docstables.go cmd/jumpgate/tui_on.go
git commit -m "feat(tui): named scenes from the golden fixtures"
```

### Task 18: ANSI→SVG, scene images on the site, frames in Help

**Files:**
- Create: `internal/ansisvg/ansisvg.go`, `internal/ansisvg/ansisvg_test.go`, `internal/ansisvg/testdata/small.svg`
- Modify: `cmd/docsgen/main.go` (renders every referenced TUI scene to `img/tui/<name>.svg`), `internal/tui/guide.go` (`v` opens the frame viewer), `internal/tui/guide_test.go`, `docs/steps/**` (add `image: "{{shot tui …}}"` where the landing had `tui-*` images: keys-init → signers, pair-box → hostadd-confirm, terminal-only → fleet, the-guide → guide-recap, job-commands → jobs, tui-screens → fleet)

**Interfaces:**
- Produces:
  ```go
  type Options struct{ Cols, Rows int; Palette [16]string; FG, BG string }
  var Landing = Options{Cols: 80, Rows: 24, FG: "#d9d6ce", BG: "#0a0a09", Palette: [16]string{
      0: "#0a0a09", 1: "#ff6f61", 2: "#6bd66b", 3: "#e8b04a", 4: "#7aa2f7", 5: "#c792ea", 6: "#5fe6a2", 7: "#d9d6ce",
      8: "#8f8b82", 9: "#ff6f61", 10: "#6bd66b", 11: "#e8b04a", 12: "#7aa2f7", 13: "#c792ea", 14: "#5fe6a2", 15: "#f3f1ec"}}
  func Render(frame string, o Options) []byte
  ```

- [ ] **Step 1: Write the failing tests**

```go
func TestRenderSmallFrame(t *testing.T) {
	frame := "\x1b[1mTitle\x1b[0m <a&b>\n\x1b[32mok\x1b[0m \x1b[7msel\x1b[0m"
	got := Render(frame, Options{Cols: 12, Rows: 2, FG: "#fff", BG: "#000", Palette: Landing.Palette})
	golden(t, "small.svg", got) // -update writes testdata/small.svg
	s := string(got)
	for _, want := range []string{`font-weight="700"`, "&lt;a&amp;b&gt;", `fill="#6bd66b"`, `xml:space="preserve"`, `textLength=`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s", want)
		}
	}
	if !bytes.Equal(got, Render(frame, Options{Cols: 12, Rows: 2, FG: "#fff", BG: "#000", Palette: Landing.Palette})) {
		t.Fatal("not byte-stable")
	}
}

func TestUnknownSGRIsIgnored(t *testing.T) {
	if s := string(Render("\x1b[38;5;200mx\x1b[0m\x1b]8;;http://x\x07y", Landing)); !strings.Contains(s, ">x<") || !strings.Contains(s, "y") {
		t.Fatal(s)
	}
}
```

```go
// internal/tui/guide_test.go: v on a step whose image is a TUI shot shows the frame through Options.Scene.
func TestViewOpensTheFrame(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	a.o.Scene = func(name string, ascii bool) (string, bool) { return "FRAME " + name, true }
	fr := tuitest.Frame(press(t, typed(t, press(t, a, ":"), "help pair-box"), "enter", "v"))
	if !strings.HasPrefix(fr, "FRAME hostadd-confirm") {
		t.Fatal(fr)
	}
}
```

`TestDocsgenWritesSceneSVGs` (in `cmd/docsgen`) asserts that `img/tui/hostadd-confirm.svg` exists after `run`, and that the pair-box page has `<img src="/docs/img/tui/hostadd-confirm.svg"`.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/ansisvg ./internal/tui ./cmd/docsgen -run 'TestRender|TestUnknownSGR|TestViewOpensTheFrame|TestDocsgenWritesSceneSVGs' -v`
Expected: FAIL, `no Go files in .../internal/ansisvg`

- [ ] **Step 3: Implement `Render`, the scene images in `docsgen` and the frame viewer**

Parse the SGR codes with `github.com/charmbracelet/x/ansi` (already a dependency): bold, reverse, 30–37, 90–97, 39 and 0. Anything else is dropped. The cell is 9.6×20 px (0.6em at 16px). There is one `<text>` per row with `textLength` set to cols×9.6, and runs become `<tspan fill>`. Reverse video paints a `<rect>` under the run. The font is `JetBrains Mono, ui-monospace, monospace`. The viewer shows the frame alone at 80x24. esc returns to the page, and `v` again moves to the next shot on the page.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/ansisvg -update && go test ./... `
Expected: all `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/ansisvg cmd/docsgen internal/tui/guide.go internal/tui/guide_test.go docs/steps
git commit -m "feat(docs): TUI scenes as SVG on the site and as frames in Help"
```

### Task 19: Playwright app scenes, site browser checks and `docs-shots`

**Files:**
- Create: `internal/appfixtures/appfixtures.go`, `internal/appfixtures/appfixtures_test.go`, `cmd/jumpgate/web/playwright.config.ts`, `cmd/jumpgate/web/shots/app.spec.ts`, `cmd/jumpgate/web/shots/site.spec.ts`, `cmd/jumpgate/web/shots/route.ts`, `cmd/jumpgate/web/shots/fonts/{JetBrainsMono-Regular,Archivo-Regular}.woff2` (OFL; with `OFL.txt`)
- Modify: `docs/shots/app.json`, `cmd/docsgen/main.go` (`fixtures -out DIR`), `cmd/jumpgate/web/package.json` + `package-lock.json` (`"@playwright/test": "1.64.1"` exact; script `"shots": "playwright test"`), `.gitignore` (`cmd/jumpgate/web/shots/.fixtures/`, `cmd/jumpgate/web/test-results/`), `.gitlab-ci.yml` (`docs-fixtures`, `docs-shots`; `docs-site` gains `needs: [docs-shots]`, `-shots out/shots` and `-optional desktop`), `docs/steps/**` (add `image: "{{shot app …}}"`: desktop-app → home, watch-it → machine, open-web-app → machines, endpoints → rpc)

**Interfaces:**
- Produces:
  - `docs/shots/app.json`: `[{"name":"home","path":"/","caption":"The app's home panel"},{"name":"machines","path":"/targets","caption":"Your machines"},{"name":"machine","path":"/machine/box-a","caption":"One machine: sync, peers and disk"},{"name":"rpc","path":"/rpc","caption":"The RPC endpoint"}]`
  - `func Routes() map[string]any` in `internal/appfixtures`: `"GET /api/…"` → a typed `internal/api` value. SSE routes map to `[]any` events. `docsgen fixtures -out DIR` writes `DIR/routes.json`.
  - The shots go to `out/shots/app/<name>.png` (viewport 1280×800, `deviceScaleFactor: 1`, `colorScheme: 'dark'`, `timezoneId: 'UTC'`, `locale: 'en-US'`, `page.clock.install({ time: new Date('2026-10-05T12:00:00Z') })`, `animations: 'disabled'`, `caret: 'hide'`).

- [ ] **Step 1: Write the failing tests**

```go
// internal/appfixtures: every route marshals, and every path the web client calls has one.
func TestRoutesCoverTheClient(t *testing.T) {
	src, _ := os.ReadFile("../../cmd/jumpgate/web/src/api.ts")
	for _, path := range apiPaths(string(src)) { // `/api/...` literals with ${…} turned into box-a / gw-a
		if _, ok := Routes()["GET "+path]; !ok && strings.HasSuffix(path, "/stream") == false && isGET(string(src), path) {
			t.Errorf("no fixture for GET %s", path)
		}
	}
	for k, v := range Routes() {
		if _, err := json.Marshal(v); err != nil {
			t.Errorf("%s: %v", k, err)
		}
	}
}
```

```ts
// shots/route.ts: an /api request with no fixture fails the scene (no silent empty screen).
// shots/app.spec.ts
for (const s of scenes) test(`app scene ${s.name}`, async ({ page }) => {
  const missed = await routeFixtures(page);           // fulfils from .fixtures/routes.json, records misses
  await page.clock.install({ time: new Date("2026-10-05T12:00:00Z") });
  await page.goto(s.path);
  await page.mouse.move(-1, -1);
  await page.screenshot({ path: `../../../out/shots/app/${s.name}.png`, animations: "disabled", caret: "hide" });
  expect(missed, "requests without a fixture").toEqual([]);
});
```

```ts
// shots/site.spec.ts: the spec's landing checks, against out/site served by playwright's webServer
test("recap toggles and its image loads lazily", async ({ page }) => {
  await page.goto("/docs/first-box/pair-box/");
  const recap = page.locator("details.recap");
  await expect(recap).not.toHaveAttribute("open", "");
  await recap.locator("summary").click();
  await expect(recap.locator('details.recap-group[data-pathway="first-box"]')).toHaveAttribute("open", "");
  const img = recap.locator("details.recap-img img").first();
  await expect(img).toHaveAttribute("loading", "lazy");
});
for (const path of ["/docs/", "/docs/first-box/pair-box/", "/docs/reference/commands/"])
  test(`no horizontal scroll at 375px: ${path}`, async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 800 });
    await page.goto(path);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  });
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/appfixtures -v`
Expected: FAIL, `no Go files`

Run (in the pinned container): `docker run --rm -v "$PWD":/w -w /w/cmd/jumpgate/web mcr.microsoft.com/playwright:v1.64.1-noble sh -c 'npm ci && npx playwright test'`
Expected: FAIL, `ENOENT … .fixtures/routes.json`

- [ ] **Step 3: Implement `appfixtures`, the fixtures subcommand, the specs, the fonts and the CI jobs**

```yaml
docs-fixtures:
  stage: test
  image: golang:1.26-bookworm
  script: [go run ./cmd/docsgen fixtures -out cmd/jumpgate/web/shots/.fixtures]
  artifacts: { paths: [cmd/jumpgate/web/shots/.fixtures/] }
docs-shots:
  stage: test
  image: mcr.microsoft.com/playwright:v1.64.1-noble
  needs: [docs-fixtures]
  script:
    - cd cmd/jumpgate/web && npm ci && npx playwright test --project=app
  artifacts: { paths: [out/shots/app/], when: always }
docs-site-check:
  stage: test
  image: mcr.microsoft.com/playwright:v1.64.1-noble
  needs: [docs-site]
  script: [cd cmd/jumpgate/web && npm ci && npx playwright test --project=site]
```

The `docs-site` job then has `needs: [docs-shots]` and runs `docsgen -shots out/shots -optional desktop`. `playwright.config.ts` defines the projects `app` (webServer `npx vite preview --port 4173` over the committed `dist/`) and `site` (webServer `npx http-server ../../../out/site -p 4174`, pinned `http-server@14.1.1`). The specs inject the bundled fonts with `page.addStyleTag` (`@font-face` for `--font-mono` and the sans stack), so no web font is fetched.

- [ ] **Step 4: Run them**

Run: `go test ./internal/appfixtures ./cmd/docsgen && go run ./cmd/docsgen fixtures -out cmd/jumpgate/web/shots/.fixtures && go run ./cmd/docsgen -out out/site/docs -optional desktop -shots out/shots`
Then the docker command from Step 2.
Expected: Go `ok`. Playwright reports `6 passed` (4 app scenes and the site checks) and `out/shots/app/*.png` exist. The second `docsgen` run must go after the shots exist; repeat it once they do.

- [ ] **Step 5: Commit**

```bash
git add internal/appfixtures cmd/docsgen cmd/jumpgate/web/playwright.config.ts cmd/jumpgate/web/shots/app.spec.ts cmd/jumpgate/web/shots/site.spec.ts cmd/jumpgate/web/shots/route.ts cmd/jumpgate/web/shots/fonts cmd/jumpgate/web/package.json cmd/jumpgate/web/package-lock.json docs/shots/app.json docs/steps .gitignore .gitlab-ci.yml
git commit -m "feat(docs): Playwright app scenes and site browser checks"
```

### Task 20: Desktop scenes on the Windows VM and macOS runners

**Files:**
- Create: `scripts/docs-shots/windows.ps1`, `scripts/docs-shots/macos.sh`, `cmd/docsgen/shots.go`, `cmd/docsgen/shots_test.go`
- Modify: `docs/shots/desktop.json`, `.gitlab-ci.yml` (`docs-shots-desktop-windows`, `docs-shots-desktop-macos`), `docs/steps/install/unsigned-downloads.md` (adds `{{shot desktop win-smartscreen}}` and `{{shot desktop mac-open-anyway}}`), `docs/steps/install/desktop-app.md` (`{{shot desktop win-tray}}`, `{{shot desktop mac-menubar}}`)

**Interfaces:**
- Produces:
  - `docs/shots/desktop.json`: `[{"name":"win-smartscreen","os":"windows","caption":"Windows SmartScreen: More info, then Run anyway"},{"name":"win-tray","os":"windows","caption":"The notification-area icon"},{"name":"mac-open-anyway","os":"darwin","caption":"Privacy & Security: Open Anyway"},{"name":"mac-menubar","os":"darwin","caption":"The menu-bar item"}]`
  - `docsgen shots-check -kind desktop -os windows|darwin -dir DIR`: exit 1 naming each scene of that OS whose `DIR/<name>.png` is missing, is not a PNG, or is a single colour.

- [ ] **Step 1: Write the failing test**

```go
// Review Focus 4.
func TestShotsCheckRejectsBlankAndMissing(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "win-smartscreen.png"), solid(64, 64, color.Black))
	writePNG(t, filepath.Join(dir, "win-tray.png"), checker(64, 64))
	var stderr strings.Builder
	code := run([]string{"shots-check", "-kind", "desktop", "-os", "windows", "-dir", dir}, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "win-smartscreen: a single colour (locked screen or no desktop session?)") || strings.Contains(stderr.String(), "win-tray") {
		t.Fatalf("%d %s", code, stderr.String())
	}
	os.Remove(filepath.Join(dir, "win-tray.png"))
	if run([]string{"shots-check", "-kind", "desktop", "-os", "windows", "-dir", dir}, &stderr); !strings.Contains(stderr.String(), "win-tray: missing") {
		t.Fatal(stderr.String())
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd/docsgen -run TestShotsCheckRejectsBlankAndMissing -v`
Expected: FAIL, `unknown command "shots-check"` (exit 2)

- [ ] **Step 3: Implement `shots-check`, the capture scripts and the jobs**

`windows.ps1` first checks `query session` for an `Active` console session, and exits 1 with "no interactive desktop session" otherwise. It builds `jumpgate-tray.exe` the way `windows-tray-cross` does, natively. It writes a `Zone.Identifier` ADS (`ZoneId=3`) so SmartScreen shows, starts the exe, and captures each scene's window with `System.Drawing.Graphics.CopyFromScreen` into `out/shots/desktop/<name>.png`. `macos.sh` runs `scripts/../cmd/jumpgate/build-macos-app.sh`, `xattr -w com.apple.quarantine`, `open`, and `screencapture -x -l<window id>`. Both end with `go run ./cmd/docsgen shots-check -kind desktop -os <os> -dir out/shots/desktop`.

```yaml
.desktop-shot-rules: &desktop-shot-rules
  - if: $CI_COMMIT_TAG                        # required on tags (spec §4)
  - changes: [packaging/**/*, cmd/jumpgate/build-macos-app.sh, cmd/jumpgate/gui*.go, cmd/jumpgate/tray*.go,
              cmd/jumpgate/statusitem*.go, cmd/jumpgate/console_windows.go, scripts/package-*.sh,
              scripts/docs-shots/**/*, docs/shots/desktop.json]
  - when: manual
    allow_failure: true
docs-shots-desktop-windows:
  stage: test
  tags: [windows]                             # the on-demand Windows VM runner (scripts/windows-vm)
  rules: *desktop-shot-rules
  script: [powershell -File scripts/docs-shots/windows.ps1]
  artifacts: { paths: [out/shots/desktop/] }
docs-shots-desktop-macos:
  stage: test
  tags: [macos]
  rules: *desktop-shot-rules
  script: [scripts/docs-shots/macos.sh]
  artifacts: { paths: [out/shots/desktop/] }
```

The `windows` job's header comment changes to say the VM runner serves the `windows` tag, and that a pending tagged job starts the VM through `scripts/windows-vm/on-demand.sh`.

- [ ] **Step 4: Run them**

Run: `go test ./cmd/docsgen -v`
Expected: PASS
Then run the windows job once on the VM runner from the GitLab UI (git.craftsmanfounder.com, project 71, play `docs-shots-desktop-windows`). Expected: the job passes and its artifact holds `win-smartscreen.png` and `win-tray.png`. Open both and check that each shows the dialog or icon.

- [ ] **Step 5: Commit**

```bash
git add scripts/docs-shots cmd/docsgen/shots.go cmd/docsgen/shots_test.go docs/shots/desktop.json .gitlab-ci.yml docs/steps/install/unsigned-downloads.md docs/steps/install/desktop-app.md
git commit -m "feat(docs): desktop scenes on the Windows VM and macOS runners"
```

---

## Phase 6: Publishing

### Task 21: `docs-site.tar.gz` on every GitHub release

**Files:**
- Modify: `.github/workflows/release.yml` (jobs `docs-desktop`, `docs-site`; `provenance.needs` gains `docs-site`)
- Create: `scripts/docs-site-archive.sh`, `scripts/docs-site-archive_test.sh`

**Interfaces:**
- Consumes: `docsgen`, `appfixtures`, the Playwright `app` project, `scripts/docs-shots/{windows.ps1,macos.sh}`, Pagefind.
- Produces: the release assets `docs-site.tar.gz` (top-level `docs/`) and `docs-site.tar.gz.sha256` (`<hex>  docs-site.tar.gz`). `scripts/docs-site-archive.sh SITE_DIR OUT_DIR` writes both reproducibly (`tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner`, `gzip -n`).

- [ ] **Step 1: Write the failing test**

```bash
# scripts/docs-site-archive_test.sh: two runs give the same bytes; the archive has docs/ at the top; the checksum verifies.
set -euo pipefail
s=$(mktemp -d); o1=$(mktemp -d); o2=$(mktemp -d)
mkdir -p "$s/docs/a" && echo hi > "$s/docs/a/index.html" && echo x > "$s/docs/index.html"
scripts/docs-site-archive.sh "$s" "$o1"; touch "$s/docs/index.html"; scripts/docs-site-archive.sh "$s" "$o2"
cmp "$o1/docs-site.tar.gz" "$o2/docs-site.tar.gz"
tar -tzf "$o1/docs-site.tar.gz" | grep -qx 'docs/index.html'
! tar -tzf "$o1/docs-site.tar.gz" | grep -qv '^docs/'
(cd "$o1" && sha256sum -c docs-site.tar.gz.sha256)
echo PASS
```

- [ ] **Step 2: Run it to verify it fails**

Run: `bash scripts/docs-site-archive_test.sh`
Expected: FAIL, `scripts/docs-site-archive.sh: No such file or directory`

- [ ] **Step 3: Implement the script and the release jobs**

`docs-desktop` is a matrix over `windows-latest` and `macos-latest`. It runs the capture script and uploads the `docs-desktop-<os>` artifact. `docs-site` runs on `ubuntu-latest` in `container: mcr.microsoft.com/playwright:v1.64.1-noble`, with `setup-go` pinned like the other jobs and `needs: [binaries, docs-desktop]`. Its steps are: the fixtures, `npx playwright test --project=app`, `go run ./cmd/docsgen -out out/site/docs -shots out/shots` (nothing optional on a tag), `npx -y pagefind@1.5.2 --site out/site/docs`, `scripts/docs-site-archive.sh out/site out`, and `gh release upload "$TAG" out/docs-site.tar.gz out/docs-site.tar.gz.sha256 --clobber`. The permissions are `contents: write`. Use the same pinned action SHAs as the jobs above.

- [ ] **Step 4: Run the test and validate the workflow**

Run: `bash scripts/docs-site-archive_test.sh && go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/release.yml`
Expected: `PASS` and no actionlint output. Then trigger `workflow_dispatch` on GitHub only with the user's permission (it publishes nothing). Expected: `docs-site` builds the archive as a run artifact. On a dispatch the upload step is skipped (`if: startsWith(github.ref, 'refs/tags/v')`).

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yml scripts/docs-site-archive.sh scripts/docs-site-archive_test.sh
git commit -m "ci(release): attach docs-site.tar.gz and its checksum"
```

### Task 22: The landing deploy pulls `/docs`; the interim renderer is retired (landing repo)

Run this task in `~/Documents/jumpgate-tech/github/landing` on a branch `docs-from-release`, and only after Precondition 4. It needs a release with `docs-site.tar.gz` (Task 21). Before that, test against a local archive from `scripts/docs-site-archive.sh`.

**Files (landing repo):**
- Create: `scripts/pull-docs.sh`, `scripts/pull-docs_test.sh`
- Modify: `README.md` (the deploy steps), `.gitignore` (`/docs/`)
- Delete: `scripts/render-docs.mjs`, `scripts/docs-images.mjs`, `docs/` (the `src/*.md`, `img/`, generated folders and `pages.json`)

**Interfaces:**
- Consumes: the release assets from Task 21 on `jumpgate-tech/jumpgate-app`.
- Produces: `scripts/pull-docs.sh DEST [ARCHIVE_DIR]`. It downloads the latest release's `docs-site.tar.gz` and `.sha256` with `gh release download -R jumpgate-tech/jumpgate-app -p 'docs-site.tar.gz*'`, or uses `ARCHIVE_DIR` when given. It verifies with `shasum -a 256 -c`, replaces `DEST/docs`, and fails without touching `DEST` when the checksum fails.

- [ ] **Step 1: Write the failing test**

```bash
# scripts/pull-docs_test.sh APP_REPO
set -euo pipefail
app=${1:?path to jumpgate-app}; a=$(mktemp -d); d=$(mktemp -d)
(cd "$app" && go run ./cmd/docsgen -out "$a/site/docs" -optional app,desktop && scripts/docs-site-archive.sh "$a/site" "$a")
mkdir -p "$d/docs" && echo stale > "$d/docs/old.html"
scripts/pull-docs.sh "$d" "$a"
test -f "$d/docs/first-box/pair-box/index.html" && test ! -e "$d/docs/old.html"
# every URL the interim docs published still answers
for u in $(git show main:docs/pages.json | tr -d '[]",'); do test -f "$d${u}index.html" || { echo "missing $u"; exit 1; }; done
echo tampered >> "$a/docs-site.tar.gz"; d2=$(mktemp -d)
! scripts/pull-docs.sh "$d2" "$a" && test -z "$(ls -A "$d2")"
echo PASS
```

- [ ] **Step 2: Run it to verify it fails**

Run: `bash scripts/pull-docs_test.sh ~/Documents/jumpgate-tech/github/jumpgate-app`
Expected: FAIL, `scripts/pull-docs.sh: No such file or directory`

- [ ] **Step 3: Implement `pull-docs.sh`, rewrite the README deploy, and delete the renderer**

```bash
D=$(mktemp -d) && git archive HEAD | tar -x -C "$D"
scripts/pull-docs.sh "$D"
npx wrangler@4 pages deploy "$D" --project-name jumpgate --branch main
```

The README drops the `docs/src` and `render-docs.mjs` paragraph. Instead, it says the docs come from the app repo's release and that `/docs` always matches the current download.

- [ ] **Step 4: Run the test**

Run: `bash scripts/pull-docs_test.sh ~/Documents/jumpgate-tech/github/jumpgate-app`
Expected: `PASS`

- [ ] **Step 5: Commit (landing repo)**

```bash
git add scripts/pull-docs.sh scripts/pull-docs_test.sh README.md .gitignore
git rm -r scripts/render-docs.mjs scripts/docs-images.mjs docs
git commit -m "feat(docs): /docs comes from the jumpgate release; retire the interim renderer"
```

Deploying the landing site is a separate, user-approved step. It uses the valaxyblackdev Cloudflare account (see the memory note on the landing deploy).

---

## Self-review

**Spec coverage.**

| Spec section | Where it is covered |
|---|---|
| §1 content model, fields, layout | Tasks 4 and 8 |
| order versus dependencies, three graph checks | Task 5 |
| `docs:covers` and the hash rule | Task 7 |
| sidecars | Tasks 4 and 7, plus D4 |
| §2 every reference row | Task 6 |
| CLI registry | Tasks 1 and 3 (D1) |
| exit and env tables | Task 2 |
| reference pages | Task 10 (Task 13 renders them) |
| TUI scenes | Tasks 17–18 |
| app scenes | Task 19 |
| desktop scenes | Task 20 |
| `{{shot}}` registered | Tasks 6 and 8 |
| §3 website sidebar, URLs, canonical, `<details>` and script, Pagefind | Tasks 13, 14 and 16 |
| recap algorithm | Tasks 5 and 14 |
| TUI navigation, `r`, Enter, frames, 80x24, ascii, NO_COLOR | Tasks 11, 12 and 18 |
| §4 `go test` gate | Task 8 |
| `docs-shots` | Task 19 |
| `docs-shots-desktop` rules | Task 20 |
| release attachment | Task 21 |
| landing pull | Task 22 |
| per-binary Help | embedded via `docs.Embedded()` |
| previews | Task 16 |
| error format | Task 7 |
| goldens and negative tests | Tasks 5, 6, 7 and 13 |
| landing browser checks | Task 19 (`site.spec.ts`) |
| §5 pathways | Tasks 8 and 9 (*Run jumpgate on a server* deferred, D15) |
| `docs/guide` split and removed | Tasks 9 and 12 |

The out-of-scope items are not planned.

**Gaps in the spec, decided above:**
- D1: package `main` cannot be imported.
- D3: the spec's `image: {{…}}` is not valid YAML.
- D4: there is no reference for hand-made images, and macOS packaging is not Go.
- D5: the spec's example covered items do not exist.
- D6: pathway order is not specified.
- D8: "step or reference page" is vacuous.
- D9: there are no network slugs, client names or key ids.
- D12: there is no registered `macos` runner yet, so a tag pipeline on GitLab waits until one is registered. The spec accepts that wait; the user should know.
- The spec mentions "landing browser checks" that do not exist in either repo, so Task 19 creates them.

**Type consistency.** These names match in every task that uses them: `docs.Model`, `Step`, `Pathway`, `Image`, `Problem`, `Tables`, `Command`, `Flag`, `Key`, `Named`, `Exit`, `EnvVar`, `Scene`, `Ref`, `Resolved`, `Block`, `Inline`, `RecapGroup`, `Section`, `Covered`, `Page`; `LoadFS`, `Embedded`, `CLI`, `Shots`, `Parse`, `ParseRef`, `Resolve`, `Blocks`, `CheckRefs`, `CheckGraph`, `CheckVerified`, `Order`, `Canonical`, `Neighbours`, `Sections`, `Recap`, `ReferencePages`; `covers.Scan`, `covers.Hash`, `covers.ModuleRoot`; `tui.DocTables`, `tui.KeyDoc.ID`, `tui.Options.Scene`; `scenes.All`, `scenes.Render`, `scenes.Text`, `scenes.Docs`; `docstables.Live`; `docsite.Options`, `docsite.Pages`, `docsite.Write`, `docsite.LoadRedirects`; `ansisvg.Render`, `ansisvg.Landing`; `appfixtures.Routes`.

**Review Focus.** Each of the five has a named test in its owning task: 6, 4 and 7, 13, 20, and 5 and 15.

**Proportion.** The spec is 191 lines and this plan is about 1,100. Code blocks are tests and signatures. The only algorithm bodies are the hash steps (Task 7) and the CI YAML, which the tests cannot determine.
