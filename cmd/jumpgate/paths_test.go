package main

import (
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// M-2: cmd.exe and Windows PowerShell never expand ~, and bash does not for
// --key=~/x.
func TestExpandHome(t *testing.T) {
	home := testutil.Home(t)
	old := hostGOOS
	t.Cleanup(func() { hostGOOS = old })
	for _, c := range []struct{ goos, in, want string }{
		{"linux", "~/.ssh/id", filepath.Join(home, ".ssh", "id")},
		{"linux", "~", home},
		{"linux", "/abs/key", "/abs/key"},
		{"linux", "rel/~/key", "rel/~/key"},
		{"linux", "~other/key", "~other/key"},
		{"windows", `~\.ssh\id`, filepath.Join(home, ".ssh", "id")},
		{"darwin", `~\x`, `~\x`},
	} {
		hostGOOS = c.goos
		got, err := expandHome(c.in)
		if err != nil || got != c.want {
			t.Errorf("%s expandHome(%q) = %q, %v; want %q", c.goos, c.in, got, err, c.want)
		}
	}
}
