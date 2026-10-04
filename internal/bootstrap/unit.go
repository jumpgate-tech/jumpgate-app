// internal/bootstrap/unit.go
package bootstrap

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

// sshdDropIn confines the tunnel user to reaching unix sockets.
const sshdDropIn = `# Managed by jumpgate. The jumpgate user may only open unix-socket channels.
Match User jumpgate
    AllowTcpForwarding no
    AllowStreamLocalForwarding local
    PermitTTY no
    X11Forwarding no
    AllowAgentForwarding no
`
