package catalog

import (
	"fmt"
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

// SnapshotManifestURL builds the Valve reth snapshot manifest URL for a
// chain, with the operator's free key in the path (per the run-valve-node
// runbook: https://one.valve.city/snapshot/<key>/evm/<chainID>/reth/manifest.json).
// key is the vk_ key obtained free at valve.city.
func SnapshotManifestURL(chainID int, key string) string {
	return fmt.Sprintf("https://one.valve.city/snapshot/%s/evm/%d/reth/manifest.json", key, chainID)
}

// RethDownloadCommand builds the `reth download` snapshot-restore command
// that populates w.DataDir before first start, turning a multi-day
// from-genesis sync into an hours-long download. reth-only (the only
// execution client with SnapshotSupported); the caller gates on
// w.ExecSnapshot and the client's SnapshotSupported flag.
func RethDownloadCommand(w WireConfig) (string, error) {
	chain, ok := rethChainName[w.ChainID]
	if !ok {
		return "", fmt.Errorf("catalog: no reth --chain mapping for chain id %d", w.ChainID)
	}
	return fmt.Sprintf(
		"reth download --chain %s --datadir %s --manifest-url %s",
		shQuote(chain), shQuote(w.DataDir), shQuote(SnapshotManifestURL(w.ChainID, w.SnapshotKey)),
	), nil
}
