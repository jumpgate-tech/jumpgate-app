package wsrpc

// A server Conn held open for a long-lived stream needs two bounds that a
// one-shot probe never did: a write must not wait forever on a peer that has
// stopped reading, and a read must not wait forever on a peer that has gone
// away without closing. Both are tested over net.Pipe, which has no buffer at
// all, so "the peer is not reading" is exact rather than a matter of filling a
// kernel socket buffer first.

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

// pipeServer builds a server Conn over one end of a pipe and hands back the
// other end as the "client".
func pipeServer(t *testing.T) (*Conn, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close(); client.Close() })
	return newServerConn(server, bufio.NewReader(server), DefaultMaxMessageBytes), client
}

func isTimeoutErr(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// A peer that stops reading must cost a writer the write timeout, not the rest
// of its life. A relay fans one poll loop out to many sockets, and a single
// write that never returns would hold whatever is waiting on it.
func TestConn_WriteTimeoutBoundsAWriteNobodyReads(t *testing.T) {
	c, _ := pipeServer(t)
	c.SetWriteTimeout(50 * time.Millisecond)

	done := make(chan error, 1)
	go func() { done <- c.WriteText([]byte("nobody is listening")) }()

	select {
	case err := <-done:
		if !isTimeoutErr(err) {
			t.Fatalf("err = %v, want a timeout", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the write never returned, so the write timeout did not apply")
	}
}

// The write timeout is per write, not a deadline set once. A Conn that has
// been open longer than the timeout must still be able to write.
func TestConn_WriteTimeoutIsPerWrite(t *testing.T) {
	c, client := pipeServer(t)
	c.SetWriteTimeout(50 * time.Millisecond)

	go func() {
		br := bufio.NewReader(client)
		for {
			if _, _, _, err := readFrame(br, DefaultMaxMessageBytes, false); err != nil {
				return
			}
		}
	}()

	if err := c.WriteText([]byte("one")); err != nil {
		t.Fatalf("first write: %v", err)
	}
	time.Sleep(80 * time.Millisecond)
	if err := c.WriteText([]byte("two")); err != nil {
		t.Fatalf("a write after the first timeout window failed: %v", err)
	}
}

// A peer that vanishes without a close frame must not hold a read open
// forever. Without this a dead customer connection, and every stream it holds,
// is never reaped.
func TestConn_IdleTimeoutReapsASilentPeer(t *testing.T) {
	c, _ := pipeServer(t)
	c.SetIdleTimeout(50 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		_, err := c.ReadMessage()
		done <- err
	}()

	select {
	case err := <-done:
		if !isTimeoutErr(err) {
			t.Fatalf("err = %v, want a timeout", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the read never returned, so the idle timeout did not apply")
	}
}

// ANY frame proves the peer is alive, a pong included. A subscriber that only
// listens sends no data at all, and answering the server's pings is the only
// sign of life it gives, so a timer that only data reset would reap it.
//
// The timer is the socket's read deadline, so "reset" means: before the read
// that waits for the next frame, a fresh deadline of now+idle is set. The test
// checks exactly that on the server's conn, frame by frame, instead of timing
// pongs with sleeps; a starved writer goroutine on a loaded machine made the
// sleep version fail with nothing wrong in the Conn. That the deadline, once
// set, does reap a silent peer is TestConn_IdleTimeoutReapsASilentPeer.
func TestConn_AnyFrameResetsTheIdleTimer(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close(); client.Close() })
	rec := &deadlineRecorder{Conn: server}
	c := newServerConn(rec, bufio.NewReader(rec), DefaultMaxMessageBytes)
	const idle = time.Hour // never fires: the order of calls is what is checked
	c.SetIdleTimeout(idle)

	const pongs = 10
	mask := []byte{1, 2, 3, 4}
	go func() {
		// net.Pipe has no buffer: each Write returns once the server has read
		// the whole frame, so the server sees one frame per read.
		for i := 0; i < pongs; i++ {
			pong := wsFrame{fin: true, opcode: opcodePong, mask: mask}
			if _, err := client.Write(pong.bytes()); err != nil {
				return
			}
		}
		text := wsFrame{fin: true, opcode: opcodeText, payload: []byte("still here"), mask: mask}
		_, _ = client.Write(text.bytes())
	}()

	got, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(got) != "still here" {
		t.Errorf("got %q, want the message that followed the pongs", got)
	}

	events := rec.snapshot()
	var seq []byte
	for _, e := range events {
		seq = append(seq, e.kind)
	}
	// One deadline before each of the eleven frames' reads: D R D R ... D R.
	want := bytes.Repeat([]byte("DR"), pongs+1)
	if !bytes.Equal(seq, want) {
		t.Fatalf("deadline/read order = %s, want %s: some frame was read without resetting the idle timer", seq, want)
	}
	for i, e := range events {
		if e.kind != 'D' {
			continue
		}
		if left := e.deadline.Sub(e.at); left <= 0 || left > idle {
			t.Errorf("event %d: deadline %v after it was set, want a fresh one within the idle timeout %v", i, left, idle)
		}
	}
}

// deadlineRecorder logs, in order, each read deadline set on a conn ('D', with
// the deadline and when it was set) and each read that returned data ('R').
type deadlineRecorder struct {
	net.Conn
	mu     sync.Mutex
	events []deadlineEvent
}

type deadlineEvent struct {
	kind         byte
	at, deadline time.Time
}

func (r *deadlineRecorder) SetReadDeadline(d time.Time) error {
	r.mu.Lock()
	r.events = append(r.events, deadlineEvent{kind: 'D', at: time.Now(), deadline: d})
	r.mu.Unlock()
	return r.Conn.SetReadDeadline(d)
}

func (r *deadlineRecorder) Read(b []byte) (int, error) {
	n, err := r.Conn.Read(b)
	if n > 0 {
		r.mu.Lock()
		r.events = append(r.events, deadlineEvent{kind: 'R'})
		r.mu.Unlock()
	}
	return n, err
}

func (r *deadlineRecorder) snapshot() []deadlineEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]deadlineEvent(nil), r.events...)
}

// WritePing sends a ping a client is required to answer. That answer is what
// keeps a listen-only subscriber inside the idle timeout.
func TestConn_WritePingSendsAPing(t *testing.T) {
	c, client := pipeServer(t)

	go func() { _ = c.WritePing([]byte("are you there")) }()

	fin, opcode, payload, err := readFrame(bufio.NewReader(client), DefaultMaxMessageBytes, false)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if !fin || opcode != opcodePing {
		t.Fatalf("got fin=%v opcode=%#x, want a ping", fin, opcode)
	}
	if string(payload) != "are you there" {
		t.Errorf("payload = %q, want the ping's payload", payload)
	}
}

// A client answers a ping too, as RFC 6455 §5.5.2 requires of either end. A
// server that reaps idle connections by ping cannot tell a listening client
// from a dead one otherwise.
func TestConn_ClientAnswersAPing(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close(); client.Close() })
	c := &Conn{conn: client, br: bufio.NewReader(client), max: DefaultMaxMessageBytes}

	go func() {
		ping := wsFrame{fin: true, opcode: opcodePing, payload: []byte("hi")}
		_, _ = server.Write(ping.bytes())
	}()
	go func() { _, _ = c.ReadMessage() }()

	_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
	fin, opcode, payload, err := readFrame(bufio.NewReader(server), DefaultMaxMessageBytes, true)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if !fin || opcode != opcodePong || string(payload) != "hi" {
		t.Fatalf("got fin=%v opcode=%#x payload=%q, want a masked pong echoing the ping", fin, opcode, payload)
	}
}

// A client whose pong cannot be written still reads the answer that follows.
// A one-shot probe cares about the answer, and failing it over a courtesy
// reply would turn a working endpoint into a wrong verdict.
func TestConn_ClientIgnoresAFailedPong(t *testing.T) {
	ping := wsFrame{fin: true, opcode: opcodePing, payload: []byte("hi")}
	text := wsFrame{fin: true, opcode: opcodeText, payload: []byte("answer")}
	stream := append(ping.bytes(), text.bytes()...)
	c := &Conn{conn: failingWriter{}, br: bufio.NewReader(bytes.NewReader(stream)), max: DefaultMaxMessageBytes}

	got, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(got) != "answer" {
		t.Errorf("got %q, want the answer after the ping", got)
	}
}

// failingWriter is a net.Conn whose writes always fail.
type failingWriter struct{ net.Conn }

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write side closed") }

// The read loop answers pings while other goroutines write notifications, so
// writes from two goroutines must each land as one whole frame.
func TestConn_ConcurrentWritesDoNotInterleave(t *testing.T) {
	c, client := pipeServer(t)
	c.SetWriteTimeout(time.Second)

	const writers, each = 4, 25
	payload := make([]byte, 300) // past the one-byte length form
	for i := range payload {
		payload[i] = 'x'
	}

	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				_ = c.WriteText(payload)
			}
		}()
	}

	br := bufio.NewReader(client)
	for i := 0; i < writers*each; i++ {
		_, opcode, got, err := readFrame(br, DefaultMaxMessageBytes, false)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if opcode != opcodeText || len(got) != len(payload) {
			t.Fatalf("frame %d: opcode=%#x len=%d, want a whole text frame", i, opcode, len(got))
		}
	}
	wg.Wait()
}
