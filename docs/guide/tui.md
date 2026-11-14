# The TUI

Run `jumpgate` with no arguments in a terminal, or open the terminal launcher for your OS. It starts the background server if none is running. The TUI holds no keys and does no work itself: it talks to the local server, so quitting leaves everything running.

## Screens

Keys `1` to `8` switch screens. At 100 columns or more a list of screens and boxes sits on the left; narrower, the screens are tabs in the header.

- **1 Fleet:** every box at once: link, network, sync, lag, peers, disk and firewall. `a` adds a box, `p` pairs or re-pairs it and `D` removes it. `v` switches to a table of addresses and pairing, where `u` upgrades a box's gate. Settings chooses the columns. `enter` opens a box, with the tabs Overview, Storage, Endpoints, Security, Logs, Services, Docker and Tools; Settings can hide any of them.
- **2 Jobs:** the jobs inbox, with each job's steps, lease and question. The "needs decision" count in the status line at the bottom is the number of jobs waiting for you across the fleet.
- **3 Signers:** this controller's key and the boxes it signs for. Test signing, create or export the key.
- **4 Gateways:** your RPC gateways: create, start, stop, provision and delete them, read their analytics and traffic, manage customer keys, and turn their modules on and off.
- **5 VPN:** WireGuard client overlays and VPN servers, with their peers.
- **6 Vault:** the encrypted vault: unlock, lock, back up, restore, recovery codes, sealing. See [The vault](vault.md).
- **7 Server:** the controller's server: its version, an update check, start and stop, and on Linux, installing and setting up the systemd service.
- **8 Settings:** fleet columns, box tabs, display choices, the AI explain provider, the vault's idle lock and other preferences, saved at once.

## Keys

The same key means the same thing on every screen: `r` reloads, `s` starts, `t` stops, `e` restarts, `n` makes a new one, `D` deletes or removes, and `x` lists everything you can do with the selected item.

- `1` to `8`: switch screens.
- Arrows or `h` `j` `k` `l`: move, and switch tabs.
- `enter`: open.
- `esc`: go back.
- `/`: filter.
- `tab`: move between the list, the main area and the activity log. `[` and `]` show or hide the tree and the activity log, and `.` opens or closes the activity drawer.
- `x`: the actions menu for the selected item: a filterable list of what you can do now, each with its key. Press `/` to filter, `enter` or a click to run one.
- `m`: on an open box, its modules. See [Modules](modules.md). On the Storage tab `r` measures the disk.
- `:`: take a command.
- `?`: show every key for the screen you are on. Press `F` there for this guide.
- `g`: the setup checklist. `i` shows the one line that sets up a box for this controller.
- `U`: unlock the vault, offered while it is locked. `L` on the Vault screen locks it, and asks first when jobs are running.
- `q`: quit.
- `R`: restart the server, offered when its version differs from this jumpgate.

Stopping or restarting a service asks you to type its name. Every other change that cannot be undone asks the same.

## Commands

Press `:` and type one of:

- `fleet`, `hosts`, `signers`, `gateways`, `vpn`, `jobs`, `vault`, `server`, `settings`
- `host NAME`, `logs NAME`, `measure NAME`, `ssh NAME`
- `start exec NAME`, `stop beacon NAME`, `restart beacon NAME`
- `remove NAME`
- `migrate-names NAME`
- `add`, `keys`, `help [TOPIC]`, `quit`

`hosts` opens the Fleet as the table of addresses and pairing. `keys` shows the same list as `?`.

## The guide

`:help` lists the guide's topics and `:help TOPIC` opens one, such as `:help jobs`. Or press `F` in the `?` key list. On a page, `j` and `k` scroll, `space` and `b` page, `g` and `G` go to the top and bottom, `h` and `l` turn to the previous and next topic, and `esc` goes back to the topics.

## Terminals

- The TUI needs 80x24.
- Mouse support is on: click tabs, rows and the `[key]` buttons. To select text, hold Shift (or Option on macOS terminals) while dragging. Turn it off in Settings.
- `NO_COLOR=1` drops colour. `JUMPGATE_ASCII=1` uses plain ASCII drawing, which is automatic on the classic Windows console and the Linux virtual console. Settings can also choose the glyphs.
- `jumpgate home`, `JUMPGATE_PLAIN=1` or `TERM=dumb` shows the plain line-based screen instead.
- `jumpgate tui` opens the TUI explicitly, and refuses to run without a terminal.
- Turn on "Bell on jobs that need you" in Settings to have the terminal ring when the jobs count goes up.
