package catalog

import "testing"

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
