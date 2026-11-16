# Durable jobs

A durable job is a multi-step operation that the box itself carries out. Each step runs as a transient systemd unit, so a running step outlives the TUI, the controller server and even the gate process. The only kind of job today is `node.migrate-names`.

## How a job behaves

- **It keeps a signed log.** The gate records every start, step outcome and answer in a signed, append-only log that any controller verifies against the box's gate key.
- **It runs a fixed plan.** A job never takes shell from anyone. The gate runs each step of a fixed plan for its kind; the controller that drives the job starts each next step. If that controller goes away, the job finishes the step it is on and waits.
- **It asks.** A step that cannot choose safely asks you, and the job waits.
- **It survives reboots.** If the box reboots in the middle of a step, the job asks before it runs that step again. It never reruns one on its own.
- **The node is the job's.** While a job on a box has not finished, the box refuses service start, stop and restart, and the controller refuses node setup and clear on it.

Jobs run on Linux boxes with systemd. Any controller, on macOS, Windows or Linux, can list and watch them. The controller that holds a job's lease answers its questions; any other can adopt it once the lease is stale.

## Who drives

One controller drives a job at a time, by holding its lease; every other controller only watches. A lease silent for 90 seconds, by the box's clock, is stale. Taking over a stale lease held by another controller is your decision: use `jumpgate jobs adopt`, or adopt in the jobs inbox.

## Commands

```
jumpgate jobs [HOST]                            # state, step, lease, open question
jumpgate jobs show HOST JOB                     # the steps and the signed log, verified
jumpgate jobs decide HOST JOB DECISION OPTION   # answer the job's question
jumpgate jobs adopt HOST JOB                    # drive a job whose lease is stale or free
jumpgate jobs output JOB                        # follow the job's output
jumpgate migrate-names HOST --yes               # start node.migrate-names
```

In the TUI, `2` opens the jobs inbox: `enter` shows a job's steps, `d` answers its question, `ctrl+t` takes it over when its controller has gone silent and `ctrl+n` starts a job on a box. The `:migrate-names NAME` command starts the migration too.

## node.migrate-names

Older installs run the node as the `valve-node-app` account. `jumpgate migrate-names HOST --yes` starts a job that moves it to `jumpgate-node`:

- it stops the node services, then renames the account and its group, rewrites the units, and moves the data root to `/var/lib/jumpgate-node` (a custom data directory stays where it is, and a data root that is or holds a mount point makes it ask);
- it keeps each unit enabled or disabled as it was, and starts again the ones that were running;
- it re-owns the data only where the new account does not already own it, and verifies the result at the end.

`--yes` is your consent to the renames; after that it asks only where it cannot choose safely. If processes of the old account are still running, it asks to stop them with SIGTERM, and asks again before any SIGKILL. The node is stopped while the job runs.

It refuses a box where another unit still runs as the old account (an eRPC gateway unit, for one), where other units use paths under the old data root, or that has both accounts or neither.

Until it has run on a box that still uses the old names, the box refuses box modules; see [Modules](modules.md). New setups use `jumpgate-node` from the start.

## When a job is stuck

- **Its controller is gone:** adopt it once its lease is stale.
- **No signed request can move it:** stop the gate, then run `sudo jumpgate agent job-abandon JOB --yes` on the box. It refuses while a step of the job is still running. It ends the job as failed, with a signed log entry saying a local operator abandoned it. Start the gate again afterwards, and look at the job's log and output before you start another one.
