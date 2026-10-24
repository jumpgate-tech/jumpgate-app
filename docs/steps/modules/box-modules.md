---
id: box-modules
title: Run a service on a box
summary: Turn on a box module, such as a bootnode, on any box you have paired.
recap: |
  A box module is its own service on a box, started and watched by the box's gate.
  It needs no node on the box: a small box can run a bootnode and nothing else.
requires: [modules-what, pair-box]
verified: {}
---

A box module is a service jumpgate runs on a box, beside your node or on a
box with no node at all. The EL and CL bootnodes are box modules.

## It runs on its own

A box module does not need a node. You can pair a small box, set up no node
on it, and turn on a bootnode. The box's gate starts the service, keeps it
running and reports its status.

If the box also runs a node, the module is a separate service. Turning it on
or off never restarts your node's clients.

## Turn it on

In the app, open the machine and go to **Modules**. Box modules are listed
under **Services on this box**. Switch one on. jumpgate then asks for the
module's settings, for example a bootnode's port and public IP.

In the TUI, press `m` on the Overview tab. Move to the module, press `enter`,
fill in the fields (`tab` moves between them) and press `enter` again.

jumpgate saves the module on the box's record and runs a short setup for that
one service. It writes the unit, installs the binary if the box lacks it and
starts the service. The progress shows under the toggle.

## Turn it off

Switch the toggle off. Setup stops the service and removes its unit. The data
dir stays, so a bootnode turned on again keeps its identity.
