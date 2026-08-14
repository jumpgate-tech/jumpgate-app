// This file implements the update surface. GET /api/update reports whether a
// newer release exists. It has two modes, folded into one endpoint:
//   - Proactive: when notices are on, the server checks in the background and
//     the UI shows a banner. The banner loads this endpoint with no refresh.
//   - On demand: the Settings page loads it with ?refresh=1, which always asks
//     GitHub now — even when notices are off ("don't prompt me"). That is how
//     the operator pulls the latest version without being nagged.
//
// The app never installs anything here — it only reports and links to the
// release page.
package server

import (
	"context"
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

// updateCheckInterval bounds how often the server asks GitHub on the proactive
// path. GitHub's unauthenticated API allows 60 requests an hour per IP, and a
// new release is rare, so one check every few hours is plenty. A manual refresh
// bypasses this window.
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

	// NotifyEnabled is false when the operator chose "don't prompt me". The
	// banner shows only when this is true; the Settings page shows the status
	// either way (it pulls with ?refresh=1).
	NotifyEnabled bool `json:"notifyEnabled"`

	// CheckError carries the reason the last check failed (offline, rate
	// limited), so the UI can say "could not check" rather than "up to date".
	// Empty when the last check succeeded or no check was made.
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

// latestRelease returns the newest release. When force is false it serves the
// cache while it is fresher than updateCheckInterval; when force is true (a
// manual refresh) it always asks GitHub now. The second return value is the
// last check's error text ("" on success). A failed check keeps whatever
// release was last cached, so one offline moment does not erase a known-good
// answer.
func (s *Server) latestRelease(ctx context.Context, force bool) (updatecheck.Release, string) {
	s.updMu.Lock()
	defer s.updMu.Unlock()

	if !force && s.updHasCache && s.clock().Sub(s.updAt) < updateCheckInterval {
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

// PrimeUpdateCheck warms the cache once, in the background, so the banner's
// first load answers instantly. It is the ONLY background call, and it runs
// only when notices are on — "don't prompt me" means the app makes no
// unprompted call to GitHub at all. main calls it after the server starts.
func (s *Server) PrimeUpdateCheck(ctx context.Context) {
	cfg, err := config.Load()
	if err != nil || cfg.UpdateNotifyDisabled {
		return
	}
	_, _ = s.latestRelease(ctx, false)
}

// buildUpdateResponse assembles the response from the current config and the
// (possibly cached) release. NotifyEnabled mirrors the setting; the version
// comparison is the same whether the caller is the banner or the Settings page.
func buildUpdateResponse(cfg config.Config, rel updatecheck.Release, checkErr string) updateResponse {
	st := updatecheck.Evaluate(buildinfo.Version(), rel.Version, rel.URL)
	return updateResponse{
		Current:         st.Current,
		Latest:          st.Latest,
		UpdateAvailable: st.UpdateAvailable,
		ReleaseURL:      st.ReleaseURL,
		NotifyEnabled:   !cfg.UpdateNotifyDisabled,
		CheckError:      checkErr,
	}
}

func (s *Server) handleGetUpdate(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	refresh := isTruthyParam(r.URL.Query().Get("refresh"))

	// "Don't prompt me" and no manual refresh: this is the banner's background
	// load, so make NO call to GitHub. Answer from the config alone.
	if cfg.UpdateNotifyDisabled && !refresh {
		writeJSON(w, http.StatusOK, buildUpdateResponse(cfg, updatecheck.Release{}, ""))
		return
	}

	rel, checkErr := s.latestRelease(r.Context(), refresh)
	writeJSON(w, http.StatusOK, buildUpdateResponse(cfg, rel, checkErr))
}

// isTruthyParam reports whether a query value means "yes" — "1" or "true".
func isTruthyParam(v string) bool {
	return v == "1" || v == "true"
}
