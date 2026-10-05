# Running a node on your desktop (the workaround)

jumpgate does not run a real node (chains 1, 369, 943) directly on macOS or
Windows, and setup refuses a non-Linux target. If you still want one on a
desktop, run it in a Linux VM and treat that VM as an ordinary Linux box:
**WSL2 Ubuntu** on Windows, or a **Lima** (or colima) Ubuntu VM on macOS. The
controller pairs with it over SSH like any remote server.

Be honest with yourself about three things first.

## The reality

- **Disk.** Chain data is 1.2 TB (PulseChain testnet v4), 3.6 TB (Ethereum) or
  3.9 TB (PulseChain), before consensus-client data. Most laptops have 0.5 to 2
  TB. Plan a dedicated external NVMe, and put the VM's virtual disk on it. Never
  point the data dir at a host-shared folder (`/mnt/c` in WSL, Lima's
  virtiofs/sshfs mounts): random I/O there is unusably slow. The data dir must
  be on the VM's own ext4 disk.
- **Sleep and reboots.** A desktop sleeps, installs updates and reboots, and
  WSL and Lima stop the VM when idle or at logout. A stopped node falls behind
  and resyncs. Disable sleep, set `vmIdleTimeout=-1` in `.wslconfig` (WSL), and
  start Lima at boot (`limactl start-at-login` or a launchd job). Do not run
  validators this way.
- **NAT.** The VM sits behind NAT, so inbound peers cannot reach it unless you
  forward the P2P ports (execution 30303 TCP/UDP, consensus ports per client) from
  your router to the host and from the host into the VM. Without that you get few
  peers and slow sync.

## WSL2 Ubuntu on Windows

1. In an admin PowerShell: `wsl --install -d Ubuntu`.
2. Enable systemd: in Ubuntu, add `[boot]` / `systemd=true` to `/etc/wsl.conf`,
   then `wsl --shutdown` and reopen.
3. Move the virtual disk to the big drive if needed (`wsl --manage Ubuntu --move
   D:\wsl`), and raise its size cap (`wsl --manage Ubuntu --resize <size>`).
4. Install and start sshd: `sudo apt install openssh-server`, then
   `sudo systemctl enable --now ssh`. Add your public key to root's (or a sudo
   user's) `authorized_keys`.
5. Find the address: from Windows, `localhost` forwards to WSL2 only in some
   modes, so use the VM address from `wsl hostname -I`, or enable mirrored
   networking (`networkingMode=mirrored` in `.wslconfig`).

## Lima Ubuntu VM on macOS

1. `brew install lima`, then
   `limactl start --name=node template://ubuntu --disk=<size>GiB`
   (set the disk location with `LIMA_HOME` on the external drive).
2. Lima's Ubuntu has systemd and passwordless sudo. Find the SSH port with
   `limactl list` (the SSH column), or use `limactl show-ssh node`.

## Pair it

From the controller, using the forwarded SSH address and port:

```bash
jumpgate hosts add desk-node --ssh root@127.0.0.1:60022 --key ~/.ssh/id_ed25519
# or, for a non-root sudo user:
jumpgate hosts add desk-node --ssh ubuntu@127.0.0.1:60022 --sudo
```

Confirm the host-key fingerprint when asked. After that, set up the node from the
panel or CLI as for any Linux server. Preflight detects a WSL2 or Lima target and
prints a warning about sleep, NAT and disk; it is a warning only and does not
block setup.

If you run `jumpgate` inside the WSL2 Ubuntu itself, you can pair that box with
`jumpgate hosts add me --local` (as root).
