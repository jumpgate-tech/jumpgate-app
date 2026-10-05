//go:build !windows

package main

// The GUI subsystem is Windows-only: elsewhere stderr reaches the terminal or
// the launcher's log, so there is nothing to redirect and no dialog to show.
func setupGUILogging()       {}
func showErrorDialog(string) {}

// webview2Installed is a Windows question; other platforms bring their own
// web engine with the build.
func webview2Installed() bool { return true }
