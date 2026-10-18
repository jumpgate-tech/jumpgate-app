# Docs system: atomic steps, pathways, docs checked against the code

Date: 2026-10-17
Status: design approved in conversation; written spec awaiting review
Research: `.claude/research/docs-system.md`
Builds on: v0.5.2 TUI guide (`feat/tui-guide`: `guide.Guide`, `tui.Keymap()`, `tui.Commands()`)

## Goal

Docs that can go deep without losing the reader, and that can't drift from the code.

- Content is split into **atomic steps**, and the steps are ordered into **pathways**.
- Any step can show **"Show previous steps"**: a recap of everything it depends on.
- Wherever the code can describe itself, the content comes **from the code**.
- When code changes and a step describing it hasn't been updated, **the build fails**: on main, on release tags, and on previews.
- One source feeds both jumpgate.app/docs and the TUI's Help view.

## Decisions

1. **Readers.** Newcomers and operators, on separate pathways that share steps. Steps carry no audience; pathways do.
2. **Staleness always blocks.** Nothing ships a step whose covered code changed without an update in the same change. Enforcement is part of the type system and the build (`go test ./...`), not a separate tool.
3. **Recap reach.** The whole dependency chain, across pathways, grouped by pathway. The current pathway's group starts open; the others start collapsed.
4. **Images.**
   - Generated from source on every build: TUI scenes and app scenes.
   - Hand-made images are allowed only with a sidecar that ties them to the code they show, under the same hash rule as prose.
   - Generated images are never committed.
5. **Renderer.** Go builds everything (`cmd/docsgen`): one resolved model, a static website and the TUI. No docs framework. Search is Pagefind, run at deploy time.

## 1. Content model

Everything lives in the app repo under `docs/`, which is a Go package, so `go:embed` and `go test` can reach it.

```
docs/
  steps/<area>/<id>.md      one atomic step each
  pathways/<id>.yaml        ordered routes through steps
  images/<name>.(png|svg)   hand-made images only
  images/<name>.yaml        sidecar: covers + verified
  docs.go                   embed, loader, checks
```

**Step file** (markdown with frontmatter):

```yaml
id: pair-box
title: Pair a box
summary: Install the agent and confirm the box's host key.   # one line; recap list
recap: |                                                     # 2–3 lines; the crucial bits
  You confirmed the fingerprint; the box now only accepts signed commands.
image: {{shot tui hosts-paired}}                             # the recap's crucial image
requires: [keys-init, box-ready]
verified: { internal/setup.Pair: 3f9a…, cmd/jumpgate.hostsAdd: 81c0… }
```

- The body is markdown of any depth. It contains no raw HTML, uses typed references (§2), and allows only headings, paragraphs, lists, fenced code, inline code, emphasis and links.
- Every field is required except `image`, `requires` and `verified`. `verified` becomes required once any code is marked as covering the step.

**Pathway file:**

```yaml
id: first-box
title: Your first box
audience: newcomer          # newcomer | operator
after: [install]            # prerequisite pathways
steps: [box-ready, keys-init, pair-box, choose-network, first-sync]
```

**Order and dependencies are separate.**
- A pathway's `steps` list gives the reading order.
- `requires` gives the dependency graph that the recap follows.
- The build checks three things:
  - every `requires` entry appears earlier in the pathway, or in a pathway reachable through `after`;
  - there are no cycles in `requires` or `after`;
  - every step belongs to at least one pathway.

**Code points back to the steps.**
- A function or method marked with `// docs:covers <step-id>` is a covered item.
- Its hash is SHA-256 over the declaration's source after `gofmt` normalisation, with comments stripped so that comment edits don't invalidate a step. The hash is truncated to 12 hex characters.
- The step's `verified` map must contain an entry for every covered item, with the current hash.

**Hand-made image sidecar:**

```yaml
alt: macOS "Open Anyway" button in Privacy & Security
covers: [cmd/jumpgate.buildMacApp]
verified: { cmd/jumpgate.buildMacApp: 77d2… }
```

## 2. Typed references and generators

References resolve against real tables. An unknown name fails `go test` with the step's file:line.

| Reference | Resolves against | Renders as |
|---|---|---|
| `{{cmd "hosts add"}}` | CLI command registry (new) | the command and its usage line |
| `{{flag "hosts add" "--jump"}}` | that command's flag set | the flag and its help |
| `{{key fleet.sort}}` | `tui.Keymap()` | the key; a tooltip on the web |
| `{{palette "logs"}}` | `tui.Commands()` | `:logs NAME` |
| `{{network pls}}`, `{{client lighthouse}}` | network catalog | the display name |
| `{{exit 4}}`, `{{env JUMPGATE_ASCII}}` | exit-code and env tables (new) | the code or variable and its meaning |
| `{{step pair-box}}` | the steps | a link with the step's title |
| `{{shot tui fleet}}`, `{{shot app machines}}`, `{{shot desktop win-smartscreen}}` | scene registry | an image on the web; a frame or caption in the TUI |

**CLI registry refactor.**
- Today each command builds its flags inside its handler. Commands move to a single table: `Command{Name, Usage, Flags func(*flag.FlagSet), Run}`.
- That table drives `jumpgate help`, the docs, and a test that every command and flag is referenced by at least one step or reference page.
- The exit-code and env-var tables are added the same way.

**Generated reference pages:** commands and flags, TUI keys, palette commands, networks and clients, exit codes, and environment variables. Steps link into them, and nobody writes these pages by hand.

**Scenes:**
- **TUI scenes** are named Go fixtures built on the golden-test machinery: the fake fleet and the fixed clock. `docsgen` renders each one to SVG with colour, and the output is deterministic.
- **App scenes** are rendered by Playwright, in a pinned container, against the web UI on a fake API. Animations are off, the clock is fixed and the fonts are bundled.
- **Desktop scenes** show OS chrome: the installer, the first-launch prompt and the tray. They are captured on real desktops: the Windows VM runner, the macOS job, or GitHub `windows-latest` and `macos-latest`. Windows containers have no desktop, so they can't produce these.
- `go test` checks that every `{{shot}}` names a registered scene. Producing the images is done by the CI jobs in §4.

## 3. Rendering

`docsgen` loads the content, resolves references, runs the checks and builds one model. The website and the TUI both render from it.

**Website** (`go run ./cmd/docsgen -out dist/docs`): static HTML from Go templates, in the landing palette.
- **Sidebar:** *Get started* (newcomer pathways), then *Operate* (operator pathways), then *Reference* (generated pages). Each pathway expands to its numbered steps.
- **Step page:** the title, the body, prev/next within the pathway, and a **Show previous steps** button.
- **URLs:** `/docs/<pathway>/<step>`. A step that appears in several pathways gets a page in each, so prev/next and the recap fit that pathway. `rel=canonical` points at the first pathway the step appears in.
- **No JavaScript needed:** the recap and its accordions are `<details>`. A small script adds "expand all" and remembers which items are open, wrapped in try/catch.
- **Search:** Pagefind over the built output.

**Recap, computed at build time:**
1. Take the transitive closure of `requires` across pathways, topologically ordered, with ties broken by pathway order.
2. Group it by pathway. The current pathway's group is open; earlier pathways are collapsed.
3. Each item shows its title (a link), `summary` and `recap`, plus a **Show image** accordion holding `image`, which loads lazily.

**TUI** (extends the v0.5.2 Help view behind `guide.Guide`):
- Navigation goes *Get started* / *Operate* / *Reference*, then a pathway, then a step.
- `r` toggles the recap pane, which is the same list in the same order. Enter on an item expands its recap.
- `{{shot tui …}}` opens the real frame. App and desktop shots show as a caption plus the jumpgate.app/docs URL.
- Everything fits 80x24 and works in ASCII mode and under NO_COLOR. The existing width and keys tests extend to the step model.

## 4. CI, publishing, errors

**CI:**
- `go test ./...` runs all docs checks: references, hashes, the dependency graph, scenes registered, pathway membership, the TUI's 80x24 fit and ASCII purity. It is merge-blocking.
- `docs-shots` (required) runs Playwright app scenes and the TUI SVG render in a pinned Linux container. It fails when a scene can't render.
- `docs-shots-desktop` runs on the Windows VM runner and the macOS job. It is required on tags, and on branches when platform packaging code changes.

**Publishing:**
- On each release tag, CI builds the site and attaches `docs-site.tar.gz` and its checksum to the GitHub release.
- The landing deploy downloads the latest release's `docs-site.tar.gz`, verifies it, unpacks it into `/docs` and deploys. jumpgate.app/docs therefore always matches the current download.
- Each binary's built-in Help matches its own version.
- Branches get a Pages preview (`docs-<branch>`) for review.

**Errors:** every failure names the file:line. A hash mismatch prints:
- the covered item;
- the old and new hash;
- the exact frontmatter line to change.

Re-approving means editing that line in the same diff, so a reviewer sees that the prose was checked.

**Tests:**
- `docsgen` golden tests on a fixture tree: recap order, cross-pathway grouping and canonical links.
- Negative tests: unknown reference, stale hash, missing `verified`, a dependency missing from the pathway, cycles, and a step in no pathway.
- The landing browser checks extend to `/docs`: the recap toggle, the image accordions, phone width and no horizontal scroll.

## 5. Initial pathways

- **Newcomer:** *Install*, *Your first box*, *One endpoint for your apps*.
- **Operator:** *Run jumpgate on a server* (see `2026-10-17-server-install-design.md`), *Migrate legacy names*, *Durable jobs in practice*, *Recover a box*.
- **Later**, once those features ship: *Validator deposit*, *Validator withdrawal*, *Approval signers*.

The seven topic files from v0.5.2 (`docs/guide/*.md`) are split into steps as part of this work, and `docs/guide` is removed.

## Rollout

1. CLI registry refactor, plus the exit-code and env tables.
2. The `docs` package: loader, frontmatter, references, the `docs:covers` hashing and the checks. Convert the v0.5.2 guide into steps.
3. The TUI Help view on the step model, with the recap pane.
4. The `docsgen` website, the recap, Pagefind and the reference pages.
5. TUI scene SVGs, then Playwright app scenes, then desktop scenes.
6. Release attachment, and the landing deploy pulling `/docs`.

## Out of scope

- Versioned docs per release on the website (only the latest is published; the binaries carry their own).
- Translations.
- Comments or feedback widgets.
- Executable doc tests that run commands against live boxes.

## Open questions

- Comment stripping in the hash means a behaviour change hidden in a comment-only edit can't happen, but neither can a change described only in comments. This is accepted.
- The desktop-scene runners are slow and sometimes offline (the Windows VM starts on demand). If they can't run, a tag build waits for them, as the "always block" rule requires.
