package jobs

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// recExec records commands and answers each by the first matching prefix.
type recExec struct {
	cmds    []string
	answers map[string]executor.Result
}

func (r *recExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	r.cmds = append(r.cmds, cmd)
	for p, res := range r.answers {
		if strings.HasPrefix(cmd, p) {
			return res, nil
		}
	}
	return executor.Result{}, nil
}
func (r *recExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (r *recExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (r *recExec) Close() error                                                 { return nil }

func TestUnitNames(t *testing.T) {
	if got := UnitName("0123456789abcdef", "stop-units", 2); got != "jumpgate-job-0123456789abcdef-stop-units-2" {
		t.Fatal(got)
	}
	if UnitPattern("0123456789abcdef") != "jumpgate-job-0123456789abcdef-*" {
		t.Fatal(UnitPattern("0123456789abcdef"))
	}
	for _, s := range []string{"", "Stop", "a b", "-x", strings.Repeat("a", 33), "a;b"} {
		if ValidStepID(s) {
			t.Errorf("ValidStepID(%q)", s)
		}
	}
}

func TestSystemdRunnerStartQuotesEveryArgument(t *testing.T) {
	ex := &recExec{}
	r := SystemdRunner{Exec: ex}
	err := r.Start(context.Background(), "jumpgate-job-0123456789abcdef-one-0",
		[]string{"/usr/local/lib/jumpgate/jumpgate", "agent", "job-step", "--job", "0123456789abcdef", "--state-dir", "/var/lib/it's"})
	if err != nil {
		t.Fatal(err)
	}
	want := "systemd-run --unit=jumpgate-job-0123456789abcdef-one-0 --description='jumpgate job 0123456789abcdef step one-0' --collect --quiet --property=Type=exec -- '/usr/local/lib/jumpgate/jumpgate' 'agent' 'job-step' '--job' '0123456789abcdef' '--state-dir' '/var/lib/it'\\''s'"
	if ex.cmds[0] != want {
		t.Fatalf("command\n%s\nwant\n%s", ex.cmds[0], want)
	}
}

func TestSystemdRunnerRefusesABadUnitName(t *testing.T) {
	if err := (SystemdRunner{Exec: &recExec{}}).Start(context.Background(), "x; rm -rf /", nil); err == nil {
		t.Fatal("a unit name with a shell metacharacter was accepted")
	}
}

func TestSystemdRunnerStartReportsTheExit(t *testing.T) {
	ex := &recExec{answers: map[string]executor.Result{"systemd-run": {ExitCode: 1, Stderr: "Unit jumpgate-job-x already exists."}}}
	err := SystemdRunner{Exec: ex}.Start(context.Background(), "jumpgate-job-0123456789abcdef-one-0", []string{"/bin/true"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
}

// A bus error must never read as "finished": the agent would record the step
// as lost and ask a person about a step that is still running.
func TestSystemdRunnerActiveKnowsInactiveFromUnknown(t *testing.T) {
	for out, want := range map[string]struct {
		active bool
		err    bool
	}{
		"active\n": {true, false}, "activating\n": {true, false}, "deactivating\n": {true, false},
		"inactive\n": {false, false}, "failed\n": {false, false},
		"": {false, true}, "Failed to connect to bus\n": {false, true},
	} {
		ex := &recExec{answers: map[string]executor.Result{"systemctl is-active": {Stdout: out, ExitCode: 3}}}
		got, err := SystemdRunner{Exec: ex}.Active(context.Background(), "jumpgate-job-0123456789abcdef-one-0")
		if got != want.active || (err != nil) != want.err {
			t.Errorf("is-active %q: %v, %v", out, got, err)
		}
	}
}

func TestSystemdRunnerSupportedNamesWhatIsMissing(t *testing.T) {
	ex := &recExec{answers: map[string]executor.Result{"test -d /run/systemd/system": {ExitCode: 1}}}
	err := SystemdRunner{Exec: ex}.Supported(context.Background())
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "systemd") {
		t.Fatalf("Supported = %v", err)
	}
}

// AnyActive lists the job's units by its pattern, built from a checked id,
// and reads each one's active state; anything it cannot place is running.
func TestSystemdRunnerAnyActive(t *testing.T) {
	const id = "0123456789abcdef"
	for out, want := range map[string]struct {
		active bool
		err    bool
	}{
		"": {false, false},
		"jumpgate-job-0123456789abcdef-one-0.service loaded inactive dead jumpgate job\n":           {false, false},
		"jumpgate-job-0123456789abcdef-one-0.service loaded failed failed jumpgate job\n":           {false, false},
		"jumpgate-job-0123456789abcdef-one-0.service loaded active running jumpgate job\n":          {true, false},
		"\u25cf jumpgate-job-0123456789abcdef-one-0.service loaded activating start jumpgate job\n": {true, false},
		"garbage\n": {false, true},
	} {
		ex := &recExec{answers: map[string]executor.Result{"systemctl list-units": {Stdout: out}}}
		got, err := SystemdRunner{Exec: ex}.AnyActive(context.Background(), id)
		if got != want.active || (err != nil) != want.err {
			t.Errorf("list-units %q: %v, %v", out, got, err)
		}
		if len(ex.cmds) != 1 || !strings.Contains(ex.cmds[0], "'jumpgate-job-"+id+"-*'") {
			t.Errorf("command %v", ex.cmds)
		}
	}
	if _, err := (SystemdRunner{Exec: &recExec{}}).AnyActive(context.Background(), "x; rm -rf /"); err == nil {
		t.Fatal("a bad job id was accepted")
	}
	ex := &recExec{answers: map[string]executor.Result{"systemctl list-units": {ExitCode: 1, Stderr: "Failed to connect to bus"}}}
	if _, err := (SystemdRunner{Exec: ex}).AnyActive(context.Background(), id); err == nil {
		t.Fatal("a failed listing read as no unit")
	}
}

// errExec fails every command, as a lost SSH session or a dead executor does.
type errExec struct{ recExec }

func (e *errExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	e.cmds = append(e.cmds, cmd)
	return executor.Result{}, errors.New("executor gone")
}

// Each reason a box cannot run jobs is named, so the refusal tells the
// operator what to fix; a box with both pieces is supported. A command that
// could not run at all is that error, not an ErrUnsupported verdict about the
// box.
func TestSystemdRunnerSupportedChecksEachPiece(t *testing.T) {
	ex := &recExec{}
	if err := (SystemdRunner{Exec: ex}).Supported(context.Background()); err != nil {
		t.Fatalf("a box with systemd and systemd-run: %v", err)
	}
	if strings.Join(ex.cmds, ";") != "test -d /run/systemd/system;command -v systemd-run" {
		t.Fatalf("commands %v", ex.cmds)
	}
	ex = &recExec{answers: map[string]executor.Result{"command -v systemd-run": {ExitCode: 1}}}
	err := SystemdRunner{Exec: ex}.Supported(context.Background())
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "systemd-run is not installed") {
		t.Fatalf("no systemd-run: %v", err)
	}
	ex = &recExec{answers: map[string]executor.Result{"test -d /run/systemd/system": {ExitCode: 1}}}
	err = SystemdRunner{Exec: ex}.Supported(context.Background())
	if !strings.Contains(err.Error(), "/run/systemd/system is absent") || len(ex.cmds) != 1 {
		t.Fatalf("no systemd: %v after %v", err, ex.cmds)
	}
	for _, failAt := range []string{"test -d", "command -v"} {
		ex := &failingAt{prefix: failAt}
		err := SystemdRunner{Exec: ex}.Supported(context.Background())
		if err == nil || errors.Is(err, ErrUnsupported) {
			t.Errorf("%s could not run: Supported = %v", failAt, err)
		}
	}
}

// failingAt runs every command but the one with prefix, which errors.
type failingAt struct {
	recExec
	prefix string
}

func (f *failingAt) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	if strings.HasPrefix(cmd, f.prefix) {
		return executor.Result{}, errors.New("executor gone")
	}
	return f.recExec.Run(ctx, cmd, o)
}

// A step whose systemd-run could not even be sent is an error naming the
// unit, never a started step.
func TestSystemdRunnerStartReportsAnExecutorError(t *testing.T) {
	err := SystemdRunner{Exec: &errExec{}}.Start(context.Background(), "jumpgate-job-0123456789abcdef-one-0", []string{"/bin/true"})
	if err == nil || !strings.Contains(err.Error(), "jumpgate-job-0123456789abcdef-one-0") {
		t.Fatalf("Start = %v", err)
	}
	if err := (SystemdRunner{Exec: &recExec{}}).Start(context.Background(), "jumpgate-job-0123456789abcdef-one-0", nil); err == nil {
		t.Fatal("a unit with no command was started")
	}
}

// Active refuses a name that is not a job unit before it reaches a command
// line, and an executor error stays an error ("cannot tell").
func TestSystemdRunnerActiveRefusesForeignUnitsAndReportsErrors(t *testing.T) {
	ex := &recExec{}
	if _, err := (SystemdRunner{Exec: ex}).Active(context.Background(), "sshd; reboot"); err == nil || len(ex.cmds) != 0 {
		t.Fatalf("Active on a foreign unit = %v, ran %v", err, ex.cmds)
	}
	if _, err := (SystemdRunner{Exec: &errExec{}}).Active(context.Background(), "jumpgate-job-0123456789abcdef-one-0"); err == nil {
		t.Fatal("an executor error read as a verdict")
	}
	if _, err := (SystemdRunner{Exec: &errExec{}}).AnyActive(context.Background(), "0123456789abcdef"); err == nil {
		t.Fatal("an executor error read as no unit running")
	}
}
