#!/usr/bin/env bash
# scripts/package-linux.sh — jumpgate-linux-<arch>.tar.gz (spec D6, D10): the
# desktop build (webview window on WebKitGTK 4.1) with embedded agents, two
# .desktop entries, the icon and a per-user installer. Needs go, a C
# compiler, pkg-config, libgtk-3-dev and libwebkit2gtk-4.1-dev.
set -euo pipefail
out="${1:-dist}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
arch="$(go env GOARCH)"
VERSION="$version" GZIP=1 scripts/build-agents.sh internal/agentbin/embedded
case "$out" in "" | /) echo "package-linux.sh: refusing OUT_DIR '$out'" >&2; exit 1 ;; esac
stage="$out/jumpgate-linux-$arch"
rm -rf "$stage"
mkdir -p "$stage"
PKG_CONFIG_PATH="$PWD/scripts/linux/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}" \
CGO_ENABLED=1 go build -tags "tray embedagents" -trimpath \
  -ldflags "-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$version" \
  -o "$stage/jumpgate" ./cmd/jumpgate
cp packaging/linux/jumpgate.desktop packaging/linux/jumpgate-window.desktop \
   packaging/linux/install.sh packaging/linux/README.txt "$stage/"
cp cmd/jumpgate/icon.svg "$stage/jumpgate.svg"
chmod 0755 "$stage/install.sh" "$stage/jumpgate"
tar --owner=0 --group=0 --numeric-owner -C "$out" -czf "$out/jumpgate-linux-$arch.tar.gz" "jumpgate-linux-$arch"
echo "$out/jumpgate-linux-$arch.tar.gz"
