#!/usr/bin/env bash
# scripts/render-icons.sh — regenerate cmd/jumpgate/AppIcon.icns from icon.svg.
# The .icns is committed so building the macOS apps needs no SVG renderer; run
# this after editing icon.svg. CI re-runs it and fails if the result differs.
#
#   scripts/render-icons.sh            # in a debian:12 container (needs docker)
#   IN_CONTAINER=1 scripts/render-icons.sh   # already on Debian with
#                                      # librsvg2-bin icnsutils installed
set -euo pipefail
cd "$(dirname "$0")/.."
if [ "${IN_CONTAINER:-}" != 1 ]; then
  # The source is mounted read-only (some docker VMs mount it that way) and
  # the result is copied out of the stopped container.
  name="jumpgate-icons-$$"
  trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
  docker run --name "$name" -v "$PWD":/src:ro -w /src debian:12 sh -c \
    'apt-get update -qq && apt-get install -y -qq librsvg2-bin icnsutils >/dev/null && IN_CONTAINER=1 OUT=/tmp/AppIcon.icns bash scripts/render-icons.sh'
  docker cp "$name:/tmp/AppIcon.icns" cmd/jumpgate/AppIcon.icns
  echo "wrote cmd/jumpgate/AppIcon.icns"
  exit 0
fi
OUT="${OUT:-cmd/jumpgate/AppIcon.icns}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
sizes="16 32 128 256 512"
files=""
for s in $sizes; do
  rsvg-convert -w "$s" -h "$s" cmd/jumpgate/icon.svg -o "$tmp/$s.png"
  files="$files $tmp/$s.png"
done
# shellcheck disable=SC2086
png2icns "$OUT" $files
echo "wrote $OUT"
