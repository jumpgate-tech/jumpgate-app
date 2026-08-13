# Jumpgate billing service — operator admin surface and default pricing

Status: build plan. Date: 2026-08-12.
Scope: the operator-facing half of the metering proxy — key management and
method pricing. It zooms into slices B1.2, B1.3, and B1.6 of `BUILD-PLAN.md`.
Read `metered-api-keys.md` for the whole service and `BUILD-PLAN.md` for the
ranked plan and the security findings this slice must satisfy.

## 1. What the operator must be able to do

One operator runs one Jumpgate box. That operator needs four powers:

1. Create API keys, with no cap on the count.
2. Revoke a key at once.
3. Rotate a key without an outage.
4. Set the price of each RPC method, and start from sensible defaults.

The customer never touches this surface. It is the operator's control panel for
the box.

## 2. Where the surface lives

Two processes, one seam:

- **The Rust billing service** owns the truth. It holds the admin API, the
  SQLite store, and the in-memory key map. It enforces every rule.
- **The Jumpgate app (Go, existing tray/desktop tree)** is the admin client. The
  operator manages keys and prices from the app they already run. The app calls
  the admin API over loopback.

The admin API binds `127.0.0.1` with mutual TLS and a bearer token delivered by
`systemd LoadCredential`, never a plain environment variable (finding S9, S10).
A `jumpgate keys …` CLI hits the same API, so scripts and the UI share one path.

```
operator ──> Jumpgate app (Go)  ──loopback + mTLS──>  billing service (Rust)
                 or  jumpgate keys CLI                      │
                                                            v
                                                     SQLite  +  in-memory key map
```

## 3. Key lifecycle

### 3.1 Create (unlimited)

A key is one row. There is no count limit in the schema or the API. The service
generates the value from 128 bits of CSPRNG entropy, formats it as
`vk_<base58>`, and returns the raw value **once**. It stores only
`HMAC-SHA256(pepper, key)` — never the raw key, never a plain SHA-256 (findings
S12). The pepper lives beside the signer secret, never in the database.

Honest bound on "unlimited": each key sits in the in-memory map as a hash plus a
small config struct, about 200–300 bytes. Ten thousand keys cost 2–3 MB;
a hundred thousand cost 20–30 MB. On the 512 MB box the map is the real ceiling,
not a hard limit in code. The operator can make as many keys as RAM allows, and
the service logs the map size so the operator can see the cost. For a very large
fleet, the next step is a lazy cache instead of a full preload — deferred until a
real workload needs it.

### 3.2 Per-key config

Each key carries a reduced `KeyConfig` (the thin form of the monorepo shape):

- `origins` — allowed HTTP origins, or any.
- `methods` — an allow list or a block list of RPC methods.
- `networks` — allowed chains, or any the box serves.
- `ip_rules` — allow or deny by CIDR.
- `rate` — per-second and per-day limits, or **unlimited** (no throttle).
  Operator-created keys default to unlimited rate. The prepaid balance is then
  the only limit. The public `vk_demo` tier keeps its per-IP limit (finding S14).
- `credit_exempt` — skip the balance debit. Off by default. An exempt key still
  keeps a hard rate and concurrency cap and a short expiry (finding S14).
- `allow_trace` — permit the trace and debug namespaces. Off by default.
- `expires_at` — optional expiry.

Unlimited rate is not unlimited cost. A billed key with no throttle still stops
at its prepaid balance, so the escrow ceiling bounds it. `credit_exempt` is the
separate, dangerous flag — a free key — so it always keeps a hard rate and
concurrency cap and a short expiry (finding S14). Box-level backpressure applies
to every key regardless: a global concurrency cap, body and batch size caps, and
a WebSocket memory cap. One unlimited key cannot then starve or OOM the shared
box (finding S15).

### 3.3 Revoke and rotate

- **Revoke** sets `disabled_at` and evicts the key from the in-memory map in the
  same call. The very next request on that key fails closed. Revocation is
  synchronous — there is no propagation delay and no cache to wait on (S9).
- **Rotate** issues a new value and keeps the old value valid for a short TTL,
  then soft-deletes it. This lets a customer roll a key with no outage.

Every create, revoke, rotate, and price change writes an append-only audit row
with the actor and the timestamp (finding S9).

## 4. Method pricing

### 4.1 The model

Each method has a `credits_per_request`. The proxy normalizes the method name
the way eRPC dispatches it, then looks up the price with this precedence
(matching the monorepo): exact method row, then a per-chain default, then the
global default. The service rejects a zero or negative price (finding S7). It
ships with the default table below, so the box meters correctly on first boot
with no configuration.

Credits are relative weights. The deposit rail sets the credit-to-USD rate, so
the operator prices in credits and does not touch dollars here. Keep the
monorepo peg of 1 USD = 1e9 credits at the deposit boundary.

### 4.2 Default price table

The weights follow the cost each method puts on the upstream. Cheap cached reads
sit near the floor. Standard reads sit at the base unit of 20. Wide scans and
tracing cost far more, because they cost the upstream far more.

| Tier | Methods | Default credits |
|---|---|---|
| Trivial (cached / static) | `eth_chainId`, `net_version`, `web3_clientVersion`, `eth_syncing` | 1 |
| Light read | `eth_blockNumber`, `eth_gasPrice`, `eth_getBalance`, `eth_getTransactionCount`, `eth_getCode`, `eth_feeHistory` | 5 |
| Standard (base unit) | `eth_call`, `eth_estimateGas`, `eth_getBlockByNumber`, `eth_getBlockByHash`, `eth_getTransactionByHash`, `eth_getTransactionReceipt`, `eth_getStorageAt`, `eth_getBlockReceipts` | 20 |
| Write | `eth_sendRawTransaction` | 20 |
| Filters / subscriptions | `eth_newFilter`, `eth_getFilterChanges`, `eth_getFilterLogs`; `eth_subscribe` per notification | 20 (filter), 5 (per WS notification) |
| Heavy scan | `eth_getLogs` | 75 base, plus 1 credit per 1,000 blocks of range; reject a range wider than the cap | 
| Trace / debug (off by default) | `debug_traceTransaction`, `debug_traceCall`, `debug_traceBlockByNumber`, `trace_transaction`, `trace_block`, `trace_filter`, `trace_replayTransaction` | 500, and blocked unless the key sets `allow_trace` |
| Unknown method | any name not in the table | Paid key: reject. Public tier: charge the global default, set high (200) |

Two guards ride with the table:

- **`eth_getLogs` range surcharge.** Price base plus range, and reject a scan
  wider than the configured cap. A wide log scan is the classic way to turn one
  cheap request into heavy upstream work (finding S7).
- **Batch pricing.** A JSON-RPC batch costs the sum of its element prices. Cap
  the batch size and the body size, and price every element — a batch must not
  slip through as one request (finding S5).

### 4.3 Operator overrides

The operator edits any row from the app or the CLI: raise a method, lower it,
block a namespace, or set a per-chain default. A change writes to SQLite, updates
the in-memory price map at once, and writes an audit row. The defaults are a
starting point the operator can reshape per box.

## 5. Admin API (the surface the app and CLI call)

- `POST   /admin/keys` — create a key. Return the raw `vk_` value once. Accept a
  `KeyConfig`. No count limit.
- `GET    /admin/keys` — list keys (hash id, label, config, usage), never the raw value.
- `PATCH  /admin/keys/:id` — change the config (origins, methods, rate, expiry, flags).
- `POST   /admin/keys/:id/rotate` — issue a new value; old value valid for a short TTL.
- `DELETE /admin/keys/:id` — revoke (soft-delete + synchronous map eviction).
- `GET    /admin/keys/:id/usage` — per-key rolling usage from SQLite.
- `GET    /admin/pricing` — read the price table.
- `PUT    /admin/pricing/:method` — set a method price (per-chain optional). Reject ≤ 0.
- `GET    /admin/audit` — read the append-only audit log.

Every route needs mTLS and the admin bearer. Every mutation writes an audit row.
High-risk mutations (pricing, `credit_exempt`, rebinding) sit behind a stronger
control than a read (finding S9).

## 6. Build order for this slice

1. **Schema + store.** `project_key`, `key_constraint`, `method_pricing`,
   `account`, `audit_log` tables in one SQLite file, WAL mode, `0600`.
2. **Key manager.** CSPRNG generate, HMAC hash with pepper, create/list/rotate/
   revoke, in-memory map load at boot and synchronous update on change.
3. **Pricing map.** Load the default table into memory; normalize method names;
   precedence lookup; reject ≤ 0; the `eth_getLogs` range surcharge.
4. **Admin API.** The routes in section 5, mTLS, bearer, append-only audit.
5. **Jumpgate app UI + CLI.** Key list, create, revoke, rotate; a pricing editor
   with the defaults visible; both over the loopback admin API.
6. **Wire into the hot path.** The proxy resolves the key from the map, prices
   from the map, then debits (slice B1.4). Metering and debit are the next doc.

Steps 1–5 stand alone. They give the operator a working control panel before the
debit path and the escrow are wired. The hot path in step 6 depends on the debit
slice (B1.4) and the deposit source decision (D3).

## 7. Decisions still needed

- **D3 — deposit source at launch.** Off-box facilitator, or the on-box
  `eth_getLogs` watcher? This sets how a balance gets funded, not how keys are
  managed, so this slice does not wait on it.
- **Very large key fleets.** Is a preload map enough for the first cut, or does a
  target workload need a lazy cache from day one? Default to the preload map.
- **Public tier.** Keep the single published `vk_demo` key on a per-IP rate limit,
  separate from real keys (finding S14). Confirm the public tier stays on.
