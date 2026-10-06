#!/usr/bin/env bash
# scripts/package-windows.sh — jumpgate-windows-amd64.zip (spec D6):
#   jumpgate.exe       console subsystem; double-click opens a terminal
#                      running the terminal home; also the CLI
#   jumpgate-tray.exe  GUI subsystem; the desktop window and tray icon
# Both embed the Linux agents. Runs on a Windows runner (Git Bash) with Go and
# a C compiler (the tray build uses cgo for WebView2). It also cross-builds
# from Linux with mingw-w64: set CC and CXX to the x86_64-w64-mingw32 compilers
# (and see the cross job in .gitlab-ci.yml for the EventToken.h include).
set -euo pipefail
out="${1:-dist}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
ld="-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$version"
VERSION="$version" GZIP=1 scripts/build-agents.sh internal/agentbin/embedded
stage="$out/jumpgate-windows-amd64"
rm -rf "$stage"
mkdir -p "$stage"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags embedagents -trimpath -ldflags "$ld" \
  -o "$stage/jumpgate.exe" ./cmd/jumpgate
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -tags "tray embedagents" -trimpath -ldflags "$ld -H windowsgui" \
  -o "$stage/jumpgate-tray.exe" ./cmd/jumpgate
cp packaging/windows/README.txt "$stage/README.txt"
rm -f "$out/jumpgate-windows-amd64.zip"
# 7-Zip is on the Windows runners; Info-ZIP stands in on a Linux cross build.
if command -v 7z >/dev/null 2>&1; then
  ( cd "$out" && 7z a -tzip jumpgate-windows-amd64.zip jumpgate-windows-amd64 >/dev/null )
else
  ( cd "$out" && zip -qr jumpgate-windows-amd64.zip jumpgate-windows-amd64 )
fi
echo "$out/jumpgate-windows-amd64.zip"
