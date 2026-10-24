---
id: checkpointz-run
title: Run your own Checkpointz
summary: Serve finalized checkpoints from your own synced beacon node, for your own boxes.
recap: |
  Checkpointz needs a synced beacon node on its chain, and it should never point at a beacon with validators attached.
  It listens on 127.0.0.1:5555 by default, or on a private address you give. It never serves the internet.
  PulseChain is beta.
requires: [box-modules]
verified: {}
---

A beacon node that starts from scratch asks a **checkpoint-sync URL** for a
recent finalized state, then syncs forward from it. By default that URL is a
public provider. Checkpointz
([ethpandaops/checkpointz](https://github.com/ethpandaops/checkpointz))
serves the same thing from your own beacon node, so your other nodes trust
only your own boxes.

jumpgate builds Checkpointz v0.32.0 from source on the box and runs it as
`jumpgate-mod-checkpointz-<chain>.service`. It runs on Ethereum (1) and Hoodi
(560048). PulseChain (369) and PulseChain testnet (943) are **beta**: there
it decodes PulseChain's beacon preset (`custom_preset`), and that has not
yet been shown to work end to end.

## What you need

- **A synced beacon node on the same chain.** Checkpointz serves a
  checkpoint only once its beacon nodes agree on a finalized one. If the box
  runs a node on that chain, its own beacon is used, for example
  `http://127.0.0.1:5052`. Without one, the module is refused with
  "catalog: Checkpointz needs at least one synced beacon node on chain
  560048".
- **A beacon with no validators attached.** Checkpointz asks its beacon for
  full states, which is heavy work. Do not point it at a beacon that your
  validators use.
- **Memory.** Checkpointz keeps a few finalized states in memory, about 3 GB
  on Ethereum.

## Where it listens

Leave **Listen address** empty and Checkpointz listens on `127.0.0.1:5555`.
Only this box can reach it. To let your other boxes use it, give a private
address they can reach: a 10.x, 172.16-31.x or 192.168.x address, an IPv6
ULA (`fd00::/8`), or a Tailscale-style 100.64.0.0/10 address. jumpgate
refuses a public address and `0.0.0.0`, because Checkpointz serves only your
own boxes. **Port** changes 5555.

The beacon URLs are kept in the module's config file on the box, readable
by root and the service only. The module's public record (`module.json`)
never names them.

## Check that it serves

The card shows **Serving · finalized epoch N** once Checkpointz has a
finalized checkpoint. Until then it says "Checkpointz has no finalized
checkpoint yet. Its beacon nodes may still be syncing." On the box itself:

```sh
curl -s http://127.0.0.1:5555/checkpointz/v1/status
```

The `finality` field holds the finalized epoch. To use it on your other
nodes, see the next step.
