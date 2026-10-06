package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/logwatch"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// LogsSinceResult answers logs.since: the new lines and where to resume.
type LogsSinceResult struct {
	Hits   []logwatch.Hit `json:"hits"`
	Cursor string         `json:"cursor"`
}

// Bounds on one logs.since answer, besides intent.LogsMaxN lines. A journal
// line can be up to 48 KiB, so the line count alone does not bound the
// answer's size; and a cursor names any point in the journal, so it alone
// does not bound how far back a caller can read.
const (
	logsSinceMaxLine  = 8 << 10 // bytes of one line's text
	logsSinceMaxBytes = 1 << 20 // the encoded hits of one answer
	logsSinceMaxAge   = 24 * time.Hour
)

// cursorRE is the shape of a journald cursor (s=…;i=…;b=…;m=…;t=…;x=…). A
// cursor comes from the controller, so it is checked before it reaches a
// command line, even though it is also quoted there.
var cursorRE = regexp.MustCompile(`^[A-Za-z0-9=;_-]{1,512}$`)

// logsSinceCommand is one journalctl call over both node units, interleaved
// by time, as JSON so each line keeps its timestamp and the cursor. The units
// are the node's own (ops.NodeUnits), as for logs.read: nothing in the
// payload reaches the command line except a validated cursor and a number.
// After a cursor, journalctl reads forward, so -n bounds the lines read and
// a caller that falls behind catches up over several calls.
func logsSinceCommand(cursor string, n int) (string, error) {
	if n <= 0 {
		n = intent.LogsDefaultN
	}
	if n > intent.LogsMaxN {
		n = intent.LogsMaxN
	}
	var b strings.Builder
	b.WriteString("journalctl")
	for _, u := range ops.NodeUnits() {
		b.WriteString(" -u " + u)
	}
	if cursor != "" {
		if !cursorRE.MatchString(cursor) {
			return "", fmt.Errorf("not a journal cursor: %q", cursor)
		}
		b.WriteString(" --after-cursor '" + cursor + "'")
	}
	b.WriteString(" -n " + strconv.Itoa(n) + " --no-pager -o json")
	return b.String(), nil
}

// journalEntry is one line of journalctl -o json.
type journalEntry struct {
	cursor string
	unit   string
	line   string
	at     time.Time
}

// readJournalJSON reads journalctl -o json output: one object per line.
// MESSAGE is a string, or an array of bytes when it is not valid UTF-8.
// Lines that do not parse are skipped. systemd's own lines about a unit
// ("Started …") carry the unit in UNIT rather than _SYSTEMD_UNIT.
func readJournalJSON(out string) []journalEntry {
	var entries []journalEntry
	for _, raw := range strings.Split(out, "\n") {
		var e struct {
			Cursor  string          `json:"__CURSOR"`
			Micros  string          `json:"__REALTIME_TIMESTAMP"`
			Unit    string          `json:"_SYSTEMD_UNIT"`
			ForUnit string          `json:"UNIT"`
			Message json.RawMessage `json:"MESSAGE"`
		}
		if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &e) != nil {
			continue
		}
		var msg string
		if json.Unmarshal(e.Message, &msg) != nil {
			var bs []byte
			var ints []int
			if json.Unmarshal(e.Message, &ints) == nil {
				for _, i := range ints {
					bs = append(bs, byte(i))
				}
			}
			msg = string(bs)
		}
		// UNIT is not a trusted field: any process logging to the journal
		// may set it. It can only relabel a line between the node's own
		// units, since journalctl -u admitted the line by a trusted field
		// (or by UNIT from pid 1), so it is taken as a label and nothing
		// more. Taking it only when _SYSTEMD_UNIT is init.scope would close
		// even that (review M3, left as is).
		unit := e.Unit
		if e.ForUnit != "" {
			unit = e.ForUnit
		}
		us, _ := strconv.ParseInt(e.Micros, 10, 64)
		entries = append(entries, journalEntry{cursor: e.Cursor, unit: unit, line: msg, at: time.UnixMicro(us)})
	}
	return entries
}

// parseJournalJSON is readJournalJSON as hits, with the last line's cursor.
func parseJournalJSON(out string, classify func(unit, line string, at time.Time) logwatch.Hit) ([]logwatch.Hit, string) {
	var hits []logwatch.Hit
	cursor := ""
	for _, e := range readJournalJSON(out) {
		hits = append(hits, classify(e.unit, e.line, e.at))
		cursor = e.cursor
	}
	return hits, cursor
}

// classifyLine is a log line as a Hit: logwatch's classification when it has
// one, otherwise a plain "info" line.
func classifyLine(unit, line string, at time.Time) logwatch.Hit {
	if h, ok := logwatch.Classify(unit, line, at); ok {
		return h
	}
	return logwatch.Hit{Unit: unit, Line: line, At: at, Severity: "info"}
}

// logsSince runs logs.since. The answer is bounded three ways: journalctl
// reads at most intent.LogsMaxN lines; each line is cut to logsSinceMaxLine
// and the answer stops near logsSinceMaxBytes, its cursor at the last line
// sent so the next call resumes there; and after a cursor, lines older than
// logsSinceMaxAge are passed over, so a cursor cannot page back through the
// box's history (without one the answer is the last n lines, as logs.read).
func logsSince(ctx context.Context, ex executor.Executor, pl intent.LogsSincePayload, now time.Time) (LogsSinceResult, *Reject, error) {
	cmd, err := logsSinceCommand(pl.Cursor, pl.N)
	if err != nil {
		return LogsSinceResult{}, reject(intent.ReasonInvalidPayload, err.Error()), nil
	}
	res, err := ex.Run(ctx, cmd, nil)
	if err != nil {
		return LogsSinceResult{}, nil, err
	}
	out := LogsSinceResult{Hits: []logwatch.Hit{}, Cursor: pl.Cursor}
	oldest := now.Add(-logsSinceMaxAge)
	size := 0
	for _, e := range readJournalJSON(res.Stdout) {
		if pl.Cursor != "" && e.at.Before(oldest) {
			out.Cursor = e.cursor
			continue
		}
		h := classifyLine(e.unit, cutLine(e.line, logsSinceMaxLine), e.at)
		// Measured as encoded: escaping can make a line several times
		// longer on the wire than in the journal.
		b, _ := json.Marshal(h)
		n := len(b) + 1
		if size+n > logsSinceMaxBytes && len(out.Hits) > 0 {
			break // the rest waits for the next call
		}
		size += n
		out.Hits = append(out.Hits, h)
		if e.cursor != "" {
			out.Cursor = e.cursor
		}
	}
	return out, nil, nil
}

// cutLine shortens s to at most max bytes without splitting a character.
func cutLine(s string, max int) string {
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}
