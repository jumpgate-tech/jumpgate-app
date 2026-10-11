# Windows test VM

`vm.sh` manages a local, headless Windows Server 2022 VM (QEMU + HVF);
`../test-windows-vm.sh` runs `go vet` and `go test` in it. See the headers of
both scripts for usage. State lives in `~/vms/jumpgate-win/` (SSH key
`id_ed25519`, `admin-password`, mode 600).

## The ssh-agent tests (`JUMPGATE_TEST_WIN_AGENT=1`)

Win32-OpenSSH's ssh-agent cannot `ssh-add` keys from a pubkey SSH logon: DPAPI
needs a password-based logon session. With `JUMPGATE_TEST_WIN_AGENT=1` the
script therefore:

1. forwards `JUMPGATE_TEST_WIN_AGENT=1` into the generated `run.cmd`;
2. pipes `~/vms/jumpgate-win/admin-password` to the VM over ssh STDIN only;
3. in PowerShell on the VM, reads it into a variable, registers a one-shot
   scheduled task (`jg-run-<id>`) as Administrator with that password, runs it,
   streams its log, reads the exit code from a file, and unregisters the task.

The password never reaches disk or any argv. Without the flag, tests run
directly in the SSH session as before.

**Never change the Administrator password** (to work around a failing logon or
for any other reason). The stored password in `admin-password` is the only
one; if it stops validating, reinstall the VM instead.

## Start-on-demand launcher

A launchd user agent (`tech.jumpgate.win-vm-on-demand`) runs `on-demand.sh`
every 180 s so the VM only uses its 8 GB / 4 vCPUs while CI needs it.

```
scripts/windows-vm/install-on-demand.sh             # render the plist, launchctl bootstrap
scripts/windows-vm/install-on-demand.sh uninstall   # bootout and remove the plist
scripts/windows-vm/on-demand.sh --dry-run           # log the decision, change nothing
scripts/windows-vm/on-demand.sh --adopt             # let the launcher manage a hand-started VM
```

Each tick asks the GitLab jobs API (project 71 on git.craftsmanfounder.com,
token from the macOS keychain via `git credential fill`, never written to
disk) for pending and running jobs and keeps those tagged `windows`:

- a `windows` job is pending and the VM is stopped: `vm.sh start`, and write
  the marker `~/vms/jumpgate-win/on-demand.managed`;
- the VM is running with the marker and no `windows` job has been pending or
  running for the idle period: `vm.sh stop` (clean shutdown). The idle start
  is kept in `~/vms/jumpgate-win/on-demand.idle-since`. A running job always
  resets the timer, so the VM is never stopped under a job;
- **manual override**: a running VM without the marker (you ran `vm.sh start`)
  is never stopped by the launcher. Run `on-demand.sh --adopt` to hand it over;
- API, token or `vm.sh` errors are logged and the tick does nothing; the next
  tick retries. Never a stop on error. A lock directory prevents overlap.

The idle period is `JUMPGATE_OD_IDLE_MIN` (default 20 minutes); set it in the
plist's `EnvironmentVariables` and re-run `install-on-demand.sh`.

The `windows` CI job is `when: manual`, so it only becomes pending once
someone plays it; the launcher reacts to that.

The agent runs copies of `on-demand.sh` and `vm.sh` placed in
`~/Library/Application Support/jumpgate-win-vm/` (launchd cannot read
scripts under `~/Documents`); re-run `install-on-demand.sh` after editing them.

Log: `~/Library/Logs/jumpgate-win-vm-on-demand.log` (launchd stdout/stderr in
`jumpgate-win-vm-on-demand.out`). Inspect the agent with
`launchctl print gui/$(id -u)/tech.jumpgate.win-vm-on-demand`.
