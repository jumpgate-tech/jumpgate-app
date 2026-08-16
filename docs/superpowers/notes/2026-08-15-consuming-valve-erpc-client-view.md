# Consuming valve's eRPC fork: the client view

Date: 2026-08-15
Status: Reference note. jumpgate is a client of valve's eRPC fork, never a contributor to it.

## Why this note exists

jumpgate operates valve's eRPC fork. It does not develop it. The fork work — the
upstream bug fixes, the one fork bug, and the polyglot svm/btc architectures —
lands in the `valve-tech/erpc` repo. jumpgate only consumes the results. This
note records the four seams through which jumpgate takes those updates. Future
work touches these seams and leaves the eRPC internals alone.

## Seam 1: adopt a fork update by moving one SHA

jumpgate builds eRPC on the target from source, at a pinned commit. It does not
pull an image. Upstream eRPC has no WebSocket support, and `valve-tech/erpc`
publishes no image, so building from the fork is the only way to get an
eth_subscribe-capable gateway. The two constants that pin the source live in
`internal/ops/docker.go`:

- `ERPCSourceRepo` (line 566) = `https://github.com/valve-tech/erpc.git`
- `ERPCSourceRef` (line 567) = a full commit SHA

The ref is a full SHA, not a branch. A full SHA is a valid Docker git ref, so
the build stays reproducible against a feature branch that keeps moving
(`ERPCBuildContext`, line 1327). The doc comment above the constants (lines
550-565) names the two fixes on the current branch: a WebSocket upgrade that
answered HTTP 500 under `Accept-Encoding: gzip`, and a multi-chain batch that
answered every entry from one entry's network through a data race.

Moving the SHA is the whole adoption lever. When a fix or the polyglot work
lands on the fork, jumpgate ships it by bumping `ERPCSourceRef` to the new
commit. The built image is tagged by SHA — `ERPCImageTag` (lines 1316-1322)
truncates the ref to eight characters and appends it to `valve-node-app/erpc`.
A bump therefore yields a distinct image and forces a clean rebuild. Leaving the
SHA behind silently keeps building the unfixed binary and reports success either
way.

Note the fallback. `DefaultERPCImage` (line 580) = `ghcr.io/erpc/erpc:0.1.1` is
the pulled image for operators who prefer not to build. It is upstream eRPC with
no WebSocket support. It is correct only for pure request/response RPC;
eth_subscribe against it fails with `ErrNoWsUpstreamAvailable`.

## Seam 2: new architectures fill the `:arch` slot

Today the architecture is hardcoded to `evm` in exactly two places. Both are in
`internal/catalog/gateway.go`:

- The `erpc.yaml` template line `architecture: evm` (line 225).
- `GatewayConfig.PathFor`, which returns `/<project>/evm/<chainId>` (line 328,
  `fmt.Sprintf("/%s/evm/%d", ...)`).

When valve's fork serves svm or btc, the client-side change is to make the
architecture a variable in those two spots, plus the metered URL shape
`/rpc/<key>/<arch>/<chainId>`. The URL shape will not have to change. The
self-hosted-metered-rpc roadmap already carries `<arch>` in the URL for exactly
this reason — see `docs/superpowers/plans/self-hosted-metered-rpc.md` (the proxy
maps `/rpc/<key>/<arch>/<chainId>`, and the plan notes svm/btc are real but not
in the first milestone). The polyglot work is fork-side; widening `evm` to a
variable is the client-side match.

## Seam 3: valve's hosted eRPC as an upstream provider

jumpgate also lists valve's public metered endpoint as a known upstream. This is
jumpgate acting as a plain client of valve's running service, separate from the
gateway it builds itself. The templates live in `internal/catalog/knownset.go`:

- `valveURLTemplate` (line 43) = `https://one.valve.city/rpc/${VALVE_API_KEY}/evm/%d`
- `valveWSURLTemplate` (line 44) = the `wss://` twin

There are two because eRPC infers WebSocket capability from the scheme. The same
`:arch` slot applies here when valve serves more architectures. The key sits in
the path, not a header, and resolves at `chainlist.Resolve`. `DefaultValveKey`
(line 20) = `vk_demo` is a published, non-secret default; it ships in the binary
and reaches the browser unredacted.

## Seam 4: read eRPC's output through the metric-label contract

eRPC's own metric labels are the last seam. `NetworkTraffic.Network` carries
eRPC's label value, e.g. `evm:369` (`internal/metrics/traffic.go`, line 30), and
`ChainID` is parsed out of the `evm:<n>` shape. The traps in these counters are
already recorded in the `valve-node-app-erpc-and-testing-gotchas` skill. Read
that skill rather than re-deriving them. The traps, by name:

- The state poller inflates counts — poller counts versus the real
  `erpc_network_*` family.
- Phantom upstream label values — `n/a`, `<error>`, and `*` render as servers
  that do not exist.
- The `erpc_selection_position` ambiguity — a value of `0` means two different
  things.

## The whole client side in one line

Consuming a valve-eRPC update means: bump the SHA to adopt a build, widen `evm`
to a variable architecture when polyglot lands, point the URL templates at the
new arch, and trust only client-facing labels. No fork work happens on
jumpgate's side.
