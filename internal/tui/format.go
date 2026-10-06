package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// formatBytes renders a size in decimal (GB) or binary (GiB) units with three
// significant figures.
func formatBytes(b uint64, units string) string {
	base, names := 1000.0, []string{"B", "kB", "MB", "GB", "TB", "PB"}
	if units == "GiB" {
		base, names = 1024.0, []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	}
	v, i := float64(b), 0
	for v >= base && i < len(names)-1 {
		v /= base
		i++
	}
	switch {
	case i == 0:
		return fmt.Sprintf("%d B", b)
	case v < 10:
		return fmt.Sprintf("%.2f %s", v, names[i])
	case v < 100:
		return fmt.Sprintf("%.1f %s", v, names[i])
	}
	return fmt.Sprintf("%.0f %s", v, names[i])
}

// formatAge is a short age: 12s, 3m, 5h, 3d.
func formatAge(d time.Duration) string {
	switch {
	case d < time.Second:
		return "0s"
	case d < time.Minute:
		return strconv.Itoa(int(d/time.Second)) + "s"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	case d < 48*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + "h"
	}
	return strconv.Itoa(int(d/(24*time.Hour))) + "d"
}

// formatCount groups digits by thousands.
func formatCount(n uint64) string {
	s := strconv.FormatUint(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// truncate cuts s to w cells, ANSI-aware, ending in ell when it cut.
func truncate(s string, w int, ell string) string { return ansi.Truncate(s, w, ell) }

// pad right-pads s with spaces to w cells (cutting it if longer).
func pad(s string, w int) string {
	if n := lipgloss.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return truncate(s, w, "")
}

// fit makes block exactly w×h: every line cut or padded to w, the block cut
// or padded to h lines. Text that does not fit is cut, never wrapped.
func fit(block string, w, h int, ell string) string {
	lines := strings.Split(block, "\n")
	out := make([]string, h)
	for i := range out {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		if lipgloss.Width(line) > w {
			line = truncate(line, w, ell)
		}
		out[i] = pad(line, w)
	}
	return strings.Join(out, "\n")
}

// bar draws used out of total in width cells, with a tick where expected
// falls (the expected-size estimate).
func bar(used, expected, total uint64, width int, g Glyphs) string {
	if total == 0 || width <= 0 {
		return strings.Repeat(g.Empty, max(width, 0))
	}
	fill := int(math.Round(float64(used) / float64(total) * float64(width)))
	fill = min(max(fill, 0), width)
	tick := -1
	if expected > 0 {
		tick = min(int(float64(expected)/float64(total)*float64(width)), width-1)
	}
	var b strings.Builder
	for i := 0; i < width; i++ {
		switch {
		case i == tick:
			b.WriteString(g.Tick)
		case i < fill:
			b.WriteString(g.Full)
		default:
			b.WriteString(g.Empty)
		}
	}
	return b.String()
}

// sparkline draws vals scaled between their min and max.
func sparkline(vals []uint64, g Glyphs) string {
	if len(vals) == 0 {
		return ""
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		lo, hi = min(lo, v), max(hi, v)
	}
	var b strings.Builder
	for _, v := range vals {
		i := 0
		if hi > lo {
			i = int(float64(v-lo) / float64(hi-lo) * float64(len(g.Spark)-1))
		}
		b.WriteString(g.Spark[i])
	}
	return b.String()
}
