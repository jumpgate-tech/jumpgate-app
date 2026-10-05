// internal/bootstrap/unit.go
package bootstrap

import "github.com/valve-tech/jumpgate/internal/agentclient"

// agentUnit runs the agent as root with the hardening that does not stop it
// from managing units and data directories. ProtectSystem=strict is
// deliberately absent: the agent's job is to change the system.
const agentUnit = `[Unit]
Description=jumpgate agent
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
NoNewPrivileges=yes
PrivateTmp=yes
ProtectHome=read-only
ProtectKernelTunables=yes
ProtectControlGroups=yes
RestrictSUIDSGID=yes

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
