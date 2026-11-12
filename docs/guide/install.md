# Install

Jumpgate runs on macOS, Windows and Linux. Every file is on the [latest release](https://github.com/jumpgate-tech/jumpgate-app/releases/latest) page. Each download carries both Linux gates, amd64 and arm64, so it can pair a Linux box with nothing else to fetch. (The gate is the small program jumpgate puts on each box.)

## Desktop app

- **macOS:** `Jumpgate-macos-arm64.zip` for Apple silicon, `Jumpgate-macos-amd64.zip` for Intel. Unzip it for `Jumpgate.app` (a window and a menubar icon) and `Jumpgate Terminal.app` (opens a terminal running jumpgate). Drag them to Applications.
- **Windows:** `jumpgate-windows-amd64.zip`. `jumpgate.exe` is the console program: double-click it, or run it from a terminal. `jumpgate-tray.exe` is the window with a notification icon; it needs the Microsoft Edge WebView2 Runtime, which Windows 11 includes. A `README.txt` is inside.
- **Linux:** `jumpgate-linux-amd64.tar.gz` or `jumpgate-linux-arm64.tar.gz`. Extract it, go into its folder and run `./install.sh` for a per-user install under `~/.local` with two menu entries: "Jumpgate" for a terminal and "Jumpgate (window)" for the desktop panel. `./install.sh --uninstall` removes exactly what it installed.

## Terminal only: TUI and CLI

The `jumpgate_<os>_<arch>` archives hold the `jumpgate` binary, for servers and scripts. There are builds for `darwin_arm64`, `darwin_amd64`, `linux_amd64`, `linux_arm64`, `windows_amd64` and `windows_arm64`. The Windows archives are `.zip`; the others are `.tar.gz`.

```
tar xzf jumpgate_linux_amd64.tar.gz
./jumpgate
```

In a terminal, bare `jumpgate` opens the terminal UI.

## A Linux server

To run the controller as a service on a Linux server, install the `jumpgate` deb or rpm from the release page, or from the apt and dnf repository at `pkg.jumpgate.app`. The package puts `jumpgate.service` in place and creates its users; it never turns the service on. The `install.sh` file on the release page, run as root with `--controller`, does the same for a release binary when there is no package manager. Then finish with the steps in [Getting started](getting-started.md).

## The downloads are not code-signed yet

- **Windows:** SmartScreen may warn. Choose "More info", then "Run anyway".
- **macOS:** the apps are ad-hoc signed, not notarized, so macOS blocks the first launch. Open System Settings > Privacy & Security and choose Open Anyway. On macOS 14 and earlier, right-clicking the app and choosing Open also works.

## Verify a download

`checksums.txt` on the release lists a SHA-256 for every archive, desktop bundle and gate. Each file also carries a build-provenance attestation:

```
gh attestation verify <file> --repo jumpgate-tech/jumpgate-app
```

## Minimum versions

- macOS 12 (Monterey).
- Windows 10 version 1803, or Windows Server 2019. The background server needs it for its local socket.
- Linux: the desktop bundle needs WebKitGTK 4.1 and GTK 3, even for its terminal entry (tested on Ubuntu 24.04 and Debian 13). Without them, use the `jumpgate_linux_<arch>` archive.
- The gate that runs on your boxes is Linux-only, amd64 and arm64.

## Build from source

You need Go 1.26+. The built web UI is committed, so Node 22+ is needed only if you change `cmd/jumpgate/web`.

```
git clone https://github.com/jumpgate-tech/jumpgate-app.git
cd jumpgate-app
go build -o jumpgate ./cmd/jumpgate
./jumpgate
```

A plain build has the TUI, the CLI and the server, but no window and no embedded gates. To pair boxes with it, run `scripts/build-agents.sh` first; it writes the gates to `~/.jumpgate/agents`.
