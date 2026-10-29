# OpenTUI TUI: Capability Inventory

Date: 2026-10-28. Input to `docs/superpowers/plans/2026-10-28-opentui-tui.md`.

Scope: every user-facing capability in the CLI (`cmd/jumpgate`), the web UI
(`cmd/jumpgate/web/src`), the Go TUI (`internal/tui`) and the server API
(`internal/server`, routes registered in `Handler()` and the `register*Routes`
functions). Where a capability is reached is taken from the code, not the docs.

## How clients reach the server today

- **Discovery** (`internal/daemon`): `daemon.Find` first reads
  `/run/jumpgate-ctl/system.json` (the systemd service; world-readable, no
  token). If none, it reads `~/.jumpgate/run/server.json` (0600:
  `pid, socket, httpAddr, token, version, startedAt, shape`) and checks that
  `server.lock` is held and `/api/health` answers over the socket.
- **Transport**: HTTP over the unix socket (`daemon.Info.Client()`, base URL
  `http://jumpgate`), or TCP `httpAddr` (the browser's path).
- **Auth** (`server.authMiddleware`): on the service's admin socket, the peer's
  kernel credentials (root or `jumpgate-admin` member), no token. Otherwise
  `Authorization: Bearer <token>`, or the `jumpgate_token` cookie (unsafe
  methods also need same origin), or `?token=` (tray webview only).
- **Login codes**: `POST /api/login-code` (authenticated) mints a 60 s one-time
  code plus an owner-only redirect file; the browser redeems `GET /login?code=`
  for the cookie. The code never goes on argv. The TUI does not need login
  codes, because it holds the socket and token the same way the Go TUI does. It
  needs one only for "open in browser".
- **Streams**: SSE (`.../stream` routes) with reconnect (`apiclient.Stream`).
- **Version skew**: `GET /api/version` is compared with the client build
  (`apiclient.Skew`), and a mismatch is shown.
- **Billing**: the Rust store (`services/billing`, `/admin/keys`, `/admin/pricing`,
  `/admin/audit`) is reachable only by the server's relay (`internal/relay/admin.go`)
  over the billing socket with an admin token. The control plane exposes only
  the key routes (`/api/gateways/{gid}/keys`).

Legend for column 4: **root**: needs root or sudo on this machine. **pass**:
a passphrase (or other secret) typed on a TTY. **TTY**: needs the real terminal
for something other than a passphrase (an ssh session, sudo's prompt).
**CLI-only** means no API route exists today.

## Inventory

Groups: A. Connection, session, server process (9); B. Vault (13); C. Controller key, signers, approvals (7); D. Fleet and hosts (17); E. Nodes and services (16); F. Modules (6); G. Jobs, intents, plans (8); H. Gateways, RPC, keys, billing (17); I. VPN (6); J. Settings and prefs (4); K. Activity and logs (2); L. Setup and install (3).

| Grp | Capability | Where today | API route | root / pass / TTY | Notes |
|---|---|---|---|---|---|
| A | Find and attach to running server | CLI, Go TUI, tray | none (files `system.json`, `server.json`) | - | TUI reads the same files; never prints the token |
| A | Health / version / skew | CLI, Go TUI, web | `GET /api/health`, `GET /api/version` | - | |
| A | Start server if absent | CLI (`daemon.EnsureRunning`), bare `jumpgate` | **CLI-only** (spawns `jumpgate serve`) | - | shell out |
| A | Stop server | CLI `stop` | `POST /api/shutdown` | - | |
| A | Restart server (load new key) | Go TUI Signers `R` | `POST /api/restart` | - | |
| A | Open web UI in browser | CLI `open`, tray | `POST /api/login-code` | - | code goes in a redirect file, never argv |
| A | Run server foreground / relay only | CLI `serve`, `relay` | **CLI-only** | - | out of scope for the TUI (process modes) |
| A | Plain status screen | CLI `home` | reads `/api/fleet` | - | superseded by the TUI fleet screen |
| A | Update check | web UpdateBanner | `GET /api/update[?refresh=1]` | - | |
| B | Vault state | CLI, Go TUI, web VaultBanner | `GET /api/vault` | - | |
| B | Unlock | CLI `unlock`, Go TUI, web UnlockDialog | `POST /api/vault/unlock` | pass | backoff on wrong passphrase (`unlock_backoff.go`) |
| B | Lock | CLI `lock`, Go TUI, web | `POST /api/vault/lock` | - | |
| B | Idle lock (minutes) | Settings (web, Go TUI) | `GET/PUT /api/settings` `vaultIdleLockMinutes` | - | 0 off, up to 1440 |
| B | Put secret (relay-token, admin-token) | CLI `vault put` | `PUT /api/vault/secrets/{name}` | pass (secret value) | vault must be unlocked |
| B | Migrate plain secrets into vault | CLI `vault migrate` | partly: local file work, then `POST /api/vault/unlock` | pass | **CLI-only** for the file step |
| B | Init vault | CLI `vault init [--keystore]` | **CLI-only** | pass (new, twice) | local file under the vault lock |
| B | Backup to file | CLI `vault backup FILE` | **CLI-only** | - | copies the encrypted file |
| B | Restore from file | CLI `vault restore FILE` | **CLI-only** | pass (backup's) | |
| B | Add recovery code | CLI `vault add-recovery` | **CLI-only** | pass | prints a recovery code once (a secret) |
| B | Raise argon2id cost | CLI `vault rekdf` | **CLI-only** | pass | |
| B | Seal (user vault) | CLI `vault seal --tpm2/--allow-host-key` | **CLI-only** | pass, root (sudo for `systemd-creds`) | Linux only |
| B | Seal (service vault) | CLI `vault seal` against service | `POST /api/system/seal` + local root steps | pass, root | API returns the slot key; root installs the credential |
| C | Show controller address | CLI `keys show`, Go TUI Signers | `GET /api/controller` | - | CLI opens the real key locally and checks it against config; API reports what the server holds |
| C | Create controller key (user server) | CLI `keys init` | **CLI-only** | pass if `--store vault` | Go TUI Signers `K` shells out |
| C | Create controller key (service) | CLI `setup server`, `keys init` | `POST /api/system/init-key` | pass | |
| C | Export keystore v3 | CLI `keys export --keystore-v3 FILE` | **CLI-only** | pass (vault + new file passphrase) | writes key material to a file: keep CLI-only |
| C | Test signing on a box | Go TUI Signers `t` | `POST /api/fleet/{id}/check` | - | |
| C | Approve / refuse a job step | CLI `jobs decide`, Go TUI Jobs, drawer | `POST /api/jobs/{target}/{job}/decide` | - | "approvals" in the user's list |
| C | Adopt a job run elsewhere | CLI `jobs adopt`, Go TUI | `POST /api/jobs/{target}/{job}/adopt` | - | |
| D | Fleet list (live) | CLI `hosts list`, Go TUI Fleet, web Targets | `GET /api/fleet`, `GET /api/fleet/stream` | - | |
| D | Box detail | Go TUI host, web Machine | `GET /api/fleet/{id}`, `GET /api/targets` | - | |
| D | Add box over SSH: probe host key | CLI `hosts add`, Go TUI hostadd, web SSHForm | `POST /api/hostkeys/probe` | - | |
| D | Confirm host key | same | `POST /api/hostkeys/confirm` | - | user compares fingerprint |
| D | Add target | same | `POST /api/targets` | - | |
| D | Pair (stream) | same | `POST /api/targets/{id}/pair` (SSE reply) | - | |
| D | Pair this Linux machine (`--local`) | CLI `hosts add --local` | partly; sudo runs on the operator's terminal | root, TTY (sudo prompt) | **CLI-only** for the sudo step: shell out |
| D | Pairing info | CLI `hosts invite`, `--self-only` | `GET /api/pairing-info` | - | |
| D | Invite one-liner | CLI `hosts invite NAME` | `GET /api/pairing-info` (CLI formats) | - | TUI formats the same line |
| D | Upgrade a box's gate | CLI `hosts upgrade`, Go TUI | `POST /api/targets/{id}/agent/upgrade` | - | |
| D | Remove box | Go TUI, web | `DELETE /api/targets/{id}` | - | confirm |
| D | Recorded host keys | CLI `hosts forget-key` | `GET /api/hostkeys` | - | |
| D | Forget host key | CLI `hosts forget-key [--jump]` | `POST /api/hostkeys/forget` | - | |
| D | SSH into a box | CLI `ssh`, Go TUI (suspends, execs ssh) | `GET /api/fleet/{id}/ssh` then local `ssh` | TTY | TUI suspends its renderer and runs the argv |
| D | Gate setup on a box (pull pairing) | CLI `agent setup` (on the box) | **CLI-only, on the box** | root | not a controller TUI action; show the invite line instead |
| D | Rename legacy names | CLI `migrate-names` | `POST /api/jobs/{target}` (job `node.migrate-names`) | - | confirm, stops the node |
| D | Orphan cleanup | web OrphanBanner | `DELETE /api/orphans/{name}` | - | |
| E | Status (sync, peers) | CLI `status`, Go TUI, web | `GET /api/fleet/{id}/status/stream`, intent `status.read` | - | |
| E | Disk | CLI `disk`, Go TUI storage | `GET /api/fleet/{id}/disk`, `GET /api/targets/{id}/disk` | - | |
| E | Measure disk | Go TUI | `POST /api/fleet/{id}/disk/measure` | - | |
| E | du by path | web | `GET /api/targets/{id}/du` | - | |
| E | Endpoints | CLI `endpoints`, Go TUI, web | `GET /api/targets/{id}/endpoints` | - | URLs masked |
| E | Firewall checklist | CLI `firewall`, Go TUI security, web Security | `GET /api/targets/{id}/firewall` | - | |
| E | Logs (recent / live) | CLI `logs`, Go TUI, web | `GET /api/targets/{id}/logs`, `.../logs/stream` | - | sanitize terminal escapes |
| E | Service start/stop/restart | CLI `service`, Go TUI, web | `POST /api/targets/{id}/services/{svc}/{action}` | - | confirm |
| E | Clear service state | web | `POST /api/targets/{id}/services/{svc}/clear` | - | confirm |
| E | Containers list/status | web | `GET /api/targets/{id}/containers[/{svc}]` | - | |
| E | Container config get/put | web | `GET/PUT .../containers/{svc}/config` | - | |
| E | Container action / provision / wipe / reset | web | `POST .../containers/{svc}/{action}`, `/provision`, `/wipe`, `/reset` | - | wipe/reset confirm |
| E | Node setup wizard (stream) | web wizard | `POST /api/targets/{id}/setup`, `GET .../setup/stream` | - | |
| E | Monitor stream | web dashboard | `GET /api/targets/{id}/monitor/stream` | - | no trend graphs; numbers only |
| E | Diagnostics run / latest | web | `GET /api/targets/{id}/diagnostics[/latest]` | - | |
| E | Explain logs (AI) | Go TUI, web | `POST /api/targets/{id}/explain` | - | uses the provider key in settings |
| F | Module catalog | Go TUI, web | `GET /api/catalog` | - | |
| F | Box modules list / enable / configure | Go TUI host_modules, web | `GET /api/targets/{id}/modules`, `PUT /api/targets/{id}/modules/{module}` | - | covers `msgboard`, `el-bootnode`, `cl-bootnode`, `watch`, `private-send`, `checkpointz` |
| F | Watch module status | Go TUI host_modules_watch | same + fleet alerts | - | |
| F | Private send settings | same | same | - | |
| F | Checkpointz settings | same | same | - | |
| F | Gateway modules | web GatewayModules | `PUT /api/gateways/{gid}/modules/{module}` | - | |
| G | Jobs inbox (live) | CLI `jobs`, Go TUI Jobs, drawer | `GET /api/jobs`, `GET /api/jobs/stream` | - | |
| G | Running jobs | Go TUI drawer | `GET /api/jobs/running` | - | |
| G | Job detail (plan, steps) | CLI `jobs show`, Go TUI | `GET /api/jobs/{target}/{job}` | - | the "plan" is the job's step list |
| G | Job output (live) | CLI `jobs output` | `GET /api/jobs/{target}/{job}/output/stream` | - | |
| G | Start job | CLI, Go TUI | `POST /api/jobs/{target}` | - | |
| G | Decide / adopt | see C | | | |
| G | Generic intent | CLI `status/disk/endpoints/firewall` | `POST /api/targets/{id}/intent/{kind}` | - | kinds in `internal/intent/kinds.go` |
| G | Gate check | Go TUI | `POST /api/fleet/{id}/check` | - | |
| H | List / get gateway | Go TUI Gateways, web Rpc | `GET /api/gateways[/{gid}]` | - | |
| H | Create gateway | web | `POST /api/gateways` | - | |
| H | Delete gateway | web | `DELETE /api/gateways/{gid}` | - | confirm |
| H | Start/stop/restart | web | `POST /api/gateways/{gid}/{action}` | - | |
| H | Config | web | `PUT /api/gateways/{gid}/config` | - | |
| H | Provision / wipe | web | `POST .../provision`, `.../wipe` | - | wipe confirm |
| H | Trust cert / TLS verify | web | `POST .../trust-cert`, `GET .../tls/verify` | trust-cert may prompt the OS | |
| H | Capabilities | web Panel | `GET /api/gateways/{gid}/capabilities` | - | |
| H | Known set per chain | web | `GET /api/gateways/{gid}/knownset/{chainId}` | - | |
| H | Analytics | web Analytics | `GET /api/gateways/{gid}/analytics` | - | table only, no sparkline |
| H | Traffic | web | `GET /api/gateways/{gid}/traffic` | - | |
| H | Chain list | web AddNetworkModal | `GET /api/chainlist[/{chainId}]` | - | |
| H | Docker readiness / start (local) | web Panel | `GET /api/docker`, `POST /api/docker/start` | - | |
| H | Local host info | web | `GET /api/host` | - | |
| H | Customer API keys list / create / revoke | web KeysSection | `GET/POST /api/gateways/{gid}/keys`, `POST .../keys/{keyId}/revoke` | - | create reply holds the raw key once: show once, never log |
| H | Billing pricing / audit | none (Rust admin only) | **no API** (`/admin/pricing`, `/admin/audit` on the billing socket) | admin token | new route needed if wanted |
| H | Relay/admin token | CLI `vault put` | see B | pass | |
| I | VPN clients list / save / delete | web | `GET/POST /api/vpns`, `DELETE /api/vpns/{id}` | - | config may hold a private key: masked input |
| I | Client up / down / status | web | `POST /api/vpns/{id}/up\|down`, `GET .../status` | - | |
| I | VPN servers list / get / create / delete | web | `GET/POST /api/vpn-servers`, `GET/DELETE /api/vpn-servers/{id}` | - | |
| I | Server up / down / status | web | `POST .../up\|down`, `GET .../status` | - | |
| I | Server endpoint | web | `POST /api/vpn-servers/{id}/endpoint` | - | |
| I | Peers add / remove | web | `POST .../peers`, `POST .../peers/remove` | - | peer config shown once |
| J | Settings read / save | Go TUI, web Settings | `GET/PUT /api/settings` | - | includes AI provider key (secret: write-only field) |
| J | UI prefs (drawer etc.) | Go TUI | `GET/PUT /api/ui-prefs` | - | the new TUI uses the same `drawer` pref |
| J | Theme | web | local | - | n/a |
| J | Server bind / shape warnings | CLI | `server.json` shape | - | read-only |
| K | Activity log (right pane) | Go TUI actlog | built client-side from fleet/jobs streams | - | |
| K | Alerts | Go TUI alerts, tray | derived from `/api/fleet` | - | |
| L | Install systemd service | CLI `setup install-service` | **CLI-only** | root | |
| L | Setup server (vault key, unlock, pair self) | CLI `setup server [--self-only]` | **CLI-only** wrapper over `init-key`, unlock, pair | root (sudo), pass | |
| L | Gate health watch | CLI `agent watch` | n/a (runs on the box) | - | not a TUI action |

**Counts per group** (rows): A 9, B 13, C 7, D 17, E 16, F 6, G 8, H 17, I 6,
J 4, K 2, L 3. Total 108.

## Capabilities with no API

| Capability | Proposal |
|---|---|
| Start server if absent | shell out: `jumpgate serve` detached (what `EnsureRunning` does), or tell the user |
| `vault init` | shell out, TUI suspended, child owns the TTY |
| `vault migrate` (file step) | shell out |
| `vault backup FILE` / `vault restore FILE` | shell out |
| `vault add-recovery` | shell out (the code prints on the real terminal, never through the TUI) |
| `vault rekdf` | shell out |
| `vault seal` (user and service) | shell out (needs sudo) |
| `keys init` (user server) | shell out (the Go TUI already does) |
| `keys export --keystore-v3` | shell out |
| `hosts add --local` | shell out (sudo prompt) |
| `setup install-service`, `setup server` | shell out (root) |
| Billing pricing / audit | new API route, only if the user wants it in the TUI |
| `serve`, `relay`, `agent *` | out of scope |

**Recommendation: shell out.** Each of these needs root, a TTY passphrase
prompt, or local file work under the vault's file lock that the server must
not do for a client. The `jumpgate` binary already does them correctly and
already has tests. The TUI suspends its renderer, runs
`jumpgate <args>` with inherited stdio so the child reads the passphrase on
the real TTY, then resumes and re-reads `/api/vault`. The TUI never sees the
secret, so it cannot leak one. A new route is worth adding only for billing
pricing and audit, and only if the user wants them.
