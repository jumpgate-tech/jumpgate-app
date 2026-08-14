// This file implements the update-notification surface: GET /api/update
// reports whether a newer release exists, and POST /api/update/skip records a
// version the operator does not want to be nagged about. The app never
// installs anything itself here — it only tells the operator and points at the
// release page.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/valve-tech/valve-node-app/internal/buildinfo"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/updatecheck"
)

// updateSource fetches the latest published release. The real one talks to
// GitHub (updatecheck.Client); tests inject a fake so they never touch the
// network. It is the same injection pattern as newExecutor / newChainlist.
type updateSource interface {
	Latest(ctx context.Context) (updatecheck.Release, error)
}

// updateCheckInterval bounds how often the server asks GitHub. GitHub's
// unauthenticated API allows 60 requests an hour per IP, and a new release is
// rare, so one check every few hours is plenty and leaves the budget alone.
const updateCheckInterval = 6 * time.Hour

// updateFetchTimeout caps one live check so a slow or unreachable GitHub can
// never hang the request that triggered it.
const updateFetchTimeout = 5 * time.Second

// updateResponse is the JSON GET /api/update returns.
type updateResponse struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseURL      string `json:"releaseUrl"`

	// CheckEnabled is false when the operator turned the check off. The UI
	// still shows the running version, but never a newer one.
	CheckEnabled bool `json:"checkEnabled"`

	// CheckError carries the reason the last check failed (offline, rate
	// limited), so the UI can say "could not check" rather than "up to date".
	// Empty when the last check succeeded or the check is disabled.
	CheckError string `json:"checkError,omitempty"`
}

// clock returns the server's time source, defaulting to time.Now. Injectable
// so a test can drive the cache window without waiting real hours.
func (s *Server) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// latestRelease returns the newest release, from the cache when it is fresher
// than updateCheckInterval and by asking GitHub otherwise. The second return
// value is the last check's error text ("" on success). A failed check keeps
// whatever release was last cached, so one offline moment does not erase a
// known-good answer.
func (s *Server) latestRelease(ctx context.Context) (updatecheck.Release, string) {
	s.updMu.Lock()
	defer s.updMu.Unlock()

	if s.updHasCache && s.clock().Sub(s.updAt) < updateCheckInterval {
		return s.updCache, s.updErr
	}

	cctx, cancel := context.WithTimeout(ctx, updateFetchTimeout)
	defer cancel()

	rel, err := s.updater.Latest(cctx)
	s.updAt = s.clock()
	s.updHasCache = true
	if err != nil {
		s.updErr = err.Error()
		return s.updCache, s.updErr
	}
	s.updCache = rel
	s.updErr = ""
	return rel, ""
}

// PrimeUpdateCheck warms the cache once, in the background, so the first UI
// poll answers instantly instead of blocking on a GitHub round-trip. It
// respects the disabled setting and swallows the result — a failed check is
// reported through GET /api/update, never here. main calls it after the
// server starts.
func (s *Server) PrimeUpdateCheck(ctx context.Context) {
	cfg, err := config.Load()
	if err != nil || cfg.UpdateCheckDisabled {
		return
	}
	_, _ = s.latestRelease(ctx)
}

// buildUpdateResponse assembles the response for the current config and the
// (possibly cached) release. It is the one place that folds the disabled
// setting, the skip version, and the running version together, so GET and the
// skip POST cannot answer differently.
func buildUpdateResponse(cfg config.Config, rel updatecheck.Release, checkErr string) updateResponse {
	current := buildinfo.Version()
	if cfg.UpdateCheckDisabled {
		return updateResponse{Current: current, CheckEnabled: false}
	}
	st := updatecheck.Evaluate(current, rel.Version, rel.URL, cfg.UpdateSkipVersion)
	return updateResponse{
		Current:         st.Current,
		Latest:          st.Latest,
		UpdateAvailable: st.UpdateAvailable,
		ReleaseURL:      st.ReleaseURL,
		CheckEnabled:    true,
		CheckError:      checkErr,
	}
}

func (s *Server) handleGetUpdate(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg.UpdateCheckDisabled {
		// Disabled means "do not reach GitHub", so answer from the config
		// alone — no network, no cache read.
		writeJSON(w, http.StatusOK, buildUpdateResponse(cfg, updatecheck.Release{}, ""))
		return
	}
	rel, checkErr := s.latestRelease(r.Context())
	writeJSON(w, http.StatusOK, buildUpdateResponse(cfg, rel, checkErr))
}

type skipUpdateRequest struct {
	Version string `json:"version"`
}

// handleSkipUpdate records the version the operator wants to skip and answers
// with the recomputed status. It does NOT hit GitHub — it reuses the cached
// release, so skipping is instant and offline-safe.
func (s *Server) handleSkipUpdate(w http.ResponseWriter, r *http.Request) {
	var req skipUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Version == "" {
		writeError(w, http.StatusBadRequest, "version is required")
		return
	}

	cfg, err := s.updateConfig(func(c *config.Config) error {
		c.UpdateSkipVersion = req.Version
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Recompute from the cache, without a fresh fetch.
	s.updMu.Lock()
	rel, checkErr := s.updCache, s.updErr
	s.updMu.Unlock()

	writeJSON(w, http.StatusOK, buildUpdateResponse(cfg, rel, checkErr))
}
