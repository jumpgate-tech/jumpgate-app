package main

import (
	"log"
	"strings"
)

// fatalf reports an error that stops the app. It is log.Fatalf, except in the
// Windows GUI build, which has no console: setupGUILogging replaces it with
// one that also writes app.log and shows a dialog (B-3).
var fatalf = log.Fatalf

// requestQuit stops the app from a tray menu. runApp sets it to the cancel
// of its context; closing the window or Ctrl-C end the app the same way.
var requestQuit = func() {}

// trayBuiltFn and hasConsoleFn are seams over trayBuilt and hasConsole, so
// the GUI decisions below are tested on every OS (a console-less Windows
// tray build never runs under `go test`).
var (
	trayBuiltFn  = func() bool { return trayBuilt }
	hasConsoleFn = hasConsole
)

// isGUIExe reports whether this is jumpgate-tray.exe started without a
// console: a Windows tray build under the GUI subsystem.
func isGUIExe() bool { return hostGOOS == "windows" && trayBuiltFn() && !hasConsoleFn() }

// guiSubcommandRefused reports a CLI subcommand given to the Windows GUI exe,
// which has no console to print to or read from (spec D20). serve and stop
// need none.
func guiSubcommandRefused(args []string) bool {
	if len(args) < 2 || strings.HasPrefix(args[1], "-") || !isGUIExe() {
		return false
	}
	return args[1] != "serve" && args[1] != "stop"
}

// guiLaunch reports a double-click of jumpgate-tray.exe: no arguments and no
// console. It opens the desktop window instead of a browser tab and an
// invisible server (B-3). Elsewhere the window is chosen by --tray or the
// macOS bundle.
func guiLaunch(args []string) bool { return len(args) == 1 && isGUIExe() }
