# Server install: one-line install, packages, the controller service, two roles

Date: 2026-10-17
Status: draft for review
Parent: `2026-10-02-controller-agent-architecture-design.md`
Sibling: `2026-10-17-encrypted-secret-store-design.md` (the vault this spec
relies on for the controller key), `2026-10-17-docs-system-design.md` (where
the operator pathway in §6 is written)
Written against: `8fef986`

## Goal

Jumpgate installs on a Linux server without an app store and runs there as a
service, driven through the TUI over SSH. A fleet has two kinds of box:

1. **The controller server**, the first box set up. It holds the controller
   signing key, encrypted at rest. It manages itself (paired with `--local`)
   and every box paired to it.
2. **Agent-only boxes**, every box after the first. They hold no controller
   key. They run the agent and trust the controller's verifying address, which
   is the existing agent model (`/etc/jumpgate/policy.json`).

## What exists today (grounding)

- goreleaser (`.goreleaser.yaml`) builds `jumpgate_linux_{amd64,arm64}.tar.gz`
  with CGO off and both agents embedded. `.github/workflows/release.yml` merges
  a single `checksums.txt` and attests every asset with
  `actions/attest-build-provenance` (`gh attestation verify <file> --repo
  jumpgate-tech/jumpgate-app`). There are no `nfpms:` yet.
- `packaging/linux/install.sh` is the per-user installer inside the desktop
  bundle (`~/.local`, manifest, `--uninstall`). It downloads nothing.
- Server discovery (`internal/daemon`) is per-user. `~/.jumpgate/run/` holds
  `server.lock`, `server.json` (0600: pid, socket, HTTP address, **session
  token**, version, shape) and `server.sock` (0600, `fsperm.MakePrivate`).
  `apiclient.Connect` calls `daemon.EnsureRunning`, which starts `jumpgate
  serve --no-open` detached. Every request, including requests over the socket,
  carries the bearer token (`server.go` auth middleware).
- `config.Dir()` is `$HOME/.jumpgate` (`os.UserHomeDir`).
- Login links: the `/login` redemption refuses non-loopback peers, the unix
  socket, and, on Linux, any peer uid other than the server's own (or root),
  read from `/proc/net/tcp` (`login.go`).
- The agent (`bootstrap/unit.go`) is `jumpgate-agent.service`, root, with
  binary `/usr/local/lib/jumpgate/jumpgate`, state `/var/lib/jumpgate`, socket
  `/run/jumpgate/agent.sock`. Bootstrap creates a **system user and group named
  `jumpgate`** (the SSH tunnel user). The agent's peer gate (`agent/listen.go`
  `peerAllowed`) admits root, any `LocalUIDs` uid, and **any member of group
  `jumpgate`**.
- `hosts add --local` (`pair_local.go`) reads `config.json` in the CLI process,
  runs `bootstrap.Run` under `sudo` with `LocalUID: os.Getuid()`, and hands the
  agent address to the server as `api.PairRequest{Installed: addr}`.
- `jumpgate service HOST …` is already taken: it is a box service action.

## Decisions

**D1. Two roles, chosen at install time.** `install.sh` and the packages know
three outcomes:

| Role | What is installed | Key |
|---|---|---|
| `--controller` | `jumpgate` binary, `jumpgate.service`, users and groups (D6) | controller key in the vault, encrypted |
| `--agent` | agent binary at `/usr/local/lib/jumpgate/jumpgate`, `jumpgate-agent.service` | none; trusts one controller address |
| none (CLI) | the binary only | per-user, as today |

With no role flag on a terminal, `install.sh` asks which one (default: CLI
only). With no role flag and no terminal, it installs the CLI only and prints
the two role commands. It never picks a server role silently.

**D2. One controller per fleet by default.** The policy already allows
several: `Policy.Signers` is a list, `AddSigner` appends, and seq and replay
are tracked per controller. A second controller is created by pairing a box
from it. That needs root on the box, as every pairing does. The TUI host view
shows "trusted by N controllers" whenever N > 1. Nothing else changes.

**D3. The controller key must be encrypted at rest on the controller server.**
On servers, `keys init` defaults to `--store vault` (sibling spec). The plain
`file` store is refused under `--controller` unless `--insecure-plain-key` is
given, and that prints a warning every time the service starts.
1Password stays available (`--store 1password`). Weigh it honestly: on an
unattended server it needs `OP_SERVICE_ACCOUNT_TOKEN`, which `secretenv`
deliberately passes through to `op`. That token is then as good as the key,
and it is stored in plaintext wherever the unit reads it from. So the vault is
the recommended default, and 1Password is for teams that already manage `op`
service accounts.

**D4. Unlock model for the service. Default: manual unlock, with the key kept
in memory only.** The service starts **locked**: `buildServer` already
tolerates a key that won't open (`sgn = nil`, box routes answer 503). That
becomes a named state, `locked`, separate from `no_controller_key`.
- **Unlocking:** `jumpgate unlock` (passphrase read from `/dev/tty`, or from
  stdin with `--stdin`), the TUI's unlock prompt (hidden input), or the web
  UI's unlock form (session-authenticated, loopback). The passphrase is never
  accepted on argv or in the environment.
- `jumpgate lock` drops the key. A restart re-locks it.
- **After a reboot:** agents keep running their current job step (durable
  jobs, v1 step-level driver). Each plan waits at its next step boundary until
  someone unlocks. The TUI header shows `LOCKED: jobs wait for unlock`, and so
  does `jumpgate home`.
- **Opt-in unattended unlock:** `jumpgate vault seal --tpm2` seals the vault's
  data key with `systemd-creds encrypt --with-key=tpm2`. The unit loads it with
  `LoadCredentialEncrypted=`, so the service unlocks itself at boot. A host-key
  seal (no TPM) protects only against copying the vault file alone. It needs
  `--allow-host-key` and says so.
- **What encryption does not protect against:** with the key unlocked, root
  can read it from the process. See D10.

**D5. The service is `jumpgate.service`, Linux only.**

```
[Service]
ExecStart=/usr/bin/jumpgate serve --no-open --system
User=jumpgate-ctl
Group=jumpgate-ctl
Environment=HOME=/var/lib/jumpgate-ctl
StateDirectory=jumpgate-ctl            # 0700
RuntimeDirectory=jumpgate-ctl          # /run/jumpgate-ctl, 0750, group set by --system
UMask=0077
LimitCORE=0
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
Restart=on-failure
```

- **State:** `HOME` makes `config.Dir()` resolve to
  `/var/lib/jumpgate-ctl/.jumpgate` with no code change to `config`.
- **Logs:** go to the journal (`journalctl -u jumpgate`). There is no
  `server.log`: that file exists only because `EnsureRunning` detaches.
- **Core dumps:** `LimitCORE=0`, plus `prctl(PR_SET_DUMPABLE, 0)` under
  `--system`, keep an unlocked key out of core files.
- **`ProtectHome=yes`:** is safe because the service keeps nothing under
  `/home`. Operator SSH keys named by `--key` are read by the CLI process,
  never by the service. This needs a check during implementation: some
  executor paths may read `KeyPath` server-side.

**D6. Names: `jumpgate-ctl` runs the service; `jumpgate-admin` may attach.**
- The name `jumpgate` is taken twice over: bootstrap's tunnel user, and the
  group whose members the agent's peer gate admits.
- Running the controller as `jumpgate`, or gating attach on group `jumpgate`,
  would let every operator, and the controller, reach the agent socket as a
  group member, bypassing the per-uid `LocalUIDs` enrollment.
- So the package creates system user and group `jumpgate-ctl`, plus group
  `jumpgate-admin` for operators. `/var/lib/jumpgate` stays the agent's alone.

**D7. Attach is a unix socket gated by SO_PEERCRED, not a shared token
file.**
- **Rejected alternative:** a group-readable `server.json`. It would hand the
  session token, which also signs in the web UI, to every group member as a
  file, and `readPrivateFile`/`CheckPrivate` refuse group bits by design.
- **Socket and admission:** `serve --system` listens on
  `/run/jumpgate-ctl/server.sock`, mode 0660, owned
  `jumpgate-ctl:jumpgate-admin`. Socket requests are authenticated by the
  peer's credentials: the server's uid, root, or a member of `jumpgate-admin`
  is admitted, without a bearer token. The agent's
  `peerCred`/`groupsOf` (`agent/peercred_linux.go`) moves to a shared
  `internal/peercred` package and is reused. The TCP listener keeps the token
  and cookie exactly as today.
- **Discovery file:** `/run/jumpgate-ctl/system.json` (0644) carries pid,
  socket, HTTP address, version, start time, the service uid and the lock
  state. It carries **no token**.
- **Lookup order:** `daemon.Find` looks at `system.json` first, then
  `~/.jumpgate/run/server.json`. `EnsureRunning` never starts a per-user
  server while a system server answers.
- **A user outside `jumpgate-admin`:** gets `EACCES` and a message: "a
  jumpgate service runs on this machine; ask an administrator to run `sudo
  usermod -aG jumpgate-admin $USER`, then log in again". There is no silent
  fallback to a second, per-user controller with a different key.
- **Per-user mode:** is unchanged on every OS.

**D8. Login links in system mode are bound to the requesting uid.**
`/api/login-code` over the socket records the requester's uid (from peer
credentials) with the code. Redemption admits a loopback peer whose
`/proc/net/tcp` owner is that uid, or root, instead of `selfUID`. Through `ssh
-L 8799:127.0.0.1:8799 server`, the forwarded connection is opened by the
operator's own sshd session process, so its owner is the operator: the
existing check works unchanged. The service cannot write the redirect file
into the operator's home, so in system mode `jumpgate open` behaves as
`--print`.

**D9. The controller server pairs itself (`--local`) as the service's uid.**
In system mode, `pairLocalForeground` must not `config.Load()` the caller's
own config:
- it takes the controller address and the service uid from the server, over
  the socket;
- it runs `bootstrap.Run` under the operator's `sudo` with
  `LocalUID: <jumpgate-ctl uid>`;
- the server then verifies `Installed` as today, which needs the key unlocked.

`install.sh --controller` and the package never pair by themselves. A pairing
needs a key and an unlock. Instead, `jumpgate setup server` walks the order:
init key, unlock, pair self, as the TUI's first-run screen.

**D10. Key custody warnings.**
- **What the warning says:** root on the controller server, with the key
  unlocked, can sign routine intents for every box that trusts it.
  Encryption protects the key at rest, in backups, and against disk theft.
  It does not protect it from live root.
- **Where it appears:**
  - `install.sh --controller`'s closing text;
  - `keys init --store vault` on a server;
  - the TUI unlock screen (once per unlock, one line);
  - the docs pathway (§6).
- **Pairing a remote box from a controller server:** shows the same line in
  the host-add confirmation.
- **The real bound on damage:** destructive kinds require an approval
  signature (the parent spec's approval tier). Shipping that is what turns
  "can sign for every box" into "can read and restart every box".

**D11. Self-only is an option, not the headline.** `--controller --self-only`
sets `controller.scope = "self"` in `config.json`.
- **Controller side:** the server refuses to pair any target that is not
  `--local`. This is a UX guard only, since a compromised controller ignores
  it.
- **Agent side:** the enforceable part. `agent enroll --scope local` (passed
  by `--local` pairing under self-only) records `scope: "local"` on the
  `SignerEntry`. `Authorise` refuses that signer unless the connection's peer
  uid is root or in `LocalUIDs`: a tunnelled connection arrives as the
  `jumpgate` user and is refused. The peer uid comes from SO_PEERCRED, so a
  remote party cannot forge it.
- **Other boxes:** no agent can tell that a key "should" be self-only. The
  guarantee is that a self-only key is enrolled in no other box's policy, and
  enrolling it anywhere needs root on that box.

**D12. Agent-only boxes: push by default, pull when the controller has no root
SSH.**
- **Push (default, existing):** `jumpgate hosts add NAME --ssh root@HOST`
  (or `--sudo`). Bootstrap uploads the embedded agent, installs the unit and
  enrolls the controller.
- **Pull (new):** on the controller, `jumpgate hosts invite NAME` prints a
  one-liner pinned to the controller's own version:

  ```
  curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent \
    --version v0.6.0 --controller 0xC0… --transport-key 'ssh-ed25519 AAAA… jumpgate-controller'
  ```

  - **On the box:** `install.sh --agent` installs the binary at
    `/usr/local/lib/jumpgate/jumpgate`, then runs a new
    `jumpgate agent setup --controller ADDR --transport-key KEY`. That
    command is `bootstrap.Run` over a root local executor, so every step is
    the same idempotent code as push.
  - **Back on the controller:** the box prints its agent address. The
    operator runs `jumpgate hosts add NAME --ssh jumpgate@HOST --installed
    0xA…`, reusing `PairRequest.Installed`: host-key confirmation, then a
    signed `agent.info` through the tunnel must answer from that address.
- **Upgrades:** a controller with root SSH re-pairs, and bootstrap's upload
  step already skips an identical binary. A pull-only box upgrades by
  re-running the pinned `install.sh --agent`. `agent init` never replaces the
  box identity.

**D13. `install.sh`: one function, verified, explicit about root.**
- **Source and publication:** `packaging/install/install.sh` (POSIX sh). It
  is a release asset and is copied to `https://jumpgate.app/install.sh` by
  the landing deploy.
- **Partial-download safety:** the whole body is `main() { … }` and the last
  line is `main "$@"`, so a truncated download defines nothing and runs
  nothing.
- **Arch and download:** `uname -m` maps to `amd64`/`arm64`; anything else
  stops with a message. It downloads
  `jumpgate_linux_<arch>.tar.gz` and `checksums.txt` from
  `github.com/jumpgate-tech/jumpgate-app/releases/download/<tag>/` into a
  `mktemp -d` that a `trap` removes. Downloads use `curl -fsSL --proto
  '=https' --tlsv1.2`, or `wget` with the same constraints.
- **`--version vX.Y.Z`:** pins the tag. The default resolves `latest` through
  the releases redirect, and the resolved tag is printed before downloading.
- **Checksum:** `sha256sum -c` (or `shasum -a 256`) against `checksums.txt`
  is always required.
- **Provenance:**
  - When `gh` is installed and authenticated, `gh attestation verify` runs on
    the archive. A failure stops the install.
  - When `gh` is missing or unauthenticated, it prints: "build provenance not
    checked (gh not available); checksum verified against the release's
    checksums.txt over HTTPS".
  - `--require-provenance` makes that case a failure.
- **Root and sudo:**
  - Installing to `/usr/local/bin` needs root. As root, it just installs.
  - As a non-root user, it prints the exact commands it will run with
    `sudo` and asks `[y/N]` on `/dev/tty`. `--yes` skips the question. With
    no tty and no `--yes`, it stops and prints the commands.
  - `--prefix ~/.local` installs the CLI role without root.
  - The server roles always need root.
- **Package-managed binary:** if `/usr/bin/jumpgate` belongs to dpkg or rpm,
  it refuses (`use apt upgrade`), so two binaries never shadow each other on
  `PATH`.
- **Testing knob:** `JUMPGATE_INSTALL_BASE_URL` overrides the download base.
  It is not secret, and checksum and provenance rules still apply.

**D14. apt and dnf repositories at `pkg.jumpgate.app`, on Cloudflare R2.**
- **Where it is hosted:**
  - `install.sh` and the landing page stay on Cloudflare Pages (jumpgate.app,
    direct upload).
  - The packages go in an R2 bucket behind the custom domain
    `pkg.jumpgate.app`. Pages caps files at 25 MiB, and a controller with two
    embedded agents is close to or over that.
  - GitHub Pages was rejected: it puts a second origin in the trust path, and
    it has its own size limits.
- **Building packages:** goreleaser gains `nfpms:` for `deb` and `rpm`,
  amd64 and arm64, package `jumpgate`, with:
  - `/usr/bin/jumpgate`;
  - `/usr/lib/systemd/system/jumpgate.service`;
  - `/usr/lib/sysusers.d/jumpgate.conf` (`jumpgate-ctl`, `jumpgate-admin`);
  - `postinst`: `systemd-sysusers`, then `daemon-reload`. It does not enable
    the service: the controller role is chosen with `jumpgate setup server`,
    which enables it.
- **The `packages` job:** a new job in `release.yml` runs after
  `provenance`. It downloads the `.deb`/`.rpm` from the release, then:
  1. `aptly` builds `dists/stable` and `createrepo_c` builds the rpm repo;
  2. it signs `InRelease`/`Release.gpg`, `repomd.xml.asc`, and each rpm
     (`rpmsign --addsign`);
  3. it syncs to R2 with an R2 API token scoped to that bucket.
- **Signing key:** an OpenPGP key used only for the repo. Its private half
  and passphrase are GitHub **environment** secrets (`PKG_SIGNING_KEY`,
  `PKG_SIGNING_PASSPHRASE`) on an environment `packages` that requires
  maintainer approval. It is imported into a `GNUPGHOME` on the runner's
  tmpfs and deleted at the end of the job. It is never in the repo, never in
  goreleaser, and never on the GitLab runners.
- **Public key:** the public half is committed at
  `packaging/repo/jumpgate-archive-keyring.asc` and published at
  `pkg.jumpgate.app/jumpgate.asc`. The docs pin its fingerprint, and apt
  sources use `signed-by=/usr/share/keyrings/jumpgate-archive-keyring.gpg`.
- **Rotation:** publish a new key alongside the old one for one release,
  then drop the old one.

**D15. Upgrades never re-lock a fleet by surprise.** The package's `postinst`
does **not** restart `jumpgate.service` while it is manual-unlock: a restart
would drop the key and stall every plan until someone unlocks. The running
server keeps its version. The next CLI or TUI attach shows the existing
`daemon.SkewWarning` ("a different jumpgate version … is running"). The TUI
offers "restart now (you will need to unlock)". When the vault is TPM-sealed
(D4), `postinst` runs `systemctl try-restart jumpgate.service`.

**D16. Coexistence on the controller server.**
- **The agent:** `jumpgate.service` (user `jumpgate-ctl`, state
  `/var/lib/jumpgate-ctl`, socket `/run/jumpgate-ctl/`) and
  `jumpgate-agent.service` (root, `/var/lib/jumpgate`, `/run/jumpgate/`)
  share no path.
- **Binaries:** the package's `/usr/bin/jumpgate` and the agent's
  `/usr/local/lib/jumpgate/jumpgate` are separate files. The package never
  owns the agent's path, so bootstrap's re-upload never fights dpkg. They come
  from the same release, as the parent spec's D1 requires.
- **Socket access:** the controller reaches the agent socket as an enrolled
  `LocalUIDs` uid (D9), which already turns the socket mode to 0666, with the
  peer gate as the authority.
- **The tunnel user:** `jumpgate` exists for remote controllers only. A
  self-paired controller server needs no sshd drop-in, and `--local`
  bootstrap skips that step today.

## Error handling

| Situation | Behaviour |
|---|---|
| Truncated `install.sh` | nothing runs (D13) |
| Checksum mismatch, provenance failure | stop; nothing is installed; the temp dir is removed |
| Unsupported arch or OS | stop, naming `uname -m` / `uname -s` and the supported list |
| `/usr/bin/jumpgate` owned by a package | refuse; point at `apt upgrade` / `dnf upgrade` |
| User not in `jumpgate-admin` | `EACCES` on the socket, then the `usermod` hint (D7); no second server |
| Service locked | box routes 503 `locked`; CLI prints `run jumpgate unlock`; TUI shows the banner |
| Wrong passphrase | `wrong_passphrase`, after the KDF's fixed delay; the passphrase is never logged |
| `hosts add --installed` address mismatch | refuse: the answering agent is not the one printed on the box |
| Remote pairing under `--self-only` | 409 `self_only`, naming `--self-only` and how to drop it |
| `systemd-creds` without a TPM | `vault seal --tpm2` refuses and names `--allow-host-key` and what it does not protect |
| `postinst` with the service manual-unlock | no restart; the skew warning on the next attach (D15) |

## Testing

Everything below extends the Docker harness in `scripts/e2e/Dockerfile` (a
systemd and sshd box, `BASE=debian:12` or `ubuntu:24.04`, users `alice` (a
sudoer) and `bob`).

- **`scripts/e2e-install.sh`.** A local "release" directory is served over
  HTTPS (a self-signed CA trusted in the container), and
  `JUMPGATE_INSTALL_BASE_URL` points at it.
  - curl-piping a truncated copy of the script does nothing;
  - a tampered archive fails the checksum;
  - `--require-provenance` without `gh` fails;
  - `--version` pins the tag;
  - non-root without a tty and without `--yes` stops before any `sudo`;
  - `--prefix` installs without root.
- **Service and attach (`scripts/e2e-service.sh`):**
  1. Install the package, then `jumpgate setup server` as alice, which:
     - creates the key in the vault;
     - unlocks with a passphrase piped to `--stdin`;
     - self-pairs.
  2. Then:
     - assert `status` answers;
     - bob (not in `jumpgate-admin`) gets the hint, and no per-user server
       appears under his home;
     - after `systemctl restart jumpgate`, `status` answers `locked`, and
       answers again after unlock;
     - with `--self-only`, a tunnelled intent from a second container is
       refused by the agent.
- **Login over a tunnel:** from a second container, alice runs `ssh -L`. A
  login code minted by alice redeems. A code minted by alice and redeemed
  through bob's tunnel is refused.
- **apt and dnf repos (`scripts/e2e-repo.sh`):**
  - A throwaway signing key is generated inside the test (never the real
    one). The repo is built with the same script the `packages` job runs and
    served over HTTP.
  - `apt install jumpgate` runs on `debian:12` and `ubuntu:24.04`. A
    `fedora` container covers dnf with `repo_gpgcheck=1`.
  - An upgrade to a second version does not restart a manual-unlock service.
  - Tampered `InRelease` and `repomd.xml` are refused.
- **Pull pairing:** `install.sh --agent` runs in box B. The controller in box
  A runs `hosts add --installed`, and a status intent round-trips. A wrong
  `--installed` address is refused.
- **Unit tests:**
  - `internal/peercred` admission;
  - `system.json` discovery order;
  - the `locked` state in `buildServer`;
  - `Authorise` with `scope: local`;
  - nfpm contents (paths, modes, sysusers).

## 6. Docs pathway: "Run jumpgate on a server"

This is an operator pathway in the docs system. Each step is an atomic step;
the scenes are TUI scenes unless marked otherwise.

| # | Step id | Covers | Scene |
|---|---|---|---|
| 1 | `server-choose-role` | controller server vs agent-only box; one controller per fleet | none (diagram, hand-made with sidecar) |
| 2 | `server-install` | `install.sh --controller`, or apt/dnf with the pinned key fingerprint | `tui install-controller-done` (terminal transcript) |
| 3 | `server-key-vault` | `jumpgate setup server`: the vault passphrase, the custody warning (D10) | `tui setup-server-passphrase` |
| 4 | `server-start-service` | `systemctl enable --now jumpgate`, `jumpgate-admin` membership | `tui service-locked-banner` |
| 5 | `server-unlock` | `jumpgate unlock`, what waits while locked, optional TPM seal | `tui unlock-prompt` |
| 6 | `server-pair-self` | `hosts add NAME --local` as the service | `tui hosts-self-paired` |
| 7 | `server-tui-over-ssh` | `ssh -t server jumpgate` | `tui fleet` |
| 8 | `server-web-tunnel` (optional) | `ssh -L 8799:127.0.0.1:8799`, `jumpgate open --print` | `app login-via-tunnel` |
| 9 | `server-add-agent-box` | push (`hosts add --ssh`) or pull (`hosts invite`, `--installed`) | `tui hosts-invite` |
| 10 | `server-backup-recover` | back up the vault; rotate by re-pairing (sibling spec) | none |

Steps 3, 6 and 9 `require` the existing `keys-init` and `pair-box` steps where
those apply, so "Show previous steps" works across pathways.

## Rollout order

1. The vault store and `locked` state (sibling spec). This is useful on its
   own for headless desktops.
2. `internal/peercred`, `serve --system`, `system.json`, discovery order,
   login-code uid binding.
3. `jumpgate.service`, sysusers, `jumpgate setup server`, `unlock`/`lock`,
   self-pair as the service uid.
4. `install.sh` (CLI and `--controller`), published as a release asset and
   on jumpgate.app.
5. nfpm packages, then the `packages` job and the R2 repo, after the signing
   key and the `packages` environment exist.
6. `install.sh --agent`, `agent setup`, `hosts invite`, `hosts add
   --installed`.
7. `--self-only` (agent `scope: local`).
8. Docs pathway steps and scenes. Each lands with the code it covers, as
   the docs system's staleness rule requires.

## Open questions

1. Should a user outside `jumpgate-admin` be offered an explicit per-user
   server (for example `JUMPGATE_USER_SERVER=1`)? Or is a second controller on
   a controller server always a mistake?
2. Should the controller auto-lock after idle? That conflicts with
   unattended plans. A "lock when no plan is running" option is a candidate.
3. Pull-only agent boxes cannot be upgraded by the controller. Should there
   be an approval-tier `agent.upgrade` intent?
4. Should the agent key (`/var/lib/jumpgate/agent.key`) also move to
   `LoadCredentialEncrypted=` with a TPM seal where available?
5. Should a `--self-only` controller be able to declare that scope to other
   agents, for example with a statement signed by its own agent that other
   agents refuse to enroll? This needs an approval-tier enroll first.
6. Should arm64 get its own dnf repo path, or one multi-arch repo?
7. Is `pkg.jumpgate.app` the final name? And should the R2 bucket live in
   the same Cloudflare account as Pages (valaxyblackdev)?
