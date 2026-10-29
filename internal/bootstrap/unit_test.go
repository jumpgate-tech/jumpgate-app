package bootstrap

import (
	"strings"
	"testing"
)

// TestAgentUnitHardening pins the gate unit's hardening, and the controller
// settings unit.go documents as left out because they would break plans.
func TestAgentUnitHardening(t *testing.T) {
	for _, want := range []string{
		"NoNewPrivileges=yes", "PrivateTmp=yes", "ProtectHome=read-only",
		"ProtectKernelTunables=yes", "ProtectKernelLogs=yes", "ProtectControlGroups=yes",
		"ProtectHostname=yes", "RestrictSUIDSGID=yes", "RestrictRealtime=yes",
		"RestrictNamespaces=yes", "LockPersonality=yes", "LimitCORE=0",
		"RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK",
		"SystemCallArchitectures=native", "SystemCallFilter=~@clock ",
		"CapabilityBoundingSet=~CAP_SYS_BOOT ",
	} {
		if !strings.Contains(agentUnit, "\n"+want) {
			t.Errorf("gate unit lacks %q", want)
		}
	}
	for _, out := range []string{
		"ProtectSystem=", "ProtectKernelModules=", "MemoryDenyWriteExecute=",
		"PrivateDevices=", "ProtectClock=", "ProtectProc=", "UMask=",
		"SystemCallFilter=@", "CapabilityBoundingSet=\n",
	} {
		if strings.Contains(agentUnit, "\n"+out) {
			t.Errorf("gate unit sets %q, which breaks plans (see unit.go)", out)
		}
	}
	for _, c := range []string{"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_SETUID", "CAP_NET_ADMIN", "CAP_SYS_ADMIN", "CAP_SYS_MODULE"} {
		if strings.Contains(agentUnit, c) {
			t.Errorf("gate unit drops %s, which plans need", c)
		}
	}
}
