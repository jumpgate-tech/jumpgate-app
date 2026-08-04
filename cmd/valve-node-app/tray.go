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
// then run it with --tray. A menubar status-item (systray) that toggles this
// window is the next step; it needs a real desktop session to iterate on, which
// is why it is not wired here yet.
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
	// The panel card is 360px wide; a little chrome around it reads as a compact
	// app window rather than a browser.
	w.SetSize(420, 720, webview.HintNone)
	w.Navigate(url)
	w.Run()
}
