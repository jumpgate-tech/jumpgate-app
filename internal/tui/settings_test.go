package tui

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/ai"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func settingsApp(t *testing.T, f *tuitest.Fake) *App {
	t.Helper()
	f.SettingsV = api.Settings{AIProvider: "gemini", AIKeySet: true, AIDisclosure: "Explain sends the selected log lines to Gemini after removing IP addresses."}
	a := newTestApp(t, f, 80, 24, "unicode")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: sampleFleet(), Has: true, State: apiclient.Live}})
	return press(t, m, "6").(*App)
}

func pick(t *testing.T, a *App, label string) *App {
	t.Helper()
	s := a.screens[scrSettings].(*settingsScreen)
	i := s.find(a, label)
	if i < 0 {
		t.Fatalf("no setting %q", label)
	}
	s.cursor = i
	return a
}

func TestSettingsFrame(t *testing.T) {
	fr := tuitest.Frame(settingsApp(t, tuitest.NewFake()))
	for _, want := range []string{"FLEET COLUMNS", "[x] HOST", "[ ] JOBS", "space toggle"} {
		if !strings.Contains(fr, want) {
			t.Errorf("settings lack %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "settings", fr)
}

func TestSettingsToggleAColumnAndSaveIt(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "JOBS"), "space")
	if !f.Called("SavePrefs") || !slices.Contains(m.(*App).prefs.FleetColumns, "jobs") || !slices.Contains(f.LastPrefs.FleetColumns, "jobs") {
		t.Fatalf("calls %v prefs %v", f.Calls, m.(*App).prefs.FleetColumns)
	}
}

func TestSettingsKeepTheHostColumn(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "HOST"), "space")
	if f.Called("SavePrefs") || !strings.Contains(tuitest.Frame(m), "always shown") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestSettingsChainFilter(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "Ethereum (1)"), "space")
	if got := m.(*App).prefs.Chains; !slices.Equal(got, []int{369}) {
		t.Fatalf("chains %v (hiding Ethereum keeps the other seen chain)", got)
	}
	m = press(t, pick(t, m.(*App), "Ethereum (1)"), "space")
	if got := m.(*App).prefs.Chains; len(got) != 0 {
		t.Fatalf("showing every seen chain again must clear the filter: %v", got)
	}
}

func TestSettingsTogglesDoNotShareSlices(t *testing.T) {
	f := tuitest.NewFake()
	a := settingsApp(t, f)
	before := slices.Clone(a.prefs.FleetColumns)
	old := a.prefs.FleetColumns
	press(t, pick(t, a, "JOBS"), "space")
	if !slices.Equal(old, before) {
		t.Fatalf("the earlier prefs' slice changed: %v -> %v", before, old)
	}
	p := api.DefaultPrefs()
	c := clonePrefs(p)
	c.FleetColumns[0], c.HostTabs[0] = "x", "x"
	if p.FleetColumns[0] == "x" || p.HostTabs[0] == "x" {
		t.Fatal("clonePrefs shares slices")
	}
}

func TestSettingsPrefsRefusedShowsTheHintAndReverts(t *testing.T) {
	f := tuitest.NewFake()
	f.Err["SavePrefs"] = &api.Error{Code: api.CodeInvalidPrefs, Message: "bad prefs", Hint: "keep at least one column"}
	a := settingsApp(t, f)
	m := press(t, pick(t, a, "JOBS"), "space")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "keep at least one column") {
		t.Fatalf("hint missing:\n%s", fr)
	}
	if slices.Contains(m.(*App).prefs.FleetColumns, "jobs") {
		t.Fatal("the refused change still shows")
	}
}

const secret = "sekrit-value"

func TestSettingsAIKeyIsMasked(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "AI key"), "enter")
	m = typed(t, m, secret)
	if fr := tuitest.Frame(m); strings.Contains(fr, "sekrit") {
		t.Fatalf("the key is shown while typing:\n%s", fr)
	}
	m = press(t, m, "enter")
	if !f.Called("SaveSettings") || f.LastUpdate.AIKey == nil || *f.LastUpdate.AIKey != secret {
		t.Fatalf("calls %v update %+v", f.Calls, f.LastUpdate)
	}
	if fr := tuitest.Frame(m); strings.Contains(fr, "sekrit") {
		t.Fatalf("the key is shown after saving:\n%s", fr)
	}
	s := m.(*App).screens[scrSettings].(*settingsScreen)
	if s.keyInput.Value() != "" || s.editingKey {
		t.Fatal("the key was kept after saving")
	}
}

func TestSettingsAIKeyErrorNeverEchoesTheKey(t *testing.T) {
	f := tuitest.NewFake()
	f.Err["SaveSettings"] = errors.New("provider rejected " + secret)
	m := press(t, pick(t, settingsApp(t, f), "AI key"), "enter")
	m = typed(t, m, secret)
	m = press(t, m, "enter")
	if fr := tuitest.Frame(m); strings.Contains(fr, "sekrit") || !strings.Contains(fr, "AI settings") {
		t.Fatalf("error frame:\n%s", fr)
	}
}

func TestSettingsAIKeyEscClears(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "AI key"), "enter")
	m = typed(t, m, secret)
	m = press(t, m, "esc")
	s := m.(*App).screens[scrSettings].(*settingsScreen)
	if s.keyInput.Value() != "" || s.editingKey || f.Called("SaveSettings") {
		t.Fatalf("value %q editing %v calls %v", s.keyInput.Value(), s.editingKey, f.Calls)
	}
}

func TestSettingsAIKeyClearedOnLeaving(t *testing.T) {
	m := press(t, pick(t, settingsApp(t, tuitest.NewFake()), "AI key"), "enter")
	m = typed(t, m, secret)
	a := m.(*App)
	a.openScreen(scrFleet)
	s := a.screens[scrSettings].(*settingsScreen)
	if s.keyInput.Value() != "" || s.editingKey {
		t.Fatal("the key survived leaving the screen")
	}
}

func TestSettingsEmptyKeyDoesNotClear(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "AI key"), "enter", "enter")
	if f.Called("SaveSettings") {
		t.Fatalf("an empty entry saved: %v %+v", f.Calls, f.LastUpdate)
	}
	if !m.(*App).screens[scrSettings].(*settingsScreen).settings.AIKeySet {
		t.Fatal("the key is no longer set")
	}
}

func TestSettingsExplicitClearNeedsTheTypedWord(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "Clear the AI key"), "enter")
	if _, ok := m.(*App).modal.(*confirmModal); !ok {
		t.Fatalf("modal %#v", m.(*App).modal)
	}
	m = typed(t, m, "nope")
	m = press(t, m, "enter")
	if f.Called("SaveSettings") {
		t.Fatal("cleared without the word")
	}
	m = typed(t, m, "clear")
	m = press(t, m, "enter")
	if f.LastUpdate.AIKey == nil || *f.LastUpdate.AIKey != "" || !f.Called("SaveSettings") {
		t.Fatalf("update %+v", f.LastUpdate)
	}
	if m.(*App).screens[scrSettings].(*settingsScreen).settings.AIKeySet {
		t.Fatal("the key still reads as set")
	}
}

func TestSettingsKeyFieldTakesGlobalKeysAsText(t *testing.T) {
	m := press(t, pick(t, settingsApp(t, tuitest.NewFake()), "AI key"), "enter")
	m = typed(t, m, ":q1?")
	a := m.(*App)
	if a.modal != nil || a.active != scrSettings || a.screens[scrSettings].(*settingsScreen).keyInput.Value() != ":q1?" {
		t.Fatalf("modal %v active %v", a.modal, a.active)
	}
}

func TestSettingsKeyFieldDisablesCtrlV(t *testing.T) {
	m := press(t, pick(t, settingsApp(t, tuitest.NewFake()), "AI key"), "enter")
	s := m.(*App).screens[scrSettings].(*settingsScreen)
	if s.keyInput.KeyMap.Paste.Enabled() {
		t.Fatal("ctrl+v is enabled")
	}
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: secret + "\n"})
	if s.keyInput.Value() != secret {
		t.Fatalf("bracketed paste not taken: %q", s.keyInput.Value())
	}
	if fr := tuitest.Frame(m); strings.Contains(fr, "sekrit") {
		t.Fatalf("pasted key shown:\n%s", fr)
	}
}

func TestSettingsShowTheDisclosure(t *testing.T) {
	a := settingsApp(t, tuitest.NewFake())
	a = pick(t, a, "AI provider")
	if fr := tuitest.Frame(a); !strings.Contains(fr, "removing IP addresses") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestSettingsProviderListMatchesTheServer(t *testing.T) {
	want := append([]string{""}, ai.Providers()...)
	if !slices.Equal(aiProviders, want) {
		t.Fatalf("TUI providers %q, server %q: update aiProviders", aiProviders, want)
	}
}

func TestSettingsUnknownProviderIsShownAndNotSilentlyCleared(t *testing.T) {
	f := tuitest.NewFake()
	a := settingsApp(t, f)
	a.screens[scrSettings].(*settingsScreen).settings.AIProvider = "mystery\x1b[2J"
	a = pick(t, a, "AI provider")
	fr := tuitest.Frame(a)
	if !strings.Contains(fr, "mystery") || !strings.Contains(fr, "unknown") || strings.Contains(fr, "\x1b") {
		t.Fatalf("frame:\n%s", fr)
	}
	press(t, a, "space")
	if p := f.LastUpdate.AIProvider; p == nil || *p == "" {
		t.Fatalf("an unknown provider was moved to none: %+v", f.LastUpdate)
	}
}

func TestSettingsKeyNeverShownInAnErrorHintOrMessage(t *testing.T) {
	for name, err := range map[string]error{
		"hint":    &api.Error{Code: "bad", Message: "refused", Hint: "the key " + secret + " is wrong"},
		"message": &api.Error{Code: "bad", Message: "refused " + secret},
		"code":    &api.Error{Code: api.Code("x" + secret), Message: "refused"},
	} {
		t.Run(name, func(t *testing.T) {
			f := tuitest.NewFake()
			f.Err["SaveSettings"] = err
			m := press(t, pick(t, settingsApp(t, f), "AI key"), "enter")
			m = typed(t, m, secret)
			m = press(t, m, "enter")
			if fr := tuitest.Frame(m); strings.Contains(fr, "sekrit") || !strings.Contains(fr, "AI settings") {
				t.Fatalf("frame:\n%s", fr)
			}
		})
	}
}

func TestSettingsEarlierRefusedSaveDoesNotUndoALaterOne(t *testing.T) {
	f := tuitest.NewFake()
	m := tea.Model(pick(t, settingsApp(t, f), "JOBS"))
	m, c1 := tuitest.Send(m, tuitest.Key("space"))
	m = pick(t, m.(*App), "AGE")
	m, c2 := tuitest.Send(m, tuitest.Key("space"))
	f.Err["SavePrefs"] = &api.Error{Code: api.CodeInvalidPrefs, Message: "no"}
	r1 := tuitest.Run(c1[0])
	delete(f.Err, "SavePrefs")
	r2 := tuitest.Run(c2[0])
	m, _ = tuitest.Send(m, r2...)
	m, _ = tuitest.Send(m, r1...) // the first one's refusal arrives after the second was answered
	cols := m.(*App).prefs.FleetColumns
	if !slices.Contains(cols, "age") {
		t.Fatalf("the later change was lost: %v", cols)
	}
}

func TestSettingsLatestRefusedSaveRefetchesFromTheServer(t *testing.T) {
	f := tuitest.NewFake()
	f.Err["SavePrefs"] = &api.Error{Code: api.CodeInvalidPrefs, Message: "no", Hint: "try again"}
	m := press(t, pick(t, settingsApp(t, f), "JOBS"), "space")
	if !f.Called("Prefs") || slices.Contains(m.(*App).prefs.FleetColumns, "jobs") {
		t.Fatalf("calls %v cols %v", f.Calls, m.(*App).prefs.FleetColumns)
	}
	if !strings.Contains(tuitest.Frame(m), "try again") {
		t.Fatal("hint missing")
	}
}

func TestSettingsLateSuccessDoesNotOverwriteANewerChange(t *testing.T) {
	f := tuitest.NewFake()
	m := tea.Model(pick(t, settingsApp(t, f), "JOBS"))
	m, c1 := tuitest.Send(m, tuitest.Key("space"))
	r1 := tuitest.Run(c1[0])
	m = pick(t, m.(*App), "AGE")
	m, _ = tuitest.Send(m, tuitest.Key("space")) // newer local change, not yet answered
	m, _ = tuitest.Send(m, r1...)
	if cols := m.(*App).prefs.FleetColumns; !slices.Contains(cols, "age") {
		t.Fatalf("a late reply overwrote the newer change: %v", cols)
	}
}

func TestSettingsProviderChangeAndRefusal(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "AI provider"), "space")
	if f.LastUpdate.AIProvider == nil || *f.LastUpdate.AIProvider != "groq" {
		t.Fatalf("update %+v", f.LastUpdate)
	}
	f.Err["SaveSettings"] = &api.Error{Code: api.CodeUnknownProvider, Message: "unknown provider \x1b[31mx"}
	m = press(t, m, "space")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "unknown provider") || strings.Contains(fr, "\x1b") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestSettingsSanitizeServerStrings(t *testing.T) {
	f := tuitest.NewFake()
	a := settingsApp(t, f)
	s := a.screens[scrSettings].(*settingsScreen)
	s.settings.AIProvider = "evil\x1b]52;c;AAAA\x07"
	s.settings.AIDisclosure = "line\x1b[2Jtwo"
	a = pick(t, a, "AI provider")
	if fr := tuitest.Frame(a); strings.ContainsAny(fr, "\x1b\x07") {
		t.Fatalf("escape in frame: %q", fr)
	}
}

func TestSettingsOpenTheWebApp(t *testing.T) {
	opened := 0
	f := tuitest.NewFake()
	a := settingsApp(t, f)
	a.o.OpenWeb = func(context.Context) error { opened++; return nil }
	press(t, pick(t, a, "Open the web app"), "enter")
	if opened != 1 {
		t.Fatal("the web app was not opened")
	}
}

func TestSettingsHideOpenWebWithoutIt(t *testing.T) {
	a := settingsApp(t, tuitest.NewFake())
	if a.screens[scrSettings].(*settingsScreen).find(a, "Open the web app") >= 0 {
		t.Fatal("offered without Options.OpenWeb")
	}
	if strings.Contains(tuitest.Frame(a), "web app") {
		t.Fatal("shown without Options.OpenWeb")
	}
}

func TestSettingsStaleRepliesAreDropped(t *testing.T) {
	a := settingsApp(t, tuitest.NewFake())
	s := a.screens[scrSettings].(*settingsScreen)
	old := s.gen
	a.openScreen(scrFleet)
	a.flash = ""
	tuitest.Send(a,
		settingsMsg{gen: old, s: api.Settings{AIProvider: "groq"}},
		settingsMsg{gen: old, err: errors.New("late")},
		webOpenedMsg{gen: old, err: errors.New("late")})
	if s.settings.AIProvider != "gemini" || a.flash != "" {
		t.Fatalf("a late reply was applied: %q flash %q", s.settings.AIProvider, a.flash)
	}
}

func TestMouseAndGlyphPreferencesApply(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, pick(t, settingsApp(t, f), "Mouse"), "space")
	if !m.(*App).prefs.Mouse || m.(*App).View().MouseMode == 0 {
		t.Fatal("mouse not enabled")
	}
	m = press(t, pick(t, m.(*App), "Mouse"), "space")
	if m.(*App).View().MouseMode != 0 {
		t.Fatal("mouse not disabled")
	}
	// newTestApp starts on unicode; one press is ascii, and it applies.
	m = press(t, pick(t, m.(*App), "Glyphs"), "space")
	if g := m.(*App).prefs.Glyphs; g != "ascii" || !m.(*App).gl.ASCII {
		t.Fatalf("glyphs %q ascii %v", g, m.(*App).gl.ASCII)
	}
}
