# Jumpgate VALIDATOR capability — design

Status: design. Author: agent draft, 2026-08-11.
Scope: Ethereum + PulseChain proof-of-stake validators, run as a managed service on a
Jumpgate-provisioned node.

## Decision (read this first)

Run a **native validator client as a third managed service**, beside the execution and
beacon clients that Jumpgate already renders. Do **not** wrap Sedge or eth-docker as a
subprocess.

The reason is that Jumpgate already **is** the Sedge pattern. Sedge's whole value is
"generate an inspectable artifact, then get out of the way." Jumpgate's `catalog` layer
does exactly this today: `RenderUnits` and `RenderERPCUnit` emit plain, hardened systemd
units the operator can read, commit, and run without the app present. A validator client
is a small process with the same shape as a beacon client. So the honest fit is a fourth
render function on the same template, not a shell-out to a second generator that would own
its own compose file and hide it behind bash verbs.

This choice keeps one code path, one hardening template, and one backend seam
(`docker` vs `systemd`) across execution, beacon, gateway, and validator. It also lets the
same binary drive Ethereum and PulseChain, which neither Sedge nor eth-docker do — the
differentiator Jumpgate can own.

## 1. A validator is three roles, not one

A post-Merge node runs three processes:

1. **Execution client (EL)** — `reth`/`geth` on Ethereum, `go-pulse` on PulseChain.
2. **Beacon / consensus client (CL)** — Lighthouse, Prysm.
3. **Validator client (VC)** — the only process that holds keys and signs.

Jumpgate already models roles 1 and 2 as managed services (`catalog.WireConfig`,
`ExecID`/`BeaconID`, engine API on `authrpc:8551` loopback — `internal/catalog/units.go`).
The VC is the missing third role. It dials the local beacon API on loopback, exactly as the
node's own gateway dials the node. PulseChain is an Ethereum fork, so the same client
families apply — the deposit is 32,001,000 PLS instead of 32 ETH, but the operational shape
is identical.

## 2. Box-planner impact — the validator CLIENT is cheap

The heavy cost is the node, not the validator. Plan the box for the node; add almost
nothing for the VC.

| Resource | Beacon node (already planned) | Validator client (this capability) |
|---|---|---|
| RAM | ~2–4 GB | tens of MB, scales gently with key count |
| CPU | ~25–50% of 4 cores | negligible (polls once per slot, signs) |
| Disk | ~130+ GiB CL, plus a large EL dataset | the keystores + one small SQLite slashing file |
| Network | full P2P | loopback only, VC → beacon HTTP |

Two planner notes:

- **The stake is not a box resource.** 32 ETH / 32,001,000 PLS is capital the operator
  supplies at deposit time. The planner must state it as a prerequisite, not size hardware
  for it.
- **Web3Signer changes the numbers.** The remote-signer tier adds a Java process (higher
  RAM) and a PostgreSQL instance. Size for those only when the operator opts into that tier
  (section 6). The flat native VC does not need them.

The box planner should therefore treat "add validator" as a small delta on an existing node
plan, and refuse to plan a validator on a target that has no beacon client to dial.

## 3. Key custody + slashing protection

Two independent safety mechanisms. Do not confuse them.

**Slashing-protection database.** Every VC keeps a local record of every block and
attestation it has signed, and refuses to sign anything slashable. In Lighthouse this is an
SQLite file at `<datadir>/validators/slashing_protection.sqlite`, locked exclusively while
the VC runs. It is portable through the EIP-3076 interchange JSON format — the operator
exports it before moving keys between machines.

**Doppelganger protection.** A Lighthouse belt-and-suspenders check: on start the VC stays
silent for 2–3 epochs and listens for another live instance of the same key before it signs.
It is best-effort (a network fault can hide the twin) and **not interoperable** — a
Lighthouse VC using it must talk to a Lighthouse beacon node. Jumpgate should expose it as an
opt-in flag on the Lighthouse path and never rely on it as the primary guard.

**The classic footgun.** Running the same key in two places — a client's built-in VC and
Web3Signer, or a failover box — with independent slashing state gets the operator slashed.
This is the one rule the wrapper must enforce in code:

> **One key → one signer → one slashing database.** Refuse to configure a key in two
> signers. Default to VC-native SQLite for the simple case.

Make this a hard invariant at the config-validation boundary, the same place `PlanGateway`
rejects a bad chain id by rendering up front. A validator plan must fail loudly at plan time
if a keystore is already assigned to another signer in the config.

## 4. Key material is the design pin — store it like a secret, never a placeholder

Validator keystores and withdrawal credentials are true secrets with **no `${PLACEHOLDER}`
indirection**. They are unlike `ProviderKeys`, which are redactable placeholder names that
the redaction seam (`internal/server/chainlist.go`) substitutes back into egress strings.

So do **not** route validator keys through the `ProviderKeys` / `redactKeys` path. Follow the
app's existing secret discipline instead:

- Store the keystore JSON and its password verbatim on the target at mode `0600`, owned by
  `catalog.ServiceUser`, under the validator data dir (covered by the wire step's `chown -R`).
- Persist only a reference and status in `config.json` — never the key bytes, or store them
  through the same atomic-`0600` `Save` used for the rest of the config
  (`internal/config/config.go`).
- The settings/API surface reports **"configured", never the value** — the exact pattern
  `settingsResponse` uses for `ProviderKeysSet` (names, not values) and `maskSecret` uses for
  fingerprints. A validator key must never touch a screen or a log.

This mirrors the operator-secret precedent the app already sets for private material (the
0600, never-echo, report-only-configured discipline), applied to a secret that has no safe
redaction form at all.

## 5. Operator flow — keygen → deposit → run VC

Three stages. Jumpgate owns the first and third; the operator owns the middle, on-chain step.

1. **Key generation.** The operator generates validator keys with `staking-deposit-cli` or
   `lighthouse account validator`, producing keystore JSON files and a `deposit_data.json`.
   Jumpgate imports the keystores into the validator data dir. It should not invent its own
   keygen — it imports the standard artifacts, which keeps withdrawal-credential choice in the
   operator's hands.
2. **Deposit (on-chain, operator-owned).** The operator submits `deposit_data.json` to the
   deposit contract (32 ETH on Ethereum, 32,001,000 PLS on PulseChain) and waits for
   activation. Jumpgate does not hold funds or broadcast the deposit. It can show the
   parameterized deposit contract address and a "pending / active" status read from the beacon
   API, and nothing more.
3. **Run the VC.** Jumpgate renders the validator unit (or container), points it at the local
   beacon API on loopback, and starts it. The run step confirms the VC has loaded the expected
   key count and is talking to the beacon node — not merely that the process is up.

Order matters for safety: import and dry-run the VC config **before** activation, so the first
time the key ever signs is on a VC whose slashing DB is the only one that has ever held it.

## 6. Two tiers — native VC default, Web3Signer opt-in

**Tier 1 — native VC (default).** Keys live on the VC host in keystore files; slashing
protection is the local SQLite file. This is the simple, low-footprint path for a solo
operator. It is client-agnostic across Lighthouse and Prysm and needs no extra services.

**Tier 2 — Web3Signer (opt-in).** Keys live only in Web3Signer (EIP-3030 BLS remote signer);
the VC sends "sign this" over HTTP and never holds a key. Choose this for key isolation,
multi-key, or high-availability / failover, where several VCs sit behind one signer through a
load balancer without double-signing.

The cost of Tier 2 is one hard external dependency: **Web3Signer's slashing protection is
default-on and requires a PostgreSQL database the operator maintains.** The slashing DB moves
from SQLite-on-disk to Postgres. Jumpgate must provision the signer and Postgres as their own
managed services, and the one-key-one-signer invariant now means "keys are imported into the
signer XOR the VC, never both" — the exact case eth-docker's docs warn against.

Keep Tier 2 opt-in, never the default. Most operators want Tier 1.

## 7. How it attaches — the four-layer path

A validator client slots into the same four-layer path the exec/beacon/gateway services use.
Each layer has a real template to mirror.

1. **`internal/catalog`.** Add a validator role. Give clients a `"validator"` kind alongside
   the existing `"exec"` / `"beacon"` kinds (`internal/catalog/clients.go`), and add
   validator fields to `WireConfig` (or a sibling `ValidatorConfig`) — a `ValidatorID`, the
   beacon endpoint it dials, the keystore dir, the fee-recipient address, and the optional
   Web3Signer URL. Render its unit with a `RenderValidatorUnit` that reuses `renderUnit` and
   the one hardened template, exactly as `RenderERPCUnit` does
   (`internal/catalog/erpc.go:66`). The `ExecStart` is `lighthouse vc …` (or
   `lighthouse-pulse vc …` / the Prysm equivalent), by binary name per the same
   per-client-binary rule the beacon command already follows. Parameterize the network
   selector by chain id, reusing `lighthouseNetworkName` and the `--pulsechain` /
   `--pulsechain-testnet-v4` selectors already proven in `units.go` — this is what covers
   PulseChain by client family, not by fork name.

2. **`internal/setup`.** Add `PlanValidator(id, cfg, backend)` returning
   preflight / config / run `Step`s, beside `PlanGateway` (`internal/setup/gateway.go:133`).
   Validate by rendering up front, and enforce the one-key-one-signer invariant here.
   Backend choice is the same seam: `BackendSystemd` (native, low-footprint, Linux target that
   already runs the node and has `catalog.ServiceUser`) vs `BackendDocker` (cross-platform).
   The 512 MB logic carries over — native unit for the small box, container otherwise.
   Preflight must confirm a reachable beacon API on the target before it accepts the plan.

3. **`internal/ops`.** Add a validator lifecycle descriptor (start / stop / status) modeled on
   `ERPCServiceFor` (`internal/ops/lifecycle.go:218`), driven through the executor. It owns no
   dataset — its only state is the keystore dir and the slashing SQLite file — so, like
   `ERPCService`, it declares no data volume of its own beyond that dir.

4. **`internal/server`.** Route under the target namespace — the generic
   `/api/targets/{id}/services/{svc}/{action}` already exists (`internal/server/api.go`) — or a
   dedicated `/api/validators` group modeled on `gateways.go`. Key material follows section 4:
   stored at 0600, never returned, reported only as "configured". No validator string reaches
   egress without going through the never-echo rule.

Every host-touching action — provision, key import, status probe — flows through the one
`internal/executor` seam, so local vs SSH stays transparent, the same as every other service.

## 8. Outline for a verify-by-running validator ops skill

Do not write the skill from this doc. A real skill is written **after** a first end-to-end run
on a throwaway box, and records only what that run actually confirmed. This is the checklist
that run must confirm — the artifact under test is the rendered unit fed to a real client, not
the Go that produced it.

A tested setup must confirm:

1. **Render is loadable.** The rendered validator unit / compose starts a real Lighthouse (and
   Prysm) VC binary with no "flag provided but not defined" — the same failure class already
   caught for `--pulsechain-testnet-v4`. Confirm on Ethereum and on PulseChain testnet (943).
2. **The VC reaches the beacon node.** After start, the VC logs the expected validator count
   and a live beacon connection — not just an "active (running)" unit line. A green systemd
   status with a VC that never connected is the exact "reports success while broken" trap.
3. **Slashing protection exists and is exclusive.** The `slashing_protection.sqlite` file is
   created and locked while the VC runs; a second VC start on the same dir fails to acquire the
   lock rather than silently opening a second signer.
4. **The one-key-one-signer invariant is enforced at plan time.** A config that assigns one
   keystore to both the native VC and Web3Signer is rejected by `PlanValidator`, with a clear
   message, before any step runs.
5. **Keys never leak.** Grep the API responses, the logs, and any config view for the keystore
   bytes and password — none appear. The status surface shows "configured" only. Redaction is
   verified by probing egress, not by reading the redaction code.
6. **Web3Signer tier round-trips (when enabled).** Keys imported into the signer only; the VC
   holds none; a sign request over EIP-3030 returns a signature; the Postgres slashing DB
   records it. Confirm the VC refuses to also load a local keystore for the same key.
7. **Doppelganger delay behaves (Lighthouse, when enabled).** The VC stays silent for the
   expected 2–3 epochs on start before it first signs.
8. **PulseChain parity.** Steps 1–5 pass with `go-pulse` + `lighthouse-pulse` on chain 943,
   proving the one code path drives both networks.

Only the checks a run actually passes go into the skill. An unconfirmed check stays a TODO,
not a claim.

## Sources

Validator toolkit research (Sedge vs eth-docker vs native Lighthouse VC), EIP-3030,
EIP-3076, Lighthouse slashing-protection and doppelganger docs, Web3Signer slashing
protection (Postgres, default-on), and the PulseChain go-pulse + Lighthouse setup path.
Codebase seams: `internal/catalog/units.go`, `internal/catalog/erpc.go`,
`internal/setup/gateway.go`, `internal/ops/lifecycle.go`, `internal/config/config.go`,
`internal/server/api.go`.
