#!/bin/bash
# scripts/windows-vm/on-demand.sh: one tick of the start-on-demand launcher for
# the local Windows CI VM (run by launchd every few minutes, see
# install-on-demand.sh).
#
#   on-demand.sh [--dry-run] [--adopt]
#
# Rules:
#  - a `windows`-tagged job pending or running in the project and the VM is
#    stopped: `vm.sh start`, and drop the marker (on-demand.managed);
#  - the VM is running, carries the marker, and no windows job has been
#    pending/running for JUMPGATE_OD_IDLE_MIN minutes (default 20): `vm.sh stop`;
#  - a running VM WITHOUT the marker was started by hand: never touched;
#  - any API/token/vm.sh error: log and exit 0, nothing destructive, retry next tick.
# --dry-run logs the decision and changes nothing (no vm.sh call, no state
# writes). --adopt writes the marker so a hand-started VM is stopped when idle.
#
# Env: JUMPGATE_WIN_VM_DIR (state dir, default ~/vms/jumpgate-win),
# JUMPGATE_OD_IDLE_MIN, JUMPGATE_OD_PROJECT (71), JUMPGATE_OD_HOST,
# JUMPGATE_OD_TAG (windows), JUMPGATE_OD_LOG.
# Test seams: JUMPGATE_OD_JOBS_FILE (jobs JSON instead of the API),
# JUMPGATE_OD_VM_STATE (running|stopped instead of `vm.sh status`),
# JUMPGATE_OD_NOW (epoch seconds).
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VM_DIR="${JUMPGATE_WIN_VM_DIR:-$HOME/vms/jumpgate-win}"
IDLE_MIN="${JUMPGATE_OD_IDLE_MIN:-20}"
PROJECT="${JUMPGATE_OD_PROJECT:-71}"
HOST="${JUMPGATE_OD_HOST:-git.craftsmanfounder.com}"
TAG="${JUMPGATE_OD_TAG:-windows}"
LOG="${JUMPGATE_OD_LOG:-$HOME/Library/Logs/jumpgate-win-vm-on-demand.log}"
MARKER="$VM_DIR/on-demand.managed"
IDLE_FILE="$VM_DIR/on-demand.idle-since"
LOCK="$VM_DIR/on-demand.lock"
export PATH="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

dry=0; adopt=0
for a in "$@"; do
  case "$a" in
    --dry-run) dry=1 ;;
    --adopt) adopt=1 ;;
    *) echo "usage: $0 [--dry-run] [--adopt]" >&2; exit 2 ;;
  esac
done

mkdir -p "$(dirname "$LOG")" "$VM_DIR"
log() { printf '%s [on-demand]%s %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$([ "$dry" = 1 ] && echo ' [dry-run]')" "$*" >> "$LOG"; [ -t 2 ] && printf '%s\n' "$*" >&2 || true; }
now() { echo "${JUMPGATE_OD_NOW:-$(date +%s)}"; }

if [ "$adopt" = 1 ]; then
  [ "$dry" = 1 ] && { log "would adopt the running VM (write marker)"; exit 0; }
  : > "$MARKER"; rm -f "$IDLE_FILE"; log "adopted: VM will be stopped after ${IDLE_MIN} min without windows jobs"; exit 0
fi

# Single instance: mkdir is atomic; a lock whose pid is gone is stale.
if [ "$dry" = 0 ]; then
  if ! mkdir "$LOCK" 2>/dev/null; then
    opid="$(cat "$LOCK/pid" 2>/dev/null || true)"
    if [ -n "$opid" ] && kill -0 "$opid" 2>/dev/null; then log "another tick (pid $opid) is running; skipping"; exit 0; fi
    rm -rf "$LOCK"; mkdir "$LOCK" 2>/dev/null || { log "cannot take lock; skipping"; exit 0; }
  fi
  echo $$ > "$LOCK/pid"
  trap 'rm -rf "$LOCK"' EXIT
fi

# --- jobs ---
if [ -n "${JUMPGATE_OD_JOBS_FILE:-}" ]; then
  jobs_json="$(cat "$JUMPGATE_OD_JOBS_FILE")" || { log "cannot read jobs file; skipping"; exit 0; }
else
  token="$(printf 'protocol=https\nhost=%s\n\n' "$HOST" | git credential fill 2>/dev/null | sed -n 's/^password=//p')" || token=""
  [ -n "$token" ] || { log "no API token from git credential; skipping"; exit 0; }
  # token goes to curl via stdin config, never argv
  jobs_json="$(printf 'header = "PRIVATE-TOKEN: %s"\n' "$token" | curl -fsS --max-time 30 -K - \
    "https://$HOST/api/v4/projects/$PROJECT/jobs?scope%5B%5D=pending&scope%5B%5D=running&per_page=100" 2>&1)" \
    || { log "jobs API failed: $(printf '%s' "$jobs_json" | tr -d '\n' | cut -c1-200); skipping"; exit 0; }
  token=""
fi
counts="$(printf '%s' "$jobs_json" | jq -r --arg t "$TAG" '
  if type != "array" then error("not an array") else . end
  | [.[] | select((.tag_list // []) | index($t))]
  | "\([.[] | select(.status=="pending")] | length) \([.[] | select(.status=="running")] | length)"' 2>/dev/null)" \
  || { log "unparseable jobs response; skipping"; exit 0; }
pending="${counts% *}"; running_jobs="${counts#* }"

# --- vm ---
if [ -n "${JUMPGATE_OD_VM_STATE:-}" ]; then vmst="$JUMPGATE_OD_VM_STATE"
else
  s="$("$here/vm.sh" status 2>/dev/null)" || { log "vm.sh status failed; skipping"; exit 0; }
  case "$s" in running*) vmst=running ;; *) vmst=stopped ;; esac
fi

act() { # run a vm.sh action unless dry-run
  if [ "$dry" = 1 ]; then log "would run: vm.sh $1"; return 0; fi
  if out="$("$here/vm.sh" "$1" 2>&1)"; then log "vm.sh $1 ok"; return 0; fi
  log "vm.sh $1 FAILED: $(printf '%s' "$out" | tail -3 | tr '\n' ' ')"; return 1
}
setstate() { [ "$dry" = 1 ] || "$@"; }

log "windows jobs: pending=$pending running=$running_jobs; vm=$vmst; managed=$([ -f "$MARKER" ] && echo yes || echo no)"

if [ "$pending" -gt 0 ] && [ "$vmst" = stopped ]; then
  log "decision: start (job waiting, VM stopped)"
  if act start; then setstate touch "$MARKER"; setstate rm -f "$IDLE_FILE"; fi
  exit 0
fi
if [ "$pending" -gt 0 ] || [ "$running_jobs" -gt 0 ]; then
  log "decision: busy, keep VM; idle timer reset"
  setstate rm -f "$IDLE_FILE"; exit 0
fi
if [ "$vmst" = stopped ]; then
  log "decision: nothing to do (VM stopped)"
  setstate rm -f "$MARKER" "$IDLE_FILE"; exit 0
fi
if [ ! -f "$MARKER" ]; then
  log "decision: leave alone (VM started by hand, no marker)"; exit 0
fi
n="$(now)"
if [ ! -f "$IDLE_FILE" ]; then
  log "decision: idle started; will stop after ${IDLE_MIN} min"
  setstate sh -c 'echo "$1" > "$2"' _ "$n" "$IDLE_FILE"; exit 0
fi
since="$(cat "$IDLE_FILE" 2>/dev/null || true)"
case "$since" in ''|*[!0-9]*) log "bad idle file; resetting"; setstate sh -c 'echo "$1" > "$2"' _ "$n" "$IDLE_FILE"; exit 0 ;; esac
mins=$(( (n - since) / 60 ))
if [ "$mins" -ge "$IDLE_MIN" ]; then
  log "decision: stop (idle ${mins} min >= ${IDLE_MIN})"
  if act stop; then setstate rm -f "$MARKER" "$IDLE_FILE"; fi
else
  log "decision: idle ${mins} of ${IDLE_MIN} min; keep VM"
fi
exit 0
