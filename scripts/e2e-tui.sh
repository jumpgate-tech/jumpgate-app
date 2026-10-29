#!/usr/bin/env bash
# scripts/e2e-tui.sh — drive the tui/ dashboard against a real `jumpgate serve`
# and two systemd+sshd boxes from scripts/e2e/Dockerfile (tui/test/live.test.ts
# through the OpenTUI test renderer). The controller runs on this machine with
# an isolated HOME; its key lives in a fresh vault. The test pairs both boxes
# from the Hosts screen, restarts a service, answers a job's decisions, turns a
# module on, invites, upgrades a gate, and locks and unlocks the vault. Each
# screen is saved as text to $FRAMES (default: $work/frames.txt, printed at the
# end). Needs Docker and Bun (~/.bun/bin).
#   KEEP=1 scripts/e2e-tui.sh          # keep the containers and work dir
#   FRAMES=/tmp/f.txt scripts/e2e-tui.sh
#
# Box 1 carries a node set up before the rename (scripts/e2e/tuiseed) on the
# image's stub clients. The test calls this script back as its hook
# (JUMPGATE_LIVE_HOOK) for what only root on a box can do: `hook pre-upgrade 2`.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
prefix=jumpgate-tuilive

box_name() { echo "$prefix-$1"; }

if [ "${1:-}" = hook ]; then
  # Hook mode: $2 the phase, $3 the box (1 or 2); E2E_TUI_WORK is the work dir.
  work="$E2E_TUI_WORK"
  # The test runs with the controller's HOME; docker and go want the real one.
  export HOME="$E2E_TUI_HOME"
  n="$(box_name "$3")"
  case "$2" in
    pre-upgrade)
      # The gate refuses an unapproved upgrade unless policy.json opts in to
      # routine upgrades; and the controller now carries a newer gate.
      docker exec "$n" sh -c 'p=/etc/jumpgate/policy.json; grep -q "\"kinds\"" "$p" || sed -i "1s/^{\$/{\n  \"kinds\": {\"agent.upgrade\": [\"routine\"]},/" "$p"; grep -q agent.upgrade "$p"'
      (cd "$here" && VERSION=v0.0.91 scripts/build-agents.sh "$work/home/.jumpgate/agents" >/dev/null)
      ;;
    *) echo "unknown hook $2" >&2; exit 2 ;;
  esac
  exit 0
fi

cd "$here"
export PATH="$HOME/.bun/bin:$PATH"
work="$(mktemp -d)"
work="$(cd "$work" && pwd -P)"
frames="${FRAMES:-$work/frames.txt}"
names=()
srv=
cleanup() {
  [ -z "$srv" ] || kill "$srv" 2>/dev/null || true
  for n in ${names[@]+"${names[@]}"}; do
    if [ -n "${KEEP:-}" ]; then echo "kept container $n" >&2; else docker rm -f "$n" >/dev/null 2>&1 || true; fi
  done
  if [ -n "${KEEP:-}" ]; then echo "kept $work" >&2; else rm -rf "$work"; fi
}
trap cleanup EXIT
fail() { echo "FAIL $*" >&2; exit 1; }

home="$work/home"
mkdir -p "$home/.ssh" && chmod 700 "$home/.ssh"
# The gates pairing uploads (v0.0.90); the pre-upgrade hook replaces them with
# v0.0.91. The controller is v0.0.91: an invite needs a released version.
VERSION=v0.0.90 scripts/build-agents.sh "$home/.jumpgate/agents" >/dev/null
go build -ldflags "-X github.com/valve-tech/jumpgate/internal/buildinfo.version=v0.0.91" -o "$work/jumpgate" ./cmd/jumpgate
# The boxes' root key, a file: the server is run with no ssh-agent.
ssh-keygen -q -t ed25519 -N '' -f "$home/.ssh/id_ed25519"
docker build -q -t jumpgate-e2e scripts/e2e >/dev/null

ports=()
for i in 1 2; do
  n="$(box_name "$i")"
  docker rm -f "$n" >/dev/null 2>&1 || true
  names+=("$n")
  # shellcheck disable=SC2086 # E2E_DOCKER_FLAGS is a list of flags
  docker run -d --name "$n" ${E2E_DOCKER_FLAGS:---privileged} -p 127.0.0.1::22 jumpgate-e2e >/dev/null
done
for i in 1 2; do
  n="$(box_name "$i")"
  port="$(docker port "$n" 22/tcp | head -1 | sed 's/.*://')"
  ready=
  for _ in $(seq 1 60); do
    state="$(docker exec "$n" systemctl is-system-running 2>/dev/null || true)"
    if { [ "$state" = running ] || [ "$state" = degraded ]; } &&
       [ -n "$(ssh-keyscan -T 2 -p "$port" 127.0.0.1 2>/dev/null)" ]; then ready=1; break; fi
    sleep 1
  done
  [ -n "$ready" ] || fail "$n never became ready (${state:-unknown})"
  docker exec -i "$n" sh -c 'cat >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys' < "$home/.ssh/id_ed25519.pub"
  ports+=("$port")
done
# Box 1: a node set up before the rename, running, with one process of its
# account that ignores SIGTERM, so the rename's stop-units step asks.
go run ./scripts/e2e/tuiseed | docker exec -i "$(box_name 1)" sh -e

# The vault, its passphrase in a file only this run reads, and the controller
# key in the vault.
pass="$work/pass"
( umask 077 && head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n' > "$pass" )
jg() { env -u SSH_AUTH_SOCK HOME="$home" JUMPGATE_HOME= "$work/jumpgate" "$@"; }
jg vault init --stdin --keystore none < "$pass" >/dev/null
jg keys init --store vault --stdin < "$pass" >/dev/null
# A recovery code, so the setup guide can reach 6/6; it prints once, to /dev/null.
jg vault add-recovery --stdin < "$pass" >/dev/null
# Not through jg: $! must be the server itself, so cleanup stops it. Port 0
# keeps clear of a real server on 8799; server.json says where it listens.
env -u SSH_AUTH_SOCK HOME="$home" JUMPGATE_HOME= "$work/jumpgate" serve --no-open --bind 127.0.0.1:0 > "$work/server.log" 2>&1 &
srv=$!
for _ in $(seq 1 60); do [ -s "$home/.jumpgate/run/server.json" ] && break; sleep 0.5; done
[ -s "$home/.jumpgate/run/server.json" ] || { cat "$work/server.log" >&2; fail "the server never wrote server.json"; }

: > "$frames"
env -u SSH_AUTH_SOCK HOME="$home" \
  JUMPGATE_LIVE_HOME="$home/.jumpgate" JUMPGATE_LIVE_PASSFILE="$pass" JUMPGATE_LIVE_FRAMES="$frames" \
  JUMPGATE_LIVE_KEY="$home/.ssh/id_ed25519" JUMPGATE_LIVE_BOXES="127.0.0.1:${ports[0]},127.0.0.1:${ports[1]}" \
  JUMPGATE_LIVE_HOOK="$here/scripts/e2e-tui.sh" E2E_TUI_WORK="$work" E2E_TUI_HOME="$HOME" \
  JUMPGATE_BIN="$work/jumpgate" \
  bun test --cwd tui test/live.test.ts --timeout 900000 || { tail -50 "$work/server.log" >&2; cat "$frames" >&2; fail "live test"; }
cat "$frames"
echo "ok e2e-tui"
