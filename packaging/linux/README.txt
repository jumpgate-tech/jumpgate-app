Jumpgate for Linux
==================

./install.sh   installs jumpgate for your user under ~/.local (no root), with
               two menu entries: "Jumpgate" opens a terminal running jumpgate,
               "Jumpgate (window)" opens the desktop panel.

Needs WebKitGTK 4.1 and GTK 3 for the window:
  Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0
  Fedora:        sudo dnf install webkit2gtk4.1 gtk3

The terminal entry and every command (`jumpgate help`) work without them on
a server, as does the headless jumpgate_linux_<arch>.tar.gz release archive.
