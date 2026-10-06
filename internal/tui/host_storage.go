package tui

import (
	"fmt"
	"math"

	"github.com/valve-tech/jumpgate/internal/api"
)

var fitWords = map[api.FitVerdict]string{
	api.FitOK:      "ok: room for the estimate with 10% headroom",
	api.FitTight:   "tight: room for the estimate, not the 10% headroom",
	api.FitShort:   "short: not enough room for the estimated size",
	api.FitUnknown: "unknown: no size estimate for this chain",
}

// viewStorage is the disk: what each client holds, what is free, the
// expected sizes (always labelled estimates), the fit verdict and the trend.
func (h *hostScreen) viewStorage(a *App, w int) string {
	u := a.prefs.Units
	switch {
	case h.measuring && !h.diskHas:
		return "\n " + a.th.Dim.Render("measuring"+a.gl.Ellipsis)
	case !h.diskHas && h.diskErr != nil:
		return "\n " + a.th.Bad.Render("disk unavailable: ") + a.errText(h.diskErr)
	case !h.diskHas:
		return "\n " + a.th.Dim.Render("reading the disk"+a.gl.Ellipsis)
	}
	d := h.disk
	state := "measured " + formatAge(a.now().Sub(d.At)) + " ago " + a.gl.Sep + " m measure now"
	if h.measuring {
		state = "measuring" + a.gl.Ellipsis
	}
	total := d.UsedBytes() + d.FreeBytes
	expected := d.ExpectedExecBytes + d.ExpectedBeaconBytes
	pct := 0.0
	if total > 0 {
		pct = math.Round(float64(d.UsedBytes()) / float64(total) * 100)
	}
	out := fmt.Sprintf("\n %s  %s\n", a.th.Title.Render("DISK"), a.th.Dim.Render(state))
	out += fmt.Sprintf(" exec      %s\n beacon    %s\n free      %s\n", formatBytes(d.ExecBytes, u), formatBytes(d.BeaconBytes, u), formatBytes(d.FreeBytes, u))
	out += fmt.Sprintf(" [%s] %.0f%% used by the clients\n", bar(d.UsedBytes(), expected, total, max(min(40, w-30), 0), a.gl), pct)
	if a.prefs.ShowEstimates {
		out += fmt.Sprintf(" expected  %s exec + %s beacon (%s)\n", formatBytes(d.ExpectedExecBytes, u), formatBytes(d.ExpectedBeaconBytes, u), sanitizeLine(d.ExpectedLabel))
	}
	fitStyle := a.th.OK
	if d.Fit == api.FitTight || d.Fit == api.FitUnknown {
		fitStyle = a.th.Warn
	} else if d.Fit == api.FitShort {
		fitStyle = a.th.Bad
	}
	fw, ok := fitWords[d.Fit]
	if !ok {
		fw = sanitizeLine(string(d.Fit))
	}
	out += " fit       " + fitStyle.Render(fw) + "\n"
	if a.prefs.ShowEstimates {
		var used []uint64
		for _, s := range d.History {
			used = append(used, s.UsedBytes)
		}
		trend := sparkline(used, a.gl)
		switch {
		case d.GrowthPerDay == nil:
			trend += a.th.Dim.Render("  not enough history yet (6 hours)")
		case *d.GrowthPerDay <= 0:
			trend += "  not growing"
		default:
			trend += fmt.Sprintf("  +%s/day (estimate)", formatBytes(uint64(*d.GrowthPerDay), u))
			if d.DaysToFull != nil {
				trend += fmt.Sprintf(", full in about %.0f days (estimate)", *d.DaysToFull)
			}
		}
		out += " trend     " + trend + "\n"
	}
	if h.diskErr != nil {
		out += "\n " + a.th.Warn.Render("last measurement failed: "+a.errText(h.diskErr))
	}
	return out
}
