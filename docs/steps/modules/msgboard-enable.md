---
id: msgboard-enable
title: Turn on the message board
summary: Add the PulseChain message board to a reth node on 369 or 943.
recap: |
  The message board runs inside reth (valve fork) on PulseChain mainnet (369) and testnet (943).
  Turning it on restarts reth only; the beacon keeps running. It is local to the box and your own gateway.
requires: [modules-what, pair-box]
verified: {}
---

The message board is a module that lives on your node. It adds the
`msgboard` namespace to what your execution client serves, so
[msgboard.xyz](https://msgboard.xyz) and other apps can read and post
messages through your node.

## Where it can run

- **Chains.** PulseChain mainnet (369) and PulseChain testnet (943). On any
  other chain the toggle is hidden.
- **Client.** reth, built from the valve fork. On a go-pulse or erigon-pulse
  node the toggle is shown but disabled, with the reason under it, for
  example: "The message board needs reth (valve fork). This node runs
  go-pulse."
- **Cost.** About 256 MB of extra RAM and 128 MB of disk.

## Turn it on

In the app, open the machine, go to **Modules**, and switch on **Message
board**. In the TUI, press `m` on the Overview tab, toggle the message board,
and type `restart` to confirm.

Before anything runs, jumpgate tells you what it will do, naming the box:

> This restarts reth on &lt;box&gt;. The beacon keeps running.

jumpgate then runs the node's setup again, with the module added:

- reth's `--http.api` becomes `eth,net,web3,msgboard`. jumpgate never sets it
  to `all`.
- If the reth on the box predates the message board, setup rebuilds it from
  the valve fork first. That takes tens of minutes, and the unit is not
  changed until the new binary has the board.
- Only the reth service restarts. The beacon unit is unchanged.
- After the restart, setup checks that reth answers `msgboard_status`.

Turning it on a second time changes nothing and restarts nothing.

## Who can reach it

By default the board is local: it is served on the box and through your own
gateway, and nowhere else. Making it public is a separate, explicit choice
per node.

## Turn it off

Switch the toggle off. Setup removes the namespace and restarts reth again;
the beacon keeps running.
