package server

import (
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---------------------------------------------------------------------
// the snapshot key, at the boundary
// ---------------------------------------------------------------------

// The snapshot key is typed by an operator and ends up interpolated into a
// command run on the target. catalog.RethDownloadCommand quotes it, and this
// is the second half of that defence: a key that cannot be a shell fragment
// never reaches command construction at all.
//
// Both halves are kept because either one alone is a single point of failure —
// the quoting is invisible at the call site, and this pattern is one edit away
// from being loosened by someone who only sees it as input validation.
func TestValidateWirePorts_RefusesASnapshotKeyThatCouldBeAShellFragment(t *testing.T) {
	for _, key := range []string{
		`vk_x; touch pwned`,
		`vk_x'; rm -rf /`,
		"vk_x`id`",
		`vk_x$(id)`,
		`vk_x&&id`,
		`vk_x|id`,
		`vk_x id`,
		`vk_x
vk_y`,
		`../../etc/passwd`,
		`no-prefix`,
		``,
		// Too short and too long: the bounds are part of the shape, and a key
		// that is only bounded on one side is a key an attacker can pad.
		`vk_short`,
		`vk_` + strings.Repeat("x", 129),
	} {
		err := validateWirePorts(catalog.WireConfig{
			ChainID:      369,
			ExecID:       "reth",
			ExecSnapshot: true,
			SnapshotKey:  key,
		})
		if err == nil {
			t.Errorf("key %q was accepted", key)
		}
	}
}

func TestValidateWirePorts_AcceptsARealSnapshotKey(t *testing.T) {
	// 8 characters after the prefix is the floor, 128 the ceiling.
	for _, key := range []string{
		"vk_abcd1234",
		"vk_A-B_c-9x",
		"vk_" + strings.Repeat("x", 8),
		"vk_" + strings.Repeat("x", 128),
	} {
		if err := validateWirePorts(catalog.WireConfig{
			ChainID:      369,
			ExecID:       "reth",
			ExecSnapshot: true,
			SnapshotKey:  key,
		}); err != nil {
			t.Errorf("key %q: %v", key, err)
		}
	}
}

// A key can be refused for its SHAPE or for its LENGTH, and those have
// different fixes: one means "you pasted the wrong thing", the other means
// "you pasted part of it". A message that names only the character set sends
// an operator with a truncated key looking for an illegal character that is
// not there.
func TestValidateWirePorts_TheSnapshotKeyRejectionNamesTheLengthRule(t *testing.T) {
	for _, key := range []string{"vk_short", "vk_" + strings.Repeat("x", 129)} {
		err := validateWirePorts(catalog.WireConfig{
			ChainID:      369,
			ExecID:       "reth",
			ExecSnapshot: true,
			SnapshotKey:  key,
		})
		if err == nil {
			t.Fatalf("key %q was accepted", key)
		}
		// The bounds are the whole diagnosis for these two, so both must appear.
		for _, want := range []string{"8", "128"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("key %q: message %q does not state the %s-character bound", key, err, want)
			}
		}
	}
}

// Snapshot restore is reth's alone. Offering it for a client that cannot do it
// produces a wizard that runs a plan whose last step always fails, hours in.
func TestValidateWirePorts_RefusesSnapshotForAClientThatCannotRestore(t *testing.T) {
	err := validateWirePorts(catalog.WireConfig{
		ChainID:      369,
		ExecID:       "go-pulse",
		ExecSnapshot: true,
		SnapshotKey:  "vk_abc123",
	})
	if err == nil {
		t.Fatal("snapshot restore was accepted for a client that does not support it")
	}
	if !strings.Contains(err.Error(), "reth") {
		t.Errorf("the message must name the client that CAN do it: %q", err)
	}
}

// 0 is "use the default" for every port, so it is the one value below 1 that
// must survive validation — rejecting it would make the wizard demand explicit
// ports for a setup that has perfectly good ones.
func TestValidateWirePorts_ZeroMeansDefaultAndIsAccepted(t *testing.T) {
	if err := validateWirePorts(catalog.WireConfig{ChainID: 369, ExecID: "reth"}); err != nil {
		t.Fatalf("an all-defaults config was rejected: %v", err)
	}
}

func TestValidateWirePorts_RejectsPortsOutsideTheRange(t *testing.T) {
	for name, wire := range map[string]catalog.WireConfig{
		"exec http too high":   {ExecHTTPPort: 65536},
		"exec http negative":   {ExecHTTPPort: -1},
		"beacon http too high": {BeaconHTTPPort: 70000},
		"exec p2p negative":    {ExecP2PPort: -8},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateWirePorts(wire); err == nil {
				t.Fatalf("%+v was accepted", wire)
			}
		})
	}
}

// A bind address that is not an IP would be written into a unit file and fail
// at start, hours after the wizard said it was fine.
func TestValidateWirePorts_RejectsABindAddressThatIsNotAnIP(t *testing.T) {
	for _, addr := range []string{"my-host", "100.64.0.999", "0.0.0.0:8545", "::/0"} {
		if err := validateWirePorts(catalog.WireConfig{RPCBindAddr: addr}); err == nil {
			t.Errorf("bind address %q was accepted", addr)
		}
	}
	for _, addr := range []string{"", "127.0.0.1", "100.64.0.7", "::1"} {
		if err := validateWirePorts(catalog.WireConfig{RPCBindAddr: addr}); err != nil {
			t.Errorf("bind address %q was rejected: %v", addr, err)
		}
	}
}

func TestValidateWirePorts_RejectsACheckpointURLThatIsNotHTTP(t *testing.T) {
	for _, cp := range []string{"file:///etc/passwd", "ftp://x.example", "not a url", "https://"} {
		if err := validateWirePorts(catalog.WireConfig{CheckpointURL: cp}); err == nil {
			t.Errorf("checkpoint %q was accepted", cp)
		}
	}
	if err := validateWirePorts(catalog.WireConfig{CheckpointURL: "https://checkpoint.example/x"}); err != nil {
		t.Errorf("a normal checkpoint URL was rejected: %v", err)
	}
}

// A validator nothing calls proves nothing. This is the route-level half:
// the same rejection, through the endpoint an operator actually reaches.
func TestStartSetup_RejectsABadSnapshotKeyBeforeRunningAnything(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)

	res := a.do(t, "POST", "/api/targets/local/setup", map[string]any{
		"ChainID":      369,
		"ExecID":       "reth",
		"BeaconID":     "lighthouse",
		"ExecSnapshot": true,
		"SnapshotKey":  "vk_x; touch pwned",
	})
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if !strings.Contains(body.Error, "SnapshotKey") {
		t.Errorf("error does not name the field: %q", body.Error)
	}
	// Nothing was run against the machine — the rejection is before the plan.
	if len(f.commands(t, "local")) != 0 {
		t.Errorf("a rejected setup still touched the machine: %q", f.commands(t, "local"))
	}
}

// ---------------------------------------------------------------------
// GET /api/host
// ---------------------------------------------------------------------

// The targets UI decides whether local setup is even offered from this, and
// it must be the platform THIS PROCESS runs on — not the browser's, and not a
// target's. On a Mac the answer is "controller only", and getting it from the
// wrong place is how a macOS user is offered a systemd install.
func TestHost_ReportsThePlatformTheAppItselfRunsOn(t *testing.T) {
	a := newAPITestServer(t)

	got := decode[struct {
		OS   string `json:"os"`
		Arch string `json:"arch"`
	}](t, a.do(t, "GET", "/api/host", nil))

	if got.OS != runtime.GOOS {
		t.Errorf("os: got %q, want this process's %q", got.OS, runtime.GOOS)
	}
	if got.Arch != runtime.GOARCH {
		t.Errorf("arch: got %q, want this process's %q", got.Arch, runtime.GOARCH)
	}
}

// ---------------------------------------------------------------------
// GET /api/chainlist/{chainId}
// ---------------------------------------------------------------------

// A chain id that is not a positive number is rejected without a feed fetch or
// a single probe — the route otherwise fans out to public endpoints, and doing
// that for "abc" is a request nobody made.
func TestChainlist_RejectsAChainIDThatIsNotOne(t *testing.T) {
	a := newAPITestServer(t)
	for _, id := range []string{"abc", "0", "-1", "1.5", "%20"} {
		res := a.do(t, "GET", "/api/chainlist/"+id, nil)
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("chain id %q: got %d, want 400", id, res.StatusCode)
		}
	}
}

// ---------------------------------------------------------------------
// pure helpers whose wrong answer is an address nothing can dial
// ---------------------------------------------------------------------

// A wildcard bind names every interface but is not itself a destination —
// macOS refuses a connect to 0.0.0.0 outright. Every URL this app hands an
// operator to click goes through here, so the wrong answer is a copyable link
// that cannot work.
func TestEndpointHost_TurnsABindAddressIntoSomethingDialable(t *testing.T) {
	for bind, want := range map[string]string{
		"":            "127.0.0.1",
		"0.0.0.0":     "127.0.0.1",
		"::":          "[::1]",
		"::0":         "[::1]",
		"127.0.0.1":   "127.0.0.1",
		"100.64.0.7":  "100.64.0.7",
		"::1":         "[::1]",
		"[::1]":       "[::1]",
		"fd00::1":     "[fd00::1]",
		"example.com": "example.com",
	} {
		if got := endpointHost(bind); got != want {
			t.Errorf("endpointHost(%q) = %q, want %q", bind, got, want)
		}
	}
}

// The check is about what the front SERVES, not about whether the operator has
// pointed DNS at their machine yet — so the name is pinned to an address known
// to route there from here. For an SSH target that is the host this app
// already reaches it on; for a local one, the published bind.
func TestTLSDialHost_PinsTheNameToAnAddressThatRoutesFromHere(t *testing.T) {
	tls := &catalog.GatewayTLS{Enabled: true, Hostname: "gw.valve.city"}

	local := config.Target{ID: "local", Mode: "local"}
	if got := tlsDialHost(local, tls); got != "127.0.0.1" {
		t.Errorf("local target: got %q, want the published bind resolved to loopback", got)
	}

	remote := config.Target{ID: "boxa", Mode: "ssh", SSH: &executor.SSHConfig{Host: "boxa.example"}}
	if got := tlsDialHost(remote, tls); got != "boxa.example" {
		t.Errorf("ssh target: got %q, want the host this app reaches it on", got)
	}

	// An ssh target with no host recorded falls back rather than returning
	// empty, which would produce a dial to ":443".
	broken := config.Target{ID: "boxb", Mode: "ssh", SSH: &executor.SSHConfig{Host: "   "}}
	if got := tlsDialHost(broken, tls); got == "" {
		t.Error("an ssh target with no host produced an empty dial address")
	}
}

// An upstream's displayed name must never be empty: it is the row an operator
// clicks to remove or reset something, and an unnamed row is unactionable.
func TestUpstreamName_AlwaysNamesTheRow(t *testing.T) {
	for name, u := range map[string]catalog.GatewayUpstream{
		"explicit id":    {ID: "public", Endpoint: "https://rpc.pulsechain.com"},
		"managed devnet": {Kind: catalog.UpstreamManagedDevnet, TargetID: "boxa"},
		"managed node":   {Kind: catalog.UpstreamManagedNode, TargetID: "boxa"},
		"bare endpoint":  {Endpoint: "https://rpc.pulsechain.com"},
		"nothing at all": {},
		"id beats kind":  {ID: "chosen", Kind: catalog.UpstreamManagedDevnet, TargetID: "boxa"},
	} {
		t.Run(name, func(t *testing.T) {
			got := upstreamName(u)
			if u.ID != "" && got != u.ID {
				t.Errorf("an explicit id must win: got %q, want %q", got, u.ID)
			}
			if u.Managed() && u.ID == "" && !strings.Contains(got, u.TargetID) {
				t.Errorf("a managed row must name its machine: got %q", got)
			}
		})
	}
}

// A dead reference has to say WHICH machine is missing, because the fix is on
// that machine's screen, not on this one.
func TestUnresolvedLabel_NamesTheMachineTheReferencePointsAt(t *testing.T) {
	devnet := unresolvedLabel(catalog.GatewayUpstream{Kind: catalog.UpstreamManagedDevnet, TargetID: "boxa"})
	if !strings.Contains(devnet, "boxa") || !strings.Contains(devnet, "devnet") {
		t.Errorf("devnet label: got %q", devnet)
	}
	node := unresolvedLabel(catalog.GatewayUpstream{Kind: catalog.UpstreamManagedNode, TargetID: "boxb"})
	if !strings.Contains(node, "boxb") || !strings.Contains(node, "node") {
		t.Errorf("node label: got %q", node)
	}
	// An external endpoint has no machine to name, so it must not invent one
	// — a label reading "node on  (unavailable)" is worse than a generic one.
	external := unresolvedLabel(catalog.GatewayUpstream{Endpoint: "https://rpc.pulsechain.com"})
	if strings.Contains(external, " on ") {
		t.Errorf("external label names a machine that does not exist: %q", external)
	}
}
