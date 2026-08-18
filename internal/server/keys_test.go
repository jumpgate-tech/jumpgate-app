package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/relay"
)

// Key management rides the control plane, behind the session token. It is the
// operator's surface, not the customer's: a customer holds a key and can do
// nothing with these routes.

type fakeKeyAdmin struct {
	created   []string
	revoked   []string
	keys      []relay.KeyView
	createID  string
	createRaw string
	err       error
}

func (f *fakeKeyAdmin) CreateKey(_ context.Context, label string) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	f.created = append(f.created, label)
	return f.createID, f.createRaw, nil
}

func (f *fakeKeyAdmin) ListKeys(context.Context) ([]relay.KeyView, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.keys, nil
}

func (f *fakeKeyAdmin) RevokeKey(_ context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	f.revoked = append(f.revoked, id)
	return nil
}

func keyServer(t *testing.T, admin KeyAdmin) *Server {
	t.Helper()
	return New(Config{Bind: "127.0.0.1:0", Token: "session-token", Keys: admin})
}

func authed(method, path, body string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	r.Header.Set("Authorization", "Bearer session-token")
	return r
}

// The raw key is returned exactly once, at creation. The store keeps only a
// hash, so an operator who loses this value must issue a new key.
func TestCreateKeyReturnsTheRawKeyOnce(t *testing.T) {
	admin := &fakeKeyAdmin{createID: "k_abc", createRaw: "jg_seenonce"}
	s := keyServer(t, admin)

	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, authed(http.MethodPost, "/api/gateways/g1/keys", `{"label":"prod"}`))

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %q)", res.Code, res.Body.String())
	}
	var out map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", res.Body.String(), err)
	}
	if out["key"] != "jg_seenonce" {
		t.Errorf("key = %q, want the raw secret", out["key"])
	}
	if out["id"] != "k_abc" {
		t.Errorf("id = %q, want k_abc", out["id"])
	}
	if len(admin.created) != 1 || admin.created[0] != "prod" {
		t.Errorf("created = %v, want [prod]", admin.created)
	}
}

// Listing must never carry a secret. This is the route an operator refreshes,
// so a leak here would be repeated into every browser cache and log.
func TestListKeysCarriesNoSecret(t *testing.T) {
	admin := &fakeKeyAdmin{keys: []relay.KeyView{
		{ID: "k1", Label: "prod", Disabled: false},
		{ID: "k2", Label: "old", Disabled: true},
	}}
	s := keyServer(t, admin)

	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, authed(http.MethodGet, "/api/gateways/g1/keys", ""))

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	for _, banned := range []string{"jg_", "secret", "hash"} {
		if strings.Contains(body, banned) {
			t.Errorf("list response contains %q: %s", banned, body)
		}
	}
	if !strings.Contains(body, "k1") || !strings.Contains(body, "k2") {
		t.Errorf("list response is missing keys: %s", body)
	}
}

func TestRevokeKey(t *testing.T) {
	admin := &fakeKeyAdmin{}
	s := keyServer(t, admin)

	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, authed(http.MethodPost, "/api/gateways/g1/keys/k_abc/revoke", ""))

	if res.Code != http.StatusOK && res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want a success (body %q)", res.Code, res.Body.String())
	}
	if len(admin.revoked) != 1 || admin.revoked[0] != "k_abc" {
		t.Errorf("revoked = %v, want [k_abc]", admin.revoked)
	}
}

// A customer key must not reach key management. These routes mint credentials,
// so the session token is the only thing that may open them.
func TestKeyRoutesNeedTheSessionToken(t *testing.T) {
	admin := &fakeKeyAdmin{createID: "k", createRaw: "jg_x"}
	s := keyServer(t, admin)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/gateways/g1/keys"},
		{http.MethodPost, "/api/gateways/g1/keys"},
		{http.MethodPost, "/api/gateways/g1/keys/k_abc/revoke"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer jg_customerkey")
		res := httptest.NewRecorder()
		s.Handler().ServeHTTP(res, req)

		if res.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want 401", tc.method, tc.path, res.Code)
		}
	}
	if len(admin.created) != 0 || len(admin.revoked) != 0 {
		t.Error("an unauthenticated caller reached the key store")
	}
}

// A gateway with no key store configured says so, rather than panicking on a
// nil client.
func TestKeyRoutesWithoutAStoreAre501(t *testing.T) {
	s := New(Config{Bind: "127.0.0.1:0", Token: "session-token"})

	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, authed(http.MethodGet, "/api/gateways/g1/keys", ""))

	if res.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", res.Code)
	}
}

// A store failure is reported, not swallowed into an empty list that would read
// as "this gateway has no keys".
func TestKeyListSurfacesAStoreFailure(t *testing.T) {
	admin := &fakeKeyAdmin{err: errors.New("socket is gone")}
	s := keyServer(t, admin)

	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, authed(http.MethodGet, "/api/gateways/g1/keys", ""))

	if res.Code == http.StatusOK {
		t.Fatalf("status = 200 with a broken store; body %q", res.Body.String())
	}
}

// The literal "keys" segment must beat the {action} wildcard that start, stop
// and restart ride on, exactly as wipe and provision already do.
func TestKeysSegmentBeatsTheActionWildcard(t *testing.T) {
	admin := &fakeKeyAdmin{createID: "k", createRaw: "jg_x"}
	s := keyServer(t, admin)

	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, authed(http.MethodPost, "/api/gateways/g1/keys", `{"label":"x"}`))

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 — the action dispatcher swallowed /keys", res.Code)
	}
}
