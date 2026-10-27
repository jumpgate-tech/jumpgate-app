package agentclient

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

type upload struct {
	env  intent.Envelope
	in   intent.Intent
	body []byte
}

// uploadAgent is a fake agent's /v1/upgrade: it records each request and
// answers with answer(seq), signed by k over the request it got.
func uploadAgent(t *testing.T, k *signer.Key, answer func(n int, in intent.Intent) (uint8, any)) (string, *[]upload) {
	t.Helper()
	var mu sync.Mutex
	var got []upload
	sock := fakeAgent(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/upgrade" {
			http.Error(w, "wrong route "+r.URL.Path, http.StatusNotFound)
			return
		}
		raw, err := base64.RawURLEncoding.DecodeString(r.Header.Get(agent.UpgradeEnvelopeHeader))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var env intent.Envelope
		_ = json.Unmarshal(raw, &env)
		in, err := env.Intent.Parse()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, upload{env, in, body})
		n := len(got)
		mu.Unlock()
		status, v := answer(n, in)
		result, _ := json.Marshal(v)
		reqHash, _ := in.Digest()
		rc := intent.Receipt{Agent: k.Address(), RequestHash: reqHash, Seq: in.Seq, Status: status, ResultHash: intent.Hash(result)}
		sig, _ := k.SignTypedData(context.Background(), rc.TypedData())
		_ = json.NewEncoder(w).Encode(intent.ReceiptEnvelope{Receipt: rc.JSON(), Result: result, Sig: sig.Hex()})
	})
	return sock, &got
}

func TestUploadSignsLikeDoAndSendsTheBlob(t *testing.T) {
	k, controller := mustKey(t), mustKey(t)
	sock, got := uploadAgent(t, k, func(int, intent.Intent) (uint8, any) {
		return intent.StatusOK, intent.AgentUpgradeResult{Upgraded: true, From: "v0.6.0", To: "v0.6.1"}
	})
	c := mustDial(t, Target{Local: true, Socket: sock, Agent: k.Address()}, controller, NewMemorySeqStore())
	blob := []byte("the new binary")
	sum := sha256.Sum256(blob)
	pl := intent.AgentUpgradePayload{Arch: "amd64", Version: "v0.6.1", SHA256: hex.EncodeToString(sum[:]), Size: int64(len(blob))}
	res, err := c.Upload(context.Background(), intent.KindAgentUpgrade, pl, blob)
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("Upload = %+v, %v", res, err)
	}
	if len(*got) != 1 {
		t.Fatalf("%d requests", len(*got))
	}
	u := (*got)[0]
	if u.in.Kind != intent.KindAgentUpgrade || u.in.Controller != controller.Address() {
		t.Fatalf("intent %+v", u.in)
	}
	s, _ := signer.ParseSignature(u.env.Sigs[0])
	if who, err := signer.Recover(u.in.TypedData(), s); err != nil || who != controller.Address() {
		t.Fatalf("not signed by the controller: %v", err)
	}
	var sent intent.AgentUpgradePayload
	_ = json.Unmarshal(u.env.Payload, &sent)
	bsum := sha256.Sum256(u.body)
	if sent.SHA256 != hex.EncodeToString(bsum[:]) {
		t.Fatal("body does not match the payload's sha256")
	}
}

func TestUploadRetriesStaleSeqWithTheSameBlob(t *testing.T) {
	k, controller := mustKey(t), mustKey(t)
	sock, got := uploadAgent(t, k, func(n int, in intent.Intent) (uint8, any) {
		if n == 1 {
			return intent.StatusRejected, intent.Rejection{Code: intent.ReasonStaleSeq, LastSeq: 9}
		}
		return intent.StatusOK, intent.AgentUpgradeResult{}
	})
	c := mustDial(t, Target{Local: true, Socket: sock, Agent: k.Address()}, controller, NewMemorySeqStore())
	blob := []byte("bin")
	res, err := c.Upload(context.Background(), intent.KindAgentUpgrade, intent.AgentUpgradePayload{Size: 3}, blob)
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("Upload = %+v, %v", res, err)
	}
	if len(*got) != 2 || (*got)[1].in.Seq != 10 || string((*got)[1].body) != "bin" || string((*got)[0].body) != "bin" {
		t.Fatalf("requests %+v", *got)
	}
}
