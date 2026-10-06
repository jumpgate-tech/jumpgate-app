//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// B-3: the GUI exe's log lines and stderr land in an owner-only ~/.jumpgate/run/app.log,
// because under the GUI subsystem stderr goes nowhere.
func TestSetupGUILoggingWritesAPrivateAppLog(t *testing.T) {
	home := testutil.Home(t)
	oldFatalf, oldOut, oldStdout, oldStderr := fatalf, log.Writer(), os.Stdout, os.Stderr
	restore := func() {
		_ = debug.SetCrashOutput(nil, debug.CrashOptions{})
		fatalf, os.Stdout, os.Stderr = oldFatalf, oldStdout, oldStderr
		log.SetOutput(oldOut)
	}
	t.Cleanup(restore)

	setupGUILogging()
	log.Print("jumpgate: opening the desktop window")
	fmt.Fprintln(os.Stderr, "jumpgate: a warning for the terminal")
	f, ok := log.Writer().(*os.File)
	restore()
	if !ok {
		t.Fatalf("the logger writes to %T, not app.log", log.Writer())
	}
	f.Close()

	path := filepath.Join(home, ".jumpgate", "run", "app.log")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "opening the desktop window") || !strings.Contains(string(b), "a warning for the terminal") {
		t.Fatalf("app.log = %q", b)
	}
	testutil.AssertPrivate(t, path)
}
