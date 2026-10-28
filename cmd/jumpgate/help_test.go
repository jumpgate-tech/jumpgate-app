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

// The v0.7.0 subcommands that live under keys, vault, hosts and setup.
func TestHelpListsTheServerAndVaultSubcommands(t *testing.T) {
	var b strings.Builder
	printHelp(&b)
	for _, name := range []string{"keys export", "vault seal", "vault put", "hosts upgrade", "setup server"} {
		if !strings.Contains(b.String(), "\n  "+name) {
			t.Errorf("help does not list %q", name)
		}
	}
}
