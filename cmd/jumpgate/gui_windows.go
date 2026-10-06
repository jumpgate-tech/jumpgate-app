//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// setupGUILogging sends log output to ~/.jumpgate/run/app.log and makes
// fatalf show a dialog too: under the GUI subsystem stdout and stderr go
// nowhere, so startup failures used to be silent. If the log cannot be
// opened, the dialog still shows the error and says why there is no log.
func setupGUILogging() {
	details := ""
	path, err := openGUILog()
	if err != nil {
		details = "\n\n(no log: " + err.Error() + ")"
	} else {
		details = "\n\nDetails are in " + path
	}
	fatalf = func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		log.Print(msg)
		// Stop serving first: the dialog waits for the user, and a server
		// left running behind it would hold the lock and the port.
		requestQuit()
		showErrorDialog(msg + details)
		os.Exit(1)
	}
}

// openGUILog points the standard logger, os.Stdout, os.Stderr and crash
// output at app.log, owner-only from the moment it exists (it can name hosts
// and addresses), and returns its path. Under the GUI subsystem the standard
// handles are invalid, so the warnings runApp prints would otherwise vanish.
func openGUILog() (string, error) {
	dir, err := daemon.RunDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "app.log")
	f, err := fsperm.OpenAppendPrivate(path)
	if err != nil {
		return "", err
	}
	log.SetOutput(f)
	os.Stdout, os.Stderr = f, f
	_ = debug.SetCrashOutput(f, debug.CrashOptions{})
	return path, nil
}

// showErrorDialog shows msg in a modal error box.
func showErrorDialog(msg string) {
	text, _ := windows.UTF16PtrFromString(msg)
	title, _ := windows.UTF16PtrFromString("Jumpgate")
	_, _ = windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
}

// webview2Installed reports whether the Microsoft Edge WebView2 Runtime is
// present (machine-wide or per user), which the desktop window needs. The
// keys and the "pv" value are the ones Microsoft documents for detection.
func webview2Installed() bool {
	const client = `Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
	for _, k := range []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\` + client},
		{registry.LOCAL_MACHINE, `SOFTWARE\` + client},
		{registry.CURRENT_USER, `SOFTWARE\` + client},
	} {
		key, err := registry.OpenKey(k.root, k.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		v, _, err := key.GetStringValue("pv")
		key.Close()
		if err == nil && v != "" && v != "0.0.0.0" {
			return true
		}
	}
	return false
}
