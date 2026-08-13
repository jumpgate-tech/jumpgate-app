# Jumpgate build plan — validator capability and metered API keys

Status: build plan. Date: 2026-08-11.
Inputs: `docs/design/metered-api-keys.md`, `docs/design/validator-capability.md`,
two security reviews, and the architecture findings folded into the validator review.

This plan folds every review finding back into the two designs. It states the
required changes, ranks the work, and lists the decisions that still need the owner.
Read section 1 for the corrected summaries. Read section 2 for the required design
changes. Read section 3 for the ranked, sequenced plan. Read section 4 for the open
decisions.

---

## 1. Corrected one-paragraph summaries

### Metered API keys and billing

Jumpgate runs one small Rust service on a 512 MB VPS beside eRPC. The service
issues `vk_` keys, meters RPC usage at a proxy, and debits a prepaid balance in
one embedded SQLite file. The proxy is the only ingress to eRPC, and eRPC binds
loopback behind a host firewall, so no client can reach `:4000` and skip the
meter. Value is anchored on-chain in a non-custodial escrow contract, metered
off-chain per request, and settled net on a cadence. The central correction from
review: the "provider draws only what it served" claim has no mechanism as
written, so the model is **"trust the operator up to the escrow ceiling"** until a
signed-usage-receipt scheme exists. The escrow contract is the highest-value
component and needs a written, audited spec with a monotonic high-water mark, a
reentrancy guard, and a withdraw-versus-draw settlement lock. Metering must price
every element of a JSON-RPC batch, meter WebSocket frames, refund only true
transport failures (a revert is served work), weight heavy methods, and normalize
the method name before the price lookup. The admin plane needs mTLS, an
append-only audit log, and synchronous key invalidation, not a single static
bearer on loopback. Secrets — the settlement signer and the admin token — need
`mlock`, no swap, no core dumps, and file-based delivery. This first cut ships as
"trust up to the ceiling"; the countersignature and dispute window are the next
slice that makes the non-custodial claim true.

### Validator capability

Jumpgate runs a native validator client (VC) as a fourth managed render function,
beside the execution client, the beacon client, and the gateway. It reuses one
hardening template and the one executor seam, drives Ethereum and PulseChain from
one code path, and keeps the VC-native SQLite slashing database for the simple
case. The central correction from review: the design's safety story leaned on
plan-time config validation and on one shared box with one service user, and both
break under an adversary. The VC must run as its own dedicated OS user that no RPC
process can read, ideally on a box with no public listener; co-location with
public RPC becomes a flagged, opt-in exception, not a silent default. Jumpgate
must **validate what it imports** — cross-check `deposit_data.json` pubkeys,
network, and amount, surface the withdrawal credential for confirmation, hardcode
the deposit contract per chain id, and validate the fee recipient — rather than
pass artifacts through untouched. The one-key-one-signer invariant stays necessary
but not sufficient, because it cannot see a second box, a snapshot restore, or a
migration; the plan adds a stop-and-confirm-old-signer-dead migration gate and a
backup-restore warning. The VC key-manager HTTP API and, in Tier 2, Web3Signer
must bind loopback with mutual TLS and never sit behind the public proxy. Tier 2's
Postgres slashing database is a must-not-roll-back component for the whole key set.

---

## 2. Required design changes (folded findings)

Each row is a change the design must adopt. "Gate" means the first cut must not
ship without it. "Slice 2" means it is required for the non-custodial or Tier-2
claim, but not for the first honest cut.

### 2A. Metered API keys — required changes

| # | Finding | Required change | When |
|---|---|---|---|
| S1 | No mechanism ties the draw to work served | Ship a signed-usage-receipt scheme: the proxy returns a running, monotonic, signed counter to the client; escrow pays only the highest customer-countersigned total. Until then, state plainly the model is "trust the operator up to the ceiling" and make the ceiling the customer's risk decision. | Honest label at gate; receipts at slice 2 |
| S2 | Escrow contract is unspecified | Write and audit the escrow contract. Store a per-account monotonic high-water mark; revert on a non-increasing total. Follow checks-effects-interactions, add a reentrancy guard, use pull-payment for draws. Gate customer withdrawal behind a settlement lock or timelock so an in-flight draw cannot be stranded. Prefer a USD stablecoin and a non-callback token standard. | Gate (blocks any on-chain draw) |
| S3 | Direct eRPC bypass gives free usage | Bind eRPC to `127.0.0.1` only. Add a host firewall rule that blocks `:4000` from every source except the proxy user. On a multi-user box, restrict loopback access too. Add a startup self-check that refuses to boot if eRPC answers on a non-loopback address. | Gate |
| S4 | Deposit path can mint credits | Verify the log came from the exact escrow address and event signature, and that funds landed. Credit only after N confirmations; reverse credits on a reorg below finality. Authenticate the webhook with a verified facilitator signature over the full payload; fail closed; keep it off the public interface. Pin the deposit token to a USD stablecoin. | Gate |
| S5 | Batch and WebSocket escape metering | Parse batch bodies; price and debit every element; cap batch size and body size. Define WS metering: debit per outbound notification or per subscription-second; cap concurrent subscriptions and connections per key; cap total WS memory. | Gate |
| S6 | Refund-on-failure pays for served work | Refund only on proxy or transport failures where no upstream answer was produced. Bill any JSON-RPC response that carries a result or an application error (a revert is served work). Do not refund on client timeouts after the upstream was reached. Make debit and refund crash-consistent. | Gate |
| S7 | Flat pricing underprices heavy methods | Weight expensive methods. Surcharge or reject `eth_getLogs` by block range and result size. Cap or block trace/debug namespaces unless the key allows them, at a high price. Reject a zero or negative `credits_per_request`. | Gate |
| S8 | Method-name normalization bypass | Normalize the method name the way eRPC dispatches it before the price lookup. Reject unknown methods on paid keys, or set the default high, not low. | Gate |
| S9 | Admin plane weak | Write an append-only audit log for every admin mutation, with actor and timestamp. Put high-risk mutations (pricing, `credit_exempt`, rebinding) behind a stronger control than the read/issue bearer. Require mTLS even on the same box. Update the in-memory key map synchronously on revoke or rotate. | Gate |
| S10 | Secret paging leaks signer and token | `mlock` the signer key and admin token; mark them `MADV_DONTDUMP`; disable core dumps. Disable swap, or use encrypted swap. Deliver the admin token by file or `systemd LoadCredential`, never a plain `Environment=`. Set `-wal`, `-shm`, `-journal`, and the directory to `0600`, owned by the service user. State that provider disk encryption does not protect against the provider or operator. | Gate |
| S11 | Key leakage on the wire and in logs | Terminate TLS on the data plane; refuse plaintext. Accept the key only in the header; reject keys in the URL; redact key-shaped tokens from all logs. | Gate |
| S12 | Unsalted SHA-256 | Generate `vk_` keys with at least 128 bits of CSPRNG entropy. Use HMAC-SHA256 with a server-side pepper stored beside the signer secret, never in the DB. | Gate |
| S13 | Binding signature replay | Use EIP-712 typed data with an explicit domain (chain id, contract), the key id, the account, a nonce, and an expiry. Reject a reused nonce or an expired message. | Gate |
| S14 | Exempt keys skip the escrow floor | Keep a hard rate and concurrency cap on exempt keys. Scope each exempt key to origins or IPs and a short expiry. Keep the count near zero. | Gate |
| S15 | Box-level DoS breaks metering | Cap body size, batch size, concurrent connections, and WS subscriptions. Fail closed on a debit write error. Add a watchdog and back-pressure so a write stall degrades to rejects, not to unmetered forwards or an OOM kill. | Gate |
| S16 | Reserve model not in SQL | Reconcile the prose reserve/capture model with the SQL. The shown `UPDATE` touches `credits_remaining` only and never `credits_reserved`. Decide one model and make the SQL match it before implementation. | Gate |

### 2B. Validator capability — required changes

| # | Finding | Required change | When |
|---|---|---|---|
| C1 | Public-RPC compromise reads the keys | Give the VC a dedicated OS user, not `catalog.ServiceUser`. No RPC, gateway, or beacon process may read the validator data dir; verify with a negative test. Do not store the keystore password on the same box by default; prompt, use `LoadCredential`, or a path the RPC user cannot read. Refuse, or loudly warn at plan time, when the target also serves public RPC. | Gate |
| C2 | Web3Signer trust boundary unspecified | The VC-to-Web3Signer channel must use mutual TLS; the signer must bind loopback or a private segment only, never behind the public proxy. Pin one shared Postgres for slashing protection; assert slashing protection stays default-on; reject a config with two private DBs or a disabled guard. After import, destroy or quarantine the source keystore. Treat the Postgres connection string as a managed 0600 secret. | Gate for Tier 2 |
| C3 | No validation of deposit data or credential | Before deposit, cross-check `deposit_data.json`: pubkeys match the imported keystores; `fork_version`/network matches the chain id; amount matches the chain. Fail loudly on any mismatch. Surface the withdrawal credential (type `0x01`, decoded address) read-only for operator confirmation. Hardcode the deposit contract address as a per-chain-id constant, never config-supplied. | Gate |
| H1 | Invariant is plan-time only | Add a migration and decommission protocol: stop the source VC, confirm it is dead, export the EIP-3076 interchange, import on the target, then start. Make "old signer confirmed stopped" an explicit gate. Warn that snapshot or backup restore of a validator box is unsafe. Recommend Doppelganger by default on the Lighthouse path; state plainly that Prysm has no equivalent. Say in operator text that the plan-time check cannot see other hosts. | Gate |
| H2 | VC key-manager API unaddressed | Disable the key-manager API unless needed. When enabled, bind it to loopback with its auth token; assert at render time it is never reachable through the public proxy. Add an off-box reachability probe to the verify checklist. | Gate |
| H3 | Fee recipient never validated | Validate the fee recipient at plan time: present, non-zero, checksum-valid, operator-confirmed. Fail the plan if it is absent. Keep the fee-recipient and withdrawal-credential fields clearly separated in every surface. | Gate |
| M1 | Public-RPC DoS starves the VC | Reserve CPU and IO for the validator and beacon with `CPUWeight`, `IOWeight`, `MemoryLow`. Add a planner note that public-RPC load competes with validator liveness; prefer a dedicated box for non-trivial stake. | Slice 2 |
| M2 | Postgres is a single point | Document Postgres as a critical, must-not-roll-back component. No snapshot restore of the slashing DB; recover through the interchange format only. Require durable, transactionally consistent storage. | Gate for Tier 2 |

---

## 3. Ranked, sequenced implementation plan

The work splits into three tracks by nature: **on-chain contract work**,
**thin Rust/Go service work**, and **landing and planner work** that ships now.
The tracks run in parallel where dependencies allow. Ship the validator planner
slice and the validator native path first, because they carry no contract
dependency and unblock revenue on the node side. Ship the metering proxy behind a
"trust up to the ceiling" label before the escrow contract is audited, because the
proxy is useful with an off-chain or stablecoin-webhook deposit source.

### Track A — On-chain contract work (longest lead; start first, gate on audit)

This is the critical path for the non-custodial claim. Start it on day one because
the audit is the long pole.

- **A1. Escrow contract spec.** Write the contract: prepaid deposit, per-account
  monotonic high-water mark, `pay(signedTotal - alreadyDrawn)` with a strict
  `signedTotal > alreadyDrawn` check, reentrancy guard, checks-effects-interactions,
  pull-payment draws, and a withdrawal settlement lock or timelock that gives an
  in-flight provider draw priority. Pin a USD stablecoin and a non-callback token
  standard. Covers S2, S4. Dependency: decision D1, D2 (chain and token).
- **A2. Deposit verification rules.** Define N-confirmation depth per chain, the
  reorg-reversal path, and the exact escrow address and event signature the
  watcher trusts. Covers S4. Dependency: A1.
- **A3. Signed-usage-receipt scheme (slice 2).** Define the running monotonic
  signed counter, the client countersignature cadence, and the on-chain check that
  pays only the highest countersigned total. Add a challenge or dispute window
  before a draw finalizes. Covers S1. Dependency: A1. This is what turns the honest
  label into a true non-custodial guarantee.
- **A4. Audit.** External audit of A1–A3 before any mainnet draw. Gate.

### Track B — Thin Rust service (the metering proxy and settlement worker)

Ships in two slices. Slice B1 is a useful, honest product before the contract
lands. Slice B2 makes the non-custodial claim true.

**Slice B1 — honest metered proxy (label: "trust the operator up to the ceiling")**

- **B1.1. Proxy skeleton and eRPC isolation.** `axum` + `tokio` + `rusqlite`,
  single `musl` binary. Bind eRPC to loopback; add the host firewall rule; add the
  startup self-check that refuses to boot if eRPC answers off-loopback. Covers S3.
- **B1.2. Auth and key store.** HMAC-SHA256 with a server-side pepper; 128-bit
  CSPRNG keys; header-only key; fail closed on unknown; `vk_demo` public tier.
  Covers S11 (header-only), S12.
- **B1.3. Pricing and normalization.** In-memory price map; normalize the method
  name the way eRPC dispatches before lookup; reject unknown methods on paid keys;
  reject zero or negative price; weight heavy methods; surcharge or reject wide
  `eth_getLogs`; cap or block trace/debug. Covers S7, S8.
- **B1.4. Debit path.** Reconcile the reserve/capture model with the SQL first
  (S16), then implement one atomic conditional `UPDATE`. Fail closed on a write
  error. Refund only on transport failure with no upstream answer; bill reverts.
  Make debit and refund crash-consistent. Covers S6, S15, S16.
- **B1.5. Batch and WebSocket metering.** Parse batch bodies; price every element;
  cap batch size, body size, connections, and WS subscriptions; debit per WS
  notification; cap WS memory. Covers S5, S15.
- **B1.6. Admin plane.** mTLS; append-only audit log; high-risk mutations behind a
  stronger control; synchronous key-map invalidation on revoke/rotate. Covers S9.
- **B1.7. Secret discipline.** `mlock` the signer and admin token; `MADV_DONTDUMP`;
  disable core dumps; disable or encrypt swap; deliver the admin token by
  `LoadCredential`; `0600` on `-wal`/`-shm`/`-journal` and the directory. Covers S10.
- **B1.8. Exempt-key and rate caps.** Hard rate and concurrency cap even on exempt
  keys; scope to origins/IPs and a short expiry. Covers S14.
- **B1.9. Deposit ingress (stablecoin).** Webhook authenticated by a verified
  facilitator signature over the full payload; fail closed; off the public
  interface; idempotent on `tx_hash`; credit only after N confirmations; reverse on
  reorg. Or the on-box `eth_getLogs` watcher with the same checks. Covers S4.
  Dependency: A2, decision D3 (facilitator).
- **B1.10. EIP-712 binding.** Typed data with domain, key id, account, nonce, and
  expiry; reject reused nonce or expired message. Covers S13.

**Slice B2 — true non-custodial**

- **B2.1. Emit per-request usage the customer can audit.** Covers S1.
- **B2.2. Return the running signed counter to the client; countersign cadence.**
  Covers S1. Dependency: A3.
- **B2.3. Draw worker.** Redeem on threshold or timer, bounded by the highest
  countersigned total, through the dispute window. Dependency: A3, A4.

### Track C — Landing and planner (ships immediately; no contract dependency)

This is the fastest visible value. None of it waits on a chain.

- **C1. Add a "Validator" service to the box planner.** Treat "add validator" as a
  small delta on an existing node plan. Show the tens-of-MB VC cost, the keystore
  and slashing-file disk, and loopback-only network. Refuse to plan a validator on
  a target with no beacon to dial. Ships now.
- **C2. Planner co-location guard.** Flag a validator on a public-RPC box as an
  opt-in exception, not a silent default. Add the planner note that public-RPC load
  competes with validator liveness. Covers C1, M1. Ships now.
- **C3. Fee-recipient and withdrawal-credential fields on the plan.** Validate the
  fee recipient at plan time (present, non-zero, checksum, confirmed). Keep the two
  fields visually separate. Covers H3. Ships now.
- **C4. Stake-as-prerequisite copy and deposit-contract constant.** State the stake
  as a prerequisite, not hardware. Hardcode the deposit contract per chain id.
  Covers C3 (constant). Ships now.

### Track D — Validator native path (thin Go render, in the existing tree)

The validator is Go in the existing `internal/*` tree, not a new service. It
follows the four-layer path in the design.

**Slice D1 — Tier 1 native VC (default)**

- **D1.1. `internal/catalog`.** Add a `"validator"` client kind and a
  `ValidatorConfig` (ValidatorID, beacon endpoint, keystore dir, fee recipient,
  optional Web3Signer URL). Add `RenderValidatorUnit` reusing the one hardened
  template. Parameterize the network by chain id, reusing the PulseChain selectors.
- **D1.2. Dedicated OS user.** Render the VC under its own user, not
  `catalog.ServiceUser`. No RPC process may read the validator data dir. Covers C1.
- **D1.3. `internal/setup` `PlanValidator`.** Preflight/config/run steps. Validate
  by rendering up front. Enforce one-key-one-signer at plan time. Confirm a
  reachable beacon API. Reserve CPU/IO for VC and beacon. Covers M1.
- **D1.4. Deposit-data validation.** Cross-check `deposit_data.json` pubkeys,
  network/`fork_version`, and amount against the target. Surface the withdrawal
  credential read-only for confirmation. Fail loudly on mismatch. Covers C3.
- **D1.5. Key-manager API lockdown.** Disable unless needed; loopback plus auth
  token when enabled; assert it is never reachable through the public proxy.
  Covers H2.
- **D1.6. Password custody.** Do not store the keystore password beside the
  keystore by default. Prompt, `LoadCredential`, or a path the RPC user cannot
  read. Covers C1.
- **D1.7. `internal/ops` lifecycle.** Start/stop/status modeled on `ERPCServiceFor`;
  no data volume beyond the keystore and slashing dir.
- **D1.8. `internal/server` routes.** Under the target namespace. Key material
  0600, never returned, reported only as "configured". Covers section 4 discipline.
- **D1.9. Migration and backup safety.** Add the stop-and-confirm-old-signer-dead
  migration gate. Warn that snapshot or backup restore of a validator box is
  unsafe; document the EIP-3076 re-import path. Recommend Doppelganger by default
  on Lighthouse; state Prysm has no equivalent. Covers H1.
- **D1.10. Verify-by-running on a throwaway box.** Run the section-8 checklist plus
  the off-box API probe (H2) and a same-key-on-two-boxes negative case (H1).
  Confirm Ethereum and PulseChain testnet 943. Only confirmed checks become the
  skill.

**Slice D2 — Tier 2 Web3Signer (opt-in)**

- **D2.1. Provision Web3Signer and Postgres as managed services.** Bind the signer
  loopback or a private segment; never behind the public proxy. Covers C2.
- **D2.2. Mutual TLS on the VC-to-signer channel.** Rendered-config invariant, not
  advice. Covers C2.
- **D2.3. One shared Postgres; slashing-protection default-on.** Reject two private
  DBs or a disabled guard at plan time. Treat the connection string as a 0600
  secret. Document Postgres as must-not-roll-back. Covers C2, M2.
- **D2.4. Destroy or quarantine the source keystore after import.** Assert the VC
  refuses to also load a local keystore for the same key. Covers C2.

### Sequencing and dependencies (summary)

1. **Day one, in parallel:** start A1 (escrow spec) and A4 (book the audit slot);
   ship C1–C4 (planner); start D1 (native validator).
2. **After A1/A2:** build B1.9 (deposit ingress). Build the rest of B1 in parallel
   with the contract, because the proxy does not depend on the contract to meter.
3. **Gate for B1 launch:** all Track-B gate rows plus S3 isolation, shipped under
   the honest "trust up to the ceiling" label, with a stablecoin deposit source.
4. **After D1 verify-by-running passes:** publish the validator ops skill; then
   start D2 (Tier 2) only when an operator asks for it.
5. **After A3/A4 audit passes:** ship B2 (signed receipts, dispute window, draw
   worker). This is the milestone that removes the "trust the operator" label.

---

## 4. Open decisions for the owner

Ranked by how much they block the plan.

1. **D1 — escrow chain.** Which chain backs the escrow contract first? Blocks A1.
2. **D2 — escrow token.** Which USD stablecoin, and confirm a non-callback token
   standard? Review requires a stablecoin to close the oracle-manipulation lever
   (S4) and the reentrancy surface (S2). Blocks A1.
3. **D3 — deposit source at launch.** An off-box facilitator, or the on-box
   `eth_getLogs` watcher? Both need the S4 checks; the choice sets B1.9. 
4. **D4 — honest label acceptance.** Do we ship B1 as "trust the operator up to the
   ceiling" before the signed-receipt scheme (A3/B2) lands? If not, the metering
   product waits on the full audit. This is a go-to-market call, not a technical one.
5. **D5 — child keys on day one.** ERC-7715 or ERC-4337 scoped child keys in the
   first cut, or one key per account? Affects the binding and escrow scope.
6. **D6 — validator co-location policy.** Is a validator on a public-RPC box a
   supported opt-in exception, or do we require a separate validator box for any
   non-trivial stake? The review's strongest fix is a separate box. Sets C2.
7. **D7 — Tier-2 timing.** Do we build Web3Signer (D2) now, or defer until an
   operator with multi-key or failover needs it? Most operators want Tier 1.
8. **D8 — external price read.** If the deposit token is ever not a stablecoin, who
   owns the single rate read, and how is it protected from oracle manipulation?
   Prefer a stablecoin and remove this question.
