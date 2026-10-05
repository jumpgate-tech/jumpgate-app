package main

import "testing"

// withGUIHost pretends to be a Windows host with the given build and console,
// so the GUI decisions are exercised on every OS (P10), not only in a
// console-less Windows tray build, which `go test` never is.
func withGUIHost(t *testing.T, goos string, tray, console bool) {
	t.Helper()
	oldGOOS, oldTray, oldConsole := hostGOOS, trayBuiltFn, hasConsoleFn
	t.Cleanup(func() { hostGOOS, trayBuiltFn, hasConsoleFn = oldGOOS, oldTray, oldConsole })
	hostGOOS = goos
	trayBuiltFn = func() bool { return tray }
	hasConsoleFn = func() bool { return console }
}

// D20: the GUI exe refuses CLI subcommands (it has no console to print to),
// except serve and stop, which need none.
func TestGUISubcommandRefused(t *testing.T) {
	withGUIHost(t, "windows", true, false)
	cases := map[string]bool{"hosts": true, "status": true, "help": true, "serve": false, "stop": false, "--tray": false, "-version": false}
	for arg, want := range cases {
		if got := guiSubcommandRefused([]string{"jumpgate-tray.exe", arg}); got != want {
			t.Errorf("%s: %v, want %v", arg, got, want)
		}
	}
	if guiSubcommandRefused([]string{"jumpgate-tray.exe"}) {
		t.Error("a bare double-click was refused")
	}
}

func TestGUISubcommandAllowedWithAConsoleOrOutsideAWindowsTrayBuild(t *testing.T) {
	for _, c := range []struct {
		name          string
		goos          string
		tray, console bool
	}{
		{"console exe", "windows", false, true},
		{"tray exe run from a terminal", "windows", true, true},
		{"console-less non-tray build", "windows", false, false},
		{"linux tray build", "linux", true, false},
		{"macOS tray build", "darwin", true, false},
	} {
		withGUIHost(t, c.goos, c.tray, c.console)
		if guiSubcommandRefused([]string{"jumpgate", "hosts", "list"}) {
			t.Errorf("%s: refused hosts", c.name)
		}
	}
}

// B-3: only a double-click of jumpgate-tray.exe (Windows, tray build, no
// arguments, no console) opens the desktop window on its own.
func TestGUILaunch(t *testing.T) {
	withGUIHost(t, "windows", true, false)
	if !guiLaunch([]string{"jumpgate-tray.exe"}) {
		t.Fatal("a double-click of the tray exe did not open the window")
	}
	if guiLaunch([]string{"jumpgate-tray.exe", "--bind", "x"}) {
		t.Fatal("guiLaunch with arguments")
	}
	for _, c := range []struct {
		name          string
		goos          string
		tray, console bool
	}{
		{"console", "windows", true, true},
		{"no tray tag", "windows", false, false},
		{"linux", "linux", true, false},
		{"macOS", "darwin", true, false},
	} {
		withGUIHost(t, c.goos, c.tray, c.console)
		if guiLaunch([]string{"jumpgate"}) {
			t.Errorf("%s: guiLaunch", c.name)
		}
	}
}
