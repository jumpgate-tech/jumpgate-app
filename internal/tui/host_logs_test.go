package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func hit(sec int, unit, sev, line string) api.LogHit {
	return api.LogHit{At: testNow.Add(time.Duration(sec) * time.Second), Unit: unit, Severity: sev, Line: line}
}

func liveLogs(hits ...api.LogHit) apiclient.Update[[]api.LogHit] {
	return apiclient.Update[[]api.LogHit]{Has: true, State: apiclient.Live, Value: hits}
}

// logsHost opens box on its Logs tab with a backlog and one live line.
func logsHost(t *testing.T, f *tuitest.Fake) *App {
	t.Helper()
	ch := f.Logs("box")
	ch <- apiclient.Update[[]api.LogHit]{Has: true, Reset: true, State: apiclient.Live, Value: []api.LogHit{
		hit(1, "valve-node-app-exec.service", "info", "Imported new chain segment number=21345678"),
		hit(2, "valve-node-app-beacon.service", "warn", "Low peer count peers=3"),
		hit(3, "valve-node-app-exec.service", "error", "peer 203.0.113.7 disconnected: too many peers"),
	}}
	ch <- liveLogs(hit(4, "valve-node-app-beacon.service", "critical", "Database corrupted"))
	return tab(t, openTestHost(t, f, "unicode"), 4)
}

func host(m tea.Model) *hostScreen { return m.(*App).detail.(*hostScreen) }

func TestLogsTabStreams(t *testing.T) {
	f := tuitest.NewFake()
	a := logsHost(t, f)
	if !f.Called("WatchLogs box 500") {
		t.Fatalf("calls %v", f.Calls)
	}
	fr := tuitest.Frame(a)
	for _, want := range []string{"12:00:01 exec", "beacon", "WARN", "ERROR", "CRIT", "Database corrupted", "4 lines", "follow on"} {
		if !strings.Contains(fr, want) {
			t.Errorf("logs lack %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "host_logs", fr)
}

func TestLogsFilterAndLevel(t *testing.T) {
	a := logsHost(t, tuitest.NewFake())
	m, _ := tuitest.Send(a, tuitest.Key("/"))
	if !m.(*App).current().capturing() {
		t.Fatal("the filter does not take the keyboard")
	}
	m, _ = tuitest.Send(m, tuitest.Type("peer")...)
	m, _ = tuitest.Send(m, tuitest.Key("enter"))
	h := host(m)
	if n := len(h.visibleLogs()); n != 2 {
		t.Fatalf("filter peer: %d lines", n)
	}
	m, _ = tuitest.Send(m, tuitest.Key("!"), tuitest.Key("!"))
	if n := len(h.visibleLogs()); n != 1 {
		t.Fatalf("peer at error and above: %d lines", n)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, `filter "peer"`) || !strings.Contains(fr, "error and above") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// While the filter has the keyboard, f, e, ! and q are text, not commands.
func TestLogsFilterTakesActionKeysAsText(t *testing.T) {
	a := logsHost(t, tuitest.NewFake())
	m, _ := tuitest.Send(a, tuitest.Key("/"))
	m, cmds := tuitest.Send(m, tuitest.Type("fe!q")...)
	h := host(m)
	if got := h.logFilter.Value(); got != "fe!q" {
		t.Fatalf("filter holds %q", got)
	}
	if !h.follow || h.logMin != 0 || m.(*App).modal != nil || len(cmds) != 0 {
		t.Fatalf("an action key fired: follow=%v min=%d modal=%v cmds=%d", h.follow, h.logMin, m.(*App).modal, len(cmds))
	}
}

// ctrl+v would read the system clipboard: the field must not ask for it.
func TestLogsFilterDoesNotReadTheClipboard(t *testing.T) {
	a := logsHost(t, tuitest.NewFake())
	m, _ := tuitest.Send(a, tuitest.Key("/"))
	_, cmds := tuitest.Send(m, tuitest.Key("ctrl+v"))
	if len(cmds) != 0 {
		t.Fatalf("ctrl+v returned %d commands", len(cmds))
	}
	if newLogFilter().KeyMap.Paste.Enabled() {
		t.Fatal("paste is enabled")
	}
}

// A terminal paste reaches the filter, cleaned of escapes and line breaks.
func TestLogsFilterPasteIsCleaned(t *testing.T) {
	a := logsHost(t, tuitest.NewFake())
	m, _ := tuitest.Send(a, tuitest.Key("/"))
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: "pe\x1b]52;c;QQ==\x07er\r\n\u202e"})
	if got := host(m).logFilter.Value(); got != "peer " { // the line break became a space
		t.Fatalf("filter holds %q", got)
	}
}

// The filter is literal text: regexp metacharacters match themselves.
func TestLogsFilterIsLiteral(t *testing.T) {
	f := tuitest.NewFake()
	f.Logs("box") <- liveLogs(hit(1, "x-exec.service", "info", "a.c matched"), hit(2, "x-exec.service", "info", "abc nope"), hit(3, "x-exec.service", "info", "ERR (x"))
	a := tab(t, openTestHost(t, f, "unicode"), 4)
	for filter, want := range map[string]int{"a.c": 1, ".*": 0, "(x": 1, "[": 0, "err": 1, "A.C": 1} {
		h := host(a)
		h.logFilter.SetValue(filter)
		if n := len(h.visibleLogs()); n != want {
			t.Errorf("filter %q: %d lines, want %d", filter, n, want)
		}
	}
}

func TestLogsKeepAtMostLogKeepAndResetClears(t *testing.T) {
	f := tuitest.NewFake()
	a := tab(t, openTestHost(t, f, "unicode"), 4)
	h := host(a)
	var many []api.LogHit
	for i := 0; i < logKeep+500; i++ {
		many = append(many, hit(i, "x-exec.service", "info", fmt.Sprintf("line %d", i)))
	}
	h.applyLogs(liveLogs(many[:1500]...))
	h.applyLogs(liveLogs(many[1500:]...))
	if len(h.logs) != logKeep || h.logs[len(h.logs)-1].Line != fmt.Sprintf("line %d", logKeep+499) || h.logs[0].Line != "line 500" {
		t.Fatalf("%d lines, first %q last %q", len(h.logs), h.logs[0].Line, h.logs[len(h.logs)-1].Line)
	}
	h.applyLogs(liveLogs(many...)) // one update bigger than the buffer
	if len(h.logs) != logKeep {
		t.Fatalf("%d lines after an oversize update", len(h.logs))
	}
	h.logBack = 7
	h.applyLogs(apiclient.Update[[]api.LogHit]{Has: true, Reset: true, State: apiclient.Live, Value: many[:2]})
	if len(h.logs) != 2 || h.logBack != 0 {
		t.Fatalf("a reset left %d lines, back %d", len(h.logs), h.logBack)
	}
	h.applyLogs(apiclient.Update[[]api.LogHit]{State: apiclient.Live, Has: true, Reset: true})
	if len(h.logs) != 0 {
		t.Fatal("an empty reset kept lines")
	}
}

func TestLogsShowTheServersNote(t *testing.T) {
	f := tuitest.NewFake()
	f.Logs("box") <- apiclient.Update[[]api.LogHit]{State: apiclient.Live, Note: "snapshot mode: re-pair the box to upgrade it"}
	a := tab(t, openTestHost(t, f, "unicode"), 4)
	if fr := tuitest.Frame(a); !strings.Contains(fr, "snapshot mode: re-pair") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// explain presses e, then enter to send after the disclosure is on screen.
func explain(t *testing.T, a *App) tea.Model {
	t.Helper()
	m, cmds := tuitest.Send(a, tuitest.Key("e"))
	m = tuitest.Settle(m, cmds...)
	m, cmds = tuitest.Send(m, tuitest.Key("enter"))
	return tuitest.Settle(m, cmds...)
}

func TestExplainShowsTheDisclosureThenSendsTheVisibleErrors(t *testing.T) {
	f := tuitest.NewFake()
	f.ExplainV = api.Explain{Text: "The node lost its peers.", SentExcerpt: []string{"peer <ip-1> disconnected"}, Redacted: true}
	f.SettingsV = api.Settings{AIProvider: "gemini", AIDisclosure: "Explain sends the selected log lines to Gemini after removing IP addresses."}
	a := logsHost(t, f)
	m, cmds := tuitest.Send(a, tuitest.Key("e"))
	m = tuitest.Settle(m, cmds...)
	if f.Called("Explain") {
		t.Fatal("explain sent before the person confirmed")
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "Gemini") || !strings.Contains(fr, "2 error lines") {
		t.Fatalf("no disclosure before sending:\n%s", fr)
	}
	m, cmds = tuitest.Send(m, tuitest.Key("enter"))
	m = tuitest.Settle(m, cmds...)
	if !f.Called("Explain box 2") {
		t.Fatalf("calls %v", f.Calls)
	}
	fr := tuitest.Frame(m)
	for _, want := range []string{"The node lost its peers.", "sent 1 line", "peer ids removed", "peer <ip-1> disconnected", "Gemini"} {
		if !strings.Contains(fr, want) {
			t.Errorf("explain lacks %q:\n%s", want, fr)
		}
	}
}

func TestExplainEscCancelsBeforeSending(t *testing.T) {
	f := tuitest.NewFake()
	f.SettingsV = api.Settings{AIDisclosure: "d"}
	a := logsHost(t, f)
	m, cmds := tuitest.Send(a, tuitest.Key("e"))
	m = tuitest.Settle(m, cmds...)
	m, _ = tuitest.Send(m, tuitest.Key("esc"))
	if m.(*App).modal != nil || f.Called("Explain") {
		t.Fatal("esc did not cancel")
	}
}

func TestExplainWithoutADisclosureSendsNothing(t *testing.T) {
	f := tuitest.NewFake()
	f.Err["Settings"] = &api.Error{Message: "boom", Code: api.CodeInternal}
	a := logsHost(t, f)
	m := explain(t, a)
	if f.Called("Explain") || !strings.Contains(tuitest.Frame(m), "nothing is sent") {
		t.Fatalf("calls %v\n%s", f.Calls, tuitest.Frame(m))
	}
}

// What leaves the machine is bounded: the newest explainMax lines, each cut.
func TestExplainSendsAtMostTheNewestLines(t *testing.T) {
	f := tuitest.NewFake()
	f.SettingsV = api.Settings{AIDisclosure: "d"}
	f.ExplainV = api.Explain{Text: "ok"}
	var hits []api.LogHit
	for i := 0; i < 1200; i++ {
		hits = append(hits, hit(i, "x-exec.service", "error", strings.Repeat("e", 3000)))
	}
	f.Logs("box") <- liveLogs(hits...)
	a := tab(t, openTestHost(t, f, "unicode"), 4)
	explain(t, a)
	// 1200 lines of 1500 bytes do not fit the body budget: the newest that do go.
	want := explainBodyMax / (explainLineMax + 3)
	if !f.Called(fmt.Sprintf("Explain box %d", want)) {
		t.Fatalf("calls %v", f.Calls)
	}
	if explainMax*explainLineMax >= 1<<20 || explainMax >= 5000 {
		t.Fatal("explain can exceed what the server accepts")
	}
	if got := len((&hostScreen{logs: []api.LogHit{{Severity: "error", Line: strings.Repeat("e", 3000)}}}).explainLines()[0]); got != explainLineMax {
		t.Fatalf("line sent at %d bytes", got)
	}
}

func TestExplainErrorsAreHonest(t *testing.T) {
	for _, tc := range []struct {
		err  *api.Error
		want []string
	}{
		{&api.Error{Status: 413, Code: api.CodeTooManyLines, Message: "too many lines", Hint: "send fewer"}, []string{"too much to send", "send fewer"}},
		{&api.Error{Status: 413, Code: api.CodeTooLarge, Message: "too large"}, []string{"too much to send"}},
		{&api.Error{Status: 502, Code: api.CodeUpstream, Message: "provider said no", Hint: "check the key"}, []string{"the AI provider failed", "check the key"}},
		{&api.Error{Status: 400, Code: api.CodeAIUnconfigured, Message: "no provider", Hint: "set one in Settings"}, []string{"no AI provider is set up", "set one in Settings"}},
	} {
		f := tuitest.NewFake()
		f.SettingsV = api.Settings{AIDisclosure: "d"}
		f.Err["Explain"] = tc.err
		m := explain(t, logsHost(t, f))
		fr := tuitest.Frame(m)
		for _, w := range tc.want {
			if !strings.Contains(fr, w) {
				t.Errorf("%s: frame lacks %q:\n%s", tc.err.Code, w, fr)
			}
		}
	}
}

// A late logs or explain reply from a closed open must not reach the reopened box.
func TestLogsAndExplainRepliesFromAnOldOpenAreDropped(t *testing.T) {
	f := tuitest.NewFake()
	a := newTestApp(t, f, 80, 24, "unicode")
	a.openHost("box")
	old := host(a)
	a.closeDetail()
	a.openHost("box")
	cur := host(a)
	stale := make(chan apiclient.Update[[]api.LogHit])
	cur.tab = 4
	m, cmds := tuitest.Send(a,
		logsMsg{id: "box", gen: old.gen, u: liveLogs(hit(1, "x-exec.service", "error", "old")), ch: stale},
		disclosureMsg{id: "box", gen: old.gen, disclosure: "old"},
		explainMsg{id: "box", gen: old.gen, e: api.Explain{Text: "old"}})
	if len(cmds) != 0 {
		t.Fatalf("a stale message produced %d commands", len(cmds))
	}
	if h := host(m); len(h.logs) != 0 {
		t.Fatalf("the reopened host took the old open's lines: %+v", h.logs)
	}
	// The reopened box's own streams use its context: leaving ends them.
	cur.cancel()
	if msgs := tuitest.Run(cur.watchLogs(f.Logs("box"))); len(msgs) != 0 {
		t.Fatalf("a watch outlived its box: %v", msgs)
	}
}

var hostileLog = []string{
	"\x1b[2J\x1b[Hfake\x1b]0;title\x07 prompt",
	"\x1b]8;;http://evil.example\x07click\x1b]8;;\x07",
	"\x1b]52;c;QUJD\x07clip",
	"a\u202eevil\u2066x\u2069\r\roverwrite",
	"line one\nline two\x9b31m\x1bP+q\x1b\\ tail " + strings.Repeat("W", 400),
}

func TestLogsAndExplainSurviveHostileText(t *testing.T) {
	f := tuitest.NewFake()
	var hits []api.LogHit
	for i, l := range hostileLog {
		hits = append(hits, hit(i, "x-exec"+l+".service", "error", l))
	}
	f.Logs("box") <- apiclient.Update[[]api.LogHit]{Has: true, State: apiclient.Live, Note: hostileLog[0], Value: hits}
	f.SettingsV = api.Settings{AIDisclosure: hostileLog[1]}
	f.ExplainV = api.Explain{Text: strings.Join(hostileLog, "\n") + "\n" + strings.Repeat("long ", 200), SentExcerpt: hostileLog, Redacted: true}
	a := tab(t, openTestHost(t, f, "unicode"), 4)
	a.th = plainTheme()
	check := func(what string, m tea.Model) {
		t.Helper()
		c := m.View().Content
		if n := strings.Count(c, "\n") + 1; n != 24 {
			t.Fatalf("%s: %d lines, want 24:\n%s", what, n, c)
		}
		for _, bad := range []string{"\x1b", "\x9b", "\r", "\x07", "\u202e", "\u2066", "\u2069"} {
			if strings.Contains(c, bad) {
				t.Fatalf("%s: frame holds %q:\n%q", what, bad, c)
			}
		}
		for i, l := range strings.Split(c, "\n") {
			if w := len([]rune(l)); w > 80 {
				t.Fatalf("%s: line %d is %d wide: %q", what, i, w, l)
			}
		}
	}
	check("logs", a)
	for _, k := range []string{"!", "!", "f", "up"} {
		m, _ := tuitest.Send(a, tuitest.Key(k))
		check("logs after "+k, m)
	}
	m, cmds := tuitest.Send(a, tuitest.Key("e"))
	m = tuitest.Settle(m, cmds...)
	check("explain asking", m)
	m, cmds = tuitest.Send(m, tuitest.Key("enter"))
	m = tuitest.Settle(m, cmds...)
	check("explain answer", m)
	if fr := tuitest.Frame(m); !strings.Contains(fr, "fake") {
		t.Fatalf("the visible text was lost:\n%s", fr)
	}
	// An error from the provider is as untrusted as its answer.
	f.Err["Explain"] = &api.Error{Status: 502, Message: hostileLog[0], Hint: hostileLog[2]}
	m, cmds = tuitest.Send(a, tuitest.Key("e"))
	m = tuitest.Settle(m, cmds...)
	m, cmds = tuitest.Send(m, tuitest.Key("enter"))
	check("explain error", tuitest.Settle(m, cmds...))
	// And the filter text itself.
	m, _ = tuitest.Send(a, tuitest.Key("/"))
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: hostileLog[0]})
	m, _ = tuitest.Send(m, tuitest.Key("enter"))
	check("hostile filter", m)
}

// The request is budgeted by its JSON-encoded size: "<" encodes to 6 bytes.
func TestExplainBodyStaysUnderTheServersLimit(t *testing.T) {
	for name, ch := range map[string]string{"angle": "<", "quote": `"`, "backslash": `\`, "ampersand": "&", "ls": "\u2028", "multibyte": "\u00e9"} {
		h := &hostScreen{}
		for i := 0; i < 600; i++ {
			l := fmt.Sprintf("%04d", i) + strings.Repeat(ch, 1700)
			h.logs = append(h.logs, api.LogHit{Severity: "error", Line: l})
		}
		lines := h.explainLines()
		body, err := json.Marshal(struct {
			Lines []string `json:"lines"`
		}{lines})
		if err != nil || len(body) >= 1<<20 || len(body) > explainBodyMax+64 {
			t.Errorf("%s: body %d bytes (err %v)", name, len(body), err)
		}
		if len(lines) == 0 || !strings.HasPrefix(lines[len(lines)-1], "0599") || len(lines) > explainMax {
			t.Errorf("%s: %d lines; the newest must be kept", name, len(lines))
		}
		for _, l := range lines {
			if !utf8.ValidString(l) {
				t.Fatalf("%s: a line was cut inside a rune", name)
			}
		}
	}
}

func scrollHost(t *testing.T, n int) (*hostScreen, func(...api.LogHit)) {
	h := &hostScreen{follow: true, logFilter: newLogFilter()}
	var hits []api.LogHit
	for i := 0; i < n; i++ {
		hits = append(hits, hit(i, "x-exec.service", "info", fmt.Sprintf("line %d", i)))
	}
	h.applyLogs(liveLogs(hits...))
	return h, func(l ...api.LogHit) { h.applyLogs(liveLogs(l...)) }
}

func logWindow(a *App, h *hostScreen) string {
	var out []string
	for _, l := range strings.Split(h.viewLogs(a, 80, 12), "\n") {
		if strings.Contains(l, "line ") {
			out = append(out, strings.TrimSpace(l))
		}
	}
	return strings.Join(out, "|")
}

func TestLogsFollowOffFreezesTheView(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	h, add := scrollHost(t, 30)
	h.follow, h.logBack = false, 3
	before := logWindow(a, h)
	add(hit(31, "x-exec.service", "info", "line 31"), hit(32, "x-exec.service", "info", "line 32"), hit(33, "x-exec.service", "info", "line 33"), hit(34, "x-exec.service", "info", "line 34"))
	if after := logWindow(a, h); after != before {
		t.Fatalf("scrolled back, lines arrived:\n%s\n%s", before, after)
	}
	// Lines the filter hides do not move the window; visible ones do count.
	h.logFilter.SetValue("line")
	h.logBack = 3
	before = logWindow(a, h)
	add(hit(40, "x-exec.service", "info", "other"), hit(41, "x-exec.service", "info", "line 41"))
	if after := logWindow(a, h); after != before {
		t.Fatalf("filtered:\n%s\n%s", before, after)
	}
	h.logFilter.SetValue("")
	// Off at the bottom: frozen too.
	h, add = scrollHost(t, 30)
	h.follow, h.logBack = false, 0
	before = logWindow(a, h)
	add(hit(31, "x-exec.service", "info", "line 31"))
	if after := logWindow(a, h); after != before {
		t.Fatalf("off at the bottom moved:\n%s\n%s", before, after)
	}
	// On: jumps to the newest.
	h.follow = true
	if after := logWindow(a, h); !strings.HasSuffix(after, "line 31") {
		t.Fatalf("follow on: %s", after)
	}
}

func TestLogsFrozenViewSurvivesTrimming(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	h, add := scrollHost(t, logKeep)
	h.follow, h.logBack = false, logKeep-2
	for i := 0; i < 5; i++ {
		var more []api.LogHit
		for j := 0; j < 700; j++ {
			more = append(more, hit(j, "x-exec.service", "info", "line x"))
		}
		add(more...)
		_ = logWindow(a, h) // must not panic
		if h.logBack > len(h.visibleLogs()) {
			t.Fatalf("logBack %d beyond %d lines", h.logBack, len(h.visibleLogs()))
		}
	}
}

func TestExplainWithNoErrorLinesSaysTheServerFetchesItsOwn(t *testing.T) {
	f := tuitest.NewFake()
	f.SettingsV = api.Settings{AIProvider: "gemini", AIDisclosure: "Sends lines to Gemini."}
	f.ExplainV = api.Explain{Text: "Nothing alarming.", Redacted: true}
	f.Logs("box") <- liveLogs(hit(1, "x-exec.service", "info", "all fine"))
	a := tab(t, openTestHost(t, f, "unicode"), 4)
	m, cmds := tuitest.Send(a, tuitest.Key("e"))
	m = tuitest.Settle(m, cmds...)
	fr := tuitest.Frame(m)
	for _, w := range []string{"No error lines in view", "fetch the box's recent error lines", "Gemini", "enter"} {
		if !strings.Contains(fr, w) {
			t.Errorf("modal lacks %q:\n%s", w, fr)
		}
	}
	m, cmds = tuitest.Send(m, tuitest.Key("enter"))
	m = tuitest.Settle(m, cmds...)
	if !f.Called("Explain box 0") || !strings.Contains(tuitest.Frame(m), "Nothing alarming.") {
		t.Fatalf("calls %v\n%s", f.Calls, tuitest.Frame(m))
	}
}
