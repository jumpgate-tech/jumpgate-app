package tui

import (
	"context"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

func loadGateways(a *App) tea.Cmd {
	return a.do("gateways", func(ctx context.Context) tea.Msg {
		gws, err := a.be.Gateways(ctx)
		return gatewaysMsg{gws: gws, err: err}
	})
}

func reach(a *App, ok bool) string {
	if ok {
		return a.th.OK.Render("reachable")
	}
	return a.th.Bad.Render("not reachable")
}

// viewEndpoints is the node's RPC endpoints and, read-only, the gateways
// placed on this box (spec D28). URLs are masked: an endpoint can carry an
// API key.
func (h *hostScreen) viewEndpoints(a *App, w, hgt int) string {
	if !h.epsHas {
		if h.epsErr != nil {
			return "\n " + a.th.Bad.Render("endpoints unavailable: ") + a.errText(h.epsErr)
		}
		return "\n " + a.th.Dim.Render("probing the endpoints"+a.gl.Ellipsis)
	}
	e := h.eps
	chain := a.th.Warn.Render("chain id does not match")
	if e.ChainIDMatches {
		chain = "chain id matches"
	}
	// A URL longer than its column puts the verdict on the next line, so a
	// long one never pushes it off the screen.
	endpoint := func(label, u, verdict string) []string {
		u = maskURL(sanitizeLine(u))
		if lipgloss.Width(u) > 26 {
			return []string{" " + label + "  " + u, "         " + verdict}
		}
		return []string{" " + label + "  " + pad(u, 26) + " " + verdict}
	}
	lines := []string{" " + a.th.Title.Render("RPC ENDPOINTS")}
	lines = append(lines, endpoint("exec  ", e.ExecHTTP, reach(a, e.ExecReachable)+" "+a.gl.Sep+" "+chain)...)
	lines = append(lines, endpoint("beacon", e.BeaconHTTP, reach(a, e.BeaconReachable))...)
	if e.Access == "ssh" && e.TunnelHint != "" {
		lines = append(lines, " access  over SSH; open a tunnel with", "         "+sanitizeLine(e.TunnelHint))
	} else {
		lines = append(lines, " access  local to this machine")
	}
	lines = append(lines, "", " "+a.th.Title.Render("GATEWAYS ON THIS BOX")+a.th.Dim.Render(" (read-only; change them in the web app)"))
	n := 0
	for _, g := range h.gws {
		if g.Placement.TargetID != h.id {
			continue
		}
		n++
		lines = append(lines, gatewayLines(a, g)...)
	}
	if n == 0 {
		lines = append(lines, " "+a.th.Dim.Render("none"))
	}
	return "\n" + strings.Join(window(lines, &h.scroll, hgt-1, w, a.gl.Ellipsis), "\n")
}

// gatewayLines is one gateway: identity, state and base URL, then one line
// per network, then its warnings. Every string is cleaned and URLs masked;
// the caller cuts lines to its width.
func gatewayLines(a *App, g api.GatewaySummary) []string {
	plain := sanitizeLine(g.Status.State)
	if plain == "" {
		plain = "unknown"
	}
	style := a.th.Warn
	switch plain {
	case "running":
		style = a.th.OK
	case "unknown":
		style = a.th.Dim
	}
	label := g.Label
	if label == "" {
		label = g.ID
	}
	// Pad before styling: the padding must count cells, not colour codes.
	lines := []string{" " + pad(sanitizeLine(label), 14) + " on " + pad(sanitizeLine(g.Placement.TargetID), 12) + " " +
		style.Render(pad(plain, 9)) + " " + maskURL(sanitizeLine(g.BaseURL))}
	for _, n := range g.Networks {
		url := n.URL
		if url == "" {
			url = n.LocalURL
		}
		lines = append(lines, "   "+sanitizeLine(n.Name)+" ("+strconv.Itoa(n.ChainID)+")  "+maskURL(sanitizeLine(url)))
	}
	for _, warn := range g.Warnings {
		lines = append(lines, "   "+a.th.Warn.Render(maskText(sanitizeLine(warn))))
	}
	return lines
}
