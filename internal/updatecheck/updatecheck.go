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
	req.Header.Set("User-Agent", "jumpgate")

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
// missing or non-numeric component as 0. Build metadata (the part after "+")
// is ignored, as semver 2.0 requires. A pre-release (the part after "-")
// sorts below its release, and two pre-releases compare by semver 2.0
// precedence (see comparePrerelease).
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
	return comparePrerelease(aPre, bPre)
}

// comparePrerelease orders two non-empty pre-release strings by semver 2.0
// section 11: dot-separated identifiers compared left to right, numeric
// identifiers numerically, alphanumeric ones in ASCII order, a numeric
// identifier below an alphanumeric one, and a longer list above a shorter
// list it extends. Plain text comparison got the common rc.10 vs rc.2 case
// backwards, which would hide a newer release candidate.
func comparePrerelease(a, b string) int {
	aIDs := strings.Split(a, ".")
	bIDs := strings.Split(b, ".")
	for i := 0; i < len(aIDs) && i < len(bIDs); i++ {
		if n := compareIdentifier(aIDs[i], bIDs[i]); n != 0 {
			return n
		}
	}
	switch {
	case len(aIDs) < len(bIDs):
		return -1
	case len(aIDs) > len(bIDs):
		return 1
	}
	return 0
}

// compareIdentifier compares one pre-release identifier pair. Numeric
// identifiers are compared by digit count and then by digits, which is
// numeric order without parsing, so an identifier too long for an int
// cannot overflow into a wrong answer.
func compareIdentifier(a, b string) int {
	aNum, bNum := isNumeric(a), isNumeric(b)
	switch {
	case aNum && bNum:
		a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
		if len(a) != len(b) {
			if len(a) < len(b) {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	case aNum:
		return -1
	case bNum:
		return 1
	}
	return strings.Compare(a, b)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// splitVersion drops one leading "v" and any "+build" metadata, then
// separates the numeric core from the pre-release text after the first "-".
// Metadata has to go first: left in place it made "3+meta" unparseable, so
// v1.2.3+meta read as patch 0.
func splitVersion(v string) (core, pre string) {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
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
// version is a real version and the latest version is newer.
func Evaluate(current, latest, releaseURL string) Status {
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

	s.UpdateAvailable = true
	return s
}
