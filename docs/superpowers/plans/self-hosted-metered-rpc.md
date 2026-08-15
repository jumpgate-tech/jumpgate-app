# Self-hosted metered RPC — the thin valve replica (roadmap)

Date: 2026-08-15. Status: design approved; slices spec'd and built one at a time.

This is a roadmap, not a spec. Each slice below gets its own spec in
`docs/superpowers/specs/` before it is built.

## Goal

Let a jumpgate operator run their own metered, branded public RPC on their own
domain — the `one.valve.city` model, self-hosted. A customer calls
`https://rpc.operator.com/rpc/<key>/evm/<chainId>`, the operator counts that
customer's use, deducts prepaid credits, and disables the key when credits run
out. Credits are topped up **on-chain only**.

## What "thin" means

"Thin" describes HOW we build each feature, not WHICH features we keep. Nothing
valve does is dropped. The rule is **delegate, do not rebuild**:

- Upstream selection, failover, caching, WebSocket multiplexing → **eRPC**. We
  already generate its config.
- Domain routing, TLS, the public front → **Caddy**. We already generate its
  Caddyfile.
- What we build is the thin glue between them, plus a small control plane.

## The reference model (valve's real architecture)

Confirmed by reading valve's `monorepo`. Two facts shaped this plan:

1. **eRPC runs keyless.** valve does NOT use eRPC's auth/key features. eRPC is a
   dumb upstream-fanout engine. All auth, metering, rate limits, and credits
   live in a **thin relay in front of eRPC**. The relay validates the key,
   **strips it**, and forwards a plain JSON-RPC call to eRPC. eRPC never sees a
   key. (An earlier draft of this plan assumed eRPC's `database` auth strategy
   and its `user` metric label — both wrong for this model.)
2. **Metering is per-request logging, not eRPC counters.** valve writes each
   request to Postgres from the relay (chain, method, key, latency, status).
   eRPC's own counters are labelled by chain, not by key — and they lie in known
   ways (see the `valve-node-app-erpc-and-testing-gotchas` skill).

Billing in valve is 100% crypto prepaid credit (x402 / permit2, a FIFO credit
ledger). No Stripe, no fiat. Our thin version keeps the on-chain-credit idea and
drops the x402 machinery (see slice D).

## Target architecture

```
customer (wallet / app)
   │  https://rpc.operator.com/rpc/<key>/evm/<chainId>
   ▼
Caddy   (operator domain · auto Let's Encrypt cert · strips /rpc)
   ▼
jumpgate relay  (Go — NEW)
   │  key → keyConfig (embedded store) → validate → check credits → log usage → STRIP key
   ▼
eRPC    (keyless · UNCHANGED · /main/evm/<chainId>)
   ▼
upstreams
                                    ▲
on-chain payment watcher → credits ledger ┘   (disable key at 0)
customer dashboard (wallet-signature login) → create key · see usage · top up
```

jumpgate is already a Go server that runs and configures eRPC, so "the relay" is
a natural fit — a Go handler jumpgate hosts. eRPC stays exactly as it is today.

## The slices

Each needs the one before it, except slice 0 (independent). `E` grows alongside
`B`–`D`.

| # | Slice | Adds (thin) | Delegate vs build | Weight | Status |
|---|-------|-------------|-------------------|--------|--------|
| 0 | Trust retry button | Graceful degrade + "Try again" on the Private (`tls internal`) tier | — | XS | done (a9ad970) |
| A | Public branded endpoint | "Public" cert source: operator domain + auto Let's Encrypt cert on the gateway's existing path (no keys yet) | Caddy (stock image) | S | spec written |
| B | Keyed access (the relay) | Go proxy: key → validate → strip → forward; embedded key store; issue/revoke; method allow/deny | Build; eRPC stays keyless | L (the heart) | — |
| C | Per-key metering | Per-request log + a couple of GROUP BY usage views; per-key usage in the UI; CSV/JSON export | Build (thin) | M | — |
| D | Credits + on-chain top-up | Integer credit ledger; on-chain payment watcher; per-request deduct; disable-at-zero | Build (thin — NOT x402/permit2) | M | — |
| E | Customer dashboard | Wallet-signature login; a subset of valve's web (create key, see usage, top up) | Build a subset | M | — |

### Slice 0 — Trust retry button (Private tier)
The one manual step for `tls internal` is trusting the gateway's root. On macOS
launched detached it fails "no user interaction was possible". Slice 0 adds a
truthful "Try again" — a verify probe (`security verify-cert`) reports success
when the cert is already trusted, so a retry after a hand-run `sudo` does not
lie. Independent of the rest.

### Slice A — Public branded endpoint
Fill the reserved `CertACME` cert source. The operator sets a real domain;
stock Caddy gets a Let's Encrypt cert over HTTP-01/TLS-ALPN (the box is publicly
reachable, so no DNS-01, no custom image). It serves the gateway's EXISTING path
on the operator's domain — no `/rpc` stripping, no keys yet (those need the relay
in slice B). This is the trusted, branded, public foundation the rest sits on.
Forks settled in its spec: the three tiers reuse the existing cert-source field
(no new enum), the `:80` bind for the ACME challenge, and the health-probe change
once the name is genuinely public.

### Slice B — Keyed access (the relay)
A Go proxy in front of eRPC. It maps `/rpc/<key>/<arch>/<chainId>`: validate the
key, apply method allow/deny, strip the key, forward to eRPC's
`/<project>/<arch>/<chainId>`. The key is a PATH segment (eRPC will not read a
key from the path — the relay is the required translation layer). Forks: the
embedded store (SQLite, since jumpgate has no DB today), and keeping the public
data plane separate from jumpgate's private loopback control plane.

### Slice C — Per-key metering
Log each request to the store (chain, method, key, latency, status). A few
GROUP BY queries drive per-key usage in the UI and a CSV/JSON export. Do not
trust eRPC's counters for per-key attribution — count in the relay.

### Slice D — Credits + on-chain top-up
An integer credit ledger. A watcher reads the chain for incoming payments and
credits the matching customer. Usage from slice C deducts credits; at zero the
key is disabled. Fork: how a payment maps to a customer — a per-customer deposit
address, a registered paying wallet, or a `pay(keyId)` contract whose events
carry the key. NOT valve's x402/permit2/treasury subsystem.

### Slice E — Customer dashboard
Wallet-signature login (one EIP-712 verify + a session token, as valve does). A
small React surface: create a key, see usage, top up. A subset of valve's web,
not the whole thing. No org/team model.

## Build order and gates

- Build order: 0 → A → B → C → D, with E alongside B–D. Billing stays on-chain
  only.
- Each slice: spec in `docs/superpowers/specs/` → user review → TDD build →
  verify by running it (real container / real probe, not the artifact) → commit.
- The dist embed is CI-enforced: any `web/src` change needs `npx vite build` and
  a committed `dist`.
- Never `git push` until asked. Commit on `main`, sign with the jumpgate key.

## Out of scope

Not part of the charge-for-RPC replica: the provider marketplace / relayer
bazaar, snapshot byte-metering, the Monero RPC, the status page (moved to
`thatis.online`), and every sibling product (msgboard, prove.cash, explore,
evm-toolkit, substreams/firehose). `:arch` beyond `evm` (svm, btc) is real
eRPC-fork work already tracked in the `erpc-polyglot-*` plans; v1 serves `evm`
only and just carries `:arch` in the URL so the shape does not change later.

## Grounding sources

- eRPC capabilities (auth, metrics, dynamic config, rate limits) — researched
  against docs.erpc.cloud and the erpc source, 2026-08-15.
- valve feature inventory — read from `valve-tech/github/monorepo`, 2026-08-15.
- jumpgate insertion points for slice A — `internal/catalog/caddy.go`
  (`CertACME` reserved at ~:45, `caddyfileTemplate` at ~:294, the ACME rejection
  in `Validate`), `internal/ops/docker.go` (`CaddyRunArgs` ~:1106 publishes only
  :443; `LoopbackRPCPort` ~:745), `internal/setup/gateway.go` (`checkPortFree`,
  the health probe), `cmd/valve-node-app/web/src/screens/Rpc/SettingsBlock.tsx`
  (cert-source select ~:214).
</content>
</invoke>
