#!/usr/bin/env bash
# scripts/e2e-agent.sh — pair a real systemd+sshd container and run the e2e
# tests against it. Needs Docker. BASE=ubuntu:24.04 scripts/e2e-agent.sh for
# the second distro.
set -euo pipefail
base="${BASE:-debian:12}"
work="$(mktemp -d)"
trap 'docker rm -f jumpgate-e2e >/dev/null 2>&1 || true; rm -rf "$work"' EXIT

scripts/build-agents.sh "$work/agents"
ssh-keygen -q -t ed25519 -N '' -f "$work/root"
ssh-keygen -q -t ed25519 -N '' -C jumpgate-controller -f "$work/transport"

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
docker run -d --name jumpgate-e2e --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw -p 127.0.0.1::22 jumpgate-e2e >/dev/null
for _ in $(seq 1 30); do docker exec jumpgate-e2e systemctl is-active ssh >/dev/null 2>&1 && break; sleep 1; done
docker exec -i jumpgate-e2e sh -c 'cat >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys' < "$work/root.pub"
port="$(docker port jumpgate-e2e 22/tcp | head -1 | sed 's/.*://')"

JUMPGATE_E2E_PORT="$port" \
JUMPGATE_E2E_ROOT_KEY="$work/root" \
JUMPGATE_E2E_TRANSPORT_KEY="$work/transport" \
JUMPGATE_E2E_TRANSPORT_PUB="$(cat "$work/transport.pub")" \
JUMPGATE_E2E_AGENTS="$work/agents" \
  go test -tags e2e -count=1 -v ./internal/bootstrap/ -run E2E
