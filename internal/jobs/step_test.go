package jobs

import (
	"os"
	"reflect"
	"testing"
)

func TestStepArgvRoundTrips(t *testing.T) {
	a := StepArgs{Job: "0123456789abcdef", Step: "rename-account", Attempt: 3}
	argv := StepArgv("/usr/local/lib/jumpgate/jumpgate", "/var/lib/jumpgate", "/etc/jumpgate", a)
	if argv[1] != "agent" || argv[2] != "job-step" {
		t.Fatalf("argv %v", argv)
	}
	got, err := ParseStepArgs(argv)
	if err != nil || !reflect.DeepEqual(got, a) {
		t.Fatalf("ParseStepArgs = %+v, %v", got, err)
	}
}

// Nothing but a valid job id, a valid step id and a small integer parses: the
// same three checks the agent applies before it builds a unit's argv.
func TestParseStepArgsRefusesAnythingElse(t *testing.T) {
	for _, argv := range [][]string{
		{"--job", "../x", "--step", "one", "--attempt", "0"},
		{"--job", "0123456789ABCDEF", "--step", "one", "--attempt", "0"},
		{"--job", "0123456789abcdef", "--step", "One", "--attempt", "0"},
		{"--job", "0123456789abcdef", "--step", "-one", "--attempt", "0"},
		{"--job", "0123456789abcdef", "--step", "one", "--attempt", "-1"},
		{"--job", "0123456789abcdef", "--step", "one", "--attempt", "+1"},
		{"--job", "0123456789abcdef", "--step", "one", "--attempt", "10000"},
		{"--job", "0123456789abcdef", "--step", "one"},
		{"--job", "0123456789abcdef", "--attempt", "0"},
	} {
		if a, err := ParseStepArgs(argv); err == nil {
			t.Errorf("ParseStepArgs(%q) = %+v, want an error", argv, a)
		}
	}
}

// A step run by hand (or by a stale unit) out of turn writes nothing.
func TestRunStepRefusesAStepTheAgentDidNotAskFor(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	for _, a := range []StepArgs{
		{Job: rec.ID, Step: "two", Attempt: 0}, // not the job's step
		{Job: rec.ID, Step: "one", Attempt: 0}, // the job's step, but nobody advanced it
	} {
		if err := RunStep(t.Context(), g.dir, a, Env{}); err == nil {
			t.Fatalf("%+v ran while the job waits at step one", a)
		}
		p, _ := ResultPath(g.dir, a.Job, a.Step, a.Attempt)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%+v wrote a result: %v", a, err)
		}
	}
}
