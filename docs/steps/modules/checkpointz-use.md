---
id: checkpointz-use
title: Checkpoint-sync your nodes from your Checkpointz
summary: Set your Checkpointz as the checkpoint-sync URL of your other nodes on the same chain.
recap: |
  This sets the node's CheckpointURL. Only its beacon restarts.
  The app offers it only for a Checkpointz on a private address that serves a finalized checkpoint.
requires: [checkpointz-run]
verified: {}
---

Once your Checkpointz serves a finalized checkpoint, your other nodes on the
same chain can use it as their checkpoint-sync URL instead of a public one.

## Use it on your nodes

In the app, open the Checkpointz box and go to **Modules**. On the
Checkpointz card, click **Use as checkpoint for my other &lt;chain&gt;
nodes**. The dialog shows the URL, for example `http://10.0.0.5:5555`, and
lists each of your other nodes on that chain with what will restart, for
example:

> Restarts lighthouse on node-1.example.

Click **Use checkpoint**. For each node, jumpgate sets the node's
**CheckpointURL** and runs its setup again. Only the beacon restarts. The
execution client keeps running. A node that already uses the URL is skipped.

The button shows only when both of these are true:

- **Checkpointz listens on a private address.** One on loopback serves only
  its own box, and the card says "This Checkpointz serves only this box. Turn
  it off and on with a private listen address to let your other boxes use
  it."
- **It serves a finalized checkpoint.** The card says **Serving · finalized
  epoch N**.

## When it takes effect

A beacon uses its checkpoint-sync URL only when it starts with an empty
database. A node that is already synced keeps its chain. The new URL is used
the next time it syncs from scratch, for example after a clear and resync.

## Stop using it

Open the node's setup and clear its checkpoint-sync URL, or set another. The
node then goes back to the network's default provider.
