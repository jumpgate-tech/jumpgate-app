# Tunneling, custom domain, and named replicas

Reconciled with the SHIPPED "Private access" overlay (internal/vpn,
internal/server/vpn*.go, screens/PrivateAccess). This plan builds ON that code;
it does not re-spec it. It ties the shipped overlay to three net-new threads:
the custom domain the hub answers on, the beacon/eRPC arch-routing
(see beacon-routing.md), and the named node replicas that fill each chain's pool.

## What already ships — reconcile to this, do not rebuild

"Private access" (route #/vpn) already stands up and manages WireGuard:

- **Provisioned server (the hub).** `vpn.ProvisionServer` creates interface
  `jumpgate0` (Address 10.9.0.1/24, ListenPort 51820) on a machine you own,
  generates the server key ON the host (`wg genkey`, umask 077) and reads back
  only the public key, writes `/etc/wireguard/jumpgate0.conf`, brings it up with
  `wg-quick up`, and verifies it is actually listening.
- **Enroll devices (the spokes).** `vpn.AddPeer` runs
  `wg set jumpgate0 peer <pub> allowed-ips <ip>/32` + `wg-quick save`, verified
  against `wg show … dump`. IPAM: `NextPeerIP` hands out the next free `/32`. The
  device's client `.conf` (Endpoint = the public host:port it dials,
  PersistentKeepalive 25, AllowedIPs = the server subnet or 0.0.0.0/0) is
  returned ONCE and never stored.
- **Transport = `wg`/`wg-quick` through the executor** (local or SSH), NOT
  wgctrl-go. This is the right choice and must stay: the executor drives both
  the local host AND remote node boxes over SSH; wgctrl-go configures only the
  local kernel interface, so it could not manage a remote node box at all.
- **Keys never leave their box.** Server key minted on the host (pubkey-only
  travels); device key generated locally in Go (curve25519) and delivered once;
  config bytes redacted from every view.
- **Lifecycle:** Disconnect (reversible `wg-quick down`, keeps conf/key/peers)
  vs Wipe (`DeprovisionServer`, removes conf+key, verified gone).
- **BYO overlays.** Paste a provider `.conf` (Proton/Mullvad/IVPN/self-hosted);
  Jumpgate brings that existing overlay up/down and grades the bound gateway
  "private". Here Jumpgate is the spoke and the provider runs the hub.

Vocabulary to use everywhere below: **server** (hub), **device/peer** (spoke),
**overlay** (tunnel), **provision** (stand up the server), **enroll/revoke**
(add/remove a device), **endpoint** (public host:port dialed), **disconnect vs
wipe** (teardown), interface **`jumpgate0`**, **provider / BYO** (pasted `.conf`).

## Correction to the earlier draft

The earlier draft called for wgctrl-go and framed the overlay as unbuilt. Both
were wrong. The overlay is shipped, and `wg-quick`-via-executor is the correct
transport because it reaches remote node boxes over SSH. **wgctrl-go is dropped.**

## Decisions locked (owner, 2026-08-13)

1. Jumpgate stands up and manages the overlay itself — DONE for the
   provisioned-server path.
2. Private by default, with a warning when the operator removes the privacy —
   PARTIAL today (see net-new #4).
3. Custom domain over public HTTPS for a remote hub — net-new (#3).
4. Named replicas per chain feed the upstream pool — net-new (#1, #2).
5. D-BTC: extend eRPC, config-driven (not a per-chain balancer) —
   see erpc-polyglot-proxy.md.
6. D-DNS: guide the operator's DNS timing, do not gate issuance.

## Net-new work — this is what the plan is actually FOR

The shipped overlay does NOT do any of the following. This is the real backlog.

1. **Named replicas per chain.** The VPN subsystem has no chain or replica
   concept. Add a human replica NAME per node of a chain (e.g. `eth-a`,
   `eth-b`), and map that name to the node's overlay address — the `/32` its
   device holds on `jumpgate0`. Two nodes of one chain become distinguishable in
   the UI and the config.
2. **Per-chain upstream pool.** Assemble a chain's pool from its named replicas'
   overlay endpoints, load-balanced with health-based failover. eRPC already
   pools EVM; the config-driven patterns (erpc-polyglot-proxy.md) cover the rest;
   beacon = L7 + `/eth/v1/node/health`.
3. **Custom-domain public HTTPS on the hub.** Entirely outside the VPN feature
   today (it only manages UDP endpoints). Internal CA for the loopback name;
   automatic public ACME over HTTP-01 for a real domain (a remote hub with a
   real A record needs no DNS credentials — the loopback constraint that shelved
   `CertACME` does not apply). D-DNS: guide the operator's DNS timing and warn
   before an action would spend an ACME rate limit; do not gate.
4. **Private-by-default posture + the disable warning.** The grading hook exists
   (a bound overlay grades the gateway "private"), but the policy/UX does not.
   Build it: bind node RPC to the `jumpgate0` address so "no overlay" means "no
   listener", not "public listener"; keep the public-interface firewall
   INDEPENDENT of overlay state (today the VPN only returns a `FirewallHint` for
   the operator to run — making it enforce is part of this); warn explicitly when
   the operator disables the overlay or rebinds a node to a public address.
5. **Beacon / eRPC arch-routing.** `/rpc/:key/:arch/:chain_id`, prefix strip, SSE
   passthrough, key check on both arches, clean 404 for a chain with no beacon.
   Wholly net-new. See beacon-routing.md.

## How it composes (over the shipped overlay)

`https://rpc.mydomain.com/rpc/:key/:arch/:chain_id/...`
1. The hub (a provisioned server, or the box the operator runs) terminates TLS
   with a real cert (net-new #3).
2. The key is validated (the jg_ billing/key service — beacon B1, key lifecycle
   below).
3. The arch router sends `evm` and every other chain to the config-driven eRPC
   proxy, `beacon` to the REST pattern.
4. Each pattern load-balances across the chain's pool — the named replicas
   (net-new #1) — reached over the WireGuard overlay on their `jumpgate0`
   addresses (the shipped provision + enroll).

## Key lifecycle (billing service)

An operator issues a jg_ key, hands it to a third party, and keeps control:
enable/disable per key (reversible), re-price/upsell after issuance tied to that
toggle, and scoped/delegated permissions. Lands in services/billing. NOTE:
"socket permissions" in the owner's words is read as scoped/delegated — confirm
before building.

## Build order

1. (SHIPPED) overlay: provision server + enroll devices.
2. Private-by-default posture + disable warning (bind to `jumpgate0`, make the
   public-interface firewall enforce independently, warn on disable).
3. Named replicas: a name per node, mapped to its overlay `/32`; pool membership.
4. Custom-domain HTTPS on the hub (internal CA vs public HTTP-01; guide DNS
   timing).
5. Config-driven eRPC patterns (beacon first, then Bitcoin, then the rest).
6. Key lifecycle in the billing service.

## Open decisions

- **B1** — where the per-arch key check runs (Caddy forward-auth to the billing
  admin API vs a Go shim). Shared with beacon-routing.
- **Replica lifecycle** — per-chain start/stop does not exist yet; how much of it
  the named-replica work needs.
- **Key delegation model** — confirm the "socket permissions" reading (scoped)
  before the billing build.
- **Firewall enforcement** — the shipped VPN only HINTS (`ufw allow`); decide
  whether Jumpgate enforces the public-interface firewall for private-by-default,
  or keeps hinting.

RESOLVED: transport is `wg-quick`-via-executor (wgctrl-go dropped); D-BTC
(extend eRPC, config-driven); D-DNS (guide, do not gate).
