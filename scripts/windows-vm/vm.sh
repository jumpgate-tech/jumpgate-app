#!/usr/bin/env bash
# scripts/windows-vm/vm.sh: a local, headless Windows Server 2022 test VM under
# QEMU + HVF (Intel Mac). Nothing leaves the machine except downloads of the
# official Microsoft ISO, Go and Git installers.
#
#   vm.sh install     # one time: unattended install, OpenSSH, then Go + Git (~30-60 min)
#   vm.sh start       # boot the installed VM (no-op if running)
#   vm.sh stop        # clean shutdown (falls back to ACPI power button, then kill)
#   vm.sh status      # running or not, and whether SSH answers
#   vm.sh wait [sec]  # wait until SSH answers (default 600 s)
#   vm.sh ssh [cmd]   # SSH in (default shell is Windows PowerShell)
#   vm.sh provision   # (re)install Go matching go.mod's minor version, and Git for Windows
#   vm.sh screenshot  # dump the VGA console to <vm dir>/screen.png (debugging)
#
# State lives in $JUMPGATE_WIN_VM_DIR (default ~/vms/jumpgate-win), never in the
# repo: disk.qcow2, the ISO, admin-password (random, mode 600), id_ed25519 (the
# SSH key for this VM only), the rendered answer ISO, sockets and logs.
#
# Hardware: q35, AHCI disk and CD-ROMs, e1000e NIC; Windows has inbox drivers
# for all of them, so no virtio-win ISO is needed. User-mode networking
# forwards 127.0.0.1:$JUMPGATE_WIN_SSH_PORT (default 2222) to the guest's 22.
# Other knobs: JUMPGATE_WIN_CPU (default Nehalem), JUMPGATE_WIN_CPUS (4),
# JUMPGATE_WIN_MEM (8G), JUMPGATE_WIN_DISK_SIZE (64G, install only),
# JUMPGATE_WIN_QEMU_EXTRA (extra qemu arguments, word-split).
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
VM_DIR="${JUMPGATE_WIN_VM_DIR:-$HOME/vms/jumpgate-win}"
SSH_PORT="${JUMPGATE_WIN_SSH_PORT:-2222}"
# -cpu host bugchecks Windows Setup (IRQL_NOT_LESS_OR_EQUAL while applying
# install.wim) under QEMU 11.1 + HVF on an i7-8700B, with 1 or 4 vCPUs; the
# Nehalem model (no AVX/XSAVE) installs and runs cleanly.
CPU="${JUMPGATE_WIN_CPU:-Nehalem}"
CPUS="${JUMPGATE_WIN_CPUS:-4}"
MEM="${JUMPGATE_WIN_MEM:-8G}"
DISK_SIZE="${JUMPGATE_WIN_DISK_SIZE:-64G}"
ISO_URL='https://go.microsoft.com/fwlink/p/?LinkID=2195280&clcid=0x409&culture=en-us&country=US'
ISO="$VM_DIR/SERVER_EVAL_x64FRE_en-us.iso"
DISK="$VM_DIR/disk.qcow2"
PIDFILE="$VM_DIR/qemu.pid"
MONITOR="$VM_DIR/monitor.sock"
SERIAL="$VM_DIR/serial.sock"
KEY="$VM_DIR/id_ed25519"

log() { printf '[windows-vm] %s\n' "$*" >&2; }
die() { log "error: $*"; exit 1; }

ssh_opts=(-i "$KEY" -p "$SSH_PORT" -o IdentitiesOnly=yes -o BatchMode=yes
  -o StrictHostKeyChecking=accept-new -o "UserKnownHostsFile=$VM_DIR/known_hosts"
  -o ConnectTimeout=10 -o ServerAliveInterval=30 -o LogLevel=ERROR)
scp_opts=(-i "$KEY" -P "$SSH_PORT" -o IdentitiesOnly=yes -o BatchMode=yes
  -o StrictHostKeyChecking=accept-new -o "UserKnownHostsFile=$VM_DIR/known_hosts"
  -o ConnectTimeout=10 -o LogLevel=ERROR)
TARGET=Administrator@127.0.0.1

vm_ssh() { ssh "${ssh_opts[@]}" "$TARGET" "$@"; }
vm_scp() { scp "${scp_opts[@]}" "$@"; }   # remote paths: "$TARGET:C:/dir/file"

running() { [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; }
ssh_up() { vm_ssh 'exit 0' </dev/null >/dev/null 2>&1; }
monitor() { printf '%s\n' "$1" | nc -U -w 2 "$MONITOR" >/dev/null 2>&1 || true; }

need_tools() {
  command -v qemu-system-x86_64 >/dev/null || die "qemu missing: brew install qemu"
  [ "$(sysctl -n kern.hv_support 2>/dev/null)" = 1 ] || die "HVF (Hypervisor.framework) not available"
}

ensure_secrets() {
  mkdir -p "$VM_DIR"
  [ -f "$KEY" ] || ssh-keygen -q -t ed25519 -N '' -C jumpgate-win-vm -f "$KEY"
  if [ ! -f "$VM_DIR/admin-password" ]; then
    (umask 077; printf 'Jg-%s!a9' "$(openssl rand -base64 18 | tr -d '/+=')" >"$VM_DIR/admin-password")
  fi
  chmod 600 "$KEY" "$VM_DIR/admin-password"
}

fetch_iso() {
  [ -f "$ISO" ] && [ "$(stat -f %z "$ISO")" -gt 4000000000 ] && return
  log "downloading the Windows Server 2022 evaluation ISO from Microsoft (~4.7 GB)"
  local url
  url="$(curl -sIL -A 'Mozilla/5.0' "$ISO_URL" | awk 'tolower($1)=="location:"{print $2}' | tr -d '\r' | tail -1)"
  case "$url" in https://*.microsoft.com/*) ;; *) die "unexpected redirect target: $url" ;; esac
  curl -fL -A 'Mozilla/5.0' -C - -o "$ISO" "$url"
}

build_answer_iso() {
  local stage="$VM_DIR/answer"
  rm -rf "$stage" "$VM_DIR/answer.iso"
  mkdir -p "$stage"
  local pw; pw="$(cat "$VM_DIR/admin-password")"
  # The password alphabet (base64 minus /+= plus "Jg-", "!a9") needs no XML escaping.
  sed "s/@ADMIN_PASSWORD@/$pw/g" "$here/autounattend.xml.tmpl" >"$stage/autounattend.xml"
  cp "$here/jumpgate-firstboot.ps1" "$stage/"
  cp "$KEY.pub" "$stage/jumpgate_ed25519.pub"
  hdiutil makehybrid -quiet -iso -joliet -default-volume-name UNATTEND -o "$VM_DIR/answer.iso" "$stage"
  chmod 600 "$VM_DIR/answer.iso" "$stage/autounattend.xml"
}

# launch [install]: with "install", boot once from the Windows ISO and attach the
# answer ISO; Setup's reboots then come up from the disk inside the same process.
launch() {
  need_tools
  running && { log "already running (pid $(cat "$PIDFILE"))"; return; }
  [ -f "$DISK" ] || die "no disk at $DISK; run: $0 install"
  rm -f "$MONITOR" "$SERIAL"
  local args=(
    -name jumpgate-win
    -machine q35,accel=hvf -cpu "$CPU" -smp "$CPUS" -m "$MEM"
    -rtc base=utc
    -drive "file=$DISK,if=none,id=hd0,format=qcow2,cache=writeback,discard=unmap"
    -device ide-hd,drive=hd0,bus=ide.0
    -netdev "user,id=n0,hostfwd=tcp:127.0.0.1:$SSH_PORT-:22"
    -device e1000e,netdev=n0
    -display none -vga std
    -monitor "unix:$MONITOR,server,nowait"
    -serial "unix:$SERIAL,server,nowait"
    -pidfile "$PIDFILE" -daemonize
  )
  if [ "${1:-}" = install ]; then
    args+=(
      -drive "file=$ISO,if=none,id=cd0,media=cdrom,readonly=on" -device ide-cd,drive=cd0,bus=ide.1
      -drive "file=$VM_DIR/answer.iso,if=none,id=cd1,media=cdrom,readonly=on" -device ide-cd,drive=cd1,bus=ide.2
      -boot order=c,once=d
    )
  else
    args+=(-boot order=c)
  fi
  # shellcheck disable=SC2206 # JUMPGATE_WIN_QEMU_EXTRA is split on purpose
  [ -n "${JUMPGATE_WIN_QEMU_EXTRA:-}" ] && args+=(${JUMPGATE_WIN_QEMU_EXTRA})
  qemu-system-x86_64 "${args[@]}" 2>>"$VM_DIR/qemu.log"
  log "started (pid $(cat "$PIDFILE")); ssh on 127.0.0.1:$SSH_PORT"
}

wait_ssh() {
  local limit="${1:-600}" start=$SECONDS
  until ssh_up; do
    running || die "VM is not running"
    [ $((SECONDS - start)) -ge "$limit" ] && die "SSH not up after ${limit}s"
    sleep 30
  done
  log "SSH is up after $((SECONDS - start))s"
}

stop_vm() {
  running || { log "not running"; return; }
  local pid; pid="$(cat "$PIDFILE")"
  if ssh_up; then
    vm_ssh 'shutdown.exe /s /t 0 /f' </dev/null >/dev/null 2>&1 || true
  else
    monitor system_powerdown
  fi
  for _ in $(seq 1 60); do kill -0 "$pid" 2>/dev/null || { log "stopped"; rm -f "$PIDFILE"; return; }; sleep 2; done
  log "no clean shutdown after 120s; quitting QEMU"
  monitor quit; sleep 2; kill "$pid" 2>/dev/null || true; rm -f "$PIDFILE"
}

# Latest Go release with go.mod's major.minor (CI's setup-go uses "1.25" the same way).
go_msi() {
  local minor
  minor="$(awk '$1=="go"{split($2,v,"."); print v[1]"."v[2]; exit}' "${GOMOD:-$repo/go.mod}")"
  curl -fsS 'https://go.dev/dl/?mode=json&include=all' | python3 -c '
import json, sys
minor = sys.argv[1]
for r in json.load(sys.stdin):
    if r["stable"] and (r["version"] + ".").startswith("go" + minor + "."):
        for f in r["files"]:
            if (f["os"], f["arch"], f["kind"]) == ("windows", "amd64", "installer"):
                print(r["version"], f["filename"], f["sha256"]); sys.exit()
sys.exit("no Windows MSI for go" + minor)' "$minor"
}

git_exe() {
  curl -fsS https://api.github.com/repos/git-for-windows/git/releases/latest | python3 -c '
import json, sys
d = json.load(sys.stdin)
for a in d["assets"]:
    n = a["name"]
    if n.startswith("Git-") and n.endswith("-64-bit.exe"):
        print(d["tag_name"], n, a["browser_download_url"], (a.get("digest") or "").removeprefix("sha256:")); sys.exit()
sys.exit("no Git for Windows 64-bit installer in the latest release")'
}

fetch_verified() { # url file sha256
  local out="$VM_DIR/downloads/$2"
  mkdir -p "$VM_DIR/downloads"
  if [ ! -f "$out" ]; then
    curl -fL -sS -o "$out.part" "$1" || die "download failed: $1"
    mv "$out.part" "$out"
  fi
  if [ -n "$3" ] && [ "$(shasum -a 256 "$out" | cut -d' ' -f1)" != "$3" ]; then
    rm -f "$out"; die "sha256 mismatch for $2"
  fi
  printf '%s\n' "$out"
}

provision() {
  wait_ssh 60
  local gover gofile gosha gotag gitfile giturl gitsha
  read -r gover gofile gosha < <(go_msi)
  read -r gotag gitfile giturl gitsha < <(git_exe)
  local have
  have="$(vm_ssh 'if (Test-Path "C:\Program Files\Go\bin\go.exe") { & "C:\Program Files\Go\bin\go.exe" env GOVERSION }' </dev/null | tr -d '\r' || true)"
  vm_ssh 'New-Item -ItemType Directory -Force C:\jg\dl | Out-Null' </dev/null
  if [ "$have" != "$gover" ]; then
    log "installing $gover (have: ${have:-none})"
    vm_scp "$(fetch_verified "https://go.dev/dl/$gofile" "$gofile" "$gosha")" "$TARGET:C:/jg/dl/$gofile"
    vm_ssh "\$p = Start-Process msiexec.exe -Wait -PassThru -ArgumentList '/i','C:\\jg\\dl\\$gofile','/qn','/norestart'; exit \$p.ExitCode" </dev/null
  else
    log "Go $gover already installed"
  fi
  if ! vm_ssh "if (-not (Test-Path 'C:\\Program Files\\Git\\cmd\\git.exe')) { exit 1 }" </dev/null; then
    log "installing Git for Windows $gotag"
    vm_scp "$(fetch_verified "$giturl" "$gitfile" "$gitsha")" "$TARGET:C:/jg/dl/$gitfile"
    vm_ssh "\$p = Start-Process 'C:\\jg\\dl\\$gitfile' -Wait -PassThru -ArgumentList '/VERYSILENT','/NORESTART','/NOCANCEL','/SP-','/SUPPRESSMSGBOXES'; exit \$p.ExitCode" </dev/null
  else
    log "Git already installed"
  fi
  vm_ssh '& "C:\Program Files\Go\bin\go.exe" version; & "C:\Program Files\Git\cmd\git.exe" --version; tar.exe --version' </dev/null
}

install_vm() {
  need_tools
  command -v hdiutil >/dev/null || die "hdiutil missing (macOS only)"
  running && die "VM is running; stop it first"
  [ -f "$DISK" ] && [ "${FORCE:-0}" != 1 ] && die "$DISK exists; FORCE=1 $0 install to wipe and reinstall"
  local avail; avail="$(df -k "$HOME" | awk 'NR==2{print int($4/1024/1024)}')"
  [ "$avail" -ge 80 ] || die "need about 80 GB free in $HOME, have ${avail} GB"
  ensure_secrets
  fetch_iso
  build_answer_iso
  rm -f "$DISK" "$VM_DIR/known_hosts"
  qemu-img create -q -f qcow2 "$DISK" "$DISK_SIZE"
  launch install
  log "Windows Setup is running unattended; waiting for SSH (up to ${INSTALL_TIMEOUT:-3600}s)"
  wait_ssh "${INSTALL_TIMEOUT:-3600}"
  provision
  log "installed. The install ISOs stay attached until the next stop/start."
}

screenshot() {
  running || die "not running"
  rm -f "$VM_DIR/screen.ppm"
  monitor "screendump $VM_DIR/screen.ppm"
  sleep 1
  [ -f "$VM_DIR/screen.ppm" ] || die "screendump failed"
  if command -v magick >/dev/null; then magick "$VM_DIR/screen.ppm" "$VM_DIR/screen.png"
  else sips -s format png "$VM_DIR/screen.ppm" --out "$VM_DIR/screen.png" >/dev/null; fi
  printf '%s\n' "$VM_DIR/screen.png"
}

cmd="${1:-status}"; [ $# -gt 0 ] && shift
case "$cmd" in
  install) install_vm ;;
  start) launch ;;
  stop) stop_vm ;;
  status)
    if running; then
      if ssh_up; then echo "running (pid $(cat "$PIDFILE")), ssh up on 127.0.0.1:$SSH_PORT"
      else echo "running (pid $(cat "$PIDFILE")), ssh not answering yet"; fi
    else echo "stopped"; fi ;;
  wait) wait_ssh "${1:-600}" ;;
  ssh) exec ssh "${ssh_opts[@]}" "$TARGET" "$@" ;;
  scp) vm_scp "$@" ;;
  provision) provision ;;
  screenshot) screenshot ;;
  *) sed -n '2,20p' "$0"; exit 2 ;;
esac
