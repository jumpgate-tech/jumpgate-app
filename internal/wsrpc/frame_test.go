package wsrpc

// The RFC 6455 frame reader. It is hand-rolled (no gorilla dependency) and it
// is load-bearing: it is how an endpoint that advertises wss:// but cannot
// actually speak it gets caught, which is true of rpc.pulsechain.com and of
// every published chain-943 endpoint.
//
// This file was the extraction's spec before it was this package's test. Three
// near-identical copies of it lived in internal/chainlist, internal/capabilities
// and internal/setup — written to the same shape on purpose, so that the copies
// could be replaced by one implementation without anyone having to read three
// implementations first. It moved here intact; only the cap constant changed.

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// wsFrame builds one frame the way a server would put it on the wire. len126
// and len127 force the extended length encodings even when the payload is
// short, which is legal and is exactly the sort of thing a real server does
// that a reader written only against the common case gets wrong.
type wsFrame struct {
	fin     bool
	opcode  byte
	payload []byte
	mask    []byte // nil = unmasked, as a server must send
	len126  bool
	len127  bool
	// declaredLen overrides the length field without changing the payload,
	// for the oversize guard.
	declaredLen uint64
}

func (f wsFrame) bytes() []byte {
	var b bytes.Buffer
	h0 := f.opcode
	if f.fin {
		h0 |= 0x80
	}
	b.WriteByte(h0)

	n := uint64(len(f.payload))
	if f.declaredLen != 0 {
		n = f.declaredLen
	}
	var h1 byte
	if f.mask != nil {
		h1 = 0x80
	}
	switch {
	case f.len127 || n > 65535:
		b.WriteByte(h1 | 127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], n)
		b.Write(ext[:])
	case f.len126 || n > 125:
		b.WriteByte(h1 | 126)
		var ext [2]byte
		binary.BigEndian.PutUint16(ext[:], uint16(n))
		b.Write(ext[:])
	default:
		b.WriteByte(h1 | byte(n))
	}
	if f.mask != nil {
		b.Write(f.mask)
		masked := make([]byte, len(f.payload))
		for i, c := range f.payload {
			masked[i] = c ^ f.mask[i%4]
		}
		b.Write(masked)
		return b.Bytes()
	}
	b.Write(f.payload)
	return b.Bytes()
}

func wsStream(frames ...wsFrame) *bufio.Reader {
	var b bytes.Buffer
	for _, f := range frames {
		b.Write(f.bytes())
	}
	return bufio.NewReader(&b)
}

// readAtDefaultCap reads with the package default, which is the cap every
// caller in the tree uses. The limit is a parameter rather than a constant
// precisely because it used to be three constants; the tests exercise the
// default, and TestReadMessage_HonoursACallerSuppliedCap covers the knob.
func readAtDefaultCap(br *bufio.Reader) ([]byte, error) {
	return readMessage(br, DefaultMaxMessageBytes, false, nil)
}

const (
	opText  = 0x1
	opCont  = 0x0
	opClose = 0x8
	opPing  = 0x9
	opPong  = 0xA
)

// ---------------------------------------------------------------------
// reassembly
// ---------------------------------------------------------------------

func TestWSReadMessage_SingleTextFrame(t *testing.T) {
	got, err := readAtDefaultCap(wsStream(wsFrame{fin: true, opcode: opText, payload: []byte(`{"result":"0x171"}`)}))
	if err != nil {
		t.Fatalf("readAtDefaultCap: %v", err)
	}
	if string(got) != `{"result":"0x171"}` {
		t.Errorf("got %q", got)
	}
}

// A server is free to split one JSON-RPC answer across frames, and several do
// under load. A reader that returned the first fragment would hand the caller
// truncated JSON, which parses as a failure and reads as "this endpoint is
// broken" — for an endpoint that is fine.
func TestWSReadMessage_ReassemblesAFragmentedMessage(t *testing.T) {
	got, err := readAtDefaultCap(wsStream(
		wsFrame{fin: false, opcode: opText, payload: []byte(`{"resu`)},
		wsFrame{fin: false, opcode: opCont, payload: []byte(`lt":"0`)},
		wsFrame{fin: true, opcode: opCont, payload: []byte(`x171"}`)},
	))
	if err != nil {
		t.Fatalf("readAtDefaultCap: %v", err)
	}
	if string(got) != `{"result":"0x171"}` {
		t.Errorf("got %q, want the three fragments joined", got)
	}
}

// Control frames arrive whenever the server feels like it — a keepalive ping
// mid-exchange is normal. Treating one as the answer would produce an empty
// body and a bogus "this endpoint answered nothing" verdict.
func TestWSReadMessage_SkipsControlFramesAndKeepsReading(t *testing.T) {
	got, err := readAtDefaultCap(wsStream(
		wsFrame{fin: true, opcode: opPing, payload: []byte("keepalive")},
		wsFrame{fin: true, opcode: opPong, payload: []byte("keepalive")},
		wsFrame{fin: true, opcode: opText, payload: []byte("answer")},
	))
	if err != nil {
		t.Fatalf("readAtDefaultCap: %v", err)
	}
	if string(got) != "answer" {
		t.Errorf("got %q, want the text frame after the control frames", got)
	}
}

// A close before an answer is the measured behaviour of an endpoint that
// advertises wss:// and cannot serve it. It must be an error, not an empty
// success — an empty success is what makes a dead endpoint look live.
func TestWSReadMessage_CloseBeforeAnAnswerIsAnError(t *testing.T) {
	_, err := readAtDefaultCap(wsStream(wsFrame{fin: true, opcode: opClose}))
	if err == nil {
		t.Fatal("a close frame before any answer reported success")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "clos") {
		t.Errorf("error %q does not say the connection closed", err)
	}
}

func TestWSReadMessage_UnknownOpcodeIsAnError(t *testing.T) {
	_, err := readAtDefaultCap(wsStream(wsFrame{fin: true, opcode: 0xB, payload: []byte("x")}))
	if err == nil {
		t.Fatal("an undefined opcode was accepted")
	}
}

// ---------------------------------------------------------------------
// framing details a real server actually produces
// ---------------------------------------------------------------------

// A server MUST NOT mask, but some proxies do it anyway. Reading a masked
// frame without unmasking yields plausible-looking garbage rather than an
// error, which is the worst kind of wrong: it would be reported as an endpoint
// serving a corrupt answer.
func TestWSReadMessage_UnmasksAMaskedFrame(t *testing.T) {
	got, err := readAtDefaultCap(wsStream(wsFrame{
		fin: true, opcode: opText, payload: []byte("masked answer"), mask: []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}))
	if err != nil {
		t.Fatalf("readAtDefaultCap: %v", err)
	}
	if string(got) != "masked answer" {
		t.Errorf("got %q, want the unmasked payload", got)
	}
}

// The three length encodings, including the extended ones used for payloads
// small enough not to need them — legal, and produced in the wild.
func TestWSReadMessage_EveryLengthEncoding(t *testing.T) {
	for name, f := range map[string]wsFrame{
		"7-bit length":           {fin: true, opcode: opText, payload: []byte("short")},
		"16-bit length, forced":  {fin: true, opcode: opText, payload: []byte("short"), len126: true},
		"64-bit length, forced":  {fin: true, opcode: opText, payload: []byte("short"), len127: true},
		"16-bit length, genuine": {fin: true, opcode: opText, payload: bytes.Repeat([]byte("x"), 300)},
		"boundary at 125 bytes":  {fin: true, opcode: opText, payload: bytes.Repeat([]byte("y"), 125)},
		"boundary at 126 bytes":  {fin: true, opcode: opText, payload: bytes.Repeat([]byte("z"), 126)},
		"empty payload":          {fin: true, opcode: opText},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := readAtDefaultCap(wsStream(f))
			if err != nil {
				t.Fatalf("readAtDefaultCap: %v", err)
			}
			if !bytes.Equal(got, f.payload) {
				t.Errorf("got %d bytes, want %d", len(got), len(f.payload))
			}
		})
	}
}

// ---------------------------------------------------------------------
// hostile and broken peers
// ---------------------------------------------------------------------

// A peer that declares an enormous frame must be refused BEFORE the reader
// allocates for it. This probes public endpoints on the open internet, so a
// declared length is attacker-controlled input: without the guard, one hostile
// endpoint in a chainlist feed would take the whole app down with it.
//
// The test would fail by exhausting memory rather than by reporting, which is
// why the declared length is absurd rather than merely over the cap.
func TestWSReadMessage_RefusesAnAbsurdDeclaredLengthWithoutAllocating(t *testing.T) {
	_, err := readAtDefaultCap(wsStream(wsFrame{
		fin: true, opcode: opText, payload: []byte("tiny"), declaredLen: 1 << 40,
	}))
	if err == nil {
		t.Fatal("a 1TiB frame was accepted")
	}
	if !strings.Contains(err.Error(), "oversized") {
		t.Errorf("error %q does not name the reason", err)
	}
}

// The boundary, with the bytes actually present. This is the case that proves
// the CAP is doing the work: the frame is complete and well-formed, so a
// reader with no cap would return it happily. A version of this test that sent
// a short payload would pass either way, because the read would fail on the
// missing bytes rather than on the limit.
func TestWSReadMessage_RefusesACompleteFrameOverTheCap(t *testing.T) {
	over := bytes.Repeat([]byte("x"), DefaultMaxMessageBytes+1)
	if _, err := readAtDefaultCap(wsStream(wsFrame{fin: true, opcode: opText, payload: over})); err == nil {
		t.Fatalf("a complete %d-byte frame was accepted (cap is %d)", len(over), DefaultMaxMessageBytes)
	}

	// And the largest legal frame still gets through, or the cap is a bug of
	// its own: an endpoint whose answer sits just under the limit would be
	// reported dead.
	atCap := bytes.Repeat([]byte("x"), DefaultMaxMessageBytes)
	got, err := readAtDefaultCap(wsStream(wsFrame{fin: true, opcode: opText, payload: atCap}))
	if err != nil {
		t.Fatalf("a frame exactly at the %d-byte cap was refused: %v", DefaultMaxMessageBytes, err)
	}
	if len(got) != len(atCap) {
		t.Errorf("got %d bytes, want %d", len(got), len(atCap))
	}
}

// A truncated stream is what a connection dropping mid-frame looks like. Every
// place the reader can run out of bytes must surface as an error rather than a
// short read that silently becomes a wrong answer.
func TestWSReadMessage_TruncatedStreamsAreErrors(t *testing.T) {
	full := wsFrame{fin: true, opcode: opText, payload: bytes.Repeat([]byte("x"), 300)}.bytes()
	masked := wsFrame{fin: true, opcode: opText, payload: []byte("hello"), mask: []byte{1, 2, 3, 4}}.bytes()

	for name, raw := range map[string][]byte{
		"nothing at all":           {},
		"half a header":            full[:1],
		"header but no ext length": full[:2],
		"partial ext length":       full[:3],
		"header but no payload":    full[:4],
		"payload cut short":        full[:100],
		"mask key cut short":       masked[:4],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readAtDefaultCap(bufio.NewReader(bytes.NewReader(raw))); err == nil {
				t.Fatalf("a truncated stream (%d bytes) was read as a complete message", len(raw))
			}
		})
	}
}

// The cap is a parameter now, where it used to be three different constants in
// three packages. A knob nothing exercises is a knob that does not work, so the
// non-default path is covered even though every caller in the tree currently
// takes the default.
func TestReadMessage_HonoursACallerSuppliedCap(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 2048)
	frame := wsFrame{fin: true, opcode: opText, payload: payload}

	if _, err := readMessage(wsStream(frame), 1024, false, nil); err == nil {
		t.Fatal("a 2048-byte frame was accepted under a 1024-byte cap")
	}
	got, err := readMessage(wsStream(frame), 4096, false, nil)
	if err != nil {
		t.Fatalf("a 2048-byte frame was refused under a 4096-byte cap: %v", err)
	}
	if len(got) != len(payload) {
		t.Errorf("got %d bytes, want %d", len(got), len(payload))
	}
}

// A fragmented message is capped on the TOTAL, not per frame. Otherwise the
// guard is trivially bypassed: a peer that wants to exhaust our memory sends
// the same bytes as a million small continuation frames, every one of them
// comfortably under the limit.
func TestReadMessage_CapsTheAssembledMessageNotJustOneFrame(t *testing.T) {
	chunk := bytes.Repeat([]byte("x"), 400)
	_, err := readMessage(wsStream(
		wsFrame{fin: false, opcode: opText, payload: chunk},
		wsFrame{fin: false, opcode: opCont, payload: chunk},
		wsFrame{fin: true, opcode: opCont, payload: chunk},
	), 1024, false, nil)
	if err == nil {
		t.Fatal("three 400-byte fragments (1200 bytes assembled) were accepted under a 1024-byte cap")
	}
}

// ---------------------------------------------------------------------
// the writing half
// ---------------------------------------------------------------------

// A client frame MUST be masked (RFC 6455 §5.3) and a server is required to
// close the connection on an unmasked one. Nothing in the three copies this
// package replaces ever tested that, so the mask bit is asserted directly —
// reading the frame back is not enough, because the reader unmasks liberally
// and would accept an unmasked frame just as happily.
func TestWriteText_MasksTheFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := writeText(&buf, []byte("hello")); err != nil {
		t.Fatalf("writeText: %v", err)
	}
	raw := buf.Bytes()
	if raw[0] != 0x81 {
		t.Errorf("first header byte is %#x, want 0x81 (FIN set, opcode 1 = text)", raw[0])
	}
	if raw[1]&0x80 == 0 {
		t.Fatal("the mask bit is clear — a server is required to close on an unmasked client frame")
	}
}

// The masking must actually be applied, not merely announced. A writer that
// set the bit and sent the plaintext would produce a frame the server unmasks
// into garbage — and the failure would land on the peer, not here.
func TestWriteText_MaskIsAppliedNotJustAnnounced(t *testing.T) {
	payload := bytes.Repeat([]byte("A"), 64)
	var buf bytes.Buffer
	if err := writeText(&buf, payload); err != nil {
		t.Fatalf("writeText: %v", err)
	}
	// header(2) + mask(4), then the body
	if body := buf.Bytes()[6:]; bytes.Equal(body, payload) {
		t.Fatal("the payload went out in the clear with the mask bit set")
	}
}

// Both extended length encodings, on the boundaries. A writer that got the
// 16-bit case wrong would corrupt every request over 125 bytes — which is
// every batch call this app makes.
func TestWriteText_RoundTripsEveryLengthEncoding(t *testing.T) {
	for name, payload := range map[string][]byte{
		"7-bit length":         []byte("short"),
		"boundary at 125":      bytes.Repeat([]byte("y"), 125),
		"boundary at 126":      bytes.Repeat([]byte("z"), 126),
		"16-bit length":        bytes.Repeat([]byte("m"), 4000),
		"boundary at 65535":    bytes.Repeat([]byte("n"), 65535),
		"64-bit length, 65536": bytes.Repeat([]byte("o"), 65536),
		"empty":                nil,
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writeText(&buf, payload); err != nil {
				t.Fatalf("writeText: %v", err)
			}
			got, err := readAtDefaultCap(bufio.NewReader(&buf))
			if err != nil {
				t.Fatalf("reading back a %d-byte frame we wrote: %v", len(payload), err)
			}
			if !bytes.Equal(got, payload) {
				t.Errorf("round trip returned %d bytes, want %d", len(got), len(payload))
			}
		})
	}
}
