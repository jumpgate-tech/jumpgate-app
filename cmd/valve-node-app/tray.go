//go:build tray

// Tray / tiny-app mode: a native desktop window hosting the local UI, so the
// panel is used in the compact popover frame it was designed for instead of a
// full browser tab.
//
// This lives behind the `tray` build tag on purpose. It pulls in a CGo WebKit
// (macOS) / WebView2 (Windows) / WebKitGTK (Linux) dependency, which the
// default build deliberately avoids so the shipped binary stays pure-Go and
// cross-compilable. Build the desktop variant with:
//
//	go build -tags tray ./cmd/valve-node-app
//
// then run it with --tray. On macOS a menubar status item is installed too
// (see statusitem_darwin.go): the app runs as a menubar accessory (no Dock
// icon), a single click toggles the panel as a popover under the icon, clicking
// away dismisses it, and a right-click menu quits. For a double-clickable app
// run build-macos-app.sh, which
// wraps this build in a Valve.app bundle; a bundled launch has no flags, so main
// enters tray mode via inAppBundle detection.
package main

import webview "github.com/webview/webview_go"

const trayBuilt = true

// runWindow opens the tiny-app window at url and blocks until it is closed.
// Must run on the main goroutine (the platform webview owns the UI run loop),
// so main serves HTTP on a background goroutine and calls this last.
func runWindow(url string) {
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Valve")
	// Tell the SPA it's the tiny app so it drops the multi-screen topbar nav and
	// lets the panel fill the window. Init runs before page scripts on every
	// load, so it survives the token→cookie redirect the first navigation makes.
	w.Init("window.__VALVE_TRAY__ = true;")
	// Snug to the 360px panel — a tiny app, not a browser window.
	w.SetSize(380, 640, webview.HintNone)
	// Add the menubar status item (macOS) into webview's own NSApp/NSWindow, so
	// there's one Cocoa run loop. No-op off macOS. Must precede Run().
	installStatusItem(w.Window())
	w.Navigate(url)
	w.Run()
}
