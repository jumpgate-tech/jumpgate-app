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
const sshdDropIn = `# Managed by jumpgate. The jumpgate user may only reach the agent's socket.
Match User jumpgate
    AllowTcpForwarding local
    AllowStreamLocalForwarding local
    PermitOpen [` + agentclient.DefaultSocket + `]:*
    PermitTTY no
    X11Forwarding no
    AllowAgentForwarding no
`
