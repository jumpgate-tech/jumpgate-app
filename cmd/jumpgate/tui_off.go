//go:build notui

package main

import (
	"context"
	"io"
)

// tuiBuilt is false in agent builds (-tags notui, spec D23): the agent runs
// as root on every box and never shows a terminal UI.
const tuiBuilt = false

func runTUI(ctx context.Context, in io.Reader, out io.Writer) int { return runPlainHome(ctx, in, out) }
