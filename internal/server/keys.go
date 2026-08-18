package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/valve-tech/valve-node-app/internal/relay"
)

// maxKeyRequestBytes caps the create-key body. It holds a label and nothing
// else, so anything larger is a mistake or an attempt to make the server hold
// memory on an operator's behalf.
const maxKeyRequestBytes = 64 << 10

// KeyAdmin manages customer API keys. relay.AdminClient implements it against
// the Rust store; a test supplies its own.
//
// It lives behind an interface so the control plane can be tested without a
// unix socket, and so the server package never has to hold the admin
// credential itself.
type KeyAdmin interface {
	CreateKey(ctx context.Context, label string) (id, rawKey string, err error)
	ListKeys(ctx context.Context) ([]relay.KeyView, error)
	RevokeKey(ctx context.Context, id string) error
}

// registerKeyRoutes mounts key management on the CONTROL plane.
//
// These routes mint and revoke customer credentials, so they belong behind the
// operator's session token and nowhere near the data plane. The literal "keys"
// segment beats the {action} wildcard that start, stop and restart ride on,
// exactly as wipe and provision already do.
func (s *Server) registerKeyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/gateways/{gid}/keys", s.handleKeyList)
	mux.HandleFunc("POST /api/gateways/{gid}/keys", s.handleKeyCreate)
	mux.HandleFunc("POST /api/gateways/{gid}/keys/{keyId}/revoke", s.handleKeyRevoke)
}

// keyAdmin returns the configured store, or writes a 501 and reports false. A
// gateway that sells no keys has no store, and that must be a clear answer
// rather than a nil dereference.
func (s *Server) keyAdmin(w http.ResponseWriter) (KeyAdmin, bool) {
	if s.cfg.Keys == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "this gateway has no key store configured",
		})
		return nil, false
	}
	return s.cfg.Keys, true
}

func (s *Server) handleKeyList(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.keyAdmin(w)
	if !ok {
		return
	}
	keys, err := admin.ListKeys(r.Context())
	if err != nil {
		// A broken store is reported. An empty list would read as "this gateway
		// has no keys", which is a different and much more alarming fact.
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "the key store did not answer"})
		return
	}
	if keys == nil {
		keys = []relay.KeyView{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleKeyCreate(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.keyAdmin(w)
	if !ok {
		return
	}

	var req struct {
		Label string `json:"label"`
	}
	// A missing or malformed body is fine: an unlabelled key is still a key,
	// and the label is cosmetic rather than a credential.
	_ = json.NewDecoder(io.LimitReader(r.Body, maxKeyRequestBytes)).Decode(&req)

	id, rawKey, err := admin.CreateKey(r.Context(), req.Label)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "the key store did not answer"})
		return
	}
	// The raw key is returned HERE AND NOWHERE ELSE. The store keeps only a
	// hash, so the operator must copy it now or issue another key.
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "key": rawKey})
}

func (s *Server) handleKeyRevoke(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.keyAdmin(w)
	if !ok {
		return
	}
	if err := admin.RevokeKey(r.Context(), r.PathValue("keyId")); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "the key store did not answer"})
		return
	}
	// Revocation is eventually consistent: the relay caches key records, so a
	// revoked key stops working within that TTL rather than instantly.
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
