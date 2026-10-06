package executor

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// The wrapper must preserve the command's stdout and exit status exactly,
// apart from one leading marker line the executor consumes.
func TestWrapInProcessGroupPreservesOutputAndStatus(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	out, err := exec.Command("sh", "-c", wrapInProcessGroup("echo one; echo two; exit 7")).Output()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 7 {
		t.Fatalf("exit = %v, want status 7", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], pgidMarker) || lines[1] != "one" || lines[2] != "two" {
		t.Fatalf("output = %q", out)
	}
}

// Stdin still reaches the command, so WriteFile works through the wrapper.
func TestWrapInProcessGroupPassesStdin(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	c := exec.Command("sh", "-c", wrapInProcessGroup("cat"))
	c.Stdin = strings.NewReader("hello")
	var out bytes.Buffer
	c.Stdout = &out
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out.String(), "hello") {
		t.Fatalf("stdin did not reach the command: %q", out.String())
	}
}

// A command containing single quotes and shell metacharacters survives the
// wrapper's quoting unchanged.
func TestWrapInProcessGroupQuoting(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	out, err := exec.Command("sh", "-c", wrapInProcessGroup(`printf '%s|%s\n' "it's" '$HOME "x"'`)).Output()
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 || lines[1] != `it's|$HOME "x"` {
		t.Fatalf("output = %q", out)
	}
}

// groupProbeCmd prints the process group of the command's own shell, so a
// test can check that the marker names the group the command really runs in.
const groupProbeCmd = `ps -o pgid= -p $$; echo err >&2; read line; echo "got:$line"; exit 5`

// runBranch runs one wrapper branch under shell and checks the contract that
// every branch shares: marker first, stdin through, stderr untouched (no job
// notices), exit status preserved. It returns the marker line and the pgid
// the command reported for itself.
func runBranch(t *testing.T, shell, script string, setup ...func(*exec.Cmd)) (marker string, ownPgid int) {
	t.Helper()
	c := exec.Command(shell, "-c", script)
	for _, f := range setup {
		f(c)
	}
	c.Stdin = strings.NewReader("in\n")
	var out, errb bytes.Buffer
	c.Stdout, c.Stderr = &out, &errb
	err := c.Run()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 5 {
		t.Fatalf("%s: exit = %v, want status 5 (stderr %q)", shell, err, errb.String())
	}
	if errb.String() != "err\n" {
		t.Fatalf("%s: stderr = %q, want %q", shell, errb.String(), "err\n")
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], pgidMarker) || lines[2] != "got:in" {
		t.Fatalf("%s: output = %q", shell, out.String())
	}
	pg, err := strconv.Atoi(strings.TrimSpace(lines[1]))
	if err != nil {
		t.Fatalf("%s: pgid line %q: %v", shell, lines[1], err)
	}
	return lines[0], pg
}

// The setsid branch puts the command in a group of its own and reports it.
// It runs the real util-linux setsid where `setsid -w` works (Linux). Where it
// does not (macOS), the same branch string runs with its setsid call swapped
// for perl doing what setsid -w does (fork, setsid, exec, wait), so the
// branch's quoting and marker are still executed.
func TestProcessGroupSetsidBranch(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	script := setsidBranch(groupProbeCmd)
	if exec.Command("sh", "-c", "setsid -w true").Run() != nil {
		if _, err := exec.LookPath("perl"); err != nil {
			t.Skip("neither setsid -w (util-linux >= 2.31) nor perl available; setsid branch not exercised")
		}
		const call = "exec setsid -w "
		if !strings.HasPrefix(script, call) {
			t.Fatalf("setsid branch %q does not start with %q", script, call)
		}
		script = "exec perl -MPOSIX -e '$p = fork; if (!$p) { POSIX::setsid(); exec @ARGV or exit 127 } " +
			"waitpid($p, 0); exit($? >> 8)' " + strings.TrimPrefix(script, call)
	}
	marker, own := runBranch(t, "sh", script)
	pg, ok := parsePgidLine(marker)
	if !ok || pg != own {
		t.Fatalf("marker %q, command's own pgid %d", marker, own)
	}
}

// With a shell that grants job control without a tty (bash, and macOS sh,
// which is bash), the job-control branch reports the command's real group.
func TestProcessGroupJobControlBranch(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not installed")
	}
	marker, own := runBranch(t, "bash", jobControlBranch(groupProbeCmd))
	pg, ok := parsePgidLine(marker)
	if !ok || pg != own {
		t.Fatalf("marker %q, command's own pgid %d", marker, own)
	}
}

func TestParsePgidLine(t *testing.T) {
	for line, want := range map[string]int{
		pgidMarker + "1234":   1234,
		pgidMarker + "none":   0,
		pgidMarker + "1":      0,
		pgidMarker + "0":      0,
		pgidMarker + "-5":     0,
		"1234":                0,
		"other " + pgidMarker: 0,
	} {
		got, ok := parsePgidLine(line)
		if ok != (want != 0) || got != want && ok {
			t.Errorf("parsePgidLine(%q) = %d, %v; want %d", line, got, ok, want)
		}
	}
}

// The marker is swallowed even when it arrives split across writes, and a
// first line that is not a marker reaches both the buffer and the stream.
func TestLineStreamerFirstLineInterception(t *testing.T) {
	var seen string
	var buf bytes.Buffer
	var lines []string
	w := &lineStreamer{buf: &buf, fn: func(s string) { lines = append(lines, s) },
		onFirst: func(l string) bool { seen = l; return strings.HasPrefix(l, pgidMarker) }}
	for _, chunk := range []string{"jumpgate-", "pgid:42\r", "\nout", "put\n"} {
		_, _ = w.Write([]byte(chunk))
	}
	w.Flush()
	if seen != pgidMarker+"42" || buf.String() != "output\n" || len(lines) != 1 || lines[0] != "output" {
		t.Fatalf("seen %q, buf %q, lines %q", seen, buf.String(), lines)
	}

	buf.Reset()
	lines = nil
	w = &lineStreamer{buf: &buf, fn: func(s string) { lines = append(lines, s) },
		onFirst: func(string) bool { return false }}
	_, _ = w.Write([]byte("a\nb"))
	w.Flush()
	if buf.String() != "a\nb" || len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Fatalf("buf %q, lines %q", buf.String(), lines)
	}

	// An unterminated first line is output, not a marker.
	buf.Reset()
	w = &lineStreamer{buf: &buf, onFirst: func(string) bool { t.Fatal("onFirst called"); return true }}
	_, _ = w.Write([]byte("partial"))
	w.Flush()
	if buf.String() != "partial" {
		t.Fatalf("buf %q", buf.String())
	}
}
