---
id: watch-run
title: Turn on the health watch
summary: Run the health watch on a box so it watches its own node and your endpoints.
recap: |
  The health watch is a box module: the box's gate runs it as jumpgate-mod-watch.service.
  It polls every 30 seconds. An alert opens after two failing polls in a row and clears on the first good one.
requires: [box-modules]
verified: {}
---

The health watch runs on the box itself, so it keeps watching when the app
on your computer is closed. It is the box's gate running `jumpgate agent
watch`, as the unit `jumpgate-mod-watch.service`, under the `jumpgate-node`
account. It opens no port.

## What it watches

If the box has a node, the watch checks it every poll:

| Alert | When |
|---|---|
| `exec-down` | The execution client is not running. |
| `beacon-down` | The beacon client is not running. |
| `no-peers` | The execution client runs but has no peers. |
| `disk-90` | The node's disk is 90% full or more. |
| `exec-behind` | The node is a minute of blocks behind a reference: 5 blocks on Ethereum and Hoodi (12 s slots), 6 on PulseChain and PulseChain testnet (10 s slots), and never under 3. |

The lag rule (`exec-behind`) is built in but not switched on yet: it needs a
public reference endpoint for the chain, and setup does not give the watch
one.

The watch also checks each endpoint you list. An `rpc` endpoint must answer
`eth_blockNumber`; a beacon endpoint must answer `/eth/v1/node/health`. When
one stops, the alert is `endpoint:<name>`, with a message such as
`gw-eu (https://rpc.example.com) stopped answering.` The message names only
the endpoint's host, never its path or query, where API keys often sit.

A box with no node can still run the watch for its endpoints. Another box's
watch can list a box as an endpoint, which covers the case where a box stops
answering altogether.

## Turn it on

- **App.** Open the machine, find **Health watch** under **Services on this
  box**, and turn it on. List the endpoints to watch, one per line, as
  `name=url`. For a beacon API, write `name=beacon+url`, for example
  `cl=beacon+http://10.0.0.2:5052`.
- **TUI.** Press `m`, move to **Health watch**, and press Enter. Type the
  endpoints separated by commas.

The box must be paired first: the watch runs the box's gate.

Setup writes the watch's settings to its own config dir, readable only by
root and the `jumpgate-node` group. On a box with a node it also copies the
node's settings there, without the snapshot key and the eRPC upstreams, so
the watch can read them.
