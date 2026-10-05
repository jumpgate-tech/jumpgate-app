package executor

import (
	"context"

	"golang.org/x/sys/windows"
)

// IMAGE_FILE_MACHINE_* values IsWow64Process2 reports.
const (
	imageFileMachineAMD64 = 0x8664
	imageFileMachineARM64 = 0xAA64
)

// nativeArch asks Windows for the machine's own architecture. An amd64
// jumpgate on Windows on Arm runs emulated, and runtime.GOARCH would say
// amd64, which is the same lie Rosetta tells on a Mac.
func nativeArch(context.Context, *local) string {
	// IsWow64Process2 arrived in Windows 10 1709; x/sys panics calling a
	// procedure that is not there, so check first.
	if windows.NewLazySystemDLL("kernel32.dll").NewProc("IsWow64Process2").Find() != nil {
		return ""
	}
	var proc, native uint16
	if err := windows.IsWow64Process2(windows.CurrentProcess(), &proc, &native); err != nil {
		return ""
	}
	switch native {
	case imageFileMachineAMD64:
		return "amd64"
	case imageFileMachineARM64:
		return "arm64"
	}
	return ""
}
