---
id: modules-what
title: What a module is
summary: Modules are optional extras you turn on per node, per box or per gateway.
recap: |
  A module lives on your node, as its own service on a box, or on your gateway.
  A hidden toggle means the module is not for this chain; a disabled one says why.
requires: [first-sync]
verified: {}
---

A module is an optional extra that jumpgate can turn on for you. Each one
lives in one of three places:

- **On your node.** It adds to what your node's execution client serves, for
  example the message board. Turning it on restarts the execution client; the
  beacon keeps running.
- **Its own service on a box.** It runs beside your node, or on a box with no
  node at all, for example a bootnode or the health watch. The box's gate
  starts and watches it.
- **On your gateway.** It changes where your gateway sends certain requests,
  for example private send on Ethereum.

## When a toggle is hidden or disabled

- A module you don't see is not for this chain. Nothing is wrong.
- A module you see but can't turn on always says why, in one sentence, right
  under the toggle. For example, a module may need a different execution
  client than the one this node runs.
