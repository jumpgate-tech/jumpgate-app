# Platform support: the controller on macOS, Windows and Linux

Date: 2026-10-05. Status: written without a live review (the user was not
available); every open question was decided and recorded under "Decisions".
Parent: `2026-10-02-controller-agent-architecture-design.md`. Builds on
`2026-10-02-foundations-design.md` (sub-project 1), which is merged.
Input: the controller platform audit (`.superpowers/sdd/followups/platform-audit.md`,
audited at `60a121d`; this spec is written against `0bfebcc`). Item IDs
(B-1 … M-14) below are the audit's.

## Goal

The controller (the tray/web app, the `jumpgate` CLI and the local server it
starts) is fully supported, and tested in CI, on macOS, Windows and Linux.
"Supported" means, on each OS:

1. A released download can be double-clicked and does something useful: a
   desktop window, and a terminal entry point.
2. A released controller can pair a Linux box over SSH without a Go toolchain.
3. Secrets the controller writes (keys, config, the session token) are readable
   only by the user who owns them.
4. `go test ./...` runs green on that OS in CI, on every push.

The agent stays Linux-only by design. This spec does not make the agent run
anywhere else; it only fixes how a controller on any OS obtains and uploads it
(B-1, B-2), and how a Linux controller pairs itself (B-4).

## Out of scope

- The Bubble Tea TUI. It is sub-project 3 of the umbrella build order. This
  spec only builds the seam it plugs into (see "Terminal entry point").
- Code signing (Authenticode, Apple Developer ID and notarization), MSI/winget,
  .deb/.rpm and AppImage packages. Listed under "Deferred".
- Moving local-host web-UI features (overlay VPN, Docker gateway, trust store)
  onto agents. That is sub-project 6.
- Any change to the agent protocol, intents or policy.

## Binding decisions (given, not reopened)

**D1. Agent binaries are embedded in release builds.** Release builds embed the
static `linux/amd64` and `linux/arm64` agent binaries (`CGO_ENABLED=0`), with
SHA-256 checks, through `go:embed` behind a build tag. `~/.jumpgate/agents`
remains a developer override. A controller never uploads `os.Executable()`
unless it is itself a static Linux build of the matching arch. Agent and
controller are released from the same commit, so their versions always match.

**D2. Windows file protection is an owner-only DACL.** Wherever unix uses
`0600`/`0700`, Windows gets a protected DACL granting the current user (and
SYSTEM) full control and nobody else anything, using `golang.org/x/sys/windows`,
which is already a dependency. This covers key files, `config.json`,
`server.json`, `confirmed_hosts` and the run directory. `server.sock` gets the
same treatment, and a test asserts it. No new dependency.

**D3. Windows key store is Credential Manager.** Through advapi32
`CredWriteW`/`CredReadW`/`CredDeleteW` via `x/sys/windows` (`LazyDLL`). No new
third-party dependency. On headless Linux the Secret Service is not the default
when there is no D-Bus session; `secret-tool` runs under a timeout; the error
points at `--store file`.

**D4. No long-lived token on a command line.** The browser opener uses a
one-time, short-lived (60 s), single-use login code exchanged for the session
cookie.

**D5. Local pairing.** A root controller runs the bootstrap steps directly,
with no `sudo`. A non-root controller runs pairing from the foreground CLI
process, so `sudo` can prompt on the terminal, never from the detached daemon.
The error names the fix.

**D6. A terminal launcher bundle on every OS.** A release ships a
double-clickable bundle that opens jumpgate in a terminal:

- Windows: a console-subsystem `jumpgate.exe` (double-click opens a console or
  Windows Terminal) beside a GUI-subsystem tray build, `jumpgate-tray.exe`. This
  also fixes B-3: the GUI build gets a tray icon with a quit item, and errors
  surface in a dialog and a log file.
- macOS: an app that opens Terminal.app, or the user's default terminal,
  running `jumpgate`.
- Linux: a `.desktop` entry with `Terminal=true`, plus a tarball. AppImage was
  considered and rejected (D10).

There is no interactive TUI yet, so the terminal entry point is the
non-interactive CLI plus a small home screen: `jumpgate` with no arguments in a
terminal shows a status overview, a short menu and the command help, and stays
open until the user quits, so a double-click does not flash and close. The
launchers start plain `jumpgate`; when sub-project 3 lands, the TUI replaces
one function, `runTerminalHome`, and the launchers start it unchanged.

The user is considering pulling the TUI forward, ahead of sub-project 2. The
seam is therefore a single function with a fixed signature and no knowledge of
the launchers or of how it was started (see "Terminal entry point"); the TUI
can take it over on any branch without touching packaging, `main`, or CI.

**D7. CI.** `go test` runs on `windows-latest` and `macos-latest` as well as
Ubuntu, and there is a Linux arm64 cross-build. Every test that assumes `/tmp`,
`sh` or unix modes is fixed with build tags or `t.TempDir()`-style helpers. The
release workflow produces every bundle in D6 and the embedded-agent builds in
D1.

## Decisions (made while writing this spec)

Each line: the decision, why, and what it costs if it is wrong.

| # | Decision | Why | Cost if wrong |
|---|---|---|---|
| D8 | The terminal home is line-based: type a letter and press Enter. No raw-mode single-key input, so no `golang.org/x/term` dependency. | "Press a key" in raw mode needs x/term or per-OS console code for a screen the TUI replaces soon. | Slightly clunkier for one sub-project; swapping in raw mode later is local to one file. |
| D9 | Bare `jumpgate` (no arguments) with stdin and stdout on a terminal opens the terminal home instead of running the web server in the foreground. `jumpgate serve` (foreground server) and the new `jumpgate open` (start if needed, open the browser) cover the old behaviour. Bare `jumpgate` without a terminal (macOS .app, Windows GUI exe, a service) still runs the app. | The launchers must start something useful in a terminal, and the TUI will take this same entry point. | Anyone who scripted bare `jumpgate` in an interactive terminal sees a menu; README and the menu itself name `jumpgate serve`. |
| D10 | Linux ships a tarball with the desktop binary, two `.desktop` files, the icon and an `install.sh` (per-user, `~/.local`). No AppImage. | An AppImage must bundle WebKitGTK, whose helper processes use absolute paths; it is large and fragile. The distro packages are the supported way to get WebKitGTK. | Users who want one-file installs wait for a .deb (deferred); the tarball works on any distro with `libwebkit2gtk-4.1-0`. |
| D11 | The Linux desktop build links WebKitGTK 4.1 through a pkg-config shim (`webkit2gtk-4.0.pc` that `Requires: webkit2gtk-4.1`), built on ubuntu-24.04. No fork of `webview_go`. | `webview_go` hardcodes `pkg-config: webkit2gtk-4.0`, which Ubuntu 24.04 and Debian 13 dropped; 4.1 is API-compatible for what webview uses. | If 4.1 breaks a symbol webview needs, we vendor a patched `webview_go` (one `#cgo` line). The CI build catches it on the first run. |
| D12 | Linux gets no notification-area icon in this sub-project; the window is the surface. | A StatusNotifierItem needs D-Bus code (~500 lines) or a new dependency, and D2/D3 set a no-new-dependency bar. B-3 and I-8's blocking parts are fixed by the `.desktop` entry. | Linux users close the window to quit, as today. Revisit with the TUI work. |
| D13 | Embedded agents are gzip-compressed (`.gz`) and extracted on first use to `~/.jumpgate/agents-cache/<version>/`, verified against the embedded `SHA256SUMS` on every use. | Two static agents are ~2×25 MB raw, ~2×10 MB compressed; every release controller carries them. | ~0.3 s of decompression on the first pairing per version; a cache dir to clean up by hand after upgrades. |
| D14 | `~/.jumpgate/agents` is the agent source in a build without embedded agents. In a build with them it is used only when `JUMPGATE_DEV_AGENTS=1` is set, and then a `WARNING: using development agents from <dir>, not the agents built into this jumpgate` line goes on the pairing stream and stderr. In every build the directory, the binary and `SHA256SUMS` must be the user's own and private (`fsperm.CheckPrivate`), or they are refused. The pairing stream always names the source. (Amended by ruling P35.) | D1 calls it the developer override; an override that loses to the embedded copy is not one, but a forgotten or writable dev dir must not silently replace a release's agents. | A developer testing agents with a release build must set the variable. |
| D15 | Every build injects the git tag (`v`-prefixed, e.g. `v0.9.0`) as the version, for controller and agent alike. goreleaser switches from `{{.Version}}` to `{{.Tag}}`. | The desktop builds already use `git describe`; one format makes "versions always match" checkable by string equality. `updatecheck` already strips a leading `v`. | None known; the update check normalises both forms. |
| D16 | On Windows, `MakePrivate` grants only the owner and SYSTEM; `CheckPrivate` accepts ACEs for the owner, SYSTEM and BUILTIN\Administrators, and ignores inherit-only ACEs. | Administrators can take ownership of any file anyway, and default profile ACLs include them; refusing them would refuse every pre-existing key file. | An admin on a shared machine can read the key, as they always could. |
| D17 | Key-store tools (`security`, `secret-tool`, `op`) get a 2-minute timeout; the Secret Service liveness probe gets 3 seconds. | macOS and 1Password may show a GUI prompt a person must answer; 15 s would cut them off. The probe never prompts. | A wedged keyring delays server start by up to 2 minutes before a clear error, instead of forever. |
| D18 | Non-root local pairing: the CLI checks `sudo -n true`, falls back to `sudo -v` on the terminal, runs `bootstrap.Run` in-process with `executor.Sudo(executor.NewLocal())`, then asks the server to verify and record with `{"installed": "<agent address>"}`. The server never prompts and refuses a non-root local pair without `installed` (`local_needs_terminal`). | Keys stay in the server (only the address is needed for enrollment); sudo's tty-scoped timestamp covers the CLI's children because the local executor keeps the session (`Setpgid`, not `Setsid`). | On a system with `timestamp_type=ppid` every command re-prompts or fails; the error names `NOPASSWD` sudo or running as root. |
| D19 | `jumpgate agent …` is refused on Windows only; macOS keeps it because the agent's tests run there. | The agent is Linux-only in production; macOS is a supported development host for its tests. | A macOS user who runs `jumpgate agent run` gets a deep error rather than an early one. |
| D20 | `jumpgate-tray.exe` given any subcommand other than `serve`/`stop` shows a dialog pointing at `jumpgate.exe` and exits 2. No `AttachConsole`. | AttachConsole output interleaves with the parent shell's prompt and cannot read stdin reliably; a separate console exe is cleaner. | One extra exe in the zip, which D6 requires anyway. |
| D21 | The Windows tray icon is created at runtime from an embedded 32×32 PNG (`CreateIconFromResourceEx`), falling back to the stock application icon. The exe's own Explorer icon (a `.syso` resource) is deferred. | No resource compiler in the build; the notification icon is what users see while it runs. | The exe shows the generic icon in Explorer until the resource is added. |
| D22 | Windows-only tests run only on GitHub Actions. The plan's executor pushes its branch to run them, after one confirmation from the user. | There is no local Windows machine. | Iteration on Windows failures is a push-and-wait loop. |
| D23 | I-12's server side is fixed here (`local_unsupported` with a hint on Windows; Linux trust install detects `update-ca-trust` and offers a `sudo` command). Hiding local-host actions in the web UI is deferred to sub-project 6, which moves those features onto agents. | Sub-project 6 rewrites those screens; changing them twice is waste. | Until then, Windows users can click a local VPN/Docker action and get a clear refusal instead of not seeing it. |
| D24 | The macOS terminal launcher is a second bundle, `Jumpgate Terminal.app`, whose executable is a shell script that `open`s a bundled `jumpgate.command`. | `open` on a `.command` uses whatever terminal the user set as its handler (Terminal.app by default) and needs no Apple Events permission, unlike `osascript … tell application "Terminal"`. | Two apps in the macOS zip; a user with an unusual handler for `.command` files gets that app. |
| D25 | The release also builds a linux/arm64 desktop bundle on `ubuntu-24.04-arm`. | GitHub hosts arm64 Linux runners; Raspberry Pi and Ampere users get a desktop build for the cost of one matrix line. | One more release job to keep green. |
| D26 | The in-process webview keeps opening `?token=` URLs; only external openers (browser) use login codes. | A webview's URL never appears in any process's argv, and the tray's health poller reads the token from that URL. | None for D4's threat (argv); the token still sits in the webview's memory, as it must. |
| D27 | `scripts/build-agents.sh` stays the single way agents are built (developer dir, embed dir, release assets), with a `sha256sum`/`shasum` fallback and `GZIP=1` to also write `.gz` files. | One script means the embedded, published and developer agents are byte-identical for a version. | Bash on Windows runners comes from Git Bash, which every GitHub Windows image has. |
| D28 | Release asset names the landing page already links to stay the same: `Jumpgate-macos-<arch>.zip`, `jumpgate-windows-amd64.zip`, `jumpgate-linux-amd64.tar.gz`; only their contents change. New assets: `jumpgate-linux-arm64.tar.gz`, `jumpgate-linux-{amd64,arm64}`, `agents-SHA256SUMS`. | jumpgate.app links to these names; renaming them breaks every download button. | None; a rename can happen together with a landing-page change. |

## Deferred, with reasons

| Item | Deferred to | Reason |
|---|---|---|
| I-8 (Linux part): notification-area icon | Sub-project 3 or later | D12. The window works; no new dependency. |
| I-11 (later parts): MSI/winget, Authenticode, Apple notarization, .deb/.rpm | A release-engineering project | Needs certificates and accounts, not code. SmartScreen and Gatekeeper warnings remain, documented in the README. |
| I-12 (UI part): hide local-host actions on Windows | Sub-project 6 | D23. |
| M-1: `JUMPGATE_HOME` / `%LOCALAPPDATA%` | When a user asks | `~/.jumpgate` works on every OS; tests isolate HOME with `testutil.Home`. |
| M-3: `x/term` terminal detection, MSYS pty hint | Sub-project 3 | The TUI brings its own terminal handling; D8 avoids x/term now. |
| M-11: target release URL uses the controller's OS | Sub-project 5 | Dormant (every `ReleaseURL` returns ""), and setup moves onto the agent, where `runtime.GOOS` is the box's. |

Every other audit item is covered by a task (see "Coverage").

## Design

### 1. Portable tests and the CI matrix (B-5, I-9, D7)

A new `internal/testutil` package holds the helpers every package's tests use:

- `ShortTempDir(t) string`: a temp dir short enough for a unix socket path
  (`/tmp` on macOS for its 104-byte `sun_path`, `os.TempDir()` elsewhere),
  removed on cleanup.
- `Home(t) string`: a `ShortTempDir` set as both `HOME` and `USERPROFILE`.
- `RequirePOSIXShell(t)`: skips on Windows, or where `sh` is not on PATH.
- `AssertPrivate(t, path)`: owner-only on every OS (unix mode bits; Windows
  DACL through `fsperm.CheckPrivate`, once section 2 exists).
- `Loosen(t, path)`: makes a file readable by others (unix `chmod 0644`;
  Windows adds an `Everyone:R` ACE), for negative tests.

Every `os.MkdirTemp("/tmp", …)` becomes `ShortTempDir`/`Home`; every
`t.Setenv("HOME", …)` becomes `Home(t)`; tests that execute `sh` call
`RequirePOSIXShell`; agent-only test files are tagged `linux || darwin`; exact
mode asserts on secret files become `AssertPrivate`; mode asserts on
non-secret files (0640, 0660 …) apply only where the code under test is
POSIX-only and are skipped on Windows.

CI (`ci.yml`) gains a `go` job matrix on `ubuntu-24.04`, `macos-14` and
`windows-latest` (`go vet`, `go test ./...`, a `CGO_ENABLED=0` build), and a
`cross` job that vets and compiles tests for `linux/arm64`, `windows/amd64`,
`darwin/arm64` and builds `linux/arm64`. The web-UI dist check moves to its own
job. Later sections add `desktop-*` and `e2e` jobs.

### 2. `internal/fsperm`: owner-only files everywhere (I-1, I-2 part, M-4, M-5, M-6, M-14, D2)

```go
package fsperm
var ErrNotPrivate = errors.New("fsperm: other users can read or change this")
func MkdirPrivate(dir string) error            // MkdirAll, then restrict dir itself (tightens an existing dir)
func MakePrivate(path string) error            // restrict an existing file, dir or socket
func WriteFilePrivate(path string, data []byte) error // temp file in the same dir, restricted, fsync, Rename
func CheckPrivate(path string) error           // ErrNotPrivate (wrapped, naming who) when others have access
func CheckPrivateFile(f *os.File) error        // the same check on an open handle (no TOCTOU)
func Rename(oldpath, newpath string) error     // os.Rename; on Windows retried ~10× over 500 ms on sharing violations
```

- Unix: `0700` dirs, `0600` files, `Chmod` after create (so an existing loose
  dir is tightened, M-6); `CheckPrivate` refuses any group/other bit.
- Windows: the object is opened with `READ_CONTROL|WRITE_DAC` and
  `FILE_FLAG_OPEN_REPARSE_POINT|FILE_FLAG_BACKUP_SEMANTICS` (so it works on
  directories and on AF_UNIX socket reparse points, and never follows a link),
  and gets a protected DACL: current token user and SYSTEM, `GENERIC_ALL`,
  inherited by children for directories. `CheckPrivate` reads the DACL and
  refuses a null DACL or any allow ACE granting read, write, `WRITE_DAC` or
  `WRITE_OWNER` to a SID other than the owner, SYSTEM or Administrators (D16).

Call sites moved onto it: `config.Save` and `lockPath`; `daemon.RunDir`,
`Holder.Publish`, `server.log`; `server.ensureTransportKey`;
`executor.RecordHostKey`; `signer.GenerateKeyFile`, `LoadKeyFile` (now checks
on every OS, and refuses a symlink or reparse point on Windows, M-4);
`signer` 1Password template file (M-14); `server` install-id file;
`server.ServeUnix` (the socket, D2). Lock files are not secret and live in
private directories; `filelock` is unchanged.

### 3. Agent binaries: embedded, verified, never the wrong build (B-1, B-2, M-9, D1)

New package `internal/agentbin`:

```go
type Source string // "dev override ~/.jumpgate/agents", "embedded in this build", "this binary"
func Load(arch string) (content []byte, src Source, err error)
```

`Load` returns the verified bytes, never a path, and bootstrap uploads exactly
those bytes (`bootstrap.Options.AgentBinary` is `func(arch) ([]byte, error)`),
so nothing re-reads a file between the check and the upload.

Resolution order, per arch (`amd64`, `arm64`; anything else is an error naming
the two supported arches):

1. `~/.jumpgate/agents/jumpgate-linux-<arch>` if it exists, checked against
   `SHA256SUMS` beside it (D14). A present binary with no or wrong sums, or a
   directory or file that is not the user's own and private, is an error,
   never a fall-through. In a build with embedded agents this step runs only
   with `JUMPGATE_DEV_AGENTS=1`, and warns (ruling P35).
2. The embedded copy (build tag `embedagents`): `embedded/jumpgate-linux-<arch>.gz`
   and `embedded/SHA256SUMS`, decompressed and checked against the sum, cached at
   `~/.jumpgate/agents-cache/<version>/jumpgate-linux-<arch>` (D13).
3. `os.Executable()`, read once, only if `buildinfo.SelfIsStaticLinux()`
   (Linux, built without cgo, and the running executable is an `ET_EXEC` ELF
   with no `PT_INTERP`) and `runtime.GOARCH == arch`. The bytes read are
   checked again with `buildinfo.IsStaticELF`.
4. Otherwise an error: this build carries no agent; use a release build, or
   run `scripts/build-agents.sh` for development.

`internal/buildinfo` gains `SelfIsStaticLinux()`, set by a `cgo`/`!cgo` file
pair (a tray build always uses cgo). The bootstrap upload step logs the source.
The server's pair handler and the CLI use `agentbin.Reporting`, which wraps
`agentbin.Load` (ruling P8).

Build: `scripts/build-agents.sh [OUT]` builds both agents with
`CGO_ENABLED=0 -trimpath`, the version from `$VERSION` (D15), writes
`SHA256SUMS` with `sha256sum` or `shasum -a 256`, and with `GZIP=1` also writes
`.gz` copies. Release builds run it into `internal/agentbin/embedded/`
(gitignored) and build with `-tags embedagents`. The raw agents and their
`SHA256SUMS` (as `agents-SHA256SUMS`) are also release assets.

### 4. Key stores: Windows Credential Manager and a sane Linux default (I-3, I-4, D3)

- `signer.StoreWinCred = "wincred"`: a generic credential, target
  `jumpgate/<ref>`, persist `CRED_PERSIST_LOCAL_MACHINE`, the key's hex as the
  blob. Same rules as the other stores: exists-check fails closed, never
  replaces, verifies by reading back. `ref` uses the keychain name rule.
- `DefaultStore()`: Windows → `wincred`; macOS → `keychain` if `security`
  exists; Linux → `keychain` only if `secret-tool` exists **and** a D-Bus
  session is reachable (`DBUS_SESSION_BUS_ADDRESS` set, or
  `$XDG_RUNTIME_DIR/bus` exists) **and** `secret-tool search service jumpgate`
  answers within 3 s; otherwise `file`.
- Every key-store tool run is bounded (D17). A timeout reads: "secret-tool did
  not answer within 2m0s; the keyring may be locked with nothing to prompt you;
  unlock it, or use --store file".
- `keychain` on Linux without D-Bus fails with an error naming `--store file`.
- `keys init --store` help lists `wincred`.

### 5. Local pairing on Linux, refused elsewhere (B-4, I-10, M-8, D5)

- CLI `hosts add NAME --local` refuses on any OS but Linux before starting the
  daemon: "--local pairs this machine, and the jumpgate agent runs only on
  Linux; pair a Linux box with --ssh". The server's pair handler has the same
  guard (`local_unsupported`).
- Root (`os.Geteuid() == 0`): the CLI asks the server to pair, as today; the
  server runs the steps through `executor.NewLocal()` with no `sudo` wrapper.
- Non-root: the CLI requires a terminal on stdin (else it fails at once,
  naming the fix), checks `sudo -n true`, and if that fails runs `sudo -v`
  attached to the terminal; then runs `bootstrap.Run` itself (agent from
  `agentbin.Reporting`, controller address from `config.json`), and finally posts
  `{"installed": "<agent address>"}` to the pair endpoint, which verifies with
  a signed `agent.info` round trip and records the pairing (D18).
- The server, asked for a non-root local pair without `installed` (the web UI,
  or an old CLI), answers 409 `local_needs_terminal` with the hint "run
  `jumpgate hosts add NAME --local` in a terminal (it asks for your sudo
  password there), or run jumpgate as root".
- `jumpgate agent …` is refused on Windows (D19).
- `scripts/e2e-local.sh` runs a static controller inside the systemd test
  container: (a) as root with `sudo` absent, (b) as a `NOPASSWD` sudo user
  through `docker exec -t`. Both pair and get a signed `not_set_up` answer from
  `jumpgate status`. CI runs it and the existing `scripts/e2e-agent.sh`.

### 6. Windows daemon lifecycle and socket (I-2, I-7, M-13)

- `detach` on Windows: `DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP |
  CREATE_BREAKAWAY_FROM_JOB`; if the start fails with `ERROR_ACCESS_DENIED`
  (the job forbids breakaway), retry without breakaway.
- On every OS the daemon's working directory is the run dir, so it never pins
  the directory the CLI was started from.
- `ServeUnix` removes a stale socket left by a killed server on every OS (test
  with `SetUnlinkOnClose(false)`); on Windows a listen failure names the
  Windows 10 1803 / Server 2019 minimum.
- Linux: README documents `loginctl enable-linger` for servers that must
  outlive an SSH logout under `KillUserProcesses=yes`.
- A Windows CI smoke script exercises auto-start, survival past the starting
  job, stale-socket recovery after `Stop-Process -Force`, and `jumpgate stop`.

### 7. SSH agent on Windows, `~` in paths, system known_hosts (I-5, M-2, M-10)

- `executor.dialAgent` is split per OS. Windows: `SSH_AUTH_SOCK` empty means
  `\\.\pipe\openssh-ssh-agent`; a `\\.\pipe\` value is opened as a file (a pipe
  handle is the `io.ReadWriter` `agent.NewClient` needs); any other value is a
  unix socket (MSYS/Cygwin). `executor.AgentAvailable()` replaces the
  `SSH_AUTH_SOCK != ""` check in the add-target handler.
- `--key ~/…` (and `~\…` on Windows) is expanded by the CLI.
- `executor.OpenSSHKnownHosts(home)` returns the user's `known_hosts` plus the
  system file (`/etc/ssh/ssh_known_hosts`, or `%ProgramData%\ssh\ssh_known_hosts`)
  when it exists; both strict-check call sites use it.

### 8. Browser handoff with a one-time login code (I-6, M-7, D4, D26)

- The server keeps a map of login codes (random 128-bit, 60 s, single use).
  `GET /login?code=…` redeems one: sets the session cookie, redirects to `/`.
  An unknown, expired or reused code gets 401 and a page naming
  `jumpgate open`. `POST /api/login-code` (authenticated) mints one for another
  process.
- `openBrowser` opens `http://<addr>/login?code=<code>`; it waits on the opener
  (`go cmd.Wait()`), and when no opener exists prints the link with its 60 s
  validity.
- The app opens the browser only after its listener accepts connections.
- New `jumpgate open`: start the server if needed, mint a code, open the browser.

### 9. Terminal entry point (D6, D8, D9)

```go
// runTerminalHome is what a person sees when they start `jumpgate` with no
// arguments in a terminal: the launchers on every OS run exactly that. The
// TUI (sub-project 3) replaces this function; nothing else changes.
func runTerminalHome(ctx context.Context, in io.Reader, out io.Writer) int
```

- `main` calls it when `len(os.Args) == 1`, stdin and stdout are terminals and
  the process is not inside a macOS .app bundle.
- It prints an overview (version; server running with pid and address, or not;
  controller key address and store, or how to make one; machines and how many
  are paired) and a menu: `[o]` open the web app (`jumpgate open`), `[s]`
  refresh status, `[h]` command help, `[q]` quit. It loops until `q` or end of
  input, so a double-clicked window stays open.
- `jumpgate help` prints the same command help.
- Errors before the home screen (for example a failed `~/.jumpgate`
  migration) wait for Enter when the process owns its window: on Windows when
  the console's process list holds only this process; elsewhere when the
  launcher set `JUMPGATE_LAUNCHER=1`.

**TUI seam contract.** The TUI must only (1) keep this signature, (2) return
an exit code, (3) leave `main`'s launch detection, the launchers and CI alone.
If the TUI is pulled forward, its branch replaces the body of
`runTerminalHome` and moves the home's overview logic into a TUI screen; no
other task in this plan depends on the home's text.

### 10. Windows desktop build (B-3, I-8 Windows, I-11 Windows, D6, D20, D21)

- `jumpgate-tray.exe` (`-tags tray`, `-H windowsgui`) launched with no
  arguments and no console opens the desktop window (`guiLaunch`).
- Under the GUI build, `log` output goes to `~/.jumpgate/run/app.log`, and a
  fatal error appends to that log and shows a `MessageBoxW`. A webview that
  cannot be created reports the WebView2 runtime download link.
- A notification-area icon (`Shell_NotifyIconW`, its own message-only window
  on a locked OS thread): left click shows the window, right click offers
  "Open Jumpgate" and "Quit Jumpgate"; the tooltip shows the health state. Quit
  cancels the app context, which closes the window and stops the server.
- Any subcommand other than `serve`/`stop` in the GUI exe shows a dialog
  pointing at `jumpgate.exe` (D20).
- `scripts/package-windows.sh` builds `jumpgate.exe` (console, `CGO_ENABLED=0`)
  and `jumpgate-tray.exe`, both with `embedagents`, into
  `jumpgate-windows-amd64.zip` with a `README.txt`.

### 11. Linux desktop build (B-3 Linux, I-8 Linux, I-11 Linux, D10, D11, D12)

- Built on ubuntu-24.04 against WebKitGTK 4.1 through the pkg-config shim.
- `scripts/package-linux.sh` produces `jumpgate-linux-<arch>.tar.gz`:
  `jumpgate`, `jumpgate.desktop` (`Terminal=true`, runs
  `env JUMPGATE_LAUNCHER=1 jumpgate`), `jumpgate-window.desktop` (runs
  `jumpgate --tray`), `jumpgate.svg`, `install.sh`, `README.txt`.
- `install.sh` installs into `${PREFIX:-$HOME/.local}` and writes the absolute
  binary path into both `.desktop` files.

### 12. macOS terminal launcher (D6, D24)

`cmd/jumpgate/build-macos-app.sh` also builds `Jumpgate Terminal.app`:
`Contents/MacOS/jumpgate-terminal` (a shell script) runs
`open <bundle>/Contents/Resources/jumpgate.command`, which sets
`JUMPGATE_LAUNCHER=1` and execs the bundled `jumpgate`. Both apps are ad-hoc
signed and zipped together.

### 13. Local-host features (I-12 server part, M-12, D23)

- A local target on a machine without a POSIX shell gets 409
  `local_unsupported` with the hint "this computer runs Windows; run this on a
  Linux machine you added with --ssh" from the VPN and host-executor paths,
  instead of a 500.
- Trust store: Windows `certutil` quotes the path with `"` (the path validator
  already forbids `"`); Linux uses `update-ca-certificates` or `update-ca-trust`,
  whichever the box has, and `ManualCommand` is the `sudo sh -c '…'` form.
- `jgFile` fails when the home directory cannot be resolved instead of writing
  into the working directory.

### 14. Release workflow and docs (D1, D6, D7, D15, D25)

- goreleaser: `before` builds the embedded agents; builds use
  `tags: [embedagents]` and `{{.Tag}}`; the raw agents and their sums are
  release assets.
- Desktop jobs: macOS arm64 and amd64 (both apps), Windows amd64 (two exes),
  Linux amd64 and arm64 (tarball). Every desktop job builds embedded agents with
  the same version string.
- The unified `checksums.txt` covers all of it.
- README: platform table, minimum Windows version, key stores per OS,
  launchers, `jumpgate open`, `loginctl enable-linger`, SmartScreen/Gatekeeper
  notes.

## Testing realism

**Mac** = this macOS dev box. **Linux** = a Docker container on colima
(`docker run --rm -v "$PWD":/src -w /src golang:1.25 …`, or the systemd image in
`scripts/e2e`). **Win** = GitHub Actions `windows-latest` only. Every task's
tests also run in CI on the OS they target.

| Section | Mac | Linux (colima) | Win (Actions only) |
|---|---|---|---|
| 1 Tests/CI | `go test ./...` | `go test ./...` in golang:1.25 | the `go` job on windows-latest goes green (acceptance gate) |
| 2 fsperm | unix tests, tightening, loose-file refusal | same, as non-root and root | DACL set/check, Everyone:R refusal, socket DACL |
| 3 agentbin | resolution order with fakes; `-tags embedagents` with real agents; ELF arch/static checks; a release-style controller pairs the e2e container with no agents dir | `SelfIsStaticLinux` true for `CGO_ENABLED=0`, never wrong for cgo; SSH e2e | `-tags embedagents` extraction test on Windows |
| 4 key stores | seam tests for every OS's default | real Secret Service under `dbus-run-session`; no-D-Bus default is file | real Credential Manager create/read/refuse/delete |
| 5 local pairing | handler and CLI guards with seams | `scripts/e2e-local.sh` (root without sudo; NOPASSWD user) | CLI refuses `--local` before the daemon starts |
| 6 daemon | `cmd.Dir`, stale socket | same | `scripts/windows-smoke.ps1` (job breakaway, Stop-Process recovery, stop) |
| 7 ssh agent | `~` expansion, known_hosts list, unix agent | same | named-pipe agent with the OpenSSH agent service |
| 8 login code | single use, expiry, no token in argv | fake `xdg-open` records argv | unit tests |
| 9 terminal home | menu loop, EOF, overview with fakes | same | `launchedStandalone` compile and unit test |
| 10 Windows GUI | `go vet` with `GOOS=windows` | — | build both exes, launch GUI exe with no args, health + app.log, stop |
| 11 Linux desktop | — | build with the 4.1 shim, `xvfb-run` window smoke, `desktop-file-validate`, install.sh | — |
| 12 macOS launcher | build both apps, lint plists, run the launcher with a fake `open` | — | — |
| 13 local-host | unit tests per OS | trust command on Debian and Fedora images | unit test for `local_unsupported` |
| 14 release | `goreleaser release --snapshot --clean` if installed | — | `workflow_dispatch` run produces every artifact |

Manual checks that cannot be automated: the Windows notification icon and its
menu on a real desktop, the sudo password prompt during a non-root local
pairing, and double-clicking each launcher on a real desktop of each OS.

## Coverage

| Audit item | Section / plan task |
|---|---|
| B-1 agent sourcing | 3 / Task 3 |
| B-2 controller uploads itself | 3 / Task 3 |
| B-3 Windows desktop launch (and Linux) | 10, 11 / Tasks 10, 11 |
| B-4 local pairing | 5 / Task 5 |
| B-5 Windows tests and CI | 1 / Task 1 |
| I-1 secret file perms | 2 / Task 2 |
| I-2 server.sock on Windows | 2, 6 / Tasks 2, 6 |
| I-3 Windows key store | 4 / Task 4 |
| I-4 headless Secret Service | 4 / Task 4 |
| I-5 SSH agent on Windows | 7 / Task 7 |
| I-6 token in argv | 8 / Task 8 |
| I-7 daemon robustness | 6 / Task 6 |
| I-8 tray on Windows/Linux | 10, 11 / Tasks 10, 11 (Linux icon deferred, D12) |
| I-9 CI desktop builds | 10, 11, 12, 14 / Tasks 10, 11, 12, 14 |
| I-10 `--local` off Linux | 5 / Task 5 |
| I-11 packaging | 10, 11, 12 / Tasks 10, 11, 12 (installers deferred) |
| I-12 local-host features | 13 / Task 13 (UI part deferred, D23) |
| M-2, M-10 | Task 7 |
| M-4, M-5, M-6, M-14 | Task 2 |
| M-7 | Task 8 |
| M-8 | Task 5 |
| M-9 | Task 3 |
| M-12 | Task 13 |
| M-13 | Task 6 |
| M-1, M-3, M-11 | Deferred (above) |

## Done when

- The CI `go` job is green on ubuntu-24.04, macos-14 and windows-latest, and
  the `cross`, `e2e` and `desktop-*` jobs are green.
- A `workflow_dispatch` release run produces: the goreleaser archives (with
  embedded agents), `jumpgate-linux-{amd64,arm64}` plus `agents-SHA256SUMS`,
  `Jumpgate-macos-{arm64,amd64}.zip` (two apps each),
  `jumpgate-windows-amd64.zip` (two exes), and
  `jumpgate-linux-{amd64,arm64}.tar.gz`.
- A release controller on macOS or Windows pairs a Linux box over SSH with no
  `~/.jumpgate/agents` directory.
- `jumpgate hosts add me --local` works on Linux as root without sudo and as a
  NOPASSWD user, and refuses with a hint everywhere else.
