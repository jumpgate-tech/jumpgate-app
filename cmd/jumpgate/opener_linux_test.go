//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The real opener path on Linux: a stand-in xdg-open on PATH records the
// argv it was given, which must be exactly the redirect file's path. Linux-only, so
// the freshly written script is never exec'd on macOS (Ruling P11).
func TestOpenBrowserHandsXdgOpenOnlyTheFile(t *testing.T) {
	dir := t.TempDir()
	rec := filepath.Join(dir, "argv")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + rec + "\n"
	if err := os.WriteFile(filepath.Join(dir, "xdg-open"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	old := hostGOOS
	hostGOOS = "linux"
	t.Cleanup(func() { hostGOOS = old })

	link := filepath.Join(dir, "open-0123456789abcdef.html")
	if err := openBrowser(link); err != nil {
		t.Fatal(err)
	}
	var got []byte
	for i := 0; i < 100 && len(got) == 0; i++ {
		got, _ = os.ReadFile(rec)
		time.Sleep(20 * time.Millisecond)
	}
	if strings.TrimSpace(string(got)) != link {
		t.Fatalf("xdg-open argv %q, want %q", got, link)
	}
}
