# Plan: machine-page collapse + fleet verdict + port-check reclaim

**Date:** 2026-07-31
**Source direction:** [2026-07-28-ui-direction-notes.md](../specs/2026-07-28-ui-direction-notes.md)
**Status:** in progress — three parallel workstreams, disjoint file ownership

Three increments from the UI-direction doc, run concurrently. File ownership is
partitioned so the streams never edit the same file; the fleet-verdict stream
ships a self-contained module that main integration wires in.

## Stream 1 — Collapse the per-machine routes (the precondition)

Merge `#/setup`, `#/dash`, `#/logs`, `#/services/<id>` into one machine page
`#/machine/<id>`. Each former route becomes a **section**: a one-line status that
**expands** to detail. No tabs.

- New `machine.ts`: `renderMachine(root, id)`. Loads the target (`api.listTargets`/
  `api.getCatalog`) once; renders a machine header + the four sections as
  collapsed one-line rows, each expandable.
- Detail on expand **reuses the existing render fns as-is** (`renderDashboard`,
  `renderLogs`, `renderServices`, `renderWizard`) lazily mounted into the
  section's expanded container; the machine page's cleanup calls every mounted
  section's cleanup (they own EventSource/timers). Stripping the reused screens'
  own `<h1>` headers is a follow-up, not this increment.
- Best-effort one-line section status from data already cheap to get (target
  `wire`/catalog); deep per-section measured verdicts are a follow-up.
- `main.ts`: add `machine` route; **redirect** old `setup`/`dash`/`logs`/
  `services` hashes to `#/machine/<id>` (keep deep links alive).
- `targets.ts`: machine-card actions link to `#/machine/<id>` (Devnet/dashboard/
  logs/setup buttons collapse into one "Open" + the card's status stays).
- `ui.ts`: `setActiveNav("machine")` maps to the Targets nav item (no new nav).
- **Owns:** `machine.ts` (new), `main.ts`, `targets.ts`, `ui.ts`, `style.css`.
- Verify: `cd cmd/valve-node-app/web && npm run build`.

## Stream 2 — Fleet verdict, one sentence, above everything

The app opens with one line: what needs attention, or that nothing does — derived
from **measured** state, never from a listener's appearance (the doc's cautionary
tales: the phantom port-8600 conflict, amber-on-devnet). A verdict that can be
wrong is worse than the nine screens it replaces.

- New **self-contained** `verdict.ts`: `computeFleetVerdict(targets, catalog, …)`
  → a typed result (`ok | attention`, one sentence, the machines implicated), and
  `renderVerdictLine(root, verdict)`. Pure/near-pure; unit-testable.
- A devnet / single-by-design chain is **not** amber (`catalog.KnownSetSize === 0`
  ⇒ "single by design", healthy). No redundancy verdict where redundancy isn't an
  axis.
- **Owns:** `verdict.ts` (new) ONLY. Does not edit existing files. Main
  integration wires `renderVerdictLine` into the top of the home screen.
- Verify: `npm run build` (module compiles; exercised via a tiny local harness or
  by main integration).

## Stream 3 — Port pre-check stops refusing (with the 8443 caveat kept)

`internal/setup/gateway.go` `checkPortFree`/`probePort`. Owner's decided position:
adopt the process, reclaim the port, let docker/systemd fail loudly on a real
collision — a false "busy" is the worse outcome.

- eRPC HTTP port + metrics port: **stop returning an error** on a foreign
  listener; proceed (optionally log/annotate) and rely on the loud, specific
  runtime failure.
- **HTTPS front port (8443 / `TLS.HTTPS()`): keep conservative** — the casualty of
  reclaiming is someone's TLS front and the failure is silent from its owner's
  side. Leave the refusal (or downgrade to a loud, explicit warning) and comment
  it as the deliberate, still-unresolved exception per the direction doc.
- **Owns:** `internal/setup/gateway.go` + its `_test.go`. No TS.
- Verify: `go build ./... && go test ./internal/setup/...`.

## Integration (me, after all three return)

Wire `renderVerdictLine` into the home screen top; reconcile nothing else (file
ownership is disjoint). Full verify: `npm run build` → `go build ./... &&
go test ./...`. Commit per stream on master. Do **not** push.
