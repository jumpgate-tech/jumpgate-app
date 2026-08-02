# Frontend Migration to React 19 — Design Spec

**Date:** 2026-08-02
**Status:** Design approved.
**Owner:** dev@valve.city

## 1. Problem

`cmd/valve-node-app/web` is ~11k lines of vanilla TypeScript across ~15 screen
modules, all rendered the same way: `render(root: HTMLElement) => () => void`,
`root.innerHTML = string`, delegated `data-action` clicks, and hand-written
`escapeHtml` on every interpolated value. There is no view diffing, so anything
that updates on a timer or a stream has to hand-roll a dirty-check to avoid
tearing down and rebuilding the DOM on every tick.

The clearest symptom is the panel's `healthSignature`/`lastHealthSig`
apparatus: the panel polls analytics every 5s to animate health dots, but a
naive re-render (`card.innerHTML = …`) destroys every `.p-dot` node, which
restarts its CSS `@keyframes` from frame 0 — so a steady dot would visibly
stutter every 5s, and focus/scroll/in-flight input would be lost. ~40 lines
plus dense explanatory comments exist purely to reimplement, by hand, what a
reconciler does for free. The same class of problem recurs anywhere the app has
live data (SSE setup streams, capability probes, container polling).

The whole frontend should be **one paradigm, one stack**, matching the React
tooling already used elsewhere in the org, so live data renders declaratively
from state instead of via manual DOM bookkeeping.

## 2. Goals

- Migrate the entire `web` frontend to **React 19**, ending on a single
  paradigm (no permanent vanilla/React hybrid).
- Match the org's existing React stack rather than introducing new libraries.
- Keep `master` shippable throughout the migration — every screen reachable at
  every commit.
- Delete the manual-rendering scaffolding (`healthSignature`, `escapeHtml`
  sprinkling, `data-action` delegation) as screens convert.
- Land **tests with every screen ported**, raising frontend coverage from
  near-zero (only `panelModel` is tested today) as the migration proceeds.
- Preserve the `//go:embed`-into-the-binary build (no CDN/runtime fetch) and
  every existing `#/…` deep link.

## 3. Stack decisions (grounded in existing repos)

| Concern | Choice | Precedent / rationale |
|---|---|---|
| View | **React 19** | Requested; `trace/examples/widgets-demo` runs React 19. |
| Build | **Vite + `@vitejs/plugin-react`** | Keeps the current Vite pipeline and `//go:embed all:web/dist`; `trace` uses this plugin. |
| Routing | **`react-router` `HashRouter`** | Hash routing needs no server config for the embedded static SPA; expresses the existing `#/setup → #/machine` redirects as `<Navigate>` and `:id` params. |
| Server data / polling | **`@tanstack/react-query` v5** | `chainlist` uses v4 on React 18; v5 is the React-19-compatible line. Owns loading/error/refetch/dedup; `refetchInterval` replaces `setInterval` polling; the reconciler preserves DOM nodes, deleting `healthSignature`. |
| SSE streams | **custom `useEventStream` hook** | `streamSetup`/`EventSource` don't fit a query; a small hook wraps subscribe/cleanup. |
| Global client state | **local state / React Query cache first; `zustand` only if a real cross-screen need appears** | Almost all state here is per-screen; `chainlist` has `zustand` available if needed. `theme.ts` already owns theme state and is kept. |
| Styling | **existing CSS files, imported globally** | Tokens + `data-theme` light/dark stay exactly as-is; no CSS-in-JS. |
| Tests | **vitest + `@testing-library/react` + jsdom + V8 coverage** | Extends the existing vitest setup; RTL is the standard React component-test tool. |

Everything bundles into `dist/` via Vite and is embedded in the Go binary —
adding React/react-router/React Query costs bundle bytes, not a network
dependency.

**Reused unchanged:** `api.ts` (the typed client), `panelModel.ts` + its
vitest suite, `theme.ts`, and all of `style.css` / `panel.css` (tokens and the
new light theme).

## 4. Architecture

### 4.1 Shell + routing
- `main.tsx` mounts `<App/>` into `#app`, wrapped in a single
  `QueryClientProvider` and `HashRouter`.
- `App.tsx` renders the topbar/nav (ported from `ui.ts:renderShell`, with the
  active-nav logic) plus a `<Routes>` switch. The nav highlights the active
  screen the same way `setActiveNav` does today.
- `initTheme()` still runs before first render (in `main.tsx`), unchanged.
- Existing routes and their redirects are reproduced exactly:
  `#/` and unknown → panel; `#/rpc`, `#/machine/:id`, `#/security/:id`,
  `#/diag/:id`, `#/analytics/:id`, `#/targets`, `#/settings`, `#/panel`; and the
  `#/setup|dash|logs|services/:id → #/machine/:id` redirects via `<Navigate>`.

### 4.2 The `<LegacyScreen>` bridge (keeps master working mid-migration)
A not-yet-ported screen still ships as a route whose element is
`<LegacyScreen render={renderRpc}/>`. The bridge:
- creates a `div`, calls the legacy `render(div)` in a `useEffect`, and calls
  the returned cleanup on unmount (mirroring `main.ts:mount`),
- so every legacy screen keeps working — SSE streams, intervals, and their
  cleanup — while wrapped in the React shell.

As each screen is converted to a real component, its `<LegacyScreen>` route is
replaced by the component and the legacy module is deleted. When the last screen
converts, `<LegacyScreen>` and the old `render(root): cleanup` signature are
deleted — the app is then 100% React.

### 4.3 Data layer patterns (applied first in the panel)
- **Reads/polling:** query hooks over `api.ts`, e.g. `useGateways()`,
  `useGatewayHealth(gid)` (with `refetchInterval: 5000`),
  `useGatewayCapabilities(gid)`. Components render from `data`; no dirty-check.
- **Mutations + provision:** `useMutation` for `putGatewayConfig`, lifecycle
  actions, wipe, etc., invalidating the relevant queries on success.
- **SSE:** `useEventStream(targetId, onEvent)` wraps `streamSetup`, tearing the
  stream down on unmount or when the id changes.

## 5. Migration sequence

1. **Foundation PR:** add React/react-router/React Query/RTL deps and the Vite
   React plugin; add `main.tsx` + `<App/>` shell + `HashRouter` routes; every
   screen wired through `<LegacyScreen>`; nav + theme working. `master` renders
   identically to today, now under a React shell. Tests: shell/routing +
   bridge mount/cleanup.
2. **Panel PR (pattern-setter):** convert `panel.ts` → React components
   (list/network/endpoint + `<SettingsSheet>` gear), backed by the query/SSE
   hooks. Deletes `healthSignature`, the `data-action` delegation, and the
   `escapeHtml` calls in that screen. Tests: component tests for each view +
   any extracted pure logic; `panelModel.ts` tests stay.
3. **One screen per PR** thereafter (rpc, machine, wizard, services, analytics,
   targets, settings, dashboard, logs, diag, security, home, verdict …), each
   PR converting the screen, deleting its legacy module + bridge entry, and
   **landing tests**.
4. **Final PR:** delete `<LegacyScreen>`, the old `render()` signature, and any
   now-dead `ui.ts` string helpers. One paradigm.

Ordering within step 3 favors smaller/self-contained screens first to build up
shared component vocabulary before tackling `rpc.ts` (2,799 lines) and
`wizard.ts` (928 lines).

## 6. Global constraints

- **React 19, Vite, react-router HashRouter, @tanstack/react-query** — no other
  view/router/data libraries without revisiting this spec.
- **Tests land with every screen ported.** Extract logic to pure modules with
  vitest unit tests; cover components with `@testing-library/react`. V8 coverage
  is enabled and expected to climb each PR.
- **`master` stays working at every commit** — a screen is either fully React or
  running through `<LegacyScreen>`, never broken.
- **No CDN / runtime fetch of code.** Everything bundles into `dist/` for
  `//go:embed`.
- **Preserve every existing `#/…` route and redirect**, so deep links and the
  Go server's served paths keep working.
- **Reuse, don't rewrite:** `api.ts`, `panelModel.ts`, `theme.ts`, and the CSS
  token system are carried over as-is.

## 7. Out of scope

- Visual redesign — this is a rendering-layer migration; screens look the same
  (the panel's markup is re-expressed, not restyled).
- Backend/API changes.
- Adopting `zustand` unless a concrete cross-screen state need is found during a
  screen's conversion (decided per-screen, not up front).
- Rewriting `api.ts` into hooks wholesale — hooks wrap it; the client stays.

## 8. Definition of done

Every screen is a React component; `<LegacyScreen>` and the vanilla `render()`
signature are gone; `go build` still embeds a working SPA; all routes/redirects
behave as before; and the frontend has a real, growing test suite with coverage
reported.
