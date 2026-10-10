package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// jobsServer is a stand-in for the server's jobs routes. It records each
// request as "METHOD path" and answers from routes; anything else is a 404.
func jobsServer(t *testing.T, routes map[string]func(w http.ResponseWriter, r *http.Request)) func() []string {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.EscapedPath()
		mu.Lock()
		seen = append(seen, key)
		mu.Unlock()
		if h, ok := routes[key]; ok {
			h(w, r)
			return
		}
		http.NotFound(w, r)
	})
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func replyJSON(v any) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}

func replyError(status int, code api.Code, msg string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprintf(w, `{"error":%q,"code":%q}`, msg, code)
	}
}

func twoBoxInbox() api.JobsInbox {
	return api.JobsInbox{Jobs: []api.JobView{
		{Target: "box-a", ID: "0123456789abcdef", Kind: "node.migrate-names", State: "running", Steps: make([]api.JobStepView, 3)},
		{Target: "box-b", ID: "fedcba9876543210", Kind: "node.migrate-names", State: "succeeded", Steps: make([]api.JobStepView, 3)},
	}}
}

// `jumpgate jobs HOST` lists only that box's jobs, from the server's inbox.
func TestJobsListsTheInboxForOneHost(t *testing.T) {
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){"GET /api/jobs": replyJSON(twoBoxInbox())})
	stdout, stderr := captureStdio(t)
	if code := cmdJobs([]string{"box-a"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if !strings.Contains(stdout(), "0123456789abcdef") || strings.Contains(stdout(), "box-b") {
		t.Fatalf("stdout:\n%s", stdout())
	}
}

// `jobs show` asks for the one job on the named box and prints its log.
func TestJobsShowPrintsTheJobAndItsLog(t *testing.T) {
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	d := api.JobDetail{Job: api.JobView{Target: "box-a", ID: "0123456789abcdef", Title: "Rename", State: "running", Steps: []api.JobStepView{{State: "done", Title: "Preflight"}}},
		Log: []api.JobLogLine{{Index: 7, At: at, Text: "lease taken by 0xA"}}}
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){"GET /api/jobs/box-a/0123456789abcdef": replyJSON(d)})
	stdout, stderr := captureStdio(t)
	if code := cmdJobs([]string{"show", "box-a", "0123456789abcdef"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	for _, want := range []string{"0123456789abcdef on box-a: Rename (running)", "Preflight", "7 2026-10-09T12:00:00Z lease taken by 0xA"} {
		if !strings.Contains(stdout(), want) {
			t.Errorf("stdout lacks %q:\n%s", want, stdout())
		}
	}
}

// `jobs decide` sends exactly the decision and option typed, to the job
// named, and prints the state the server answers with.
func TestJobsDecideSendsTheDecisionTyped(t *testing.T) {
	var got api.JobDecision
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/jobs/box-a/0123456789abcdef/decide": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&got)
			replyJSON(api.JobView{Target: "box-a", ID: "0123456789abcdef", State: "running"})(w, r)
		}})
	stdout, stderr := captureStdio(t)
	if code := cmdJobs([]string{"decide", "box-a", "0123456789abcdef", "stop-units-0-busy-user", "retry"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if got != (api.JobDecision{Decision: "stop-units-0-busy-user", Option: "retry"}) {
		t.Fatalf("server got %+v", got)
	}
	if !strings.Contains(stdout(), "box-a 0123456789abcdef: running") {
		t.Fatalf("stdout %q", stdout())
	}
}

func TestJobsAdoptSaysThisControllerDrivesIt(t *testing.T) {
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/jobs/box-a/0123456789abcdef/adopt": replyJSON(api.JobView{Target: "box-a", ID: "0123456789abcdef", State: "running"})})
	stdout, stderr := captureStdio(t)
	if code := cmdJobs([]string{"adopt", "box-a", "0123456789abcdef"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if !strings.Contains(stdout(), "this controller drives it now (running)") {
		t.Fatalf("stdout %q", stdout())
	}
}

// Each error class reaches the operator with its exit code. A job log that
// does not verify is a security failure (4), not an ordinary one: a script
// must not retry or act on a history the box's key does not sign.
func TestJobsExitsWithTheErrorsClass(t *testing.T) {
	cases := []struct {
		code   api.Code
		status int
		want   int
	}{
		{api.CodeJobLogInvalid, http.StatusBadGateway, api.ExitSecurity},
		{api.CodeJobNotFound, http.StatusNotFound, api.ExitFailed},
		{api.CodeJobState, http.StatusConflict, api.ExitFailed},
		{api.CodeUnreachable, http.StatusGatewayTimeout, api.ExitUnreachable},
		{api.CodeJobKindNotAllowed, http.StatusBadRequest, api.ExitUsage},
	}
	for _, c := range cases {
		t.Run(string(c.code), func(t *testing.T) {
			jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
				"GET /api/jobs/box-a/0123456789abcdef": replyError(c.status, c.code, "boom")})
			_, stderr := captureStdio(t)
			if got := cmdJobs([]string{"show", "box-a", "0123456789abcdef"}); got != c.want {
				t.Fatalf("exit %d, want %d; stderr %q", got, c.want, stderr())
			}
			if !strings.Contains(stderr(), "boom") {
				t.Fatalf("stderr %q lacks the server's message", stderr())
			}
			if security := c.want == api.ExitSecurity; security != strings.Contains(stderr(), "SECURITY") {
				t.Fatalf("SECURITY prefix = %v, want %v: %q", !security, security, stderr())
			}
		})
	}
}

// The server's error text came from a box; its escapes must not reach the
// operator's terminal, whatever field they ride in.
func TestJobsErrorsReachTheTerminalClean(t *testing.T) {
	evil := "x\x1b]52;c;AAAA\x07y\x1b[2Jz\r"
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/jobs/box-a/0123456789abcdef/adopt": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(api.Error{Message: evil, Hint: evil, Reason: evil, Code: api.CodeJobState})
		}})
	_, stderr := captureStdio(t)
	if code := cmdJobs([]string{"adopt", "box-a", "0123456789abcdef"}); code != api.ExitFailed {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if strings.ContainsAny(stderr(), "\x1b\x07\r") || !strings.Contains(stderr(), "xyz") {
		t.Fatalf("stderr %q", stderr())
	}
}

// A connection that fails before any request is an ordinary failure, and
// its text is cleaned too.
func TestJobsReportsAFailedConnection(t *testing.T) {
	old := connect
	connect = func(context.Context) (*apiclient.Client, error) { return nil, errors.New("no server\x1b[2J here") }
	t.Cleanup(func() { connect = old })
	_, stderr := captureStdio(t)
	if code := cmdJobs(nil); code != api.ExitFailed {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr(), "no server here") || strings.Contains(stderr(), "\x1b") {
		t.Fatalf("stderr %q", stderr())
	}
}

// A plain transport failure after connecting (the inbox body is not JSON) is
// reported as a failure, not swallowed as an empty inbox.
func TestJobsReportsANonAPIError(t *testing.T) {
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /api/jobs": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, "{not json")
		}})
	stdout, stderr := captureStdio(t)
	if code := cmdJobs(nil); code != api.ExitFailed {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout(), stderr())
	}
	if strings.Contains(stdout(), "HOST") {
		t.Fatalf("an unreadable inbox was printed as an empty one: %q", stdout())
	}
}

// `jobs output ID` finds the box from the inbox and follows that box's
// journal; the stream's refusal ends it with the refusal's exit code.
func TestJobsOutputFollowsTheJobOnItsBox(t *testing.T) {
	seen := jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /api/jobs": replyJSON(twoBoxInbox()),
		"GET /api/jobs/box-b/fedcba9876543210/output/stream": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: [{\"unit\":\"u.service\",\"line\":\"renamed\",\"at\":\"2026-10-09T12:00:00Z\"}]\n\n")
			fmt.Fprint(w, "event: error\ndata: {\"error\":\"job log changed\",\"code\":\"job_log_invalid\"}\n\n")
		}})
	stdout, stderr := captureStdio(t)
	if code := cmdJobs([]string{"output", "fedcba9876543210"}); code != api.ExitSecurity {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout(), stderr())
	}
	if !strings.Contains(stdout(), "2026-10-09T12:00:00Z u.service renamed") {
		t.Fatalf("stdout %q", stdout())
	}
	if s := seen(); len(s) < 2 || s[1] != "GET /api/jobs/box-b/fedcba9876543210/output/stream" {
		t.Fatalf("requests %v", s)
	}
}

// Ruling P13: an id found on two boxes is refused, never guessed, and no
// box's output is opened; an id on no box is refused too.
func TestJobsOutputRefusesAnAmbiguousOrUnknownID(t *testing.T) {
	in := twoBoxInbox()
	in.Jobs = append(in.Jobs, api.JobView{Target: "box-c", ID: "0123456789abcdef"})
	for id, want := range map[string]string{
		"0123456789abcdef": "is on more than one box (box-a, box-c)",
		"1111111111111111": "no box that answered has a job 1111111111111111",
	} {
		t.Run(id, func(t *testing.T) {
			seen := jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){"GET /api/jobs": replyJSON(in)})
			_, stderr := captureStdio(t)
			if code := cmdJobs([]string{"output", id}); code != api.ExitFailed {
				t.Fatalf("exit %d, stderr %q", code, stderr())
			}
			if !strings.Contains(stderr(), want) {
				t.Fatalf("stderr %q lacks %q", stderr(), want)
			}
			if s := seen(); len(s) != 1 {
				t.Fatalf("a stream was opened: %v", s)
			}
		})
	}
}

// When the inbox itself fails, `jobs output` reports that error rather than
// claiming the job does not exist.
func TestJobsOutputReportsAFailedInbox(t *testing.T) {
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /api/jobs": replyError(http.StatusGatewayTimeout, api.CodeUnreachable, "boxes asleep")})
	_, stderr := captureStdio(t)
	if code := cmdJobs([]string{"output", "0123456789abcdef"}); code != api.ExitUnreachable {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if strings.Contains(stderr(), "no box that answered") {
		t.Fatalf("stderr %q", stderr())
	}
}

// migrate-names starts exactly node.migrate-names on the named box and points
// at the command that follows it.
func TestMigrateNamesStartsTheJobAndSaysHowToFollowIt(t *testing.T) {
	var got api.JobStart
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/jobs/box-a": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&got)
			replyJSON(api.JobView{Target: "box-a", ID: "0123456789abcdef", State: "running"})(w, r)
		}})
	stdout, stderr := captureStdio(t)
	if code := cmdMigrateNames([]string{"--yes", "box-a"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if got.Kind != intent.KindNodeMigrateNames {
		t.Fatalf("started kind %q", got.Kind)
	}
	if !strings.Contains(stdout(), "job 0123456789abcdef started on box-a; follow it with `jumpgate 'jobs' 'box-a'` or the jobs inbox") {
		t.Fatalf("stdout %q", stdout())
	}
}

// A target name the server returns that is not shaped like one is not put
// in a command to paste; the inbox is offered instead.
func TestMigrateNamesSuggestsNoCommandForAMisshapenTarget(t *testing.T) {
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/jobs/box-a": replyJSON(api.JobView{Target: "box-a;rm -rf ~", ID: "0123456789abcdef"})})
	stdout, _ := captureStdio(t)
	if code := cmdMigrateNames([]string{"box-a", "--yes"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(stdout(), "jumpgate 'jobs'") || !strings.Contains(stdout(), "follow it with the jobs inbox") {
		t.Fatalf("stdout %q", stdout())
	}
}

// A refused start (a job already runs) exits with the refusal's class.
func TestMigrateNamesReportsARefusal(t *testing.T) {
	jobsServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/jobs/box-a": replyError(http.StatusConflict, api.CodeJobActive, "a job runs")})
	_, stderr := captureStdio(t)
	if code := cmdMigrateNames([]string{"box-a", "--yes"}); code != api.ExitFailed || !strings.Contains(stderr(), "a job runs") {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
}

// The age column of a silent lease reads in the largest whole unit, and two
// days is where hours stop.
func TestAgoReadsInTheLargestWholeUnit(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                               "0s",
		59 * time.Second:                "59s",
		time.Minute:                     "1m",
		59*time.Minute + 59*time.Second: "59m",
		time.Hour:                       "1h",
		47 * time.Hour:                  "47h",
		48 * time.Hour:                  "2d",
		10*24*time.Hour + time.Hour:     "10d",
	} {
		if got := ago(d); got != want {
			t.Errorf("ago(%v) = %q, want %q", d, got, want)
		}
	}
}
