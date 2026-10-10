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

// logsSinceCommand is one journalctl call over units, interleaved by time, as
// JSON so each line keeps its timestamp and the cursor. The units are the
// agent's own: the node's (ops.NodeUnits, under the box's names, as for
// logs.read) or a job's pattern built from a validated job id. Each is quoted,
// because a job's pattern is a glob the shell must not expand. Nothing in the
// payload reaches the command line except a validated cursor and a number.
// After a cursor, journalctl reads forward, so -n bounds the lines read and a
// caller that falls behind catches up over several calls.
func logsSinceCommand(units []string, cursor string, n int) (string, error) {
	if n <= 0 {
		n = intent.LogsDefaultN
	}
	if n > intent.LogsMaxN {
		n = intent.LogsMaxN
	}
	var b strings.Builder
	b.WriteString("journalctl")
	for _, u := range units {
		b.WriteString(" -u '" + u + "'")
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

// logsSince is the one bounded journal read, behind logs.since and job.output
// (D-J8). The answer is bounded three ways: journalctl reads at most
// intent.LogsMaxN lines; each line is cut to logsSinceMaxLine and the answer
// stops near logsSinceMaxBytes, its cursor at the last line sent so the next
// call resumes there; and after a cursor, lines older than logsSinceMaxAge
// are passed over, so a cursor cannot page back through the box's history
// (without one the answer is the last n lines, as logs.read). item turns one
// journal line, already cut, into what the answer carries; the byte bound is
// measured on item's encoding, since escaping can make a line several times
// longer on the wire than in the journal.
func logsSince[T any](ctx context.Context, ex executor.Executor, units []string, cursor string, n int, now time.Time, item func(journalEntry) T) ([]T, string, *Reject, error) {
	cmd, err := logsSinceCommand(units, cursor, n)
	if err != nil {
		return nil, "", reject(intent.ReasonInvalidPayload, err.Error()), nil
	}
	res, err := ex.Run(ctx, cmd, nil)
	if err != nil {
		return nil, "", nil, err
	}
	out, next := []T{}, cursor
	oldest, size := now.Add(-logsSinceMaxAge), 0
	for _, e := range readJournalJSON(res.Stdout) {
		if cursor != "" && e.at.Before(oldest) {
			next = e.cursor
			continue
		}
		e.line = cutLine(e.line, logsSinceMaxLine)
		it := item(e)
		b, _ := json.Marshal(it)
		sz := len(b) + 1
		if size+sz > logsSinceMaxBytes && len(out) > 0 {
			break // the rest waits for the next call
		}
		size += sz
		out = append(out, it)
		if e.cursor != "" {
			next = e.cursor
		}
	}
	return out, next, nil, nil
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
