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
