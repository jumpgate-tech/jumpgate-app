#!/usr/bin/env bash
# scripts/build-agents.sh — build the static Linux agent binaries that pairing
# uploads, plus the SHA256SUMS they are checked against. One script for every
# use, so developer, embedded and published agents are byte-identical for a
# version (spec D27):
#
#   scripts/build-agents.sh                                    # ~/.jumpgate/agents (dev override)
#   GZIP=1 scripts/build-agents.sh internal/agentbin/embedded  # for -tags embedagents
#   VERSION=v0.9.0 ...                                         # release: the git tag
set -euo pipefail
out="${1:-$HOME/.jumpgate/agents}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
# gzip(1) reads options from $GZIP, so take ours out of the environment.
compress="${GZIP:-0}"
unset GZIP
mkdir -p "$out"
sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$@"; else shasum -a 256 "$@"; fi
}
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$version" \
    -o "$out/jumpgate-linux-$arch" ./cmd/jumpgate
done
( cd "$out" && sha256 jumpgate-linux-amd64 jumpgate-linux-arm64 > SHA256SUMS )
if [ "$compress" = 1 ]; then
  # -n leaves the name and time out of the header, so the .gz is reproducible.
  for arch in amd64 arm64; do
    gzip -9 -n -c "$out/jumpgate-linux-$arch" > "$out/jumpgate-linux-$arch.gz"
  done
fi
echo "agents ($version) in $out"
