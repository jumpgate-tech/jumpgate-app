package catalog

// learnBaseURL is the LearnURL every Network and Client carries, per the
// plan decision that every catalog entry links back to the learn.valve.city
// runbook it is documented in.
const learnBaseURL = "https://learn.valve.city/rpc"

// networks is the source of truth for supported chains. Chain ids,
// checkpoint URLs, and the exec/beacon client ids valid on each are ported
// verbatim from packages/web/src/learn/data/networks.ts.
//
// SnapshotSizeTB below is that file's `snapshot.sizeTB` — the size of
// Valve's reth snapshot artifact for the chain, and nothing more. It was
// previously imported under the name ArchiveSizeTB, which quietly turned a
// reth-artifact size into a claim about what any archive node costs; it
// isn't one. See the field's doc comment on Network for the full caveat,
// and note there is still NO per-client size data in the learn source for
// go-pulse, erigon-pulse or geth, so none is invented here.
var networks = []Network{
	{
		ChainID:          1,
		Name:             "Ethereum",
		CheckpointURL:    "https://beaconstate.ethstaker.cc",
		ExecClients:      []string{"reth", "geth"},
		BeaconClients:    []string{"lighthouse"},
		LearnURL:         learnBaseURL,
		SnapshotSizeTB:   3.6, // networks.ts: snapshot.rethChain "mainnet"
		SyncLabel:        "~12 hrs (snapshot)",
		GenesisSyncLabel: "~10 days (genesis)",
	},
	{
		ChainID:          369,
		Name:             "PulseChain",
		CheckpointURL:    "https://checkpoint.pulsechain.com",
		ExecClients:      []string{"reth", "go-pulse", "erigon-pulse"},
		BeaconClients:    []string{"lighthouse-pulse", "prysm-pulse"},
		LearnURL:         learnBaseURL,
		SnapshotSizeTB:   3.9, // networks.ts: snapshot.rethChain "pulsechain"
		SyncLabel:        "< 1 day (snapshot)",
		GenesisSyncLabel: "~8 days (genesis)",
	},
	{
		ChainID:          943,
		Name:             "PulseChain Testnet v4",
		CheckpointURL:    "https://checkpoint.v4.testnet.pulsechain.com",
		ExecClients:      []string{"reth", "go-pulse", "erigon-pulse"},
		BeaconClients:    []string{"lighthouse-pulse", "prysm-pulse"},
		LearnURL:         learnBaseURL,
		SnapshotSizeTB:   1.2, // networks.ts: snapshot.rethChain "pulsechain-testnet-v4"
		SyncLabel:        "~4 hrs (snapshot)",
		GenesisSyncLabel: "~2 days (genesis)",
	},
}

// defaultERPCUpstreams maps a chain id to the public RPC endpoint(s) shipped
// as eRPC's default archive/backup fallback — the operator can edit or
// remove them and add their own (e.g. a dedicated archive node). Each was
// verified to answer eth_chainId with the matching chain id (2026-07-24).
// They provide failover and cover historical state a local full node can't;
// whether a given public endpoint is archive varies, so operators wanting
// guaranteed historical coverage should add a known archive upstream.
var defaultERPCUpstreams = map[int][]string{
	1:   {"https://ethereum-rpc.publicnode.com"},
	369: {"https://rpc.pulsechain.com"},
	943: {"https://rpc.v4.testnet.pulsechain.com"},
}

// DefaultUpstreams returns the shipped default eRPC fallback upstreams for a
// chain, or nil for an unknown chain.
func DefaultUpstreams(chainID int) []string {
	return defaultERPCUpstreams[chainID]
}

// rethChainName maps a chain id to the --chain value reth expects, ported
// verbatim from the snapshot.rethChain values in networks.ts. erigon-pulse
// (the third independent PulseChain execution codebase) is assumed to
// share the same --chain naming convention as reth/go-pulse's fork family;
// this is not independently confirmed by learn data — see task-3-report.md.
var rethChainName = map[int]string{
	1:   "mainnet",
	369: "pulsechain",
	943: "pulsechain-testnet-v4",
}

// lighthouseNetworkName maps a chain id to the --network value the
// lighthouse client family (lighthouse-pulse and upstream sigp lighthouse)
// expects, per task-3-brief.md Step 3 verbatim.
var lighthouseNetworkName = map[int]string{
	1:   "mainnet",
	369: "pulsechain",
	943: "pulsechain_testnet_v4",
}
