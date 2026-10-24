---
id: msgboard-expose
title: Open the message board to the public
summary: Choose who can reach the message board through a Public-tier gateway.
recap: |
  The message board is local by default. Behind a Public-tier gateway you can open its reads, or its reads and posting.
  Posting is a write, msgboard_content is never public, and a relay key posts only if its allow list names msgboard_addMessage.
requires: [msgboard-enable]
verified: {}
---

The message board starts local: your box and your own gateways reach it,
and a gateway with a public certificate routes none of its methods. Making
it public is a separate, explicit choice per node.

## When the choice appears

The **Public** choice shows on the message board's card only when it is on
and a Public-tier gateway (one with a publicly trusted certificate) routes
to this node. In the app it is a radio under the card. In the TUI, press
`m`, move to the message board, and press `p` to cycle it.

The three values:

- **Local only.** The default. A Public-tier gateway routes no `msgboard_`
  method to this node.
- **Public reads.** Anyone can call `msgboard_status`,
  `msgboard_categories`, `msgboard_contentPage` and `msgboard_getMessage`.
- **Public reads and posting.** Reads, plus `msgboard_addMessage`. The app
  warns: "Anyone can post through your node. Posts cost the sender proof of
  work, not you." The TUI asks you to press enter before it saves.

## How the gateway enforces it

jumpgate does not trust a client to stay polite. For each upstream it
derives the gateway's `ignoreMethods` list (`moduleIgnoreMethods`):

- An upstream that does not serve the message board ignores the whole
  `msgboard_*` namespace, so a call never fails over to an endpoint that
  would answer "method not found" or, worse, accept a post you did not mean
  to send elsewhere.
- Subscriptions (`msgboard_subscribe`, `msgboard_unsubscribe`) are ignored
  behind every gateway: the node has no WebSocket listener for them. Poll
  `msgboard_contentPage` instead.
- Behind a Public-tier gateway, whatever your choice does not open is
  ignored.
- `msgboard_content` is never public, whatever you choose. It builds the
  whole board in one call, and nothing can price or bound that. Public
  readers page with `msgboard_contentPage`.

## Posting is a write

`msgboard_addMessage` gossips a message to the network. It is a write, not
a read, and it is treated like one at every layer:

- The gateway routes it only under **Public reads and posting**.
- The relay (`checkOneMethod`) never lets an empty allow list's
  "everything" cover a module write. To post through a relay key, the key's
  method allow list must name `msgboard_addMessage`. Otherwise the call is
  refused with: "msgboard_addMessage needs to be on this key's method allow
  list".
- A key's block list still wins over its allow list.

## What it costs

Through the relay, `msgboard_contentPage` and `msgboard_addMessage` are
priced from the price book. Until the book names them they take the
default price. `contentPage` can return a full page in one call.

## Turn it back

Pick **Local only** again. The gateway config is re-rendered; the node is
not restarted.
