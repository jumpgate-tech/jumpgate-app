package tui

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
)

// The activity log (spec U6): events found by diffing successive snapshots
// of what the TUI already holds. Nothing here asks a server or a box for
// anything; the log lives in memory for the session and is never saved.

type logSev int

const (
	logInfo logSev = iota
	logOK
	logWarn
	logFail
)

func (s logSev) String() string { return [...]string{"info", "ok", "warn", "FAIL"}[s] }

// logEvent is one line of the log. Box is the box (or gateway) it is
// about, Node "exec"/"beacon" when it is about one client; Text is
// sanitized, and its free text from a server or box masked, ready to draw.
type logEvent struct {
	At        time.Time
	Sev       logSev
	Box, Node string
	Text      string
}

// snapshot is what the App holds at one moment; the Has flags say a reading
// of that kind exists. A kind without a reading on either side is not
// compared, so the first reading logs nothing.
type snapshot struct {
	Fleet       api.Fleet
	FleetHas    bool
	Jobs        []api.JobView
	JobsHas     bool // the inbox, while the Jobs view watches it
	Gateways    []api.GatewaySummary
	GatewaysHas bool
	Targets     []api.TargetView
	TargetsHas  bool
	Modules     map[string]api.ModulesView // expanded boxes with a reading
	Now         time.Time
}

// actKeep is the ring's size.
const actKeep = 200

// actLog is the ring of events, oldest first.
type actLog struct{ ev []logEvent }

func (l *actLog) add(evs ...logEvent) {
	l.ev = append(l.ev, evs...)
	if n := len(l.ev) - actKeep; n > 0 {
		l.ev = append(l.ev[:0:0], l.ev[n:]...)
	}
}

// scoped is the events about box (all when box is ""), and about node when
// one is given, newest first.
func (l *actLog) scoped(box, node string) []logEvent {
	var out []logEvent
	for i := len(l.ev) - 1; i >= 0; i-- {
		e := l.ev[i]
		if box != "" && e.Box != box {
			continue
		}
		if node != "" && e.Node != node {
			continue
		}
		out = append(out, e)
	}
	return out
}

var numbers = regexp.MustCompile(`[0-9]+`)

// alertKey names an alert across readings: the numbers in its text (an age,
// a count, a percentage) fold, so "down 4m" and "down 5m" are one alert.
func alertKey(x alert) string { return x.Box + "\x00" + numbers.ReplaceAllString(x.Text, "#") }

// diffEvents is what changed from prev to next. Pure; every event is at
// next.Now.
func diffEvents(prev, next snapshot) []logEvent {
	var out []logEvent
	add := func(sev logSev, box, node, text string) {
		out = append(out, logEvent{At: next.Now, Sev: sev, Box: sanitizeLine(box), Node: node, Text: sanitizeLine(text)})
	}
	if prev.FleetHas && next.FleetHas {
		diffFleet(prev.Fleet, next.Fleet, add)
	}
	if prev.TargetsHas && next.TargetsHas {
		was := map[string]api.TargetView{}
		for _, t := range prev.Targets {
			was[t.ID] = t
		}
		for _, t := range next.Targets {
			if p, ok := was[t.ID]; ok && p.Agent == nil && t.Agent != nil {
				add(logOK, t.ID, "", "gate paired: signs for this controller")
			}
		}
	}
	if prev.JobsHas && next.JobsHas {
		diffJobs(prev.Jobs, next.Jobs, add)
	}
	for _, box := range slices.Sorted(maps.Keys(next.Modules)) {
		if pv, ok := prev.Modules[box]; ok {
			diffModules(box, pv, next.Modules[box], add)
		}
	}
	if prev.FleetHas && next.FleetHas {
		diffAlerts(prev, next, add)
	}
	return out
}

type addFunc func(sev logSev, box, node, text string)

func diffFleet(prev, next api.Fleet, add addFunc) {
	was := map[string]api.FleetRow{}
	for _, r := range prev.Rows {
		was[r.ID] = r
	}
	seen := map[string]bool{}
	for _, r := range next.Rows {
		seen[r.ID] = true
		p, ok := was[r.ID]
		if !ok {
			add(logInfo, r.ID, "", "box added")
			continue
		}
		who := "box"
		if r.Link == api.LinkAgent || r.Link == api.LinkAgentLocal {
			who = "gate"
		}
		switch {
		case p.Reachable && !r.Reachable && r.LastSeen != nil:
			add(logFail, r.ID, "", who+" down")
		case !p.Reachable && r.Reachable:
			add(logOK, r.ID, "", who+" up")
		}
		if p.Status == nil || r.Status == nil {
			continue
		}
		for _, n := range []struct {
			name string
			p, c api.ClientStatus
		}{{"exec", p.Status.Exec, r.Status.Exec}, {"beacon", p.Status.Beacon, r.Status.Beacon}} {
			switch {
			case n.p.State != n.c.State && n.c.State != "" && n.p.State != "":
				sev := logInfo
				switch n.c.State {
				case api.SyncSynced:
					sev = logOK
				case api.SyncSyncing, api.SyncNoData:
					sev = logWarn
				case api.SyncStopped:
					sev = logFail
				}
				add(sev, r.ID, n.name, fmt.Sprintf("%s %s (was %s)", n.name, syncWord(n.c.State), syncWord(n.p.State)))
			case n.c.State == api.SyncSyncing && n.c.Head == n.p.Head && r.Status.At.After(p.Status.At):
				add(logWarn, r.ID, n.name, fmt.Sprintf("%s stalled at %d", n.name, n.c.Head))
			}
		}
	}
	for _, r := range prev.Rows {
		if !seen[r.ID] {
			add(logInfo, r.ID, "", "box removed")
		}
	}
}

func syncWord(s api.SyncState) string { return strings.ReplaceAll(string(s), "_", " ") }

func diffJobs(prev, next []api.JobView, add addFunc) {
	was := map[string]api.JobView{}
	for _, j := range prev {
		was[j.Target+"/"+j.ID] = j
	}
	for _, j := range next {
		name := "job " + j.Kind
		p, ok := was[j.Target+"/"+j.ID]
		switch {
		case !ok:
			add(logInfo, j.Target, "", name+" started")
		case p.State != "needs-decision" && j.State == "needs-decision":
			add(logWarn, j.Target, "", name+" needs decision")
		case p.State == "needs-decision" && j.State != "needs-decision":
			add(logOK, j.Target, "", name+" decision answered")
		case p.State != j.State:
			add(logInfo, j.Target, "", name+" "+alertText(j.State))
		case p.Step != j.Step:
			add(logInfo, j.Target, "", fmt.Sprintf("%s step %d/%d", name, j.Step+1, len(j.Steps)))
		}
	}
}

func diffModules(box string, prev, next api.ModulesView, add addFunc) {
	switch {
	case !prev.GateStale && next.GateStale:
		add(logWarn, box, "", "gate stale: module status needs a newer gate")
		return
	case prev.GateStale && !next.GateStale:
		add(logOK, box, "", "gate current: module status is back")
	}
	was := map[string]api.ModuleStatus{}
	for _, m := range prev.Modules {
		was[m.ID+"/"+m.Instance] = m
	}
	for _, m := range next.Modules {
		p := was[m.ID+"/"+m.Instance]
		on := func(s api.ModuleStatus) bool { return s.Served || s.Active }
		switch {
		case m.Error != "" && m.Error != p.Error:
			add(logFail, box, "", "module "+m.ID+": "+alertText(m.Error))
		case on(m) && !on(p):
			add(logOK, box, "", "module "+m.ID+" on")
		case !on(m) && on(p):
			add(logInfo, box, "", "module "+m.ID+" off")
		}
	}
}

// diffAlerts logs each alert raised (or changed severity) or cleared. A box
// going down is already its own event, so its "down" alert is not logged
// twice; an alert about a gateway is compared only when both snapshots hold
// a gateways reading, so a missing reading never "clears" one.
func diffAlerts(prev, next snapshot, add addFunc) {
	both := prev.GatewaysHas && next.GatewaysHas
	list := func(s snapshot) ([]alert, map[string]alert) {
		boxes := map[string]bool{}
		for _, r := range s.Fleet.Rows {
			boxes[r.ID] = true
		}
		var l []alert
		m := map[string]alert{}
		for _, x := range alerts(alertInput{Fleet: s.Fleet, FleetHas: true, Gateways: s.Gateways, GatewaysHas: s.GatewaysHas, Now: s.Now}) {
			if strings.HasPrefix(x.Text, "down ") || (!boxes[x.Box] && !both) {
				continue
			}
			l, m[alertKey(x)] = append(l, x), x
		}
		return l, m
	}
	wasL, was := list(prev)
	nowL, now := list(next)
	sev := map[alertSev]logSev{sevWarn: logWarn, sevFail: logFail}
	for _, x := range nowL {
		if p, ok := was[alertKey(x)]; !ok || p.Sev != x.Sev {
			add(sev[x.Sev], x.Box, "", x.Text)
		}
	}
	for _, x := range wasL {
		if _, ok := now[alertKey(x)]; !ok {
			add(logOK, x.Box, "", "cleared: "+x.Text)
		}
	}
}
