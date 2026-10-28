package sysunit

import (
	"os"
	"strings"
	"testing"
)

func TestUnitCarriesTheSpecsHardening(t *testing.T) {
	u := Unit("/usr/bin/jumpgate")
	lines := map[string]bool{}
	for _, l := range strings.Split(u, "\n") {
		lines[l] = true
		if k, v, ok := strings.Cut(l, "="); ok && strings.Contains(v, "#") {
			t.Errorf("%s has a comment after its value: systemd would read it as part of the value", k)
		}
	}
	for _, want := range []string{
		"[Unit]", "Description=jumpgate controller", "After=network-online.target", "Wants=network-online.target",
		"[Service]",
		"ExecStart=/usr/bin/jumpgate serve --no-open --system", "User=jumpgate-ctl", "Group=jumpgate-ctl",
		"SupplementaryGroups=jumpgate-admin", "Environment=HOME=/var/lib/jumpgate-ctl",
		"StateDirectory=jumpgate-ctl", "StateDirectoryMode=0700", "RuntimeDirectory=jumpgate-ctl", "RuntimeDirectoryMode=0750",
		"UMask=0077", "LimitCORE=0", "NoNewPrivileges=yes", "ProtectSystem=strict", "ProtectHome=yes", "PrivateTmp=yes",
		"CapabilityBoundingSet=", "RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK", "ProtectKernelTunables=yes", "ProtectKernelModules=yes", "ProtectKernelLogs=yes", "ProtectControlGroups=yes", "RestrictSUIDSGID=yes", "LockPersonality=yes", "MemoryDenyWriteExecute=yes", "SystemCallFilter=@system-service", "PrivateDevices=yes",
		"Restart=on-failure",
		"[Install]", "WantedBy=multi-user.target",
	} {
		if !lines[want] {
			t.Errorf("unit has no line %q:\n%s", want, u)
		}
	}
	if !strings.HasSuffix(u, "\n") {
		t.Error("unit does not end in a newline")
	}
}

func TestPackagingFilesMatchSysunit(t *testing.T) {
	unit, err := os.ReadFile("../../packaging/linux/server/jumpgate.service")
	if err != nil {
		t.Fatal(err)
	}
	if string(unit) != Unit("/usr/bin/jumpgate") {
		t.Errorf("packaging/linux/server/jumpgate.service is not Unit(\"/usr/bin/jumpgate\"); regenerate it")
	}
	sysusers, err := os.ReadFile("../../packaging/linux/server/jumpgate.sysusers")
	if err != nil {
		t.Fatal(err)
	}
	if string(sysusers) != Sysusers {
		t.Errorf("packaging/linux/server/jumpgate.sysusers is not Sysusers; regenerate it")
	}
}
