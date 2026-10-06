#!/bin/sh
# install.sh — install jumpgate for this user, without root: the binary, two
# menu entries (a terminal and a window) and the icon, under $PREFIX
# (default ~/.local; PREFIX=/usr/local works when run as root).
#
#   ./install.sh              install (refuses to replace files it did not install)
#   ./install.sh --force      replace such files anyway
#   ./install.sh --uninstall  remove exactly the files a previous run installed
set -eu

force=0
uninstall=0
for arg in "$@"; do
  case "$arg" in
    --force) force=1 ;;
    --uninstall) uninstall=1 ;;
    *) echo "usage: install.sh [--force | --uninstall]" >&2; exit 2 ;;
  esac
done

if [ -z "${PREFIX:-}" ] && [ -z "${HOME:-}" ]; then
  echo "install.sh: HOME is not set; set HOME, or PREFIX to the install location" >&2
  exit 1
fi
here="$(cd "$(dirname "$0")" && pwd)"
prefix="${PREFIX:-$HOME/.local}"
bin="$prefix/bin"
apps="$prefix/share/applications"
icons="$prefix/share/icons/hicolor/scalable/apps"
# The manifest is the marker that these files are ours: a file is only
# replaced or removed when a previous run recorded it here.
state="$prefix/share/jumpgate"
manifest="$state/install-manifest"
targets="$bin/jumpgate
$apps/jumpgate.desktop
$apps/jumpgate-window.desktop
$icons/jumpgate.svg"

refresh() {
  if command -v update-desktop-database >/dev/null 2>&1; then update-desktop-database "$apps" 2>/dev/null || true; fi
}

if [ "$uninstall" = 1 ]; then
  if [ ! -f "$manifest" ]; then
    echo "install.sh: no jumpgate install recorded under $prefix; nothing to remove" >&2
    exit 1
  fi
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    rm -f -- "$f"
  done < "$manifest"
  rm -f -- "$manifest"
  rmdir "$state" 2>/dev/null || true
  refresh
  echo "removed the jumpgate files installed under $prefix"
  exit 0
fi

# Never overwrite a file we did not put there (say, another program called
# jumpgate); the existing file is not executed to find out.
if [ "$force" = 0 ]; then
  printf '%s\n' "$targets" | while IFS= read -r f; do
    if { [ -e "$f" ] || [ -L "$f" ]; } && ! { [ -f "$manifest" ] && grep -Fxq -- "$f" "$manifest"; }; then
      echo "install.sh: $f exists and was not installed by this script; use --force to replace it" >&2
      exit 1
    fi
  done
fi

mkdir -p "$bin" "$apps" "$icons" "$state"
install -m 0755 "$here/jumpgate" "$bin/jumpgate"
install -m 0644 "$here/jumpgate.svg" "$icons/jumpgate.svg"

# The Exec value, per the desktop entry spec: the path in double quotes (it may
# hold spaces), with " ` $ and \ backslash-escaped for the quoting layer, every
# backslash doubled again for the string-value layer, and % doubled because it
# starts a field code.
q=$(printf '%s' "$bin/jumpgate" | sed -e 's/[\\"`$]/\\&/g' -e 's/\\/\\\\/g' -e 's/%/%%/g')
# A desktop session often lacks ~/.local/bin on PATH, so the entries name the
# binary by its absolute path. Built line by line, not with sed, so no path
# character can act as a replacement metacharacter.
for f in jumpgate.desktop jumpgate-window.desktop; do
  while IFS= read -r line; do
    case "$line" in
      "Exec=env JUMPGATE_LAUNCHER=1 jumpgate") printf '%s\n' "Exec=env JUMPGATE_LAUNCHER=1 \"$q\"" ;;
      "Exec=jumpgate --tray") printf '%s\n' "Exec=\"$q\" --tray" ;;
      *) printf '%s\n' "$line" ;;
    esac
  done < "$here/$f" > "$apps/$f"
  chmod 0644 "$apps/$f"
done
printf '%s\n' "$targets" > "$manifest"
refresh
echo "installed $bin/jumpgate; Jumpgate and Jumpgate (window) are in your applications menu"
case ":$PATH:" in
  *":$bin:"*) ;;
  *) echo "note: $bin is not on your PATH; add it to run jumpgate from a shell" ;;
esac
