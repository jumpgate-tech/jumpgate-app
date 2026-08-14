// Package updatecheck asks the GitHub Releases API for the latest release of
// one repository and decides whether the running app is out of date. It uses
// only the standard library. The server package calls Latest to read the newest
// release, CompareSemver to order two versions, and Evaluate to build the Status
// that the UI shows.
package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Release is one GitHub release, reduced to what the app needs.
type Release struct {
	Version     string    // the tag as published, e.g. "v0.4.0" (GitHub "tag_name")
	URL         string    // the release page ("html_url")
	PublishedAt time.Time // "published_at"
}

// Status is the update state the API returns to the UI.
type Status struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseURL      string `json:"releaseUrl"`
}

// Client fetches the latest release for one "owner/repo".
type Client struct {
	repo    string
	apiBase string
	http    *http.Client
}

// New builds a Client for repo (form "owner/repo"). It targets the public
// GitHub API and gives the HTTP client a 10-second timeout.
func New(repo string) *Client {
	return &Client{
		repo:    repo,
		apiBase: "https://api.github.com",
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// releaseJSON holds only the three fields the app reads from the API response.
type releaseJSON struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
}

// Latest fetches the latest release and parses it into a Release. It honors ctx
// for deadline and cancel. It returns an error when the request fails, when the
// server answers with a non-2xx status, or when the body is not valid JSON.
func (c *Client) Latest(ctx context.Context) (Release, error) {
	url := c.apiBase + "/repos/" + c.repo + "/releases/latest"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "valve-node-app")

	resp, err := c.http.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Release{}, fmt.Errorf("updatecheck: GitHub returned status %d", resp.StatusCode)
	}

	var raw releaseJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Release{}, fmt.Errorf("updatecheck: cannot parse release JSON: %w", err)
	}

	return Release{
		Version:     raw.TagName,
		URL:         raw.HTMLURL,
		PublishedAt: raw.PublishedAt,
	}, nil
}

// CompareSemver returns -1, 0, or 1 comparing a to b. It strips a single leading
// "v". It compares the dotted numeric components major.minor.patch, and reads a
// missing or non-numeric component as 0. A pre-release (the part after "-")
// sorts below its release. Two pre-releases compare by their text.
func CompareSemver(a, b string) int {
	aCore, aPre := splitVersion(a)
	bCore, bPre := splitVersion(b)

	if n := compareCore(aCore, bCore); n != 0 {
		return n
	}

	// The cores are equal. A version without a pre-release outranks one with a
	// pre-release.
	if aPre == "" && bPre != "" {
		return 1
	}
	if aPre != "" && bPre == "" {
		return -1
	}
	return strings.Compare(aPre, bPre)
}

// splitVersion drops one leading "v" and separates the numeric core from the
// pre-release text after the first "-".
func splitVersion(v string) (core, pre string) {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '-'); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

// compareCore compares two dotted numeric cores. It reads missing or
// non-numeric components as 0.
func compareCore(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	n := len(aParts)
	if len(bParts) > n {
		n = len(bParts)
	}

	for i := 0; i < n; i++ {
		av := numericAt(aParts, i)
		bv := numericAt(bParts, i)
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// numericAt reads the component at index i as an integer. It returns 0 when the
// index is out of range or the text is not a number.
func numericAt(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n, err := strconv.Atoi(parts[i])
	if err != nil {
		return 0
	}
	return n
}

// Evaluate builds a Status. It marks an update available only when the current
// version is a real version, the latest version is newer, and the user has not
// chosen to skip that latest version.
func Evaluate(current, latest, releaseURL, skip string) Status {
	s := Status{
		Current:    current,
		Latest:     latest,
		ReleaseURL: releaseURL,
	}

	if current == "" || current == "dev" {
		return s
	}
	if latest == "" {
		return s
	}
	if CompareSemver(latest, current) <= 0 {
		return s
	}
	if skip != "" && CompareSemver(latest, skip) == 0 {
		return s
	}

	s.UpdateAvailable = true
	return s
}
