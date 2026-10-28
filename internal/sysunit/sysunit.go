// Package sysunit is the jumpgate service's systemd unit and sysusers entry
// (spec D5, D6): one source for the package's files and for
// `jumpgate setup install-service`.
package sysunit

// Where the unit and sysusers files live: the package owns /usr/lib, an
// administrator's install (install.sh, setup install-service) owns /etc.
const (
	PackageUnitPath     = "/usr/lib/systemd/system/jumpgate.service"
	AdminUnitPath       = "/etc/systemd/system/jumpgate.service"
	PackageSysusersPath = "/usr/lib/sysusers.d/jumpgate.conf"
	AdminSysusersPath   = "/etc/sysusers.d/jumpgate.conf"
	SealDropInPath      = "/etc/systemd/system/jumpgate.service.d/50-vault-seal.conf"
	SealCredPath        = "/etc/jumpgate-ctl/vault-key.cred"
)

// Sysusers creates the operator group jumpgate-admin and the service user
// and group jumpgate-ctl, whose home is the service's state directory.
const Sysusers = "g jumpgate-admin -\nu jumpgate-ctl - \"jumpgate controller\" /var/lib/jumpgate-ctl /usr/sbin/nologin\n"

// Unit is jumpgate.service running bin. systemd allows no comment after a
// value, so every setting stands alone on its line.
//
// The service is an unprivileged Go server that spawns ssh: it needs no
// capability, device, kernel knob or address family beyond these, and
// never maps writable code.
//
// SupplementaryGroups lets the service, not root, give its socket and
// runtime directory to jumpgate-admin: only a member may chgrp to a group.
func Unit(bin string) string {
	return "[Unit]\n" +
		"Description=jumpgate controller\n" +
		"After=network-online.target\n" +
		"Wants=network-online.target\n" +
		"\n" +
		"[Service]\n" +
		"ExecStart=" + bin + " serve --no-open --system\n" +
		"User=jumpgate-ctl\n" +
		"Group=jumpgate-ctl\n" +
		"SupplementaryGroups=jumpgate-admin\n" +
		"Environment=HOME=/var/lib/jumpgate-ctl\n" +
		"StateDirectory=jumpgate-ctl\n" +
		"StateDirectoryMode=0700\n" +
		"RuntimeDirectory=jumpgate-ctl\n" +
		"RuntimeDirectoryMode=0750\n" +
		"UMask=0077\n" +
		"LimitCORE=0\n" +
		"NoNewPrivileges=yes\n" +
		"ProtectSystem=strict\n" +
		"ProtectHome=yes\n" +
		"PrivateTmp=yes\n" +
		"CapabilityBoundingSet=\n" +
		"RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK\n" +
		"ProtectKernelTunables=yes\n" +
		"ProtectKernelModules=yes\n" +
		"ProtectKernelLogs=yes\n" +
		"ProtectControlGroups=yes\n" +
		"RestrictSUIDSGID=yes\n" +
		"LockPersonality=yes\n" +
		"MemoryDenyWriteExecute=yes\n" +
		"SystemCallFilter=@system-service\n" +
		"PrivateDevices=yes\n" +
		"Restart=on-failure\n" +
		"\n" +
		"[Install]\n" +
		"WantedBy=multi-user.target\n"
}
