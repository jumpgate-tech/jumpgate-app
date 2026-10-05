//go:build tray && !darwin && !windows

package main

import "unsafe"

// installStatusItem is a no-op on Linux: macOS has the menubar item and
// Windows the notification-area icon. Linux tray builds still get the webview
// window, just without a status-bar entry.
func installStatusItem(unsafe.Pointer) {}

// setHealth is a no-op on Linux (no status dot to paint).
func setHealth(healthKind) {}

func removeStatusItem() {}
