#!/usr/bin/env bash
# scripts/build-pkg-repo.sh — build the signed apt and dnf repository served
# at pkg.jumpgate.app (D14) from a directory of jumpgate .deb and .rpm files.
#
#   GNUPGHOME=… PKG_SIGNING_PASSPHRASE_FILE=… \
#     scripts/build-pkg-repo.sh --in DIR --out DIR --key-id ID
#
# The signing key is the one in $GNUPGHOME; its passphrase is read from the
# file PKG_SIGNING_PASSPHRASE_FILE names, never from the command line.
# Output under --out (which must not exist yet):
#   apt/            aptly: dists/stable/{InRelease,Release,Release.gpg}, pool/
#   rpm/            the signed rpms, repodata/ and repodata/repomd.xml.asc
#   jumpgate.asc    the public key
#   jumpgate.repo   the dnf repo file
# Every package in --in is published, so pass every version the repo keeps.
# Needs aptly, createrepo_c, rpmsign and gpg.
set -euo pipefail
usage() { echo "usage: scripts/build-pkg-repo.sh --in DIR --out DIR --key-id ID" >&2; exit 2; }
in='' out='' key=''
while [ $# -gt 0 ]; do
  case "$1" in
    --in) [ $# -ge 2 ] || usage; in="$2"; shift ;;
    --out) [ $# -ge 2 ] || usage; out="$2"; shift ;;
    --key-id) [ $# -ge 2 ] || usage; key="$2"; shift ;;
    *) usage ;;
  esac
  shift
done
if [ -z "$in" ] || [ -z "$out" ] || [ -z "$key" ]; then usage; fi
pass="${PKG_SIGNING_PASSPHRASE_FILE:-}"
if [ -z "$pass" ] || [ ! -r "$pass" ]; then echo "build-pkg-repo: set PKG_SIGNING_PASSPHRASE_FILE to a readable passphrase file" >&2; exit 1; fi
[ -n "${GNUPGHOME:-}" ] || { echo "build-pkg-repo: set GNUPGHOME to the keyring holding $key" >&2; exit 1; }
[ ! -e "$out" ] || { echo "build-pkg-repo: $out exists; give a new directory" >&2; exit 1; }
for tool in aptly createrepo_c rpmsign gpg; do
  command -v "$tool" >/dev/null || { echo "build-pkg-repo: $tool is not installed" >&2; exit 1; }
done
here="$(cd "$(dirname "$0")/.." && pwd)"
shopt -s nullglob
debs=("$in"/*.deb)
rpms=("$in"/*.rpm)
[ ${#debs[@]} -gt 0 ] || { echo "build-pkg-repo: no .deb in $in" >&2; exit 1; }
[ ${#rpms[@]} -gt 0 ] || { echo "build-pkg-repo: no .rpm in $in" >&2; exit 1; }

gpg_sign=(gpg --batch --yes --pinentry-mode loopback --passphrase-file "$pass" --local-user "$key")
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$out"

# apt, through aptly in a throwaway root.
cat >"$tmp/aptly.conf" <<EOF
{"rootDir": "$tmp/aptly", "gpgProvider": "gpg2", "architectures": ["amd64", "arm64"]}
EOF
aptly() { command aptly -config="$tmp/aptly.conf" "$@"; }
aptly repo create -distribution=stable -component=main jumpgate >/dev/null
aptly repo add jumpgate "${debs[@]}" >/dev/null
aptly publish repo -batch -distribution=stable -component=main -architectures=amd64,arm64 \
  -gpg-key="$key" -passphrase-file="$pass" jumpgate >/dev/null
cp -a "$tmp/aptly/public" "$out/apt"

# rpm: sign each package, index them, sign the index.
mkdir -p "$out/rpm"
cp "${rpms[@]}" "$out/rpm/"
for r in "$out"/rpm/*.rpm; do
  rpmsign --addsign --define "__gpg $(command -v gpg)" --define "_gpg_name $key" \
    --define "_gpg_sign_cmd_extra_args --batch --pinentry-mode loopback --passphrase-file $pass" \
    "$r" >/dev/null
done
createrepo_c --quiet "$out/rpm"
"${gpg_sign[@]}" --detach-sign --armor -o "$out/rpm/repodata/repomd.xml.asc" "$out/rpm/repodata/repomd.xml"

gpg --batch --armor --export "$key" >"$out/jumpgate.asc"
cp "$here/packaging/repo/jumpgate.repo" "$out/jumpgate.repo"
echo "built $out (apt, rpm) signed by $key"
