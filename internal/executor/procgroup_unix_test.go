//go:build unix

package executor

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// dash refuses job control without a tty. The branch must then still run the
// command correctly, and its marker must not claim a group id. The child is
// started in a new session, with no controlling terminal, as sshd starts an
// exec request; otherwise dash would find /dev/tty when the tests run from a
// terminal and turn job control on.
func TestProcessGroupJobControlBranchWithoutJobControl(t *testing.T) {
	if _, err := exec.LookPath("dash"); err != nil {
		t.Skip("dash not installed")
	}
	marker, _ := runBranch(t, "dash", jobControlBranch(groupProbeCmd), func(c *exec.Cmd) {
		c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	})
	if _, ok := parsePgidLine(marker); ok {
		t.Fatalf("marker %q carries a group id although dash has no job control", marker)
	}
}

// killGroupCmd must work in dash, the usual /bin/sh on Debian and Ubuntu,
// whose kill builtin needs the POSIX "-s SIGNAL" form.
func TestKillGroupCmdUnderDash(t *testing.T) {
	if _, err := exec.LookPath("dash"); err != nil {
		t.Skip("dash not installed; killGroupCmd not checked under dash")
	}
	// A real group: a shell leading its own group, with a child in it.
	group := exec.Command("sh", "-c", "sleep 60 & wait")
	group.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := group.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := group.Process.Pid
	exited := make(chan struct{})
	go func() { _ = group.Wait(); close(exited) }()
	t.Cleanup(func() { _ = syscall.Kill(-pgid, syscall.SIGKILL) })

	if out, err := exec.Command("dash", "-c", killGroupCmd(pgid)).CombinedOutput(); err != nil {
		t.Fatalf("killGroupCmd: %v: %s", err, out)
	}
	select {
	case <-exited:
	case <-time.After(10 * time.Second):
		t.Fatal("group leader survived killGroupCmd under dash")
	}
	deadline := time.Now().Add(5 * time.Second)
	for syscall.Kill(-pgid, 0) == nil {
		if time.Now().After(deadline) {
			t.Fatalf("process group %d survived killGroupCmd under dash", pgid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
