---
id: msgboard-troubleshoot
title: When the message board does not work
summary: What each message board error means and what to do about it.
recap: |
  Setup checks reth's binary before it writes the unit, and reth's answer after the restart.
  "Waiting for sync" and "No peers" are not failures; a rebuild takes tens of minutes.
requires: [msgboard-status]
verified: {}
---

jumpgate checks for the message board at two points, so a box never ends up
with a reth that crashes on start:

- **Before the unit is written.** Setup asks `reth node --help` whether it
  lists `--msgboard.block-range`. It does not trust `reth --version`, because
  the valve fork builds from `main` without tags.
- **After the restart.** Setup asks reth for `msgboard_status` until it
  answers or the wait runs out.

## The node runs another client

On a go-pulse or erigon-pulse node the toggle is disabled and says so, for
example: "The message board needs reth (valve fork). This node runs
go-pulse." The server refuses the change too. To use the board, set the node
up again with reth.

If you switch a node that has the board to another client in the setup
wizard, the review step says "The message board will be turned off: go-pulse
does not support it.", and the module is removed from the new setup.

## The toggle is not there

The board is only for PulseChain mainnet (369) and testnet (943). On other
chains, and on a gateway-only box, there is nothing to turn on.

## Rebuilding reth

If the reth on the box predates the message board, setup says "Rebuilding
reth to add message board support (tens of minutes)" and builds it from the
valve fork's `main`. reth is restarted only once the new binary is in place.

If the rebuilt binary still lacks the board, setup stops with:

> reth at /usr/local/bin/reth has no message board support (`reth node
> --help` lists no `--msgboard.*` flags), even after a rebuild from
> valve-tech/reth main.

The unit is not changed, so reth keeps running as it was.

## reth runs but does not serve the board

If reth answers but does not know `msgboard_status`, setup stops with:

> reth is running but not serving msgboard. The unit's --http.api may have
> been edited.

When reth's own log says why, for example `Unknown RPC module: 'msgboard'`,
that line is quoted under the error. The card's status shows the same error.

There is no automatic rollback. The error offers **Turn the message board
off**, which runs the same change with the board off.

## Waiting for sync, or no peers

Neither is a failure. You can turn the board on while the node syncs; it
starts accepting messages once the node has caught up and has peers.

## Other refusals

- **A job is running on the box.** One job at a time: the change is refused
  until the job ends.
- **Root SSH is off on a paired box.** A module change needs the same SSH
  access setup needs.
- **The gateway says the method is unsupported.** No upstream behind that
  gateway serves the board on that chain. The gateway screen lists which
  chains route it.
