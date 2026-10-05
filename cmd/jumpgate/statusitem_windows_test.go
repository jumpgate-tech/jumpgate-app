//go:build tray && windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

// The icon is added, its tooltip follows health, and it is removed. A CI
// session may have no notification area; then the add is refused and the
// test says so instead of failing.
func TestTrayIconLifecycle(t *testing.T) {
	installStatusItem(nil)
	trayIcon.mu.Lock()
	hwnd, added := trayIcon.hwnd, trayIcon.added
	trayIcon.mu.Unlock()
	if hwnd == 0 {
		t.Fatal("the tray window was not created")
	}
	if !added {
		removeStatusItem()
		t.Skip("Shell_NotifyIconW refused: no notification area in this session")
	}
	setHealth(healthOK)
	trayIcon.mu.Lock()
	tip := windows.UTF16ToString(trayIcon.data.SzTip[:])
	trayIcon.mu.Unlock()
	if tip != healthTooltip(healthOK) {
		t.Fatalf("tooltip %q, want %q", tip, healthTooltip(healthOK))
	}
	removeStatusItem()
	trayIcon.mu.Lock()
	defer trayIcon.mu.Unlock()
	if trayIcon.added || trayIcon.hwnd != 0 {
		t.Fatal("the icon or its window survived removeStatusItem")
	}
}

// A second window (an attach after a quit, or a reinstall) must get a tray
// window again: the class is registered once, not refused the second time.
func TestTrayWindowCanBeCreatedTwice(t *testing.T) {
	for i := 0; i < 2; i++ {
		installStatusItem(nil)
		trayIcon.mu.Lock()
		hwnd := trayIcon.hwnd
		trayIcon.mu.Unlock()
		removeStatusItem()
		if hwnd == 0 {
			t.Fatalf("install %d created no tray window", i+1)
		}
	}
}
