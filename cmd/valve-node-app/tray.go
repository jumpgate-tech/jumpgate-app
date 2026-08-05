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
// wraps this build in a Jumpgate.app bundle; a bundled launch has no flags, so main
// enters tray mode via inAppBundle detection.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	webview "github.com/webview/webview_go"
)

const trayBuilt = true

// runWindow opens the tiny-app window at url and blocks until it is closed.
// Must run on the main goroutine (the platform webview owns the UI run loop),
// so main serves HTTP on a background goroutine and calls this last.
//
// When ctx is cancelled (Ctrl-C / SIGTERM), Terminate breaks the run loop so
// runWindow returns and the process exits — otherwise a terminal signal would
// stop the HTTP server but leave the webview (and its menubar icon) running.
// Terminate is documented safe to call from a background goroutine.
func runWindow(ctx context.Context, url string) {
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Jumpgate") // product #3's name; the tray surface is Jumpgate
	// Tell the SPA it's the tiny app so it drops the multi-screen topbar nav and
	// lets the panel fill the window. Init runs before page scripts on every
	// load, so it survives the token→cookie redirect the first navigation makes.
	w.Init("window.__VALVE_TRAY__ = true;")
	// Snug to the 360px panel — a tiny app, not a browser window.
	w.SetSize(380, 640, webview.HintNone)
	// Add the menubar status item (macOS) into webview's own NSApp/NSWindow, so
	// there's one Cocoa run loop. No-op off macOS. Must precede Run().
	installStatusItem(w.Window())

	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
			w.Terminate()
		case <-stopped:
		}
	}()

	go pollHealth(ctx, w, url)

	w.Navigate(url)
	w.Run()
}

// pollHealth drives the menubar status dot from live gateway health. It polls
// the local /api/gateways (same server this process runs) on a timer and, only
// when the folded health changes, marshals a dot repaint onto the main thread
// via webview.Dispatch — the status item is Cocoa UI and must be touched there.
// browserURL is the http://host/?token=… the window opened with; its host and
// token authorize the poll.
func pollHealth(ctx context.Context, w webview.WebView, browserURL string) {
	u, err := url.Parse(browserURL)
	if err != nil {
		return
	}
	apiURL := u.Scheme + "://" + u.Host + "/api/gateways"
	token := u.Query().Get("token")
	client := &http.Client{Timeout: 3 * time.Second}

	last := healthKind(-1)
	check := func() {
		k := fetchHealth(ctx, client, apiURL, token)
		if k == last {
			return
		}
		last = k
		w.Dispatch(func() { setHealth(k) })
	}

	check()
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			check()
		}
	}
}

// fetchHealth reads /api/gateways once and folds it to a dot state. Any failure
// to reach or parse our own server reads as "off" (grey) — the honest signal
// when the control plane isn't answering.
func fetchHealth(ctx context.Context, client *http.Client, apiURL, token string) healthKind {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return healthOff
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return healthOff
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return healthOff
	}
	var r gwListResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return healthOff
	}
	return overallHealth(r.Gateways)
}
