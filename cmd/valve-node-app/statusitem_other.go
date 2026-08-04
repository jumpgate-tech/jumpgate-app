//go:build tray && !darwin

package main

import "unsafe"

// installStatusItem is a no-op off macOS: the menubar status item is a Cocoa
// feature. Linux/Windows tray builds still get the webview window, just without
// a status-bar entry.
func installStatusItem(unsafe.Pointer) {}

// setHealth is a no-op off macOS (no status dot to paint).
func setHealth(healthKind) {}
