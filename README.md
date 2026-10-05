# jumpgate

**jumpgate** (formerly valve-node-app) sets up and monitors an Ethereum, PulseChain, or PulseChain-v4
node — one binary, guided setup, sync monitoring, and AI-generated log
explanations, all behind a token-gated local web UI.

- **Guided setup** — walks you through installing and wiring an execution
  client + consensus client for the network of your choice.
- **Sync monitoring** — watches your node while it syncs and once it's live,
  surfacing peer count, block height, and sync status at a glance.
- **AI log explanations** — turns cryptic client log lines into plain-English
  explanations of what's happening and whether you need to act.

Supported networks:

- **Ethereum** mainnet
- **PulseChain**
- **PulseChain v4** (testnet)

## v0.2

v0.2 rounds out day-to-day node operation from the same UI: start, stop, and
restart each service independently, or clear a service's data directory and
kick off a fresh resync — gated behind a typed confirmation so it can't
happen by accident. A storage panel compares actual disk usage against
expected-size estimates per client and network, labeled with rough
sync-time expectations. An endpoints panel lists each service's local RPC/P2P
URLs with live reachability checks, plus an SSH tunnel hint for reaching a
remote target's ports from your own machine. A security section runs a
probe-backed firewall checklist against the target and only ever *suggests*
the commands to lock it down — it never runs anything on your behalf. RPC and
P2P ports are configurable per client instead of fixed at their defaults.

## v0.3 (unreleased)

v0.3 de-roots the node services: the execution and beacon clients now run
as a dedicated unprivileged system user (`valve-node-app`) under hardened
systemd units (`NoNewPrivileges`, `ProtectSystem=strict` with the data
directory carved out, private `/tmp` and devices). Setup itself still
requires root — it creates the user, writes units, and owns the data
directory to the service account. Existing installs migrate automatically:
re-run setup against the target and the units are rewritten, the data
directory re-owned, and the services restarted.

v0.3 also adds **network diagnostics**: a read-only troubleshooting ladder
(services → local RPC/API → p2p listeners → inbound/outbound reachability →
peers → sync → known journal error signatures) that runs check-by-check and
**stops at the first failure** — the last item in the report is where your
node's network stack breaks, with a copy-paste fix. Diagnostics run
**automatically** when an error signature appears in the journal or a
connection fails (service inactive, zero peers), rate-limited to one run
per 10 minutes, and on demand from the Diagnostics screen, which always
shows the latest report and what triggered it. In SSH mode the ladder also
dials the target's public p2p ports from your own machine, catching
hosting-provider firewalls an on-box check can't see.

Finally, the node's execution/beacon **RPC can bind to a host address** of
your choice (an "RPC bind address" field in the wizard's Advanced section),
defaulting to loopback as before. Set it to the box's **Tailscale** IP (or
another trusted overlay address) to reach the node's RPC from your own
machine over the tailnet — no SSH tunnel needed. The engine API stays
loopback-only regardless, and the security checklist grades the bind
(loopback and Tailscale pass, a LAN address warns, a public/all-interfaces
bind fails) — remember the RPC is unauthenticated, so only ever bind it to
a trusted, private network.

## Command line

The `jumpgate` binary also has a command line for managing boxes from a
terminal. Run `jumpgate` with no arguments in a terminal for the terminal home: status, a menu, and the command list (`jumpgate help`). `jumpgate open` opens the web app in your browser, starting the background server if needed; `jumpgate serve` runs the server in the foreground. The desktop launchers on each OS open a terminal running `jumpgate`. Flags such as `--bind`, or no terminal (a pipe, the app bundle), start the web app as before. The web app and `jumpgate serve` share one server per
user: if one is already running, launching the app opens it instead of
starting a second.

```bash
jumpgate keys init                      # create the controller signing key: --store keychain (default where macOS `security` or Linux `secret-tool` exists), file (a 0600 file) or 1password (--ref op://vault/item/field)
jumpgate hosts add box-a --ssh root@203.0.113.7   # pair a box (or: --local for this machine)
jumpgate status box-a                   # also: disk, endpoints, firewall
jumpgate logs box-a -n 200
jumpgate service box-a beacon restart   # exec|beacon, start|stop|restart
jumpgate serve                          # run the controller server (the other commands start it for you)
jumpgate stop                           # stop it, via its local API
jumpgate relay --relay-bind 127.0.0.1:8545 --billing-socket /run/jumpgate-billing/billing.sock --meter
                                        # run the metered RPC data plane on its own (token via JUMPGATE_RELAY_TOKEN_FILE)
```

### Selling RPC access: run `jumpgate relay`

`jumpgate relay` is the recommended way to serve keyed, metered RPC. It runs
the metered data plane as a process of its own, separate from the controller.
It never opens `config.json`, the controller key or an SSH executor. It also
never takes the controller server's lock or migrates its state. That keeps the
internet-facing proxy away from the key that controls your fleet.

The relay takes its settings only from flags and the environment:
`--relay-bind`, `--billing-socket`, `--erpc-url`, `--erpc-project` and
`--meter`, or the matching `JUMPGATE_*` variables. The relay token comes from
`JUMPGATE_RELAY_TOKEN_FILE` (preferred) or `JUMPGATE_RELAY_TOKEN`; see below.
Point Caddy at `--relay-bind`, and bind the relay to
loopback or the interface Caddy reaches, never to `0.0.0.0`.

For now, the controller (`jumpgate --relay-bind …` or `jumpgate serve
--relay-bind …`) can still serve the relay in-process. That mode is kept for
existing setups, but new deployments should use `jumpgate relay`.

The web app and `jumpgate serve` build the server the same way and take the
same server flags: `--bind`, `--relay-bind`, `--billing-socket`, `--erpc-url`,
`--erpc-project` and `--meter`. Each of these flags also reads an environment
variable (`JUMPGATE_RELAY_BIND`, `JUMPGATE_BILLING_SOCKET`, `JUMPGATE_ERPC_URL`,
`JUMPGATE_ERPC_PROJECT`, `JUMPGATE_METER`), and a flag on the command line wins
over its variable. A server that a CLI command starts for you inherits that
command's environment. So set these variables once, in your shell profile or
service unit, and every entry point builds the same relay and key admin.

The two credentials, the relay token and the admin token, never come from a
flag. Put each one in its own file, readable only by you (`chmod 600`), and
point to the file with `JUMPGATE_RELAY_TOKEN_FILE` and
`JUMPGATE_ADMIN_TOKEN_FILE`. The server reads the file once and keeps the token
in memory. On Linux and macOS it refuses, with a message naming the file and
the `chmod 600` that fixes it, a token file that:

- is a symlink, or is not a regular file;
- grants any permission to group or others;
- is owned by another user (unless the server runs as root). `JUMPGATE_RELAY_TOKEN` and `JUMPGATE_ADMIN_TOKEN` also work, but
don't put the token values themselves in a shell profile. Every program you
start from that shell would see them. Setting a token both ways is an error.
After reading them, the server removes these variables from its own
environment. Nothing it starts inherits them, including commands on a local
target and the keychain and 1Password helpers. If the app finds a server already running with different
options, it prints which options are not in effect. Run `jumpgate stop` and
launch again to apply them. The same applies after an upgrade. If the server
still running is a different jumpgate version, commands warn you, and you run
`jumpgate stop` to replace it with the new version.

When metering is on, the metered relay can refuse a call in three ways:

- `402 account is out of credits`: the customer has to top up.
- `403 account not provisioned`: the key's funding account does not exist in
  the billing store yet. The server log names the account to create.
- `503`: the credit ledger did not answer.

On a WebSocket these are JSON-RPC errors -32003, -32006 and -32004. A relay
started without `--meter` serves every valid key for free, and it logs a
warning at startup to say so.

If the controller key will not open, the server still starts. This can happen
when the keychain is locked or the key file is missing. The web UI keeps
working, and box commands fail with `no_controller_key` and the reason. Fix the
key store, then run `jumpgate stop` so the server restarts with the key.

`jumpgate hosts add` installs a small agent (`jumpgate-agent.service`) on the
box, a restricted `jumpgate` tunnel user, and an sshd drop-in that confines that
user to the agent's unix socket. Every command is a signed intent that the agent
checks against its policy and answers with a signed receipt. Host keys are
confirmed by you, by fingerprint, and remembered in `~/.jumpgate/confirmed_hosts`.
The web UI also checks a box strictly when it is paired, confirmed, or listed in
your `~/.ssh/known_hosts`: a changed key is refused, never re-learned. Only
boxes on no such record are still trusted on first use, in
`~/.jumpgate/known_hosts`.
Running `hosts add` again on an existing name re-pairs the address on record
(to finish an interrupted pairing or upgrade the agent); it refuses a different
`--ssh` address, so remove the host first to re-point it. Once pairing
succeeds, **disable root SSH login** on the box
(`PermitRootLogin no`); you will not need it again unless you have to repair the
agent.

Exit codes: 0 success; 1 the operation failed or was refused; 2 bad usage, or a
missing prerequisite (no controller key yet, box not paired); 3 box unreachable;
4 security failure (a host key nobody confirmed, a host key that changed since it
was confirmed, a reply not signed by the paired agent, or a controller key whose
address is not the one `config.json` records), whether it is found while pairing
or by any later box command. `jumpgate keys show` prints the address of the key
actually in the key store, and flags it if that address differs from the
recorded one. `jumpgate keys show --recorded` prints only the recorded address,
without opening the key, so it never prompts the keychain or 1Password.

**Linux over SSH:** the server jumpgate starts in the background survives closing the terminal. On distributions where systemd-logind kills a user's processes at logout (`KillUserProcesses=yes`), run `loginctl enable-linger $USER` once so it also survives the SSH session ending.

**Windows:** the background server needs Windows 10 version 1803 or Windows Server 2019 or later (for its local socket).

## Requirements

- The **target** being set up (the box that will run the execution + beacon
  clients) must be **Debian or Ubuntu Linux**.
- The **SSH user must be root** — setup writes systemd units under
  `/etc/systemd/system`, manages services via `systemctl`, and installs
  binaries to `/usr/local/bin`. In **local mode** (setting up the same
  machine jumpgate itself is running on), run jumpgate as root. Preflight
  checks this (`id -u`) and fails fast with a clear message if it isn't met.
- Node services (the execution and beacon clients) run as the dedicated
  unprivileged `valve-node-app` system user, which setup creates. (In v0.1–v0.2
  they ran as root; re-running setup migrates an existing install.)

## Quickstart

### Download a release

Grab the archive for your platform from the
[latest release](https://github.com/valve-tech/jumpgate/releases/latest),
extract it, and run the binary:

```bash
tar xzf jumpgate_<os>_<arch>.tar.gz   # the Windows archive is a .zip
./jumpgate
```

In a terminal, bare `jumpgate` opens the terminal home: status and a menu, where `o` opens the web app in your browser. `jumpgate serve` runs the server in the foreground and prints a local URL with a one-time session token:

```
http://127.0.0.1:8799/?token=<token>
```

Without a terminal (a pipe, the app bundle) or with flags such as `--bind`, the app behaves as before: it starts the server and opens it in your browser.

Pass `--bind` to change the listen address, or `--no-open` to skip opening a
browser automatically.

### Build from source

Requires Go 1.25+ and Node 22+.

```bash
git clone https://github.com/valve-tech/jumpgate.git
cd jumpgate
cd cmd/jumpgate/web && npm ci && npm run build && cd ../../..
go build -o jumpgate ./cmd/jumpgate
./jumpgate
```

## How it's built

jumpgate is a single Go binary with the web UI (Vite + TypeScript)
compiled to static assets and embedded directly into the binary via
`go:embed`. There's no separate frontend server and no external dependency
to run — just the binary.

The local server binds to `127.0.0.1` by default and requires a session
token for every request (via `Authorization: Bearer`, a cookie set from the
initial `?token=` link, or the query parameter itself), so nothing on your
machine can drive it without that token.

jumpgate itself always runs locally — the UI and API bind to your own
machine. What it sets up can be **local** (the same machine) or **remote
over SSH**: point a target at a `host:port` + root SSH credentials and
jumpgate drives the whole install/wire/start/handshake flow on that box
instead, so you can run jumpgate on a laptop while it provisions a
dedicated server. Both modes need root on the target for setup itself (see
Requirements above); the node services it installs run unprivileged.

## Contributing

The web UI (`cmd/jumpgate/web/`) has no end-to-end (Playwright) test suite
by design for v1 — the API layer it talks to (`internal/server`) is fully
covered by Go tests, and the UI itself is a thin, framework-free render
layer over that API. Verify UI changes with:

```bash
cd cmd/jumpgate/web && npm run build   # tsc --noEmit (strict) + vite build
go build ./...                            # confirms the rebuilt dist/ still embeds
```

then a manual smoke test: run `./jumpgate --no-open` against a scratch
`$HOME` and curl the printed token URL.

## Learn more

For a deeper guide to running your own RPC node, see
[learn.valve.city/rpc](https://learn.valve.city/rpc).

## License

[MIT](LICENSE)
