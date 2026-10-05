#!/usr/bin/env bash
# scripts/test-linux.sh — run Go tests in a Linux container (Docker via colima
# on macOS) as an unprivileged user, the way CI's ubuntu runner does. Root
# would pass permission tests that a normal user fails.
#
#   scripts/test-linux.sh                         # go test ./...
#   scripts/test-linux.sh ./internal/fsperm/ -v   # any go test arguments
#   AS_ROOT=1 scripts/test-linux.sh ./...         # as root (pairing as root)
#   CGO_ENABLED=0 scripts/test-linux.sh ./internal/buildinfo/   # passed through
set -euo pipefail
image="${GO_IMAGE:-golang:1.25}"
[ $# -eq 0 ] && set -- ./...
docker volume create jumpgate-gocache >/dev/null
drop='exec setpriv --reuid=1000 --regid=1000 --clear-groups "$@"'
[ "${AS_ROOT:-0}" = 1 ] && drop='exec "$@"'
envs=()
[ -n "${CGO_ENABLED:-}" ] && envs+=(-e "CGO_ENABLED=$CGO_ENABLED")
docker run --rm \
  -v "$PWD":/src -w /src \
  -v jumpgate-gocache:/cache \
  -e HOME=/tmp/home -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod -e GOFLAGS=-buildvcs=false \
  ${envs[@]+"${envs[@]}"} \
  "$image" sh -c 'mkdir -p /tmp/home /cache && chown -R 1000:1000 /tmp/home /cache && '"$drop" sh go test "$@"
