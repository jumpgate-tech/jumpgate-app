//go:build !windows

package main

import "os"

// launchedStandalone reports whether this process owns its terminal window
// and should wait before closing it on an error. Off Windows only the
// launchers know, and they say so with JUMPGATE_LAUNCHER=1.
func launchedStandalone() bool { return os.Getenv("JUMPGATE_LAUNCHER") == "1" }

// hasConsole is always true off Windows: there is no GUI subsystem there.
func hasConsole() bool { return true }
