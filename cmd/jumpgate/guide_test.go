package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/docs/guide"
)

// Every "jumpgate ..." line in the guide's code blocks names a command that
// `jumpgate help` lists, and for the commands that take a subcommand, a
// subcommand it lists, so the guide cannot teach a command that does not
// exist.
func TestGuideCommandsAreInHelp(t *testing.T) {
	words := regexp.MustCompile(`[^A-Za-z0-9-]+`)
	byCommand := map[string]map[string]bool{} // first word of a command line -> every word on its lines
	for _, c := range commandHelp {
		ws := words.Split(c.name, -1)
		if byCommand[ws[0]] == nil {
			byCommand[ws[0]] = map[string]bool{}
		}
		for _, w := range ws {
			byCommand[ws[0]][w] = true
		}
	}
	hasSub := map[string]bool{"keys": true, "hosts": true, "vault": true, "setup": true}
	n := 0
	for _, tp := range guide.Embedded().Topics() {
		inCode := false
		for _, l := range strings.Split(tp.Body, "\n") {
			if strings.HasPrefix(l, "```") {
				inCode = !inCode
				continue
			}
			l, _, _ = strings.Cut(l, "#")
			f := strings.Fields(l)
			if !inCode || len(f) < 2 || f[0] != "jumpgate" {
				continue
			}
			n++
			ws, ok := byCommand[f[1]]
			if !ok {
				t.Errorf("%s.md: %q: jumpgate has no command %q", tp.Name, strings.TrimSpace(l), f[1])
				continue
			}
			if hasSub[f[1]] && len(f) > 2 && !ws[f[2]] {
				t.Errorf("%s.md: %q: %s has no subcommand %q", tp.Name, strings.TrimSpace(l), f[1], f[2])
			}
		}
	}
	if n < 20 {
		t.Fatalf("checked only %d command lines", n)
	}
}
