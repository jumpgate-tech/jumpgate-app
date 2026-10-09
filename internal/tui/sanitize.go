package tui

import "github.com/valve-tech/jumpgate/internal/termsafe"

// sanitize makes text from outside the TUI safe to draw. The rules live in
// termsafe, which the CLI shares; this is the one choke point in the TUI.
func sanitize(s string) string { return termsafe.Text(s) }

// sanitizeLine is sanitize for a one-line field (the status bar, a flash, the
// skew banner, a box name): line breaks and tabs become spaces, so hostile
// text cannot add rows to the frame.
func sanitizeLine(s string) string { return termsafe.Line(s) }
