//go:build windows

package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleWindow      = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
)

// hasConsole reports whether this process has a console window. It is false
// for jumpgate-tray.exe (GUI subsystem) and for a server started detached.
func hasConsole() bool {
	h, _, _ := procGetConsoleWindow.Call()
	return h != 0
}

// launchedStandalone reports whether this process owns its console window:
// double-clicked in Explorer, the console holds only this process and closes
// the moment it exits, so an error would flash and vanish.
func launchedStandalone() bool {
	if os.Getenv("JUMPGATE_LAUNCHER") == "1" {
		return true
	}
	var pids [2]uint32
	n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return n == 1
}
