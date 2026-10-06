package tui

import (
	"context"
	"errors"
	"io"

	tea "charm.land/bubbletea/v2"
)

// Run shows the TUI on in/out until the person quits or ctx ends. Bubble Tea
// detects the colour profile from out and the environment, which is how
// NO_COLOR and TERM=dumb take every colour away.
func Run(ctx context.Context, in io.Reader, out io.Writer, o Options) error {
	a := New(o)
	defer a.cancel()
	_, err := tea.NewProgram(a, tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(out)).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return nil
	}
	return err
}
