# React 19 Frontend Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the vanilla-TS `cmd/valve-node-app/web` frontend to React 19 without ever breaking `master`, ending on a single paradigm.

**Architecture:** A React shell (`main.tsx` → `<App>`) wrapped in `QueryClientProvider` + `HashRouter`. Screens convert one at a time; any not-yet-converted screen runs through a `<LegacyScreen>` bridge that mounts the old `render(root): cleanup` into a ref'd div. When the last screen converts, the bridge and the old signature are deleted.

**Tech Stack:** React 19, Vite + `@vitejs/plugin-react`, `react-router-dom` (v7, React-19 compatible) `HashRouter`, `@tanstack/react-query` v5, vitest + `@testing-library/react` + jsdom + `@vitest/coverage-v8`. Reused as-is: `api.ts`, `panelModel.ts`, `theme.ts`, `style.css`, `panel.css`.

## Global Constraints

- **Only these view/router/data libs:** React 19, `react-router-dom` v7 `HashRouter`, `@tanstack/react-query` v5. No others without revisiting the spec.
- **`master` builds and every screen renders at every commit.** A screen is either fully React or bridged via `<LegacyScreen>` — never mid-air.
- **Tests land with every task.** Extract logic to pure modules with vitest unit tests; cover components with `@testing-library/react`. `noUnusedLocals`/`noUnusedParameters` are on — keep them green.
- **No CDN / runtime code fetch.** Everything bundles into `dist/` (committed; `//go:embed all:web/dist`). Run `npm run build` at the end of any task that changes the shipped bundle and commit the rebuilt `dist/`.
- **Preserve every `#/…` route and redirect** exactly: `#/`,`#/panel`→panel; `#/rpc`,`#/targets`,`#/settings`; `#/machine/:id`,`#/security/:id`,`#/diag/:id`,`#/analytics/:id`; redirects `#/setup|dash|logs|services/:id → #/machine/:id`; `#/machine`(no id)→`#/targets`; `#/analytics`(no id)→`#/rpc`.
- **Stage files explicitly (never `git add -A`).** The repo's `.gitignore` mis-points at `cmd/valve-node/web/node_modules/`; Task 1 fixes it, but until then `node_modules` is trackable.
- **Reuse, don't rewrite** `api.ts`, `panelModel.ts`, `theme.ts`, and the CSS token system.

---

## File Structure

- `src/main.tsx` (new) — replaces `main.ts`; mounts providers + `<App>`.
- `src/App.tsx` (new) — shell (topbar/nav) + `<Routes>`.
- `src/components/LegacyScreen.tsx` (new) — bridge for unconverted screens.
- `src/lib/queryClient.ts` (new) — the shared `QueryClient` config.
- `src/hooks/useEventStream.ts` (new) — SSE (`streamSetup`) wrapped as a hook.
- `src/hooks/gateway.ts` (new) — React Query hooks over `api.ts` (gateways, health, capabilities, analytics) + mutations. Grows as screens need more.
- `src/screens/Panel/*` (new) — the panel as components (Task 4).
- `src/screens/<Name>.tsx` (new, one per converted screen) — replaces each `*.ts`.
- Deleted as they convert: `panel.ts`, then each legacy `*.ts` screen; finally `LegacyScreen.tsx` + the `render(root)` helpers in `ui.ts`.
- Kept: `api.ts`, `panelModel.ts`(+test), `theme.ts`, `style.css`, `panel.css`. `ui.ts` shrinks to pure helpers (`fmtInt`,`fmtPct`,`fmtDuration`,`fmtBytes`,`badge`→component,`copyToClipboard`,`LEARN_ROOT`); `escapeHtml`/`onAction`/`openModal`/`dropdown`/`renderShell` retire once no legacy screen uses them.

---

### Task 1: Tooling foundation (deps, config, gitignore)

**Files:**
- Modify: `.gitignore` (repo root, line 7)
- Modify: `cmd/valve-node-app/web/package.json`
- Modify: `cmd/valve-node-app/web/vite.config.ts`
- Modify: `cmd/valve-node-app/web/tsconfig.json`
- Modify: `cmd/valve-node-app/web/vitest.config.ts`
- Create: `cmd/valve-node-app/web/src/test/setup.ts`
- Create: `cmd/valve-node-app/web/src/smoke.test.tsx`

**Interfaces:**
- Produces: a toolchain where `.tsx` compiles, JSX + React Testing Library run under jsdom, and coverage reports. Later tasks rely on `@testing-library/react`, `@tanstack/react-query`, `react-router-dom`, `react`/`react-dom` being installed.

- [ ] **Step 1: Fix the gitignore path.** In repo-root `.gitignore`, change `cmd/valve-node/web/node_modules/` to `cmd/valve-node-app/web/node_modules/`. Verify: `git status --short | grep node_modules` prints nothing.

- [ ] **Step 2: Add dependencies.** In `web/`, run:
```bash
npm i react@^19 react-dom@^19 react-router-dom@^7 @tanstack/react-query@^5
npm i -D @vitejs/plugin-react @types/react @types/react-dom \
  @testing-library/react @testing-library/dom @testing-library/jest-dom \
  jsdom @vitest/coverage-v8
```

- [ ] **Step 3: Wire the Vite React plugin.** Replace `vite.config.ts`:
```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: { outDir: "dist" },
});
```

- [ ] **Step 4: Enable JSX in TS.** In `tsconfig.json` `compilerOptions`, add `"jsx": "react-jsx"`.

- [ ] **Step 5: Configure vitest for jsdom + coverage.** Replace `vitest.config.ts`:
```ts
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.{ts,tsx}"],
    setupFiles: ["src/test/setup.ts"],
    coverage: { provider: "v8", reporter: ["text", "html"], include: ["src/**/*.{ts,tsx}"] },
  },
});
```

- [ ] **Step 6: Test setup file.** Create `src/test/setup.ts`:
```ts
import "@testing-library/jest-dom/vitest";
```

- [ ] **Step 7: Write the smoke test.** Create `src/smoke.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

function Hello() {
  return <p>hello react</p>;
}

describe("toolchain", () => {
  it("renders a component under jsdom", () => {
    render(<Hello />);
    expect(screen.getByText("hello react")).toBeInTheDocument();
  });
});
```

- [ ] **Step 8: Run it.** `npx vitest run` → smoke test + existing `panelModel.test.ts` all pass. `npx tsc --noEmit` → clean.

- [ ] **Step 9: Commit** (stage explicitly): `.gitignore`, `web/package.json`, `web/package-lock.json`, `web/vite.config.ts`, `web/tsconfig.json`, `web/vitest.config.ts`, `web/src/test/setup.ts`, `web/src/smoke.test.tsx`.
```bash
git commit -m "build(web): React 19 + RTL toolchain; fix node_modules gitignore path"
```

---

### Task 2: App shell, router, and the LegacyScreen bridge

**Files:**
- Create: `src/lib/queryClient.ts`
- Create: `src/components/LegacyScreen.tsx`
- Create: `src/App.tsx`
- Create: `src/main.tsx`
- Modify: `index.html` (script src)
- Create: `src/components/LegacyScreen.test.tsx`
- Create: `src/App.test.tsx`
- Delete: `src/main.ts` (after `main.tsx` verified)

**Interfaces:**
- Consumes: every legacy `render*` export (`renderPanel`, `renderRPC`, `renderTargets`, `renderSettings`, `renderMachine`, `renderSecurity`, `renderDiagnostics`, `renderAnalytics`) from their current modules; `initTheme` from `theme.ts`.
- Produces: `<LegacyScreen render={(el)=>cleanup}>`; the `<App>` route table; the running provider tree. Later screen tasks replace one `<LegacyScreen>` route element with a real component and delete that legacy module.

- [ ] **Step 1: Query client.** Create `src/lib/queryClient.ts`:
```ts
import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false, refetchOnWindowFocus: false },
  },
});
```

- [ ] **Step 2: Bridge test first.** Create `src/components/LegacyScreen.test.tsx`:
```tsx
import { render } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { LegacyScreen } from "./LegacyScreen";

describe("LegacyScreen", () => {
  it("mounts the legacy render and runs cleanup on unmount", () => {
    const cleanup = vi.fn();
    const legacy = vi.fn((el: HTMLElement) => {
      el.textContent = "legacy";
      return cleanup;
    });
    const { unmount, container } = render(<LegacyScreen render={legacy} />);
    expect(legacy).toHaveBeenCalledTimes(1);
    expect(container.textContent).toContain("legacy");
    unmount();
    expect(cleanup).toHaveBeenCalledTimes(1);
  });
});
```

- [ ] **Step 3: Implement the bridge.** Create `src/components/LegacyScreen.tsx`:
```tsx
import { useEffect, useRef } from "react";

export type LegacyRender = (root: HTMLElement) => (() => void) | void;

// LegacyScreen mounts a not-yet-converted vanilla screen (its render(root):
// cleanup function) into a React-owned div, mirroring main.ts's old mount():
// call render on mount, run the returned cleanup on unmount, then clear the
// node. `render` MUST be stable (useCallback keyed on its inputs) so the
// effect doesn't re-run every parent render.
export function LegacyScreen({ render }: { render: LegacyRender }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current!;
    const cleanup = render(el);
    return () => {
      try {
        cleanup?.();
      } catch {
        // A screen's cleanup throwing must not block navigating away.
      }
      el.replaceChildren();
    };
  }, [render]);
  return <div ref={ref} />;
}
```

- [ ] **Step 4: App shell + routes.** Create `src/App.tsx`. Reproduce the topbar/nav from `ui.ts:renderShell` and `setActiveNav`, and the exact route/redirect table:
```tsx
import { useCallback } from "react";
import { Link, Navigate, Route, Routes, useLocation, useParams } from "react-router-dom";
import { LegacyScreen } from "./components/LegacyScreen";
import { renderPanel } from "./panel";
import { renderRPC } from "./rpc";
import { renderTargets } from "./targets";
import { renderSettings } from "./settings";
import { renderMachine } from "./machine";
import { renderSecurity } from "./security";
import { renderDiagnostics } from "./diag";
import { renderAnalytics } from "./analytics";

const NAV = [
  { to: "#/rpc", nav: "rpc", label: "RPC" },
  { to: "#/targets", nav: "targets", label: "Machines" },
  { to: "#/settings", nav: "settings", label: "Settings" },
];

function activeNav(screen: string): string {
  return screen === "machine"
    ? "targets"
    : screen === "home" || screen === "panel"
      ? "rpc"
      : screen;
}

// Bridge wrappers: memoize the legacy render so LegacyScreen's effect is stable.
function Bridge({ render }: { render: (el: HTMLElement) => (() => void) | void }) {
  return <LegacyScreen render={render} />;
}
function LegacyPanel() {
  return <Bridge render={useCallback((el: HTMLElement) => renderPanel(el), [])} />;
}
function LegacyRPC() {
  return <Bridge render={useCallback((el: HTMLElement) => renderRPC(el), [])} />;
}
function LegacyTargets() {
  return <Bridge render={useCallback((el: HTMLElement) => renderTargets(el), [])} />;
}
function LegacySettings() {
  return <Bridge render={useCallback((el: HTMLElement) => renderSettings(el), [])} />;
}
function LegacyMachine() {
  const { id } = useParams();
  return <Bridge render={useCallback((el: HTMLElement) => renderMachine(el, id!), [id])} />;
}
function LegacySecurity() {
  const { id } = useParams();
  return <Bridge render={useCallback((el: HTMLElement) => renderSecurity(el, id!), [id])} />;
}
function LegacyDiag() {
  const { id } = useParams();
  return <Bridge render={useCallback((el: HTMLElement) => renderDiagnostics(el, id!), [id])} />;
}
function LegacyAnalytics() {
  const { id } = useParams();
  return <Bridge render={useCallback((el: HTMLElement) => renderAnalytics(el, id!), [id])} />;
}
function RedirectToMachine() {
  const { id } = useParams();
  return <Navigate to={`/machine/${encodeURIComponent(id!)}`} replace />;
}

export function App() {
  const { pathname } = useLocation();
  const screen = pathname.split("/").filter(Boolean)[0] ?? "home";
  const nav = activeNav(screen);
  return (
    <div className="shell">
      <header className="topbar">
        <a className="brand" href="#/">
          valve-node-app
        </a>
        <nav className="nav">
          {NAV.map((n) => (
            <a key={n.nav} href={n.to} className={n.nav === nav ? "active" : undefined} data-nav={n.nav}>
              {n.label}
            </a>
          ))}
        </nav>
      </header>
      <main id="content" className="content">
        <Routes>
          <Route path="/" element={<LegacyPanel />} />
          <Route path="/panel" element={<LegacyPanel />} />
          <Route path="/rpc" element={<LegacyRPC />} />
          <Route path="/targets" element={<LegacyTargets />} />
          <Route path="/settings" element={<LegacySettings />} />
          <Route path="/machine/:id" element={<LegacyMachine />} />
          <Route path="/machine" element={<Navigate to="/targets" replace />} />
          <Route path="/security/:id" element={<LegacySecurity />} />
          <Route path="/diag/:id" element={<LegacyDiag />} />
          <Route path="/analytics/:id" element={<LegacyAnalytics />} />
          <Route path="/analytics" element={<Navigate to="/rpc" replace />} />
          {["setup", "dash", "logs", "services"].map((p) => (
            <Route key={p} path={`/${p}/:id`} element={<RedirectToMachine />} />
          ))}
          {["setup", "dash", "logs", "services"].map((p) => (
            <Route key={`${p}-noid`} path={`/${p}`} element={<Navigate to="/targets" replace />} />
          ))}
          <Route path="*" element={<LegacyPanel />} />
        </Routes>
      </main>
    </div>
  );
}
```

- [ ] **Step 5: Entry point.** Create `src/main.tsx`:
```tsx
import "./style.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { HashRouter } from "react-router-dom";
import { initTheme } from "./theme";
import { queryClient } from "./lib/queryClient";
import { App } from "./App";

initTheme();

createRoot(document.getElementById("app")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <HashRouter>
        <App />
      </HashRouter>
    </QueryClientProvider>
  </StrictMode>,
);
```

- [ ] **Step 6: Point index.html at the new entry.** In `index.html`, change `src="/src/main.ts"` to `src="/src/main.tsx"`.

- [ ] **Step 7: App routing test.** Create `src/App.test.tsx` — render `<App>` inside `MemoryRouter`-equivalent for HashRouter (use `HashRouter` and set `location.hash`), assert the nav renders and the active class tracks the hash. Mock the legacy render modules with `vi.mock` so the test doesn't touch `api.ts`. Minimum: asserts nav labels present; asserts `activeNav` mapping via a small exported pure helper test (export `activeNav`).
```tsx
import { describe, it, expect } from "vitest";
import { activeNav } from "./App";

describe("activeNav", () => {
  it("maps machine→targets and home/panel→rpc, else identity", () => {
    expect(activeNav("machine")).toBe("targets");
    expect(activeNav("home")).toBe("rpc");
    expect(activeNav("panel")).toBe("rpc");
    expect(activeNav("settings")).toBe("settings");
  });
});
```

- [ ] **Step 8: Delete `src/main.ts`.**

- [ ] **Step 9: Verify.** `npx tsc --noEmit` clean; `npx vitest run` green; `npm run build` succeeds; `npm run dev` (manual, in review) shows every route rendering its (bridged) screen and the nav highlighting correctly.

- [ ] **Step 10: Commit** (explicit stage incl. rebuilt `dist/`):
```bash
git commit -m "feat(web): React shell + HashRouter + LegacyScreen bridge (all screens bridged)"
```

---

### Task 3: Data hooks foundation (queries + SSE)

**Files:**
- Create: `src/hooks/useEventStream.ts`
- Create: `src/hooks/gateway.ts`
- Create: `src/hooks/gateway.test.tsx`
- Create: `src/hooks/useEventStream.test.tsx`

**Interfaces:**
- Consumes: `api.ts` (`getGateways`, `getGatewayAnalytics`, `getGatewayCapabilities`, `streamSetup`, `provisionGateway`, `gatewayAction`, `putGatewayConfig`, `wipeGateway`, and their types).
- Produces:
  - `useGateways()` → `UseQueryResult<GatewayView[]>` (key `["gateways"]`).
  - `useGatewayHealth(gid, enabled)` → analytics with `refetchInterval: 5000` (key `["gwHealth", gid]`).
  - `useGatewayCapabilities(gid, enabled)` → caps, `staleTime: Infinity` (key `["gwCaps", gid]`), plus a `refetch` for "recheck".
  - Mutations: `useGatewayAction()`, `usePutGatewayConfig()`, `useWipeGateway()` — each invalidating `["gateways"]`/health on success.
  - `useEventStream(targetId | null, onEvent)` — subscribes via `streamSetup`, unsubscribes on unmount/id-change; returns nothing (caller keeps its own state).

- [ ] **Step 1: SSE hook test** (fake stream): assert subscribe on mount, teardown on unmount, and re-subscribe when `targetId` changes. Mock `api.streamSetup` to return a stop spy.
- [ ] **Step 2: Implement `useEventStream`** — `useEffect` keyed on `targetId`; when non-null, call `api.streamSetup(targetId, onEvent)`, store the returned stop, call it in cleanup. Keep `onEvent` in a ref so a changing callback identity doesn't re-subscribe.
- [ ] **Step 3: Query hooks test** — wrap in a `QueryClientProvider` test helper; mock `api.getGateways` etc.; assert `useGateways` returns data and `useGatewayHealth` is disabled when `enabled=false`.
- [ ] **Step 4: Implement `src/hooks/gateway.ts`** with the queries + mutations above.
- [ ] **Step 5: Verify** `tsc` + `vitest` green.
- [ ] **Step 6: Commit** `feat(web): gateway query hooks + useEventStream`.

---

### Task 4: Convert the panel to React (pattern-setter)

**Files:**
- Create: `src/screens/Panel/Panel.tsx` (container: gateway load, view state list|network|endpoint, error/empty)
- Create: `src/screens/Panel/ListView.tsx`, `NetworkView.tsx`, `EndpointView.tsx`
- Create: `src/screens/Panel/PowerBand.tsx`, `HealthDot.tsx`, `CapabilityMeter.tsx`, `SettingsSheet.tsx`, `icons.tsx` (the SVG sprite/`Icon` component)
- Create: `src/screens/Panel/Panel.test.tsx` (+ per-component tests)
- Modify: `src/App.tsx` (replace `LegacyPanel` route element with `<Panel/>`; drop the `renderPanel` import)
- Delete: `src/panel.ts`

**Interfaces:**
- Consumes: `panelModel.ts` (`masterState`, `healthClass`, `capabilityCells`, `endpointNameFromUrl`, `networkSlowRate`, `endpointSlowRate`, `with*`/`without*`), the Task 3 hooks, `theme.ts` (`getThemePref`/`setThemePref`), `api.ts` types, `panel.css`.
- Produces: `<Panel/>`, the first fully-React screen. `SettingsSheet` (gear → appearance segmented control + Danger-zone wipe) as a component.

**Approach:** `src/panel.ts` is the behavioral source of truth — re-express its three views, the lifecycle two-path (start/stop/restart via `gatewayAction`; create/recreate via `provisionGateway` + `useSetupStream`; wipe via `wipeGateway`), add-network/endpoint/rename/edit modals, TLS verify, and the settings sheet — but driven by hooks and JSX state instead of `innerHTML` + `healthSignature`. **The `healthSignature`/`lastHealthSig` mechanism is deleted:** `useGatewayHealth` polls every 5s and the reconciler keeps the dot nodes, so animations don't restart. `escapeHtml` and `data-action` delegation are gone (JSX + `onClick`).

- [ ] **Step 1: `icons.tsx`** — export an `<Icon name=…/>` rendering the existing SVG symbols (port the `SPRITE` + `ic()` set incl. `gear`), and `import "./panel.css"`.
- [ ] **Step 2: `HealthDot`, `CapabilityMeter`** with unit tests driven by `panelModel` outputs (reuse `healthClass`, `capabilityCells`).
- [ ] **Step 3: `SettingsSheet`** — appearance segmented control (System/Light/Dark via `theme.ts`), Danger-zone Wipe shown only when the server lists `wipe`; test that clicking a theme option calls `setThemePref` and that Wipe is hidden when `actions` lacks `wipe`.
- [ ] **Step 4: `PowerBand`** — round button (primary transition from `masterState`) + secondary chips (server actions minus primary and `wipe`); test tone/label/chip set from a fixture `GatewayView`.
- [ ] **Step 5: `ListView` / `NetworkView` / `EndpointView`** — render from props + hooks; modals via a small `<Modal>` component (Task 4a may extract it; for now a local component). Tests: renders rows from a fixture gateway; empty state shows "Set up my endpoint".
- [ ] **Step 6: `Panel.tsx`** — wires hooks, view state (`useState<View>`), setup stream, mutations; recheck refetches caps+health.
- [ ] **Step 7: Swap the route.** In `App.tsx`, replace `LegacyPanel` with `<Panel/>` and remove the `renderPanel` import + `LegacyPanel` wrapper.
- [ ] **Step 8: Delete `src/panel.ts`.** Confirm nothing else imports it (`grep -rn "from \"./panel\"" src`).
- [ ] **Step 9: Verify** — `tsc` clean; `vitest` green; `npm run build`; **live-verify against the real binary + Docker** (repo rule): one-click setup, lifecycle (start/stop/restart/recreate/wipe), add/remove network+endpoint, rename, theme flip persists, health dots animate. Rebuild `dist/`.
- [ ] **Step 10: Commit** `feat(web): panel as React (deletes healthSignature/escapeHtml/data-action)`.

---

### Tasks 5–N: Convert remaining screens (one per task)

Each task follows the **same playbook** and is independently reviewable/testable. Order (small/self-contained → large, to build shared component vocabulary first):

5. `verdict.ts` (121) 6. `diag.ts` (165) 7. `security.ts` (142) 8. `logs.ts` (230) 9. `settings.ts` (275) 10. `targets.ts` (343) 11. `home.ts` (350) 12. `analytics.ts` (589) 13. `dashboard.ts` (506) 14. `machine.ts` (197, composes dashboard/logs/setup/devnet sections) 15. `services.ts` (668) 16. `wizard.ts` (928) 17. `rpc.ts` (2,799 — split into subcomponents).

**Playbook (applies to each):**

**Files:** Create `src/screens/<Name>.tsx` (+ subcomponents/tests as size warrants); Modify `src/App.tsx` (swap the `Legacy<Name>` route element for the component, drop the legacy import + wrapper); Delete the legacy `src/<name>.ts`.

- [ ] **Step 1:** Read the legacy module; identify its data reads (which `api.ts` calls), any `setInterval` polling → a query with `refetchInterval`, any `EventSource`/`streamSetup` → `useEventStream`, and any pure formatting/derivation → extract to a `*Model.ts` with unit tests.
- [ ] **Step 2:** Write failing tests first — a pure-logic unit test for any extracted model, and an RTL test asserting the screen renders its key states (loading/empty/error/populated) from mocked hooks/`api.ts`.
- [ ] **Step 3:** Implement the component(s) using the Task 3 hooks (add new query/mutation hooks to `src/hooks/*` as needed — with their own tests) and existing CSS classes. No `innerHTML`, no `escapeHtml`, no `data-action`.
- [ ] **Step 4:** Swap the route in `App.tsx`; delete the legacy module; `grep` to confirm no remaining imports.
- [ ] **Step 5:** `tsc` clean; `vitest` green (coverage up vs previous task); `npm run build`; live-verify any screen with real side effects (provisioning, streams, container actions) against the real binary. Rebuild `dist/`.
- [ ] **Step 6:** Commit `feat(web): <name> screen as React`.

**Per-screen notes (non-obvious specifics):**
- `machine.ts` composes `setup`/`dashboard`/`logs`/`devnet` sections — convert those it depends on first (dashboard #13, logs #8), then machine assembles them as child components.
- `services.ts` and `wizard.ts` carry the heaviest live/streaming logic (container actions, provision streams, fit-check maths) — lean on `useEventStream` + mutations; keep the fit-check maths in a tested pure module.
- `rpc.ts` (2,799) is oversized — split into `screens/Rpc/` subcomponents (endpoints list, known-set picker, TLS/trust panel, gateway lifecycle) each with tests; this is several steps, not one code block.
- `home.ts` exports `SETUP_CHAINS`/`internalTLSConfig` consumed by the panel — keep those exports (move to a shared `src/lib/gatewaySetup.ts` when `home.ts` is deleted; update the panel import).

---

### Task Final: Remove the bridge and dead helpers

**Files:** Delete `src/components/LegacyScreen.tsx`; Modify `src/App.tsx` (no more `Legacy*`); Modify `src/ui.ts` (delete `renderShell`, `onAction`, `openModal`/`closeModal`/`modalBody`/`confirmModal`, `dropdown`/`wireDropdowns`, `escapeHtml`, `footer` if now componentized — keep pure `fmt*`, `copyToClipboard`, `LEARN_ROOT`, and anything still imported).

- [ ] **Step 1:** `grep -rn "LegacyScreen\|renderShell\|onAction\|openModal\|escapeHtml\|data-action" src` → confirm zero non-test references before deleting each.
- [ ] **Step 2:** Delete the bridge and now-dead `ui.ts` exports (delete only symbols with zero references; `noUnusedLocals` + `tsc` are the backstop).
- [ ] **Step 3:** `tsc` clean; `vitest` green; `npm run build`; full manual pass over every route.
- [ ] **Step 4:** Commit `refactor(web): remove LegacyScreen bridge and vanilla render helpers — 100% React`.

---

## Self-Review

- **Spec coverage:** stack (Task 1), shell/router/bridge (Task 2), data layer incl. deleting `healthSignature` (Tasks 3–4), panel-first (Task 4), one-screen-per-PR with tests (Tasks 5–N), one-paradigm end state (Task Final), routes/redirects preserved (Task 2), coverage climbing (every task) — all present.
- **Type consistency:** `LegacyRender` return type `(() => void) | void` matches both the bridge and legacy `render` signatures (some return cleanup, all callers guard). Hook keys/signatures declared once in Task 3 and referenced by name after.
- **Placeholders:** foundation + bridge + panel carry real code; Tasks 5–N are a uniform playbook over the existing modules (the panel is the worked reference), each with concrete file/data/test steps — intentional for a large uniform migration, not a placeholder.
