package logwatch

import (
	"testing"
	"time"
)

// TestEngineAuth_MatchesRealAuthFailures pins the engine-auth signature
// against auth-failure lines in the formats the supported clients actually
// write: prysm's logfmt, lighthouse's abbreviated level tag, and the
// geth/reth/erigon level columns. Each must still be reported as a
// critical engine-auth failure once the pattern is word-bounded.
func TestEngineAuth_MatchesRealAuthFailures(t *testing.T) {
	lines := []string{
		// prysm
		`time="2026-07-23 03:32:09" level=error msg="Could not connect to execution endpoint" error="401 Unauthorized: invalid JWT token" prefix=execution`,
		`level=error msg="401 Unauthorized invalid JWT"`,
		// lighthouse
		`Jul 23 06:00:00.000 ERRO Error during execution engine upcheck   error: HttpClient(url: http://127.0.0.1:8551/, kind: status, detail: 401 Unauthorized), service: exec`,
		`ERRO Failed to connect to execution client err="401 Unauthorized: invalid jwt"`,
		// geth-style level column ("ERROR[date|time]")
		`ERROR[07-23|03:00:00.000] Engine API request rejected            err="401 Unauthorized"`,
		// reth (tracing format)
		`2026-07-23T03:00:00.000000Z ERROR rpc::auth: JWT validation failed: unauthorized`,
		// erigon (log15 abbreviated level)
		`[EROR] [07-23|03:00:00.000] [engine] request unauthorized, bad jwt`,
		// lighthouse warning-level auth failure
		`Jul 23 06:00:00.000 WARN Execution engine auth failed   error: Auth(InvalidToken), jwt: /var/lib/valve-node-app/943/jwt.hex`,
		// no level at all, but explicitly invalid jwt — the gate's "invalid"
		// alternative exists for exactly this shape.
		`401 Unauthorized invalid jwt`,
	}
	for _, line := range lines {
		hit, ok := classify("u", line, time.Now())
		if !ok || hit.Signature != "engine-auth" || hit.Severity != "critical" {
			t.Errorf("classify(%q) = %+v ok=%v, want critical engine-auth", line, hit, ok)
		}
	}
}

// TestEngineAuth_IgnoresIncidental401 is the regression for the engine-auth
// misfire: "401" (or "jwt") appearing inside a hash, block number or
// unrelated identifier on an error line must not be reported as a JWT
// secret failure. Roughly 1.5% of random 64-hex hashes contain "401", so
// an unbounded pattern turned ordinary errors into critical auth alerts.
func TestEngineAuth_IgnoresIncidental401(t *testing.T) {
	lines := []string{
		`ERRO Block processing failed hash=0x9a3f401bc7d2e8f1a0b9c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f70819 error="state root mismatch"`,
		`ERROR[07-23|03:00:00.000] Failed to import block number=14012345 err="unknown ancestor"`,
		`level=error msg="Could not process block" slot=8401 root=0x401a`,
		`2026-07-23T03:00:00.000000Z ERROR reth::stages: stage failed block=24019 err=Database(Read)`,
		`[EROR] [07-23|03:00:00.000] Invalid block peer=enode://4401abc@10.0.0.1 hash=0xjwtx`,
	}
	for _, line := range lines {
		hit, ok := classify("u", line, time.Now())
		if ok && hit.Signature == "engine-auth" {
			t.Errorf("classify(%q) reported engine-auth; an incidental 401/jwt substring is not an auth failure", line)
		}
	}
}

// TestLevelSeverity_IgnoresLevelWordsInsideOtherWords is the regression for
// the substring fallback: an INFO line reporting "errors=0", "warnings: 0"
// or mentioning "criteria" is healthy output, not an error, warning or
// critical hit.
func TestLevelSeverity_IgnoresLevelWordsInsideOtherWords(t *testing.T) {
	lines := []string{
		`INFO Sync summary blocks=120 errors=0`,
		`INFO Config check complete warnings: 0`,
		`INFO Fork choice criteria satisfied slot=987654`,
		`level=info msg="Peer scoring" errors=0 warnings=0`,
		`2026-07-23T03:00:00.000000Z  INFO reth::cli: Status connected_peers=12 errors=0`,
		`INFO[07-23|03:00:00.000] Imported new chain segment  number=1 fatalities=0`,
		`[INFO] [07-23|03:00:00.000] [p2p] GoodPeers  eth68=12 errored=0`,
		`Jul 23 06:00:00.000 INFO Synced  slot: 100, warnings_suppressed: true`,
	}
	for _, line := range lines {
		if hit, ok := classify("u", line, time.Now()); ok {
			t.Errorf("classify(%q) = %+v, want no hit for a healthy INFO line", line, hit)
		}
	}
}

// TestLevelSeverity_ReadsTheLevelFieldOfKnownFormats pins the level forms
// each supported client writes, so token matching does not lose any of
// them.
func TestLevelSeverity_ReadsTheLevelFieldOfKnownFormats(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{`ERROR panic: index out of range [3] with length 3`, "error"},
		{`WARN unexpected response from peer, dropping connection`, "warn"},
		{`Jul 23 06:00:00 ERRO Head is stuck, prune check failed`, "error"},
		{`Jul 23 06:00:00 CRIT Beacon node shutting down`, "critical"},
		{`FATAL failed to start`, "critical"},
		{`time="2026-07-23 03:32:09" level=error msg="Could not process block"`, "error"},
		{`time="2026-07-23 03:32:09" level=warning msg="Slow block processing"`, "warn"},
		{`time="2026-07-23 03:32:09" level=fatal msg="Could not start node"`, "critical"},
		{`t=2026-07-23T03:00:00+0000 lvl=eror msg="Failed to import block"`, "error"},
		{`{"level":"error","msg":"Could not process block"}`, "error"},
		{`ERROR[07-23|03:00:00.000] Failed to import block`, "error"},
		{`WARN [07-23|03:00:00.000] Served eth_call`, "warn"},
		{`CRIT [07-23|03:00:00.000] Fatal startup error`, "critical"},
		{`2026-07-23T03:00:00.000000Z ERROR reth::stages: stage failed`, "error"},
		{`2026-07-23T03:00:00.000000Z  WARN reth::net: peer dropped`, "warn"},
		{`[EROR] [07-23|03:00:00.000] Staged sync failed`, "error"},
		{`[WARN] [07-23|03:00:00.000] [p2p] peer timeout`, "warn"},
	}
	for _, tc := range cases {
		hit, ok := classify("u", tc.line, time.Now())
		if !ok {
			t.Errorf("classify(%q): no hit, want severity %q", tc.line, tc.want)
			continue
		}
		if hit.Severity != tc.want {
			t.Errorf("classify(%q).Severity = %q, want %q", tc.line, hit.Severity, tc.want)
		}
	}
}
