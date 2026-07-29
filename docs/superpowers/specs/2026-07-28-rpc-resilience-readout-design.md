# The RPC page as a resilience readout, and a known set worth one click

**Date:** 2026-07-28
**Status:** approved, implementing

The RPC page can say what a gateway is configured to do. It cannot say whether that
configuration would survive anything. Every chain on this machine runs on exactly one
endpoint, two of the three cannot serve `eth_subscribe` at all, and the page reports
none of that — it lists upstreams and leaves the reader to work out the consequences.

This turns the page into an answer to one question — *can each chain I serve still be
answered if something goes down* — and makes fixing the gap a single click, using a
vetted set the codebase already contains.

## What was measured first

Every endpoint below was probed from this machine on 2026-07-28. Latency is one
sample. **Archive** means `eth_getBalance` at block 1,000,000 answered rather than
erroring. **WebSocket** means the upgrade completed, not that the docs claim support.

| Chain | Endpoint | Latency | Archive | WebSocket |
|---|---|---|---|---|
| evm:1 | `one.valve.city/rpc/vk_demo/evm/1` | 295 ms | yes | yes |
| evm:1 | `eth.drpc.org` | 197 ms | yes | yes |
| evm:1 | `ethereum-rpc.publicnode.com` | 237 ms | **no** | yes |
| evm:1 | `eth.merkle.io` | 252 ms | yes | **no** |
| evm:1 | `cloudflare-eth.com` | 266 ms | error | — |
| evm:369 | `one.valve.city/rpc/vk_demo/evm/369` | 247 ms | — | yes |
| evm:369 | `pulsechain-rpc.publicnode.com` | 197 ms | — | yes |
| evm:369 | `rpc-pulsechain.g4mm4.io` | 274 ms | — | **no** |
| evm:369 | `rpc.pulsechain.com` | **1700 ms** | — | **no** |

Four findings that decided the design:

1. **publicnode is not archive on Ethereum** — it returns `"Archive requests
   require…"`. It is the endpoint people reach for as a general default, and it cannot
   serve history.
2. **merkle is archive but has no WebSocket.** It and publicnode are complements, not
   alternatives; a set containing only one of them loses a capability.
3. **`rpc.pulsechain.com` is ~9× slower than its alternatives** (1700 ms vs 197–274 ms).
   It is the official endpoint and belongs in the set, but preferring it is wrong — and
   today it is the only upstream configured for that chain.
4. **valve's endpoint passes everything on both chains**, and covers 943 as well
   (`0x3af`). Its URL is `https://one.valve.city/rpc/<key>/evm/<chainId>` — the key sits
   in the path, so it is per-request, not a header.

Also true of this machine right now, and currently invisible on the page: every public
upstream is scored `tier:fallback` at `overall: 0.2`. That is correct when a local node
serves the chain, and wrong on `evm:1` and `evm:369`, where the de-prioritised fallback
is the **only** path.

## 1. The page is one machine's gateway, not a list

A machine hosts one managed eRPC, so the gateway-list wrapper and its "Add a gateway"
button describe a collection that cannot have a second member. The page becomes: a quiet
infrastructure line (machine, image, TLS, base URL), then the chains.

Adding a gateway appears only for a registered machine that has none.

## 2. Each chain is a row, and the row leads with redundancy

Redundancy is a count, so the count is the mark. Each chain shows a segmented bar: filled
segments are endpoints answering now, hollow segments are ones that could be added. Four
segments, matching the set size below.

Under it: the endpoints themselves, each tagged with what it actually does — `websocket`,
`archive`, `fallback tier` — and one sentence naming the gap, in place of a banner. The
sentence states a fact and its consequence, e.g. *no WebSocket upstream, so
`eth_subscribe` fails on this chain*.

Colour carries meaning only: teal answering, amber degraded-or-single-path, red down.
Nothing is coloured for decoration — on an operations surface a decorative accent is a
lie about state.

## 3. The five warning surfaces collapse into the row

Today a gateway renders `errorBlock`, `blocked`, `warnings[]`, `tlsBanner` and
`actionErr` as five separate banners, plus page-level orphan banners. They become one
attention strip per subject, carrying the exact command where one exists — the orphan
banner's `docker rm -f` is the model, since it tells the operator precisely what to run.

Orphan records keep their own strip: a leftover container is not a property of a chain,
and it is the one warning that blocks provisioning outright (measured: a merged-away
gateway still publishing `127.0.0.1:4001` fails the survivor's metrics-port preflight).

## 4. The known set: four providers, hardcoded, ordered by capability

`chainlist.Vendored()` already holds a curated per-chain list and is only consulted when
`chainid.network` is unreachable. It is promoted to the primary action — "Add valve's
set" — with "choose from N discovered" demoted to secondary.

The set is **four providers per chain**, listed in the order eRPC should prefer them.
A provider that serves both schemes contributes **two entries**, an `https://` and a
`wss://` one, because eRPC infers WebSocket capability from the scheme and has no
separate flag — so the entry count (what the button adds, and what the redundancy bar
counts against) is always the larger number:

**evm:1** — valve, drpc, publicnode, merkle. **Seven entries.**
Archive on three providers (valve, drpc, merkle — five entries, since valve and drpc are
archive on both schemes) and WebSocket on three (valve, drpc, publicnode — three
entries, every one `wss://`), so neither capability rests on one provider.

**evm:369** — valve, publicnode, g4mm4, `rpc.pulsechain.com`. **Six entries**, WebSocket
on two providers (valve, publicnode). The official endpoint is last, on measurement.

**evm:943** — valve, plus the existing vendored testnet entries. **Six entries**,
WebSocket on two providers (valve, publicnode).

valve carries a `wss://` entry on all three chains because all three were measured on
2026-07-28: dialed with `internal/wsrpc`, upgrade completed, `eth_chainId` answering
`0x1`, `0x171` and `0x3af`. An unmeasured WebSocket claim is precisely what the scheme
split exists to keep out of the set.

Before it is offered, the set is probed, and the preview shows what will be added versus
what is already present, so the resulting count is never a surprise. An endpoint already
configured is shown as *already added* rather than silently skipped.

## 5. The valve key is per chain, defaulting to vk_demo

The set ships `vk_demo` so it works with no setup. Where the operator has supplied a
valve key, that key is substituted into the path.

**The key is configured per chain, not once globally.** A key's entitlements are a
per-chain matter, and an operator may want valve on one chain and not another; a single
global key cannot express either. The stored shape is therefore a key per chain id, with
`vk_demo` as the fallback when none is set. The app already collects a free `vk_` key for
snapshots (`catalog/snapshot.go`), so the input and its wording are established.

`vk_demo` is a shared quota. The failure mode if it runs dry is that **valve's own
endpoint becomes the least reliable entry in a set that lists it first**, so the UI names
the key in use per chain and says plainly that a free key of your own removes the shared
limit.

## 6. Public upstreams are not fallbacks when they are all you have

`tier:fallback` with `overall: 0.2` is applied to every external upstream. It is added
only when the chain also has a managed node or devnet upstream. A chain served entirely
by public endpoints gets them at normal weight, because there is nothing for them to be a
fallback *to*.

Existing configs are not rewritten: this governs what the set adds, and the page reports
the contradiction where it already exists.

## Delivery

1. The known set + per-chain key, behind the existing picker.
2. The chain row: redundancy bar, capability tags, gap sentence.
3. The page shell: one gateway, collapsed warnings.
4. The fallback-tier rule.

§1 is the useful half on its own and lands first; the page can be reshaped around it.

## Verification

- Go tests for the set (contents, ordering, key substitution per chain, dedupe against
  what is configured) and for the fallback-tier rule.
- `cd cmd/valve-node-app/web && npm run build`, then `go build ./...` so the rebuilt
  `dist/` still embeds.
- **Add the set on this machine and watch a chain go from one endpoint to four**, then
  `eth_subscribe` on `evm:1` — which cannot work today, because no Ethereum upstream
  speaks WebSocket. That is the check that the set did something, rather than that the
  code ran.

## Explicitly not in scope

- Rewriting existing upstream weights.
- Any per-endpoint health history or charting; the row states current capability only.
- A fleet view. One machine is assumed; if the remote box behind the SSH tunnels becomes
  a managed target, the page needs a way to name which machine it is showing.
