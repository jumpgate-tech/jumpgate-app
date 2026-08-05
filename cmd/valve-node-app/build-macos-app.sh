#!/usr/bin/env bash
#
# Build Jumpgate.app — a double-clickable macOS application bundle around the
# tray/tiny-app build of valve-node-app.
#
# Why a bundle (vs. just `go build -tags tray` and running the binary):
#   - A bare Mach-O binary is a "background" process to the OS: no Dock icon,
#     and its window opens behind whatever launched it (you had to Cmd-Tab to
#     find it). A .app is a foreground app — it gets a Dock icon and its window
#     comes to the front on launch.
#   - Double-clicking passes no CLI flags, so the binary detects it is running
#     inside a .app (see inAppBundle in main.go) and enters tray mode itself.
#
# The tray build needs CGo (WebKit via webview_go), so this is macOS-only and
# cannot cross-compile. Requires: go, rsvg-convert, iconutil, sips.
#
# Usage:  cmd/valve-node-app/build-macos-app.sh [output-dir]
#   output-dir defaults to the repo root, producing <repo>/Jumpgate.app
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_DIR="${1:-$REPO_ROOT}"

APP_NAME="Jumpgate"
BUNDLE_ID="city.valve.jumpgate"
EXE_NAME="valve-node-app"
APP="$OUT_DIR/$APP_NAME.app"

VERSION="$(git -C "$REPO_ROOT" describe --tags --always --dirty 2>/dev/null || echo "dev")"

echo "==> Building $APP_NAME.app ($VERSION)"

# --- assemble the bundle skeleton -------------------------------------------
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

# --- compile the tray binary straight into the bundle -----------------------
echo "--> go build -tags tray (CGo)"
CGO_ENABLED=1 go build -tags tray \
	-ldflags "-s -w" \
	-o "$APP/Contents/MacOS/$EXE_NAME" \
	"$REPO_ROOT/cmd/valve-node-app"

# --- icon: icon.svg -> AppIcon.icns -----------------------------------------
echo "--> rendering AppIcon.icns from icon.svg"
ICONSET="$(mktemp -d)/AppIcon.iconset"
mkdir -p "$ICONSET"
render() { rsvg-convert -w "$1" -h "$1" "$SCRIPT_DIR/icon.svg" -o "$ICONSET/$2"; }
render 16   icon_16x16.png
render 32   icon_16x16@2x.png
render 32   icon_32x32.png
render 64   icon_32x32@2x.png
render 128  icon_128x128.png
render 256  icon_128x128@2x.png
render 256  icon_256x256.png
render 512  icon_256x256@2x.png
render 512  icon_512x512.png
render 1024 icon_512x512@2x.png
iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/AppIcon.icns"
rm -rf "$(dirname "$ICONSET")"

# --- Info.plist -------------------------------------------------------------
cat >"$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>            <string>$APP_NAME</string>
	<key>CFBundleDisplayName</key>     <string>$APP_NAME</string>
	<key>CFBundleIdentifier</key>      <string>$BUNDLE_ID</string>
	<key>CFBundleExecutable</key>      <string>$EXE_NAME</string>
	<key>CFBundleIconFile</key>        <string>AppIcon</string>
	<key>CFBundlePackageType</key>     <string>APPL</string>
	<key>CFBundleShortVersionString</key> <string>$VERSION</string>
	<key>CFBundleVersion</key>         <string>$VERSION</string>
	<key>LSMinimumSystemVersion</key>  <string>10.15</string>
	<key>NSHighResolutionCapable</key> <true/>
</dict>
</plist>
PLIST

# Ad-hoc sign so the bundle is launchable and window-frontable without a
# developer certificate. (A real Developer ID + notarization is only needed to
# distribute it to other machines.)
codesign --force --deep --sign - "$APP" >/dev/null 2>&1 || \
	echo "    (codesign skipped — bundle still runs locally)"

echo "==> Built $APP"
echo "    open \"$APP\"   # or double-click it in Finder"
