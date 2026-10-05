#!/usr/bin/env bash
# scripts/test-macos-bundle.sh — build both macOS apps and check them without
# a GUI: plists lint, bundle structure, and the terminal app hands its bundled
# jumpgate.command to `open` (spec D24). macOS only.
#
# macOS stalls exec of freshly built binaries, so locally nothing built here is
# run: the hand-off check runs the launcher through /bin/sh with /bin/echo
# standing in for `open`. Run it locally on a Mac (README, developer section). With CI=true (a runner, where that is safe) the app
# binaries also run `--version`.
set -euo pipefail
cd "$(dirname "$0")/.."
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT
bash cmd/jumpgate/build-macos-app.sh "$out"
tapp="$out/Jumpgate Terminal.app"
for app in "$out/Jumpgate.app" "$tapp"; do
  plutil -lint "$app/Contents/Info.plist"
  codesign --verify "$app"
done
test -x "$out/Jumpgate.app/Contents/MacOS/jumpgate"
test -x "$tapp/Contents/Resources/jumpgate"
test -x "$tapp/Contents/MacOS/jumpgate-terminal"
test -x "$tapp/Contents/Resources/jumpgate.command"
test -f "$tapp/Contents/Resources/AppIcon.icns"
grep -q '^export JUMPGATE_LAUNCHER=1$' "$tapp/Contents/Resources/jumpgate.command"
sh -n "$tapp/Contents/Resources/jumpgate.command"
sh -n "$tapp/Contents/MacOS/jumpgate-terminal"

if [ "${CI:-}" = "true" ]; then
  "$out/Jumpgate.app/Contents/MacOS/jumpgate" --version
  "$tapp/Contents/Resources/jumpgate" --version
fi

got="$(JUMPGATE_OPEN=/bin/echo sh "$tapp/Contents/MacOS/jumpgate-terminal")"
test "$got" = "$tapp/Contents/Resources/jumpgate.command" || {
  echo "hand-off argv: got '$got'" >&2; exit 1; }
echo "macos bundles: OK"
