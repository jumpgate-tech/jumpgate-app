package server

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/vault"
)

// The passphrase routes (POST /api/vault/unlock and POST /api/system/seal)
// back off per peer uid after wrong passphrases: the first few are free (a
// typo), then each further one doubles the wait, up to passBackoffMax. A
// right passphrase clears the uid's count. Failures are logged with the uid
// only; the passphrase is never logged.
var (
	passBackoffFree = 3
	passBackoffBase = time.Second
	passBackoffMax  = 5 * time.Minute
)

// unknownUID is the bucket for a request whose peer uid cannot be read.
const unknownUID = -1

type passFailure struct {
	n     int       // wrong passphrases in a row
	until time.Time // no attempt before this
}

type passBackoff struct {
	mu   sync.Mutex
	now  func() time.Time
	fail map[int]passFailure
}

func (b *passBackoff) clock() time.Time {
	if b.now != nil {
		return b.now()
	}
	return time.Now()
}

// wait is how long uid must still wait before its next attempt (0: go).
func (b *passBackoff) wait(uid int) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	f, ok := b.fail[uid]
	if !ok {
		return 0
	}
	return max(f.until.Sub(b.clock()), 0)
}

// failed records a wrong passphrase from uid and returns the count in a
// row and the wait it now owes.
func (b *passBackoff) failed(uid int) (int, time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fail == nil {
		b.fail = map[int]passFailure{}
	}
	f := b.fail[uid]
	f.n++
	var d time.Duration
	if over := f.n - passBackoffFree; over > 0 {
		d = passBackoffMax
		if over <= 30 {
			if x := float64(passBackoffBase) * math.Pow(2, float64(over-1)); x < float64(passBackoffMax) {
				d = time.Duration(x)
			}
		}
	}
	f.until = b.clock().Add(d)
	b.fail[uid] = f
	return f.n, d
}

func (b *passBackoff) succeeded(uid int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.fail, uid)
}

// requestUID is the uid of the peer behind r: the admin socket's peer
// credentials, else the loopback socket's owner where it can be read, else
// unknownUID.
func (s *Server) requestUID(r *http.Request) int {
	if c, ok := peerFrom(r.Context()); ok {
		return c.UID
	}
	if s.peerUID != nil {
		if uid, v := s.peerUID(r); v == peerFound {
			return uid
		}
	}
	return unknownUID
}

// passBackedOff answers 429 and returns true when uid must still wait.
func (s *Server) passBackedOff(w http.ResponseWriter, route string, uid int) bool {
	d := s.passBackoff.wait(uid)
	if d <= 0 {
		return false
	}
	secs := int(math.Ceil(d.Seconds()))
	log.Printf("jumpgate: %s refused for uid %d: too many wrong passphrases, %ds left to wait", route, uid, secs)
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	writeError(w, http.StatusTooManyRequests, "too many wrong passphrases; try again in "+strconv.Itoa(secs)+"s")
	return true
}

// passResult records how a passphrase attempt from uid went.
func (s *Server) passResult(route string, uid int, err error) {
	switch {
	case err == nil:
		s.passBackoff.succeeded(uid)
	case errors.Is(err, vault.ErrWrongPassphrase), errors.Is(err, vault.ErrSlotRefused):
		n, d := s.passBackoff.failed(uid)
		log.Printf("jumpgate: %s: wrong passphrase from uid %d (%d in a row; next attempt in %s)", route, uid, n, d)
	}
}
