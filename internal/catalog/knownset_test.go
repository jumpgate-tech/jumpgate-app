package catalog

import (
	"strings"
	"testing"
)

// The order is the whole point: eRPC prefers earlier upstreams, so the set is
// sorted by what each endpoint can do, not by name or by ping. These were
// measured on 2026-07-28 — see the design doc.
func TestKnownSetOrderAndContents(t *testing.T) {
	// Four PROVIDERS is the promise. A provider offering both http and wss
	// contributes two entries, because eRPC reads the capability off the
	// scheme — so entry count and provider count are different numbers and the
	// test must not conflate them.
	if got := KnownSetProviders(1); got != 4 {
		t.Fatalf("evm:1 is four providers, got %d", got)
	}
	eth := KnownSet(1, DefaultValveKey)
	var order []string
	for _, e := range eth {
		if len(order) == 0 || order[len(order)-1] != e.Provider {
			order = append(order, e.Provider)
		}
		if e.URL == "" {
			t.Errorf("every entry needs a URL; %q has none", e.Provider)
		}
	}
	wantProviders := []string{"valve", "drpc", "publicnode", "merkle"}
	for i, want := range wantProviders {
		if i >= len(order) || order[i] != want {
			t.Errorf("evm:1 provider %d: want %q, got %v — order encodes capability", i, want, order)
		}
	}

	if got := KnownSetProviders(369); got != 4 {
		t.Fatalf("evm:369 is four providers, got %d", got)
	}
	pls := KnownSet(369, DefaultValveKey)
	// The official endpoint measured 1700ms against 197-274ms for the rest, so
	// it is kept but never preferred.
	if pls[len(pls)-1].Provider != "official" {
		t.Errorf("evm:369 must end on the official endpoint, got %q", pls[len(pls)-1].Provider)
	}

	if got := KnownSet(999999, DefaultValveKey); got != nil {
		t.Errorf("an unknown chain has no set, got %+v", got)
	}
}

// Neither capability may rest on a single provider: publicnode is not archive
// on Ethereum and merkle has no WebSocket, so a set carrying only one of them
// loses a capability outright.
func TestKnownSetCoversBothCapabilities(t *testing.T) {
	var ws, archive int
	for _, e := range KnownSet(1, DefaultValveKey) {
		if e.WebSocket {
			ws++
		}
		if e.Archive {
			archive++
		}
	}
	if ws < 2 {
		t.Errorf("evm:1 needs more than one WebSocket source, got %d", ws)
	}
	if archive < 2 {
		t.Errorf("evm:1 needs more than one archive source, got %d", archive)
	}
}

// The WebSocket flag is a claim about the SCHEME and nothing else, because
// that is all eRPC reads. An https:// entry flagged WebSocket puts a
// "websocket" tag in the offer modal on a URL that will never serve
// eth_subscribe, while the chain row's own scheme check refuses to count it —
// the same screen saying two different things about one URL. Assert the
// invariant on every chain rather than on the one that regressed.
func TestKnownSetWebSocketFlagMatchesTheScheme(t *testing.T) {
	for chainID := range knownSets {
		for _, e := range KnownSet(chainID, DefaultValveKey) {
			ws := strings.HasPrefix(e.URL, "wss://") || strings.HasPrefix(e.URL, "ws://")
			if ws != e.WebSocket {
				t.Errorf("evm:%d %s: WebSocket=%v but the URL is %q — eRPC infers the capability from the scheme, so these cannot disagree",
					chainID, e.Provider, e.WebSocket, e.URL)
			}
		}
	}
}

// valve contributes both schemes wherever both were measured, exactly as drpc
// and publicnode do, and stays first in preference order. All three chains
// were measured on 2026-07-28 — see knownSets' comment for what "measured"
// means here.
func TestKnownSetValveHasBothSchemesWhereMeasured(t *testing.T) {
	for _, chainID := range []int{1, 369, 943} {
		set := KnownSet(chainID, DefaultValveKey)
		if set[0].Provider != "valve" || set[1].Provider != "valve" {
			t.Fatalf("evm:%d must open on valve's two entries, got %q then %q",
				chainID, set[0].Provider, set[1].Provider)
		}
		if set[0].WebSocket {
			t.Errorf("evm:%d: valve's https entry must not claim WebSocket: %+v", chainID, set[0])
		}
		if !set[1].WebSocket || !strings.HasPrefix(set[1].URL, "wss://") {
			t.Errorf("evm:%d: valve needs a wss:// twin, got %+v", chainID, set[1])
		}
	}
}

// KnownSetSize is what the redundancy bar counts against, so it must be the
// ENTRY count — what the button actually adds — not the provider count.
func TestKnownSetSizeIsTheEntryCount(t *testing.T) {
	for chainID, want := range map[int]int{1: 7, 369: 6, 943: 6, 999999: 0} {
		if got := KnownSetSize(chainID); got != want {
			t.Errorf("KnownSetSize(%d) = %d, want %d", chainID, got, want)
		}
	}
	for chainID := range knownSets {
		if got, want := KnownSetSize(chainID), len(KnownSet(chainID, DefaultValveKey)); got != want {
			t.Errorf("evm:%d: KnownSetSize is %d but the set adds %d", chainID, got, want)
		}
	}
}

func TestKnownSetSubstitutesTheValveKey(t *testing.T) {
	got := KnownSet(1, "vk_mine")[0].URL
	want := "https://one.valve.city/rpc/vk_mine/evm/1"
	if got != want {
		t.Errorf("valve URL: want %q, got %q", want, got)
	}

	// An empty key must not produce a URL with an empty path segment — that
	// would 404 on every call while looking configured.
	if got := KnownSet(1, "")[0].URL; got != "https://one.valve.city/rpc/vk_demo/evm/1" {
		t.Errorf("an empty key falls back to the demo key, got %q", got)
	}
}
