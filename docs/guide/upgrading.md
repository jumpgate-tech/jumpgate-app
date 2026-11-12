# Upgrading

## The controller

Download the new release and replace the app or the binary. The background server from the old version may still be running: commands warn you when the server is a different jumpgate version. Run `jumpgate stop` to replace it; the next command or app launch starts the new one. When the versions differ, the TUI also offers `R` to restart the server.

If the app finds a server already running with different options, it prints which options are not in effect. Run `jumpgate stop` and launch again to apply them.

On a Linux server where a package installed the service, upgrade with `apt upgrade jumpgate` or `dnf upgrade jumpgate`. The package does not restart a running service, because a restarted service comes back locked and stops every box it drives until someone unlocks it; a service whose vault key is sealed (see [The vault](vault.md)) is the exception and restarts onto the new binary. Otherwise restart it yourself and unlock it.

## The gate on a box

After you upgrade the controller, replace each box's gate with the controller's:

```
jumpgate hosts upgrade box-a
```

It reports the gate's version before and after, or that it is already current. In the TUI, `u` on the table of addresses and pairing (`v` on the Fleet) does the same.

Running `jumpgate hosts add` again on an existing name re-pairs the address on record. That finishes an interrupted pairing. It refuses a different address (`--ssh`, `--jump`, or `--local` instead of `--ssh`). To point the name somewhere else, remove the host in the app or with `:remove NAME` in the TUI, then add it again.

```
jumpgate hosts add box-a --ssh root@203.0.113.7
```

## Moving off the legacy names

New setups use the `jumpgate-node` account from the start. Older boxes still on `valve-node-app` move with the `node.migrate-names` job.

- **Upgrade every controller that manages a box before you migrate it.** A controller from before the migration does not know about it; running setup from it afterwards would set the box up under the legacy names again. A current controller refuses to set up a migrated box under the legacy names.
- Then run `jumpgate migrate-names HOST --yes`, or `:migrate-names NAME` in the TUI. The node stops while the job runs.
- Box modules (bootnodes, the health watch, Checkpointz) are refused on a box whose node still runs under the legacy names, until it has been migrated. See [Modules](modules.md).
