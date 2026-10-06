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
