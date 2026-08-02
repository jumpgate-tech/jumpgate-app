# Easy-Button Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the sprawling multi-screen web UI with a single ~360px Wi-Fi-menu-style panel that presents the eRPC gateway as one power button and its chains as a glanceable, drill-in list.

**Architecture:** A new self-contained frontend module `panel.ts` (`renderPanel(root): Cleanup`) driven by internal view-state (list → network detail → endpoint detail — NOT hash routes, so it stays a popover). It reads/writes the existing gateway via the typed `api.ts` client, operating on the "primary" gateway (placement `targetId === "local"`, else the first). It reuses the server-decided `actions`/`blocked` model so the UI never offers an impossible action. Pure logic lives in `panelModel.ts` (unit-tested with vitest); rendering and wiring live in `panel.ts` and verify by building + running the binary.

**Tech Stack:** Vanilla TypeScript + Vite (no framework), the existing `api.ts` client, hand-rolled DOM helpers in `ui.ts` (`onAction`, `openModal`, `confirmModal`, `copyToClipboard`, `escapeHtml`, `fmtInt`). Vitest (new dev dependency) for `panelModel.ts` only. Go backend for one small struct change.

## Global Constraints

- **No framework.** Screens are `render(root: HTMLElement) => () => void` (a cleanup that closes streams/timers). Render via `innerHTML` template strings; wire events by delegation with `onAction(root, handler)`. Match `rpc.ts`/`home.ts` patterns exactly.
- **Escape all untrusted text** (URLs, hostnames, endpoint labels, server messages) with `escapeHtml` from `ui.ts` before concatenating into HTML.
- **Server decides actions.** Only render lifecycle actions present in `GatewayView.actions`; show `GatewayView.blocked` as the reason when set. Never hardcode which actions are legal.
- **Icons are inline SVG** (a `<symbol>` sprite), currentColor stroke — NOT SF Symbols (must render cross-platform in a browser).
- **Fixed 360px width, single card, flat bands on a 16px gutter, 18px leading slot.** One rounded container; everything inside is a full-bleed band split by 1px dividers. Design source of truth: the approved mockup `.superpowers/brainstorm/54055-1785634289/content/panel-v8.html` and `docs/superpowers/specs/2026-08-01-easy-button-panel-design.md`.
- **Dots: stillness = health.** Healthy = solid, no animation. Motion only on trouble; glitch frequency ∝ slow-request rate (`stable` | `occasional` | `frequent` | `off`).
- **Verify by running it** (repo rule — see memory `valve-node-verify-by-running`): every UI task ends by running the real binary and exercising the flow, not just `tsc`.
- **Fixed chain catalog:** Ethereum (1), PulseChain (369), PulseChain Testnet v4 (943), Devnet (1337 — opt-in, listed last).

## Three corrections the real backend forces on the mockups

1. **No per-network or per-endpoint on/off toggle.** A gateway fronts all its chains together; there is no per-chain or per-upstream start/stop (`gateways.go` state machine). So: the **only** power control is the master gateway button. A network is "on" because it is present in the config; turning it "off" = **Remove**. The mockups' per-network and per-endpoint toggle switches are dropped. The big round power button is the master control (tap green→stop, red→start); the redundant master toggle-switch is dropped too.
2. **Endpoints have no stored name.** `GatewayUpstream` (api.ts L644-654) has no name/label field — `UpstreamView.label` is derived. Custom endpoint rename therefore requires a backend field (Task 8). Until then, names are auto-derived from the domain. Network titles are catalog identities and are **not** renamable (drop the pencil on network titles; keep hover-rename only on endpoints).
3. **No single-gateway GET and two-pathed actions.** Get one gateway by filtering `getGateways()`. `start`/`stop`/`restart` → `api.gatewayAction(gid, kind)`; `create`/`recreate` → `api.provisionGateway(gid)` (then follow `api.streamSetup(started.targetId, …)`); `wipe` → `api.wipeGateway(gid)`.

---

## File Structure

- **Create** `cmd/valve-node-app/web/src/panelModel.ts` — pure helpers (no DOM): endpoint-name derivation, health classification, master-state mapping, capability-cell folding, config mutation (add/remove network, add/remove/edit upstream). Unit-tested.
- **Create** `cmd/valve-node-app/web/src/panelModel.test.ts` — vitest tests for the above.
- **Create** `cmd/valve-node-app/web/src/panel.ts` — `renderPanel(root)`, the icon sprite, view-state machine, all three views, event wiring.
- **Create** `cmd/valve-node-app/web/src/panel.css` — the panel visual language (imported from `panel.ts`).
- **Modify** `cmd/valve-node-app/web/src/main.ts` — register `#/panel`; make the empty hash land on the panel.
- **Modify** `cmd/valve-node-app/web/package.json` — add vitest + `test` script.
- **Create** `cmd/valve-node-app/web/vitest.config.ts` — vitest config (node env).
- **Modify (Task 8, backend)** `internal/catalog/gateway.go` + `internal/server/gateways.go` — optional `Name` on the upstream, surfaced in `UpstreamView.label`.

---

## Task 1: Test tooling + `endpointNameFromUrl`

**Files:**
- Modify: `cmd/valve-node-app/web/package.json`
- Create: `cmd/valve-node-app/web/vitest.config.ts`
- Create: `cmd/valve-node-app/web/src/panelModel.ts`
- Test: `cmd/valve-node-app/web/src/panelModel.test.ts`

**Interfaces:**
- Produces: `endpointNameFromUrl(endpoint: string): string`

- [ ] **Step 1: Add vitest** — in `web/`, run `npm i -D vitest@^2`. Add to `package.json` scripts: `"test": "vitest run"`, `"test:watch": "vitest"`.

- [ ] **Step 2: vitest config** — create `vitest.config.ts`:

```ts
import { defineConfig } from "vitest/config";
export default defineConfig({ test: { environment: "node", include: ["src/**/*.test.ts"] } });
```

- [ ] **Step 3: Write the failing test** — `src/panelModel.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { endpointNameFromUrl } from "./panelModel";

describe("endpointNameFromUrl", () => {
  it("takes the registrable label from the host", () => {
    expect(endpointNameFromUrl("https://rpc.publicnode.com/pulsechain")).toBe("publicnode");
  });
  it("handles a bare host", () => {
    expect(endpointNameFromUrl("https://mainnet.infura.io")).toBe("infura");
  });
  it("falls back to host for localhost/ip", () => {
    expect(endpointNameFromUrl("http://127.0.0.1:8545")).toBe("127.0.0.1");
    expect(endpointNameFromUrl("http://localhost:8545")).toBe("localhost");
  });
  it("returns 'endpoint' for an unparseable string", () => {
    expect(endpointNameFromUrl("not a url")).toBe("endpoint");
  });
});
```

- [ ] **Step 4: Run it, verify it fails** — `npm test`. Expected: FAIL (module/function missing).

- [ ] **Step 5: Implement** — create `src/panelModel.ts`:

```ts
// Pure helpers for the panel. No DOM, no api calls — unit-tested in panelModel.test.ts.

// endpointNameFromUrl derives a friendly default name from an endpoint URL:
// the second-level domain label ("publicnode" from rpc.publicnode.com), or the
// bare host for localhost / IPs, or "endpoint" if the URL won't parse.
export function endpointNameFromUrl(endpoint: string): string {
  let host: string;
  try {
    host = new URL(endpoint).hostname;
  } catch {
    return "endpoint";
  }
  if (!host) return "endpoint";
  if (host === "localhost" || /^[0-9.]+$/.test(host) || /^\[.*\]$/.test(host)) return host;
  const parts = host.split(".").filter(Boolean);
  if (parts.length <= 1) return host;
  // second-level label: publicnode.com → "publicnode"; infura.io → "infura".
  return parts[parts.length - 2];
}
```

- [ ] **Step 6: Run it, verify it passes** — `npm test`. Expected: PASS.

- [ ] **Step 7: Commit** — `git add cmd/valve-node-app/web/{package.json,package-lock.json,vitest.config.ts,src/panelModel.ts,src/panelModel.test.ts}` then commit `test(web): add vitest + endpointNameFromUrl helper`.

---

## Task 2: `masterState` — gateway → power-button state

**Files:**
- Modify: `cmd/valve-node-app/web/src/panelModel.ts`
- Test: `cmd/valve-node-app/web/src/panelModel.test.ts`

**Interfaces:**
- Consumes: `GatewayView` from `./api` (fields: `status.State: ContainerState`, `actions: string[] | null`, `blocked?: string`).
- Produces:
```ts
export type PowerTone = "on" | "off" | "blocked";
export interface MasterState { tone: PowerTone; label: string; sub: string; actions: string[]; blocked?: string; }
export function masterState(gw: GatewayView | null): MasterState;
```

- [ ] **Step 1: Write the failing test** — append to `panelModel.test.ts`:

```ts
import { masterState } from "./panelModel";
import type { GatewayView } from "./api";

const gw = (over: Partial<GatewayView>): GatewayView => ({
  id: "default", label: "", containerName: "", placement: { targetId: "local", backend: "docker" },
  status: { State: "running" } as GatewayView["status"], docker: {} as GatewayView["docker"],
  baseUrl: "", tls: {} as GatewayView["tls"], networks: [], actions: [], wipeDiscards: "",
  config: {} as GatewayView["config"], ...over,
});

describe("masterState", () => {
  it("running gateway is green with a stop action", () => {
    const m = masterState(gw({ status: { State: "running" } as GatewayView["status"], actions: ["stop", "restart"] }));
    expect(m.tone).toBe("on"); expect(m.label).toBe("Running"); expect(m.actions).toContain("stop");
  });
  it("stopped gateway is red and offers start", () => {
    const m = masterState(gw({ status: { State: "created-but-stopped" } as GatewayView["status"], actions: ["start"] }));
    expect(m.tone).toBe("off"); expect(m.actions).toContain("start");
  });
  it("blocked gateway surfaces the reason and offers nothing", () => {
    const m = masterState(gw({ status: { State: "unknown" } as GatewayView["status"], actions: [], blocked: "Docker unreachable" }));
    expect(m.tone).toBe("blocked"); expect(m.blocked).toBe("Docker unreachable");
  });
  it("null gateway (nothing set up) is off", () => {
    expect(masterState(null).tone).toBe("off");
  });
});
```

- [ ] **Step 2: Run it, verify it fails** — `npm test`. Expected: FAIL.

- [ ] **Step 3: Implement** — append to `panelModel.ts`:

```ts
import type { GatewayView } from "./api";

export type PowerTone = "on" | "off" | "blocked";
export interface MasterState { tone: PowerTone; label: string; sub: string; actions: string[]; blocked?: string; }

// masterState maps a gateway to the master power button. tone drives the color
// (green/red/grey); actions/blocked come straight from the server so the button
// never offers an impossible transition.
export function masterState(gw: GatewayView | null): MasterState {
  if (!gw) return { tone: "off", label: "Not set up", sub: "Press to set up your endpoint", actions: [] };
  const actions = gw.actions ?? [];
  if (gw.blocked) return { tone: "blocked", label: "Unavailable", sub: gw.blocked, actions, blocked: gw.blocked };
  const count = gw.networks?.length ?? 0;
  if (gw.status.State === "running") {
    return { tone: "on", label: "Running", sub: `${count} network${count === 1 ? "" : "s"} served`, actions };
  }
  return { tone: "off", label: "Stopped", sub: count ? `${count} network${count === 1 ? "" : "s"} configured` : "Press to start", actions };
}
```

- [ ] **Step 4: Run it, verify it passes** — `npm test`. Expected: PASS.

- [ ] **Step 5: Commit** — `feat(web): masterState power-button mapping`.

---

## Task 3: `healthClass` + `capabilityCells`

**Files:**
- Modify: `cmd/valve-node-app/web/src/panelModel.ts`
- Test: `cmd/valve-node-app/web/src/panelModel.test.ts`

**Interfaces:**
- Produces:
```ts
export type HealthClass = "stable" | "occasional" | "frequent" | "off";
// slowRate is the fraction (0..1) of recent requests that were slow/errored; unknown → undefined.
export function healthClass(input: { running: boolean; serviceable: boolean; slowRate?: number }): HealthClass;

export interface CapCell { key: string; label: string; lit: boolean; hot: boolean; }
// statuses maps capability key → "supported"|"unsupported"|"inconclusive"|... ; "http" is passed in
// pre-synthesized from reachability by the caller (there is no "http" Capability row).
export function capabilityCells(statuses: Record<string, string>): CapCell[];
```

- [ ] **Step 1: Write the failing tests**:

```ts
import { healthClass, capabilityCells } from "./panelModel";

describe("healthClass", () => {
  it("off when the gateway is not running", () => {
    expect(healthClass({ running: false, serviceable: true })).toBe("off");
  });
  it("frequent when running but not serviceable", () => {
    expect(healthClass({ running: true, serviceable: false })).toBe("frequent");
  });
  it("stable when serviceable and slow requests are rare/unknown", () => {
    expect(healthClass({ running: true, serviceable: true })).toBe("stable");
    expect(healthClass({ running: true, serviceable: true, slowRate: 0.02 })).toBe("stable");
  });
  it("occasional then frequent as the slow rate climbs", () => {
    expect(healthClass({ running: true, serviceable: true, slowRate: 0.15 })).toBe("occasional");
    expect(healthClass({ running: true, serviceable: true, slowRate: 0.6 })).toBe("frequent");
  });
});

describe("capabilityCells", () => {
  it("lights supported caps in fixed order, archive is the hot one", () => {
    const cells = capabilityCells({ http: "supported", ws: "supported", archive: "supported", trace: "unsupported" });
    expect(cells.map((c) => c.key)).toEqual(["http", "ws", "archive", "trace"]);
    expect(cells.find((c) => c.key === "archive")).toMatchObject({ lit: true, hot: true });
    expect(cells.find((c) => c.key === "trace")).toMatchObject({ lit: false });
  });
});
```

- [ ] **Step 2: Run it, verify it fails.**

- [ ] **Step 3: Implement** — append to `panelModel.ts`:

```ts
export type HealthClass = "stable" | "occasional" | "frequent" | "off";

// healthClass turns coarse signals into the dot's motion. Stillness = health, so
// a serviceable endpoint with rare slow requests is "stable" (no animation).
// Motion frequency tracks the slow-request rate. Thresholds: <10% stable,
// 10–40% occasional, >40% frequent. Not-serviceable while running is "frequent".
export function healthClass(input: { running: boolean; serviceable: boolean; slowRate?: number }): HealthClass {
  if (!input.running) return "off";
  if (!input.serviceable) return "frequent";
  const r = input.slowRate ?? 0;
  if (r > 0.4) return "frequent";
  if (r >= 0.1) return "occasional";
  return "stable";
}

export interface CapCell { key: string; label: string; lit: boolean; hot: boolean; }
const CAP_ORDER: { key: string; label: string; hot?: boolean }[] = [
  { key: "http", label: "HTTP" }, { key: "ws", label: "WS" },
  { key: "archive", label: "Archive", hot: true }, { key: "trace", label: "Trace" },
];

// capabilityCells folds probed capability statuses into the fixed-order meter.
// A cell is "lit" when supported; "hot" marks the standout (archive) when lit.
export function capabilityCells(statuses: Record<string, string>): CapCell[] {
  return CAP_ORDER.map(({ key, label, hot }) => {
    const lit = statuses[key] === "supported";
    return { key, label, lit, hot: !!hot && lit };
  });
}
```

- [ ] **Step 4: Run it, verify it passes.**

- [ ] **Step 5: Commit** — `feat(web): healthClass + capabilityCells helpers`.

---

## Task 4: Config-mutation helpers (add/remove network, add/remove/edit upstream)

**Files:**
- Modify: `cmd/valve-node-app/web/src/panelModel.ts`
- Test: `cmd/valve-node-app/web/src/panelModel.test.ts`

**Background:** `GatewayConfig` carries `Networks: GatewayNetwork[]` (PascalCase; see `home.ts` `internalTLSConfig`, and read the exact `GatewayConfig` shape in `api.ts` before implementing — copy the object through, mutate only `Networks`). A `GatewayNetwork` = `{ ChainID: number; Upstreams: GatewayUpstream[] }`. These helpers return a **new** config; the caller persists with `api.putGatewayConfig(gid, cfg)` then `api.provisionGateway(gid)`.

**Interfaces:**
- Produces:
```ts
export function withNetwork(cfg: GatewayConfig, chainId: number, upstreams: GatewayUpstream[]): GatewayConfig;
export function withoutNetwork(cfg: GatewayConfig, chainId: number): GatewayConfig;
export function withUpstream(cfg: GatewayConfig, chainId: number, up: GatewayUpstream): GatewayConfig; // add or replace by ID
export function withoutUpstream(cfg: GatewayConfig, chainId: number, upstreamId: string): GatewayConfig;
```

- [ ] **Step 1: Write failing tests** covering: adding a network appends it; adding a duplicate chainId replaces its upstreams; removing a network drops it; `withUpstream` replaces an existing ID and appends a new one; `withoutUpstream` removes by ID and leaves the network present. (Build fixtures with a minimal `GatewayConfig` cast, as in Task 2.) Assert the input config is not mutated (immutability).

- [ ] **Step 2: Run it, verify it fails.**

- [ ] **Step 3: Implement** the four helpers with structural cloning (`{ ...cfg, Networks: … }`), never mutating inputs. Import `GatewayConfig, GatewayNetwork, GatewayUpstream` from `./api`.

- [ ] **Step 4: Run it, verify it passes.**

- [ ] **Step 5: Commit** — `feat(web): gateway-config mutation helpers`.

---

## Task 5: Panel scaffold — module, route, icons, CSS bands, list skeleton

**Files:**
- Create: `cmd/valve-node-app/web/src/panel.ts`
- Create: `cmd/valve-node-app/web/src/panel.css`
- Modify: `cmd/valve-node-app/web/src/main.ts`

**Interfaces:**
- Produces: `export function renderPanel(root: HTMLElement): () => void`

- [ ] **Step 1: Register the route** — in `main.ts`: import `renderPanel`; add `case "panel": currentCleanup = mount((root) => renderPanel(root)); break;`; change the empty-hash/default case to mount the panel instead of home: `case "home": default: currentCleanup = mount((root) => renderPanel(root)); break;`. Keep every other route (rpc/targets/machine/settings/etc.) untouched so deep screens stay reachable during migration. In `parseHash`, `panel` falls through to the generic `return { screen }`.

- [ ] **Step 2: Create the icon sprite + shell** — `panel.ts`:

```ts
import "./panel.css";
import * as api from "./api";
import { onAction, escapeHtml, copyToClipboard } from "./ui";

// Inline SVG sprite (currentColor stroke) — cross-platform, no SF Symbols.
const SPRITE = `<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
  <symbol id="p-power" viewBox="0 0 24 24"><line x1="12" y1="3.5" x2="12" y2="11.5"/><path d="M7.5 7a7 7 0 1 0 9 0"/></symbol>
  <symbol id="p-globe" viewBox="0 0 24 24"><circle cx="12" cy="12" r="8.5"/><path d="M3.5 12h17M12 3.5c2.5 2.4 2.5 14.6 0 17M12 3.5c-2.5 2.4-2.5 14.6 0 17"/></symbol>
  <symbol id="p-ws" viewBox="0 0 24 24"><path d="M4 9h13l-3.5-3.5M20 15H7l3.5 3.5"/></symbol>
  <symbol id="p-archive" viewBox="0 0 24 24"><path d="M12 3 3 7.5l9 4.5 9-4.5L12 3ZM3 12l9 4.5 9-4.5M3 16.5 12 21l9-4.5"/></symbol>
  <symbol id="p-trace" viewBox="0 0 24 24"><circle cx="12" cy="12" r="5.5"/><path d="M12 1.5v3M12 19.5v3M1.5 12h3M19.5 12h3"/></symbol>
  <symbol id="p-lock" viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9.5" rx="2.2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></symbol>
  <symbol id="p-pencil" viewBox="0 0 24 24"><path d="M14 5.5l4.5 4.5M4 20l1.2-4.4L16 4.8a2 2 0 0 1 2.8 0l.4.4a2 2 0 0 1 0 2.8L8.4 18.8 4 20Z"/></symbol>
  <symbol id="p-trash" viewBox="0 0 24 24"><path d="M4 6.5h16M9.5 6.5V5a2 2 0 0 1 2-2h1a2 2 0 0 1 2 2v1.5M6.5 6.5l1 13.5h9l1-13.5M10 10.5v6M14 10.5v6"/></symbol>
  <symbol id="p-copy" viewBox="0 0 24 24"><rect x="9" y="9" width="11" height="11" rx="2.2"/><path d="M5 15V5a2 2 0 0 1 2-2h8"/></symbol>
  <symbol id="p-scale" viewBox="0 0 24 24"><path d="M12 3v18M7 21h10M12 5 5 8m7-3 7 3M5 8l-3 6a3 3 0 0 0 6 0L5 8Zm14 0-3 6a3 3 0 0 0 6 0l-3-6Z"/></symbol>
  <symbol id="p-refresh" viewBox="0 0 24 24"><path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3M19.5 4.5v4h-4"/></symbol>
  <symbol id="p-chevR" viewBox="0 0 24 24"><path d="M9.5 5.5l6.5 6.5-6.5 6.5"/></symbol>
  <symbol id="p-chevL" viewBox="0 0 24 24"><path d="M14.5 5.5 8 12l6.5 6.5"/></symbol>
  <symbol id="p-plus" viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></symbol>
</defs></svg>`;
const ic = (id: string) => `<svg class="p-i"><use href="#p-${id}"/></svg>`;
```

- [ ] **Step 3: Port the CSS** — create `panel.css` from the approved mockup `.superpowers/brainstorm/54055-1785634289/content/panel-v8.html` `<style>` block (namespace all classes under `.p-` to avoid colliding with `style.css`): the `.p-panel` single card, `.p-band` + flat dividers, `.p-lblrow`/`.p-seclbl`, `.p-row` + `.p-lead` (18px slot), `.p-dot` with `.stable/.occasional/.frequent/.off` and the `@keyframes p-glitch`, `.p-caps`/`.p-caprow`, `.p-gwurl`, `.p-srow`, `.p-remove`, `.p-dhead`/`.p-back`/`.p-dtitle` + hover pencil, `.p-power`/`.p-pbtn`. Center the card in `#content`. Keep the exact color tokens from the mockup `:root`.

- [ ] **Step 4: View-state machine + skeleton** — `panel.ts`:

```ts
type View = { name: "list" } | { name: "network"; chainId: number } | { name: "endpoint"; chainId: number; upstreamId: string };

export function renderPanel(root: HTMLElement): () => void {
  let gw: api.GatewayView | null = null;
  let view: View = { name: "list" };
  let err: string | null = null;
  let poll: number | null = null;

  root.innerHTML = SPRITE + `<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>`;
  const card = root.querySelector<HTMLElement>("#p-card")!;

  async function load(): Promise<void> {
    try { const res = await api.getGateways(); gw = primaryGateway(res.gateways); err = null; }
    catch (e) { err = message(e); }
    render();
  }
  function render(): void { card.innerHTML = renderView(); }
  function renderView(): string {
    if (err) return bandError(err);
    if (view.name === "network") return renderNetwork(gw, view.chainId);
    if (view.name === "endpoint") return renderEndpoint(gw, view.chainId, view.upstreamId);
    return renderList(gw);
  }

  onAction(card, (action, el) => { void handleAction(action, el); });
  async function handleAction(action: string, el: HTMLElement): Promise<void> { /* filled in Tasks 6-10 */ }

  void load();
  return () => { if (poll) window.clearInterval(poll); };
}

// primaryGateway: the panel manages one gateway — the local one, else the first.
function primaryGateway(gws: api.GatewayView[] | null): api.GatewayView | null {
  if (!gws || gws.length === 0) return null;
  return gws.find((g) => g.placement.targetId === "local") ?? gws[0];
}
function message(e: unknown): string { return e instanceof Error ? e.message : String(e); }
function bandError(m: string): string { return `<div class="p-band" style="padding:16px;color:var(--red)">${escapeHtml(m)}</div>`; }
function renderList(_gw: api.GatewayView | null): string { return `<div class="p-band" style="padding:16px">Panel loading…</div>`; }
function renderNetwork(_gw: api.GatewayView | null, _c: number): string { return ""; }
function renderEndpoint(_gw: api.GatewayView | null, _c: number, _u: string): string { return ""; }
```

> Note: `getGateways()` returns `GatewaysResponse`; read its exact field name (`.gateways`) in `api.ts` L869 and adjust `primaryGateway(res.gateways)` accordingly.

- [ ] **Step 5: Build** — `npm run build`. Expected: `tsc` clean, vite build succeeds.

- [ ] **Step 6: Run it** — use the `/run` skill (or `go run ./cmd/valve-node-app` and open the served UI). Confirm the empty hash lands on the panel and shows the "Panel loading…" card without console errors.

- [ ] **Step 7: Commit** — `feat(web): panel scaffold, route, icons, css`.

---

## Task 6: Networks list view (master power + rows)

**Files:** Modify `cmd/valve-node-app/web/src/panel.ts`.

**Interfaces:** Consumes `masterState`, `healthClass`, `capabilityCells` from `./panelModel`; `api.gatewayAction`, `api.provisionGateway`, `api.streamSetup`.

- [ ] **Step 1: Implement `renderList(gw)`** — header (brand dot + "Valve" + summary from `masterState().sub`); the master power band: a round `.p-pbtn` button whose class is the tone (`on`/`off`/`blocked`) showing `ic("power")`, with `masterState().label`; if `blocked`, show the reason text and no button action. Below, a `Networks` `.p-lblrow` then one `.p-row` per `gw.networks` (NetworkView): `.p-lead` health dot (`healthClass({ running: gw.status.State==='running', serviceable: nv.serviceable })`), the `nv.name`, a `.p-caps` meter (from that network's folded capabilities — for this task pass `{}` → all unlit; real caps arrive in Task 9), and a `chevR`. Add a final `.p-row.addr` "Add a network" with `data-action="add-network"`. Escape `nv.name`.

- [ ] **Step 2: Wire the power button + drill-in** in `handleAction`:

```ts
if (action === "power" && gw) {
  const m = masterState(gw);
  if (m.tone === "blocked") return;
  if (gw.status.State === "running" && m.actions.includes("stop")) { await runAction(gw.id, "stop"); return; }
  if (m.actions.includes("start")) { await runAction(gw.id, "start"); return; }
  if (m.actions.includes("create")) { await provision(gw.id); return; }
  return;
}
if (action === "open-network") { view = { name: "network", chainId: Number(el.dataset.chainId) }; render(); return; }
```

Add `runAction` (mirrors rpc.ts L1743): sets a `busy` flag, `await api.gatewayAction(gid, kind)`, catch→`err`, then `await load()`. Add `provision` (mirrors rpc.ts): `const started = await api.provisionGateway(gid)`, then `api.streamSetup(started.targetId, ev => { if (ev.err || (ev.stepId === "run" && ev.done)) { stop?.(); void load(); } })`, storing the unsub in a module/closure var cleared on cleanup. Rows carry `data-action="open-network" data-chain-id="${nv.chainId}"`; the button carries `data-action="power"`.

- [ ] **Step 3: Build** — `npm run build` clean.

- [ ] **Step 4: Run it** — set up a local gateway first if none (Task 7's one-click, or existing `#/rpc`), then load the panel: confirm the power button reflects running/stopped and toggles the gateway (watch it actually start/stop — verify by running, not by the label), networks list renders, clicking a row drills into an (empty) network view.

- [ ] **Step 5: Commit** — `feat(web): panel networks list + power button`.

---

## Task 7: Empty / blocked / one-click set-up

**Files:** Modify `cmd/valve-node-app/web/src/panel.ts`.

**Background:** Model the one-click on `home.ts` `setupEndpoint()` (see plan spec §6 of the API reference / `home.ts` L156-310). Reuse `home.ts` exports if practical: `import { setupEndpoint } from "./home"` if it's exported and side-effect-free enough; otherwise replicate the sequence: `listTargets` → `addTarget({id:"local",mode:"local"})` if absent → `getContainers("local")` docker gate → `createGateway({id:"default",placement:{targetId:"local",backend:"docker"},config:internalTLSConfig([])})` → for each of Ethereum(1)/PulseChain(369): `knownSet(gid,chainId)` → assemble `GatewayNetwork[]` → `putGatewayConfig(gid, internalTLSConfig(networks))` → `provisionGateway` → `streamSetup` to `"run"`/done.

- [ ] **Step 1: Empty state** — when `gw === null`, `renderList` shows a centered hero: the power button (tone `off`, big), "Set up my endpoint" copy, `data-action="setup"`. Devnet excluded (opt-in).

- [ ] **Step 2: Blocked state** — when `gw.blocked` is set (e.g. Docker unreachable), `masterState().tone === "blocked"`: render the button greyed, show `blocked` as the sub-line, and if `gw.hint` exists show it under. No power action fires (Task 6 already guards).

- [ ] **Step 3: Wire `setup`** — in `handleAction`, `case "setup"`: run the sequence above with a simple in-card progress line (append setup stream lines to a status element); on completion `await load()` and land on the list. Disable the button while running.

- [ ] **Step 4: Build** — clean.

- [ ] **Step 5: Run it** — on a machine with **no** gateway, confirm the panel shows "Set up my endpoint", the click provisions a real gateway end-to-end (watch the stream), and the panel then shows Ethereum + PulseChain running. Then stop Docker (or simulate) and confirm the blocked reason renders instead of a dead button.

- [ ] **Step 6: Commit** — `feat(web): panel empty/blocked states + one-click setup`.

---

## Task 8: Backend — optional endpoint name

**Files:**
- Modify: `internal/catalog/gateway.go` (the `GatewayUpstream`/upstream struct)
- Modify: `internal/server/gateways.go` (`resolveGateway`/upstream→`UpstreamView` mapping)
- Modify: `cmd/valve-node-app/web/src/api.ts` (`GatewayUpstream` + `UpstreamView` types)
- Test: `internal/server/gateways_test.go` (or the nearest existing gateway test)

**Interfaces:** Adds an optional `Name string` to the stored upstream and threads it to `UpstreamView.label` (use `Name` when non-empty, else the existing derived label).

- [ ] **Step 1: Find the structs** — `grep -n "Endpoint" internal/catalog/gateway.go` and locate the Go struct backing `GatewayUpstream` and the function building `UpstreamView` in `gateways.go` (the resolve path referenced at gateways.go ~L694+).

- [ ] **Step 2: Write a failing Go test** — a gateway config with an upstream whose `Name: "my node"` resolves to an `UpstreamView` with `Label == "my node"`; an upstream with empty `Name` keeps the derived label. Run `go test ./internal/server/ -run Upstream -v` → FAIL.

- [ ] **Step 3: Implement** — add `Name string \`json:"Name,omitempty"\`` to the upstream struct; in the resolver, `label := up.Name; if label == "" { label = derivedLabel }`. Keep JSON round-trip stable.

- [ ] **Step 4: Run tests** — `go test ./internal/... ` green.

- [ ] **Step 5: Mirror the TS types** — add `Name?: string` to `GatewayUpstream` in `api.ts`; `UpstreamView.label` already carries the resolved name (no change needed there). `npm run build` clean.

- [ ] **Step 6: Commit** — `feat(gateway): optional per-upstream Name surfaced as label`.

---

## Task 9: Network detail view

**Files:** Modify `cmd/valve-node-app/web/src/panel.ts`.

**Interfaces:** Consumes `api.getGatewayCapabilities`, `api.verifyGatewayTls`, `withoutNetwork` (Task 4), `capabilityCells`. Uses `confirmModal`, `copyToClipboard` from `ui.ts`.

- [ ] **Step 1: Implement `renderNetwork(gw, chainId)`** — find the `NetworkView` by `chainId`. Header: `.p-back` (`data-action="back"`), health dot, `nv.name` (NO rename pencil — networks are catalog identities). Bands:
  - **Gateway** `.p-lblrow` "Gateway · balanced across all" + lock (`data-action="verify-tls"`, green when `gw.tls` ok — read `TlsView` shape in api.ts) + copy (`data-action="copy-url" data-url="${nv.url}"`); the `nv.url` as `.p-gwurl`.
  - **Endpoints · N** — one `.p-row` per `nv.upstreams` (UpstreamView): dot (health from `up.problem ? 'frequent' : 'stable'` for now), `up.label`, chevR, `data-action="open-endpoint" data-chain-id data-upstream-id="${up.id}"`. Plus `.p-row.addr` "Add endpoint" `data-action="add-endpoint"`.
  - **Capabilities** — folded from `getGatewayCapabilities` for this chain's upstreams (union). Cache the capabilities on first render of the detail; pass synthesized `http` from `EndpointCapabilities.reachable` and `ws/archive/trace` from the `capabilities` array (see rpc.ts `statusOf`, L1195-1202) into `capabilityCells`.
  - **Status** `.p-lblrow` "Status" + refresh (`data-action="recheck"`); rows Health (dot + word) and Chain head (`fmtInt` of the network's head if available; else omit).
  - **Remove network** `.p-remove` `data-action="remove-network"`.

- [ ] **Step 2: Wire** `back` (→ `view={name:"list"}; render()`), `copy-url` (`await copyToClipboard(el.dataset.url!)`, flash the icon), `verify-tls` (`await api.verifyGatewayTls(gw.id)`, reflect `.ok` on the lock), `open-endpoint` (→ endpoint view), `remove-network` (`confirmModal({title:"Remove network",body:…,confirmLabel:"Remove",danger:true})` → `api.putGatewayConfig(gw.id, withoutNetwork(gw.config, chainId))` → `api.provisionGateway(gw.id)` → `view={name:"list"}` → `load()`), `recheck` (`getGatewayCapabilities(gid, true)` + `load()`).

- [ ] **Step 3: Build** — clean.

- [ ] **Step 4: Run it** — drill into a network: confirm the dialable URL copies, the lock reflects a real `tls/verify`, capabilities light up per real probe, and **Remove network** actually removes the chain (verify the gateway re-provisions without it — by running, watch it disappear).

- [ ] **Step 5: Commit** — `feat(web): panel network detail`.

---

## Task 10: Endpoint detail + add flows

**Files:** Modify `cmd/valve-node-app/web/src/panel.ts`.

**Interfaces:** Consumes `endpointNameFromUrl`, `withUpstream`, `withoutUpstream`, `withNetwork` (Tasks 1/4); `api.knownSet`, `api.getCatalog`, `api.getGatewayCapabilities`; `openModal`, `confirmModal`.

- [ ] **Step 1: `renderEndpoint(gw, chainId, upstreamId)`** — find the UpstreamView. Header: back, health dot, `up.label` with hover-rename pencil (`data-action="rename-endpoint"`). Bands: **Address** label + copy; the `up.endpoint` URL as `.p-gwurl` (or an inline editable field — `data-action="edit-address"`). **Capabilities** for this one upstream. **Status** (Health, and "behind N blocks" if `headLag` from health is available; else Health only). **Remove endpoint** `.p-remove` `data-action="remove-endpoint"`.

- [ ] **Step 2: Wire endpoint actions** — `rename-endpoint`: `openModal` with a text input prefilled with `up.label`; on save build the updated `GatewayUpstream` (set `Name`) and `api.putGatewayConfig(gw.id, withUpstream(gw.config, chainId, up))` → `provisionGateway` → `load()`. `edit-address`: same pattern, updating `Endpoint`. `remove-endpoint`: `confirmModal` → `withoutUpstream` → put+provision → back to network view.

- [ ] **Step 3: Add endpoint** — `add-endpoint` (from network view): `openModal` with a URL input; on submit derive `endpointNameFromUrl(url)` as the default `Name`, build `{ ID: crypto.randomUUID(), Kind: "external", Endpoint: url, Local: false, RecentOnly: false, Name }`, `withUpstream(gw.config, chainId, up)` → put + provision → `load()`.

- [ ] **Step 4: Add network** — `add-network` (from list): `openModal` listing catalog chains (`getCatalog().networks` mapped to chainId/name, or the fixed 4) **not already in `gw.networks`**, Devnet last; on pick, `knownSet(gw.id, chainId)` → assemble `GatewayNetwork` upstreams (filter `!alreadyAdded`, `Kind:"external"`) → `withNetwork(gw.config, chainId, upstreams)` → put + provision → open that network's detail.

- [ ] **Step 5: Build** — clean.

- [ ] **Step 6: Run it** — add a network from the catalog and confirm it appears + serves; add an external endpoint by URL and confirm it's auto-named from the domain and joins the balancer; rename it and confirm the label persists after reload (needs Task 8); remove an endpoint and confirm it drops.

- [ ] **Step 7: Commit** — `feat(web): panel endpoint detail + add network/endpoint`.

---

## Task 11: Live dots via polling + refresh

**Files:** Modify `cmd/valve-node-app/web/src/panel.ts`.

- [ ] **Step 1: Poll** — in `renderPanel`, after first `load()`, set `poll = window.setInterval(refreshHealth, 5000)`. `refreshHealth` fetches `api.getGatewayAnalytics(gw.id)` (and/or capabilities), computes per-network `slowRate` from `NetworkAnalytics` (`failed`/`received`, or the latency buckets — the fraction of requests slower than a threshold `le`), stores it, and re-renders only if a class changed. Guard against overlap (skip if a fetch is in flight). Ensure the cleanup returned by `renderPanel` clears the interval AND any open `streamSetup` unsub.

- [ ] **Step 2: Feed `healthClass`** — pass the computed `slowRate` into `healthClass` in `renderList`/`renderNetwork` so dots go stable→occasional→frequent from real traffic; the `↻` refresh action forces an immediate `refreshHealth`.

- [ ] **Step 3: Build** — clean.

- [ ] **Step 4: Run it** — generate some load against a chain (or point one endpoint at a deliberately slow/unreachable upstream) and confirm its dot starts twitching, and a healthy one stays perfectly still. Confirm no leaked intervals when navigating away (the cleanup runs).

- [ ] **Step 5: Commit** — `feat(web): panel live health dots`.

---

## Self-Review

**Spec coverage** (against `2026-08-01-easy-button-panel-design.md`):
- §2 shell (self-contained 360px, native-later) → Tasks 5 (card, fixed width, internal nav not hash).
- §3 concept mapping (power=lifecycle, network=NetworkView, endpoint=upstream, gateway=url, lock=tls, caps, dot) → Tasks 2/3/6/9/10/11.
- §4 visual language (one card, flat bands, 18px lead, icons, caps meter, stillness=health, hover-rename, red remove) → Tasks 5 (CSS/icons) + all render tasks; dots in 3/11.
- §5 hierarchy list→network→endpoint → Tasks 6/9/10.
- §6 flows (add network, add endpoint, off/blocked/empty, one-click, devnet-last) → Tasks 7/10.
- §7 approach (new module, api.ts, server-driven actions, SSE, verify-by-running) → all tasks; corrections block encodes the deviations.
- Endpoint naming (auto + optional rename) → Tasks 1/8/10.

**Deviations from the mockups (intentional, backend-driven)** — captured in the "Three corrections" block: no per-network/per-endpoint toggles; network titles not renamable; endpoint rename needs Task 8.

**Placeholder scan:** `handleAction` is introduced as a stub in Task 5 and filled incrementally in 6/7/9/10/11 — each of those tasks specifies the exact cases it adds. Capability-cell wiring, TlsView, GatewaysResponse, and GatewayConfig shapes are referenced by exact type name with an instruction to read the fields in `api.ts` (they exist; the plan doesn't invent field names it hasn't confirmed).

**Type consistency:** `masterState`, `healthClass`, `capabilityCells`, `withNetwork/withoutNetwork/withUpstream/withoutUpstream`, `endpointNameFromUrl`, `renderPanel`, `primaryGateway` names are used identically across tasks. Action strings (`power`, `open-network`, `back`, `open-endpoint`, `copy-url`, `verify-tls`, `recheck`, `remove-network`, `rename-endpoint`, `edit-address`, `remove-endpoint`, `add-endpoint`, `add-network`, `setup`) are each defined where introduced and reused verbatim.

**Open item to resolve at implementation time (not a blocker):** the exact `slowRate` formula from `NetworkAnalytics`/latency buckets (Task 11) — pick a concrete `le` threshold (e.g. requests slower than 1s ÷ total) when the real analytics shape is in front of you.
