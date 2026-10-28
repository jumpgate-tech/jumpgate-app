#!/usr/bin/env bash
# scripts/e2e-install.sh — packaging/install/install.sh against a fake
# release server (scripts/e2e/relsrv) over TLS, one fresh container per case:
# truncation, checksum, provenance, pinning and latest, sudo without a tty,
# --prefix, an unsupported arch, a package-owned binary, and --controller.
# Needs Docker.
#   BASE=ubuntu:24.04 scripts/e2e-install.sh   # the second distro
#   KEEP=1 scripts/e2e-install.sh              # keep the containers to inspect
set -euo pipefail
base="${BASE:-debian:12}"
script=packaging/install/install.sh
work="$(mktemp -d)"
net="jumpgate-e2e-install-$$"
names=()
cleanup() {
  for n in ${names[@]+"${names[@]}"}; do
    if [ -n "${KEEP:-}" ]; then
      echo "kept container $n (docker rm -f $n when done)" >&2
    else
      docker rm -f "$n" >/dev/null 2>&1 || true
    fi
  done
  [ -n "${KEEP:-}" ] || docker network rm "$net" >/dev/null 2>&1 || true
  rm -rf "$work"
}
trap cleanup EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }
ok() { echo "ok - $*"; }

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
docker network create "$net" >/dev/null

case "$(docker run --rm --entrypoint uname jumpgate-e2e -m)" in
  x86_64) goarch=amd64 ;;
  aarch64) goarch=arm64 ;;
  *) fail "unsupported container arch" ;;
esac

# --- fixtures: a release dir, a TLS cert for "relsrv", the server ---------
rel="$work/rel"
sha() { if command -v sha256sum >/dev/null; then sha256sum "$@"; else shasum -a 256 "$@"; fi; }
publish() { # tag binary-file
  local d="$rel/download/$1" a="jumpgate_linux_${goarch}.tar.gz"
  mkdir -p "$d" "$work/pack-$1"
  cp "$2" "$work/pack-$1/jumpgate"
  chmod 0755 "$work/pack-$1/jumpgate"
  local x=()
  [ "$(uname -s)" != Darwin ] || x=(--no-mac-metadata --no-xattrs)
  COPYFILE_DISABLE=1 tar ${x[@]+"${x[@]}"} -C "$work/pack-$1" -czf "$d/$a" jumpgate
  (cd "$d" && sha "$a" >checksums.txt)
}
for tag in v0.0.1 v0.0.2; do
  printf '#!/bin/sh\necho %s\n' "$tag" >"$work/stub-$tag"
  publish "$tag" "$work/stub-$tag"
done
# A tampered release: checksums.txt from v0.0.1, an archive that differs.
publish v0.0.1-tampered "$work/stub-v0.0.1"
printf '#!/bin/sh\necho evil\n' >"$work/evil"
cp "$rel/download/v0.0.1-tampered/checksums.txt" "$work/sums"
publish v0.0.1-tampered "$work/evil"
cp "$work/sums" "$rel/download/v0.0.1-tampered/checksums.txt"
# A real controller, for --controller (a prerelease tag, so latest stays v0.0.2).
CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build \
  -ldflags "-X github.com/valve-tech/jumpgate/internal/buildinfo.version=v0.0.2" \
  -o "$work/jumpgate-real" ./cmd/jumpgate
publish v0.0.2-real "$work/jumpgate-real"
# The raw gate binary beside the archive, as a release ships it, for --agent.
cp "$work/jumpgate-real" "$rel/download/v0.0.2-real/jumpgate-linux-$goarch"
(cd "$rel/download/v0.0.2-real" && sha "jumpgate_linux_${goarch}.tar.gz" "jumpgate-linux-$goarch" >checksums.txt)
CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build -o "$work/relsrv" ./scripts/e2e/relsrv

# No bind mounts (colima does not share every temp dir): make the cert in a
# container and copy it out.
gen="jumpgate-e2e-install-$$-cert"
names+=("$gen")
docker create --name "$gen" jumpgate-e2e openssl req -x509 -newkey ec \
  -pkeyopt ec_paramgen_curve:prime256v1 -nodes -days 1 -subj /CN=relsrv \
  -addext subjectAltName=DNS:relsrv -addext basicConstraints=critical,CA:TRUE \
  -keyout /relsrv.key -out /relsrv.crt >/dev/null
docker start -a "$gen" >/dev/null 2>&1
docker cp -q "$gen:/relsrv.key" "$work/relsrv.key"
docker cp -q "$gen:/relsrv.crt" "$work/relsrv.crt"

srv="jumpgate-e2e-install-$$-relsrv"
names+=("$srv")
docker create --name "$srv" --network "$net" --network-alias relsrv jumpgate-e2e \
  /relsrv -dir /rel -cert /relsrv.crt -key /relsrv.key -addr :8443 >/dev/null
docker cp -q "$work/relsrv" "$srv:/relsrv"
docker cp -q "$rel" "$srv:/rel"
docker cp -q "$work/relsrv.crt" "$srv:/relsrv.crt"
docker cp -q "$work/relsrv.key" "$srv:/relsrv.key"
docker start "$srv" >/dev/null

# fresh_box starts a new container as $name that trusts relsrv's cert and
# holds the script at /install.sh.
fresh_box() {
  # With a dozen systemd containers up at once, ubuntu's systemd never came
  # up in the last one, so the previous case's box goes first.
  if [ -n "${name:-}" ] && [ -z "${KEEP:-}" ]; then docker rm -f "$name" >/dev/null 2>&1 || true; fi
  name="jumpgate-e2e-install-$$-$1"
  names+=("$name")
  # shellcheck disable=SC2086 # E2E_DOCKER_FLAGS is a list of flags
  docker run -d --name "$name" --network "$net" ${E2E_DOCKER_FLAGS:---privileged} jumpgate-e2e >/dev/null
  local ready='' state=''
  for _ in $(seq 1 60); do
    state="$(docker exec "$name" systemctl is-system-running 2>/dev/null || true)"
    if [ "$state" = running ] || [ "$state" = degraded ]; then ready=1; break; fi
    sleep 1
  done
  [ -n "$ready" ] || { docker logs "$name" >&2 || true; fail "container $name never became ready (${state:-unknown})"; }
  docker cp -q "$work/relsrv.crt" "$name:/usr/local/share/ca-certificates/relsrv.crt"
  docker exec "$name" update-ca-certificates >/dev/null 2>&1
  if [ -e "$script" ]; then docker cp -q "$script" "$name:/install.sh"; fi
}

# run_as user cmd...: no tty, the base URL pointed at relsrv; sets $out, $rc.
run_as() {
  local u="$1"
  shift
  rc=0
  out="$(docker exec -u "$u" -e JUMPGATE_INSTALL_BASE_URL=https://relsrv:8443 "$name" "$@" 2>&1)" || rc=$?
  printf '%s\n' "$out" | sed 's/^/    /'
}
want() { case "$out" in *"$1"*) ;; *) fail "$2: output lacks '$1'" ;; esac; }
no_tmp_left() {
  local n
  n="$(docker exec "$name" sh -c 'ls /tmp | grep -c jumpgate-install || true')"
  [ "$n" = 0 ] || fail "$1: $n jumpgate-install temp dirs left in /tmp"
}

echo "== 1 a truncated script runs nothing"
fresh_box trunc
[ -e "$script" ] || fail "$script: No such file"
run_as root sh -c 'head -c 200 /install.sh | sh'
[ "$rc" = 0 ] || fail "truncated: exit $rc"
docker exec "$name" test ! -e /usr/local/bin/jumpgate || fail "truncated: jumpgate installed"
ok "truncated download"

echo "== 2 a tampered archive"
fresh_box tamper
run_as root sh /install.sh --version v0.0.1-tampered --yes
[ "$rc" != 0 ] || fail "tampered: exit 0"
want checksum tampered
no_tmp_left tampered
docker exec "$name" test ! -e /usr/local/bin/jumpgate || fail "tampered: jumpgate installed"
ok "tampered archive refused"

echo "== 3 --require-provenance without gh"
fresh_box prov
run_as root sh /install.sh --version v0.0.1 --yes --require-provenance
[ "$rc" != 0 ] || fail "provenance: exit 0"
want provenance provenance
ok "provenance required"

echo "== 3b with gh signed in, provenance pins the workflow and the tag"
fresh_box ghprov
# A stand-in gh: signed in, every attestation passes, its arguments logged.
docker exec "$name" sh -c 'printf "%s\n" "#!/bin/sh" "echo \"\$*\" >>/tmp/gh.log" "exit 0" >/usr/local/bin/gh && chmod 0755 /usr/local/bin/gh'
run_as root sh /install.sh --version v0.0.1 --yes --require-provenance
[ "$rc" = 0 ] || fail "gh provenance: exit $rc"
want "build provenance verified" "gh provenance"
out="$(docker exec "$name" cat /tmp/gh.log)"
want "attestation verify" "gh provenance"
want "--repo jumpgate-tech/jumpgate-app" "gh provenance"
want "--signer-workflow jumpgate-tech/jumpgate-app/.github/workflows/release.yml" "gh provenance"
want "--source-ref refs/tags/v0.0.1" "gh provenance"
ok "provenance pins the signer workflow and the tag"

echo "== 4 pinned and latest, as root"
fresh_box pin
run_as root sh /install.sh --version 0.0.1 --yes
[ "$rc" = 0 ] || fail "pinned: exit $rc"
run_as root /usr/local/bin/jumpgate
[ "$out" = v0.0.1 ] || fail "pinned: binary prints '$out'"
run_as root sh /install.sh --yes
[ "$rc" = 0 ] || fail "latest: exit $rc"
want "jumpgate v0.0.2" latest
run_as root /usr/local/bin/jumpgate
[ "$out" = v0.0.2 ] || fail "latest: binary prints '$out'"
no_tmp_left latest
ok "pinned and latest"

echo "== 5 alice, no tty, no --yes"
fresh_box alice
run_as alice sh /install.sh --version v0.0.1
[ "$rc" = 1 ] || fail "alice: exit $rc, want 1"
want "sudo install" alice
docker exec "$name" test ! -e /usr/local/bin/jumpgate || fail "alice: jumpgate installed"
ok "no sudo without a tty"

echo "== 6 bob, --prefix"
fresh_box bob
run_as bob sh /install.sh --version v0.0.1 --prefix /home/bob/.local
[ "$rc" = 0 ] || fail "bob: exit $rc"
docker exec "$name" test -x /home/bob/.local/bin/jumpgate || fail "bob: no /home/bob/.local/bin/jumpgate"
ok "--prefix without root"

echo "== 7 an unsupported arch"
fresh_box arch
docker exec "$name" sh -c 'mkdir -p /shim && printf "%s\n" "#!/bin/sh" "if [ \"\$1\" = -m ]; then echo mips64; else exec /bin/uname \"\$@\"; fi" >/shim/uname && chmod 0755 /shim/uname'
run_as root env PATH=/shim:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin sh /install.sh --yes
[ "$rc" != 0 ] || fail "arch: exit 0"
want mips64 arch
want "amd64 or arm64" arch
ok "unsupported arch refused"

echo "== 8 a package owns /usr/bin/jumpgate"
fresh_box pkg
docker exec "$name" sh -c 'set -e; d=/tmp/pkg; mkdir -p $d/DEBIAN $d/usr/bin
printf "#!/bin/sh\necho pkg\n" >$d/usr/bin/jumpgate; chmod 0755 $d/usr/bin/jumpgate
printf "Package: jumpgate\nVersion: 0.0.0\nArchitecture: all\nMaintainer: e2e <e2e@invalid>\nDescription: dummy\n" >$d/DEBIAN/control
dpkg-deb --build $d /tmp/jumpgate.deb >/dev/null; dpkg -i /tmp/jumpgate.deb >/dev/null; rm -rf $d /tmp/jumpgate.deb'
run_as root sh /install.sh --version v0.0.1 --yes
[ "$rc" != 0 ] || fail "package: exit 0"
want "apt upgrade" package
ok "package-owned binary refused"

echo "== 9 --controller as root"
fresh_box ctl
run_as root sh /install.sh --controller --version v0.0.2-real --yes
[ "$rc" = 0 ] || fail "controller: exit $rc"
want "can sign routine intents for every box" controller
want "jumpgate setup server" controller
docker exec "$name" test -f /etc/systemd/system/jumpgate.service || fail "controller: no jumpgate.service"
docker exec "$name" getent group jumpgate-admin >/dev/null || fail "controller: no jumpgate-admin group"
enabled="$(docker exec "$name" systemctl is-enabled jumpgate 2>/dev/null || true)"
[ "$enabled" != enabled ] || fail "controller: jumpgate.service is enabled"
ok "--controller installs the service, not enabled"

echo "== 10 --agent without --transport-key"
fresh_box agentargs
run_as root sh /install.sh --agent --controller 0x0000000000000000000000000000000000000001 --version v0.0.2-real --yes
[ "$rc" != 0 ] || fail "agent args: exit 0"
want --transport-key "agent args"
docker exec "$name" test ! -e /usr/local/lib/jumpgate/jumpgate || fail "agent args: gate installed"
ok "--agent needs --transport-key"

echo "== 11 --agent sets the gate up for a controller"
fresh_box agent
ssh-keygen -q -t ed25519 -N '' -C jumpgate-controller -f "$work/transport" # throwaway
tkey="$(cat "$work/transport.pub")"
run_as root sh /install.sh --agent --controller 0x0000000000000000000000000000000000000001 \
  --transport-key "$tkey" --version v0.0.2-real --yes
[ "$rc" = 0 ] || fail "agent: exit $rc"
want "on the controller, run: jumpgate hosts add NAME --ssh jumpgate@" agent
want "--installed 0x" agent
docker exec "$name" test -x /usr/local/lib/jumpgate/jumpgate || fail "agent: no /usr/local/lib/jumpgate/jumpgate"
docker exec "$name" test ! -e /usr/local/bin/jumpgate || fail "agent: the CLI was installed too"
docker exec "$name" grep -qF "$(cut -d' ' -f2 "$work/transport.pub")" /var/lib/jumpgate/home/.ssh/authorized_keys ||
  fail "agent: the transport key is not authorized"
ok "--agent installs and pairs the gate"

echo "e2e-install: OK"
