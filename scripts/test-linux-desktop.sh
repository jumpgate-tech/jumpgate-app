#!/usr/bin/env bash
# scripts/test-linux-desktop.sh — the Linux desktop bundle end to end, in
# containers: build on Ubuntu 24.04 against WebKitGTK 4.1 (spec D11), validate
# the .desktop files, install as a normal user, open the window under Xvfb and
# check the server answers, then check the binary's libraries resolve on
# Debian 13. Needs Docker; CI runs the same script.
set -euo pipefail
# The bundle moves between the two containers in a named volume, not a host
# directory: a colima host mount may be read-only or not cover the temp dir.
work="jumpgate-desktop-$$"
docker volume create "$work" >/dev/null
trap 'docker volume rm -f "$work" >/dev/null' EXIT
go_version="${GO_VERSION:-1.25.0}"
# Resolved on the host: inside the container a linked worktree's .git file
# points at a path that does not exist, so git there cannot describe it.
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"

docker run --rm -v "$PWD":/src:ro -v "$work":/out -w /build -e GOFLAGS=-buildvcs=false -e VERSION="$version" ubuntu:24.04 bash -euc '
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y -qq build-essential pkg-config git ca-certificates curl gzip \
    libgtk-3-dev libwebkit2gtk-4.1-dev xvfb xauth desktop-file-utils >/dev/null
  curl -fsSL "https://go.dev/dl/go'"$go_version"'.linux-$(dpkg --print-architecture).tar.gz" | tar -C /usr/local -xz
  export PATH=/usr/local/go/bin:$PATH
  # The package script writes into the tree (embedded agents), and the source
  # mount is read-only on some hosts, so build from a copy.
  mkdir -p /build && tar -C /src --exclude=.git -cf - . | tar -C /build -xf -
  scripts/package-linux.sh /out
  arch=$(go env GOARCH)
  b=/out/jumpgate-linux-$arch
  desktop-file-validate "$b/jumpgate.desktop" "$b/jumpgate-window.desktop"
  grep -qx "Terminal=true" "$b/jumpgate.desktop"

  useradd -m u
  cp -r "$b" /home/u/bundle && chown -R u:u /home/u/bundle
  su u -c "/home/u/bundle/install.sh"
  test -x /home/u/.local/bin/jumpgate
  grep -q "^Exec=env JUMPGATE_LAUNCHER=1 /home/u/.local/bin/jumpgate$" /home/u/.local/share/applications/jumpgate.desktop
  grep -q "^Exec=/home/u/.local/bin/jumpgate --tray$" /home/u/.local/share/applications/jumpgate-window.desktop
  test -f /home/u/.local/share/icons/hicolor/scalable/apps/jumpgate.svg

  # The window entry: the server must come up behind the webview. WebKit'"'"'s
  # bubblewrap sandbox cannot run inside an unprivileged container.
  su u -c "cd ~ && WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1 xvfb-run -a ~/.local/bin/jumpgate --tray >/tmp/window.log 2>&1 &"
  ok=
  for _ in $(seq 1 60); do
    f=/home/u/.jumpgate/run/server.json
    if [ -f "$f" ]; then
      token=$(sed -n "s/.*\"token\": \"\(.*\)\".*/\1/p" "$f")
      addr=$(sed -n "s/.*\"httpAddr\": \"\(.*\)\".*/\1/p" "$f")
      if curl -fsS -H "Authorization: Bearer $token" "http://$addr/api/health" >/dev/null 2>&1; then ok=1; break; fi
    fi
    sleep 0.5
  done
  [ -n "$ok" ] || { cat /tmp/window.log; echo "the desktop window build never served" >&2; exit 1; }
  su u -c "~/.local/bin/jumpgate stop"
  echo "ubuntu 24.04: OK"
'

docker run --rm -v "$work":/out:ro debian:trixie bash -euc '
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq && apt-get install -y -qq libwebkit2gtk-4.1-0 libgtk-3-0 >/dev/null
  b=$(ls -d /out/jumpgate-linux-*/ | head -1)
  if ldd "$b/jumpgate" | grep "not found"; then echo "missing libraries on Debian 13" >&2; exit 1; fi
  "$b/jumpgate" --version
  echo "debian 13: OK"
'
