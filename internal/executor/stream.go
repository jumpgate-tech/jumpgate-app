package executor

import "bytes"

// maxStreamLine bounds the amount of a single line lineStreamer will buffer
// for RunOpts.Stream delivery. It does NOT bound Result.Stdout capture: raw
// bytes are always written to buf verbatim, regardless of line length. A
// line longer than maxStreamLine is simply dropped from the Stream
// callbacks (never delivered, never an error) rather than growing without
// bound or overflowing a scanner.
const maxStreamLine = 1 << 20 // 1MB

// lineStreamer is an io.Writer that captures every byte written to it
// verbatim into buf — so Result.Stdout ends up byte-exact: no fabricated
// trailing newline, and \r\n is preserved exactly as produced — while also
// splitting the same bytes on '\n' to invoke fn once per complete line, in
// order, as they arrive. This replaces a bufio.Scanner-based capture, whose
// fixed max-token-size makes a single oversized line fail the whole Run
// with bufio.ErrTooLong; lineStreamer can never error.
type lineStreamer struct {
	buf      *bytes.Buffer
	fn       StreamFunc
	line     []byte
	overflow bool

	// onFirst, if set, sees the first complete line (without its line
	// ending) and reports whether to swallow it, in which case that line
	// reaches neither buf nor fn. It is cleared after the first line. Until
	// then the line is held back in first.
	onFirst func(line string) bool
	first   []byte
}

// maxFirstLine bounds how much onFirst interception holds back. A first line
// longer than this cannot be the marker, so it is released as output.
const maxFirstLine = 256

func (w *lineStreamer) Write(p []byte) (int, error) {
	if w.onFirst == nil {
		w.write(p)
		return len(p), nil
	}
	i := bytes.IndexByte(p, '\n')
	if i < 0 {
		w.first = append(w.first, p...)
		if len(w.first) > maxFirstLine {
			w.releaseFirst()
		}
		return len(p), nil
	}
	w.first = append(w.first, p[:i+1]...)
	line := bytes.TrimSuffix(w.first[:len(w.first)-1], []byte{'\r'})
	swallow := w.onFirst(string(line))
	if swallow {
		w.first = nil
		w.onFirst = nil
	} else {
		w.releaseFirst()
	}
	w.write(p[i+1:])
	return len(p), nil
}

// releaseFirst ends interception and passes the held-back bytes on as output.
func (w *lineStreamer) releaseFirst() {
	held := w.first
	w.first = nil
	w.onFirst = nil
	w.write(held)
}

// write captures p into buf and splits it into lines for fn.
func (w *lineStreamer) write(p []byte) {
	w.buf.Write(p)
	if w.fn == nil {
		return
	}
	for _, b := range p {
		if b == '\n' {
			w.emit()
			continue
		}
		if len(w.line) >= maxStreamLine {
			// Drop bytes past the cap for streaming purposes only; buf
			// above already has them verbatim.
			w.overflow = true
			continue
		}
		w.line = append(w.line, b)
	}
}

// emit delivers the currently buffered line to fn (trimming a trailing '\r'
// to match text-mode line splitting), unless it overflowed maxStreamLine, in
// which case it is silently dropped, and resets state for the next line.
func (w *lineStreamer) emit() {
	if !w.overflow {
		line := w.line
		if n := len(line); n > 0 && line[n-1] == '\r' {
			line = line[:n-1]
		}
		w.fn(string(line))
	}
	w.line = w.line[:0]
	w.overflow = false
}

// Flush delivers a final line that was never terminated by '\n' (i.e. the
// command's output didn't end in a newline). Call once after the writer has
// seen all input.
func (w *lineStreamer) Flush() {
	if w.onFirst != nil {
		// An unterminated first line is output, never a marker.
		w.releaseFirst()
	}
	if w.fn != nil && !w.overflow && len(w.line) > 0 {
		w.fn(string(w.line))
	}
	w.line = w.line[:0]
	w.overflow = false
}
