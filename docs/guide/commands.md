# Commands

`jumpgate help` lists every command. The ones you will use most:

```
jumpgate keys init                       # create the controller signing key
jumpgate keys show                       # the address of the key in the key store
jumpgate unlock                          # unlock the vault so the server can sign
jumpgate hosts add box-a --ssh root@203.0.113.7
jumpgate hosts list                      # boxes and their pairing
jumpgate hosts invite box-a              # the one line that sets up a box's gate
jumpgate hosts upgrade box-a             # replace a box's gate with this controller's
jumpgate status box-a                    # sync, peers and disk of a paired box
jumpgate disk box-a                      # disk usage
jumpgate endpoints box-a                 # RPC endpoints
jumpgate firewall box-a                  # the firewall checklist
jumpgate logs box-a -n 200               # recent node logs
jumpgate service box-a beacon restart    # exec|beacon, start|stop|restart
jumpgate ssh box-a                       # a shell on the box, under the host keys you confirmed
jumpgate jobs                            # durable jobs on every box
jumpgate migrate-names box-a --yes       # move a node to the jumpgate-node names
jumpgate open                            # open the web app in your browser
jumpgate serve                           # run the server in the foreground
jumpgate stop                            # stop the background server
```

The vault has its own commands (`jumpgate vault init`, `backup`, `restore` and more; see [The vault](vault.md)), and so does the server service (`jumpgate setup install-service` and `jumpgate setup server`; see [Getting started](getting-started.md)).

## Opening the web app

`jumpgate open` starts the server if needed and signs your browser in with a one-time login link. The link works once, within 60 seconds, and only over a loopback connection: a browser on this computer, or one reaching it through an SSH tunnel. `jumpgate open --print` prints the link instead of opening a browser.

To use jumpgate from another machine, keep the default bind and tunnel to it, for example `ssh -L 8799:127.0.0.1:8799 you@host`. Then run `jumpgate open --print` on the host and open the printed link on your machine.

## Serving metered RPC: jumpgate relay

`jumpgate relay` serves keyed, metered RPC as a process of its own. It never opens the controller's config, key or SSH executor, which keeps the internet-facing proxy away from the key that controls your fleet.

```
jumpgate relay --relay-bind 127.0.0.1:8545 --billing-socket /run/jumpgate-billing/billing.sock --meter
```

- It takes its settings from flags and `JUMPGATE_*` environment variables.
- The relay token comes from a file named by `JUMPGATE_RELAY_TOKEN_FILE`, readable only by you (`chmod 600`), or from `JUMPGATE_RELAY_TOKEN`; never from a flag.
- Bind it to loopback or the interface your proxy reaches, never to `0.0.0.0`.
- With metering on, a call can be refused with `402` (out of credits), `403` (account not provisioned) or `503` (the credit ledger did not answer).

## Exit codes

- `0`: success.
- `1`: the operation failed or was refused.
- `2`: bad usage, or a missing prerequisite, such as no controller key yet or a box that is not paired.
- `3`: the box is unreachable.
- `4`: a security failure: a host key nobody confirmed, a host key that changed since it was confirmed, a confirmed fingerprint that is not the key the box presented, a reply not signed by the paired gate, or a controller key that is not the one on record.
