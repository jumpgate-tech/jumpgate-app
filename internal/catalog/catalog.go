// Package catalog is the pure-data knowledge base for valve-node-app: the
// networks it knows how to configure (Ethereum, PulseChain, PulseChain
// Testnet v4), the execution/beacon clients it knows how to run on each,
// and the systemd unit templates that pair an execution client with a
// beacon client for a given network. It performs no I/O and never touches
// an executor.Executor — writing the rendered units to disk is Task 4's
// job.
//
// Network and client data (chain ids, checkpoint URLs, client repos and
// build commands) are ported verbatim from the learn.valve.city runbook
// data in the monorepo (packages/web/src/learn/data/{networks,clients}.ts).
package catalog

import "fmt"

// Network describes one of the chains valve-node-app can configure an
// execution+beacon client pair for.
type Network struct {
	ChainID       int    // 1 | 369 | 943
	Name          string // "Ethereum" | "PulseChain" | "PulseChain Testnet v4"
	CheckpointURL string
	ExecClients   []string // client ids valid as the execution client on this chain
	BeaconClients []string // client ids valid as the beacon client on this chain
	LearnURL      string

	// SnapshotSizeTB is the size, in decimal terabytes, of *Valve's reth
	// snapshot artifact* for this chain. Ported verbatim from
	// packages/web/src/learn/data/networks.ts, where it is stored as
	// `snapshot.sizeTB` under a comment that reads:
	//
	//	`snapshot` present ⇒ a Valve free snapshot exists (fast sync)
	//
	// Read that literally, because the field was previously misnamed
	// ArchiveSizeTB and the misnomer propagated: this number is the size of
	// ONE artifact — a reth database, cut at one particular block height,
	// as published for `reth download`. It is not a survey of what a node
	// costs on disk. Specifically it is NOT:
	//
	//   - a per-client figure. go-pulse, erigon-pulse and geth lay their
	//     databases out completely differently (erigon in particular is
	//     advertised by its own maintainers as far more compact for
	//     archive), and the learn data publishes no size for any of them.
	//   - a full-vs-archive figure. The learn data says nothing about which
	//     pruning tier the snapshot corresponds to, and publishes no second
	//     number to contrast it with.
	//   - stable over time. A chain grows; the snapshot is re-cut.
	//
	// Everything the app derives from it for another client or another tier
	// is an estimate, not a measurement — see fullTierFraction and
	// ExpectedBytes, and keep the UI's wording honest about that.
	SnapshotSizeTB float64
	// SyncLabel and GenesisSyncLabel are the human sync-time estimates
	// shown on learn.valve.city — SyncLabel for a snapshot-assisted sync,
	// GenesisSyncLabel for a from-genesis sync.
	SyncLabel        string
	GenesisSyncLabel string
}

// fullTierFraction is the fraction of Network.SnapshotSizeTB that
// ExpectedBytes reports for the full(pruned) tier.
//
// THIS NUMBER HAS NO SOURCE. It is an unsourced placeholder, not a
// measurement and not a figure taken from the learn data — the only size
// the learn data publishes for a chain is the reth snapshot artifact's
// (see Network.SnapshotSizeTB). "Half the snapshot" was chosen when the
// size heuristic was first written and nothing has ever validated it, for
// any client or any chain.
//
// It is kept, unchanged, for one reason only: internal/setup's preflight
// disk floor is derived from ExpectedBytes (minDiskBytesFor = expected *
// 1.10), so changing this silently changes which machines pass preflight.
// Replacing a made-up number with a differently made-up number would be
// churn, not an improvement.
//
// It lives here, alone, as a single named constant so that when real
// figures do arrive the fix is mechanical: delete this constant and give
// ExpectedBytes a per-(client, tier, chain) lookup. Do not scatter `/ 2`
// back through the codebase; the UI mirrors this one value deliberately
// (see FULL_TIER_FRACTION in cmd/valve-node-app/web/src/wizard.ts).
const fullTierFraction = 0.5

// ExpectedBytes returns the expected on-disk dataset size, in bytes, for a
// chain at either the archive or full(pruned) tier. This is the single
// shared implementation of the size heuristic — setup's preflight disk
// check imports it rather than keeping its own copy, so both agree by
// construction.
//
// Be clear-eyed about what this returns. The archive tier reports Valve's
// reth *snapshot* size for the chain, which is the only published figure
// that exists (Network.SnapshotSizeTB) — it is reth-specific and says
// nothing about go-pulse, erigon-pulse or geth. The full tier reports that
// same figure scaled by fullTierFraction, which is an unsourced
// placeholder. Treat both as a coarse floor for "is this disk plausibly
// big enough", never as a promise of how much space a given client will
// actually consume, and never present either to an operator as measured
// fact.
func ExpectedBytes(chainID int, archive bool) (uint64, error) {
	net, ok := NetworkByChainID(chainID)
	if !ok {
		return 0, fmt.Errorf("catalog: no size guidance for chain id %d", chainID)
	}
	sizeTB := net.SnapshotSizeTB
	if !archive {
		sizeTB *= fullTierFraction
	}
	return uint64(sizeTB * 1e12), nil
}

// Client describes one execution or beacon client valve-node-app knows how to
// obtain and wire up.
type Client struct {
	ID   string // "reth" "go-pulse" "erigon-pulse" "geth" "lighthouse-pulse" "prysm-pulse" "lighthouse"
	Kind string // "exec" | "beacon"
	Repo string // canonical source URL

	// ReleaseURL returns a prebuilt-binary URL for goos/goarch, "" if the
	// project publishes none for that platform (=> source build).
	ReleaseURL func(goos, goarch, version string) string

	PinVersion string // known-good default version tag
	BuildCmd   string // source-build recipe: a full sh script, run in a fresh
	// working dir, that ends with the binary installed executable at
	// /usr/local/bin/<ID> (matching what setup's install-step Verify checks).
	Toolchain string // "go" | "rust" — the build toolchain BuildCmd needs
	LearnURL  string

	// DataSubdirs lists the path(s), relative to a WireConfig's DataDir,
	// that this client owns exclusively — the data a "clear & resync"
	// deletes for this client and no other (v0.2 spec §2). Some clients
	// (geth-family, reth, erigon) write these subdirs implicitly under a
	// --datadir that IS the shared DataDir; others (prysm/lighthouse
	// families) are given a --datadir that already points at their own
	// subdir. Either way DataSubdirs names the on-disk owned path(s), and
	// RenderUnits' datadir flags must agree with it (see the golden
	// agreement tests in catalog_test.go).
	DataSubdirs []string

	// SnapshotSupported is true for execution clients that can restore from
	// Valve's free snapshot via `reth download` (reth only today), letting
	// the operator fast-sync instead of syncing from genesis.
	SnapshotSupported bool

	// Gotchas records operator-facing caveats about running this client
	// that the catalog knows but deliberately does not act on — typically
	// because the caveat is conditional on a version or environment
	// valve-node-app does not itself produce, so applying it
	// unconditionally would be wrong (see erigon-pulse's --externalcl note
	// in clients.go for the worked example).
	//
	// These are ported from the `diversityNote` prose on
	// packages/web/src/learn/data/clients.ts, which is where the runbook
	// records them for human readers. Each entry is a complete sentence
	// aimed at an operator, not a flag fragment.
	//
	// Not yet on the wire: internal/server's catalog DTO would have to
	// carry the field before the UI could show it. Until then this is
	// documentation that at least lives next to the client it constrains,
	// rather than only in a monorepo the node operator never reads.
	Gotchas []string
}

// Networks returns the full catalog of supported chains.
func Networks() []Network {
	out := make([]Network, len(networks))
	copy(out, networks)
	return out
}

// NetworkByChainID looks up a network by its chain id.
func NetworkByChainID(id int) (Network, bool) {
	for _, n := range networks {
		if n.ChainID == id {
			return n, true
		}
	}
	return Network{}, false
}

// ClientByID looks up a client by its id.
func ClientByID(id string) (Client, bool) {
	c, ok := clients[id]
	return c, ok
}
