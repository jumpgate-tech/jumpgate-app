package wsrpc

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// DefaultMaxMessageBytes caps an assembled message when a caller names no
// limit of its own.
//
// One megabyte is a deliberate middle. The three implementations this package
// replaced had drifted to 64KB, 1MB and 8MB: 64KB was sized for an
// eth_chainId answer, 8MB for a newHeads header, and a real newHeads is a few
// kilobytes — so both ends were wrong by an order of magnitude, in opposite
// directions. Nothing in this tree passes anything else.
const DefaultMaxMessageBytes = 1 << 20

// writeText writes payload as one masked text frame — the client side of
// this package's own traffic. Server-side writes go through writeFrame with
// masked set to false; see Conn.WriteText for the direction switch.
//
// The mask is not optional and not decoration: RFC 6455 §5.3 requires every
// client frame to be masked, and a server is required to fail the connection
// on one that is not.
func writeText(w io.Writer, payload []byte) error {
	return writeFrame(w, opcodeText, payload, true)
}

// writeFrame writes one complete, unfragmented frame.
//
// FIN is always set: this package never sends a fragmented frame from
// either side, so there is never a reason to clear it. masked chooses the
// direction — true for a frame this package's client sends, false for one
// its server (Accept) sends — because RFC 6455 §5.1 requires exactly one of
// those and forbids the other.
func writeFrame(w io.Writer, opcode byte, payload []byte, masked bool) error {
	var buf bytes.Buffer
	buf.WriteByte(0x80 | opcode) // FIN | opcode

	var lenBit byte
	if masked {
		lenBit = 0x80
	}
	switch n := len(payload); {
	case n < 126:
		buf.WriteByte(lenBit | byte(n))
	case n <= 0xFFFF:
		buf.WriteByte(lenBit | 126)
		_ = binary.Write(&buf, binary.BigEndian, uint16(n))
	default:
		buf.WriteByte(lenBit | 127)
		_ = binary.Write(&buf, binary.BigEndian, uint64(n))
	}

	if !masked {
		buf.Write(payload)
		_, err := w.Write(buf.Bytes())
		return err
	}
	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return fmt.Errorf("websocket: mask: %w", err)
	}
	buf.Write(mask[:])
	for i, b := range payload {
		buf.WriteByte(b ^ mask[i%4])
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// readMessage reads frames until a complete data message arrives, reassembling
// continuation frames and skipping the control frames a server may interleave
// (a keepalive ping before the answer is normal on an endpoint with an idle
// timer).
//
// max bounds the ASSEMBLED message, not one frame. Capping per frame would be
// no cap at all: the same bytes arriving as many small continuation frames
// would each pass, and the guard exists because a declared length is
// attacker-controlled input from a public endpoint.
//
// requireMasked is passed straight to readFrame: true on the server side,
// where RFC 6455 requires every incoming frame masked, false on the client
// side, where the peer must not mask but some proxies do anyway and get
// tolerated.
//
// onPing, if not nil, is called with a ping frame's payload before the loop
// keeps reading. The server side uses it to answer with a pong, as RFC 6455
// §5.5.2 requires; the client side of a one-shot request/response exchange
// passes nil and just skips the ping.
func readMessage(br *bufio.Reader, max int64, requireMasked bool, onPing func([]byte) error) ([]byte, error) {
	var msg []byte
	for {
		fin, opcode, payload, err := readFrame(br, max-int64(len(msg)), requireMasked)
		if err != nil {
			return nil, err
		}
		switch opcode {
		case opcodeContinuation, opcodeText, opcodeBinary:
			msg = append(msg, payload...)
			if fin {
				return msg, nil
			}
		case opcodeClose:
			return nil, errors.New("websocket closed before answering")
		case opcodePing:
			if onPing != nil {
				if err := onPing(payload); err != nil {
					return nil, err
				}
			}
		case opcodePong: // nothing to do for a request/response exchange
		default:
			return nil, fmt.Errorf("websocket: unexpected opcode %#x", opcode)
		}
	}
}

const (
	opcodeContinuation = 0x0
	opcodeText         = 0x1
	opcodeBinary       = 0x2
	opcodeClose        = 0x8
	opcodePing         = 0x9
	opcodePong         = 0xA
)

// readFrame reads one frame header and its payload, unmasking if the peer
// masked it. A server must not mask, but some proxies do, and reading a masked
// frame without unmasking yields plausible garbage rather than an error —
// which would be reported as an endpoint serving a corrupt answer.
//
// requireMasked enforces the other direction: RFC 6455 §5.1 requires every
// CLIENT frame masked, and a server MUST fail the connection on one that is
// not. The server side (Accept) passes true; the client side passes false,
// since a server frame arriving unmasked is the required case, not a fault.
//
// budget is what remains of the caller's message cap. It is checked against
// the DECLARED length, before the allocation, so a peer claiming a terabyte
// costs us an error rather than the process.
func readFrame(br *bufio.Reader, budget int64, requireMasked bool) (fin bool, opcode byte, payload []byte, err error) {
	var hdr [2]byte
	if _, err = io.ReadFull(br, hdr[:]); err != nil {
		return false, 0, nil, err
	}
	fin = hdr[0]&0x80 != 0
	opcode = hdr[0] & 0x0F
	masked := hdr[1]&0x80 != 0
	if requireMasked && !masked {
		return false, 0, nil, errors.New("websocket: unmasked client frame — RFC 6455 §5.1 requires masking")
	}

	length := uint64(hdr[1] & 0x7F)
	switch length {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	// budget can go negative only if a previous fragment already exhausted the
	// message cap, which the comparison below catches as an oversize.
	if budget < 0 || length > uint64(budget) {
		return false, 0, nil, fmt.Errorf("websocket: oversized frame (%d bytes)", length)
	}

	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(br, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(br, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return fin, opcode, payload, nil
}
