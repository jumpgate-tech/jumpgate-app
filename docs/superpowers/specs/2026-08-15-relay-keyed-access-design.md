# Slice B — Keyed access (the relay) (design)

Date: 2026-08-15. Revised 2026-08-17 after owner review.
Part of [self-hosted metered RPC](../plans/self-hosted-metered-rpc.md).
Status: design for review, not yet built.

## Goal

Put a thin Go relay in front of the keyless eRPC. A customer calls
`/rpc/<key>/<arch>/<chainId>`. The relay validates the key, applies the key's
method policy, strips the key, and forwards a plain JSON-RPC call to eRPC. eRPC
never sees a key.

This slice is the heart of the metered product. Slice A gave the gateway a real
public certificate but left the endpoint open — anyone with the URL calls it for
free. Slice B closes that door with a per-key gate.

## What the review changed

The first draft got six things wrong. Each correction is load-bearing, so they
are listed here rather than buried.

| # | First draft | Now |
|---|-------------|-----|
| 1 | Build a new Go SQLite key store | Use the shipped Rust `services/billing` over a unix socket |
| 2 | `beacon` is an arch under `/rpc` | `rpc` and `beacon` are sibling categories; each carries an arch slot |
| 3 | No health surface | `/health` is a third category, and a rollup over the matrix |
| 4 | WebSocket passes through to eRPC | The relay terminates ws and speaks HTTP to upstreams |
| 5 | gzip is a hazard to avoid | gzip is a test axis on every acceptance case |
| 6 | End-to-end is untestable here | Acceptance runs on a real box (`the test VPS`) |

The key format does NOT change. It stays `jg_<base58 of 16 CSPRNG bytes>`, opaque
and unversioned. See "Key format" for why.

## 1. The URL grammar

One shape covers every surface:

```
/<category>/<key>/<arch>/<chainId>[/...]
```

**Category** names the kind of API. **Arch** names the chain family. They are
different dimensions, and both are open sets.

| Route | Backend |
|-------|---------|
| `/rpc/<key>/evm/<chainId>` | eRPC, JSON-RPC over POST and ws |
| `/rpc/<key>/svm/<chainId>` | reserved — the arch slot, unused in v1 |
| `/rpc/<key>/btc/<chainId>` | reserved — same |
| `/beacon/<key>/evm/<chainId>/eth/v1/...` | the beacon client HTTP port (5052) |
| `/health[/<key>[/<arch|category>[/<chainId>]]]` | the relay itself |

v1 accepts `arch = evm` only and rejects the rest with a definite `501`. The
relay still parses and carries `arch` from the first line of code, so widening it
later changes a value and not a shape.

`beacon` is populated only on `evm` today. It keeps its arch slot anyway. A
future chain family with a consensus-layer REST API then costs a value, not a new
route. This is the same trade the roadmap already took for `arch` itself.

### Two grammars, not one

The categories parse differently, and the spec must say so plainly.

- **`/rpc` is fixed depth.** Exactly `key/arch/chainId`. Any extra segment is a
  `400`.
- **`/beacon` is prefix plus tree.** `key/arch/chainId` then an arbitrary
  remainder, which the relay forwards. The beacon API is a REST tree, so extra
  segments are the normal case.
- **`/health` is variable depth.** Each level pins one more dimension. See
  section 3.

Reject a malformed path before any store lookup. A missing key, a missing arch, a
missing chainId, a non-numeric chainId, or an unknown category returns `400` or
`404` and never reaches an upstream.

## 2. Beacon is not eRPC

eRPC understands `/evm/` and nothing else. The beacon API is a different
protocol — a REST tree under `/eth/...` plus a Server-Sent Events stream at
`/eth/v1/events`. So the relay proxies beacon directly to the beacon client, not
through eRPC.

Four rules govern the beacon branch:

1. **Strip the whole prefix.** `/beacon/<key>/evm/369/eth/v1/beacon/genesis`
   becomes `/eth/v1/beacon/genesis`. The beacon client sees its own native tree.
2. **Do not buffer.** `/eth/v1/events` is SSE. The relay must flush as it reads,
   or a subscriber stalls.
3. **Pool the upstreams.** The beacon route for a chain is every target running
   that chain with a beacon endpoint. Round-robin, and drop an upstream that
   answers `503` on its own `/eth/v1/node/health`. A `206` means syncing, which
   stays in rotation but is reported as degraded.
4. **No beacon means a clean refusal.** A chain with no consensus layer answers
   `501`, never a dead `502`. The catalog already knows this: a network's
   `BeaconClients` is empty, and a running target's `WireConfig.BeaconID` is
   empty.

This closes decision **B1** in `docs/superpowers/plans/beacon-routing.md`, which
asked whether the key check for beacon belongs in Caddy forward-auth or a Go
shim. The relay is the shim. Caddy stays a plain reverse proxy.

## 3. Health is a rollup, not a route

Health filters the category × arch × chain matrix. Each path level pins one more
dimension, and the response nests whatever the caller left open.

| Path | Auth | Returns |
|------|------|---------|
| `/health` | none | bare `200` or `503`, no body detail |
| `/health/<key>` | key | every category, arch, and chain |
| `/health/<key>/evm` | key | all evm chains, each with `rpc` and `beacon` |
| `/health/<key>/beacon` | key | every chain with a beacon, any arch |
| `/health/<key>/evm/<chainId>` | key | one cell, both categories |

```json
// GET /health/<key>/evm/1
{ "arch": "evm", "chain_id": 1,
  "rpc":    { "ok": true,  "head": 21504331, "upstreams": 3 },
  "beacon": { "ok": false, "status": "syncing", "upstreams": 1 } }
```

Three rules:

- **The unkeyed level stays shallow.** A bare `200` or `503`, nothing more. A
  detailed unkeyed health page tells any scanner which chains the operator runs
  and when one lags.
- **Health is credit-exempt but rate-limited.** A monitor polling every ten
  seconds must not drain a customer's credits. An unmetered, unlimited endpoint is
  a free amplification surface. `project_key.credit_exempt` and
  `per_second_limit` already carry both levers.
- **Category names and arch names share one namespace.** Level three of `/health`
  accepts either. So never add an arch named `beacon`, and never add a category
  named `evm`. This constraint is cheap to hold and silent when broken.

## 4. Three processes, two planes

| Plane | Serves | Bind | Auth |
|-------|--------|------|------|
| Control (private) | jumpgate's API and UI | `127.0.0.1:8799` | session token (`server.go:206`) |
| Data (public) | customer RPC traffic | its own listener, behind Caddy | per-key |
| Billing (private) | the key and credit store | a unix socket | least-privilege credential |

The control plane exists today. `Server.Handler` builds one mux, wraps it in
`authMiddleware`, and serves it on loopback (`server.go:180-199`). That token
authorizes full control of the operator's servers, so the data plane must never
touch this mux.

The data plane is new. It is a second `http.Server` with its own handler and its
own bind. Keeping it a separate listener — rather than a route group on the
existing mux — makes the auth boundary a fact of the wiring instead of a rule a
future edit can forget.

Bind the relay to the interface Caddy reaches, never to `0.0.0.0`. Caddy is the
public door and the TLS terminator. A metered gateway MUST front through Caddy,
because a plaintext keyed URL exposes the key on the wire.

## 5. The relay's request path

1. Parse the path (section 1). Reject a bad shape before anything else.
2. Resolve the key (section 6). Reject an unknown key with `401` and a disabled
   key with `403`.
3. Apply the key's constraints — `method_allow`, `method_block`, `origin`,
   `network`, `ip_allow`, `ip_deny`, and `allow_trace`.
4. Reserve credits (section 7).
5. **Strip the key.** It must not appear in the forwarded path, in a query
   parameter, or in any header.
6. Forward. `/rpc` goes to eRPC at `/<project>/<arch>/<chainId>`; `/beacon` goes
   to the beacon pool at the stripped remainder.

The strip is the whole point of the slice, and one test asserts it directly.

`internal/catalog/gateway.go` hardcodes `evm` in `PathFor` (`gateway.go:325-329`).
Add `PathForArch(arch, chainId)` beside it and let the keyless path and the relay
share one definition. A second, drifting copy would send the relay to a path eRPC
does not serve.

**Batches.** A batch is a JSON array. The relay checks every entry's method and
refuses the whole batch if any entry is denied. A partial batch would complicate
per-request metering in slice C for no real gain. The fork's multi-chain-batch fix
(`docker.go:558-560`) means eRPC answers a batch per entry, so the relay does not
split by chain.

## 6. WebSocket: terminate and translate

The relay terminates the customer's WebSocket and speaks plain HTTP to every
upstream. It does not proxy the upgrade.

This is a deliberate exception to the roadmap's "delegate, do not rebuild" rule.
The roadmap assigns WebSocket multiplexing to eRPC. Terminating at the relay takes
subscription management back. The exception is written down here because it is
worth the cost:

| Win | Why |
|-----|-----|
| Upstream ws support stops mattering | the pool widens to every HTTP-only upstream and managed node |
| The gzip-on-upgrade bug class disappears | there is no upgrade on the relay-to-eRPC hop at all |
| N subscribers collapse to one poll loop | not N upstream connections — real money, and it stops tripping upstream rate limits |
| Failover works mid-subscription | a native ws subscription dies with its connection; a polled one re-targets on the next tick |
| Metering is clean | count delivered notifications, with no ws stream to re-parse |

Plain calls that arrive as ws frames (`eth_call`, `eth_getBalance`) map to an HTTP
POST with no caveat. Everything below concerns `eth_subscribe` only.

**Frame parsing is in scope.** The relay reads each client frame, parses the
JSON-RPC method, and applies the key's method policy. This is what lets
`allow_trace` and `per_second_limit` apply on ws at all. The implementation must
handle continuation frames (one call can span several), control frames (ping,
pong, close), and backpressure when a subscriber reads slower than heads arrive.

**Supported subscriptions in v1:**

| Subscription | How | Note |
|--------------|-----|------|
| `newHeads` | poll the head, push on change | latency floor is the poll interval |
| `logs` | `eth_getLogs` per new block range | filter applied relay-side |
| `syncing` | poll `eth_syncing` | cheap |
| `newPendingTransactions` | **not supported** | see below |

`newPendingTransactions` has no honest HTTP polling equivalent. `txpool_content`
is non-standard and heavy. The relay answers the subscribe request with a clear
error rather than a silent stream that never fires.

**Two costs to hold:**

- **Latency floor.** A native push delivers a head at propagation speed. A one
  second poll adds 500ms on average. That is noise on a twelve second chain and
  noticeable on a fast L2. Make the interval per-chain configurable.
- **Reorgs are now the relay's problem.** A poller must not deliver an orphaned
  head, and must not deliver the same head twice. A native subscription inherits
  this from the node; the relay must implement it.

The relay becomes stateful. It holds a subscription registry, memory scales with
concurrent subscribers, and it must clean up on disconnect.

## 7. The store: the Rust billing service over a unix socket

`services/billing` already ships the store. It has the key manager
(`keys.rs`), the SQLite store (`store.rs`, `schema.sql`), method constraints,
pricing, the credit ledger, and an append-only audit log. It is CI-gated and it
has no Go caller today. The relay calls it rather than rebuilding it.

### The transport is a unix socket, not TCP loopback

`admin::serve` binds a `TcpListener` and refuses a non-loopback address
(`admin.rs:113`). That is server-side posture. Nothing tells the client that
whatever answers on `127.0.0.1:8787` is really billing.

The attack is plain. Binding an unused high port on loopback needs no privilege
on Linux or macOS. If billing has not started yet, or has crashed, any local
process binds the port first, the relay connects, and the relay hands over its
credential. Two secrets are at risk, not one: the credential, and every raw
customer key the relay forwards on the hot path.

| Control | Kills | Cost |
|---------|-------|------|
| Unix socket in a `0700` service-owned directory | the whole class — no port to squat, other uids cannot connect | `serve()` takes a `UnixListener`; Go dials `unix` via `Transport.DialContext` |
| Peer credential check (`UnixStream::peer_cred()`) | a same-uid impostor | ~10 lines, std and tokio, no new dependency |
| Least-privilege split | a leaked relay credential cannot mint keys | a second route group and a second credential |
| Fail closed when billing is unreachable | reject all traffic, never allow-all | one branch |

**The least-privilege split matters most.** The relay needs exactly one
operation: authenticate a raw key and read its record. It never needs create,
rotate, revoke, pricing, or audit. Today one bearer grants all of them. Put
authenticate on `/internal/authenticate` behind a relay credential, and leave
`/admin/*` on the operator's. A full leak of the relay credential then buys an
attacker the ability to test whether a key is valid — bad, bounded, not fatal.

Do NOT instead give the relay the pepper so it can hash keys locally. A pepper
leak means an offline brute force of every hash in the store. That is strictly
worse than what it prevents.

**Windows.** `AF_UNIX` exists on Windows but is patchy in both toolchains. If the
desktop app must run the relay on Windows, that build falls back to TCP with HMAC
request signing, so the credential never crosses the wire.

## 8. Caching, leases, and consistency

Separate processes cost the instant revocation the first draft promised. That
promise came from a shared in-process SQLite handle, which this design does not
have.

**Key records: cache with a short TTL.** Five seconds. Revocation lag is bounded
by the TTL. Do NOT build push invalidation. It is point-to-point, it breaks at
more than one relay, and it degrades into a message bus that replaces it. The TTL
cache is a subset of the end state and survives unchanged.

**Credits: lease, do not cache.** A stale balance is money, not a wart. At high
QPS a customer at zero keeps spending for TTL × QPS requests. So the relay
reserves a block of credits, decrements locally at wire speed, and settles back
periodically. `account.credits_reserved` and `account.escrow_ceiling` are already
in the schema — the store anticipated this.

Slice D implements the settle loop. Slice B implements the reservation call and
the local decrement, so the shape is right from the start.

## 9. Key format — unchanged

Keys stay `jg_<base58 of 16 CSPRNG bytes>`: opaque, unversioned, 128 bits of
entropy, looked up by `HMAC-SHA256(pepper, raw_key)` against a UNIQUE column.

Signed self-describing keys were considered and rejected. No wallet handles a
signed request; every wallet takes a static key in a path. A signed key would also
grow the URL segment from ~25 characters to ~130, and that segment lands in logs
and browser history. Versioning is unnecessary for the same reason: the store
resolves a key once, and every later check is a memory hit on the cached record.

**The path placement is safe enough, and the constraints are why.** TLS covers the
path in transit — an observer reads the host from SNI, not the path. The real
exposure is logging at rest, which section 10 handles. The security boundary is
`key_constraint`: `origin` bounds a browser-exposed key, `ip_allow` bounds a
server-side one, `per_second_limit` and `escrow_ceiling` bound the damage of a
leak, and `rotate` already exists. This matches what Alchemy, Infura, and
QuickNode all do.

## 10. Logging and redaction

`caddyfileTemplate` (`caddy.go:294-301`) has no `log` directive, so nothing leaks
today. That is clean by accident. The first person who adds `log` to debug
metering writes every customer key to disk.

So the redaction ships in the template, not in a runbook:

```
log {
  format filter {
    wrap console
    fields { request>uri regexp "jg_[1-9A-HJ-NP-Za-km-z]+" "jg_REDACTED" }
  }
}
```

The character class is the base58 alphabet, which excludes `0`, `O`, `I`, and `l`.

Two properties already help. The strip means eRPC's own logs are clean by
construction. And the relay logs the key id, never the raw key — the id is what
usage rows and audit entries carry.

## 11. The Caddy change

Point the front at the relay for a metered gateway. Today the Caddyfile renders
`reverse_proxy <erpcContainer>:4000`, and `resolveTLSFront` fills that upstream
(`tls.go:67-75`, `gateway.go:567`). For a metered gateway the upstream becomes the
relay instead.

Caddy gains one matcher per category, all pointing at the same relay upstream.
Caddy still rewrites nothing — the relay owns every strip. A non-metered gateway
keeps pointing straight at eRPC, exactly as today.

## 12. Issue and revoke in the UI

The Rust admin API already serves `/admin/keys` for create, list, patch, rotate,
revoke, and `/admin/pricing`. jumpgate's control plane proxies these onto its own
token-gated routes, riding the `registerGatewayRoutes` seam
(`gateways.go:309-344`) where literal segments already beat the `{action}`
wildcard. The web UI shows the raw key exactly once, at issue.

## 13. Testing

**In-process.** The relay logic is pure and testable against an `httptest` stub
that records what it received.

- Path parsing, per category. Table tests for a missing key, a missing arch, a
  missing chainId, a non-numeric chainId, an unknown category, an unknown arch, a
  trailing slash, url-encoded segments, and extra segments (a `400` on `/rpc`, a
  forwarded remainder on `/beacon`).
- Key validation. Unknown returns `401`. Wrong secret returns `401`. Disabled
  returns `403`. The comparison runs in constant time.
- Constraints. `method_allow`, `method_block`, `origin`, `ip_allow`, `ip_deny`,
  and `allow_trace` each pass and each refuse. A batch with one denied entry is
  refused whole. A garbage body is rejected, not forwarded.
- **The strip.** Assert on what the stub received: the key is not in the path, not
  in any query parameter, and not in any header. This is the load-bearing test.
- Beacon rewrite. The stub receives `/eth/v1/beacon/genesis` and nothing more.
- ws frame policy. A denied method in a frame is refused. A continuation-framed
  call reassembles. A close frame cleans up the subscription registry.
- Subscription synthesis. A stubbed head sequence produces one notification per
  head, no duplicates, and no orphaned head after a simulated reorg.
- Store transport. A squatting listener on the socket path is refused by the peer
  credential check. An unreachable billing service fails closed.
- `PathForArch` and the relay build the same path, so the two cannot drift.

**gzip is an axis, not a footnote.** Every acceptance case above runs twice, with
and without `Accept-Encoding: gzip`, on POST and on the ws handshake. That is the
exact shape of the original eRPC bug, and the relay adds a second proxy that can
re-add the header.

**End-to-end, on a real box.** The first draft called this untestable. It is not —
`the test VPS` (512MB, 1 vCPU) and `the test VPS` both exist. Reach them on
port 2222; outbound port 22 is blocked from this network.

Acceptance on a real box:

1. `eth_blockNumber` through `/rpc/<key>/evm/1`, over a real Let's Encrypt cert.
2. `eth_subscribe newHeads` over ws, terminated at the relay, HTTP to eRPC.
3. An SSE stream from `/beacon/<key>/evm/369/eth/v1/events`.
4. `/health` unkeyed, and `/health/<key>/evm/1` keyed.
5. A revoked key stops working within the cache TTL.
6. Every one of the above, with and without gzip.

The footprint fits. That box already runs WireGuard plus a native eRPC with ~345MB
free. The Go relay and a static musl Rust binary add roughly 30–50MB combined —
eRPC is the memory consumer, not the glue.

## Open forks

Everything else is settled. Two remain:

1. **Poll interval per chain.** A default plus a per-chain override, or one global
   value. Recommendation: default one second, override per chain in the catalog.
2. **Relay placement for a remote target.** The in-process listener works when
   jumpgate shares a box with the gateway, which is the base case. A gateway on a
   remote target needs a relay container on that target's docker network. Flagged,
   not built.

## Files touched

- `internal/server/` — a new data-plane listener and relay handler; the relay
  never joins the token-gated mux.
- `internal/relay/` (new) — path parsing, the strip, the ws terminator and
  subscription registry, the billing client over a unix socket, the TTL cache and
  the credit lease.
- `internal/catalog/gateway.go` — `PathForArch(arch, chainId)` beside `PathFor`
  (`gateway.go:325-329`).
- `internal/catalog/caddy.go` — a matcher per category, and the log redaction
  filter (`caddy.go:294-301`).
- `internal/setup/tls.go`, `internal/setup/gateway.go` — point the front at the
  relay for a metered gateway (`tls.go:67-75`, `gateway.go:567`).
- `services/billing/src/admin.rs` — `serve()` takes a `UnixListener`; add
  `/internal/authenticate` behind a relay credential; add the peer credential
  check.
- `cmd/valve-node-app/main.go` — wire the relay listener beside the control-plane
  bind (`main.go:36-70`).
- Web UI — issue, list, revoke, and show a key once. Rebuild `dist`
  (`npx vite build`), CI-enforced.
