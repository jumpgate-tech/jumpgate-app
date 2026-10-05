#!/usr/bin/env bash
# scripts/test-trust-linux.sh — run the Linux CA trust-store install command
# that internal/setup builds on real distros (Docker via colima on macOS):
# Debian takes the update-ca-certificates branch, Fedora the update-ca-trust one.
#
#   scripts/test-trust-linux.sh
set -euo pipefail
image="${GO_IMAGE:-golang:1.25}"
# colima mounts the host's home read-only, so the files the containers hand to
# each other live in a throwaway volume, not in a host directory.
vol="jumpgate-trust-$$"
docker volume create "$vol" >/dev/null
trap 'docker volume rm -f "$vol" >/dev/null' EXIT
docker volume create jumpgate-gocache >/dev/null

# Build the command string inside a Go container, so no test binary is built
# or run on the host.
docker run --rm -v "$PWD":/src -w /src -v "$vol":/out -v jumpgate-gocache:/cache \
  -e HOME=/tmp/home -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod -e GOFLAGS=-buildvcs=false \
  -e JUMPGATE_TRUST_CMD_OUT=/out/cmd.sh \
  "$image" go test -tags trustdistro ./internal/setup/ -run TestPrintLinuxTrustCommand -count=1

# A throwaway self-signed root: the stores accept any well-formed certificate.
docker run --rm -v "$vol":/out "$image" sh -c \
  '[ -s /out/cmd.sh ] && openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=jumpgate-test -keyout /dev/null -out /out/ca.crt -days 1 2>/dev/null'

docker run --rm -v "$vol":/out:ro debian:12 sh -c \
  'cp /out/ca.crt /tmp/ca.crt && apt-get update -qq && apt-get install -y -qq ca-certificates >/dev/null && sh /out/cmd.sh && echo debian:12 trusted'
docker run --rm -v "$vol":/out:ro fedora:41 sh -c \
  'cp /out/ca.crt /tmp/ca.crt && sh /out/cmd.sh && echo fedora:41 trusted'
