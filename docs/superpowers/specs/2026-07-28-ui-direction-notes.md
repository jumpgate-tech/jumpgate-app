# UI direction: one machine, one page, and health before measurement

**Date:** 2026-07-28
**Status:** direction agreed, not yet specced for implementation

Notes from a design pass on "the UI is shit". Captured because the mockups live in
gitignored scratch and the reasoning is worth more than the pictures.

## The diagnosis

Nine screens, one per noun: `#/targets`, `#/rpc`, `#/analytics/<gid>`, `#/setup/<id>`,
`#/dash/<id>`, `#/logs/<id>`, `#/services/<id>`, `#/settings`. That is an org chart of
the data model. Answering *"is anything wrong?"* means visiting five of them and
assembling the verdict by hand.

Four of those routes are **per machine**, so the nav grows with the fleet. Backwards: the
fleet should grow inside a screen, not beside it.

## Agreed direction

**1. Collapse the per-machine routes.** `setup`/`dash`/`logs`/`services` become sections
of one machine page. Subtraction, and a precondition for anything else — a single verdict
cannot sit above nav that multiplies.

**2. One verdict, above everything.** The app opens with one sentence: what needs
attention, or that nothing does. Screens become drill-downs rather than destinations.
Never make the operator assemble a verdict from parts.

**3. Sections state their own verdict as one line.** No tabs. Each former route is a line
— *"Containers · healthy · none restarting"* — that expands only if the operator wants
detail. The whole machine is readable without clicking.

## What got rejected, and why it matters

**"Comparison as the default rendering" was wrong for this page.** The idea was: never
render a number alone, always against peers or its past — `197 ms` is unreadable,
`197 of 7, worst 295` is not. That reasoning holds, but it answers *"how is it
performing?"* and this page answers *"is it healthy?"* Latency, block heights and share
percentages went to analytics, where the operator goes **to measure** rather than **to
check**.

**A redundancy verdict on a devnet is nonsense.** An early mockup painted `evm:1337`
amber for having one endpoint. A devnet is private and singular by design; amber implies
something to fix and there is nothing to fix. The app already holds the signal that
distinguishes the cases: **`catalog.KnownSetSize` returns 0 for 1337**, which is not a gap
to fill but proof redundancy is not an axis for that chain. It now reads
*"healthy · single by design"*.

Both mistakes are the same shape as one made earlier in the eRPC work — painting a
fully-redundant chain amber because its traffic share diverged. **A metric is only worth
rendering where it can be acted on.**

## A verdict is only worth the top of the screen if it is right

The first mockup led with *"two reths are fighting over port 8600"*. There was no
conflict. The `ssh` process holding that port is
`~/.colima/_lima/colima/ssh.sock [mux]` — colima's own forwarder, which holds **every**
published port (4020, 4030, 8443, 4001, 8600, 8601). `:8600` and the gateway's internal
`ws://valve-node-app-devnet:8546` return the same block with zero delta: one node,
forwarded normally.

A confident wrong headline is worse than the nine screens it replaces. Whatever computes
the verdict needs the same standard as the rest of this codebase — derived from what was
measured, not from what a listener looked like.

## Related decision, taken separately

Port pre-checks should stop refusing. `internal/setup/gateway.go`'s `checkPortFree`
blocked provisioning twice in one session — once because Caddy, its own front, held 8443,
and once because a merged-away orphan held 4001. Its own comment already argues the case:
docker fails loudly and specifically on a collision, so a false *busy* is the worse
outcome. Owner's position: adopt the process, reclaim the port, and if that takes down
someone's devnet, so be it.

One caveat raised and unresolved: that is right for a devnet, weaker for **8443**, where
the casualty is a TLS front and the failure is silent from its owner's side.

## Not yet decided

- Whether the fleet gets a top-level surface at all once per-machine routes collapse.
- Whether analytics keeps its own route or becomes the expanded state of the Traffic line.
- Time-as-the-spine (an event stream: provisions, image bumps, sync milestones, endpoint
  failures) was the most interesting idea generated and needs history the app does not
  record yet.
