package catalog

import "fmt"

// DefaultValveKey is the shared demo key, so the set works with no setup at
// all. It is a shared quota: if it runs dry, valve's endpoint becomes the
// least reliable entry in a set that lists it first, which is why the UI says
// plainly that a free key of the operator's own removes the limit.
const DefaultValveKey = "vk_demo"

// valveURLTemplate is valve's unified endpoint. The key is a PATH segment, not
// a header, so it has to be substituted per chain rather than set once on a
// client.
const valveURLTemplate = "https://one.valve.city/rpc/%s/evm/%d"

// KnownEndpoint is one entry in a chain's known set, with the two capabilities
// that decide whether it is a good gateway upstream. Both were measured
// (2026-07-28), not read off a provider's docs: Archive means eth_getBalance at
// block 1,000,000 answered rather than erroring, and WebSocket means the
// upgrade actually completed.
type KnownEndpoint struct {
	URL      string
	Provider string
	// WebSocket is true for a wss:// entry. eRPC infers the capability from the
	// scheme, so a provider offering both appears as two entries.
	WebSocket bool
	Archive   bool
}

// knownSets is the hardcoded per-chain set, IN PREFERENCE ORDER. eRPC scores
// upstreams and prefers earlier ones, so this order is load-bearing: it is
// sorted by measured capability, never by name or latency alone.
//
// evm:1 carries two archive sources and three WebSocket sources so neither
// capability rests on one provider — publicnode is NOT archive on Ethereum and
// merkle has no WebSocket, so they are complements rather than alternatives.
//
// evm:369 keeps the official endpoint but never prefers it: it measured
// 1700ms against 197-274ms for the others.
//
// The valve entry's URL is filled in per call, since it carries the key.
var knownSets = map[int][]KnownEndpoint{
	1: {
		{Provider: "valve", WebSocket: true, Archive: true},
		{URL: "https://eth.drpc.org", Provider: "drpc", Archive: true},
		{URL: "wss://eth.drpc.org", Provider: "drpc", WebSocket: true, Archive: true},
		{URL: "https://ethereum-rpc.publicnode.com", Provider: "publicnode"},
		{URL: "wss://ethereum-rpc.publicnode.com", Provider: "publicnode", WebSocket: true},
		{URL: "https://eth.merkle.io", Provider: "merkle", Archive: true},
	},
	369: {
		{Provider: "valve", WebSocket: true},
		{URL: "https://pulsechain-rpc.publicnode.com", Provider: "publicnode"},
		{URL: "wss://pulsechain-rpc.publicnode.com", Provider: "publicnode", WebSocket: true},
		{URL: "https://rpc-pulsechain.g4mm4.io", Provider: "g4mm4"},
		{URL: "https://rpc.pulsechain.com", Provider: "official"},
	},
	943: {
		{Provider: "valve", WebSocket: true},
		{URL: "https://pulsechain-testnet-rpc.publicnode.com", Provider: "publicnode"},
		{URL: "wss://pulsechain-testnet-rpc.publicnode.com", Provider: "publicnode", WebSocket: true},
		{URL: "https://rpc-testnet-pulsechain.g4mm4.io", Provider: "g4mm4"},
		{URL: "https://rpc.v4.testnet.pulsechain.com", Provider: "official"},
	},
}

// KnownSet returns a chain's set with valve's URL resolved against key, or nil
// for a chain with no set. The result is FOUR PROVIDERS; a provider offering
// both schemes contributes two entries, because eRPC reads the capability off
// the scheme.
func KnownSet(chainID int, key string) []KnownEndpoint {
	src, ok := knownSets[chainID]
	if !ok {
		return nil
	}
	if key == "" {
		key = DefaultValveKey
	}
	out := make([]KnownEndpoint, 0, len(src))
	for _, e := range src {
		if e.Provider == "valve" && e.URL == "" {
			e.URL = fmt.Sprintf(valveURLTemplate, key, chainID)
		}
		out = append(out, e)
	}
	return out
}

// KnownSetProviders is the distinct provider count for a chain, which is what
// "four endpoints" means to an operator — the scheme split is an eRPC detail.
func KnownSetProviders(chainID int) int {
	seen := map[string]bool{}
	for _, e := range knownSets[chainID] {
		seen[e.Provider] = true
	}
	return len(seen)
}
