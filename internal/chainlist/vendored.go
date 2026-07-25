package chainlist

// Vendored snapshot of the chainlist feed for the chains valve-node-app knows
// how to provision (catalog.Networks): Ethereum, PulseChain and PulseChain
// Testnet v4.
//
// Why a baked-in copy exists: the setup wizard runs on freshly imaged boxes
// and, for some operators, on racks with no egress at all. A node that cannot
// reach chainid.network would otherwise get an empty fallback list and an
// erpc.yaml with nothing behind the local node. Falling back to a known-good
// list keeps the failure mode "possibly stale" instead of "silently empty" —
// and every vendored entry still goes through the same eth_chainId probe as a
// freshly fetched one, so a retired endpoint is caught at discovery time
// rather than at 3am.
//
// Maintenance: refresh this from the live feed when the chain lists move.
// Nothing breaks if it goes stale — stale entries fail their probe and are
// reported as rejected — but the offline experience degrades as the list
// shrinks.

// VendoredAsOf records when these lists were taken from
// https://chainid.network/chains.json.
const VendoredAsOf = "2026-07-25"

// vendoredRPCs is the snapshot, in the feed's own order.
//
// Chains 369 and 943 are the feed's rpc arrays verbatim as of VendoredAsOf.
//
// Chain 1 is a deliberate subset. The feed lists ~18 endpoints for Ethereum
// mainnet, but most are unsuitable as a general eRPC fallback: two are
// API-key templates (filtered anyway), and the bulk of the rest are
// transaction-privacy relays (rpc.flashbots.net, rpc.mevblocker.io and their
// /fast, /noreverts, /fullprivacy variants, rpc.blocknative.com/boost) whose
// job is accepting eth_sendRawTransaction, not serving general reads. Baking
// those in would hand eRPC upstreams that answer eth_chainId — so they would
// pass the probe — while being poor at everything eRPC would actually route
// to them. What remains are the general-purpose public endpoints, each
// verified against the feed on VendoredAsOf.
var vendoredRPCs = map[int][]string{
	1: {
		"https://ethereum-rpc.publicnode.com",
		"wss://ethereum-rpc.publicnode.com",
		"https://cloudflare-eth.com",
		"https://mainnet.gateway.tenderly.co",
		"wss://mainnet.gateway.tenderly.co",
		"https://eth.drpc.org",
		"wss://eth.drpc.org",
	},
	369: {
		"https://rpc.pulsechain.com",
		"https://pulsechain-rpc.publicnode.com",
		"wss://pulsechain-rpc.publicnode.com",
		"https://rpc-pulsechain.g4mm4.io",
	},
	943: {
		"https://rpc.v4.testnet.pulsechain.com",
		"wss://rpc.v4.testnet.pulsechain.com",
		"https://pulsechain-testnet-rpc.publicnode.com",
		"wss://pulsechain-testnet-rpc.publicnode.com",
		"https://rpc-testnet-pulsechain.g4mm4.io",
		"wss://rpc-testnet-pulsechain.g4mm4.io",
	},
}

// Vendored returns the baked-in RPC list for a chain and whether one exists.
// The returned slice is a copy: callers routinely sort, trim and probe these,
// and a shared backing array would let one caller corrupt the snapshot for
// the rest of the process.
func Vendored(chainID int) ([]string, bool) {
	rpcs, ok := vendoredRPCs[chainID]
	if !ok {
		return nil, false
	}
	out := make([]string, len(rpcs))
	copy(out, rpcs)
	return out, true
}

// VendoredChainIDs returns the chain ids covered by the snapshot, in the
// catalog's own order, so a caller can tell an offline operator which chains
// still get fallback upstreams.
func VendoredChainIDs() []int {
	// Fixed order rather than a map range: this is user-visible output and
	// must not shuffle between runs.
	return []int{1, 369, 943}
}
