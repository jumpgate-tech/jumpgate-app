#!/usr/bin/env bash
# scripts/test-linux-desktop.sh — the Linux desktop bundle end to end, in
# containers: build on Ubuntu 24.04 against WebKitGTK 4.1 (spec D11), then work
# from the finished tarball: validate the .desktop files, install as a normal
# user (including awkward paths, the refusal to replace foreign files and the
# uninstall), open the window under Xvfb and check the server answers, then
# check the binary's libraries resolve on Debian 13. Needs Docker; CI runs the
# same script.
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

# The source goes in on stdin as the files git knows (tracked, plus new files
# not ignored), so dist/, .claude/ and other local clutter stay out. The
# package script writes into its tree, and the host mount may be read-only.
git ls-files -z --cached --others --exclude-standard |
COPYFILE_DISABLE=1 tar --null -T - -cf - |
docker run --rm -i -v "$work":/out -w /build -e GOFLAGS=-buildvcs=false -e VERSION="$version" ubuntu:24.04 bash -euc '
  export DEBIAN_FRONTEND=noninteractive
  mkdir -p /build && tar -xf - -C /build
  apt-get update -qq
  apt-get install -y -qq build-essential pkg-config git ca-certificates curl gzip \
    libgtk-3-dev libwebkit2gtk-4.1-dev xvfb xauth x11-utils desktop-file-utils >/dev/null
  curl -fsSL "https://go.dev/dl/go'"$go_version"'.linux-$(dpkg --print-architecture).tar.gz" | tar -C /usr/local -xz
  export PATH=/usr/local/go/bin:$PATH
  scripts/package-linux.sh /out
  arch=$(go env GOARCH)

  # From here on only the tarball is used, as a user would receive it.
  mkdir /tmp/x && tar -xzf /out/jumpgate-linux-$arch.tar.gz -C /tmp/x
  b=/tmp/x/jumpgate-linux-$arch
  [ "$(stat -c %a "$b/install.sh")" = 755 ] && [ "$(stat -c %a "$b/jumpgate")" = 755 ]
  [ "$(tar -tvzf /out/jumpgate-linux-$arch.tar.gz --numeric-owner | awk "{print \$2}" | sort -u)" = "0/0" ]
  go version -m "$b/jumpgate" | grep -q "embedagents"
  go version -m "$b/jumpgate" | grep -q "tray"
  desktop-file-validate "$b/jumpgate.desktop" "$b/jumpgate-window.desktop"
  grep -qx "Terminal=true" "$b/jumpgate.desktop"

  useradd -m u
  cp -r "$b" /home/u/bundle && chown -R u:u /home/u/bundle
  inst=/home/u/bundle/install.sh

  # A PREFIX with a space and shell/desktop metacharacters must yield a valid,
  # correctly quoted entry.
  runuser -u u -- env PREFIX="/home/u/we ird/\$d\\e" $inst
  ent="/home/u/we ird/\$d\\e/share/applications"
  desktop-file-validate "$ent/jumpgate.desktop" "$ent/jumpgate-window.desktop"
  grep -Fxq "Exec=env JUMPGATE_LAUNCHER=1 \"/home/u/we ird/\\\\\$d\\\\\\\\e/bin/jumpgate\"" "$ent/jumpgate.desktop"
  grep -Fxq "Exec=\"/home/u/we ird/\\\\\$d\\\\\\\\e/bin/jumpgate\" --tray" "$ent/jumpgate-window.desktop"
  # Reinstalling our own files is fine; uninstall removes exactly them.
  runuser -u u -- env PREFIX="/home/u/we ird/\$d\\e" $inst
  runuser -u u -- env PREFIX="/home/u/we ird/\$d\\e" $inst --uninstall
  ! test -e "/home/u/we ird/\$d\\e/bin/jumpgate"
  ! test -e "$ent/jumpgate.desktop"
  ! test -e "/home/u/we ird/\$d\\e/share/jumpgate"

  # A file that is not ours is never replaced without --force, and survives
  # uninstall.
  runuser -u u -- mkdir -p /home/u/o/bin
  runuser -u u -- sh -c "echo foreign > /home/u/o/bin/jumpgate"
  if runuser -u u -- env PREFIX=/home/u/o $inst 2>/tmp/refuse.log; then echo "install replaced a foreign file" >&2; exit 1; fi
  grep -q "not installed by this script" /tmp/refuse.log
  [ "$(cat /home/u/o/bin/jumpgate)" = foreign ]
  runuser -u u -- env PREFIX=/home/u/o $inst --force
  runuser -u u -- env PREFIX=/home/u/o $inst --uninstall
  ! test -e /home/u/o/bin/jumpgate
  # No HOME and no PREFIX is a clear error.
  if runuser -u u -- env -u HOME $inst 2>/tmp/nohome.log; then echo "installed with no HOME" >&2; exit 1; fi
  grep -q "HOME is not set" /tmp/nohome.log

  # A HOME with a space, the default prefix.
  runuser -u u -- env HOME="/home/u/sp ace" $inst
  grep -Fxq "Exec=\"/home/u/sp ace/.local/bin/jumpgate\" --tray" "/home/u/sp ace/.local/share/applications/jumpgate-window.desktop"
  runuser -u u -- env HOME="/home/u/sp ace" $inst --uninstall

  su u -c "/home/u/bundle/install.sh"
  test -x /home/u/.local/bin/jumpgate
  grep -q "^Exec=env JUMPGATE_LAUNCHER=1 \"/home/u/.local/bin/jumpgate\"$" /home/u/.local/share/applications/jumpgate.desktop
  grep -q "^Exec=\"/home/u/.local/bin/jumpgate\" --tray$" /home/u/.local/share/applications/jumpgate-window.desktop
  test -f /home/u/.local/share/icons/hicolor/scalable/apps/jumpgate.svg

  # The window entry: the server must come up behind the webview. WebKit'"'"'s
  # bubblewrap sandbox cannot run inside an unprivileged container.
  su u -c "cd ~ && WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1 xvfb-run -n 99 -s \"-ac -screen 0 1024x768x24\" ~/.local/bin/jumpgate --tray >/tmp/window.log 2>&1 &"
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

  # The window entry names a StartupWMClass only if the real window carries it.
  want=$(sed -n "s/^StartupWMClass=//p" /home/u/.local/share/applications/jumpgate-window.desktop)
  classes=
  for _ in $(seq 1 20); do
    classes=$(DISPLAY=:99 xwininfo -root -tree 2>/dev/null | grep -o "0x[0-9a-f]*" |
      while read -r id; do DISPLAY=:99 xprop -id "$id" WM_CLASS 2>/dev/null; done | grep "WM_CLASS" || true)
    [ -n "$classes" ] && break
    sleep 0.5
  done
  echo "window classes: $classes"
  if [ -n "$want" ]; then echo "$classes" | grep -q "\"$want\"" || { echo "StartupWMClass=$want matches no window class" >&2; exit 1; }; fi
  su u -c "~/.local/bin/jumpgate stop"
  echo "ubuntu 24.04: OK"
'

docker run --rm -v "$work":/out:ro debian:trixie bash -euc '
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq && apt-get install -y -qq libwebkit2gtk-4.1-0 libgtk-3-0 >/dev/null
  mkdir /tmp/x && tar -xzf /out/jumpgate-linux-*.tar.gz -C /tmp/x
  b=$(ls -d /tmp/x/jumpgate-linux-*/ | head -1)
  if ldd "$b/jumpgate" | grep "not found"; then echo "missing libraries on Debian 13" >&2; exit 1; fi
  "$b/jumpgate" --version
  echo "debian 13: OK"
'
