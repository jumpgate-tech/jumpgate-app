# Gateway metrics and the Control Surface screen

**Date:** 2026-07-27
**Status:** approved, implementing

The RPC screen can say what a gateway is configured to do. It cannot say what the
gateway is actually doing. This closes that gap: eRPC counts its own requests, the
app reads those counters, and the screen becomes the Control Surface — network bands
wrapping their endpoints, capability tags where latency used to be, and traffic share
measured against intent.

Nothing is sent anywhere. The counters live in the gateway process, are read over
loopback, and are rendered locally.

## What was measured first

Every design decision below rests on readings taken from the two eRPC containers
running on this machine on 2026-07-27, not on eRPC's documentation (`erpc.dist.yaml`
is stale — `internal/catalog/gateway.go` already says so).

1. **Metrics are already on.** The valve-ws build defaults `metrics.enabled: true` on
   port 4001. We render no `metrics:` block, so both gateways have been serving
   Prometheus since they were created. "Turn metrics on" is therefore not the work;
   *stating* the decision in the file, and making it switchable, is.

2. **The port is published nowhere,** and `docker exec` cannot reach it: the image is
   `scratch` with a single `/erpc-server` binary — no shell, no `wget`, no `curl`.
   This is the constraint that decides how scraping works.

3. **`erpc_upstream_request_total` is unusable for traffic share.** It counts the
   state poller, which hits every upstream on `statePollerInterval` regardless of
   client traffic. Measured: `upstream="public-1-1"` carrying 354
   `eth_getBlockByNumber` calls that no client ever sent, against 5 real ones.

4. **`erpc_network_successful_request_total{project,network,upstream}` is the clean
   per-upstream client counter.** It read exactly 5 after 5 client requests, and the
   poller never touches it — the `erpc_network_*` family is populated on the
   client-facing path, which the poller bypasses.

5. **`internal/capabilities` is imported by nothing.** It is complete and tested but
   wired into no handler and no screen, so the mockup's capability column is a
   subsystem to connect, not a rendering change.

## 1. Metrics as an explicit block

`catalog.GatewayConfig` gains two fields:

```go
// MetricsOff turns the gateway's own request counters off. The zero value
// means ON, which is both eRPC's default and the decision the owner made, so
// no stored configuration needs migrating.
MetricsOff bool
// MetricsPort is the HOST port the counters are published on (0 → 4001).
MetricsPort int
```

The template renders the block **unconditionally**, in both states:

```yaml
metrics:
  enabled: true
  hostV4: "0.0.0.0"
  port: 4001
```

Rendering it always is the point. Today the file is silent and a default is inherited,
which means the file does not say what is happening and an off switch would have
nothing to turn off. `hostV4` follows the same rule `httpHostV4` already does:
`ops.GatewayContainerConfig` widens it to `0.0.0.0` for the container's private
namespace, and the systemd render binds `127.0.0.1`.

Validation rejects a metrics port equal to the gateway's RPC port, and
`checkPortFree` gains the metrics port so a collision fails preflight rather than
`docker run`.

**Operator-facing copy**, per the owner's wording: *"Your gateway counts its own
requests so this app can show which endpoints are carrying the load. The counters
stay on the machine the gateway runs on — nothing is sent anywhere."*

## 2. Scraping: publish on loopback, read through the executor

`ops.ERPCRunSpec` gains `MetricsPort`, emitting:

```
-p 127.0.0.1:<port>:4001
```

Two deliberate asymmetries with the RPC port:

- **Always loopback**, whatever the gateway's own `BindAddr` is. The RPC front door
  is meant to be exposed — that is what `ERPCBind` is for. The counters are not.
- **Published even when `NoPublish` is set.** A fronted gateway publishes nothing for
  RPC, on purpose, because a plaintext RPC port would be a second unauthenticated way
  in. A read-only counter endpoint on loopback is a much smaller door, and without it
  a fronted gateway — the recommended configuration — could never show a share bar.
  The comment in `runDocker` that says a fronted gateway publishes NOTHING gets
  amended rather than quietly falsified.

New package **`internal/metrics`**:

- `ParseText(io.Reader) ([]Sample, error)` — a minimal Prometheus text-exposition
  parser: metric name, label set, value. No dependency; we need three families.
- `Traffic` — per `(network, upstream)` successful counts, per-network received
  counts, and `Since` from `process_start_time_seconds`.
- `Read(ctx, executor.Executor, port) (Traffic, error)` — runs `curl` through the
  existing executor, exactly as `gatewayCheck` does. One code path, and an SSH target
  works with no second mechanism.
- `Shares(...)` — actual against intended, per network.

Counters are cumulative since the gateway process started. The API reports `Since` and
the UI says so; a rate would need a baseline the app loses on restart, and a number
labelled with the window it covers is more honest than one that pretends to be live.

## 3. Intended share

eRPC has exactly two tiers: preferred, and a `0.2`-scored fallback. It does not
weight-round-robin, so a multiplier is not a target percentage and normalising one
into a bar would be a fiction.

The rule:

- Intended share is `1/N` across the **preferred-tier** upstreams of that chain.
- Fallback upstreams intend **0** — anything they carry is failover.
- A chain with no preferred upstream spreads intent evenly across its fallbacks,
  because there is nothing to prefer.

Divergence past **20 percentage points** turns the number amber. The mockup's 88%/12%
ticks are illustrative; this rule is what gets drawn.

## 4. Capabilities

`GET /api/gateways/{gid}/capabilities`, cached per endpoint with a timestamp, the same
shape as `s.tlsChecks` — refreshed by an explicit action and after provisioning, never
on every poll, because probing opens real sockets.

Two limits, surfaced rather than hidden:

- An upstream addressed by **container name** (`ws://valve-node-app-devnet:8546`) is
  not dialable from the app's host. The probe uses the devnet's **published** address
  instead; the eRPC dial address and the probe address are different things and the
  code says so.
- Upstreams of a gateway on a **remote SSH box** are not reachable from here at all.
  That reports as inconclusive *with the reason stated* — an unknown whose cause is
  named is a different thing from a blank cell.

**`StatusInconsistent`** is added to `internal/capabilities`. External endpoints are
probed 3×; disagreement becomes that status rather than being majority-voted away.
Measured 2026-07-25: `msgboard_status` on `rpc.pulsechain.com` returned 5 supported /
10 × -32601 across 15 probes. A load balancer whose members disagree is the useful
signal, not noise to be smoothed. Managed endpoints are probed once — a single node
cannot disagree with itself.

## 5. The Control Surface screen

`rpc.ts` tiers 2 and 3 collapse into one table. Chips and per-chain selection go away:
every chain is visible at once, which is the whole argument of the design — the
routing hierarchy becomes structural rather than described.

- **Network band** — a row spanning every column: chain id, name, path, a state pill
  and `+ Endpoint`.
- **Endpoint rows** beneath it: Endpoint / Role / State / Capabilities / Share /
  actions.
- **Capabilities** always render the full set (HTTP, WS, ARCHIVE, TRACE) so an absence
  is a visible gap. Grey struck-through means not offered; red means the absence is
  breaking this chain — which is what makes a chain whose every endpoint lacks WS read
  as "subscriptions unavailable" at a glance.
- **Share** — an 88px track, actual as fill, intended as a tick, the number amber on
  divergence.

The gateway bar, the settings panel, the TLS panel, the provisioning stream and every
modal are unchanged. Latency, request rate and error history are deliberately not
added: they answer "how is it doing", which is diagnosis after detection, and they
belong on the analytics page this screen links to rather than hosts.

The file's existing rules still bind: no native `confirm()`/`alert()`, never offer an
action that can only fail, and state plus the reason for unavailability live on the
thing you interact with.

## Delivery

Four commits on master, each building and testing green:

1. `catalog` + `ops` — the metrics block, the loopback publish, validation, preflight.
2. `internal/metrics` + the traffic API.
3. `internal/capabilities` inconsistency + the capabilities API.
4. The Control Surface screen.

## Verification

Per this repo's own hardest-won lesson, tests are not the acceptance gate. The two
gateways running on this machine are: `valve-node-app-erpc` (fronted, publishes
nothing today) and `valve-node-app-erpc-edge` (published on 127.0.0.1:4200). Both get
re-provisioned, real traffic is sent through them, and the share bar is checked against
counts taken independently from the metrics endpoint. A bar that renders is not the
same as a bar that is right.

## Explicitly not in scope

- The analytics page (latency, rate, error history).
- Cross-machine `ws://` resolution for managed devnets — backlog item 2, unchanged.
- Per-client dataset sizes, which remain owner-supplied only.
