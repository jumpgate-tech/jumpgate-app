package main

import "testing"

// M-12: with no home directory jumpgate stops, rather than writing keys into
// whatever directory it was started from.
func TestJgFileStopsWithoutAHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "") // plan9
	code := -1
	old := exit
	exit = func(c int) { code = c; panic("exit") }
	t.Cleanup(func() { exit = old })
	func() {
		defer func() { _ = recover() }()
		jgFile("keys", "controller.key")
	}()
	if code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
}
