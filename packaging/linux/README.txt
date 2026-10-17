Jumpgate for Linux
==================

./install.sh   installs jumpgate for your user under ~/.local (no root), with
               two menu entries: "Jumpgate" opens a terminal running jumpgate,
               "Jumpgate (window)" opens the desktop panel.

./install.sh --force      replaces a jumpgate file it did not install itself.
./install.sh --uninstall  removes exactly the files it installed.
PREFIX=/usr/local ./install.sh, run as root, installs system-wide instead.
(HOME must be set unless PREFIX is.)

Needs WebKitGTK 4.1 and GTK 3, for both entries and every command: this
jumpgate binary links them, so it does not start without them.
  Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0
  Fedora:        sudo dnf install webkit2gtk4.1 gtk3

On a server without them, use the headless jumpgate_linux_<arch>.tar.gz
release archive instead: the same TUI and commands, with no window.
