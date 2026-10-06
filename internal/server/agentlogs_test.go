package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/logwatch"
)

// journalJSON is one journalctl -o json line stamped now: the agent passes
// over lines older than a day once it reads after a cursor.
func journalJSON(cursor, msg string) string {
	return fmt.Sprintf(`{"__CURSOR":%q,"__REALTIME_TIMESTAMP":"%d","_SYSTEMD_UNIT":"u","MESSAGE":%q}`+"\n", cursor, time.Now().UnixMicro(), msg)
}

// journalExec answers the agent's journalctl -o json with two lines,
// logs.read's journalctl -o cat with one, and everything else with nothing.
type journalExec struct{ nopExec }

func (journalExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	switch {
	case strings.Contains(cmd, "-o json"):
		return executor.Result{Stdout: journalJSON("s=1", "one") + journalJSON("s=2", "two")}, nil
	case strings.Contains(cmd, "-o cat"):
		return executor.Result{Stdout: "a plain line\n"}, nil
	}
	return executor.Result{}, nil
}

// cursorJournal is a journal that honours the cursor: "one" and "two" are
// history, "three" arrives after them, then nothing more. It counts the
// logs.since calls, and those without a cursor (a follower's first poll).
type cursorJournal struct {
	nopExec
	calls, fresh atomic.Int32
}

func (j *cursorJournal) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	if !strings.Contains(cmd, "-o json") {
		return executor.Result{}, nil
	}
	j.calls.Add(1)
	switch {
	case strings.Contains(cmd, "--after-cursor 's=3'"):
		return executor.Result{}, nil
	case strings.Contains(cmd, "--after-cursor 's=2'"):
		return executor.Result{Stdout: journalJSON("s=3", "three")}, nil
	default:
		j.fresh.Add(1)
		return executor.Result{Stdout: journalJSON("s=1", "one") + journalJSON("s=2", "two")}, nil
	}
}

// waitCalls waits until j has answered at least n logs.since calls.
func (j *cursorJournal) waitCalls(t *testing.T, n int32) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for j.calls.Load() < n {
		if time.Now().After(deadline) {
			t.Fatalf("%d logs.since calls, want %d", j.calls.Load(), n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// settledCalls is the call count once a request the follower abandoned as
// it stopped has had time to reach the agent's executor anyway.
func (j *cursorJournal) settledCalls() int32 {
	time.Sleep(5 * agentLogsFollowInterval)
	return j.calls.Load()
}

// fastFollow shortens the follower's timing. Its cleanup is registered
// before the server's, so it runs after pairedBoxWith's cleanup has closed
// the server and stopped every follower (Ruling T8); a follower reads only
// the copies it took when it started anyway.
func fastFollow(t *testing.T) {
	t.Helper()
	oldF, oldL := agentLogsFollowInterval, logFollowerLinger
	agentLogsFollowInterval, logFollowerLinger = 20*time.Millisecond, 50*time.Millisecond
	t.Cleanup(func() { agentLogsFollowInterval, logFollowerLinger = oldF, oldL })
}

func isFrameLine(l string) bool {
	return strings.HasPrefix(l, "data:") || strings.HasPrefix(l, "event:")
}

func TestPairedLogsStreamFollowsTheJournal(t *testing.T) {
	requireAgentPeer(t)
	fastFollow(t)
	ts, token := pairedBox(t, journalExec{}, true)
	frames, h := readFrames(t, ts, token, "/api/targets/box/logs/stream?backlog=10", 2)
	if h.Get("X-Jumpgate-Via") != "agent" {
		t.Fatalf("via %q", h.Get("X-Jumpgate-Via"))
	}
	if !strings.HasPrefix(frames[0], "event: reset\n") || !strings.Contains(frames[0], `"line":"one"`) {
		t.Fatalf("reset frame %q", frames[0])
	}
	if !strings.HasPrefix(frames[1], "data: {") {
		t.Fatalf("live frame %q", frames[1])
	}
}

// The web UI opens the stream with no backlog, after fetching the recent
// lines itself, and reads only default events: it must get each new line
// once, as a default event, and none of the history it already has.
func TestPairedLogsStreamWithoutABacklogSendsOnlyNewLinesAsDefaultEvents(t *testing.T) {
	requireAgentPeer(t)
	fastFollow(t)
	j := &cursorJournal{}
	ts, token := pairedBox(t, j, true)
	r, stop := openSSE(t, &apiTestServer{ts: ts, token: token}, "/api/targets/box/logs/stream")
	defer stop()
	var first string
	if !readLines(r, 5*time.Second, func(l string) bool { first = l; return isFrameLine(l) }) {
		t.Fatal("no frame")
	}
	if !strings.HasPrefix(first, "data: {") || !strings.Contains(first, `"line":"three"`) {
		t.Fatalf("first frame line %q, want the new line as a default event", first)
	}
	// Several more polls answer nothing new: nothing more is sent.
	j.waitCalls(t, j.calls.Load()+3)
	var extra atomic.Value
	if readLines(r, 150*time.Millisecond, func(l string) bool { extra.Store(l); return isFrameLine(l) }) {
		t.Fatalf("sent again: %q", extra.Load())
	}
}

// ?backlog=0 is an empty reset, then live lines; ?backlog=1 is the newest
// line of history (Task 3's meaning, kept on paired boxes).
func TestPairedLogsStreamBacklogZeroIsAnEmptyReset(t *testing.T) {
	requireAgentPeer(t)
	fastFollow(t)
	ts, token := pairedBox(t, &cursorJournal{}, true)
	frames, _ := readFrames(t, ts, token, "/api/targets/box/logs/stream?backlog=0", 2)
	if frames[0] != "event: reset\ndata: []\n" || !strings.HasPrefix(frames[1], "data: {") || !strings.Contains(frames[1], `"line":"three"`) {
		t.Fatalf("frames %q", frames)
	}
	frames, _ = readFrames(t, ts, token, "/api/targets/box/logs/stream?backlog=1", 1)
	if !strings.HasPrefix(frames[0], "event: reset\n") || !strings.Contains(frames[0], `"line":"three"`) || strings.Contains(frames[0], `"line":"two"`) {
		t.Fatalf("backlog=1: %q", frames[0])
	}
}

// One follower serves every viewer of a box, and it stops polling
// logFollowerLinger after the last one leaves; a viewer who comes back
// within the linger finds it still running.
func TestFollowerIsSharedAndStopsAfterTheLastViewerLeaves(t *testing.T) {
	requireAgentPeer(t)
	fastFollow(t)
	logFollowerLinger = time.Second
	j := &cursorJournal{}
	var srv *Server
	ts, token := pairedBoxWith(t, j, pairedOpts{setUp: true, onServer: func(s *Server) { srv = s }})
	a := &apiTestServer{ts: ts, token: token}
	r1, stop1 := openSSE(t, a, "/api/targets/box/logs/stream?backlog=0")
	r2, stop2 := openSSE(t, a, "/api/targets/box/logs/stream?backlog=0")
	if !readLines(r1, 5*time.Second, func(l string) bool { return strings.Contains(l, `"three"`) }) ||
		!readLines(r2, 5*time.Second, func(l string) bool { return strings.Contains(l, `"three"`) }) {
		t.Fatal("a viewer did not get the new line")
	}
	stop1()
	stop2()
	frames, _ := readFrames(t, ts, token, "/api/targets/box/logs/stream?backlog=1", 1)
	if !strings.Contains(frames[0], `"three"`) || j.fresh.Load() != 1 {
		t.Fatalf("returning viewer: %q, %d first polls (want the same follower)", frames[0], j.fresh.Load())
	}
	entry := srv.reg.get("box")
	deadline := time.Now().Add(5 * time.Second)
	for {
		entry.mu.Lock()
		f := entry.logs
		entry.mu.Unlock()
		if f == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the follower is still running with no viewer")
		}
		time.Sleep(10 * time.Millisecond)
	}
	n := j.settledCalls()
	time.Sleep(10 * agentLogsFollowInterval)
	if j.calls.Load() != n {
		t.Fatalf("still polling: %d calls, then %d", n, j.calls.Load())
	}
}

// An agent from before logs.since answers unknown_kind; the stream falls back
// to snapshots and says why. The requirement is restored only after the
// server and its agent are closed (Ruling T8): the cleanup is registered
// before pairedBox's.
func TestOldAgentsGetSnapshotModeWithAnUpgradeNote(t *testing.T) {
	requireAgentPeer(t)
	fastFollow(t)
	req := agent.DefaultRequirements[intent.KindLogsSince]
	delete(agent.DefaultRequirements, intent.KindLogsSince)
	t.Cleanup(func() { agent.DefaultRequirements[intent.KindLogsSince] = req })
	ts, token := pairedBox(t, journalExec{}, true)
	frames, _ := readFrames(t, ts, token, "/api/targets/box/logs/stream?backlog=10", 2)
	if !strings.HasPrefix(frames[0], "event: note\n") || !strings.Contains(frames[0], "re-pair") || !strings.HasPrefix(frames[1], "event: reset\n") || !strings.Contains(frames[1], "a plain line") {
		t.Fatalf("frames %q", frames)
	}
	// backlog=0 keeps its meaning in snapshot mode: the first reset is empty.
	frames, _ = readFrames(t, ts, token, "/api/targets/box/logs/stream?backlog=0", 2)
	if !strings.HasPrefix(frames[0], "event: note\n") || frames[1] != "event: reset\ndata: []\n" {
		t.Fatalf("backlog=0: %q", frames)
	}
}

func TestFollowerRingIsBounded(t *testing.T) {
	f := &logFollower{}
	for i := 0; i < logRingSize+10; i++ {
		f.append(logwatch.Hit{Line: "x"})
	}
	if len(f.ring) != logRingSize {
		t.Fatalf("ring %d", len(f.ring))
	}
}

// A follower's poll is bounded well inside apiclient's 45 s idle timeout.
func TestFollowerPollsCarryADeadline(t *testing.T) {
	if agentPollTimeout <= 0 || agentPollTimeout >= 30*time.Second {
		t.Fatalf("agentPollTimeout %s", agentPollTimeout)
	}
	f := newLogFollower(func() {})
	if f.pollTimeout != agentPollTimeout || f.interval != agentLogsFollowInterval || f.linger != logFollowerLinger {
		t.Fatalf("follower timing %+v", f)
	}
}

// A box unpaired while its stream is open is not asked again, by the agent
// or by SSH: the follower stops, the stream ends, and the client's reconnect
// picks whatever transport the target has now.
func TestFollowerStopsWhenTheBoxIsUnpaired(t *testing.T) {
	requireAgentPeer(t)
	fastFollow(t)
	j := &cursorJournal{}
	var legacy countingExec
	ts, token := pairedBoxWith(t, j, pairedOpts{setUp: true,
		server: func(c *Config) { c.NewExecutor = legacy.newExecutor },
		target: func(tg *config.Target) { tg.Wire = wiredForLegacy }})
	r, stop := openSSE(t, &apiTestServer{ts: ts, token: token}, "/api/targets/box/logs/stream?backlog=0")
	defer stop()
	if !readLines(r, 5*time.Second, func(l string) bool { return strings.Contains(l, `"three"`) }) {
		t.Fatal("no live line")
	}
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets[0].Agent = nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		for {
			if _, err := r.ReadString('\n'); err != nil {
				return
			}
		}
	}()
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the stream outlived the pairing")
	}
	n := j.settledCalls()
	time.Sleep(10 * agentLogsFollowInterval)
	if j.calls.Load() != n || legacy.n.Load() != 0 {
		t.Fatalf("after unpairing: agent calls %d -> %d, legacy executor opened %d times", n, j.calls.Load(), legacy.n.Load())
	}
}

// staleCursorJournal answers a fresh read with history and fails every read
// after a cursor, as journalctl does with a cursor from another journal (a
// box re-paired onto a reinstalled system, a cursor that no longer seeks).
type staleCursorJournal struct {
	nopExec
	fresh atomic.Int32
}

func (j *staleCursorJournal) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	if !strings.Contains(cmd, "-o json") {
		return executor.Result{}, nil
	}
	if strings.Contains(cmd, "--after-cursor") {
		return executor.Result{}, errors.New("Failed to seek to cursor: Invalid argument")
	}
	j.fresh.Add(1)
	return executor.Result{Stdout: journalJSON("s=1", "one") + journalJSON("s=2", "two")}, nil
}

// After logsCursorRetries polls in a row fail after a cursor, the follower
// drops the cursor and starts again from the newest lines: one coded error
// says so, and a backlog viewer gets a fresh reset rather than the history
// again as live lines.
func TestFollowerDropsACursorThatKeepsFailing(t *testing.T) {
	requireAgentPeer(t)
	fastFollow(t)
	j := &staleCursorJournal{}
	ts, token := pairedBox(t, j, true)
	r, stop := openSSE(t, &apiTestServer{ts: ts, token: token}, "/api/targets/box/logs/stream?backlog=5")
	defer stop()
	var frames []string
	var cur strings.Builder
	sawRestart := false
	ok := readLines(r, 5*time.Second, func(l string) bool {
		if l != "\n" {
			cur.WriteString(l)
			return false
		}
		f := cur.String()
		cur.Reset()
		if strings.HasPrefix(f, ": ") {
			return false
		}
		frames = append(frames, f)
		if strings.HasPrefix(f, "event: error\n") && strings.Contains(f, "newest lines") {
			sawRestart = true
			return false
		}
		return sawRestart && strings.HasPrefix(f, "event: reset\n")
	})
	if !ok {
		t.Fatalf("frames %q", frames)
	}
	errs, restarts := 0, 0
	for _, f := range frames {
		switch {
		case strings.Contains(f, "newest lines"):
			restarts++
			if !strings.Contains(f, `"code":"agent_failed"`) {
				t.Errorf("restart error is not coded: %q", f)
			}
		case strings.HasPrefix(f, "event: error\n"):
			errs++
		case !strings.HasPrefix(f, "event: reset\n"):
			t.Errorf("unexpected frame %q (history sent as live lines?)", f)
		}
	}
	last := frames[len(frames)-1]
	if restarts != 1 || errs != logsCursorRetries-1 || !strings.Contains(last, `"line":"two"`) || j.fresh.Load() < 2 {
		t.Fatalf("%d restarts, %d errors, %d fresh reads; frames %q", restarts, errs, j.fresh.Load(), frames)
	}
}

// A config the server cannot read stops the follower rather than leaving it
// asking a target it can no longer confirm: the stream ends on a coded error.
func TestFollowerStopsWhenTheConfigCannotBeRead(t *testing.T) {
	requireAgentPeer(t)
	fastFollow(t)
	j := &cursorJournal{}
	ts, token := pairedBox(t, j, true)
	r, stop := openSSE(t, &apiTestServer{ts: ts, token: token}, "/api/targets/box/logs/stream?backlog=0")
	defer stop()
	if !readLines(r, 5*time.Second, func(l string) bool { return strings.Contains(l, `"three"`) }) {
		t.Fatal("no live line")
	}
	home, _ := os.UserHomeDir()
	if err := os.WriteFile(filepath.Join(home, ".jumpgate", "config.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	var lines []string
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		for {
			l, err := r.ReadString('\n')
			if err != nil {
				return
			}
			lines = append(lines, l)
		}
	}()
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the stream outlived an unreadable config")
	}
	all := strings.Join(lines, "")
	if !strings.Contains(all, "event: error\n") || !strings.Contains(all, `"code":"internal"`) || !strings.Contains(all, "config") {
		t.Fatalf("stream ended without a coded error: %q", all)
	}
	n := j.settledCalls()
	time.Sleep(10 * agentLogsFollowInterval)
	if j.calls.Load() != n {
		t.Fatalf("still polling after the config broke: %d -> %d", n, j.calls.Load())
	}
}
