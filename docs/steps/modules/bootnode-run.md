---
id: bootnode-run
title: Run your own bootnodes
summary: Run an EL bootnode (reth) and a CL bootnode (lighthouse) on a box with a public IP.
recap: |
  A bootnode needs a public IP and an open UDP port: 30301 for the EL bootnode, 9200 for the CL bootnode.
  Its enode (EL) or ENR (CL) shows on its card once it runs; copy it from there.
requires: [box-modules]
verified: {}
---

A bootnode is the first peer a node asks for other peers. Running your own
means your nodes find the network even when the public bootnodes are slow or
down.

jumpgate has two bootnode modules. They run on Ethereum (1), Hoodi (560048),
PulseChain (369) and PulseChain testnet (943):

- **EL bootnode.** Runs `reth p2p bootnode` for execution clients. Its address
  is an enode (`enode://...`). Default port: UDP 30301.
- **CL bootnode.** Runs lighthouse's `boot_node` (lighthouse-pulse on
  PulseChain) for beacon clients. Its address is an ENR (`enr:-...`). Default
  port: UDP 9200.

## What you need

- **A public IP.** Peers on the internet must be able to reach the box. Give
  that IP as the **public IP** when you turn the bootnode on. The CL bootnode
  will not start without it. The EL bootnode starts, but its enode may
  advertise `0.0.0.0`, and other nodes cannot dial that. The card then says
  "Set the box's public IP (ExtIP) on the bootnode."
- **An open UDP port.** Open the port in the box's firewall and in any cloud
  security group. The box's firewall checklist lists it, for example "UDP
  30301 listening publicly (EL bootnode)". That item checks that the service
  is bound to a public address. It cannot see a cloud security group, so check
  that yourself.

Each bootnode makes its own node key in its data dir the first time it starts.
The key is readable only by the service. jumpgate never copies it.

## Read your enode or ENR

Once the bootnode runs, its card shows the address it logged:

- **App.** The address is under **Running** on the bootnode's card. Click
  **Copy** next to it.
- **TUI.** Press `m` and move to the bootnode. The full address is printed
  under it. Press `c` to copy it through your terminal.

To use it on your own nodes, see the next step.
