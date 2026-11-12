# Getting started

Jumpgate runs on your own computer or server, the controller. It sets up nodes on Debian or Ubuntu servers you point it at, over SSH or on the same machine. It puts a small program, the gate, on each box; the gate carries out what the controller asks.

## What a box needs

- **Debian or Ubuntu Linux** with systemd. Setup installs packages with `apt-get`.
- **Root over SSH, for pairing and setup,** or a user with passwordless sudo and `--sudo`. Setup writes systemd units, manages services with `systemctl` and installs binaries to `/usr/local/bin`. Preflight checks for Linux, root, systemd and free disk, and stops with a clear message if one is missing.
- The node services themselves run as an unprivileged system account, `jumpgate-node`, that setup creates, never as root.

## 1. Create the controller key

```
jumpgate keys init
```

Pick where the key lives with `--store`. Without it, jumpgate uses this machine's own key store when one works, and otherwise the vault:

- `keychain`: the macOS Keychain, the macOS default; or on Linux, a Secret Service that answers (`secret-tool`).
- `wincred`: Windows Credential Manager, the Windows default.
- `vault`: jumpgate's encrypted vault, protected by a passphrase. The default on Linux when no Secret Service answers, as on headless boxes and most SSH logins. Run `jumpgate unlock` afterwards so the server can sign. See [The vault](vault.md).
- `1password`: with `--ref op://vault/item/field`.
- `file`: an unencrypted owner-only file. It needs `--insecure-plain` to confirm.

## 2. Pair a box

```
jumpgate hosts add box-a --ssh root@203.0.113.7
```

You confirm the box's host key by its fingerprint. Jumpgate then installs the gate, `jumpgate-agent.service`, a restricted `jumpgate` tunnel user, and an sshd drop-in that confines that user to the gate's socket. `--jump USER@HOST` reaches the box through a jump host. On Linux, `--local` pairs the machine you are on (as root, or with sudo on the terminal).

If you would rather not give the controller root over SSH, run `jumpgate hosts invite box-a`. It prints one line to run as root on the box, which sets up its gate for this controller, and the `jumpgate hosts add` line that attaches the box afterwards.

## 3. Turn off root SSH login

Once pairing succeeds, set `PermitRootLogin no` on the box. You will not need root SSH again unless you have to repair the gate.

## 4. Watch it

```
jumpgate status box-a
jumpgate logs box-a -n 200
```

Or open the desktop app, or run `jumpgate` in a terminal for the TUI. In the TUI, `g` opens a setup checklist that shows what is done and what comes next.

## One server for everything

The app, the TUI and the CLI share one server per user. The TUI and the CLI start it in the background, and quitting the TUI leaves it running; `jumpgate stop` ends it. If the desktop app is first, the server runs inside the app and stops when you quit it.

On Linux over SSH, the background server survives closing the terminal. Where systemd-logind kills a user's processes at logout (`KillUserProcesses=yes`), run `loginctl enable-linger $USER` once so it also survives the SSH session ending.

## A controller on a Linux server

A Linux server can run the controller as a systemd service, `jumpgate.service`, under its own `jumpgate-ctl` account, so it keeps running with nobody logged in. Install it as described in [Install](install.md); with a plain binary, `sudo jumpgate setup install-service` writes the unit. Then run:

```
sudo jumpgate setup server
```

It starts the service, adds you to the `jumpgate-admin` group, has the service create its controller key (in the vault by default, so it asks for a new passphrase), unlocks it and pairs this machine. Each step is skipped when it is already done, so running it again finishes an interrupted run. `--self-only` makes a controller that manages only this machine. Log out and back in so the group applies, then run `jumpgate` on the server, or `ssh -t SERVER jumpgate` from your own machine. Where the service runs, the CLI and the TUI use it first.

A restarted service comes back locked. Unlock it, or seal the vault key so it unlocks at boot; see [The vault](vault.md).

## A node on your desktop

Nodes run on Linux only. To run one on a Mac or PC, use a WSL2 or Lima VM as an ordinary Linux box and pair it over SSH.
