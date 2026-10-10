// internal/server/jobsroutes_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// jobJournalExec answers journalctl with one journal entry the first time and
// nothing after, as a journal with one line and no new ones does. It records
// every command so a test can see what reached the box's shell.
type jobJournalExec struct {
	mu    sync.Mutex
	cmds  []string
	entry string
}

func (j *jobJournalExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cmds = append(j.cmds, cmd)
	if strings.HasPrefix(cmd, "journalctl") && !strings.Contains(cmd, "--after-cursor") {
		return executor.Result{Stdout: j.entry + "\n"}, nil
	}
	return executor.Result{}, nil
}
func (j *jobJournalExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (j *jobJournalExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (j *jobJournalExec) Close() error                                                 { return nil }

func (j *jobJournalExec) ran(substr string) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, c := range j.cmds {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

// The output stream carries the box's journal lines, with their unit and
// time, and the next poll reads after the cursor the box returned: without
// it every poll would resend the same lines.
func TestJobOutputStreamSendsJournalLinesAndFollowsTheCursor(t *testing.T) {
	requireAgentPeer(t)
	prev := jobsOutputEvery
	jobsOutputEvery = 20 * time.Millisecond
	t.Cleanup(func() { jobsOutputEvery = prev })
	ex := &jobJournalExec{}
	b := newBoxRigExec(t, "box", ex, withoutDriver)
	rec := b.startJob(t, `{}`)
	unit := "jumpgate-job-" + rec.ID + "-one-0.service"
	ex.mu.Lock()
	ex.entry = `{"__CURSOR":"s=abc1","__REALTIME_TIMESTAMP":"` +
		jsonMicros(b.clock()) + `","_SYSTEMD_UNIT":"` + unit + `","MESSAGE":"renaming the account"}`
	ex.mu.Unlock()
	// One stream, polled every 20ms: the first poll sends the line, every
	// later one asks only for what follows its cursor.
	body, code := readFramesFor(t, b, "/api/jobs/box/"+rec.ID+"/output/stream", time.Second)
	if code != http.StatusOK {
		t.Fatalf("output stream %d", code)
	}
	var frames []string
	for _, f := range strings.Split(body, "\n\n") {
		if strings.HasPrefix(f, "data: ") {
			frames = append(frames, strings.TrimPrefix(f, "data: "))
		}
	}
	if len(frames) != 1 {
		t.Fatalf("data frames %q, want the one line once", frames)
	}
	var lines []api.JobOutputLine
	if err := json.Unmarshal([]byte(frames[0]), &lines); err != nil {
		t.Fatalf("frame %q: %v", frames[0], err)
	}
	if len(lines) != 1 || lines[0].Unit != unit || lines[0].Line != "renaming the account" || lines[0].At.Unix() != b.clock().Unix() {
		t.Fatalf("lines %+v", lines)
	}
	if !ex.ran("-u 'jumpgate-job-"+rec.ID+"-*'") || !ex.ran("--after-cursor 's=abc1'") {
		t.Fatalf("the journal was not followed from the returned cursor: %v", ex.cmds)
	}
}

// jsonMicros is t as journalctl's __REALTIME_TIMESTAMP, in microseconds.
func jsonMicros(t time.Time) string {
	b, _ := json.Marshal(t.UnixMicro())
	return string(b)
}

// readFramesFor reads a stream for d and returns its raw body; the stream
// ends when d does.
func readFramesFor(t *testing.T, b *boxRig, path string, d time.Duration) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, b.ts.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+b.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := res.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			return sb.String(), res.StatusCode
		}
	}
}

// A cursor the box would not pass to journalctl is refused by the agent;
// the stream says so as an error event the client stops on, and nothing of
// the cursor reaches the box's shell.
func TestJobOutputStreamSendsTheBoxsRefusalAsAnErrorEvent(t *testing.T) {
	requireAgentPeer(t)
	ex := &jobJournalExec{}
	b := newBoxRigExec(t, "box", ex, withoutDriver)
	rec := b.startJob(t, `{}`)
	frames, _ := readFrames(t, b.ts, b.token, "/api/jobs/box/"+rec.ID+"/output/stream?cursor=x%27%3B+rm+-rf+%2F", 1)
	f := frames[0]
	if !strings.HasPrefix(f, "event: error\n") {
		t.Fatalf("frame %q is not an error event", f)
	}
	var e api.Error
	data := f[strings.Index(f, "data: ")+len("data: "):]
	if err := json.Unmarshal([]byte(strings.TrimSpace(data)), &e); err != nil || e.Code == "" || !strings.Contains(e.Message, "cursor") {
		t.Fatalf("error event %q: %+v %v", f, e, err)
	}
	if ex.ran("rm -rf") || ex.ran("journalctl") {
		t.Fatalf("the refused cursor reached the box: %v", ex.cmds)
	}
}

// A decision must name both the question and the option; a body without
// them is refused before the box is asked anything.
func TestDecideRefusesAMalformedBody(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	before := b.readOnBox(t, rec.ID).LogLen
	for _, body := range []any{"not an object", api.JobDecision{Decision: "q"}, api.JobDecision{Option: "go"}} {
		if code, raw := b.req(t, "POST", "/api/jobs/box/"+rec.ID+"/decide", body); code != http.StatusBadRequest {
			t.Errorf("decide %v = %d %s", body, code, raw)
		}
	}
	if got := b.readOnBox(t, rec.ID).LogLen; got != before {
		t.Fatalf("a refused decision reached the job's log: %d -> %d", before, got)
	}
}

// A decision for a job the box does not have is that job's not-found, not a
// generic failure.
func TestDecideOnAJobTheBoxDoesNotHave(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	var e api.Error
	if code := b.do(t, "POST", "/api/jobs/box/0123456789abcdef/decide", api.JobDecision{Decision: "q", Option: "go"}, &e); code != http.StatusNotFound || e.Code != api.CodeJobNotFound {
		t.Fatalf("decide = %d %+v", code, e)
	}
}

// A job whose lease was released is held by nobody; deciding it would act
// without the lease, so the server says to adopt it first and signs nothing.
func TestDecideOnAReleasedJobSaysToAdoptIt(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	b.send(t, intent.KindJobLease, intent.JobLeasePayload{ID: rec.ID, Op: intent.LeaseRelease})
	b.srv.jobs.refresh(context.Background(), true)
	before := b.readOnBox(t, rec.ID).LogLen
	var e api.Error
	if code := b.do(t, "POST", "/api/jobs/box/"+rec.ID+"/decide", api.JobDecision{Decision: "q", Option: "go"}, &e); code != http.StatusConflict || e.Code != api.CodeLeaseHeld || !strings.Contains(e.Message, "adopt") {
		t.Fatalf("decide = %d %+v", code, e)
	}
	if got := b.readOnBox(t, rec.ID).LogLen; got != before {
		t.Fatalf("the refused decision reached the job's log: %d -> %d", before, got)
	}
}
