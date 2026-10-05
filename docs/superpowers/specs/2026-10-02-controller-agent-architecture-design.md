# Jumpgate controller/agent architecture and the `jumpgate` TUI

Date: 2026-10-02. Status: design approved in conversation, awaiting written-spec
review. Umbrella spec: it records every cross-cutting decision. Each sub-project
in "Build order" gets its own spec and plan; the first one is
`2026-10-02-foundations-design.md`.

## Why

Operators run Jumpgate against fleets of boxes, often from a terminal or a jump
host where a browser is impractical. Today every operation is a shell command on
an SSH session held open by the controller: when the session drops, work is
orphaned or lost; a root SSH key is the only authority; and new host keys are
trusted silently. Validators are on the roadmap
(`docs/design/validator-capability.md`), and a leaked or replayed command there
costs a slashing, not an outage.

Success means:

- An operator can run the whole fleet from `jumpgate` in a terminal, without the
  web UI, and the web UI keeps working beside it.
- Long operations outlive the controller. Any controller can find them again and
  adopt them.
- Every action on a box is an explicit, signed, replay-proof intent that the box
  checks against its own policy. Destructive ones need a human-held key.

## Roles

One binary, `jumpgate`, plays two roles. A machine can hold either or both.

- **Controller.** Holds the fleet config, the operator's signing keys, and the
  UIs. It runs as a local server process (`jumpgate serve`) that the TUI and the
  web UI both talk to.
- **Agent.** Runs on a node box as `jumpgate agent`, a root systemd service. It
  owns the box: it executes intents, runs jobs, keeps the job log, and signs
  everything it reports.

A box you work from is both: its controller reaches its own agent over the local
socket, and other boxes over SSH. A single-box operator runs both roles on one
machine. The local path is still signed and policy-checked; there is no
"local = trusted" shortcut, because that would let any shell on the box bypass
the approval tier.

## Transport: SSH-tunneled unix socket

- The agent listens only on `/run/jumpgate/agent.sock`, `root:jumpgate`, mode
  `0660` (`0666` when local uids are enrolled, with the Accept-time peer gate
  as the authority; see the foundations spec). It opens no TCP port.
- Remote controllers reach it through an SSH `direct-streamlocal@openssh.com`
  channel (`ssh.Client.Dial("unix", path)` in `golang.org/x/crypto/ssh`), as a
  dedicated unprivileged `jumpgate` user that can do nothing else:
  - `authorized_keys`: `restrict,port-forwarding,permitopen="[/run/jumpgate/agent.sock]:*",command="/bin/false" <key>`
  - sshd drop-in: `Match User jumpgate` / `AllowTcpForwarding local` /
    `AllowStreamLocalForwarding local` / `PermitOpen [/run/jumpgate/agent.sock]:*`
    / `PermitTTY no`. Not `AllowTcpForwarding no`: OpenSSH checks
    direct-streamlocal against the same local-forward permissions, so that
    would refuse the socket as well; PermitOpen pins forwarding to the socket.
- Local controllers connect to the socket directly. The agent identifies them
  with `SO_PEERCRED` (uid 0 or a member of group `jumpgate`).
- Jump hosts are supported by chaining SSH clients (ProxyJump semantics). The
  SSH hop may itself ride the existing `jumpgate0` WireGuard overlay; nothing
  here depends on it.
- SSH is otherwise reduced to **bootstrap**: one privileged session installs the
  agent and pairs keys. After pairing, root SSH can be disabled, and the security
  checklist recommends it. Console or root SSH remains the documented break-glass
  path.

Two independent locks: the SSH key only reaches the socket; the intent signature
is what authorizes an action. Either key alone is useless.

## Intents, not commands

The agent never accepts shell. It accepts typed **intents**
(`ServiceAction{service, action}`, `Setup{wire, planVersion}`, `Resync{service}`,
…), turns each into a plan with its own copy of `internal/setup` and
`internal/ops`, and applies its own validation (for example `DataDir` rules)
before running anything. What is signed is therefore the decision itself, and
the privilege boundary is the intent list, which can be audited.

## Signing

- **One scheme everywhere:** secp256k1 over EIP-712 typed data. Every identity
  is an Ethereum address: the agent, each controller, each approval wallet.
- **Dependencies:** `golang.org/x/crypto/sha3` (Keccak, already present),
  `github.com/decred/dcrd/dcrec/secp256k1/v4`, and a hand-written EIP-712
  encoder for our fixed set of types, tested against the EIP's vectors. Not
  go-ethereum.
- **Domain:** `{name: "jumpgate", version: "1"}`. No `chainId`: wallets bind it
  to the selected network and these messages are not on-chain. A node's chain id
  is ordinary message data.
- **Every intent carries** `agent`, `controller`, `seq`, `nonce`, `issuedAt`,
  `expiry`, `kind`. The agent rejects `seq` not greater than the last it saw from
  that controller, rejects expired intents (lifetime at most 5 minutes, 60 s
  clock-skew allowance), and remembers nonces within the expiry window.
- **Routine intents** carry a `payloadHash`. **Approval-tier intents** carry
  readable fields so a wallet screen shows the decision, e.g.
  `Wipe{agent, host, service, chainId, dataDir, …}`.
- **Responses** are signed receipts over `{requestHash, agent, seq, status,
  resultHash}`, binding each answer to its request.

### Keys and tiers

| Key | Where it lives | Signs |
|---|---|---|
| Agent key | Generated on the box; `/var/lib/jumpgate/agent.key` (0600 root), loaded with systemd `LoadCredential` | Receipts, job-log entries, decision requests |
| Controller key (routine tier) | One per controller machine. OS keychain by default; root-only key file on headless boxes; 1Password (`op`) as an alternative. Never in `config.json` | Routine intents, automatically |
| Approval key (high tier) | Operator wallet: browser wallet or hardware wallet | Destructive and sensitive intents |

All signers implement one `Signer` interface (`Address()`,
`SignTypedData(ctx, td)`).

### Policy (on the agent)

- An authorized-signer list (address, tier, label) and a table mapping each
  intent kind to the signatures it needs.
- Defaults: read-only intents and start/stop/restart need routine. Setup,
  resync, wipe, key and policy changes need routine **and** approval. Validator
  intents are disabled until an approval key is enrolled.
- Changing the policy requires approval tier, so a stolen controller key cannot
  weaken it.
- Rotating or revoking a controller key is an approval-tier intent. A lost
  approval key is recovered by re-pairing through break-glass.

### Approval signers

- **Browser page.** The local server serves a signing page and the wallet signs
  with `eth_signTypedData_v4` (MetaMask, Rabby, Frame, and every hardware
  wallet behind them). The TUI shows the link and waits. On a headless box the
  page is reached through an SSH port-forward.
- **Native hardware signers:** Ledger and Trezor (USB HID, cgo), Keystone
  (air-gapped QR: request rendered in the terminal, response read back as a
  pasted UR string or through the browser page's camera), GridPlus Lattice
  (network API). The USB code is behind a build tag and ships in controller
  builds only; the agent stays pure Go.
- A hash-only signature and a clear-signed one are the same bytes, so "a human
  saw the fields" cannot be enforced. Jumpgate warns when the connected model
  can only display hashes (original Ledger Nano S, Trezor One).

## Durable jobs

- Each long operation runs as a transient systemd unit
  `jumpgate-job-<id>[-<step>]` started by the agent with `systemd-run`. It
  survives agent restarts and controller disconnects.
- A **manifest** on the box (`/var/lib/jumpgate/jobs/<id>.json`) records the
  job's kind, its **inputs** (for setup: wire config plus plan version, never
  rendered commands), the current step, a `driver` field, a version stamp, and a
  lease (owner and heartbeat).
- Output goes to the journal. Clients resume with `journalctl --after-cursor`.
- The agent keeps a signed, append-only log of every intent and step outcome per
  job, so an adopting controller can verify history instead of trusting it.
- **Autonomy:** the agent runs a job as far as it can. It stops only at a
  **decision point**, published as a signed question with options. The
  controller answers with a signed decision of the tier the policy requires.
  Losing the controller never stops the agent.
- **Adoption:** any paired controller lists jobs, reads the manifest and log, and
  either watches read-only (lease is fresh) or takes the lease over (lease is
  stale).
- **v1 driver is step-level:** the controller drives a multi-step plan; each step
  is its own unit. If no controller is present, the current step finishes and
  the plan waits at the next step boundary. Because setup steps' `Verify`
  already doubles as an "already done" check, adoption is "rerun the plan". This
  requires completion markers where `Verify` is too loose today (snapshot,
  recursive chown).
- **Later driver:** the agent drives the whole plan itself. The manifest format
  above is chosen so this needs no format change. The agent refuses to resume a
  job whose plan version it does not know.

## Controller server lifetime and discovery

- `jumpgate` looks for a running local server; if none, it starts
  `jumpgate serve` detached. Quitting the TUI leaves the server and its jobs
  running; `jumpgate stop` ends it.
- Discovery file (pid, socket, token, version) under the config directory,
  guarded by a lock held for the server's lifetime, so there is only ever one
  server per user. The existing web-app binary takes the same lock and writes the
  same file, so the TUI attaches to it too.
- `config.json` gets a file lock as a safety net; the single-server rule is the
  real guarantee of one writer.
- One composition root (`buildServer` in `cmd/jumpgate`) builds the server for
  both entry points. `serve` takes the app's relay, meter and key-admin flags,
  and each flag also reads a `JUMPGATE_*` environment variable, so an
  auto-started `serve` inherits the operator's relay through the environment.
  The discovery file also records the server's shape: the relay bind, the
  billing socket, metering and key admin. An app launch that attaches to a
  server with a different shape warns and names each difference. A recorded
  controller key that will not open is tolerated by both entry points: the box
  routes answer 503 `no_controller_key` with the reason.

## The TUI

Bubble Tea, Lip Gloss and Bubbles. The TUI holds no keys and does no work: it
renders server state and sends requests, so it can quit and reattach freely.

**Layout:** host list left, main view right, status bar with the selected host,
link state, and a fleet-wide ⚑ count of jobs needing a decision.

**Screens:**

1. **Fleet** — one row per box: host, link (agent / local socket / SSH-only /
   unreachable, with last-seen age), network, EL and CL sync, head lag, peers,
   disk bar (used, expected-size tick, free), firewall grade, jobs (running, ⚑).
   Sort, `/` filter, tags and grouping.
2. **Host detail** — tabs: Overview (live sync), Storage, Services, Endpoints,
   Security, Logs (with AI explain), Diagnostics, Jobs. Later: Setup, Gateways,
   VPN.
3. **Jobs inbox** — fleet-wide running, paused and needs-decision jobs, with the
   signed question and its options.
4. **Hosts** — add (bootstrap: address, user, key or ssh-agent, ProxyJump chain,
   fingerprint confirm, agent install, pairing), import `~/.ssh/config`, edit,
   retire (revokes pairing).
5. **Signers** — controller keys, hardware wallet enrollment, test signing, which
   tier each signer holds on which boxes.
6. **Settings / toggles** — stored by the server: visible columns per view,
   visible tabs per host, refresh interval per view, units (GB/GiB), compact or
   wide rows, show estimates (always labelled "estimate"), bell on ⚑.

**Keys:** arrows or `hjkl`, `enter`, `esc`, `x` actions, `:` command palette
(`:resync beacon box-a`), `?` help.

**Destructive actions** need a typed confirmation in the TUI (against mistakes)
and an approval-tier signature (for authority).

**Honest data:** every value shows its age; stale values are greyed, never
shown as current; a failed probe reads "unavailable", never 0; expected sizes
always carry their estimate label.

## Validators: constraints only

Not built in this programme. The protocol must not rule out, and must enforce
once built, the rules in `docs/design/validator-capability.md`:

- One key → one signer → one slashing-protection database. The agent owns that
  database and refuses a key another agent has a signed claim on.
- Keys move between boxes only with EIP-3076 interchange.
- Import, exit and move are approval-tier intents with hardware signing; never
  auto-signed.
- Keystores never transit the controller in plaintext and never land in
  `config.json`. Web3Signer remains the preferred remote-signer tier.

## Build order

| # | Sub-project | Contents | Depends on |
|---|---|---|---|
| 0 | Review fixes (own branch) | Relay/billing: WebSocket and beacon metering, idempotent settle, no caching of the fallback price. Quick wins: Origin check, `ReadHeaderTimeout`, Gemini key in errors, `.gitignore` sidecars | — |
| 1 | Foundations | EIP-712, signers (key file, keychain, 1Password), intents and replay rules, policy, agent on unix socket (SSH tunnel and `SO_PEERCRED` paths), bootstrap and pairing, detached server and discovery, config lock; executor fixes (secrets via stdin under `umask 077`, real cancel, handshake deadline, ssh-agent, host-key decider); `DataDir` validation; first intents (status, disk, endpoints, firewall, logs tail, start/stop/restart) | 0 (Origin check) |
| 2 | Durable jobs | systemd-run units, manifest, signed log, leases, decision points, adoption, completion markers, on-box rename migration (`node.migrate-names`) | 1 |
| 3 | TUI core + browser approvals | Fleet, host detail, jobs inbox, hosts, signers, toggles, `/api/fleet`, browser signing page | 1, 2 |
| 4 | Native hardware signers | Ledger, Trezor, Keystone, Lattice | 1 (3 for UI) |
| 5 | Setup wizard over intents | `Setup` as a durable job | 2, 3 |
| 6 | Full web-UI parity | Gateways, VPN (fixes the re-provision peer loss), customer keys; web UI moves onto agents; legacy SSH-executor path retired | 3, 5 |
| 7 | Validators | Per the constraints above | 4, 6 |

**Migration:** until sub-project 6 the web UI keeps using today's SSH executor.
That executor passes its host-key policy explicitly. Boxes that are confirmed or
paired use Strict, and only boxes that nobody has confirmed use TOFU. So
retiring TOFU means deleting the call sites that pass `TOFUHostKeyCallback`.
Unpaired boxes show as `SSH-only` in the fleet, so both front ends migrate
gradually.

## Naming

The product is Jumpgate and the binary is `jumpgate`. "valve-node-app" is a
stale name ("node" meant a blockchain node) and is retired as the **first step of
sub-project 1**, before any new package is written, so new code is born under
the new import path:

- Go module `github.com/valve-tech/valve-node-app` → `…/jumpgate`;
  `cmd/valve-node-app` → `cmd/jumpgate` (the web-app entry point becomes
  `jumpgate serve`; the tray build keeps its tag).
- Controller config directory `~/.valve-node-app` → `~/.jumpgate`, migrated
  once on first start (move, leave a pointer file, never copy secrets twice).
- On-box service user and group `valve-node-app` → `jumpgate-node`, along with
  the units, `/var/lib/valve-node-app`, container names and `VALVE_*` names, is
  migrated by a sub-project 2 durable-job intent (`node.migrate-names`), not by
  pairing; pairing in sub-project 1 leaves them untouched. The migration will: stop units, `usermod`/`groupmod` rename,
  re-render units, `chown` only if ownership is by name rather than uid, start
  units, verify. Boxes keep the old names until that intent runs; the catalog
  accepts both names during the transition.
- `jumpgate` (the SSH-only tunnel user) and `jumpgate-node` (runs the clients)
  are deliberately different accounts.
