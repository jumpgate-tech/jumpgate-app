package tui

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

// The hosts view's upkeep (inventory D): upgrade a box's gate, the host
// keys recorded for it (forget one, typed confirmation), and leftover
// containers a gateway merge stopped managing (dismiss one, typed
// confirmation).

type (
	upgradedMsg struct {
		id  string
		v   api.AgentUpgradeView
		err error
	}
	hostKeysMsg struct {
		id   string
		recs []api.RecordedHostKeys
		err  error
	}
	forgotMsg struct {
		hostPort, fp string
		err          error
	}
	// orphan is one of GET /api/gateways' leftover containers.
	orphan struct {
		ContainerName string `json:"containerName"`
		TargetID      string `json:"targetId"`
		MergedInto    string `json:"mergedInto"`
	}
	orphansMsg struct {
		os  []orphan
		err error
	}
	dismissedMsg struct {
		name string
		err  error
	}
)

func targetAPI(id, rest string) string { return "/api/targets/" + url.PathEscape(id) + rest }

func upgradeGate(a *App, id string) tea.Cmd {
	a.flash = "upgrading the gate on " + sanitizeLine(id) + a.gl.Ellipsis
	return a.do("upgrade", func(ctx context.Context, be Backend) tea.Msg {
		var v api.AgentUpgradeView
		err := be.Do(ctx, http.MethodPost, targetAPI(id, "/agent/upgrade"), nil, &v)
		return upgradedMsg{id: id, v: v, err: err}
	})
}

// hostPorts is the box's SSH address and each jump host's, as the
// confirmed store records them.
func hostPorts(s *api.SSHView) []string {
	var out []string
	for ; s != nil; s = s.Jump {
		p := s.Port
		if p == 0 {
			p = 22
		}
		out = append(out, net.JoinHostPort(s.Host, strconv.Itoa(p)))
	}
	return out
}

func loadHostKeys(a *App, id string, ssh *api.SSHView) tea.Cmd {
	hps := hostPorts(ssh)
	return a.do("host keys", func(ctx context.Context, be Backend) tea.Msg {
		var recs []api.RecordedHostKeys
		for _, hp := range hps {
			var r api.RecordedHostKeys
			if err := be.Do(ctx, http.MethodGet, "/api/hostkeys?hostPort="+url.QueryEscape(hp), nil, &r); err != nil {
				return hostKeysMsg{id: id, err: err}
			}
			r.HostPort = hp
			recs = append(recs, r)
		}
		return hostKeysMsg{id: id, recs: recs}
	})
}

func loadOrphans(a *App) tea.Cmd {
	return a.do("leftover containers", func(ctx context.Context, be Backend) tea.Msg {
		var v struct {
			Orphans []orphan `json:"orphans"`
		}
		err := be.Do(ctx, http.MethodGet, "/api/gateways", nil, &v)
		return orphansMsg{os: v.Orphans, err: err}
	})
}

// adminMsg handles the replies above; ok is false for any other message.
func adminMsg(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case upgradedMsg:
		id := sanitizeLine(msg.id)
		switch {
		case msg.err != nil:
			a.flash = "upgrade the gate on " + id + ": " + a.errText(msg.err)
		case msg.v.Upgraded:
			a.flash = "gate on " + id + ": " + sanitizeLine(msg.v.From) + " -> " + sanitizeLine(msg.v.To)
		default:
			a.flash = "the gate on " + id + " is already " + sanitizeLine(msg.v.To)
		}
		return loadTargets(a), true
	case hostKeysMsg:
		if msg.err != nil {
			a.flash = "host keys for " + sanitizeLine(msg.id) + ": " + a.errText(msg.err)
			return nil, true
		}
		if a.modal == nil {
			a.modal = newHostKeysModal(msg.id, msg.recs)
		}
		return nil, true
	case forgotMsg:
		if msg.err != nil {
			a.flash = "forget " + sanitizeLine(msg.fp) + ": " + a.errText(msg.err)
		} else {
			a.flash = "forgot " + sanitizeLine(msg.fp) + " for " + sanitizeLine(msg.hostPort) + "; the next connection asks to confirm the key again"
		}
		return nil, true
	case orphansMsg:
		if msg.err != nil {
			a.flash = "leftover containers: " + a.errText(msg.err)
			return nil, true
		}
		if a.modal == nil {
			a.modal = newOrphansModal(msg.os)
		}
		return nil, true
	case dismissedMsg:
		if msg.err != nil {
			a.flash = "dismiss " + sanitizeLine(msg.name) + ": " + a.errText(msg.err)
		} else {
			a.flash = "dismissed " + sanitizeLine(msg.name) + "; the container itself was not touched"
		}
		return nil, true
	}
	return nil, false
}

// listModal is a list with one action on the selected row. The action
// opens a typed confirmation, so neither a key nor a click here changes
// anything by itself.
type listModal struct {
	title, note, empty string
	rows               []string
	cur                int
	act                key.Binding
	why                func(i int) string // why row i cannot be acted on, or ""
	run                func(a *App, i int)
}

func (m *listModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch {
	case k.String() == "esc":
		return true, nil
	case key.Matches(k, navKeys.Up):
		m.cur = max(m.cur-1, 0)
	case key.Matches(k, navKeys.Down):
		m.cur = min(m.cur+1, max(len(m.rows)-1, 0))
	case key.Matches(k, m.act) && m.cur < len(m.rows):
		if why := m.why(m.cur); why != "" {
			m.note = why
			return false, nil
		}
		m.run(a, m.cur)
	}
	return false, nil
}

func (m *listModal) view(a *App, w, _ int) string {
	lines := []string{a.th.Title.Render(" " + m.title), ""}
	if len(m.rows) == 0 {
		lines = append(lines, " "+a.th.Dim.Render(m.empty))
	}
	for i, r := range m.rows {
		mark := " "
		if i == m.cur {
			mark = a.gl.Sel
		}
		lines = append(lines, a.selIf(i == m.cur, "%s %s", mark, truncate(r, max(w-3, 1), a.gl.Ellipsis)))
	}
	if m.note != "" {
		lines = append(lines, "", " "+a.th.Warn.Render(m.note))
	}
	h := m.act.Help()
	btn := " [" + h.Key + "] " + h.Desc + "   [esc] close"
	if len(m.rows) == 0 {
		btn = " [esc] close"
	}
	return strings.Join(append(lines, "", btn), "\n")
}

func newHostKeysModal(id string, recs []api.RecordedHostKeys) *listModal {
	type row struct {
		hp string
		k  api.RecordedHostKey
	}
	var rs []row
	m := &listModal{title: "Host keys recorded for " + sanitizeLine(id), empty: "no host key recorded for this box",
		act: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "forget, typed confirmation"))}
	for _, r := range recs {
		for _, k := range r.Keys {
			rs = append(rs, row{r.HostPort, k})
			m.rows = append(m.rows, sanitizeLine(r.HostPort)+"  "+sanitizeLine(k.KeyType)+"  "+sanitizeLine(k.Fingerprint)+"  "+sanitizeLine(string(k.Store)))
		}
	}
	m.why = func(i int) string {
		if rs[i].k.Store != api.HostKeyStoreJumpgate {
			return "this key is in OpenSSH's known_hosts, which jumpgate never edits (ssh-keygen -R)"
		}
		return ""
	}
	m.run = func(a *App, i int) {
		hp, fp := rs[i].hp, rs[i].k.Fingerprint
		a.modal = newConfirm("Forget "+fp+" for "+hp,
			"The next connection to "+hp+" asks you to confirm its key again.", "forget", func() tea.Cmd {
				return a.do("forget", func(ctx context.Context, be Backend) tea.Msg {
					err := be.Do(ctx, http.MethodPost, "/api/hostkeys/forget", api.HostKeyForget{HostPort: hp, Fingerprint: fp}, nil)
					return forgotMsg{hostPort: hp, fp: fp, err: err}
				})
			})
	}
	return m
}

func newOrphansModal(os []orphan) *listModal {
	m := &listModal{title: "Leftover containers", empty: "none: no gateway merge left a container behind",
		act: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "dismiss, typed confirmation")),
		why: func(int) string { return "" }}
	for _, o := range os {
		m.rows = append(m.rows, sanitizeLine(o.ContainerName)+" on "+sanitizeLine(o.TargetID)+", merged into "+sanitizeLine(o.MergedInto))
	}
	m.run = func(a *App, i int) {
		name, on := os[i].ContainerName, os[i].TargetID
		a.modal = newConfirm("Dismiss "+name,
			"This forgets the record only: the container keeps running on "+on+" until you wipe it there.", name, func() tea.Cmd {
				return a.do("dismiss", func(ctx context.Context, be Backend) tea.Msg {
					return dismissedMsg{name: name, err: be.Do(ctx, http.MethodDelete, "/api/orphans/"+url.PathEscape(name), nil, nil)}
				})
			})
	}
	return m
}
