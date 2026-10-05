# Sub-project 1: Foundations — signed intents, the agent, pairing

Date: 2026-10-02. Status: awaiting written-spec review.
Parent: `2026-10-02-controller-agent-architecture-design.md` (read it first; this
spec does not repeat its rationale).

## Goal

At the end of this sub-project an operator can, from the command line:

1. Pair a box: `jumpgate hosts add box-a --ssh root@203.0.113.7` installs the
   agent, confirms keys, and verifies a signed round trip.
2. Ask it things and act on it: `jumpgate status box-a`,
   `jumpgate disk box-a`, `jumpgate endpoints box-a`, `jumpgate firewall box-a`,
   `jumpgate logs box-a [-n 200]`, `jumpgate service box-a beacon restart`.
3. Do the same on the machine it is running on (`jumpgate hosts add local
   --local`), through the local socket.

Every one of those operations is a signed intent checked by the agent's policy
and answered with a signed receipt. No TUI yet (sub-project 3); no durable jobs
yet (sub-project 2). The web UI is unchanged except for the shared fixes listed
under "Shared-code changes".

## Out of scope

Durable jobs, decision points, approval-tier signers (browser page, hardware
wallets), the TUI, setup over intents, gateways/VPN/keys over intents,
validators. Approval-tier intents therefore do not exist yet; policy changes in
this sub-project happen only through bootstrap (root SSH).

## Step 0: retire the valve-node-app name

Done first, as its own commit, per "Naming" in the parent spec: module path,
`cmd/jumpgate`, `~/.jumpgate` with one-time migration, and the catalog accepting
both `valve-node-app` and `jumpgate-node` service users. **On-box names are not
migrated by pairing.** Units (`valve-node-app-exec.service` and friends),
`/var/lib/valve-node-app`, container names, the `valve-node-app` service user
and the `VALVE_*` environment placeholders are untouched by this sub-project;
pairing only adds the `jumpgate` tunnel user and the agent. Their migration is a
durable-job intent in sub-project 2 (`node.migrate-names`: stop units, rename
user and group, re-render units, start, verify). Paths below use the new names.

The `~/.jumpgate` migration runs at the top of `main()`, before any subcommand
(except the on-box `agent` ones), because most commands create `~/.jumpgate`
first (R24). If `~/.jumpgate` exists without a `config.json`, the legacy
entries are moved in one by one, `config.json` last; a jumpgate-created file is
never overwritten, and on a clash both copies are kept, the legacy one is named
on stderr and in the `MOVED` note, and stored paths keep pointing at it. Both
directories holding a `config.json` is still an error to resolve by hand.

## Components

New packages, each usable and testable alone:

| Package | Purpose | Depends on |
|---|---|---|
| `internal/eip712` | Encode and hash EIP-712 typed data for a fixed type set | `x/crypto/sha3` |
| `internal/signer` | `Signer` interface; key-file, OS-keychain and 1Password implementations; address recovery and verification | `eip712`, `decred/secp256k1` |
| `internal/intent` | Intent and receipt types, their EIP-712 schemas, envelope encoding, replay rules | `eip712`, `signer` |
| `internal/agent` | The agent: socket listener, peer-credential gating at `Accept`, policy, replay state, intent dispatch to `ops` | `intent`, `ops`, `executor` (local) |
| `internal/agentclient` | Controller side: dial (SSH tunnel or local socket), sign, send, verify receipt | `intent`, `signer`, `executor` |
| `internal/bootstrap` | Install and pair an agent over a privileged executor | `executor`, `agentclient`, `catalog` |
| `internal/daemon` | Single-instance lock, discovery file, detached start of the controller server | `config`, `filelock` |
| `internal/filelock` | Cross-platform advisory file locks (`flock` on unix, `LockFileEx` on Windows), shared by `config` and `daemon` | stdlib |
| `cmd/jumpgate` | Subcommands: `serve`, `stop`, `agent`, `hosts`, and the intent commands above | all of the above |

### `internal/eip712`

- Supports the EIP-712 primitive types we use: `address`, `uint64`, `uint256`,
  `bytes32`, `string`, `bool`, plus nested struct types. No arrays in v1 (none of
  our types need them); adding them later is additive.
- API: `TypedData{Types, PrimaryType, Domain, Message}`,
  `HashStruct`, `Digest(td) [32]byte` (= `keccak256(0x1901 ‖ domainSeparator ‖
  hashStruct(message))`).
- Tests: the EIP-712 specification's `Mail` example must produce its published
  domain separator, struct hash and digest. Every intent type gets a golden-digest
  test so an accidental schema change fails loudly.

### `internal/signer`

```go
type Signer interface {
    Address() Address
    SignTypedData(ctx context.Context, td eip712.TypedData) (Signature, error) // 65 bytes, r‖s‖v, v∈{27,28}
}
func Recover(td eip712.TypedData, sig Signature) (Address, error)
```

Implementations in this sub-project:

- **KeyFile** — a raw secp256k1 key in a 0600 file. Refuses to load a file
  readable by group or others. Used by the agent and by headless controllers.
- **Keychain** — macOS Keychain and Linux Secret Service. If no keychain
  service is available, it fails with an error naming the KeyFile alternative
  rather than silently falling back.
- **OnePassword** — reads the key with `op read <ref>` at startup and holds it
  in memory. Requires the `op` CLI to be signed in.

`jumpgate keys init [--store keychain|file|1password]` creates the controller
key; `jumpgate keys show` prints the address. Default store: keychain where
available, otherwise file.

Signatures are low-s normalized on signing and rejected on verify if high-s.

### `internal/intent`

EIP-712 domain: `{name: "jumpgate", version: "1"}`.

```
Intent(
  address agent, address controller, uint64 seq, bytes32 nonce,
  uint64 issuedAt, uint64 expiry, string kind, bytes32 payloadHash)

Receipt(
  address agent, bytes32 requestHash, uint64 seq,
  uint8 status, bytes32 resultHash)
```

- `payloadHash` is `keccak256` of the exact payload bytes sent. Payloads are
  JSON produced by `encoding/json` from typed Go structs. The receiver hashes the
  bytes it received before decoding, so no JSON canonicalization is needed.
- `requestHash` is the intent's EIP-712 digest.
- `status`: 0 ok, 1 rejected (policy, replay, expiry, validation), 2 failed
  (the operation ran and errored).
- Wire envelope (one per request):
  `{"intent": <Intent message>, "payload": <base64>, "sigs": ["0x…"]}`.
  The receipt envelope mirrors it with the agent's single signature.
- Kinds in this sub-project, all routine tier:

  | Kind | Payload | Agent calls |
  |---|---|---|
  | `status.read` | `{}` | `monitor`'s probes run once (factored out of its polling loop if they are not already callable): local heads, peers, syncing, disk %. Head lag against a reference is computed by the controller, which holds `RefRPCBase`: the server's reply to `status` carries an extra `refHead` field (omitted when unavailable) next to the agent's result |
  | `disk.read` | `{}` | `ops.DiskUsage`, `ops.FreeBytesAt` |
  | `endpoints.read` | `{"sshLogin": "…"}` | `ops.Endpoints` |
  | `firewall.read` | `{}` | `ops.FirewallChecklist` |
  | `logs.read` | `{"n": 200}` (max 2000) | `journalctl` tail of the node's units, classified with `logwatch`'s signatures |
  | `service.action` | `{"service": "exec"\|"beacon", "action": "start"\|"stop"\|"restart"}` | `ops.ServiceAction` |
  | `agent.info` | `{}` | agent version, address, plan versions, policy summary |

### Replay and expiry rules (agent side)

1. Reject unless `intent.agent` equals this agent's address.
2. Reject unless `now ∈ [issuedAt − 60s, expiry + 60s]` and
   `expiry − issuedAt ≤ 300s`.
3. Recover every signature; reject unless the policy's requirement for `kind`
   is met by enrolled signers.
4. Reject unless `seq` is greater than the stored last-seq for
   `intent.controller`, and the nonce has not been seen within the window.
5. **Persist the new last-seq (write, fsync, rename) before executing.** A crash
   after persisting loses one intent, which is safe. Executing before persisting
   would allow a replay.
6. Execute, then return a signed receipt. Rejections are signed too, so a
   controller can prove why it was refused.

The controller keeps its own next-seq per agent in the controller config, and
re-syncs from `agent.info` (which reports the last seq it accepted from that
controller) if it is rejected for staleness.

### `internal/agent`

- Runs as `jumpgate agent` under `jumpgate-agent.service`:
  - `User=root`, `RuntimeDirectory=jumpgate` (creates `/run/jumpgate`),
    `StateDirectory=jumpgate` (`/var/lib/jumpgate`).
  - `LoadCredential=agent.key:/var/lib/jumpgate/agent.key`.
  - Hardening that does not block its job: `NoNewPrivileges=yes`,
    `PrivateTmp=yes`, `ProtectHome=read-only`, `ProtectKernelTunables=yes`,
    `ProtectControlGroups=yes`, `RestrictSUIDSGID=yes`. It is not given
    `ProtectSystem=strict`, because it must manage units and data directories.
- Listens on `/run/jumpgate/agent.sock`, chowned `root:jumpgate`. The mode is
  `0660` unless `policy.json` `localUids` is non-empty, and then `0666` (R23):
  `connect(2)` needs write permission on the socket file, so at `0660` the
  kernel would refuse an enrolled local uid outside group `jumpgate` before the
  peer check below ever ran. That check is the authority either way, and every
  intent still needs an enrolled controller's signature; a remote-only box
  keeps the kernel's group check as defence in depth. `agent enroll
  --local-uid` re-applies the mode to a live socket, and bootstrap restarts the
  agent after enrolling.
  HTTP/1.1 over the unix listener: `POST /v1/intent` (envelope in, receipt out).
  No other routes.
- Peer identification with `SO_PEERCRED` (`LOCAL_PEERCRED` on macOS, for tests):
  the listener is wrapped so the check runs inside `Accept`, and a refused peer
  is closed before net/http reads a byte of it. Accepted: uid 0 (or the agent's
  own uid), uids listed in `policy.json` `localUids`, and members of group
  `jumpgate`. A remote controller arrives as the `jumpgate` user through sshd,
  so the same check covers both paths. A refusal is not an error: `Accept` moves
  on to the next connection.
- Request body capped at 1 MiB; one intent at a time per controller (a second
  concurrent request from the same controller is rejected, which also keeps seq
  ordering simple).
- On-box files:
  - `/etc/jumpgate/policy.json` (root 0600): enrolled signers
    `{address, tier, label}`, the kind → requirement table, and `localUids`
    (uids allowed on the socket without group membership).
  - `/etc/jumpgate/node.json` (root 0600): this box's `catalog.WireConfig`,
    written at pairing.
  - `/var/lib/jumpgate/agent.key` (root 0600) and
    `/var/lib/jumpgate/replay.json` (last-seq per controller plus recent nonces).
- The agent runs `ops` functions with `executor.NewLocal()`. All of
  `ops`'s GNU `df`/`du` assumptions now hold by construction, because the agent
  only runs on Linux.

### `internal/agentclient`

- `Dial(ctx, target)` returns a connection to the agent socket:
  - local target: `net.Dial("unix", "/run/jumpgate/agent.sock")`;
  - remote target: SSH as user `jumpgate` with the controller's transport key,
    then `client.Dial("unix", "/run/jumpgate/agent.sock")`, through an optional
    jump chain (each hop is a `ssh.Client` dialled through the previous one).
- `Do(ctx, kind, payload) (Receipt, []byte, error)`: build the intent with the
  next seq, a random nonce, `issuedAt = now`, `expiry = now + 120s`; sign with
  the controller signer; send; verify the receipt's signature recovers to the
  agent address recorded at pairing **and** that `requestHash` and `seq` match.
  A receipt failing either check is an error, never a result.
- Controller transport key: an ed25519 SSH key generated once per controller at
  `~/.jumpgate/ssh/jumpgate_ed25519` (0600), or an ssh-agent key if the
  operator configures one.

### `internal/bootstrap`

Runs over a privileged executor (root, or a user with passwordless sudo; the
existing SSH executor with the improvements below). Steps, each idempotent with
a verify check like `setup.Step`:

1. **Preflight** — Linux, systemd, and `/etc/ssh/sshd_config` includes
   `sshd_config.d/*.conf`; fail with a clear message if the include is missing
   rather than editing the main file.
2. **Upload agent binary** — pick `jumpgate-linux-<arch>` from
   `uname -m`. If the controller itself is that platform, use
   `os.Executable()`; otherwise use `~/.jumpgate/agents/` (populated by
   `scripts/build-agents.sh`, with a `SHA256SUMS` file). Upload to
   `/usr/local/lib/jumpgate/jumpgate.new`, verify the SHA-256 on the box, then
   rename into place.
3. **User and group** — system user `jumpgate`, shell `/usr/sbin/nologin`, no
   password; group `jumpgate`.
4. **sshd** — write `/etc/ssh/sshd_config.d/50-jumpgate.conf` (the `Match User
   jumpgate` block), run `sshd -t`; on failure remove the file and stop; on
   success reload sshd. Install `authorized_keys` (0600, owned by
   `jumpgate`) with the restricted controller transport key. The tunnel user's
   home is `/var/lib/jumpgate/home`, so the file is
   `/var/lib/jumpgate/home/.ssh/authorized_keys`.
5. **Agent identity** — `jumpgate agent init`, run on the box, generates the
   agent key if absent and prints its address. The controller records the address. Trust in this
   value comes from the host-key confirmation of the privileged session. If the
   key exists but `replay.json` does not, `init` refuses (and so pairing stops
   at this step) unless `--reset-replay` is given on the box: an empty record
   would silently reopen the replay window the running agent fails closed on.
6. **Policy and node config** — `jumpgate agent enroll --address … --tier
   routine --label …` on the box enrolls the controller's signing address in
   `policy.json` (adding to an existing file, never replacing it), and `node.json` is written
   from the target's `Wire` (if setup has run).
7. **Service** — install and start `jumpgate-agent.service`.
8. **Verify** — dial as `jumpgate` through the tunnel, send `agent.info`, check
   the signed receipt. Only then mark the target paired in `config.json`
   (`Target.Agent = {Address, PairedAt, Transport}`).

The output of `hosts add` ends with the security recommendation to disable root
SSH login, and states the break-glass path. Pairing a second controller to an
already-paired box is the same flow; step 6 appends a signer instead of
replacing the file.

`--local` runs the same steps through `executor.NewLocal()` with `sudo`, skips
step 4, and enrolls the invoking user's uid in `policy.json` `localUids`
(`jumpgate agent enroll --local-uid N`) instead of adding the user to group
`jumpgate`, so no re-login is needed for the new group to take effect. That
enrollment is what opens the socket to mode `0666` (see the agent section).

### `internal/daemon` and `jumpgate serve`

- Runtime directory: `~/.jumpgate/run/` (0700).
- `server.lock`: an exclusive `flock` held for the server's lifetime. A second
  `serve` fails with "already running, pid N".
- `server.json` (0600): `{pid, socket, httpAddr, token, version, startedAt}`,
  written after the listeners are up, removed on clean exit.
- `server.sock` (0600; the server is its own authentication boundary, so the
  session token is still required): the existing `/api` mux, also served on the unix
  socket. The HTTP listener (`--bind`, default `127.0.0.1:8799`) remains for the
  web UI.
- `jumpgate` subcommands that need the server find it with `server.json`, check
  that the lock is held and `GET /api/health` answers, and otherwise start
  `jumpgate serve --no-open` as a child in its own session (`Setsid` on unix;
  there is no `--detach` flag), output to `run/server.log` (0600) and wait up to 10 s for it.
- `jumpgate stop` calls `POST /api/shutdown` on the running server (found through
  `server.json`, authenticated with its token) and returns once the server
  answers 202; it never signals a pid, because a pid read from a file may by now
  belong to another process. The server stops accepting requests and waits for destructive operations in
  flight: `criticalOp` runs clear and wipe on a context their client cannot
  cancel, so a closed browser tab, a dropped connection or a shutdown does not
  leave a half-cleared data directory.
- The tray/web entry point takes the same lock and writes the same file, so the two
  binaries never run two servers for one user (R25). It also serves
  `server.sock` and holds the controller key as `serve` does, so CLI commands
  that find it work against it; a key that will not open is logged and those
  routes answer 503 rather than keeping the UI down. When another server
  already holds the lock, the app opens that server's URL from `server.json`
  (in the tray window when launched windowed, otherwise the browser) and exits
  instead of starting a second server; its own `--bind`/`--relay-bind` flags
  are then ignored, with a note on stderr.
- To let `jumpgate serve` serve the web UI, the `//go:embed all:web/dist`
  moves into a small package at `cmd/jumpgate/web`.

The intent commands in this sub-project (`status`, `disk`, …) go through the
server: `POST /api/targets/{id}/intent/{kind}`, which signs with the controller
key held by the server and returns the verified receipt and result. Keys are
therefore only ever loaded by the server process. Because this endpoint makes
the server sign, it must sit behind the Origin / `Sec-Fetch-Site` check from
sub-project 0; this sub-project does not ship before that check lands.

## Shared-code changes (benefit the web UI too)

1. **Config file lock.** `config.Update(fn func(*Config) error) (Config,
   error)` takes an exclusive `flock` on `config.json.lock`, loads, applies,
   saves, releases. `Server.updateConfig` calls it (keeping `cfgMu` for
   in-process ordering). `Load` takes a shared lock.
2. **Executor `WriteFile` over SSH** sends content on stdin
   (`umask 077; cat > tmp && chmod <mode> tmp && mv tmp path`), so secrets never
   appear in the remote command line, the file is never wider than its final
   mode, and size is no longer limited by argv.
3. **Executor cancel.** Each remote command runs in its own process group, and
   a wrapper reports that group's id as the first line of output before the
   command's own output (the wrapper must not fork away from the session; the
   plan settles its exact form). On context cancel the executor sends `SIGTERM`
   through the session `signal` request where sshd supports it, then
   `kill -TERM -<pgid>` on a fresh session, then `kill -KILL -<pgid>` after 5 s
   if anything is still alive. The container test proves no process from the
   group survives.
4. **SSH handshake deadline.** The TCP connection gets a deadline covering the
   whole handshake (10 s), cleared once the client is established. `NewSSH`
   takes a context.
5. **ssh-agent auth.** If `SSH_AUTH_SOCK` is set, its keys are offered before
   `KeyPath`. Passphrase-protected key files are supported only through the
   agent (documented).
6. **Host-key decider.** `SSHConfig.HostKey` is an `ssh.HostKeyCallback`, and it
   is required. `NewSSH` and `DialSSH` refuse a nil one with
   `executor.ErrNoHostKeyPolicy` before they connect, so trust-on-first-use is
   never a silent default. Implementations in `internal/executor`:
   - `TOFUHostKeyCallback` writes the target's `HostKeyFile`. Only the legacy
     web-UI executor passes it, and that executor (`server.legacySSHConfig`)
     chooses a policy for each hop explicitly. A hop whose key is in
     `confirmed_hosts` uses `Strict`, and so does the box itself once it is
     paired with an agent. Every other hop uses TOFU until sub-project 6.
   - `Strict(confirmedFile, opensshFiles...)`: accepts a host only if its key
     is in the confirmed-only store `~/.jumpgate/confirmed_hosts`
     (`config.ConfirmedHostsFile()`) or in the operator's OpenSSH
     `~/.ssh/known_hosts`. It never records anything: an unknown host is an
     `*UnknownHostError` the caller must put in front of a person. The OpenSSH
     files are always consulted, so a `@revoked` entry or a different key there
     is fatal even when the confirmed store matches. TOFU never writes the
     confirmed store and Strict never reads the TOFU file, so a key
     auto-accepted by the web UI cannot count as confirmed by the CLI. To keep
     that true, the server always sets a new target's `HostKeyFile` (and each
     jump hop's) to `~/.jumpgate/known_hosts` and ignores a client-supplied
     path. Every mismatch or revocation wraps `executor.ErrHostKeyMismatch`;
     strict dials also set `SSHConfig.HostKeyAlgorithms` to
     `KnownHostKeyAlgorithms(confirmedFile, opensshFiles...)`. One builder,
     `config.StrictHostKey()`, assembles `Strict` with
     `~/.jumpgate/confirmed_hosts` and `~/.ssh/known_hosts`. The CLI, the
     server's agent dials and the legacy path all use it.
   - `CaptureHostKey` + `RecordHostKey` (the CLI's confirm flow): capture
     connects far enough to read the key and hangs up before authenticating;
     the CLI prints the SHA256 fingerprint, a person confirms, and
     `RecordHostKey` appends the key to the confirmed store. `CaptureHostKey`
     needs `Jump.HostKey` when a jump is configured and refuses otherwise, and a
     strict dial applies its `HostKey` to the jump host too (never TOFU).
   `hosts add` dials with `Strict`, and on `UnknownHostError` captures,
   confirms and records, then retries. A mismatch is always a hard error.
7. **`DataDir` validation.** `catalog.ValidateDataDir`: absolute, already clean,
   at least two path components, no whitespace or control characters, and not
   one of `/bin /boot /dev /etc /home /lib /lib64 /opt /proc /root /run /sbin
   /srv /sys /tmp /usr /var /var/lib`. Called by the server's setup handler,
   `setup.Plan`, `ops.clearPaths`, and the agent when it loads `node.json`.
   Existing configs that fail it are reported as a config error rather than
   acted on.

## Error handling

- Every rejection carries a stable reason code (`wrong_agent`, `expired`,
  `clock_skew`, `bad_signature`, `unauthorized_kind`, `stale_seq`,
  `replayed_nonce`, `busy`, `invalid_payload`, `validation`) inside a signed
  receipt, and the CLI prints the code plus a one-line remedy (for example
  `clock_skew` reports both clocks).
- CLI exit codes: 0 ok; 1 failed, including a rejection, an `agent_http`
  refusal and any server error code the CLI does not know; 2 usage, which also
  covers `no_controller_key` and `not_paired` (the operator has to run a
  different command first); 3 `unreachable`; 4 security: `bad_receipt` (a
  reply not signed by the paired agent), `unknown_host` (nobody confirmed the
  box's SSH host key) and `host_key` (the presented key contradicts the
  confirmed store or the operator's OpenSSH known_hosts, is revoked there, or
  the box offers none of the key types on record). The two host-key codes come
  from `hosts add`'s own confirmation, the pairing login, the pairing verify
  step and every box command alike (`executor.UnknownHostError` and
  `executor.ErrHostKeyMismatch` survive the agent dial), and the CLI prefixes
  all three with `SECURITY`. Scripts can depend on these.
- Transport failures are distinct from rejections: "could not reach agent on
  box-a (ssh: …)" never looks like "box-a refused". Host-key failures are not
  transport failures: they never report as `unreachable`.
- Strict dials ask each host for the key types on record (the confirmed store
  plus the OpenSSH files; RSA as `rsa-sha2-512`/`rsa-sha2-256`) through
  `HostKeyAlgorithms`, and keep x/crypto's defaults for a host known nowhere,
  so a box known by its ed25519 key is not mistaken for a changed one because
  x/crypto prefers ECDSA.
- A receipt that fails verification is reported as a security error naming the
  expected and recovered addresses, and the result is discarded.
- Bootstrap failures name the step, the command's exit code and stderr tail,
  and what was left in place.

## Testing

- `eip712`: specification vectors plus golden digests for every type.
- `signer`: sign/recover round trip; high-s rejection; KeyFile permission
  refusal; keychain and 1Password tested behind interfaces with fakes; one real
  keychain test gated behind a build tag for manual runs.
- `intent` / `agent`: table tests for each replay rule (wrong agent, expired,
  skew, bad signature, unknown signer, stale seq, replayed nonce, oversized
  body, concurrent request; bad signature and busy are exercised at the
  `Handle` level); crash-safety test that the last-seq is persisted
  before execution (inject a failure between persist and execute); `ops`
  dispatch through the existing fake executor.
- `agentclient`: an in-process SSH server (`gliderlabs/ssh`, already a test
  dependency) with a `direct-streamlocal@openssh.com` channel handler registered
  through its `ChannelHandlers` map, forwarding to a temp-dir agent socket; tests
  for receipt verification failures (wrong agent key, mismatched requestHash,
  mismatched seq).
- `bootstrap`: each step against a fake executor (commands and verify checks),
  plus an end-to-end test in a systemd-enabled container (Debian 12 and Ubuntu
  24.04) run with `scripts/e2e-agent.sh` (needs Docker; `BASE=ubuntu:24.04` for the second distro), not in the default `go test ./...`.
- Executor fixes: `WriteFile` never puts content in the command string; cancel
  kills the remote process group (verified against the container); handshake
  deadline against a TCP listener that never speaks SSH; host-key deciders.
- `DataDir`: table of accepted and rejected paths, including the review's
  examples (`/`, `/var`, `data`, a path with a newline).
- `daemon`: second `serve` refused, naming the running pid; stale
  `server.json` with no lock holder is replaced; `stop` sends an authenticated
  `POST /api/shutdown` to the server found through `server.json` and returns on
  202 (it never signals a pid); the tray/web entry point takes the same lock
  and, when a server already holds it, finds that server instead of starting a
  second.
- Test isolation: every package whose tests reach `~/.jumpgate` points `HOME`
  at a throwaway directory in `TestMain`, so no test reads or writes the
  developer's real controller state.

## Done when

- The three goal flows above work against a fresh Debian 12 box and the local
  machine, and `go test ./...` plus `scripts/e2e-agent.sh` pass.
- A captured intent replayed to the same agent is rejected with `stale_seq`;
  the same intent sent to another agent is rejected with `wrong_agent`.
- The web UI behaves as before, apart from the shared-code fixes.
