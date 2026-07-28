# One managed eRPC per device Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce one managed eRPC gateway per device, merge the configs that already violate it without silently killing containers, delete the `enableGzip: false` workaround the upstream fix made obsolete, and make the endpoint picker add several endpoints at once.

**Architecture:** A gateway names its machine via `Placement.TargetID`, so gateway↔device is one-to-one. The merge is a pure function over `[]config.Gateway` called from `Config.migrate()`; the create path gets a guard so the invariant cannot be re-broken; containers left behind by a merge are recorded in the config file (not derived, see Global Constraints) and surfaced for the operator to wipe. The gzip removal is a template edit plus three test flips. The picker becomes checkboxes over the already-probed candidate list.

**Tech Stack:** Go 1.25 (stdlib + `text/template`), TypeScript (tsc strict + vite), no new dependencies.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-28-one-erpc-per-device-design.md`. Read it first.
- **Upstream dedupe key is `(kind, targetID, endpoint)`, never endpoint alone.** A managed upstream carries no endpoint — both `managed-devnet` rows in the real merge are `{kind: managed-devnet, targetId: local, endpoint: ""}` and an endpoint-only key keeps them as two identical rows.
- **Survivor order: fronted (`GatewayConfig.Fronted()`) → `ID == config.DefaultGatewayID` → earliest in the slice.**
- **A merge never stops a container.** Precedent: `handleGatewayDelete` returns "its container was NOT touched — stop or wipe it before removing it if you wanted it gone." Match that stance.
- **Orphan records must be persisted**, not computed at load. `migrate()` runs in memory and is only written back by the next `Save()`; a derived notice disappears on the first save while the container keeps running.
- Verify with `go build ./... && go test ./...`. UI changes additionally need `cd cmd/valve-node-app/web && npm run build`, then `go build ./...` to confirm the rebuilt `dist/` still embeds.
- Commit directly to `master`. Do not `git push`.
- Task 5 (gzip) requires the gateways to already run eRPC `a7a53ec2`. That re-provision is operational and happens outside this plan.

## File Structure

| File | Responsibility |
|---|---|
| `internal/config/merge.go` (create) | Pure merge: survivor choice, network union, upstream dedupe. No I/O. |
| `internal/config/merge_test.go` (create) | Table-driven tests for the merge rule. |
| `internal/config/config.go` (modify) | `OrphanedContainer` type, `Config.Orphans` field, call merge from `migrate()`. |
| `internal/server/gateways.go` (modify) | Create-path guard; expose orphans; dismiss endpoint. |
| `internal/catalog/gateway.go` (modify) | Delete the `Fronted` gzip branch from the template. |
| `internal/catalog/caddy.go` (modify) | Delete `MustDisableGzipBehindProxy`. |
| `cmd/valve-node-app/web/src/rpc.ts` (modify) | Multi-select picker; orphan banner. |

---

### Task 1: The merge rule as a pure function

**Files:**
- Create: `internal/config/merge.go`
- Create: `internal/config/merge_test.go`

**Interfaces:**
- Consumes: `catalog.GatewayConfig`, `catalog.GatewayNetwork`, `catalog.GatewayUpstream`, `catalog.GatewayUpstream.KindOrDefault()`, `catalog.GatewayConfig.Fronted()`.
- Produces: `func mergeGatewaysPerTarget(gws []Gateway) ([]Gateway, []OrphanedContainer)`, `type OrphanedContainer struct{ ContainerName, TargetID, MergedInto string }`.

- [ ] **Step 1: Write the failing test**

Create `internal/config/merge_test.go`:

```go
package config

import (
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
)

// gw is a terse gateway literal for these tables.
func gw(id, target string, fronted bool, nets ...catalog.GatewayNetwork) Gateway {
	c := catalog.GatewayConfig{Networks: nets}
	if fronted {
		c.TLS = &catalog.GatewayTLS{Enabled: true, Hostname: id + ".example"}
	}
	return Gateway{ID: id, Placement: GatewayPlacement{TargetID: target, Backend: "docker"}, Config: c}
}

func devnetUp(target string) catalog.GatewayUpstream {
	return catalog.GatewayUpstream{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: target}
}

func extUp(id, url string) catalog.GatewayUpstream {
	return catalog.GatewayUpstream{ID: id, Kind: catalog.UpstreamExternal, Endpoint: url}
}

// Two gateways on one device is the violation this exists to fix: it means two
// managed eRPC containers, overlapping chains, two pollers against one node.
func TestMergeGatewaysPerTarget(t *testing.T) {
	tests := []struct {
		name        string
		in          []Gateway
		wantIDs     []string
		wantOrphans []string
		check       func(t *testing.T, got []Gateway)
	}{
		{
			name:    "one gateway per target is left alone",
			in:      []Gateway{gw("default", "local", true), gw("other", "remote", false)},
			wantIDs: []string{"default", "other"},
		},
		{
			name: "the fronted gateway survives even when it is not first",
			in: []Gateway{
				gw("edge", "local", false),
				gw("default", "local", true),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
		},
		{
			name: "with no TLS anywhere the DefaultGatewayID survives",
			in: []Gateway{
				gw("edge", "local", false),
				gw("default", "local", false),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
		},
		{
			name: "with neither, the earliest survives",
			in: []Gateway{
				gw("alpha", "local", false),
				gw("beta", "local", false),
			},
			wantIDs:     []string{"alpha"},
			wantOrphans: []string{"valve-node-app-erpc-beta"},
		},
		{
			name: "networks are unioned and the survivor keeps its own door",
			in: []Gateway{
				gw("default", "local", true, catalog.GatewayNetwork{ChainID: 1337, Upstreams: []catalog.GatewayUpstream{devnetUp("local")}}),
				gw("edge", "local", false,
					catalog.GatewayNetwork{ChainID: 369, Upstreams: []catalog.GatewayUpstream{extUp("public-369-1", "https://rpc.pulsechain.com")}},
				),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
			check: func(t *testing.T, got []Gateway) {
				if len(got[0].Config.Networks) != 2 {
					t.Fatalf("want chains 1337 and 369, got %d networks", len(got[0].Config.Networks))
				}
				if !got[0].Config.Fronted() {
					t.Error("the survivor must keep its TLS; a merge changes which chains are served, not the door")
				}
			},
		},
		{
			// The case an endpoint-only key gets wrong: a managed upstream has
			// no endpoint of its own, so both rows are {managed-devnet, local, ""}.
			name: "identical managed upstreams on a shared chain collapse to one",
			in: []Gateway{
				gw("default", "local", true, catalog.GatewayNetwork{ChainID: 1337, Upstreams: []catalog.GatewayUpstream{devnetUp("local")}}),
				gw("edge", "local", false, catalog.GatewayNetwork{ChainID: 1337, Upstreams: []catalog.GatewayUpstream{devnetUp("local")}}),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
			check: func(t *testing.T, got []Gateway) {
				ups := got[0].Config.Networks[0].Upstreams
				if len(ups) != 1 {
					t.Fatalf("the same devnet must not appear twice, got %d: %+v", len(ups), ups)
				}
			},
		},
		{
			name: "a colliding upstream id is re-suffixed rather than dropped",
			in: []Gateway{
				gw("default", "local", true, catalog.GatewayNetwork{ChainID: 1, Upstreams: []catalog.GatewayUpstream{extUp("public-1-1", "https://a.example")}}),
				gw("edge", "local", false, catalog.GatewayNetwork{ChainID: 1, Upstreams: []catalog.GatewayUpstream{extUp("public-1-1", "https://b.example")}}),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
			check: func(t *testing.T, got []Gateway) {
				ups := got[0].Config.Networks[0].Upstreams
				if len(ups) != 2 {
					t.Fatalf("two different endpoints must both survive, got %d", len(ups))
				}
				if ups[0].ID == ups[1].ID {
					t.Errorf("upstream ids must be unique after a merge, both are %q", ups[0].ID)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, orphans := mergeGatewaysPerTarget(tc.in)

			var ids []string
			for _, g := range got {
				ids = append(ids, g.ID)
			}
			if len(ids) != len(tc.wantIDs) {
				t.Fatalf("gateways: want %v, got %v", tc.wantIDs, ids)
			}
			for i := range ids {
				if ids[i] != tc.wantIDs[i] {
					t.Fatalf("gateways: want %v, got %v", tc.wantIDs, ids)
				}
			}

			var names []string
			for _, o := range orphans {
				names = append(names, o.ContainerName)
			}
			if len(names) != len(tc.wantOrphans) {
				t.Fatalf("orphans: want %v, got %v", tc.wantOrphans, names)
			}
			for i := range names {
				if names[i] != tc.wantOrphans[i] {
					t.Fatalf("orphans: want %v, got %v", tc.wantOrphans, names)
				}
			}

			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestMergeGatewaysPerTarget -v`
Expected: FAIL — `undefined: mergeGatewaysPerTarget`, `undefined: OrphanedContainer`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/config/merge.go`:

```go
package config

import (
	"fmt"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// OrphanedContainer is a container valve-node-app stopped managing but did NOT
// stop. It is persisted rather than derived: migrate() runs in memory and is
// only written back by the next Save, so a notice computed at load would
// disappear on the first save while the container kept serving.
type OrphanedContainer struct {
	// ContainerName is the container left running on the target.
	ContainerName string `json:"containerName"`
	// TargetID is the machine it is running on.
	TargetID string `json:"targetId"`
	// MergedInto is the gateway that absorbed its networks.
	MergedInto string `json:"mergedInto"`
}

// mergeGatewaysPerTarget folds every gateway sharing a Placement.TargetID into
// one. A gateway NAMES the machine it runs on, so two gateways on one device
// mean two managed eRPC containers: overlapping chains, two state pollers
// against the same node, and a reverse proxy that can only front one of them.
//
// The containers the merge leaves behind are returned, never stopped. This app
// does not stop a container it did not just start — see handleGatewayDelete,
// which says exactly that and leaves the container alone.
func mergeGatewaysPerTarget(gws []Gateway) ([]Gateway, []OrphanedContainer) {
	// Group by target, preserving first-seen order so the result is stable.
	order := make([]string, 0, len(gws))
	groups := make(map[string][]int, len(gws))
	for i, g := range gws {
		t := g.Placement.TargetID
		if _, seen := groups[t]; !seen {
			order = append(order, t)
		}
		groups[t] = append(groups[t], i)
	}

	var out []Gateway
	var orphans []OrphanedContainer

	for _, t := range order {
		idx := groups[t]
		if len(idx) == 1 {
			out = append(out, gws[idx[0]])
			continue
		}

		keep := survivorIndex(idx, gws)
		survivor := gws[keep]
		for _, i := range idx {
			if i == keep {
				continue
			}
			mergeConfigInto(&survivor.Config, gws[i].Config)
			orphans = append(orphans, OrphanedContainer{
				ContainerName: ops.ERPCContainerNameFor(gws[i].ID),
				TargetID:      t,
				MergedInto:    survivor.ID,
			})
		}
		out = append(out, survivor)
	}

	return out, orphans
}

// survivorIndex picks which gateway absorbs the others: the fronted one first,
// so the merge keeps the secure door and the stable hostname; then the one
// named DefaultGatewayID; then the earliest, so the choice is deterministic
// rather than map-order.
func survivorIndex(idx []int, gws []Gateway) int {
	for _, i := range idx {
		if gws[i].Config.Fronted() {
			return i
		}
	}
	for _, i := range idx {
		if gws[i].ID == DefaultGatewayID {
			return i
		}
	}
	return idx[0]
}

// upstreamKey identifies an upstream by what actually distinguishes it. Keying
// on endpoint alone is wrong: a managed upstream carries no endpoint of its own
// (the address is derived at render time), so every managed-devnet row on a
// target looks like {managed-devnet, <target>, ""} and two of them would both
// survive as "different".
func upstreamKey(u catalog.GatewayUpstream) [3]string {
	return [3]string{u.KindOrDefault(), u.TargetID, u.Endpoint}
}

// mergeConfigInto unions src's networks into dst. dst keeps its own ProjectID,
// BindAddr, Port, TLS and metrics settings: a merge changes WHICH chains a
// gateway serves, never the door it serves them on.
func mergeConfigInto(dst *catalog.GatewayConfig, src catalog.GatewayConfig) {
	seenUp := make(map[[3]string]bool)
	takenID := make(map[string]bool)
	for _, n := range dst.Networks {
		for _, u := range n.Upstreams {
			seenUp[upstreamKey(u)] = true
			takenID[u.ID] = true
		}
	}

	netAt := make(map[int]int, len(dst.Networks))
	for i, n := range dst.Networks {
		netAt[n.ChainID] = i
	}

	for _, sn := range src.Networks {
		at, ok := netAt[sn.ChainID]
		if !ok {
			dst.Networks = append(dst.Networks, catalog.GatewayNetwork{ChainID: sn.ChainID})
			at = len(dst.Networks) - 1
			netAt[sn.ChainID] = at
		}
		for _, u := range sn.Upstreams {
			if seenUp[upstreamKey(u)] {
				continue
			}
			seenUp[upstreamKey(u)] = true
			u.ID = uniqueUpstreamID(u.ID, takenID)
			takenID[u.ID] = true
			dst.Networks[at].Upstreams = append(dst.Networks[at].Upstreams, u)
		}
	}
}

// uniqueUpstreamID suffixes a colliding id rather than dropping the upstream:
// two gateways can legitimately carry different endpoints under the same
// generated name, and losing one silently would be a merge that reports success
// while removing an endpoint.
func uniqueUpstreamID(id string, taken map[string]bool) string {
	if id == "" || !taken[id] {
		return id
	}
	for n := 2; ; n++ {
		cand := fmt.Sprintf("%s-%d", id, n)
		if !taken[cand] {
			return cand
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestMergeGatewaysPerTarget -v`
Expected: PASS, all seven subtests.

If `internal/config` importing `internal/ops` creates an import cycle, replace the `ops.ERPCContainerNameFor` call with a local `erpcContainerName(id string) string` helper in `merge.go` that reproduces the same naming, and note the duplication in a comment. Check with `go build ./...` before assuming.

- [ ] **Step 5: Commit**

```bash
git add internal/config/merge.go internal/config/merge_test.go
git commit -m "feat(config): fold gateways that share a device into one"
```

---

### Task 2: Persist the orphan record and run the merge on load

**Files:**
- Modify: `internal/config/config.go` (add `Orphans` field; call merge from `migrate()`)
- Modify: `internal/config/merge_test.go` (add the migrate-level test)

**Interfaces:**
- Consumes: `mergeGatewaysPerTarget`, `OrphanedContainer` from Task 1.
- Produces: `Config.Orphans []OrphanedContainer` (JSON `orphanedContainers`), populated by `Config.migrate()`.

- [ ] **Step 1: Write the failing test**

Append to `internal/config/merge_test.go`:

```go
// migrate must record the leftover container, and must not record it twice
// when the config is loaded again after a save.
func TestMigrateRecordsOrphansIdempotently(t *testing.T) {
	c := Config{Gateways: []Gateway{
		gw("edge", "local", false),
		gw("default", "local", true),
	}}

	c.migrate()

	if len(c.Gateways) != 1 || c.Gateways[0].ID != "default" {
		t.Fatalf("want a single surviving gateway 'default', got %+v", c.Gateways)
	}
	if len(c.Orphans) != 1 {
		t.Fatalf("the leftover container must be recorded, got %+v", c.Orphans)
	}
	if c.Orphans[0].ContainerName != "valve-node-app-erpc-edge" {
		t.Errorf("orphan name: got %q", c.Orphans[0].ContainerName)
	}
	if c.Orphans[0].MergedInto != "default" {
		t.Errorf("orphan must name what absorbed it: got %q", c.Orphans[0].MergedInto)
	}

	// Second load of the already-merged config: nothing left to merge, and the
	// existing record must survive rather than being duplicated or dropped.
	c.migrate()
	if len(c.Orphans) != 1 {
		t.Fatalf("migrate must be idempotent, got %+v", c.Orphans)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestMigrateRecordsOrphansIdempotently -v`
Expected: FAIL — `c.Orphans undefined`.

- [ ] **Step 3: Write minimal implementation**

In `internal/config/config.go`, add the field to `Config` immediately after the `Gateways` field:

```go
	// Orphans are containers a merge stopped managing but did NOT stop. They
	// are stored rather than recomputed: migrate() runs in memory and is only
	// written back by the next Save, so a derived notice would vanish on the
	// first save while the container kept serving.
	Orphans []OrphanedContainer `json:"orphanedContainers,omitempty"`
```

At the **end** of `func (c *Config) migrate()` (after the existing `LegacyGateway` loop, so gateways adopted from the legacy shape are also subject to the invariant), add:

```go
	// One managed eRPC per device. Two gateways on one target mean two
	// containers, overlapping chains and two pollers against the same node.
	merged, orphans := mergeGatewaysPerTarget(c.Gateways)
	c.Gateways = merged
	for _, o := range orphans {
		known := false
		for _, have := range c.Orphans {
			if have.ContainerName == o.ContainerName {
				known = true
				break
			}
		}
		if !known {
			c.Orphans = append(c.Orphans, o)
		}
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS, including the pre-existing migration tests.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/merge_test.go
git commit -m "feat(config): record the container a merge leaves behind"
```

---

### Task 3: Refuse a second gateway on a device

**Files:**
- Modify: `internal/server/gateways.go` (in `handleGatewayCreate`)
- Modify: `internal/server/gateways_test.go:49-72` (**rewrite** `TestGateways_TwoGatewaysCoexistWithDistinctContainers`, which currently asserts the behaviour this task forbids)
- Modify: `cmd/valve-node-app/web/src/rpc.ts` (hide "add gateway" for a device that has one)

**Interfaces:**
- Consumes: `config.Config.Gateways`, `config.Gateway.Placement.TargetID`. Existing test helpers: `gatewayServer(t) *apiTestServer`, `addGateway(t, a, id, targetID string, cfg catalog.GatewayConfig) gatewayView`, `a.do(t, method, path, body) *http.Response`, `decode[T](t, res) T`, `addTarget(t, a)` (which adds target `"local"`).
- Produces: `POST /api/gateways` returns 400 when the target already has a gateway.

**Read this first.** `TestGateways_TwoGatewaysCoexistWithDistinctContainers` creates `default` and `edge` **both on target `local`** and asserts they coexist. That is precisely the state this task makes illegal, so the test must be rewritten rather than left to fail. Its underlying intent is still valid and must be preserved: container names are distinct per gateway, and `default` keeps the historical name `valve-node-app-erpc` so an existing install is not orphaned. Express that across two *different* targets.

- [ ] **Step 1: Rewrite the contradicting test and add the rejection test**

Replace `TestGateways_TwoGatewaysCoexistWithDistinctContainers` (lines 49-72) with:

```go
// Distinct container names still matter — docker run --name refuses a
// duplicate — but two gateways now mean two MACHINES. The same-machine case is
// covered by TestGateways_RefuseASecondGatewayOnOneMachine below.
func TestGateways_TwoGatewaysCoexistWithDistinctContainers(t *testing.T) {
	a := gatewayServer(t)

	res := a.do(t, "POST", "/api/targets", map[string]any{"id": "second", "mode": "local"})
	res.Body.Close()

	net := []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
		{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
	}}}
	first := addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100, Networks: net})
	second := addGateway(t, a, "edge", "second", catalog.GatewayConfig{Port: 4200, Networks: net})

	if first.ContainerName == second.ContainerName {
		t.Fatalf("two gateways share the container name %q — docker run --name would refuse the second one outright", first.ContainerName)
	}
	if first.ContainerName != "valve-node-app-erpc" {
		t.Errorf("the default gateway must keep the historical container name so an existing install is not orphaned: got %q", first.ContainerName)
	}
	if second.ContainerName != "valve-node-app-erpc-edge" {
		t.Errorf("second container name: got %q", second.ContainerName)
	}

	body := decode[gatewaysResponse](t, a.do(t, "GET", "/api/gateways", nil))
	if len(body.Gateways) != 2 {
		t.Fatalf("got %d gateways, want 2", len(body.Gateways))
	}
}

// A gateway NAMES the machine it runs on, so a second one on that machine is a
// second managed eRPC container: overlapping chains, two state pollers against
// one node, and only one of them reachable through the reverse proxy.
func TestGateways_RefuseASecondGatewayOnOneMachine(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	res := a.do(t, "POST", "/api/gateways", map[string]any{
		"id":        "edge",
		"placement": map[string]string{"targetId": "local", "backend": "docker"},
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(b), "default") {
		t.Errorf("the error must name the gateway already on that machine: %s", b)
	}
}
```

Add `"io"` and `"strings"` to the test file's imports if they are not already there.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run TestGatewayCreateRefusesSecondOnSameTarget -v`
Expected: FAIL — the create succeeds and returns 200.

- [ ] **Step 3: Write minimal implementation**

In `handleGatewayCreate`, after the backend validation block and before `gwCfg` is built, add:

```go
	// One managed eRPC per device. The gateway's Placement names its machine,
	// so a second gateway on that machine is a second container fighting over
	// the same chains — and only one of them can be behind the reverse proxy.
	targetID := strings.TrimSpace(req.Placement.TargetID)
	if existing, err := s.loadConfig(); err == nil {
		for _, g := range existing.Gateways {
			if g.Placement.TargetID == targetID {
				writeError(w, http.StatusBadRequest, fmt.Sprintf(
					"machine %q already runs gateway %q — a machine hosts one managed eRPC, so add the chains to that gateway instead of creating a second one",
					targetID, g.ID))
				return
			}
		}
	}
```

Use whatever the file's existing config-reading helper is called (read the neighbouring handlers — `handleGatewayDelete` uses `s.updateConfig`, and the list handler reads config directly). Match it rather than adding a new accessor.

- [ ] **Step 4: Stop the UI offering a second gateway**

In `rpc.ts`, where the "add gateway" control is rendered, omit it for any target that already appears as a `placement.targetId` in the gateway list, and say why rather than rendering a dead control:

```ts
  // A machine hosts one managed eRPC. Offering "add gateway" on a machine that
  // already has one would render a button whose only outcome is a 400.
  function canAddGatewayOn(targetId: string, gateways: api.Gateway[]): boolean {
    return !gateways.some((g) => g.placement?.targetId === targetId);
  }
```

Where the control would have been, render:
`<p class="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>`

Match the file's actual gateway-list type and property names — read the render function before writing this.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/server/ -v`
Expected: PASS, including the rewritten coexistence test.
Then: `cd cmd/valve-node-app/web && npm run build && cd - && go build ./...`
Expected: tsc strict clean, `dist/` re-embeds.

- [ ] **Step 6: Commit**

```bash
git add internal/server/gateways.go internal/server/gateways_test.go cmd/valve-node-app/web/src/rpc.ts cmd/valve-node-app/web/dist
git commit -m "feat(server): one managed eRPC per machine, refused at creation"
```

---

### Task 4: Surface the orphaned container

**Files:**
- Modify: `internal/server/gateways.go` (include orphans in the list response; add dismiss route)
- Modify: `cmd/valve-node-app/web/src/rpc.ts` (banner)
- Modify: `cmd/valve-node-app/web/src/api.ts` (typed field + dismiss call)

**Interfaces:**
- Consumes: `config.Config.Orphans` from Task 2.
- Produces: `GET /api/gateways` response gains `"orphans": [{containerName, targetId, mergedInto}]`; `DELETE /api/orphans/{name}` removes one record.

- [ ] **Step 1: Write the failing test**

Append to `internal/server/gateways_test.go`:

```go
// The operator has to be told what is still running, or a merged-away gateway
// keeps serving stale config forever with nothing pointing at it.
func TestGatewayListReportsOrphans(t *testing.T) {
	srv, cleanup := newTestServerWithConfig(t, config.Config{
		Targets:  []config.Target{{ID: "local"}},
		Gateways: []config.Gateway{{ID: "default", Placement: config.GatewayPlacement{TargetID: "local", Backend: "docker"}}},
		Orphans: []config.OrphanedContainer{{
			ContainerName: "valve-node-app-erpc-edge", TargetID: "local", MergedInto: "default",
		}},
	})
	defer cleanup()

	rec := srv.get(t, "/api/gateways")
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "valve-node-app-erpc-edge") {
		t.Errorf("the leftover container must appear in the listing: %s", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run TestGatewayListReportsOrphans -v`
Expected: FAIL — the container name is absent from the response.

- [ ] **Step 3: Write minimal implementation**

In `handleGatewayList`, add `"orphans": cfg.Orphans` to the JSON object it writes alongside `"gateways"`.

Register the dismiss route beside the others in the same `mux.HandleFunc` block:

```go
	mux.HandleFunc("DELETE /api/orphans/{name}", s.handleOrphanDismiss)
```

And add the handler:

```go
// handleOrphanDismiss forgets a leftover container record. It does NOT stop the
// container: this app does not stop containers it did not start, and the
// operator dismisses the record once they have wiped it themselves.
func (s *Server) handleOrphanDismiss(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Orphans {
			if c.Orphans[i].ContainerName == name {
				c.Orphans = append(c.Orphans[:i], c.Orphans[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("no leftover container %q", name)
	}); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}
```

In `cmd/valve-node-app/web/src/api.ts`, add to the gateways-list response type:

```ts
export interface OrphanedContainer {
  containerName: string;
  targetId: string;
  mergedInto: string;
}
```

add `orphans?: OrphanedContainer[];` to the list response interface, and:

```ts
export async function dismissOrphan(name: string): Promise<void> {
  await request(`/api/orphans/${encodeURIComponent(name)}`, { method: "DELETE" });
}
```

Match the file's existing `request` helper name and signature — read it first.

In `rpc.ts`, render a banner above the gateway list for each orphan:

```ts
function orphanBanner(o: api.OrphanedContainer): string {
  return `
    <div class="banner banner-warn">
      <strong>${escapeHtml(o.containerName)}</strong> is still running on
      ${escapeHtml(o.targetId)}. Its chains were folded into
      <code>${escapeHtml(o.mergedInto)}</code>, but valve-node-app does not stop
      containers it did not start. Remove it yourself:
      <code>docker rm -f ${escapeHtml(o.containerName)}</code>
      <button class="btn btn-ghost" data-dismiss-orphan="${escapeHtml(o.containerName)}">Dismiss</button>
    </div>`;
}
```

Wire the `data-dismiss-orphan` click to `api.dismissOrphan(name)` then re-render, following the file's existing delegated-click pattern.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server/ -v`
Then: `cd cmd/valve-node-app/web && npm run build && cd - && go build ./...`
Expected: tests PASS, tsc clean, `dist/` re-embeds.

- [ ] **Step 5: Commit**

```bash
git add internal/server/gateways.go internal/server/gateways_test.go cmd/valve-node-app/web/src/api.ts cmd/valve-node-app/web/src/rpc.ts cmd/valve-node-app/web/dist
git commit -m "feat(server): say what a merge left running"
```

---

### Task 5: Retire the gzip workaround

**Requires:** the gateways already run eRPC `a7a53ec2` (`ERPCSourceRef`). Removing this while one still runs `e909aacb` restores the HTTP 500 on every WebSocket upgrade behind the proxy.

**Files:**
- Modify: `internal/catalog/gateway.go` (delete the `Fronted` branch from `gatewayConfigTemplate`)
- Modify: `internal/catalog/caddy.go` (delete `MustDisableGzipBehindProxy`)
- Modify: `internal/catalog/gateway_test.go:227,237`
- Modify: `internal/setup/tls_test.go:324`

**Interfaces:**
- Consumes: nothing new.
- Produces: `RenderGatewayConfig` no longer emits `enableGzip` for any gateway. `GatewayConfig.Fronted()` stays — it is still used by the template data struct and by TLS logic.

- [ ] **Step 1: Flip the failing tests**

In `internal/catalog/gateway_test.go`, replace the fronted assertion (currently at ~line 237):

```go
	if strings.Contains(fronted, "enableGzip") {
		// eRPC skips its gzip wrapper on upgrade requests as of a7a53ec2, so a
		// fronted gateway no longer has to trade away response compression to
		// keep eth_subscribe working.
		t.Errorf("a fronted gateway must no longer disable gzip:\n%s", fronted)
	}
```

Leave the unfronted assertion above it exactly as it is — it already asserts absence, and now both cases agree.

In `internal/setup/tls_test.go`, replace the block at ~line 322:

```go
	// erpc.yaml must NOT disable gzip: the upgrade-path fix landed in eRPC
	// a7a53ec2, so compression and eth_subscribe now coexist behind Caddy.
	cfg, err := e.ReadFile(context.Background(), "/Users/dev/.valve-node-app/erpc.yaml")
	if err != nil {
		t.Fatalf("erpc.yaml: %v", err)
	}
	if strings.Contains(string(cfg), "enableGzip") {
		t.Errorf("a fronted gateway must no longer disable gzip:\n%s", cfg)
	}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/catalog/ ./internal/setup/ -run 'Gzip|TLS|Gateway' -v`
Expected: FAIL — the renderer still emits `enableGzip: false`.

- [ ] **Step 3: Delete the workaround**

In `internal/catalog/gateway.go`, delete these five lines from `gatewayConfigTemplate`:

```
{{- if .Fronted}}
  # This gateway sits behind a reverse proxy, and every reverse proxy adds
  # Accept-Encoding: gzip to the requests it forwards. eRPC's gzip response
  # writer does not implement http.Hijacker, so a WebSocket upgrade carrying
  # that header fails with HTTP 500 and eth_subscribe stops working entirely.
  # Losing response compression is the cheaper half of that trade.
  enableGzip: false
{{- end}}
```

Replace the comment above the template that explains the branch (currently beginning "enableGzip is rendered only when the gateway is FRONTED") with:

```go
// enableGzip is no longer rendered. It existed because eRPC's WebSocket upgrade
// returned HTTP 500 whenever the client advertised Accept-Encoding: gzip, which
// every reverse proxy adds — so a fronted gateway had to trade away response
// compression to keep eth_subscribe. eRPC a7a53ec2 skips the gzip wrapper on
// upgrade requests, so both work. The measurement that retired this lives in
// docs/superpowers/specs/2026-07-28-one-erpc-per-device-design.md.
```

In `internal/catalog/caddy.go`, delete `MustDisableGzipBehindProxy` entirely and any reference to it.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go test ./...`
Expected: PASS across all 14 packages. If anything still references `MustDisableGzipBehindProxy`, the build names the file.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/gateway.go internal/catalog/caddy.go internal/catalog/gateway_test.go internal/setup/tls_test.go
git commit -m "fix(catalog): stop trading compression for eth_subscribe"
```

---

### Task 6: Multi-select in the endpoint picker

**Files:**
- Modify: `cmd/valve-node-app/web/src/rpc.ts` (`openDiscoverModal`, `addExternalUpstream`)

**Interfaces:**
- Consumes: `api.discoverEndpoints(chainId): Promise<api.ChainlistResult>`, existing `saveConfig(gid, cfg, label)`.
- Produces: `addExternalUpstreams(gid: string, chainId: number, urls: string[]): Promise<void>` replacing the single-URL `addExternalUpstream`.

- [ ] **Step 1: Replace the single-add helper**

Rewrite `addExternalUpstream` as a plural version that saves once. Find the existing function (it ends with `await saveConfig(gid, cfg, "Adding the endpoint");`) and replace it with:

```ts
  // One save for the whole selection, not one per endpoint: each save is a
  // config write plus a re-render, and adding three endpoints should not look
  // like three separate operator decisions.
  async function addExternalUpstreams(gid: string, chainId: number, urls: string[]): Promise<void> {
    if (!urls.length) return;
    const cfg = await configFor(gid);
    const nets = cfg.Networks ?? [];
    const net = nets.find((n) => n.ChainID === chainId) ?? { ChainID: chainId, Upstreams: [] };
    if (!nets.includes(net)) nets.push(net);

    // Continue the public-<chain>-<n> scheme from the highest suffix already
    // present, so ids stay unique and stable across repeated discoveries.
    let next = 1;
    for (const u of net.Upstreams ?? []) {
      const m = /^public-\d+-(\d+)$/.exec(u.ID ?? "");
      if (m) next = Math.max(next, Number(m[1]) + 1);
    }

    for (const url of urls) {
      if ((net.Upstreams ?? []).some((u) => u.Endpoint === url)) continue;
      net.Upstreams = [...(net.Upstreams ?? []), {
        ID: `public-${chainId}-${next++}`,
        Kind: "external",
        TargetID: "",
        Endpoint: url,
        Local: false,
        RecentOnly: false,
      }];
    }

    cfg.Networks = nets;
    await saveConfig(gid, cfg, urls.length === 1 ? "Adding the endpoint" : `Adding ${urls.length} endpoints`);
  }
```

Match the existing `configFor`/`saveConfig` helper names in the file; read the original function body first and keep its exact accessors.

- [ ] **Step 2: Pre-select the default three**

Add above `openDiscoverModal`:

```ts
  // Redundancy should be what happens when the operator does the easy thing.
  // Pre-tick the three fastest live endpoints, except that one slot goes to the
  // fastest live wss:// candidate when the feed offers one — a chain with only
  // http upstreams cannot serve eth_subscribe at all. The ws entry replaces the
  // slowest of the three rather than becoming a fourth.
  function defaultSelection(live: api.ChainlistEndpoint[]): Set<string> {
    const byLatency = [...live].sort((a, b) => (a.latencyMs ?? 1e9) - (b.latencyMs ?? 1e9));
    const picked = byLatency.slice(0, 3);
    const ws = byLatency.find((e) => e.url.startsWith("wss://") || e.url.startsWith("ws://"));
    if (ws && !picked.some((e) => e.url === ws.url)) {
      if (picked.length === 3) picked.pop();
      picked.push(ws);
    }
    return new Set(picked.map((e) => e.url));
  }
```

- [ ] **Step 3: Render checkboxes instead of buttons**

In `openDiscoverModal`, replace the `live.map(...)` list-item template with checkbox rows, and the single Close action with an Add action:

```ts
        ${
          live.length
            ? `<p class="muted small">${live.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes a chain survive an outage.</p>
               <ul class="plain-list rpc-picker">
                 ${live
                   .map((e) => {
                     const checked = preselected.has(e.url) ? " checked" : "";
                     return `
                   <li>
                     <label class="rpc-picker-option">
                       <input type="checkbox" value="${escapeHtml(e.url)}"${checked}>
                       <span><code>${escapeHtml(e.url)}</code></span>
                       <span class="muted small">${e.status === "live" ? `answered in ${e.latencyMs ?? 0} ms` : "not probed (WebSocket)"}</span>
                     </label>
                   </li>`;
                   })
                   .join("")}
               </ul>`
            : `<p class="error small">Nothing in the feed answered for chain ${chainId} right now.</p>`
        }
```

and the actions block:

```ts
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="add">Add selected</button>
        </div>
```

Compute `const preselected = defaultSelection(live);` immediately after `live` is derived.

Replace the action handler's `add:` branch with:

```ts
        if (action === "add") {
          const panel = modalBody();
          const urls = panel
            ? Array.from(panel.querySelectorAll<HTMLInputElement>('input[type="checkbox"]:checked')).map((i) => i.value)
            : [];
          closeModal();
          void addExternalUpstreams(gid, chainId, urls);
          return;
        }
```

- [ ] **Step 4: Build and verify**

Run: `cd cmd/valve-node-app/web && npm run build`
Expected: tsc strict clean, vite writes `dist/`.
Then: `cd - && go build ./... && go test ./...`
Expected: PASS — the rebuilt `dist/` still embeds.

Then drive it: start the app, open the RPC page, discover endpoints for chain 1, confirm three are pre-ticked including a `wss://` one, click Add, and confirm the config gains three upstreams in **one** save.

- [ ] **Step 5: Commit**

```bash
git add cmd/valve-node-app/web/src/rpc.ts cmd/valve-node-app/web/dist
git commit -m "feat(rpc): pick several endpoints at once, and default to redundancy"
```

---

## Final verification

- [ ] `go build ./... && go test ./...` — all 14 packages pass.
- [ ] `cd cmd/valve-node-app/web && npm run build` — tsc strict clean.
- [ ] Re-provision the surviving gateway. Confirm the rendered `~/.valve-node-app/erpc.yaml` contains **no** `enableGzip` line and lists chains 1337, 369 and 1.
- [ ] **Run the path the workaround protected.** `eth_subscribe` over `wss://` through Caddy on the TLS door — the thing never verified end to end. A gateway that cannot upgrade still starts and still reports itself healthy, so this must be observed, not inferred:

Verify against Caddy's own CA root, which the app already writes to
`~/.valve-node-app/caddy-root.crt`. Do **not** reach for `CERT_NONE`: the point of
`tls internal` is a real chain the operator installs once, so disabling
verification would hide exactly the failure this step exists to catch — and it
would leave a copy-pasteable MITM-friendly snippet in the repo.

```bash
python3 - <<'PY'
import asyncio, json, os, ssl, websockets

CA = os.path.expanduser("~/.valve-node-app/caddy-root.crt")

async def main():
    ctx = ssl.create_default_context(cafile=CA)  # verification ON, hostname checked
    url = "wss://default-07fcdc.localhost-valaxy.com:8443/main/evm/1337"
    async with websockets.connect(url, ssl=ctx, additional_headers={"Accept-Encoding": "gzip"}) as ws:
        await ws.send(json.dumps({"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}))
        print("subscribe:", await asyncio.wait_for(ws.recv(), 20))
        print("newHeads :", (await asyncio.wait_for(ws.recv(), 60))[:120])

asyncio.run(main())
PY
```

If this fails on certificate verification rather than on the upgrade, that is a
finding about the TLS setup, not something to switch off — the hostname must
resolve to the gateway and `caddy-root.crt` must match the CA Caddy is currently
using. `GET /api/gateways/{gid}/tls/verify` checks the same chain.

- [ ] Confirm `valve-node-app-erpc-edge` is reported in the UI as a leftover container and is still running (not stopped by the app), then remove it by hand and dismiss the notice.
