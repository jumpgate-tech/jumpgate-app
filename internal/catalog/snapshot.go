package catalog

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// shQuote single-quotes s for safe interpolation into a `sh -c` command
// string (the executor runs commands via sh -c), escaping any embedded
// single quotes. RethDownloadCommand's output is executed through the
// shell, so every interpolated value — including the operator-supplied
// snapshot key inside the manifest URL — must pass through this.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// SnapshotVersionsURL is the KEYLESS discovery endpoint that lists the reth
// execution snapshots published for a chain. It is the only listing the
// gateway exposes — there is no `latest` resolver — so restore always starts
// here: fetch this, match the target's actual reth version to one of the
// advertised version ranges, and only then is a concrete manifest URL known.
//
// Mainnet (evm/1) 404s here because no snapshot is published for it; the
// caller must treat a non-200 as "no snapshot for this chain", not a bug.
func SnapshotVersionsURL(chainID int) string {
	return fmt.Sprintf("https://one.valve.city/snapshot/evm/%d/reth/versions.json", chainID)
}

// snapshotVersions mirrors the versions.json document. The published data is
// inconsistent about the range key — the first evm/369 entry carries
// `version_range`, every other entry carries `reth_version_range` — so both
// are decoded and versionRange() below prefers whichever is populated.
type snapshotVersions struct {
	ChainID           int               `json:"chain_id"`
	AvailableVersions []snapshotVersion `json:"available_versions"`
}

type snapshotVersion struct {
	VersionRange     string `json:"version_range"`
	RethVersionRange string `json:"reth_version_range"`
	ManifestURL      string `json:"manifest_url"`
	// generated_at is the primary recency key; timestamp is the fallback for
	// the older entries that predate generated_at (both are unix seconds).
	GeneratedAt int64 `json:"generated_at"`
	Timestamp   int64 `json:"timestamp"`
}

// versionRange returns the entry's advertised reth version range, tolerating
// the two spellings the gateway emits.
func (v snapshotVersion) versionRange() string {
	if v.RethVersionRange != "" {
		return v.RethVersionRange
	}
	return v.VersionRange
}

// recency is the sort key for "newest snapshot": generated_at when present,
// else timestamp. An entry with neither sorts oldest, so a well-formed entry
// always beats a malformed one.
func (v snapshotVersion) recency() int64 {
	if v.GeneratedAt != 0 {
		return v.GeneratedAt
	}
	return v.Timestamp
}

// majorMinor pattern captures the "2.3" out of a version-ish token like
// "2.3.x", "2.3.0-pulse", or "Reth Version: 2.2.0-pulse". Snapshots are
// storage-format-specific and are cut per reth minor line, so the minor is
// exactly the granularity a snapshot must match the running binary at.
var majorMinorRe = regexp.MustCompile(`(\d+)\.(\d+)`)

// majorMinor extracts the leading "<major>.<minor>" from any version-ish
// string, or "" if none is present. Used for both the reth --version output
// and the advertised range so the two are compared on the same footing.
func majorMinor(s string) string {
	m := majorMinorRe.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1] + "." + m[2]
}

// ParseRethMajorMinor pulls the "<major>.<minor>" line out of `reth --version`
// output. reth prints its version as e.g. "reth Version: 2.2.0-pulse" (the
// pulse fork) or "reth 1.1.0" (upstream); either way the first version-shaped
// token is the one we want. The app builds reth from `main`, so this is read
// at runtime rather than assumed — a hardcoded version is exactly how the old
// URL went stale.
func ParseRethMajorMinor(versionOutput string) (string, error) {
	mm := majorMinor(versionOutput)
	if mm == "" {
		return "", fmt.Errorf("catalog: could not find a version in reth --version output %q", strings.TrimSpace(versionOutput))
	}
	return mm, nil
}

// SelectSnapshotManifest is the pure core of snapshot discovery: given the
// bytes of a versions.json document and the target's reth major.minor (e.g.
// "2.2"), it returns the newest advertised manifest URL whose version range
// matches. It does no I/O so it is fully unit-testable; the caller fetches the
// bytes on the target and injects the key into the result.
//
// chainID is threaded through only to name the chain in the not-found error —
// this repo's bugs report success while broken, so "no snapshot" has to say
// which chain and which reth version it could not satisfy.
func SelectSnapshotManifest(versionsJSON []byte, rethVersion string, chainID int) (string, error) {
	want := majorMinor(rethVersion)
	if want == "" {
		return "", fmt.Errorf("catalog: %q is not a reth major.minor version", rethVersion)
	}

	var doc snapshotVersions
	if err := json.Unmarshal(versionsJSON, &doc); err != nil {
		return "", fmt.Errorf("catalog: versions.json for chain %d is not valid JSON: %w", chainID, err)
	}
	if len(doc.AvailableVersions) == 0 {
		return "", fmt.Errorf("catalog: chain %d publishes no reth snapshots", chainID)
	}

	best := snapshotVersion{}
	found := false
	for _, v := range doc.AvailableVersions {
		if majorMinor(v.versionRange()) != want || v.ManifestURL == "" {
			continue
		}
		if !found || v.recency() > best.recency() {
			best, found = v, true
		}
	}
	if !found {
		// Name every range on offer: the operator's reth is on a line Valve
		// has not cut a snapshot for, and the fix is to see which lines exist.
		var offered []string
		for _, v := range doc.AvailableVersions {
			if r := v.versionRange(); r != "" {
				offered = append(offered, r)
			}
		}
		return "", fmt.Errorf(
			"catalog: chain %d has no reth snapshot for version %s (published ranges: %s)",
			chainID, want, strings.Join(offered, ", "),
		)
	}
	return best.ManifestURL, nil
}

// snapshotKeylessMarker is the path segment versions.json manifest URLs carry
// (they are published keyless). InjectSnapshotKey rewrites it to carry the
// operator's key.
const snapshotKeylessMarker = "/snapshot/evm/"

// InjectSnapshotKey rewrites a keyless manifest URL from versions.json
// (…/snapshot/evm/…) to carry the operator's key (…/snapshot/<key>/evm/…).
// This is not cosmetic: the gateway echoes whatever key was in the request
// path into the manifest's base_url, and `reth download` pulls every chunk
// from that base_url — so a keyless manifest yields a keyless, unauthorized
// chunk fetch. The key must be present before the URL reaches reth.
//
// The input shape is validated so a manifest URL that does not look like the
// gateway's is a loud error here rather than a silent 404 hours into a
// download.
func InjectSnapshotKey(manifestURL, key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("catalog: cannot inject an empty snapshot key into %q", manifestURL)
	}
	if !strings.Contains(manifestURL, snapshotKeylessMarker) {
		return "", fmt.Errorf("catalog: manifest URL %q does not have the expected %q shape", manifestURL, snapshotKeylessMarker)
	}
	// Rewrite only the first occurrence: the marker is the path prefix, and a
	// second literal "/snapshot/evm/" later in the URL would be part of a
	// filename, not the segment to key.
	return strings.Replace(manifestURL, snapshotKeylessMarker, "/snapshot/"+key+"/evm/", 1), nil
}

// RethDownloadCommand builds the `reth download` snapshot-restore command that
// populates w.DataDir before first start, turning a multi-day from-genesis
// sync into an hours-long download. manifestURL is the ALREADY-RESOLVED,
// key-injected URL (SelectSnapshotManifest → InjectSnapshotKey) — this
// function no longer guesses the URL, which is what used to 404. reth-only
// (the only execution client with SnapshotSupported); the caller gates on
// w.ExecSnapshot and the client's SnapshotSupported flag.
func RethDownloadCommand(w WireConfig, manifestURL string) (string, error) {
	chain, ok := rethChainName[w.ChainID]
	if !ok {
		return "", fmt.Errorf("catalog: no reth --chain mapping for chain id %d", w.ChainID)
	}
	if strings.TrimSpace(manifestURL) == "" {
		return "", fmt.Errorf("catalog: reth download needs a resolved manifest URL, got empty")
	}
	return fmt.Sprintf(
		"reth download --chain %s --datadir %s --manifest-url %s",
		shQuote(chain), shQuote(w.DataDir), shQuote(manifestURL),
	), nil
}
