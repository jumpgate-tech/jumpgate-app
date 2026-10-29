package tui

import (
	"context"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

// The invite: the one line a person runs as root on a new box to install
// its gate and pull-pair it to this controller, shown unboxed under the
// panes. i shows or hides it, c copies it. Everything in it is public: the
// release, the controller's address and the tunnel key's public half.

var (
	inviteRelease = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	inviteAddress = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
)

// inviteCommand is `jumpgate hosts invite`'s install command on one line,
// as the CLI prints it, or why there is none.
func inviteCommand(version string, info api.PairingInfo) (cmd, fail string) {
	key := strings.TrimSpace(info.TransportKey)
	switch {
	case !inviteRelease.MatchString(version):
		return "", "this controller is a development build (" + sanitizeLine(version) + "); an invite needs a released version"
	case info.Scope == "self":
		return "", "this controller is self-only (self_only)"
	case !inviteAddress.MatchString(info.Controller):
		return "", "the server gave no valid controller address"
	case key == "" || strings.ContainsAny(key, "\r\n") || sanitizeLine(key) != key:
		return "", "the server gave no usable transport key"
	}
	return "curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent --version " + version +
		" --controller " + info.Controller + " --transport-key '" + strings.ReplaceAll(key, "'", `'"'"'`) + "'", ""
}

type inviteMsg struct {
	info api.PairingInfo
	err  error
}

// toggleInvite shows the invite (reading the pairing info) or hides it.
func (a *App) toggleInvite() tea.Cmd {
	if a.inviteOn {
		a.inviteOn, a.invite, a.inviteFail = false, "", ""
		a.fitFocus()
		return nil
	}
	a.inviteOn, a.invite, a.inviteFail = true, "", "reading the pairing info"+a.gl.Ellipsis
	a.fitFocus()
	return a.do("invite", func(ctx context.Context, be Backend) tea.Msg {
		info, err := be.PairingInfo(ctx, "")
		return inviteMsg{info: info, err: err}
	})
}

func (a *App) gotInvite(m inviteMsg) {
	if !a.inviteOn {
		return
	}
	if m.err != nil {
		a.invite, a.inviteFail = "", "pairing info: "+a.errText(m.err)
		return
	}
	server, _, _ := a.be.Skew()
	a.invite, a.inviteFail = inviteCommand(server, m.info)
}

// inviteLine is the invite's one line, w cells: the buttons first, so a
// narrow window still shows them.
func (a *App) inviteLine(w int) string {
	if a.invite == "" {
		return truncate(" [i] hide  invite: "+a.th.Bad.Render("FAIL")+" "+a.inviteFail, w, a.gl.Ellipsis)
	}
	return truncate(" [c] copy  [i] hide  on the box, as root: "+a.invite, w, a.gl.Ellipsis)
}

// copyInvite puts the invite on the clipboard.
func (a *App) copyInvite() tea.Cmd {
	if a.invite == "" {
		a.flash = "no invite to copy"
		return nil
	}
	a.log.add(logEvent{At: a.now(), Sev: logInfo, Text: "invite copied; run it as root on the new box"})
	return a.copyText(a.invite, "invite")
}
