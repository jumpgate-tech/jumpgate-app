package tuitest

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// updating reports go test's -update flag. teatest's golden package
// registers it (a second registration would panic), and every internal/tui
// test binary links teatest, so the flag is looked up rather than defined.
func updating() bool {
	f := flag.Lookup("update")
	return f != nil && f.Value.String() == "true"
}

// Frame is what m shows: its view without colour codes or trailing spaces.
func Frame(m tea.Model) string { return normalize(ansi.Strip(m.View().Content)) }

func normalize(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n")
}

// Golden compares got with testdata/<name>.golden, or rewrites it under
// -update. The file is LF on every OS (.gitattributes).
func Golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if updating() {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no golden %s: run go test ./internal/tui/ -run %s -update and review the file", path, t.Name())
	}
	want := normalize(string(b))
	if want == got {
		return
	}
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < max(len(wl), len(gl)); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			t.Fatalf("%s differs at line %d:\nwant %q\n got %q\n\nfull frame:\n%s", path, i+1, w, g, got)
		}
	}
}
