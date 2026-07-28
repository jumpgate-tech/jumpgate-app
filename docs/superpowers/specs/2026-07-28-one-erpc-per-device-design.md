# One managed eRPC per device, and the workaround it lets us delete

**Date:** 2026-07-28
**Status:** approved, implementing

Four changes that turned out to be one story. A gateway *names* the machine it runs
on, so gateway and device are one-to-one — but nothing enforced that, and this
machine accumulated two managed eRPC containers overlapping on the same chain. The
same investigation found that the reason fronted gateways ship with compression
disabled has just been fixed upstream in our own eRPC fork, and that the endpoint
picker makes redundancy a chore rather than the default.

## What was measured first

Readings from this machine on 2026-07-28, against the two running gateways and the
newly pinned eRPC build.

1. **Two managed eRPC containers, one device.** `valve-node-app-erpc` (gateway
   `default`, chain 1337, TLS on :8443) and `valve-node-app-erpc-edge` (gateway
   `edge`, chains 1337/369/1, plain HTTP on :4200). Both carry
   `placement.targetId: local`, `backend: docker`. They overlap on 1337, so two
   pollers hit the same devnet, and Caddy fronts only the first — the TLS door
   reaches the gateway with the *fewest* chains.

2. **The Caddyfile confirms it.** One site block,
   `reverse_proxy valve-node-app-erpc:4000`. Chains 369 and 1 are reachable only
   over plain HTTP on :4200.

3. **One upstream per network, everywhere.** Every network in both gateways has
   exactly one upstream. The RPC screen showing a single endpoint is accurate, not a
   display bug. `internal/chainlist` offers 18 probed candidates for chain 1 and 4
   for chain 369, several live and unused.

4. **The picker adds one at a time.** `openDiscoverModal` closes on selection and
   calls `addExternalUpstream` for a single URL. Building redundancy means reopening
   it per endpoint, and nothing suggests you should.

5. **The gzip workaround is now obsolete.** `MustDisableGzipBehindProxy` renders
   `enableGzip: false` on every fronted gateway, costing response compression, and
   its own comment prescribes the cure: *"skip the gzip wrapper on upgrade requests,
   or make the wrapper implement Hijacker."* That is exactly what shipped today in
   `valve-tech/erpc` `a7a53ec2`, verified against real containers — an upgrade
   carrying `Accept-Encoding: gzip` answers **500 on e909aacb and 101 on a7a53ec2**.

6. **Config and image only ever move together.** `PUT /api/gateways/{gid}/config`
   validates and stages; `erpc.yaml` is re-rendered and the image ensured at
   *provision* time, and `ensureImage` always builds `ops.ERPCImageTag()` from the
   pinned ref. So a gateway cannot end up with new config on an old binary.

7. **Deleting a gateway deliberately leaves its container alone**, and says so:
   *"its container was NOT touched — stop or wipe it before removing it."* The
   codebase's existing stance is never to orphan silently, but to name what was left
   behind.

## 1. The invariant: at most one gateway per target

A gateway names its host via `Placement`. Multiple gateways therefore mean multiple
machines; two gateways on one machine is a violation the app permitted rather than a
configuration anyone chose.

Enforced in two places:

- `POST /api/gateways` refuses a gateway whose `Placement.TargetID` already has one,
  with the existing gateway named in the error.
- Whole-config validation applies the same rule, so a hand-edited `config.json` is
  refused where it is loaded rather than producing a second container at provision.

The UI stops offering "add gateway" for a device that already has one.

## 2. Merging what already exists

Configs that already violate the invariant are merged on load rather than being
refused — refusing would lock an operator out of their own app over a state the app
itself allowed.

**Survivor**, in order: the gateway with TLS configured; then `DefaultGatewayID`;
then the oldest (first in the list). Prefer the secure door and the stable hostname.

**Merge:** networks unioned by `ChainID`; colliding upstream IDs re-suffixed. The
survivor keeps its own `Port`, `BindAddr`, `TLS`, `ProjectID` and metrics settings —
a merge changes *which chains* a gateway serves, never the door it serves them on.

Dedupe cannot key on endpoint alone: a managed upstream carries no endpoint of its
own (its address is derived at render time), so the two `managed-devnet` upstreams
being merged here are both `{kind: managed-devnet, targetId: local, endpoint: ""}`
and would survive an endpoint-only comparison as two identical rows. Key on
`(kind, targetID, endpoint)` — which collapses managed upstreams by what actually
identifies them, and external ones by URL.

**The retired container is reported, never removed.** Following the delete path's
precedent, the merge records the leftover container name and surfaces it so the
operator can wipe it. The app does not stop a container it did not just start, and a
migration that silently killed a running gateway would be worse than the duplication
it fixes.

For this machine: `edge` folds into `default`, chains 369 and 1 move behind
`https://default-07fcdc.localhost-valaxy.com:8443`, the duplicate 1337 upstream
collapses, and `valve-node-app-erpc-edge` is reported as leftover. **Anything pointed
at :4200 stops working** — the one user-visible regression, accepted deliberately.

## 3. Retiring the gzip workaround

Delete the `{{- if .Fronted}}` branch from `gatewayConfigTemplate` and delete
`MustDisableGzipBehindProxy`. Three tests currently assert the workaround is present
(`internal/catalog/gateway_test.go` twice, `internal/setup/tls_test.go` once); they
flip to asserting a fronted gateway no longer disables gzip.

This is safe only because of measurement 6: config and binary move together, and
provisioning always builds the pinned ref. It is gated on the gateways actually
running `a7a53ec2` — removing it while a gateway still runs `e909aacb` would restore
the 500.

The rationale does not simply vanish. `MustDisableGzipBehindProxy` recorded a real
measurement, and this document is where it goes: the constraint existed, it was
correctly diagnosed, and it was removed by fixing the cause rather than by deciding
it no longer mattered.

## 4. Multi-select in the endpoint picker

`openDiscoverModal` renders checkboxes over the live candidates and adds them in one
`saveConfig` rather than one call per endpoint. `addExternalUpstream` becomes
`addExternalUpstreams(gid, chainId, urls[])`.

It opens with **three endpoints pre-ticked: the fastest live candidates, except that
one of the three is the fastest live `wss://` candidate where the feed offers one.**
The WebSocket entry replaces the slowest of the three rather than becoming a fourth,
so the pre-ticked count is always three. The operator clicks once and gets
redundancy; unticking is easy and visible. Making the good outcome the lazy one is
the point — the current flow makes the good outcome the effortful one, which is why
every network here has exactly one upstream.

External upstreams keep `tier:fallback` and the `overall: 0.2` score multiplier, and
the `public-<chain>-<n>` ID scheme continues from the highest existing suffix.

Pre-ticking a `wss://` upstream means public chains gain `eth_subscribe`, which is
only true because of the eRPC fix in measurement 5. Chain 1 currently has no
WebSocket upstream at all, so subscriptions against it cannot work today.

## Delivery

1. Re-provision `default` onto `a7a53ec2` (operational; unblocks §3).
2. The invariant: validation, UI guard, merge-on-load, leftover-container reporting.
3. Retire the gzip workaround.
4. Multi-select picker.

§2 and §3 both change what is rendered, so they land as separate commits but are
provisioned once.

## Verification

- Go tests for the merge rule and the validation, table-driven, breaking one rule at
  a time — the shape this repo already uses, because a merge that quietly returns a
  tidy result is indistinguishable from one that worked.
- Catalog render tests assert a fronted gateway no longer emits `enableGzip`.
- `cd cmd/valve-node-app/web && npm run build` (tsc strict + vite), then
  `go build ./...` to confirm the rebuilt `dist/` still embeds.
- **Run it, don't infer it.** After provisioning, `eth_subscribe` over `wss://`
  through Caddy on the TLS door, which is the path the workaround existed to protect
  and which has never been verified end to end. A gateway that cannot upgrade still
  starts and still reports itself healthy.

## Explicitly not in scope

- Removing or stopping the leftover container automatically.
- Any change to how gateways are placed on *remote* targets; the SSH target and the
  systemd backend remain unexercised.
- Reporting the batch-routing race upstream to `erpc/erpc`. It is fixed in the fork
  and present on upstream `main`; the owner has deferred the upstream PR.
