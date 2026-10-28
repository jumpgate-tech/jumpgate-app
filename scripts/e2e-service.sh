#!/usr/bin/env bash
# scripts/e2e-service.sh — the jumpgate service end to end, on systemd boxes
# from scripts/e2e/Dockerfile on one docker network: A runs the service, B is
# a box that pulls its gate with install.sh --agent, A2 is a --self-only
# controller. Checks attach and admission (jumpgate-admin), locking and
# unlocking, a sealed host key, the login link through an ssh tunnel (and a
# stranger's tunnel), pull pairing, hosts upgrade, and --self-only. Each check
# prints `ok NAME`; the first failure prints `FAIL NAME` and exits 1.
# The controller is built as goreleaser builds it, with the gates embedded.
# Its versions look released (v0.0.90, then v0.0.91 for hosts upgrade):
# hosts invite refuses a development build, and a -e2e suffix is one.
# Needs Docker.
#   BASE=ubuntu:24.04 scripts/e2e-service.sh   # the second distro
#   KEEP=1 scripts/e2e-service.sh              # keep the containers to inspect
set -euo pipefail
base="${BASE:-debian:12}"
work="$(mktemp -d)"
net="jumpgate-e2e-service-$$"
pass='correct horse battery staple'
embedded=internal/agentbin/embedded
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
  # Put back whatever gates the developer had embedded before this run.
  if [ -d "$work/embedded-saved" ]; then
    find "$embedded" -mindepth 1 ! -name .gitkeep -delete
    cp -pR "$work/embedded-saved/." "$embedded/"
  fi
  rm -rf "$work"
}
trap cleanup EXIT

fail() { echo "FAIL $*" >&2; exit 1; }
ok() { echo "ok $*"; }
sha() { if command -v sha256sum >/dev/null; then sha256sum "$@"; else shasum -a 256 "$@"; fi; }

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
docker network create "$net" >/dev/null
case "$(docker run --rm --entrypoint uname jumpgate-e2e -m)" in
  x86_64) goarch=amd64 ;;
  aarch64) goarch=arm64 ;;
  *) fail "unsupported container arch" ;;
esac

mkdir -p "$work/embedded-saved"
cp -pR "$embedded/." "$work/embedded-saved/"

# build_controller VERSION OUT: the controller as goreleaser builds it, and
# the uncompressed gate it embeds for the boxes' arch at OUT.gate.
build_controller() {
  find "$embedded" -mindepth 1 ! -name .gitkeep -delete
  env GZIP=1 VERSION="$1" bash scripts/build-agents.sh "$embedded" >/dev/null
  cp "$embedded/jumpgate-linux-$goarch" "$2.gate"
  CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build -tags embedagents \
    -ldflags "-X github.com/valve-tech/jumpgate/internal/buildinfo.version=$1" -o "$2" ./cmd/jumpgate
}
build_controller v0.0.90 "$work/jumpgate"
build_controller v0.0.91 "$work/jumpgate-next"
CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build -o "$work/relsrv" ./scripts/e2e/relsrv

# --- the release server: v0.0.90 as a release ships it -----------------
rel="$work/rel/download/v0.0.90"
mkdir -p "$rel" "$work/pack"
cp "$work/jumpgate" "$work/pack/jumpgate"
x=()
[ "$(uname -s)" != Darwin ] || x=(--no-mac-metadata --no-xattrs)
COPYFILE_DISABLE=1 tar ${x[@]+"${x[@]}"} -C "$work/pack" -czf "$rel/jumpgate_linux_${goarch}.tar.gz" jumpgate
cp "$work/jumpgate.gate" "$rel/jumpgate-linux-$goarch"
(cd "$rel" && sha "jumpgate_linux_${goarch}.tar.gz" "jumpgate-linux-$goarch" >checksums.txt)

gen="jumpgate-e2e-service-$$-cert"
names+=("$gen")
docker create --name "$gen" jumpgate-e2e openssl req -x509 -newkey ec \
  -pkeyopt ec_paramgen_curve:prime256v1 -nodes -days 1 -subj /CN=relsrv \
  -addext subjectAltName=DNS:relsrv -addext basicConstraints=critical,CA:TRUE \
  -keyout /relsrv.key -out /relsrv.crt >/dev/null
docker start -a "$gen" >/dev/null 2>&1
docker cp -q "$gen:/relsrv.key" "$work/relsrv.key"
docker cp -q "$gen:/relsrv.crt" "$work/relsrv.crt"
srv="jumpgate-e2e-service-$$-relsrv"
names+=("$srv")
docker create --name "$srv" --network "$net" --network-alias relsrv jumpgate-e2e \
  /relsrv -dir /rel -cert /relsrv.crt -key /relsrv.key -addr :8443 >/dev/null
docker cp -q "$work/relsrv" "$srv:/relsrv"
docker cp -q "$work/rel" "$srv:/rel"
docker cp -q "$work/relsrv.crt" "$srv:/relsrv.crt"
docker cp -q "$work/relsrv.key" "$srv:/relsrv.key"
docker start "$srv" >/dev/null

# box ALIAS: a systemd box on the network, reachable as ALIAS; prints its name.
box() {
  local n="jumpgate-e2e-service-$$-$1" ready='' state=''
  names+=("$n")
  # shellcheck disable=SC2086 # E2E_DOCKER_FLAGS is a list of flags
  docker run -d --name "$n" --hostname "$1" --network "$net" --network-alias "$1" \
    ${E2E_DOCKER_FLAGS:---privileged} jumpgate-e2e >/dev/null
  for _ in $(seq 1 60); do
    state="$(docker exec "$n" systemctl is-system-running 2>/dev/null || true)"
    if [ "$state" = running ] || [ "$state" = degraded ]; then ready=1; break; fi
    sleep 1
  done
  [ -n "$ready" ] || { docker logs "$n" >&2 || true; fail "boot: $n (${state:-unknown})"; }
  docker cp -q "$work/relsrv.crt" "$n:/usr/local/share/ca-certificates/relsrv.crt"
  docker exec "$n" update-ca-certificates >/dev/null 2>&1
  docker exec "$n" systemctl start ssh >/dev/null 2>&1 || docker exec "$n" systemctl start ssh.socket >/dev/null 2>&1 || true
}
A="jumpgate-e2e-service-$$-a"
B="jumpgate-e2e-service-$$-b"
A2="jumpgate-e2e-service-$$-a2"

# install_controller NAME FILE: root-owned in /usr/local/bin, as install.sh puts it.
install_controller() {
  docker exec "$1" mkdir -p /var/lib/jumpgate/staging
  docker cp -q "$2" "$1:/var/lib/jumpgate/staging/jumpgate.new"
  docker exec "$1" sh -c 'install -o root -g root -m 0755 /var/lib/jumpgate/staging/jumpgate.new /usr/local/bin/jumpgate && rm -f /var/lib/jumpgate/staging/jumpgate.new'
}

# in NAME USER cmd...: run as USER with its home; sets $out and $rc.
in_box() {
  local n="$1" u="$2" h
  shift 2
  if [ "$u" = root ]; then h=/root; else h="/home/$u"; fi
  rc=0
  out="$(docker exec -i -u "$u" -w "$h" -e HOME="$h" -e USER="$u" "$n" "$@" 2>&1)" || rc=$?
  printf '%s\n' "$out" | sed 's/^/    /'
}
# admin NAME cmd: run a shell command as alice with jumpgate-admin applied.
admin() { in_box "$1" alice sg jumpgate-admin -c "$2"; }
# tty NAME ANSWERS cmd: as alice on a terminal (host keys need one), typing ANSWERS.
tty() {
  in_box "$1" alice sh -c "(sleep 4; printf '$2') | script -qec \"sg jumpgate-admin -c '$3'\" /dev/null"
}
want() { case "$out" in *"$1"*) ;; *) fail "$2: output lacks '$1'" ;; esac; }
status_ok() { # NAME TARGET
  local i
  for i in $(seq 1 20); do
    admin "$1" "jumpgate status $2" >"$work/status.out"
    # A signed not_set_up is an answer: socket, admission and signature worked.
    { [ "$rc" = 0 ] || grep -q "no node is set up" "$work/status.out"; } && return 0
    sleep 1
  done
  cat "$work/status.out" >&2; return 1
}

# --- setup in A ------------------------------------------------------------
box a
install_controller "$A" "$work/jumpgate"
in_box "$A" root jumpgate setup install-service --bin /usr/local/bin/jumpgate
[ "$rc" = 0 ] || fail "install-service: exit $rc"
in_box "$A" alice sh -c "printf '%s\n' '$pass' | jumpgate setup server --stdin --name a"
[ "$rc" = 0 ] || fail "setup-server: exit $rc"

# --- attach and admission --------------------------------------------------
status_ok "$A" a || fail status-answers
ok status-answers

in_box "$A" bob jumpgate hosts list
[ "$rc" != 0 ] || fail "bob-refused: exit 0"
want "usermod -aG jumpgate-admin" bob-refused
docker exec "$A" test ! -e /home/bob/.jumpgate/run || fail "bob-refused: /home/bob/.jumpgate/run exists"
ok bob-refused

# --- locking ---------------------------------------------------------------
docker exec "$A" systemctl restart jumpgate
sleep 2
admin "$A" "jumpgate status a"
[ "$rc" != 0 ] || fail "restart-locks: status answered"
want locked restart-locks
ok restart-locks

admin "$A" "printf '%s\n' '$pass' | jumpgate unlock --stdin"
[ "$rc" = 0 ] || fail "unlock: exit $rc"
status_ok "$A" a || fail unlock
ok unlock

docker exec "$A" mkdir -p /etc/credstore.encrypted
admin "$A" "printf '%s\n' '$pass' | sudo jumpgate vault seal --allow-host-key --stdin"
[ "$rc" = 0 ] || fail "seal-host-key: seal exit $rc"
docker exec "$A" systemctl restart jumpgate
status_ok "$A" a || fail "seal-host-key: no answer after restart"
ok seal-host-key

# --- tunnel login ----------------------------------------------------------
box b
docker exec "$B" sh -c 'ssh-keygen -q -t ed25519 -N "" -f /root/.ssh/id_ed25519'
pub="$(docker exec "$B" cat /root/.ssh/id_ed25519.pub)"
for u in alice bob; do
  docker exec -i "$A" sh -c "mkdir -p /home/$u/.ssh && cat >> /home/$u/.ssh/authorized_keys && chown -R $u:$u /home/$u/.ssh && chmod 700 /home/$u/.ssh && chmod 600 /home/$u/.ssh/authorized_keys" <<<"$pub"
done
# tunnel USER PORT: B forwards PORT to A's 127.0.0.1:8799 as USER.
tunnel() {
  docker exec -d "$B" ssh -o StrictHostKeyChecking=accept-new -o ExitOnForwardFailure=yes \
    -N -L "$2:127.0.0.1:8799" "$1@a"
  local i
  for i in $(seq 1 20); do
    docker exec "$B" curl -s -o /dev/null "http://127.0.0.1:$2/" && return 0
    sleep 1
  done
  fail "tunnel: $1's forward never answered"
}
login_code() {
  admin "$A" "jumpgate open --print" >/dev/null
  code="$(printf '%s\n' "$out" | grep -o 'login?code=[^[:space:]]*' | head -1 | cut -d= -f2)"
  [ -n "$code" ] || fail "$1: no login link from jumpgate open --print"
}
tunnel alice 8799
login_code login-tunnel
got="$(docker exec "$B" curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:8799/login?code=$code")"
[ "$got" = 200 ] || fail "login-tunnel: HTTP $got"
ok login-tunnel

tunnel bob 8800
login_code login-foreign-tunnel
got="$(docker exec "$B" curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:8800/login?code=$code")"
[ "$got" = 403 ] || fail "login-foreign-tunnel: HTTP $got"
ok login-foreign-tunnel

# --- pull, upgrade and invite ----------------------------------------------
admin "$A" "jumpgate hosts invite b"
[ "$rc" = 0 ] || fail "pull-pair: invite exit $rc"
ctl="$(printf '%s\n' "$out" | grep -o -- '--controller 0x[0-9a-fA-F]*' | head -1 | cut -d' ' -f2)"
tkey="$(printf '%s\n' "$out" | sed -n "s/.*--transport-key '\([^']*\)'.*/\1/p" | head -1)"
[ -n "$ctl" ] && [ -n "$tkey" ] || fail "pull-pair: the invite names no controller or transport key"
docker cp -q packaging/install/install.sh "$B:/install.sh"
rc=0
out="$(docker exec -e JUMPGATE_INSTALL_BASE_URL=https://relsrv:8443 "$B" sh /install.sh --agent \
  --version v0.0.90 --controller "$ctl" --transport-key "$tkey" --yes 2>&1)" || rc=$?
printf '%s\n' "$out" | sed 's/^/    /'
[ "$rc" = 0 ] || fail "pull-pair: install.sh --agent exit $rc"
gate="$(printf '%s\n' "$out" | grep -o -- '--installed 0x[0-9a-fA-F]*' | head -1 | cut -d' ' -f2)"
[ -n "$gate" ] || fail "pull-pair: install.sh printed no gate address"

tty "$A" 'yes\n' "jumpgate hosts add b --ssh jumpgate@b --installed 0x0000000000000000000000000000000000000001"
[ "$rc" != 0 ] || fail "pull-mismatch: exit 0"
want 'is not the one `jumpgate agent setup` printed' pull-mismatch # installed_mismatch
ok pull-mismatch

tty "$A" 'yes\n' "jumpgate hosts add b --ssh jumpgate@b --installed $gate"
[ "$rc" = 0 ] || fail "pull-pair: hosts add exit $rc"
admin "$A" "jumpgate status b"
case "$out" in *"no node is set up"*) ;; *) [ "$rc" = 0 ] || fail "pull-pair: status b exit $rc" ;; esac
ok pull-pair

install_controller "$A" "$work/jumpgate-next"
docker exec "$A" systemctl restart jumpgate
status_ok "$A" a || fail "upgrade: the new controller does not answer"
admin "$A" "jumpgate hosts upgrade b"
[ "$rc" = 0 ] || fail "upgrade: exit $rc"
want "-> v0.0.91" upgrade
wantsum="$(sha "$work/jumpgate-next.gate" | cut -d' ' -f1)"
gotsum=
for _ in $(seq 1 20); do
  gotsum="$(docker exec "$B" sha256sum /usr/local/lib/jumpgate/jumpgate | cut -d' ' -f1)"
  [ "$gotsum" = "$wantsum" ] && break
  sleep 1
done
[ "$gotsum" = "$wantsum" ] || fail "upgrade: B's gate is $gotsum, want $wantsum"
ok upgrade

# --- self-only -------------------------------------------------------------
box a2
install_controller "$A2" "$work/jumpgate"
in_box "$A2" root jumpgate setup install-service --bin /usr/local/bin/jumpgate
[ "$rc" = 0 ] || fail "self-only: install-service exit $rc"
in_box "$A2" alice sh -c "printf '%s\n' '$pass' | jumpgate setup server --stdin --name a2 --self-only"
[ "$rc" = 0 ] || fail "self-only: setup server exit $rc"
tty "$A2" 'yes\n' "jumpgate hosts add x --ssh root@b"
[ "$rc" != 0 ] || fail "self-only: hosts add exit 0"
want self_only "self-only (hosts add)"
admin "$A2" "jumpgate hosts invite x"
[ "$rc" != 0 ] || fail "self-only: hosts invite exit 0"
want self_only "self-only (hosts invite)"
ok self-only

echo "e2e-service: OK"
