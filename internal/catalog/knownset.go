package catalog

import "fmt"

// DefaultValveKey is the shared demo key, so the set works with no setup at
// all. It is a shared quota: if it runs dry, valve's endpoints become the least
// reliable entries in a set that lists them first, which is why the offer says
// so plainly and why the rest of the set is there.
//
// Config.ValveKeys can hold a key of the operator's own per chain and this
// package will use it, but NOTHING IN THE APP WRITES ONE — there is no route
// and no field — so every install is on the shared key until that is built.
// The offer's copy has to match that, not the intention.
const DefaultValveKey = "vk_demo"

// valveURLTemplate and valveWSURLTemplate are valve's unified endpoint on each
// scheme. The key is a PATH segment, not a header, so it has to be substituted
// per chain rather than set once on a client.
//
// There are two because eRPC infers WebSocket capability from the scheme: an
// https:// upstream can never serve eth_subscribe however well the host speaks
// WebSocket, so valve contributes two entries on a chain where both were
// measured, exactly as drpc and publicnode do.
const (
	valveURLTemplate   = "https://one.valve.city/rpc/%s/evm/%d"
	valveWSURLTemplate = "wss://one.valve.city/rpc/%s/evm/%d"
)

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
// A provider that serves both schemes appears TWICE, an https:// entry and a
// wss:// one, because eRPC reads the capability off the scheme. That is why the
// entry count and the provider count are different numbers throughout.
//
// evm:1 holds seven entries for four providers: archive on three of them
// (valve, drpc, merkle — five entries, since valve and drpc are archive on both
// schemes) and WebSocket on three of them (valve, drpc, publicnode — three
// entries, every one wss://). Neither capability rests on a single provider:
// publicnode is NOT archive on Ethereum and merkle has no WebSocket, so they
// are complements rather than alternatives.
//
// evm:369 holds six entries for four providers, WebSocket on two of them
// (valve, publicnode). It keeps the official endpoint but never prefers it: it
// measured 1700ms against 197-274ms for the others.
//
// evm:943 holds six entries for four providers, WebSocket on two of them
// (valve, publicnode).
//
// valve's wss:// entries are on all three chains because all three were
// measured on 2026-07-28 — dialed with internal/wsrpc, upgrade completed, and
// eth_chainId answered 0x1, 0x171 and 0x3af respectively. That is the bar for
// setting the flag at all: a WebSocket claim on an endpoint nobody dialed is
// the exact bug the scheme split exists to prevent.
//
// The valve entries' URLs are filled in per call, since they carry the key;
// which scheme each gets follows its WebSocket flag.
var knownSets = map[int][]KnownEndpoint{
	1: {
		{Provider: "valve", Archive: true},
		{Provider: "valve", WebSocket: true, Archive: true},
		{URL: "https://eth.drpc.org", Provider: "drpc", Archive: true},
		{URL: "wss://eth.drpc.org", Provider: "drpc", WebSocket: true, Archive: true},
		{URL: "https://ethereum-rpc.publicnode.com", Provider: "publicnode"},
		{URL: "wss://ethereum-rpc.publicnode.com", Provider: "publicnode", WebSocket: true},
		{URL: "https://eth.merkle.io", Provider: "merkle", Archive: true},
	},
	369: {
		{Provider: "valve"},
		{Provider: "valve", WebSocket: true},
		{URL: "https://pulsechain-rpc.publicnode.com", Provider: "publicnode"},
		{URL: "wss://pulsechain-rpc.publicnode.com", Provider: "publicnode", WebSocket: true},
		{URL: "https://rpc-pulsechain.g4mm4.io", Provider: "g4mm4"},
		{URL: "https://rpc.pulsechain.com", Provider: "official"},
	},
	943: {
		{Provider: "valve"},
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
			tmpl := valveURLTemplate
			if e.WebSocket {
				tmpl = valveWSURLTemplate
			}
			e.URL = fmt.Sprintf(tmpl, key, chainID)
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

// KnownSetSize is how many UPSTREAMS "Add valve's set" actually adds to a chain
// that has none, which is a different number from KnownSetProviders and is the
// one the redundancy bar has to count against: a bar denominated in providers
// while its numerator counts configured upstreams makes the page's own primary
// action overshoot its target every time. Zero for a chain with no set — the
// caller must render that as "no target", never as a target of zero.
func KnownSetSize(chainID int) int {
	return len(knownSets[chainID])
}
