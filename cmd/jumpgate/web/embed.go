// Package web embeds the built UI so every entry point (the tray app, jumpgate
// serve) serves the same files.
package web

import "embed"

// FS holds dist/; use fs.Sub(FS, "dist").
//
//go:embed all:dist
var FS embed.FS
