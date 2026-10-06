package tui

import (
	"fmt"
	"strings"

	"github.com/valve-tech/jumpgate/internal/api"
)

func checkWord(a *App, c api.CheckItem) string {
	switch c.Status {
	case "pass":
		return a.th.OK.Render("ok  ")
	case "warn":
		return a.th.Warn.Render("warn")
	case "fail":
		return a.th.Bad.Render("FAIL")
	}
	return a.th.Dim.Render("?   ")
}

// viewSecurity is the firewall checklist: every check with its reason, and
// for anything not passing what was found and the command that fixes it.
func (h *hostScreen) viewSecurity(a *App, w, hgt int) string {
	if !h.fwHas {
		if h.fwErr != nil {
			return "\n " + a.th.Bad.Render("checklist unavailable: ") + a.errText(h.fwErr)
		}
		return "\n " + a.th.Dim.Render("running the checklist"+a.gl.Ellipsis)
	}
	var lines []string
	if h.fwErr != nil {
		lines = append(lines, " "+a.th.Warn.Render("last check failed, showing the previous result: "+a.errText(h.fwErr)))
	}
	toFix := 0
	for _, c := range h.fw {
		lines = append(lines, " "+checkWord(a, c)+" "+sanitizeLine(c.Title))
		if c.Status == "pass" {
			continue
		}
		toFix++
		if c.Why != "" {
			lines = append(lines, "      "+a.th.Dim.Render(maskText(sanitizeLine(c.Why))))
		}
		if c.Detail != "" {
			lines = append(lines, "      found: "+maskText(sanitizeLine(c.Detail)))
		}
		for _, fix := range strings.Split(sanitize(c.Fix), "\n") {
			if fix = strings.TrimSpace(sanitizeLine(fix)); fix != "" {
				lines = append(lines, "      fix:   "+a.th.Key.Render(maskText(fix)))
			}
		}
	}
	head := fmt.Sprintf(" %s  %d checks, %d to fix %s ↑↓ scroll", a.th.Title.Render("FIREWALL"), len(h.fw), toFix, a.gl.Sep)
	// The blank line and the head take two rows.
	return "\n" + truncate(head, w, a.gl.Ellipsis) + "\n" + strings.Join(window(lines, &h.scroll, hgt-2, w, a.gl.Ellipsis), "\n")
}
