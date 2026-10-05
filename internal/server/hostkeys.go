package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// hostKeyProbeTTL is how long a probed key waits for a person's confirmation.
var hostKeyProbeTTL = 5 * time.Minute

// hostKeyProbes holds keys captured by a probe until a person confirms one.
// Recording the held key, not a fresh capture and never a key a client sends,
// means what the person compared is exactly what is written.
type hostKeyProbes struct {
	mu sync.Mutex
	m  map[string]heldKey

	// record serialises the check-then-append in confirm, so two
	// confirmations for one host cannot both find it unknown and append two
	// different keys.
	record sync.Mutex
}

type heldKey struct {
	hostPort string
	key      ssh.PublicKey
	expires  time.Time
}

func newHostKeyProbes() *hostKeyProbes { return &hostKeyProbes{m: map[string]heldKey{}} }

// put holds key for hostPort and returns its probe id: 128 random bits, so an
// id cannot be guessed, only read from the probe's answer.
func (p *hostKeyProbes) put(hostPort string, key ssh.PublicKey) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // never fails (crypto/rand, Go 1.24+)
	id := hex.EncodeToString(b)
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for k, v := range p.m {
		if now.After(v.expires) {
			delete(p.m, k)
		}
	}
	p.m[id] = heldKey{hostPort: hostPort, key: key, expires: now.Add(hostKeyProbeTTL)}
	return id
}

// take removes and returns a live probe: every probe id is single use, and a
// wrong fingerprint spends it too, so a fingerprint cannot be guessed at.
func (p *hostKeyProbes) take(id string) (heldKey, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	h, ok := p.m[id]
	delete(p.m, id)
	if !ok || time.Now().After(h.expires) {
		return heldKey{}, false
	}
	return h, true
}

// hostPortOf is the exact address DialSSH hands the host-key callback, and so
// the one a confirmation must record.
func hostPortOf(c executor.SSHConfig) string {
	port := c.Port
	if port == 0 {
		port = 22
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(port))
}

// hostKeyPath lists the hops of c's path in the order they are dialled, the
// outermost jump host first and c last. Each keeps its own jump chain, so a
// hop is captured over the same path a real dial takes, through hops that
// carry c's policy.
func hostKeyPath(c *executor.SSHConfig) []executor.SSHConfig {
	if c == nil {
		return nil
	}
	return append(hostKeyPath(c.Jump), *c)
}

// validSSHView reports whether every hop of v names a host and a user.
func validSSHView(v *api.SSHView) bool {
	for h := v; h != nil; h = h.Jump {
		if h.Host == "" || h.User == "" {
			return false
		}
	}
	return true
}

// handleHostKeyProbe captures the key each hop presents, without trusting
// it, and says which are confirmed. It stops at the first hop that is not:
// the next one is reached only through a confirmed host, under Strict (the
// jump is authenticated to, so it must never be one nobody confirmed). An
// unconfirmed key is held under a probe id for handleHostKeyConfirm.
func (s *Server) handleHostKeyProbe(w http.ResponseWriter, r *http.Request) {
	var req api.HostKeyProbeRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil || !validSSHView(&req.SSH) {
		writeError(w, http.StatusBadRequest, `want {"ssh": {"Host", "User", "Port", "KeyPath", "jump"}} with a host and user on every hop`)
		return
	}
	check, algos, err := config.StrictHostKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Every hop carries Strict: CaptureHostKey dials (and authenticates to)
	// the jump hosts with it and refuses a jump without a policy. The last
	// hop's own callback is not consulted by the capture; check below is.
	target := sshConfig(&req.SSH, check)
	for h := target; h != nil; h = h.Jump {
		h.HostKeyAlgorithms = algos
	}
	hops := hostKeyPath(target)

	out := api.HostKeyProbe{Hops: []api.HostKeyHop{}}
	for _, h := range hops {
		hp := hostPortOf(h)
		key, err := executor.CaptureHostKey(r.Context(), h)
		if err != nil {
			// A host that no longer offers any key type on record is a
			// mismatch, not an outage.
			state := api.HostKeyUnreachable
			if errors.Is(err, executor.ErrHostKeyMismatch) {
				state = api.HostKeyMismatch
			}
			out.Hops = append(out.Hops, api.HostKeyHop{HostPort: hp, State: state, Error: err.Error()})
			break
		}
		hop := api.HostKeyHop{HostPort: hp, Fingerprint: executor.Fingerprint(key), KeyType: key.Type()}
		var unknown *executor.UnknownHostError
		switch err := check(hp, nil, key); {
		case err == nil:
			hop.State = api.HostKeyConfirmed
		case errors.As(err, &unknown):
			hop.State, hop.ProbeID = api.HostKeyUnknown, s.probes.put(hp, key)
		case errors.Is(err, executor.ErrHostKeyMismatch):
			// No probe id: a key that contradicts one on record is never
			// offered for confirmation.
			hop.State, hop.Error = api.HostKeyMismatch, err.Error()
		default:
			// The stores could not be read: nothing can be said about the key.
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out.Hops = append(out.Hops, hop)
		if hop.State != api.HostKeyConfirmed {
			break
		}
	}
	out.AllConfirmed = len(out.Hops) == len(hops) && out.Pending() == nil
	writeJSON(w, http.StatusOK, out)
}

// handleHostKeyConfirm records the key a probe captured, once, and only when
// the fingerprint the person compared is exactly that key's. It re-checks the
// stores first: a key on record for the host by now is either this one
// (nothing to write) or a different one, which is a mismatch and never
// overwritten. Replacing a key on record is a separate, deliberate act.
func (s *Server) handleHostKeyConfirm(w http.ResponseWriter, r *http.Request) {
	var req api.HostKeyConfirm
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	held, ok := s.probes.take(req.ProbeID)
	if !ok {
		writeErrorDetail(w, http.StatusGone, "that probe expired or was already used", "", api.CodeProbeExpired)
		return
	}
	if executor.Fingerprint(held.key) != strings.TrimSpace(req.Fingerprint) {
		writeErrorDetail(w, http.StatusConflict, "the fingerprint does not match the key "+held.hostPort+" presented; nothing was recorded", "", api.CodeFingerprintMismatch)
		return
	}

	s.probes.record.Lock()
	defer s.probes.record.Unlock()
	check, _, err := config.StrictHostKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var unknown *executor.UnknownHostError
	switch err := check(held.hostPort, nil, held.key); {
	case err == nil:
		// Already on record with this very key: confirmed, nothing to add.
	case errors.As(err, &unknown):
		confirmed, err := config.ConfirmedHostsFile()
		if err == nil {
			err = executor.RecordHostKey(confirmed, held.hostPort, held.key)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	case errors.Is(err, executor.ErrHostKeyMismatch):
		writeErrorDetail(w, http.StatusConflict, err.Error(), "", api.CodeHostKey)
		return
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
