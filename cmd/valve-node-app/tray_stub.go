//go:build !tray

// The default build has no desktop window: --tray prints how to get one. This
// stub keeps main.go compiling without the CGo webview dependency, so the
// shipped binary stays pure-Go and cross-compilable.
package main

import "context"

const trayBuilt = false

// runWindow is never called in this build — main guards on trayBuilt first — but
// it must exist for main.go to compile.
func runWindow(context.Context, string) {}
