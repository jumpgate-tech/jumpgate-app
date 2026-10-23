#!/bin/bash
# scripts/macos-runner/install.sh [install|uninstall]
# Registers a project-locked GitLab shell runner (tag `macos`) for this Mac and
# runs it as a launchd user agent (tech.jumpgate.macos-runner). Same layout as
# scripts/windows-vm: installed files in ~/Library/Application Support, logs in
# ~/Library/Logs. Host is git.craftsmanfounder.com, never gitlab.com.
# Needs a GitLab token with `api` scope, read from the keychain via
# `git credential fill` (never printed, never written to disk by this script).
# A shell executor runs jobs as this user, so the runner is locked to project
# 71 and runs only jobs tagged `macos`.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOST=git.craftsmanfounder.com
PROJECT=71
label=tech.jumpgate.macos-runner
bin="$HOME/Library/Application Support/jumpgate-macos-runner"
cfg="$bin/config.toml"
plist="$HOME/Library/LaunchAgents/$label.plist"
dom="gui/$(id -u)"
case "${1:-install}" in
  install)
    mkdir -p "$bin/builds" "$HOME/Library/LaunchAgents" "$HOME/Library/Logs"
    if [ ! -x "$bin/gitlab-runner" ]; then
      arch=amd64; [ "$(uname -m)" = arm64 ] && arch=arm64
      curl -fsSL -o "$bin/gitlab-runner" "https://gitlab-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-runner-darwin-$arch"
      chmod +x "$bin/gitlab-runner"
    fi
    if [ ! -f "$cfg" ]; then
      pat="$(printf 'protocol=https\nhost=%s\n\n' "$HOST" | git credential fill | sed -n 's/^password=//p')"
      resp="$(printf 'header = "PRIVATE-TOKEN: %s"\n' "$pat" | curl -fsS -K - -X POST "https://$HOST/api/v4/user/runners" \
        --data-urlencode runner_type=project_type --data-urlencode "project_id=$PROJECT" \
        --data-urlencode "description=jumpgate macOS (this Mac)" --data-urlencode tag_list=macos \
        --data-urlencode run_untagged=false --data-urlencode locked=true)"
      pat=""
      umask 077
      printf '%s' "$resp" | jq -r --arg h "https://$HOST" \
        '"concurrent = 1\n\n[[runners]]\n  name = \"jumpgate macOS (this Mac)\"\n  url = \"\($h)\"\n  token = \"\(.token)\"\n  executor = \"shell\"\n  shell = \"bash\"\n"' > "$cfg"
      resp=""
      echo "registered runner; config at $cfg"
    fi
    path="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"
    sed -e "s|@BIN@|$bin|g" -e "s|@PATH@|$path|g" -e "s|@HOME@|$HOME|g" "$here/runner.plist.tmpl" > "$plist"
    plutil -lint "$plist" >/dev/null
    launchctl bootout "$dom/$label" 2>/dev/null || true
    launchctl bootstrap "$dom" "$plist"
    echo "installed $plist (log: ~/Library/Logs/jumpgate-macos-runner.log)" ;;
  uninstall)
    launchctl bootout "$dom/$label" 2>/dev/null || true
    rm -f "$plist"; echo "removed $label (runner config left in $bin; delete the runner in GitLab too)" ;;
  *) echo "usage: $0 [install|uninstall]" >&2; exit 2 ;;
esac
