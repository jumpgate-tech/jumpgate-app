# Plan: eRPC-first landing + snapshot manifest discovery

**Date:** 2026-07-31
**Status:** in progress — two independent workstreams (frontend reframe, Go snapshot fix)

Two approved changes. Disjoint file ownership: the reframe is `cmd/valve-node-app/web/src/*`;
the snapshot fix is `internal/catalog/*` + `internal/setup/*`.

## Stream A — snapshot manifest discovery (fix `snapshot.go`)

**The bug:** `SnapshotManifestURL` builds `…/evm/<id>/reth/manifest.json`, but the gateway
requires an exact `…/evm/<id>/reth/<version>/<timestamp>/manifest.json` and exposes **no**
`latest` resolver or listing. So every `reth download` 404s.

**The real contract (verified live 2026-07-31):**
- Discovery: `GET https://one.valve.city/snapshot/evm/<chainID>/reth/versions.json` (KEYLESS).
  Returns `{ chain_id, available_versions: [ { reth_version_range: "2.3.x",
  manifest_url: "https://one.valve.city/snapshot/evm/<id>/reth/<ver>/<ts>/manifest.json",
  block, timestamp, generated_at, size_bytes }, … ] }`. Mainnet (evm/1) 404s — no snapshot.
  NB: the data is inconsistent — the first 369 entry used `version_range`, others
  `reth_version_range`. **Accept both keys.**
- The manifest's `base_url` **echoes the request key** and `reth download` pulls chunks from
  it — so the resolved `manifest_url` (keyless in versions.json) must have the operator's
  `vk_` key **injected** (`/snapshot/<key>/evm/…`) before it reaches reth.
- Snapshots are reth-version-specific (`storage_version` differs) — must match the target's
  ACTUAL reth version (the app builds reth from `main`, so the version isn't a constant).

**Implementation (all network on the TARGET, parsing/selection in Go — testable):**
1. `catalog.SnapshotVersionsURL(chainID) string` → the keyless versions.json URL.
2. In `snapshotStep` (runs after install-exec, so reth exists): `reth --version` on the target
   → parse major.minor (`"Reth Version: 2.2.0-pulse"` → `2.2`); `curl -fsSL <versionsURL>` on
   the target → versions.json bytes.
3. `catalog.SelectSnapshotManifest(versionsJSON []byte, rethVersion string) (string, error)`:
   pure — pick entries whose range (`2.2.x`) matches the reth major.minor (`2.2`), newest by
   `generated_at`; error clearly if none (covers mainnet / version with no cut).
4. `catalog.InjectSnapshotKey(manifestURL, key string) (string, error)`: rewrite
   `/snapshot/evm/…` → `/snapshot/<key>/evm/…`; validate shape.
5. `RethDownloadCommand(w, manifestURL)` now takes the resolved keyed URL.
6. Keep the api.go validation (key required + `vk_` pattern). Errors must be loud and specific
   (this repo reports success while broken).
- Verify: `go build ./... && go test ./internal/catalog/... ./internal/setup/...`; and prove the
  resolver against the LIVE endpoint (curl versions.json for 369/943, build the URL, confirm 200).

**Follow-up (noted, not this stream):** the wizard offers snapshot for mainnet
(`SnapshotSizeTB` set on chain 1) but no snapshot is published — the step will now fail clearly,
and Stream B / a later gate should stop offering it when `versions.json` 404s.

## Stream B — eRPC-first, capability-detected landing

**The problem:** the app centers the node-operator flow (Machines → "which client?") even for
GUI users who can never run a node; eRPC (their only usable path) is a side tab.

**Capability signal:** a machine can run a node if it's an SSH target (Linux by construction) or
a local target on a Linux host (`hostOS === "linux"`, already computed in `targets.ts`). Define
"node-capable fleet" = at least one such target exists OR the host itself is Linux.

**Landing:**
- **When the fleet is NOT node-capable** (e.g. macOS/Windows controller, no SSH targets): the app
  opens on the **gateway / "your RPC endpoint"** as the hero — a 1-click "create endpoint" using
  `vk_demo` + the chain's known-set upstreams, no node required. Node-running appears only as a
  secondary "Run your own node — needs a Linux server" card that leads to add-SSH.
- **When it IS node-capable:** keep machines prominent, but eRPC stays first-class.
- The client / snapshot / checkpoint questions live ONLY on the node path — never shown to a user
  who can't act on them.
- Nav: RPC/Endpoint becomes the first, default nav item; Machines second.

**Reuse, don't rewrite:** the eRPC screen (`rpc.ts`) and the machine page (`machine.ts`) already
exist. This stream is about the LANDING decision + demoting the node questions, not rebuilding
those screens. Keep the fleet verdict (`verdict.ts`) on whichever screen is the home.

- Owns: `main.ts` (landing decision), `targets.ts`/a new `home.ts`, `ui.ts` (nav order), `rpc.ts`
  only if the 1-click create needs a hook, `style.css`. Plus `api.ts` if a capability field helps.
- Verify: `npm run build`, then a runtime click-through (start the server, load as a
  non-node-capable controller, confirm eRPC-first landing).

## Integration (me)

Both streams are disjoint. Verify each independently, then full `npm run build` +
`go build ./... && go test ./...`, and a runtime smoke test of the landing. Commit per stream on
master; no push.
