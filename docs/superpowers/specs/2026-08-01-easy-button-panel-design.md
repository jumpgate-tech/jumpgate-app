# Easy-Button Panel — UI Redesign Spec

**Date:** 2026-08-01
**Status:** Design approved (brainstorm v8). Ready for implementation plan.
**Owner:** dev@valve.city

## 1. Problem

The current web UI (`cmd/valve-node-app/web/src`) exposes 9 screens over 40+ endpoints:
`rpc.ts` alone is 2,799 lines. Operators are drowned in operator-grade detail. The
product is, underneath, a **load balancer (eRPC gateway) and a decision about what to
point it at** — nothing more. The UI should read like the macOS **Wi-Fi / VPN menu**:
glance, click, done.

## 2. North star

> A tray-style popover. One big power button (the load balancer). A list of networks you
> point it at, each a row you can glance at and drill into. It collapses away. Nobody
> "reads" a Wi-Fi menu — they glance and click.

**Shell decision:** build a **self-contained ~360px popover panel** now (served from the
existing Go binary at localhost), designed so a **native tray shell can host it later**
without a rewrite. Fixed width, no browser-chrome assumptions, single screen.

## 3. Concept mapping (UI term → backend reality)

| UI term | Backend (`internal/…`, `api.ts`) |
|---|---|
| **Power button** | Gateway lifecycle: `start`/`stop`/`create` (`gateways.go` `gatewayActions` state machine). UI renders the server-provided `actions` list verbatim. |
| **Network** (row) | A chain fronted by the gateway — `GatewayNetwork`/`NetworkView` (Ethereum 1, PulseChain 369, Testnet v4 943, Devnet 1337). |
| **Endpoint** | A `GatewayUpstream` (`managed-node` / `managed-devnet` / `external`) resolved to `UpstreamView`. |
| **Gateway** (URL) | The one dialable, balanced URL for a chain (internal-TLS loopback domain). "Balanced across all upstreams." |
| **Lock** | `TlsVerification` passed (`/api/gateways/{gid}/tls/verify`). |
| **Capability meter** | `EndpointCapabilities` probes: HTTP, WS, Archive, Trace. |
| **Speed / health dot** | `EndpointHealth` + analytics: rate of slow/failed requests. |

There is **no per-chain start/stop** in the backend today — a gateway fronts all its
chains together. "Turn a network on/off" = add/remove that chain from the gateway and
re-provision. The power button is the gateway master switch. (Per-chain enable is a
possible future backend capability, out of scope here.)

## 4. Visual language (locked)

- **One card only.** The panel is the single rounded container. Everything inside is a
  **full-bleed band** separated by **flat 1px dividers** (no nested/inset sub-cards). All
  content sits on one **16px gutter**; every row leads with an **18px slot** so names align
  regardless of the leading glyph (dot / icon / `+`).
- **Symbols over words.** Inline SVG icon set (portable — not SF Symbols):
  power, globe (HTTP), ⇄ (WS), layers (Archive), crosshair (Trace), lock (TLS), pulse,
  pencil (rename), trash (remove), copy, scale (balanced), refresh, chevrons, plus, check.
- **Capability meter** replaces prose: lit icon = probed-present; green = standout (e.g.
  archive). Labeled once in detail so the vocabulary is learnable.
- **Stillness = health. Motion = trouble.** A healthy dot is **solid and still** (soft
  static glow, no animation). A dot only moves when there's a problem, and **glitch
  frequency maps to the rate of slow requests**: rare slow calls → occasional twitch
  (green); frequent → constant twitch (amber). No millisecond numbers in the glance view.
- **Rename is hover, not a button.** Hover a title/endpoint name → pencil appears.
- **Destructive is a single red bar** at the bottom of a detail ("Remove network" /
  "Remove endpoint") — very "Forget This Network."
- Dark theme baseline; must remain theme-aware if adopted into the existing app shell.

## 5. Navigation hierarchy

```
Networks list  →  Network detail  →  Endpoint detail
```

### 5.1 Networks list (home)
- Header: brand dot + "Valve" + one-line summary ("3 networks · online").
- **Master power band**: big round power button, state label ("Running"), subtext
  ("3 networks served"), master toggle. Reflects `gatewayActions` (green/red/blocked).
- **Networks** section: one row per chain — `[18px lead: health dot] name [capability
  meter] [chevron]`. Off/disabled networks dim. `+ Add a network` row.

### 5.2 Network detail
- Header: `←` · health dot · **network name (hover-rename)** · master-for-this toggle.
- **Gateway** band: label "Gateway · balanced across all" + **lock** + **copy**; the
  dialable URL as plain mono content (no box).
- **Endpoints · N** band: one drill-in row per upstream `[dot] name · source [chevron]`;
  `+ Add endpoint`.
- **Capabilities** band: flat inline meter (union of upstreams).
- **Status** band: vertical rows (Health, Chain head) + small `↻` refresh (force re-probe;
  otherwise auto-polls).
- **Remove network** (red bar).

### 5.3 Endpoint detail
- Header: `←` · health dot · **endpoint name (hover-rename)** · in-rotation toggle.
- **Address** band: the endpoint URL, **editable** (tap to edit) + copy.
- **Capabilities** band: this endpoint's probed caps.
- **Status** band: Health, "Slow requests" (qualitative), Chain head / "behind N blocks".
- **Remove endpoint** (red bar).

**Endpoint naming:** auto-assigned from domain (`rpc.publicnode.com` → "publicnode");
rename is optional, offered during add and via hover thereafter.

## 6. Flows still to design during build

These follow the same language; to be mocked/validated by running the real app:

1. **Add a network** — pick from the fixed catalog (Eth / PulseChain / Testnet / Devnet)
   not already added; wire Valve's known-set upstreams; provision.
2. **Add endpoint** — enter/paste a URL (auto-name from domain, optional rename); probe
   capabilities; add as `external` upstream. Also "use my managed node."
3. **Power off / blocked / empty states** — stopped gateway (red), Docker missing /
   unreachable (blocked with server-supplied reason), and **first-run empty** (zero
   networks → the one-click "set up my endpoint," modeled on `home.ts:setupEndpoint`).
4. **Devnet** — opt-in, de-emphasized (last in the list).

## 7. Implementation approach

- New self-contained panel, vanilla TS + the existing typed `api.ts` client. Reuse the
  hash-router (`main.ts`) with a new route (e.g. `#/panel`) that can become the default.
- Reuse the server-driven `actions`/`blocked` model so the UI never offers an impossible
  action. Reuse SSE health/snapshot streams to drive the dots.
- Keep the old screens reachable during migration; make the panel the landing view.
- **Verify by running it** (repo rule): drive the real binary, not just types — the worst
  bugs here report success while broken.

## 8. Out of scope

Native tray shell; per-chain backend start/stop; multi-machine/fleet management screens
(analytics, diagnostics, firewall) — those remain in the existing deeper UI for now.
