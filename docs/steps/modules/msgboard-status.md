---
id: msgboard-status
title: Read the message board's status
summary: What the message board's one status line means.
recap: |
  The status comes from asking reth msgboard_status on every poll, not from config.
  "Accepting" with a count and size means the board works; anything else names what it waits for.
requires: [msgboard-enable]
verified: {}
---

Once the message board is on, the machine's Modules card in the app and the
Modules block on the TUI's Overview tab show one status line for it.

## Where the line comes from

On every poll, the box's gate asks reth for `msgboard_status` on loopback.
That one cheap call is made only on reth nodes on 369 and 943. The answer
says whether the board is accepting messages, how many it holds, how large
it is, and the last block it has seen.

Because the status is read from what reth actually serves, it is right even
when the config says something else: for example on a box whose unit was
edited by hand.

## What the line says

The line is decided in this order:

1. **Node stopped.** The execution client is not running.
2. **Starting.** reth was restarted less than a minute ago and does not
   answer yet. After a minute this becomes an error; see the troubleshooting
   step.
3. **Waiting for sync.** The node is still syncing. The board accepts nothing
   until it has caught up. Nothing is wrong.
4. **No peers.** The node has no peers to exchange messages with. The count
   shown is the node's eth peer count.
5. **Starting.** reth answers but the board is not accepting yet.
6. **Accepting · 1,234 messages · 3.1 MB.** The board works.
7. **Served but not set by jumpgate.** jumpgate did not turn the board on,
   but reth serves it. This is a hand-edited unit; jumpgate shows it and
   leaves it alone.

## An older gate

A box whose gate predates modules does not report module status. The card
says "Status needs a newer gate on this box". The toggle still works, because
turning a module on or off goes through setup, not the gate.
