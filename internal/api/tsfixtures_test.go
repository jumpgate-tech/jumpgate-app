package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteTSFixtures marshals one populated value of each api type the
// OpenTUI dashboard (tui/) reads. With JUMPGATE_TS_FIXTURES set to a
// directory it writes them there as <name>.json; tui/test/routes.test.ts then
// checks every fixture against the TypeScript mirror, so a renamed Go field
// breaks a TS test. Without the variable it only checks that each value
// marshals.
func TestWriteTSFixtures(t *testing.T) {
	at := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	lag := uint64(3)
	pct := 61.5
	growth := int64(12 << 30)
	days := 40.5

	status := NodeStatus{
		At:      at,
		Exec:    ClientStatus{Active: true, Syncing: false, Head: 21000000, Distance: 1, Peers: 50, State: SyncSynced},
		Beacon:  ClientStatus{Active: true, Syncing: true, Head: 10500000, Distance: 2, Peers: 80, State: SyncSyncing},
		RefHead: 21000003, HeadLag: &lag, Overall: SyncSyncing, DiskUsedPct: &pct,
	}
	disk := DiskView{
		At: at, ExecBytes: 1 << 40, BeaconBytes: 200 << 30, FreeBytes: 800 << 30,
		ExpectedExecBytes: 1300 << 30, ExpectedBeaconBytes: 250 << 30, ExpectedLabel: "mainnet full",
		SyncLabel: "synced", Fit: FitOK,
		History:      []DiskSample{{At: at, UsedBytes: 1 << 40, FreeBytes: 800 << 30}},
		GrowthPerDay: &growth, DaysToFull: &days,
	}
	probeErr := &ProbeError{Error: Error{Message: "box unreachable", Code: CodeUnreachable, Hint: "check ssh"}, At: at}
	fleet := Fleet{
		At: at, IntervalSeconds: 15, Locked: true,
		Rows: []FleetRow{{
			ID: "alpha", Link: LinkAgent, ThisMachine: true, Reachable: true, LastSeen: &at, CheckedAt: &at,
			ChainID: 1, Network: "mainnet", Agent: "0.6.0", Status: &status, Disk: &disk,
			Firewall:   &FirewallSummary{Pass: 5, Warn: 1, Fail: 0, Unknown: 1, Grade: "B"},
			FirewallAt: &at, Jobs: &JobsSummary{Running: 1, NeedsDecision: 1},
			Error:       &Error{Message: "stale", Code: CodeTimeout, Hint: "wait"},
			StatusError: probeErr, DiskError: probeErr, FirewallError: probeErr,
			Stale:      Stale{Status: true, Disk: false, Firewall: true},
			AlertCount: 1, AlertMessage: "disk tight",
			Alerts: []FleetAlert{{Key: "disk", Message: "disk tight", Since: at}},
		}},
	}
	job := JobView{
		Target: "alpha", ID: "j1", Kind: "node.upgrade", Title: "Upgrade the node", State: "needs-decision",
		Step: 1, Steps: []JobStepView{{ID: "s0", Title: "Stop", State: "done"}, {ID: "s1", Title: "Swap binary", State: "asking"}},
		Attempt: 1,
		Question: &JobQuestion{
			ID: "q1", Prompt: "Swap the binary now?", Tier: "confirm", AskedAt: at, Signable: true,
			Options: []JobOption{{ID: "go", Label: "Go", Detail: "swap and restart"}, {ID: "abort", Label: "Stop the job here"}},
		},
		Lease:     JobLease{Owner: "0xabc", Label: "this controller", HeartbeatAt: &at, Fresh: true, Mine: true},
		CreatedAt: at, UpdatedAt: at, Error: "", Driving: true,
	}
	fixtures := map[string]any{
		"fleet":          fleet,
		"nodestatus":     status,
		"diskview":       disk,
		"vaultview":      VaultView{State: "locked", Slots: []string{"passphrase"}, Plain: []string{"relay-token"}, Reason: "idle"},
		"jobsinbox":      JobsInbox{At: at, Jobs: []JobView{job}, Errors: []JobsError{{Target: "beta", Error: Error{Message: "unreachable", Code: CodeUnreachable}}}},
		"jobdetail":      JobDetail{Job: job, Log: []JobLogLine{{Index: 0, Kind: "step", At: at, Text: "stopped the node"}}},
		"jobview":        job,
		"jobdecision":    JobDecision{Decision: "q1", Option: "go"},
		"runningjobs":    RunningJobs{At: at, Jobs: []RunningJob{{Target: "alpha", ID: "j1", Kind: "node.upgrade", State: "needs-decision", Step: 1, Total: 2, NeedsDecision: true}}},
		"joboutput":      []JobOutputLine{{Unit: "geth", Line: "imported block", At: at}},
		"settings":       Settings{AIProvider: "anthropic", AIKeySet: true, RefRPCBase: "https://rpc.example", ProviderKeysSet: []string{"anthropic"}, UpdateNotifyEnabled: true, AIDisclosure: "logs are sent", VaultIdleLockMinutes: 30},
		"uiprefs":        DefaultPrefs(),
		"controllerview": ControllerView{Recorded: "0xabc", Address: "0xabc", Store: "vault", State: "ok", Reason: ""},
		"targets":        []TargetView{{ID: "alpha", Mode: "ssh", SSH: &SSHView{Host: "10.0.0.2", User: "root", KeyPath: "/k", HostKeyFile: "/h", Port: 22}, Agent: &AgentView{Address: "0xdef", Transport: "ssh", PairedAt: at}, Link: LinkAgent, ThisMachine: false}},
		"endpoints":      Endpoints{ExecHTTP: "http://127.0.0.1:8545", BeaconHTTP: "http://127.0.0.1:5052", ExecReachable: true, BeaconReachable: false, ChainIDMatches: true, Access: "ssh", TunnelHint: "ssh -L 8545:127.0.0.1:8545 alpha"},
		"firewall":       []CheckItem{{ID: "ssh", Title: "SSH is key-only", Why: "passwords get guessed", Status: "pass", Detail: "PasswordAuthentication no", Fix: ""}},
		"loghits":        []LogHit{{Unit: "geth", Line: "peer dropped", At: at, Signature: "peer-drop", Severity: "warn", Explain: "a peer left", LearnURL: "https://example.org/peers"}},
		"serviceresult":  ServiceResult{Active: true},
		"gateways":       []GatewaySummary{{ID: "g1", Label: "main", BaseURL: "https://rpc.example", Networks: []GatewayNetwork{{ChainID: 1, Name: "mainnet", URL: "https://rpc.example/1"}}, Tier: "free"}},
		"modulesview":    ModulesView{Modules: []ModuleStatus{{ID: "watch", Served: true, Active: true, Accepting: true, Count: 2}}, GateStale: false, Message: ""},
		"agentcheck":     AgentCheck{Agent: "0xdef", Version: "0.6.0", ChainID: 1, SetUp: true, Signers: 1, Controllers: 1, ElapsedMs: 42},
		"sshcommand":     SSHCommand{Argv: []string{"ssh", "-p", "22", "root@10.0.0.2"}, Display: "ssh root@10.0.0.2"},
		"error":          Error{Message: "wrong passphrase", Code: CodeBadRequest, Hint: "try again"},
	}

	dir := os.Getenv("JUMPGATE_TS_FIXTURES")
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, v := range fixtures {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if dir == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
