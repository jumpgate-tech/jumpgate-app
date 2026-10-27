package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
)

// vaultIdleTick is how often the idle lock looks (a variable so tests
// shorten it; New reads it once).
var vaultIdleTick = 30 * time.Second

// maxIdleLockMinutes bounds the vaultIdleLockMinutes setting (one day).
const maxIdleLockMinutes = 1440

// markActivity restarts the idle clock.
func (s *Server) markActivity() { s.lastWrite.Store(s.clock().UnixNano()) }

// noteRequest restarts the idle clock for an authenticated write. Reads and
// streams are not activity: an open TUI or browser tab streams forever (S9).
func (s *Server) noteRequest(r *http.Request) {
	if !isSafeMethod(r.Method) {
		s.markActivity()
	}
}

// idleLoop locks the vault after the idle-lock setting's minutes with no
// write and no job driver running, until ctx ends.
func (s *Server) idleLoop(ctx context.Context, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.idleCheck()
		}
	}
}

// idleCheck locks the vault when it has been idle long enough, and reports
// whether it did. A running plan keeps the vault unlocked, however long it
// runs: its driver needs the controller key.
func (s *Server) idleCheck() bool {
	c, err := config.Load()
	if err != nil || c.VaultIdleLockMinutes <= 0 {
		return false
	}
	n := c.VaultIdleLockMinutes
	// Under vctl.mu, so an unlock (which restarts the clock under the same
	// lock) is never undone by a decision made before it.
	s.vctl.mu.Lock()
	defer s.vctl.mu.Unlock()
	v := s.vault()
	if v == nil || v.Locked() {
		return false
	}
	if s.jobs.driverCount() > 0 {
		return false
	}
	idle := s.clock().Sub(time.Unix(0, s.lastWrite.Load()))
	if idle < time.Duration(n)*time.Minute {
		return false
	}
	s.lockLocked()
	log.Printf("vault locked after %d minutes idle", n)
	return true
}

// checkIdleLockMinutes validates the setting.
func checkIdleLockMinutes(n int) bool { return n >= 0 && n <= maxIdleLockMinutes }
