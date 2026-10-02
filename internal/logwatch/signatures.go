package logwatch

import "regexp"

// signature is one recognized error/failure pattern: a regex over a raw
// journald line, the severity it implies, a canned plain-English
// explanation for an operator, and (where a page exists) a learn.valve.city
// deep link for further reading.
type signature struct {
	name     string
	pattern  *regexp.Regexp
	severity string
	explain  string
	learnURL string
	// requireErrLevel, when true, means pattern alone is not enough — the
	// line must ALSO carry an error-level indicator (see hasErrLevel)
	// before it counts as a match. Used by engine-auth: benign lines like
	// prysm's routine "Finished reading JWT secret from ...jwt.hex" INFO
	// line match the engine-auth pattern too, so without this gate they'd
	// be misclassified as a critical auth failure. Same lesson as
	// internal/setup/steps.go's handshake authErrorLines gate, propagated
	// here since the underlying patterns can each drift independently.
	requireErrLevel bool
}

// errLevelPattern gates requireErrLevel signatures. It started as a copy of
// internal/setup/steps.go's errLevelPattern; it now accepts any level token
// levelSeverity recognizes (so geth's "ERROR[...]", reth's "ERROR" column and
// erigon's "[EROR]" count, not just lighthouse's ERRO and prysm's level=),
// plus the two phrases that mark an auth failure on a line with no level
// field at all. "invalid" is word-bounded so identifiers such as
// invalid_blocks=0 on an INFO line do not pass the gate.
var errLevelPattern = regexp.MustCompile(`(?i)authentication failed|\binvalid\b`)

func hasErrLevel(line string) bool {
	if _, ok := levelSeverity(line); ok {
		return true
	}
	return errLevelPattern.MatchString(line)
}

const learnRPCBase = "https://learn.valve.city/rpc"

// signatures is checked in order, first match wins. Order matters only
// where patterns could otherwise overlap; today's patterns are already
// disjoint but new entries should be added with that in mind.
var signatures = []signature{
	{
		name:     "beacon-stalled",
		pattern:  regexp.MustCompile(`(?i)\bstalled\b`),
		severity: "critical",
		explain:  "The beacon client's sync state machine has stopped making progress. Restart the beacon client to force it to recover.",
		learnURL: learnRPCBase + "#syncing",
	},
	{
		// Word boundaries matter here: without them "401" matches inside
		// any hash, block number or slot (about 1.5% of random 64-hex
		// hashes contain it), turning ordinary error lines into critical
		// JWT alerts. "jwt.hex" and "--jwt-secret" still match, since "."
		// and "-" are boundaries.
		name:            "engine-auth",
		pattern:         regexp.MustCompile(`(?i)\bjwt\b|\b401\b|\bunauthorized\b`),
		severity:        "critical",
		explain:         "The execution and beacon clients can't authenticate to each other over the engine API. This is almost always a mismatched or missing JWT secret file — check both clients point at the same jwt.hex.",
		learnURL:        learnRPCBase + "#jwt-secret",
		requireErrLevel: true,
	},
	{
		name:     "checkpoint-sync-failed",
		pattern:  regexp.MustCompile(`(?i)checkpoint sync`),
		severity: "error",
		explain:  "The beacon client failed to bootstrap from a checkpoint sync provider. Check the checkpoint-sync URL is reachable and serving a recent state, or fall back to genesis sync.",
		learnURL: learnRPCBase + "#syncing",
	},
	{
		name:     "low-peer-count",
		pattern:  regexp.MustCompile(`(?i)low peer count|\b0 peers\b`),
		severity: "warn",
		explain:  "The client has too few (or zero) peers to sync or serve requests reliably. Check outbound P2P ports are open and reachable from the internet.",
		learnURL: learnRPCBase + "#ports",
	},
	{
		name:     "disk-full",
		pattern:  regexp.MustCompile(`(?i)no space left on device`),
		severity: "critical",
		explain:  "The data disk is full. The client will stall or crash-loop until space is freed — prune, expand the volume, or move the data directory to a larger disk.",
	},
	{
		name:     "database-corrupt",
		pattern:  regexp.MustCompile(`(?i)database.*corrupt`),
		severity: "critical",
		explain:  "The client's local database is corrupt, usually from an unclean shutdown or a disk fault. Restore from a snapshot or resync from scratch; do not keep restarting against a corrupt database.",
	},
	{
		name:     "oom-killed",
		pattern:  regexp.MustCompile(`(?i)killed process|\boom\b`),
		severity: "critical",
		explain:  "The kernel OOM-killer terminated the process because the box ran out of memory. Reduce cache sizes, add swap, or move to a box with more RAM.",
	},
	{
		name:     "port-in-use",
		pattern:  regexp.MustCompile(`(?i)bind: address already in use`),
		severity: "critical",
		explain:  "The client failed to bind a listening port because something else already holds it — often a previous instance of the same process that didn't exit cleanly. Check for a stray process and stop it before restarting.",
		learnURL: learnRPCBase + "#ports",
	},
}

// levelSeverity maps an unclassified line's log level to a Hit severity,
// checked in this priority order since a line could in principle carry more
// than one level token.
//
// The level is matched as a whole token, never as a substring: matching
// bare "erro", "warn" or "crit" turned healthy INFO lines reporting
// errors=0, warnings: 0 or fork-choice "criteria" into error, warn and
// critical hits. Two shapes are recognised:
//
//   - an upper-case level column, as written by lighthouse (ERRO, CRIT),
//     geth (ERROR[date|time], WARN [...]), reth (ERROR, WARN after the
//     timestamp) and erigon ([EROR], [WARN]). Upper case only, since the
//     same words in lower case are ordinary message text ("no error").
//   - a level key/value field, as written by prysm and logfmt loggers
//     (level=error, lvl=eror) or JSON loggers ("level":"error"), where the
//     value is matched case-insensitively.
var (
	levelCriticalRe = levelRe(`CRIT|CRITICAL|FATAL`, `crit|critical|fatal`)
	levelErrorRe    = levelRe(`ERROR|ERRO|EROR`, `error|erro|eror|err`)
	levelWarnRe     = levelRe(`WARN|WARNING`, `warn|warning`)
)

// levelRe builds one severity's matcher from its upper-case column tokens
// and its (case-insensitive) key/value spellings.
func levelRe(columnTokens, fieldValues string) *regexp.Regexp {
	return regexp.MustCompile(`\b(?:` + columnTokens + `)\b` +
		`|(?i:\b(?:level|lvl)=["']?(?:` + fieldValues + `)\b)` +
		`|(?i:"level"\s*:\s*"(?:` + fieldValues + `)")`)
}

func levelSeverity(line string) (string, bool) {
	switch {
	case levelCriticalRe.MatchString(line):
		return "critical", true
	case levelErrorRe.MatchString(line):
		return "error", true
	case levelWarnRe.MatchString(line):
		return "warn", true
	}
	return "", false
}
