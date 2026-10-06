package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/ai"
	"github.com/valve-tech/jumpgate/internal/api"
)

// aiProviders are the providers the server knows (ai.Known), "" being none.
// A test keeps this list equal to the server's.
var aiProviders = []string{"", "gemini", "groq", "ollama"}

// Replies carry the screen's gen: one that started before the screen was
// left or re-entered is dropped.
type (
	settingsMsg struct {
		gen uint64
		s   api.Settings
		err error
	}
	webOpenedMsg struct {
		gen uint64
		err error
	}
)

// settingItem is one line of Settings: its value, and what space/enter does.
type settingItem struct {
	section, label string
	value          func(a *App) string
	act            func(a *App, s *settingsScreen) tea.Cmd
}

// settingsScreen is every display choice, saved to the server the moment it
// changes (spec D20), plus the AI provider and key and the web app.
//
// The AI key is typed into keyInput, which masks it. It is sent once, only
// through SaveSettings, and the field is emptied on save, on esc and on
// leaving the screen. The screen only ever learns whether a key is set.
type settingsScreen struct {
	cursor, offset int
	settings       api.Settings
	settingsHas    bool
	keyInput       textinput.Model
	editingKey     bool

	gen    uint64
	ctx    context.Context
	cancel context.CancelFunc
}

func (s *settingsScreen) capturing() bool { return s.editingKey }

func (s *settingsScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Up, navKeys.Down,
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space/enter", "toggle or change"))}
}

// begin starts a visit: earlier commands end and their replies are dropped.
func (s *settingsScreen) begin(a *App) {
	s.end()
	s.gen++
	s.ctx, s.cancel = context.WithCancel(a.ctx)
}

// end cancels the visit's commands and forgets any key being typed.
func (s *settingsScreen) end() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.stopEditing()
}

func (s *settingsScreen) stopEditing() {
	s.keyInput.SetValue("")
	s.keyInput.Blur()
	s.editingKey = false
}

func clonePrefs(p api.UIPrefs) api.UIPrefs {
	p.FleetColumns = slices.Clone(p.FleetColumns)
	p.HostTabs = slices.Clone(p.HostTabs)
	p.Chains = slices.Clone(p.Chains)
	return p
}

// savePrefs shows the change at once and saves it. If the server refuses
// (invalid_prefs, with a hint, or any error) the screen goes back to what it
// showed before and says why.
func savePrefs(a *App, p api.UIPrefs) tea.Cmd {
	old := clonePrefs(a.prefs)
	a.setPrefs(p)
	be := a.be
	return a.do("settings", func(ctx context.Context) tea.Msg {
		saved, err := be.SavePrefs(ctx, p)
		if err != nil {
			return prefsMsg{p: old, err: err, save: true}
		}
		return prefsMsg{p: saved}
	})
}

func check(on bool) string {
	if on {
		return "[x]"
	}
	return "[ ]"
}

func toggle(list []string, id string) []string {
	if i := slices.Index(list, id); i >= 0 {
		return slices.Delete(list, i, i+1)
	}
	return append(list, id)
}

// seenChains are the chains the fleet runs, for the chain filter.
func seenChains(a *App) []api.FleetRow {
	byID := map[int]api.FleetRow{}
	for _, r := range a.fleet.Rows {
		if r.ChainID > 0 {
			byID[r.ChainID] = r
		}
	}
	var out []api.FleetRow
	for _, r := range byID {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChainID < out[j].ChainID })
	return out
}

func (s *settingsScreen) providerName() string {
	if !s.settingsHas {
		return "loading"
	}
	if s.settings.AIProvider == "" {
		return "none"
	}
	return sanitizeLine(s.settings.AIProvider)
}

// disclosure is what Explain sends and to whom for the provider in use: the
// server's line, or the shared one when the server sent none.
func (s *settingsScreen) disclosure() string {
	d := s.settings.AIDisclosure
	if d == "" {
		d = ai.Disclosure(s.settings.AIProvider)
	}
	return sanitizeLine(d)
}

func (s *settingsScreen) items(a *App) []settingItem {
	var items []settingItem
	for _, c := range api.FleetColumns {
		c := c
		items = append(items, settingItem{"FLEET COLUMNS", c.Title,
			func(a *App) string { return check(slices.Contains(a.prefs.FleetColumns, c.ID)) },
			func(a *App, _ *settingsScreen) tea.Cmd {
				if c.ID == "host" {
					a.flash = "the host column is always shown"
					return nil
				}
				p := clonePrefs(a.prefs)
				// Keep registry order, whatever order they were toggled in.
				on := toggle(p.FleetColumns, c.ID)
				p.FleetColumns = nil
				for _, rc := range api.FleetColumns {
					if slices.Contains(on, rc.ID) {
						p.FleetColumns = append(p.FleetColumns, rc.ID)
					}
				}
				return savePrefs(a, p)
			}})
	}
	for _, tab := range api.HostTabs {
		tab := tab
		items = append(items, settingItem{"HOST TABS", tabTitles[tab],
			func(a *App) string { return check(slices.Contains(a.prefs.HostTabs, tab)) },
			func(a *App, _ *settingsScreen) tea.Cmd {
				p := clonePrefs(a.prefs)
				on := toggle(p.HostTabs, tab)
				if len(on) == 0 {
					a.flash = "at least one tab stays"
					return nil
				}
				p.HostTabs = nil
				for _, t := range api.HostTabs {
					if slices.Contains(on, t) {
						p.HostTabs = append(p.HostTabs, t)
					}
				}
				return savePrefs(a, p)
			}})
	}
	for _, r := range seenChains(a) {
		id, name := r.ChainID, sanitizeLine(r.Network)
		items = append(items, settingItem{"CHAINS SHOWN (none ticked off: all)", fmt.Sprintf("%s (%d)", name, id),
			func(a *App) string { return check(len(a.prefs.Chains) == 0 || slices.Contains(a.prefs.Chains, id)) },
			func(a *App, _ *settingsScreen) tea.Cmd {
				p := clonePrefs(a.prefs)
				var all []int
				for _, c := range seenChains(a) {
					all = append(all, c.ChainID)
				}
				if len(p.Chains) == 0 {
					p.Chains = all // the filter starts from "everything shown"
				}
				if i := slices.Index(p.Chains, id); i >= 0 {
					p.Chains = slices.Delete(p.Chains, i, i+1)
				} else {
					p.Chains = append(p.Chains, id)
				}
				sort.Ints(p.Chains)
				if slices.Equal(p.Chains, all) {
					p.Chains = []int{}
				}
				return savePrefs(a, p)
			}})
	}
	cycle := func(label string, get func(api.UIPrefs) string, set func(*api.UIPrefs, string), choices []string) settingItem {
		return settingItem{"DISPLAY", label,
			func(a *App) string { return sanitizeLine(get(a.prefs)) },
			func(a *App, _ *settingsScreen) tea.Cmd {
				p := clonePrefs(a.prefs)
				i := slices.Index(choices, get(p))
				set(&p, choices[(i+1)%len(choices)])
				return savePrefs(a, p)
			}}
	}
	flag := func(label string, get func(api.UIPrefs) bool, set func(*api.UIPrefs, bool)) settingItem {
		return settingItem{"DISPLAY", label,
			func(a *App) string { return check(get(a.prefs)) },
			func(a *App, _ *settingsScreen) tea.Cmd {
				p := clonePrefs(a.prefs)
				set(&p, !get(p))
				return savePrefs(a, p)
			}}
	}
	var refresh []string
	for _, r := range api.RefreshChoices {
		refresh = append(refresh, fmt.Sprintf("%ds", r))
	}
	items = append(items,
		cycle("Refresh", func(p api.UIPrefs) string { return fmt.Sprintf("%ds", p.RefreshSeconds) },
			func(p *api.UIPrefs, v string) { fmt.Sscanf(v, "%ds", &p.RefreshSeconds) }, refresh),
		cycle("Units", func(p api.UIPrefs) string { return p.Units }, func(p *api.UIPrefs, v string) { p.Units = v }, []string{"GB", "GiB"}),
		cycle("Glyphs", func(p api.UIPrefs) string { return p.Glyphs }, func(p *api.UIPrefs, v string) { p.Glyphs = v }, []string{"auto", "unicode", "ascii"}),
		flag("Compact rows", func(p api.UIPrefs) bool { return p.Compact }, func(p *api.UIPrefs, v bool) { p.Compact = v }),
		flag("Show estimates", func(p api.UIPrefs) bool { return p.ShowEstimates }, func(p *api.UIPrefs, v bool) { p.ShowEstimates = v }),
		flag("Bell on jobs that need you (from sub-project 2)", func(p api.UIPrefs) bool { return p.Bell }, func(p *api.UIPrefs, v bool) { p.Bell = v }),
		flag("Mouse (off keeps text selectable)", func(p api.UIPrefs) bool { return p.Mouse }, func(p *api.UIPrefs, v bool) { p.Mouse = v }),
	)
	items = append(items,
		settingItem{"AI EXPLAIN", "AI provider",
			func(a *App) string { return s.providerName() },
			func(a *App, s *settingsScreen) tea.Cmd {
				if !s.settingsHas {
					a.flash = "the AI settings are still loading"
					return nil
				}
				next := aiProviders[(slices.Index(aiProviders, s.settings.AIProvider)+1)%len(aiProviders)]
				return s.save(a, api.SettingsUpdate{AIProvider: &next}, "")
			}},
		settingItem{"AI EXPLAIN", "AI key",
			func(a *App) string {
				switch {
				case !s.settingsHas:
					return "loading"
				case s.settings.AIKeySet:
					return "set (enter to replace)"
				}
				return "not set (enter to type one)"
			},
			func(a *App, s *settingsScreen) tea.Cmd {
				in := textinput.New()
				in.EchoMode = textinput.EchoPassword
				in.Prompt = "key> "
				in.CharLimit = 512
				in.KeyMap.Paste.SetEnabled(false) // terminal paste still works
				s.keyInput = in
				s.editingKey = true
				return s.keyInput.Focus()
			}},
		settingItem{"AI EXPLAIN", "Clear the AI key",
			func(a *App) string { return "" },
			func(a *App, s *settingsScreen) tea.Cmd {
				if !s.settingsHas || !s.settings.AIKeySet {
					a.flash = "no AI key is set"
					return nil
				}
				a.modal = newConfirm("clear the AI key",
					"Explain stops working until a new key is entered.", "clear",
					func() tea.Cmd {
						empty := ""
						return s.save(a, api.SettingsUpdate{AIKey: &empty}, "")
					})
				return nil
			}},
	)
	if a.o.OpenWeb != nil {
		items = append(items, settingItem{"WEB APP", "Open the web app", func(*App) string { return "" },
			func(a *App, s *settingsScreen) tea.Cmd {
				if s.ctx == nil {
					return nil
				}
				gen, ctx, open := s.gen, s.ctx, a.o.OpenWeb
				return func() tea.Msg { return webOpenedMsg{gen: gen, err: open(ctx)} }
			}})
	}
	return items
}

// save sends a settings update. secret, if not empty, is a value that must
// never come back out in an error message.
func (s *settingsScreen) save(a *App, u api.SettingsUpdate, secret string) tea.Cmd {
	if s.ctx == nil {
		return nil
	}
	gen, ctx, be := s.gen, s.ctx, a.be
	return func() tea.Msg {
		st, err := be.SaveSettings(ctx, u)
		if err != nil && secret != "" && strings.Contains(err.Error(), secret) {
			err = errors.New("the server refused the AI key")
		}
		return settingsMsg{gen: gen, s: st, err: err}
	}
}

// find is the index of the item with this label, or -1.
func (s *settingsScreen) find(a *App, label string) int {
	items := s.items(a)
	for i, it := range items {
		if it.label == label {
			return i
		}
	}
	for i, it := range items {
		if strings.HasPrefix(it.label, label) {
			return i
		}
	}
	return -1
}

func (s *settingsScreen) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case enterMsg:
		s.begin(a)
		gen, ctx, be := s.gen, s.ctx, a.be
		return func() tea.Msg {
			st, err := be.Settings(ctx)
			return settingsMsg{gen: gen, s: st, err: err}
		}
	case leaveMsg:
		s.end()
		s.gen++ // replies still in flight are for a visit that is over
	case settingsMsg:
		if msg.gen != s.gen {
			return nil
		}
		if msg.err != nil {
			a.flash = "AI settings: " + a.errText(msg.err)
			return nil
		}
		s.settings, s.settingsHas = msg.s, true
	case webOpenedMsg:
		if msg.gen != s.gen {
			return nil
		}
		if msg.err != nil {
			a.flash = "could not open the web app: " + a.errText(msg.err)
		} else {
			a.flash = "the web app is open in your browser"
		}
	case tea.PasteMsg:
		if s.editingKey && a.active == scrSettings {
			var cmd tea.Cmd
			s.keyInput, cmd = s.keyInput.Update(tea.PasteMsg{Content: strings.TrimSpace(sanitizeLine(msg.Content))})
			return cmd
		}
	case tea.KeyPressMsg:
		if a.active != scrSettings || a.detail != nil {
			return nil
		}
		if s.editingKey {
			switch msg.String() {
			case "esc":
				s.stopEditing()
			case "enter":
				v := strings.TrimSpace(s.keyInput.Value())
				s.stopEditing()
				if v == "" {
					a.flash = "nothing entered; the key is unchanged (use Clear the AI key to remove it)"
					return nil
				}
				a.flash = "saving the AI key" + a.gl.Ellipsis
				return s.save(a, api.SettingsUpdate{AIKey: &v}, v)
			default:
				var cmd tea.Cmd
				s.keyInput, cmd = s.keyInput.Update(msg)
				return cmd
			}
			return nil
		}
		items := s.items(a)
		s.cursor = min(s.cursor, len(items)-1)
		switch {
		case key.Matches(msg, navKeys.Up):
			s.cursor = max(s.cursor-1, 0)
		case key.Matches(msg, navKeys.Down):
			s.cursor = min(s.cursor+1, len(items)-1)
		case msg.String() == "space" || msg.String() == "enter":
			return items[s.cursor].act(a, s)
		}
	}
	return nil
}

func (s *settingsScreen) view(a *App, w, h int) string {
	items := s.items(a)
	s.cursor = min(s.cursor, len(items)-1)
	var lines []string
	cursorLine := 0
	section := ""
	for i, it := range items {
		if it.section != section {
			section = it.section
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, " "+a.th.Title.Render(section))
		}
		mark := " "
		if i == s.cursor {
			mark, cursorLine = a.gl.Sel, len(lines)
		}
		v := it.value(a)
		if strings.HasPrefix(v, "[") {
			lines = append(lines, fmt.Sprintf("%s %s %s", mark, v, it.label))
		} else {
			lines = append(lines, fmt.Sprintf("%s %-48s %s", mark, it.label, v))
		}
		if i == s.cursor && it.label == "AI provider" {
			if d := s.disclosure(); d != "" {
				lines = append(lines, "    "+a.th.Dim.Render(d))
				cursorLine = len(lines) - 1 // keep the disclosure in view with its item
			}
		}
	}
	room := max(h-2, 1)
	if cursorLine < s.offset {
		s.offset = cursorLine
	}
	if cursorLine >= s.offset+room {
		s.offset = cursorLine - room + 1
	}
	end := min(s.offset+room, len(lines))
	out := strings.Join(lines[s.offset:end], "\n")
	foot := " ↑↓ move  space toggle  enter change  (saved at once)"
	if s.editingKey {
		foot = " " + s.keyInput.View() + "   enter save, esc cancel"
	}
	pad := strings.Repeat("\n", max(room-(end-s.offset), 0))
	return out + pad + "\n" + a.th.Dim.Render(foot)
}
