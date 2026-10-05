package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
)

// D/F: a key bound to an account the ledger has never seen is the operator's
// to fix (provision the account), not an outage and not a top-up. It gets its
// own 403, distinct from 402 (pay) and 503 (the ledger did not answer).
func TestUnprovisionedAccountIs403NotAnOutage(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	var got capturedRequest
	credits := &countingCredits{failure: ErrNoAccount}
	h := creditedHandler(t, fundedKey(), credits, &got)

	res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	if body["error"] != "account not provisioned" {
		t.Fatalf("error = %q, want %q", body["error"], "account not provisioned")
	}
	if got.hits != 0 {
		t.Error("a call for an unprovisioned account reached the upstream")
	}
	// The operator is told, with the account to provision and never the key.
	if !strings.Contains(buf.String(), "not provisioned") || !strings.Contains(buf.String(), "0xcustomer") {
		t.Fatalf("log = %q, want a line naming the unprovisioned account", buf.String())
	}
	if strings.Contains(buf.String(), "jg_k") {
		t.Fatal("the log line carries the customer's key")
	}
}

func TestWSUnprovisionedAccountHasItsOwnCode(t *testing.T) {
	h := newWSHarnessWith(t, meteredKey(), func(c *WSConfig) {
		c.Charge = func(context.Context, string) error { return ErrNoAccount }
	})
	h.send(t, blockNumber)
	got := h.read(t)
	if errorCode(got) != codeAccountNotProvisioned {
		t.Fatalf("got %v, want error %d", got, codeAccountNotProvisioned)
	}
	if e, _ := got["error"].(map[string]any); e["message"] != "account not provisioned" {
		t.Fatalf("message = %v", e["message"])
	}
}
