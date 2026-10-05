// cmd/jumpgate/console_test.go
package main

import "testing"

func TestLaunchedStandaloneHonoursTheLauncherVariable(t *testing.T) {
	t.Setenv("JUMPGATE_LAUNCHER", "1")
	if !launchedStandalone() {
		t.Fatal("JUMPGATE_LAUNCHER=1 not honoured")
	}
	// Under `go test` the console (if any) is shared with go and the shell,
	// so without the variable this process does not own its window.
	t.Setenv("JUMPGATE_LAUNCHER", "")
	if launchedStandalone() {
		t.Fatal("launchedStandalone() = true under go test")
	}
}
