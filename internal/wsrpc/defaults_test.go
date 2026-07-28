package wsrpc

import (
	"strings"
	"testing"
)

// Options is reached through a POINTER that is routinely nil — every call site
// that has nothing to configure passes nil rather than an empty struct — so
// the accessors have to survive that, not merely an empty value.
func TestOptions_NilIsTheSameAsUnset(t *testing.T) {
	var nilOpts *Options

	if got := nilOpts.dialer(); got == nil {
		t.Error("a nil Options produced a nil dialer, which panics at dial time")
	}
	if got := nilOpts.maxMessageBytes(); got != DefaultMaxMessageBytes {
		t.Errorf("maxMessageBytes = %d, want %d", got, DefaultMaxMessageBytes)
	}

	empty := &Options{}
	if got := empty.dialer(); got == nil {
		t.Error("an empty Options produced a nil dialer")
	}
	if got := empty.maxMessageBytes(); got != DefaultMaxMessageBytes {
		t.Errorf("maxMessageBytes = %d, want %d", got, DefaultMaxMessageBytes)
	}

	// A zero cap would mean "no message may be read", so it must never be
	// taken literally.
	if empty.maxMessageBytes() <= 0 {
		t.Fatal("a cap of zero rejects every frame, including the handshake reply")
	}
}

func TestOptions_ExplicitSettingsWin(t *testing.T) {
	o := &Options{MaxMessageBytes: 4096}
	if got := o.maxMessageBytes(); got != 4096 {
		t.Errorf("maxMessageBytes = %d, want 4096", got)
	}

	// A negative cap is nonsense and falls back rather than being honoured.
	neg := &Options{MaxMessageBytes: -1}
	if got := neg.maxMessageBytes(); got != DefaultMaxMessageBytes {
		t.Errorf("maxMessageBytes = %d, want the default for a negative cap", got)
	}
}

// An elided message that looks complete is worse than an obviously clipped
// one: the operator quotes it back and nobody can tell what was cut.
func TestTruncate_MarksThatItClipped(t *testing.T) {
	if got := truncate("short", 32); got != "short" {
		t.Errorf("got %q, want it left alone", got)
	}

	got := truncate(strings.Repeat("x", 100), 10)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("got %q, want a marker that it was clipped", got)
	}
	if len(got) <= 10 {
		t.Errorf("got %q, want 10 bytes plus the marker", got)
	}
	if strings.Count(got, "x") != 10 {
		t.Errorf("got %d bytes of payload, want 10", strings.Count(got, "x"))
	}

	// Exactly at the limit is not clipped — an off-by-one here puts an
	// ellipsis on messages that were complete.
	exact := strings.Repeat("y", 10)
	if got := truncate(exact, 10); got != exact {
		t.Errorf("got %q, want the exact-length string untouched", got)
	}
}
