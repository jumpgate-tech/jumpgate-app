//go:build windows

package daemon

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

// A job that forbids breakaway refuses CREATE_BREAKAWAY_FROM_JOB with
// ERROR_ACCESS_DENIED; the server then starts inside the job rather than not
// at all.
func TestStartDetachedRetriesWithoutBreakaway(t *testing.T) {
	var flags []uint32
	old := startProc
	startProc = func(cmd *exec.Cmd) error {
		flags = append(flags, cmd.SysProcAttr.CreationFlags)
		if len(flags) == 1 {
			return windows.ERROR_ACCESS_DENIED
		}
		return nil
	}
	t.Cleanup(func() { startProc = old })

	started, err := startDetached(exec.Command("jumpgate.exe", "serve", "--no-open"))
	if err != nil || started == nil {
		t.Fatalf("startDetached = %v, %v", started, err)
	}
	if len(flags) != 2 || flags[0]&createBreakawayFromJob == 0 || flags[1]&createBreakawayFromJob != 0 {
		t.Fatalf("creation flags %#x, want breakaway then none", flags)
	}
	for _, f := range flags {
		if f&detachedProcess == 0 || f&createNewProcessGroup == 0 {
			t.Fatalf("flags %#x lack DETACHED_PROCESS or CREATE_NEW_PROCESS_GROUP", f)
		}
	}
}
