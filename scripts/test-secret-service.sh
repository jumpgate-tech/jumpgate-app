#!/usr/bin/env bash
# scripts/test-secret-service.sh — the Linux key store against a REAL
# secret-tool in a debian:12 container (spec D3, D17), one case per bus setup:
#   no D-Bus session          -> the default is a key file; keychain names --store file
#   a hung secret-tool        -> cut off at the timeout, its process group killed
#   D-Bus, no Secret Service  -> the default is a key file
#   unlocked gnome-keyring    -> keychain round trip; a locked keyring is not absence
# Each case must PASS, not skip: the script fails if a test did not run.
#
#   scripts/test-secret-service.sh            # in a fresh container (Docker)
#   scripts/test-secret-service.sh --inside   # already in a disposable debian:12
#                                             # Go image as root (GitLab CI)
set -euo pipefail

if [ "${1:-}" != "--inside" ]; then
  image="${GO_IMAGE:-golang:1.25-bookworm}"
  docker volume create jumpgate-gocache >/dev/null
  exec docker run --rm --init \
    -v "$PWD":/src -w /src \
    -v jumpgate-gocache:/cache \
    -e HOME=/tmp/home -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod -e GOFLAGS=-buildvcs=false \
    "$image" bash scripts/test-secret-service.sh --inside
fi

mkdir -p "${HOME:-/tmp/home}"
apt-get update -qq
apt-get install -y -qq --no-install-recommends libsecret-tools gnome-keyring dbus systemd >/dev/null
dpkg-query -W libsecret-tools gnome-keyring

# run_expect REGEX TEST... : run the matching tests verbosely and require a
# PASS line for each named test, so a silent skip fails the script.
run_expect() {
  local re=$1; shift
  local out
  out=$(go test -count=1 -tags realsecret -run "$re" -v ./internal/signer/ 2>&1) || { echo "$out"; exit 1; }
  echo "$out" | grep -E "^(--- |ok|FAIL)"
  for name in "$@"; do
    echo "$out" | grep -q -- "--- PASS: $name " || { echo "$out"; echo "!! $name did not pass"; exit 1; }
  done
}
export -f run_expect

echo "== no D-Bus session; hung secret-tool"
(
  unset DBUS_SESSION_BUS_ADDRESS XDG_RUNTIME_DIR
  run_expect "^(TestRealSecretServiceDefaultWithoutDBus|TestRealSecretToolTimeoutKillsItsProcessGroup|TestRealSecretServiceProbeTimesOut)$" \
    TestRealSecretServiceDefaultWithoutDBus TestRealSecretToolTimeoutKillsItsProcessGroup TestRealSecretServiceProbeTimesOut
)

echo "== D-Bus session with no Secret Service"
conf=$(mktemp)
trap 'rm -f "$conf"' EXIT
cat >"$conf" <<'EOF'
<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:tmpdir=/tmp</listen>
  <auth>EXTERNAL</auth>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>
EOF
JUMPGATE_TEST_NO_SECRET_SERVICE=1 dbus-run-session --config-file="$conf" -- bash -euo pipefail -c '
  run_expect "^TestRealSecretServiceDefaultWithDBusButNoService\$" TestRealSecretServiceDefaultWithDBusButNoService
'

echo "== unlocked gnome-keyring"
dbus-run-session -- bash -euo pipefail -c '
  printf test | gnome-keyring-daemon --unlock --components=secrets >/dev/null
  run_expect "^TestRealSecret(RoundTrip|LockedKeyringIsNotAbsence)\$" TestRealSecretRoundTrip TestRealSecretLockedKeyringIsNotAbsence
'
echo "secret service: all cases passed"
