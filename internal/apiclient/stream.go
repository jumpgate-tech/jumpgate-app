package apiclient

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// Event is one SSE event: its name ("" for the default) and its data lines
// joined with "\n".
type Event struct {
	Name string
	Data []byte
}

// maxEventLine bounds one SSE line (a fleet snapshot of many boxes fits).
const maxEventLine = 8 << 20

// parseSSE reads events until r ends. Comments (": ping") and unknown fields
// are skipped; an event without a terminating blank line is dropped, since a
// stream cut mid-event never finished it.
func parseSSE(r io.Reader, emit func(Event)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), maxEventLine)
	var name string
	var data bytes.Buffer
	has := false
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		switch {
		case line == "":
			if has {
				emit(Event{Name: name, Data: bytes.Clone(data.Bytes())})
			}
			name, has = "", false
			data.Reset()
		case strings.HasPrefix(line, ":"):
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			if has {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			has = true
		}
	}
	return sc.Err()
}
