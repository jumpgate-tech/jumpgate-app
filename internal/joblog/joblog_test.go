// internal/joblog/joblog_test.go
package joblog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

const job = "0123456789abcdef"

func TestSchemaIsFrozen(t *testing.T) {
	got, err := eip712.EncodeType(Types, "JobLogEntry")
	want := "JobLogEntry(address agent,string job,uint64 index,bytes32 prev,string kind,bytes32 bodyHash,uint64 at)"
	if err != nil || got != want {
		t.Fatalf("EncodeType = %q, %v; want %q", got, err, want)
	}
}

// chain signs n entries the way the agent appends them.
func chain(t *testing.T, k *signer.Key, n int) []Entry {
	t.Helper()
	var out []Entry
	var prev [32]byte
	for i := 0; i < n; i++ {
		e, d, err := Sign(context.Background(), k, job, uint64(i), prev, KindStepFinished,
			[]byte(`{"step":"s`+string(rune('a'+i))+`","attempt":0,"ok":true}`), time.Unix(1_800_000_000+int64(i), 0))
		if err != nil {
			t.Fatal(err)
		}
		out, prev = append(out, e), d
	}
	return out
}

func TestVerifyAcceptsAnUnbrokenChainAndResumes(t *testing.T) {
	k, _ := signer.GenerateKey()
	es := chain(t, k, 4)
	head, err := Verify(k.Address(), job, es, 0, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	// A controller that already verified the first two verifies only the rest.
	mid, _ := Verify(k.Address(), job, es[:2], 0, [32]byte{})
	again, err := Verify(k.Address(), job, es[2:], 2, mid)
	if err != nil || again != head {
		t.Fatalf("resumed verify: %v, head %x vs %x", err, again, head)
	}
}

// Review Focus 5.
func TestVerifyRejectsEditsReordersAndForeignSigners(t *testing.T) {
	k, _ := signer.GenerateKey()
	other, _ := signer.GenerateKey()
	for name, mutate := range map[string]func([]Entry) []Entry{
		"edited body":    func(es []Entry) []Entry { es[1].Body = []byte(`{"ok":false}`); return es },
		"edited kind":    func(es []Entry) []Entry { es[2].Kind = KindStepLost; return es },
		"dropped middle": func(es []Entry) []Entry { return append(es[:1], es[2:]...) },
		"reordered":      func(es []Entry) []Entry { es[1], es[2] = es[2], es[1]; return es },
		"foreign signer": func(es []Entry) []Entry {
			e, _, err := Sign(context.Background(), other, job, 3, es[3].Prev, es[3].Kind, es[3].Body, time.Unix(int64(es[3].At), 0))
			if err != nil {
				t.Fatal(err)
			}
			es[3] = e
			return es
		},
		"other job": func(es []Entry) []Entry {
			e, _, err := Sign(context.Background(), k, "fedcba9876543210", 3, es[3].Prev, es[3].Kind, es[3].Body, time.Unix(int64(es[3].At), 0))
			if err != nil {
				t.Fatal(err)
			}
			es[3] = e
			return es
		},
	} {
		es := mutate(chain(t, k, 4))
		if _, err := Verify(k.Address(), job, es, 0, [32]byte{}); !errors.Is(err, ErrBroken) {
			t.Errorf("%s: Verify = %v, want ErrBroken", name, err)
		}
	}
}

func TestJSONRoundTripKeepsTheSignature(t *testing.T) {
	k, _ := signer.GenerateKey()
	es := chain(t, k, 2)
	b, _ := json.Marshal(ToJSON(es[1]))
	var j intent.JobLogEntryJSON
	if err := json.Unmarshal(b, &j); err != nil {
		t.Fatal(err)
	}
	back, err := FromJSON(j)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(k.Address(), job, []Entry{es[0], back}, 0, [32]byte{}); err != nil {
		t.Fatalf("a round-tripped entry no longer verifies: %v", err)
	}
}

// signedEnvelope is an advance request addressed to agent, for job id, that
// controller's intent is signed by signs.
func signedEnvelope(t *testing.T, agent eip712.Address, id string, controller, signs *signer.Key) intent.Envelope {
	t.Helper()
	body := []byte(`{"id":"` + id + `"}`)
	n, err := intent.NewNonce()
	if err != nil {
		t.Fatal(err)
	}
	in := intent.Intent{Agent: agent, Controller: controller.Address(), Seq: 1, Nonce: n, IssuedAt: 1, Expiry: 2, Kind: intent.KindJobAdvance, PayloadHash: intent.Hash(body)}
	sig, err := signs.SignTypedData(context.Background(), in.TypedData())
	if err != nil {
		t.Fatal(err)
	}
	return intent.Envelope{Intent: in.JSON(), Payload: body, Sigs: []string{sig.Hex()}}
}

// Review Focus 5: an intent entry proves who asked only if the controller it
// names signed it, over exactly the payload it carries, for this agent and job.
func TestVerifyIntentsNeedsTheNamedController(t *testing.T) {
	agentKey, _ := signer.GenerateKey()
	otherAgent, _ := signer.GenerateKey()
	ctl, _ := signer.GenerateKey()
	mallory, _ := signer.GenerateKey()
	a := agentKey.Address()
	good := signedEnvelope(t, a, job, ctl, ctl)
	swapped := good
	swapped.Payload = []byte(`{"id":"ffffffffffffffff"}`)
	for name, c := range map[string]struct {
		env intent.Envelope
		ok  bool
	}{
		"good":                              {good, true},
		"signed by someone else":            {signedEnvelope(t, a, job, ctl, mallory), false},
		"payload swapped":                   {swapped, false},
		"intent addressed to another agent": {signedEnvelope(t, otherAgent.Address(), job, ctl, ctl), false},
		"intent for another job":            {signedEnvelope(t, a, "fedcba9876543210", ctl, ctl), false},
	} {
		body, err := json.Marshal(IntentBody{Envelope: c.env, Outcome: "advance"})
		if err != nil {
			t.Fatal(err)
		}
		e, _, err := Sign(context.Background(), agentKey, job, 0, [32]byte{}, KindIntent, body, time.Unix(1, 0))
		if err != nil {
			t.Fatal(err)
		}
		if err := VerifyIntents(a, job, []Entry{e}); (err == nil) != c.ok {
			t.Errorf("%s: VerifyIntents = %v", name, err)
		}
	}
}

// A log cut short still verifies; only its head gives the cut away.
func TestVerifyOfATruncatedLogReturnsADifferentHead(t *testing.T) {
	k, _ := signer.GenerateKey()
	es := chain(t, k, 4)
	full, err := Verify(k.Address(), job, es, 0, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	short, err := Verify(k.Address(), job, es[:3], 0, [32]byte{})
	if err != nil {
		t.Fatalf("a valid prefix must verify: %v", err)
	}
	if short == full {
		t.Fatal("a truncated log returned the full chain's head")
	}
}

func TestDescribeStripsControlCharacters(t *testing.T) {
	got := Describe(Entry{Kind: KindStepFinished, Body: []byte(`{"step":"s","attempt":1,"error":"boom\u001b[2J\u001b]0;x\u0007\r\nbad"}`)})
	for _, r := range got {
		if r < 0x20 || r == 0x7f {
			t.Fatalf("Describe kept a control character: %q", got)
		}
	}
	if !strings.Contains(got, "boom") {
		t.Fatalf("Describe lost the text: %q", got)
	}
}

func TestDescribeIsOneReadableLine(t *testing.T) {
	q, _ := json.Marshal(intent.JobQuestion{ID: "stop-units-0-busy-user", Prompt: "2 processes still run"})
	for kind, body := range map[string][]byte{
		KindStepStarted:   []byte(`{"step":"stop-units","attempt":0,"unit":"jumpgate-job-x-stop-units-0"}`),
		KindDecisionAsked: q,
		KindFinished:      []byte(`{"state":"succeeded"}`),
	} {
		got := Describe(Entry{Kind: kind, Body: body})
		if got == "" || len(got) > 200 {
			t.Errorf("Describe(%s) = %q", kind, got)
		}
	}
}
