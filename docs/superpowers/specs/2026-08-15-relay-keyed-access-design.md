# Slice B — Keyed access (the relay) (design)

Date: 2026-08-15. Part of [self-hosted metered RPC](../plans/self-hosted-metered-rpc.md).
Status: design for review, not yet built.

## Goal

Put a thin Go relay in front of the keyless eRPC. A customer calls
`/rpc/<key>/<arch>/<chainId>`. The relay validates the key, applies the key's
method policy, strips the key, and forwards a plain JSON-RPC call to eRPC's
existing path `/<project>/<arch>/<chainId>`. eRPC never sees a key.

This slice is the heart of the metered product. Slice A gave the gateway a real
public certificate but left the endpoint open — anyone with the URL calls it for
free. Slice B closes that door with a per-key gate. Slice C (metering) and slice
D (credits) then build on the key store this slice creates.

## Why the key is a path segment

eRPC will not read a key from the path. eRPC reads auth from a header or a query,
and this deployment runs eRPC with no auth at all (see the roadmap's reference
model). So something must translate a keyed public URL into a keyless internal
one. That something is the relay. The key rides in the path because the path is
where the roadmap's URL shape puts it, and because a path segment survives every
proxy hop without special header handling.

v1 serves `arch = evm` only. The relay still parses and carries `:arch` through,
so the URL shape does not change when the polyglot arches (svm, btc) land. That
matches seam 2 of the eRPC client-view note
(`docs/superpowers/notes/2026-08-15-consuming-valve-erpc-client-view.md`), which
records `evm` as hardcoded in exactly two spots and names widening it as the
client-side match to the fork work.

## Scope

**In:** the public data-plane listener; path parsing for
`/rpc/<key>/<arch>/<chainId>`; key validation; per-key method allow/deny; the
key strip; the forward to eRPC; the embedded key store (SQLite); issue and
revoke endpoints on the private control plane; the Caddy change that points the
front at the relay for a metered gateway.

**Out (later slices):** per-request usage logging and usage views (slice C); the
credit ledger, the on-chain watcher, and disable-at-zero (slice D); the customer
dashboard and wallet-signature login (slice E). The store shape this slice
defines leaves room for all three, but this slice writes none of them.

**Out (fork work):** serving svm or btc. That is eRPC-fork work tracked in the
`erpc-polyglot-*` plans. v1 accepts `evm` only and rejects any other arch, while
still carrying the segment through.

## The two planes

The core constraint is a hard split between two planes that never share a
listener, a port, or an auth mechanism.

| Plane | Serves | Bind | Auth |
|-------|--------|------|------|
| Control (private) | jumpgate's own API and UI | `127.0.0.1:8799` (`cmd/valve-node-app/main.go:36`) | session token (`server.authMiddleware`, `server.go:206`) |
| Data (public) | customer RPC traffic | separate listener, behind Caddy | per-key, no session token |

The control plane exists today. `Server.Handler` builds one `http.ServeMux`,
wraps it in `authMiddleware`, and `ListenAndServe` serves it on the loopback bind
(`internal/server/server.go:180-271`). Every route on it demands the session
token. That token authorizes full control of the operator's servers, so the data
plane must never touch this mux. A customer holds a key, not the operator's
session token.

The data plane is new. It is a second `http.Server` with its own handler and its
own bind. It authenticates each request by key, not by session token. Keeping it
a separate listener — rather than a route group on the existing mux — is what
makes the auth boundary a fact of the wiring rather than a rule a future edit can
forget.

## Design

### 1. Where the relay runs (the central fork — see Open forks)

The relay is Go code jumpgate authors. Two placements are possible, and the
choice shapes several later points. This spec recommends the in-process listener
for v1 and defers the container.

**Recommended for v1 — a second listener in the jumpgate process.** jumpgate
already runs a Go server. Adding a second `http.Server` for the data plane is the
thinnest move, and it reuses two mechanisms that already exist:

- The relay reaches eRPC over the loopback RPC port. A fronted gateway already
  publishes eRPC on `127.0.0.1:<LoopbackRPCPort>` — the plaintext "wallet door"
  for the same machine (`internal/ops/docker.go:725-745`,
  `internal/setup/gateway.go:923`). The relay is one more same-machine consumer
  of that door. eRPC still publishes nothing to any network
  (`NoPublish`, `gateway.go:915`).
- Caddy reaches the relay over `host.docker.internal`. The eRPC container already
  runs with `--add-host host.docker.internal:host-gateway` on a plain Linux
  engine, and a VM-backed engine provides the alias natively
  (`docker.go:603-608`, `gateway.go:908`). Caddy can reverse-proxy to
  `host.docker.internal:<relayPort>` the same way.

This placement works when jumpgate runs on the same box as the gateway — the
self-host case. The control plane and the relay then share one in-process SQLite
handle, so issue and revoke need no cross-process coordination at all.

**Deferred — a container on the target network.** A gateway can also run on a
remote target that jumpgate drives over SSH. A relay in the jumpgate process
cannot reach a remote target's eRPC by container name, because that name resolves
only on the target's docker network (`docker.go:968-985`). The general answer is
a small relay container that jumpgate builds and places on the `valve-node-app`
network beside eRPC, exactly as it does for Caddy. That is more moving parts (a
new image, a new lifecycle) and is not needed for the first metered milestone. I
flag it rather than build it.

### 2. The public data-plane listener (`internal/server/`)

Add the data plane as a distinct handler and server, not a branch of the existing
mux. `registerAPIRoutes` (`internal/server/api.go:427`) and
`registerGatewayRoutes` (`internal/server/gateways.go:309`) mount the control
plane; the relay gets its own `registerRelayRoutes` on a fresh mux with no
`authMiddleware` wrapper. `ListenAndServe` (`server.go:251`) grows a sibling that
serves the relay mux on the relay bind.

Bind the relay to the interface Caddy reaches — loopback plus the docker-gateway
address — never straight to `0.0.0.0` on the public internet. Caddy is the public
door and the TLS terminator, mirroring the reasoning that makes eRPC set
`NoPublish` for a fronted gateway. A metered gateway must front through Caddy; a
plaintext keyed endpoint would leak the key on the wire (see point 8).

### 3. Path parsing

Parse `/rpc/<key>/<arch>/<chainId>` into three fields. Reject anything that does
not match: a missing key, a missing arch, a missing chainId, extra segments, a
non-numeric chainId, or an arch other than `evm` in v1. A malformed path returns
`404` (unknown route) or `400` (bad shape) and never reaches eRPC. Carry `arch`
as a variable from the first line of code, even though only `evm` passes today.

### 4. Key validation

Look the key up, verify its secret, and check that it is enabled. The relay reads
the key store (point 6), finds the record by its public id, compares the supplied
secret against the stored hash in constant time
(`crypto/subtle.ConstantTimeCompare`, the same primitive `server.tokensEqual`
uses at `server.go:246`), and confirms the `enabled` flag. An unknown id or a
wrong secret returns `401`. A disabled key returns `403`. Only a valid, enabled
key proceeds.

### 5. Method allow/deny

Apply the key's method policy before forwarding. Read the JSON-RPC body — a single
object or a batch array — and check each `method` against the key's allow list
and deny list. A denied method is refused. For a batch, this spec recommends
refusing the whole batch if any entry is denied (see Open forks), because a
partial batch complicates per-request metering in slice C. An empty allow list
means "allow all except the deny list", so a key with no policy still works.

### 6. The embedded key store (SQLite)

jumpgate has no database today. This slice adds one. The recommendation is
**SQLite through the pure-Go driver** (`modernc.org/sqlite`), not a hand-rolled
file store.

Rationale. Slice C needs per-request usage rows and "a couple of GROUP BY usage
views" (roadmap). Slice D needs an integer credit ledger with an atomic
decrement and a disable-at-zero check. Both are exactly what an embedded SQL
engine gives cheaply and safely under concurrent requests. A file-backed store
would reinvent locking, indexing, and aggregation, and would do each worse. The
cost is one new dependency. The pure-Go driver keeps the statically
cross-compiled, single-binary shape the project already ships — a cgo driver
(`mattn/go-sqlite3`) would break that. The store is a single file on disk, which
matches the "one file, no server to run" spirit of the rest of the app.

Key record shape, high level (enough for slices C and D, no schema bikeshed):

- **id** — the public key id. It is indexed and safe to log. It is the lookup
  handle.
- **hashed secret** — the secret half, stored only as a hash. The plaintext
  secret is shown once at issue and never again.
- **owner** — who holds the key. A free-text label in v1; a wallet address once
  slice E adds wallet login.
- **arch and method policy** — the allow list and deny list point 5 reads.
- **enabled** — the on/off flag. Revoke clears it; slice D's disable-at-zero
  clears it too.
- **timestamps** — created and updated.

Slice C adds a usage table keyed by the key id. Slice D adds a credits table
keyed by the owner or the key id. Neither needs a change to this record.

Key format. Issue a prefixed token, for example `jg_<id>_<secret>`. The relay
splits it into the id and the secret, looks up by the indexed id, then verifies
the secret hash. This lets the id appear in logs and usage rows while the secret
never leaves the customer.

### 7. The key strip — the load-bearing step

Build the forward request so the key appears nowhere in it. The forward path is
`/<project>/<arch>/<chainId>`, where `project` is the gateway's
`ProjectIDOrDefault` (`internal/catalog/gateway.go:277-283`). The relay must not
put the key in the forwarded path, in a query parameter, or in any header. This
strip is the whole point of the slice, and point 7 of Testing proves it directly.

The forward target must match eRPC's real path shape. Today `PathFor` builds
`/<project>/evm/<chainId>` and hardcodes `evm`
(`gateway.go:325-329`). Add `PathForArch(arch, chainId)` beside it, and let both
the keyless direct path and the relay share one definition. A second, drifting
copy of the path shape would send the relay to a path eRPC does not serve — the
same drift-avoidance reasoning the file already applies to
`GeneratedUpstreamID`.

### 8. The forward and JSON-RPC semantics

Forward with `net/http/httputil.ReverseProxy` (or an equivalent manual proxy) to
eRPC over the loopback RPC port. Preserve the HTTP method, the body, and the
headers that carry JSON-RPC meaning. Drop any header that carried the key.

**Batches.** A batch is a JSON array of calls. eRPC serves it on the same path.
The relay forwards the array intact after the method check in point 5. Note the
fork's multi-chain-batch race fix here (`docker.go:558-560`): a batch posted to
one path is answered per entry by eRPC's fixed build, so the relay does not need
to split a batch by chain.

**WebSocket upgrade for eth_subscribe.** The relay must pass the upgrade through.
`ReverseProxy` proxies a WebSocket upgrade when it forwards the `Upgrade` and
`Connection` headers to an http upstream. The fork's ws-upgrade-behind-gzip fix
matters directly here (`docker.go:550-557`): eRPC used to answer HTTP 500 on an
upgrade whenever the client sent `Accept-Encoding: gzip`, which every proxy adds.
The relay is now a **second** proxy in front of eRPC, so a customer's ws upgrade
crosses two hops (Caddy → relay → eRPC). The relay must not reintroduce the gzip
condition on an upgrade request, and it relies on the fork's fix for the eRPC
hop. State plainly that the ws path now survives two proxies, not one.

**A ws method-policy limit.** `eth_subscribe` and its kin travel in frames after
the upgrade, not in the opening HTTP request. So per-method allow/deny (point 5)
can only act on the initial handshake, which carries no method. v1 therefore
allows or denies WebSocket wholesale per key, and does not filter individual
subscription methods. Per-frame filtering would mean parsing the ws stream, which
is out of scope. This is an honest limit, flagged in Open forks.

### 9. The Caddy change (`internal/catalog/caddy.go`, `internal/setup/`)

Point the front at the relay for a metered gateway. Today the Caddyfile renders
`reverse_proxy <erpcContainer>:4000`
(`caddy.go:294-301`), and `resolveTLSFront` fills that upstream with the eRPC
container name and `ERPCContainerPort` (`internal/setup/tls.go:67-75`,
`internal/setup/gateway.go:567`). For a metered gateway, the front's upstream
becomes the relay instead — `host.docker.internal:<relayPort>` for the
in-process placement. Caddy still strips nothing: slice A already established that
Caddy does no path rewriting, so `/rpc/...` reaches the relay untouched, and the
relay is the one thing that reads and removes the key.

A non-metered gateway keeps pointing Caddy straight at eRPC, exactly as today.
The relay is added only where keys are wanted.

### 10. Issue and revoke (the control-plane surface)

Add key management to the private, token-gated API. These endpoints ride the
existing `registerGatewayRoutes` seam (`gateways.go:309-344`), where literal
segments already win over the `{action}` wildcard:

- `POST /api/gateways/{gid}/keys` — issue a key. Return the full secret **once**.
- `GET /api/gateways/{gid}/keys` — list keys. Never return a secret.
- `POST /api/gateways/{gid}/keys/{keyId}/revoke` — clear the enabled flag.

These write the store; the relay reads it. In the in-process placement, both the
control plane and the relay hold the same SQLite handle, so a revoke takes effect
on the next request with no message-passing. The container placement would need
the control plane to reach the store on the target — flagged in Open forks.

## Testing

The relay logic is pure and testable in process. Drive it against an
`httptest` stub upstream that records what it received. This is the reliable core
and it proves the security-critical strip.

- **Path parsing.** Table tests for `/rpc/<key>/<arch>/<chainId>`, including a
  missing key, a missing arch, a missing chainId, extra segments, a non-numeric
  chainId, an unknown arch, a trailing slash, and url-encoded segments. A
  malformed path never reaches the stub.
- **Key validation.** An unknown id returns `401`. A wrong secret returns `401`
  through the constant-time compare. A disabled key returns `403`. A valid,
  enabled key passes.
- **Method allow/deny.** An allowed method passes. A denied method is refused. A
  batch with one denied entry is refused whole (the recommended policy). A
  garbage body is rejected, not forwarded.
- **The key strip.** Assert on the request the stub upstream received: the path
  is `/<project>/<arch>/<chainId>`, the key is not in the path, not in any query
  parameter, and not in any header. This is the load-bearing test.
- **Batch pass-through.** A valid batch forwards intact to the stub, and the
  response array returns to the caller.
- **WebSocket pass-through.** A stub upstream that upgrades proves the relay
  proxies the upgrade and passes frames. This exercises the relay hop, not eRPC.
- **Forward-target correctness.** `PathForArch` and the relay build the same
  path, so the two cannot drift.

**verify-by-running-it — HONEST LIMIT.** The relay's own logic tests fully in
process, and that covers the strip, the key gate, and the method policy. What a
build box with no real upstreams **cannot** exercise:

- A real metered call that reaches actual chain data. That needs a live eRPC
  built from the fork, real upstream RPC endpoints, and network. The stub proves
  the relay forwards correctly; it does not prove eRPC then answers from a chain.
- The full two-hop WebSocket subscription. `eth_subscribe newHeads` through
  Caddy → relay → real eRPC → a real ws upstream needs the whole stack. A stub
  proves the relay's upgrade hop; it does not prove a subscription delivers heads
  across both proxies.

Where a real check is possible, run the relay against a locally built fork eRPC
container pointed at a public RPC endpoint, and confirm one `eth_blockNumber`
through `/rpc/<key>/evm/1`, plus one `eth_subscribe` if a ws upstream is at hand.
State plainly in the PR which end-to-end paths were and were not exercised on the
build box.

## Open forks — for the owner

Some choices here are genuinely the owner's. I recommend one option for each and
say why, but I do not decide them.

1. **Relay placement.** In-process second listener (recommended for v1,
   co-located self-host) vs a container on the target network (needed for remote
   targets). This choice drives points 1, 2, 8, and 10. Recommendation: ship the
   in-process listener now; add the container when a remote metered gateway is
   real.
2. **The eRPC forward target.** The existing loopback RPC port (recommended — it
   is already published for a fronted gateway) vs a dedicated relay-only eRPC
   port vs the docker network by container name (which the container placement
   would use).
3. **Batch policy.** Refuse the whole batch when any entry is denied
   (recommended — predictable, and it keeps slice C metering simple) vs strip the
   denied entries and forward the rest.
4. **WebSocket method policy.** Allow or deny ws wholesale per key (recommended
   for v1) vs per-subscription-method frame inspection (out of scope; needs ws
   stream parsing).
5. **The metered-gateway TLS rule.** Require Caddy/TLS for any metered gateway
   (recommended — a key in a plaintext URL path is readable on the wire) vs allow
   a plaintext keyed endpoint. Tied to this: the key sits in the URL path, so it
   lands in access logs and browser history. Mitigate by requiring TLS and by
   logging only the key id, never the secret. The owner should accept this
   property knowingly.
6. **The store driver.** Pure-Go SQLite (`modernc.org/sqlite`, recommended — it
   keeps the static single-binary build) vs cgo SQLite (`mattn/go-sqlite3`) vs a
   file-backed store (not recommended — slices C and D need SQL).
7. **The owner model for a key.** A free-text owner label in v1, widening to a
   wallet address when slice E lands. The owner confirms this is enough for now.

## Files touched (summary)

- `internal/server/` — a new data-plane listener and relay handler
  (`registerRelayRoutes`, a sibling server in `ListenAndServe`); the relay never
  joins the token-gated mux. Key issue, list, and revoke endpoints ride
  `registerGatewayRoutes` (`gateways.go:309-344`).
- `internal/catalog/gateway.go` — add `PathForArch(arch, chainId)` beside
  `PathFor` (`gateway.go:325-329`), shared by the keyless path and the relay.
- A new key-store package — SQLite through `modernc.org/sqlite`; the record shape
  in point 6; a handle shared with the relay in the in-process placement.
- `internal/catalog/caddy.go`, `internal/setup/tls.go`, `internal/setup/gateway.go`
  — point the front at the relay for a metered gateway
  (`caddy.go:294-301`, `tls.go:67-75`, `gateway.go:567`); Caddy still strips
  nothing.
- `cmd/valve-node-app/main.go` — wire the relay listener and its bind beside the
  control-plane bind (`main.go:36-70`).
- Web UI — a place to issue and revoke keys, and to show a key's secret once.
  Rebuild dist (`npx vite build`), CI-enforced.
</content>
</invoke>
