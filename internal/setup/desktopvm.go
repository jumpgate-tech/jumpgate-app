package setup

import (
	"context"
	"strings"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// desktopVMKind classifies a Linux target that is really a desktop VM: WSL2
// (/proc/version mentions "microsoft") or a Lima VM (hostname lima-*, or an
// /mnt/lima-cidata directory). It returns "" for anything else. Pure, so detection is
// testable with injected file contents.
func desktopVMKind(procVersion, hostname string, limaDir bool) string {
	if strings.Contains(strings.ToLower(procVersion), "microsoft") {
		return "WSL2"
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(hostname)), "lima-") || limaDir {
		return "Lima"
	}
	return ""
}

// desktopVMWarning is the operator-facing text for a detected desktop VM.
func desktopVMWarning(kind string) string {
	return "warning: this target looks like a " + kind + " VM on a desktop. A real node can run here, but expect trouble: " +
		"the host may sleep, reboot or stop the VM when idle (the node falls behind or stops); " +
		"the VM sits behind NAT, so inbound peers need port forwards; " +
		"and chain data is 1.2 TB (testnet) to 3.9 TB (PulseChain) before consensus data, so the VM disk must live on a large, fast drive, never a host-shared folder. " +
		"See docs/run-a-node-on-your-desktop.md."
}

// warnIfDesktopVM probes the target and emits a one-line warning when it is a
// desktop VM. It never fails: a probe that cannot be answered means no warning.
func warnIfDesktopVM(ctx context.Context, e executor.Executor, st *State) {
	out := func(cmd string) string {
		res, err := e.Run(ctx, cmd, nil)
		if err != nil || res.ExitCode != 0 {
			return ""
		}
		return res.Stdout
	}
	kind := desktopVMKind(
		out("cat /proc/version"),
		out("hostname"),
		strings.TrimSpace(out("[ -d /mnt/lima-cidata ] && echo lima")) == "lima",
	)
	if kind != "" {
		_ = emit(ctx, st, Event{StepID: "preflight", Line: desktopVMWarning(kind)})
	}
}
