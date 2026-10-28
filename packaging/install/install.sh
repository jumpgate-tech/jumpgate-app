#!/bin/sh
# install.sh — install jumpgate on Linux, from a GitHub release, verified.
#
#   curl -fsSL https://jumpgate.app/install.sh | sh
#   curl -fsSL https://jumpgate.app/install.sh | sh -s -- --controller
#
# Options:
#   --controller            install the controller server role (root): the
#                           binary, then `jumpgate setup install-service`
#   --version vX.Y.Z        pin a release (default: the latest)
#   --yes                   do not ask before running commands with sudo
#   --prefix DIR            install the CLI to DIR/bin, without root
#   --require-provenance    fail unless `gh attestation verify` can run
#   --self-only             with --controller: the next step is a self-only
#                           controller (`jumpgate setup server --self-only`)
#   --agent --controller ADDR --transport-key KEY
#                           set this box's gate up for that controller (root);
#                           `jumpgate hosts invite NAME` prints this line
#
# Every download is checked against the release's checksums.txt; with gh
# signed in, its build provenance is checked too. The whole script is one
# function, called on the last line, so a truncated download runs nothing.

main() {
	set -eu

	base="${JUMPGATE_INSTALL_BASE_URL:-https://github.com/jumpgate-tech/jumpgate-app/releases}"
	repo=jumpgate-tech/jumpgate-app
	usage="usage: install.sh [--controller | --agent --controller ADDR --transport-key KEY] [--version vX.Y.Z] [--yes] [--prefix DIR] [--require-provenance] [--self-only]"

	say() { printf '%s\n' "$*"; }
	die() {
		printf 'install.sh: %s\n' "$*" >&2
		exit 1
	}
	has_tty() { (: </dev/tty) 2>/dev/null; }

	role=
	version=latest
	yes=0
	prefix=
	require_provenance=0
	self_only=0
	controller=
	transport_key=
	# --agent changes what --controller means (an address, not the role), so
	# find it before parsing.
	agent=0
	for arg in "$@"; do
		[ "$arg" != --agent ] || agent=1
	done
	[ "$agent" = 0 ] || role=agent
	while [ $# -gt 0 ]; do
		case "$1" in
		--agent) ;;
		--controller)
			if [ "$agent" = 1 ]; then
				[ $# -ge 2 ] || die "--controller needs the controller's address with --agent; $usage"
				controller="$2"
				shift
			else
				role=controller
			fi
			;;
		--controller=*)
			[ "$agent" = 1 ] || die "--controller takes no value without --agent; $usage"
			controller="${1#--controller=}"
			;;
		--transport-key)
			[ $# -ge 2 ] || die "--transport-key needs a value; $usage"
			transport_key="$2"
			shift
			;;
		--transport-key=*) transport_key="${1#--transport-key=}" ;;
		--version)
			[ $# -ge 2 ] || die "--version needs a value; $usage"
			version="$2"
			shift
			;;
		--version=*) version="${1#--version=}" ;;
		--yes | -y) yes=1 ;;
		--prefix)
			[ $# -ge 2 ] || die "--prefix needs a value; $usage"
			prefix="$2"
			shift
			;;
		--prefix=*) prefix="${1#--prefix=}" ;;
		--require-provenance) require_provenance=1 ;;
		--self-only) self_only=1 ;;
		-h | --help)
			say "$usage"
			return 0
			;;
		*) die "unknown option $1; $usage" ;;
		esac
		shift
	done
	if [ -n "$prefix" ]; then
		[ -z "$role" ] || die "--prefix installs only the CLI; the $role role needs root"
		role=cli
	fi
	if [ "$self_only" = 1 ] && [ "$role" != controller ]; then
		die "--self-only goes with --controller"
	fi
	if [ "$role" = agent ]; then
		[ -n "$controller" ] || die "--agent needs --controller ADDR (run \`jumpgate hosts invite NAME\` on the controller for the full command)"
		[ -n "$transport_key" ] || die "--agent needs --transport-key KEY (run \`jumpgate hosts invite NAME\` on the controller for the full command)"
	elif [ -n "$transport_key" ]; then
		die "--transport-key goes with --agent"
	fi
	if [ "$version" != latest ]; then
		case "$version" in
		v*) ;;
		*) version="v$version" ;;
		esac
	fi

	os="$(uname -s)"
	[ "$os" = Linux ] || die "unsupported OS $os; supported: Linux on amd64 or arm64"
	machine="$(uname -m)"
	case "$machine" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) die "unsupported architecture $machine; supported: Linux on amd64 or arm64" ;;
	esac

	if [ "$role" != agent ] && [ -e /usr/bin/jumpgate ]; then
		if { command -v dpkg >/dev/null 2>&1 && dpkg -S /usr/bin/jumpgate >/dev/null 2>&1; } ||
			{ command -v rpm >/dev/null 2>&1 && rpm -qf /usr/bin/jumpgate >/dev/null 2>&1; }; then
			die "/usr/bin/jumpgate belongs to a package; use apt upgrade jumpgate or dnf upgrade jumpgate"
		fi
	fi

	if [ -z "$role" ]; then
		if has_tty; then
			printf '%s\n' "install jumpgate as:" "  1) CLI only (default)  2) controller server  3) gate-only box" >/dev/tty
			printf '%s' "choice [1]: " >/dev/tty
			read -r choice </dev/tty || choice=
			case "$choice" in
			"" | 1) role=cli ;;
			2) role=controller ;;
			3)
				say "a gate-only box pairs with a controller: run \`jumpgate hosts invite NAME\` there, then run the command it prints here, as root"
				return 0
				;;
			*) die "no such choice: $choice" ;;
			esac
		else
			role=cli
			hint_roles=1
		fi
	fi

	if command -v curl >/dev/null 2>&1; then
		fetch() { curl -fsSL --proto '=https' --tlsv1.2 -o "$2" "$1"; }
		latest_tag() {
			eff="$(curl -fsSLI --proto '=https' --tlsv1.2 -o /dev/null -w '%{url_effective}' "$base/latest")" || return 1
			printf '%s\n' "${eff##*/}"
		}
	elif command -v wget >/dev/null 2>&1; then
		fetch() { wget -q --https-only --secure-protocol=TLSv1_2 -O "$2" "$1"; }
		latest_tag() {
			eff="$(wget --https-only --secure-protocol=TLSv1_2 -S --spider "$base/latest" 2>&1 |
				sed -n 's/^ *[Ll]ocation: *//p' | tail -n 1 | tr -d '\r')"
			[ -n "$eff" ] || return 1
			printf '%s\n' "${eff##*/}"
		}
	else
		die "need curl or wget to download jumpgate"
	fi
	if command -v sha256sum >/dev/null 2>&1; then
		sumcheck() { sha256sum -c -; }
	elif command -v shasum >/dev/null 2>&1; then
		sumcheck() { shasum -a 256 -c -; }
	else
		die "need sha256sum or shasum to verify the download"
	fi

	tag="$version"
	if [ "$tag" = latest ]; then
		tag="$(latest_tag)" || die "could not resolve the latest release from $base/latest"
	fi
	case "$tag" in
	v[0-9]*.[0-9]*.[0-9]*) ;;
	*) die "not a release tag: $tag" ;;
	esac
	say "jumpgate $tag"

	tmp="$(mktemp -d "${TMPDIR:-/tmp}/jumpgate-install.XXXXXX")"
	# shellcheck disable=SC2064 # expand $tmp now, it is fixed from here on
	trap "rm -rf '$tmp'" EXIT
	trap 'exit 130' INT
	trap 'exit 143' TERM

	# verify FILE: the release's checksum, then provenance when gh can check it.
	verify() {
		(cd "$tmp" && awk -v f="$1" '$2 == f || $2 == "*" f' checksums.txt | sumcheck >/dev/null 2>&1) ||
			die "checksum mismatch for $1 against the release's checksums.txt; nothing was installed"
		if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
			gh attestation verify "$tmp/$1" --repo "$repo" \
				--signer-workflow "$repo/.github/workflows/release.yml" \
				--source-ref "refs/tags/$tag" >/dev/null ||
				die "build provenance check failed for $1; nothing was installed"
			say "build provenance verified (gh attestation verify)"
		elif [ "$require_provenance" = 1 ]; then
			die "--require-provenance: build provenance not checked (gh not available or not signed in); nothing was installed"
		else
			say "build provenance not checked (gh not available); checksum verified against the release's checksums.txt over HTTPS"
		fi
	}

	dl="$base/download/$tag"
	fetch "$dl/checksums.txt" "$tmp/checksums.txt" || die "download $dl/checksums.txt failed"
	if [ "$role" = agent ]; then
		# The gate is the raw static binary the release ships beside the archives.
		file="jumpgate-linux-${arch}"
		fetch "$dl/$file" "$tmp/$file" || die "download $dl/$file failed"
		verify "$file"
		src="$tmp/$file"
		bin=/usr/local/lib/jumpgate/jumpgate
	else
		file="jumpgate_linux_${arch}.tar.gz"
		fetch "$dl/$file" "$tmp/$file" || die "download $dl/$file failed"
		verify "$file"
		mkdir "$tmp/x"
		tar -xzf "$tmp/$file" -C "$tmp/x" jumpgate || die "$file holds no jumpgate binary"
		if [ ! -f "$tmp/x/jumpgate" ] || [ -L "$tmp/x/jumpgate" ]; then
			die "$file holds no jumpgate binary"
		fi
		src="$tmp/x/jumpgate"
		bin=/usr/local/bin/jumpgate
	fi

	if [ "$role" = cli ] && [ -n "$prefix" ]; then
		mkdir -p "$prefix/bin"
		install -m 0755 "$src" "$prefix/bin/jumpgate"
		say "installed $prefix/bin/jumpgate"
		case ":$PATH:" in
		*":$prefix/bin:"*) ;;
		*) say "note: $prefix/bin is not on your PATH" ;;
		esac
		return 0
	fi

	sudo=
	if [ "$(id -u)" != 0 ]; then
		command -v sudo >/dev/null 2>&1 || die "installing to $bin needs root, and sudo is not installed; run as root, or use --prefix \"\$HOME/.local\""
		sudo=sudo
		say "this runs, with sudo:"
		say "  sudo install -D -o root -g root -m 0755 $src $bin"
		case "$role" in
		controller) say "  sudo $bin setup install-service" ;;
		agent) say "  sudo $bin agent setup --controller $controller --transport-key '$transport_key'" ;;
		esac
		if [ "$yes" != 1 ]; then
			if ! has_tty; then
				say "no terminal to confirm on; rerun with --yes, or with --prefix \"\$HOME/.local\" for no root"
				exit 1
			fi
			printf '%s' "continue? [y/N] " >/dev/tty
			read -r answer </dev/tty || answer=
			case "$answer" in
			y | Y | yes | YES) ;;
			*) die "stopped; nothing was installed" ;;
			esac
		fi
	fi

	$sudo install -D -o root -g root -m 0755 "$src" "$bin"
	say "installed $bin"
	case "$role" in
	controller)
		$sudo "$bin" setup install-service
		next="sudo jumpgate setup server"
		[ "$self_only" = 0 ] || next="$next --self-only"
		say ""
		say "root on this controller server, with the key unlocked, can sign routine intents for every box that trusts it, including agent.upgrade, which replaces a box's gate. Encryption protects the key at rest, in backups and against disk theft; it does not protect it from live root."
		say ""
		say "next: $next"
		;;
	agent)
		$sudo "$bin" agent setup --controller "$controller" --transport-key "$transport_key"
		;;
	*)
		if [ "${hint_roles:-0}" = 1 ]; then
			say ""
			say "for a controller server: curl -fsSL https://jumpgate.app/install.sh | sh -s -- --controller"
			say "for a gate-only box: run \`jumpgate hosts invite NAME\` on its controller, and run the command it prints on the box"
		fi
		;;
	esac
}

main "$@"
