package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/logwatch"
	"github.com/valve-tech/jumpgate/internal/ops"
)

func TestLogsSinceCommand(t *testing.T) {
	cmd, err := logsSinceCommand("", 50)
	if err != nil || !strings.Contains(cmd, "-n 50") || strings.Contains(cmd, "--after-cursor") || !strings.Contains(cmd, "-o json") {
		t.Fatalf("no cursor: %q %v", cmd, err)
	}
	cmd, _ = logsSinceCommand("s=ab12;i=9f;b=cd;m=1a;t=5;x=77", 50)
	if !strings.Contains(cmd, "--after-cursor 's=ab12;i=9f;b=cd;m=1a;t=5;x=77'") {
		t.Fatalf("cursor: %q", cmd)
	}
	for _, bad := range []string{"x'; rm -rf /", "a b", strings.Repeat("a", 600)} {
		if _, err := logsSinceCommand(bad, 50); err == nil {
			t.Errorf("accepted cursor %q", bad)
		}
	}
	if cmd, _ := logsSinceCommand("", 99999); !strings.Contains(cmd, "-n 2000") {
		t.Errorf("n not capped: %q", cmd)
	}
}

// The command names exactly the node's units, as logs.read does: a payload
// has no way to choose a unit or add a journalctl argument.
func TestLogsSinceCommandReadsOnlyTheNodeUnits(t *testing.T) {
	cmd, err := logsSinceCommand("", 0)
	if err != nil {
		t.Fatal(err)
	}
	want := "journalctl"
	for _, u := range ops.NodeUnits() {
		want += " -u " + u
	}
	want += " -n " + fmt.Sprint(intent.LogsDefaultN) + " --no-pager -o json"
	if cmd != want {
		t.Fatalf("command\n got %q\nwant %q", cmd, want)
	}
}

func TestParseJournalJSON(t *testing.T) {
	out := `{"__CURSOR":"s=1","__REALTIME_TIMESTAMP":"1700000000000000","_SYSTEMD_UNIT":"valve-node-app-exec.service","MESSAGE":"Imported new chain segment"}
{"__CURSOR":"s=2","__REALTIME_TIMESTAMP":"1700000001000000","_SYSTEMD_UNIT":"valve-node-app-beacon.service","MESSAGE":[104,105]}
not json
`
	classify := func(unit, line string, at time.Time) logwatch.Hit {
		return logwatch.Hit{Unit: unit, Line: line, At: at, Severity: "info"}
	}
	hits, cursor := parseJournalJSON(out, classify)
	if cursor != "s=2" || len(hits) != 2 {
		t.Fatalf("cursor %q hits %+v", cursor, hits)
	}
	if hits[1].Line != "hi" || !hits[0].At.Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("hits %+v (byte-array MESSAGE and timestamps)", hits)
	}
}

// systemd's own lines about a unit ("Started …") carry UNIT, not
// _SYSTEMD_UNIT; they still belong to that unit.
func TestParseJournalJSONNamesSystemdsLinesByTheirUnit(t *testing.T) {
	out := `{"__CURSOR":"s=1","__REALTIME_TIMESTAMP":"1700000000000000","_SYSTEMD_UNIT":"init.scope","UNIT":"valve-node-app-exec.service","MESSAGE":"Started."}` + "\n"
	hits, _ := parseJournalJSON(out, func(unit, line string, at time.Time) logwatch.Hit { return logwatch.Hit{Unit: unit, Line: line} })
	if len(hits) != 1 || hits[0].Unit != "valve-node-app-exec.service" {
		t.Fatalf("hits %+v", hits)
	}
}

func TestLogsSinceIsARoutineKindAndAgentInfoNamesTheChain(t *testing.T) {
	r := newRig(t, true)
	r.exec.res = executor.Result{Stdout: `{"__CURSOR":"s=9","__REALTIME_TIMESTAMP":"1700000000000000","_SYSTEMD_UNIT":"u","MESSAGE":"m"}` + "\n"}
	rec, body := r.send(t, intent.KindLogsSince, intent.LogsSincePayload{N: 10}, nil)
	var res LogsSinceResult
	if rec.Status != intent.StatusOK || json.Unmarshal(body, &res) != nil || res.Cursor != "s=9" || len(res.Hits) != 1 {
		t.Fatalf("status %d body %s", rec.Status, body)
	}
	rec, body = r.send(t, intent.KindAgentInfo, struct{}{}, nil)
	var info intent.AgentInfo
	if rec.Status != intent.StatusOK || json.Unmarshal(body, &info) != nil || info.ChainID != 369 {
		t.Fatalf("agent.info %s", body)
	}
	if DefaultRequirements[intent.KindLogsSince] == nil || DefaultRequirements[intent.KindLogsSince][0] != TierRoutine {
		t.Fatalf("logs.since requirement %v", DefaultRequirements[intent.KindLogsSince])
	}
}

// agent.info on a box with no node yet still answers, with no chain.
func TestAgentInfoBeforeSetupHasNoChain(t *testing.T) {
	r := newRig(t, false)
	rec, body := r.send(t, intent.KindAgentInfo, struct{}{}, nil)
	var info intent.AgentInfo
	if rec.Status != intent.StatusOK || json.Unmarshal(body, &info) != nil || info.ChainID != 0 || info.SetUp {
		t.Fatalf("agent.info %s", body)
	}
}

// A cursor that is not a journal cursor is refused before any command runs.
func TestLogsSinceRefusesABadCursor(t *testing.T) {
	r := newRig(t, true)
	rec, body := r.send(t, intent.KindLogsSince, intent.LogsSincePayload{Cursor: "x'; reboot; '", N: 10}, nil)
	var rej intent.Rejection
	if rec.Status != intent.StatusRejected || json.Unmarshal(body, &rej) != nil || rej.Code != intent.ReasonInvalidPayload {
		t.Fatalf("status %d body %s", rec.Status, body)
	}
	if len(r.exec.cmds) != 0 {
		t.Fatalf("ran %q", r.exec.cmds)
	}
}

// With nothing new, the caller keeps its place.
func TestLogsSinceKeepsTheCursorWhenNothingIsNew(t *testing.T) {
	r := newRig(t, true)
	r.exec.res = executor.Result{}
	rec, body := r.send(t, intent.KindLogsSince, intent.LogsSincePayload{Cursor: "s=7", N: 10}, nil)
	var res LogsSinceResult
	if rec.Status != intent.StatusOK || json.Unmarshal(body, &res) != nil || res.Cursor != "s=7" || len(res.Hits) != 0 {
		t.Fatalf("status %d body %s", rec.Status, body)
	}
}

func journalLine(cursor string, at time.Time, msg string) string {
	b, _ := json.Marshal(map[string]string{
		"__CURSOR": cursor, "__REALTIME_TIMESTAMP": fmt.Sprint(at.UnixMicro()),
		"_SYSTEMD_UNIT": "valve-node-app-exec.service", "MESSAGE": msg,
	})
	return string(b) + "\n"
}

// One answer is bounded in bytes: each line is cut to logsSinceMaxLine, and
// once the answer would pass logsSinceMaxBytes the rest waits for the next
// call, which resumes at the last line sent (nothing is skipped).
func TestLogsSinceIsBoundedInBytes(t *testing.T) {
	r := newRig(t, true)
	var out strings.Builder
	out.WriteString(journalLine("s=long", r.now, "x"+strings.Repeat("é", logsSinceMaxLine)))
	for i := 0; i < intent.LogsMaxN; i++ {
		out.WriteString(journalLine(fmt.Sprintf("s=%d", i), r.now, strings.Repeat("x", 4096)))
	}
	r.exec.res = executor.Result{Stdout: out.String()}
	rec, body := r.send(t, intent.KindLogsSince, intent.LogsSincePayload{N: intent.LogsMaxN}, nil)
	var res LogsSinceResult
	if rec.Status != intent.StatusOK || json.Unmarshal(body, &res) != nil {
		t.Fatalf("status %d body %.200s", rec.Status, body)
	}
	if len(body) > logsSinceMaxBytes+64<<10 {
		t.Fatalf("answer is %d bytes, cap %d", len(body), logsSinceMaxBytes)
	}
	if len(res.Hits) < 2 || len(res.Hits) > intent.LogsMaxN {
		t.Fatalf("%d hits", len(res.Hits))
	}
	if l := res.Hits[0].Line; len(l) > logsSinceMaxLine || !strings.HasPrefix(l, "xé") || strings.ContainsRune(l, utf8.RuneError) {
		t.Fatalf("long line not cut cleanly: %d bytes", len(l))
	}
	last := len(res.Hits) - 2 // hits[0] is s=long
	if res.Cursor != fmt.Sprintf("s=%d", last) {
		t.Fatalf("cursor %q, want the last line sent (s=%d)", res.Cursor, last)
	}
}

// A cursor cannot page back through history: lines older than
// logsSinceMaxAge are passed over (the cursor still moves past them).
func TestLogsSinceAfterACursorSkipsOldLines(t *testing.T) {
	r := newRig(t, true)
	old := r.now.Add(-logsSinceMaxAge - time.Hour)
	r.exec.res = executor.Result{Stdout: journalLine("s=1", old, "ancient") + journalLine("s=2", r.now.Add(-time.Minute), "fresh") + journalLine("s=3", old, "ancient")}
	rec, body := r.send(t, intent.KindLogsSince, intent.LogsSincePayload{Cursor: "s=0", N: 10}, nil)
	var res LogsSinceResult
	if rec.Status != intent.StatusOK || json.Unmarshal(body, &res) != nil {
		t.Fatalf("status %d body %s", rec.Status, body)
	}
	if len(res.Hits) != 1 || res.Hits[0].Line != "fresh" || res.Cursor != "s=3" {
		t.Fatalf("hits %+v cursor %q", res.Hits, res.Cursor)
	}
}
