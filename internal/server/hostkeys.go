package server

import (
	"context"
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

// maxLiveProbes bounds the keys held for confirmation; past it a probe is
// refused (429) rather than letting the store grow. A person confirms one
// key at a time, so 64 is far more than any real use. A var for tests.
var maxLiveProbes = 64

// maxProbeDials bounds the outbound handshakes all probes make at once, so
// a burst of probe requests cannot turn the server into a port scanner.
const maxProbeDials = 8

// hostKeyProbes holds keys captured by a probe until a person confirms one.
// Recording the held key, not a fresh capture and never a key a client sends,
// means what the person compared is exactly what is written.
type hostKeyProbes struct {
	mu sync.Mutex
	m  map[string]heldKey

	// record serialises every write of the confirmed store: confirm's
	// check-then-append, so two confirmations for one host cannot both find
	// it unknown and append two different keys, and forget's rewrite.
	record sync.Mutex

	// dials holds one token per outbound probe handshake in flight.
	dials chan struct{}
}

type heldKey struct {
	hostPort string
	key      ssh.PublicKey
	expires  time.Time
}

func newHostKeyProbes() *hostKeyProbes {
	return &hostKeyProbes{m: map[string]heldKey{}, dials: make(chan struct{}, maxProbeDials)}
}

// put holds key for hostPort and returns its probe id: 128 random bits, so an
// id cannot be guessed, only read from the probe's answer. ok is false when
// maxLiveProbes keys are already waiting.
func (p *hostKeyProbes) put(hostPort string, key ssh.PublicKey) (id string, ok bool) {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // never fails (crypto/rand, Go 1.24+)
	id = hex.EncodeToString(b)
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for k, v := range p.m {
		if now.After(v.expires) {
			delete(p.m, k)
		}
	}
	if len(p.m) >= maxLiveProbes {
		return "", false
	}
	p.m[id] = heldKey{hostPort: hostPort, key: key, expires: now.Add(hostKeyProbeTTL)}
	return id, true
}

// dropHost forgets every held probe for hostPort, so a key probed before a
// forget can never be confirmed after it.
func (p *hostKeyProbes) dropHost(hostPort string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.m {
		if v.hostPort == hostPort {
			delete(p.m, k)
		}
	}
}

// capture is executor.CaptureHostKey within a dial slot. It waits for a free
// slot for as long as the request lasts.
func (p *hostKeyProbes) capture(ctx context.Context, c executor.SSHConfig) (ssh.PublicKey, error) {
	select {
	case p.dials <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-p.dials }()
	return executor.CaptureHostKey(ctx, c)
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
	//
	// KeyPath comes from the client, and the server reads that private key to
	// authenticate to a jump host. That is the token-holder trust model every
	// route shares (add target sends the same field): whoever holds the
	// session token is the operator, running as this same user. The key is
	// only offered to a jump host Strict has confirmed, never returned.
	target := sshConfig(&req.SSH, check)
	for h := target; h != nil; h = h.Jump {
		h.HostKeyAlgorithms = algos
	}
	hops := hostKeyPath(target)

	out := api.HostKeyProbe{Hops: []api.HostKeyHop{}}
	for _, h := range hops {
		hp := hostPortOf(h)
		key, err := s.probes.capture(r.Context(), h)
		if r.Context().Err() != nil {
			return // the client went away; nobody reads an answer
		}
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
			id, ok := s.probes.put(hp, key)
			if !ok {
				writeErrorDetail(w, http.StatusTooManyRequests, "too many host-key probes are waiting for a confirmation", "", api.CodeTooManyProbes)
				return
			}
			hop.State, hop.ProbeID = api.HostKeyUnknown, id
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
	// Take the probe under the record lock: forget drops a host's probes
	// under the same lock, so a probe taken here was not issued before a
	// forget that has already run, and a forget cannot run between this take
	// and the write below.
	s.probes.record.Lock()
	defer s.probes.record.Unlock()
	held, ok := s.probes.take(req.ProbeID)
	if !ok {
		writeErrorDetail(w, http.StatusGone, "that probe expired or was already used", "", api.CodeProbeExpired)
		return
	}
	if executor.Fingerprint(held.key) != strings.TrimSpace(req.Fingerprint) {
		writeErrorDetail(w, http.StatusConflict, "the fingerprint does not match the key "+held.hostPort+" presented; nothing was recorded", "", api.CodeFingerprintMismatch)
		return
	}

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

// canonicalHostPort reports whether hp is a host:port exactly as DialSSH
// writes it (and so as the confirmed store records it).
func canonicalHostPort(hp string) bool {
	host, port, err := net.SplitHostPort(hp)
	if err != nil || host == "" {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n > 0 && n < 65536 && net.JoinHostPort(host, strconv.Itoa(n)) == hp
}

// handleRecordedHostKeys lists every key Strict trusts for ?hostPort=, by
// store, so a person can see (and type) the fingerprint before forgetting it.
func (s *Server) handleRecordedHostKeys(w http.ResponseWriter, r *http.Request) {
	hp := r.URL.Query().Get("hostPort")
	if !canonicalHostPort(hp) {
		writeError(w, http.StatusBadRequest, "want ?hostPort=host:port")
		return
	}
	confirmed, openssh, err := config.RecordedHostKeys(hp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := api.RecordedHostKeys{HostPort: hp, Keys: []api.RecordedHostKey{}}
	for _, k := range confirmed {
		out.Keys = append(out.Keys, api.RecordedHostKey{Fingerprint: executor.Fingerprint(k), KeyType: k.Type(), Store: api.HostKeyStoreJumpgate})
	}
	for _, k := range openssh {
		out.Keys = append(out.Keys, api.RecordedHostKey{Fingerprint: executor.Fingerprint(k), KeyType: k.Type(), Store: api.HostKeyStoreOpenSSH})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleHostKeyForget removes one confirmed key: the line for hostPort whose
// key has exactly the fingerprint the person typed. It is the deliberate act
// that replacing a changed key needs (forget, then probe and confirm); a
// mismatch is never resolved by confirm. Only jumpgate's confirmed store is
// edited: a key in the operator's OpenSSH known_hosts is theirs to remove.
func (s *Server) handleHostKeyForget(w http.ResponseWriter, r *http.Request) {
	var req api.HostKeyForget
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&req); err != nil || !canonicalHostPort(req.HostPort) || strings.TrimSpace(req.Fingerprint) == "" {
		writeError(w, http.StatusBadRequest, `want {"hostPort": "host:port", "fingerprint": "SHA256:…"}`)
		return
	}
	fp := strings.TrimSpace(req.Fingerprint)
	confirmedFile, err := config.ConfirmedHostsFile()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.probes.record.Lock()
	defer s.probes.record.Unlock()
	err = executor.ForgetHostKey(confirmedFile, req.HostPort, fp)
	if err == nil {
		// Under the record lock, so no confirmation of an earlier probe can
		// slip in between the rewrite and the drop.
		s.probes.dropHost(req.HostPort)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !errors.Is(err, executor.ErrHostKeyNotRecorded) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	confirmed, openssh, err := config.RecordedHostKeys(req.HostPort)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	inOpenSSH := false
	for _, k := range openssh {
		inOpenSSH = inOpenSSH || executor.Fingerprint(k) == fp
	}
	switch {
	case inOpenSSH || (len(confirmed) == 0 && len(openssh) > 0):
		writeErrorDetail(w, http.StatusConflict, "the key for "+req.HostPort+" is in your OpenSSH ~/.ssh/known_hosts, not jumpgate's confirmed store; jumpgate never edits known_hosts. Remove it there yourself: ssh-keygen -R "+knownHostsName(req.HostPort), "", api.CodeHostKeyNotOurs)
	case len(confirmed) > 0:
		writeErrorDetail(w, http.StatusConflict, "that fingerprint is not the key jumpgate has on record for "+req.HostPort+"; nothing was removed", "", api.CodeFingerprintMismatch)
	default:
		writeErrorDetail(w, http.StatusNotFound, "no confirmed host key for "+req.HostPort, "", api.CodeNotFound)
	}
}

// knownHostsName is hp as OpenSSH's known_hosts names it: bare for port 22,
// [host]:port otherwise.
func knownHostsName(hp string) string {
	host, port, _ := net.SplitHostPort(hp)
	if port == "22" {
		return host
	}
	return "'[" + host + "]:" + port + "'"
}
