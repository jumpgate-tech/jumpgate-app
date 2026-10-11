#!/bin/bash
# scripts/windows-vm/install-on-demand.sh [install|uninstall]
# Renders on-demand.plist.tmpl to ~/Library/LaunchAgents and (un)loads it.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
label=tech.jumpgate.win-vm-on-demand
plist="$HOME/Library/LaunchAgents/$label.plist"
dom="gui/$(id -u)"
case "${1:-install}" in
  install)
    mkdir -p "$HOME/Library/LaunchAgents" "$HOME/Library/Logs"
    # launchd's bash cannot read scripts under ~/Documents (TCC: "Operation not
    # permitted"), so run copies kept outside the protected folders. Re-run
    # install after editing vm.sh or on-demand.sh.
    bin="$HOME/Library/Application Support/jumpgate-win-vm"
    mkdir -p "$bin"
    cp "$here/on-demand.sh" "$here/vm.sh" "$bin/"
    chmod +x "$bin/on-demand.sh" "$bin/vm.sh"
    path="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"
    sed -e "s|@BIN@|$bin|g" -e "s|@PATH@|$path|g" -e "s|@HOME@|$HOME|g" "$here/on-demand.plist.tmpl" > "$plist"
    plutil -lint "$plist" >/dev/null
    launchctl bootout "$dom/$label" 2>/dev/null || true
    launchctl bootstrap "$dom" "$plist"
    echo "installed $plist (log: ~/Library/Logs/jumpgate-win-vm-on-demand.log)" ;;
  uninstall)
    launchctl bootout "$dom/$label" 2>/dev/null || true
    rm -f "$plist"; echo "removed $label" ;;
  *) echo "usage: $0 [install|uninstall]" >&2; exit 2 ;;
esac
