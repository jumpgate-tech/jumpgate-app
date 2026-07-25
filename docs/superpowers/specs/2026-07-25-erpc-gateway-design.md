# eRPC gateway — design

Date: 2026-07-25
Status: in progress; catalog rendering and discovery landed, setup path in flight

## The principle

Every capability the app offers should be available on every machine with the
capacity to support it. Not "nodes are Linux-only, so a Mac is useless" — a
Mac may well be able to run a gateway, and should be offered one.

That makes the model a matrix, not a hierarchy:

|             | via systemd            | via Docker                                    |
| ----------- | ---------------------- | --------------------------------------------- |
| **Gateway** | Linux + root           | any OS with Docker                            |
| **Node**    | Linux + root + disk    | any OS with Docker + disk (see gaps)          |

Each of the four cells is reachable locally or over SSH. The UI's job is to
say, per machine, which cells it can fill and why not for the rest.

### Why the previous shape was wrong

The screen asked "which machine?" before "what do you want?". Asked that way
round, a Mac can only be answered with *"you can't"*. Asked goal-first, the
same machine answers *"here's how"*. The confusing states all came from that
ordering, not from the wording.

## The gateway is its own entity

eRPC addresses a chain by URL path — `/<project>/evm/<chainId>` — so **one
instance, one port and one config serve every chain**. WebSocket rides the
*same* path and port with a `ws://` scheme; there is no second listener.

This is why a gateway is not a mode of `WireConfig`. A `WireConfig` describes
one chain's node; a gateway deliberately spans chains, and its config path
therefore cannot live in the per-chain data dir (`/var/lib/valve-node-app/369`).

```go
type GatewayConfig struct {
    ProjectID string          // "" -> "main"; appears in every request path
    BindAddr  string          // "" -> 127.0.0.1; the front door, meant to be exposed
    Port      int             // 0 -> 4000
    Networks  []GatewayNetwork
}
type GatewayNetwork struct { ChainID int; Upstreams []GatewayUpstream }
type GatewayUpstream struct {
    ID, Endpoint string       // http(s):// or ws(s)://
    Local        bool         // an upstream we run: preferred
    RecentOnly   bool         // a pruned node: bounded to recent history
}
```

**A gateway with no local upstreams needs no special representation.** It is
just a config whose networks carry only fallbacks. The "this box can't hold a
node but I still want my own RPC" case therefore falls out of the same code
path instead of being a second mode — which is what killed the earlier
`NodeMode` enum idea.

## Verified facts

Everything below was checked against a real eRPC container, not inferred.

- **WebSocket works** on `valve-tech/erpc` branch `valve-ws`. `eth_subscribe
  newHeads` delivered a head in 4.4s; eRPC remaps subscription ids rather than
  passing them through. Upstream eRPC has no WebSocket support at all, so the
  fork is genuinely required.
- **Schema**: upstreams are a flat list at project level, each tagged with its
  `chainId` — not nested under their network. A `networks:` block is required;
  upstreams alone do not declare a network.
- **`failsafe` is a list, not a map.** The repo's own `erpc.dist.yaml` is stale
  and shows the old map form. A back-compat shim rewrites the map silently, so
  generating the deprecated shape would never be noticed.
- **Archive upstreams omit `blockAvailability` entirely** rather than emitting
  `lower: null`. eRPC treats absent bounds as "feature off", so omission is
  unambiguous where an explicit null is a present-but-empty bound.
- **Endpoints are quoted** in the rendered YAML — an unquoted URL is only
  accidentally safe.
- **Bug in `valve-ws`**: a multi-chain *batch* posted to `/<project>` answers
  the whole batch from the last entry's network. The per-chain path is
  unaffected, so it does not block us, but it silently returns wrong data and
  is worth reporting upstream.

### Fork risk

`valve-ws` is **ahead 15 / behind 73** of upstream and is based on erpc#836,
which is still open and actively moving. It publishes no releases and no
container image — its CI builds and tests but never pushes. Pin a SHA rather
than the branch head, and expect to own the rebase.

## Upstream discovery

`chainid.network/chains.json` (the feed chainlist.org renders) is the source.
Two filters are mandatory, not optional:

1. **API-key templates.** The feed carries `${INFURA_API_KEY}`-style URLs — two
   on chain 1 alone — which would be written into `erpc.yaml` as dead upstreams.
2. **Liveness.** The feed has *no* liveness data. chainlist.org's "active"
   indicator is a client-side probe the website performs, not a field. So
   "active" requires probing `eth_chainId` ourselves and keeping only endpoints
   whose reported id matches.

WebSocket endpoints are probed on equal terms rather than trusted. This is not
theoretical: **all four `wss://` entries for chain 943 fail the handshake**
(HTTP 200 or 401). Trusting the feed would have written dead WS upstreams into
every testnet gateway config.

## Gaps that are ours to close

- **No published images.** `valve-tech/reth`, `lighthouse-pulse` and `erpc` all
  publish nothing. Adding a build-push job to those three is what makes the
  Docker column real.
- **No arm64 PulseChain images**, so node-in-Docker on Apple Silicon means qemu
  emulation — too slow to sync. Ethereum is fine today (`paradigmxyz/reth` and
  `sigp/lighthouse` are multi-arch).
- **Chain data over a VM filesystem** (virtiofs on Mac/Windows) is poor for the
  random-write workload reth/geth do. A node problem; a non-issue for a gateway.
- **`--platform linux/arm64` is required on `docker run`**, not just build. The
  Docker CLI under Rosetta reports x86_64, so a locally-built arm64 image is
  reported "not found locally" and a pull is attempted.

## TLS

An `https://` page cannot call an `http://` RPC endpoint. Chrome and Firefox
exempt `http://localhost`; **Safari does not**, and *any* non-localhost bind
(LAN, Tailscale, public) is blocked everywhere — which is exactly the remote
and fleet cases. Ranked:

1. **`tailscale cert`** — a real Let's Encrypt cert for `*.ts.net`, per-device,
   no domain ownership, no trust-store import, no third party in the data path.
   The app already grades Tailscale binds as a pass tier.
2. **Public A record → 127.0.0.1 with DNS-01** — the Plex `*.plex.direct`
   pattern, proven at scale. HTTP-01 cannot work, since the name resolves to
   loopback. Per-device keys generated locally; never ship a shared wildcard
   key to users.
3. **mkcert / Caddy internal CA** — offline, but per-device trust import, and it
   teaches users to click through cert warnings.
4. **ngrok / Cloudflare Tunnel** — easy, but routes RPC through a third party,
   undermining the reason for running your own node. Offer it; don't default to
   it; say what it costs.

Choose per bind tier rather than picking one mechanism.

## Build order

1. Gateway entity + multi-chain rendering — **done**
2. Upstream discovery + liveness probing — **done**
3. Docker probe + container rendering — **done**
4. Gateway setup path (`PlanGateway`: preflight, config, run) for both backends
5. Publish a multi-arch `valve-erpc` image from `valve-ws`
6. Goal-first UI: what do you want → where can it run
7. Local nodes auto-populate as upstreams — the fleet view
8. TLS per bind tier

Cross-cutting, per the owner: everything **editable** and **replicatable**
(save/clone a prior setup).

## Corrections to existing data

`learn/data/networks.ts` stores `snapshot.sizeTB` — the size of Valve's **reth
snapshot artifact**. The catalog imported it as `ArchiveSizeTB` and then
derived a full tier by halving it, a number with no source. Both are wrong: the
figure is reth-specific, and the halved value is invented. Renaming to
`SnapshotSizeTB` and isolating the placeholder is in flight; real per-client
figures have to be measured or supplied, not guessed.

Two gotchas in learn data are missing from the catalog: `lighthouse-pulse`
pins `RUSTUP_TOOLCHAIN=1.81.0`, and `erigon-pulse` below v2.3.0 needs
`--externalcl`.
