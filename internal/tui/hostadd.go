package tui

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

type addStep int

const (
	stepForm addStep = iota
	stepProbing
	stepConfirm
	stepPairing
	stepDone
	stepFailed
)

// Every message carries the generation of the flow that asked, so a reply
// that lands after the flow was left and another begun is dropped.
type (
	probeMsg struct {
		gen uint64
		p   api.HostKeyProbe
		err error
	}
	hostKeyConfirmedMsg struct {
		gen uint64
		err error
	}
	targetAddedMsg struct {
		gen uint64
		err error
	}
	pairMsg struct {
		gen    uint64
		ev     api.PairEvent
		ch     <-chan api.PairEvent
		err    error
		closed bool
	}
)

const maxPairLines = 200

var (
	formLabels = []string{"name", "address (user@host[:port])", "key file (empty: ssh-agent)", "jump host (optional)", "use sudo (y/n)"}
	nameRE     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
)

// addFlow is the add-and-pair conversation: a form, then each unconfirmed
// host key on the path (jump first) typed "yes" by the person, then the
// pairing stream. Nothing is trusted or recorded before its own step, and a
// key is only ever confirmed by the probe ID and fingerprint the person was
// shown.
type addFlow struct {
	gen    uint64
	ctx    context.Context
	cancel context.CancelFunc

	step    addStep
	fields  []textinput.Model
	focus   int
	errLine string
	name    string
	addr    api.SSHView
	sudo    bool
	repair  bool // pair an existing target: no form, no AddTarget
	hop     *api.HostKeyHop
	cm      *confirmModal // the typed "yes" for the key on screen
	asked   bool          // cm was answered yes and ConfirmHostKey is in flight
	adding  bool          // AddTarget is in flight
	started bool          // Pair returned a channel: the server is pairing
	lines   []string
	agent   string
}

func newAddFlow() *addFlow {
	f := &addFlow{}
	for range formLabels {
		in := textinput.New()
		in.Prompt = ""
		in.CharLimit = 256
		// ctrl+v would run the system clipboard tool; a terminal's bracketed
		// paste still fills the field, and is sanitized on its way in.
		in.KeyMap.Paste.SetEnabled(false)
		f.fields = append(f.fields, in)
	}
	f.fields[4].SetValue("n")
	return f
}

func newRepairFlow(id string, addr api.SSHView) *addFlow {
	f := newAddFlow()
	f.name, f.addr, f.repair, f.step = id, addr, true, stepProbing
	return f
}

func (f *addFlow) focusCmd() tea.Cmd { return f.fields[f.focus].Focus() }

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}

// submit validates the form and starts probing.
func (f *addFlow) submit(a *App) tea.Cmd {
	name := strings.TrimSpace(f.fields[0].Value())
	if !nameRE.MatchString(name) {
		f.errLine = "the name is lower-case letters, digits and dashes, up to 32"
		return nil
	}
	addr, err := api.ParseLogin(strings.TrimSpace(f.fields[1].Value()))
	if err != nil {
		f.errLine = "address: " + sanitizeLine(err.Error())
		return nil
	}
	if k := strings.TrimSpace(f.fields[2].Value()); k != "" {
		addr.KeyPath = expandHome(k)
	}
	if j := strings.TrimSpace(f.fields[3].Value()); j != "" {
		jump, err := api.ParseLogin(j)
		if err != nil {
			f.errLine = "jump host: " + sanitizeLine(err.Error())
			return nil
		}
		jump.KeyPath = addr.KeyPath
		addr.Jump = &jump
	}
	sudo := strings.ToLower(strings.TrimSpace(f.fields[4].Value()))
	f.name, f.addr, f.sudo, f.errLine = name, addr, sudo == "y" || sudo == "yes", ""
	return f.probe(a)
}

// run is a.do on the flow's own context, so leaving the flow stops its work.
func (f *addFlow) run(what string, fn func(ctx context.Context) tea.Msg) tea.Cmd {
	ctx := f.ctx
	return func() tea.Msg { return fn(ctx) }
}

func (f *addFlow) probe(a *App) tea.Cmd {
	f.step, f.hop, f.cm, f.asked = stepProbing, nil, nil, false
	addr, gen, be := f.addr, f.gen, a.be
	return f.run("probe", func(ctx context.Context) tea.Msg {
		p, err := be.ProbeHostKeys(ctx, addr)
		return probeMsg{gen: gen, p: p, err: err}
	})
}

func (f *addFlow) fail(text string) {
	f.step, f.errLine, f.hop, f.cm = stepFailed, text, nil, nil
}

// askHop shows one unconfirmed key and asks for the typed word. The
// fingerprint shown is the fingerprint sent: a key whose fields the screen
// would have to alter to show is refused, never confirmed.
func (f *addFlow) askHop(a *App, hop *api.HostKeyHop) {
	if hop.ProbeID == "" || hop.Fingerprint == "" || sanitizeLine(hop.Fingerprint) != hop.Fingerprint ||
		sanitizeLine(hop.ProbeID) != hop.ProbeID || sanitizeLine(hop.KeyType) != hop.KeyType {
		f.fail("the server sent a host key this screen cannot show faithfully for " + sanitizeLine(hop.HostPort) + "; nothing was trusted. Use `jumpgate hosts add` in a terminal")
		return
	}
	h := *hop // what is shown and what is sent are this one copy
	detail := fmt.Sprintf("%s\n\n   %s  %s\n\n"+
		" Compare it with the box's console:\n   ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub",
		f.whose(h.HostPort), h.KeyType, h.Fingerprint)
	f.step, f.hop = stepConfirm, &h
	be, gen := a.be, f.gen
	f.cm = newConfirm("unconfirmed host key", detail, "yes", func() tea.Cmd {
		f.asked = true
		return f.run("confirm", func(ctx context.Context) tea.Msg {
			return hostKeyConfirmedMsg{gen: gen, err: be.ConfirmHostKey(ctx, h.ProbeID, h.Fingerprint)}
		})
	})
	f.cm.prompt = "Type %s to trust it, esc to stop:"
}

// whose says which machine a host key is for: the jump host on the way to
// the box, or the box itself.
func (f *addFlow) whose(hostPort string) string {
	name := sanitizeLine(f.name)
	if j := f.addr.Jump; j != nil {
		port := j.Port
		if port == 0 {
			port = 22
		}
		if hostPort == net.JoinHostPort(j.Host, strconv.Itoa(port)) {
			return sanitizeLine(hostPort) + " is the jump host for " + name
		}
	}
	return sanitizeLine(hostPort) + " is " + name
}

// update runs one step; done means the flow is over and the list returns.
func (f *addFlow) update(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case probeMsg:
		if msg.gen != f.gen || f.step != stepProbing {
			return nil, false
		}
		if msg.err != nil {
			f.fail("probe: " + a.errText(msg.err))
			return nil, false
		}
		hop := msg.p.Pending()
		switch {
		case msg.p.AllConfirmed && hop == nil:
			return f.addAndPair(a), false
		case hop == nil:
			// Neither confirmed nor a pending hop: trust nothing.
			f.fail("the server's answer named no host key to confirm; nothing was trusted")
		case hop.State == api.HostKeyMismatch:
			f.fail(fmt.Sprintf("SECURITY: %s presents a host key that does NOT match the one on record. It may be a different machine, or someone in between. %s %s Nothing was changed and this screen offers no way to accept it.",
				sanitizeLine(hop.HostPort), a.gl.Dash, sanitizeLine(api.HintFor(api.CodeHostKey))))
		case hop.State == api.HostKeyUnreachable:
			f.fail(fmt.Sprintf("cannot reach %s: %s", sanitizeLine(hop.HostPort), sanitizeLine(hop.Error)))
		case hop.State == api.HostKeyUnknown:
			f.askHop(a, hop)
		default:
			f.fail("unexpected host key state " + sanitizeLine(string(hop.State)) + "; nothing was trusted")
		}
		return nil, false
	case hostKeyConfirmedMsg:
		if msg.gen != f.gen || !f.asked {
			return nil, false
		}
		f.asked = false
		if msg.err != nil {
			f.fail("confirm: " + a.errText(msg.err))
			return nil, false
		}
		return f.probe(a), false // the next hop, or the same host if it still is not recorded: ask again
	case targetAddedMsg:
		if msg.gen != f.gen || !f.adding {
			return nil, false
		}
		f.adding = false
		var e *api.Error
		// An existing name is re-paired, as `jumpgate hosts add` does.
		if msg.err != nil && !(errors.As(msg.err, &e) && e.Code == api.CodeTargetExists) {
			f.fail("add: " + a.errText(msg.err))
			return nil, false
		}
		return f.pair(a), false
	case pairMsg:
		if msg.gen != f.gen || f.step != stepPairing {
			return nil, false
		}
		if msg.ch != nil {
			f.started = true
		}
		switch {
		case msg.err != nil:
			f.fail("pair: " + a.errText(msg.err))
		case msg.closed:
			f.fail("the server ended the pairing before it finished")
		case msg.ev.Done:
			f.step, f.agent = stepDone, sanitizeLine(msg.ev.Agent)
			f.cancel()
		case msg.ev.Error != "":
			hint := sanitizeLine(msg.ev.Hint)
			if hint == "" {
				hint = sanitizeLine(api.HintFor(msg.ev.Code))
			}
			f.fail(fmt.Sprintf("pairing failed at %s: %s %s %s", sanitizeLine(msg.ev.Step), sanitizeLine(msg.ev.Error), a.gl.Dash, hint))
			f.cancel()
		default:
			f.lines = append(f.lines, fmt.Sprintf("[%s] %s", sanitizeLine(msg.ev.Step), sanitizeLine(msg.ev.Line)))
			if len(f.lines) > maxPairLines {
				f.lines = f.lines[len(f.lines)-maxPairLines:]
			}
			return waitPair(f.ctx, f.gen, msg.ch), false
		}
		return nil, false
	case tea.PasteMsg:
		p := tea.PasteMsg{Content: sanitizeLine(msg.Content)}
		switch f.step {
		case stepForm:
			var cmd tea.Cmd
			f.fields[f.focus], cmd = f.fields[f.focus].Update(p)
			return cmd, false
		case stepConfirm:
			return f.cm.paste(a, p), false
		}
	case tea.KeyPressMsg:
		return f.key(a, msg)
	}
	return nil, false
}

func (f *addFlow) key(a *App, k tea.KeyPressMsg) (tea.Cmd, bool) {
	switch f.step {
	case stepForm:
		switch k.String() {
		case "esc":
			return nil, true
		case "enter", "tab", "down":
			if f.focus == len(f.fields)-1 && k.String() == "enter" {
				return f.submit(a), false
			}
			f.fields[f.focus].Blur()
			f.focus = (f.focus + 1) % len(f.fields)
			return f.focusCmd(), false
		case "shift+tab", "up":
			f.fields[f.focus].Blur()
			f.focus = (f.focus + len(f.fields) - 1) % len(f.fields)
			return f.focusCmd(), false
		}
		// Every other key is text, whatever it would do outside the field.
		var cmd tea.Cmd
		f.fields[f.focus], cmd = f.fields[f.focus].Update(k)
		return cmd, false
	case stepConfirm:
		// The typed confirmation owns the keys: only the word "yes", then
		// enter, trusts the key on screen. Enter alone, or anything else,
		// asks again; esc stops with nothing trusted.
		done, cmd := f.cm.key(a, k)
		if done && !f.asked {
			return nil, true
		}
		if done {
			f.step, f.cm = stepProbing, nil
		}
		return cmd, false
	case stepDone, stepFailed:
		if k.String() == "esc" || k.String() == "enter" {
			return nil, true
		}
	case stepProbing, stepPairing:
		if k.String() == "esc" {
			a.flash = f.leaveNote(a)
			return nil, true
		}
	}
	return nil, false
}

// leaveNote says only what is true when the person leaves mid-flow. The
// server pairs on a context the client cannot cancel, so once Pair has
// returned a channel the pairing finishes there. Before that nothing has
// been paired, and a box added by this flow may be left unpaired.
func (f *addFlow) leaveNote(a *App) string {
	if f.started {
		return "stopped watching; a pairing already under way finishes on the server"
	}
	note := "stopped; no pairing was started"
	if !f.repair && (f.adding || f.step == stepPairing) {
		note += " " + a.gl.Dash + " the box may have been added unpaired: check the Hosts list"
	}
	return note
}

func (f *addFlow) addAndPair(a *App) tea.Cmd {
	f.step = stepPairing
	if f.repair {
		return f.pair(a)
	}
	f.adding = true
	req := api.AddTarget{ID: f.name, Mode: "ssh", SSH: &f.addr}
	gen, be := f.gen, a.be
	return f.run("add", func(ctx context.Context) tea.Msg { return targetAddedMsg{gen: gen, err: be.AddTarget(ctx, req)} })
}

func (f *addFlow) pair(a *App) tea.Cmd {
	f.step = stepPairing
	name, sudo, gen, be := f.name, f.sudo, f.gen, a.be
	return f.run("pair", func(ctx context.Context) tea.Msg {
		ch, err := be.Pair(ctx, name, api.PairRequest{Sudo: sudo})
		if err != nil {
			return pairMsg{gen: gen, err: err}
		}
		return waitPair(ctx, gen, ch)()
	})
}

// waitPair reads the next pairing event. It ends with the flow's context
// whether or not the backend closes the channel.
func waitPair(ctx context.Context, gen uint64, ch <-chan api.PairEvent) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return pairMsg{gen: gen, ch: ch, closed: true}
		case ev, ok := <-ch:
			if !ok {
				return pairMsg{gen: gen, ch: ch, closed: true}
			}
			return pairMsg{gen: gen, ev: ev, ch: ch}
		}
	}
}

func (f *addFlow) view(a *App, w, h int) string {
	title := " " + a.th.Title.Render("ADD A BOX")
	if f.repair {
		title = " " + a.th.Title.Render("PAIR "+sanitizeLine(f.name))
	}
	if f.step == stepConfirm && f.cm != nil {
		return title + "\n\n" + f.cm.view(a, w, h)
	}
	var b strings.Builder
	b.WriteString(title + "\n\n")
	switch f.step {
	case stepForm:
		for i, l := range formLabels {
			mark := " "
			if i == f.focus {
				mark = a.gl.Sel
			}
			fmt.Fprintf(&b, " %s %-28s %s\n", mark, l, f.fields[i].View())
		}
		b.WriteString("\n " + a.th.Dim.Render("enter next field, enter on the last to continue, esc cancel") + "\n")
	case stepProbing:
		b.WriteString(" reading the host keys of " + sanitizeLine(f.addr.Address()) + a.gl.Ellipsis + "\n")
	case stepPairing, stepDone, stepFailed:
		lines := f.lines
		if keep := max(h-10, 3); len(lines) > keep {
			lines = lines[len(lines)-keep:]
		}
		for _, l := range lines {
			b.WriteString(" " + l + "\n")
		}
		switch f.step {
		case stepDone:
			b.WriteString("\n " + a.th.OK.Render("paired: agent "+f.agent) + "\n")
			b.WriteString(" Recommended now: disable root SSH login on the box (PermitRootLogin no).\n Keep console access as the way back in. (enter to close)\n")
		case stepPairing:
			b.WriteString(" " + a.th.Dim.Render("pairing"+a.gl.Ellipsis) + "\n")
		}
	}
	if f.errLine != "" {
		style := a.th.Warn
		if f.step == stepFailed {
			style = a.th.Bad
		}
		b.WriteString("\n " + style.Render(f.errLine) + "\n")
		if f.step == stepFailed {
			b.WriteString("\n " + a.th.Dim.Render("enter or esc to close") + "\n")
		}
	}
	return b.String()
}
