package intent

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// The schemas are frozen. Changing a field name, type or order changes every
// digest and silently invalidates every signature in flight; this test is
// there so that can only happen on purpose.
func TestSchemasAreFrozen(t *testing.T) {
	cases := map[string]string{
		"Intent":       "Intent(address agent,address controller,uint64 seq,bytes32 nonce,uint64 issuedAt,uint64 expiry,string kind,bytes32 payloadHash)",
		"Receipt":      "Receipt(address agent,bytes32 requestHash,uint64 seq,uint8 status,bytes32 resultHash)",
		"EIP712Domain": "EIP712Domain(string name,string version)",
	}
	for primary, want := range cases {
		got, err := eip712.EncodeType(Types, primary)
		if err != nil || got != want {
			t.Errorf("EncodeType(%s) = %q, %v; want %q", primary, got, err, want)
		}
	}
}

func sample() Intent {
	var i Intent
	i.Agent[19] = 0xA
	i.Controller[19] = 0xC
	i.Seq = 7
	i.Nonce[0] = 0x11
	i.IssuedAt = 1_800_000_000
	i.Expiry = 1_800_000_120
	i.Kind = KindStatusRead
	i.PayloadHash = Hash([]byte("{}"))
	return i
}

// Recompute the intent's struct hash by hand from the ABI rules, independent of
// the eip712 package, so an encoder bug cannot hide behind its own tests.
func TestIntentHashMatchesAHandEncoding(t *testing.T) {
	i := sample()
	word := func(b []byte) []byte { w := make([]byte, 32); copy(w[32-len(b):], b); return w }
	u64 := func(n uint64) []byte { b := make([]byte, 8); binary.BigEndian.PutUint64(b, n); return word(b) }
	typeHash := eip712.Keccak256([]byte("Intent(address agent,address controller,uint64 seq,bytes32 nonce,uint64 issuedAt,uint64 expiry,string kind,bytes32 payloadHash)"))
	kind := eip712.Keccak256([]byte(i.Kind))
	var buf []byte
	buf = append(buf, typeHash[:]...)
	buf = append(buf, word(i.Agent[:])...)
	buf = append(buf, word(i.Controller[:])...)
	buf = append(buf, u64(i.Seq)...)
	buf = append(buf, i.Nonce[:]...)
	buf = append(buf, u64(i.IssuedAt)...)
	buf = append(buf, u64(i.Expiry)...)
	buf = append(buf, kind[:]...)
	buf = append(buf, i.PayloadHash[:]...)
	want := eip712.Keccak256(buf)

	td := i.TypedData()
	got, err := eip712.HashStruct(td.Types, "Intent", td.Message)
	if err != nil || got != want {
		t.Fatalf("hashStruct = %x, %v; want %x", got, err, want)
	}
}

func TestEnvelopeRoundTripsAndStillVerifies(t *testing.T) {
	k, _ := signer.GenerateKey()
	i := sample()
	i.Controller = k.Address()
	sig, err := k.SignTypedData(context.Background(), i.TypedData())
	if err != nil {
		t.Fatal(err)
	}
	env := Envelope{Intent: i.JSON(), Payload: []byte("{}"), Sigs: []string{sig.Hex()}}
	raw, _ := json.Marshal(env)

	var back Envelope
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	parsed, err := back.Intent.Parse()
	if err != nil || parsed != i {
		t.Fatalf("Parse = %+v, %v; want %+v", parsed, err, i)
	}
	s, _ := signer.ParseSignature(back.Sigs[0])
	who, err := signer.Recover(parsed.TypedData(), s)
	if err != nil || who != k.Address() {
		t.Fatalf("recovered %s, %v", who.Hex(), err)
	}
}

func TestParseRejectsMalformedFields(t *testing.T) {
	good := sample().JSON()
	bad := []func(j *IntentJSON){
		func(j *IntentJSON) { j.Agent = "0x12" },
		func(j *IntentJSON) { j.Nonce = "0xzz" },
		func(j *IntentJSON) { j.PayloadHash = "" },
		func(j *IntentJSON) { j.Kind = "" },
	}
	for n, mutate := range bad {
		j := good
		mutate(&j)
		if _, err := j.Parse(); err == nil {
			t.Errorf("case %d parsed", n)
		}
	}
}
