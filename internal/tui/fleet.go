package tui

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

var fleetKeys = struct{ Sort, Columns, Refresh key.Binding }{
	Sort:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort: host, sync, disk, lag")),
	Columns: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "choose columns (Settings)")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh now")),
}

var fleetSorts = []string{"host", "sync", "disk", "lag"}

// fleetScreen is every box at once (spec screen 1).
type fleetScreen struct {
	cursor, sortBy, offset int
	filter                 textinput.Model
	editing                bool
}

func newFleetScreen() *fleetScreen {
	in := textinput.New()
	in.Prompt = "/"
	in.CharLimit = 64
	// ctrl+v would read the clipboard through a command; a bracketed paste
	// arrives as a PasteMsg and still works.
	in.KeyMap.Paste.SetEnabled(false)
	return &fleetScreen{filter: in}
}

func (s *fleetScreen) capturing() bool { return s.editing }

func (s *fleetScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Up, navKeys.Down, navKeys.Enter, navKeys.Filter, fleetKeys.Sort, fleetKeys.Columns, fleetKeys.Refresh}
}

// rows are the fleet after the chain preference and the text filter, sorted.
// A box whose chain is not known yet is never hidden by the chain filter.
func (s *fleetScreen) rows(a *App) ([]api.FleetRow, int) {
	q := strings.ToLower(strings.TrimSpace(s.filter.Value()))
	var out []api.FleetRow
	hidden := 0
	for _, r := range a.fleet.Rows {
		if len(a.prefs.Chains) > 0 && r.ChainID != 0 && !slices.Contains(a.prefs.Chains, r.ChainID) {
			hidden++
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(sanitizeLine(r.ID+" "+r.Network)), q) {
			continue
		}
		out = append(out, r)
	}
	by := fleetSorts[s.sortBy]
	sort.SliceStable(out, func(i, j int) bool { return fleetLess(by, out[i], out[j]) })
	return out, hidden
}

// syncRank orders worst first: stopped, unavailable, syncing, no data, synced.
func syncRank(r api.FleetRow) int {
	if r.Status == nil {
		return 1
	}
	switch r.Status.Overall {
	case api.SyncStopped:
		return 0
	case api.SyncUnavailable, "":
		return 1
	case api.SyncSyncing:
		return 2
	case api.SyncNoData:
		return 3
	case api.SyncSynced:
		return 4
	}
	return 1 // a state this build does not know is no better than unavailable
}

func diskFrac(r api.FleetRow) float64 {
	if r.Status != nil && r.Status.DiskUsedPct != nil {
		return *r.Status.DiskUsedPct / 100
	}
	if r.Disk != nil && r.Disk.UsedBytes()+r.Disk.FreeBytes > 0 {
		return float64(r.Disk.UsedBytes()) / float64(r.Disk.UsedBytes()+r.Disk.FreeBytes)
	}
	return -1
}

func lagOf(r api.FleetRow) int64 {
	if r.Status == nil || r.Status.HeadLag == nil {
		return -1
	}
	return int64(*r.Status.HeadLag)
}

func fleetLess(by string, x, y api.FleetRow) bool {
	switch by {
	case "sync":
		if syncRank(x) != syncRank(y) {
			return syncRank(x) < syncRank(y)
		}
	case "disk":
		if diskFrac(x) != diskFrac(y) {
			return diskFrac(x) > diskFrac(y)
		}
	case "lag":
		if lagOf(x) != lagOf(y) {
			return lagOf(x) > lagOf(y)
		}
	}
	return x.ID < y.ID
}

// shortCount is 1234567 as 1.2M, for narrow columns.
func shortCount(n uint64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 10_000:
		return fmt.Sprintf("%.0fk", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

// shortAddr is 0x1234…5678.
func shortAddr(addr string, g Glyphs) string {
	r := []rune(addr)
	if len(r) <= 12 {
		return addr
	}
	return string(r[:6]) + g.Ellipsis + string(r[len(r)-4:])
}

// cell renders one column of one row. A value the server could not read is
// "n/a"; a row not yet probed is "…"; a stale value carries "~".
func (s *fleetScreen) cell(a *App, r api.FleetRow, col string) string {
	const na = "n/a"
	stale := r.Stale.Status || a.conn != apiclient.Live
	// mark dims a value that may be out of date and tags it, as the sync cells do.
	mark := func(v string) string {
		if stale {
			return a.th.Dim.Render(v + "~")
		}
		return v
	}
	switch col {
	case "host":
		if r.ThisMachine {
			return sanitizeLine(r.ID) + "*"
		}
		return sanitizeLine(r.ID)
	case "link":
		w := linkWord(r)
		if w == "down" && r.LastSeen != nil {
			w += " " + formatAge(a.now().Sub(*r.LastSeen))
		}
		if w == "HOSTKEY" {
			return a.th.Bad.Render(w)
		}
		return w
	case "network":
		if n := sanitizeLine(r.Network); n != "" {
			return n
		}
		return "?"
	case "sync", "el", "cl":
		if r.Status == nil {
			if r.Error == nil && r.LastSeen == nil {
				return a.gl.Ellipsis
			}
			return a.th.Dim.Render(na)
		}
		st := r.Status.Overall
		if col == "el" {
			st = r.Status.Exec.State
		} else if col == "cl" {
			st = r.Status.Beacon.State
		}
		if st == api.SyncUnavailable || st == "" {
			return a.th.Dim.Render(na) // no reading is not "synced", and not 0
		}
		w := sanitizeLine(strings.ReplaceAll(string(st), "_", " "))
		if stale {
			return a.th.Dim.Render(w + "~")
		}
		return a.th.state(st).Render(w)
	case "lag":
		if r.Status == nil || r.Status.HeadLag == nil {
			return a.th.Dim.Render(na)
		}
		return mark(shortCount(*r.Status.HeadLag))
	case "peers":
		if r.Status == nil {
			return a.th.Dim.Render(na)
		}
		if r.Status.Exec.State == api.SyncUnavailable && r.Status.Beacon.State == api.SyncUnavailable {
			return a.th.Dim.Render(na)
		}
		side := func(c api.ClientStatus) string {
			if c.State == api.SyncUnavailable {
				return "?"
			}
			return fmt.Sprint(c.Peers)
		}
		return mark(side(r.Status.Exec) + "/" + side(r.Status.Beacon))
	case "disk":
		frac := diskFrac(r)
		if frac < 0 {
			return a.th.Dim.Render(na)
		}
		// The bar and the number come from the same fraction, so they cannot
		// disagree; the tick marks the expected size where the disk is known.
		var expected uint64
		if d := r.Disk; d != nil && d.UsedBytes()+d.FreeBytes > 0 {
			expected = uint64(float64(d.ExpectedExecBytes+d.ExpectedBeaconBytes) / float64(d.UsedBytes()+d.FreeBytes) * 1000)
		}
		s := fmt.Sprintf("[%s] %.0f%%", bar(uint64(frac*1000), expected, 1000, 8, a.gl), frac*100)
		if r.Stale.Disk {
			return a.th.Dim.Render(s)
		}
		return s
	case "firewall":
		if r.Firewall == nil {
			return a.th.Dim.Render(na)
		}
		switch r.Firewall.Grade {
		case "ok":
			return a.th.OK.Render("ok")
		case "warn":
			return a.th.Warn.Render("warn")
		case "fail":
			return a.th.Bad.Render("FAIL")
		}
		return "?"
	case "jobs":
		return a.gl.Dash // sub-project 2 (spec D27)
	case "age":
		if r.LastSeen == nil {
			return a.th.Dim.Render(na)
		}
		return mark(formatAge(a.now().Sub(*r.LastSeen)))
	case "agent":
		if r.Agent == "" {
			return a.gl.Dash
		}
		return shortAddr(sanitizeLine(r.Agent), a.gl)
	}
	return ""
}

// columns are the preferred columns that fit w after the one-cell selection
// mark (one space between columns), and whether any were cut.
func (s *fleetScreen) columns(a *App, w int) ([]api.ColumnSpec, bool) {
	var out []api.ColumnSpec
	used, cut := 1, false
	for _, id := range a.prefs.FleetColumns {
		c, ok := api.Column(id)
		if !ok {
			continue
		}
		need := c.Width
		if len(out) > 0 {
			need++
		}
		if used+need > w {
			cut = true
			break
		}
		out = append(out, c)
		used += need
	}
	if cut && used+1 > w && len(out) > 1 { // room for the cut marker
		out = out[:len(out)-1]
	}
	return out, cut
}

func (s *fleetScreen) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case fleetMsg, enterMsg, prefsMsg:
		s.clamp(a)
		return nil
	case tea.PasteMsg:
		if !s.editing {
			return nil
		}
		msg.Content = strings.TrimSpace(sanitizeLine(msg.Content))
		var cmd tea.Cmd
		s.filter, cmd = s.filter.Update(msg)
		s.cursor = 0
		s.clamp(a)
		return cmd
	case tea.MouseClickMsg:
		if a.prefs.Mouse && a.detail == nil && a.modal == nil && a.active == scrFleet && msg.Button == tea.MouseLeft {
			top := 2 // the top bar and the header
			if a.skew != "" {
				top++
			}
			rows, _ := s.rows(a)
			if i := s.offset + msg.Y - top; i >= 0 && i < len(rows) {
				s.cursor = i
				s.clamp(a)
			}
		}
		return nil
	case tea.KeyPressMsg:
		if s.editing {
			switch msg.String() {
			case "enter":
				s.editing = false
				s.filter.Blur()
			case "esc":
				s.editing = false
				s.filter.SetValue("")
				s.filter.Blur()
			default:
				var cmd tea.Cmd
				s.filter, cmd = s.filter.Update(msg)
				s.cursor = 0
				return cmd
			}
			s.clamp(a)
			return nil
		}
		rows, _ := s.rows(a)
		switch {
		case key.Matches(msg, navKeys.Up):
			s.cursor--
		case key.Matches(msg, navKeys.Down):
			s.cursor++
		case key.Matches(msg, navKeys.Filter):
			s.editing = true
			return s.filter.Focus()
		case key.Matches(msg, fleetKeys.Sort):
			s.sortBy = (s.sortBy + 1) % len(fleetSorts)
		case key.Matches(msg, fleetKeys.Columns):
			return a.openScreen(scrSettings)
		case key.Matches(msg, fleetKeys.Refresh):
			return a.do("refresh", func(ctx context.Context) tea.Msg {
				if _, err := a.be.Fleet(ctx); err != nil {
					return errMsg{what: "refresh", err: err}
				}
				return flashMsg("refreshing the fleet")
			})
		case key.Matches(msg, navKeys.Enter) && s.cursor < len(rows):
			return a.openHost(rows[s.cursor].ID)
		}
		s.clamp(a)
	}
	return nil
}

// clamp keeps the cursor on a row and names that row in the status bar.
func (s *fleetScreen) clamp(a *App) {
	rows, _ := s.rows(a)
	s.cursor = min(max(s.cursor, 0), max(len(rows)-1, 0))
	if a.detail == nil && a.active == scrFleet && len(rows) > 0 {
		a.sel = rows[s.cursor].ID
	}
}

func (s *fleetScreen) view(a *App, w, h int) string {
	cols, cut := s.columns(a, w)
	titles := make([]string, len(cols))
	for i, c := range cols {
		titles[i] = pad(c.Title, c.Width)
	}
	var head strings.Builder
	head.WriteString(" " + strings.Join(titles, " "))
	if cut {
		marker := "›"
		if a.gl.ASCII {
			marker = ">"
		}
		head.WriteString(marker)
	}
	out := []string{a.th.Title.Render(head.String())}
	if !a.fleetHas {
		return strings.Join(append(out, " "+a.th.Dim.Render("waiting for the fleet"+a.gl.Ellipsis)), "\n")
	}
	rows, hidden := s.rows(a)
	room := max(h-3, 1)
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+room {
		s.offset = s.cursor - room + 1
	}
	for i := s.offset; i < len(rows) && i < s.offset+room; i++ {
		mark := " "
		if i == s.cursor {
			mark = a.gl.Sel
		}
		cells := make([]string, len(cols))
		for j, c := range cols {
			cells[j] = pad(truncate(s.cell(a, rows[i], c.ID), c.Width, a.gl.Ellipsis), c.Width)
		}
		out = append(out, mark+strings.Join(cells, " "))
	}
	foot := fmt.Sprintf(" %d boxes", len(rows))
	if hidden > 0 {
		foot += fmt.Sprintf(", %d hidden by the chain filter", hidden)
	}
	foot += fmt.Sprintf(" %s sort: %s %s ~ stale  * this machine", a.gl.Sep, fleetSorts[s.sortBy], a.gl.Sep)
	if s.editing || s.filter.Value() != "" {
		foot = " " + s.filter.View()
	}
	for len(out) < h-1 {
		out = append(out, "")
	}
	return strings.Join(append(out, a.th.Dim.Render(foot)), "\n")
}
