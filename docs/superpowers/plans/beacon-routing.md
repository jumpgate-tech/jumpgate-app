# Beacon routing — one base URL, per-architecture paths

Goal: when an operator starts a chain that has a consensus layer, expose BOTH
the execution RPC and the beacon (consensus) REST API under one hostname, so a
consumer reaches each with a path prefix.

Chosen pattern: Valve's, because its architecture slot already handles
non-EVM chains.

```
https://<host>/rpc/:key/:arch/:chain_id[/...]
```

`:arch` is the selector for "which internal server":

| arch     | protocol             | backend                    | example                                              |
|----------|----------------------|----------------------------|------------------------------------------------------|
| `evm`    | JSON-RPC (POST + WS) | eRPC (existing)            | `…/rpc/:key/evm/369`                                 |
| `beacon` | REST + SSE           | beacon client HTTP (:5052) | `…/rpc/:key/beacon/369/eth/v1/beacon/genesis`        |
| (future) | —                    | —                          | `svm`, `btc`, … slot into the same grammar           |

eRPC already serves `<project>/evm/<chainId>` (project `rpc`, default `main`).
So `evm` needs no change — beacon is a new branch beside it.

## Why beacon is a separate backend

eRPC is EVM-only; it understands `/evm/` and nothing else. The beacon API is a
different protocol — a REST tree under `/eth/...` plus an SSE stream at
`/eth/v1/events`. So `beacon` is NOT proxied by eRPC. It is a plain reverse
proxy to the beacon client's HTTP port.

Both live behind ONE front — the Caddy container that already terminates TLS
in front of eRPC (the "Serve HTTPS" option). The front routes by arch:

```
/rpc/:key/beacon/:chain_id/*   →  beacon upstream pool for :chain_id   (more specific, matched first)
/rpc/:key/*                    →  eRPC                                  (evm and everything else)
```

## Path handling — the rules that bite

1. **Strip the prefix for beacon.** The front rewrites
   `/rpc/:key/beacon/:chain_id/eth/v1/...` → `/eth/v1/...` before proxying, so
   the beacon client sees its own native tree. eRPC's `/evm` branch is passed
   through unchanged (eRPC owns that path).
2. **Stream, do not buffer.** Beacon `/eth/v1/events` is Server-Sent Events and
   exec `…/evm/:chain_id` upgrades to WebSocket on the same path. The front must
   disable response buffering on both, or subscriptions stall.
3. **Key check on both arches.** eRPC validates `:key` for evm. Beacon does not
   go through eRPC, so the front must validate the SAME key before it proxies
   beacon — otherwise beacon is an open door beside a locked one. This is the
   join point with the billing/key service (jg_ keys, HMAC lookup) — that
   service lives in the jumpgate-app tree (services/billing), not this repo.
   DECISION NEEDED (B1): key check in Caddy (forward-auth to the billing admin
   API) vs a small Go auth shim.
4. **No beacon → clean 404/501.** A chain with no consensus layer (an L2, or a
   PulseChain variant that exposes none) must answer `/beacon/:chain_id` with a
   definite 404/501, never a dead 502. The catalog already knows this:
   `catalog.Network.BeaconClients` is empty for such a chain, and a running
   target's `WireConfig.BeaconID` is empty when no beacon was set up.

## Which chains have a beacon

Catalog-driven, not guessed:
- `catalog.Network.BeaconClients` lists the consensus clients valid on a chain.
- A running node's `WireConfig.BeaconID` + `BeaconHTTPPort` (default 5052) name
  the actual beacon endpoint on the target.
- The front's beacon route for `:chain_id` exists only when at least one target
  running that chain has a beacon endpoint. Otherwise the route returns 404.

## Multiple nodes per chain → the beacon upstream pool

This is where beacon routing meets the multi-node / named-replica work (speced
separately — see the tunneling + replicas plan). The beacon route for a chain
is not one upstream, it is a POOL: the beacon endpoints of every target running
that chain.

- Load-balance across the pool (round-robin), fail over on error.
- Health per upstream from beacon's own `/eth/v1/node/health` (200 healthy, 206
  syncing, 503 down) — the front drops a 503 upstream from rotation.
- Each replica has a name/ID (the target ID). The pool is "all beacon endpoints
  for chain N", assembled from the named replicas.
- eRPC already does this for the evm side (multi-upstream failover). Beacon
  needs the equivalent in the front's proxy config.

## Build order (once the model is agreed)

1. Add the beacon reverse-proxy branch to the front (Caddy) with prefix strip +
   SSE passthrough, single upstream first.
2. Wire the "has a beacon?" gate from the catalog / WireConfig so the route is
   present only for chains that have one.
3. Resolve B1 (key validation for beacon) — reuse the jg_ key service.
4. Turn the single upstream into a per-chain pool once named replicas exist.

## Open questions

- **B1** — where the beacon key check runs (Caddy forward-auth vs Go shim).
- **B2** — does the front live in the Caddy container config we already
  generate, or does the app grow its own Go reverse proxy that fronts both eRPC
  and beacon? Caddy reuses existing machinery; a Go front gives tighter control
  over the pool + health logic.
- **B3** — beacon API version skew across clients (Lighthouse/Prysm/Teku expose
  slightly different `/eth/v2` vs `/eth/v1` coverage). The proxy is transparent,
  so this is a consumer concern, but the docs must name it.
