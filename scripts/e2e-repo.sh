#!/usr/bin/env bash
# scripts/e2e-repo.sh — the signed apt and dnf repository, end to end, with a
# throwaway key: build two snapshot releases' deb and rpm packages, sign a
# repo with scripts/build-pkg-repo.sh in a builder container, serve it over
# HTTP on a docker network, and check that apt (debian:12, ubuntu:24.04) and
# dnf (fedora:41) install from it, that an upgrade does not restart a running
# service, and that a tampered index is refused. The key is generated in the
# builder and dies with it. Needs Docker and goreleaser.
#   KEEP=1 scripts/e2e-repo.sh   # keep the containers to inspect
set -euo pipefail
work="$(mktemp -d)"
net="jumpgate-e2e-repo-$$"
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
[ -x scripts/build-pkg-repo.sh ] || fail "scripts/build-pkg-repo.sh: No such file"

# --- packages: two snapshot releases --------------------------------------
for v in v0.0.1 v0.0.2; do
  echo "== building $v packages"
  GORELEASER_CURRENT_TAG="$v" goreleaser release --snapshot --clean --skip=publish >"$work/goreleaser-$v.log" 2>&1 ||
    { tail -20 "$work/goreleaser-$v.log" >&2; fail "goreleaser $v"; }
  mkdir -p "$work/$v"
  cp dist/*.deb dist/*.rpm "$work/$v/"
done

# --- builder: signs the repo, then serves it -------------------------------
docker build -q -t jumpgate-e2e-pkgbuild - >/dev/null <<'EOF'
FROM debian:12
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      aptly createrepo-c rpm gnupg python3 ca-certificates \
    && rm -rf /var/lib/apt/lists/*
EOF
docker network create "$net" >/dev/null
pkg="jumpgate-e2e-repo-$$-pkg"
names+=("$pkg")
docker run -d --name "$pkg" --network "$net" --network-alias pkg jumpgate-e2e-pkgbuild \
  sh -c 'mkdir -p /repo0 && ln -sfn /repo0 /www && exec python3 -m http.server 8000 --directory /www' >/dev/null
docker exec "$pkg" mkdir -p /src/scripts /src/packaging /in1 /in2
docker cp -q scripts/build-pkg-repo.sh "$pkg:/src/scripts/build-pkg-repo.sh"
docker cp -q packaging/repo "$pkg:/src/packaging/repo"
for f in "$work"/v0.0.1/*; do docker cp -q "$f" "$pkg:/in1/"; done
for f in "$work"/v0.0.1/* "$work"/v0.0.2/*; do docker cp -q "$f" "$pkg:/in2/"; done
docker exec "$pkg" sh -c 'set -e
umask 077
export GNUPGHOME=/gnupg; mkdir -p $GNUPGHOME
head -c 24 /dev/urandom | od -An -tx1 | tr -d " \n" >/pass
gpg --batch --pinentry-mode loopback --passphrase-file /pass --quick-gen-key "jumpgate e2e <e2e@invalid>" ed25519 sign 1d 2>/dev/null'
keyid="$(docker exec -e GNUPGHOME=/gnupg "$pkg" sh -c 'gpg --list-keys --with-colons | awk -F: "/^fpr/ {print \$10; exit}"')"
[ -n "$keyid" ] || fail "no throwaway key"

build_repo() { # in-dir out-dir
  docker exec -e GNUPGHOME=/gnupg -e PKG_SIGNING_PASSPHRASE_FILE=/pass -w /src "$pkg" \
    scripts/build-pkg-repo.sh --in "$1" --out "$2" --key-id "$keyid"
}
serve() { docker exec "$pkg" ln -sfn "$1" /www; }

echo "== build-pkg-repo, v0.0.1"
build_repo /in1 /repo1
for f in apt/dists/stable/InRelease apt/dists/stable/Release apt/dists/stable/Release.gpg rpm/repodata/repomd.xml rpm/repodata/repomd.xml.asc jumpgate.asc jumpgate.repo; do
  docker exec "$pkg" test -f "/repo1/$f" || fail "repo lacks $f"
done
docker exec "$pkg" sh -c 'ls /repo1/apt/pool/main/j/jumpgate/ | grep -q _amd64.deb && ls /repo1/apt/pool/main/j/jumpgate/ | grep -q _arm64.deb' ||
  fail "apt pool lacks a deb per arch"
serve /repo1
ok "repo built and signed"

# The test address for pkg.jumpgate.app.
sed 's|https://pkg.jumpgate.app|http://pkg:8000|g' packaging/repo/jumpgate.sources >"$work/jumpgate.sources"
sed 's|https://pkg.jumpgate.app|http://pkg:8000|g' packaging/repo/jumpgate.repo >"$work/jumpgate.repo"
docker exec "$pkg" sh -c 'gpg --dearmor </repo1/jumpgate.asc >/keyring.gpg'
docker cp -q "$pkg:/keyring.gpg" "$work/keyring.gpg"

wait_ready() {
  local ready='' state=''
  for _ in $(seq 1 60); do
    state="$(docker exec "$1" systemctl is-system-running 2>/dev/null || true)"
    if [ "$state" = running ] || [ "$state" = degraded ]; then ready=1; break; fi
    sleep 1
  done
  [ -n "$ready" ] || { docker logs "$1" >&2 || true; fail "container $1 never became ready (${state:-unknown})"; }
}

apt_box() { # label base
  local n="jumpgate-e2e-repo-$$-$1"
  names+=("$n")
  docker build -q -t "jumpgate-e2e-$1" --build-arg BASE="$2" scripts/e2e >/dev/null
  docker run -d --name "$n" --network "$net" --privileged "jumpgate-e2e-$1" >/dev/null
  wait_ready "$n"
  docker cp -q "$work/keyring.gpg" "$n:/usr/share/keyrings/jumpgate-archive-keyring.gpg"
  docker cp -q "$work/jumpgate.sources" "$n:/etc/apt/sources.list.d/jumpgate.sources"
  docker exec "$n" chmod 0644 /usr/share/keyrings/jumpgate-archive-keyring.gpg /etc/apt/sources.list.d/jumpgate.sources
  box="$n"
}

box_debian=''
box_ubuntu=''
for b in debian:12 ubuntu:24.04; do
  label="${b%%:*}"
  echo "== 1 apt install on $b"
  apt_box "$label" "$b"
  if [ "$label" = debian ]; then box_debian="$box"; else box_ubuntu="$box"; fi
  docker exec "$box" sh -c 'apt-get update -q && apt-get install -y -q jumpgate' >"$work/apt-$label.log" 2>&1 ||
    { cat "$work/apt-$label.log" >&2; fail "$b: apt-get install jumpgate"; }
  docker exec "$box" test -x /usr/bin/jumpgate || fail "$b: no /usr/bin/jumpgate"
  docker exec "$box" getent passwd jumpgate-ctl >/dev/null || fail "$b: no jumpgate-ctl user"
  enabled="$(docker exec "$box" systemctl is-enabled jumpgate 2>/dev/null || true)"
  [ "$enabled" != enabled ] || fail "$b: the package enabled jumpgate.service"
  ok "$b: apt install"
done

echo "== 2 an upgrade does not restart the service"
docker exec "$pkg" sh -c 'cp -a /repo1 /repo-old'
build_repo /in2 /repo2
serve /repo2
for label in debian ubuntu; do
  box="$box_debian"
  [ "$label" = debian ] || box="$box_ubuntu"
  docker exec "$box" systemctl start jumpgate || { docker exec "$box" journalctl -u jumpgate --no-pager >&2 || true; fail "$label: start jumpgate"; }
  sleep 1
  before="$(docker exec "$box" systemctl show -p MainPID --value jumpgate)"
  v1="$(docker exec "$box" dpkg-query -W -f '${Version}' jumpgate)"
  [ "$before" != 0 ] || fail "$label: jumpgate is not running"
  docker exec "$box" sh -c 'apt-get update -q && apt-get install -y -q --only-upgrade jumpgate' >"$work/up-$label.log" 2>&1 ||
    { cat "$work/up-$label.log" >&2; fail "$label: apt-get upgrade"; }
  v2="$(docker exec "$box" dpkg-query -W -f '${Version}' jumpgate)"
  [ "$v1" != "$v2" ] || fail "$label: still at $v1 after the upgrade"
  after="$(docker exec "$box" systemctl show -p MainPID --value jumpgate)"
  [ "$before" = "$after" ] || fail "$label: MainPID $before became $after; the upgrade restarted the service"
  ok "$label: $v1 -> $v2, MainPID $before kept"
done

echo "== 3 a tampered InRelease"
docker exec "$pkg" sh -c 'cp -a /repo2 /repo3 && python3 - /repo3/apt/dists/stable/InRelease <<"PY"
import sys
p = sys.argv[1]
b = bytearray(open(p, "rb").read())
i = b.index(b"Suite: stable") + len("Suite: ")
b[i] ^= 0x01
open(p, "wb").write(bytes(b))
PY'
serve /repo3
box="$box_debian"
rc=0
# apt refuses the new index but would keep the old one and exit 0; --error-on=any
# makes that refusal the exit status.
out="$(docker exec "$box" apt-get update --error-on=any 2>&1)" || rc=$?
printf '%s\n' "$out" | sed 's/^/    /'
[ "$rc" != 0 ] || fail "apt-get update accepted a tampered InRelease"
case "$out" in *NOSPLIT* | *BADSIG* | *"not signed"*) ;; *) fail "apt-get update: no signature error" ;; esac
ok "tampered InRelease refused"

echo "== 4 dnf on fedora:41"
serve /repo2
fed="jumpgate-e2e-repo-$$-fedora"
names+=("$fed")
docker build -q -t jumpgate-e2e-fedora -f scripts/e2e/Dockerfile.fedora scripts/e2e >/dev/null
docker run -d --name "$fed" --network "$net" --privileged jumpgate-e2e-fedora >/dev/null
wait_ready "$fed"
docker cp -q "$work/jumpgate.repo" "$fed:/etc/yum.repos.d/jumpgate.repo"
docker exec "$fed" chmod 0644 /etc/yum.repos.d/jumpgate.repo
docker exec "$fed" dnf install -y jumpgate >"$work/dnf.log" 2>&1 || { cat "$work/dnf.log" >&2; fail "dnf install jumpgate"; }
docker exec "$fed" test -x /usr/bin/jumpgate || fail "fedora: no /usr/bin/jumpgate"
docker exec "$fed" getent passwd jumpgate-ctl >/dev/null || fail "fedora: no jumpgate-ctl user"
ok "fedora: dnf install with repo_gpgcheck=1"
docker exec "$pkg" sh -c 'cp -a /repo2 /repo4 && sed -i "s|<revision>|<revision>9|" /repo4/rpm/repodata/repomd.xml'
serve /repo4
docker exec "$fed" dnf clean all >/dev/null 2>&1
rc=0
# dnf skips a repo it cannot verify and exits 0; skip_if_unavailable=0 makes
# the refusal the exit status.
out="$(docker exec "$fed" dnf makecache --repo jumpgate --setopt=jumpgate.skip_if_unavailable=0 2>&1)" || rc=$?
printf '%s\n' "$out" | sed 's/^/    /'
[ "$rc" != 0 ] || fail "dnf makecache accepted a tampered repomd.xml"
case "$out" in *signature*) ;; *) fail "dnf makecache: no signature error" ;; esac
ok "tampered repomd.xml refused"

echo "e2e-repo: OK"
