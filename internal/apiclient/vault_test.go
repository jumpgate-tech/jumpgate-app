package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
)

// Unlock sends the passphrase as UnlockRequest JSON and clears the caller's
// slice; Vault and Lock use their routes.
func TestVaultMethods(t *testing.T) {
	var got []string
	var sent []byte
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path)
		state := "locked"
		if r.URL.Path == "/api/vault/unlock" {
			var req api.UnlockRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode: %v", err)
			}
			sent, state = req.Passphrase, "unlocked"
		}
		fmt.Fprintf(w, `{"state":%q,"slots":["passphrase"]}`, state)
	})
	ctx := context.Background()
	if v, err := c.Vault(ctx); err != nil || v.State != "locked" {
		t.Fatalf("Vault = %+v, %v", v, err)
	}
	pw := []byte("p\"w\\ü")
	v, err := c.Unlock(ctx, pw)
	if err != nil || v.State != "unlocked" {
		t.Fatalf("Unlock = %+v, %v", v, err)
	}
	if string(sent) != "p\"w\\ü" {
		t.Fatalf("server saw %q", sent)
	}
	if !bytes.Equal(pw, make([]byte, len(pw))) {
		t.Fatalf("passphrase not cleared: %q", pw)
	}
	if v, err := c.Lock(ctx); err != nil || v.State != "locked" {
		t.Fatalf("Lock = %+v, %v", v, err)
	}
	want := []string{"GET /api/vault", "POST /api/vault/unlock", "POST /api/vault/lock"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("routes %v", got)
	}
}

func TestUnlockErrorClearsToo(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"wrong passphrase","code":"wrong_passphrase"}`)
	})
	pw := []byte("nope")
	_, err := c.Unlock(context.Background(), pw)
	var e *api.Error
	if !errors.As(err, &e) || e.Code != api.CodeWrongPassphrase {
		t.Fatalf("err = %#v", err)
	}
	if !bytes.Equal(pw, make([]byte, 4)) {
		t.Fatalf("passphrase not cleared")
	}
}
