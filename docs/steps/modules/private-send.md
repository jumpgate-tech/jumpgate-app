---
id: private-send
title: Send transactions privately
summary: Turn on private send for an Ethereum gateway, so your transactions skip the public mempool.
recap: |
  Private send is a gateway module, for Ethereum mainnet only.
  eth_sendRawTransaction goes only to Flashbots Protect and MEV Blocker; if neither answers, the send fails and is never sent publicly.
requires: [modules-what]
verified: {}
---

Private send is a gateway module. On a gateway that serves Ethereum mainnet
(chain 1), it sends every `eth_sendRawTransaction` to two private routes
instead of the public mempool:

| Route | Endpoint |
|---|---|
| Flashbots Protect | `https://rpc.flashbots.net/fast` |
| MEV Blocker | `https://rpc.mevblocker.io` |

Only the send goes there. Reads (`eth_call`, `eth_getBalance`, nonces, and
so on) stay on the gateway's normal upstreams, your own node first.

Every key on the gateway gets private send, including keys used through
`jumpgate relay`. Their rate limits and pricing apply as usual.

## A failed private send is an error

Private send is strict. If neither route answers, the call returns an error.
jumpgate never falls back to a public upstream: every other upstream on
chain 1 is set to ignore `eth_sendRawTransaction`, including the public ones
used as fallbacks for reads. Retry the send, or turn the module off if you
want public sends again.

## Only Ethereum mainnet

On other chains the toggle is disabled, with the reason:

- **Hoodi:** no private send route exists. Flashbots Protect lists no Hoodi
  endpoint.
- **PulseChain and PulseChain testnet:** no private send route exists. The
  one known PulseChain private RPC refuses anonymous calls, and its relay is
  down.

A gateway that serves Ethereum and PulseChain gets private send on chain 1
only. Its PulseChain upstreams are not changed.

## Not in this release

The routes are called with no options: no refund address and no builder
hints. Their URLs carry no key or query string, so nothing secret is
written to the gateway's config or logs.

## Turn it on

- **App.** Open **RPC**, find the gateway, and turn on **Private send**
  under **Modules**. The gateway is re-provisioned on its gate.
- **TUI.** The gateways screen is read-only; a gateway with private send
  shows the line `private send: transactions go only to Flashbots Protect
  and MEV Blocker`.
