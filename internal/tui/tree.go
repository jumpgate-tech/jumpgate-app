package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

// The fleet tree (spec §1): the fleet, each box, and under an expanded box
// its nodes, the gateways that route to it and its modules. Modules are
// read only for expanded boxes: on expand, then every 60 s.

// treeNode is one line of the tree. Kind is fleet, box, exec, beacon, gw,
// mod or note; Box is the box it belongs to ("" for the fleet line).
type treeNode struct {
	Kind, Box, Node, Label, State string
}

// treeMods is the last modules reading for a box.
type treeMods struct {
	has    bool
	err    error
	cat    []api.CatalogModule
	target *api.TargetView
	view   api.ModulesView
}

type (
	treeModsMsg struct {
		box    string
		cat    []api.CatalogModule
		target *api.TargetView
		view   api.ModulesView
		err    error
	}
	treeTickMsg struct{}
)

// treeModulesEvery is how often an expanded box's modules are read again.
const treeModulesEvery = 60 * time.Second

func treeTick() tea.Cmd {
	return tea.Tick(treeModulesEvery, func(time.Time) tea.Msg { return treeTickMsg{} })
}

// loadTreeModules reads what the module lines need for one box: the
// catalog and the box's record from the server's store, and its modules.
func loadTreeModules(a *App, box string) tea.Cmd {
	return a.do("modules", func(ctx context.Context, be Backend) tea.Msg {
		msg := treeModsMsg{box: box}
		cat, err := be.Catalog(ctx)
		if err != nil {
			msg.err = err
			return msg
		}
		ts, err := be.Targets(ctx)
		if err != nil {
			msg.err = err
			return msg
		}
		for i := range ts {
			if ts[i].ID == box {
				msg.target = &ts[i]
			}
		}
		msg.cat = cat
		msg.view, msg.err = be.ListModules(ctx, box)
		return msg
	})
}

// toggleExpand opens or closes a box in the tree; opening reads its
// modules the first time.
func (a *App) toggleExpand(box string) tea.Cmd {
	if a.expanded == nil {
		a.expanded = map[string]bool{}
	}
	a.expanded[box] = !a.expanded[box]
	if !a.expanded[box] {
		return nil
	}
	if _, ok := a.mods[box]; ok {
		return nil
	}
	return loadTreeModules(a, box)
}

// treeRefresh re-reads the modules of the expanded boxes only.
func (a *App) treeRefresh() tea.Cmd {
	var cmds []tea.Cmd
	for _, r := range a.fleet.Rows {
		if a.expanded[r.ID] {
			cmds = append(cmds, loadTreeModules(a, r.ID))
		}
	}
	return tea.Batch(cmds...)
}

// gotTreeMods keeps a reading; an error keeps the last good one.
func (a *App) gotTreeMods(msg treeModsMsg) {
	if a.mods == nil {
		a.mods = map[string]treeMods{}
	}
	if msg.err != nil {
		m := a.mods[msg.box]
		m.err = msg.err
		a.mods[msg.box] = m
		return
	}
	a.mods[msg.box] = treeMods{has: true, cat: msg.cat, target: msg.target, view: msg.view}
}

// fleetRows is the fleet in the table's order (its sort and filters).
func (a *App) fleetRows() []api.FleetRow {
	if fs, ok := a.screens[scrFleet].(*fleetScreen); ok {
		rows, _ := fs.rows(a)
		return rows
	}
	return a.fleet.Rows
}

// boxState is a box's one-word state for the tree: its link when that is
// the news (down, HOSTKEY, not set up), else its sync state.
func (a *App) boxState(r api.FleetRow) string {
	w := linkWord(r)
	switch {
	case w == "down" && r.LastSeen != nil:
		return "down " + formatAge(a.now().Sub(*r.LastSeen))
	case w == "HOSTKEY" || w == "locked" || r.Status == nil || r.Status.Overall == "" || r.Status.Overall == api.SyncUnavailable:
		if r.Status == nil && w == "gate" {
			return "n/a"
		}
		return w
	}
	s := syncWord(r.Status.Overall)
	if r.Stale.Status {
		s += "~"
	}
	return s
}

func clientState(st *api.NodeStatus, c func(*api.NodeStatus) api.ClientStatus) string {
	if st == nil {
		return "n/a"
	}
	cs := c(st)
	if cs.State == "" || cs.State == api.SyncUnavailable {
		return "n/a"
	}
	return fmt.Sprintf("%s %dp", syncWord(cs.State), cs.Peers)
}

// treeNodes is the tree as it shows now.
func (a *App) treeNodes() []treeNode {
	rows := a.fleetRows()
	out := []treeNode{{Kind: "fleet", Label: "fleet", State: fmt.Sprintf("%d boxes", len(rows))}}
	if !a.fleetHas {
		out[0].State = "n/a"
	}
	for _, r := range rows {
		id := sanitizeLine(r.ID)
		name := id
		if r.ThisMachine {
			name += "*"
		}
		out = append(out, treeNode{Kind: "box", Box: r.ID, Label: name, State: a.boxState(r)})
		if !a.expanded[r.ID] {
			continue
		}
		out = append(out,
			treeNode{Kind: "exec", Box: r.ID, Node: "exec", Label: "exec", State: clientState(r.Status, func(s *api.NodeStatus) api.ClientStatus { return s.Exec })},
			treeNode{Kind: "beacon", Box: r.ID, Node: "beacon", Label: "beacon", State: clientState(r.Status, func(s *api.NodeStatus) api.ClientStatus { return s.Beacon })})
		for _, g := range a.gws {
			if !g.RoutesTo(r.ID) {
				continue
			}
			label := g.Label
			if label == "" {
				label = g.ID
			}
			st := "n/a"
			switch g.Status.State {
			case "running":
				st = "live"
			case "":
			default:
				st = alertText(g.Status.State)
			}
			out = append(out, treeNode{Kind: "gw", Box: r.ID, Label: "gw " + sanitizeLine(label), State: st})
		}
		m, ok := a.mods[r.ID]
		switch {
		case !ok || (!m.has && m.err == nil):
			out = append(out, treeNode{Kind: "note", Box: r.ID, Label: "modules loading" + a.gl.Ellipsis})
		case !m.has:
			out = append(out, treeNode{Kind: "note", Box: r.ID, Label: "modules n/a"})
		default:
			// The host screen's module cards, read-only: a throwaway screen
			// holding this reading.
			h := &hostScreen{id: r.ID, modCat: m.cat, modTarget: m.target, modView: m.view}
			for _, c := range h.moduleCards(a) {
				if c.state == "disabled" {
					continue
				}
				st := c.state
				if m.view.GateStale {
					st = "gate stale"
				}
				out = append(out, treeNode{Kind: "mod", Box: r.ID, Label: "mod " + sanitizeLine(c.m.ID), State: st})
			}
		}
	}
	return out
}

// treeSelected is the index of the selected line: the tree's own cursor
// while it still names the selection, else the selection's box line.
func (a *App) treeSelected(ns []treeNode) int {
	if a.treeCur >= 0 && a.treeCur < len(ns) {
		n := ns[a.treeCur]
		if (a.selAll && n.Kind == "fleet") || (!a.selAll && n.Box == a.sel && n.Node == a.selNode && n.Kind != "fleet") {
			return a.treeCur
		}
	}
	for i, n := range ns {
		if a.selAll && n.Kind == "fleet" {
			return i
		}
		if !a.selAll && n.Kind == "box" && n.Box == a.sel {
			return i
		}
	}
	return 0
}

// treeView draws the tree in w×h, keeping the selected line in view.
func (a *App) treeView(w, h int) string {
	ns := a.treeNodes()
	cur := a.treeSelected(ns)
	first := 0
	if cur >= h {
		first = cur - h + 1
	}
	var lines []string
	for i := first; i < len(ns) && len(lines) < h; i++ {
		n := ns[i]
		mark := " "
		if i == cur {
			mark = a.gl.Sel
		}
		var left string
		switch n.Kind {
		case "fleet":
			left = mark + "  " + n.Label
		case "box":
			g := a.gl.Closed
			if a.expanded[n.Box] {
				g = a.gl.Open
			}
			left = mark + g + " " + n.Label
		default:
			left = mark + "    " + n.Label
		}
		state := truncate(n.State, max(w/2-1, 1), a.gl.Ellipsis)
		room := max(w-lipgloss.Width(state)-1, 1)
		left = truncate(left, room, a.gl.Ellipsis)
		line := left + strings.Repeat(" ", max(w-lipgloss.Width(left)-lipgloss.Width(state), 1)) + state
		if i == cur {
			line = a.th.Sel.Render(line)
		}
		lines = append(lines, line)
	}
	return fit(strings.Join(lines, "\n"), w, h, a.gl.Ellipsis)
}
