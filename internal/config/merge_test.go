package config

import (
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
)

// gw is a terse gateway literal for these tables.
func gw(id, target string, fronted bool, nets ...catalog.GatewayNetwork) Gateway {
	c := catalog.GatewayConfig{Networks: nets}
	if fronted {
		c.TLS = &catalog.GatewayTLS{Enabled: true, Hostname: id + ".example"}
	}
	return Gateway{ID: id, Placement: GatewayPlacement{TargetID: target, Backend: "docker"}, Config: c}
}

func devnetUp(target string) catalog.GatewayUpstream {
	return catalog.GatewayUpstream{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: target}
}

func extUp(id, url string) catalog.GatewayUpstream {
	return catalog.GatewayUpstream{ID: id, Kind: catalog.UpstreamExternal, Endpoint: url}
}

// Two gateways on one device is the violation this exists to fix: it means two
// managed eRPC containers, overlapping chains, two pollers against one node.
func TestMergeGatewaysPerTarget(t *testing.T) {
	tests := []struct {
		name        string
		in          []Gateway
		wantIDs     []string
		wantOrphans []string
		check       func(t *testing.T, got []Gateway)
	}{
		{
			name:    "one gateway per target is left alone",
			in:      []Gateway{gw("default", "local", true), gw("other", "remote", false)},
			wantIDs: []string{"default", "other"},
		},
		{
			name: "the fronted gateway survives even when it is not first",
			in: []Gateway{
				gw("edge", "local", false),
				gw("default", "local", true),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
		},
		{
			name: "with no TLS anywhere the DefaultGatewayID survives",
			in: []Gateway{
				gw("edge", "local", false),
				gw("default", "local", false),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
		},
		{
			name: "with neither, the earliest survives",
			in: []Gateway{
				gw("alpha", "local", false),
				gw("beta", "local", false),
			},
			wantIDs:     []string{"alpha"},
			wantOrphans: []string{"valve-node-app-erpc-beta"},
		},
		{
			name: "networks are unioned and the survivor keeps its own door",
			in: []Gateway{
				gw("default", "local", true, catalog.GatewayNetwork{ChainID: 1337, Upstreams: []catalog.GatewayUpstream{devnetUp("local")}}),
				gw("edge", "local", false,
					catalog.GatewayNetwork{ChainID: 369, Upstreams: []catalog.GatewayUpstream{extUp("public-369-1", "https://rpc.pulsechain.com")}},
				),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
			check: func(t *testing.T, got []Gateway) {
				if len(got[0].Config.Networks) != 2 {
					t.Fatalf("want chains 1337 and 369, got %d networks", len(got[0].Config.Networks))
				}
				if !got[0].Config.Fronted() {
					t.Error("the survivor must keep its TLS; a merge changes which chains are served, not the door")
				}
			},
		},
		{
			// The case an endpoint-only key gets wrong: a managed upstream has
			// no endpoint of its own, so both rows are {managed-devnet, local, ""}.
			name: "identical managed upstreams on a shared chain collapse to one",
			in: []Gateway{
				gw("default", "local", true, catalog.GatewayNetwork{ChainID: 1337, Upstreams: []catalog.GatewayUpstream{devnetUp("local")}}),
				gw("edge", "local", false, catalog.GatewayNetwork{ChainID: 1337, Upstreams: []catalog.GatewayUpstream{devnetUp("local")}}),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
			check: func(t *testing.T, got []Gateway) {
				ups := got[0].Config.Networks[0].Upstreams
				if len(ups) != 1 {
					t.Fatalf("the same devnet must not appear twice, got %d: %+v", len(ups), ups)
				}
			},
		},
		{
			name: "a colliding upstream id is re-suffixed rather than dropped",
			in: []Gateway{
				gw("default", "local", true, catalog.GatewayNetwork{ChainID: 1, Upstreams: []catalog.GatewayUpstream{extUp("public-1-1", "https://a.example")}}),
				gw("edge", "local", false, catalog.GatewayNetwork{ChainID: 1, Upstreams: []catalog.GatewayUpstream{extUp("public-1-1", "https://b.example")}}),
			},
			wantIDs:     []string{"default"},
			wantOrphans: []string{"valve-node-app-erpc-edge"},
			check: func(t *testing.T, got []Gateway) {
				ups := got[0].Config.Networks[0].Upstreams
				if len(ups) != 2 {
					t.Fatalf("two different endpoints must both survive, got %d", len(ups))
				}
				if ups[0].ID == ups[1].ID {
					t.Errorf("upstream ids must be unique after a merge, both are %q", ups[0].ID)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, orphans := mergeGatewaysPerTarget(tc.in)

			var ids []string
			for _, g := range got {
				ids = append(ids, g.ID)
			}
			if len(ids) != len(tc.wantIDs) {
				t.Fatalf("gateways: want %v, got %v", tc.wantIDs, ids)
			}
			for i := range ids {
				if ids[i] != tc.wantIDs[i] {
					t.Fatalf("gateways: want %v, got %v", tc.wantIDs, ids)
				}
			}

			var names []string
			for _, o := range orphans {
				names = append(names, o.ContainerName)
			}
			if len(names) != len(tc.wantOrphans) {
				t.Fatalf("orphans: want %v, got %v", tc.wantOrphans, names)
			}
			for i := range names {
				if names[i] != tc.wantOrphans[i] {
					t.Fatalf("orphans: want %v, got %v", tc.wantOrphans, names)
				}
			}

			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}
