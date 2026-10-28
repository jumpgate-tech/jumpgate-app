#!/usr/bin/env bash
# scripts/e2e-agent.sh — pair a real systemd+sshd container and run the e2e
# tests against it. Needs Docker. BASE=ubuntu:24.04 scripts/e2e-agent.sh for
# the second distro.
set -euo pipefail
# -run PATTERN runs only the e2e tests matching it (default: every E2E test).
run=E2E
if [ "${1:-}" = -run ] && [ -n "${2:-}" ]; then run="$2"; fi
base="${BASE:-debian:12}"
work="$(mktemp -d)"
name="jumpgate-e2e-$$"
# KEEP=1 leaves the container running for inspection (docker exec -it NAME bash).
cleanup() {
  if [ -n "${KEEP:-}" ]; then
    echo "kept container $name (docker rm -f $name when done)" >&2
  else
    docker rm -f "$name" >/dev/null 2>&1 || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT

scripts/build-agents.sh "$work/agents"
ssh-keygen -q -t ed25519 -N '' -f "$work/root"
ssh-keygen -q -t ed25519 -N '' -C jumpgate-controller -f "$work/transport"

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
# shellcheck disable=SC2086 # E2E_DOCKER_FLAGS is a list of flags
docker run -d --name "$name" ${E2E_DOCKER_FLAGS:---privileged} -p 127.0.0.1::22 jumpgate-e2e >/dev/null
port="$(docker port "$name" 22/tcp | head -1 | sed 's/.*://')"

# Ready means systemd finished booting (running, or degraded by some unrelated
# unit) and sshd answers on the published port. On Ubuntu 24.04 sshd is socket
# activated, so a connection is the right probe, not `is-active ssh`.
ready=
for _ in $(seq 1 60); do
  state="$(docker exec "$name" systemctl is-system-running 2>/dev/null || true)"
  if { [ "$state" = running ] || [ "$state" = degraded ]; } &&
     [ -n "$(ssh-keyscan -T 2 -p "$port" 127.0.0.1 2>/dev/null)" ]; then
    ready=1
    break
  fi
  sleep 1
done
if [ -z "$ready" ]; then
  echo "container $name never became ready (systemd state: ${state:-unknown})" >&2
  docker logs "$name" >&2 || true
  exit 1
fi
docker exec -i "$name" sh -c 'cat >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys' < "$work/root.pub"

# -p 1: both packages share the one container, and the server's module test
# creates the node account the bootstrap migrate test must not see yet. The
# server's vault test (a plan waits while locked) resets the box to a legacy
# node, so the server package runs after the bootstrap one.
JUMPGATE_E2E_PORT="$port" \
JUMPGATE_E2E_ROOT_KEY="$work/root" \
JUMPGATE_E2E_TRANSPORT_KEY="$work/transport" \
JUMPGATE_E2E_TRANSPORT_PUB="$(cat "$work/transport.pub")" \
JUMPGATE_E2E_AGENTS="$work/agents" \
  go test -tags e2e -count=1 -p 1 -v ./internal/bootstrap/ ./internal/server/ -run "$run"

# The --local case below runs only with the full set.
[ "$run" = E2E ] || exit 0

# The --local case runs inside the box: a linux build of the same tests, run as
# alice (a non-root sudoer) against the copied agents; bob is the stranger.
case "$(docker exec "$name" uname -m)" in
  x86_64) arch=amd64 ;;
  aarch64) arch=arm64 ;;
  *) echo "unsupported container arch" >&2; exit 1 ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go test -c -tags e2e -o "$work/bootstrap.test" ./internal/bootstrap/
docker exec "$name" mkdir -p /opt/jumpgate-e2e
docker cp -q "$work/bootstrap.test" "$name:/opt/jumpgate-e2e/bootstrap.test"
docker cp -q "$work/agents" "$name:/opt/jumpgate-e2e/agents"
docker exec "$name" chmod -R a+rX /opt/jumpgate-e2e
docker exec -u alice -w /home/alice \
  -e JUMPGATE_E2E_AGENTS=/opt/jumpgate-e2e/agents \
  -e JUMPGATE_E2E_STRANGER=bob \
  "$name" /opt/jumpgate-e2e/bootstrap.test -test.run '^TestE2ELocalPairAndPeerGate$' -test.v -test.count=1

# --self-only on the box: alice enrolls a local-scoped controller, then the
# same signed intent sent as user jumpgate (the uid every tunnelled
# connection arrives as, admitted by group) gets scope_local.
docker exec -u alice -w /home/alice \
  -e JUMPGATE_E2E_AGENTS=/opt/jumpgate-e2e/agents \
  -e JUMPGATE_E2E_SCOPE_PHASE=pair \
  "$name" /opt/jumpgate-e2e/bootstrap.test -test.run '^TestE2EScopeLocalRefusesAGroupPeer$' -test.v -test.count=1
docker exec -u jumpgate -w / \
  -e JUMPGATE_E2E_SCOPE_PHASE=send \
  "$name" /opt/jumpgate-e2e/bootstrap.test -test.run '^TestE2EScopeLocalRefusesAGroupPeer$' -test.v -test.count=1
