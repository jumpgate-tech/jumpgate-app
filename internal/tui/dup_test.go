package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// Every fact is shown once: no pane's first line repeats its title (or a
// breadcrumb), and no pane repeats what the status strip (header and KPI
// strip) says.

var nonWord = regexp.MustCompile(`[^a-z0-9/]+`)

func norm(s string) string {
	return strings.TrimSpace(nonWord.ReplaceAllString(strings.ToLower(s), " "))
}

// repeats: line says nothing but the title, or starts with it as a label.
func repeats(line, title string) bool {
	l, t := norm(line), norm(title)
	return t != "" && (l == t || strings.HasPrefix(l, t+" "))
}

func firstLine(lines []string) string {
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			return l
		}
	}
	return ""
}

func TestRepeatsCatchesATitleLine(t *testing.T) {
	for _, c := range []struct {
		line, title string
		want        bool
	}{
		{"Jobs", "Jobs", true}, {" JOBS", "Jobs", true}, {"jobs n/a", "jobs", true}, {"Vault:", "Vault", true},
		{" box-a | gate | PulseChain", "box-a", true},
		{"No jobs.", "Jobs", false}, {" link gate | PulseChain", "box-a", false}, {"slots  passphrase", "Vault", false},
	} {
		if got := repeats(c.line, c.title); got != c.want {
			t.Errorf("repeats(%q, %q) = %v", c.line, c.title, got)
		}
	}
}

// statusFacts are the status strip's segments as the header draws them.
func statusFacts(a *App) []string {
	fs := []string{"server " + a.conn.String(), ansi.Strip(a.alertCount())}
	if a.fleetHas {
		fs = append(fs, "updated "+formatAge(a.now().Sub(a.fleet.At))+" ago")
		if a.fleet.Locked {
			fs = append(fs, "LOCKED")
		}
	}
	if a.srvVer != "" {
		fs = append(fs, a.srvVer)
	}
	if seg := a.setupSegment(); seg != "" {
		fs = append(fs, seg)
	}
	return fs
}

// checkOnce fails when a pane drawn in m opens with its title or repeats
// the status strip.
func checkOnce(t *testing.T, label string, m tea.Model) {
	t.Helper()
	fr := tuitest.Frame(m)
	a := m.(*App)
	if _, ok := a.shown["main"]; !ok {
		t.Fatalf("%s: main was not recorded:\n%s", label, fr)
	}
	for pn, p := range a.shown {
		if pn == "header" || pn == "kpi" {
			continue
		}
		titles := map[string]bool{p.title: true} // hints: no title, ""
		if pn == "main" {
			for _, c := range a.crumbs() {
				titles[c] = true
			}
		}
		if pn == "log" {
			titles["log"] = true
		}
		for ti := range titles {
			if repeats(firstLine(p.lines), ti) {
				t.Errorf("%s %s: the first line %q repeats the title %q:\n%s", label, pn, firstLine(p.lines), ti, fr)
			}
		}
		for _, l := range p.lines {
			for _, f := range statusFacts(a) {
				if strings.Contains(l, f) {
					t.Errorf("%s %s: %q repeats the status strip's %q:\n%s", label, pn, l, f, fr)
				}
			}
		}
	}
}

// TestEveryFactOnce draws every view (the five screens, the hosts table and
// a box on each tab) at 120x40 and 80x24.
func TestEveryFactOnce(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}} {
		w, h := size[0], size[1]
		app := func() tea.Model { return dashApp(t, tuitest.NewFake(), w, h, "ascii", "auto") }
		for i, n := range screenNames {
			checkOnce(t, fmt.Sprintf("%dx%d %s", w, h, n), press(t, app(), fmt.Sprint(i+1)))
		}
		checkOnce(t, fmt.Sprintf("%dx%d hosts", w, h), press(t, app(), "v"))
		m := press(t, app(), "enter")
		for range m.(*App).prefs.HostTabs {
			checkOnce(t, fmt.Sprintf("%dx%d %s", w, h, m.(*App).breadcrumb()), m)
			m = press(t, m, "right")
		}
	}
}
