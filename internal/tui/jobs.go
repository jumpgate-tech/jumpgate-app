package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// jobsScreen is the jobs inbox's place (spec D27): durable jobs are
// sub-project 2, and the TUI is stateless, so the inbox arrives as an
// additive screen.
type jobsScreen struct{}

func (jobsScreen) update(*App, tea.Msg) tea.Cmd { return nil }
func (jobsScreen) keys() []key.Binding          { return nil }
func (jobsScreen) capturing() bool              { return false }

func (jobsScreen) view(a *App, w, h int) string {
	return a.th.Title.Render(" Jobs inbox") + "\n\n" +
		" Durable jobs arrive with sub-project 2: long operations that outlive\n" +
		" this window, and the decisions they wait for. Nothing runs as a job yet.\n\n" +
		a.th.Dim.Render(" The "+a.gl.Flag+" count in the status bar will show jobs that need you.")
}
