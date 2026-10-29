// internal/bootstrap/unit.go
package bootstrap

import "github.com/valve-tech/jumpgate/internal/agentclient"

// agentUnit runs the gate as root with the controller unit's hardening
// (packaging/linux/server/jumpgate.service) wherever the gate's job allows
// it. The gate changes the system: plans run apt-get, useradd, ufw and
// wg-quick, write units under /etc/systemd/system, binaries under /usr/local
// and node data under /var/lib, and drive systemctl over D-Bus. So these
// controller settings are deliberately left out:
//
//   - ProtectSystem=strict (or yes): apt-get writes /usr, the upgrade writes
//     /usr/local/lib/jumpgate, and setup writes /etc and data directories.
//   - ProtectHome=yes: kept at read-only, so a plan can still read an
//     operator's files but never change them.
//   - CapabilityBoundingSet= (empty): a root without capabilities cannot
//     chown, useradd, mount or configure the firewall. The set instead drops
//     only capabilities no plan uses (boot, ptrace, raw I/O, clock, MAC,
//     BPF, ...).
//   - ProtectKernelModules: ufw and wg-quick can need iptables or wireguard
//     modules loaded on first use.
//   - MemoryDenyWriteExecute: it is inherited by every child, and plans run
//     package maintainer scripts and toolchains the gate does not choose.
//   - SystemCallFilter=@system-service: children need @mount, @privileged
//     and other groups (useradd, iptables); only groups no plan uses are
//     denied.
//   - PrivateDevices: dpkg maintainer scripts and disk work need the real
//     /dev.
//   - ProtectClock: it implies a DeviceAllow list, which closes every other
//     device (disks among them); @clock and CAP_SYS_TIME are denied instead.
//   - ProtectProc=invisible: with CAP_SYS_PTRACE dropped it would hide the
//     node services' processes from the tools plans run (ps, pgrep).
//   - UMask=0077 and StateDirectoryMode=0700: data directories and files
//     created by plans for the node accounts must stay readable to them.
const agentUnit = `[Unit]
Description=jumpgate gate
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/lib/jumpgate/jumpgate agent run
Restart=on-failure
RestartSec=5
User=root
RuntimeDirectory=jumpgate
RuntimeDirectoryMode=0755
StateDirectory=jumpgate
LoadCredential=agent.key:/var/lib/jumpgate/agent.key
LimitCORE=0
NoNewPrivileges=yes
PrivateTmp=yes
ProtectHome=read-only
ProtectKernelTunables=yes
ProtectKernelLogs=yes
ProtectControlGroups=yes
ProtectHostname=yes
RestrictSUIDSGID=yes
RestrictRealtime=yes
RestrictNamespaces=yes
LockPersonality=yes
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK
SystemCallArchitectures=native
SystemCallFilter=~@clock @cpu-emulation @debug @obsolete @raw-io @reboot @swap
CapabilityBoundingSet=~CAP_SYS_BOOT CAP_SYS_PTRACE CAP_SYS_RAWIO CAP_SYS_TIME CAP_SYS_PACCT CAP_SYS_TTY_CONFIG CAP_MAC_ADMIN CAP_MAC_OVERRIDE CAP_WAKE_ALARM CAP_BLOCK_SUSPEND CAP_LINUX_IMMUTABLE CAP_SYSLOG CAP_LEASE CAP_BPF CAP_PERFMON CAP_CHECKPOINT_RESTORE

[Install]
WantedBy=multi-user.target
`

// sshdDropIn confines the tunnel user to the agent's socket. OpenSSH checks
// direct-streamlocal channels against the same local-forward permissions as
// TCP ones, so `AllowTcpForwarding no` would refuse the socket too. Instead
// local forwarding stays on and PermitOpen pins it to the one socket path: a
// TCP request names a host:port that never matches, and any other socket is
// refused. The brackets let PermitOpen parse a path as the host.
//
// That path form is undocumented: sshd_config(5) and the authorized_keys
// permitopen option only describe host:port. It works because sshd stores the
// bracketed path as the "host" with any port, and open_match (channels.c),
// which channel_connect_to_path calls with port PORT_STREAMLOCAL, compares the
// requested path to that host. open_match is identical in OpenSSH 9.2p1
// (Debian 12) and 9.6p1 (Ubuntu 24.04). scripts/e2e-agent.sh guards it on both:
// the agent socket must be reachable, another socket and TCP refused.
const sshdDropIn = `# Managed by jumpgate. The jumpgate user may only reach the agent's socket.
Match User jumpgate
    AllowTcpForwarding local
    AllowStreamLocalForwarding local
    PermitOpen [` + agentclient.DefaultSocket + `]:*
    PermitTTY no
    X11Forwarding no
    AllowAgentForwarding no
`

// transportKeyOptions restrict the tunnel key itself. permitopen repeats the
// drop-in's PermitOpen so the key stays pinned to the agent socket even if an
// admin's Match block, loaded before 50-jumpgate.conf, sets PermitOpen or
// AllowTcpForwarding first (sshd keeps the first value it reads). sshd only
// allows an open that both the key options and PermitOpen admit.
const transportKeyOptions = `restrict,port-forwarding,permitopen="[` + agentclient.DefaultSocket + `]:*",command="/bin/false"`
