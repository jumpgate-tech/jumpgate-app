---
id: bootnode-use
title: Point your nodes at your bootnodes
summary: Add your bootnode's enode or ENR to every node of yours on the same chain.
recap: |
  enode:// entries go to the execution client and enr:- entries to the beacon client.
  Adding one restarts only the client it goes to, on each node that changed.
requires: [bootnode-run]
verified: {}
---

Once a bootnode runs, you can add it to the other nodes in your fleet on the
same chain.

## Use it on your nodes

In the app, open the bootnode's box and go to **Modules**. On the running
bootnode's card, click **Use on my other &lt;chain&gt; nodes**. The dialog
lists each of your other nodes on that chain and says what will restart, for
example:

> Restarts reth on node-1.example.

Click **Use bootnode**. For each node, jumpgate adds the address to the node's
bootnode list and runs its setup again. The dialog then says, per node,
whether setup started. A node that already lists the address is skipped.

## Where each entry goes

- **enode:// entries** go to the execution client: `--bootnodes a,b` on reth,
  geth, go-pulse and erigon-pulse.
- **enr:- entries** go to the beacon client: `--boot-nodes a,b` on lighthouse
  and lighthouse-pulse, and one `--bootstrap-node=` flag per entry on
  prysm-pulse.

An enode never reaches a beacon client, and an ENR never reaches an execution
client. Only the unit that changed restarts. Adding an enode restarts the
execution client, and the beacon keeps running.

## What jumpgate accepts

Each entry must start with `enode://` or `enr:-`. An enode must have a
128-hex node id, a host and a port. An ENR must be base64url. jumpgate refuses
anything else, including spaces, commas and shell characters, with "catalog: a
bootnode must start with enode:// or enr:-" or a reason for the malformed
entry.

## Stop using it

A node with no bootnodes has exactly the units it had before. Removing every
entry puts the clients back on the network's built-in bootnodes.
