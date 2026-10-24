package catalog

// private-send is a gateway module: on an Ethereum mainnet gateway it adds
// one upstream per private route, each serving only eth_sendRawTransaction,
// and every other chain-1 upstream ignores that method
// (gatewayModuleUpstreams). It is strict: when no private route answers, the
// send fails with eRPC's error and is never sent to a public upstream. Reads
// stay on the gateway's normal upstreams.
//
// The URLs carry no key and no query string: refunds and builder hints are
// not set in this release. Both answered eth_chainId 0x1 on 2026-10-23.
//
// docs:covers private-send
func init() {
	modules = append(modules, Module{
		ID:        "private-send",
		Name:      "Private send",
		Summary:   "Sends your transactions to Flashbots Protect and MEV Blocker instead of the public mempool.",
		DocsStep:  "private-send",
		Placement: PlaceGateway,
		Chains:    []int{1},
		Unsupported: map[int]string{
			560048: "No private send route exists on Hoodi.",
			369:    "No private send route exists on PulseChain.",
			943:    "No private send route exists on PulseChain.",
		},
		Routes: map[int][]GatewayRoute{1: {
			{ID: "flashbots", Endpoint: "https://rpc.flashbots.net/fast", Methods: []string{"eth_sendRawTransaction"}},
			{ID: "mevblocker", Endpoint: "https://rpc.mevblocker.io", Methods: []string{"eth_sendRawTransaction"}},
		}},
	})
}
