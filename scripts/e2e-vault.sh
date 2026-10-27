#!/usr/bin/env bash
# scripts/e2e-vault.sh — seal alice's vault with systemd-creds on a real
# systemd box and check that `jumpgate serve`, started with the credential,
# comes up unlocked, and without it, locked. Needs Docker. The TPM2 run is
# manual (spec). BASE=ubuntu:24.04 for the second distro; KEEP=1 keeps the box.
set -euo pipefail
base="${BASE:-debian:12}"
work="$(mktemp -d)"
name="jumpgate-e2e-vault-$$"
pass='correct horse battery staple'
cleanup() {
  if [ -n "${KEEP:-}" ]; then
    echo "kept container $name (docker rm -f $name when done)" >&2
  else
    docker rm -f "$name" >/dev/null 2>&1 || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT
fail() { echo "FAIL $*" >&2; exit 1; }

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
# shellcheck disable=SC2086 # E2E_DOCKER_FLAGS is a list of flags
docker run -d --name "$name" ${E2E_DOCKER_FLAGS:---privileged} jumpgate-e2e >/dev/null
ready=
for _ in $(seq 1 60); do
  state="$(docker exec "$name" systemctl is-system-running 2>/dev/null || true)"
  if [ "$state" = running ] || [ "$state" = degraded ]; then ready=1; break; fi
  sleep 1
done
[ -n "$ready" ] || { docker logs "$name" >&2 || true; fail "container never booted (${state:-unknown})"; }

# 1. jumpgate for the container's arch.
case "$(docker exec "$name" uname -m)" in
  x86_64) arch=amd64 ;;
  aarch64) arch=arm64 ;;
  *) fail "unsupported container arch" ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -tags notui -o "$work/jumpgate" ./cmd/jumpgate
docker cp -q "$work/jumpgate" "$name:/usr/local/bin/jumpgate"
docker exec "$name" chmod 0755 /usr/local/bin/jumpgate

as_alice() { docker exec -i -u alice -w /home/alice -e HOME=/home/alice "$name" "$@"; }
vault=/home/alice/.jumpgate/vault
owner() { docker exec "$name" stat -c %U "$vault"; }

# 2. alice makes her vault.
printf '%s\n' "$pass" | as_alice jumpgate vault init --stdin --keystore none >/dev/null
[ "$(owner)" = alice ] || fail "vault owner after init: $(owner)"

# 3. Root must not seal (write) alice's vault: it refuses, and the vault
# stays hers. alice seals it; only systemd-creds encrypt runs as root (sudo).
if printf '%s\n' "$pass" | docker exec -i -e HOME=/home/alice "$name" \
    jumpgate vault seal --allow-host-key --stdin >"$work/root-seal" 2>&1; then
  cat "$work/root-seal" >&2
  fail "root sealed alice's vault"
fi
grep -q "as the vault's owner" "$work/root-seal" || { cat "$work/root-seal" >&2; fail "root's refusal does not say how to seal"; }
[ "$(owner)" = alice ] || fail "vault owner after root's seal attempt: $(owner)"
docker exec "$name" mkdir -p /etc/credstore.encrypted
printf '%s\n' "$pass" | as_alice jumpgate vault seal --allow-host-key --stdin >"$work/seal" 2>&1 ||
  { cat "$work/seal" >&2; fail "alice could not seal"; }
grep -q "sealed the vault key" "$work/seal" || { cat "$work/seal" >&2; fail "seal output"; }
[ "$(owner)" = alice ] || fail "vault owner after seal: $(owner)"
[ "$(docker exec "$name" stat -c %U /home/alice/.jumpgate)" = alice ] || fail "~/.jumpgate is no longer alice's"
docker exec "$name" test -s /etc/credstore.encrypted/jumpgate-vault-key || fail "no credential written"
echo "PASS seal as the owner (vault still alice's)"

# 4. The service, with the credential.
start() {
  docker exec "$name" systemctl stop jg-vault-e2e >/dev/null 2>&1 || true
  docker exec "$name" systemctl reset-failed jg-vault-e2e >/dev/null 2>&1 || true
  docker exec "$name" rm -f /home/alice/.jumpgate/run/server.json
  docker exec "$name" systemd-run --quiet --uid=alice -E HOME=/home/alice \
    -p LoadCredentialEncrypted=jumpgate-vault-key --unit=jg-vault-e2e \
    /usr/local/bin/jumpgate serve --no-open >/dev/null
}
# 5. Poll GET /api/vault as alice with the token from server.json.
vault_state() {
  as_alice python3 -c '
import json, sys, urllib.request
info = json.load(open("/home/alice/.jumpgate/run/server.json"))
req = urllib.request.Request("http://" + info["httpAddr"] + "/api/vault",
                             headers={"Authorization": "Bearer " + info["token"]})
print(json.load(urllib.request.urlopen(req, timeout=5))["state"])' 2>/dev/null || true
}
wait_state() {
  local want="$1" got=
  for _ in $(seq 1 60); do
    got="$(vault_state)"
    [ "$got" = "$want" ] && return 0
    sleep 1
  done
  docker exec "$name" journalctl -u jg-vault-e2e --no-pager >&2 || true
  fail "vault state ${got:-none}, want $want"
}
start
wait_state unlocked
echo "PASS systemd-creds unlock at start"

# 6. Remove the credential and restart: systemd finds none to load, the
# server starts, and the vault stays locked (the passphrase still works).
docker exec "$name" rm -f /etc/credstore.encrypted/jumpgate-vault-key
start
wait_state locked
[ "$(owner)" = alice ] || fail "vault owner at the end: $(owner)"
echo "PASS locked without the credential"
