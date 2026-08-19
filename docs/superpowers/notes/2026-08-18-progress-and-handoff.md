# Progress and handoff — 2026-08-18

A working ledger for picking the thread up on another machine.

**Slice status lives in [`../plans/self-hosted-metered-rpc.md`](../plans/self-hosted-metered-rpc.md), not here.**
That plan carries the status table, the slice B carry-over, the "still not done"
list and the acceptance record. This note deliberately does not restate them —
two places tracking the same status is how the two drift apart. What follows is
only what that plan does not already say.

## One conflict to resolve

This ledger arrived carrying acceptance numbers that disagree with the ones in
the plan. They look like two different runs rather than a correction, so both
are recorded here and neither has been overwritten. Somebody who knows which
run is current should reconcile them.

| | plan doc | this ledger |
|---|---|---|
| credit test | 100 credits at 5/call → 20 calls, then 402 | 40 credits at 5/call → 8 calls, then 402 |
| settle loop | 980 remaining, 0 reserved (1000 funded − 20 spent) | 0 remaining, 0 reserved |
| footprint | relay ~15 MB RSS, ~341 MB free | billing 4.5 MB + jumpgate 15.9 MB, 336 MB free |

## Traps found the hard way

Do not re-derive these.

- **`billing init` takes a POSITIONAL path; `billing serve` takes `--db`.** So
  `billing init --db foo.db` silently creates a database named `--db`. This cost
  real debugging time. Confirmed in `services/billing/src/main.rs` (`cmd_init`
  reads `args.next()`). Making `init` accept `--db` is a good small task.

- **Key records are cached in memory by the key manager.** Editing `project_key`
  with `sqlite3` behind a RUNNING service does NOT take effect. Bind accounts
  through `PATCH /admin/keys/{id}`, not with SQL, unless the service is stopped.

- **Unix socket paths are capped near 104 bytes on macOS.** Go's `t.TempDir()`
  under a long test name exceeds it, so socket tests use a short dir of their own.

- **`DOCKER_DEFAULT_PLATFORM=linux/amd64` is exported on the dev Mac.** Emulated
  amd64 Go binaries CRASH under QEMU (real Caddy panicked in the runtime). Pass
  `--platform linux/arm64` explicitly to run Go-based images locally.

- **Cross-building the Rust store for the box needs Docker** (no musl target on
  the Mac). ~6 minutes under QEMU. The output dir is gitignored.

- **Docker Desktop does not share `/tmp` or `/private/tmp`.** Mount from under
  `/Users`.

- **Outbound TCP :22 is blocked from the dev network**, so the test box listens
  on a non-standard ssh port. Its address, alias and port are operator
  environment facts and must never be written into a tracked file.

## How to run it again

Local, no metering:

```sh
cd services/billing && cargo build && cd ../..
export JUMPGATE_KEY_PEPPER=... JUMPGATE_ADMIN_TOKEN=... JUMPGATE_RELAY_TOKEN=...
./services/billing/target/debug/billing init /tmp/jg/b.db
./services/billing/target/debug/billing serve --db /tmp/jg/b.db --socket /tmp/jg/b.sock &
go run ./cmd/valve-node-app --no-open \
    --relay-bind 127.0.0.1:8890 --billing-socket /tmp/jg/b.sock \
    --erpc-url http://127.0.0.1:4000
```

Add `--meter` to charge credits. Mint a key:

```sh
curl --unix-socket /tmp/jg/b.sock -X POST http://x/admin/keys \
  -H "Authorization: Bearer $JUMPGATE_ADMIN_TOKEN" \
  -H 'Content-Type: application/json' -d '{"label":"dev"}'
```

Bind it to a funded account (the `account` row still needs seeding by hand —
that is the gating gap, tracked as item 4 in the plan):

```sh
curl --unix-socket /tmp/jg/b.sock -X PATCH http://x/admin/keys/<id> \
  -H "Authorization: Bearer $JUMPGATE_ADMIN_TOKEN" \
  -H 'Content-Type: application/json' -d '{"account_address":"0x..."}'
```

The acceptance box keeps its binaries and scripts under `/opt` on the test VPS.
Its address, ssh alias and port are in the operator's own environment notes,
deliberately not in this repository. Processes are stopped; re-run them there.

Gates CI enforces:

```sh
go build ./... && go test ./...
cd services/billing && cargo test && cargo clippy --all-targets
cd cmd/valve-node-app/web && npx vite build && npx vitest run
git diff --exit-code cmd/valve-node-app/web/dist     # dist MUST be committed
```

## Open questions for the owner

- **"the ladder, the curve, the depth — drop the 'the'."** The ledger recorded
  these as absent from both repos. That is not accurate for this one: `the
  ladder` occurs 17 times as a real named concept (the diagnostics ladder that
  stops at its first failure — `internal/ops/diagnose_gaps_test.go`,
  `internal/server/diag.go`, `README.md`, the Diagnostics React screen), and
  `the depth` occurs 4 times in `internal/relay/route.go`, but only ever about
  path-segment depth, which looks unrelated. `the curve` occurs nowhere.
  Crucially, none of them appear inside a user-visible string literal — every
  occurrence is a comment, a test name, or prose.

  The landing repo (`jumpgate-tech/landing`) was then checked too: none of the
  three appear there either, in any casing, and its section headings are already
  bare single words — `Route`, `Tunnel`, `Yours`, `Connect`, `Host` — with no
  "the" left to drop. So the instruction cannot be applied literally to either
  repo as it stands. Either it was already carried out on the landing copy, or
  it refers to a surface outside both repos. Nothing was changed; the owner
  should say which surface was meant.

- Should `billing init` be changed to accept `--db`, matching `serve`? Small fix
  with a real chance of biting somebody again.

- The reth "minimal" sync tier is captured but NOT started, and its size figure
  is unverified. See [`../plans/minimal-sync-tier.md`](../plans/minimal-sync-tier.md).

## Method notes

Honesty about how this was built.

- The Go and Rust work was strict test-first throughout. Several tests caught
  real defects while red: the credit concurrency test accepted 1280 spends
  against 500 funded credits before the lease was written properly.

- The React `KeysSection` was written IMPLEMENTATION-FIRST and tested after. To
  avoid shipping tests that merely describe existing behaviour, they were
  mutation-checked and do fail when the code is broken.

- One Caddy redaction check initially passed VACUOUSLY — it reported "no key in
  the log" when no log lines existed at all. The grep pattern was wrong, not the
  filter. It was re-run and produced real evidence. Watch for this shape.

- Commit `d4c0419` was amended: a `git add -A` had swept a concurrent agent's
  in-progress Rust edits into a commit whose message did not mention them.
