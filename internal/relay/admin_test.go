package relay

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// The control plane manages keys with the ADMIN credential, over the same unix
// socket the relay uses with its least-privilege one. Two credentials, one
// transport — so these tests care most about which token goes out and what
// comes back.

func TestAdminClientCreateReturnsTheRawKeyOnce(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusCreated
	stub.body = `{"id":"k_abc","key":"jg_theonlytimeyouseethis"}`

	c := NewAdminClient(stub.socket, "admin-token")
	id, raw, err := c.CreateKey(context.Background(), "prod")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if id != "k_abc" {
		t.Errorf("id = %q, want k_abc", id)
	}
	if raw != "jg_theonlytimeyouseethis" {
		t.Errorf("raw = %q, want the secret", raw)
	}
	if stub.gotPath != "/admin/keys" {
		t.Errorf("path = %q, want /admin/keys", stub.gotPath)
	}
}

// The control plane must send the ADMIN token here. The relay credential opens
// only /internal/authenticate and would fail, and sending it would also blur the
// separation the two credentials exist to keep.
func TestAdminClientSendsTheAdminToken(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusCreated
	stub.body = `{"id":"k1","key":"jg_x"}`

	c := NewAdminClient(stub.socket, "admin-token")
	if _, _, err := c.CreateKey(context.Background(), "prod"); err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if want := "Bearer admin-token"; stub.gotAuth != want {
		t.Errorf("Authorization = %q, want %q", stub.gotAuth, want)
	}
}

func TestAdminClientListNeverCarriesASecret(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `[{"id":"k1","label":"prod","allow_trace":false,"credit_exempt":false,
	               "created_at":1,"disabled_at":null},
	              {"id":"k2","label":"old","allow_trace":true,"credit_exempt":true,
	               "created_at":2,"disabled_at":99}]`

	c := NewAdminClient(stub.socket, "admin-token")
	keys, err := c.ListKeys(context.Background())
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(keys))
	}
	if keys[0].Disabled {
		t.Error("k1 reads as disabled, want active")
	}
	if !keys[1].Disabled {
		t.Error("k2 reads as active, want disabled — disabled_at is what marks a revoked key")
	}
	// The struct has nowhere to put a secret, which is the point. Prove the
	// serialised form carries none either, so a future field cannot leak one.
	blob, _ := json.Marshal(keys)
	for _, banned := range []string{"key_hash", "\"key\"", "secret", "hash"} {
		if strings.Contains(string(blob), banned) {
			t.Errorf("serialised key list contains %q: %s", banned, blob)
		}
	}
}

func TestAdminClientRevoke(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusNoContent
	stub.body = ``

	c := NewAdminClient(stub.socket, "admin-token")
	if err := c.RevokeKey(context.Background(), "k_abc"); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}
	if stub.gotPath != "/admin/keys/k_abc" {
		t.Errorf("path = %q, want /admin/keys/k_abc", stub.gotPath)
	}
}

// A refusal from the store is an error, not a silent success. An operator who
// clicked revoke must not be told it worked when it did not.
func TestAdminClientSurfacesAFailure(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusNotFound
	stub.body = `{"error":"no such key"}`

	c := NewAdminClient(stub.socket, "admin-token")
	if err := c.RevokeKey(context.Background(), "k_missing"); err == nil {
		t.Fatal("err = nil, want the 404 surfaced")
	}
}
