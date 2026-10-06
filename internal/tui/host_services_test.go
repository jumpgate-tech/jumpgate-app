package tui

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func servicesHost(t *testing.T, f *tuitest.Fake) *App {
	t.Helper()
	return tab(t, openTestHost(t, f, "unicode"), 5)
}

func press(t *testing.T, m tea.Model, keys ...string) tea.Model {
	t.Helper()
	for _, k := range keys {
		var cmds []tea.Cmd
		m, cmds = tuitest.Send(m, tuitest.Key(k))
		m = tuitest.Settle(m, cmds...)
	}
	return m
}

func typed(t *testing.T, m tea.Model, s string) tea.Model {
	t.Helper()
	m, _ = tuitest.Send(m, tuitest.Type(s)...)
	return m
}

// stubShell makes the exec seam record the command instead of building a
// real one; the returned cmd is never run by the tests.
func stubShell(a *App) *[]string {
	var got []string
	a.o.Command = func(name string, args ...string) *exec.Cmd {
		got = append([]string{name}, args...)
		return exec.Command("true")
	}
	return &got
}

// noEscapes: the app was built with plainTheme, so an ESC or BEL in the frame
// came from data.
func noEscapes(t *testing.T, what string, m tea.Model) {
	t.Helper()
	c := m.View().Content
	for _, bad := range []string{"\x1b", "\x9b", "\r", "\x07"} {
		if strings.Contains(c, bad) {
			t.Fatalf("%s: frame contains %q:\n%q", what, bad, c)
		}
	}
}

func TestServicesFrame(t *testing.T) {
	a := servicesHost(t, tuitest.NewFake())
	fr := tuitest.Frame(a)
	for _, want := range []string{"SERVICES", "exec", "beacon", "active", "s start", "t stop", "r restart", "S  open an SSH shell"} {
		if !strings.Contains(fr, want) {
			t.Errorf("services lack %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "host_services", fr)
}

func TestStartNeedsNoConfirmation(t *testing.T) {
	f := tuitest.NewFake()
	f.ServiceV = api.ServiceResult{Active: true}
	m := press(t, servicesHost(t, f), "s")
	if !f.Called("ServiceAction box exec start") {
		t.Fatalf("calls %v", f.Calls)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "start exec on box: active") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// Review Focus 4: only the exact service name stops it; near misses and esc
// send nothing.
func TestStopNeedsTheExactServiceName(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, servicesHost(t, f), "j", "t")
	if _, ok := m.(*App).modal.(*confirmModal); !ok {
		t.Fatal("stop did not ask for confirmation")
	}
	for _, near := range []string{"Beacon", "beacon.", "exec"} {
		m = typed(t, m, near)
		m = press(t, m, "enter")
		if f.Called("ServiceAction") {
			t.Fatalf("%q stopped the service", near)
		}
	}
	m = press(t, m, "esc")
	if m.(*App).modal != nil || f.Called("ServiceAction") {
		t.Fatal("esc must cancel without a request")
	}
	m = press(t, m, "t")
	m = typed(t, m, " beacon ")
	press(t, m, "enter")
	if !f.Called("ServiceAction box beacon stop") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestRestartAlsoNeedsTheName(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, servicesHost(t, f), "r")
	press(t, m, "enter") // nothing typed
	if f.Called("ServiceAction") {
		t.Fatal("a bare enter restarted the service")
	}
	m = typed(t, m, "exec")
	press(t, m, "enter")
	if !f.Called("ServiceAction box exec restart") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestShellRunsTheServersSSHCommand(t *testing.T) {
	f := tuitest.NewFake()
	f.SSHV["box"] = api.SSHCommand{Argv: []string{"ssh", "-p", "22", "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile=/h/c", "-o", "GlobalKnownHostsFile=none", "-o", "ProxyJump=none", "-o", "ProxyCommand=none", "-l", "root", "--", "10.0.0.5"}, Display: "ssh \x1b[31m"}
	a := openTestHost(t, f, "unicode")
	a.th = plainTheme()
	got := stubShell(a)
	m := press(t, a, "S")
	if !slices.Equal(*got, f.SSHV["box"].Argv) {
		t.Fatalf("ran %v", *got)
	}
	noEscapes(t, "shell", m)
}

// A server that sends a hostile command gets nothing exec'd.
func TestShellRefusesAHostileArgv(t *testing.T) {
	for _, argv := range [][]string{
		{"ssh", "-o", "ProxyCommand=nc evil 1", "--", "h"},
		{"ssh", "-J", "evil", "--", "h"},
		{"/bin/sh", "-c", "id"},
		{"ssh", "h"},
		{"ssh", "-p", "22", "-o", " ProxyCommand=echo PWN", "--", "h"},
		{"ssh", "-p", "22", "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile=/dev/null", "--", "h"},
		{},
	} {
		f := tuitest.NewFake()
		f.SSHV["box"] = api.SSHCommand{Argv: argv}
		a := openTestHost(t, f, "unicode")
		got := stubShell(a)
		m := press(t, a, "S")
		if len(*got) != 0 {
			t.Fatalf("%q was exec'd: %v", argv, *got)
		}
		if fr := tuitest.Frame(m); !strings.Contains(fr, "refusing") {
			t.Fatalf("%q: no refusal:\n%s", argv, fr)
		}
	}
}

func TestShellJumpTargetShowsTheHint(t *testing.T) {
	f := tuitest.NewFake()
	f.Err = map[string]error{"SSHCommand": &api.Error{Code: api.CodeSSHJumpUnsupported, Message: "jump \x1b[31mhost", Hint: "use the TUI"}}
	a := openTestHost(t, f, "unicode")
	a.th = plainTheme()
	got := stubShell(a)
	m := press(t, a, "S")
	fr := tuitest.Frame(m)
	if len(*got) != 0 || !strings.Contains(fr, "use the TUI") {
		t.Fatalf("ran %v:\n%s", *got, fr)
	}
	noEscapes(t, "jump", m)
}

func TestShellExitFlashesSanitizedError(t *testing.T) {
	a := openTestHost(t, tuitest.NewFake(), "unicode")
	a.th = plainTheme()
	h := a.detail.(*hostScreen)
	m, _ := tuitest.Send(a, shellDoneMsg{id: "box", gen: h.gen, err: errors.New("exit status 255\x1b[31m")})
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "exit status 255") {
		t.Fatalf("frame:\n%s", fr)
	}
	noEscapes(t, "shell done", m)
	m, _ = tuitest.Send(m, shellDoneMsg{id: "box", gen: h.gen})
	if !strings.Contains(tuitest.Frame(m), "back from box") {
		t.Fatal("no resume notice")
	}
}

func TestActionsMenuRunsTheChosenAction(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, openTestHost(t, f, "unicode"), "x")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "restart beacon") || !strings.Contains(fr, "measure disk") {
		t.Fatalf("frame:\n%s", fr)
	}
	m = press(t, m, "j", "j", "j", "enter") // the fourth item: restart beacon
	if f.Called("ServiceAction") {
		t.Fatal("the menu ran a restart without the typed name")
	}
	if _, ok := m.(*App).modal.(*confirmModal); !ok {
		t.Fatal("no confirmation after choosing restart")
	}
	m = typed(t, m, "beacon")
	press(t, m, "enter")
	if !f.Called("ServiceAction box beacon restart") {
		t.Fatalf("calls %v", f.Calls)
	}
}

// No path from the menu stops or restarts without the typed word; start and
// the others do not need it.
func TestActionsMenuStopAndRestartAlwaysConfirm(t *testing.T) {
	for i, item := range newActions(nil, nil).items {
		f := tuitest.NewFake()
		m := press(t, openTestHost(t, f, "unicode"), "x")
		for j := 0; j < i; j++ {
			m = press(t, m, "j")
		}
		m = press(t, m, "enter")
		stops := strings.HasPrefix(item, "stop ") || strings.HasPrefix(item, "restart ")
		_, confirming := m.(*App).modal.(*confirmModal)
		if stops != confirming {
			t.Errorf("%q: confirming=%v", item, confirming)
		}
		if stops && f.Called("ServiceAction") {
			t.Errorf("%q ran before the word was typed", item)
		}
		if stops {
			m = typed(t, m, "wrong")
			press(t, m, "enter")
			if f.Called("ServiceAction") {
				t.Errorf("%q ran on a wrong word", item)
			}
		}
	}
}

func TestActionsMenuShellAndEsc(t *testing.T) {
	f := tuitest.NewFake()
	f.SSHV["box"] = api.SSHCommand{Argv: []string{"ssh", "-p", "22", "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile=/h/c", "-o", "GlobalKnownHostsFile=none", "-o", "ProxyJump=none", "-o", "ProxyCommand=none", "--", "h"}}
	a := openTestHost(t, f, "unicode")
	got := stubShell(a)
	m := press(t, a, "x", "esc")
	if m.(*App).modal != nil {
		t.Fatal("esc left the menu open")
	}
	m = press(t, m, "x", "j", "j", "j", "j", "j", "j", "j", "enter")
	if len(*got) == 0 || m.(*App).modal != nil {
		t.Fatalf("ran %v modal %v", *got, m.(*App).modal)
	}
}

// Typing S or x into the logs filter is text, not a command.
func TestShellKeysAreTextWhileFiltering(t *testing.T) {
	f := tuitest.NewFake()
	a := tab(t, openTestHost(t, f, "unicode"), 4)
	got := stubShell(a)
	m := press(t, a, "/")
	m = typed(t, m, "Sx")
	if len(*got) != 0 || f.Called("SSHCommand") || m.(*App).modal != nil {
		t.Fatal("a filter keystroke opened the shell or menu")
	}
}

// Replies from an earlier open of the same box are dropped: no flash, no
// exec, no commands.
func TestServiceAndShellRepliesCarryTheOpenGeneration(t *testing.T) {
	f := tuitest.NewFake()
	a := newTestApp(t, f, 80, 24, "unicode")
	a.openHost("box")
	old := a.detail.(*hostScreen)
	a.closeDetail()
	a.openHost("box")
	got := stubShell(a)
	m, cmds := tuitest.Send(a,
		serviceMsg{id: "box", gen: old.gen, svc: "exec", action: "stop", res: api.ServiceResult{Active: true}},
		sshCmdMsg{id: "box", gen: old.gen, c: api.SSHCommand{Argv: []string{"ssh", "--", "h"}}},
		shellDoneMsg{id: "box", gen: old.gen, err: errors.New("boom")})
	if len(cmds) != 0 || len(*got) != 0 {
		t.Fatalf("stale replies produced %d commands, ran %v", len(cmds), *got)
	}
	if h := m.(*App).detail.(*hostScreen); h.svcLast != "" || m.(*App).flash != "" {
		t.Fatalf("stale reply took effect: %q %q", h.svcLast, m.(*App).flash)
	}
}

func TestServiceResultsAreSanitized(t *testing.T) {
	a := openTestHost(t, tuitest.NewFake(), "unicode")
	a.th = plainTheme()
	h := a.detail.(*hostScreen)
	m, _ := tuitest.Send(a, serviceMsg{id: "box", gen: h.gen, svc: "exec", action: "stop", err: errors.New("denied \x1b]0;x\x07 \x1b[31mred")})
	noEscapes(t, "service error", m)
	if !strings.Contains(tuitest.Frame(m), "denied") {
		t.Fatal("error not shown")
	}
}

// Leaving the box closes the confirmation or menu that belongs to it.
func TestLeavingTheHostClosesItsModal(t *testing.T) {
	for _, keys := range [][]string{{"x"}, {"j", "t"}} {
		f := tuitest.NewFake()
		a := press(t, servicesHost(t, f), keys...).(*App)
		if a.modal == nil {
			t.Fatalf("%v opened no modal", keys)
		}
		a.closeDetail()
		if a.modal != nil || a.detail != nil {
			t.Fatalf("%v: modal survived leaving the host", keys)
		}
	}
}
