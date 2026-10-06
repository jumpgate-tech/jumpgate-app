// cmd/jumpgate/help_test.go
package main

import (
	"strings"
	"testing"
)

// Every subcommand a person runs on a controller is in the help. `agent`
// runs on the boxes and is left out on purpose.
func TestHelpListsEveryCommand(t *testing.T) {
	var b strings.Builder
	printHelp(&b)
	for name := range subcommands {
		if name == "agent" {
			continue
		}
		if !strings.Contains(b.String(), "\n  "+name) {
			t.Errorf("help does not list %q:\n%s", name, b.String())
		}
	}
}
