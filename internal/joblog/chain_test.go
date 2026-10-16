package joblog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// Chain is Verify less the signatures: the same head for a good log, the
// same refusals for a broken order or link, and a forged signature passes.
func TestChainChecksLinksAndNotSignatures(t *testing.T) {
	k, other := mustKey(t), mustKey(t)
	es := chain(t, k, 3)
	want, err := Verify(k.Address(), job, es, 0, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := Chain(k.Address(), job, es, 0, [32]byte{}); err != nil || got != want {
		t.Fatalf("Chain = %x, %v; Verify %x", got, err, want)
	}
	if _, err := Chain(k.Address(), job, []Entry{es[0], es[2]}, 0, [32]byte{}); !errors.Is(err, ErrBroken) {
		t.Fatalf("a skipped entry: %v", err)
	}
	relinked := es[1]
	relinked.Index = 0
	if _, err := Chain(k.Address(), job, []Entry{relinked}, 0, [32]byte{}); !errors.Is(err, ErrBroken) {
		t.Fatalf("an entry that does not follow: %v", err)
	}
	forged := append([]Entry(nil), es...)
	forged[1].Sig = es[0].Sig
	if _, err := Chain(k.Address(), job, forged, 0, [32]byte{}); err != nil {
		t.Fatalf("Chain checked a signature: %v", err)
	}
	if _, err := Verify(k.Address(), job, forged, 0, [32]byte{}); !errors.Is(err, ErrBroken) {
		t.Fatalf("Verify passed a forged signature: %v", err)
	}
	if _, err := Verify(other.Address(), job, es, 0, [32]byte{}); !errors.Is(err, ErrBroken) {
		t.Fatalf("another agent's log: %v", err)
	}
}

// VerifyIntents reads only request entries, and refuses one it cannot read.
func TestVerifyIntentsRefusesUnreadableRequests(t *testing.T) {
	k, ctl := mustKey(t), mustKey(t)
	sign := func(kind string, body []byte) Entry {
		e, _, err := Sign(context.Background(), k, job, 0, [32]byte{}, kind, body, time.Unix(1, 0))
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	if err := VerifyIntents(k.Address(), job, []Entry{sign(KindStepStarted, []byte(`not json`))}); err != nil {
		t.Fatalf("a step entry is not a request: %v", err)
	}
	if err := VerifyIntents(k.Address(), "", nil); !errors.Is(err, ErrBroken) {
		t.Fatalf("an empty job id: %v", err)
	}
	env := signedEnvelope(t, k.Address(), job, ctl, ctl)
	badIntent := env
	badIntent.Intent.Agent = "nope"
	badSig := env
	badSig.Sigs = []string{"0xzz"}
	for name, body := range map[string][]byte{
		"not json":            []byte(`{`),
		"an unparsed intent":  mustJSON(t, IntentBody{Envelope: badIntent}),
		"no usable signature": mustJSON(t, IntentBody{Envelope: badSig}),
	} {
		if err := VerifyIntents(k.Address(), job, []Entry{sign(KindIntent, body)}); !errors.Is(err, ErrBroken) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFromJSONRefusesABadPrevOrSignature(t *testing.T) {
	k := mustKey(t)
	j := ToJSON(chain(t, k, 1)[0])
	badPrev, badSig := j, j
	badPrev.Prev = "0x1234"
	badSig.Sig = "0xzz"
	for name, in := range map[string]intent.JobLogEntryJSON{"prev": badPrev, "sig": badSig} {
		if _, err := FromJSON(in); !errors.Is(err, ErrBroken) {
			t.Errorf("a bad %s: %v", name, err)
		}
	}
}
