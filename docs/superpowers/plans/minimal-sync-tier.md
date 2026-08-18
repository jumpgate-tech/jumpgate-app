# A third sync tier — reth "minimal"

Date: 2026-08-17. Status: captured, not designed. Raised by the owner.

Reth can run a node far smaller than a full node — on the order of a couple of
hundred GB rather than a terabyte or more. Today an operator picks between
archive and full. A third, much cheaper tier changes who can afford to run a
node at all, so it belongs both in the wizard and on the landing page.

## Verify this first

The tier's real name, its exact pruning configuration, and its true on-disk size
are NOT verified here. They come from the owner's reading, not from reth's docs
or a measured node. Confirm all three against reth upstream before any of the
work below, because the sizing number is the whole point of the feature and a
wrong one puts operators on a disk that fills.

## Why this is wider than it looks

The tier is a **boolean** today, not an enum. `archive bool` is threaded through:

| Place | Today |
|-------|-------|
| `catalog.ExpectedBytes(chainID, archive bool)` | archive → `SnapshotSizeTB`; full → that × `fullTierFraction` |
| `catalog.fullTierFraction` | the constant `0.5`, and its comment says plainly it has no source |
| `catalog.GatewayUpstream.RecentOnly` | set from `!w.Archive` (`gateway.go:546`) |
| `internal/setup` preflight | the disk floor is `ExpectedBytes × 1.10` |
| `web/src/wizard.ts` | mirrors the same fraction as `FULL_TIER_FRACTION` |

So adding a tier is mechanical but wide: every one of those has to stop asking
"is it archive?" and start asking "which tier?".

The good news is that `fullTierFraction` already anticipates it. Its comment
says the fix is to "delete this constant and give ExpectedBytes a
per-(client, tier, chain) lookup". A real measured size for a real third tier is
exactly the figure that unblocks that. Do the enum and the lookup in one change
rather than bolting a second fraction beside the first.

## Two things that are easy to get wrong

**A tier is a client capability, not a universal setting.** Reth prunes one way,
erigon another (`--prune=hrtc`, already in `units.go:322`), and geth another
again. A minimal tier that appears for a client which cannot serve it is a
support ticket. The tier list must come from the client, the way
`Network.BeaconClients` already gates which consensus clients a chain offers.

**`RecentOnly` is currently binary and would lump minimal in with full.** The
gateway trusts a non-archive upstream only for recent history, bounded by
`recentBlockWindow` (`gateway.go:28`). A minimal node's usable window is not a
full node's. Leaving them equal would route historical queries to a node that
cannot answer them, and eRPC would fail them one at a time rather than never
sending them. The window needs a per-tier value at the same time as the tier
lands, not after.

## Landing page

`github/landing` is a separate repo — one static `index.html` on Cloudflare
Pages. It is low risk and instantly revertible, so it ships proactively with
`./deploy-landing.sh` rather than waiting on the app work. It can carry the
three-tier story as soon as the size figure is confirmed, since it makes a claim
about cost rather than a promise about a wizard option.

## Order

1. Confirm the tier name, the pruning flags, and the real size against reth.
2. Widen the boolean to a tier enum, per client, and replace `fullTierFraction`
   with the per-(client, tier, chain) lookup its comment already asks for.
3. Give `recentBlockWindow` a per-tier value.
4. Wizard option, then the landing page.

Steps 2 and 3 land together. Splitting them ships a gateway that routes
historical queries at a node which cannot serve them.
