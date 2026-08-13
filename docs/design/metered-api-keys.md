# Jumpgate metered API-key and billing service — design

Status: draft for review. Date: 2026-08-11. Author: design pass.

## 1. Purpose and hard constraint

Jumpgate needs a thin service that issues API keys, meters RPC usage per key,
and bills that usage against an on-chain-anchored balance. The service must
co-run with eRPC on the smallest tier: a single 512 MB VPS.

The 512 MB ceiling is the design driver. eRPC is a Go process with a garbage
collector. It already claims the largest share of the box. The billing service
gets what is left, which is small. Every choice below defends that budget.

One rule frames the whole design: **anchor value on-chain once, meter off-chain
many times, settle net on a cadence.** The hot path writes nothing to a chain
and reads nothing from a chain.

## 2. Language choice — Rust

I choose Rust. The deciding factor is resident memory, not speed.

- **No second garbage collector.** eRPC already runs a Go GC. A Go billing
  service adds a second GC heap with its own `GOGC` headroom. On a 512 MB box the
  two collectors compete for the page cache. Rust adds zero GC. Its resident set
  stays flat and predictable.
- **Small, flat RSS.** A Rust service on `tokio` with two worker threads,
  `axum` for HTTP, and `rusqlite` for storage holds about 10–15 MB resident. A
  Go equivalent holds about 30–50 MB and the number moves with GC pressure.
- **Embedded store, no extra process.** `rusqlite` links SQLite in-process. This
  removes Redis and Postgres from the box. Those two alone would break the
  budget. See section 6.
- **Single static binary.** A `musl` build ships one file with no runtime. This
  matches how a small VPS should be operated.

Honest tradeoff. Go would be faster for the team to write. The one existing Go
service (`services/firehose-meter`) and eRPC itself set a precedent, and
familiarity has real value. I still pick Rust, because the constraint is the
RSS ceiling on a shared box, and a second Go GC is the exact pressure to avoid.
`modernc` (cgo-free SQLite for Go) is also heavier and slower than `rusqlite`.

Budget target on the 512 MB box:

| Process | Rough RSS |
|---|---|
| eRPC (Go) | 150–300 MB |
| Caddy (front, if present) | 20–40 MB |
| Jumpgate billing (Rust) | 10–15 MB |
| Kernel + page cache + headroom | the rest |

## 3. Shape of the service

The service is one binary with three planes:

1. **Data plane (the metering proxy).** It listens on the public edge, resolves
   the `jg_` key, weights the method, debits the account balance in SQLite, and
   forwards the request to eRPC at `http://127.0.0.1:4000`. This plane is the
   metering source. See section 4.
2. **Admin plane.** A localhost or mTLS API that issues, rotates, and revokes
   keys, binds keys to funding accounts, and reads usage. See section 7.
3. **Settlement plane.** A low-frequency worker that detects deposits and
   redeems provider earnings on a threshold or a timer. This is the only plane
   that touches a chain, and it never runs on the hot path. See section 5.

```
client --jg_ key--> [Jumpgate proxy : auth + meter + debit] --> eRPC :4000
                              |                    ^
                              v                    |
                        SQLite (WAL)        deposit watcher / settle worker --> chain
```

## 4. Metering source

### 4.1 Do not bill from eRPC counters

eRPC exposes Prometheus counters. They look like a metering source. They are
not, for two reasons.

First, eRPC does not know a Jumpgate API key. It meters upstreams and networks,
not customers. There is no per-key counter to read.

Second, the counters lie in specific ways. These traps are documented from
running the real thing:

- `erpc_upstream_request_total` counts the **state poller**, not client traffic.
- The poller is **not** reliably labelled `network="n/a"`. Poller calls appear
  under a real network label (for example `eth_getBlockByNumber` on `evm:1`), so
  a filter that trusts the `n/a` label fails silently.
- `upstream="n/a"` means served from cache, `upstream="<error>"` means the
  request failed, and `upstream="*"` is a per-network rollup. None of the three
  is a real endpoint.
- `erpc_selection_position` is `0` for both "preferred" and "never scored", and
  `-1` for an excluded upstream.
- The histogram bounds are 0.05/0.5/5/30/+Inf — too coarse for an honest p95.

The conclusion is firm: **eRPC counters are for fleet health, never for
billing.** Jumpgate meters its own traffic at the proxy, before eRPC sees it.

### 4.2 Meter at the proxy

The proxy is the single source of truth for usage. For each request it does:

1. **Authenticate.** Read `x-api-key`. Hash it with SHA-256. Look up the hash in
   an in-memory map (loaded from SQLite at boot, refreshed on change). Fail
   **closed** on an unknown key. The only exception is the published public key
   `jg_demo`, which maps to a per-IP public tier.
2. **Authorize.** Check origin, method, network, and IP rules against the
   resolved `KeyConfig`. Reject on a policy miss.
3. **Price.** Look up the method cost in credits from an in-memory price map.
   Precedence mirrors the monorepo: exact row, then a chain default, then a
   global default (`DEFAULT_CU = 20`).
4. **Debit.** Run one atomic conditional update against the account balance in
   SQLite (section 5.3). If the balance is short, return `-32005` (credits
   exceeded) and do not forward.
5. **Forward.** Proxy to eRPC at `127.0.0.1:4000`.
6. **Capture or refund.** Debit on the way in, then reconcile on the response.
   If eRPC fails the request (upstream excluded, timeout), credit the reserved
   amount back. Bill only served work. This mirrors the monorepo
   `creditCaptureFailures` idea without its Redis machinery.

Rate limiting on the public tier uses a per-(IP, method) token bucket held in
process (`DEFAULT_METHOD_RPS = 5`, hot methods raised). Per-key and unlimited
keys are exempt. This is the thin form of `method-rps.ts`.

### 4.3 A warning kept from the monorepo meter

The monorepo `relay/src/meter.ts` documents a real 2026-08 production wedge: a
credits-per-second Redis bucket whose TTL had to self-repair. The thin design
removes Redis, so it removes that failure surface. The lesson stays: any
per-second throttle must not depend on a key that can lose its TTL and wedge.
The in-process token bucket has no TTL and no external key, so it cannot wedge
the same way.

## 5. On-chain account model, decrement, and settlement

### 5.1 The pattern

The model is **prepaid on-chain escrow, off-chain metered decrement, and
net batch settlement on a cadence.** It is the recommended pattern from the
on-chain billing survey. It is the only cadence that fits RPC volume, because
it adds zero on-chain writes per request.

Why not the alternatives on this box:

- **Plain per-request x402** loses to the verify-to-settle race and to
  per-request settlement cost. Keep x402 only as the deposit rail.
- **Superfluid streaming** suits flat, steady consumers. It maps poorly to
  spiky agent traffic and needs a liquidation path. Offer it later, not first.
- **Web2 prepaid credits** is the model to copy mechanically (off-chain
  metering) and to reject in custody (the provider must not hold the balance).

### 5.2 Account and deposit

An account is a funding address. The customer funds a non-custodial escrow
contract keyed by that address. This is the on-chain anchor.

The deposit rail is x402 / EIP-3009 `transferWithAuthorization`, so the top-up
is gasless and wallet-neutral. An EOA signs; no smart wallet is required.
EIP-3009 carries a unique nonce and a `validAfter`/`validBefore` window, and the
EIP-712 domain separator binds the signature to one chain and one token. That
gives replay resistance for free.

Money math keeps the monorepo peg: **1 USD = 1e9 credits.** Keep it identical to
`api/src/credits`, or the two systems disagree on value.

### 5.3 Decrement (the hot path, off-chain)

Each account row in SQLite holds `credits_remaining` and `credits_reserved`. A
debit is one atomic statement:

```
UPDATE account
   SET credits_remaining = credits_remaining - :cost
 WHERE address = :addr
   AND credits_remaining >= :cost;
```

`rowsAffected == 0` means the balance is short. The proxy then rejects the
request. This closes the double-debit race with a single writer and no lock. Run
SQLite in WAL mode with a small page cache. Debits are short, so the single
writer is not a bottleneck at this tier.

The proxy cuts or degrades a key as its running total nears the escrow ceiling.
Exposure is therefore always bounded by escrow. The service never serves against
an unsettled promise — only against already-anchored funds. That single choice
closes the free-ride window that plain x402 leaves open.

### 5.4 Settlement (off the hot path)

Two on-chain events exist, and only two:

1. **Deposit in.** A lightweight watcher polls `eth_getLogs` on the escrow
   contract, or accepts a signed settle callback from an off-box facilitator. On
   a new deposit it inserts a `credit_deposit` row (unique `tx_hash`, idempotent)
   and increments `credits_remaining`. Late detection only delays the credit; it
   never double-credits, because `tx_hash` is unique.
2. **Provider draw out.** The provider redeems its earned total on a threshold
   (value crossed) or a timer (interval elapsed), whichever comes first, and at
   session close. One on-chain write amortizes over thousands of requests. The
   draw is bounded by the customer's signed monotonic total, so the provider can
   only ever draw what it has served.

The hot path performs neither event. It only reads and writes SQLite.

### 5.5 Key-to-account binding

The proxy authenticates on an API key, but value lives at a funding address.
Bind them explicitly. The account owner signs "API key K draws from account A"
with an EIP-191 or EIP-712 message. The admin plane verifies the signature and
stores the binding. Every debit for key K then hits account A.

Support many keys per account. ERC-7715 spend permissions or ERC-4337 session
keys scope a child key to one recipient, one token, a spend ceiling, and an
expiry. A leaked child key can then only spend within its budget. Bind by
signature, never by wallet vendor. That is what keeps the model wallet-neutral.

## 6. Storage — one embedded SQLite file

The whole persistent state lives in one SQLite database in WAL mode. No Redis.
No Postgres. This is the single biggest memory saving on the box.

Tables (thin subsets of the monorepo schema):

- `project_key` — `id`, `key_hash` (SHA-256, unique), `account_address`,
  `per_second_limit`, `per_day_limit`, `credit_exempt`, `rate_limit_mode`,
  `disabled_at`. The service stores the **hash**, never the raw key.
- `key_constraint` — origin, method, network, IP, and chain rules per key.
- `method_pricing` — `method`, `chain_id`, `credits_per_request`. Loaded into an
  in-memory map at boot and refreshed on change.
- `account` — `address`, `credits_remaining`, `credits_reserved`,
  `escrow_ceiling`.
- `credit_deposit` — `tx_hash` (unique), token amount, `credits_issued`,
  `rate_token_per_credit`, `rate_source`. The idempotent deposit ledger.
- `key_binding` — `key_id`, `account_address`, the binding signature.

The monorepo splits credit state across Redis keys `ceiling / pending / spend /
closing / cps`. The thin design collapses those into columns on the `account`
row. There is no Redis mirror, so the "credit-key mirror hazard" (the relay
meter hardcoding a copy of the api key helpers) disappears by construction.

## 7. Interfaces

### 7.1 Admin API

Bind it to localhost, or require mTLS if it must be remote. Gate every call with
a bearer token from `$JUMPGATE_ADMIN_KEY`. Fail closed.

- `POST /admin/keys` — issue a key. Return the raw `jg_` value once. Store only
  the hash.
- `POST /admin/keys/:id/rotate` — issue a new value. Keep the old value valid
  until a short TTL, then soft-delete. This lets a client roll a key with no
  outage.
- `PATCH /admin/keys/:id` — constrain origins, methods, networks, IPs, limits.
- `DELETE /admin/keys/:id` — soft-delete (revoke).
- `GET /admin/keys/:id/usage` — read the per-key rolling usage from SQLite.

This is the thin form of `api/src/router/v1/keys.ts`.

### 7.2 Binding API

- `POST /accounts/:address/bind` — body carries the key id and the owner's
  signature over "API key K draws from account A". Verify the signature against
  `:address`. Store the binding. Reject a bad signature.

### 7.3 Deposit / settlement webhook

- `POST /internal/deposit` — optional signed callback from an off-box
  facilitator that a deposit settled. Idempotent on `tx_hash`. Increments the
  balance. The on-box `eth_getLogs` watcher is the alternative when no
  facilitator is available.

### 7.4 Data plane

- `POST /` (and WebSocket) — the metered RPC endpoint. It forwards to eRPC after
  auth, pricing, and debit. This is the only high-volume interface.

## 8. Secrets and data at rest on a shared small VPS

The box is small and shared, so treat every secret as exposed to a co-tenant
process unless it is protected.

- **The DB holds key hashes, never raw keys.** A stolen database file cannot
  replay a customer's key. Hash with SHA-256 on issue, compare on use.
- **File permissions.** The SQLite file and its WAL are `0600`, owned by the
  dedicated service user. eRPC and Caddy run as other users and cannot read it.
- **The settlement signer key is the crown jewel.** Hold it in an OS keyring or
  an `age`-encrypted file loaded into memory at boot. Never write it to the
  SQLite database. Its blast radius is already bounded by the escrow model, but
  protect it anyway.
- **The admin bearer token** lives in `$JUMPGATE_ADMIN_KEY`, injected by the
  service manager, never committed and never in the DB.
- **No Redis.** Removing Redis removes a second network-listening data store and
  its default-open failure mode. That is a security win, not only a memory win.
- **Disk.** Prefer full-disk encryption on the VPS. At minimum keep the DB file
  out of any backup that leaves the box unencrypted.
- **`jg_demo` is public by design.** It is a published constant. It is not
  redacted and not billed. Keep it clearly separated from real keys, and keep it
  on the per-IP rate-limited public tier so it cannot drain an account.

## 9. Failure modes

- **eRPC excludes an out-of-sync upstream.** During that window eRPC does not
  fail over; every request fails. The proxy must return `502` with the reason and
  **refund the reserved credits**. Never bill a request eRPC did not serve.
- **eRPC down.** The proxy returns `502`. Auth and admin reads still work,
  because they only touch SQLite. Config and key management must keep working
  when the upstream is down.
- **Balance race / double debit.** The atomic conditional `UPDATE` prevents it.
  There is one writer and no lock.
- **Verify-to-settle gap.** Removed by construction. The service serves only
  against already-anchored escrow, never against an unsettled transfer.
- **Deposit watcher lag.** Credits appear late. The unique `tx_hash` makes the
  credit idempotent, so a slow or repeated poll never double-credits.
- **Signer key compromise.** Bounded by the escrow ceiling and by ERC-7715
  scoped child keys. The attacker cannot draw more than the customer signed.
- **Restart / crash.** All state is durable in SQLite. There is no in-Redis
  projection to rebuild, so recovery is a plain reopen of the file.
- **Per-second throttle wedge.** The monorepo hit a TTL self-repair wedge on a
  Redis `cps` bucket. The in-process token bucket has no external key and no TTL,
  so it cannot wedge that way.
- **Auth fails open.** The monorepo API middleware fails open for public routes.
  A billing gate must fail **closed** on an unknown key, with `jg_demo` as the
  one explicit public exception.
- **SQLite write contention.** WAL mode plus short transactions handle this tier.
  If write volume ever outgrows one writer, batch debits per short interval
  before that point is reached.

## 10. Exactly which monorepo code to port thin vs drop

The monorepo is TypeScript on Postgres, Redis, and a multi-service fleet. None of
it runs as-is on a 512 MB box. "Port thin" means reimplement the concept small in
Rust. "Drop" means do not bring it.

### Port thin (reimplement the concept)

| Monorepo item | Thin form in Jumpgate |
|---|---|
| `@valve/utils hashApiKey` | SHA-256 hash helper. |
| `packages/keystore` `KeyConfig` shape | A reduced struct: origins, methods, networks, IP rules, limits, `credit_exempt`. |
| `db-schema/api/project-key.ts` | `project_key` SQLite table (subset). |
| `api/src/auth/middleware.ts` | Proxy auth — but fail **closed**, not open. |
| `relay/src/meter.ts` (debit idea) | One atomic SQLite `UPDATE`. Drop the Lua and Redis. |
| `api/src/credits/keys.ts` (ceiling/pending/spend) | Columns on the `account` row. |
| `relay/src/method-pricing-cache.ts` | In-memory `method → credits` map from SQLite. |
| `relay/src/method-rps.ts` | In-process per-(IP, method) token bucket for the public tier. |
| `db-schema/api/credit-deposit.ts` | `credit_deposit` table, unique `tx_hash`. |
| pricing peg `1 USD = 1e9 credits` | Keep the constant, identical value. |
| `api/src/router/v1/keys.ts` | Admin key CRUD endpoints. |
| x402 / EIP-3009 deposit verify | Keep as the **top-up rail only**; verify in-service or delegate to one external facilitator URL. |

### Drop (do not run on this box)

- **Redis** entirely: pub/sub key invalidation, `keyconfig-reconcile.ts`, the
  two-store model.
- **Postgres**: replaced by SQLite.
- **The full credits engine** (~40 files): treasury, price windows, price
  sampler, Permit2 collect/execute, settle-watcher, leader election, live PulseX
  reserves. Keep a stable USD peg or one external price read.
- **The full `packages/facilitator`**: settlement queue, multi-wallet gas fleet,
  multicall, circuit breaker. Run settlement off-box, or call one external
  facilitator endpoint.
- **Per-request x402 hot path**: `per-request-x402.ts`, `per-request-payment.ts`,
  and the lock / reconciliation / recovery / drift files. This is the
  verify-to-settle race. Prepaid escrow replaces it.
- **`services/firehose-meter` (Go)**: not RPC metering. Drop.
- **`packages/bazaar`, `provider-sync` / payout, analytics attribution,
  `monero`, `byte-debit`, snapshot / bandwidth debit**: out of scope for the
  first thin cut. Defer.
- **`x402/funding.ts` three-source dashboard**: it joins a Prometheus scrape on
  the box. Drop; the metrics join does not belong on a 512 MB VPS.
- **Session-token, wallet, combined, and signature auth** beyond the admin
  bearer and the one binding-signature check. Drop the rest.

## 11. Open questions

- ~~Which chain and which token back the escrow contract first?~~
  **Resolved.** The operator picks the chain and the token at deploy time. One
  escrow instance runs per domain, so each domain sets its own pair. The token
  is a constructor parameter; the chain is wherever the operator deploys. See
  `escrow.sol`.
- Is an off-box facilitator available, or must the on-box `eth_getLogs` watcher
  be the deposit source at launch?
- Does the first customer set want ERC-7715 child keys on day one, or is one key
  per account enough for the first cut?
