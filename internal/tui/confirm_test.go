package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func TestConfirmNeedsTheExactWord(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	ran := 0
	a.modal = newConfirm("Stop beacon on box", "the beacon client stops", "beacon", func() tea.Cmd { ran++; return nil })
	for _, attempt := range []string{"Beacon", "BEACON", "beaco", "beacons", "bea con", "exec", ""} {
		tuitest.Send(a, tuitest.Type(attempt)...)
		tuitest.Send(a, tuitest.Key("enter"))
		if ran != 0 || a.modal == nil {
			t.Fatalf("%q ran the action", attempt)
		}
	}
	tuitest.Send(a, tuitest.Type("  beacon ")...)
	tuitest.Send(a, tuitest.Key("enter"))
	if ran != 1 || a.modal != nil {
		t.Fatalf("exact word (spaces trimmed): ran %d", ran)
	}
	a.modal = newConfirm("x", "y", "beacon", func() tea.Cmd { ran++; return nil })
	tuitest.Send(a, tuitest.Type("beacon")...)
	tuitest.Send(a, tuitest.Key("esc"))
	if ran != 1 || a.modal != nil {
		t.Fatal("esc must cancel without running")
	}
}

// A miss says what was wanted and clears the field for the next try.
func TestConfirmMissReasks(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	a.modal = newConfirm("Remove box-a", "jumpgate forgets box-a", "box-a", func() tea.Cmd { return nil })
	m, _ := tuitest.Send(a, tuitest.Type("box-b")...)
	m, _ = tuitest.Send(m, tuitest.Key("enter"))
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, `that is not "box-a"`) || strings.Contains(fr, "box-b") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// A pasted word (bracketed paste) lands in the field like typing; trailing
// whitespace from a copied line, newline included, is trimmed; a newline in
// the paste never confirms by itself, and case still matters.
func TestConfirmAcceptsAPaste(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	ran := 0
	a.modal = newConfirm("Stop beacon on box", "the beacon client stops", "beacon", func() tea.Cmd { ran++; return nil })

	tuitest.Send(a, tea.PasteMsg{Content: "Beacon\n"})
	tuitest.Send(a, tuitest.Key("enter"))
	if ran != 0 || a.modal == nil {
		t.Fatal("a pasted Beacon ran the action")
	}

	tuitest.Send(a, tea.PasteMsg{Content: "beacon \t\r\n"})
	if ran != 0 || a.modal == nil {
		t.Fatal("the paste's newline confirmed without enter")
	}
	tuitest.Send(a, tuitest.Key("enter"))
	if ran != 1 || a.modal != nil {
		t.Fatalf("pasted word with trailing whitespace: ran %d", ran)
	}
}

// A word that is not shown as itself cannot be typed honestly: one that
// sanitizes to nothing would confirm on a bare enter, and two ids could
// sanitize to the same word. Such a confirmation refuses to run at all.
func TestConfirmRefusesAWordItCannotShow(t *testing.T) {
	for _, want := range []string{"\x1b[0m\u202e", "box\x1b[2Ja", "", "   ", "a\nb"} {
		a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
		ran := 0
		a.modal = newConfirm("Remove", "detail", want, func() tea.Cmd { ran++; return nil })
		tuitest.Send(a, tuitest.Key("enter"))
		tuitest.Send(a, tea.PasteMsg{Content: want})
		tuitest.Send(a, tuitest.Key("enter"))
		tuitest.Send(a, tuitest.Type(sanitize(want))...)
		tuitest.Send(a, tuitest.Key("enter"))
		if ran != 0 {
			t.Fatalf("want %q: the action ran", want)
		}
		if a.modal == nil {
			t.Fatalf("want %q: enter closed the refusal; only esc should", want)
		}
		if fr := tuitest.Frame(a); !strings.Contains(fr, "cannot be confirmed here") {
			t.Fatalf("want %q: frame:\n%s", want, fr)
		}
		tuitest.Send(a, tuitest.Key("esc"))
		if a.modal != nil || ran != 0 {
			t.Fatalf("want %q: esc did not close", want)
		}
	}
}

// ctrl+v never reads the system clipboard (no pbpaste or xclip child):
// only the terminal's bracketed paste fills the field.
func TestConfirmCtrlVReadsNoClipboard(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m := newConfirm("x", "y", "beacon", func() tea.Cmd { return nil })
	if done, cmd := m.key(a, tuitest.Key("ctrl+v")); done || cmd != nil {
		t.Fatalf("ctrl+v: done %v, cmd %v", done, cmd != nil)
	}
	if m.in.Value() != "" {
		t.Fatalf("ctrl+v typed %q", m.in.Value())
	}
}
