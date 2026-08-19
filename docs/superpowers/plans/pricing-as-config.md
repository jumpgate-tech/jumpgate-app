# Method pricing belongs in a config file, not in the binary

Date: 2026-08-18. Status: captured, not started. Raised by the owner.

## The problem

`pricing::DEFAULT_PRICES` is a hardcoded array in the Rust binary, and
`Store::open()` calls `seed_default_pricing()` on EVERY open. Three things follow,
and all three are wrong:

1. **Changing a price needs a recompile.** An operator cannot ship or review a
   price list. The only runtime path is `PUT /admin/pricing/{method}`, one method
   at a time, against a running service.
2. **Opening the store writes to it.** Seeding is a side effect on what should be
   a read path. `billing keys list` creates and populates a database.
3. **It manufactures databases nobody asked for.** This is not hypothetical: a
   mistyped `billing init --db foo` created a fully-seeded store named `--db` at
   the repo root, because opening a store always produces a complete one. A
   design where billing data appears only when billing is switched on would not
   have produced that file at all.

## What it should be

Prices ship as a **config file** that prefills the store only if the operator
turns billing on.

- A default price list ships as data (TOML or JSON) beside the binary, readable
  and reviewable without a Rust toolchain.
- `Store::open()` creates the schema and NOTHING else. No seeding, no writes on a
  read path.
- Seeding happens on an explicit step — `billing init`, or the first start with
  billing enabled — and reads the config file. `INSERT OR IGNORE` still protects
  an operator's edits, as it does today.
- The relay already reads prices through `GET /internal/price`, so nothing on the
  hot path changes.

## Worth fixing at the same time

`billing init` takes a POSITIONAL path while `billing serve` takes `--db`. That
inconsistency is what created the stray database. Make both take `--db`.

## Scope

Small and contained: `pricing.rs` (load from a file rather than a const),
`store.rs` (stop seeding in `open`), `main.rs` (seed on init, unify the flag),
plus the tests that assume a freshly opened store is already priced.
