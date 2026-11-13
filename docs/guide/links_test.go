package guide

import (
	"regexp"
	"strings"
	"testing"
)

// Every link to another page names a topic that exists, so the TUI's
// ":help TOPIC" hint is always a command that works.
func TestPageLinksResolve(t *testing.T) {
	have := map[string]bool{}
	for _, tp := range Embedded().Topics() {
		have[tp.Name] = true
	}
	link := regexp.MustCompile(`\]\(([^)]+)\)`)
	for _, tp := range Embedded().Topics() {
		for _, m := range link.FindAllStringSubmatch(tp.Body, -1) {
			if target, ok := strings.CutSuffix(m[1], ".md"); ok && !strings.Contains(target, ":") && !have[target] {
				t.Errorf("%s.md links to %s, which is not a topic", tp.Name, m[1])
			}
		}
	}
}

// The per-box program is the gate in everything a reader sees. The word
// agent appears only as part of a command, file or unit name, in code.
func TestSayGateNotAgent(t *testing.T) {
	code := regexp.MustCompile("(?s)```.*?```|`[^`]*`")
	for _, tp := range Embedded().Topics() {
		if strings.Contains(strings.ToLower(code.ReplaceAllString(tp.Body, "")), "agent") {
			t.Errorf("%s.md says agent outside code; say gate", tp.Name)
		}
	}
}
