package tui

import (
	"testing"

	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// :help opens the guide's topic list and ? stays the key list; :keys is the
// palette's way to the same key list.
func TestHelpAndKeysCommands(t *testing.T) {
	m := press(t, typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":"), "help"), "enter")
	if g, ok := m.(*App).modal.(*manualModal); !ok || g.open {
		t.Fatalf(":help opened %#v, want the topic list", m.(*App).modal)
	}
	m = press(t, typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":"), "keys"), "enter")
	if _, ok := m.(*App).modal.(*helpModal); !ok {
		t.Fatalf(":keys opened %#v, want the key list", m.(*App).modal)
	}
}

// In the key list H is not the guide's key, since two views bind H: F opens
// the guide, and H does nothing there.
func TestHelpKeyListLeavesHAlone(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), "?", "H")
	if _, ok := m.(*App).modal.(*helpModal); !ok {
		t.Fatalf("H left the key list: %#v", m.(*App).modal)
	}
}
