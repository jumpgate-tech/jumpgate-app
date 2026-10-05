package apiclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
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

// ConnState is where a stream's connection stands.
type ConnState int

const (
	Connecting ConnState = iota
	Live
	Retrying
	Failed // a client error ended it (a removed target); it will not retry
)

func (s ConnState) String() string {
	switch s {
	case Connecting:
		return "connecting"
	case Live:
		return "live"
	case Retrying:
		return "retrying"
	}
	return "failed"
}

// StreamMsg is one thing that happened on a stream: an event, or a change of
// connection state with the error behind it.
type StreamMsg struct {
	Event   *Event
	State   ConnState
	Err     error
	Attempt int
}

// Reconnect timing (spec D4). Variables so tests can shrink them; a stream
// reads them only from its own goroutine, which tests end before restoring.
var (
	idleTimeout  = 45 * time.Second // the server pings every 15 s
	healthyAfter = 30 * time.Second // a connection this old resets the backoff
	baseDelay    = 250 * time.Millisecond
	maxDelay     = 8 * time.Second
)

var (
	errSilent      = errors.New("the stream went silent")
	errStreamEnded = errors.New("the server ended the stream")
)

// backoff is the wait before reconnect attempt n (0-based): baseDelay
// doubling to maxDelay, scaled by ±20 % from jitter in [0, 1). The jitter
// keeps many clients of a restarted server from reconnecting in lockstep.
func backoff(attempt int, jitter float64) time.Duration {
	d := maxDelay
	if attempt >= 0 && attempt < 16 {
		if b := baseDelay << attempt; b < maxDelay {
			d = b
		}
	}
	return time.Duration(float64(d) * (0.8 + 0.4*jitter))
}

// final reports an error a retry cannot fix: a 4xx other than a timeout or
// a rate limit (the target is gone, the request is wrong).
func final(err error) bool {
	var e *api.Error
	return errors.As(err, &e) && e.Status >= 400 && e.Status < 500 && e.Status != http.StatusRequestTimeout && e.Status != http.StatusTooManyRequests
}

// Stream keeps path open as an SSE stream until ctx ends, reconnecting with
// backoff. Every state change and every event is delivered; the channel
// closes when ctx ends or the stream fails for good. There is no
// Last-Event-ID resume: state streams resend their whole value on connect and
// the logs stream starts with a reset frame (spec A4), so a reconnect needs no
// event ids.
func (c *Client) Stream(ctx context.Context, path string) <-chan StreamMsg {
	out := make(chan StreamMsg, 64)
	go func() {
		defer close(out)
		send := func(m StreamMsg) bool {
			select {
			case out <- m:
				return true
			case <-ctx.Done():
				return false
			}
		}
		for attempt := 0; ; attempt++ {
			if !send(StreamMsg{State: Connecting, Attempt: attempt}) {
				return
			}
			started := time.Now()
			err := c.streamOnce(ctx, path, func() bool { return send(StreamMsg{State: Live}) }, func(ev Event) bool {
				return send(StreamMsg{Event: &ev, State: Live})
			})
			switch {
			case ctx.Err() != nil:
				return
			case final(err):
				send(StreamMsg{State: Failed, Err: err})
				return
			}
			if time.Since(started) >= healthyAfter {
				attempt = 0
			}
			if err == nil {
				err = errStreamEnded
			}
			if !send(StreamMsg{State: Retrying, Err: err, Attempt: attempt}) {
				return
			}
			timer := time.NewTimer(backoff(attempt, rand.Float64()))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	return out
}

// streamOnce runs one connection: it reports live once the server answers,
// emits events, and gives up when the connection ends or stays silent for
// idleTimeout. The silence clock starts before the request, so a server that
// accepts the connection and never answers is noticed too.
func (c *Client) streamOnce(ctx context.Context, path string, live func() bool, emit func(Event) bool) error {
	sctx, cancel := context.WithCancel(ctx)
	defer cancel()
	idle := time.AfterFunc(idleTimeout, cancel)
	defer idle.Stop()
	silent := func(err error) error {
		if sctx.Err() != nil && ctx.Err() == nil {
			return errSilent
		}
		return err
	}
	res, err := c.Open(sctx, http.MethodGet, path, nil, http.Header{"Accept": {"text/event-stream"}})
	if err != nil {
		return silent(err)
	}
	defer res.Body.Close()
	idle.Reset(idleTimeout)
	if !live() {
		return ctx.Err()
	}
	body := &activityReader{r: res.Body, touch: func() { idle.Reset(idleTimeout) }}
	err = parseSSE(body, func(ev Event) {
		if sctx.Err() == nil && !emit(ev) {
			cancel()
		}
	})
	return silent(err)
}

// activityReader calls touch on every read that returns bytes, so a ping
// comment keeps the idle timer from firing.
type activityReader struct {
	r     io.Reader
	touch func()
}

func (a *activityReader) Read(p []byte) (int, error) {
	n, err := a.r.Read(p)
	if n > 0 {
		a.touch()
	}
	return n, err
}

// Update is one typed thing from a watched stream: a value (Has), a change of
// state, a server-sent error event (Err with Has false while State stays
// Live), or a server notice about the stream itself (Note, from an event named
// "note", such as "snapshot mode"). Reset marks a value that replaces what the
// consumer holds.
type Update[T any] struct {
	Value   T
	Has     bool
	Reset   bool
	State   ConnState
	Err     error
	Attempt int
	Note    string
}

// watch decodes a stream's default events with decode. An event named
// "error" carries an api.Error the server could not send as a status (the
// stream was already open); a frame decode cannot read is skipped.
func watch[T any](ctx context.Context, c *Client, path string, decode func(Event) (T, bool, error)) <-chan Update[T] {
	out := make(chan Update[T], 64)
	go func() {
		defer close(out)
		for m := range c.Stream(ctx, path) {
			u := Update[T]{State: m.State, Err: m.Err, Attempt: m.Attempt}
			if ev := m.Event; ev != nil {
				switch ev.Name {
				case "error":
					e := &api.Error{}
					if json.Unmarshal(ev.Data, e) != nil {
						continue
					}
					u.Err = e
				case "note":
					if json.Unmarshal(ev.Data, &u.Note) != nil {
						continue
					}
				default:
					v, reset, err := decode(*ev)
					if err != nil {
						continue
					}
					u.Value, u.Has, u.Reset = v, true, reset
				}
			}
			select {
			case out <- u:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// decodeJSON reads a default event's data as one JSON value.
func decodeJSON[T any](ev Event) (T, bool, error) {
	var v T
	if ev.Name != "" {
		return v, false, errors.New("unexpected event " + ev.Name)
	}
	err := json.Unmarshal(ev.Data, &v)
	return v, false, err
}
