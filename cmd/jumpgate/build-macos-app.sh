#!/usr/bin/env bash
#
# Build Jumpgate.app — a double-clickable macOS application bundle around the
# tray/tiny-app build of jumpgate.
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
# cannot cross-compile. Requires: go and the Xcode command line tools.
#
# Usage:  cmd/jumpgate/build-macos-app.sh [output-dir]
#   output-dir defaults to the repo root, producing <repo>/Jumpgate.app
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_DIR="${1:-$REPO_ROOT}"

APP_NAME="Jumpgate"
BUNDLE_ID="city.valve.jumpgate"
EXE_NAME="jumpgate"
APP="$OUT_DIR/$APP_NAME.app"

VERSION="${VERSION:-$(git -C "$REPO_ROOT" describe --tags --always --dirty 2>/dev/null || echo "dev")}"

echo "==> Building $APP_NAME.app ($VERSION)"

# --- assemble the bundle skeleton -------------------------------------------
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

# --- the Linux agents this app uploads when pairing (spec D1) --------------
echo "--> building the embedded Linux agents"
( cd "$REPO_ROOT" && VERSION="$VERSION" GZIP=1 bash scripts/build-agents.sh internal/agentbin/embedded )

# --- compile the tray binary straight into the bundle -----------------------
echo "--> go build -tags \"tray embedagents\" (CGo)"
CGO_ENABLED=1 go build -tags "tray embedagents" \
	-ldflags "-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$VERSION" \
	-o "$APP/Contents/MacOS/$EXE_NAME" \
	"$REPO_ROOT/cmd/jumpgate"

# --- icon: the committed AppIcon.icns (regenerate with scripts/render-icons.sh)
cp "$SCRIPT_DIR/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"

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

# --- Jumpgate Terminal.app: opens the user's terminal running jumpgate ------
TAPP="$OUT_DIR/$APP_NAME Terminal.app"
echo "==> Building $APP_NAME Terminal.app"
rm -rf "$TAPP"
mkdir -p "$TAPP/Contents/MacOS" "$TAPP/Contents/Resources"
install -m 0755 "$REPO_ROOT/packaging/macos/jumpgate-terminal" "$TAPP/Contents/MacOS/jumpgate-terminal"
install -m 0755 "$REPO_ROOT/packaging/macos/jumpgate.command" "$TAPP/Contents/Resources/jumpgate.command"
# The same binary as the window app: with a terminal and no arguments it
# opens the terminal home, and it starts the server when a command needs it.
install -m 0755 "$APP/Contents/MacOS/$EXE_NAME" "$TAPP/Contents/Resources/jumpgate"
cp "$APP/Contents/Resources/AppIcon.icns" "$TAPP/Contents/Resources/AppIcon.icns"
cat >"$TAPP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>            <string>$APP_NAME Terminal</string>
	<key>CFBundleDisplayName</key>     <string>$APP_NAME Terminal</string>
	<key>CFBundleIdentifier</key>      <string>$BUNDLE_ID.terminal</string>
	<key>CFBundleExecutable</key>      <string>jumpgate-terminal</string>
	<key>CFBundleIconFile</key>        <string>AppIcon</string>
	<key>CFBundlePackageType</key>     <string>APPL</string>
	<key>CFBundleShortVersionString</key> <string>$VERSION</string>
	<key>CFBundleVersion</key>         <string>$VERSION</string>
	<key>LSMinimumSystemVersion</key>  <string>10.15</string>
	<key>LSUIElement</key>             <true/>
</dict>
</plist>
PLIST
codesign --force --deep --sign - "$TAPP" >/dev/null 2>&1 || \
	echo "    (codesign skipped — bundle still runs locally)"
echo "==> Built $TAPP"

echo "==> Built $APP"
echo "    open \"$APP\"   # or double-click it in Finder"
