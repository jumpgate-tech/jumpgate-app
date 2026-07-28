# The analytics page

**Date:** 2026-07-28
**Status:** approved, implementing

The Control Surface answers detection: what can this endpoint do, and is it carrying the
share you intended. It deliberately does not answer diagnosis — how is it doing — and
says so in its own header comment. This is that page.

Everything it needs is already in the dump the gateway serves on loopback. No new
scraping mechanism, no storage, nothing sent anywhere.

## What was measured first

Read off `internal/metrics/testdata/erpc.txt`, a real dump from a real gateway, not off
eRPC's documentation. Five client requests had been sent through it (`eth_blockNumber`
on `evm:369`, via curl); the gateway had been up long enough for its state poller to
make several hundred calls of its own.

1. **`erpc_network_request_duration_seconds` is client-facing and carries the upstream
   label.** Its `_count` reads exactly 5 — the five real requests — with
   `network="evm:369"`, `upstream="public-369-1"`, `category="eth_blockNumber"`. This
   is the honest latency source: per chain, per endpoint, per method, on the path
   clients actually take.

2. **Its histogram has four buckets: 0.05, 0.5, 5, 30, +Inf.** That is coarse enough
   that a p95 drawn from it would be fiction. It is *not* too coarse to say "5 of 5
   answered in under 500ms", which is the same fact without the invented precision.
   The page renders the buckets as bands and the mean as a number, and computes no
   quantiles.

3. **`erpc_upstream_request_duration_seconds` is NOT client-facing, and its `network`
   label does not reveal that.** The dump has 413 `eth_getBlockByNumber` calls to
   `public-1-1` labelled `network="evm:1"` — a chain no client ever called, since
   `erpc_network_request_received_total` has no `evm:1` row at all. This is the state
   poller, wearing a real network label. It is the same trap
   `erpc_upstream_request_total` set for traffic share, one family over, and it is
   worse here because the earlier finding ("the poller shows up as `network=n/a`")
   suggests a filter that does not work.

4. **`erpc_upstream_request_errors_total` is mostly poller errors, and they are worth
   showing.** The dump's only row is 56 `ErrEndpointTransportFailure` against
   `upstream="devnet"` at `severity="critical"` — the devnet was down. No client saw
   any of them. They are still the single most useful line in the dump: an endpoint
   that cannot be reached AT ALL, discovered before a client finds it.

5. **`erpc_selection_score` / `_position` / `_primary_switch_total` say WHY eRPC is
   choosing what it chooses** — `public-1-1` scored 0.105, `public-369-1` scored 0.047,
   both at position 0. This is the direct answer to the question the Control Surface's
   amber share bar raises and cannot answer.

## The shape of the page

Two sections, labelled for what they are, never averaged together:

**What your clients experienced** — from the `erpc_network_*` family only.
Per chain: requests received, answered, failed (the received/answered gap, already
computed for the share bars). Per method and per endpoint within a chain: request
count, mean latency, and the bucket bands.

**What the gateway sees from your endpoints** — from the `erpc_upstream_*` and
`erpc_selection_*` families, with the poller's contribution stated in the copy rather
than filtered out (it cannot be filtered out reliably — finding 3). Per endpoint:
error classes with counts and severity, head lag, finalization lag, selection score,
position, and how many times eRPC has switched primary.

Conflating the two is the specific failure this page exists to avoid. A section header
that says which question a number answers is cheaper than a footnote nobody reads.

## Rate and history without storing anything

The counters are cumulative since the gateway process started, and the server stays
stateless — `traffic.go` already refuses to cache for the same reason.

- **Totals** are shown against `Since`, exactly as the share bars are.
- **Average rate** since start is total ÷ uptime, which needs no second reading.
- **Live rate** is computed in the browser by diffing consecutive polls. It is
  labelled with the window it covers, and it is empty until the second poll — which is
  honest, not a bug to paper over with a zero.
- **History** is a client-side ring of the last readings, drawn as a sparkline, and it
  covers "since you opened this page". Nothing survives a reload, and the page says so.

A counter that resets is a gateway that restarted: a reading whose value fell below the
previous one is dropped from the rate rather than rendered as a negative or an
enormous positive.

## Delivery

1. `internal/metrics` — `Analytics`, `AnalyticsFromSamples`. Pure, as the rest of the
   package is: bytes in, values out.
2. `internal/setup` — one scrape, two readings. `ReadGatewaySamples` is factored out of
   `ReadGatewayTraffic` so analytics costs no second curl, and both callers share the
   failure wording.
3. `internal/server` — `GET /api/gateways/{gid}/analytics`, uncached, 200-with-error on
   an unreadable gateway exactly as `traffic` answers.
4. `analytics.ts` at `#/analytics/<gid>`, linked from each gateway's bar on the Control
   Surface.

## Verification

Not the tests. Both gateways on this machine get real traffic sent through them — a
slow method and a fast one, a failing endpoint and a healthy one — and every number on
the page is checked against the raw dump read independently. A number that renders is
not the same as a number that is right.
