#!/bin/sh
# install.sh — install jumpgate for this user, without root: the binary, two
# menu entries (a terminal and a window) and the icon, under $PREFIX
# (default ~/.local).
set -eu
here="$(cd "$(dirname "$0")" && pwd)"
prefix="${PREFIX:-$HOME/.local}"
bin="$prefix/bin"
apps="$prefix/share/applications"
icons="$prefix/share/icons/hicolor/scalable/apps"
mkdir -p "$bin" "$apps" "$icons"
install -m 0755 "$here/jumpgate" "$bin/jumpgate"
install -m 0644 "$here/jumpgate.svg" "$icons/jumpgate.svg"
# A desktop session often lacks ~/.local/bin on PATH, so the entries name the
# binary by its absolute path.
for f in jumpgate.desktop jumpgate-window.desktop; do
  sed -e "s|^Exec=env JUMPGATE_LAUNCHER=1 jumpgate\$|Exec=env JUMPGATE_LAUNCHER=1 $bin/jumpgate|" \
      -e "s|^Exec=jumpgate --tray\$|Exec=$bin/jumpgate --tray|" "$here/$f" > "$apps/$f"
  chmod 0644 "$apps/$f"
done
if command -v update-desktop-database >/dev/null 2>&1; then update-desktop-database "$apps" || true; fi
echo "installed $bin/jumpgate; Jumpgate and Jumpgate (window) are in your applications menu"
case ":$PATH:" in
  *":$bin:"*) ;;
  *) echo "note: $bin is not on your PATH; add it to run jumpgate from a shell" ;;
esac
