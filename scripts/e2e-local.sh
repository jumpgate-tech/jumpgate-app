#!/usr/bin/env bash
# scripts/e2e-local.sh — pair the machine the controller runs on, through the
# CLI (B-4, D5): (a) as root on a box without sudo, where the server runs the
# steps itself; (b) as alice, a NOPASSWD sudoer from scripts/e2e/Dockerfile, on
# a terminal, where `hosts add --local` runs them in the foreground. Each must
# pair and then get a signed answer from the agent: not_set_up in (a); in (b)
# the target carries a node configuration, which must reach node.json, and
# status answers with a snapshot. e2e-agent.sh's TestE2ELocalPairAndPeerGate
# is the library-level proof of (b); this is the CLI-level one. Needs Docker.
#   BASE=ubuntu:24.04 scripts/e2e-local.sh    # the second distro
#   KEEP=1 scripts/e2e-local.sh               # keep the containers to inspect
set -euo pipefail
base="${BASE:-debian:12}"
work="$(mktemp -d)"
cleanup() {
  for n in ${names[@]+"${names[@]}"}; do
    if [ -n "${KEEP:-}" ]; then
      echo "kept container $n (docker rm -f $n when done)" >&2
    else
      docker rm -f "$n" >/dev/null 2>&1 || true
    fi
  done
  rm -rf "$work"
}
trap cleanup EXIT

docker build -q -t jumpgate-e2e --build-arg BASE="$base" scripts/e2e >/dev/null
goarch=
names=()

# fresh_box starts a new container as $name, so each case pairs a box no
# earlier case touched, and copies the controller in.
fresh_box() {
  name="jumpgate-e2e-local-$$-$1"
  names+=("$name")
  # shellcheck disable=SC2086 # E2E_DOCKER_FLAGS is a list of flags
  docker run -d --name "$name" ${E2E_DOCKER_FLAGS:---privileged} jumpgate-e2e >/dev/null
  if [ -z "$goarch" ]; then
    case "$(docker exec "$name" uname -m)" in
      x86_64) goarch=amd64 ;;
      aarch64) goarch=arm64 ;;
      *) echo "unsupported container arch" >&2; exit 1 ;;
    esac
    # A static Linux controller of the box's arch uploads itself as the
    # agent (agentbin's "this binary" source), so no agents directory is
    # needed.
    CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build -o "$work/jumpgate" ./cmd/jumpgate
  fi
  local ready= state=
  for _ in $(seq 1 60); do
    state="$(docker exec "$name" systemctl is-system-running 2>/dev/null || true)"
    if [ "$state" = running ] || [ "$state" = degraded ]; then
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
  docker cp -q "$work/jumpgate" "$name:/usr/local/bin/jumpgate"
}

as() { # user home command...
  local u="$1" h="$2"
  shift 2
  docker exec -u "$u" -e HOME="$h" "$name" "$@"
}

expect_paired() { # user home
  if ! as "$1" "$2" jumpgate hosts list | grep -q '^me .*agent 0x'; then
    as "$1" "$2" jumpgate hosts list >&2 || true
    echo "FAIL: $1's controller recorded no pairing" >&2
    exit 1
  fi
}

# status is a signed intent; "no node is set up" is the agent's signed
# not_set_up answer, so the whole path (socket, peer gate, signature) worked.
expect_signed_answer() { # user home
  local out
  out="$(as "$1" "$2" jumpgate status me 2>&1 || true)"
  echo "$out"
  echo "$out" | grep -q 'no node is set up' || { echo "FAIL: no signed answer for $1" >&2; exit 1; }
}

echo "== (a) root, no sudo on the box"
fresh_box root
docker exec "$name" mv /usr/bin/sudo /usr/bin/sudo.off
as root /root jumpgate keys init --store file
# No -t: as root the CLI needs no terminal, the server runs the steps.
as root /root jumpgate hosts add me --local
expect_paired root /root
expect_signed_answer root /root
# P7: root is admitted by the peer gate already, so no local uid is enrolled
# and the socket keeps its group-only mode.
mode="$(docker exec "$name" stat -c %a /run/jumpgate/agent.sock)"
[ "$mode" = 660 ] || { echo "FAIL: agent socket mode $mode after a root pairing, want 660" >&2; exit 1; }
as root /root jumpgate stop

echo "== (b) alice, a NOPASSWD sudoer, on a terminal"
fresh_box alice
# The target already carries a node configuration, as after the setup
# wizard; the foreground pairing must write it to node.json (P6).
docker exec -i -u alice "$name" sh -c 'umask 077 && mkdir -p /home/alice/.jumpgate && cat > /home/alice/.jumpgate/config.json' <<'JSON'
{"targets": [{"id": "me", "mode": "local",
  "wire": {"ChainID": 369, "ExecID": "reth", "BeaconID": "lighthouse", "DataDir": "/var/lib/jumpgate-e2e/369"}}]}
JSON
as alice /home/alice jumpgate keys init --store file
# -t gives the CLI a terminal on stdin, which non-root local pairing requires.
docker exec -t -u alice -e HOME=/home/alice "$name" jumpgate hosts add me --local
expect_paired alice /home/alice
node="$(docker exec "$name" cat /etc/jumpgate/node.json)" ||
  { echo "FAIL: no /etc/jumpgate/node.json after alice's pairing" >&2; exit 1; }
echo "$node"
for want in '"ChainID":369' '"ExecID":"reth"' '"BeaconID":"lighthouse"' '"DataDir":"/var/lib/jumpgate-e2e/369"'; do
  case "$node" in *"$want"*) ;; *) echo "FAIL: node.json lacks $want" >&2; exit 1 ;; esac
done
# With a node set up, status is a real signed snapshot: exit 0 means the CLI
# verified the agent's receipt, and the agent parsed and validated node.json.
if ! out="$(as alice /home/alice jumpgate status me 2>&1)"; then
  echo "$out" >&2
  echo "FAIL: no signed status for alice" >&2
  exit 1
fi
as alice /home/alice jumpgate stop

echo "e2e-local: OK"
