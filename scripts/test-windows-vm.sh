#!/usr/bin/env bash
# scripts/test-windows-vm.sh: run go vet and go test on real Windows, in the
# local QEMU VM managed by scripts/windows-vm/vm.sh (install it once with
# `scripts/windows-vm/vm.sh install`). The source goes over SSH to 127.0.0.1;
# nothing leaves the machine.
#
#   scripts/test-windows-vm.sh                          # this worktree, go vet + go test -p 2 ./...
#   scripts/test-windows-vm.sh ../other-worktree        # another worktree
#   scripts/test-windows-vm.sh . ./internal/fsperm/ -v  # any go test arguments after the dir
#   HEAD_ONLY=1 scripts/test-windows-vm.sh              # git archive HEAD instead of the working tree
#   VET=0 scripts/test-windows-vm.sh                    # skip go vet
#   KEEP_RUN=1 scripts/test-windows-vm.sh               # keep C:\jg\runs\<id> in the VM afterwards
#
# By default the working tree is sent as is: tracked files with uncommitted
# edits plus untracked files that .gitignore does not exclude. In the VM,
# USERPROFILE, HOME, APPDATA and LOCALAPPDATA point at a fresh per-run directory;
# GOPATH (module cache) and GOCACHE are shared under C:\jg so reruns are fast.
# Exit status: 0 only if both go vet and go test pass.
#
# JUMPGATE_TEST_WIN_AGENT=1 also enables the OpenSSH agent tests. Win32-OpenSSH's
# ssh-agent cannot ssh-add keys from a pubkey SSH logon (DPAPI needs a
# password-based logon session), so with this flag run.cmd executes inside a
# one-shot scheduled task running as Administrator with the stored password.
# The host pipes ~/vms/jumpgate-win/admin-password to the VM over ssh STDIN only;
# PowerShell reads it into a variable, registers the task, waits, collects the
# log and exit code, and deletes the task. The password never reaches disk or
# any argv. NEVER change the Administrator password to make this work; the
# stored one is the only one.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
vm="$here/windows-vm/vm.sh"

src="${1:-.}"; [ $# -gt 0 ] && shift
[ $# -eq 0 ] && set -- ./...
src="$(git -C "$src" rev-parse --show-toplevel)"

"$vm" start
"$vm" wait 900

run="$(date +%Y%m%d-%H%M%S)-$$"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT

if [ "${HEAD_ONLY:-0}" = 1 ]; then
  git -C "$src" archive --format=tar.gz -o "$tmp/src.tgz" HEAD
  echo "[test-windows-vm] $src at $(git -C "$src" rev-parse --short HEAD) (HEAD only)" >&2
else
  # Tracked (with uncommitted edits) and untracked-but-not-ignored files; skip
  # tracked files deleted in the working tree.
  (cd "$src" && git ls-files -z -co --exclude-standard \
    | while IFS= read -r -d '' f; do [ -e "$f" ] || [ -L "$f" ] && printf '%s\0' "$f"; done \
    | COPYFILE_DISABLE=1 tar --null -czf "$tmp/src.tgz" -T -)
  echo "[test-windows-vm] $src at $(git -C "$src" rev-parse --short HEAD) plus working-tree changes" >&2
fi

# go test arguments, quoted for cmd.exe.
targs=""
for a in "$@"; do targs+=" \"${a//\"/\\\"}\""; done

# CRLF line endings: cmd.exe misparses some LF-only batch files.
{
  cat <<EOF
@echo off
setlocal
set RUN=C:\\jg\\runs\\$run
set PATH=C:\\Program Files\\Go\\bin;C:\\Program Files\\Git\\cmd;%PATH%
set USERPROFILE=%RUN%\\home
set HOME=%RUN%\\home
set APPDATA=%RUN%\\home\\AppData\\Roaming
set LOCALAPPDATA=%RUN%\\home\\AppData\\Local
set GOPATH=C:\\jg\\gopath
set GOCACHE=C:\\jg\\gocache
set GOFLAGS=-buildvcs=false
$([ "${JUMPGATE_TEST_WIN_AGENT:-0}" = 1 ] && echo 'set JUMPGATE_TEST_WIN_AGENT=1')
mkdir "%APPDATA%" "%LOCALAPPDATA%" "%RUN%\\src" 2>nul
tar.exe -xzf "%RUN%\\src.tgz" -C "%RUN%\\src" || exit /b 3
cd /d "%RUN%\\src"
go version
set VETRC=0
if "${VET:-1}"=="0" goto test
echo ==== go vet ./...
go vet ./...
set VETRC=%ERRORLEVEL%
:test
echo ==== go test -p 2$targs
go test -p 2$targs
set TESTRC=%ERRORLEVEL%
echo ==== go vet exit %VETRC%, go test exit %TESTRC%
if not "%VETRC%"=="0" exit /b 1
if not "%TESTRC%"=="0" exit /b 1
exit /b 0
EOF
} | sed 's/$/\r/' >"$tmp/run.cmd"

"$vm" ssh "New-Item -ItemType Directory -Force C:\\jg\\runs\\$run | Out-Null" </dev/null
"$vm" scp "$tmp/src.tgz" "$tmp/run.cmd" "Administrator@127.0.0.1:C:/jg/runs/$run/"
rc=0
if [ "${JUMPGATE_TEST_WIN_AGENT:-0}" = 1 ]; then
  pwfile="${JUMPGATE_WIN_VM_DIR:-$HOME/vms/jumpgate-win}/admin-password"
  [ -r "$pwfile" ] || { echo "[test-windows-vm] missing $pwfile" >&2; exit 1; }
  # The PowerShell below holds no secret; the password arrives on stdin.
  cat >"$tmp/agent.ps1.in" <<'PS'
$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue'
$pw = [Console]::In.ReadLine()
$dir = 'C:\jg\runs\@RUN@'
$name = 'jg-run-@RUN@'
$log = "$dir\out.log"; $rcf = "$dir\rc.txt"
$inner = '& cmd.exe /c "' + "$dir\run.cmd >$log 2>&1" + '"; ' + '$LASTEXITCODE' + " | Out-File -Encoding ascii '$rcf'"
$enc = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($inner))
$act = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument "-NoProfile -NonInteractive -EncodedCommand $enc"
$rc = 1; $off = 0
try {
  Register-ScheduledTask -TaskName $name -Action $act -User "$env:COMPUTERNAME\Administrator" -Password $pw -RunLevel Highest -Force | Out-Null
  $pw = $null
  Start-ScheduledTask -TaskName $name
  $t0 = Get-Date
  while ($true) {
    Start-Sleep -Seconds 2
    if (Test-Path $log) {
      try {
        $fs = [IO.File]::Open($log, 'Open', 'Read', 'ReadWrite,Delete')
        $len = $fs.Length
        if ($len -gt $off) {
          [void]$fs.Seek($off, 'Begin')
          $buf = New-Object byte[] ($len - $off)
          $n = $fs.Read($buf, 0, $buf.Length); $off += $n
          [Console]::Out.Write([Text.Encoding]::Default.GetString($buf, 0, $n))
        }
        $fs.Close()
      } catch {}
    }
    if (Test-Path $rcf) { Start-Sleep -Seconds 1; break }
    if ((Get-ScheduledTask -TaskName $name).State -ne 'Running' -and ((Get-Date) - $t0).TotalSeconds -gt 60) { Write-Output 'task ended without an exit code'; break }
  }
  if (Test-Path $rcf) { $rc = [int](Get-Content $rcf -Raw).Trim() }
} finally {
  $pw = $null
  Unregister-ScheduledTask -TaskName $name -Confirm:$false -ErrorAction SilentlyContinue
}
exit $rc
PS
  enc="$(sed "s/@RUN@/$run/g" "$tmp/agent.ps1.in" | iconv -f UTF-8 -t UTF-16LE | base64 | tr -d '\n')"
  "$vm" ssh "powershell.exe -NoProfile -NonInteractive -OutputFormat Text -EncodedCommand $enc" <"$pwfile" || rc=$?
else
  "$vm" ssh "cmd.exe /c C:\\jg\\runs\\$run\\run.cmd; exit \$LASTEXITCODE" </dev/null || rc=$?
fi
if [ "${KEEP_RUN:-0}" != 1 ]; then
  "$vm" ssh "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue C:\\jg\\runs\\$run" </dev/null || true
fi
exit "$rc"
