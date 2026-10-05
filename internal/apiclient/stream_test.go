package apiclient

import (
	"strings"
	"testing"
)

func TestParseSSE(t *testing.T) {
	in := ": ping\n\n" +
		"data: {\"a\":1}\n\n" +
		"event: reset\ndata: [1,\ndata: 2]\n\n" +
		"id: 7\ndata: x\r\n\r\n" +
		"data: unterminated"
	var got []Event
	if err := parseSSE(strings.NewReader(in), func(e Event) { got = append(got, e) }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d events: %+v", len(got), got)
	}
	if got[0].Name != "" || string(got[0].Data) != `{"a":1}` {
		t.Errorf("event 0 = %+v", got[0])
	}
	if got[1].Name != "reset" || string(got[1].Data) != "[1,\n2]" {
		t.Errorf("event 1 = %+v", got[1])
	}
	if string(got[2].Data) != "x" {
		t.Errorf("event 2 = %+v (CRLF must be accepted)", got[2])
	}
}
