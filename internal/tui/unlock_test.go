package tui

import (
	"bytes"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func lockedFleet() api.Fleet {
	f := sampleFleet()
	f.Locked = true
	f.Rows[0].Reachable, f.Rows[0].Status = false, nil
	f.Rows[0].Error = &api.Error{Message: "the vault is locked", Code: api.CodeLocked}
	return f
}

func sendFleet(t *testing.T, m tea.Model, f api.Fleet) tea.Model {
	t.Helper()
	m, _ = tuitest.Send(m, fleetMsg{u: apiclient.Update[api.Fleet]{Value: f, Has: true, State: apiclient.Live}})
	return m
}

func TestTopBarShowsLocked(t *testing.T) {
	m := sendFleet(t, newTestApp(t, tuitest.NewFake(), 120, 30, "unicode"), lockedFleet())
	first := strings.SplitN(ansi.Strip(tuitest.Frame(m)), "\n", 2)[0]
	if !strings.Contains(first, "LOCKED: jobs wait for unlock") {
		t.Fatalf("first line %q", first)
	}
	m = sendFleet(t, m, sampleFleet())
	if strings.Contains(tuitest.Frame(m), "LOCKED") {
		t.Fatal("badge stays after unlock")
	}
}

func TestUnlockPromptHidesInput(t *testing.T) {
	for glyphs, mask := range map[string]string{"unicode": "••••••", "ascii": "******"} {
		m := press(t, newTestApp(t, tuitest.NewFake(), 100, 30, glyphs), "U")
		m = typed(t, m, "secret")
		frame := ansi.Strip(tuitest.Frame(m))
		if !strings.Contains(frame, mask) || strings.Contains(frame, "secret") {
			t.Fatalf("%s frame:\n%s", glyphs, frame)
		}
		if glyphs == "ascii" {
			requireASCII(t, frame)
		}
	}
}

func TestUnlockPromptSendsAndClears(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, newTestApp(t, f, 100, 30, "unicode"), "U")
	m = typed(t, m, "secret")
	um := m.(*App).modal.(*unlockModal)
	buf := um.buf[:cap(um.buf)]
	m = press(t, m, "enter")
	if len(f.Unlocks) != 1 || string(f.Unlocks[0]) != "secret" {
		t.Fatalf("sent %q", f.Unlocks)
	}
	if !bytes.Equal(buf, make([]byte, len(buf))) {
		t.Fatal("the field's buffer was not cleared")
	}
	if m.(*App).modal != nil {
		t.Fatal("modal still open")
	}
}

func TestUnlockPromptShowsWrongPassphrase(t *testing.T) {
	f := tuitest.NewFake()
	f.UnlockErr = &api.Error{Status: 403, Message: "wrong passphrase", Code: api.CodeWrongPassphrase}
	m := press(t, newTestApp(t, f, 100, 30, "unicode"), "U")
	m = press(t, typed(t, m, "nope"), "enter")
	um, ok := m.(*App).modal.(*unlockModal)
	if !ok {
		t.Fatal("modal closed on a wrong passphrase")
	}
	if len(um.buf) != 0 || !strings.Contains(ansi.Strip(tuitest.Frame(m)), "wrong passphrase") {
		t.Fatalf("buf %d, frame:\n%s", len(um.buf), tuitest.Frame(m))
	}
	m = press(t, m, "esc")
	if m.(*App).modal != nil {
		t.Fatal("esc did not close")
	}
}

func TestUnlockPromptBackspaceAndPaste(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, newTestApp(t, f, 100, 30, "unicode"), "U")
	m = typed(t, m, "ab")
	m = press(t, m, "backspace")
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: "cü\n"})
	press(t, m, "enter")
	if len(f.Unlocks) != 1 || string(f.Unlocks[0]) != "acü" {
		t.Fatalf("sent %q", f.Unlocks)
	}
}

func TestLockedRowsSayUnlockToRead(t *testing.T) {
	m := sendFleet(t, newTestApp(t, tuitest.NewFake(), 120, 30, "unicode"), lockedFleet())
	frame := ansi.Strip(tuitest.Frame(m))
	var row string
	for _, l := range strings.Split(frame, "\n") {
		if strings.Contains(l, "box-a") && strings.Contains(l, "unlock") {
			row = l
		}
	}
	if !strings.Contains(row, "locked: unlock to read") || strings.Contains(row, "unreachable") {
		t.Fatalf("frame:\n%s", frame)
	}
}

func TestHelpListsTheVaultKeysWhenRelevant(t *testing.T) {
	m := sendFleet(t, newTestApp(t, tuitest.NewFake(), 100, 40, "unicode"), lockedFleet())
	fr := ansi.Strip(tuitest.Frame(press(t, m, "?")))
	if !strings.Contains(fr, "unlock the vault") || strings.Contains(fr, "ctrl+l") {
		t.Fatalf("locked help:\n%s", fr)
	}
	m = press(t, m, "?", "U")
	m = press(t, typed(t, m, "pw"), "enter")
	m = sendFleet(t, m, sampleFleet())
	fr = ansi.Strip(tuitest.Frame(press(t, m, "?")))
	if !strings.Contains(fr, "ctrl+l") || strings.Contains(fr, "unlock the vault") {
		t.Fatalf("unlocked help:\n%s", fr)
	}
}

func TestLockKeyLocks(t *testing.T) {
	f := tuitest.NewFake()
	f.VaultV.State = "unlocked"
	m := press(t, newTestApp(t, f, 100, 30, "unicode"), "ctrl+l")
	if f.VaultV.State != "locked" || !strings.Contains(strings.Join(f.Calls, ","), "Lock") {
		t.Fatalf("state %q calls %v", f.VaultV.State, f.Calls)
	}
	if !strings.Contains(m.(*App).flash, "locked") {
		t.Fatalf("flash %q", m.(*App).flash)
	}
}
