# Agent brief — generalize the valve-tech/erpc fork into a polyglot proxy

Hand this to an agent that will work in the eRPC fork.

## Mission

The direction is DECIDED (owner, 2026-08-13): extend our eRPC fork into a
polyglot proxy where **the chain pattern lives in CONFIGURATION**, so adding a
new chain is a config exercise, not new Go code and not a second proxy per
chain. A per-chain balancer was rejected outright — Bitcoin, Zcash, Dogecoin,
Solana, and more are all coming, and we will not write or run one product each.

Today eRPC load-balances Ethereum JSON-RPC across upstreams with health checks
and failover. We want that same machinery driven by a **declarative chain
pattern**: each pattern says how to health-check an upstream, how to read its
tip/height, and how to classify a failure/rotate — expressed in config, not
hardcoded per chain. Bitcoin (bitcoind JSON-RPC) and the Ethereum beacon
(consensus REST) are the first two new patterns; the design must generalize to
arbitrary chains.

Your job is therefore NOT "should we extend or adopt" — that is settled. It is:
find the seams, confirm the config-driven pattern is achievable there (flag a
hard blocker if one truly exists), and design it. Keep every change additive so
we can still pull upstream eRPC.

## Where the code is

- Fork: `/Users/michaelmclaughlin/Documents/valve-tech/github/erpc/erpc`
- Remote: `git@github.com:valve-tech/erpc.git`
- Current branch: `fix/empty-result-rotation` (note: failure/empty-result
  rotation is already being worked here — your abstraction must fit that work,
  not fight it). Confirm the branch to build on with the owner.
- Language: Go. The tree is a flat package plus `testdata/`.

## Step 1 — feasibility map (do this FIRST, report before designing)

Read the fork and produce a map, with `file:line` evidence, of:

1. **Upstream model** — how an upstream is defined, configured, and selected for
   a request. Where does `architecture: evm` (or any chain-type field) enter?
   Grep starts: `architecture`, `chainType`, `ChainType`, `bitcoin` already
   appear in `erpc.go`, `request_processor.go`, `grpc_server.go`.
2. **Health / liveness** — `healthcheck.go` and anything that decides an
   upstream is up, synced, or lagging. What EVM calls are hardwired
   (`eth_syncing`, `eth_blockNumber`, `eth_chainId`)?
3. **Tip / height** — how eRPC learns an upstream's latest block, and where that
   feeds "serve from the most caught-up upstream" / finality / block-number
   integrity (there are `http_server_blocknumber_*` tests — read them).
4. **Failover / failsafe** — the retry/hedge/rotation logic (`failsafe_*`,
   `consensus_*`, `networks_*rotation*`, the `fix/empty-result-rotation` work).
   How is "this response means try another upstream" decided today?
5. **Request shape** — JSON-RPC method routing, batch handling, caching keyed on
   block number. What assumes JSON-RPC specifically (a problem for beacon, which
   is REST)?

Deliver a **verdict**: is eRPC's core genuinely generalizable behind an
interface, or is it so EVM-coupled that a sibling proxy for non-EVM is the
saner build? Say which, and why, with evidence. If the honest answer is "don't
extend it," say so — that is a valid and useful outcome.

## Step 2 — the config-driven pattern (if feasible)

Design the **declarative chain pattern**. The bar: a new chain is added by
writing config — a pattern block — not by adding a Go type. Prefer config-driven
probes (a method/path to call + a JSON path to read + a rule to judge it) over a
compiled adapter per chain; fall back to a small pluggable Go interface ONLY for
the parts config genuinely cannot express, and say which those are.

The behavior a pattern must capture (a sketch to react to, not to copy):

- **Health(ctx, upstream) → (alive, synced bool, detail)** — EVM: `eth_syncing`
  + `eth_blockNumber`; Bitcoin: `getblockchaininfo` → `verificationprogress`,
  `blocks`, `headers`; beacon: `GET /eth/v1/node/health` (200 synced / 206
  syncing / 503 down).
- **Tip(ctx, upstream) → height** — for "pick the highest / most caught-up".
  EVM: block number; Bitcoin: block height; beacon: head slot.
- **ClassifyResponse(req, resp) → {serve | rotate | clientError}** — the
  family's take on "empty result / not-found / behind ⇒ rotate to another
  upstream" (this is the live `fix/empty-result-rotation` question, generalized).
- **Transport / routing** — is this family JSON-RPC (EVM, Bitcoin) or REST
  (beacon)? How a request maps to an upstream call, batch semantics, and whether
  block-keyed caching even applies (it does not for beacon; Bitcoin differs).
- **Config surface (the heart of this)** — the pattern is DECLARED in config:
  the probe call + how to read it, the tip path, the rotate rule, and the
  transport (JSON-RPC vs REST). Extend `architecture:` to name a pattern, or add
  a `family:`/`pattern:` block. A new chain = a new config block. Keep existing
  EVM configs working unchanged.

State clearly what is IN scope for v1 (health, tip, failover, routing) and what
is deferred (family-specific caching/finality can wait — do not block v1 on it).

## Step 2b — observable at every level (owner requirement)

Each layer must carry a test that shows the operator whether it works. For a
pattern that means: a probe result per upstream ("healthy / syncing / down",
with the tip it reported), surfaced so the UI can say a chain's pool is serving
or not. Design the pattern so its health check doubles as this operator-facing
signal — the same result gates rotation AND answers "is this chain enabled and
working." Do not build a silent balancer.

## Step 3 — spike (if feasible)

Implement a minimal **Bitcoin** family: health-check a real (or faked) bitcoind
via `getblockchaininfo`, extract height, and fail over to the most caught-up
upstream. TDD. Prove EVM still passes its existing tests unchanged. Keep the
spike additive.

## Constraints

- **Merge-friendly.** We track upstream eRPC. Prefer an additive interface at a
  clean seam over an invasive rewrite; note anywhere you must touch shared code
  and how risky an upstream merge becomes.
- **Do not break EVM.** Existing behavior and tests stay green.
- Go, standard fork conventions, TDD, evidence with `file:line`.

## Why extend rather than adopt (context, already decided)

The owner has chosen to extend. This section is context for why, and a signal to
STOP and escalate only if your feasibility map finds a true hard blocker. Do not
re-open the decision otherwise.

Findings from our own research (2026-08, so you need not repeat it):
- **Dshackle** (emeraldpay, Apache-2.0; the actively-maintained line is the
  dRPC fork `drpcorg/dshackle`, release v0.79.13) is the ONE purpose-built
  polyglot balancer that fronts BOTH Bitcoin and EVM. It is chain-aware —
  tracks height/peers/state, drops a lagging node, round-robins with failover.
  Costs: a JVM service, its own config model beside eRPC, and its own docs say
  "use with caution / pin a release."
- **Beacon REST** needs no special balancer — plain L7 with
  `/eth/v1/node/health` (200 in, 206/503 out) is the standard.
- **Bitcoin** via a generic L7 proxy works too, but the honest health gate needs
  a side service that parses `getblockchaininfo` (`initialblockdownload`,
  `verificationprogress`) — you build the "is it caught up / on the best tip"
  logic yourself.

Dshackle proves a polyglot balancer is a solved shape; we are choosing to own
it in our fork, config-driven, rather than run a JVM service with a second
config model beside eRPC. Your feasibility map answers HOW cleanly the
config-driven pattern plugs into our fork — not WHETHER to extend.

## Deliverables

1. The feasibility map (Step 1) — the seams, the EVM assumptions, and a clear
   statement of whether the config-driven pattern is achievable there (escalate
   only a true hard blocker).
2. The declarative chain-pattern design (Step 2) + the per-level health/observe
   surface (Step 2b).
3. The Bitcoin spike + green EVM tests (Step 3).
Report with `file:line` evidence throughout. Do not push; branch and report.
