//go:build unix && !linux

package vault

import (
	"os"
	"os/exec"
	"testing"

	"golang.org/x/sys/unix"
)

// TestDisableCoreDumpsZeroesTheCoreLimit runs in a child process, because
// the change (and PT_DENY_ATTACH on darwin) cannot be undone.
func TestDisableCoreDumpsZeroesTheCoreLimit(t *testing.T) {
	if os.Getenv("JG_NODUMP_CHILD") == "1" {
		if err := DisableCoreDumps(); err != nil {
			t.Fatal(err)
		}
		var r unix.Rlimit
		if err := unix.Getrlimit(unix.RLIMIT_CORE, &r); err != nil {
			t.Fatal(err)
		}
		if r.Cur != 0 || r.Max != 0 {
			t.Fatalf("RLIMIT_CORE = %+v", r)
		}
		if err := unix.Setrlimit(unix.RLIMIT_CORE, &unix.Rlimit{Cur: 1 << 20, Max: 1 << 20}); err == nil {
			t.Fatal("the core limit could be raised again")
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDisableCoreDumpsZeroesTheCoreLimit$")
	cmd.Env = append(os.Environ(), "JG_NODUMP_CHILD=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v\n%s", err, out)
	}
}
