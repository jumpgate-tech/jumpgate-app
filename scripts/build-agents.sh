#!/usr/bin/env bash
# scripts/build-agents.sh — cross-build the Linux agent binaries that pairing
# uploads, plus the SHA256SUMS the server checks them against.
set -euo pipefail
out="${1:-$HOME/.jumpgate/agents}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
mkdir -p "$out"
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/valve-tech/jumpgate/internal/buildinfo.version=$version" \
    -o "$out/jumpgate-linux-$arch" ./cmd/jumpgate
done
( cd "$out" && shasum -a 256 jumpgate-linux-amd64 jumpgate-linux-arm64 > SHA256SUMS )
echo "agents in $out"
