# RPC resilience readout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a chain's redundancy visible on the RPC page, and make fixing it one click via a hardcoded, measured set of four endpoints per chain — with valve's endpoint keyed per chain.

**Architecture:** A new `internal/catalog/knownset.go` holds the sets as data plus one function that resolves valve's URL against a per-chain key. `Config.ValveKeys` stores those keys. The gateway renderer stops marking an upstream a fallback when nothing local exists for it to fall back to. The RPC page then leads with a per-chain redundancy count, capability tags and one gap sentence, and its five warning surfaces collapse into that row.

**Tech Stack:** Go 1.25 (stdlib), TypeScript (tsc strict + vite), no new dependencies.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-28-rpc-resilience-readout-design.md`. Read it first — it carries the measurements the set's ordering depends on.
- **valve's URL shape is `https://one.valve.city/rpc/<key>/evm/<chainId>`.** The key is in the path, not a header. Default key is the literal `vk_demo`.
- **Set order is deliberate and measured — do not re-sort alphabetically or by latency.** eRPC prefers earlier upstreams; the order encodes capability. evm:1 is valve, drpc, publicnode, merkle. evm:369 is valve, publicnode, g4mm4, official.
- Colour in the UI means state and nothing else: teal answering, amber degraded-or-single-path, red down. No decorative accents.
- Verify with `go build ./... && go test ./...`. UI changes also need `cd cmd/valve-node-app/web && npm run build`, then `go build ./...` so the rebuilt `dist/` still embeds. Commit the rebuilt `dist/`.
- Commit directly to `master`. Do NOT `git push`.
- End every commit message body with:
  `Claude-Session: https://claude.ai/code/session_01RxZPVou958Y4a43Hh6dEKf`

## Verified interfaces (these exist — do not redefine or rename them)

**Go**
- `chainlist.Vendored(chainID int) ([]string, bool)` — returns a copy
- `catalog.DefaultUpstreams(chainID int) []string`
- `catalog.GatewayUpstream{ID, Kind, TargetID, Endpoint string; Local, RecentOnly bool}`, method `KindOrDefault() string`
- Constants `catalog.UpstreamExternal`, `UpstreamManagedNode`, `UpstreamManagedDevnet`
- `catalog.GatewayNetwork{ChainID int; Upstreams []GatewayUpstream}`
- `internal/catalog/gateway.go:249` — `gatewayUpstreamVars` has field `Fallback bool`
- `internal/catalog/gateway.go:394` — currently `Fallback: !u.Local`
- `internal/catalog/gateway.go:228-234` — the `{{- if .Fallback}}` block emitting `tags: [tier:fallback]` and `scoreMultipliers: [{overall: 0.2}]`
- `config.Config` fields carry json tags, e.g. `RefRPCBase string \`json:"refRpcBase"\``

**TypeScript**
- `api.GatewayUpstream{ID: string; Kind?: UpstreamKind; TargetID?: string; Endpoint: string; Local: boolean; RecentOnly: boolean}`
- `api.GatewayNetwork`, `api.GatewayConfig`, `api.GatewayView`
- `api.ChainlistEndpoint{url, kind: "http"|"ws", status: "live"|"unprobed"|"rejected", chainId?, latencyMs?, reason?}`
- `api.ChainlistResult{chainId, source, fetchError?, endpoints: ChainlistEndpoint[] | null, live}` — **`endpoints` is nullable**
- `api.discoverEndpoints(chainId: number): Promise<ChainlistResult>`
- `rpc.ts`: `gatewayOf(gid)`, `storedConfig(gw)`, `saveConfig(gid, cfg, note?)`, `addExternalUpstreams(gid, chainId, urls, recentOnly?)`
- `ui.ts`: `openModal(innerHtml, onModalAction)`, `modalBody()`, `escapeHtml()`
- Delegated clicks use `data-action` + the file's `onAction` switch.

## File Structure

| File | Responsibility |
|---|---|
| `internal/catalog/knownset.go` (create) | The sets as data; valve URL resolution against a key. Pure. |
| `internal/catalog/knownset_test.go` (create) | Set contents, ordering, key substitution. |
| `internal/config/config.go` (modify) | `ValveKeys map[int]string`. |
| `internal/server/gateways.go` (modify) | `GET /api/gateways/{gid}/knownset/{chainId}`; key read/write. |
| `internal/catalog/gateway.go` (modify) | Fallback only when the network has a local upstream. |
| `cmd/valve-node-app/web/src/api.ts` (modify) | Typed known-set call. |
| `cmd/valve-node-app/web/src/rpc.ts` (modify) | Set offer, redundancy row, collapsed warnings. |

---

### Task 1: The known set as data

**Files:**
- Create: `internal/catalog/knownset.go`
- Create: `internal/catalog/knownset_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `func KnownSet(chainID int, key string) []KnownEndpoint`, `type KnownEndpoint struct{ URL, Provider string; WebSocket, Archive bool }`, `const DefaultValveKey = "vk_demo"`.

- [ ] **Step 1: Write the failing test**

Create `internal/catalog/knownset_test.go`:

```go
package catalog

import "testing"

// The order is the whole point: eRPC prefers earlier upstreams, so the set is
// sorted by what each endpoint can do, not by name or by ping. These were
// measured on 2026-07-28 — see the design doc.
func TestKnownSetOrderAndContents(t *testing.T) {
	// Four PROVIDERS is the promise. A provider offering both http and wss
	// contributes two entries, because eRPC reads the capability off the
	// scheme — so entry count and provider count are different numbers and the
	// test must not conflate them.
	if got := KnownSetProviders(1); got != 4 {
		t.Fatalf("evm:1 is four providers, got %d", got)
	}
	eth := KnownSet(1, DefaultValveKey)
	var order []string
	for _, e := range eth {
		if len(order) == 0 || order[len(order)-1] != e.Provider {
			order = append(order, e.Provider)
		}
		if e.URL == "" {
			t.Errorf("every entry needs a URL; %q has none", e.Provider)
		}
	}
	wantProviders := []string{"valve", "drpc", "publicnode", "merkle"}
	for i, want := range wantProviders {
		if i >= len(order) || order[i] != want {
			t.Errorf("evm:1 provider %d: want %q, got %v — order encodes capability", i, want, order)
		}
	}

	if got := KnownSetProviders(369); got != 4 {
		t.Fatalf("evm:369 is four providers, got %d", got)
	}
	pls := KnownSet(369, DefaultValveKey)
	// The official endpoint measured 1700ms against 197-274ms for the rest, so
	// it is kept but never preferred.
	if pls[len(pls)-1].Provider != "official" {
		t.Errorf("evm:369 must end on the official endpoint, got %q", pls[len(pls)-1].Provider)
	}

	if got := KnownSet(999999, DefaultValveKey); got != nil {
		t.Errorf("an unknown chain has no set, got %+v", got)
	}
}

// Neither capability may rest on a single provider: publicnode is not archive
// on Ethereum and merkle has no WebSocket, so a set carrying only one of them
// loses a capability outright.
func TestKnownSetCoversBothCapabilities(t *testing.T) {
	var ws, archive int
	for _, e := range KnownSet(1, DefaultValveKey) {
		if e.WebSocket {
			ws++
		}
		if e.Archive {
			archive++
		}
	}
	if ws < 2 {
		t.Errorf("evm:1 needs more than one WebSocket source, got %d", ws)
	}
	if archive < 2 {
		t.Errorf("evm:1 needs more than one archive source, got %d", archive)
	}
}

func TestKnownSetSubstitutesTheValveKey(t *testing.T) {
	got := KnownSet(1, "vk_mine")[0].URL
	want := "https://one.valve.city/rpc/vk_mine/evm/1"
	if got != want {
		t.Errorf("valve URL: want %q, got %q", want, got)
	}

	// An empty key must not produce a URL with an empty path segment — that
	// would 404 on every call while looking configured.
	if got := KnownSet(1, "")[0].URL; got != "https://one.valve.city/rpc/vk_demo/evm/1" {
		t.Errorf("an empty key falls back to the demo key, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestKnownSet -v`
Expected: FAIL — `undefined: KnownSet`, `undefined: DefaultValveKey`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/catalog/knownset.go`:

```go
package catalog

import "fmt"

// DefaultValveKey is the shared demo key, so the set works with no setup at
// all. It is a shared quota: if it runs dry, valve's endpoint becomes the
// least reliable entry in a set that lists it first, which is why the UI says
// plainly that a free key of the operator's own removes the limit.
const DefaultValveKey = "vk_demo"

// valveURLTemplate is valve's unified endpoint. The key is a PATH segment, not
// a header, so it has to be substituted per chain rather than set once on a
// client.
const valveURLTemplate = "https://one.valve.city/rpc/%s/evm/%d"

// KnownEndpoint is one entry in a chain's known set, with the two capabilities
// that decide whether it is a good gateway upstream. Both were measured
// (2026-07-28), not read off a provider's docs: Archive means eth_getBalance at
// block 1,000,000 answered rather than erroring, and WebSocket means the
// upgrade actually completed.
type KnownEndpoint struct {
	URL      string
	Provider string
	// WebSocket is true for a wss:// entry. eRPC infers the capability from the
	// scheme, so a provider offering both appears as two entries.
	WebSocket bool
	Archive   bool
}

// knownSets is the hardcoded per-chain set, IN PREFERENCE ORDER. eRPC scores
// upstreams and prefers earlier ones, so this order is load-bearing: it is
// sorted by measured capability, never by name or latency alone.
//
// evm:1 carries two archive sources and three WebSocket sources so neither
// capability rests on one provider — publicnode is NOT archive on Ethereum and
// merkle has no WebSocket, so they are complements rather than alternatives.
//
// evm:369 keeps the official endpoint but never prefers it: it measured
// 1700ms against 197-274ms for the others.
//
// The valve entry's URL is filled in per call, since it carries the key.
var knownSets = map[int][]KnownEndpoint{
	1: {
		{Provider: "valve", WebSocket: true, Archive: true},
		{URL: "https://eth.drpc.org", Provider: "drpc", Archive: true},
		{URL: "wss://eth.drpc.org", Provider: "drpc", WebSocket: true, Archive: true},
		{URL: "https://ethereum-rpc.publicnode.com", Provider: "publicnode"},
		{URL: "wss://ethereum-rpc.publicnode.com", Provider: "publicnode", WebSocket: true},
		{URL: "https://eth.merkle.io", Provider: "merkle", Archive: true},
	},
	369: {
		{Provider: "valve", WebSocket: true},
		{URL: "https://pulsechain-rpc.publicnode.com", Provider: "publicnode"},
		{URL: "wss://pulsechain-rpc.publicnode.com", Provider: "publicnode", WebSocket: true},
		{URL: "https://rpc-pulsechain.g4mm4.io", Provider: "g4mm4"},
		{URL: "https://rpc.pulsechain.com", Provider: "official"},
	},
	943: {
		{Provider: "valve", WebSocket: true},
		{URL: "https://pulsechain-testnet-rpc.publicnode.com", Provider: "publicnode"},
		{URL: "wss://pulsechain-testnet-rpc.publicnode.com", Provider: "publicnode", WebSocket: true},
		{URL: "https://rpc-testnet-pulsechain.g4mm4.io", Provider: "g4mm4"},
		{URL: "https://rpc.v4.testnet.pulsechain.com", Provider: "official"},
	},
}

// KnownSet returns a chain's set with valve's URL resolved against key, or nil
// for a chain with no set. The result is FOUR PROVIDERS; a provider offering
// both schemes contributes two entries, because eRPC reads the capability off
// the scheme.
func KnownSet(chainID int, key string) []KnownEndpoint {
	src, ok := knownSets[chainID]
	if !ok {
		return nil
	}
	if key == "" {
		key = DefaultValveKey
	}
	out := make([]KnownEndpoint, 0, len(src))
	for _, e := range src {
		if e.Provider == "valve" && e.URL == "" {
			e.URL = fmt.Sprintf(valveURLTemplate, key, chainID)
		}
		out = append(out, e)
	}
	return out
}

// KnownSetProviders is the distinct provider count for a chain, which is what
// "four endpoints" means to an operator — the scheme split is an eRPC detail.
func KnownSetProviders(chainID int) int {
	seen := map[string]bool{}
	for _, e := range knownSets[chainID] {
		seen[e.Provider] = true
	}
	return len(seen)
}
```

**Why the counts differ, since it will look like a bug:** evm:1 holds six *entries* for four *providers*, because drpc and publicnode each contribute an http and a wss entry. The operator was promised four providers; eRPC needs the scheme split. `KnownSetProviders` is what the UI and the tests count.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run TestKnownSet -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/knownset.go internal/catalog/knownset_test.go
git commit -m "feat(catalog): a measured known set per chain"
```

---

### Task 2: Per-chain valve key

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/server/gateways.go`
- Modify: `internal/server/gateways_test.go`

**Interfaces:**
- Consumes: `catalog.KnownSet`, `catalog.DefaultValveKey` from Task 1.
- Produces: `Config.ValveKeys map[int]string` (json `valveKeys,omitempty`); `GET /api/gateways/{gid}/knownset/{chainId}` returning `{endpoints:[{url,provider,websocket,archive,alreadyAdded}], key, usingDefaultKey}`.

- [ ] **Step 1: Write the failing test**

Append to `internal/server/gateways_test.go`, reusing the file's real helpers (`gatewayServer`, `addGateway`, `a.do`, `decode[T]`):

```go
// The set is offered with what is already configured marked, so the count the
// operator sees before clicking matches what actually lands.
func TestKnownSetMarksWhatIsAlreadyConfigured(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "public-1-1", Kind: catalog.UpstreamExternal, Endpoint: "https://eth.drpc.org"},
		}}},
	})

	body := decode[knownSetResponse](t, a.do(t, "GET", "/api/gateways/default/knownset/1", nil))

	if len(body.Endpoints) == 0 {
		t.Fatal("the set must be offered for chain 1")
	}
	var marked int
	for _, e := range body.Endpoints {
		if e.URL == "https://eth.drpc.org" && e.AlreadyAdded {
			marked++
		}
	}
	if marked != 1 {
		t.Errorf("the configured endpoint must come back marked, got %d marks: %+v", marked, body.Endpoints)
	}
	if !body.UsingDefaultKey {
		t.Error("with no key stored, the set must report it is using the shared demo key")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run TestKnownSet -v`
Expected: FAIL — `undefined: knownSetResponse`, 404 on the route.

- [ ] **Step 3: Write minimal implementation**

In `internal/config/config.go`, add to `Config` beside `RefRPCBase`:

```go
	// ValveKeys is the valve API key per chain id. The key is a PATH segment of
	// valve's unified endpoint, and a key's entitlements are a per-chain matter,
	// so one global key could express neither "different keys per chain" nor
	// "valve on this chain but not that one". Absent, catalog.DefaultValveKey
	// (the shared demo key) is used.
	ValveKeys map[int]string `json:"valveKeys,omitempty"`
```

In `internal/server/gateways.go`, register beside the other gateway routes:

```go
	mux.HandleFunc("GET /api/gateways/{gid}/knownset/{chainId}", s.handleKnownSet)
```

and add:

```go
type knownSetEndpoint struct {
	URL          string `json:"url"`
	Provider     string `json:"provider"`
	WebSocket    bool   `json:"websocket"`
	Archive      bool   `json:"archive"`
	AlreadyAdded bool   `json:"alreadyAdded"`
}

type knownSetResponse struct {
	Endpoints       []knownSetEndpoint `json:"endpoints"`
	Key             string             `json:"key"`
	UsingDefaultKey bool               `json:"usingDefaultKey"`
}

// handleKnownSet offers the hardcoded set for one chain, marking what this
// gateway already has so the count the operator sees is the count that lands.
func (s *Server) handleKnownSet(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")
	chainID, err := strconv.Atoi(r.PathValue("chainId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "chain id must be a number")
		return
	}
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	gw, ok := cfg.FindGateway(gid)
	if !ok {
		writeErrorDetail(w, http.StatusNotFound, "no gateway "+gid, "", codeGatewayNotFound)
		return
	}

	have := map[string]bool{}
	for _, n := range gw.Config.Networks {
		if n.ChainID != chainID {
			continue
		}
		for _, u := range n.Upstreams {
			have[u.Endpoint] = true
		}
	}

	key := cfg.ValveKeys[chainID]
	out := knownSetResponse{Key: key, UsingDefaultKey: key == ""}
	if out.UsingDefaultKey {
		out.Key = catalog.DefaultValveKey
	}
	for _, e := range catalog.KnownSet(chainID, key) {
		out.Endpoints = append(out.Endpoints, knownSetEndpoint{
			URL: e.URL, Provider: e.Provider, WebSocket: e.WebSocket,
			Archive: e.Archive, AlreadyAdded: have[e.URL],
		})
	}
	writeJSON(w, http.StatusOK, out)
}
```

Use the file's real config accessor — read the neighbouring handlers first; `handleGatewayList` shows how config is read and `FindGateway` is used. Match it rather than adding a new accessor. Add `strconv` to imports if absent.

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./internal/server/ ./internal/config/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/server/gateways.go internal/server/gateways_test.go
git commit -m "feat(server): offer the known set, keyed per chain"
```

---

### Task 3: Public upstreams are not fallbacks when they are all you have

**Files:**
- Modify: `internal/catalog/gateway.go:394`
- Modify: `internal/catalog/gateway_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: no signature change; only what `RenderGatewayConfig` emits.

- [ ] **Step 1: Write the failing test**

Append to `internal/catalog/gateway_test.go`:

```go
// tier:fallback at 0.2 tells eRPC to avoid an upstream. That is right when a
// local node serves the chain and wrong when the "fallback" is the only path
// there is — the gateway would be de-prioritising the one thing that can answer.
func TestRenderGatewayConfig_PublicOnlyChainIsNotAFallback(t *testing.T) {
	g := GatewayConfig{Port: 4000, Networks: []GatewayNetwork{
		{ChainID: 1, Upstreams: []GatewayUpstream{
			{ID: "pub", Kind: UpstreamExternal, Endpoint: "https://eth.example"},
		}},
		{ChainID: 1337, Upstreams: []GatewayUpstream{
			{ID: "devnet", Kind: UpstreamManagedDevnet, TargetID: "local", Local: true},
			{ID: "pub2", Kind: UpstreamExternal, Endpoint: "https://backup.example"},
		}},
	}}

	out, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// Chain 1 has nothing local, so its only upstream must go in at full weight.
	pub := section(t, out, "pub")
	if strings.Contains(pub, "tier:fallback") {
		t.Errorf("a chain with no local node has no fallback tier:\n%s", pub)
	}
	// Chain 1337 does have a local node, so the public one stays a fallback.
	pub2 := section(t, out, "pub2")
	if !strings.Contains(pub2, "tier:fallback") {
		t.Errorf("a public upstream beside a local node is still a fallback:\n%s", pub2)
	}
}
```

Write the `section` helper only if the file has no equivalent — read `gateway_test.go` first; it already asserts on rendered output and may have one. If you add it, keep it small: find the `- id: <name>` line and return through to the next `- id:` or end.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run PublicOnly -v`
Expected: FAIL — chain 1's upstream carries `tier:fallback`.

- [ ] **Step 3: Write minimal implementation**

At `internal/catalog/gateway.go`, before the upstream loop, determine per network whether anything local serves it, then use it at line 394:

```go
		// A public upstream is a fallback only when there is something for it to
		// fall back TO. On a chain served entirely by public endpoints, marking
		// them tier:fallback tells eRPC to avoid the only paths that exist.
		hasLocal := false
		for _, u := range n.Upstreams {
			if u.Local {
				hasLocal = true
				break
			}
		}
```

and change the assignment to:

```go
				Fallback:   !u.Local && hasLocal,
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./...`
Expected: PASS. Existing render tests that assert `tier:fallback` on a chain with a local node still pass; any that assert it on a public-only chain now encode the old rule and must be updated — read each before changing it and say in your report what you changed and why.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/gateway.go internal/catalog/gateway_test.go
git commit -m "fix(catalog): the only path is not a fallback"
```

---

### Task 4: Offer the set in the UI

**Files:**
- Modify: `cmd/valve-node-app/web/src/api.ts`
- Modify: `cmd/valve-node-app/web/src/rpc.ts` (+ rebuilt `dist/`)

**Interfaces:**
- Consumes: `GET /api/gateways/{gid}/knownset/{chainId}` from Task 2; `addExternalUpstreams(gid, chainId, urls, recentOnly?)` from the existing picker.
- Produces: `api.knownSet(gid: string, chainId: number): Promise<KnownSetResponse>`.

- [ ] **Step 1: Add the typed call**

In `api.ts`, following the file's existing `request` helper signature (read it first):

```ts
export interface KnownSetEndpoint {
  url: string;
  provider: string;
  websocket: boolean;
  archive: boolean;
  alreadyAdded: boolean;
}

export interface KnownSetResponse {
  endpoints: KnownSetEndpoint[] | null;
  key: string;
  usingDefaultKey: boolean;
}

export function knownSet(gid: string, chainId: number): Promise<KnownSetResponse> {
  return request<KnownSetResponse>(`/api/gateways/${encodeURIComponent(gid)}/knownset/${chainId}`);
}
```

- [ ] **Step 2: Make "Add valve's set" the primary action**

In `rpc.ts`'s discover modal, open with the set rather than the probed list. Show each entry with its provider, its capability tags, and `already added` where `alreadyAdded` is true; the button adds only the ones not already present, in the order returned:

```ts
  // The set is offered before the probed list because it is vetted and ordered
  // by measured capability; the feed is the escape hatch, not the default.
  async function offerKnownSet(gid: string, chainId: number): Promise<void> {
    let set: api.KnownSetResponse;
    try {
      set = await api.knownSet(gid, chainId);
    } catch (err) {
      openModal(`<h2>Endpoints for chain ${chainId}</h2>
        <p class="error small">Could not read the set: ${escapeHtml(message(err))}</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>`,
        () => closeModal());
      return;
    }

    const eps = set.endpoints ?? [];
    const toAdd = eps.filter((e) => !e.alreadyAdded).map((e) => e.url);
    const providers = new Set(eps.map((e) => e.provider)).size;

    const rows = eps.map((e) => {
      const caps = [
        e.websocket ? `<span class="t ws">websocket</span>` : "",
        e.archive ? `<span class="t ar">archive</span>` : "",
        e.alreadyAdded ? `<span class="t dup">already added</span>` : "",
      ].join("");
      return `<li><code>${escapeHtml(e.url)}</code>
                <span class="muted small">${escapeHtml(e.provider)}</span> ${caps}</li>`;
    }).join("");

    openModal(
      `<h2>Endpoints for chain ${chainId}</h2>
       <p class="muted small">${providers} providers valve has measured, in the order the gateway
         should prefer them.</p>
       <ul class="plain-list">${rows}</ul>
       ${set.usingDefaultKey
          ? `<p class="muted small">Using the shared <code>${escapeHtml(set.key)}</code> key, so this
               works with no setup. A free key of your own removes the shared limit.</p>`
          : `<p class="muted small">Using your key for this chain.</p>`}
       <div class="modal-actions">
         <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
         <button class="btn btn-ghost" data-modal-action="discover">Choose from the full list</button>
         <button class="btn" data-modal-action="add"${toAdd.length ? "" : " disabled"}>
           ${toAdd.length ? `Add ${toAdd.length}` : "Nothing to add"}</button>
       </div>`,
      (action) => {
        closeModal();
        if (action === "add") void addExternalUpstreams(gid, chainId, toAdd);
        if (action === "discover") void openDiscoverModal(gid, chainId);
      },
    );
  }
```

Match the file's real `message(err)` / `escapeHtml` imports and its modal-action convention — read `openDiscoverModal` directly above before writing this, and reuse whatever it uses. The `.t`, `.ws`, `.ar`, `.dup` classes need adding to `style.css` if absent; keep them to the state palette.

Keep "Choose from N discovered" as a secondary button into the existing `openDiscoverModal`.

- [ ] **Step 3: Build**

Run: `cd cmd/valve-node-app/web && npm run build`
Expected: tsc strict clean.
Then: `cd - && go build ./... && go test ./...`
Expected: PASS, `dist/` re-embeds.

- [ ] **Step 4: Drive it**

Start the app, open the RPC page, and add the set on `evm:1`. Confirm the chain goes from one upstream to four providers **and** that `eth_subscribe` on `evm:1` then works — it cannot today, because no Ethereum upstream speaks WebSocket. That is the check that the set did something, not that the code ran.

- [ ] **Step 5: Commit**

```bash
git add cmd/valve-node-app/web/src/api.ts cmd/valve-node-app/web/src/rpc.ts cmd/valve-node-app/web/dist
git commit -m "feat(rpc): offer the known set first"
```

---

### Task 5: The chain row leads with redundancy

**Files:**
- Modify: `cmd/valve-node-app/web/src/rpc.ts`, `style.css` (+ rebuilt `dist/`)

**Interfaces:**
- Consumes: the existing `networksTable(gw)` render and `api.GatewayView`.
- Produces: no new exports; `networksTable` becomes a per-chain row renderer.

- [ ] **Step 1: Replace the table with rows**

Each chain renders: name and `evm:<id>`; a segmented bar whose filled count is the configured upstreams and whose total is four; the upstreams with capability tags; and one sentence naming the gap. Colour means state only — teal answering, amber single-path-or-missing-capability, red down.

The sentence states a fact and its consequence. Derive it, do not hardcode per chain:
- no upstream with a `ws://`/`wss://` endpoint → "No WebSocket upstream, so `eth_subscribe` fails on this chain."
- exactly one upstream → "One endpoint, so this chain stops when it does."
- every upstream external and none local → "No node of your own serves this chain."

- [ ] **Step 2: Drop the gateway-list wrapper**

A machine hosts one managed eRPC, so the list describes a collection that cannot have a
second member. In `render()` (`rpc.ts:235`), replace `gateways.map(gatewayBlock)` with a
single gateway render, and replace the always-present "Add a gateway" button with one that
appears only when a registered machine has no gateway — the guard added in the previous
plan already computes that. `gatewayBlock`'s `<section class="rpc-gateway">` wrapper goes
with it; the infra line (machine, image, TLS, base URL) becomes the page header.

Keep `emptyState()` — it is what a machine with no gateway shows, and it is now the only
route to creating one.

**If more than one gateway is present** (a fleet, once remote targets exist), render the
machine name as a heading per gateway rather than reinstating the list chrome. Do not
assume exactly one in the code; assume one *per machine*.

- [ ] **Step 3: Collapse the warning surfaces**

`errorBlock`, `blocked`, `warnings[]`, `tlsBanner` and `actionErr` currently render as five separate banners per gateway. Fold them into one attention strip above the chains, each line carrying the exact command where one exists — the orphan banner's `docker rm -f` is the model. Orphan records keep their own strip: a leftover container is not a property of a chain.

- [ ] **Step 4: Build, drive, commit**

Run: `cd cmd/valve-node-app/web && npm run build && cd - && go build ./... && go test ./...`
Then open the page and confirm each of your three chains shows its true state — `evm:1337` one endpoint with WebSocket, `evm:369` and `evm:1` one endpoint without.

```bash
git add cmd/valve-node-app/web/src cmd/valve-node-app/web/dist
git commit -m "feat(rpc): lead each chain with how many ways it can be answered"
```

---

## Final verification

- [ ] `go build ./... && go test ./...` green.
- [ ] `cd cmd/valve-node-app/web && npm run build` clean under tsc strict.
- [ ] Add the set on `evm:369` and re-provision; confirm the rendered `~/.valve-node-app/erpc.yaml` lists four providers for that chain and that **none of them carries `tier:fallback`**, since no local node serves 369.
- [ ] `eth_subscribe` on `evm:1` through Caddy, which is impossible before this plan:

```bash
python3 - <<'PY'
import asyncio, json, os, ssl, websockets
CA = os.path.expanduser("~/.valve-node-app/caddy-root.crt")
async def main():
    ctx = ssl.create_default_context(cafile=CA)   # verification ON
    url = "wss://default-07fcdc.localhost-valaxy.com:8443/main/evm/1"
    async with websockets.connect(url, ssl=ctx, additional_headers={"Accept-Encoding":"gzip"}) as ws:
        await ws.send(json.dumps({"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}))
        print("subscribe:", await asyncio.wait_for(ws.recv(), 20))
        print("newHeads :", (await asyncio.wait_for(ws.recv(), 90))[:120])
asyncio.run(main())
PY
```

Do not switch TLS verification off if this fails on the certificate — that would hide the failure this step exists to catch. `GET /api/gateways/{gid}/tls/verify` checks the same chain.
