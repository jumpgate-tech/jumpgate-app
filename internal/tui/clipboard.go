package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// The clipboard: copy a line of public text (the invite command). OSC 52
// asks the terminal to set its clipboard, which works over ssh; a local
// pbcopy, wl-copy or xclip, found by absolute path, is fed the text as well
// when there. Never call this with a secret: OSC 52 goes through every
// terminal layer.

// clipboardArgv is the local copy program for this platform, by absolute
// path, or nil.
func clipboardArgv(getenv func(string) string, goos string) []string {
	var tries [][]string
	switch goos {
	case "darwin":
		tries = [][]string{{"pbcopy"}}
	case "windows":
	default:
		if getenv("WAYLAND_DISPLAY") != "" {
			tries = append(tries, []string{"wl-copy"})
		}
		if getenv("DISPLAY") != "" {
			tries = append(tries, []string{"xclip", "-selection", "clipboard"})
		}
	}
	for _, t := range tries {
		if bin := onPath(t[0], getenv, goos); bin != "" {
			return append([]string{bin}, t[1:]...)
		}
	}
	return nil
}

// copyText sets the clipboard both ways; the flash says what it did.
func (a *App) copyText(text, what string) tea.Cmd {
	osc := tea.SetClipboard(text)
	argv := clipboardArgv(a.o.Getenv, a.o.GOOS)
	if argv == nil {
		return tea.Batch(osc, func() tea.Msg { return flashMsg(what + " copied (terminal clipboard, OSC 52)") })
	}
	cmd := a.o.Command(argv[0], argv[1:]...)
	return tea.Batch(osc, func() tea.Msg { return flashMsg(what + " " + runCopy(cmd, argv[0], text)) })
}

// runCopy feeds text to the copy program and says how it went.
func runCopy(cmd *exec.Cmd, bin, text string) string {
	cmd.Stdin = strings.NewReader(text)
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return "copied (OSC 52 and " + bin + ")"
	case errors.As(err, &ee):
		return fmt.Sprintf("copied (OSC 52); %s exited %d", bin, ee.ExitCode())
	}
	return "copied (OSC 52); " + bin + " did not run"
}
