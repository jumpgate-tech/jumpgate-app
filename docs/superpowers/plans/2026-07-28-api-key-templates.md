# API-key templates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make an API-key template a URL the app can *fill in* rather than only discard, so valve stops being a special case and an operator with an Infura key gains two live Ethereum endpoints the feed currently throws away.

**Architecture:** `chainlist.Candidates` learns to substitute a placeholder when a key for it is known and to name the missing key when it is not. Keys live in `Config.ProviderKeys`, keyed by placeholder name, handled as secrets the way `AIKey` already is. `catalog.KnownSet` stops resolving anything and returns templates. Settings gains one row per placeholder something actually wants.

**Tech Stack:** Go 1.25 (stdlib), TypeScript (tsc strict + vite), no new dependencies.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-28-api-key-templates-design.md`. Read it first. It supersedes §5 of `2026-07-28-rpc-resilience-readout-design.md`.
- **A resolved template is still probed.** It is trusted because it answered, never because it resolved.
- **Keys are secrets: stored, never returned.** The existing convention is `settingsResponse.AIKeySet bool` (`internal/server/api.go:1435`) with the input reading "•••••••• (leave blank to keep)". Follow it exactly. This also means correcting `GET /api/gateways/{gid}/knownset/{chainId}`, which currently returns the key itself.
- **The zero-setup path must survive.** With no keys configured at all, `${VALVE_API_KEY}` resolves to `vk_demo` and the known set still adds successfully. This is a regression risk, not a given — test it.
- **Do not claim `vk_demo` is rate-limited or a shared quota.** Measured 2026-07-28: `x-valve-tier: FULL`, no rate-limit headers, 20/20 requests answered, bogus key `401`. Earlier copy saying otherwise was wrong and is being removed.
- Verify with `go build ./... && go test ./...`. UI changes also need `cd cmd/valve-node-app/web && npm run build`, then `go build ./...` so the rebuilt `dist/` still embeds. Commit the rebuilt `dist/`.
- Commit directly to `master`. Do NOT `git push`.
- End every commit message body with:
  `Claude-Session: https://claude.ai/code/session_01RxZPVou958Y4a43Hh6dEKf`

## Verified interfaces (read from source — do not rename or redefine)

**Go**
- `chainlist.Candidates(rpcs []string) []Endpoint` (`chainlist.go:349`) — exported, pure. **One production caller**: `chainlist.go:318`, inside `Discover`. Also 3 call sites in `chainlist_test.go`.
- `chainlist.isTemplated(raw string) bool` (`chainlist.go:387`) = `strings.Contains(raw, "${")`
- `chainlist.kindOf(raw string) Kind`; `Kind` is `KindHTTP` / `KindWS`
- `chainlist.Endpoint{URL string; Kind Kind; Status Status; Reason string; ...}`
- Statuses: `StatusPending`, `StatusLive`, `StatusRejected`, `StatusUnprobed`
- `chainlist.Discoverer` built by `New()`; `(*Discoverer).Discover(ctx, chainID) (Result, error)`
- `internal/server/chainlist.go:81` — `s.newChainlist().Discover(ctx, chainID)`
- `catalog.KnownSet(chainID int, key string) []KnownEndpoint`, `catalog.KnownSetSize(chainID int) int`, `catalog.KnownSetProviders(chainID int) int`, `catalog.DefaultValveKey = "vk_demo"`
- `catalog.KnownEndpoint{URL, Provider string; WebSocket, Archive bool}`
- `config.Config.ValveKeys map[int]string` `json:"valveKeys,omitempty"` — being replaced
- `config.Config.AIKey string`; `(*Config).migrate()` runs on every `Load()` and is written back by the next `Save()`
- `internal/server/api.go:1433` `settingsResponse{AIProvider, AIKeySet, RefRPCBase}`; `:1462` `settingsRequest{AIProvider, AIKey *string, RefRPCBase *string}`
- `internal/server/gateways.go` — `knownSetResponse{Endpoints, Key, UsingDefaultKey}`, `handleKnownSet`

**TypeScript**
- `cmd/valve-node-app/web/src/settings.ts` — `#ai-key` password input, `settings.aiKeySet`, `data-action="clear-key"`, submit builds a body and only sends `aiKey` when the field was touched
- `api.ts` — its `request` helper; `KnownSetResponse{endpoints, key, usingDefaultKey}`
- `rpc.ts` — `offerKnownSet`, `addExternalUpstreams`

## File Structure

| File | Responsibility |
|---|---|
| `internal/chainlist/template.go` (create) | Placeholder detection by name, and substitution. Pure. |
| `internal/chainlist/template_test.go` (create) | Naming, substitution, and the no-key case. |
| `internal/chainlist/chainlist.go` (modify) | `Candidates` takes keys; rejection names the placeholder. |
| `internal/config/config.go` (modify) | `ProviderKeys`; migration off `ValveKeys`. |
| `internal/server/api.go` (modify) | Settings expose which placeholders are set, never values. |
| `internal/server/gateways.go` (modify) | Stop returning the key; resolve at the one seam. |
| `internal/catalog/knownset.go` (modify) | Return templates; drop the key argument. |
| `cmd/valve-node-app/web/src/settings.ts`, `api.ts` (modify) | Provider-keys section. |

---

### Task 1: Placeholders have names, and a name can be filled

**Files:**
- Create: `internal/chainlist/template.go`
- Create: `internal/chainlist/template_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func PlaceholderName(raw string) string` (empty when untemplated), `func Resolve(raw string, keys map[string]string) (string, bool)`.

- [ ] **Step 1: Write the failing test**

Create `internal/chainlist/template_test.go`:

```go
package chainlist

import "testing"

// The feed writes provider slots as ${NAME}. The NAME is the identity: it is
// what the operator has to go and obtain, so it is what the app stores a key
// under and what a rejection has to say out loud.
func TestPlaceholderName(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"https://mainnet.infura.io/v3/${INFURA_API_KEY}", "INFURA_API_KEY"},
		{"wss://mainnet.infura.io/ws/v3/${INFURA_API_KEY}", "INFURA_API_KEY"},
		{"https://one.valve.city/rpc/${VALVE_API_KEY}/evm/1", "VALVE_API_KEY"},
		{"https://eth.drpc.org", ""},
		{"", ""},
		// Malformed: an opening brace with no close is not a usable name.
		{"https://x.example/${UNCLOSED", ""},
	}
	for _, tt := range tests {
		if got := PlaceholderName(tt.raw); got != tt.want {
			t.Errorf("PlaceholderName(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestResolve(t *testing.T) {
	keys := map[string]string{"INFURA_API_KEY": "abc123"}

	got, ok := Resolve("https://mainnet.infura.io/v3/${INFURA_API_KEY}", keys)
	if !ok || got != "https://mainnet.infura.io/v3/abc123" {
		t.Errorf("resolved = %q, %v", got, ok)
	}

	// No key for this placeholder: the URL is unusable and must NOT come back
	// half-substituted, which would be a live-looking URL that 401s forever.
	if got, ok := Resolve("https://x.example/${ALCHEMY_API_KEY}", keys); ok {
		t.Errorf("want unresolved, got %q", got)
	}

	// An untemplated URL resolves to itself, so callers need no special case.
	if got, ok := Resolve("https://eth.drpc.org", keys); !ok || got != "https://eth.drpc.org" {
		t.Errorf("plain URL: got %q, %v", got, ok)
	}

	// An empty stored value is not a key. Substituting it yields a URL with an
	// empty path segment that looks configured and answers nothing.
	if _, ok := Resolve("https://x.example/${K}", map[string]string{"K": ""}); ok {
		t.Error("an empty key must not resolve")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/chainlist/ -run 'TestPlaceholderName|TestResolve' -v`
Expected: FAIL — `undefined: PlaceholderName`, `undefined: Resolve`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/chainlist/template.go`:

```go
package chainlist

import (
	"regexp"
	"strings"
)

// placeholderRe matches the feed's shell-style provider slot. isTemplated stays
// deliberately broader (a bare "${" counts), because a malformed placeholder is
// still not a usable endpoint and must not fall through to being probed.
var placeholderRe = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)

// PlaceholderName returns the name inside a URL's ${...} slot, or "" when there
// is none. The name is the identity an operator recognises — it is what they go
// and obtain, what the app stores a key under, and what a rejection names.
func PlaceholderName(raw string) string {
	m := placeholderRe.FindStringSubmatch(raw)
	if m == nil {
		return ""
	}
	return m[1]
}

// Resolve substitutes every placeholder in raw from keys, reporting whether the
// result is usable. An untemplated URL resolves to itself so callers need no
// special case.
//
// A missing OR empty key fails rather than substituting nothing: a URL with an
// empty path segment looks configured and answers nothing, which is worse than
// a rejection that says which key is missing.
func Resolve(raw string, keys map[string]string) (string, bool) {
	if !isTemplated(raw) {
		return raw, true
	}
	out := raw
	for _, m := range placeholderRe.FindAllStringSubmatch(raw, -1) {
		v := keys[m[1]]
		if strings.TrimSpace(v) == "" {
			return "", false
		}
		out = strings.ReplaceAll(out, m[0], v)
	}
	// A leftover "${" means a malformed slot the regex could not name.
	if isTemplated(out) {
		return "", false
	}
	return out, true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/chainlist/ -run 'TestPlaceholderName|TestResolve' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/chainlist/template.go internal/chainlist/template_test.go
git commit -m "feat(chainlist): a placeholder has a name, and a name can be filled"
```

---

### Task 2: Candidates fills what it can and names what it cannot

**Files:**
- Modify: `internal/chainlist/chainlist.go` (`Candidates`, its one caller at `:318`, `Discoverer`)
- Modify: `internal/chainlist/chainlist_test.go` (3 `Candidates(` call sites)

**Interfaces:**
- Consumes: `PlaceholderName`, `Resolve` from Task 1.
- Produces: `Candidates(rpcs []string, keys map[string]string) []Endpoint`; `Discoverer.Keys map[string]string`.

- [ ] **Step 1: Write the failing test**

Append to `internal/chainlist/chainlist_test.go`:

```go
// A provider slot the app can fill is an endpoint, not a dead end — and it is
// then probed like any other, because it is trusted for answering, never for
// resolving.
func TestCandidatesResolvesTemplatesItHasKeysFor(t *testing.T) {
	keys := map[string]string{"INFURA_API_KEY": "abc123"}
	got := Candidates([]string{
		"https://mainnet.infura.io/v3/${INFURA_API_KEY}",
		"https://x.example/${ALCHEMY_API_KEY}",
	}, keys)

	if len(got) != 2 {
		t.Fatalf("want 2 endpoints, got %d", len(got))
	}
	if got[0].Status != StatusPending {
		t.Errorf("a resolved template must be probed, got %s (%s)", got[0].Status, got[0].Reason)
	}
	if got[0].URL != "https://mainnet.infura.io/v3/abc123" {
		t.Errorf("resolved URL = %q", got[0].URL)
	}
	if got[1].Status != StatusRejected {
		t.Errorf("an unresolvable template stays rejected, got %s", got[1].Status)
	}
	// The reason has to name the key, or the operator knows something is
	// missing without knowing what to go and get.
	if !strings.Contains(got[1].Reason, "ALCHEMY_API_KEY") {
		t.Errorf("reason must name the placeholder: %q", got[1].Reason)
	}
}
```

Add `strings` to the test file's imports if absent.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/chainlist/ -run TestCandidatesResolves -v`
Expected: FAIL — `Candidates` takes one argument.

- [ ] **Step 3: Write minimal implementation**

Change the signature and the templated branch in `chainlist.go`:

```go
func Candidates(rpcs []string, keys map[string]string) []Endpoint {
```

and replace the `case isTemplated(raw):` branch with:

```go
		case isTemplated(raw):
			// A provider slot. If a key for it is known, fill it in and let it
			// be probed like anything else — resolving is not evidence, only
			// answering is. Otherwise reject as before, but name the key: the
			// old message said an account was required without saying which,
			// which is the difference between a dead end and a next step.
			name := PlaceholderName(raw)
			if resolved, ok := Resolve(raw, keys); ok {
				kind := kindOf(resolved)
				if kind == "" {
					ep.Status = StatusRejected
					ep.Reason = "unsupported URL scheme (want http, https, ws or wss)"
					break
				}
				ep.URL = resolved
				ep.Kind = kind
				ep.Status = StatusPending
				break
			}
			ep.Kind = kindOf(raw)
			ep.Status = StatusRejected
			if name != "" {
				ep.Reason = "needs " + name + " — add it in Settings"
			} else {
				ep.Reason = "API-key template (contains ${...}); requires a provider account"
			}
```

Add a `Keys map[string]string` field to `Discoverer` with a doc comment saying it is provider keys by placeholder name, and pass it at `chainlist.go:318`:

```go
	res.Endpoints = d.probeAll(ctx, chainID, Candidates(rpcs, d.Keys))
```

Update the 3 existing `Candidates(` call sites in `chainlist_test.go` to pass `nil`. **Read each first** — if any asserts the old "requires a provider account" reason, that test's subject is the rejection message and it should now assert the named form.

**Do NOT touch `internal/server/chainlist.go` in this task.** `Discoverer.Keys` exists but nothing populates it yet, because `Config.ProviderKeys` does not exist until Task 3 — which owns that wiring. Leaving it nil here is correct and harmless: `Candidates(rpcs, nil)` rejects every template exactly as today, so this task changes no behaviour on its own. It is the mechanism; Task 3 turns it on.

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./internal/chainlist/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/chainlist/
git commit -m "feat(chainlist): fill the provider slots we hold keys for"
```

---

### Task 3: Keys live per placeholder, and are never handed back

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/server/api.go` (settings), `internal/server/chainlist.go` (wire `Keys`), `internal/server/gateways.go` (stop returning the key)
- Modify: `internal/config/config_test.go` or `persistence_test.go` (migration), `internal/server/gateways_test.go`

**Interfaces:**
- Consumes: Task 1 and 2.
- Produces: `Config.ProviderKeys map[string]string` `json:"providerKeys,omitempty"`; `settingsResponse.ProviderKeysSet []string`; `settingsRequest.ProviderKeys map[string]string`.

- [ ] **Step 1: Write the failing test**

Append to the config test file that already covers `migrate()` (read `persistence_test.go` first and follow its style):

```go
// valveKeys was per chain because valve's key sits in a URL path. That is wrong
// as a general rule — a provider key is an account, not a chain — so it
// collapses to one entry under the placeholder name.
func TestMigrateCollapsesValveKeysToAProviderKey(t *testing.T) {
	c := Config{ValveKeys: map[int]string{1: "vk_mine", 369: "vk_mine"}}

	c.migrate()

	if got := c.ProviderKeys["VALVE_API_KEY"]; got != "vk_mine" {
		t.Errorf("VALVE_API_KEY = %q, want vk_mine", got)
	}
	if c.ValveKeys != nil {
		t.Errorf("valveKeys must be cleared once migrated, got %+v", c.ValveKeys)
	}

	// Disagreeing keys: the first by chain id wins and the rest are REPORTED,
	// never silently dropped — the same stance the orphan record takes.
	d := Config{ValveKeys: map[int]string{1: "vk_a", 369: "vk_b"}}
	d.migrate()
	if d.ProviderKeys["VALVE_API_KEY"] != "vk_a" {
		t.Errorf("lowest chain id wins, got %q", d.ProviderKeys["VALVE_API_KEY"])
	}
	if len(d.Notices) == 0 {
		t.Error("a discarded key must be reported to the operator, not dropped in silence")
	}
}
```

`Notices` is a new field. **I checked: there is no existing notice mechanism** — `Config.Orphans` is the closest precedent, and it is purpose-built for one thing rather than general. So add `Notices`, and keep it as narrow as `Orphans` is: a migration message the operator needs to see, not a general logging channel.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestMigrateCollapses -v`
Expected: FAIL — `ProviderKeys` undefined.

- [ ] **Step 3: Write minimal implementation**

In `config.go`, replace the `ValveKeys` field's role. Keep the field for migration input, marked deprecated, and add:

```go
	// ProviderKeys are API keys by PLACEHOLDER NAME — "VALVE_API_KEY",
	// "INFURA_API_KEY" — matching the ${NAME} slots the chain feed uses. Keyed
	// by placeholder rather than by chain because a provider key is an account,
	// not a chain.
	//
	// Secrets: stored here, never returned by the API. See settingsResponse,
	// which reports which placeholders are set and never their values.
	ProviderKeys map[string]string `json:"providerKeys,omitempty"`

	// Notices are one-off messages from a migration that the operator needs to
	// see, e.g. a key discarded when per-chain keys collapsed. Not persisted as
	// advice — cleared once shown.
	Notices []string `json:"notices,omitempty"`
```

At the end of `migrate()`, collapse `ValveKeys`: take the lowest chain id's value as `ProviderKeys["VALVE_API_KEY"]` when that placeholder has none, append a notice naming each discarded differing value, then set `ValveKeys = nil`.

In `internal/server/api.go`, add to `settingsResponse`:

```go
	// ProviderKeysSet names the placeholders that have a key, never the keys.
	// Same rule as AIKeySet.
	ProviderKeysSet []string `json:"providerKeysSet"`
```

populate it in `settingsResponseFrom` (sorted, for a stable UI), and accept `ProviderKeys map[string]string` in `settingsRequest`, merging non-empty values and deleting on empty string — mirroring how `AIKey` is only applied when present.

In `internal/server/chainlist.go`, set `Keys: cfg.ProviderKeys` on the Discoverer, defaulting `VALVE_API_KEY` to `catalog.DefaultValveKey` when unset so the zero-setup path holds.

In `internal/server/gateways.go`, **remove `Key` from `knownSetResponse`** and keep `UsingDefaultKey`. Update `handleKnownSet` and the test that asserts on `key`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./...`
Expected: PASS. Any test asserting `knownSetResponse.Key` now encodes the leak — update it to assert the key is absent.

- [ ] **Step 5: Commit**

```bash
git add internal/config internal/server
git commit -m "feat(config): keys by placeholder, and never handed back"
```

---

### Task 4: The set returns templates, resolved at one seam

**Files:**
- Modify: `internal/catalog/knownset.go`, `internal/catalog/knownset_test.go`
- Modify: `internal/server/gateways.go`

**Interfaces:**
- Consumes: Tasks 1–3.
- Produces: `catalog.KnownSet(chainID int) []KnownEndpoint` — **the key argument is gone**; `catalog.ValveKeyPlaceholder = "VALVE_API_KEY"`.

- [ ] **Step 1: Write the failing test**

Replace `TestKnownSetSubstitutesTheValveKey` in `knownset_test.go` with:

```go
// The set no longer resolves anything. It returns a template like any other
// API-key endpoint, and resolution happens at the one seam every endpoint
// crosses — so the set and the feed cannot drift.
func TestKnownSetReturnsAnUnresolvedTemplate(t *testing.T) {
	got := KnownSet(1)[0].URL
	want := "https://one.valve.city/rpc/${" + ValveKeyPlaceholder + "}/evm/1"
	if got != want {
		t.Errorf("valve URL: want %q, got %q", want, got)
	}
}
```

Update every other `KnownSet(` call in that file to drop the key argument.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestKnownSet -v`
Expected: FAIL — too many arguments.

- [ ] **Step 3: Write minimal implementation**

In `knownset.go`: add `const ValveKeyPlaceholder = "VALVE_API_KEY"`, change the templates to embed `${VALVE_API_KEY}`, drop the `key` parameter from `KnownSet`, and delete the substitution. Keep `DefaultValveKey = "vk_demo"` — it is now the *default value* for that placeholder, and say so in its comment.

In `handleKnownSet`, resolve each entry with `chainlist.Resolve(e.URL, keys)` where `keys` is `cfg.ProviderKeys` with `VALVE_API_KEY` defaulted to `catalog.DefaultValveKey`. An entry that will not resolve is returned with its placeholder named rather than dropped, so the modal can say which key it wants.

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./...`
Expected: PASS.

**Then verify the zero-setup path by hand**, because it is the regression this task risks: with no `providerKeys` in config, `GET /api/gateways/default/knownset/1` must return valve's entry resolved to `vk_demo`, not a template and not a rejection.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog internal/server
git commit -m "feat(catalog): the set returns templates, not resolved URLs"
```

---

### Task 5: One place to enter a key

**Files:**
- Modify: `cmd/valve-node-app/web/src/api.ts`, `src/settings.ts` (+ rebuilt `dist/`)

**Interfaces:**
- Consumes: `settingsResponse.providerKeysSet`, `settingsRequest.providerKeys` from Task 3.
- Produces: no new exports.

- [ ] **Step 1: Add the section**

In `api.ts`, add `providerKeysSet: string[]` to the settings response interface and `providerKeys?: Record<string, string>` to the update request.

In `settings.ts`, render one row per placeholder in `providerKeysSet` **plus** any the app knows it wants — at minimum `VALVE_API_KEY`. Each row follows the `#ai-key` pattern exactly: a password input whose placeholder reads `•••••••• (leave blank to keep)` when set and `no key set` when not, and a clear action. On submit, send only the fields the operator touched this session, exactly as `aiKey` is handled.

Copy rules: a row says what the key is for and that a key is optional where a default ships. **Do not describe `vk_demo` as limited, shared, or a demo quota** — measured, it is a real credential on the full tier.

- [ ] **Step 2: Build**

Run: `cd cmd/valve-node-app/web && npm run build`
Expected: tsc strict clean.
Then: `cd - && go build ./... && go test ./...`
Expected: PASS, `dist/` re-embeds.

- [ ] **Step 3: Drive it**

Start the app. In Settings, confirm `VALVE_API_KEY` shows as set-by-default or clearly optional. Then the check that proves the whole plan: **`evm:1`'s discover list should show two Infura endpoints rejected with `needs INFURA_API_KEY`.** Enter any non-empty string as an Infura key, reopen the list, and confirm those two entries are now probed candidates rather than rejections. (A junk key will fail its probe — that is correct and is the point: resolving is not evidence.)

Restore any config you changed.

- [ ] **Step 4: Commit**

```bash
git add cmd/valve-node-app/web/src cmd/valve-node-app/web/dist
git commit -m "feat(settings): somewhere to put a provider key"
```

---

## Final verification

- [ ] `go build ./... && go test ./...` green; `npm run build` clean under tsc strict.
- [ ] **Zero-setup path:** with `providerKeys` absent from config entirely, adding the known set to a chain still works and valve's URL lands as `.../vk_demo/...` in the rendered `erpc.yaml`.
- [ ] **The check the whole plan exists for:** `evm:1` reports two endpoints rejected with `needs INFURA_API_KEY`; supplying a key turns them into probed candidates. Two entries the app used to discard become reachable through one operator action.
- [ ] **No key is ever returned by the API.** `GET /api/settings` reports `providerKeysSet` names only, and `GET /api/gateways/{gid}/knownset/{chainId}` no longer carries `key`. Grep both responses for a stored key value and find nothing.
