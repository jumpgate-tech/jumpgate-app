#!/usr/bin/env bash
# scripts/e2e-agent.sh — pair a real systemd+sshd container and run the e2e
# tests against it. Needs Docker. BASE=ubuntu:24.04 scripts/e2e-agent.sh for
# the second distro.
set -euo pipefail
base="${BASE:-debian:12}"
work="$(mktemp -d)"
name="jumpgate-e2e-$$"
trap 'docker rm -f "$name" >/dev/null 2>&1 || true; rm -rf "$work"' EXIT

scripts/build-agents.sh "$work/agents"
ssh-keygen -q -t ed25519 -N '' -f "$work/root"
ssh-keygen -q -t ed25519 -N '' -C jumpgate-controller -f "$work/transport"

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
docker run -d --name "$name" --privileged -p 127.0.0.1::22 jumpgate-e2e >/dev/null
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

JUMPGATE_E2E_PORT="$port" \
JUMPGATE_E2E_ROOT_KEY="$work/root" \
JUMPGATE_E2E_TRANSPORT_KEY="$work/transport" \
JUMPGATE_E2E_TRANSPORT_PUB="$(cat "$work/transport.pub")" \
JUMPGATE_E2E_AGENTS="$work/agents" \
  go test -tags e2e -count=1 -v ./internal/bootstrap/ -run E2E
