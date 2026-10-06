package executor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Windows: a .bat or .cmd "docker" would run through cmd.exe, which re-parses
// its arguments (BatBadBut), so only .exe and .com are ever started, even
// when PATHEXT lists the others. goos is a parameter, so this runs anywhere.
func TestLookPathInOnWindowsStartsOnlyExeOrCom(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"docker.bat", "docker.cmd"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("@echo off\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const pathext = ".COM;.EXE;.BAT;.CMD"
	if got, err := lookPathIn("docker", dir, "windows", pathext); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("lookPathIn found %q (%v); a batch file must never be started", got, err)
	}
	for _, n := range []string{"docker.bat", "docker.cmd"} {
		if got, err := lookPathIn(filepath.Join(dir, n), "", "windows", pathext); !errors.Is(err, exec.ErrNotFound) {
			t.Errorf("an explicit %s path was accepted: %q (%v)", n, got, err)
		}
	}
	exe := filepath.Join(dir, "docker.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := lookPathIn("docker", dir, "windows", pathext); err != nil || !strings.EqualFold(got, exe) {
		t.Fatalf("lookPathIn = %q, %v; want %q", got, err, exe)
	}
}
