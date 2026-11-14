# Modules

A module is an optional extra that jumpgate turns on for you. Each one lives in one of three places:

- **On your node.** It adds to what your node's execution client serves. The message board is one. Turning it on or off restarts the execution client; the beacon keeps running.
- **As its own service on a box.** It runs beside your node, or on a box with no node at all, as a systemd unit named `jumpgate-mod-NAME.service` (with the chain id added for a per-chain module). The box's gate starts it, keeps it running and reports its status. The EL and CL bootnodes, Checkpointz and the health watch are box modules.
- **On a gateway.** It changes where the gateway sends certain requests. Private send is one.

A module you do not see is not for this chain, so nothing is wrong. A module you see but cannot turn on says why in one line under it.

## Turning one on

- **A box's modules.** Open the box and press `m`. Move to a module and press `enter` to turn it on or off. A box module asks for its settings first: `tab` moves between the fields and `enter` turns it on. The progress shows under the module.
- **A node module's reach.** A node module is local to the node unless the node is behind a public gateway. Then `p` cycles it between local only, public reads, and public reads and posting. Allowing posting asks you to confirm.
- **A gateway's modules.** On the Gateways screen, `m` lists the selected gateway's modules; press a module's number to turn it on or off.

Turning a box module on saves it on the box's record and runs a short setup for that one service: it checks the box, creates the `jumpgate-node` account if needed, installs the client the service runs if the box lacks it, writes the unit and starts the service. Turning it off stops the service and removes its unit; the data directory stays, so a bootnode turned on again keeps its identity. A box module never restarts your node's clients.

## A legacy box must be renamed first

A box set up before the `jumpgate-node` names runs its node as the `valve-node-app` account. Box modules are refused there until the `node.migrate-names` job has run on it: a module runs as `jumpgate-node`, and creating that account beside the old one would leave the rename with two accounts to choose from. The refusal says so. Run the job (see [Durable jobs](jobs.md)), then turn the module on.

## The modules

- **EL bootnode** and **CL bootnode** (box modules): discovery-only bootnodes other nodes seed from, for Ethereum, PulseChain, PulseChain Testnet v4 and Hoodi. The execution bootnode listens on UDP port 30301 and the consensus one on UDP port 9200, unless you give another port. The CL bootnode also needs the box's public IP, for its ENR. Each keeps its node key in its data directory. When one is running, `c` copies its enode or ENR.
- **Checkpointz** (box module): serves finalized checkpoints from your own beacon node, so your other nodes can checkpoint-sync from it. It serves your own boxes only: it binds to loopback or a private address (such as a Tailscale one), never a public one, on port 5555 unless you give another. It is available for Ethereum and Hoodi; PulseChain says why it is not.
- **Health watch** (box module): the box's gate runs `jumpgate agent watch` as `jumpgate-mod-watch.service`, so it keeps watching with the app closed. It polls every 30 seconds. An alert opens after two failing polls in a row and clears on the first good one. On a box with a node it watches `exec-down`, `beacon-down`, `no-peers` and `disk-90` (disk 90% full). It also probes each endpoint you list as `name=url`, or `name=beacon+url` for a beacon API, and raises `endpoint:name` when one stops answering. Alerts name only an endpoint's host, never its path or query. A box with no node can still watch endpoints.
- **Message board** (node module): serves the PulseChain message board from your node. It needs reth, the Valve build, on PulseChain or its testnet.
- **Private send** (gateway module): on a gateway that serves Ethereum mainnet, sends `eth_sendRawTransaction` only to Flashbots Protect and MEV Blocker, never to the public mempool. It is strict: if neither answers, the send fails, and no public upstream is tried. Reads stay on the gateway's normal upstreams. On other chains it is disabled, with the reason.
