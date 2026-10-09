# Durable Jobs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Long operations on a box run as durable, signed, adoptable jobs: transient systemd units started by the agent, described by an on-box manifest, recorded in a hash-chained log the agent signs, held by a controller lease, paused at signed decision points, and found again by any paired controller after either side restarts. The first job is `node.migrate-names`, which renames a box's `valve-node-app` account, units and data root to `jumpgate-node`.

**Architecture:** Three new packages carry the box side. `internal/joblog` is the pure signed log: an EIP-712 `JobLogEntry` type, hash chaining, and verification. Controllers use it too. `internal/jobs` is the agent's engine: the manifest store, leases, the systemd-run runner, the plan registry, the step runner behind `jumpgate agent job-step`, and the `node.migrate-names` plan. `internal/agent` answers seven new intent kinds through a `jobs.Manager` and reconciles finished units every 5 s. On the controller, `internal/server/jobs.go` holds a jobs hub. The hub lists and verifies jobs on paired boxes, serves the inbox, drives the jobs whose lease it holds (step-level v1 driver), resumes its own leases at startup, and fills `jobs` on fleet rows. `internal/apiclient`, the TUI (inbox, ⚑ count, bell) and the CLI sit on top. `internal/catalog` learns both name sets, so every reader accepts a box before and after the rename.

**Tech Stack:** Go 1.25 (`go 1.25.0` directive), `systemd-run`/`systemctl`/`journalctl` on the box through `executor.Executor`, secp256k1 EIP-712 through the existing `internal/eip712` and `internal/signer`, net/http + SSE over the existing unix socket, Bubble Tea v2 (already in `go.mod`). No new module.

**Spec:** `docs/superpowers/specs/2026-10-02-controller-agent-architecture-design.md`. Its sections "Durable jobs", "Naming" and "Build order" row 2 are sub-project 2. Related parts: `2026-10-02-foundations-design.md` lines 20–40 (what sub-project 1 left for this one); `2026-10-05-tui-design.md` D27, D3, screen 6, the Settings bell, and the reserved `jobs` field on fleet rows (`internal/api/fleet.go`); `2026-10-05-platform-support-design.md` D19 and "The agent stays Linux-only by design". Executors read the umbrella spec's "Durable jobs" section before Task 1.

## Open questions

These need a human. Each one changes security or product behaviour, and the spec does not settle it. The plan implements the **recommended default** and names the task that owns each choice, so changing an answer later touches only that task.

| # | Question | Recommended default (implemented) | Owner |
|---|---|---|---|
| Q1 | Which tier does `node.migrate-names` need? The umbrella lists setup, resync, wipe, key and policy changes as routine **and** approval; a rename is not listed. The approval tier does not exist yet (TUI spec, "Out of scope"). | **Routine.** It stops and starts the node and renames an account but deletes no data. An operator who wants approval adds `"node.migrate-names": ["routine","approval"]` to `policy.json` `kinds`, which already works. | Task 4 (`DefaultRequirements`) |
| Q2 | Where does the data go? The spec says `/var/lib/valve-node-app` is migrated but names no new path. | **`/var/lib/jumpgate-node`** (the account's name, kept apart from the agent's own `/var/lib/jumpgate`). A data dir outside the legacy root stays where it is. | Task 1 (`catalog.CurrentNames`), Task 8 |
| Q3 | The spec also lists "container names and `VALVE_*` names". On boxes, the only legacy-named things the node owns are the account, the two units and the data root. The eRPC/Caddy/devnet containers, `valve-node-app-erpc.service` and the gateway data are driven by the controller's legacy SSH executor by fixed name. No node unit carries a `VALVE_*` variable: `VALVE_API_KEY` is the Valve RPC provider's placeholder in controller config, and `VALVE_CERT_URL` is a controller environment variable. | **Out of scope for sub-project 2.** Gateways and containers are renamed when sub-project 6 moves them onto agents. `node.migrate-names` refuses, changing nothing, on a box where any other unit runs as `valve-node-app` (the systemd eRPC backend, for example). | Task 8 (preflight) |
| Q4 | Does a controller take over a stale lease by itself, or only when an operator says so? The spec says "takes the lease over (lease is stale)". | **Only when an operator says so** (`adopt` in the TUI, CLI or API). A controller resumes **its own** leases automatically after it restarts. Automatic takeover lets two controllers that both lost and regained the network flap over a job. | Task 12 |
| Q5 | Approval-tier decisions before the approval tier exists. | The protocol carries each question's tier, and the agent enforces it (the `job.decide` intent must also carry an enrolled approval signature). Until approval signers ship, the server answers `409 approval_unavailable` instead of sending. `node.migrate-names` asks only routine questions. The readable EIP-712 struct that approval-tier decisions need for wallet screens is left to the approval-tier sub-project. | Task 9, Task 12 |
| Q6 | Does a **new** setup use the new names? The foundations spec's `DefaultDataDir` comment says sub-project 2 changes the prefix "once". | **Yes.** A target set up for the first time after this lands gets `Names: "jumpgate-node"` and `/var/lib/jumpgate-node/<chain>`. A target that already has a wire keeps its names and data dir on every setup re-run, including a re-run from the web UI that sends no data dir. | Task 1 |
| Q7 | "The agent keeps a signed, append-only log of every intent." Lease **renewals** arrive every 20 s per held job. | **Log every intent that changes the job except renewals.** That covers creation, lease take and release, advance and decide. A renewal only moves the heartbeat in the manifest. Logging renewals would add about 1,000 entries to a 6-hour job and tell an adopter nothing it cannot read from the lease. | Task 7 |
| Q8 | What does "abort" do to a half-migrated box? | **No automatic rollback in v1.** Every step is idempotent, so the remedy is to run the job again (`retry`) or start a new one. Each step's abort option says exactly what state it leaves, for example "the account is renamed but the units are not yet: the node cannot start until a new `node.migrate-names` job finishes". | Task 8 |
| Q9 | A transient unit does not survive a **reboot**. The spec covers agent restarts and controller disconnects only. | When the agent finds a running step's unit gone with no result file, it records `step.lost` and asks a routine question ("retry" / "abort"). Nothing reruns silently, because a step killed mid-way (a `mv`, a `usermod`) needs a human to look first. | Task 7 |

## Decisions made in this plan

These are not open. Each one is recorded so a reviewer can object, together with what it costs if it is wrong.

| # | Decision | Why | Cost if wrong |
|---|---|---|---|
| J1 | One non-terminal job per box. `service.action` is refused with `job_active` while a job runs. | `node.migrate-names` stops and renames the units that `service.action` drives. Two jobs on one node can only collide. | An operator waits for a job before restarting a client; the refusal says which job. |
| J2 | Lease TTL 90 s; the holder renews every 20 s; the driver ticks every 5 s; the agent reconciles every 5 s and on every job intent. | Three missed renewals mean the controller is gone. 5 s keeps a step boundary prompt without load. | Constants in one place each (`jobs.LeaseTTL`, `jobsRenewEvery`, `jobsDriveEvery`, `agent.ReconcileEvery`). |
| J3 | Units are `jumpgate-job-<id>-<step>-<attempt>` (the spec's `<step>` part is `<step>-<attempt>`), started with `--collect --property=Type=exec`. A step's **result file** is the truth about how it ended, never the unit's state. | Without the attempt, a retry collides with the failed unit still loaded. `--collect` unloads finished units; the result file outlives both the unit and the agent. | A unit name is at most 13+16+1+32+1+4 characters, well inside systemd's 256. |
| J4 | The agent process is the only writer of manifests and the only signer. A step process (`jumpgate agent job-step`) reads the manifest and writes one unsigned result file, which the agent folds into the signed log. | The agent key is a systemd credential of the agent unit alone. A transient unit must not hold it. | A step's own output is not signed. Its journal lines are output, not history. The signed log records the outcome. |
| J5 | A job id is 16 lowercase hex characters from `crypto/rand`. | Safe in unit names, paths and journal patterns without quoting. | None. |
| J6 | Terminal jobs older than 30 days are pruned, and at most 100 terminal jobs are kept (the newest). | The log is audit data, not an archive. | An operator wanting older history copies `/var/lib/jumpgate/jobs` first. |
| J7 | Log bodies are stored as bytes (base64 in JSON), hashed as stored. | The same rule as intent payloads: no JSON canonicalisation, ever. | None. |
| J8 | `job.output` is `journalctl -u 'jumpgate-job-<id>-*' --after-cursor … -o json`, bounded like `logs.since` (2,000 lines, 8 KiB per line, 1 MiB per answer, 24 h back). | Output goes to the journal (spec); remote controllers cannot run journalctl, so it travels in a signed answer. | None. |
| J9 | The agent refuses every job kind with `jobs_unsupported` on any OS but Linux, and on Linux without systemd as init (`/run/systemd/system` absent) or without `systemd-run`. Controllers on macOS, Windows and Linux list, watch, adopt and decide alike: that is only intents. | The agent is Linux-only by design (platform spec); macOS keeps it only for its tests (D19), and Windows refuses `jumpgate agent` outright. A box without systemd cannot run a transient unit. | A Linux box without systemd cannot run jobs; the refusal names the reason. |
| J10 | The marker `/etc/jumpgate/names.json`, written by `node.migrate-names`, overrides the names in `node.json`. | A re-pair writes `node.json` from the controller's copy of the wire. A controller that missed the migration would otherwise point the agent back at units that no longer exist. | One more file the agent reads per intent; it is tiny. |
| J11 | A job log that fails verification makes the server answer `502 job_log_invalid` (CLI exit 4). The detail route then shows none of that log. | Unverified history must never read as history. | An operator with a damaged box sees an error, not a partial log; the message names the first bad entry. |
| J12 | The TUI's host-detail Jobs tab stays out of v1 (TUI spec: "Diagnostics and Jobs tabs are not in v1"). The fleet-wide inbox, the ⚑ count, the bell and the fleet `jobs` column ship here. | The inbox already filters by box with `/`. | A per-box tab is an additive screen later. |

### Rulings applied

The pre-flight scan (`.superpowers/sdd/2026-10-09-durable-jobs/preflight-scan.md`) raised these; each ruling is decided and is already written into the task it names.

- **P1** Task 5 `TestPruneKeepsRecentAndUnfinishedJobs` moves the store's clock 31 days on after creating the old jobs (D-J6 ages from the last change).
- **P2** Task 5 `TestLoadDetectsATruncatedLog` writes one whole entry without an extra `"\n"`, so the length check fires; a torn line has its own test.
- **P3** Task 8 `TestRenameAccountRenamesUserThenGroup` scripts `id -u jumpgate-node` as `fail1, idOK("998")`; `mnRenameAccount` probes the new account before and after the rename.
- **P4** Task 12 `TestARestartedServerResumesItsOwnJobs` makes no HTTP request and checks that only `sweep` started the driver.
- **P5** Task 12 adds the test-only `Config.noJobsDriver` and `newBoxRig(t, withoutDriver)` for Task 11's two mid-job tests.
- **P6** Execution is sequential; every commit step adds exact paths, never `-A`, `.` or a directory.
- **P7** Task 12's `POST /api/jobs/{target}` starts only `jobStartKinds` (`node.migrate-names` in v1); others get 400 `job_kind_not_allowed` (Task 10).
- **P8** Task 5 creates `internal/fsatomic` and moves the agent's `writeAtomic`/`syncDir` into it; agent, store and plan share one copy.
- **P9** Task 9 generalises `logsSince` over units and an item type; `job.output` reads through it.
- **P10** Tasks 15 and 16 build the user-facing names from `catalog.LegacyNames`/`CurrentNames`, never a literal.
- **P11** Task 9 adds `setupExisting`: with no controller wire, setup asks a paired box's `agent.info` for its names and keeps legacy ones.
- **P12** Task 10 adds `unknown_plan`; Files lists, Task 12 imports and the stale `cachedRecord` comment are fixed.
- **P13** Task 16 adds `jumpgate jobs output JOB`, the first consumer of the output stream; the TUI follows later (J12).

## Preconditions

- Branch `feat/durable-jobs` from `main` at or after `b3eeb71`. Baseline: `go test ./...` green on macOS; record it.
- Docker is available for `scripts/e2e-agent.sh` (Task 17). Without it, Task 17's run step is done in CI.
- Tests isolate `HOME` with `testutil.Home(t)` where the surrounding file does, and gate agent persistence on `testutil.RequireUnix(t)`. The agent fsyncs directories, which Windows cannot do.
- Never push without the user's permission.

## Global Constraints

- Module path `github.com/valve-tech/jumpgate`. The product and binary are `jumpgate`. New code never says "valve-node-app" except through `catalog.LegacyNames`, the one place the legacy names live.
- No new Go module. The `go` directive stays `go 1.25.0`.
- The agent never accepts shell. Every new intent kind is typed, and nothing from a payload reaches a command line except a job id matching `^[0-9a-f]{16}$`, a step id matching `^[a-z0-9][a-z0-9-]{0,31}$`, a journal cursor matching the existing `cursorRE`, and integers.
- One signing scheme: secp256k1 over EIP-712 with the domain `{name: "jumpgate", version: "1"}`, with no `chainId`. The frozen `Intent` and `Receipt` schemas do not change. `JobLogEntry` is a new type with its own frozen-schema test.
- On-box paths: manifests at `/var/lib/jumpgate/jobs/<id>.json` (spec), logs at `/var/lib/jumpgate/jobs/<id>.log`, step results at `/var/lib/jumpgate/jobs/<id>.d/<step>-<attempt>.json`, all `0600` in a `0700` directory.
- Manifest `format` is `1`, and `driver` is `"controller"` in v1. The agent refuses to advance a job whose `format`, `driver` or `planVersion` it does not know (`unknown_plan`), and still lets it be read.
- Agent clock rules: lease freshness is judged by the agent's clock, never the controller's. Answers carry `agentTime` so a controller can show ages without skew.
- Every `/api/*` error body is `{"error", "hint"?, "code"}` with a code from `internal/api`. The server never writes a code as a string literal.
- The TUI holds no keys and does no work. Destructive choices need typed confirmation in the TUI (against mistakes) and the right signature tier on the box (for authority).
- Every commit message is conventional (`feat(…)`, `fix(…)`, `test(…)`, `refactor(…)`, `docs: …`) and ends with the line `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Comments are full sentences that explain *why*, at the density of the surrounding code.

## Review Focus

These five input classes or failure modes are the ones most likely to bite a person, and no single task's happy path exercises them. Each line names the test that pins it, and that test lives in the owning task.

1. **The data root, or something under it, is a mount point** (a second disk at `/var/lib/valve-node-app`, listed in `/etc/fstab`). Expected: the job never runs `mv`. It asks "keep the data where it is" or "check again", and the node keeps working with its data at the old path. Test: Task 8 `TestMoveDataNeverMovesAMountedTree`.
2. **The agent restarts, or the box reboots, while a step runs.** Expected: a step that finished leaves its result file, and the restarted agent folds it into the signed log exactly once. A step that never reported becomes `step.lost` and a question. Nothing reruns on its own. Tests: Task 7 `TestReconcileFoldsAResultWrittenWhileTheAgentWasDown` and `TestLostStepAsksBeforeRerunning`; Task 17 `TestE2EMigrateNamesSurvivesAnAgentRestart`.
3. **Two controllers.** B lists the job while A holds a fresh lease, so B watches read-only and B's decide is `409 lease_held`. Once A has been silent for 90 s, B can adopt; A's driver then stops on `lease_held` instead of fighting. Tests: Task 7 `TestLeaseFreshRefusesAnotherController`; Task 12 `TestSecondControllerWatchesThenAdoptsAStaleJob`.
4. **Setup re-run from the web UI**, which sends no data dir. On a legacy box it must keep the legacy names and `/var/lib/valve-node-app/<chain>`; on a migrated box, the new ones. It must never render units for an account that does not exist, including on a paired box the controller keeps no wire for. Tests: Task 1 `TestSetupKeepsTheBoxNames`; Task 9 `TestSetupAsksAPairedBoxForItsNames`.
5. **A tampered or truncated job log**: an edited body, a dropped tail, a reordered entry, an entry signed by another key, an intent entry not signed by the controller it names. The controller refuses with `job_log_invalid`, and the agent notices a log shorter than its manifest says. Tests: Task 3 `TestVerifyRejectsEditsReordersAndForeignSigners` and `TestVerifyIntentsNeedsTheNamedController`; Task 5 `TestLoadDetectsATruncatedLog`; Task 11 `TestJobDetailRefusesABrokenLog`.

Also pinned, though less likely: another unit (a gateway) running as `valve-node-app` blocks the migration before any change (Task 8 `TestPreflightRefusesOtherUnitsOfTheLegacyUser`). A process that ignores SIGTERM is killed only after the operator chooses so (Task 8 `TestStopUnitsAsksBeforeKillingProcesses`).

## Task dependencies and parallel waves

```
Task 1 names ────────────┬──► Task 2 markers ─────────────┐
                         │                                ▼
Task 4 intents ─┬─► Task 3 joblog ─► Task 5 store ─► Task 7 engine ─► Task 8 migrate-names
                │                                    ▲                       │
Task 6 runners ─┼────────────────────────────────────┘                       ▼
                │                                                    Task 9 agent wiring
                └─► Task 10 api types ─► Task 13 apiclient ─┬─► Task 14 TUI inbox     │
                          │                                 ├─► Task 15 TUI status    ├─► Task 17 e2e
                          │                                 └─► Task 16 CLI           │
                          └──────────► Task 11 server hub (also needs 3, 9) ◄─────────┘
                                              │
                                              ▼
                                       Task 12 driver ─► Task 18 docs and final verification
```

| Wave | Tasks (parallel within a wave) | Each waits for |
|---|---|---|
| 1 | 1, 4, 6 | — |
| 2 | 2, 3, 10 | 2←1; 3←4; 10←4 |
| 3 | 5, 13 | 5←3, 4; 13←10 |
| 4 | 7, 14, 15, 16 | 7←5, 6; 14, 15, 16←13 |
| 5 | 8 | 1, 2, 7 |
| 6 | 9 | 7, 8 |
| 7 | 11, 17 | 11←3, 9, 10; 17←9 |
| 8 | 12 | 11 |
| 9 | 18 | all |

The waves show what may start when. Execution is sequential, one task at a time, in the order 1, 4, 6, 2, 3, 10, 5, 13, 7, 14, 15, 16, 8, 9, 11, 17, 12, 18 (ruling: parallel implementers must not share a branch). Some tasks in one wave touch the same files: Task 15 edits `internal/tui/app.go`, `jobs.go` and `backend.go` after Tasks 13 and 14; Task 4 edits `internal/api/errors.go` (rejection hints) and Task 10 edits it again (codes); Task 5 edits `internal/agent/policy.go` after Task 4. So every commit step names exact paths and never runs `git add -A`, `git add .` or `git add <directory>` (ruling P6).

## File Structure

Created:

| Path | Responsibility |
|---|---|
| `internal/catalog/names.go`, `names_test.go` | `NodeNames`, `LegacyNames`, `CurrentNames`, `NamesByID`, `UnderRoot`, `Rebase`, marker file names |
| `internal/fsatomic/fsatomic.go`, `fsatomic_test.go` | The one atomic file write and directory fsync, moved out of `internal/agent` (ruling P8) |
| `internal/intent/jobs.go`, `jobs_test.go` | Job intent kinds, reasons, states, payloads, `JobRecord`, `JobLogEntryJSON` |
| `internal/joblog/joblog.go`, `bodies.go`, `joblog_test.go` | Signed, hash-chained entries; `Verify`; `VerifyIntents`; body types; `Describe` |
| `internal/jobs/manifest.go`, `store.go`, `lease.go`, `store_test.go`, `lease_test.go` | Manifest, atomic store, log append, truncation check, prune, leases |
| `internal/jobs/runner.go`, `runner_linux.go`, `runner_other.go`, `runner_test.go`, `runner_other_test.go` | `Runner`, `SystemdRunner`, `DefaultRunner`, unit names |
| `internal/jobs/plan.go`, `step.go`, `manager.go`, `manager_test.go`, `step_test.go`, `fakes_test.go` | Plan registry, `RunStep`, `Manager` (start, lease, advance, decide, reconcile, prune) |
| `internal/jobs/migratenames.go`, `marker.go`, `migratenames_test.go`, `scriptexec_test.go` | The `node.migrate-names` plan and the names marker |
| `internal/jobs/jobstest/jobstest.go` | In-process runner and a test plan, for agent and server tests |
| `internal/agent/jobs.go`, `jobs_test.go` | Job intents on the agent, `job.output`, the reconcile loop |
| `internal/api/jobs.go`, `jobs_test.go` | `JobView`, `JobsInbox`, `JobDetail`, `JobQuestion`, `JobOutputLine`, `JobDecision` |
| `internal/server/jobs.go`, `jobs_test.go` | Jobs hub: list, verify, inbox, summary, log cache, read routes |
| `internal/server/jobsdrive.go`, `jobsdrive_test.go` | Driver, lease renewal, startup sweep, migrate result into config, mutation routes |
| `internal/tui/jobs_test.go`, `internal/tui/testdata/jobs_*.golden` | Inbox tests and frames |
| `cmd/jumpgate/cli_jobs.go`, `cli_jobs_test.go` | `jumpgate jobs …`, `jumpgate migrate-names HOST` |
| `internal/bootstrap/e2e_jobs_test.go` | Real systemd: migrate-names across an agent restart |

Modified: `internal/catalog/{units.go,catalog.go}`, `internal/ops/{ops.go,diagnose.go}`, `internal/monitor/monitor.go`, `internal/setup/{steps.go,gateway.go}`, `internal/intent/kinds.go`, `internal/agent/{agent.go,dispatch.go,policy.go,replay.go,logs.go,agent_test.go,replay_test.go,logs_test.go}`, `internal/api/{errors.go,errors_test.go,fleet.go}`, `internal/server/{server.go,api.go,api_test.go,node.go,fleet.go,node_test.go}`, `internal/apiclient/methods.go`, `internal/tui/{backend.go,app.go,jobs.go,fleet.go,palette.go,settings.go}`, `internal/tui/testdata/shell_jobs_{80x24,ascii}.golden`, `internal/tui/tuitest/fake.go`, `cmd/jumpgate/{cli.go,cli_agent.go,help.go,tui_on.go}`, `README.md`.

---

## Phase A: the box (Tasks 1–9)

### Task 1: Both name sets — `catalog.NodeNames`, and every reader takes the box's names

Covers spec "Naming" ("the catalog accepts both names during the transition") and Q2, Q6.

**Files:**
- Create: `internal/catalog/names.go`, `internal/catalog/names_test.go`
- Modify: `internal/catalog/units.go` (add `WireConfig.Names`; `ServiceUser`/`ServiceGroup` become legacy aliases; template prefix), `internal/catalog/catalog.go:194-198` (`DefaultDataDir`), `internal/ops/ops.go:25-45,56-70,105-110,760-761` (unit names from names), `internal/ops/diagnose.go:43,117-150,377-385`, `internal/monitor/monitor.go:36-39,313-315`, `internal/setup/steps.go:26-31,34-37,194-230,470-640,661-700` (names from `st.Wire`), `internal/setup/gateway.go:127` (comment and `catalog.LegacyNames.User`), `internal/agent/dispatch.go:134,175`, `internal/agent/logs.go:42-66`, `internal/server/node.go:301`, `internal/server/api.go:1038-1100` (`handleStartSetup`), and the tests that hard-code `valve-node-app-exec.service` (find them with `grep -rln 'valve-node-app-' --include='*_test.go' internal`).
- Test: `internal/catalog/names_test.go`, `internal/server/api_test.go` (new test), plus the existing `internal/catalog`, `internal/ops`, `internal/setup`, `internal/monitor`, `internal/agent` suites.

**Interfaces:**
- Produces (package `catalog`): `const NamesLegacy = ""`, `const NamesCurrent = "jumpgate-node"`; `type NodeNames struct { ID, Prefix, User, Group, DataRoot string }`; `var LegacyNames, CurrentNames NodeNames`; `func (NodeNames) ExecUnit() string`, `BeaconUnit() string`, `Units() []string`; `func UnitPath(unit string) string`; `func NamesByID(id string) (NodeNames, error)`; `func (WireConfig) NodeNames() NodeNames`; `func ValidateNames(w WireConfig) error`; `func DefaultDataDirFor(n NodeNames, chainID int) string`; `func UnderRoot(dir, root string) bool`; `func Rebase(dir, from, to string) (string, bool)`; new field `WireConfig.Names string`.
- Produces (package `ops`): `func ServiceAction(ctx context.Context, e executor.Executor, n catalog.NodeNames, svc, action string) (bool, error)`; `func NodeUnits(n catalog.NodeNames) []string`.
- `catalog.DefaultDataDir(chainID)` now returns `/var/lib/jumpgate-node/<chainID>`.

- [ ] **Step 1: Write the failing catalog tests**

```go
// internal/catalog/names_test.go
package catalog

import "testing"

func TestNamesByIDKnowsBothSetsAndNothingElse(t *testing.T) {
	for id, want := range map[string]NodeNames{NamesLegacy: LegacyNames, NamesCurrent: CurrentNames} {
		got, err := NamesByID(id)
		if err != nil || got != want {
			t.Errorf("NamesByID(%q) = %+v, %v", id, got, err)
		}
	}
	if _, err := NamesByID("valve-node-app"); err == nil {
		t.Error("the legacy prefix is stored as \"\", never by name; anything else must be refused")
	}
}

func TestUnitNamesFollowThePrefix(t *testing.T) {
	if LegacyNames.ExecUnit() != "valve-node-app-exec.service" || LegacyNames.BeaconUnit() != "valve-node-app-beacon.service" {
		t.Fatalf("legacy units %v", LegacyNames.Units())
	}
	if CurrentNames.ExecUnit() != "jumpgate-node-exec.service" || CurrentNames.User != "jumpgate-node" || CurrentNames.DataRoot != "/var/lib/jumpgate-node" {
		t.Fatalf("current names %+v", CurrentNames)
	}
	if UnitPath("x.service") != "/etc/systemd/system/x.service" {
		t.Fatal(UnitPath("x.service"))
	}
}

func TestDefaultDataDirIsUnderTheNewRoot(t *testing.T) {
	if got := DefaultDataDir(369); got != "/var/lib/jumpgate-node/369" {
		t.Fatalf("DefaultDataDir = %s", got)
	}
	if got := DefaultDataDirFor(LegacyNames, 369); got != "/var/lib/valve-node-app/369" {
		t.Fatalf("legacy DefaultDataDirFor = %s", got)
	}
}

func TestRebaseOnlyMovesPathsUnderTheRoot(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"/var/lib/valve-node-app/369", "/var/lib/jumpgate-node/369", true},
		{"/var/lib/valve-node-app", "/var/lib/jumpgate-node", true},
		{"/var/lib/valve-node-app-other/1", "/var/lib/valve-node-app-other/1", false}, // a sibling, not a child
		{"/data/node", "/data/node", false},
	} {
		got, ok := Rebase(c.in, LegacyNames.DataRoot, CurrentNames.DataRoot)
		if got != c.want || ok != c.ok {
			t.Errorf("Rebase(%s) = %s, %v", c.in, got, ok)
		}
	}
}

// Legacy boxes must render byte-identical units, or setup's wire Verify would
// see a "changed" config and restart every existing node on its next run.
func TestLegacyUnitsRenderAsBefore(t *testing.T) {
	w := WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node-app/369"}
	execUnit, _, err := RenderUnits(w)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Description=valve-node-app ", "User=valve-node-app\n", "Group=valve-node-app\n"} {
		if !strings.Contains(execUnit, want) {
			t.Errorf("legacy unit lacks %q:\n%s", want, execUnit)
		}
	}
	w.Names, w.DataDir = NamesCurrent, "/var/lib/jumpgate-node/369"
	execUnit, _, _ = RenderUnits(w)
	for _, want := range []string{"Description=jumpgate-node ", "User=jumpgate-node\n", "ReadWritePaths=/var/lib/jumpgate-node/369\n"} {
		if !strings.Contains(execUnit, want) {
			t.Errorf("current unit lacks %q:\n%s", want, execUnit)
		}
	}
}
```

Add `"strings"` to the imports. Use a valid exec/beacon pair for chain 369 from `internal/catalog/networks.go`; `lighthouse-pulse` is what `internal/server/node_test.go` uses.

- [ ] **Step 2: Run the tests to make sure they fail**

Run: `go test ./internal/catalog/ -run 'Names|UnitNames|DefaultDataDir|Rebase|LegacyUnits' -count=1`
Expected: FAIL to compile with "undefined: NamesByID", "undefined: LegacyNames", and so on.

- [ ] **Step 3: Implement `names.go` and thread names through the catalog**

```go
// internal/catalog/names.go
package catalog

import (
	"fmt"
	"path"
	"strings"
)

// Names ids, as WireConfig.Names stores them. The legacy id is the empty
// string so every wire saved before sub-project 2 reads as legacy unchanged.
const (
	NamesLegacy  = ""
	NamesCurrent = "jumpgate-node"
)

// NodeNames are a node's on-box names: its service account, its two units and
// the default root of its data. A box uses one set or the other, never a mix:
// node.migrate-names moves a box from LegacyNames to CurrentNames in one job.
type NodeNames struct {
	ID       string // NamesLegacy or NamesCurrent
	Prefix   string // unit names and unit descriptions
	User     string
	Group    string
	DataRoot string
}

var (
	LegacyNames  = NodeNames{ID: NamesLegacy, Prefix: "valve-node-app", User: "valve-node-app", Group: "valve-node-app", DataRoot: "/var/lib/valve-node-app"}
	CurrentNames = NodeNames{ID: NamesCurrent, Prefix: "jumpgate-node", User: "jumpgate-node", Group: "jumpgate-node", DataRoot: "/var/lib/jumpgate-node"}
)

func (n NodeNames) ExecUnit() string   { return n.Prefix + "-exec.service" }
func (n NodeNames) BeaconUnit() string { return n.Prefix + "-beacon.service" }
func (n NodeNames) Units() []string    { return []string{n.ExecUnit(), n.BeaconUnit()} }

// UnitPath is where setup writes a node unit.
func UnitPath(unit string) string { return "/etc/systemd/system/" + unit }

// NamesByID resolves a stored id. An unknown id is an error, never a guess:
// acting on the wrong account would stop or chown the wrong things.
func NamesByID(id string) (NodeNames, error) {
	switch id {
	case NamesLegacy:
		return LegacyNames, nil
	case NamesCurrent:
		return CurrentNames, nil
	}
	return NodeNames{}, fmt.Errorf("catalog: unknown node names %q (want %q or %q)", id, NamesLegacy, NamesCurrent)
}

// NodeNames is w's names. An unknown id reads as legacy here so a renderer
// never panics; ValidateNames is where it is refused (setup.Plan, the agent's
// node.json load).
func (w WireConfig) NodeNames() NodeNames {
	n, err := NamesByID(w.Names)
	if err != nil {
		return LegacyNames
	}
	return n
}

// ValidateNames refuses a wire whose names id is unknown.
func ValidateNames(w WireConfig) error {
	_, err := NamesByID(w.Names)
	return err
}

// DefaultDataDirFor is where a chain's data lives by default under a name set.
func DefaultDataDirFor(n NodeNames, chainID int) string {
	return fmt.Sprintf("%s/%d", n.DataRoot, chainID)
}

// UnderRoot reports whether dir is root itself or inside it. Paths on a box
// are POSIX, so this uses package path, never filepath.
func UnderRoot(dir, root string) bool {
	dir = path.Clean(dir)
	return dir == root || strings.HasPrefix(dir, root+"/")
}

// Rebase moves dir from one root to another. ok is false, and dir comes back
// unchanged, when dir is not under from.
func Rebase(dir, from, to string) (string, bool) {
	if !UnderRoot(dir, from) {
		return dir, false
	}
	return to + strings.TrimPrefix(path.Clean(dir), from), true
}
```

In `internal/catalog/units.go`:
- Add to `WireConfig`, after `JWTPath`:
  ```go
  	// Names is the box's name set: "" (legacy, every box set up before
  	// sub-project 2) or NamesCurrent. Resolved via NodeNames(), never read
  	// directly; node.migrate-names moves a box from one to the other.
  	Names string
  ```
- Replace the `ServiceUser`/`ServiceGroup` block with:
  ```go
  // ServiceUser/ServiceGroup are the legacy account, kept for the gateway's
  // systemd backend (internal/setup/gateway.go), which sub-project 6 moves onto
  // agents and renames. Node code uses WireConfig.NodeNames().
  const (
  	ServiceUser  = "valve-node-app"
  	ServiceGroup = "valve-node-app"
  )
  ```
  Add a test assertion in `names_test.go` that `ServiceUser == LegacyNames.User`.
- In `unitTemplate`, change `Description=valve-node-app {{.Description}}` to `Description={{.Prefix}} {{.Description}}`. Add `Prefix string` to `unitVars`. In `RenderUnits`, set `Prefix: w.NodeNames().Prefix, User: w.NodeNames().User, Group: w.NodeNames().Group`. Start `RenderUnits` with `if err := ValidateNames(w); err != nil { return "", "", err }`.
- Fix the stale comment above `RenderUnits` that says naming the units is "Task 4's job": say the names come from `w.NodeNames()`.

In `internal/catalog/catalog.go`, make `DefaultDataDir` return `DefaultDataDirFor(CurrentNames, chainID)`, and replace its comment: "the default for a new setup; an existing box keeps the data dir its wire records".

- [ ] **Step 4: Run the catalog tests**

Run: `go test ./internal/catalog/ -count=1`
Expected: PASS. If an existing test asserted `DefaultDataDir` returns the legacy path, change it to the new path. That is the intended change (Q6).

- [ ] **Step 5: Thread names through ops, monitor, setup and the agent**

`internal/ops/ops.go`: delete the `execUnitName`/`beaconUnitName` constants. Change `unitFor(svc string)` to `unitFor(n catalog.NodeNames, svc string) (string, bool)`, returning `n.ExecUnit()` or `n.BeaconUnit()`. Change `ServiceAction` to take `n catalog.NodeNames` after `e`. `ClearService` already has `w`: call `unitFor(w.NodeNames(), svc)`. Change `NodeUnits` to `func NodeUnits(n catalog.NodeNames) []string { return n.Units() }`.

`internal/ops/diagnose.go`: `unitActiveStates(ctx, e, n catalog.NodeNames)` and `journalItem(ctx, e, n catalog.NodeNames)`. `NetworkDiagnostics` passes `w.NodeNames()`. `servicesItem` keeps its boolean parameters, but its fix strings and the "both … are active" detail use `n.ExecUnit()`/`n.BeaconUnit()`; add an `n catalog.NodeNames` parameter for that.

`internal/monitor/monitor.go`: delete the unit constants. `activeCmd()` becomes `activeCmd(n catalog.NodeNames)`, and its caller passes `m.cfg.Wire.NodeNames()`.

`internal/setup/steps.go`: delete the `execUnitName`/`beaconUnitName`/`execUnitPath`/`beaconUnitPath` constants. In each step, read `n := st.Wire.NodeNames()` and use `n.ExecUnit()`, `catalog.UnitPath(n.ExecUnit())`, `n.User`, `n.Group`, and `n.DataRoot` (the useradd home dir in `accountStep`, which today hard-codes `/var/lib/valve-node-app`). `Plan` starts with `if err := catalog.ValidateNames(w); err != nil { return nil, fmt.Errorf("setup: %w", err) }`. The preflight's `ownActiveUnitPorts` gets `n` the same way.

`internal/setup/gateway.go:127`: the comment says the unit runs as `catalog.ServiceUser`; keep the constant and add "the legacy account; node.migrate-names refuses a box where this unit exists (sub-project 6 renames it)".

`internal/agent/dispatch.go`: `ops.NodeUnits()` becomes `ops.NodeUnits(w.NodeNames())`, and `ops.ServiceAction(ctx, ex, pl.Service, pl.Action)` becomes `ops.ServiceAction(ctx, ex, w.NodeNames(), pl.Service, pl.Action)`. In `loadNode`, after `ValidateDataDir`, add `if err := catalog.ValidateNames(w); err != nil { return w, reject(intent.ReasonValidation, err.Error()) }`.

`internal/agent/logs.go`: `logsSinceCommand(cursor string, n int)` becomes `logsSinceCommand(units []string, cursor string, n int)`, and `logsSince` takes `w catalog.WireConfig` and passes `ops.NodeUnits(w.NodeNames())`. In `dispatch`, the `KindLogsSince` case passes `w`.

`internal/server/node.go:301`: `ops.ServiceAction(ctx, ex, t.Wire.NodeNames(), svc, action)`. `legacyExec` guarantees `t.Wire != nil`.

Run: `go build ./... && go vet ./...`
Expected: success. Fix every compile error the signature changes raise; the compiler lists each call site.

- [ ] **Step 6: Write the failing setup-names test (Review Focus 4)**

```go
// internal/server/api_test.go — add
// Review Focus 4: the web UI sends no data dir. A legacy box keeps its names
// and data dir, a migrated box keeps the new ones, and only a box set up for
// the first time gets the new defaults.
func TestSetupKeepsTheBoxNames(t *testing.T) {
	for _, c := range []struct {
		name     string
		existing *catalog.WireConfig
		wantIDs  string
		wantDir  string
	}{
		{"first setup", nil, catalog.NamesCurrent, "/var/lib/jumpgate-node/369"},
		{"legacy box", &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node-app/369"}, catalog.NamesLegacy, "/var/lib/valve-node-app/369"},
		{"migrated box", &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", Names: catalog.NamesCurrent, DataDir: "/var/lib/jumpgate-node/369"}, catalog.NamesCurrent, "/var/lib/jumpgate-node/369"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := setupWire(catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"}, c.existing)
			if got.Names != c.wantIDs || got.DataDir != c.wantDir || got.JWTPath != c.wantDir+"/jwt.hex" {
				t.Fatalf("setupWire = names %q dir %s jwt %s", got.Names, got.DataDir, got.JWTPath)
			}
		})
	}
	// A request may not switch a box's names: only node.migrate-names does.
	got := setupWire(catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", Names: catalog.NamesCurrent},
		&catalog.WireConfig{ChainID: 369, DataDir: "/var/lib/valve-node-app/369"})
	if got.Names != catalog.NamesLegacy {
		t.Fatalf("a request renamed a legacy box: %q", got.Names)
	}
}
```

- [ ] **Step 7: Run it to make sure it fails**

Run: `go test ./internal/server/ -run TestSetupKeepsTheBoxNames -count=1`
Expected: FAIL to compile with "undefined: setupWire".

- [ ] **Step 8: Implement `setupWire` and use it in `handleStartSetup`**

```go
// internal/server/api.go — add near handleStartSetup
// setupWire fills a setup request's defaults from the box's current wire.
// Names never come from the request: a box keeps the names it has (legacy for
// every box set up before sub-project 2) until node.migrate-names changes
// them, and only a box with no wire yet starts with the new names. A data dir
// the request leaves empty is the box's own when the chain is unchanged, so a
// re-run from the web UI (which sends none) never points a node at an empty
// directory.
func setupWire(req catalog.WireConfig, existing *catalog.WireConfig) catalog.WireConfig {
	w := req
	switch {
	case existing == nil:
		w.Names = catalog.NamesCurrent
	default:
		w.Names = existing.Names
		if w.DataDir == "" && existing.ChainID == w.ChainID && existing.DataDir != "" {
			w.DataDir, w.JWTPath = existing.DataDir, existing.JWTPath
		}
	}
	if w.DataDir == "" {
		w.DataDir = catalog.DefaultDataDirFor(w.NodeNames(), w.ChainID)
	}
	if w.JWTPath == "" {
		w.JWTPath = catalog.DefaultJWTPath(w.DataDir)
	}
	return w
}
```

In `handleStartSetup`, move the `loadConfig`/`findTarget` lookup above the data-dir defaults. Then replace the two `if wire.DataDir == ""`/`if wire.JWTPath == ""` blocks with `wire = setupWire(wire, target.Wire)`. Keep `catalog.ValidateDataDir(wire.DataDir)` and `setup.Plan(wire)` after that call.

- [ ] **Step 9: Run every affected package**

Run: `go test ./internal/catalog/ ./internal/ops/ ./internal/monitor/ ./internal/setup/ ./internal/agent/ ./internal/server/ -count=1`
Expected: PASS. A test that asserted a literal `valve-node-app-exec.service` keeps passing, because a wire with no `Names` is legacy. A test that built a *new* setup and expected the legacy data dir now expects `/var/lib/jumpgate-node/<chain>`; update it.

- [ ] **Step 10: Commit**

```bash
git add internal/catalog/names.go internal/catalog/names_test.go internal/catalog/units.go internal/catalog/catalog.go \
  internal/ops/ops.go internal/ops/diagnose.go internal/monitor/monitor.go \
  internal/setup/steps.go internal/setup/gateway.go \
  internal/agent/dispatch.go internal/agent/logs.go \
  internal/server/node.go internal/server/api.go internal/server/api_test.go
# Then each existing test Steps 4, 5 and 9 changed, one path per argument as
# `git status --short` lists them (for example internal/catalog/catalog_test.go).
# Never `git add -A`, `.` or a directory: another task's work may be in the tree.
git status --short
git commit -m "feat(catalog): accept both node name sets; new setups get jumpgate-node

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 2: Completion markers where `Verify` is too loose (snapshot, recursive chown)

Covers spec "Durable jobs": "This requires completion markers where `Verify` is too loose today (snapshot, recursive chown)."

**Files:**
- Modify: `internal/catalog/names.go` (two marker names), `internal/setup/steps.go` (`snapshotStep` Run and Verify; `wireStep` Run and Verify)
- Test: `internal/setup/snapshot_test.go`, `internal/setup/wire_test.go`, `internal/setup/fake_executor_test.go` (adds the `indexContaining` helper)

**Interfaces:**
- Consumes: Task 1's `st.Wire.NodeNames()`.
- Produces (package `catalog`): `const SnapshotMarker = ".jumpgate-snapshot-complete"`, `const OwnerMarker = ".jumpgate-owner"`. Both are files directly in `WireConfig.DataDir`. Task 8 rewrites `OwnerMarker`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/setup/snapshot_test.go — add
// A half-unpacked snapshot has a db/ directory too: only the marker written
// after a successful download proves the restore finished.
func TestSnapshot_VerifyNeedsTheCompletionMarker(t *testing.T) {
	w := snapshotWire() // ExecSnapshot set
	f := newFakeExecutor().
		script("test -f '"+w.DataDir+"/"+catalog.SnapshotMarker+"'", executor.Result{ExitCode: 1}).
		script("test -d '"+w.DataDir+"/db'", executor.Result{ExitCode: 0}).
		script("test -f '"+catalog.UnitPath(w.NodeNames().ExecUnit())+"'", executor.Result{ExitCode: 1})
	if err := snapshotStep().Verify(context.Background(), f, &State{Wire: w}); err == nil {
		t.Fatal("a db/ directory without the marker or a finished setup passed as restored")
	}
}

// A box set up before markers existed has db/ and its units: Verify adopts it
// (writes the marker) instead of downloading the snapshot again over live data.
func TestSnapshot_VerifyAdoptsAFinishedPreMarkerInstall(t *testing.T) {
	w := snapshotWire()
	f := newFakeExecutor().
		script("test -f '"+w.DataDir+"/"+catalog.SnapshotMarker+"'", executor.Result{ExitCode: 1}).
		script("test -d '"+w.DataDir+"/db'", executor.Result{ExitCode: 0}).
		script("test -f '"+catalog.UnitPath(w.NodeNames().ExecUnit())+"'", executor.Result{ExitCode: 0})
	if err := snapshotStep().Verify(context.Background(), f, &State{Wire: w}); err != nil {
		t.Fatalf("a finished pre-marker install failed Verify: %v", err)
	}
	if indexContaining(f, "> '"+w.DataDir+"/"+catalog.SnapshotMarker+"'") < 0 {
		t.Fatalf("the marker was not written; calls %v", f.callLog())
	}
}

func TestSnapshot_RunWritesTheMarkerOnlyAfterTheDownload(t *testing.T) {
	w := snapshotWire()
	f := resolvable(newFakeExecutor()) // the network calls succeed, as in TestSnapshot_CreatesTheDataDirThenDownloadsIntoIt
	if err := snapshotStep().Run(context.Background(), f, &State{Wire: w}); err != nil {
		t.Fatal(err)
	}
	dl, mk := indexContaining(f, "reth download"), indexContaining(f, catalog.SnapshotMarker)
	if dl < 0 || mk < 0 || mk < dl {
		t.Fatalf("marker at %d, download at %d: %v", mk, dl, f.callLog())
	}
}
```

```go
// internal/setup/wire_test.go — add
// The top directory's owner says nothing about the files under it: an
// interrupted chown -R leaves it right and the rest root-owned.
func TestWire_VerifyNeedsTheOwnerMarkerOrAFullWalk(t *testing.T) {
	w := testWire()
	n := w.NodeNames()
	f := wiredExecutor(). // the JWT and unit files are present
		script("stat -c %U '"+w.DataDir+"'", executor.Result{Stdout: n.User + "\n"}).
		script("cat '"+w.DataDir+"/"+catalog.OwnerMarker+"'", executor.Result{ExitCode: 1}).
		script("find '"+w.DataDir+"' ! -user "+n.User, executor.Result{Stdout: w.DataDir + "/db/mdbx.dat\n"})
	if err := wireStep().Verify(context.Background(), f, &State{Wire: w}); err == nil {
		t.Fatal("a tree with root-owned files passed as wired")
	}
}

func TestWire_RunWritesTheOwnerMarkerAfterTheChown(t *testing.T) {
	w := testWire()
	f := newFakeExecutor()
	if err := wireStep().Run(context.Background(), f, &State{Wire: w}); err != nil {
		t.Fatal(err)
	}
	ch, mk := indexContaining(f, "chown -R"), indexContaining(f, catalog.OwnerMarker)
	if ch < 0 || mk < ch {
		t.Fatalf("marker at %d, chown at %d: %v", mk, ch, f.callLog())
	}
}
```

`snapshotWire`, `testWire`, `resolvable`, `wiredExecutor` and `callLog` already exist in `internal/setup`'s tests. Add one helper to `fake_executor_test.go`:

```go
// indexContaining is the position of the first command containing s, or -1.
func indexContaining(f *fakeExecutor, s string) int {
	for i, c := range f.callLog() {
		if strings.Contains(c, s) {
			return i
		}
	}
	return -1
}
```

`wiredExecutor` answers every `test -f` with success, and the marker test's longer scripts win over it because `fakeExecutor` matches the longest substring first. In `wireStep().Verify`, put the marker check right after the top-level `stat -c %U` check and before the unit-content comparison. The test then needs no unit files.

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/setup/ -run 'Snapshot_Verify|Snapshot_RunWrites|Wire_VerifyNeeds|Wire_RunWrites' -count=1`
Expected: FAIL (undefined `catalog.SnapshotMarker`, then the assertions).

- [ ] **Step 3: Implement**

Add to `internal/catalog/names.go`:

```go
// Completion markers live directly in a node's data dir. Setup writes each one
// only after the work it vouches for finished, so a rerun (and adopting a
// durable job, which is a rerun) can trust it where a looser probe cannot.
const (
	SnapshotMarker = ".jumpgate-snapshot-complete" // the snapshot restore finished
	OwnerMarker    = ".jumpgate-owner"             // chown -R finished; holds the owning account's name
)
```

In `snapshotStep`:
- At the end of `Run`, after the download succeeded: `e.Run(ctx, "date -u +%FT%TZ > "+shQuote(path.Join(w.DataDir, catalog.SnapshotMarker)), opts)`, and check its exit code like the other commands.
- `Verify` (when `ExecSnapshot`): `test -f <marker>` passes. Otherwise, when `test -d <db>` and `test -f <exec unit path>` both pass, write the marker and pass (a finished install from before markers existed). Otherwise fail with "the snapshot restore has not finished (no completion marker)".

In `wireStep`:
- `Run`, right after the `chown -R` succeeds: `printf '%s\n' <user> > <DataDir>/.jumpgate-owner && chown <user>:<group> <marker>`.
- `Verify`, after the top-level `stat -c %U` check: `cat <marker>` whose trimmed stdout equals `n.User` passes. Otherwise, `find <DataDir> ! -user <user> -print -quit` with empty stdout and exit 0 writes the marker and passes: the tree really is owned, it is just from before markers. Anything else fails with "files under <DataDir> are not owned by <user> yet (for example <first path>)".

Comment the find fallback: "a full walk, read-only and with no downtime, run once per pre-marker box; the marker makes every later Verify one `cat`."

- [ ] **Step 4: Run the setup suite**

Run: `go test ./internal/setup/ -count=1`
Expected: PASS. Existing wire Verify tests that script only the `stat` check now also need the marker `cat` scripted to succeed. Update them so they script `cat …/.jumpgate-owner` returning the user.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/names.go internal/setup/steps.go internal/setup/snapshot_test.go internal/setup/wire_test.go internal/setup/fake_executor_test.go
# Plus each existing setup test Step 4 updated to script the marker `cat`, by path.
git status --short
git commit -m "feat(setup): completion markers for the snapshot restore and the data-dir chown

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 3: The signed job log — `internal/joblog`

Covers spec "The agent keeps a signed, append-only log of every intent and step outcome per job, so an adopting controller can verify history instead of trusting it", plus J7 and Review Focus 5.

**Files:**
- Create: `internal/joblog/joblog.go`, `internal/joblog/bodies.go`, `internal/joblog/joblog_test.go`

**Interfaces:**
- Consumes: Task 4's `intent.JobLogEntryJSON`, `intent.Envelope`, `intent.JobQuestion`; `eip712.Digest`, `eip712.Keccak256`; `signer.Signer`, `signer.Recover`, `signer.ParseSignature`.
- Produces (package `joblog`): `var Types eip712.Types`; `type Entry struct { Index uint64; Prev [32]byte; Kind string; Body []byte; At uint64; Sig signer.Signature }`; `func (Entry) TypedData(agent eip712.Address, job string) eip712.TypedData`; `func (Entry) Digest(agent eip712.Address, job string) ([32]byte, error)`; `func Sign(ctx context.Context, s signer.Signer, job string, index uint64, prev [32]byte, kind string, body []byte, at time.Time) (Entry, [32]byte, error)`; `var ErrBroken error`; `func Verify(agent eip712.Address, job string, entries []Entry, start uint64, prev [32]byte) ([32]byte, error)`; `func VerifyIntents(entries []Entry) error`; `func ToJSON(Entry) intent.JobLogEntryJSON`; `func FromJSON(intent.JobLogEntryJSON) (Entry, error)`; `func Describe(Entry) string`; kind constants `KindCreated, KindIntent, KindStepStarted, KindStepFinished, KindStepLost, KindDecisionAsked, KindDecisionAnswered, KindLeaseTaken, KindLeaseReleased, KindFinished`; bodies `IntentBody{Envelope intent.Envelope; Outcome string}`, `StepBody{Step string; Attempt int; Unit string; OK bool; Error string}`, `AnswerBody{Question, Option, Controller string; Signers []string}`, `LeaseBody{Owner, Previous string}`, `FinishedBody{State, Error string}`. A question body is `intent.JobQuestion`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/joblog/joblog_test.go
package joblog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

const job = "0123456789abcdef"

func TestSchemaIsFrozen(t *testing.T) {
	got, err := eip712.EncodeType(Types, "JobLogEntry")
	want := "JobLogEntry(address agent,string job,uint64 index,bytes32 prev,string kind,bytes32 bodyHash,uint64 at)"
	if err != nil || got != want {
		t.Fatalf("EncodeType = %q, %v; want %q", got, err, want)
	}
}

// chain signs n entries the way the agent appends them.
func chain(t *testing.T, k *signer.Key, n int) []Entry {
	t.Helper()
	var out []Entry
	var prev [32]byte
	for i := 0; i < n; i++ {
		e, d, err := Sign(context.Background(), k, job, uint64(i), prev, KindStepFinished,
			[]byte(`{"step":"s`+string(rune('a'+i))+`","attempt":0,"ok":true}`), time.Unix(1_800_000_000+int64(i), 0))
		if err != nil {
			t.Fatal(err)
		}
		out, prev = append(out, e), d
	}
	return out
}

func TestVerifyAcceptsAnUnbrokenChainAndResumes(t *testing.T) {
	k, _ := signer.GenerateKey()
	es := chain(t, k, 4)
	head, err := Verify(k.Address(), job, es, 0, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	// A controller that already verified the first two verifies only the rest.
	mid, _ := Verify(k.Address(), job, es[:2], 0, [32]byte{})
	again, err := Verify(k.Address(), job, es[2:], 2, mid)
	if err != nil || again != head {
		t.Fatalf("resumed verify: %v, head %x vs %x", err, again, head)
	}
}

// Review Focus 5.
func TestVerifyRejectsEditsReordersAndForeignSigners(t *testing.T) {
	k, _ := signer.GenerateKey()
	other, _ := signer.GenerateKey()
	for name, mutate := range map[string]func([]Entry) []Entry{
		"edited body":    func(es []Entry) []Entry { es[1].Body = []byte(`{"ok":false}`); return es },
		"edited kind":    func(es []Entry) []Entry { es[2].Kind = KindStepLost; return es },
		"dropped middle": func(es []Entry) []Entry { return append(es[:1], es[2:]...) },
		"reordered":      func(es []Entry) []Entry { es[1], es[2] = es[2], es[1]; return es },
		"foreign signer": func(es []Entry) []Entry {
			e, _, _ := Sign(context.Background(), other, job, 3, es[3].Prev, es[3].Kind, es[3].Body, time.Unix(int64(es[3].At), 0))
			es[3] = e
			return es
		},
		"other job": func(es []Entry) []Entry {
			e, _, _ := Sign(context.Background(), k, "fedcba9876543210", 3, es[3].Prev, es[3].Kind, es[3].Body, time.Unix(int64(es[3].At), 0))
			es[3] = e
			return es
		},
	} {
		es := mutate(chain(t, k, 4))
		if _, err := Verify(k.Address(), job, es, 0, [32]byte{}); !errors.Is(err, ErrBroken) {
			t.Errorf("%s: Verify = %v, want ErrBroken", name, err)
		}
	}
}

func TestJSONRoundTripKeepsTheSignature(t *testing.T) {
	k, _ := signer.GenerateKey()
	es := chain(t, k, 2)
	b, _ := json.Marshal(ToJSON(es[1]))
	var j intent.JobLogEntryJSON
	if err := json.Unmarshal(b, &j); err != nil {
		t.Fatal(err)
	}
	back, err := FromJSON(j)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(k.Address(), job, []Entry{es[0], back}, 0, [32]byte{}); err != nil {
		t.Fatalf("a round-tripped entry no longer verifies: %v", err)
	}
}

func signedEnvelope(t *testing.T, controller, signs *signer.Key) intent.Envelope {
	t.Helper()
	body := []byte(`{"id":"` + job + `"}`)
	n, _ := intent.NewNonce()
	in := intent.Intent{Controller: controller.Address(), Seq: 1, Nonce: n, IssuedAt: 1, Expiry: 2, Kind: intent.KindJobAdvance, PayloadHash: intent.Hash(body)}
	sig, _ := signs.SignTypedData(context.Background(), in.TypedData())
	return intent.Envelope{Intent: in.JSON(), Payload: body, Sigs: []string{sig.Hex()}}
}

// Review Focus 5: an intent entry proves who asked only if the controller it
// names signed it, over exactly the payload it carries.
func TestVerifyIntentsNeedsTheNamedController(t *testing.T) {
	agentKey, _ := signer.GenerateKey()
	ctl, _ := signer.GenerateKey()
	mallory, _ := signer.GenerateKey()
	good := signedEnvelope(t, ctl, ctl)
	forged := signedEnvelope(t, ctl, mallory)
	swapped := good
	swapped.Payload = []byte(`{"id":"ffffffffffffffff"}`)
	for name, c := range map[string]struct {
		env intent.Envelope
		ok  bool
	}{"good": {good, true}, "signed by someone else": {forged, false}, "payload swapped": {swapped, false}} {
		body, _ := json.Marshal(IntentBody{Envelope: c.env, Outcome: "advance"})
		e, _, _ := Sign(context.Background(), agentKey, job, 0, [32]byte{}, KindIntent, body, time.Unix(1, 0))
		if err := VerifyIntents([]Entry{e}); (err == nil) != c.ok {
			t.Errorf("%s: VerifyIntents = %v", name, err)
		}
	}
}

func TestDescribeIsOneReadableLine(t *testing.T) {
	q, _ := json.Marshal(intent.JobQuestion{ID: "stop-units-0-busy-user", Prompt: "2 processes still run"})
	for kind, body := range map[string][]byte{
		KindStepStarted:   []byte(`{"step":"stop-units","attempt":0,"unit":"jumpgate-job-x-stop-units-0"}`),
		KindDecisionAsked: q,
		KindFinished:      []byte(`{"state":"succeeded"}`),
	} {
		got := Describe(Entry{Kind: kind, Body: body})
		if got == "" || len(got) > 200 {
			t.Errorf("Describe(%s) = %q", kind, got)
		}
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/joblog/ -count=1`
Expected: FAIL to compile (package has no non-test files).

- [ ] **Step 3: Implement**

```go
// internal/joblog/joblog.go

// Package joblog is a durable job's history: an append-only list of entries,
// each signed by the agent over EIP-712 and chained to the one before by its
// digest. The agent writes it; any controller verifies it, so a controller
// that adopts a job reads a history it can check rather than one it must
// trust. The package is pure: no files, no clocks of its own.
package joblog

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// Types is the frozen schema. See TestSchemaIsFrozen.
var Types = eip712.Types{
	"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}},
	"JobLogEntry": {
		{Name: "agent", Type: "address"}, {Name: "job", Type: "string"}, {Name: "index", Type: "uint64"},
		{Name: "prev", Type: "bytes32"}, {Name: "kind", Type: "string"}, {Name: "bodyHash", Type: "bytes32"},
		{Name: "at", Type: "uint64"},
	},
}

// Entry is one signed entry. Body is the exact bytes hashed and signed.
type Entry struct {
	Index uint64
	Prev  [32]byte // the previous entry's digest; zero for entry 0
	Kind  string
	Body  []byte
	At    uint64 // unix seconds, the agent's clock
	Sig   signer.Signature
}

// ErrBroken is every way a log can fail verification.
var ErrBroken = errors.New("joblog: the job log does not verify")

// TypedData binds an entry to its agent and job, so an entry cannot be moved
// to another job's log or claimed by another agent.
func (e Entry) TypedData(agent eip712.Address, job string) eip712.TypedData {
	return eip712.TypedData{Types: Types, PrimaryType: "JobLogEntry",
		Domain: map[string]any{"name": "jumpgate", "version": "1"},
		Message: map[string]any{"agent": agent, "job": job, "index": e.Index, "prev": e.Prev,
			"kind": e.Kind, "bodyHash": eip712.Keccak256(e.Body), "at": e.At}}
}

// Digest is the entry's EIP-712 digest: what the agent signs, and what the
// next entry carries as Prev.
func (e Entry) Digest(agent eip712.Address, job string) ([32]byte, error) {
	return eip712.Digest(e.TypedData(agent, job))
}

// Sign makes entry index of job, chained to prev, and returns it with its digest.
func Sign(ctx context.Context, s signer.Signer, job string, index uint64, prev [32]byte, kind string, body []byte, at time.Time) (Entry, [32]byte, error) {
	e := Entry{Index: index, Prev: prev, Kind: kind, Body: body, At: uint64(max(at.Unix(), 0))}
	sig, err := s.SignTypedData(ctx, e.TypedData(s.Address(), job))
	if err != nil {
		return Entry{}, [32]byte{}, fmt.Errorf("joblog: sign: %w", err)
	}
	e.Sig = sig
	d, err := e.Digest(s.Address(), job)
	return e, d, err
}

// Verify checks entries as positions start, start+1, … of job's log, the
// first chained to prev, every one signed by agent. It returns the last
// digest, which a caller keeps to verify the next page without the first.
func Verify(agent eip712.Address, job string, entries []Entry, start uint64, prev [32]byte) ([32]byte, error) {
	for k, e := range entries {
		want := start + uint64(k)
		if e.Index != want {
			return prev, fmt.Errorf("%w: position %d holds entry %d", ErrBroken, want, e.Index)
		}
		if e.Prev != prev {
			return prev, fmt.Errorf("%w: entry %d does not follow entry %d", ErrBroken, want, int64(want)-1)
		}
		who, err := signer.Recover(e.TypedData(agent, job), e.Sig)
		if err != nil || who != agent {
			return prev, fmt.Errorf("%w: entry %d is not signed by agent %s", ErrBroken, want, agent.Hex())
		}
		d, err := e.Digest(agent, job)
		if err != nil {
			return prev, fmt.Errorf("%w: entry %d: %v", ErrBroken, want, err)
		}
		prev = d
	}
	return prev, nil
}

// VerifyIntents checks every entry that records a request: the controller the
// intent names signed it, over exactly the payload recorded. The agent
// checked this when it admitted the intent; a controller checks it again
// because the log is how it learns who asked for what.
func VerifyIntents(entries []Entry) error {
	for _, e := range entries {
		if e.Kind != KindCreated && e.Kind != KindIntent {
			continue
		}
		var b IntentBody
		if err := json.Unmarshal(e.Body, &b); err != nil {
			return fmt.Errorf("%w: entry %d: %v", ErrBroken, e.Index, err)
		}
		in, err := b.Envelope.Intent.Parse()
		if err != nil {
			return fmt.Errorf("%w: entry %d: %v", ErrBroken, e.Index, err)
		}
		if intent.Hash(b.Envelope.Payload) != in.PayloadHash {
			return fmt.Errorf("%w: entry %d: the payload is not the one signed", ErrBroken, e.Index)
		}
		named := false
		for _, s := range b.Envelope.Sigs {
			sig, err := signer.ParseSignature(s)
			if err != nil {
				continue
			}
			if who, err := signer.Recover(in.TypedData(), sig); err == nil && who == in.Controller {
				named = true
			}
		}
		if !named {
			return fmt.Errorf("%w: entry %d: not signed by controller %s", ErrBroken, e.Index, in.Controller.Hex())
		}
	}
	return nil
}

// ToJSON is the wire and file form.
func ToJSON(e Entry) intent.JobLogEntryJSON {
	return intent.JobLogEntryJSON{Index: e.Index, Prev: "0x" + hex.EncodeToString(e.Prev[:]), Kind: e.Kind, Body: e.Body, At: e.At, Sig: e.Sig.Hex()}
}

// FromJSON parses the wire and file form. It does not verify.
func FromJSON(j intent.JobLogEntryJSON) (Entry, error) {
	e := Entry{Index: j.Index, Kind: j.Kind, Body: j.Body, At: j.At}
	p, err := hex.DecodeString(strings.TrimPrefix(j.Prev, "0x"))
	if err != nil || len(p) != 32 {
		return e, fmt.Errorf("%w: entry %d: prev is not 32 bytes of hex", ErrBroken, j.Index)
	}
	copy(e.Prev[:], p)
	if e.Sig, err = signer.ParseSignature(j.Sig); err != nil {
		return e, fmt.Errorf("%w: entry %d: %v", ErrBroken, j.Index, err)
	}
	return e, nil
}
```

```go
// internal/joblog/bodies.go
package joblog

import (
	"encoding/json"
	"fmt"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// Entry kinds.
const (
	KindCreated          = "job.created"       // IntentBody: the request that created the job
	KindIntent           = "intent"            // IntentBody: a later request that changed the job
	KindStepStarted      = "step.started"      // StepBody
	KindStepFinished     = "step.finished"     // StepBody with OK and Error
	KindStepLost         = "step.lost"         // StepBody: the unit ended without a result
	KindDecisionAsked    = "decision.asked"    // intent.JobQuestion
	KindDecisionAnswered = "decision.answered" // AnswerBody
	KindLeaseTaken       = "lease.taken"       // LeaseBody
	KindLeaseReleased    = "lease.released"    // LeaseBody
	KindFinished         = "job.finished"      // FinishedBody
)

// IntentBody records a request exactly as the controller signed it.
type IntentBody struct {
	Envelope intent.Envelope `json:"envelope"`
	Outcome  string          `json:"outcome"` // "created", "lease take", "advance", "decide", …
}

type StepBody struct {
	Step    string `json:"step"`
	Attempt int    `json:"attempt"`
	Unit    string `json:"unit,omitempty"`
	OK      bool   `json:"ok,omitempty"`
	Error   string `json:"error,omitempty"`
}

type AnswerBody struct {
	Question   string   `json:"question"`
	Option     string   `json:"option"`
	Controller string   `json:"controller"`
	Signers    []string `json:"signers"`
}

type LeaseBody struct {
	Owner    string `json:"owner"`
	Previous string `json:"previous,omitempty"`
}

type FinishedBody struct {
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}

// Describe is an entry as one line for a person. It reads only verified
// entries' bodies; an unreadable body says so rather than guessing.
func Describe(e Entry) string {
	bad := fmt.Sprintf("%s (unreadable body)", e.Kind)
	switch e.Kind {
	case KindCreated, KindIntent:
		var b IntentBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		return fmt.Sprintf("%s by %s: %s", b.Envelope.Intent.Kind, b.Envelope.Intent.Controller, b.Outcome)
	case KindStepStarted, KindStepFinished, KindStepLost:
		var b StepBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		switch e.Kind {
		case KindStepStarted:
			return fmt.Sprintf("step %s attempt %d started (%s)", b.Step, b.Attempt, b.Unit)
		case KindStepLost:
			return fmt.Sprintf("step %s attempt %d stopped without a result", b.Step, b.Attempt)
		}
		if b.OK {
			return fmt.Sprintf("step %s attempt %d finished", b.Step, b.Attempt)
		}
		return fmt.Sprintf("step %s attempt %d failed: %s", b.Step, b.Attempt, b.Error)
	case KindDecisionAsked:
		var q intent.JobQuestion
		if json.Unmarshal(e.Body, &q) != nil {
			return bad
		}
		// Describe is for one-line lists: a long prompt (a process list) is
		// cut to 160 runes; the inbox shows the whole question.
		p := []rune(strings.ReplaceAll(q.Prompt, "\n", " "))
		if len(p) > 160 {
			p = append(p[:159], '…')
		}
		return fmt.Sprintf("asked (%s tier): %s", q.Tier, string(p))
	case KindDecisionAnswered:
		var b AnswerBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		return fmt.Sprintf("answered %s by %s", b.Option, b.Controller)
	case KindLeaseTaken, KindLeaseReleased:
		var b LeaseBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		if e.Kind == KindLeaseReleased {
			return "lease released by " + b.Owner
		}
		if b.Previous != "" {
			return fmt.Sprintf("lease taken by %s from %s", b.Owner, b.Previous)
		}
		return "lease taken by " + b.Owner
	case KindFinished:
		var b FinishedBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		if b.Error != "" {
			return fmt.Sprintf("job %s: %s", b.State, b.Error)
		}
		return "job " + b.State
	}
	return e.Kind
}
```

Add `"strings"` to `bodies.go`'s imports.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/joblog/ -count=1 -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/joblog/joblog.go internal/joblog/bodies.go internal/joblog/joblog_test.go
git commit -m "feat(joblog): signed, hash-chained job log with controller-side verification

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 4: Job intents — kinds, reasons, records, policy defaults and hints

Covers spec "Intents, not commands" for jobs, "Policy (on the agent)", Q1.

**Files:**
- Create: `internal/intent/jobs.go`, `internal/intent/jobs_test.go`
- Modify: `internal/intent/kinds.go` (`AgentInfo` gains `Names`, `DataDir`), `internal/agent/policy.go` (`DefaultRequirements`), `internal/api/errors.go` (`rejectionHints`), `internal/api/errors_test.go` (`TestEveryRejectionReasonHasAHint` list)
- Test: `internal/intent/jobs_test.go`, `internal/agent/policy_test.go`, `internal/api/errors_test.go`

**Interfaces:**
- Produces (package `intent`):
  - kinds `KindJobList = "job.list"`, `KindJobRead = "job.read"`, `KindJobOutput = "job.output"`, `KindJobLease = "job.lease"`, `KindJobAdvance = "job.advance"`, `KindJobDecide = "job.decide"`, `KindNodeMigrateNames = "node.migrate-names"`;
  - reasons `ReasonJobsUnsupported = "jobs_unsupported"`, `ReasonJobNotFound = "job_not_found"`, `ReasonJobActive = "job_active"`, `ReasonLeaseHeld = "lease_held"`, `ReasonUnknownPlan = "unknown_plan"`, `ReasonJobState = "job_state"`;
  - states `JobWaiting = "waiting"`, `JobRunning = "running"`, `JobNeedsDecision = "needs-decision"`, `JobSucceeded = "succeeded"`, `JobFailed = "failed"`, `JobAborted = "aborted"`, `JobDamaged = "damaged"`, and `func JobTerminal(state string) bool`;
  - lease ops `LeaseTake = "take"`, `LeaseRenew = "renew"`, `LeaseRelease = "release"`;
  - options `OptRetry = "retry"`, `OptAbort = "abort"`;
  - question tiers `TierRoutine = "routine"`, `TierApproval = "approval"`;
  - types `JobStep`, `JobLease`, `JobOption`, `JobQuestion`, `JobRecord`, `JobListResult`, `JobReadPayload`, `JobReadResult`, `JobOutputPayload`, `JobOutputLine`, `JobOutputResult`, `JobLeasePayload`, `JobAdvancePayload`, `JobDecidePayload`, `MigrateNamesPayload`, `JobLogEntryJSON` (fields below).
- `AgentInfo` gains `Names string \`json:"names,omitempty"\``, `DataDir string \`json:"dataDir,omitempty"\``.

- [ ] **Step 1: Write the failing tests**

```go
// internal/intent/jobs_test.go
package intent

import (
	"encoding/json"
	"testing"
)

func TestJobTerminalStates(t *testing.T) {
	for s, want := range map[string]bool{
		JobWaiting: false, JobRunning: false, JobNeedsDecision: false,
		JobSucceeded: true, JobFailed: true, JobAborted: true, JobDamaged: true,
	} {
		if JobTerminal(s) != want {
			t.Errorf("JobTerminal(%s) = %v", s, !want)
		}
	}
}

// The record is the agent's signed answer and the manifest's core; its JSON
// names are a wire contract with every controller version.
func TestJobRecordWireNames(t *testing.T) {
	b, _ := json.Marshal(JobRecord{ID: "0123456789abcdef", Kind: KindNodeMigrateNames, State: JobWaiting,
		Lease: JobLease{Owner: "0xabc", HeartbeatAt: 5, TTLSeconds: 90},
		Question: &JobQuestion{ID: "q", Tier: TierRoutine, Options: []JobOption{{ID: OptRetry, Label: "Retry"}}}})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"format", "id", "kind", "title", "planVersion", "driver", "steps", "step", "attempt", "state", "lease", "question", "createdBy", "createdAt", "updatedAt", "logLen", "logHead", "agentVersion"} {
		if _, ok := m[k]; !ok {
			t.Errorf("JobRecord JSON lacks %q: %s", k, b)
		}
	}
}
```

```go
// internal/agent/policy_test.go — add
// Q1: every job kind is routine by default; a policy may raise any of them.
func TestJobKindsAreRoutineByDefault(t *testing.T) {
	for _, k := range []string{intent.KindJobList, intent.KindJobRead, intent.KindJobOutput, intent.KindJobLease,
		intent.KindJobAdvance, intent.KindJobDecide, intent.KindNodeMigrateNames} {
		if got := DefaultRequirements[k]; len(got) != 1 || got[0] != TierRoutine {
			t.Errorf("%s requires %v", k, got)
		}
	}
}
```

In `internal/api/errors_test.go`, add the six new reasons to the list in `TestEveryRejectionReasonHasAHint`.

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/intent/ ./internal/agent/ ./internal/api/ -run 'JobTerminal|JobRecordWire|JobKindsAreRoutine|EveryRejectionReason' -count=1`
Expected: FAIL to compile (undefined `JobTerminal`, `KindJobList`, …).

- [ ] **Step 3: Implement `internal/intent/jobs.go`**

```go
// internal/intent/jobs.go
package intent

import (
	"encoding/json"
	"time"
)

// Job kinds (sub-project 2). node.migrate-names starts a job; the others act
// on one. All are routine tier by default (agent.DefaultRequirements).
const (
	KindJobList          = "job.list"
	KindJobRead          = "job.read"
	KindJobOutput        = "job.output"
	KindJobLease         = "job.lease"
	KindJobAdvance       = "job.advance"
	KindJobDecide        = "job.decide"
	KindNodeMigrateNames = "node.migrate-names"
)

// Job rejection reasons. Stable, like the others: the server maps each to a hint.
const (
	ReasonJobsUnsupported = "jobs_unsupported" // not Linux, no systemd, or no job manager
	ReasonJobNotFound     = "job_not_found"
	ReasonJobActive       = "job_active"   // another job is not finished; one at a time per box
	ReasonLeaseHeld       = "lease_held"   // another controller holds the job's lease
	ReasonUnknownPlan     = "unknown_plan" // the manifest's format, driver or plan version is unknown here
	ReasonJobState        = "job_state"    // the job is not where this request expects it
)

// Job states.
const (
	JobWaiting       = "waiting"        // at a step boundary, for the lease holder to advance
	JobRunning       = "running"        // a step's unit is running
	JobNeedsDecision = "needs-decision" // stopped at a decision point
	JobSucceeded     = "succeeded"
	JobFailed        = "failed"
	JobAborted       = "aborted"
	JobDamaged       = "damaged" // its log or manifest no longer verifies on the box
)

// JobTerminal reports whether a job in state will never change again.
func JobTerminal(state string) bool {
	switch state {
	case JobSucceeded, JobFailed, JobAborted, JobDamaged:
		return true
	}
	return false
}

// Lease operations, options every failure question offers, question tiers.
const (
	LeaseTake    = "take"
	LeaseRenew   = "renew"
	LeaseRelease = "release"

	OptRetry = "retry"
	OptAbort = "abort"

	TierRoutine  = "routine"
	TierApproval = "approval"
)

type JobStep struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// JobLease is who drives a job. Freshness is the agent's to judge, by its own
// clock: HeartbeatAt is agent unix time.
type JobLease struct {
	Owner       string `json:"owner,omitempty"`
	HeartbeatAt int64  `json:"heartbeatAt,omitempty"`
	TTLSeconds  int    `json:"ttlSeconds,omitempty"`
}

type JobOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// JobQuestion is a decision point. ID is unique within the job; Key is the
// step's own name for it (what the step reads back from the answer).
type JobQuestion struct {
	ID      string      `json:"id"`
	Step    string      `json:"step"`
	Key     string      `json:"key,omitempty"`
	Prompt  string      `json:"prompt"`
	Options []JobOption `json:"options"`
	Tier    string      `json:"tier"`
	AskedAt int64       `json:"askedAt"`
}

// JobRecord is a job as the agent reports it: the manifest without its inputs.
type JobRecord struct {
	Format       int             `json:"format"`
	ID           string          `json:"id"`
	Kind         string          `json:"kind"`
	Title        string          `json:"title"`
	PlanVersion  string          `json:"planVersion"`
	Driver       string          `json:"driver"`
	Steps        []JobStep       `json:"steps"`
	Step         int             `json:"step"` // index into Steps; len(Steps) once finished
	Attempt      int             `json:"attempt"`
	State        string          `json:"state"`
	Unit         string          `json:"unit,omitempty"`
	Question     *JobQuestion    `json:"question,omitempty"`
	Lease        JobLease        `json:"lease"`
	CreatedBy    string          `json:"createdBy"`
	CreatedAt    int64           `json:"createdAt"`
	UpdatedAt    int64           `json:"updatedAt"`
	Error        string          `json:"error,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	LogLen       uint64          `json:"logLen"`
	LogHead      string          `json:"logHead"`
	AgentVersion string          `json:"agentVersion"`
}

type JobListResult struct {
	Jobs      []JobRecord `json:"jobs"`
	AgentTime int64       `json:"agentTime"`
}

type JobReadPayload struct {
	ID   string `json:"id"`
	From uint64 `json:"from"`
}

// JobReadResult pages the log: at most 500 entries from From.
type JobReadResult struct {
	Job       JobRecord         `json:"job"`
	Entries   []JobLogEntryJSON `json:"entries"`
	More      bool              `json:"more"`
	AgentTime int64             `json:"agentTime"`
}

type JobOutputPayload struct {
	ID     string `json:"id"`
	Cursor string `json:"cursor"`
	N      int    `json:"n"`
}

type JobOutputLine struct {
	Unit string    `json:"unit"`
	Line string    `json:"line"`
	At   time.Time `json:"at"`
}

type JobOutputResult struct {
	Lines  []JobOutputLine `json:"lines"`
	Cursor string          `json:"cursor"`
}

type JobLeasePayload struct {
	ID string `json:"id"`
	Op string `json:"op"`
}

// JobAdvancePayload names the position the controller saw, so an advance sent
// twice, or after another controller moved the job, is refused (job_state).
type JobAdvancePayload struct {
	ID      string `json:"id"`
	Step    int    `json:"step"`
	Attempt int    `json:"attempt"`
}

type JobDecidePayload struct {
	ID       string `json:"id"`
	Decision string `json:"decision"`
	Option   string `json:"option"`
}

// MigrateNamesPayload starts node.migrate-names. It takes no inputs: the box's
// own node.json says what to rename.
type MigrateNamesPayload struct{}

// JobLogEntryJSON is one signed log entry on the wire and on disk.
// Body is base64 in JSON, hashed as stored (joblog).
type JobLogEntryJSON struct {
	Index uint64 `json:"index"`
	Prev  string `json:"prev"`
	Kind  string `json:"kind"`
	Body  []byte `json:"body"`
	At    uint64 `json:"at"`
	Sig   string `json:"sig"`
}
```

In `kinds.go`, add to `AgentInfo`:

```go
	Names   string `json:"names,omitempty"`   // catalog names id of the box's node; "" is legacy
	DataDir string `json:"dataDir,omitempty"` // the node's data dir, after any names marker
```

In `internal/agent/policy.go`, add the seven kinds to `DefaultRequirements` with `{TierRoutine}`, and change the comment to "sub-project 1's kinds and sub-project 2's job kinds, all routine".

In `internal/api/errors.go` `rejectionHints`:

```go
	intent.ReasonJobsUnsupported: "durable jobs run only on Linux boxes with systemd as init; the reason above says which is missing",
	intent.ReasonJobNotFound:     "no such job on this box; `jumpgate jobs HOST` lists them",
	intent.ReasonJobActive:       "one job runs at a time on a box; wait for it, or decide it in the jobs inbox",
	intent.ReasonLeaseHeld:       "another controller drives this job; watch it, or adopt it once its lease is stale (90 s without a heartbeat)",
	intent.ReasonUnknownPlan:     "this box's agent does not know that job's plan (it was downgraded?); re-pair the box to upgrade its agent",
	intent.ReasonJobState:        "the job moved on since this request was made; reload it",
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/intent/ ./internal/agent/ ./internal/api/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/intent/jobs.go internal/intent/jobs_test.go internal/intent/kinds.go \
  internal/agent/policy.go internal/agent/policy_test.go internal/api/errors.go internal/api/errors_test.go
git commit -m "feat(intent): job intent kinds, records and reasons; routine by default

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 5: The job store — manifest, atomic files, signed appends, truncation check, leases

Covers spec "A manifest on the box (`/var/lib/jumpgate/jobs/<id>.json`) records the job's kind, its inputs, the current step, a `driver` field, a version stamp, and a lease (owner and heartbeat)", and J5, J6, J7. Review Focus 5 (agent side).

**Files:**
- Create: `internal/fsatomic/fsatomic.go`, `internal/fsatomic/fsatomic_test.go` (ruling P8: the one copy of the atomic write and the directory fsync), `internal/jobs/manifest.go`, `internal/jobs/store.go`, `internal/jobs/lease.go`, `internal/jobs/store_test.go`, `internal/jobs/lease_test.go`
- Modify: `internal/agent/policy.go` (delete `writeAtomic` and `syncDir` at lines 157-205; `Save` calls `fsatomic.WriteFile`), `internal/agent/replay.go:43,100` (`fsatomic.WriteFile`), `internal/agent/agent_test.go:274-277` and `internal/agent/replay_test.go:109-111` (stub `fsatomic.SyncDir`)
- Test: `internal/fsatomic/fsatomic_test.go`, `internal/jobs/store_test.go`, `internal/jobs/lease_test.go`; these agent tests must still pass unchanged in meaning: `TestAdmissionPersistFailureRefusesAndRunsNothing` (`agent_test.go`), `TestAdmitFailsWhenTheDirectorySyncFails` (`replay_test.go`), `TestPolicyAddSignerIsIdempotentAndSaveRoundTrips` (`policy_test.go`), and the rest of `go test ./internal/agent/`.

**Interfaces:**
- Consumes: Task 3 (`joblog.Sign`, `joblog.ToJSON`, `joblog.FromJSON`, `joblog.Verify`, kinds and bodies), Task 4 (`intent.JobRecord`, states, reasons, `intent.JobLease`).
- Produces (package `fsatomic`): `func WriteFile(path string, b []byte) error` (0600 through a fsynced temp file, rename, then a directory fsync); `var SyncDir func(dir string) error` (a variable so a test can make the directory fsync fail). Tasks 7 and 8 call `fsatomic.WriteFile` too (step results, the names marker, `node.json`).
- Produces (package `jobs`):
  - `const FormatVersion = 1`, `const DriverController = "controller"`, `const DriverAgent = "agent"`;
  - `type Manifest struct { intent.JobRecord; Inputs json.RawMessage; Results map[string]json.RawMessage; Decisions map[string]string; Answer *Answer }`, with `type Answer struct { Key, Option string; Step, Attempt int }`;
  - `type Refusal struct { Code, Message string }` (an `error`), `var ErrNotFound`, `var ErrDamaged`;
  - `func NewID() (string, error)`, `func ValidID(id string) bool`;
  - `type Store`, `func OpenStore(dir string, key signer.Signer, now func() time.Time) (*Store, error)`;
  - `(*Store) Create(ctx context.Context, m *Manifest, created joblog.IntentBody) error`, `Load(id string) (Manifest, error)`, `List() ([]Manifest, error)`, `Append(ctx context.Context, m *Manifest, kind string, body any) error`, `Save(m *Manifest) error`, `ReadLog(id string, from uint64, max int) ([]joblog.Entry, bool, error)`, `Prune(maxAge time.Duration, keep int) error`, `Dir() string`;
  - `func LoadManifest(dir, id string) (Manifest, error)` (read-only, no key, for step processes);
  - `func ResultPath(dir, id, step string, attempt int) string`;
  - `const LeaseTTL = 90 * time.Second`; `(Manifest) LeaseFresh(now time.Time) bool`; `(*Manifest) TakeLease(who eip712.Address, now time.Time) (changed bool, prev string, err error)`; `(*Manifest) RenewLease(who eip712.Address, now time.Time) error`; `(*Manifest) ReleaseLease(who eip712.Address) error`; `(Manifest) Holds(who eip712.Address) error`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/jobs/store_test.go
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func testStore(t *testing.T) (*Store, *signer.Key, *time.Time) {
	t.Helper()
	testutil.RequireUnix(t) // the store fsyncs directories, as the agent's replay record does
	k, _ := signer.GenerateKey()
	now := time.Unix(1_800_000_000, 0)
	s, err := OpenStore(filepath.Join(t.TempDir(), "jobs"), k, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return s, k, &now
}

func newManifest(t *testing.T) Manifest {
	id, err := NewID()
	if err != nil || !ValidID(id) {
		t.Fatalf("NewID = %q, %v", id, err)
	}
	return Manifest{JobRecord: intent.JobRecord{Format: FormatVersion, ID: id, Kind: "test.steps", PlanVersion: "test.steps/1",
		Driver: DriverController, State: intent.JobWaiting}, Inputs: json.RawMessage(`{}`)}
}

func TestCreateWritesTheManifestAndASignedFirstEntry(t *testing.T) {
	s, k, _ := testStore(t)
	m := newManifest(t)
	if err := s.Create(context.Background(), &m, joblog.IntentBody{Outcome: "created"}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(s.Dir(), m.ID+".json"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("manifest: %v, mode %v", err, fi.Mode())
	}
	got, err := s.Load(m.ID)
	if err != nil || got.LogLen != 1 {
		t.Fatalf("Load = %+v, %v", got.JobRecord, err)
	}
	es, more, err := s.ReadLog(m.ID, 0, 10)
	if err != nil || more || len(es) != 1 || es[0].Kind != joblog.KindCreated {
		t.Fatalf("ReadLog = %v, %v, %v", es, more, err)
	}
	if _, err := joblog.Verify(k.Address(), m.ID, es, 0, [32]byte{}); err != nil {
		t.Fatal(err)
	}
}

// Review Focus 5: a log shorter than the manifest says is damage, never a
// fresh start; one entry longer is an append the crash cut off before the
// manifest save, which Load completes.
func TestLoadDetectsATruncatedLog(t *testing.T) {
	s, _, _ := testStore(t)
	m := newManifest(t)
	ctx := context.Background()
	_ = s.Create(ctx, &m, joblog.IntentBody{})
	_ = s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"})
	_ = s.Save(&m)

	logPath := filepath.Join(s.Dir(), m.ID+".log")
	b, _ := os.ReadFile(logPath)
	lines := strings.SplitAfter(strings.TrimSpace(string(b)), "\n")
	// lines[0] already ends in "\n": the file keeps one whole, valid entry, so
	// the parse and the chain check pass and only the length check (one entry
	// where the manifest expects two) can refuse it.
	_ = os.WriteFile(logPath, []byte(lines[0]), 0o600)
	_, err := s.Load(m.ID)
	if !errors.Is(err, ErrDamaged) || !strings.Contains(err.Error(), "its manifest expects 2") {
		t.Fatalf("Load of a truncated log = %v, want ErrDamaged from the length check", err)
	}
}

func TestLoadDetectsALogCutMidEntry(t *testing.T) {
	s, _, _ := testStore(t)
	m := newManifest(t)
	ctx := context.Background()
	_ = s.Create(ctx, &m, joblog.IntentBody{})
	_ = s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"})
	_ = s.Save(&m)

	logPath := filepath.Join(s.Dir(), m.ID+".log")
	b, _ := os.ReadFile(logPath)
	_ = os.WriteFile(logPath, b[:len(b)-10], 0o600) // a torn last line
	if _, err := s.Load(m.ID); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Load of a truncated log = %v, want ErrDamaged", err)
	}
}

func TestLoadCompletesAnAppendTheManifestMissed(t *testing.T) {
	s, _, _ := testStore(t)
	m := newManifest(t)
	ctx := context.Background()
	_ = s.Create(ctx, &m, joblog.IntentBody{})
	stale := m // the manifest as saved
	_ = s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"})
	_ = s.Save(&stale) // a crash between the append and the save
	got, err := s.Load(m.ID)
	if err != nil || got.LogLen != 2 || got.LogHead != m.LogHead {
		t.Fatalf("Load = len %d head %s, %v; want len 2 head %s", got.LogLen, got.LogHead, err, m.LogHead)
	}
}

func TestListIsNewestFirstAndShowsDamagedJobs(t *testing.T) {
	s, _, now := testStore(t)
	ctx := context.Background()
	a := newManifest(t)
	a.CreatedAt = now.Unix()
	_ = s.Create(ctx, &a, joblog.IntentBody{})
	b := newManifest(t)
	b.CreatedAt = now.Unix() + 10
	_ = s.Create(ctx, &b, joblog.IntentBody{})
	_ = os.WriteFile(filepath.Join(s.Dir(), a.ID+".log"), nil, 0o600)
	ms, err := s.List()
	if err != nil || len(ms) != 2 || ms[0].ID != b.ID || ms[1].State != intent.JobDamaged {
		t.Fatalf("List = %+v, %v", ms, err)
	}
}

func TestPruneKeepsRecentAndUnfinishedJobs(t *testing.T) {
	s, _, now := testStore(t)
	ctx := context.Background()
	// Age is judged from a job's last change (D-J6), and every append stamps
	// UpdatedAt with the store's clock. So the test writes the old jobs first
	// and then moves the store's clock on 31 days, instead of presetting
	// UpdatedAt (Create would overwrite it).
	old := newManifest(t)
	old.State = intent.JobSucceeded
	_ = s.Create(ctx, &old, joblog.IntentBody{})
	running := newManifest(t)
	running.State = intent.JobRunning
	_ = s.Create(ctx, &running, joblog.IntentBody{})
	*now = now.Add(31 * 24 * time.Hour)
	recent := newManifest(t)
	recent.State = intent.JobSucceeded
	_ = s.Create(ctx, &recent, joblog.IntentBody{})
	if err := s.Prune(30*24*time.Hour, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(old.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("an old finished job survived: %v", err)
	}
	if _, err := s.Load(running.ID); err != nil {
		t.Errorf("an unfinished job was pruned: %v", err)
	}
	if _, err := s.Load(recent.ID); err != nil {
		t.Errorf("a recent finished job was pruned: %v", err)
	}
}

func TestValidIDRefusesAnythingButSixteenHex(t *testing.T) {
	for _, id := range []string{"", "0123456789ABCDEF", "0123456789abcde", "../../etc/passwd", "0123456789abcdef0"} {
		if ValidID(id) {
			t.Errorf("ValidID(%q) = true", id)
		}
	}
}
```

```go
// internal/jobs/lease_test.go
package jobs

import (
	"errors"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

func TestLeaseTakeRenewRelease(t *testing.T) {
	a, _ := signer.GenerateKey()
	b, _ := signer.GenerateKey()
	now := time.Unix(1_800_000_000, 0)
	var m Manifest

	if changed, prev, err := m.TakeLease(a.Address(), now); err != nil || !changed || prev != "" {
		t.Fatalf("first take: %v %q %v", changed, prev, err)
	}
	// A fresh lease refuses another controller, by the agent's clock.
	var r *Refusal
	if _, _, err := m.TakeLease(b.Address(), now.Add(LeaseTTL-time.Second)); !errors.As(err, &r) || r.Code != intent.ReasonLeaseHeld {
		t.Fatalf("take of a fresh lease = %v", err)
	}
	if err := m.RenewLease(b.Address(), now); err == nil {
		t.Fatal("a controller renewed a lease it does not hold")
	}
	// Taking your own lease again is a renewal: no change to log.
	if changed, _, err := m.TakeLease(a.Address(), now.Add(10*time.Second)); err != nil || changed {
		t.Fatalf("own retake: %v %v", changed, err)
	}
	// Stale: b may take it over, and the previous owner is named.
	if changed, prev, err := m.TakeLease(b.Address(), now.Add(10*time.Second+LeaseTTL+time.Second)); err != nil || !changed || prev != a.Address().Hex() {
		t.Fatalf("takeover: %v %q %v", changed, prev, err)
	}
	if err := m.Holds(a.Address()); err == nil {
		t.Fatal("the old owner still holds the lease")
	}
	if err := m.ReleaseLease(b.Address()); err != nil || m.Lease.Owner != "" {
		t.Fatalf("release: %v %+v", err, m.Lease)
	}
}

// A clock that steps backwards must not make a held lease look stale.
func TestLeaseIsFreshWhenTheClockStepsBack(t *testing.T) {
	a, _ := signer.GenerateKey()
	now := time.Unix(1_800_000_000, 0)
	var m Manifest
	_, _, _ = m.TakeLease(a.Address(), now)
	if !m.LeaseFresh(now.Add(-time.Hour)) {
		t.Fatal("a heartbeat in the future read as stale")
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/jobs/ -count=1`
Expected: FAIL to compile (package has no non-test files).

- [ ] **Step 3: Implement `manifest.go`, `lease.go` and `store.go`, and move the atomic write into `internal/fsatomic`**

```go
// internal/jobs/manifest.go

// Package jobs is the agent's durable-job engine: the manifests and signed
// logs under /var/lib/jumpgate/jobs, the leases controllers hold, the plans a
// job can run, and the transient systemd units each step runs in. The agent
// process is the only writer of manifests and the only signer; a step
// process reads its manifest and writes one result file (spec D-J4).
package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// FormatVersion is the manifest format this agent reads and writes. The later
// agent-driven driver needs no new format (spec): it only sets Driver.
const FormatVersion = 1

const (
	DriverController = "controller" // v1: the lease holder advances step by step
	DriverAgent      = "agent"      // later: the agent drives the whole plan
)

// Manifest is /var/lib/jumpgate/jobs/<id>.json.
type Manifest struct {
	intent.JobRecord
	// Inputs are the job's inputs as the intent carried them: decisions, never
	// rendered commands (for setup, the wire config plus the plan version).
	Inputs json.RawMessage `json:"inputs"`
	// Results holds each finished step's result, by step id; later steps read them.
	Results map[string]json.RawMessage `json:"results,omitempty"`
	// Decisions is every answer given, by question id.
	Decisions map[string]string `json:"decisions,omitempty"`
	// Answer is the last answer, which the next attempt of its step reads.
	Answer *Answer `json:"answer,omitempty"`
}

// Answer is what a step asked and what it was told, so the step's next attempt
// can act on it (Env.Answer).
type Answer struct {
	Key     string `json:"key"`
	Option  string `json:"option"`
	Step    int    `json:"step"`
	Attempt int    `json:"attempt"`
}

// Refusal is a request the engine refuses; Code is an intent.Reason*. The
// agent turns it into a signed rejection.
type Refusal struct{ Code, Message string }

func (r *Refusal) Error() string { return r.Code + ": " + r.Message }

func refuse(code, msg string) error { return &Refusal{Code: code, Message: msg} }

var (
	ErrNotFound = errors.New("jobs: no such job")
	ErrDamaged  = errors.New("jobs: the job's log or manifest is damaged")
)

var idRE = regexp.MustCompile(`^[0-9a-f]{16}$`)

// NewID is 16 lowercase hex characters: safe in unit names, paths and journal
// patterns without quoting (D-J5).
func NewID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func ValidID(id string) bool { return idRE.MatchString(id) }
```

```go
// internal/jobs/lease.go
package jobs

import (
	"fmt"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// LeaseTTL is how long a lease stays fresh without a heartbeat (D-J2): three
// missed 20-second renewals.
const LeaseTTL = 90 * time.Second

// LeaseFresh is judged by the agent's clock. A heartbeat in the future (the
// clock stepped back) is fresh: losing a lease to a clock step would hand a
// running job to whoever asks next.
func (m Manifest) LeaseFresh(now time.Time) bool {
	if m.Lease.Owner == "" {
		return false
	}
	return now.Unix()-m.Lease.HeartbeatAt < int64(LeaseTTL/time.Second)
}

func (m Manifest) ownedBy(who eip712.Address) bool {
	a, err := eip712.ParseAddress(m.Lease.Owner)
	return err == nil && a == who
}

// Holds is nil when who holds the lease. A holder whose lease went stale but
// that nobody took still holds it: it may carry on.
func (m Manifest) Holds(who eip712.Address) error {
	if m.ownedBy(who) {
		return nil
	}
	if m.Lease.Owner == "" {
		return refuse(intent.ReasonLeaseHeld, fmt.Sprintf("nobody holds job %s; take its lease first", m.ID))
	}
	return refuse(intent.ReasonLeaseHeld, fmt.Sprintf("controller %s holds job %s", m.Lease.Owner, m.ID))
}

// TakeLease gives who the lease if it is free, stale or already who's.
// changed is false for a retake by the owner (a renewal, not logged: Q7).
func (m *Manifest) TakeLease(who eip712.Address, now time.Time) (changed bool, prev string, err error) {
	if m.ownedBy(who) {
		m.Lease.HeartbeatAt = now.Unix()
		return false, "", nil
	}
	if m.LeaseFresh(now) {
		age := now.Unix() - m.Lease.HeartbeatAt
		return false, "", refuse(intent.ReasonLeaseHeld, fmt.Sprintf(
			"controller %s holds job %s (heartbeat %ds ago); watch it, or adopt it once it has been silent for %ds",
			m.Lease.Owner, m.ID, age, int(LeaseTTL/time.Second)))
	}
	prev = m.Lease.Owner
	m.Lease = intent.JobLease{Owner: who.Hex(), HeartbeatAt: now.Unix(), TTLSeconds: int(LeaseTTL / time.Second)}
	return true, prev, nil
}

func (m *Manifest) RenewLease(who eip712.Address, now time.Time) error {
	if err := m.Holds(who); err != nil {
		return err
	}
	m.Lease.HeartbeatAt = now.Unix()
	return nil
}

func (m *Manifest) ReleaseLease(who eip712.Address) error {
	if err := m.Holds(who); err != nil {
		return err
	}
	m.Lease = intent.JobLease{}
	return nil
}
```

```go
// internal/jobs/store.go
package jobs

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// Store is the jobs directory. Every write is the agent's: Save replaces a
// manifest atomically, Append adds one signed line to a log and fsyncs it.
type Store struct {
	dir string
	key signer.Signer
	now func() time.Time
	mu  sync.Mutex
}

// OpenStore creates dir (0700) if absent.
func OpenStore(dir string, key signer.Signer, now func() time.Time) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	return &Store{dir: dir, key: key, now: now}, nil
}

func (s *Store) Dir() string { return s.dir }

func manifestPath(dir, id string) string { return filepath.Join(dir, id+".json") }
func logPath(dir, id string) string      { return filepath.Join(dir, id+".log") }
func resultDir(dir, id string) string    { return filepath.Join(dir, id+".d") }

// ResultPath is where step step's attempt writes its result.
func ResultPath(dir, id, step string, attempt int) string {
	return filepath.Join(resultDir(dir, id), step+"-"+strconv.Itoa(attempt)+".json")
}

// Create writes entry 0 (job.created) and then the manifest. The order
// matters: a crash in between leaves a log Load completes, never a manifest
// whose history is missing.
func (s *Store) Create(ctx context.Context, m *Manifest, created joblog.IntentBody) error {
	if !ValidID(m.ID) {
		return fmt.Errorf("jobs: %q is not a job id", m.ID)
	}
	if err := os.MkdirAll(resultDir(s.dir, m.ID), 0o700); err != nil {
		return err
	}
	m.LogLen, m.LogHead = 0, ""
	if err := s.Append(ctx, m, joblog.KindCreated, created); err != nil {
		return err
	}
	return s.Save(m)
}

// Append signs body as the next entry of m's log, writes it and fsyncs, and
// moves m's LogLen and LogHead on. The caller saves m.
func (s *Store) Append(ctx context.Context, m *Manifest, kind string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	var prev [32]byte
	if m.LogHead != "" {
		p, err := hex.DecodeString(strings.TrimPrefix(m.LogHead, "0x"))
		if err != nil || len(p) != 32 {
			return fmt.Errorf("%w: job %s has a bad log head", ErrDamaged, m.ID)
		}
		copy(prev[:], p)
	}
	e, d, err := joblog.Sign(ctx, s.key, m.ID, m.LogLen, prev, kind, b, s.now())
	if err != nil {
		return err
	}
	line, _ := json.Marshal(joblog.ToJSON(e))
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(logPath(s.dir, m.ID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	m.LogLen++
	m.LogHead = "0x" + hex.EncodeToString(d[:])
	m.UpdatedAt = s.now().Unix()
	return nil
}

// Save replaces m's manifest atomically, 0600.
func (s *Store) Save(m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(manifestPath(s.dir, m.ID), b)
}

// LoadManifest reads a manifest without checking its log: for step processes,
// which hold no key and only read.
func LoadManifest(dir, id string) (Manifest, error) {
	var m Manifest
	if !ValidID(id) {
		return m, ErrNotFound
	}
	b, err := os.ReadFile(manifestPath(dir, id))
	if errors.Is(err, fs.ErrNotExist) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("%w: manifest %s: %v", ErrDamaged, id, err)
	}
	return m, nil
}

// Load reads a manifest and checks its log against it. The log may be one
// entry longer (an append whose manifest save a crash cut off), which Load
// completes; any other mismatch is ErrDamaged.
func (s *Store) Load(id string) (Manifest, error) {
	m, err := LoadManifest(s.dir, id)
	if err != nil {
		return m, err
	}
	entries, err := s.readAll(id)
	if err != nil {
		return m, err
	}
	head, err := joblog.Verify(s.key.Address(), id, entries, 0, [32]byte{})
	if err != nil {
		return m, fmt.Errorf("%w: %v", ErrDamaged, err)
	}
	n := uint64(len(entries))
	switch {
	case n == m.LogLen && "0x"+hex.EncodeToString(head[:]) == m.LogHead:
		return m, nil
	case n == m.LogLen+1:
		m.LogLen, m.LogHead = n, "0x"+hex.EncodeToString(head[:])
		return m, s.Save(&m)
	}
	return m, fmt.Errorf("%w: job %s's log has %d entries, its manifest expects %d", ErrDamaged, id, n, m.LogLen)
}

func (s *Store) readAll(id string) ([]joblog.Entry, error) {
	es, _, err := s.ReadLog(id, 0, 0)
	return es, err
}

// ReadLog returns up to max entries from index from (max 0: all), and
// whether more follow.
func (s *Store) ReadLog(id string, from uint64, max int) ([]joblog.Entry, bool, error) {
	f, err := os.Open(logPath(s.dir, id))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	var out []joblog.Entry
	var i uint64
	for sc.Scan() {
		if i < from {
			i++
			continue
		}
		if max > 0 && len(out) == max {
			return out, true, nil
		}
		var j intent.JobLogEntryJSON
		if err := json.Unmarshal(sc.Bytes(), &j); err != nil {
			return out, false, fmt.Errorf("%w: job %s entry %d: %v", ErrDamaged, id, i, err)
		}
		e, err := joblog.FromJSON(j)
		if err != nil {
			return out, false, fmt.Errorf("%w: %v", ErrDamaged, err)
		}
		out = append(out, e)
		i++
	}
	return out, false, sc.Err()
}

// List is every job, newest first. A job whose log does not check out is
// listed as damaged with the reason, never hidden.
func (s *Store) List() ([]Manifest, error) {
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var out []Manifest
	for _, e := range ents {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !ValidID(id) {
			continue
		}
		m, err := s.Load(id)
		if errors.Is(err, ErrDamaged) {
			m.State, m.Error = intent.JobDamaged, err.Error()
		} else if err != nil {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// Prune removes finished jobs last changed more than maxAge ago, and all but
// the newest keep finished jobs (D-J6). Unfinished jobs are never pruned.
func (s *Store) Prune(maxAge time.Duration, keep int) error {
	ms, err := s.List()
	if err != nil {
		return err
	}
	cutoff, kept := s.now().Add(-maxAge).Unix(), 0
	for _, m := range ms {
		if !intent.JobTerminal(m.State) {
			continue
		}
		kept++
		if m.UpdatedAt >= cutoff && kept <= keep {
			continue
		}
		_ = os.RemoveAll(resultDir(s.dir, m.ID))
		_ = os.Remove(logPath(s.dir, m.ID))
		if err := os.Remove(manifestPath(s.dir, m.ID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return fsatomic.SyncDir(s.dir)
}
```

Move `writeAtomic` and `syncDir` out of `internal/agent/policy.go:157-205` into a package of their own (ruling P8). The jobs store cannot import the agent (the agent imports jobs), and two copies that "must stay alike" drift. After the move there is one copy:

```go
// internal/fsatomic/fsatomic.go

// Package fsatomic writes a file so that a crash leaves either the old
// contents or the new, never a torn mix, and the new name is durable once
// WriteFile returns. The agent's policy and replay record and the jobs store
// all persist through it.
package fsatomic

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes b to path (0600) through a fsynced temp file and a rename,
// then fsyncs the directory so the rename itself survives a power cut.
func WriteFile(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return SyncDir(dir)
}

// SyncDir fsyncs a directory. It is a variable so a test can make the
// directory fsync fail.
var SyncDir = func(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("fsatomic: open %s to sync it: %w", dir, err)
	}
	if err := d.Sync(); err != nil {
		d.Close()
		return fmt.Errorf("fsatomic: sync directory %s: %w", dir, err)
	}
	if err := d.Close(); err != nil {
		return fmt.Errorf("fsatomic: close directory %s: %w", dir, err)
	}
	return nil
}
```

```go
// internal/fsatomic/fsatomic_test.go
package fsatomic

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestWriteFileReplacesAtomicallyAt0600(t *testing.T) {
	testutil.RequireUnix(t) // directories are fsynced, which Windows cannot do
	path := filepath.Join(t.TempDir(), "sub", "f.json")
	for _, body := range []string{"one", "two"} {
		if err := WriteFile(path, []byte(body)); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil || string(b) != body {
			t.Fatalf("after writing %q: %q, %v", body, b, err)
		}
		if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode after writing %q: %v, %v", body, fi, err)
		}
	}
	if ents, _ := os.ReadDir(filepath.Dir(path)); len(ents) != 1 {
		t.Fatalf("temp files left behind: %v", ents)
	}
}

// If the directory fsync fails the rename is not known to be durable, so the
// caller must hear about it.
func TestWriteFileReportsAFailedDirectorySync(t *testing.T) {
	testutil.RequireUnix(t)
	orig := SyncDir
	SyncDir = func(string) error { return errors.New("injected dir sync failure") }
	defer func() { SyncDir = orig }()
	if err := WriteFile(filepath.Join(t.TempDir(), "f"), []byte("x")); err == nil {
		t.Fatal("WriteFile succeeded although the directory sync failed")
	}
}
```

Then point the agent at it:
- `internal/agent/policy.go`: delete `writeAtomic` and `syncDir` (lines 157-205). In `Save` (line 78), `return writeAtomic(path, b)` becomes `return fsatomic.WriteFile(path, b)`. Import `github.com/valve-tech/jumpgate/internal/fsatomic` and drop `path/filepath`, which nothing else in the file uses.
- `internal/agent/replay.go:43,100`: `writeAtomic(` becomes `fsatomic.WriteFile(`; add the import.
- `internal/agent/agent_test.go:274-277` (`TestAdmissionPersistFailureRefusesAndRunsNothing`) and `internal/agent/replay_test.go:109-111` (`TestAdmitFailsWhenTheDirectorySyncFails`): `orig := syncDir` / `syncDir = …` become `orig := fsatomic.SyncDir` / `fsatomic.SyncDir = …`; add the import. The tests' meaning does not change: a failed directory fsync still refuses admission and runs nothing.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/fsatomic/ ./internal/jobs/ -count=1 -race`
Expected: PASS.

Run: `go test ./internal/agent/ -run 'TestAdmissionPersistFailureRefusesAndRunsNothing|TestAdmitFailsWhenTheDirectorySyncFails|TestPolicyAddSignerIsIdempotentAndSaveRoundTrips' -count=1 -v`
Expected: PASS, all three listed. Then `go test ./internal/agent/ -count=1` passes whole, and `grep -rn 'func writeAtomic\|var syncDir' internal` prints nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/fsatomic/fsatomic.go internal/fsatomic/fsatomic_test.go \
  internal/jobs/manifest.go internal/jobs/store.go internal/jobs/lease.go internal/jobs/store_test.go internal/jobs/lease_test.go \
  internal/agent/policy.go internal/agent/replay.go internal/agent/agent_test.go internal/agent/replay_test.go
git commit -m "feat(jobs): job manifests, signed append-only logs and leases on the box

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 6: Step runners — systemd-run on Linux, a clear refusal everywhere else

Covers spec "Each long operation runs as a transient systemd unit `jumpgate-job-<id>[-<step>]` started by the agent with `systemd-run`", plus J3 and J9 (what jobs do on other OSes).

**Files:**
- Create: `internal/jobs/runner.go`, `internal/jobs/runner_linux.go`, `internal/jobs/runner_other.go`, `internal/jobs/runner_test.go`, `internal/jobs/runner_other_test.go` (`//go:build !linux`), `internal/jobs/jobstest/jobstest.go` (the runner half; Task 7 adds the plan half)

**Interfaces:**
- Consumes: `executor.Executor`.
- Produces (package `jobs`):
  - `type Runner interface { Supported(ctx context.Context) error; Start(ctx context.Context, unit string, argv []string) error; Active(ctx context.Context, unit string) (bool, error) }`;
  - `var ErrUnsupported`;
  - `func UnitName(id, step string, attempt int) string` (no `.service` suffix), `func UnitPattern(id string) string` (`jumpgate-job-<id>-*`), `func ValidStepID(s string) bool`;
  - `type SystemdRunner struct { Exec executor.Executor }`, `func DefaultRunner(ex executor.Executor) Runner`.
- Produces (package `jobstest`): `type Runner struct { Run func(ctx context.Context, unit string, argv []string) error; Unsupported error; … }` with `Supported`, `Start`, `Active`, `Wait()`, `Started() []string`, `Lose(unit string)`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/jobs/runner_test.go
package jobs

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// recExec records commands and answers each by the first matching prefix.
type recExec struct {
	cmds    []string
	answers map[string]executor.Result
}

func (r *recExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	r.cmds = append(r.cmds, cmd)
	for p, res := range r.answers {
		if strings.HasPrefix(cmd, p) {
			return res, nil
		}
	}
	return executor.Result{}, nil
}
func (r *recExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (r *recExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (r *recExec) Close() error                                                 { return nil }

func TestUnitNames(t *testing.T) {
	if got := UnitName("0123456789abcdef", "stop-units", 2); got != "jumpgate-job-0123456789abcdef-stop-units-2" {
		t.Fatal(got)
	}
	if UnitPattern("0123456789abcdef") != "jumpgate-job-0123456789abcdef-*" {
		t.Fatal(UnitPattern("0123456789abcdef"))
	}
	for _, s := range []string{"", "Stop", "a b", "-x", strings.Repeat("a", 33), "a;b"} {
		if ValidStepID(s) {
			t.Errorf("ValidStepID(%q)", s)
		}
	}
}

func TestSystemdRunnerStartQuotesEveryArgument(t *testing.T) {
	ex := &recExec{}
	r := SystemdRunner{Exec: ex}
	err := r.Start(context.Background(), "jumpgate-job-0123456789abcdef-one-0",
		[]string{"/usr/local/lib/jumpgate/jumpgate", "agent", "job-step", "--job", "0123456789abcdef", "--state-dir", "/var/lib/it's"})
	if err != nil {
		t.Fatal(err)
	}
	want := "systemd-run --unit=jumpgate-job-0123456789abcdef-one-0 --description='jumpgate job 0123456789abcdef step one-0' --collect --quiet --property=Type=exec -- '/usr/local/lib/jumpgate/jumpgate' 'agent' 'job-step' '--job' '0123456789abcdef' '--state-dir' '/var/lib/it'\\''s'"
	if ex.cmds[0] != want {
		t.Fatalf("command\n%s\nwant\n%s", ex.cmds[0], want)
	}
}

func TestSystemdRunnerRefusesABadUnitName(t *testing.T) {
	if err := (SystemdRunner{Exec: &recExec{}}).Start(context.Background(), "x; rm -rf /", nil); err == nil {
		t.Fatal("a unit name with a shell metacharacter was accepted")
	}
}

func TestSystemdRunnerStartReportsTheExit(t *testing.T) {
	ex := &recExec{answers: map[string]executor.Result{"systemd-run": {ExitCode: 1, Stderr: "Unit jumpgate-job-x already exists."}}}
	err := SystemdRunner{Exec: ex}.Start(context.Background(), "jumpgate-job-0123456789abcdef-one-0", []string{"/bin/true"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
}

// A bus error must never read as "finished": the agent would record the step
// as lost and ask a person about a step that is still running.
func TestSystemdRunnerActiveKnowsInactiveFromUnknown(t *testing.T) {
	for out, want := range map[string]struct {
		active bool
		err    bool
	}{
		"active\n": {true, false}, "activating\n": {true, false}, "deactivating\n": {true, false},
		"inactive\n": {false, false}, "failed\n": {false, false},
		"": {false, true}, "Failed to connect to bus\n": {false, true},
	} {
		ex := &recExec{answers: map[string]executor.Result{"systemctl is-active": {Stdout: out, ExitCode: 3}}}
		got, err := SystemdRunner{Exec: ex}.Active(context.Background(), "jumpgate-job-0123456789abcdef-one-0")
		if got != want.active || (err != nil) != want.err {
			t.Errorf("is-active %q: %v, %v", out, got, err)
		}
	}
}

func TestSystemdRunnerSupportedNamesWhatIsMissing(t *testing.T) {
	ex := &recExec{answers: map[string]executor.Result{"test -d /run/systemd/system": {ExitCode: 1}}}
	err := SystemdRunner{Exec: ex}.Supported(context.Background())
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "systemd") {
		t.Fatalf("Supported = %v", err)
	}
}
```

Add `internal/jobs/runner_other_test.go` with `//go:build !linux`:

```go
//go:build !linux

package jobs

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
)

// J9: on macOS (where the agent runs only for its tests) jobs are refused
// with a reason that names the OS.
func TestDefaultRunnerRefusesOffLinux(t *testing.T) {
	err := DefaultRunner(nil).Supported(context.Background())
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), runtime.GOOS) {
		t.Fatalf("Supported = %v", err)
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/jobs/ -run 'Unit|Systemd|DefaultRunner' -count=1`
Expected: FAIL to compile (undefined `UnitName`, `SystemdRunner`, …).

- [ ] **Step 3: Implement**

```go
// internal/jobs/runner.go
package jobs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// Runner starts a step's process somewhere that outlives the agent.
type Runner interface {
	// Supported is nil when this box can run jobs, or an ErrUnsupported
	// that says why not.
	Supported(ctx context.Context) error
	// Start launches argv as the transient unit unit. It returns once
	// systemd accepted (and, with Type=exec, executed) it.
	Start(ctx context.Context, unit string, argv []string) error
	// Active reports whether unit is still running. An error means "cannot
	// tell", which callers treat as "still running".
	Active(ctx context.Context, unit string) (bool, error)
}

var ErrUnsupported = errors.New("jobs: durable jobs are not available on this box")

var (
	stepRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	unitRE = regexp.MustCompile(`^jumpgate-job-[0-9a-f]{16}-[a-z0-9][a-z0-9-]{0,31}-[0-9]{1,4}$`)
)

func ValidStepID(s string) bool { return stepRE.MatchString(s) }

// UnitName is the spec's jumpgate-job-<id>-<step>, where <step> is the step
// id and the attempt (D-J3), so a retry never collides with its failed
// predecessor.
func UnitName(id, step string, attempt int) string {
	return "jumpgate-job-" + id + "-" + step + "-" + strconv.Itoa(attempt)
}

// UnitPattern matches every unit of a job, for journalctl -u.
func UnitPattern(id string) string { return "jumpgate-job-" + id + "-*" }

// SystemdRunner runs steps with systemd-run. The unit is PID 1's child, not
// the agent's, so it survives an agent restart; --collect unloads it when it
// ends, and the step's result file, not the unit, says how it ended.
type SystemdRunner struct{ Exec executor.Executor }

func (r SystemdRunner) Supported(ctx context.Context) error {
	res, err := r.Exec.Run(ctx, "test -d /run/systemd/system", nil)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("%w: this box does not run systemd as its init (/run/systemd/system is absent)", ErrUnsupported)
	}
	res, err = r.Exec.Run(ctx, "command -v systemd-run", nil)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("%w: systemd-run is not installed", ErrUnsupported)
	}
	return nil
}

func (r SystemdRunner) Start(ctx context.Context, unit string, argv []string) error {
	if !unitRE.MatchString(unit) || len(argv) == 0 {
		return fmt.Errorf("jobs: refusing to start unit %q", unit)
	}
	step := unit[len("jumpgate-job-")+17:]
	id := unit[len("jumpgate-job-") : len("jumpgate-job-")+16]
	var b strings.Builder
	b.WriteString("systemd-run --unit=" + unit)
	b.WriteString(" --description=" + shQuote("jumpgate job "+id+" step "+step))
	b.WriteString(" --collect --quiet --property=Type=exec --")
	for _, a := range argv {
		b.WriteString(" " + shQuote(a))
	}
	res, err := r.Exec.Run(ctx, b.String(), nil)
	if err != nil {
		return fmt.Errorf("jobs: systemd-run %s: %w", unit, err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("jobs: systemd-run %s exited %d: %s", unit, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

func (r SystemdRunner) Active(ctx context.Context, unit string) (bool, error) {
	if !unitRE.MatchString(unit) {
		return false, fmt.Errorf("jobs: %q is not a job unit", unit)
	}
	// is-active exits non-zero for every state but active; only stdout says
	// which. A collected unit reads "inactive".
	res, err := r.Exec.Run(ctx, "systemctl is-active "+unit+".service", nil)
	if err != nil {
		return false, err
	}
	switch strings.TrimSpace(res.Stdout) {
	case "active", "activating", "deactivating", "reloading", "refreshing":
		return true, nil
	case "inactive", "failed":
		return false, nil
	}
	return false, fmt.Errorf("jobs: systemctl is-active %s: %q %s", unit, strings.TrimSpace(res.Stdout), strings.TrimSpace(res.Stderr))
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
```

```go
// internal/jobs/runner_linux.go
//go:build linux

package jobs

import "github.com/valve-tech/jumpgate/internal/executor"

// DefaultRunner is systemd-run. Supported still checks this box boots with
// systemd: a container or a WSL distro without it is refused, not half-run.
func DefaultRunner(ex executor.Executor) Runner { return SystemdRunner{Exec: ex} }
```

```go
// internal/jobs/runner_other.go
//go:build !linux

package jobs

import (
	"context"
	"fmt"
	"runtime"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// The agent is Linux-only by design; macOS keeps it only so its tests run
// there (platform spec D19), and Windows refuses `jumpgate agent` outright.
// Off Linux every job kind is refused with this reason (D-J9).
type unsupportedRunner struct{ goos string }

func DefaultRunner(executor.Executor) Runner { return unsupportedRunner{goos: runtime.GOOS} }

func (u unsupportedRunner) Supported(context.Context) error {
	return fmt.Errorf("%w: durable jobs run only on Linux boxes with systemd; this agent runs on %s", ErrUnsupported, u.goos)
}

func (u unsupportedRunner) Start(ctx context.Context, _ string, _ []string) error { return u.Supported(ctx) }
func (unsupportedRunner) Active(context.Context, string) (bool, error)        { return false, nil }
```

```go
// internal/jobs/jobstest/jobstest.go

// Package jobstest runs durable jobs in-process for the agent's and the
// server's tests: a Runner whose units are goroutines.
package jobstest

import (
	"context"
	"sync"
)

// Runner runs each step's argv through Run in a goroutine. A unit is active
// until Run returns, or until Lose says the box rebooted under it.
type Runner struct {
	Run         func(ctx context.Context, unit string, argv []string) error
	Unsupported error

	mu      sync.Mutex
	active  map[string]bool
	started []string
	wg      sync.WaitGroup
}

func (r *Runner) Supported(context.Context) error { return r.Unsupported }

func (r *Runner) Start(_ context.Context, unit string, argv []string) error {
	r.mu.Lock()
	if r.active == nil {
		r.active = map[string]bool{}
	}
	r.active[unit] = true
	r.started = append(r.started, unit)
	r.mu.Unlock()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if r.Run != nil {
			_ = r.Run(context.Background(), unit, argv)
		}
		r.mu.Lock()
		delete(r.active, unit)
		r.mu.Unlock()
	}()
	return nil
}

func (r *Runner) Active(_ context.Context, unit string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active[unit], nil
}

// Wait waits for every started unit's goroutine.
func (r *Runner) Wait() { r.wg.Wait() }

func (r *Runner) Started() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.started...)
}

// Lose marks unit finished without running anything more: a reboot.
func (r *Runner) Lose(unit string) {
	r.mu.Lock()
	delete(r.active, unit)
	r.mu.Unlock()
}
```

- [ ] **Step 4: Run the tests on the host and cross-compile**

Run: `go test ./internal/jobs/... -count=1 -race && GOOS=linux go vet ./internal/jobs/... && GOOS=windows go vet ./internal/jobs/...`
Expected: PASS, and both vets clean.

- [ ] **Step 5: Commit**

```bash
git add internal/jobs/runner.go internal/jobs/runner_linux.go internal/jobs/runner_other.go \
  internal/jobs/runner_test.go internal/jobs/runner_other_test.go internal/jobs/jobstest/jobstest.go
git commit -m "feat(jobs): systemd-run step runner; jobs refused off Linux and without systemd

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 7: The engine — plans, `RunStep`, and `Manager` (start, lease, advance, decide, reconcile)

Covers spec "Autonomy … decision point, published as a signed question with options", "Adoption", "v1 driver is step-level", "The agent refuses to resume a job whose plan version it does not know", Q7, Q9, J1, J4. Review Focus 2 and 3.

**Files:**
- Create: `internal/jobs/plan.go`, `internal/jobs/step.go`, `internal/jobs/manager.go`, `internal/jobs/manager_test.go`, `internal/jobs/step_test.go`, `internal/jobs/fakes_test.go`
- Modify: `internal/jobs/jobstest/jobstest.go` (the test plan and `InProcess`)

**Interfaces:**
- Consumes: Tasks 3–6.
- Produces (package `jobs`):
  - `type Env struct { Exec executor.Executor; Inputs json.RawMessage; Results map[string]json.RawMessage; Answer *Answer; NodePath, NamesPath string; Out io.Writer }`;
  - `type Outcome struct { Result json.RawMessage; Ask *intent.JobQuestion }`;
  - `type StepDef struct { ID, Title, AbortNote string; Run func(ctx context.Context, env Env) (Outcome, error) }`;
  - `type Plan struct { Kind, Version, Title string; Steps []StepDef; Validate func(inputs json.RawMessage) error }`;
  - `func Register(p Plan)`, `func Unregister(version string)` (tests only), `func PlanFor(version string) (Plan, bool)`, `func PlanForKind(kind string) (Plan, bool)`, `func Versions() []string`;
  - `type StepArgs struct { Job, Step string; Attempt int }`, `func StepArgv(self, stateDir, configDir string, a StepArgs) []string`, `func ParseStepArgs(argv []string) (StepArgs, error)`;
  - `type StepResult struct { OK bool; Error string; Result json.RawMessage; Ask *intent.JobQuestion; FinishedAt int64 }`, `func RunStep(ctx context.Context, dir string, a StepArgs, base Env) error`;
  - `type ManagerConfig struct { Dir string; Key signer.Signer; Runner Runner; StepArgv func(StepArgs) []string; Now func() time.Time; Version string }`;
  - `func NewManager(cfg ManagerConfig) (*Manager, error)`;
  - methods `Start(ctx, kind string, inputs []byte, who eip712.Address, env intent.Envelope) (intent.JobRecord, error)`, `List() (intent.JobListResult, error)`, `Read(id string, from uint64) (intent.JobReadResult, error)`, `Lease(ctx, id, op string, who eip712.Address, env intent.Envelope) (intent.JobRecord, error)`, `Advance(ctx, id string, step, attempt int, who eip712.Address, env intent.Envelope) (intent.JobRecord, error)`, `Decide(ctx, id, decision, option string, who eip712.Address, signers []eip712.Address, env intent.Envelope) (intent.JobRecord, error)`, `QuestionTier(id, decision string) string`, `Active() (intent.JobRecord, bool)`, `Reconcile(ctx context.Context) error`, `Prune() error`.
- Produces (package `jobstest`): `const TestKind = "test.steps"`; `func RegisterPlan(t testing.TB)` (registers `test.steps/1` with steps `one`, `two`; inputs `{"failAt": step, "askAt": step}`; unregisters on cleanup); `func InProcess(dir string, base jobs.Env) func(ctx context.Context, unit string, argv []string) error`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/jobs/fakes_test.go
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// stepRunner is an in-package Runner: jobstest imports jobs, so jobs' own
// tests cannot import it. Start runs the step synchronously unless hold is
// set, so a test decides when a unit "finishes".
type stepRunner struct {
	mu      sync.Mutex
	dir     string
	active  map[string]bool
	argv    map[string][]string
	hold    bool
	started []string
	fail    error
}

func (r *stepRunner) Supported(context.Context) error { return nil }
func (r *stepRunner) Start(ctx context.Context, unit string, argv []string) error {
	if r.fail != nil {
		return r.fail
	}
	r.mu.Lock()
	r.started = append(r.started, unit)
	r.active[unit], r.argv[unit] = true, argv
	r.mu.Unlock()
	if !r.hold {
		r.finish(ctx, unit)
	}
	return nil
}
func (r *stepRunner) Active(_ context.Context, unit string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active[unit], nil
}

// finish runs the unit's step to completion, as the transient unit would.
func (r *stepRunner) finish(ctx context.Context, unit string) {
	r.mu.Lock()
	argv := r.argv[unit]
	r.mu.Unlock()
	a, _ := ParseStepArgs(argv)
	_ = RunStep(ctx, r.dir, a, Env{Out: io.Discard})
	r.mu.Lock()
	delete(r.active, unit)
	r.mu.Unlock()
}

// lose ends a unit without a result: a reboot.
func (r *stepRunner) lose(unit string) {
	r.mu.Lock()
	delete(r.active, unit)
	r.mu.Unlock()
}

// twoSteps is the engine's test plan. Inputs say which step fails or asks.
func twoSteps(t *testing.T) Plan {
	t.Helper()
	type in struct{ FailAt, AskAt string }
	step := func(id string) StepDef {
		return StepDef{ID: id, Title: "Step " + id, AbortNote: "nothing after " + id + " ran",
			Run: func(ctx context.Context, env Env) (Outcome, error) {
				var i in
				_ = json.Unmarshal(env.Inputs, &i)
				if i.AskAt == id && env.Answer == nil {
					return Outcome{Ask: &intent.JobQuestion{Key: "pick", Prompt: "pick one", Tier: intent.TierRoutine,
						Options: []intent.JobOption{{ID: "go", Label: "Go on"}, {ID: intent.OptAbort, Label: "Stop"}}}}, nil
				}
				if i.FailAt == id && env.Answer == nil {
					return Outcome{}, errors.New(id + " broke")
				}
				return Outcome{Result: json.RawMessage(fmt.Sprintf(`{"did":%q}`, id))}, nil
			}}
	}
	p := Plan{Kind: "test.two", Version: "test.two/1", Title: "Two steps", Steps: []StepDef{step("one"), step("two")}}
	Register(p)
	t.Cleanup(func() { Unregister(p.Version) })
	return p
}

type rig struct {
	m      *Manager
	r      *stepRunner
	a, b   *signer.Key
	now    time.Time
	dir    string
}

func newRig(t *testing.T) *rig {
	t.Helper()
	testutil.RequireUnix(t)
	twoSteps(t)
	dir := filepath.Join(t.TempDir(), "jobs")
	g := &rig{dir: dir, now: time.Unix(1_800_000_000, 0)}
	g.r = &stepRunner{dir: dir, active: map[string]bool{}, argv: map[string][]string{}}
	g.a, _ = signer.GenerateKey()
	g.b, _ = signer.GenerateKey()
	key, _ := signer.GenerateKey()
	var err error
	g.m, err = NewManager(ManagerConfig{Dir: dir, Key: key, Runner: g.r, Now: func() time.Time { return g.now },
		StepArgv: func(a StepArgs) []string { return StepArgv("/jumpgate", "/state", "/etc/jumpgate", a) }, Version: "v-test"})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func env() intent.Envelope { return intent.Envelope{} }

// advance moves the job one step as the lease holder would, then reconciles.
func (g *rig) advance(t *testing.T, who eip712.Address, id string) intent.JobRecord {
	t.Helper()
	rec, _ := g.read(t, id)
	if _, err := g.m.Advance(context.Background(), id, rec.Step, rec.Attempt, who, env()); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, _ = g.read(t, id)
	return rec
}

func (g *rig) read(t *testing.T, id string) (intent.JobRecord, intent.JobReadResult) {
	t.Helper()
	res, err := g.m.Read(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	return res.Job, res
}
```

```go
// internal/jobs/manager_test.go
package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
)

func refusalCode(err error) string {
	var r *Refusal
	if errors.As(err, &r) {
		return r.Code
	}
	return ""
}

func TestAJobRunsStepByStepAndSucceeds(t *testing.T) {
	g := newRig(t)
	rec, err := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env())
	if err != nil || rec.State != intent.JobWaiting || rec.Lease.Owner != g.a.Address().Hex() || rec.Driver != DriverController {
		t.Fatalf("Start = %+v, %v", rec, err)
	}
	if rec = g.advance(t, g.a.Address(), rec.ID); rec.State != intent.JobWaiting || rec.Step != 1 {
		t.Fatalf("after one step: %+v", rec)
	}
	if rec = g.advance(t, g.a.Address(), rec.ID); rec.State != intent.JobSucceeded || string(rec.Result) != `{"did":"two"}` {
		t.Fatalf("after two steps: %+v", rec)
	}
	_, res := g.read(t, rec.ID)
	var kinds []string
	for _, j := range res.Entries {
		kinds = append(kinds, j.Kind)
	}
	want := []string{joblog.KindCreated, joblog.KindIntent, joblog.KindStepStarted, joblog.KindStepFinished,
		joblog.KindIntent, joblog.KindStepStarted, joblog.KindStepFinished, joblog.KindFinished}
	if len(kinds) != len(want) {
		t.Fatalf("log kinds %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("log kinds %v, want %v", kinds, want)
		}
	}
}

// J1.
func TestOneJobAtATimePerBox(t *testing.T) {
	g := newRig(t)
	_, _ = g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env())
	if _, err := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.b.Address(), env()); refusalCode(err) != intent.ReasonJobActive {
		t.Fatalf("second Start = %v", err)
	}
}

// Review Focus 3.
func TestLeaseFreshRefusesAnotherController(t *testing.T) {
	g := newRig(t)
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env())
	if _, err := g.m.Advance(context.Background(), rec.ID, 0, 0, g.b.Address(), env()); refusalCode(err) != intent.ReasonLeaseHeld {
		t.Fatalf("advance by a non-holder = %v", err)
	}
	if _, err := g.m.Lease(context.Background(), rec.ID, intent.LeaseTake, g.b.Address(), env()); refusalCode(err) != intent.ReasonLeaseHeld {
		t.Fatalf("take of a fresh lease = %v", err)
	}
	g.now = g.now.Add(LeaseTTL + time.Second)
	got, err := g.m.Lease(context.Background(), rec.ID, intent.LeaseTake, g.b.Address(), env())
	if err != nil || got.Lease.Owner != g.b.Address().Hex() {
		t.Fatalf("adopting a stale lease: %+v, %v", got.Lease, err)
	}
	// Adoption is "rerun the plan" from where the job stands.
	if got = g.advance(t, g.b.Address(), rec.ID); got.Step != 1 {
		t.Fatalf("the adopter could not drive: %+v", got)
	}
	if _, err := g.m.Lease(context.Background(), rec.ID, intent.LeaseRenew, g.a.Address(), env()); refusalCode(err) != intent.ReasonLeaseHeld {
		t.Fatalf("the old holder renewed: %v", err)
	}
}

func TestAStaleAdvanceIsRefused(t *testing.T) {
	g := newRig(t)
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env())
	g.advance(t, g.a.Address(), rec.ID)
	if _, err := g.m.Advance(context.Background(), rec.ID, 0, 0, g.a.Address(), env()); refusalCode(err) != intent.ReasonJobState {
		t.Fatalf("an advance for a step already done = %v", err)
	}
}

func TestAFailedStepAsksAndRetryRunsTheNextAttempt(t *testing.T) {
	g := newRig(t)
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{"FailAt":"one"}`), g.a.Address(), env())
	rec = g.advance(t, g.a.Address(), rec.ID)
	if rec.State != intent.JobNeedsDecision || rec.Question == nil || rec.Question.Tier != intent.TierRoutine {
		t.Fatalf("after a failure: %+v", rec)
	}
	if _, err := g.m.Decide(context.Background(), rec.ID, "wrong-id", intent.OptRetry, g.a.Address(), nil, env()); refusalCode(err) != intent.ReasonJobState {
		t.Fatalf("an answer to another question = %v", err)
	}
	rec, err := g.m.Decide(context.Background(), rec.ID, rec.Question.ID, intent.OptRetry, g.a.Address(), nil, env())
	if err != nil || rec.State != intent.JobWaiting || rec.Attempt != 1 {
		t.Fatalf("after retry: %+v, %v", rec, err)
	}
	if rec = g.advance(t, g.a.Address(), rec.ID); rec.Step != 1 {
		t.Fatalf("the retry did not pass: %+v", rec)
	}
	if got := g.r.started; len(got) != 2 || got[1] != UnitName(rec.ID, "one", 1) {
		t.Fatalf("units %v", got)
	}
}

func TestAStepQuestionReachesTheNextAttempt(t *testing.T) {
	g := newRig(t)
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{"AskAt":"two"}`), g.a.Address(), env())
	g.advance(t, g.a.Address(), rec.ID)
	rec = g.advance(t, g.a.Address(), rec.ID)
	if rec.Question == nil || rec.Question.Key != "pick" || rec.Question.Step != "two" {
		t.Fatalf("question %+v", rec.Question)
	}
	rec, _ = g.m.Decide(context.Background(), rec.ID, rec.Question.ID, "go", g.a.Address(), nil, env())
	if rec = g.advance(t, g.a.Address(), rec.ID); rec.State != intent.JobSucceeded {
		t.Fatalf("the answered step did not finish: %+v", rec)
	}
}

func TestAbortEndsTheJob(t *testing.T) {
	g := newRig(t)
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{"FailAt":"two"}`), g.a.Address(), env())
	g.advance(t, g.a.Address(), rec.ID)
	rec = g.advance(t, g.a.Address(), rec.ID)
	rec, err := g.m.Decide(context.Background(), rec.ID, rec.Question.ID, intent.OptAbort, g.a.Address(), nil, env())
	if err != nil || rec.State != intent.JobAborted {
		t.Fatalf("abort: %+v, %v", rec, err)
	}
	if _, ok := g.m.Active(); ok {
		t.Fatal("an aborted job still counts as active")
	}
}

// Review Focus 2: the step finished while the agent was down; the next agent
// reconciles it exactly once.
func TestReconcileFoldsAResultWrittenWhileTheAgentWasDown(t *testing.T) {
	g := newRig(t)
	g.r.hold = true
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env())
	_, _ = g.m.Advance(context.Background(), rec.ID, 0, 0, g.a.Address(), env())
	g.r.finish(context.Background(), UnitName(rec.ID, "one", 0)) // while "the agent" is away

	key2 := g.m.cfg.Key // the same box key; a new Manager is a restarted agent
	m2, err := NewManager(ManagerConfig{Dir: g.dir, Key: key2, Runner: g.r, Now: func() time.Time { return g.now }, StepArgv: g.m.cfg.StepArgv})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := m2.Reconcile(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	res, _ := m2.Read(rec.ID, 0)
	finished := 0
	for _, e := range res.Entries {
		if e.Kind == joblog.KindStepFinished {
			finished++
		}
	}
	if finished != 1 || res.Job.Step != 1 || res.Job.State != intent.JobWaiting {
		t.Fatalf("step.finished x%d, job %+v", finished, res.Job)
	}
}

// Review Focus 2 / Q9: a unit gone with no result (a reboot) is never rerun
// silently.
func TestLostStepAsksBeforeRerunning(t *testing.T) {
	g := newRig(t)
	g.r.hold = true
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env())
	_, _ = g.m.Advance(context.Background(), rec.ID, 0, 0, g.a.Address(), env())
	g.r.lose(UnitName(rec.ID, "one", 0))
	_ = g.m.Reconcile(context.Background())
	rec, res := g.read(t, rec.ID)
	if rec.State != intent.JobNeedsDecision || len(g.r.started) != 1 {
		t.Fatalf("a lost step: %+v, units %v", rec, g.r.started)
	}
	if last := res.Entries[len(res.Entries)-1]; last.Kind != joblog.KindDecisionAsked {
		t.Fatalf("last entry %s", last.Kind)
	}
}

func TestAStepThatCannotStartAsks(t *testing.T) {
	g := newRig(t)
	g.r.fail = errors.New("Unit already exists")
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env())
	rec, err := g.m.Advance(context.Background(), rec.ID, 0, 0, g.a.Address(), env())
	if err != nil || rec.State != intent.JobNeedsDecision {
		t.Fatalf("advance with a failing runner: %+v, %v", rec, err)
	}
}

// The agent refuses to resume a job whose plan it does not know (spec).
func TestUnknownPlanCanBeReadButNotAdvanced(t *testing.T) {
	g := newRig(t)
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env())
	Unregister("test.two/1") // a downgraded agent
	if _, err := g.m.Read(rec.ID, 0); err != nil {
		t.Fatalf("read of an unknown-plan job: %v", err)
	}
	if _, err := g.m.Advance(context.Background(), rec.ID, 0, 0, g.a.Address(), env()); refusalCode(err) != intent.ReasonUnknownPlan {
		t.Fatalf("advance = %v", err)
	}
}

func TestStartRefusesWhenTheRunnerIsUnsupported(t *testing.T) {
	g := newRig(t)
	g.m.cfg.Runner = refusingRunner{}
	if _, err := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), env()); refusalCode(err) != intent.ReasonJobsUnsupported {
		t.Fatalf("Start = %v", err)
	}
}

type refusingRunner struct{}

func (refusingRunner) Supported(context.Context) error { return ErrUnsupported }
func (refusingRunner) Start(context.Context, string, []string) error {
	return ErrUnsupported
}
func (refusingRunner) Active(context.Context, string) (bool, error) { return false, nil }

// QuestionTier lets the agent demand an approval signature before admission.
func TestQuestionTierIsTheOpenQuestions(t *testing.T) {
	g := newRig(t)
	rec, _ := g.m.Start(context.Background(), "test.two", []byte(`{"FailAt":"one"}`), g.a.Address(), env())
	rec = g.advance(t, g.a.Address(), rec.ID)
	if got := g.m.QuestionTier(rec.ID, rec.Question.ID); got != intent.TierRoutine {
		t.Fatalf("tier %q", got)
	}
	if got := g.m.QuestionTier(rec.ID, "nope"); got != intent.TierRoutine {
		t.Fatalf("an unknown question's tier %q; Decide refuses it later", got)
	}
}
```

```go
// internal/jobs/step_test.go
package jobs

import (
	"reflect"
	"testing"
)

func TestStepArgvRoundTrips(t *testing.T) {
	a := StepArgs{Job: "0123456789abcdef", Step: "rename-account", Attempt: 3}
	argv := StepArgv("/usr/local/lib/jumpgate/jumpgate", "/var/lib/jumpgate", "/etc/jumpgate", a)
	if argv[1] != "agent" || argv[2] != "job-step" {
		t.Fatalf("argv %v", argv)
	}
	got, err := ParseStepArgs(argv)
	if err != nil || !reflect.DeepEqual(got, a) {
		t.Fatalf("ParseStepArgs = %+v, %v", got, err)
	}
	if _, err := ParseStepArgs([]string{"--job", "../x", "--step", "one", "--attempt", "0"}); err == nil {
		t.Fatal("a bad job id parsed")
	}
}

// A step run by hand (or by a stale unit) out of turn writes nothing.
func TestRunStepRefusesAStepTheAgentDidNotAskFor(t *testing.T) {
	g := newRig(t)
	rec, _ := g.m.Start(t.Context(), "test.two", []byte(`{}`), g.a.Address(), env())
	if err := RunStep(t.Context(), g.dir, StepArgs{Job: rec.ID, Step: "two", Attempt: 0}, Env{}); err == nil {
		t.Fatal("step two ran while the job stands at step one")
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/jobs/ -count=1`
Expected: FAIL to compile (undefined `Register`, `NewManager`, `RunStep`, …).

- [ ] **Step 3: Implement `plan.go` and `step.go`**

```go
// internal/jobs/plan.go
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// Env is what a step runs with, on the box, as root, in its transient unit.
type Env struct {
	Exec      executor.Executor
	Inputs    json.RawMessage
	Results   map[string]json.RawMessage // earlier steps' results, by step id
	Answer    *Answer                    // the answer to this step's last question, if the previous attempt asked one
	NodePath  string                     // /etc/jumpgate/node.json
	NamesPath string                     // /etc/jumpgate/names.json
	Out       io.Writer                  // the unit's stdout: the journal
}

// Outcome is a step's answer. Ask stops the job at a decision point; the step
// runs again (next attempt) with Env.Answer set once someone decides.
type Outcome struct {
	Result json.RawMessage
	Ask    *intent.JobQuestion
}

// StepDef is one step. Run must be idempotent: adoption reruns the plan from
// the step the job stands at, and Run may find its work already done.
// AbortNote says what state the box is left in if the job is aborted at
// this step (Q8); it is shown with the abort option.
type StepDef struct {
	ID, Title, AbortNote string
	Run                  func(ctx context.Context, env Env) (Outcome, error)
}

// Plan is a job kind's steps at one version. A plan's steps never change
// within a version: a manifest records the version, and the agent refuses
// to advance a version it does not know.
type Plan struct {
	Kind, Version, Title string
	Steps                []StepDef
	Validate             func(inputs json.RawMessage) error
}

var (
	plansMu sync.RWMutex
	plans   = map[string]Plan{} // by version
)

// Register adds a plan; plans register in init. A duplicate or a bad step id
// panics: it is a programming error.
func Register(p Plan) {
	plansMu.Lock()
	defer plansMu.Unlock()
	if _, dup := plans[p.Version]; dup {
		panic("jobs: plan " + p.Version + " registered twice")
	}
	for _, s := range p.Steps {
		if !ValidStepID(s.ID) {
			panic(fmt.Sprintf("jobs: plan %s: bad step id %q", p.Version, s.ID))
		}
	}
	plans[p.Version] = p
}

// Unregister is for tests that register their own plans.
func Unregister(version string) {
	plansMu.Lock()
	delete(plans, version)
	plansMu.Unlock()
}

func PlanFor(version string) (Plan, bool) {
	plansMu.RLock()
	defer plansMu.RUnlock()
	p, ok := plans[version]
	return p, ok
}

// PlanForKind is the newest registered version of kind. v1 has one version
// per kind; when a second appears, new jobs take the newest and running jobs
// keep theirs.
func PlanForKind(kind string) (Plan, bool) {
	plansMu.RLock()
	defer plansMu.RUnlock()
	var best Plan
	found := false
	for _, p := range plans {
		if p.Kind == kind && (!found || p.Version > best.Version) {
			best, found = p, true
		}
	}
	return best, found
}

// Versions are the plan versions this agent runs, for agent.info.
func Versions() []string {
	plansMu.RLock()
	defer plansMu.RUnlock()
	out := make([]string, 0, len(plans))
	for v := range plans {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func (p Plan) index(step string) int {
	for i, s := range p.Steps {
		if s.ID == step {
			return i
		}
	}
	return -1
}

func (p Plan) stepInfos() []intent.JobStep {
	out := make([]intent.JobStep, len(p.Steps))
	for i, s := range p.Steps {
		out[i] = intent.JobStep{ID: s.ID, Title: s.Title}
	}
	return out
}
```

```go
// internal/jobs/step.go
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/valve-tech/jumpgate/internal/fsatomic"
)

// StepArgs names one attempt of one step of one job.
type StepArgs struct {
	Job, Step string
	Attempt   int
}

// StepArgv is the command line of a step's unit: the agent's own binary.
func StepArgv(self, stateDir, configDir string, a StepArgs) []string {
	return []string{self, "agent", "job-step", "--state-dir", stateDir, "--config-dir", configDir,
		"--job", a.Job, "--step", a.Step, "--attempt", strconv.Itoa(a.Attempt)}
}

// ParseStepArgs reads --job, --step and --attempt from anywhere in argv.
func ParseStepArgs(argv []string) (StepArgs, error) {
	var a StepArgs
	attempt := ""
	for i := 0; i+1 < len(argv); i++ {
		switch argv[i] {
		case "--job":
			a.Job = argv[i+1]
		case "--step":
			a.Step = argv[i+1]
		case "--attempt":
			attempt = argv[i+1]
		}
	}
	n, err := strconv.Atoi(attempt)
	if err != nil || n < 0 || n > 9999 || !ValidID(a.Job) || !ValidStepID(a.Step) {
		return a, fmt.Errorf("jobs: bad step arguments %q", argv)
	}
	a.Attempt = n
	return a, nil
}

// StepResult is a step's result file: the only thing a step process writes.
type StepResult struct {
	OK         bool                `json:"ok"`
	Error      string              `json:"error,omitempty"`
	Result     json.RawMessage     `json:"result,omitempty"`
	Ask        *intent.JobQuestion `json:"ask,omitempty"`
	FinishedAt int64               `json:"finishedAt"`
}

// RunStep is `jumpgate agent job-step`: it runs one attempt and writes its
// result file. It never touches the manifest or the log (D-J4). A step that
// fails still writes a result and returns nil, so the unit exits 0 and the
// result file alone says how the step ended.
func RunStep(ctx context.Context, dir string, a StepArgs, base Env) error {
	m, err := LoadManifest(dir, a.Job)
	if err != nil {
		return err
	}
	plan, ok := PlanFor(m.PlanVersion)
	if !ok {
		return fmt.Errorf("jobs: this agent does not know plan %s", m.PlanVersion)
	}
	idx := plan.index(a.Step)
	if idx < 0 || idx != m.Step || a.Attempt != m.Attempt || m.State != intent.JobRunning && m.State != intent.JobWaiting {
		return errors.New("jobs: the agent did not ask for this step; refusing to run it out of turn")
	}
	env := base
	env.Inputs, env.Results = m.Inputs, m.Results
	if m.Answer != nil && m.Answer.Step == idx && m.Answer.Attempt == a.Attempt-1 {
		env.Answer = m.Answer
	}
	if env.Out != nil {
		fmt.Fprintf(env.Out, "jumpgate job %s: step %s (%s), attempt %d\n", a.Job, a.Step, plan.Steps[idx].Title, a.Attempt)
	}
	out, runErr := plan.Steps[idx].Run(ctx, env)
	res := StepResult{OK: runErr == nil, Result: out.Result, Ask: out.Ask, FinishedAt: time.Now().Unix()}
	if runErr != nil {
		res.Error = runErr.Error()
		if env.Out != nil {
			fmt.Fprintln(env.Out, "step failed:", runErr)
		}
	}
	b, _ := json.Marshal(res)
	return fsatomic.WriteFile(ResultPath(dir, a.Job, a.Step, a.Attempt), b)
}
```

(Import `intent` in `step.go`.) `RunStep` accepts `JobWaiting` as well as `JobRunning`: `Advance` starts the unit before it saves `running` (see below), so a fast step may read the manifest first.

- [ ] **Step 4: Implement `manager.go`**

```go
// internal/jobs/manager.go
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/signer"
)

const (
	readPageMax = 500
	pruneAge    = 30 * 24 * time.Hour
	pruneKeep   = 100
)

type ManagerConfig struct {
	Dir      string
	Key      signer.Signer
	Runner   Runner
	StepArgv func(StepArgs) []string
	Now      func() time.Time
	Version  string // the agent's build, stamped into each manifest
}

// Manager is every job operation the agent performs. Its mutex makes it the
// single writer of manifests, between concurrent intents and the reconcile loop.
type Manager struct {
	cfg   ManagerConfig
	store *Store
	mu    sync.Mutex
}

func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	s, err := OpenStore(cfg.Dir, cfg.Key, cfg.Now)
	if err != nil {
		return nil, err
	}
	m := &Manager{cfg: cfg, store: s}
	return m, nil
}

func (m *Manager) lock()   { m.mu.Lock() }
func (m *Manager) unlock() { m.mu.Unlock() }

// load reads a job for a change: known format, driver and plan, or refused.
func (m *Manager) load(id string) (Manifest, Plan, error) {
	man, err := m.store.Load(id)
	if errors.Is(err, ErrNotFound) {
		return man, Plan{}, refuse(intent.ReasonJobNotFound, "no job "+id+" on this box")
	}
	if errors.Is(err, ErrDamaged) {
		return man, Plan{}, refuse(intent.ReasonJobState, err.Error())
	}
	if err != nil {
		return man, Plan{}, err
	}
	plan, ok := PlanFor(man.PlanVersion)
	if man.Format != FormatVersion || man.Driver != DriverController || !ok {
		return man, plan, refuse(intent.ReasonUnknownPlan, fmt.Sprintf(
			"job %s is format %d, driver %q, plan %s; this agent runs format %d, driver %q and plans %v",
			id, man.Format, man.Driver, man.PlanVersion, FormatVersion, DriverController, Versions()))
	}
	return man, plan, nil
}

func (m *Manager) activeLocked() (Manifest, bool) {
	ms, err := m.store.List()
	if err != nil {
		return Manifest{}, false
	}
	for _, j := range ms {
		if !intent.JobTerminal(j.State) {
			return j, true
		}
	}
	return Manifest{}, false
}

// Active is the box's unfinished job, if any (J1).
func (m *Manager) Active() (intent.JobRecord, bool) {
	m.lock()
	defer m.unlock()
	j, ok := m.activeLocked()
	return j.JobRecord, ok
}

func (m *Manager) Start(ctx context.Context, kind string, inputs []byte, who eip712.Address, env intent.Envelope) (intent.JobRecord, error) {
	plan, ok := PlanForKind(kind)
	if !ok {
		return intent.JobRecord{}, refuse(intent.ReasonUnknownKind, "this agent has no plan for "+kind)
	}
	if err := m.cfg.Runner.Supported(ctx); err != nil {
		return intent.JobRecord{}, refuse(intent.ReasonJobsUnsupported, err.Error())
	}
	if !json.Valid(inputs) {
		return intent.JobRecord{}, refuse(intent.ReasonInvalidPayload, "the job's inputs are not JSON")
	}
	if plan.Validate != nil {
		if err := plan.Validate(inputs); err != nil {
			return intent.JobRecord{}, refuse(intent.ReasonInvalidPayload, err.Error())
		}
	}
	m.lock()
	defer m.unlock()
	if j, busy := m.activeLocked(); busy {
		return intent.JobRecord{}, refuse(intent.ReasonJobActive, fmt.Sprintf("job %s (%s) is %s on this box; one job runs at a time", j.ID, j.Kind, j.State))
	}
	id, err := NewID()
	if err != nil {
		return intent.JobRecord{}, err
	}
	now := m.cfg.Now()
	man := Manifest{JobRecord: intent.JobRecord{Format: FormatVersion, ID: id, Kind: kind, Title: plan.Title,
		PlanVersion: plan.Version, Driver: DriverController, Steps: plan.stepInfos(), State: intent.JobWaiting,
		CreatedBy: who.Hex(), CreatedAt: now.Unix(), UpdatedAt: now.Unix(), AgentVersion: m.cfg.Version},
		Inputs: json.RawMessage(inputs)}
	_, _, _ = man.TakeLease(who, now) // a new job is its creator's to drive
	if err := m.store.Create(ctx, &man, joblog.IntentBody{Envelope: env, Outcome: "created"}); err != nil {
		return intent.JobRecord{}, err
	}
	return man.JobRecord, nil
}

func (m *Manager) List() (intent.JobListResult, error) {
	m.lock()
	defer m.unlock()
	ms, err := m.store.List()
	if err != nil {
		return intent.JobListResult{}, err
	}
	out := intent.JobListResult{Jobs: []intent.JobRecord{}, AgentTime: m.cfg.Now().Unix()}
	for _, j := range ms {
		out.Jobs = append(out.Jobs, j.JobRecord)
	}
	return out, nil
}

func (m *Manager) Read(id string, from uint64) (intent.JobReadResult, error) {
	m.lock()
	defer m.unlock()
	man, err := m.store.Load(id)
	if errors.Is(err, ErrNotFound) {
		return intent.JobReadResult{}, refuse(intent.ReasonJobNotFound, "no job "+id+" on this box")
	}
	if err != nil && !errors.Is(err, ErrDamaged) {
		return intent.JobReadResult{}, err
	}
	if errors.Is(err, ErrDamaged) {
		man.State, man.Error = intent.JobDamaged, err.Error()
	}
	es, more, rerr := m.store.ReadLog(id, from, readPageMax)
	if rerr != nil && err == nil {
		return intent.JobReadResult{}, rerr
	}
	out := intent.JobReadResult{Job: man.JobRecord, Entries: []intent.JobLogEntryJSON{}, More: more, AgentTime: m.cfg.Now().Unix()}
	for _, e := range es {
		out.Entries = append(out.Entries, joblog.ToJSON(e))
	}
	return out, nil
}

func (m *Manager) Lease(ctx context.Context, id, op string, who eip712.Address, env intent.Envelope) (intent.JobRecord, error) {
	m.lock()
	defer m.unlock()
	man, _, err := m.load(id)
	var r *Refusal
	if errors.As(err, &r) && r.Code == intent.ReasonUnknownPlan {
		err = nil // a lease on a job this agent cannot advance is still a lease; it lets a controller watch and decide nothing
	}
	if err != nil {
		return intent.JobRecord{}, err
	}
	now := m.cfg.Now()
	switch op {
	case intent.LeaseTake:
		changed, prev, err := man.TakeLease(who, now)
		if err != nil {
			return intent.JobRecord{}, err
		}
		if changed {
			if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: "lease take"}); err != nil {
				return intent.JobRecord{}, err
			}
			if err := m.store.Append(ctx, &man, joblog.KindLeaseTaken, joblog.LeaseBody{Owner: who.Hex(), Previous: prev}); err != nil {
				return intent.JobRecord{}, err
			}
		}
	case intent.LeaseRenew:
		if err := man.RenewLease(who, now); err != nil {
			return intent.JobRecord{}, err
		}
	case intent.LeaseRelease:
		if err := man.ReleaseLease(who); err != nil {
			return intent.JobRecord{}, err
		}
		if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: "lease release"}); err != nil {
			return intent.JobRecord{}, err
		}
		if err := m.store.Append(ctx, &man, joblog.KindLeaseReleased, joblog.LeaseBody{Owner: who.Hex()}); err != nil {
			return intent.JobRecord{}, err
		}
	default:
		return intent.JobRecord{}, refuse(intent.ReasonInvalidPayload, fmt.Sprintf("lease op %q", op))
	}
	return man.JobRecord, m.store.Save(&man)
}

func (m *Manager) Advance(ctx context.Context, id string, step, attempt int, who eip712.Address, env intent.Envelope) (intent.JobRecord, error) {
	m.lock()
	defer m.unlock()
	man, plan, err := m.load(id)
	if err != nil {
		return intent.JobRecord{}, err
	}
	if err := man.Holds(who); err != nil {
		return intent.JobRecord{}, err
	}
	if man.State != intent.JobWaiting || man.Step != step || man.Attempt != attempt {
		return intent.JobRecord{}, refuse(intent.ReasonJobState, fmt.Sprintf(
			"job %s is %s at step %d attempt %d; this advance was for step %d attempt %d", id, man.State, man.Step, man.Attempt, step, attempt))
	}
	_ = man.RenewLease(who, m.cfg.Now())
	s := plan.Steps[man.Step]
	unit := UnitName(id, s.ID, man.Attempt)
	if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: "advance to " + s.ID}); err != nil {
		return intent.JobRecord{}, err
	}
	if err := m.cfg.Runner.Start(ctx, unit, m.cfg.StepArgv(StepArgs{Job: id, Step: s.ID, Attempt: man.Attempt})); err != nil {
		_ = m.store.Append(ctx, &man, joblog.KindStepFinished, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, Error: "could not start: " + err.Error()})
		if err := m.ask(ctx, &man, failureQuestion(s, man.Attempt, "it could not start: "+err.Error())); err != nil {
			return intent.JobRecord{}, err
		}
		return man.JobRecord, m.store.Save(&man)
	}
	man.State, man.Unit = intent.JobRunning, unit
	if err := m.store.Append(ctx, &man, joblog.KindStepStarted, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit}); err != nil {
		return intent.JobRecord{}, err
	}
	return man.JobRecord, m.store.Save(&man)
}

// QuestionTier is the tier the job's open question needs; routine when there
// is no such question (Decide then refuses it as job_state).
func (m *Manager) QuestionTier(id, decision string) string {
	m.lock()
	defer m.unlock()
	man, err := m.store.Load(id)
	if err != nil || man.Question == nil || man.Question.ID != decision || man.Question.Tier == "" {
		return intent.TierRoutine
	}
	return man.Question.Tier
}

func (m *Manager) Decide(ctx context.Context, id, decision, option string, who eip712.Address, signers []eip712.Address, env intent.Envelope) (intent.JobRecord, error) {
	m.lock()
	defer m.unlock()
	man, plan, err := m.load(id)
	if err != nil {
		return intent.JobRecord{}, err
	}
	if err := man.Holds(who); err != nil {
		return intent.JobRecord{}, err
	}
	q := man.Question
	if man.State != intent.JobNeedsDecision || q == nil || q.ID != decision {
		open := "none"
		if q != nil {
			open = q.ID
		}
		return intent.JobRecord{}, refuse(intent.ReasonJobState, fmt.Sprintf("job %s is not waiting on decision %q (open: %s)", id, decision, open))
	}
	known := false
	for _, o := range q.Options {
		known = known || o.ID == option
	}
	if !known {
		return intent.JobRecord{}, refuse(intent.ReasonInvalidPayload, fmt.Sprintf("%q is not an option of %s", option, decision))
	}
	_ = man.RenewLease(who, m.cfg.Now())
	hexes := make([]string, len(signers))
	for i, s := range signers {
		hexes[i] = s.Hex()
	}
	if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: "decide " + option}); err != nil {
		return intent.JobRecord{}, err
	}
	if err := m.store.Append(ctx, &man, joblog.KindDecisionAnswered, joblog.AnswerBody{Question: decision, Option: option, Controller: who.Hex(), Signers: hexes}); err != nil {
		return intent.JobRecord{}, err
	}
	if man.Decisions == nil {
		man.Decisions = map[string]string{}
	}
	man.Decisions[decision] = option
	man.Question = nil
	if option == intent.OptAbort {
		man.State = intent.JobAborted
		man.Error = fmt.Sprintf("aborted at step %s by %s", plan.Steps[man.Step].ID, who.Hex())
		if err := m.store.Append(ctx, &man, joblog.KindFinished, joblog.FinishedBody{State: man.State, Error: man.Error}); err != nil {
			return intent.JobRecord{}, err
		}
		return man.JobRecord, m.store.Save(&man)
	}
	man.Answer = &Answer{Key: q.Key, Option: option, Step: man.Step, Attempt: man.Attempt}
	man.Attempt++
	man.State = intent.JobWaiting
	return man.JobRecord, m.store.Save(&man)
}

// Reconcile folds every finished step into the log: run on agent start,
// every 5 s, and before each job intent. A unit whose state cannot be read
// is left for the next pass, never guessed at.
func (m *Manager) Reconcile(ctx context.Context) error {
	m.lock()
	defer m.unlock()
	ms, err := m.store.List()
	if err != nil {
		return err
	}
	for _, j := range ms {
		if j.State != intent.JobRunning {
			continue
		}
		man, plan, err := m.load(j.ID)
		if err != nil {
			continue // unknown plan or damage: shown by List, never advanced
		}
		active, err := m.cfg.Runner.Active(ctx, man.Unit)
		if err != nil || active {
			continue
		}
		if err := m.fold(ctx, &man, plan); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) fold(ctx context.Context, man *Manifest, plan Plan) error {
	s := plan.Steps[man.Step]
	unit := man.Unit
	man.Unit = ""
	b, err := os.ReadFile(ResultPath(m.store.Dir(), man.ID, s.ID, man.Attempt))
	var res StepResult
	if err == nil {
		err = json.Unmarshal(b, &res)
	}
	switch {
	case err != nil:
		why := "it stopped without reporting a result (the box may have rebooted, or the step was killed)"
		if !errors.Is(err, fs.ErrNotExist) {
			why = "its result file does not parse: " + err.Error()
		}
		if err := m.store.Append(ctx, man, joblog.KindStepLost, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, Error: why}); err != nil {
			return err
		}
		q := failureQuestion(s, man.Attempt, why)
		q.ID = fmt.Sprintf("%s-%d-lost", s.ID, man.Attempt)
		if err := m.ask(ctx, man, q); err != nil {
			return err
		}
	case !res.OK:
		if err := m.store.Append(ctx, man, joblog.KindStepFinished, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, Error: res.Error}); err != nil {
			return err
		}
		if err := m.ask(ctx, man, failureQuestion(s, man.Attempt, res.Error)); err != nil {
			return err
		}
	case res.Ask != nil:
		if err := m.store.Append(ctx, man, joblog.KindStepFinished, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, OK: true}); err != nil {
			return err
		}
		q := *res.Ask
		q.ID, q.Step = fmt.Sprintf("%s-%d-%s", s.ID, man.Attempt, q.Key), s.ID
		if err := m.ask(ctx, man, q); err != nil {
			return err
		}
	default:
		if err := m.store.Append(ctx, man, joblog.KindStepFinished, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, OK: true}); err != nil {
			return err
		}
		if man.Results == nil {
			man.Results = map[string]json.RawMessage{}
		}
		man.Results[s.ID] = res.Result
		man.Step, man.Attempt, man.Answer = man.Step+1, 0, nil
		man.State = intent.JobWaiting
		if man.Step == len(plan.Steps) {
			man.State, man.Result = intent.JobSucceeded, res.Result
			if err := m.store.Append(ctx, man, joblog.KindFinished, joblog.FinishedBody{State: man.State}); err != nil {
				return err
			}
		}
	}
	return m.store.Save(man)
}

func (m *Manager) ask(ctx context.Context, man *Manifest, q intent.JobQuestion) error {
	if q.Tier == "" {
		q.Tier = intent.TierRoutine
	}
	q.AskedAt = m.cfg.Now().Unix()
	man.Question, man.State = &q, intent.JobNeedsDecision
	return m.store.Append(ctx, man, joblog.KindDecisionAsked, q)
}

// failureQuestion is the decision every failed or lost step stops at: nothing
// reruns without a person (Q9).
func failureQuestion(s StepDef, attempt int, why string) intent.JobQuestion {
	return intent.JobQuestion{ID: fmt.Sprintf("%s-%d-failed", s.ID, attempt), Step: s.ID,
		Prompt: fmt.Sprintf("Step %q failed: %s", s.Title, why), Tier: intent.TierRoutine,
		Options: []intent.JobOption{
			{ID: intent.OptRetry, Label: "Retry the step", Detail: "every step checks what is already done before it acts"},
			{ID: intent.OptAbort, Label: "Stop the job here", Detail: s.AbortNote},
		}}
}

// Prune applies D-J6.
func (m *Manager) Prune() error {
	m.lock()
	defer m.unlock()
	return m.store.Prune(pruneAge, pruneKeep)
}
```

- [ ] **Step 5: Add the test plan to `jobstest`**

```go
// internal/jobs/jobstest/jobstest.go — add
import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
)

// TestKind is the plan RegisterPlan adds. A policy must list it under kinds
// (it has no default requirement), so tests also prove the policy is read.
const TestKind = "test.steps"

// RegisterPlan registers test.steps/1: steps one and two, failing or asking at
// the step its inputs name, once per job (an answer or a retry passes).
func RegisterPlan(t testing.TB) {
	type in struct{ FailAt, AskAt string }
	step := func(id string) jobs.StepDef {
		return jobs.StepDef{ID: id, Title: "Step " + id, AbortNote: "test",
			Run: func(ctx context.Context, env jobs.Env) (jobs.Outcome, error) {
				var i in
				_ = json.Unmarshal(env.Inputs, &i)
				if env.Out != nil {
					fmt.Fprintln(env.Out, "running", id)
				}
				if i.AskAt == id && env.Answer == nil {
					return jobs.Outcome{Ask: &intent.JobQuestion{Key: "pick", Prompt: "pick one", Tier: intent.TierRoutine,
						Options: []intent.JobOption{{ID: "go", Label: "Go on"}, {ID: intent.OptAbort, Label: "Stop"}}}}, nil
				}
				if i.FailAt == id && env.Answer == nil {
					return jobs.Outcome{}, errors.New(id + " broke")
				}
				return jobs.Outcome{Result: json.RawMessage(`{"did":"` + id + `"}`)}, nil
			}}
	}
	jobs.Register(jobs.Plan{Kind: TestKind, Version: TestKind + "/1", Title: "Test steps", Steps: []jobs.StepDef{step("one"), step("two")}})
	t.Cleanup(func() { jobs.Unregister(TestKind + "/1") })
}

// InProcess is a Runner.Run that runs the step the argv names, as the unit
// would, against the jobs directory dir.
func InProcess(dir string, base jobs.Env) func(ctx context.Context, unit string, argv []string) error {
	return func(ctx context.Context, _ string, argv []string) error {
		a, err := jobs.ParseStepArgs(argv)
		if err != nil {
			return err
		}
		return jobs.RunStep(ctx, dir, a, base)
	}
}
```

Add `"fmt"` to the imports.

- [ ] **Step 6: Run the tests**

Run: `go test ./internal/jobs/... -count=1 -race`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/jobs/plan.go internal/jobs/step.go internal/jobs/manager.go \
  internal/jobs/manager_test.go internal/jobs/step_test.go internal/jobs/fakes_test.go internal/jobs/jobstest/jobstest.go
git commit -m "feat(jobs): plan registry, step runner and job manager with decisions, leases and reconcile

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 8: The `node.migrate-names` plan

Covers spec "Naming": "migrated by a sub-project 2 durable-job intent (`node.migrate-names`) … stop units, `usermod`/`groupmod` rename, re-render units, `chown` only if ownership is by name rather than uid, start units, verify". Also Q2, Q3, Q8, J10, and Review Focus 1.

**Files:**
- Create: `internal/jobs/migratenames.go`, `internal/jobs/marker.go`, `internal/jobs/migratenames_test.go`, `internal/jobs/scriptexec_test.go`

**Interfaces:**
- Consumes: Task 1 (`catalog.LegacyNames`, `CurrentNames`, `UnderRoot`, `Rebase`, `UnitPath`, `RenderUnits`, `WireConfig.Names`), Task 2 (`catalog.OwnerMarker`), Task 7 (`Register`, `Plan`, `StepDef`, `Env`, `Outcome`).
- Produces (package `jobs`):
  - `const MigrateNamesVersion = "node.migrate-names/1"`;
  - `type MigrateNamesResult struct { Names, DataDir, JWTPath string }` (the job's final result, which the controller copies into its config in Task 12);
  - `type NamesMarker struct { Names, DataRootFrom, DataRootTo string }`, `func ReadNamesMarker(path string) (NamesMarker, bool, error)`, `func (NamesMarker) Apply(w *catalog.WireConfig)`;
  - step ids, in order: `preflight`, `stop-units`, `rename-account`, `move-data`, `rewrite-units`, `ownership`, `start-units`, `verify`;
  - question keys `busy-user` (options `retry`, `stop-processes`, `abort`) and `mounted-data` (options `keep-path`, `retry`).
- Every step reads `node.json` from `env.NodePath` with `os.ReadFile` (the step runs on the box) and runs commands through `env.Exec`. Every step is idempotent, and every step after `preflight` does nothing when preflight found the box already migrated.

- [ ] **Step 1: Write the scripted executor and the failing tests**

```go
// internal/jobs/scriptexec_test.go
package jobs

import (
	"context"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// scriptExec answers each command by the longest scripted substring it
// contains (as internal/setup's fake does); an unscripted command succeeds
// with no output. Files written through it are kept in memory.
type scriptExec struct {
	mu      sync.Mutex
	scripts map[string][]executor.Result // a list answers successive calls; the last repeats
	calls   []string
	files   map[string][]byte
}

func newScript() *scriptExec {
	return &scriptExec{scripts: map[string][]executor.Result{}, files: map[string][]byte{}}
}

func (s *scriptExec) on(substr string, rs ...executor.Result) *scriptExec {
	s.scripts[substr] = rs
	return s
}

func (s *scriptExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, cmd)
	var keys []string
	for k := range s.scripts {
		if strings.Contains(cmd, k) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return executor.Result{}, nil
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	rs := s.scripts[keys[0]]
	r := rs[0]
	if len(rs) > 1 {
		s.scripts[keys[0]] = rs[1:]
	}
	return r, nil
}

func (s *scriptExec) WriteFile(_ context.Context, path string, b []byte, _ os.FileMode) error {
	s.mu.Lock()
	s.files[path] = b
	s.mu.Unlock()
	return nil
}

func (s *scriptExec) ReadFile(_ context.Context, path string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.files[path]; ok {
		return b, nil
	}
	return nil, os.ErrNotExist
}

func (s *scriptExec) Close() error { return nil }

func (s *scriptExec) ran(substr string) bool { return s.index(substr) >= 0 }

func (s *scriptExec) index(substr string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.calls {
		if strings.Contains(c, substr) {
			return i
		}
	}
	return -1
}

var ok0 = executor.Result{}
var fail1 = executor.Result{ExitCode: 1}

func idOK(uid string) executor.Result { return executor.Result{Stdout: uid + "\n"} }

// legacyBox scripts a box set up before sub-project 2: the legacy account
// exists, the new one does not.
func legacyBox() *scriptExec {
	return newScript().
		on("id -u valve-node-app", idOK("998")).
		on("id -u jumpgate-node", fail1).
		on("getent group valve-node-app", executor.Result{Stdout: "valve-node-app:x:998:\n"}).
		on("getent group jumpgate-node", fail1)
}
```

```go
// internal/jobs/migratenames_test.go
package jobs

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func legacyWire() catalog.WireConfig {
	return catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
		DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
}

func mnEnv(t *testing.T, ex *scriptExec, w catalog.WireConfig) Env {
	t.Helper()
	dir := t.TempDir()
	b, _ := json.Marshal(w)
	node := filepath.Join(dir, "node.json")
	if err := os.WriteFile(node, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return Env{Exec: ex, NodePath: node, NamesPath: filepath.Join(dir, "names.json"), Out: io.Discard, Results: map[string]json.RawMessage{}}
}

func step(t *testing.T, id string) StepDef {
	t.Helper()
	p, ok := PlanFor(MigrateNamesVersion)
	if !ok {
		t.Fatal("node.migrate-names is not registered")
	}
	return p.Steps[p.index(id)]
}

func TestThePlanIsRegisteredInOrder(t *testing.T) {
	p, ok := PlanForKind(intent.KindNodeMigrateNames)
	if !ok || p.Version != MigrateNamesVersion {
		t.Fatalf("plan %+v", p)
	}
	var ids []string
	for _, s := range p.Steps {
		ids = append(ids, s.ID)
	}
	if strings.Join(ids, ",") != "preflight,stop-units,rename-account,move-data,rewrite-units,ownership,start-units,verify" {
		t.Fatalf("steps %v", ids)
	}
}

func TestPreflightRecordsWhatRuns(t *testing.T) {
	ex := legacyBox().on("systemctl is-active valve-node-app-exec.service", executor.Result{Stdout: "active\n"}).
		on("systemctl is-active valve-node-app-beacon.service", executor.Result{Stdout: "inactive\n", ExitCode: 3})
	out, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire()))
	if err != nil {
		t.Fatal(err)
	}
	var pre migratePreflight
	_ = json.Unmarshal(out.Result, &pre)
	if !pre.WasActive["exec"] || pre.WasActive["beacon"] || pre.AlreadyDone {
		t.Fatalf("preflight %+v", pre)
	}
}

// Q3: a gateway unit running as the legacy account would break; refuse
// before changing anything.
func TestPreflightRefusesOtherUnitsOfTheLegacyUser(t *testing.T) {
	ex := legacyBox().on("grep -l -x", executor.Result{Stdout: "/etc/systemd/system/valve-node-app-exec.service\n/etc/systemd/system/valve-node-app-erpc.service\n"})
	_, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire()))
	if err == nil || !strings.Contains(err.Error(), "valve-node-app-erpc.service") {
		t.Fatalf("preflight = %v", err)
	}
	for _, c := range ex.calls {
		if strings.Contains(c, "systemctl stop") || strings.Contains(c, "usermod") {
			t.Fatalf("preflight changed the box: %s", c)
		}
	}
}

func TestPreflightRefusesBothAccounts(t *testing.T) {
	ex := legacyBox().on("id -u jumpgate-node", idOK("997"))
	if _, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil || !strings.Contains(err.Error(), "both") {
		t.Fatalf("preflight = %v", err)
	}
}

func TestAMigratedBoxIsANoOp(t *testing.T) {
	w := legacyWire()
	w.Names, w.DataDir, w.JWTPath = catalog.NamesCurrent, "/var/lib/jumpgate-node/369", "/var/lib/jumpgate-node/369/jwt.hex"
	ex := newScript().on("id -u valve-node-app", fail1).on("id -u jumpgate-node", idOK("998"))
	env := mnEnv(t, ex, w)
	out, err := step(t, "preflight").Run(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	env.Results["preflight"] = out.Result
	for _, id := range []string{"stop-units", "rename-account", "move-data", "rewrite-units", "ownership", "start-units"} {
		if _, err := step(t, id).Run(context.Background(), env); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	for _, c := range ex.calls {
		if strings.Contains(c, "systemctl stop") || strings.Contains(c, "usermod") || strings.Contains(c, "mv ") {
			t.Fatalf("a no-op job ran %q", c)
		}
	}
}

// Review Focus (also pinned): a process ignoring SIGTERM is killed only on
// the operator's say-so.
func TestStopUnitsAsksBeforeKillingProcesses(t *testing.T) {
	ex := legacyBox().on("pgrep -a -u valve-node-app", executor.Result{Stdout: "4242 reth node\n"})
	env := mnEnv(t, ex, legacyWire())
	out, err := step(t, "stop-units").Run(context.Background(), env)
	if err != nil || out.Ask == nil || out.Ask.Key != "busy-user" || !strings.Contains(out.Ask.Prompt, "4242") {
		t.Fatalf("stop-units = %+v, %v", out.Ask, err)
	}
	if ex.ran("pkill") {
		t.Fatal("processes were signalled before anyone decided")
	}
	ex.on("pgrep -a -u valve-node-app", executor.Result{Stdout: "4242 reth node\n"}, executor.Result{ExitCode: 1})
	env.Answer = &Answer{Key: "busy-user", Option: "stop-processes"}
	prev := stopGrace
	stopGrace = 0 // no real wait in tests
	t.Cleanup(func() { stopGrace = prev })
	if out, err := step(t, "stop-units").Run(context.Background(), env); err != nil || out.Ask != nil {
		t.Fatalf("after stop-processes: %+v, %v", out.Ask, err)
	}
	if !ex.ran("pkill -TERM -u valve-node-app") {
		t.Fatalf("calls %v", ex.calls)
	}
}

func TestRenameAccountRenamesUserThenGroup(t *testing.T) {
	// The new account is absent before the rename and present after it:
	// rename-account probes it on both sides of the usermod.
	ex := legacyBox().on("id -u jumpgate-node", fail1, idOK("998"))
	if _, err := step(t, "rename-account").Run(context.Background(), mnEnv(t, ex, legacyWire())); err != nil {
		t.Fatal(err)
	}
	u, g := ex.index("usermod -l jumpgate-node -d /var/lib/jumpgate-node valve-node-app"), ex.index("groupmod -n jumpgate-node valve-node-app")
	if u < 0 || g < u {
		t.Fatalf("usermod at %d, groupmod at %d: %v", u, g, ex.calls)
	}
}

func TestRenameAccountFinishesAHalfDoneRename(t *testing.T) {
	// The user was renamed by an earlier attempt; the group was not.
	ex := newScript().on("id -u valve-node-app", fail1).on("id -u jumpgate-node", idOK("998")).
		on("getent group valve-node-app", executor.Result{Stdout: "valve-node-app:x:998:\n"}).on("getent group jumpgate-node", fail1)
	if _, err := step(t, "rename-account").Run(context.Background(), mnEnv(t, ex, legacyWire())); err != nil {
		t.Fatal(err)
	}
	if ex.ran("usermod") || !ex.ran("groupmod -n jumpgate-node valve-node-app") {
		t.Fatalf("calls %v", ex.calls)
	}
}

// Review Focus 1: a mount at or under the old root would be left behind by
// fstab at the old path after a reboot.
func TestMoveDataNeverMovesAMountedTree(t *testing.T) {
	ex := newScript().
		on("test -d '/var/lib/valve-node-app'", ok0).on("test -e '/var/lib/jumpgate-node'", fail1).
		on("findmnt -rn -o TARGET", executor.Result{Stdout: "/\n/var/lib/valve-node-app\n"})
	env := mnEnv(t, ex, legacyWire())
	out, err := step(t, "move-data").Run(context.Background(), env)
	if err != nil || out.Ask == nil || out.Ask.Key != "mounted-data" {
		t.Fatalf("move-data = %+v, %v", out, err)
	}
	if ex.ran("mv ") {
		t.Fatal("a mounted tree was moved")
	}
	env.Answer = &Answer{Key: "mounted-data", Option: "keep-path"}
	out, err = step(t, "move-data").Run(context.Background(), env)
	var mv migrateMove
	_ = json.Unmarshal(out.Result, &mv)
	if err != nil || mv.Moved || mv.DataDir != "/var/lib/valve-node-app/369" || ex.ran("mv ") {
		t.Fatalf("keep-path: %+v, %v", mv, err)
	}
}

func TestMoveDataMovesTheRootAndIsIdempotent(t *testing.T) {
	ex := newScript().on("test -d '/var/lib/valve-node-app'", ok0).on("test -e '/var/lib/jumpgate-node'", fail1).
		on("findmnt -rn -o TARGET", executor.Result{Stdout: "/\n"})
	env := mnEnv(t, ex, legacyWire())
	out, err := step(t, "move-data").Run(context.Background(), env)
	var mv migrateMove
	_ = json.Unmarshal(out.Result, &mv)
	if err != nil || !mv.Moved || mv.DataDir != "/var/lib/jumpgate-node/369" || mv.JWTPath != "/var/lib/jumpgate-node/369/jwt.hex" {
		t.Fatalf("move-data = %+v, %v", mv, err)
	}
	if !ex.ran("mv -T '/var/lib/valve-node-app' '/var/lib/jumpgate-node'") {
		t.Fatalf("calls %v", ex.calls)
	}
	// Already moved: the old root is gone, the new one is there.
	again := newScript().on("test -d '/var/lib/valve-node-app'", fail1).on("test -e '/var/lib/jumpgate-node'", ok0)
	env.Exec = again
	if out, err := step(t, "move-data").Run(context.Background(), env); err != nil || again.ran("mv ") {
		t.Fatalf("second run: %s, %v", out.Result, err)
	}
}

func TestMoveDataLeavesACustomDataDirAlone(t *testing.T) {
	w := legacyWire()
	w.DataDir, w.JWTPath = "/data/pulse", "/data/pulse/jwt.hex"
	ex := newScript()
	out, err := step(t, "move-data").Run(context.Background(), mnEnv(t, ex, w))
	var mv migrateMove
	_ = json.Unmarshal(out.Result, &mv)
	if err != nil || mv.Moved || mv.DataDir != "/data/pulse" || len(ex.calls) != 0 {
		t.Fatalf("move-data = %+v, %v, calls %v", mv, err, ex.calls)
	}
}

func TestRewriteUnitsSwitchesNodeJSONAndTheMarker(t *testing.T) {
	ex := newScript()
	env := mnEnv(t, ex, legacyWire())
	env.Results["move-data"] = json.RawMessage(`{"moved":true,"dataDir":"/var/lib/jumpgate-node/369","jwtPath":"/var/lib/jumpgate-node/369/jwt.hex"}`)
	if _, err := step(t, "rewrite-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	unit := string(ex.files["/etc/systemd/system/jumpgate-node-exec.service"])
	if !strings.Contains(unit, "User=jumpgate-node\n") || !strings.Contains(unit, "ReadWritePaths=/var/lib/jumpgate-node/369\n") {
		t.Fatalf("unit:\n%s", unit)
	}
	if !ex.ran("rm -f '/etc/systemd/system/valve-node-app-exec.service' '/etc/systemd/system/valve-node-app-beacon.service'") ||
		!ex.ran("systemctl enable jumpgate-node-exec.service jumpgate-node-beacon.service") {
		t.Fatalf("calls %v", ex.calls)
	}
	b, _ := os.ReadFile(env.NodePath)
	var w catalog.WireConfig
	_ = json.Unmarshal(b, &w)
	if w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/jumpgate-node/369" {
		t.Fatalf("node.json %+v", w)
	}
	mk, ok, err := ReadNamesMarker(env.NamesPath)
	if err != nil || !ok || mk.Names != catalog.NamesCurrent || mk.DataRootTo != "/var/lib/jumpgate-node" {
		t.Fatalf("marker %+v %v %v", mk, ok, err)
	}
}

// Spec: chown only if ownership is by name rather than uid. usermod keeps the
// uid, so a normal box needs no walk at all.
func TestOwnershipChownsOnlyWhenTheNameDoesNotFollowTheUID(t *testing.T) {
	w := legacyWire()
	w.Names, w.DataDir, w.JWTPath = catalog.NamesCurrent, "/var/lib/jumpgate-node/369", "/var/lib/jumpgate-node/369/jwt.hex"
	ex := newScript().on("stat -c %U", executor.Result{Stdout: "jumpgate-node\n"})
	if _, err := step(t, "ownership").Run(context.Background(), mnEnv(t, ex, w)); err != nil || ex.ran("chown -R") {
		t.Fatalf("calls %v, %v", ex.calls, err)
	}
	if !ex.ran("printf '%s\\n' jumpgate-node > '/var/lib/jumpgate-node/369/" + catalog.OwnerMarker + "'") {
		t.Fatalf("the owner marker was not rewritten: %v", ex.calls)
	}
	ex = newScript().on("stat -c %U", executor.Result{Stdout: "UNKNOWN\n"})
	if _, err := step(t, "ownership").Run(context.Background(), mnEnv(t, ex, w)); err != nil || !ex.ran("chown -R jumpgate-node:jumpgate-node") {
		t.Fatalf("calls %v, %v", ex.calls, err)
	}
}

func TestStartUnitsRestoresOnlyWhatRan(t *testing.T) {
	w := legacyWire()
	w.Names = catalog.NamesCurrent
	ex := newScript().on("systemctl is-active jumpgate-node-exec.service", executor.Result{Stdout: "active\n"})
	env := mnEnv(t, ex, w)
	env.Results["preflight"] = json.RawMessage(`{"wasActive":{"exec":true,"beacon":false}}`)
	if _, err := step(t, "start-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	if !ex.ran("systemctl start jumpgate-node-exec.service") || ex.ran("systemctl start jumpgate-node-beacon.service") {
		t.Fatalf("calls %v", ex.calls)
	}
}

func TestNamesMarkerAppliesToAStaleNodeJSON(t *testing.T) {
	w := legacyWire()
	NamesMarker{Names: catalog.NamesCurrent, DataRootFrom: "/var/lib/valve-node-app", DataRootTo: "/var/lib/jumpgate-node"}.Apply(&w)
	if w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/jumpgate-node/369" || w.JWTPath != "/var/lib/jumpgate-node/369/jwt.hex" {
		t.Fatalf("applied %+v", w)
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/jobs/ -run 'Plan|Preflight|MigratedBox|StopUnits|RenameAccount|MoveData|RewriteUnits|Ownership|StartUnits|NamesMarker' -count=1`
Expected: FAIL to compile (undefined `MigrateNamesVersion`, `migratePreflight`, …).

- [ ] **Step 3: Implement `marker.go`**

```go
// internal/jobs/marker.go
package jobs

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/fsatomic"
)

// NamesMarker is /etc/jumpgate/names.json, written by node.migrate-names when
// it switches a box's names. The agent applies it over node.json on every
// read (D-J10): pairing rewrites node.json from the controller's copy of the
// wire, and a controller that missed the migration must not point the agent
// back at units and an account that no longer exist.
type NamesMarker struct {
	Names        string `json:"names"`
	DataRootFrom string `json:"dataRootFrom,omitempty"` // empty when the data stayed put
	DataRootTo   string `json:"dataRootTo,omitempty"`
}

func ReadNamesMarker(path string) (NamesMarker, bool, error) {
	var m NamesMarker
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return m, false, nil
	}
	if err != nil {
		return m, false, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, false, err
	}
	if _, err := catalog.NamesByID(m.Names); err != nil {
		return m, false, err
	}
	return m, true, nil
}

func writeNamesMarker(path string, m NamesMarker) error {
	b, _ := json.Marshal(m)
	return fsatomic.WriteFile(path, b)
}

// Apply rewrites a node config the marker post-dates.
func (m NamesMarker) Apply(w *catalog.WireConfig) {
	w.Names = m.Names
	if m.DataRootFrom == "" {
		return
	}
	if d, ok := catalog.Rebase(w.DataDir, m.DataRootFrom, m.DataRootTo); ok {
		w.DataDir = d
	}
	if j, ok := catalog.Rebase(w.JWTPath, m.DataRootFrom, m.DataRootTo); ok {
		w.JWTPath = j
	}
}
```

- [ ] **Step 4: Implement `migratenames.go`**

```go
// internal/jobs/migratenames.go
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// MigrateNamesVersion is the plan version node.migrate-names runs.
const MigrateNamesVersion = "node.migrate-names/1"

// stopGrace is how long stop-processes waits after SIGTERM before SIGKILL.
// A variable so tests do not wait.
var stopGrace = 30 * time.Second

func init() {
	Register(Plan{Kind: intent.KindNodeMigrateNames, Version: MigrateNamesVersion,
		Title: "Rename the node's account, units and data to jumpgate-node",
		Validate: func(in json.RawMessage) error {
			var p intent.MigrateNamesPayload
			return json.Unmarshal(in, &p)
		},
		Steps: []StepDef{
			{ID: "preflight", Title: "Check what to rename", Run: mnPreflight,
				AbortNote: "nothing was changed"},
			{ID: "stop-units", Title: "Stop the node services", Run: mnStopUnits,
				AbortNote: "nothing was renamed; the node services stay stopped until you start them (`jumpgate service HOST start exec`, then beacon)"},
			{ID: "rename-account", Title: "Rename the service account", Run: mnRenameAccount,
				AbortNote: "the account may already be renamed while the units still name the old one: the node cannot start until a new node.migrate-names job finishes"},
			{ID: "move-data", Title: "Move the data root", Run: mnMoveData,
				AbortNote: "the account is renamed but the units are not yet: the node cannot start until a new node.migrate-names job finishes"},
			{ID: "rewrite-units", Title: "Rewrite the node units", Run: mnRewriteUnits,
				AbortNote: "the units may be half rewritten: the node cannot start until a new node.migrate-names job finishes"},
			{ID: "ownership", Title: "Check data ownership", Run: mnOwnership,
				AbortNote: "the names are switched; some data may not be owned by jumpgate-node yet, so a client may fail to write"},
			{ID: "start-units", Title: "Start the node services", Run: mnStartUnits,
				AbortNote: "the migration is done; the node services are not running, start them by hand"},
			{ID: "verify", Title: "Verify", Run: mnVerify,
				AbortNote: "the migration ran but did not verify; read this job's log before you change anything"},
		}})
}

var (
	legacy  = catalog.LegacyNames
	current = catalog.CurrentNames
)

type migratePreflight struct {
	WasActive   map[string]bool `json:"wasActive"` // "exec", "beacon": running when the job began
	AlreadyDone bool            `json:"alreadyDone"`
}

type migrateMove struct {
	Moved   bool   `json:"moved"`
	DataDir string `json:"dataDir"`
	JWTPath string `json:"jwtPath"`
}

// MigrateNamesResult is the job's result; the controller copies it into its
// own record of the box (Task 12).
type MigrateNamesResult struct {
	Names   string `json:"names"`
	DataDir string `json:"dataDir"`
	JWTPath string `json:"jwtPath"`
}

func readNode(env Env) (catalog.WireConfig, error) {
	var w catalog.WireConfig
	b, err := os.ReadFile(env.NodePath)
	if errors.Is(err, fs.ErrNotExist) {
		return w, errors.New("no node is set up on this box (no node.json); there is nothing to rename")
	}
	if err != nil {
		return w, err
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return w, fmt.Errorf("node.json does not parse: %w", err)
	}
	return w, nil
}

func writeNode(env Env, w catalog.WireConfig) error {
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(env.NodePath, b)
}

// sh runs cmd and turns a non-zero exit into an error quoting the box.
func sh(ctx context.Context, env Env, cmd string) (executor.Result, error) {
	res, err := env.Exec.Run(ctx, cmd, nil)
	if env.Out != nil {
		fmt.Fprintf(env.Out, "$ %s\n%s%s", cmd, res.Stdout, res.Stderr)
	}
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%s exited %d: %s", strings.Fields(cmd)[0], res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return res, nil
}

// probe runs a check whose exit code is the answer.
func probe(ctx context.Context, env Env, cmd string) (bool, error) {
	res, err := env.Exec.Run(ctx, cmd, nil)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

func userExists(ctx context.Context, env Env, u string) (bool, error) {
	return probe(ctx, env, "id -u "+u)
}

func groupExists(ctx context.Context, env Env, g string) (bool, error) {
	return probe(ctx, env, "getent group "+g)
}

func alreadyDone(env Env) bool {
	var pre migratePreflight
	_ = json.Unmarshal(env.Results["preflight"], &pre)
	return pre.AlreadyDone
}

func isActive(ctx context.Context, env Env, unit string) bool {
	res, err := env.Exec.Run(ctx, "systemctl is-active "+unit, nil)
	return err == nil && strings.TrimSpace(res.Stdout) == "active"
}

func mnPreflight(ctx context.Context, env Env) (Outcome, error) {
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	hasLegacy, err := userExists(ctx, env, legacy.User)
	if err != nil {
		return Outcome{}, err
	}
	hasCurrent, err := userExists(ctx, env, current.User)
	if err != nil {
		return Outcome{}, err
	}
	switch {
	case hasLegacy && hasCurrent:
		return Outcome{}, fmt.Errorf("both accounts exist (%s and %s); remove the one the node does not use, then retry", legacy.User, current.User)
	case !hasLegacy && !hasCurrent:
		return Outcome{}, fmt.Errorf("neither %s nor %s exists; set the node up first", legacy.User, current.User)
	case !hasLegacy && hasCurrent && w.Names == catalog.NamesCurrent:
		b, _ := json.Marshal(migratePreflight{AlreadyDone: true})
		return Outcome{Result: b}, nil
	}
	// Any other unit running as the legacy account would break when it is
	// renamed (the systemd eRPC backend, for one). Q3: those move in
	// sub-project 6; until then, refuse before touching anything.
	res, err := env.Exec.Run(ctx, "grep -l -x 'User="+legacy.User+"' /etc/systemd/system/*.service 2>/dev/null", nil)
	if err != nil {
		return Outcome{}, err
	}
	var others []string
	for _, p := range strings.Fields(res.Stdout) {
		if p != catalog.UnitPath(legacy.ExecUnit()) && p != catalog.UnitPath(legacy.BeaconUnit()) {
			others = append(others, p)
		}
	}
	if len(others) > 0 {
		return Outcome{}, fmt.Errorf("other units run as %s and would break if it were renamed: %s. Remove them (jumpgate renames gateways in a later release), then retry",
			legacy.User, strings.Join(others, ", "))
	}
	pre := migratePreflight{WasActive: map[string]bool{
		"exec":   isActive(ctx, env, legacy.ExecUnit()) || isActive(ctx, env, current.ExecUnit()),
		"beacon": isActive(ctx, env, legacy.BeaconUnit()) || isActive(ctx, env, current.BeaconUnit()),
	}}
	b, _ := json.Marshal(pre)
	return Outcome{Result: b}, nil
}

func mnStopUnits(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	units := append(legacy.Units(), current.Units()...)
	// A unit that does not exist is not an error here: either name set may be absent.
	if _, err := env.Exec.Run(ctx, "systemctl stop "+strings.Join(units, " ")+" 2>/dev/null", nil); err != nil {
		return Outcome{}, err
	}
	user := legacy.User
	if ok, _ := userExists(ctx, env, user); !ok {
		user = current.User
	}
	// lines is every process of the account, one "pid command" line each;
	// pgrep exits 1 when there is none.
	lines := func() []string {
		res, _ := env.Exec.Run(ctx, "pgrep -a -u "+user, nil)
		if res.ExitCode != 0 {
			return nil
		}
		return strings.Split(strings.TrimSpace(res.Stdout), "\n")
	}
	left := lines()
	if len(left) == 0 {
		return Outcome{}, nil
	}
	if env.Answer != nil && env.Answer.Key == "busy-user" && env.Answer.Option == "stop-processes" {
		if _, err := env.Exec.Run(ctx, "pkill -TERM -u "+user, nil); err != nil {
			return Outcome{}, err
		}
		deadline := time.Now().Add(stopGrace)
		for len(lines()) > 0 && time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return Outcome{}, ctx.Err()
			case <-time.After(time.Second):
			}
		}
		if len(lines()) > 0 {
			_, _ = env.Exec.Run(ctx, "pkill -KILL -u "+user, nil)
			time.Sleep(min(stopGrace, 5*time.Second))
		}
		if left := lines(); len(left) > 0 {
			return Outcome{}, fmt.Errorf("%d process(es) still run as %s after SIGKILL: %s", len(left), user, strings.Join(left, "; "))
		}
		return Outcome{}, nil
	}
	shown := left[:min(len(left), 10)]
	return Outcome{Ask: &intent.JobQuestion{Key: "busy-user", Tier: intent.TierRoutine,
		Prompt: fmt.Sprintf("%d process(es) still run as %s after the node services stopped, so the account cannot be renamed:\n%s",
			len(left), user, strings.Join(shown, "\n")),
		Options: []intent.JobOption{
			{ID: intent.OptRetry, Label: "Check again", Detail: "stop them yourself first"},
			{ID: "stop-processes", Label: "Stop them", Detail: fmt.Sprintf("SIGTERM, then SIGKILL after %s", stopGrace)},
			{ID: intent.OptAbort, Label: "Stop the migration here", Detail: "nothing has been renamed; the node services stay stopped"},
		}}}, nil
}

func mnRenameAccount(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	hasLegacy, err := userExists(ctx, env, legacy.User)
	if err != nil {
		return Outcome{}, err
	}
	hasCurrent, err := userExists(ctx, env, current.User)
	if err != nil {
		return Outcome{}, err
	}
	// Preflight refused a box with both accounts. One that has both now got
	// the second from something other than this job (an operator, between a
	// failed attempt and its retry): usermod onto it would fail half-way.
	if hasLegacy && hasCurrent {
		return Outcome{}, fmt.Errorf("both accounts exist (%s and %s); remove the one the node does not use, then retry", legacy.User, current.User)
	}
	// The user first: it is the call that fails while a process still runs as
	// the account, and then nothing at all has changed.
	if hasLegacy {
		if _, err := sh(ctx, env, "usermod -l "+current.User+" -d "+current.DataRoot+" "+legacy.User); err != nil {
			return Outcome{}, err
		}
	}
	if ok, err := groupExists(ctx, env, legacy.Group); err != nil {
		return Outcome{}, err
	} else if ok {
		if _, err := sh(ctx, env, "groupmod -n "+current.Group+" "+legacy.Group); err != nil {
			return Outcome{}, err
		}
	}
	if ok, _ := userExists(ctx, env, current.User); !ok {
		return Outcome{}, fmt.Errorf("%s does not exist after the rename", current.User)
	}
	return Outcome{}, nil
}

func mnMoveData(ctx context.Context, env Env) (Outcome, error) {
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	stay := func() (Outcome, error) {
		b, _ := json.Marshal(migrateMove{DataDir: w.DataDir, JWTPath: w.JWTPath})
		return Outcome{Result: b}, nil
	}
	if alreadyDone(env) || !catalog.UnderRoot(w.DataDir, legacy.DataRoot) {
		return stay() // a custom data dir stays where it is (Q2)
	}
	newDir, _ := catalog.Rebase(w.DataDir, legacy.DataRoot, current.DataRoot)
	newJWT, _ := catalog.Rebase(w.JWTPath, legacy.DataRoot, current.DataRoot)
	moved := func() (Outcome, error) {
		b, _ := json.Marshal(migrateMove{Moved: true, DataDir: newDir, JWTPath: newJWT})
		return Outcome{Result: b}, nil
	}
	oldThere, err := probe(ctx, env, "test -d "+shQuote(legacy.DataRoot))
	if err != nil {
		return Outcome{}, err
	}
	newThere, err := probe(ctx, env, "test -e "+shQuote(current.DataRoot))
	if err != nil {
		return Outcome{}, err
	}
	switch {
	case !oldThere && newThere:
		return moved() // an earlier attempt moved it
	case oldThere && newThere:
		return Outcome{}, fmt.Errorf("both %s and %s exist; move one aside, then retry", legacy.DataRoot, current.DataRoot)
	case !oldThere:
		return Outcome{}, fmt.Errorf("%s does not exist", legacy.DataRoot)
	}
	// Review Focus 1: a mount at or under the old root stays mounted at the
	// old path after a reboot (fstab, mount units), so it is never moved.
	if env.Answer != nil && env.Answer.Key == "mounted-data" && env.Answer.Option == "keep-path" {
		return stay()
	}
	res, err := sh(ctx, env, "findmnt -rn -o TARGET")
	if err != nil {
		return Outcome{}, err
	}
	var mounts []string
	for _, m := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		if catalog.UnderRoot(m, legacy.DataRoot) {
			mounts = append(mounts, m)
		}
	}
	if len(mounts) > 0 {
		return Outcome{Ask: &intent.JobQuestion{Key: "mounted-data", Tier: intent.TierRoutine,
			Prompt: fmt.Sprintf("%s is, or holds, a mount point (%s). Moving it would leave /etc/fstab or its mount unit pointing at the old path.",
				legacy.DataRoot, strings.Join(mounts, ", ")),
			Options: []intent.JobOption{
				{ID: "keep-path", Label: "Keep the data where it is", Detail: "the node runs as " + current.User + " with its data still at " + w.DataDir},
				{ID: intent.OptRetry, Label: "Check again", Detail: "after you move the mount yourself"},
			}}}, nil
	}
	// Both roots sit in /var/lib, and no mount lies under the old one, so
	// this is a rename(2) on one filesystem: instant, nothing copied.
	if _, err := sh(ctx, env, "mv -T "+shQuote(legacy.DataRoot)+" "+shQuote(current.DataRoot)); err != nil {
		return Outcome{}, err
	}
	return moved()
}

func mnRewriteUnits(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	var mv migrateMove
	if err := json.Unmarshal(env.Results["move-data"], &mv); err != nil {
		return Outcome{}, fmt.Errorf("move-data left no result: %w", err)
	}
	nw := w
	nw.Names, nw.DataDir, nw.JWTPath = catalog.NamesCurrent, mv.DataDir, mv.JWTPath
	execUnit, beaconUnit, err := catalog.RenderUnits(nw)
	if err != nil {
		return Outcome{}, err
	}
	if err := env.Exec.WriteFile(ctx, catalog.UnitPath(current.ExecUnit()), []byte(execUnit), 0o644); err != nil {
		return Outcome{}, err
	}
	if err := env.Exec.WriteFile(ctx, catalog.UnitPath(current.BeaconUnit()), []byte(beaconUnit), 0o644); err != nil {
		return Outcome{}, err
	}
	if _, err := env.Exec.Run(ctx, "systemctl disable "+strings.Join(legacy.Units(), " ")+" 2>/dev/null", nil); err != nil {
		return Outcome{}, err
	}
	if _, err := sh(ctx, env, "rm -f "+shQuote(catalog.UnitPath(legacy.ExecUnit()))+" "+shQuote(catalog.UnitPath(legacy.BeaconUnit()))); err != nil {
		return Outcome{}, err
	}
	if _, err := sh(ctx, env, "systemctl daemon-reload && systemctl enable "+strings.Join(current.Units(), " ")); err != nil {
		return Outcome{}, err
	}
	// node.json and the marker switch together, after the units: from here on
	// every agent read names the new units.
	if err := writeNode(env, nw); err != nil {
		return Outcome{}, err
	}
	mk := NamesMarker{Names: catalog.NamesCurrent}
	if mv.Moved {
		mk.DataRootFrom, mk.DataRootTo = legacy.DataRoot, current.DataRoot
	}
	return Outcome{}, writeNamesMarker(env.NamesPath, mk)
}

func mnOwnership(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	owner := func(p string) string {
		res, _ := env.Exec.Run(ctx, "stat -c %U "+shQuote(p), nil)
		return strings.TrimSpace(res.Stdout)
	}
	marker := shQuote(w.DataDir + "/" + catalog.OwnerMarker)
	// usermod -l keeps the uid, so files the old account owned now show the
	// new name: nothing to walk. Only a tree owned by a name the uid does not
	// follow (a mismatched NFS or idmapped mount) needs chown -R (spec).
	if owner(w.DataDir) != current.User || owner(w.JWTPath) != current.User {
		if _, err := sh(ctx, env, fmt.Sprintf("chown -R %s:%s %s %s", current.User, current.Group, shQuote(w.DataDir), shQuote(w.JWTPath))); err != nil {
			return Outcome{}, err
		}
	}
	if _, err := sh(ctx, env, fmt.Sprintf("printf '%%s\\n' %s > %s && chown %s:%s %s", current.User, marker, current.User, current.Group, marker)); err != nil {
		return Outcome{}, err
	}
	return Outcome{}, nil
}

func mnStartUnits(ctx context.Context, env Env) (Outcome, error) {
	if alreadyDone(env) {
		return Outcome{}, nil
	}
	var pre migratePreflight
	_ = json.Unmarshal(env.Results["preflight"], &pre)
	for svc, unit := range map[string]string{"exec": current.ExecUnit(), "beacon": current.BeaconUnit()} {
		if !pre.WasActive[svc] {
			continue // it was stopped before the job: leave it so
		}
		if _, err := sh(ctx, env, "systemctl start "+unit); err != nil {
			return Outcome{}, err
		}
		if !isActive(ctx, env, unit) {
			return Outcome{}, fmt.Errorf("%s did not stay active; read `journalctl -u %s`", unit, unit)
		}
	}
	return Outcome{}, nil
}

func mnVerify(ctx context.Context, env Env) (Outcome, error) {
	w, err := readNode(env)
	if err != nil {
		return Outcome{}, err
	}
	var problems []string
	if ok, _ := userExists(ctx, env, current.User); !ok {
		problems = append(problems, current.User+" does not exist")
	}
	if ok, _ := userExists(ctx, env, legacy.User); ok {
		problems = append(problems, legacy.User+" still exists")
	}
	if ok, _ := groupExists(ctx, env, current.Group); !ok {
		problems = append(problems, "group "+current.Group+" does not exist")
	}
	if ok, _ := probe(ctx, env, "test -e "+shQuote(catalog.UnitPath(legacy.ExecUnit()))); ok {
		problems = append(problems, catalog.UnitPath(legacy.ExecUnit())+" is still there")
	}
	if ok, _ := probe(ctx, env, "systemctl is-enabled "+strings.Join(current.Units(), " ")); !ok {
		problems = append(problems, "the "+current.Prefix+" units are not enabled")
	}
	if w.Names != catalog.NamesCurrent {
		problems = append(problems, "node.json still names the legacy account")
	}
	if len(problems) > 0 {
		return Outcome{}, errors.New(strings.Join(problems, "; "))
	}
	b, _ := json.Marshal(MigrateNamesResult{Names: w.Names, DataDir: w.DataDir, JWTPath: w.JWTPath})
	return Outcome{Result: b}, nil
}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/jobs/... -count=1 -race`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/jobs/migratenames.go internal/jobs/marker.go internal/jobs/migratenames_test.go internal/jobs/scriptexec_test.go
git commit -m "feat(jobs): node.migrate-names renames the node account, units and data root

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 9: The agent answers job intents — dispatch, decision tiers, `job.output`, reconcile loop, `agent job-step`

Covers spec "Durable jobs" on the agent, "Policy (on the agent)" for decisions ("a signed decision of the tier the policy requires"), Q5, J1, J8, J9, J10.

**Files:**
- Create: `internal/agent/jobs.go`, `internal/agent/jobs_test.go`
- Modify: `internal/agent/agent.go` (`Config.Jobs`, `Config.NamesPath`; `Handle` passes `env` and `signers`; the decision tier check), `internal/agent/dispatch.go` (`dispatch` signature; `PlanVersions`; `loadNode` applies the names marker; `service.action` refused while a job runs; `agent.info` names and data dir), `internal/agent/policy.go` (`Authorise` takes extra tiers), `internal/agent/logs.go` (ruling P9: `logsSince` generalised over a unit list and an item type, so `logs.since` and `job.output` share one bounded read), `internal/agent/logs_test.go` (units are quoted), `internal/agent/agent_test.go` (the `sendSignedBy`, `enroll` and `setQuestionTier` helpers), `cmd/jumpgate/cli_agent.go` (`job-step` subcommand; the manager and the reconcile loop in `agentRun`), `internal/server/api.go` (ruling P11: `setupExisting` asks a paired box's `agent.info` for its names when the controller keeps no wire)
- Test: `internal/agent/jobs_test.go`, `internal/agent/policy_test.go`, `internal/agent/logs_test.go`, `cmd/jumpgate/cli_test.go`, `internal/server/api_test.go`

**Interfaces:**
- Consumes: Tasks 4–8.
- Produces (package `agent`): `Config.Jobs *jobs.Manager`; `Config.NamesPath string`; `func (a *Agent) RunJobs(ctx context.Context)` (reconciles at once, then every `ReconcileEvery = 5 * time.Second`, and prunes daily); `func (p Policy) Authorise(kind string, controller eip712.Address, signers []eip712.Address, extra ...Tier) error`.
- Produces (CLI): `jumpgate agent job-step --state-dir D --config-dir C --job ID --step S --attempt N`.
- Changes (package `agent`, ruling P9): `func logsSinceCommand(units []string, cursor string, n int) (string, error)` quotes each unit (a job's unit pattern is a glob the shell must not expand); `func logsSince[T any](ctx context.Context, ex executor.Executor, units []string, cursor string, n int, now time.Time, item func(journalEntry) T) ([]T, string, *Reject, error)` is the one bounded journal read (D-J8) behind both `logs.since` and `job.output`.
- Produces (package `server`, ruling P11): `func (s *Server) setupExisting(ctx context.Context, cfg config.Config, t config.Target) (*catalog.WireConfig, error)`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/agent/jobs_test.go
package agent

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// jobsRig is newRig with a job manager whose units run in-process.
func jobsRig(t *testing.T) (*rig, *jobstest.Runner) {
	t.Helper()
	jobstest.RegisterPlan(t)
	r := newRig(t, true)
	dir := filepath.Join(r.dir, "jobs")
	run := &jobstest.Runner{}
	run.Run = jobstest.InProcess(dir, jobs.Env{Exec: r.exec, NodePath: filepath.Join(r.dir, "node.json"), Out: io.Discard})
	m, err := jobs.NewManager(jobs.ManagerConfig{Dir: dir, Key: r.agentKey, Runner: run, Now: func() time.Time { return r.now },
		StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv("/jumpgate", r.dir, r.dir, a) }})
	if err != nil {
		t.Fatal(err)
	}
	r.a.cfg.Jobs = m
	// test.steps has no default requirement: the policy must name it.
	p, _ := LoadPolicy(filepath.Join(r.dir, "policy.json"))
	p.Kinds = map[string][]Tier{jobstest.TestKind: {TierRoutine}}
	_ = p.Save(filepath.Join(r.dir, "policy.json"))
	return r, run
}

func okResult[T any](t *testing.T, rc intent.Receipt, res []byte) T {
	t.Helper()
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	var v T
	if err := json.Unmarshal(res, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAJobRoundTripsThroughSignedIntents(t *testing.T) {
	r, run := jobsRig(t)
	rec := okResult[intent.JobRecord](t, r.send(t, jobstest.TestKind, map[string]string{}, nil))
	for i := 0; i < 2; i++ {
		rec = okResult[intent.JobRecord](t, r.send(t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID, Step: rec.Step, Attempt: rec.Attempt}, nil))
		run.Wait()
		if err := r.a.cfg.Jobs.Reconcile(context.Background()); err != nil {
			t.Fatal(err)
		}
		rec = okResult[intent.JobReadResult](t, r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, nil)).Job
	}
	res := okResult[intent.JobReadResult](t, r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, nil))
	if res.Job.State != intent.JobSucceeded {
		t.Fatalf("job %+v", res.Job)
	}
	var es []joblog.Entry
	for _, j := range res.Entries {
		e, err := joblog.FromJSON(j)
		if err != nil {
			t.Fatal(err)
		}
		es = append(es, e)
	}
	if _, err := joblog.Verify(r.agentKey.Address(), rec.ID, es, 0, [32]byte{}); err != nil {
		t.Fatal(err)
	}
	if err := joblog.VerifyIntents(es); err != nil {
		t.Fatalf("the log's intents do not verify as the controller's: %v", err)
	}
	list := okResult[intent.JobListResult](t, r.send(t, intent.KindJobList, struct{}{}, nil))
	if len(list.Jobs) != 1 || list.AgentTime != r.now.Unix() {
		t.Fatalf("list %+v", list)
	}
}

// J1: the node services belong to the job while it runs.
func TestServiceActionWaitsForARunningJob(t *testing.T) {
	r, _ := jobsRig(t)
	_ = okResult[intent.JobRecord](t, r.send(t, jobstest.TestKind, map[string]string{}, nil))
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonJobActive {
		t.Fatalf("service.action during a job: %s", got)
	}
}

// J9: no job manager (or no systemd) is a signed, explained refusal.
func TestJobsWithoutAManagerAreUnsupported(t *testing.T) {
	r := newRig(t, true)
	rc, res := r.send(t, intent.KindJobList, struct{}{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonJobsUnsupported {
		t.Fatalf("job.list without a manager: %s", got)
	}
}

func TestJobsOnAnUnsupportedBoxSayWhy(t *testing.T) {
	r, run := jobsRig(t)
	run.Unsupported = jobs.ErrUnsupported
	rc, res := r.send(t, jobstest.TestKind, map[string]string{}, nil)
	if got := rejection(t, rc, res); got.Code != intent.ReasonJobsUnsupported {
		t.Fatalf("start on an unsupported box: %+v", got)
	}
}

// Q5: a question of the approval tier needs an approval signature on the
// decide intent, besides the controller's.
func TestAnApprovalQuestionNeedsAnApprovalSignature(t *testing.T) {
	r, run := jobsRig(t)
	rec := okResult[intent.JobRecord](t, r.send(t, jobstest.TestKind, map[string]string{"AskAt": "one"}, nil))
	_ = okResult[intent.JobRecord](t, r.send(t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID}, nil))
	run.Wait()
	_ = r.a.cfg.Jobs.Reconcile(context.Background())
	rec = okResult[intent.JobReadResult](t, r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, nil)).Job
	setQuestionTier(t, r, rec.ID, intent.TierApproval) // the test plan asks routine; raise it on disk

	rc, res := r.send(t, intent.KindJobDecide, intent.JobDecidePayload{ID: rec.ID, Decision: rec.Question.ID, Option: "go"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonUnauthorizedKind {
		t.Fatalf("an approval question answered with a routine signature: %s", got)
	}
	wallet, _ := signer.GenerateKey()
	enroll(t, r, wallet, TierApproval)
	rc, res = r.sendSignedBy(t, intent.KindJobDecide, intent.JobDecidePayload{ID: rec.ID, Decision: rec.Question.ID, Option: "go"}, wallet)
	if rc.Status != intent.StatusOK {
		t.Fatalf("with an approval signature: %s", res)
	}
}

// J10: a re-pair that rewrote node.json with the legacy names cannot undo
// a migration.
func TestTheNamesMarkerWinsOverAStaleNodeJSON(t *testing.T) {
	r := newRig(t, true) // node.json names the legacy root
	r.a.cfg.NamesPath = filepath.Join(r.dir, "names.json")
	b, _ := json.Marshal(jobs.NamesMarker{Names: catalog.NamesCurrent, DataRootFrom: "/var/lib/valve-node-app", DataRootTo: "/var/lib/jumpgate-node"})
	_ = os.WriteFile(r.a.cfg.NamesPath, b, 0o600)
	info := okResult[intent.AgentInfo](t, r.send(t, intent.KindAgentInfo, struct{}{}, nil))
	if info.Names != catalog.NamesCurrent || info.DataDir != "/var/lib/jumpgate-node/369" {
		t.Fatalf("agent.info %+v", info)
	}
	_, _ = r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	if last := r.exec.cmds[len(r.exec.cmds)-1]; !strings.Contains(last, "jumpgate-node-exec.service") {
		t.Fatalf("service.action ran %q", last)
	}
}

func TestJobOutputReadsTheJobsUnitsOnly(t *testing.T) {
	r, _ := jobsRig(t)
	rec := okResult[intent.JobRecord](t, r.send(t, jobstest.TestKind, map[string]string{}, nil))
	r.exec.res = executor.Result{Stdout: `{"__CURSOR":"s=1","__REALTIME_TIMESTAMP":"1800000000000000","_SYSTEMD_UNIT":"jumpgate-job-` + rec.ID + `-one-0.service","MESSAGE":"running one"}` + "\n"}
	out := okResult[intent.JobOutputResult](t, r.send(t, intent.KindJobOutput, intent.JobOutputPayload{ID: rec.ID, N: 50}, nil))
	if len(out.Lines) != 1 || out.Lines[0].Line != "running one" || out.Cursor != "s=1" {
		t.Fatalf("output %+v", out)
	}
	cmd := r.exec.cmds[len(r.exec.cmds)-1]
	if !strings.Contains(cmd, "-u 'jumpgate-job-"+rec.ID+"-*'") || !strings.Contains(cmd, "-o json") {
		t.Fatalf("command %q", cmd)
	}
	rc, res := r.send(t, intent.KindJobOutput, intent.JobOutputPayload{ID: "../../etc"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonInvalidPayload {
		t.Fatalf("a bad id: %s", got)
	}
}

// D-J8 through the shared read (ruling P9): a job's output line is cut to
// logsSinceMaxLine exactly as a logs.since line is.
func TestJobOutputIsBoundedLikeLogsSince(t *testing.T) {
	r, _ := jobsRig(t)
	rec := okResult[intent.JobRecord](t, r.send(t, jobstest.TestKind, map[string]string{}, nil))
	r.exec.res = executor.Result{Stdout: `{"__CURSOR":"s=1","__REALTIME_TIMESTAMP":"1800000000000000","_SYSTEMD_UNIT":"jumpgate-job-` + rec.ID + `-one-0.service","MESSAGE":"` + strings.Repeat("x", 2*logsSinceMaxLine) + `"}` + "\n"}
	out := okResult[intent.JobOutputResult](t, r.send(t, intent.KindJobOutput, intent.JobOutputPayload{ID: rec.ID}, nil))
	if len(out.Lines) != 1 || len(out.Lines[0].Line) != logsSinceMaxLine {
		t.Fatalf("output %d lines, want 1 of %d bytes", len(out.Lines), logsSinceMaxLine)
	}
}

func TestAgentInfoListsPlanVersions(t *testing.T) {
	r, _ := jobsRig(t)
	info := okResult[intent.AgentInfo](t, r.send(t, intent.KindAgentInfo, struct{}{}, nil))
	found := false
	for _, v := range info.PlanVersions {
		found = found || v == jobs.MigrateNamesVersion
	}
	if !found {
		t.Fatalf("plan versions %v", info.PlanVersions)
	}
}
```

In `internal/agent/logs_test.go`, `TestLogsSinceCommandReadsOnlyTheNodeUnits` builds `want` with `" -u '" + u + "'"`: units are quoted now (ruling P9). Every other logs test (`TestLogsSinceRefusesABadCursor`, `TestLogsSinceKeepsTheCursorWhenNothingIsNew`, `TestLogsSinceIsBoundedInBytes`, `TestLogsSinceAfterACursorSkipsOldLines`) passes unchanged: they go through the `logs.since` intent, whose answer does not change.

```go
// internal/server/api_test.go — add
// Review Focus 4, ruling P11: a paired box the controller keeps no wire for
// says in agent.info which names its node uses. A legacy box keeps them, so
// setup never renders units for a jumpgate-node account that does not exist;
// a box with no node yet starts with the new names.
func TestSetupAsksAPairedBoxForItsNames(t *testing.T) {
	requireAgentPeer(t)
	for _, c := range []struct {
		name      string
		node      *catalog.WireConfig // the box's node.json; nil: not set up
		wantNames string
		wantDir   string
	}{
		{"legacy box", &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node-app/369"}, catalog.NamesLegacy, "/var/lib/valve-node-app/369"},
		{"migrated box", &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", Names: catalog.NamesCurrent, DataDir: "/var/lib/jumpgate-node/369"}, catalog.NamesCurrent, "/var/lib/jumpgate-node/369"},
		{"no node yet", nil, catalog.NamesCurrent, "/var/lib/jumpgate-node/369"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var srv *Server
			pairedBoxWith(t, nopExec{}, pairedOpts{onServer: func(s *Server) { srv = s }})
			if c.node != nil {
				b, _ := json.Marshal(*c.node)
				if err := os.WriteFile(filepath.Join(os.Getenv("HOME"), "node.json"), b, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := config.Load()
			if err != nil || cfg.Targets[0].Wire != nil {
				t.Fatalf("the controller must keep no wire for this box: %v", err)
			}
			existing, err := srv.setupExisting(context.Background(), cfg, cfg.Targets[0])
			if err != nil {
				t.Fatal(err)
			}
			// The request asks for the new names; only the box decides.
			got := setupWire(catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", Names: catalog.NamesCurrent}, existing)
			if got.Names != c.wantNames || got.DataDir != c.wantDir {
				t.Fatalf("setup would use names %q dir %s", got.Names, got.DataDir)
			}
		})
	}
}
```

Add `"context"`, `"encoding/json"`, `"os"`, `"path/filepath"` and `internal/config` to `api_test.go`'s imports where they are missing.

Add these helpers to `agent_test.go`:
- `sendSignedBy(t, kind, payload, extra *signer.Key)`: `send` with the extra key's signature appended to `Sigs`.
- `enroll(t, r, k, tier)`: adds a `SignerEntry` to the rig's policy and saves it.
- `setQuestionTier(t, r, id, tier)`: loads `<r.dir>/jobs/<id>.json`, sets `question.tier`, and saves it with `os.WriteFile(…, 0o600)`. The manifest is not signed, so this is a legal test shortcut; the log is untouched.

```go
// internal/agent/policy_test.go — add
func TestAuthoriseWithAnExtraApprovalTier(t *testing.T) {
	ctl, _ := signer.GenerateKey()
	wallet, _ := signer.GenerateKey()
	p := Policy{Signers: []SignerEntry{{Address: ctl.Address().Hex(), Tier: TierRoutine}, {Address: wallet.Address().Hex(), Tier: TierApproval}}}
	if err := p.Authorise(intent.KindJobDecide, ctl.Address(), []eip712.Address{ctl.Address()}, TierApproval); err == nil {
		t.Fatal("an approval question passed with only the controller's signature")
	}
	if err := p.Authorise(intent.KindJobDecide, ctl.Address(), []eip712.Address{ctl.Address(), wallet.Address()}, TierApproval); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/agent/ -count=1`
Expected: FAIL to compile (undefined `cfg.Jobs`, `cfg.NamesPath`, `jobstest`, …).

Run: `go test ./internal/server/ -run TestSetupAsksAPairedBoxForItsNames -count=1`
Expected: FAIL to compile with "srv.setupExisting undefined".

- [ ] **Step 3: Implement the agent side**

`internal/agent/policy.go`: change `Authorise` to `func (p Policy) Authorise(kind string, controller eip712.Address, signers []eip712.Address, extra ...Tier) error`. After resolving `req` from `p.Kinds` and `DefaultRequirements`, do `req = append(append([]Tier{}, req...), extra...)`. Also fall back to `jobs.PlanForKind` for nothing: a job kind with no default requirement and no policy entry stays `unknown_kind`. That is deliberate. The existing call site keeps compiling because `extra` is variadic.

`internal/agent/agent.go`:
- Add `Jobs *jobs.Manager` and `NamesPath string` to `Config` (document `/etc/jumpgate/names.json`).
- In `Handle`, after `LoadPolicy` and before `Authorise`, compute the extra tiers:

  ```go
  var extra []Tier
  if i.Kind == intent.KindJobDecide && a.cfg.Jobs != nil {
  	// A decision needs the tier its question asks for (spec: "a signed
  	// decision of the tier the policy requires"). The question is the
  	// agent's own, read before admission so a refused decision does not
  	// consume the controller's sequence.
  	if pl, rej := decode[intent.JobDecidePayload](env.Payload); rej == nil && jobs.ValidID(pl.ID) {
  		if a.cfg.Jobs.QuestionTier(pl.ID, pl.Decision) == intent.TierApproval {
  			extra = append(extra, TierApproval)
  		}
  	}
  }
  if err := policy.Authorise(i.Kind, i.Controller, signers, extra...); err != nil {
  ```
- Change the dispatch call to `a.dispatch(ctx, i, env, signers, policy)`.

`internal/agent/dispatch.go`:
- Delete `var PlanVersions = []string{}`; `agent.info` sets `PlanVersions: jobs.Versions()`, `Names: w.Names`, `DataDir: w.DataDir` (when set up, and after the names marker is applied, so a migrated box reports the new names even if its `node.json` is stale).
- In the `KindLogsSince` case, call the generalised read and wrap its answer (ruling P9):

  ```go
  hits, cursor, rej, err := logsSince(ctx, ex, ops.NodeUnits(w.NodeNames()), pl.Cursor, pl.N, a.cfg.Now(),
  	func(e journalEntry) logwatch.Hit { return classifyLine(e.unit, e.line, e.at) })
  if rej != nil {
  	return nil, 0, rej
  }
  if err != nil {
  	return failed(err)
  }
  return mustJSON(LogsSinceResult{Hits: hits, Cursor: cursor}), intent.StatusOK, nil
  ```
- `dispatch(ctx context.Context, i intent.Intent, env intent.Envelope, signers []eip712.Address, p Policy)`; `payload := env.Payload`.
- Before the `loadNode` that gates node kinds: `if isJobKind(i.Kind) { return a.dispatchJob(ctx, i, env, signers) }`. A job kind does not need a set-up node: `node.migrate-names` reports a missing node.json itself, in preflight.
- In `KindServiceAction`, before running: `if a.cfg.Jobs != nil { if j, busy := a.cfg.Jobs.Active(); busy { return nil, 0, reject(intent.ReasonJobActive, fmt.Sprintf("job %s (%s) is %s and owns the node services until it ends", j.ID, j.Kind, j.State)) } }`.
- `loadNode`: after unmarshalling, apply the marker:

  ```go
  if a.cfg.NamesPath != "" {
  	mk, ok, err := jobs.ReadNamesMarker(a.cfg.NamesPath)
  	if err != nil {
  		return w, reject(intent.ReasonValidation, "names.json: "+err.Error())
  	}
  	if ok {
  		mk.Apply(&w)
  	}
  }
  ```
  Apply it before `ValidateDataDir` and `ValidateNames`.

```go
// internal/agent/jobs.go
package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
)

// ReconcileEvery is how often RunJobs folds finished steps into their logs (D-J2).
var ReconcileEvery = 5 * time.Second

func isJobKind(kind string) bool {
	switch kind {
	case intent.KindJobList, intent.KindJobRead, intent.KindJobOutput, intent.KindJobLease, intent.KindJobAdvance, intent.KindJobDecide:
		return true
	}
	_, ok := jobs.PlanForKind(kind)
	return ok
}

func jobFailure(err error) ([]byte, uint8, *Reject) {
	var r *jobs.Refusal
	if errors.As(err, &r) {
		return nil, 0, reject(r.Code, r.Message)
	}
	return failed(err)
}

func idPayload(id string) *Reject {
	if !jobs.ValidID(id) {
		return reject(intent.ReasonInvalidPayload, fmt.Sprintf("%q is not a job id", id))
	}
	return nil
}

// dispatchJob answers the job kinds. Reconcile runs first, so every answer
// reflects units that finished since the last pass.
func (a *Agent) dispatchJob(ctx context.Context, i intent.Intent, env intent.Envelope, signers []eip712.Address) ([]byte, uint8, *Reject) {
	m := a.cfg.Jobs
	if m == nil {
		return nil, 0, reject(intent.ReasonJobsUnsupported, "this agent runs without durable jobs (no job store)")
	}
	_ = m.Reconcile(ctx)
	switch i.Kind {
	case intent.KindJobList:
		res, err := m.List()
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(res), intent.StatusOK, nil
	case intent.KindJobRead:
		pl, rej := decode[intent.JobReadPayload](env.Payload)
		if rej == nil {
			rej = idPayload(pl.ID)
		}
		if rej != nil {
			return nil, 0, rej
		}
		res, err := m.Read(pl.ID, pl.From)
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(res), intent.StatusOK, nil
	case intent.KindJobOutput:
		pl, rej := decode[intent.JobOutputPayload](env.Payload)
		if rej == nil {
			rej = idPayload(pl.ID)
		}
		if rej != nil {
			return nil, 0, rej
		}
		res, rej, err := jobOutput(ctx, a.cfg.Exec, pl, a.cfg.Now())
		if rej != nil {
			return nil, 0, rej
		}
		if err != nil {
			return failed(err)
		}
		return mustJSON(res), intent.StatusOK, nil
	case intent.KindJobLease:
		pl, rej := decode[intent.JobLeasePayload](env.Payload)
		if rej == nil {
			rej = idPayload(pl.ID)
		}
		if rej != nil {
			return nil, 0, rej
		}
		rec, err := m.Lease(ctx, pl.ID, pl.Op, i.Controller, env)
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(rec), intent.StatusOK, nil
	case intent.KindJobAdvance:
		pl, rej := decode[intent.JobAdvancePayload](env.Payload)
		if rej == nil {
			rej = idPayload(pl.ID)
		}
		if rej != nil {
			return nil, 0, rej
		}
		rec, err := m.Advance(ctx, pl.ID, pl.Step, pl.Attempt, i.Controller, env)
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(rec), intent.StatusOK, nil
	case intent.KindJobDecide:
		pl, rej := decode[intent.JobDecidePayload](env.Payload)
		if rej == nil {
			rej = idPayload(pl.ID)
		}
		if rej != nil {
			return nil, 0, rej
		}
		rec, err := m.Decide(ctx, pl.ID, pl.Decision, pl.Option, i.Controller, signers, env)
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(rec), intent.StatusOK, nil
	}
	rec, err := m.Start(ctx, i.Kind, env.Payload, i.Controller, env)
	if err != nil {
		return jobFailure(err)
	}
	return mustJSON(rec), intent.StatusOK, nil
}

// jobOutput is a job's journal through logsSince, the one bounded read
// (D-J8, ruling P9). The units are matched by the job's own pattern, built
// from a validated id.
func jobOutput(ctx context.Context, ex executor.Executor, pl intent.JobOutputPayload, now time.Time) (intent.JobOutputResult, *Reject, error) {
	lines, cursor, rej, err := logsSince(ctx, ex, []string{jobs.UnitPattern(pl.ID)}, pl.Cursor, pl.N, now,
		func(e journalEntry) intent.JobOutputLine { return intent.JobOutputLine{Unit: e.unit, Line: e.line, At: e.at} })
	if rej != nil || err != nil {
		return intent.JobOutputResult{}, rej, err
	}
	return intent.JobOutputResult{Lines: lines, Cursor: cursor}, nil, nil
}

// RunJobs reconciles at once (an agent restart folds what finished while it
// was down), then every ReconcileEvery, and prunes once a day.
func (a *Agent) RunJobs(ctx context.Context) {
	if a.cfg.Jobs == nil {
		return
	}
	_ = a.cfg.Jobs.Reconcile(ctx)
	_ = a.cfg.Jobs.Prune()
	tick := time.NewTicker(ReconcileEvery)
	defer tick.Stop()
	lastPrune := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			_ = a.cfg.Jobs.Reconcile(ctx)
			if time.Since(lastPrune) > 24*time.Hour {
				_ = a.cfg.Jobs.Prune()
				lastPrune = time.Now()
			}
		}
	}
}
```

The journal unit label for a step line is `_SYSTEMD_UNIT`. The `UNIT` override in `readJournalJSON` applies to systemd's own "Started …" lines, which is what we want here too.

In `internal/agent/logs.go`, generalise the read that Task 1 left taking the node's units (ruling P9). `logsSinceCommand` keeps Task 1's signature and quotes each unit: change `b.WriteString(" -u " + u)` to `b.WriteString(" -u '" + u + "'")`, and say in its comment that the units are the agent's own (the node's units, or a job's pattern from a validated id) and quoted because a pattern is a glob. Replace `logsSince` with:

```go
// logsSince is the one bounded journal read, behind logs.since and job.output
// (D-J8). The answer is bounded three ways: journalctl reads at most
// intent.LogsMaxN lines; each line is cut to logsSinceMaxLine and the answer
// stops near logsSinceMaxBytes, its cursor at the last line sent so the next
// call resumes there; and after a cursor, lines older than logsSinceMaxAge
// are passed over, so a cursor cannot page back through the box's history
// (without one the answer is the last n lines, as logs.read). item turns one
// journal line, already cut, into what the answer carries; the byte bound is
// measured on item's encoding, since escaping can make a line several times
// longer on the wire than in the journal.
func logsSince[T any](ctx context.Context, ex executor.Executor, units []string, cursor string, n int, now time.Time, item func(journalEntry) T) ([]T, string, *Reject, error) {
	cmd, err := logsSinceCommand(units, cursor, n)
	if err != nil {
		return nil, "", reject(intent.ReasonInvalidPayload, err.Error()), nil
	}
	res, err := ex.Run(ctx, cmd, nil)
	if err != nil {
		return nil, "", nil, err
	}
	out, next := []T{}, cursor
	oldest, size := now.Add(-logsSinceMaxAge), 0
	for _, e := range readJournalJSON(res.Stdout) {
		if cursor != "" && e.at.Before(oldest) {
			next = e.cursor
			continue
		}
		e.line = cutLine(e.line, logsSinceMaxLine)
		it := item(e)
		b, _ := json.Marshal(it)
		sz := len(b) + 1
		if size+sz > logsSinceMaxBytes && len(out) > 0 {
			break // the rest waits for the next call
		}
		size += sz
		out = append(out, it)
		if e.cursor != "" {
			next = e.cursor
		}
	}
	return out, next, nil, nil
}
```

`logs.go` no longer names the node's units (the dispatch case passes them), so drop the `ops` and `catalog` imports Task 1 left there; it keeps `logwatch` for `LogsSinceResult` and `classifyLine`.

In `internal/server/api.go`, ask the box when the controller keeps no wire for it (ruling P11). Task 1's `setupWire` stays as it is:

```go
// setupExisting is the wire setup keeps a box's names and data dir from: the
// controller's own copy, or, when it keeps none, what a paired box's agent
// reports in agent.info (a box paired from another controller, or a config
// restored from before the box was set up). Review Focus 4: without this a
// legacy box would be treated as new, given the jumpgate-node names, and set
// up with units for an account that does not exist. A box that cannot answer
// refuses the setup rather than guess.
func (s *Server) setupExisting(ctx context.Context, cfg config.Config, t config.Target) (*catalog.WireConfig, error) {
	if t.Wire != nil || t.Agent == nil {
		return t.Wire, nil
	}
	info, err := decodeResult[intent.AgentInfo](s.agentResult(ctx, cfg, t, intent.KindAgentInfo, struct{}{}))
	if err != nil {
		return nil, fmt.Errorf("ask the box which names its node uses: %w", err)
	}
	if !info.SetUp {
		return nil, nil
	}
	// An agent from before sub-project 2 sends no names: every box it set up
	// is legacy, which Names "" already says.
	return &catalog.WireConfig{ChainID: info.ChainID, Names: info.Names, DataDir: info.DataDir}, nil
}
```

In `handleStartSetup`, replace Task 1's `wire = setupWire(wire, target.Wire)` with:

```go
existing, err := s.setupExisting(r.Context(), cfg, target)
if err != nil {
	writeError(w, http.StatusBadGateway, err.Error())
	return
}
wire = setupWire(wire, existing)
```

- [ ] **Step 4: Wire the CLI**

In `cmd/jumpgate/cli_agent.go`:
- Add flags `job := fset.String("job", "", "job id (job-step)")`, `step := fset.String("step", "", "step id (job-step)")`, `attempt := fset.Int("attempt", -1, "attempt (job-step)")`, and the case `"job-step": err = agentJobStep(*state, *conf, *job, *step, *attempt)`. Update the usage line to `init|enroll|run|reset-replay|job-step`.
- Add:

  ```go
  // agentJobStep is the body of a job step's transient unit. It holds no key:
  // it reads its manifest and writes one result file (spec D-J4).
  func agentJobStep(stateDir, configDir, job, step string, attempt int) error {
  	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
  	defer stop()
  	return jobs.RunStep(ctx, filepath.Join(stateDir, "jobs"), jobs.StepArgs{Job: job, Step: step, Attempt: attempt}, jobs.Env{
  		Exec: executor.NewLocal(), NodePath: filepath.Join(configDir, "node.json"),
  		NamesPath: filepath.Join(configDir, "names.json"), Out: os.Stdout,
  	})
  }
  ```
- In `agentRun`, after loading the key:

  ```go
  self, err := os.Executable()
  if err != nil {
  	return err
  }
  ex := executor.NewLocal()
  mgr, err := jobs.NewManager(jobs.ManagerConfig{
  	Dir: filepath.Join(stateDir, "jobs"), Key: k, Runner: jobs.DefaultRunner(ex), Version: buildinfo.Version(),
  	StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv(self, stateDir, configDir, a) },
  })
  if err != nil {
  	return err
  }
  ```
  Pass `Exec: ex, Jobs: mgr, NamesPath: filepath.Join(configDir, "names.json")` into `agent.New`. After `signal.NotifyContext`, add `go a.RunJobs(ctx)`.

A test in `cmd/jumpgate/cli_test.go` checks that `cmdAgent([]string{"job-step", "--job", "nothex", "--step", "one", "--attempt", "0"})` returns 1 with a "bad step arguments" or "no such job" message. It exercises the flag wiring without a manifest.

- [ ] **Step 5: Run the tests and cross-compile**

Run: `go test ./internal/agent/ ./internal/jobs/... ./cmd/jumpgate/ ./internal/server/ -count=1 -race && GOOS=linux GOARCH=arm64 go build ./... && GOOS=windows go vet ./...`
Expected: PASS, including Task 1's `TestSetupKeepsTheBoxNames` and every `TestLogsSince*`; both cross builds clean.

- [ ] **Step 6: Commit**

```bash
git add internal/agent/jobs.go internal/agent/jobs_test.go internal/agent/agent.go internal/agent/agent_test.go \
  internal/agent/dispatch.go internal/agent/policy.go internal/agent/policy_test.go internal/agent/logs.go internal/agent/logs_test.go \
  cmd/jumpgate/cli_agent.go cmd/jumpgate/cli_test.go internal/server/api.go internal/server/api_test.go
git commit -m "feat(agent): durable job intents, decision tiers, job output and the reconcile loop

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

## Phase B: the controller (Tasks 10–12)

The per-target route list is frozen (`internal/server/freeze_test.go`, TUI spec A5): "a new node operation is a new intent kind … not a new route". The job routes therefore live under `/api/jobs/…`, keyed by target id, and never under `/api/targets/{id}/…`. The freeze test stays unchanged; Task 11 asserts that it still passes.

### Task 10: API wire types and codes for jobs

Covers the TUI spec's reserved `jobs` field (D27) and the error contract for the new routes.

**Files:**
- Create: `internal/api/jobs.go`, `internal/api/jobs_test.go`
- Modify: `internal/api/errors.go` (codes and the registry), `internal/api/fleet.go` (the `JobsSummary` comment)

**Interfaces:**
- Consumes: Task 4's reasons.
- Produces (package `api`):
  - `type JobLease struct { Owner, Label string; HeartbeatAt *time.Time; Fresh, Mine bool }`;
  - `type JobStepView struct { ID, Title, State string }` (State: `done`, `running`, `asking`, `waiting`, `pending`, `stopped`);
  - `type JobOption struct { ID, Label, Detail string }`;
  - `type JobQuestion struct { ID, Prompt string; Options []JobOption; Tier string; AskedAt time.Time; Signable bool }`;
  - `type JobView struct { Target, ID, Kind, Title, State string; Step int; Steps []JobStepView; Attempt int; Question *JobQuestion; Lease JobLease; CreatedAt, UpdatedAt time.Time; Error string; Driving bool }`;
  - `type JobsError struct { Target string; Error Error }`;
  - `type JobsInbox struct { At time.Time; Jobs []JobView; Errors []JobsError }` with `func (JobsInbox) NeedsDecision() int`;
  - `type JobLogLine struct { Index uint64; Kind string; At time.Time; Text string }`;
  - `type JobDetail struct { Job JobView; Log []JobLogLine }`;
  - `type JobOutputLine struct { Unit, Line string; At time.Time }`;
  - `type JobDecision struct { Decision, Option string }`;
  - `type JobStart struct { Kind string; Inputs json.RawMessage }`;
  - codes `CodeJobNotFound = "job_not_found"`, `CodeLeaseHeld = "lease_held"`, `CodeJobActive = "job_active"`, `CodeJobsUnsupported = "jobs_unsupported"`, `CodeJobState = "job_state"`, `CodeApprovalUnavailable = "approval_unavailable"`, `CodeJobLogInvalid = "job_log_invalid"`, `CodeUnknownPlan = "unknown_plan"` (ruling P12: Task 4's sixth reason gets its own code instead of the generic `rejected`), `CodeJobKindNotAllowed = "job_kind_not_allowed"` (ruling P7: `POST /api/jobs/{target}` refuses a kind outside its allowlist, 400).

- [ ] **Step 1: Write the failing tests**

```go
// internal/api/jobs_test.go
package api

import (
	"encoding/json"
	"testing"
)

func TestInboxCountsJobsThatNeedADecision(t *testing.T) {
	in := JobsInbox{Jobs: []JobView{{State: "needs-decision"}, {State: "running"}, {State: "needs-decision"}}}
	if in.NeedsDecision() != 2 {
		t.Fatal(in.NeedsDecision())
	}
}

func TestJobViewWireNames(t *testing.T) {
	b, _ := json.Marshal(JobView{Target: "box-a", ID: "0123456789abcdef", Question: &JobQuestion{ID: "q", Signable: true}})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"target", "id", "kind", "title", "state", "step", "steps", "attempt", "question", "lease", "createdAt", "updatedAt", "driving"} {
		if _, ok := m[k]; !ok {
			t.Errorf("JobView JSON lacks %q: %s", k, b)
		}
	}
}

func TestJobCodesAreRegisteredWithTheirExitClass(t *testing.T) {
	for c, exit := range map[Code]int{
		CodeJobNotFound: ExitFailed, CodeLeaseHeld: ExitFailed, CodeJobActive: ExitFailed, CodeJobsUnsupported: ExitFailed,
		CodeJobState: ExitFailed, CodeApprovalUnavailable: ExitUsage, CodeJobLogInvalid: ExitSecurity,
		CodeUnknownPlan: ExitFailed, CodeJobKindNotAllowed: ExitUsage,
	} {
		if !Known(c) || c.Exit() != exit || HintFor(c) == "" {
			t.Errorf("%s: known %v exit %d hint %q", c, Known(c), c.Exit(), HintFor(c))
		}
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/api/ -count=1`
Expected: FAIL to compile.

- [ ] **Step 3: Implement**

```go
// internal/api/jobs.go
package api

import (
	"encoding/json"
	"time"
)

// JobLease is who drives a job, as this server sees it. Fresh is judged by
// the box's clock (the agent reports its time with every list).
type JobLease struct {
	Owner       string     `json:"owner,omitempty"`
	Label       string     `json:"label,omitempty"` // "this controller" when Mine
	HeartbeatAt *time.Time `json:"heartbeatAt,omitempty"`
	Fresh       bool       `json:"fresh"`
	Mine        bool       `json:"mine"`
}

type JobStepView struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	State string `json:"state"` // done, running, asking, waiting, pending, stopped
}

type JobOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// JobQuestion is a decision point. Signable is false while its tier has no
// signer this server can use (the approval tier, until it ships).
type JobQuestion struct {
	ID       string      `json:"id"`
	Prompt   string      `json:"prompt"`
	Options  []JobOption `json:"options"`
	Tier     string      `json:"tier"`
	AskedAt  time.Time   `json:"askedAt"`
	Signable bool        `json:"signable"`
}

// JobView is one durable job on one box.
type JobView struct {
	Target    string        `json:"target"`
	ID        string        `json:"id"`
	Kind      string        `json:"kind"`
	Title     string        `json:"title"`
	State     string        `json:"state"`
	Step      int           `json:"step"`
	Steps     []JobStepView `json:"steps"`
	Attempt   int           `json:"attempt"`
	Question  *JobQuestion  `json:"question,omitempty"`
	Lease     JobLease      `json:"lease"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
	Error     string        `json:"error,omitempty"`
	Driving   bool          `json:"driving"` // this server is advancing it now
}

type JobsError struct {
	Target string `json:"target"`
	Error  Error  `json:"error"`
}

// JobsInbox is GET /api/jobs and each /api/jobs/stream event: every job on
// every paired box, and the boxes that could not be asked.
type JobsInbox struct {
	At     time.Time   `json:"at"`
	Jobs   []JobView   `json:"jobs"`
	Errors []JobsError `json:"errors"`
}

// NeedsDecision is the fleet-wide ⚑ count.
func (in JobsInbox) NeedsDecision() int {
	n := 0
	for _, j := range in.Jobs {
		if j.State == "needs-decision" {
			n++
		}
	}
	return n
}

// JobLogLine is one verified log entry, described for a person.
type JobLogLine struct {
	Index uint64    `json:"index"`
	Kind  string    `json:"kind"`
	At    time.Time `json:"at"`
	Text  string    `json:"text"`
}

// JobDetail is GET /api/jobs/{target}/{job}. Log holds only entries that
// verified; a log that does not is an error (job_log_invalid), never a partial list.
type JobDetail struct {
	Job JobView      `json:"job"`
	Log []JobLogLine `json:"log"`
}

type JobOutputLine struct {
	Unit string    `json:"unit"`
	Line string    `json:"line"`
	At   time.Time `json:"at"`
}

// JobDecision is POST /api/jobs/{target}/{job}/decide.
type JobDecision struct {
	Decision string `json:"decision"`
	Option   string `json:"option"`
}

// JobStart is POST /api/jobs/{target}.
type JobStart struct {
	Kind   string          `json:"kind"`
	Inputs json.RawMessage `json:"inputs,omitempty"`
}
```

In `internal/api/errors.go`, add the constants under a `// Durable jobs.` comment, and add these registry entries:

```go
	CodeJobNotFound:         {"`jumpgate jobs` lists the jobs on every box", ExitFailed},
	CodeLeaseHeld:           {"another controller drives this job; you can watch it, and adopt it once it has been silent for 90 s (`jumpgate jobs adopt HOST JOB`, or `a` in the jobs inbox)", ExitFailed},
	CodeJobActive:           {"one job runs at a time on a box; finish or decide it in the jobs inbox first", ExitFailed},
	CodeJobsUnsupported:     {"durable jobs need a Linux box with systemd; the message says what is missing", ExitFailed},
	CodeJobState:            {"the job moved on since you looked; reload it", ExitFailed},
	CodeApprovalUnavailable: {"this decision needs an approval-tier signature (a hardware or browser wallet), which this jumpgate cannot make yet", ExitUsage},
	CodeJobLogInvalid:       {"this job's history on the box does not verify against the box's agent key: do not act on it; inspect /var/lib/jumpgate/jobs on the box, and re-pair the box if its key changed", ExitSecurity},
	CodeUnknownPlan:         {"this box's agent does not know that job's plan (it was downgraded?); re-pair the box to upgrade its agent", ExitFailed},
	CodeJobKindNotAllowed:   {"this server starts only the jobs it knows (for example `jumpgate migrate-names HOST`); other node operations have their own commands", ExitUsage},
```

In `internal/api/fleet.go`, change the `JobsSummary` comment to: "counts a box's durable jobs. Null when the box is SSH-only, its agent cannot run jobs, or it has not been asked yet; `Running` counts waiting and running jobs."

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/api/ -count=1`
Expected: PASS (including `TestEveryCodeIsSnakeCaseAndRegistered`).

- [ ] **Step 5: Commit**

```bash
git add internal/api/jobs.go internal/api/jobs_test.go internal/api/errors.go internal/api/fleet.go
git commit -m "feat(api): job inbox, detail and decision wire types; job error codes

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 11: The jobs hub — list, verify, inbox, detail, output, fleet `jobs`

Covers spec "Adoption: any paired controller lists jobs, reads the manifest and log … watches read-only", "Clients resume with `journalctl --after-cursor`", the TUI spec's `jobs` on fleet rows (D27), J11, and Review Focus 5 (controller side).

**Files:**
- Create: `internal/server/jobs.go`, `internal/server/jobs_test.go`
- Modify: `internal/server/server.go` (`jobs *jobsHub` field; `New`; stop it in `ListenAndServe`/`ServeUnix`), `internal/server/api.go` (routes), `internal/server/node.go` (`classifyNodeError`: job reasons and `errJobLogInvalid`), `internal/server/fleet.go` (`snapshot` fills `row.Jobs`; `round` asks the hub to refresh; `jobsChanged`), `internal/server/node_test.go` (`pairedOpts.agent` hook)
- Test: `internal/server/jobs_test.go` (the new tests, `TestFleetRowsCarryJobCounts` among them), `internal/server/freeze_test.go` and `internal/server/fleet_test.go` (both unchanged, must pass; the fleet test's `Jobs == nil` assertions stay true)

**Interfaces:**
- Consumes: Tasks 3, 4, 9, 10; `sendIntent`, `agentResult`, `decodeResult`, `startSSE`, `writeNodeError`, `findTarget`, `loadConfig`.
- Produces (package `server`, unexported): `type jobsHub`; `newJobsHub(s *Server) *jobsHub`; `(h) refresh(ctx context.Context, force bool)`; `(h) inbox() api.JobsInbox`; `(h) summary(target string) *api.JobsSummary`; `(h) subscribe() (chan api.JobsInbox, func())`; `(h) readVerified(ctx context.Context, cfg config.Config, t config.Target, id string) (intent.JobRecord, []joblog.Entry, int64, error)`; `(h) view(target string, r intent.JobRecord, agentTime int64) api.JobView`; `(h) mine(r intent.JobRecord) bool`; `(h) afterList` hook field `func(t config.Target, jobs []intent.JobRecord)` (Task 12 sets it); `(h) stopWithin(d time.Duration)`; `var errJobLogInvalid`; variables `jobsListEvery = 10 * time.Second`, `jobsOutputEvery = 2 * time.Second`.
- Routes: `GET /api/jobs`, `GET /api/jobs/stream`, `GET /api/jobs/{target}/{job}`, `GET /api/jobs/{target}/{job}/output/stream`.

- [ ] **Step 1: Add the test hook and write the failing tests**

In `internal/server/node_test.go`, add a field to `pairedOpts`: `agent func(home string, c *agent.Config)`. In `pairedBoxWith`, call it just before `agent.New(...)`, building the `agent.Config` in a variable first. Also add `srv *Server` to what `pairedBoxWith` can hand back: tests here use `o.onServer` to capture it.

```go
// internal/server/jobs_test.go
package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
)

// boxRig is a paired box whose agent runs jobs in-process, on a clock the
// test moves. It returns the server, its token, the agent's home and the clock.
type boxRig struct {
	ts    *httptest.Server
	token string
	srv   *Server
	home  string
	mu    sync.Mutex
	now   time.Time
}

func (b *boxRig) clock() time.Time { b.mu.Lock(); defer b.mu.Unlock(); return b.now }
func (b *boxRig) advance(d time.Duration) { b.mu.Lock(); b.now = b.now.Add(d); b.mu.Unlock() }

func newBoxRig(t *testing.T) *boxRig {
	t.Helper()
	jobstest.RegisterPlan(t)
	b := &boxRig{now: time.Now()}
	prevDrive, prevList := jobsDriveEvery, jobsListEvery
	jobsDriveEvery, jobsListEvery = 20*time.Millisecond, 0
	t.Cleanup(func() { jobsDriveEvery, jobsListEvery = prevDrive, prevList })
	ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{
		setUp: true,
		agent: func(home string, c *agent.Config) {
			b.home = home
			dir := filepath.Join(home, "jobs")
			run := &jobstest.Runner{}
			run.Run = jobstest.InProcess(dir, jobs.Env{Exec: c.Exec, NodePath: c.NodePath, Out: io.Discard})
			m, err := jobs.NewManager(jobs.ManagerConfig{Dir: dir, Key: c.Key, Runner: run, Now: b.clock,
				StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv("/jumpgate", home, home, a) }})
			if err != nil {
				t.Fatal(err)
			}
			c.Jobs, c.Now = m, b.clock
			p, _ := agent.LoadPolicy(c.PolicyPath)
			p.Kinds = map[string][]agent.Tier{jobstest.TestKind: {agent.TierRoutine}}
			_ = p.Save(c.PolicyPath)
		},
		onServer: func(s *Server) { b.srv = s },
	})
	b.ts, b.token = ts, token
	t.Cleanup(func() { b.srv.jobs.stopWithin(time.Second) })
	return b
}

func (b *boxRig) do(t *testing.T, method, path string, body any, out any) int {
	t.Helper()
	var r io.Reader
	if body != nil {
		j, _ := json.Marshal(body)
		r = strings.NewReader(string(j))
	}
	req, _ := http.NewRequest(method, b.ts.URL+path, r)
	req.Header.Set("Authorization", "Bearer "+b.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		_ = json.NewDecoder(res.Body).Decode(out)
	}
	return res.StatusCode
}

// startJob creates a job straight through the agent (Task 12 adds the route).
func (b *boxRig) startJob(t *testing.T, inputs string) intent.JobRecord {
	t.Helper()
	cfg, _ := b.srv.loadConfig()
	tg, _ := findTarget(cfg, "box")
	rec, err := decodeResult[intent.JobRecord](b.srv.agentResult(context.Background(), cfg, tg, jobstest.TestKind, json.RawMessage(inputs)))
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestInboxListsEveryPairedBoxsJobs(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`)
	var in api.JobsInbox
	if code := b.do(t, "GET", "/api/jobs", nil, &in); code != 200 {
		t.Fatalf("GET /api/jobs = %d", code)
	}
	if len(in.Jobs) != 1 || in.Jobs[0].ID != rec.ID || in.Jobs[0].Target != "box" || !in.Jobs[0].Lease.Mine || !in.Jobs[0].Lease.Fresh {
		t.Fatalf("inbox %+v", in)
	}
	if len(in.Jobs[0].Steps) != 2 || in.Jobs[0].Steps[0].State != "waiting" {
		t.Fatalf("steps %+v", in.Jobs[0].Steps)
	}
}

func TestJobDetailVerifiesTheLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`)
	var d api.JobDetail
	if code := b.do(t, "GET", "/api/jobs/box/"+rec.ID, nil, &d); code != 200 || len(d.Log) != 1 || d.Log[0].Kind != "job.created" {
		t.Fatalf("detail %d %+v", code, d)
	}
}

// Review Focus 5: an edited entry on the box is refused, not shown.
func TestJobDetailRefusesABrokenLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`)
	logPath := filepath.Join(b.home, "jobs", rec.ID+".log")
	raw, _ := os.ReadFile(logPath)
	var j intent.JobLogEntryJSON
	_ = json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &j)
	j.Body = []byte(`{"outcome":"forged"}`)
	forged, _ := json.Marshal(j)
	_ = os.WriteFile(logPath, append(forged, '\n'), 0o600)

	var e api.Error
	code := b.do(t, "GET", "/api/jobs/box/"+rec.ID, nil, &e)
	// The agent itself notices first and reports the job damaged; either way
	// the controller shows no history it could not verify.
	if code == 200 || (e.Code != api.CodeJobLogInvalid && e.Code != api.CodeJobState) {
		t.Fatalf("a forged log answered %d %+v", code, e)
	}
}

func TestFleetRowsCarryJobCounts(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	b.startJob(t, `{}`)
	b.srv.jobs.refresh(context.Background(), true)
	s := b.srv.jobs.summary("box")
	if s == nil || s.Running != 1 || s.NeedsDecision != 0 {
		t.Fatalf("summary %+v", s)
	}
	var f api.Fleet
	b.do(t, "GET", "/api/fleet", nil, &f)
	if row, ok := f.Row("box"); !ok || row.Jobs == nil || row.Jobs.Running != 1 {
		t.Fatalf("fleet row jobs %+v", row.Jobs)
	}
}

func TestJobsStreamSendsTheInbox(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	b.startJob(t, `{}`)
	frames, _ := readFrames(t, b.ts, b.token, "/api/jobs/stream", 1)
	var in api.JobsInbox
	if err := json.Unmarshal([]byte(strings.TrimPrefix(frames[0], "data: ")), &in); err != nil || len(in.Jobs) != 1 {
		t.Fatalf("frame %q: %v", frames[0], err)
	}
}
```

`readFrames` already exists in `node_test.go`; it returns each frame as read. If it strips the `data: ` prefix, drop the `TrimPrefix`.

In `fleet_test.go`, the existing assertions that a fake-probed box has `Jobs == nil` stay true: a box without a hub reading has no summary.

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/server/ -run 'Inbox|JobDetail|FleetRowsCarryJobCounts|JobsStream' -count=1`
Expected: FAIL to compile (`s.jobs` undefined, `jobsDriveEvery` undefined: declare `jobsDriveEvery` here in `jobs.go` so this task compiles; Task 12 uses it).

- [ ] **Step 3: Implement `internal/server/jobs.go`**

```go
// internal/server/jobs.go
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/jobs"
)

// Cadences (variables so tests shorten them). D-J2.
var (
	jobsListEvery   = 10 * time.Second // a refresh within this is reused
	jobsDriveEvery  = 5 * time.Second
	jobsRenewEvery  = 20 * time.Second
	jobsOutputEvery = 2 * time.Second
)

// errJobLogInvalid is a log that does not verify against the box's agent key.
var errJobLogInvalid = errors.New("the job's log does not verify")

type jobKey struct{ target, id string }

type boxJobs struct {
	jobs        []intent.JobRecord
	agentTime   int64
	err         *api.Error
	unsupported bool // the agent cannot run jobs (or predates them): no summary
}

type verifiedLog struct {
	entries []joblog.Entry
	head    [32]byte
}

// jobsHub is this server's view of every paired box's durable jobs: it lists
// them (lazily, at most every jobsListEvery), verifies their logs, serves the
// inbox, and (Task 12) drives the jobs whose lease this controller holds.
type jobsHub struct {
	s      *Server
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// afterList runs after each box is listed, outside mu (Task 12 resumes
	// this controller's own jobs there).
	afterList func(t config.Target, jobs []intent.JobRecord)

	mu       sync.Mutex
	boxes    map[string]*boxJobs
	listedAt time.Time
	listing  chan struct{}
	subs     map[chan api.JobsInbox]struct{}
	logs     map[jobKey]*verifiedLog
	driving  map[jobKey]context.CancelFunc
	stopped  bool
}

func newJobsHub(s *Server) *jobsHub {
	ctx, cancel := context.WithCancel(context.Background())
	return &jobsHub{s: s, ctx: ctx, cancel: cancel, boxes: map[string]*boxJobs{}, subs: map[chan api.JobsInbox]struct{}{},
		logs: map[jobKey]*verifiedLog{}, driving: map[jobKey]context.CancelFunc{}}
}

func (h *jobsHub) me() (eip712.Address, bool) {
	if h.s.cfg.Signer == nil {
		return eip712.Address{}, false
	}
	return h.s.cfg.Signer.Address(), true
}

// mine reports whether this controller's key holds r's lease.
func (h *jobsHub) mine(r intent.JobRecord) bool {
	me, ok := h.me()
	owner, err := eip712.ParseAddress(r.Lease.Owner)
	return ok && err == nil && owner == me
}

// refresh lists every paired box's jobs. Callers that arrive while one runs
// wait for it; a refresh newer than jobsListEvery is reused unless force.
func (h *jobsHub) refresh(ctx context.Context, force bool) {
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return
	}
	if wait := h.listing; wait != nil {
		h.mu.Unlock()
		select {
		case <-wait:
		case <-ctx.Done():
		}
		return
	}
	if !force && time.Since(h.listedAt) < jobsListEvery {
		h.mu.Unlock()
		return
	}
	done := make(chan struct{})
	h.listing = done
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.listing, h.listedAt = nil, time.Now()
		h.mu.Unlock()
		close(done)
		h.publish()
	}()

	cfg, err := h.s.loadConfig()
	if err != nil {
		return
	}
	sem := make(chan struct{}, fleetWorkers)
	var wg sync.WaitGroup
	keep := map[string]bool{}
	for _, t := range cfg.Targets {
		if t.Agent == nil {
			continue // SSH-only: no agent, no jobs
		}
		keep[t.ID] = true
		wg.Add(1)
		sem <- struct{}{}
		go func(t config.Target) {
			defer wg.Done()
			defer func() { <-sem }()
			pctx, cancel := context.WithTimeout(ctx, fleetProbeTimeout)
			defer cancel()
			h.listBox(pctx, cfg, t)
		}(t)
	}
	wg.Wait()
	h.mu.Lock()
	for id := range h.boxes {
		if !keep[id] {
			delete(h.boxes, id)
		}
	}
	h.mu.Unlock()
}

func (h *jobsHub) listBox(ctx context.Context, cfg config.Config, t config.Target) {
	res, err := decodeResult[intent.JobListResult](h.s.agentResult(ctx, cfg, t, intent.KindJobList, struct{}{}))
	box := &boxJobs{jobs: res.Jobs, agentTime: res.AgentTime}
	var rej *agentRejected
	switch {
	case errors.As(err, &rej) && (rej.Code == intent.ReasonJobsUnsupported || rej.Code == intent.ReasonUnknownKind):
		box = &boxJobs{unsupported: true} // an old agent, or a box that cannot run jobs
	case err != nil:
		_, e := apiErrorFor(err)
		box = &boxJobs{err: &e}
		h.mu.Lock()
		if old := h.boxes[t.ID]; old != nil {
			box.jobs, box.agentTime = old.jobs, old.agentTime // keep what we knew; the error says it is old
		}
		h.mu.Unlock()
	}
	h.mu.Lock()
	h.boxes[t.ID] = box
	h.mu.Unlock()
	if err == nil && h.afterList != nil {
		h.afterList(t, res.Jobs)
	}
}

// setRecord updates one job in the cache (after a read or a mutation).
func (h *jobsHub) setRecord(target string, r intent.JobRecord, agentTime int64) {
	h.mu.Lock()
	box := h.boxes[target]
	if box == nil {
		box = &boxJobs{}
		h.boxes[target] = box
	}
	box.agentTime = max(box.agentTime, agentTime)
	for i := range box.jobs {
		if box.jobs[i].ID == r.ID {
			box.jobs[i] = r
			h.mu.Unlock()
			h.publish()
			return
		}
	}
	box.jobs = append([]intent.JobRecord{r}, box.jobs...)
	h.mu.Unlock()
	h.publish()
}

func (h *jobsHub) inbox() api.JobsInbox {
	h.mu.Lock()
	defer h.mu.Unlock()
	in := api.JobsInbox{At: time.Now(), Jobs: []api.JobView{}, Errors: []api.JobsError{}}
	targets := make([]string, 0, len(h.boxes))
	for id := range h.boxes {
		targets = append(targets, id)
	}
	sort.Strings(targets)
	for _, id := range targets {
		box := h.boxes[id]
		if box.err != nil {
			in.Errors = append(in.Errors, api.JobsError{Target: id, Error: *box.err})
		}
		for _, r := range box.jobs {
			in.Jobs = append(in.Jobs, h.viewLocked(id, r, box.agentTime))
		}
	}
	// Needs-you first, then unfinished, then finished; newest first within each.
	rank := func(s string) int {
		switch {
		case s == intent.JobNeedsDecision:
			return 0
		case !intent.JobTerminal(s):
			return 1
		}
		return 2
	}
	sort.SliceStable(in.Jobs, func(i, j int) bool {
		a, b := in.Jobs[i], in.Jobs[j]
		if rank(a.State) != rank(b.State) {
			return rank(a.State) < rank(b.State)
		}
		return a.CreatedAt.After(b.CreatedAt)
	})
	return in
}

// summary is a fleet row's jobs: nil when the box was never listed or cannot
// run jobs, so the row shows "unavailable", never 0 (TUI spec, honest data).
func (h *jobsHub) summary(target string) *api.JobsSummary {
	h.mu.Lock()
	defer h.mu.Unlock()
	box := h.boxes[target]
	if box == nil || box.unsupported || (box.err != nil && box.jobs == nil) {
		return nil
	}
	s := &api.JobsSummary{}
	for _, r := range box.jobs {
		switch {
		case r.State == intent.JobNeedsDecision:
			s.NeedsDecision++
		case !intent.JobTerminal(r.State):
			s.Running++
		}
	}
	return s
}

func (h *jobsHub) view(target string, r intent.JobRecord, agentTime int64) api.JobView {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.viewLocked(target, r, agentTime)
}

func (h *jobsHub) viewLocked(target string, r intent.JobRecord, agentTime int64) api.JobView {
	v := api.JobView{Target: target, ID: r.ID, Kind: r.Kind, Title: r.Title, State: r.State, Step: r.Step, Attempt: r.Attempt,
		Error: r.Error, CreatedAt: time.Unix(r.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(r.UpdatedAt, 0).UTC(),
		Steps: make([]api.JobStepView, len(r.Steps))}
	for i, s := range r.Steps {
		state := "pending"
		switch {
		case i < r.Step || r.State == intent.JobSucceeded:
			state = "done"
		case i == r.Step && r.State == intent.JobRunning:
			state = "running"
		case i == r.Step && r.State == intent.JobNeedsDecision:
			state = "asking"
		case i == r.Step && r.State == intent.JobWaiting:
			state = "waiting"
		case i == r.Step:
			state = "stopped"
		}
		v.Steps[i] = api.JobStepView{ID: s.ID, Title: s.Title, State: state}
	}
	if r.Lease.Owner != "" {
		hb := time.Unix(r.Lease.HeartbeatAt, 0).UTC()
		ttl := int64(r.Lease.TTLSeconds)
		if ttl == 0 {
			ttl = 90
		}
		v.Lease = api.JobLease{Owner: r.Lease.Owner, HeartbeatAt: &hb, Fresh: agentTime-r.Lease.HeartbeatAt < ttl, Mine: h.mine(r)}
		if v.Lease.Mine {
			v.Lease.Label = "this controller"
		}
	}
	if q := r.Question; q != nil {
		v.Question = &api.JobQuestion{ID: q.ID, Prompt: q.Prompt, Tier: q.Tier, AskedAt: time.Unix(q.AskedAt, 0).UTC(),
			Signable: q.Tier != intent.TierApproval} // Q5: no approval signer yet
		for _, o := range q.Options {
			v.Question.Options = append(v.Question.Options, api.JobOption{ID: o.ID, Label: o.Label, Detail: o.Detail})
		}
	}
	_, v.Driving = h.driving[jobKey{target, r.ID}]
	return v
}

// readVerified reads a job and its whole log, verifying only the entries
// this server has not verified before. A log that does not verify drops the
// cache for the job and is errJobLogInvalid.
func (h *jobsHub) readVerified(ctx context.Context, cfg config.Config, t config.Target, id string) (intent.JobRecord, []joblog.Entry, int64, error) {
	at, err := agentTarget(t)
	if err != nil {
		return intent.JobRecord{}, nil, 0, &localError{err}
	}
	k := jobKey{t.ID, id}
	h.mu.Lock()
	vl := h.logs[k]
	if vl == nil {
		vl = &verifiedLog{}
	}
	entries, head := append([]joblog.Entry(nil), vl.entries...), vl.head
	h.mu.Unlock()
	var rec intent.JobRecord
	var agentTime int64
	for {
		res, err := decodeResult[intent.JobReadResult](h.s.agentResult(ctx, cfg, t, intent.KindJobRead, intent.JobReadPayload{ID: id, From: uint64(len(entries))}))
		if err != nil {
			return rec, nil, 0, err
		}
		rec, agentTime = res.Job, res.AgentTime
		page := make([]joblog.Entry, 0, len(res.Entries))
		for _, j := range res.Entries {
			e, err := joblog.FromJSON(j)
			if err != nil {
				return h.invalid(k, rec, err)
			}
			page = append(page, e)
		}
		if head, err = joblog.Verify(at.Agent, id, page, uint64(len(entries)), head); err != nil {
			return h.invalid(k, rec, err)
		}
		if err := joblog.VerifyIntents(page); err != nil {
			return h.invalid(k, rec, err)
		}
		entries = append(entries, page...)
		if !res.More {
			break
		}
	}
	// The record must describe the log just verified, or the box is lying
	// about one of them.
	if uint64(len(entries)) != rec.LogLen || fmt.Sprintf("0x%x", head) != rec.LogHead {
		return h.invalid(k, rec, fmt.Errorf("the job says its log has %d entries ending %s; %d verified, ending 0x%x", rec.LogLen, rec.LogHead, len(entries), head))
	}
	h.mu.Lock()
	h.logs[k] = &verifiedLog{entries: entries, head: head}
	h.mu.Unlock()
	h.setRecord(t.ID, rec, agentTime)
	return rec, entries, agentTime, nil
}

func (h *jobsHub) invalid(k jobKey, rec intent.JobRecord, err error) (intent.JobRecord, []joblog.Entry, int64, error) {
	h.mu.Lock()
	delete(h.logs, k)
	h.mu.Unlock()
	return rec, nil, 0, fmt.Errorf("%w: %v", errJobLogInvalid, err)
}

func (h *jobsHub) subscribe() (chan api.JobsInbox, func()) {
	ch := make(chan api.JobsInbox, 4)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

// publish sends the inbox to every subscriber (dropping a frame for one that
// is behind: the next frame carries everything) and tells the fleet poller,
// whose rows carry the counts.
func (h *jobsHub) publish() {
	in := h.inbox()
	h.mu.Lock()
	for ch := range h.subs {
		select {
		case ch <- in:
		default:
		}
	}
	h.mu.Unlock()
	h.s.fleet.jobsChanged()
}

// stopWithin ends every driver and waits at most d for them.
func (h *jobsHub) stopWithin(d time.Duration) bool {
	h.mu.Lock()
	h.stopped = true
	for _, cancel := range h.driving {
		cancel()
	}
	h.mu.Unlock()
	h.cancel()
	done := make(chan struct{})
	go func() { h.wg.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

// Routes.

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	s.jobs.refresh(r.Context(), false)
	writeJSON(w, http.StatusOK, s.jobs.inbox())
}

func (s *Server) handleJobsStream(w http.ResponseWriter, r *http.Request) {
	conn, ok := startSSE(w)
	if !ok {
		return
	}
	defer conn.Close()
	ch, unsub := s.jobs.subscribe()
	defer unsub()
	go s.jobs.refresh(r.Context(), false)
	conn.Send(s.jobs.inbox())
	tick := time.NewTicker(max(jobsListEvery, time.Second))
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.jobs.ctx.Done():
			return
		case <-conn.Pings():
			conn.Ping()
		case <-tick.C:
			go s.jobs.refresh(r.Context(), false)
		case in := <-ch:
			conn.Send(in)
		}
	}
}

// jobTarget resolves {target} and {job} for a job route: a paired box and a
// well-formed id.
func (s *Server) jobTarget(w http.ResponseWriter, r *http.Request) (config.Config, config.Target, string, bool) {
	cfg, t, ok := s.nodeTarget(w, r.PathValue("target"))
	if !ok {
		return cfg, t, "", false
	}
	if t.Agent == nil {
		writeErrorDetail(w, http.StatusConflict, "this box has no paired agent, so it runs no jobs", "", api.CodeNotPaired)
		return cfg, t, "", false
	}
	id := r.PathValue("job")
	if id != "" && !jobs.ValidID(id) {
		writeErrorDetail(w, http.StatusNotFound, "no such job", "", api.CodeJobNotFound)
		return cfg, t, "", false
	}
	return cfg, t, id, true
}

func (s *Server) handleJobDetail(w http.ResponseWriter, r *http.Request) {
	cfg, t, id, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	rec, entries, agentTime, err := s.jobs.readVerified(r.Context(), cfg, t, id)
	if err != nil {
		writeNodeError(w, err)
		return
	}
	d := api.JobDetail{Job: s.jobs.view(t.ID, rec, agentTime), Log: make([]api.JobLogLine, 0, len(entries))}
	for _, e := range entries {
		d.Log = append(d.Log, api.JobLogLine{Index: e.Index, Kind: e.Kind, At: time.Unix(int64(e.At), 0).UTC(), Text: joblog.Describe(e)})
	}
	writeJSON(w, http.StatusOK, d)
}

// handleJobOutput follows a job's journal through job.output, a batch per
// event, from the start (or from ?cursor=).
func (s *Server) handleJobOutput(w http.ResponseWriter, r *http.Request) {
	cfg, t, id, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	conn, ok := startSSE(w)
	if !ok {
		return
	}
	defer conn.Close()
	cursor := r.URL.Query().Get("cursor")
	tick := time.NewTicker(jobsOutputEvery)
	defer tick.Stop()
	for {
		pctx, cancel := context.WithTimeout(r.Context(), agentPollTimeout)
		res, err := decodeResult[intent.JobOutputResult](s.agentResult(pctx, cfg, t, intent.KindJobOutput, intent.JobOutputPayload{ID: id, Cursor: cursor, N: intent.LogsMaxN}))
		cancel()
		if err != nil {
			_, e := apiErrorFor(err)
			conn.SendNamed("error", e) // apiclient's watch turns it into Update.Err
		} else if len(res.Lines) > 0 {
			lines := make([]api.JobOutputLine, len(res.Lines))
			for i, l := range res.Lines {
				lines[i] = api.JobOutputLine{Unit: l.Unit, Line: l.Line, At: l.At}
			}
			conn.Send(lines)
			cursor = res.Cursor
		}
		select {
		case <-r.Context().Done():
			return
		case <-conn.Pings():
			conn.Ping()
		case <-tick.C:
		}
	}
}
```

Register the routes in `api.go` beside the fleet routes:

```go
	mux.HandleFunc("GET /api/jobs", s.handleJobs)
	mux.HandleFunc("GET /api/jobs/stream", s.handleJobsStream)
	mux.HandleFunc("GET /api/jobs/{target}/{job}", s.handleJobDetail)
	mux.HandleFunc("GET /api/jobs/{target}/{job}/output/stream", s.handleJobOutput)
```

`server.go`: add the field `jobs *jobsHub`, set `s.jobs = newJobsHub(s)` in `New` before `s.fleet`, and call `s.jobs.stopWithin(fleetStopWait)` beside each `s.fleet.stopWithin(fleetStopWait)`.

`node.go` `classifyNodeError`: before the generic `agentRejected` case, add

```go
	case errors.Is(err, errJobLogInvalid):
		return http.StatusBadGateway, api.Error{Message: msg, Code: api.CodeJobLogInvalid}
	case errors.As(err, &rej) && jobRejectionCodes[rej.Code] != "":
		status := http.StatusConflict
		if rej.Code == intent.ReasonJobNotFound {
			status = http.StatusNotFound
		}
		return status, api.Error{Message: rej.Message, Code: jobRejectionCodes[rej.Code], Reason: rej.Code}
```

with

```go
// jobRejectionCodes gives job refusals their own codes, so a client can tell
// "someone else drives this" from any other refusal without parsing text.
var jobRejectionCodes = map[string]api.Code{
	intent.ReasonLeaseHeld: api.CodeLeaseHeld, intent.ReasonJobNotFound: api.CodeJobNotFound,
	intent.ReasonJobActive: api.CodeJobActive, intent.ReasonJobsUnsupported: api.CodeJobsUnsupported,
	intent.ReasonJobState: api.CodeJobState, intent.ReasonUnknownPlan: api.CodeUnknownPlan,
}
```

`fleet.go`:
- In `snapshot`, after `row := st.row`: `row.Jobs = p.s.jobs.summary(id)`. Lock order is fleet `p.mu` then hub `h.mu`; the hub never takes `p.mu` while it holds `h.mu`, because `publish` calls `jobsChanged` after unlocking.
- In `round`, after `p.seedLocked(cfg)`'s unlock: `go p.s.jobs.refresh(ctx, false)`.
- Add:

  ```go
  // jobsChanged republishes the fleet when a box's job counts change, so the
  // ⚑ count follows the inbox without waiting for the next poll round.
  func (p *fleetPoller) jobsChanged() {
  	p.mu.Lock()
  	defer p.mu.Unlock()
  	if len(p.subs) > 0 {
  		p.notifyLocked()
  	}
  }
  ```

- [ ] **Step 4: Run the server suite, including the freeze test**

Run: `go test ./internal/server/ -count=1 -race`
Expected: PASS, including `TestLegacyTargetRoutesAreFrozen` with no change to its list.

- [ ] **Step 5: Commit**

```bash
git add internal/server/jobs.go internal/server/jobs_test.go internal/server/server.go internal/server/api.go \
  internal/server/node.go internal/server/fleet.go internal/server/node_test.go
git commit -m "feat(server): jobs hub with verified logs, inbox and output streams, and job counts on fleet rows

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 12: The step-level driver — leases, advance, resume after restart, adopt, decide, start

Covers spec "v1 driver is step-level: the controller drives a multi-step plan … If no controller is present, the current step finishes and the plan waits at the next step boundary", "Adoption … takes the lease over (lease is stale)", "Long operations outlive the controller. Any controller can find them again and adopt them". Also Q4, Q5, the migrate result written into config, and Review Focus 3.

**Files:**
- Create: `internal/server/jobsdrive.go`, `internal/server/jobsdrive_test.go`
- Modify: `internal/server/server.go` (start the sweep in `ListenAndServe` and `ServeUnix`; the test-only `Config.noJobsDriver`, ruling P5), `internal/server/api.go` (routes), `internal/server/jobs_test.go` (ruling P5: `newBoxRig` takes server options and allows the test kind to start; `TestInboxListsEveryPairedBoxsJobs` and `TestFleetRowsCarryJobCounts` run without the driver)

**Interfaces:**
- Consumes: Task 11's hub; `jobs.MigrateNamesResult` (Task 8); `updateConfig`; Task 10's `api.CodeJobKindNotAllowed`.
- Produces (package `server`, unexported): `(h) drive(target, id string)`, `(h) driveLoop(ctx context.Context, k jobKey)`, `(h) sweep(ctx context.Context)`, `(h) finished(t config.Target, rec intent.JobRecord)`; `var jobStartKinds map[string]bool` (ruling P7: the only kinds the start route sends, `node.migrate-names` in v1); `Config.noJobsDriver bool` (ruling P5, tests only). Routes: `POST /api/jobs/{target}` (body `api.JobStart`, answers `api.JobView`), `POST /api/jobs/{target}/{job}/decide` (body `api.JobDecision`), `POST /api/jobs/{target}/{job}/adopt`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/server/jobsdrive_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// waitJob polls the inbox until the job reaches state or the test times out.
func waitJob(t *testing.T, b *boxRig, id, state string) api.JobView {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var in api.JobsInbox
		b.do(t, "GET", "/api/jobs", nil, &in)
		for _, j := range in.Jobs {
			if j.ID == id && j.State == state {
				return j
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s never reached %s", id, state)
	return api.JobView{}
}

func TestAStartedJobIsDrivenToTheEnd(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var v api.JobView
	if code := b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{}`)}, &v); code != 200 {
		t.Fatalf("start = %d", code)
	}
	if got := waitJob(t, b, v.ID, intent.JobSucceeded); !got.Lease.Mine {
		t.Fatalf("finished job %+v", got)
	}
}

func TestADecisionPointWaitsForAPersonThenContinues(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var v api.JobView
	b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v)
	asked := waitJob(t, b, v.ID, intent.JobNeedsDecision)
	if asked.Question == nil || !asked.Question.Signable {
		t.Fatalf("question %+v", asked.Question)
	}
	var after api.JobView
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, &after); code != 200 {
		t.Fatalf("decide = %d", code)
	}
	waitJob(t, b, v.ID, intent.JobSucceeded)
}

// Q5: an approval-tier question is refused here, before anything is signed.
func TestApprovalQuestionsAreNotSignedYet(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var v api.JobView
	b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"one"}`)}, &v)
	asked := waitJob(t, b, v.ID, intent.JobNeedsDecision)
	raiseQuestionTier(t, b, v.ID) // rewrites question.tier to approval in the agent's manifest
	b.srv.jobs.refresh(context.Background(), true)
	var e api.Error
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, &e); code != 409 || e.Code != api.CodeApprovalUnavailable {
		t.Fatalf("decide = %d %+v", code, e)
	}
}

// Review Focus 3: two controllers, one job.
func TestSecondControllerWatchesThenAdoptsAStaleJob(t *testing.T) {
	requireAgentPeer(t)
	a := newBoxRig(t)
	var v api.JobView
	a.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v)
	asked := waitJob(t, a, v.ID, intent.JobNeedsDecision)

	// Controller B: its own key, enrolled on the box, its own server.
	bKey, _ := signer.GenerateKey()
	p, _ := agent.LoadPolicy(filepath.Join(a.home, "policy.json"))
	p.AddSigner(agent.SignerEntry{Address: bKey.Address().Hex(), Tier: agent.TierRoutine, Label: "b"})
	_ = p.Save(filepath.Join(a.home, "policy.json"))
	bSrv := New(Config{Token: a.token, UI: fstest.MapFS{}, Signer: bKey})
	t.Cleanup(func() { bSrv.jobs.stopWithin(time.Second) })
	bTS := httptest.NewServer(bSrv.Handler())
	t.Cleanup(bTS.Close)
	b := &boxRig{ts: bTS, token: a.token, srv: bSrv, home: a.home}

	var in api.JobsInbox
	b.do(t, "GET", "/api/jobs", nil, &in)
	if len(in.Jobs) != 1 || in.Jobs[0].Lease.Mine || !in.Jobs[0].Lease.Fresh {
		t.Fatalf("B's view %+v", in.Jobs)
	}
	var e api.Error
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, &e); code != 409 || e.Code != api.CodeLeaseHeld {
		t.Fatalf("B decided a job A holds: %d %+v", code, e)
	}
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/adopt", nil, &e); code != 409 || e.Code != api.CodeLeaseHeld {
		t.Fatalf("B adopted a fresh lease: %d %+v", code, e)
	}

	a.srv.jobs.stopWithin(time.Second) // A goes away
	a.advance(91 * time.Second)        // and stays silent past the TTL, by the box's clock
	var got api.JobView
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/adopt", nil, &got); code != 200 || !got.Lease.Mine {
		t.Fatalf("adopt of a stale lease: %d %+v", code, got)
	}
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, &got); code != 200 {
		t.Fatalf("B's decision after adopting: %d", code)
	}
	waitJob(t, b, v.ID, intent.JobSucceeded)
}

// readOnBox reads a job straight from the agent, past the hub: nothing is
// listed, so nothing resumes as a side effect (GET /api/jobs would).
func (b *boxRig) readOnBox(t *testing.T, id string) intent.JobRecord {
	t.Helper()
	cfg, _ := b.srv.loadConfig()
	tg, _ := findTarget(cfg, "box")
	res, err := decodeResult[intent.JobReadResult](b.srv.agentResult(context.Background(), cfg, tg, intent.KindJobRead, intent.JobReadPayload{ID: id}))
	if err != nil {
		t.Fatal(err)
	}
	return res.Job
}

// Q4: after a restart a controller drives its own jobs again, by itself.
// Ruling P4: the test makes no HTTP request, because GET /api/jobs resumes
// jobs on its own; only sweep can have started this driver.
func TestARestartedServerResumesItsOwnJobs(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`) // created, never driven: the old server "died" at once
	time.Sleep(5 * jobsDriveEvery)
	b.srv.jobs.mu.Lock()
	n := len(b.srv.jobs.driving)
	b.srv.jobs.mu.Unlock()
	if got := b.readOnBox(t, rec.ID); n != 0 || got.State != intent.JobWaiting || got.Step != 0 {
		t.Fatalf("before the sweep: %d drivers, job %s at step %d", n, got.State, got.Step)
	}
	b.srv.jobs.sweep(context.Background())
	deadline := time.Now().Add(10 * time.Second)
	for b.readOnBox(t, rec.ID).State != intent.JobSucceeded {
		if time.Now().After(deadline) {
			t.Fatal("the sweep did not resume the job")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Ruling P7: the start route is not a generic intent proxy. A kind outside
// jobStartKinds is refused before anything is signed or sent.
func TestStartRefusesKindsThatAreNotJobs(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	before, _ := config.Load()
	for _, kind := range []string{intent.KindServiceAction, intent.KindJobDecide, "test.unlisted"} {
		var e api.Error
		in := api.JobStart{Kind: kind, Inputs: json.RawMessage(`{"service":"exec","action":"stop"}`)}
		if code := b.do(t, "POST", "/api/jobs/box", in, &e); code != 400 || e.Code != api.CodeJobKindNotAllowed {
			t.Fatalf("start of %s = %d %+v", kind, code, e)
		}
	}
	after, _ := config.Load()
	if after.Targets[0].Agent.NextSeq != before.Targets[0].Agent.NextSeq {
		t.Fatalf("an intent was signed and sent (next seq %d -> %d)", before.Targets[0].Agent.NextSeq, after.Targets[0].Agent.NextSeq)
	}
}

func TestADriverStopsWhenItsLeaseIsTaken(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{"AskAt":"one"}`)
	takeLeaseAsAnotherController(t, b, rec.ID) // enrolls a key, ages the lease, takes it over the agent directly
	b.srv.jobs.drive("box", rec.ID)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		b.srv.jobs.mu.Lock()
		n := len(b.srv.jobs.driving)
		b.srv.jobs.mu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the driver kept running after another controller took the lease")
}

func TestMigrateNamesSuccessUpdatesTheTargetsWire(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	_, _ = config.Update(func(c *config.Config) error {
		c.Targets[0].Wire = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
		return nil
	})
	cfg, _ := config.Load()
	b.srv.jobs.finished(cfg.Targets[0], intent.JobRecord{Kind: intent.KindNodeMigrateNames, State: intent.JobSucceeded,
		Result: json.RawMessage(`{"names":"jumpgate-node","dataDir":"/var/lib/jumpgate-node/369","jwtPath":"/var/lib/jumpgate-node/369/jwt.hex"}`)})
	cfg, _ = config.Load()
	if w := cfg.Targets[0].Wire; w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/jumpgate-node/369" {
		t.Fatalf("wire after migrate %+v", w)
	}
}
```

Write `raiseQuestionTier` and `takeLeaseAsAnotherController` in `jobsdrive_test.go` (the import block above already carries `os`, `agentclient` and `eip712` for them):
- `raiseQuestionTier` edits `<home>/jobs/<id>.json` with `os.ReadFile`/`json`/`os.WriteFile(…, 0o600)`, setting `question.tier` to `"approval"`.
- `takeLeaseAsAnotherController` generates a key and enrolls it in `policy.json`, calls `b.advance(91*time.Second)`, then sends `job.lease take` with `agentclient.Dial` over the box's local socket (`agentclient.Target{Agent: <the target's Agent.Address via eip712.ParseAddress>, Socket: <its Agent.Socket>, Local: true}`) and `agentclient.NewMemorySeqStore()`.

Then update Task 11's rig and two of its tests in `internal/server/jobs_test.go` (ruling P5). Once `wireDriver` is in, every list drives this controller's own jobs, so a test that looks at a job mid-way races a driver that finishes the in-process job in about 40 ms. Those tests turn the driver off; Q4 itself does not change. Replace `newBoxRig` with:

```go
// newBoxRig is a paired box whose agent runs jobs in-process. server adjusts
// the server's Config (withoutDriver, for one). The test kind may be started
// through POST /api/jobs/{target} for as long as the test runs.
func newBoxRig(t *testing.T, server ...func(*Config)) *boxRig {
	t.Helper()
	jobstest.RegisterPlan(t)
	b := &boxRig{now: time.Now()}
	prevDrive, prevList, prevKinds := jobsDriveEvery, jobsListEvery, jobStartKinds
	jobsDriveEvery, jobsListEvery = 20*time.Millisecond, 0
	jobStartKinds = map[string]bool{intent.KindNodeMigrateNames: true, jobstest.TestKind: true}
	t.Cleanup(func() { jobsDriveEvery, jobsListEvery, jobStartKinds = prevDrive, prevList, prevKinds })
	ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{
		setUp: true,
		agent: func(home string, c *agent.Config) {
			b.home = home
			dir := filepath.Join(home, "jobs")
			run := &jobstest.Runner{}
			run.Run = jobstest.InProcess(dir, jobs.Env{Exec: c.Exec, NodePath: c.NodePath, Out: io.Discard})
			m, err := jobs.NewManager(jobs.ManagerConfig{Dir: dir, Key: c.Key, Runner: run, Now: b.clock,
				StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv("/jumpgate", home, home, a) }})
			if err != nil {
				t.Fatal(err)
			}
			c.Jobs, c.Now = m, b.clock
			p, _ := agent.LoadPolicy(c.PolicyPath)
			p.Kinds = map[string][]agent.Tier{jobstest.TestKind: {agent.TierRoutine}}
			_ = p.Save(c.PolicyPath)
		},
		server: func(c *Config) {
			for _, f := range server {
				f(c)
			}
		},
		onServer: func(s *Server) { b.srv = s },
	})
	b.ts, b.token = ts, token
	t.Cleanup(func() { b.srv.jobs.stopWithin(time.Second) })
	return b
}

// withoutDriver keeps the hub listing and verifying but never driving, so a
// test can look at a job that stays where it was put.
func withoutDriver(c *Config) { c.noJobsDriver = true }
```

In `TestInboxListsEveryPairedBoxsJobs` and `TestFleetRowsCarryJobCounts`, change `b := newBoxRig(t)` to `b := newBoxRig(t, withoutDriver)`. Every other Task 11 test keeps `newBoxRig(t)`: none of them depends on where the job stands.

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/server/ -run 'Driven|DecisionPoint|ApprovalQuestions|SecondController|ResumesItsOwn|StartRefuses|DriverStops|MigrateNamesSuccess' -count=1`
Expected: FAIL to compile (undefined `sweep`, `drive`, `finished`, `jobStartKinds`, `noJobsDriver`).

- [ ] **Step 3: Implement `internal/server/jobsdrive.go`**

```go
// internal/server/jobsdrive.go
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
)

// jobStartKinds are the kinds POST /api/jobs/{target} starts (ruling P7).
// The route signs nothing but a kind from this list, so it never becomes a
// way to send an arbitrary intent ("intents, not commands"; a new node
// operation is a new intent kind, not a new route). A new job kind is added
// here on purpose, with its plan on the agent. A variable so tests can add
// jobstest.TestKind.
var jobStartKinds = map[string]bool{intent.KindNodeMigrateNames: true}

// The v1 driver is step-level (spec): the agent runs each step to its end on
// its own, and the lease holder starts the next one. A job whose controller
// is gone finishes its current step and waits at the boundary.

// wireDriver connects the hub's list hook: every list resumes this
// controller's own unfinished jobs (Q4: own leases resume by themselves;
// other controllers' never do).
func (h *jobsHub) wireDriver() {
	h.afterList = func(t config.Target, recs []intent.JobRecord) {
		for _, r := range recs {
			if !intent.JobTerminal(r.State) && h.mine(r) {
				h.drive(t.ID, r.ID)
			}
		}
	}
}

// sweep is run when the server starts: list every paired box now, which
// resumes every job this controller held when it last ran.
func (h *jobsHub) sweep(ctx context.Context) { h.refresh(ctx, true) }

// drive starts the driver for one job, once.
func (h *jobsHub) drive(target, id string) {
	k := jobKey{target, id}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stopped || h.driving[k] != nil {
		return
	}
	ctx, cancel := context.WithCancel(h.ctx)
	h.driving[k] = cancel
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		defer func() {
			h.mu.Lock()
			delete(h.driving, k)
			h.mu.Unlock()
			h.publish()
		}()
		h.driveLoop(ctx, k)
	}()
}

func isRejection(err error, code string) bool {
	var rej *agentRejected
	return errors.As(err, &rej) && rej.Code == code
}

// driveLoop reads the job every jobsDriveEvery: advances it at a step
// boundary, renews the lease every jobsRenewEvery, and returns when the job
// ends, another controller holds it, or its log does not verify (a job
// whose history cannot be checked is never driven).
func (h *jobsHub) driveLoop(ctx context.Context, k jobKey) {
	var lastRenew time.Time
	tick := time.NewTicker(jobsDriveEvery)
	defer tick.Stop()
	for {
		cfg, err := h.s.loadConfig()
		if err != nil {
			return
		}
		t, ok := findTarget(cfg, k.target)
		if !ok || t.Agent == nil {
			return
		}
		rec, _, _, err := h.readVerified(ctx, cfg, t, k.id)
		switch {
		case errors.Is(err, errJobLogInvalid), isRejection(err, intent.ReasonJobNotFound):
			return
		case err != nil:
			// Unreachable or slow: keep trying; the lease ages on the box.
		case intent.JobTerminal(rec.State):
			h.finished(t, rec)
			return
		case !h.mine(rec):
			return // adopted by another controller
		case rec.State == intent.JobWaiting:
			_, err := h.s.agentResult(ctx, cfg, t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: k.id, Step: rec.Step, Attempt: rec.Attempt})
			if isRejection(err, intent.ReasonLeaseHeld) {
				return
			}
			lastRenew = time.Now() // an advance renews the lease
		case time.Since(lastRenew) >= jobsRenewEvery:
			_, err := h.s.agentResult(ctx, cfg, t, intent.KindJobLease, intent.JobLeasePayload{ID: k.id, Op: intent.LeaseRenew})
			if isRejection(err, intent.ReasonLeaseHeld) {
				return
			}
			if err == nil {
				lastRenew = time.Now()
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// finished copies what a finished job changed into this controller's record
// of the box. node.migrate-names moves the box's names and data dir; without
// this, the next setup re-run would render units for an account that no
// longer exists (Review Focus 4).
func (h *jobsHub) finished(t config.Target, rec intent.JobRecord) {
	if rec.Kind != intent.KindNodeMigrateNames || rec.State != intent.JobSucceeded || t.Wire == nil {
		return
	}
	var res jobs.MigrateNamesResult
	if json.Unmarshal(rec.Result, &res) != nil || res.Names == "" {
		return
	}
	_, _ = h.s.updateConfig(func(c *config.Config) error {
		for i := range c.Targets {
			if c.Targets[i].ID == t.ID && c.Targets[i].Wire != nil {
				w := c.Targets[i].Wire
				w.Names, w.DataDir, w.JWTPath = res.Names, res.DataDir, res.JWTPath
			}
		}
		return nil
	})
}

// record reads a job fresh from the box and verifies its log, so a decision
// is checked against what the box says now, never against a cached record.
func (h *jobsHub) record(ctx context.Context, cfg config.Config, t config.Target, id string) (intent.JobRecord, int64, error) {
	rec, _, agentTime, err := h.readVerified(ctx, cfg, t, id)
	return rec, agentTime, err
}

func (s *Server) handleJobStart(w http.ResponseWriter, r *http.Request) {
	cfg, t, _, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	var in api.JobStart
	if err := json.NewDecoder(io.LimitReader(r.Body, maxIntentBody)).Decode(&in); err != nil || in.Kind == "" {
		writeError(w, http.StatusBadRequest, "want {\"kind\": …, \"inputs\": …}")
		return
	}
	if !jobStartKinds[in.Kind] {
		writeErrorDetail(w, http.StatusBadRequest, fmt.Sprintf("%q is not a job this server starts", in.Kind), "", api.CodeJobKindNotAllowed)
		return
	}
	if len(in.Inputs) == 0 {
		in.Inputs = json.RawMessage(`{}`)
	}
	rec, err := decodeResult[intent.JobRecord](s.agentResult(r.Context(), cfg, t, in.Kind, in.Inputs))
	if err != nil {
		writeNodeError(w, err)
		return
	}
	s.jobs.setRecord(t.ID, rec, time.Now().Unix())
	s.jobs.drive(t.ID, rec.ID)
	writeJSON(w, http.StatusOK, s.jobs.view(t.ID, rec, rec.UpdatedAt))
}

func (s *Server) handleJobAdopt(w http.ResponseWriter, r *http.Request) {
	cfg, t, id, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	rec, err := decodeResult[intent.JobRecord](s.agentResult(r.Context(), cfg, t, intent.KindJobLease, intent.JobLeasePayload{ID: id, Op: intent.LeaseTake}))
	if err != nil {
		writeNodeError(w, err)
		return
	}
	s.jobs.setRecord(t.ID, rec, rec.Lease.HeartbeatAt)
	s.jobs.drive(t.ID, id)
	writeJSON(w, http.StatusOK, s.jobs.view(t.ID, rec, rec.Lease.HeartbeatAt))
}

func (s *Server) handleJobDecide(w http.ResponseWriter, r *http.Request) {
	cfg, t, id, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	var d api.JobDecision
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&d); err != nil || d.Decision == "" || d.Option == "" {
		writeError(w, http.StatusBadRequest, "want {\"decision\": …, \"option\": …}")
		return
	}
	rec, _, err := s.jobs.record(r.Context(), cfg, t, id)
	if err != nil {
		writeNodeError(w, err)
		return
	}
	switch {
	case !s.jobs.mine(rec):
		writeErrorDetail(w, http.StatusConflict, "controller "+rec.Lease.Owner+" holds this job; you are watching it", "", api.CodeLeaseHeld)
		return
	case rec.Question != nil && rec.Question.ID == d.Decision && rec.Question.Tier == intent.TierApproval:
		writeErrorDetail(w, http.StatusConflict, "this decision needs an approval-tier signature", "", api.CodeApprovalUnavailable)
		return
	}
	rec, err = decodeResult[intent.JobRecord](s.agentResult(r.Context(), cfg, t, intent.KindJobDecide, intent.JobDecidePayload{ID: id, Decision: d.Decision, Option: d.Option}))
	if err != nil {
		writeNodeError(w, err)
		return
	}
	s.jobs.setRecord(t.ID, rec, rec.Lease.HeartbeatAt)
	s.jobs.drive(t.ID, id)
	writeJSON(w, http.StatusOK, s.jobs.view(t.ID, rec, rec.Lease.HeartbeatAt))
}
```

In `server.go`, add to `Config`:

```go
	// noJobsDriver keeps the jobs hub listing and verifying but never
	// driving. Tests only (ruling P5): a test that inspects a job mid-way
	// must not race the driver. Unexported, so no caller outside the
	// package can turn the driver off.
	noJobsDriver bool
```

In `New`, after `s.jobs = newJobsHub(s)`, add `if !cfg.noJobsDriver { s.jobs.wireDriver() }`.

Routes in `api.go`:

```go
	mux.HandleFunc("POST /api/jobs/{target}", s.handleJobStart)
	mux.HandleFunc("POST /api/jobs/{target}/{job}/decide", s.handleJobDecide)
	mux.HandleFunc("POST /api/jobs/{target}/{job}/adopt", s.handleJobAdopt)
```

`server.go`: in `ListenAndServe` and `ServeUnix`, start `go s.jobs.sweep(ctx)` before serving. A server with no controller key (`Signer == nil`) holds no leases, so the sweep only lists.

The `view` calls pass the lease heartbeat as the agent time right after a mutation: the heartbeat was just set by the agent, so the lease reads fresh. That is true at that moment.

- [ ] **Step 4: Run the server suite**

Run: `go test ./internal/server/ -count=1 -race`
Expected: PASS, including Task 11's `TestInboxListsEveryPairedBoxsJobs` and `TestFleetRowsCarryJobCounts` under `-count=20` (`go test ./internal/server/ -run 'InboxListsEvery|FleetRowsCarryJobCounts' -count=20 -race`), which shows the driver no longer races them.

- [ ] **Step 5: Commit**

```bash
git add internal/server/jobsdrive.go internal/server/jobsdrive_test.go internal/server/jobs_test.go internal/server/server.go internal/server/api.go
git commit -m "feat(server): drive held jobs step by step, resume them after a restart, adopt and decide

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

## Phase C: front ends (Tasks 13–16)

### Task 13: `apiclient` job methods, the TUI `Backend`, and the fake

**Files:**
- Modify: `internal/apiclient/methods.go`, `internal/tui/backend.go` (`Backend` gains the job methods), `internal/tui/tuitest/fake.go`
- Test: `internal/apiclient/client_test.go` (new tests), `internal/tui/tuitest/tuitest_test.go` (the fake satisfies `tui.Backend`, already asserted by `var _ Backend = …`)

**Interfaces:**
- Consumes: Task 10's types; the routes of Tasks 11–12.
- Produces (package `apiclient`): `(c *Client) Jobs(ctx) (api.JobsInbox, error)`; `WatchJobs(ctx) <-chan Update[api.JobsInbox]`; `Job(ctx, target, id string) (api.JobDetail, error)`; `WatchJobOutput(ctx, target, id string) <-chan Update[[]api.JobOutputLine]`; `StartJob(ctx, target string, s api.JobStart) (api.JobView, error)`; `MigrateNames(ctx, target string) (api.JobView, error)`; `Decide(ctx, target, id string, d api.JobDecision) (api.JobView, error)`; `Adopt(ctx, target, id string) (api.JobView, error)`.
- Produces (package `tuitest`): `Fake` fields `JobsV api.JobsInbox`, `JobsCh chan apiclient.Update[api.JobsInbox]` (made by `NewFake`, capacity 16), `JobV map[string]api.JobDetail` (key `target/id`), `JobViewV api.JobView` (the answer of `MigrateNames`, `Decide` and `Adopt`), `LastDecision api.JobDecision`; methods matching `Backend`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/apiclient/client_test.go — add
func TestJobMethodsUseTheJobRoutes(t *testing.T) {
	var seen []string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch {
		case r.URL.Path == "/api/jobs":
			fmt.Fprint(w, `{"at":"2026-10-09T00:00:00Z","jobs":[{"target":"box","id":"0123456789abcdef","state":"needs-decision"}],"errors":[]}`)
		case strings.HasSuffix(r.URL.Path, "/decide"):
			var d api.JobDecision
			_ = json.NewDecoder(r.Body).Decode(&d)
			if d.Decision != "q" || d.Option != "retry" {
				http.Error(w, "bad body", 400)
				return
			}
			fmt.Fprint(w, `{"target":"box","id":"0123456789abcdef","state":"waiting"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/jobs/box":
			var s api.JobStart
			_ = json.NewDecoder(r.Body).Decode(&s)
			if s.Kind != intent.KindNodeMigrateNames {
				http.Error(w, "bad kind", 400)
				return
			}
			fmt.Fprint(w, `{"target":"box","id":"fedcba9876543210","state":"waiting"}`)
		default:
			fmt.Fprint(w, `{"target":"box","id":"0123456789abcdef","state":"waiting"}`)
		}
	})
	ctx := context.Background()
	in, err := c.Jobs(ctx)
	if err != nil || in.NeedsDecision() != 1 {
		t.Fatalf("Jobs = %+v, %v", in, err)
	}
	if v, err := c.Decide(ctx, "box", "0123456789abcdef", api.JobDecision{Decision: "q", Option: "retry"}); err != nil || v.State != "waiting" {
		t.Fatalf("Decide = %+v, %v", v, err)
	}
	if v, err := c.MigrateNames(ctx, "box"); err != nil || v.ID != "fedcba9876543210" {
		t.Fatalf("MigrateNames = %+v, %v", v, err)
	}
	if _, err := c.Adopt(ctx, "box", "0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	want := []string{"GET /api/jobs", "POST /api/jobs/box/0123456789abcdef/decide", "POST /api/jobs/box", "POST /api/jobs/box/0123456789abcdef/adopt"}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("routes %v", seen)
	}
}
```

Add `encoding/json` to the imports if the file lacks it.

- [ ] **Step 2: Run it to make sure it fails**

Run: `go test ./internal/apiclient/ -run TestJobMethods -count=1`
Expected: FAIL to compile (`c.Jobs` undefined).

- [ ] **Step 3: Implement**

```go
// internal/apiclient/methods.go — add
func jobPath(target, id, rest string) string {
	return "/api/jobs/" + url.PathEscape(target) + "/" + url.PathEscape(id) + rest
}

// Jobs is every job on every paired box.
func (c *Client) Jobs(ctx context.Context) (api.JobsInbox, error) {
	var in api.JobsInbox
	err := c.Do(ctx, http.MethodGet, "/api/jobs", nil, &in)
	return in, err
}

// WatchJobs streams the inbox whenever it changes.
func (c *Client) WatchJobs(ctx context.Context) <-chan Update[api.JobsInbox] {
	return watch(ctx, c, "/api/jobs/stream", decodeJSON[api.JobsInbox])
}

// Job is one job with its verified log.
func (c *Client) Job(ctx context.Context, target, id string) (api.JobDetail, error) {
	var d api.JobDetail
	err := c.Do(ctx, http.MethodGet, jobPath(target, id, ""), nil, &d)
	return d, err
}

// WatchJobOutput follows a job's journal, a batch of lines per value.
func (c *Client) WatchJobOutput(ctx context.Context, target, id string) <-chan Update[[]api.JobOutputLine] {
	return watch(ctx, c, jobPath(target, id, "/output/stream"), decodeJSON[[]api.JobOutputLine])
}

func (c *Client) StartJob(ctx context.Context, target string, s api.JobStart) (api.JobView, error) {
	var v api.JobView
	err := c.Do(ctx, http.MethodPost, "/api/jobs/"+url.PathEscape(target), s, &v)
	return v, err
}

// MigrateNames starts node.migrate-names on target.
func (c *Client) MigrateNames(ctx context.Context, target string) (api.JobView, error) {
	return c.StartJob(ctx, target, api.JobStart{Kind: intent.KindNodeMigrateNames})
}

func (c *Client) Decide(ctx context.Context, target, id string, d api.JobDecision) (api.JobView, error) {
	var v api.JobView
	err := c.Do(ctx, http.MethodPost, jobPath(target, id, "/decide"), d, &v)
	return v, err
}

func (c *Client) Adopt(ctx context.Context, target, id string) (api.JobView, error) {
	var v api.JobView
	err := c.Do(ctx, http.MethodPost, jobPath(target, id, "/adopt"), nil, &v)
	return v, err
}
```

Import `internal/intent` in `methods.go`.

In `internal/tui/backend.go`, add to `Backend`:

```go
	Jobs(ctx context.Context) (api.JobsInbox, error)
	WatchJobs(ctx context.Context) <-chan apiclient.Update[api.JobsInbox]
	Job(ctx context.Context, target, id string) (api.JobDetail, error)
	MigrateNames(ctx context.Context, target string) (api.JobView, error)
	Decide(ctx context.Context, target, id string, d api.JobDecision) (api.JobView, error)
	Adopt(ctx context.Context, target, id string) (api.JobView, error)
```

In `internal/tui/tuitest/fake.go`, add the fields listed under Interfaces, make `JobsCh` and `JobV` in `NewFake`, and add:

```go
func (f *Fake) Jobs(context.Context) (api.JobsInbox, error) { return f.JobsV, f.call("Jobs") }

func (f *Fake) WatchJobs(ctx context.Context) <-chan apiclient.Update[api.JobsInbox] {
	_ = f.call("WatchJobs")
	return relay(ctx, f.JobsCh)
}

func (f *Fake) Job(_ context.Context, target, id string) (api.JobDetail, error) {
	f.mu.Lock()
	d := f.JobV[target+"/"+id]
	f.mu.Unlock()
	return d, f.call("Job", target, id)
}

func (f *Fake) MigrateNames(_ context.Context, target string) (api.JobView, error) {
	return f.JobViewV, f.call("MigrateNames", target)
}

func (f *Fake) Decide(_ context.Context, target, id string, d api.JobDecision) (api.JobView, error) {
	f.mu.Lock()
	f.LastDecision = d
	f.mu.Unlock()
	return f.JobViewV, f.call("Decide", target, id, d.Decision, d.Option)
}

func (f *Fake) Adopt(_ context.Context, target, id string) (api.JobView, error) {
	return f.JobViewV, f.call("Adopt", target, id)
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/apiclient/ ./internal/tui/... -count=1`
Expected: PASS. `var _ Backend = (*apiclient.Client)(nil)` compiles, which proves the client implements every new method.

- [ ] **Step 5: Commit**

```bash
git add internal/apiclient/methods.go internal/apiclient/client_test.go internal/tui/backend.go internal/tui/tuitest/fake.go
git commit -m "feat(apiclient): job inbox, detail, decide, adopt and migrate-names methods

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 14: The TUI jobs inbox

Covers umbrella screen 3 ("Jobs inbox — fleet-wide running, paused and needs-decision jobs, with the signed question and its options"), TUI spec screen 6 and D27 (the placeholder becomes real), "Destructive actions need a typed confirmation in the TUI", Q4, Q5.

**Files:**
- Modify: `internal/tui/jobs.go` (rewrite), `internal/tui/app.go:143` (`a.screens[scrJobs] = &jobsScreen{}`)
- Create: `internal/tui/jobs_test.go`, `internal/tui/testdata/jobs_80x24.golden`, `internal/tui/testdata/jobs_ascii.golden`
- Test: also regenerate `internal/tui/testdata/shell_jobs_80x24.golden` and `internal/tui/testdata/shell_jobs_ascii.golden` (both showed the placeholder; `-update` rewrites both)

**Interfaces:**
- Consumes: Task 13's `Backend` methods and `Fake` fields; `confirmModal` (`newConfirm`); `visit`; `navKeys`; `a.do`; `a.errText`; `sanitizeLine`, `truncate`, `formatAge`.
- Produces (package `tui`): `type jobsScreen struct`; messages `jobsMsg{gen; u apiclient.Update[api.JobsInbox]; ch}`, `jobDetailMsg`, `jobActionMsg`; `var jobKeys = struct{ Decide, Adopt, Reload key.Binding }`; `type choiceModal struct` (pick an option by number).

- [ ] **Step 1: Write the failing tests**

```go
// internal/tui/jobs_test.go
package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func sampleInbox() api.JobsInbox {
	hb := testNow.Add(-5 * time.Second)
	old := testNow.Add(-10 * time.Minute)
	return api.JobsInbox{At: testNow, Jobs: []api.JobView{
		{Target: "box-a", ID: "0123456789abcdef", Kind: "node.migrate-names", Title: "Rename the node's account, units and data to jumpgate-node",
			State: "needs-decision", Step: 1, Steps: []api.JobStepView{{ID: "preflight", State: "done"}, {ID: "stop-units", State: "asking"}, {ID: "rename-account", State: "pending"}},
			Question: &api.JobQuestion{ID: "stop-units-0-busy-user", Prompt: "1 process(es) still run as valve-node-app", Tier: "routine", Signable: true,
				Options: []api.JobOption{{ID: "retry", Label: "Check again"}, {ID: "stop-processes", Label: "Stop them"}, {ID: "abort", Label: "Stop the migration here"}}},
			Lease: api.JobLease{Owner: "0xc0ffee00000000000000000000000000000000aa", Label: "this controller", HeartbeatAt: &hb, Fresh: true, Mine: true}, CreatedAt: old},
		{Target: "box-b", ID: "fedcba9876543210", Kind: "node.migrate-names", Title: "Rename the node's account, units and data to jumpgate-node",
			State: "running", Step: 3, Steps: make([]api.JobStepView, 8),
			Lease: api.JobLease{Owner: "0xbbbb000000000000000000000000000000000bbb", HeartbeatAt: &old, Fresh: false}, CreatedAt: old},
	}}
}

func jobsApp(t *testing.T, f *tuitest.Fake, glyphs string) *App {
	t.Helper()
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: sampleInbox(), Has: true, State: apiclient.Live}
	return press(t, newTestApp(t, f, 80, 24, glyphs), "5").(*App)
}

func TestInboxShowsWhatNeedsYouFirst(t *testing.T) {
	fr := tuitest.Frame(jobsApp(t, tuitest.NewFake(), "unicode"))
	for _, want := range []string{"Jobs inbox", "1 needs you", "box-a", "⚑", "step 2/3", "1 process(es) still run", "1 Check again", "2 Stop them", "box-b", "running", "silent 10m", "a: adopt"} {
		if !strings.Contains(fr, want) {
			t.Errorf("inbox lacks %q:\n%s", want, fr)
		}
	}
	if strings.Index(fr, "box-a") > strings.Index(fr, "box-b") {
		t.Errorf("the job that needs you is not first:\n%s", fr)
	}
	tuitest.Golden(t, "jobs_80x24", fr)
	ascii := tuitest.Frame(jobsApp(t, tuitest.NewFake(), "ascii"))
	requireASCII(t, ascii)
	tuitest.Golden(t, "jobs_ascii", ascii)
}

// Destructive choices need typed confirmation (umbrella spec); the request is
// sent only for the exact option id.
func TestDecidingNeedsTheOptionTypedExactly(t *testing.T) {
	f := tuitest.NewFake()
	m := jobsApp(t, f, "unicode")
	m = press(t, m, "d", "2").(*App)
	m = typed(t, m, "stop-process").(*App)
	m = press(t, m, "enter").(*App)
	if f.Called("Decide") {
		t.Fatal("a near miss was sent")
	}
	m = typed(t, m, "stop-processes").(*App)
	press(t, m, "enter")
	if !f.Called("Decide box-a 0123456789abcdef stop-units-0-busy-user stop-processes") {
		t.Fatalf("calls %v", f.Calls)
	}
}

// Q5: an approval-tier question is shown, never answered from here.
func TestApprovalQuestionsSayWhyTheyCannotBeAnswered(t *testing.T) {
	f := tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0].Question.Tier, in.Jobs[0].Question.Signable = "approval", false
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "5", "d")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "approval") || f.Called("Decide") {
		t.Fatalf("frame:\n%s\ncalls %v", fr, f.Calls)
	}
}

// Review Focus 3, front end: a job another controller holds freshly is
// watched; d says who holds it instead of asking.
func TestAJobHeldElsewhereIsWatchedOnly(t *testing.T) {
	f := tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0].Lease = api.JobLease{Owner: "0xbbbb000000000000000000000000000000000bbb", Fresh: true}
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "5", "d").(*App)
	if f.Called("Decide") || !strings.Contains(m.flash, "holds this job") {
		t.Fatalf("flash %q calls %v", m.flash, f.Calls)
	}
}

func TestAdoptOnlyAStaleJob(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, jobsApp(t, f, "unicode"), "a").(*App) // box-a is ours: nothing to adopt
	if f.Called("Adopt") {
		t.Fatal("adopted a job we hold")
	}
	m = press(t, m, "down", "a").(*App)
	m = typed(t, m, "box-b").(*App)
	press(t, m, "enter")
	if !f.Called("Adopt box-b fedcba9876543210") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestEnterShowsTheVerifiedLog(t *testing.T) {
	f := tuitest.NewFake()
	f.JobV["box-a/0123456789abcdef"] = api.JobDetail{Log: []api.JobLogLine{{Index: 0, Kind: "job.created", At: testNow, Text: "node.migrate-names by 0xc0ffee: created"}}}
	m := press(t, jobsApp(t, f, "unicode"), "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "node.migrate-names by 0xc0ffee: created") {
		t.Fatalf("frame:\n%s", fr)
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/tui/ -run 'Inbox|Deciding|ApprovalQuestions|HeldElsewhere|AdoptOnly|VerifiedLog' -count=1`
Expected: FAIL (the placeholder shows none of it).

- [ ] **Step 3: Implement the inbox**

```go
// internal/tui/jobs.go
package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

var jobKeys = struct{ Decide, Adopt, Reload key.Binding }{
	Decide: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "answer the selected job's question")),
	Adopt:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "adopt a job whose controller went silent")),
	Reload: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
}

type (
	jobsMsg struct {
		gen uint64
		u   apiclient.Update[api.JobsInbox]
		ch  <-chan apiclient.Update[api.JobsInbox]
	}
	jobDetailMsg struct {
		gen uint64
		key string
		d   api.JobDetail
		err error
	}
	jobActionMsg struct {
		gen  uint64
		what string
		v    api.JobView
		err  error
	}
)

// jobsScreen is the fleet-wide inbox (umbrella screen 3): jobs that need a
// decision first, then running and waiting ones, then finished ones. It
// watches /api/jobs/stream while it is visible.
type jobsScreen struct {
	in     api.JobsInbox
	has    bool
	conn   apiclient.ConnState
	err    error
	cursor int
	open   string // "target/id" whose log is shown
	detail *api.JobDetail
	detErr error
	visit
}

func (s *jobsScreen) capturing() bool { return false }

func (s *jobsScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Up, navKeys.Down, navKeys.Enter, jobKeys.Decide, jobKeys.Adopt, jobKeys.Reload}
}

func (s *jobsScreen) selected() (api.JobView, bool) {
	if s.cursor < 0 || s.cursor >= len(s.in.Jobs) {
		return api.JobView{}, false
	}
	return s.in.Jobs[s.cursor], true
}

func (s *jobsScreen) watch(a *App) tea.Cmd {
	s.visit.begin(a.ctx)
	ch := a.be.WatchJobs(s.ctx)
	return s.next(s.gen, ch)
}

func (s *jobsScreen) next(gen uint64, ch <-chan apiclient.Update[api.JobsInbox]) tea.Cmd {
	return func() tea.Msg {
		u, ok := <-ch
		if !ok {
			return nil
		}
		return jobsMsg{gen: gen, u: u, ch: ch}
	}
}

func (s *jobsScreen) loadDetail(a *App, j api.JobView) tea.Cmd {
	gen, ctx, be, k := s.gen, s.ctx, a.be, j.Target+"/"+j.ID
	return func() tea.Msg {
		d, err := be.Job(ctx, j.Target, j.ID)
		return jobDetailMsg{gen: gen, key: k, d: d, err: err}
	}
}

func (s *jobsScreen) act(a *App, what string, f func(ctx context.Context, be Backend) (api.JobView, error)) tea.Cmd {
	gen, ctx, be := s.gen, s.ctx, a.be
	return func() tea.Msg {
		v, err := f(ctx, be)
		return jobActionMsg{gen: gen, what: what, v: v, err: err}
	}
}

func (s *jobsScreen) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case enterMsg:
		return s.watch(a)
	case leaveMsg:
		s.visit.invalidate()
		return nil
	case jobsMsg:
		if !s.current(msg.gen) {
			return nil
		}
		s.conn, s.err = msg.u.State, msg.u.Err
		if msg.u.Has {
			s.in, s.has = msg.u.Value, true
			s.cursor = min(s.cursor, max(len(s.in.Jobs)-1, 0))
		}
		return s.next(msg.gen, msg.ch)
	case jobDetailMsg:
		if s.current(msg.gen) && msg.key == s.open {
			s.detail, s.detErr = &msg.d, msg.err
		}
		return nil
	case jobActionMsg:
		if !s.current(msg.gen) {
			return nil
		}
		if msg.err != nil {
			a.flash = msg.what + ": " + a.errText(msg.err)
			return nil
		}
		a.flash = msg.what + " sent; the box answered " + msg.v.State
		return nil
	case tea.KeyPressMsg:
		if a.active != scrJobs || a.detail != nil {
			return nil
		}
		switch {
		case key.Matches(msg, navKeys.Up):
			s.cursor = max(s.cursor-1, 0)
		case key.Matches(msg, navKeys.Down):
			s.cursor = min(s.cursor+1, max(len(s.in.Jobs)-1, 0))
		case key.Matches(msg, navKeys.Enter):
			j, ok := s.selected()
			if !ok {
				return nil
			}
			if k := j.Target + "/" + j.ID; s.open != k {
				s.open, s.detail, s.detErr = k, nil, nil
				return s.loadDetail(a, j)
			}
			s.open = ""
		case key.Matches(msg, jobKeys.Decide):
			return s.decide(a)
		case key.Matches(msg, jobKeys.Adopt):
			return s.adopt(a)
		case key.Matches(msg, jobKeys.Reload):
			return s.watch(a)
		}
	}
	return nil
}

// decide opens the option list for the selected job's question. Who may
// answer is the box's to enforce (the lease, the question's tier); the TUI
// says so first rather than sending a request it knows will be refused.
func (s *jobsScreen) decide(a *App) tea.Cmd {
	j, ok := s.selected()
	switch {
	case !ok || j.Question == nil:
		a.flash = "this job is not waiting on a decision"
	case !j.Question.Signable:
		a.flash = "this decision needs an " + j.Question.Tier + "-tier signature (a hardware or browser wallet); jumpgate cannot make one yet"
	case !j.Lease.Mine && j.Lease.Fresh:
		a.flash = "controller " + shortAddr(sanitizeLine(j.Lease.Owner), a.gl) + " holds this job; you are watching it"
	case !j.Lease.Mine:
		a.flash = "nobody drives this job now; adopt it first (a)"
	default:
		a.modal = &choiceModal{job: j, screen: s}
	}
	return nil
}

func (s *jobsScreen) adopt(a *App) tea.Cmd {
	j, ok := s.selected()
	switch {
	case !ok || j.Lease.Mine:
		a.flash = "this controller already drives that job"
	case j.Lease.Fresh:
		a.flash = "its controller is still alive (heartbeat " + formatAge(a.now().Sub(*j.Lease.HeartbeatAt)) + " ago); you can adopt it after 90 s of silence"
	default:
		a.modal = newConfirm("Adopt job "+j.ID+" on "+j.Target,
			"This controller takes over driving the job. The controller that started it stops when it next checks in.", j.Target,
			func() tea.Cmd {
				return s.act(a, "adopt", func(ctx context.Context, be Backend) (api.JobView, error) { return be.Adopt(ctx, j.Target, j.ID) })
			})
	}
	return nil
}

// choiceModal lists a question's options by number; a number opens the typed
// confirmation for that option.
type choiceModal struct {
	job    api.JobView
	screen *jobsScreen
}

func (m *choiceModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	if k.String() == "esc" {
		return true, nil
	}
	q := m.job.Question
	for i, o := range q.Options {
		if k.String() == fmt.Sprint(i+1) {
			o, j := o, m.job
			a.modal = newConfirm("Answer: "+o.Label, q.Prompt+"\n\n"+o.Detail, o.ID, func() tea.Cmd {
				return m.screen.act(a, "decision", func(ctx context.Context, be Backend) (api.JobView, error) {
					return be.Decide(ctx, j.Target, j.ID, api.JobDecision{Decision: q.ID, Option: o.ID})
				})
			})
			return false, nil
		}
	}
	return false, nil
}

func (m *choiceModal) view(a *App, w, h int) string {
	q := m.job.Question
	var b strings.Builder
	b.WriteString(a.th.Title.Render(" "+sanitizeLine(m.job.Target)+" asks") + "\n\n")
	for _, line := range strings.Split(sanitize(q.Prompt), "\n") {
		b.WriteString(" " + truncate(line, w-2, a.gl.Ellipsis) + "\n")
	}
	b.WriteString("\n")
	for i, o := range q.Options {
		b.WriteString(fmt.Sprintf(" %d %s", i+1, sanitizeLine(o.Label)))
		if o.Detail != "" {
			b.WriteString(a.th.Dim.Render(" " + a.gl.Sep + " " + sanitizeLine(o.Detail)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + a.th.Dim.Render(" press a number, or esc"))
	return b.String()
}

func (s *jobsScreen) view(a *App, w, h int) string {
	var b strings.Builder
	need := s.in.NeedsDecision()
	head := " Jobs inbox"
	if s.has {
		head += fmt.Sprintf(" %s %d needs you", a.gl.Sep, need)
	}
	b.WriteString(a.th.Title.Render(head) + "\n")
	if !s.has {
		b.WriteString(a.th.Dim.Render(" loading…") + "\n")
		return b.String()
	}
	if len(s.in.Jobs) == 0 {
		b.WriteString("\n No durable jobs on any paired box. Start one with :migrate-names NAME.\n")
	}
	for i, j := range s.in.Jobs {
		mark := "  "
		if i == s.cursor {
			mark = a.gl.Sel + " "
		}
		flag := " "
		if j.State == "needs-decision" {
			flag = a.th.Warn.Render(a.gl.Flag)
		}
		line := fmt.Sprintf("%s%s %-8s %-12s step %d/%d %s", mark, flag, truncate(sanitizeLine(j.Target), 8, a.gl.Ellipsis),
			j.State, min(j.Step+1, len(j.Steps)), len(j.Steps), s.leaseWord(a, j))
		b.WriteString(truncate(line, w, a.gl.Ellipsis) + "\n")
		if i == s.cursor {
			b.WriteString(s.selectedBody(a, j, w))
		}
	}
	for _, e := range s.in.Errors {
		b.WriteString(a.th.Bad.Render(truncate(" "+sanitizeLine(e.Target)+": "+sanitizeLine(e.Error.Message), w, a.gl.Ellipsis)) + "\n")
	}
	return fit(b.String(), w, h, a.gl.Ellipsis)
}

func (s *jobsScreen) leaseWord(a *App, j api.JobView) string {
	switch {
	case j.Lease.Mine:
		return "driven here"
	case j.Lease.Owner == "":
		return "nobody drives it"
	case j.Lease.Fresh:
		return "watching " + shortAddr(sanitizeLine(j.Lease.Owner), a.gl)
	case j.Lease.HeartbeatAt != nil:
		return "silent " + formatAge(a.now().Sub(*j.Lease.HeartbeatAt)) + " " + a.gl.Sep + " a: adopt"
	}
	return "a: adopt"
}

func (s *jobsScreen) selectedBody(a *App, j api.JobView, w int) string {
	var b strings.Builder
	ind := "      "
	b.WriteString(a.th.Dim.Render(truncate(ind+sanitizeLine(j.Title), w, a.gl.Ellipsis)) + "\n")
	if q := j.Question; q != nil {
		for _, line := range strings.Split(sanitize(q.Prompt), "\n") {
			b.WriteString(truncate(ind+line, w, a.gl.Ellipsis) + "\n")
		}
		for i, o := range q.Options {
			b.WriteString(truncate(fmt.Sprintf("%s%d %s", ind, i+1, sanitizeLine(o.Label)), w, a.gl.Ellipsis) + "\n")
		}
		if !q.Signable {
			b.WriteString(a.th.Warn.Render(ind+"needs an "+q.Tier+"-tier signature") + "\n")
		}
	}
	if s.open == j.Target+"/"+j.ID {
		switch {
		case s.detErr != nil:
			b.WriteString(a.th.Bad.Render(truncate(ind+a.errText(s.detErr), w, a.gl.Ellipsis)) + "\n")
		case s.detail == nil:
			b.WriteString(a.th.Dim.Render(ind+"reading the signed log…") + "\n")
		default:
			for _, l := range s.detail.Log {
				b.WriteString(truncate(fmt.Sprintf("%s%3d %s", ind, l.Index, sanitizeLine(l.Text)), w, a.gl.Ellipsis) + "\n")
			}
		}
	}
	return b.String()
}
```

In `app.go:143`, use `a.screens[scrJobs] = &jobsScreen{}`. `shortAddr(addr string, g Glyphs)` (`fleet.go`), `fit(block string, w, h int, ell string)` and `formatAge` (`format.go`), and `sanitize`/`sanitizeLine` (`sanitize.go`) already exist with the signatures used above.

- [ ] **Step 4: Regenerate the goldens this task changes, then run the TUI suite**

Run: `go test ./internal/tui/ -run 'TestShellFrame|TestInboxShowsWhatNeedsYouFirst' -update -count=1 && go test ./internal/tui/... -count=1 -race`
Expected: PASS. Open `internal/tui/testdata/jobs_80x24.golden`, `jobs_ascii.golden`, `shell_jobs_80x24.golden` and `shell_jobs_ascii.golden` and read them: 24 lines, nothing cut mid-word that matters, ASCII only in the ascii golden.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/jobs.go internal/tui/app.go internal/tui/jobs_test.go \
  internal/tui/testdata/jobs_80x24.golden internal/tui/testdata/jobs_ascii.golden \
  internal/tui/testdata/shell_jobs_80x24.golden internal/tui/testdata/shell_jobs_ascii.golden
git commit -m "feat(tui): the jobs inbox shows questions, the signed log, decisions and adoption

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 15: The ⚑ count, the bell, the fleet `jobs` column, `:migrate-names`

Covers umbrella "status bar with … a fleet-wide ⚑ count of jobs needing a decision", Settings "bell on ⚑", fleet column "jobs (running, ⚑)", the TUI spec's reserved `jobs` field and its `–` until now, and the command palette (`:resync beacon box-a` style) for the first job.

**Files:**
- Modify: `internal/tui/app.go` (`statusBar`; the bell on `fleetMsg`; `openJobsMsg` in `Update`), `internal/tui/backend.go` (`Options.Bell`), `internal/tui/jobs.go` (`openJobsMsg`; runs after Task 14 rewrote it), `internal/tui/fleet.go:246-247` (the `jobs` column), `internal/tui/palette.go` (`migrate-names NAME`; `paletteHelp`; names from `catalog`), `internal/tui/settings.go:258` (the label), `cmd/jumpgate/tui_on.go` (pass `Bell`)
- Test: `internal/tui/app_test.go`, `internal/tui/fleet_test.go`, `internal/tui/palette_test.go`

**Interfaces:**
- Consumes: `api.FleetRow.Jobs` (filled by Task 11), Task 13's `MigrateNames`.
- Produces: `Options.Bell func()` (nil: no bell); `(a *App) jobsFlag() string`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/tui/app_test.go — add
func fleetWithJobs(needs ...int) api.Fleet {
	f := sampleFleet()
	for i, n := range needs {
		f.Rows[i].Jobs = &api.JobsSummary{NeedsDecision: n}
	}
	return f
}

func TestStatusBarCountsJobsThatNeedYou(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	if !strings.Contains(a.statusBar(), "jobs ⚑ –") {
		t.Fatalf("before any box reports jobs: %q", a.statusBar())
	}
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fleetWithJobs(1, 2), Has: true, State: apiclient.Live}})
	if bar := m.(*App).statusBar(); !strings.Contains(bar, "jobs ⚑ 3") {
		t.Fatalf("status bar %q", bar)
	}
}

func TestTheBellRingsWhenMoreJobsNeedYou(t *testing.T) {
	rings := 0
	f := tuitest.NewFake()
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" }, Now: func() time.Time { return testNow }, Bell: func() { rings++ }})
	t.Cleanup(func() { a.cancel() })
	on := api.UIPrefs{Glyphs: "unicode", Bell: true}
	m, cmds := tuitest.Send(a, tea.WindowSizeMsg{Width: 80, Height: 24}, prefsMsg{p: on},
		fleetMsg{u: apiclient.Update[api.Fleet]{Value: fleetWithJobs(1), Has: true}}) // the first reading never rings
	m = tuitest.Settle(m, cmds...)
	m, cmds = tuitest.Send(m, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fleetWithJobs(2), Has: true}})
	m = tuitest.Settle(m, cmds...)
	m, cmds = tuitest.Send(m, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fleetWithJobs(1), Has: true}}) // fewer: no ring
	tuitest.Settle(m, cmds...)
	if rings != 1 {
		t.Fatalf("rang %d times, want 1", rings)
	}
}

func TestNoBellWhenTheSettingIsOff(t *testing.T) {
	rings := 0
	a := New(Options{Backend: tuitest.NewFake(), GOOS: "linux", Getenv: func(string) string { return "" }, Now: func() time.Time { return testNow }, Bell: func() { rings++ }})
	t.Cleanup(func() { a.cancel() })
	m, cmds := tuitest.Send(a, prefsMsg{p: api.UIPrefs{Glyphs: "unicode", Bell: false}},
		fleetMsg{u: apiclient.Update[api.Fleet]{Value: fleetWithJobs(0), Has: true}},
		fleetMsg{u: apiclient.Update[api.Fleet]{Value: fleetWithJobs(3), Has: true}})
	tuitest.Settle(m, cmds...)
	if rings != 0 {
		t.Fatalf("rang %d times with the bell off", rings)
	}
}
```

```go
// internal/tui/fleet_test.go — add
func TestFleetJobsColumn(t *testing.T) {
	f := tuitest.NewFake()
	f.PrefsV.FleetColumns = append(f.PrefsV.FleetColumns, "jobs")
	a := fleetApp(t, f, "unicode", apiclient.Live)
	fl := sampleFleet()
	fl.Rows[0].Jobs = &api.JobsSummary{Running: 1, NeedsDecision: 1}
	fl.Rows[1].Jobs = &api.JobsSummary{Running: 2}
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fl, Has: true, State: apiclient.Live}})
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "⚑1") || !strings.Contains(fr, "JOBS") {
		t.Fatalf("frame:\n%s", fr)
	}
}
```

`FleetColumns` is the field in `api.UIPrefs` (`internal/api/prefs.go`); `jobs` is a registered column that is off by default.

```go
// internal/tui/palette_test.go — add
func TestPaletteMigrateNamesNeedsTheHostName(t *testing.T) {
	f := tuitest.NewFake()
	f.JobViewV = api.JobView{Target: "box-a", ID: "0123456789abcdef", State: "waiting"}
	m := fleetApp(t, f, "unicode", apiclient.Live)
	m2 := press(t, m, ":")
	m2 = typed(t, m2, "migrate-names box-a")
	m2 = press(t, m2, "enter")
	if f.Called("MigrateNames") {
		t.Fatal("ran before the typed confirmation")
	}
	m2 = typed(t, m2, "box-a")
	m2 = press(t, m2, "enter")
	if !f.Called("MigrateNames box-a") {
		t.Fatalf("calls %v", f.Calls)
	}
	if a := m2.(*App); a.active != scrJobs {
		t.Fatalf("the inbox did not open; active %v", a.active)
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./internal/tui/ -run 'StatusBarCounts|TheBellRings|NoBell|FleetJobsColumn|PaletteMigrateNames' -count=1`
Expected: FAIL (`Options.Bell` undefined, then the assertions).

- [ ] **Step 3: Implement**

`backend.go` `Options`: add

```go
	// Bell rings the terminal bell (the Settings "bell on ⚑" choice); nil
	// is silent. cmd/jumpgate writes BEL to stderr: stdout is Bubble Tea's.
	Bell func()
```

`app.go`:
- Add `needs int` and `needsKnown bool` to `App`.
- In `case fleetMsg:`, after the fleet value is taken:

  ```go
  		var ring tea.Cmd
  		if msg.u.Has {
  			n := 0
  			for _, r := range msg.u.Value.Rows {
  				if r.Jobs != nil {
  					n += r.Jobs.NeedsDecision
  				}
  			}
  			// Ring when more jobs need a person than before; never for the
  			// first reading, which would ring at every start-up.
  			if a.needsKnown && n > a.needs && a.prefs.Bell && a.o.Bell != nil {
  				bell := a.o.Bell
  				ring = func() tea.Msg { bell(); return nil }
  			}
  			a.needs, a.needsKnown = n, true
  		}
  		return a, tea.Batch(a.watchFleet(msg.ch), a.broadcast(msg), ring)
  ```
- Add:

  ```go
  // jobsFlag is the fleet-wide count of jobs that need a decision, or the
  // dash until some box has reported its jobs (honest data: unknown is not 0).
  func (a *App) jobsFlag() string {
  	known, n := false, 0
  	for _, r := range a.fleet.Rows {
  		if r.Jobs != nil {
  			known, n = true, n+r.Jobs.NeedsDecision
  		}
  	}
  	if !known {
  		return a.gl.Dash
  	}
  	return strconv.Itoa(n)
  }
  ```
- In `statusBar`, replace `a.gl.Dash` in the `jobs %s %s` format with `a.jobsFlag()`, and fix the comment above `statusBar` ("the jobs flag (always – until sub-project 2)" becomes "the fleet-wide ⚑ count").

`fleet.go` `case "jobs":`

```go
	case "jobs":
		if r.Jobs == nil {
			return a.th.Dim.Render(na) // not reported: never 0
		}
		if r.Jobs.NeedsDecision > 0 {
			return a.th.Warn.Render(a.gl.Flag + strconv.Itoa(r.Jobs.NeedsDecision))
		}
		return strconv.Itoa(r.Jobs.Running)
```

`palette.go`: add `migrate-names NAME` to `paletteHelp`, and to the known-box check `f[0] == "migrate-names"`. Add the case:

```go
	case boxCmd("migrate-names"):
		id := f[1]
		a.modal = newConfirm("Rename the node on "+id,
			"Starts the node.migrate-names job: it stops the node services, renames the "+catalog.LegacyNames.Prefix+
				" account, units and data root to "+catalog.CurrentNames.Prefix+", and starts the services again. It asks you before anything it cannot undo.",
			id, func() tea.Cmd {
				return a.do("migrate-names", func(ctx context.Context, be Backend) tea.Msg {
					v, err := be.MigrateNames(ctx, id)
					if err != nil {
						return errMsg{what: "migrate-names " + id, err: err}
					}
					return openJobsMsg{flash: "job " + v.ID + " started on " + id}
				})
			})
		return false, nil
```

Import `github.com/valve-tech/jumpgate/internal/catalog` in `palette.go`. The names come from `catalog.LegacyNames` and `catalog.CurrentNames`, never a literal (Global Constraint 1, ruling P10); `catalog` imports nothing else in the module, so the TUI gains no dependency.

Add `type openJobsMsg struct{ flash string }` in `jobs.go`, and handle it in `App.Update`: `case openJobsMsg: a.flash = sanitizeLine(msg.flash); return a, a.openScreen(scrJobs)`.

`settings.go:258`: label `"Bell on jobs that need you"`.

`cmd/jumpgate/tui_on.go`: in the `tui.Options{…}` literal add `Bell: func() { fmt.Fprint(os.Stderr, "\a") },`.

- [ ] **Step 4: Run the TUI suite; regenerate only goldens whose change you can explain**

Run: `go test ./internal/tui/... ./cmd/jumpgate/ -count=1 -race`
Expected: PASS without `-update`. Rows in `sampleFleet` have no `Jobs`, so existing frames still read `jobs ⚑ –`. If a settings golden fails only because of the bell label, regenerate it with `-update` and read the diff.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/backend.go internal/tui/jobs.go internal/tui/fleet.go internal/tui/palette.go internal/tui/settings.go \
  internal/tui/app_test.go internal/tui/fleet_test.go internal/tui/palette_test.go cmd/jumpgate/tui_on.go
# Plus internal/tui/testdata/settings.golden, by path, if Step 4 regenerated it.
git status --short
git commit -m "feat(tui): fleet-wide ⚑ count, the bell, the jobs column and :migrate-names

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 16: The CLI — `jumpgate jobs …` and `jumpgate migrate-names HOST`

Covers the terminal success criterion ("An operator can run the whole fleet from `jumpgate` in a terminal"), for scripts and for operators without the TUI.

**Files:**
- Create: `cmd/jumpgate/cli_jobs.go`, `cmd/jumpgate/cli_jobs_test.go`
- Modify: `cmd/jumpgate/cli.go` (`subcommands`), `cmd/jumpgate/help.go` (`commandHelp`), and `cmd/jumpgate/help_test.go` only if it pins the help text

**Interfaces:**
- Consumes: Task 13's client methods, including `WatchJobOutput` (ruling P13: the CLI is its first consumer; the TUI follows later, J12); `connect`, `reportServerErrorFrom`, `failed`, `usage`, `exitCode`.
- Produces: `cmdJobs(args []string) int` with subcommands `jobs [HOST]`, `jobs show HOST JOB`, `jobs decide HOST JOB DECISION OPTION`, `jobs adopt HOST JOB`, `jobs output JOB` (follows the job's journal until Ctrl-C; the host is found from the inbox); `cmdMigrateNames(args []string) int` (`migrate-names HOST --yes`); `printInbox(w io.Writer, in api.JobsInbox, host string, now time.Time)`, `printJobDetail(w io.Writer, d api.JobDetail)`, `followOutput(w io.Writer, ch <-chan apiclient.Update[[]api.JobOutputLine]) error`.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/jumpgate/cli_jobs_test.go
package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
)

func TestPrintInboxPutsQuestionsUnderTheirJob(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	hb := now.Add(-2 * time.Minute)
	in := api.JobsInbox{Jobs: []api.JobView{
		{Target: "box-a", ID: "0123456789abcdef", Kind: "node.migrate-names", State: "needs-decision", Step: 1, Steps: make([]api.JobStepView, 8),
			Question: &api.JobQuestion{ID: "stop-units-0-busy-user", Prompt: "1 process still runs", Options: []api.JobOption{{ID: "retry", Label: "Check again"}, {ID: "stop-processes", Label: "Stop them"}}},
			Lease:    api.JobLease{Mine: true, Label: "this controller"}},
		{Target: "box-b", ID: "fedcba9876543210", Kind: "node.migrate-names", State: "running", Step: 3, Steps: make([]api.JobStepView, 8),
			Lease: api.JobLease{Owner: "0xbbbb", HeartbeatAt: &hb}},
	}, Errors: []api.JobsError{{Target: "box-c", Error: api.Error{Message: "could not reach the agent"}}}}
	var b bytes.Buffer
	printInbox(&b, in, "", now)
	out := b.String()
	for _, want := range []string{"HOST", "box-a", "0123456789abcdef", "needs-decision", "2/8", "this controller",
		"? 1 process still runs", "jumpgate jobs decide box-a 0123456789abcdef stop-units-0-busy-user retry|stop-processes",
		"box-b", "silent 2m", "box-c: could not reach the agent"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	b.Reset()
	printInbox(&b, in, "box-b", now)
	if strings.Contains(b.String(), "box-a") {
		t.Errorf("a host filter still printed other boxes:\n%s", b.String())
	}
}

func TestMigrateNamesWantsYes(t *testing.T) {
	if code := cmdMigrateNames([]string{"box-a"}); code != exitCode("usage") {
		t.Fatalf("exit %d without --yes", code)
	}
}

// Global Constraint 1, ruling P10: the legacy name reaches the help text only
// through catalog.LegacyNames.
func TestMigrateNamesHelpNamesBothSets(t *testing.T) {
	for _, h := range commandHelp {
		if strings.HasPrefix(h.name, "migrate-names") {
			if !strings.Contains(h.text, catalog.LegacyNames.Prefix) || !strings.Contains(h.text, catalog.CurrentNames.Prefix) {
				t.Fatalf("help %q", h.text)
			}
			return
		}
	}
	t.Fatal("no migrate-names help")
}

// Ruling P13: jobs output prints each journal line as it arrives, without
// the control bytes a step's output may carry, and stops when the server
// refuses the job.
func TestFollowOutputPrintsLinesAndStopsOnARefusal(t *testing.T) {
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	ch := make(chan apiclient.Update[[]api.JobOutputLine], 3)
	ch <- apiclient.Update[[]api.JobOutputLine]{Has: true, Value: []api.JobOutputLine{
		{Unit: "jumpgate-job-0123456789abcdef-stop-units-0.service", Line: "stopping\x1b[2J the units", At: at}}}
	ch <- apiclient.Update[[]api.JobOutputLine]{Err: &api.Error{Message: "no such job", Code: api.CodeJobNotFound}}
	ch <- apiclient.Update[[]api.JobOutputLine]{Has: true, Value: []api.JobOutputLine{{Line: "after the refusal", At: at}}}
	close(ch)
	var b bytes.Buffer
	err := followOutput(&b, ch)
	var e *api.Error
	if !errors.As(err, &e) || e.Code != api.CodeJobNotFound {
		t.Fatalf("followOutput = %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "stop-units-0") || !strings.Contains(out, "stopping[2J the units") || strings.Contains(out, "\x1b") || strings.Contains(out, "after the refusal") {
		t.Fatalf("output:\n%q", out)
	}
}

func TestJobsUsage(t *testing.T) {
	// Only malformed verbs: a bare HOST would connect to (and maybe start) a server.
	for _, args := range [][]string{{"show", "box-a"}, {"decide", "box-a", "j", "q"}, {"adopt"}, {"adopt", "box-a"}, {"output"}, {"output", "box-a", "0123456789abcdef"}} {
		if code := cmdJobs(args); code != exitCode("usage") {
			t.Errorf("jobs %v: exit %d", args, code)
		}
	}
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `go test ./cmd/jumpgate/ -run 'PrintInbox|MigrateNamesWantsYes|MigrateNamesHelp|FollowOutput|JobsUsage' -count=1`
Expected: FAIL to compile.

- [ ] **Step 3: Implement**

```go
// cmd/jumpgate/cli_jobs.go
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
)

const jobsUsage = "usage: jumpgate jobs [HOST] | jobs show HOST JOB | jobs decide HOST JOB DECISION OPTION | jobs adopt HOST JOB | jobs output JOB"

func cmdJobs(args []string) int {
	switch {
	case len(args) <= 1 && (len(args) == 0 || !isJobsVerb(args[0])):
		host := ""
		if len(args) == 1 {
			host = args[0]
		}
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			in, err := c.Jobs(ctx)
			if err == nil {
				printInbox(os.Stdout, in, host, time.Now())
			}
			return err
		})
	case args[0] == "show" && len(args) == 3:
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			d, err := c.Job(ctx, args[1], args[2])
			if err == nil {
				printJobDetail(os.Stdout, d)
			}
			return err
		})
	case args[0] == "decide" && len(args) == 5:
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			v, err := c.Decide(ctx, args[1], args[2], api.JobDecision{Decision: args[3], Option: args[4]})
			if err == nil {
				fmt.Printf("%s %s: %s\n", v.Target, v.ID, v.State)
			}
			return err
		})
	case args[0] == "adopt" && len(args) == 3:
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			v, err := c.Adopt(ctx, args[1], args[2])
			if err == nil {
				fmt.Printf("%s %s: this controller drives it now (%s)\n", v.Target, v.ID, v.State)
			}
			return err
		})
	case args[0] == "output" && len(args) == 2:
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
			defer stop()
			host, err := jobHost(ctx, c, args[1])
			if err != nil {
				return err
			}
			return followOutput(os.Stdout, c.WatchJobOutput(ctx, host, args[1]))
		})
	}
	return usage(jobsUsage)
}

func isJobsVerb(s string) bool { return s == "show" || s == "decide" || s == "adopt" || s == "output" }

// jobHost finds the box that runs job id, so `jobs output` takes only the id
// `jumpgate jobs` prints (ruling P13). Ids are 64 random bits; an id on two
// boxes is refused rather than guessed.
func jobHost(ctx context.Context, c *apiclient.Client, id string) (string, error) {
	in, err := c.Jobs(ctx)
	if err != nil {
		return "", err
	}
	var hosts []string
	for _, j := range in.Jobs {
		if j.ID == id {
			hosts = append(hosts, j.Target)
		}
	}
	switch len(hosts) {
	case 0:
		return "", fmt.Errorf("no box that answered has a job %s (`jumpgate jobs` lists them)", id)
	case 1:
		return hosts[0], nil
	}
	return "", fmt.Errorf("job %s is on more than one box (%s); use `jumpgate jobs show HOST %s`", id, strings.Join(hosts, ", "), id)
}

// followOutput prints a job's journal as it arrives, until the stream ends
// (Ctrl-C) or the server refuses the job. Other stream errors are the
// connection's, which watch retries by itself.
func followOutput(w io.Writer, ch <-chan apiclient.Update[[]api.JobOutputLine]) error {
	for u := range ch {
		var e *api.Error
		if errors.As(u.Err, &e) {
			return e
		}
		if !u.Has {
			continue
		}
		for _, l := range u.Value {
			fmt.Fprintf(w, "%s %s %s\n", l.At.Format(time.RFC3339), l.Unit, terminalSafe(l.Line))
		}
	}
	return nil
}

// terminalSafe drops control bytes (but tab) from a line a step wrote, so a
// journal line cannot move the cursor or retitle the operator's terminal.
func terminalSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if (r < 0x20 && r != '\t') || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// cmdMigrateNames starts node.migrate-names. It stops the node's services
// for the length of the job, so it wants --yes, as reset-replay does.
func cmdMigrateNames(args []string) int {
	if len(args) != 2 || args[1] != "--yes" {
		return usage("usage: jumpgate migrate-names HOST --yes  (stops the node services on HOST while it renames the " +
			catalog.LegacyNames.Prefix + " account, units and data root to " + catalog.CurrentNames.Prefix + ")")
	}
	return withClient(func(ctx context.Context, c *apiclient.Client) error {
		v, err := c.MigrateNames(ctx, args[0])
		if err == nil {
			fmt.Printf("job %s started on %s; follow it with `jumpgate jobs %s` or the jobs inbox (5 in jumpgate)\n", v.ID, v.Target, v.Target)
		}
		return err
	})
}

// withClient connects, runs f, and maps an API error to the CLI's exit codes.
func withClient(f func(ctx context.Context, c *apiclient.Client) error) int {
	ctx := context.Background()
	c, err := connect(ctx)
	if err != nil {
		return failed("%v", err)
	}
	apiclient.WarnSkew(os.Stderr, c)
	err = f(ctx, c)
	var e *api.Error
	if errors.As(err, &e) {
		return reportServerErrorFrom(os.Stderr, "jobs", c.Info(), *e)
	}
	if err != nil {
		return failed("%v", err)
	}
	return 0
}

func printInbox(w io.Writer, in api.JobsInbox, host string, now time.Time) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "HOST\tJOB\tKIND\tSTATE\tSTEP\tLEASE")
	for _, j := range in.Jobs {
		if host != "" && j.Target != host {
			continue
		}
		lease := "nobody"
		switch {
		case j.Lease.Mine:
			lease = "this controller"
		case j.Lease.Fresh:
			lease = j.Lease.Owner
		case j.Lease.HeartbeatAt != nil:
			lease = fmt.Sprintf("%s, silent %s (adoptable)", j.Lease.Owner, ago(now.Sub(*j.Lease.HeartbeatAt)))
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d/%d\t%s\n", j.Target, j.ID, j.Kind, j.State, min(j.Step+1, len(j.Steps)), len(j.Steps), lease)
		if q := j.Question; q != nil {
			ids := make([]string, len(q.Options))
			for i, o := range q.Options {
				ids[i] = o.ID
			}
			fmt.Fprintf(tw, "\t? %s\n", strings.ReplaceAll(q.Prompt, "\n", " "))
			fmt.Fprintf(tw, "\t  jumpgate jobs decide %s %s %s %s\n", j.Target, j.ID, q.ID, strings.Join(ids, "|"))
		}
	}
	tw.Flush()
	for _, e := range in.Errors {
		if host == "" || e.Target == host {
			fmt.Fprintf(w, "%s: %s\n", e.Target, e.Error.Message)
		}
	}
}

// ago is a duration the way a person reads an age: 45s, 2m, 3h, 2d.
func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
}

func printJobDetail(w io.Writer, d api.JobDetail) {
	j := d.Job
	fmt.Fprintf(w, "%s on %s: %s (%s), step %d/%d\n", j.ID, j.Target, j.Title, j.State, min(j.Step+1, len(j.Steps)), len(j.Steps))
	for _, s := range j.Steps {
		fmt.Fprintf(w, "  %-8s %s\n", s.State, s.Title)
	}
	fmt.Fprintln(w, "signed log (verified against the box's agent key):")
	for _, l := range d.Log {
		fmt.Fprintf(w, "  %3d %s %s\n", l.Index, l.At.Format(time.RFC3339), l.Text)
	}
}
```

If `cmd/jumpgate` already has a duration formatter (`grep -n "func.*Age\|func ago" cmd/jumpgate/*.go`), use it instead of adding `ago`.

In `cli.go`, add `"jobs": cmdJobs, "migrate-names": cmdMigrateNames` to `subcommands`. In `help.go` `commandHelp`, after `service` (import `internal/catalog` there; the names come from it, never a literal, ruling P10):

```go
	{"jobs [HOST]", "list durable jobs; `jobs show|decide|adopt …` to act on one, `jobs output JOB` to follow its output"},
	{"migrate-names HOST --yes", "rename HOST's " + catalog.LegacyNames.Prefix + " account, units and data to " + catalog.CurrentNames.Prefix + " (a durable job)"},
```

- [ ] **Step 4: Run the tests**

Run: `go test ./cmd/jumpgate/ -count=1`
Expected: PASS, including `help_test.go`. If it pins the help text, update it in this commit.

- [ ] **Step 5: Commit**

```bash
git add cmd/jumpgate/cli_jobs.go cmd/jumpgate/cli_jobs_test.go cmd/jumpgate/cli.go cmd/jumpgate/help.go
# Plus cmd/jumpgate/help_test.go, by path, if Step 4 had to update it.
git status --short
git commit -m "feat(cli): jumpgate jobs and jumpgate migrate-names

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

## Phase D: proof on a real box, and wrap-up (Tasks 17–18)

### Task 17: End to end on systemd — `node.migrate-names` across an agent restart

Covers spec "survives agent restarts and controller disconnects", "Output goes to the journal", "a signed, append-only log … so an adopting controller can verify history", and Review Focus 2, against the real `systemd-run` on Debian 12 and Ubuntu 24.04.

**Files:**
- Create: `internal/bootstrap/e2e_jobs_test.go`
- Test: run through `scripts/e2e-agent.sh` (unchanged: it runs `go test -tags e2e ./internal/bootstrap/ -run E2E`)

**Interfaces:**
- Consumes: `e2ePort`, `e2eEnv`, `e2eAgents`, `waitAgentSocket` and `Run` from `internal/bootstrap`; `agentclient`; `joblog`; `intent`; `catalog.RenderUnits`.

- [ ] **Step 1: Write the e2e test**

```go
// internal/bootstrap/e2e_jobs_test.go
//go:build e2e

package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// seedLegacyNode makes the container look like a box set up before
// sub-project 2: the valve-node-app account, its data root, its two units
// (not running: there is no client binary here), node.json, and one
// process of the account that ignores SIGTERM.
func seedLegacyNode(ctx context.Context, t *testing.T, ex executor.Executor) {
	t.Helper()
	w := catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
		DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
	execUnit, beaconUnit, err := catalog.RenderUnits(w)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{
		"useradd --system --user-group --home-dir /var/lib/valve-node-app --no-create-home --shell /usr/sbin/nologin valve-node-app",
		"mkdir -p /var/lib/valve-node-app/369/db && head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \\n' > /var/lib/valve-node-app/369/jwt.hex",
		"chown -R valve-node-app:valve-node-app /var/lib/valve-node-app",
	} {
		if r, err := ex.Run(ctx, cmd, nil); err != nil || r.ExitCode != 0 {
			t.Fatalf("%s: %+v %v", cmd, r, err)
		}
	}
	_ = ex.WriteFile(ctx, catalog.UnitPath(catalog.LegacyNames.ExecUnit()), []byte(execUnit), 0o644)
	_ = ex.WriteFile(ctx, catalog.UnitPath(catalog.LegacyNames.BeaconUnit()), []byte(beaconUnit), 0o644)
	b, _ := json.Marshal(w)
	_ = ex.WriteFile(ctx, NodePath, b, 0o600)
	busy := `systemd-run --unit=e2e-busy --uid=valve-node-app --quiet /bin/sh -c "trap '' TERM; while :; do sleep 1; done"`
	if r, err := ex.Run(ctx, "systemctl daemon-reload && "+busy, nil); err != nil || r.ExitCode != 0 {
		t.Fatalf("busy process: %+v %v", r, err)
	}
}

func TestE2EMigrateNamesSurvivesAnAgentRestart(t *testing.T) {
	port := e2ePort(t)
	root := executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root", KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()
	controller, _ := signer.GenerateKey()
	agentAddr, err := Run(ctx, Options{Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e-jobs",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"), AgentBinary: e2eAgents(t)})
	if err != nil {
		t.Fatal(err)
	}
	waitAgentSocket(ctx, t, ex)
	seedLegacyNode(ctx, t, ex)

	tunnel := agentclient.Target{Agent: agentAddr, SSH: executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "jumpgate",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}}
	seqs := agentclient.NewMemorySeqStore()
	do := func(kind string, payload any, out any) {
		t.Helper()
		for attempt := 0; ; attempt++ {
			c, err := agentclient.Dial(ctx, tunnel, controller, seqs)
			if err == nil {
				var res agentclient.Response
				res, err = c.Do(ctx, kind, payload)
				c.Close()
				if err == nil {
					if res.Status != intent.StatusOK {
						t.Fatalf("%s: %s", kind, res.Result)
					}
					if out != nil {
						_ = json.Unmarshal(res.Result, out)
					}
					return
				}
			}
			// The agent restarts below: a refused dial is expected for a few seconds.
			if !errors.Is(err, agentclient.ErrUnreachable) || attempt > 30 {
				t.Fatalf("%s: %v", kind, err)
			}
			time.Sleep(time.Second)
		}
	}

	var rec intent.JobRecord
	do(intent.KindNodeMigrateNames, intent.MigrateNamesPayload{}, &rec)
	restarted := false
	for !intent.JobTerminal(rec.State) {
		var res intent.JobReadResult
		do(intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, &res)
		rec = res.Job
		switch rec.State {
		case intent.JobWaiting:
			do(intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID, Step: rec.Step, Attempt: rec.Attempt}, &rec)
		case intent.JobNeedsDecision:
			opt := intent.OptRetry
			if rec.Question.Key == "busy-user" {
				opt = "stop-processes"
			} else {
				t.Fatalf("unexpected question: %+v", rec.Question)
			}
			do(intent.KindJobDecide, intent.JobDecidePayload{ID: rec.ID, Decision: rec.Question.ID, Option: opt}, &rec)
		case intent.JobRunning:
			// stop-processes waits 30 s for a process that ignores SIGTERM: restart
			// the agent inside that window. The step's unit is PID 1's child and
			// must outlive it; the restarted agent must fold its result once.
			if strings.Contains(rec.Unit, "stop-units-1") && !restarted {
				if r, err := ex.Run(ctx, "systemctl restart jumpgate-agent", nil); err != nil || r.ExitCode != 0 {
					t.Fatalf("restart agent: %+v %v", r, err)
				}
				r, _ := ex.Run(ctx, "systemctl is-active "+rec.Unit+".service", nil)
				if strings.TrimSpace(r.Stdout) != "active" {
					t.Fatalf("the step's unit did not survive the agent restart: %q", r.Stdout)
				}
				restarted = true
				waitAgentSocket(ctx, t, ex)
			}
			time.Sleep(time.Second)
		}
	}
	if rec.State != intent.JobSucceeded || !restarted {
		t.Fatalf("job ended %s (restarted %v): %s", rec.State, restarted, rec.Error)
	}

	// The box is renamed.
	for cmd, wantOK := range map[string]bool{
		"id -u jumpgate-node": true, "id -u valve-node-app": false, "getent group jumpgate-node": true,
		"test -d /var/lib/jumpgate-node/369": true, "test -e /var/lib/valve-node-app": false,
		"test -f /etc/systemd/system/jumpgate-node-exec.service": true, "test -e /etc/systemd/system/valve-node-app-exec.service": false,
		"systemctl is-enabled jumpgate-node-exec.service": true, "test -f /etc/jumpgate/names.json": true,
		"systemctl is-active e2e-busy.service": false,
	} {
		r, err := ex.Run(ctx, cmd, nil)
		if err != nil || (r.ExitCode == 0) != wantOK {
			t.Errorf("%s: exit %d, want ok=%v", cmd, r.ExitCode, wantOK)
		}
	}
	var info intent.AgentInfo
	do(intent.KindAgentInfo, struct{}{}, &info)
	if info.Names != catalog.NamesCurrent || info.DataDir != "/var/lib/jumpgate-node/369" {
		t.Errorf("agent.info %+v", info)
	}

	// The whole history verifies, and the restart left exactly one outcome for the step.
	var res intent.JobReadResult
	do(intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, &res)
	var es []joblog.Entry
	finished := 0
	for _, j := range res.Entries {
		e, err := joblog.FromJSON(j)
		if err != nil {
			t.Fatal(err)
		}
		es = append(es, e)
		if e.Kind == joblog.KindStepFinished && strings.Contains(string(e.Body), `"step":"stop-units","attempt":1`) {
			finished++
		}
	}
	if _, err := joblog.Verify(agentAddr, rec.ID, es, 0, [32]byte{}); err != nil {
		t.Fatal(err)
	}
	if err := joblog.VerifyIntents(es); err != nil {
		t.Fatal(err)
	}
	if finished != 1 {
		t.Fatalf("stop-units attempt 1 finished %d times in the log", finished)
	}

	// The step's output is in the journal, and job.output reads it.
	var out intent.JobOutputResult
	do(intent.KindJobOutput, intent.JobOutputPayload{ID: rec.ID, N: 500}, &out)
	if len(out.Lines) == 0 || out.Cursor == "" {
		t.Fatalf("job.output %+v", out)
	}
}
```

Before relying on `--uid` in the test, check that `systemd-run --uid=` works on both bases (systemd 252 and 255 both support it).

- [ ] **Step 2: Run it**

Run: `scripts/e2e-agent.sh && BASE=ubuntu:24.04 scripts/e2e-agent.sh`
Expected: both PASS, including `TestE2EMigrateNamesSurvivesAnAgentRestart`. Without Docker locally, run this step in CI and record the run link in the commit message body.

- [ ] **Step 3: Commit**

```bash
git add internal/bootstrap/e2e_jobs_test.go
git commit -m "test(e2e): node.migrate-names on real systemd, across an agent restart

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 18: Docs and final verification

**Files:**
- Modify: `README.md` (a "Durable jobs" section next to the CLI and TUI sections, and the exit-code table if it lists codes)

- [ ] **Step 1: Write the README section**

Cover, in the README's voice:
- what a durable job is: it runs on the box as a transient systemd unit, outlives the TUI, the server and the agent process, and is recorded in a signed log;
- where jobs run: Linux boxes with systemd only. Any controller (macOS, Windows, Linux) can list them, watch them, decide them and adopt them;
- what a reboot does: the step in flight is reported as stopped without a result, and the job asks before rerunning it;
- leases: one controller drives a job; others watch it, and can adopt it after 90 s of silence. A controller resumes its own jobs when it restarts;
- `jumpgate jobs`, `jobs show|decide|adopt`, `jumpgate migrate-names HOST --yes`, the TUI inbox (`5`), `:migrate-names NAME`, the ⚑ count and the bell setting;
- `node.migrate-names`: what it renames, what it refuses (a box where another unit runs as `valve-node-app`; a mounted data root is kept in place after asking), and that a new setup uses `jumpgate-node` from the start.

- [ ] **Step 2: Run the whole verification**

Run, in order, and read each result:

```bash
go vet ./... && go test ./... -count=1 -race
GOOS=windows GOARCH=amd64 go vet ./...
GOOS=linux GOARCH=arm64 go build ./...
GOOS=darwin GOARCH=arm64 go build ./...
go test ./internal/server/ -run TestLegacyTargetRoutesAreFrozen -count=1
scripts/e2e-agent.sh && BASE=ubuntu:24.04 scripts/e2e-agent.sh
```

Expected: every command succeeds; the e2e runs as in Task 17. Then grep for leftovers: `grep -rn 'valve-node-app' --include='*.go' internal cmd | grep -v _test.go` lists only `catalog.LegacyNames`, the gateway and container constants left for sub-project 6 (Q3), and comments that name the legacy account.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: durable jobs and node.migrate-names

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

## Spec coverage

| Spec requirement (umbrella "Durable jobs", "Naming", build-order row 2; TUI spec) | Task |
|---|---|
| Transient unit `jumpgate-job-<id>[-<step>]` via `systemd-run`, survives agent restarts and controller disconnects | 6, 7, 17 |
| Manifest at `/var/lib/jumpgate/jobs/<id>.json`: kind, inputs (never rendered commands), current step, `driver`, version stamp, lease | 5, 7 |
| Output to the journal; resume with `--after-cursor` | 9 (`job.output`), 11 (stream), 16 (`jumpgate jobs output`) |
| Signed, append-only log of every intent and step outcome; adopter verifies | 3, 5, 7, 11, 17 (renewals excluded: Q7) |
| Autonomy: the agent runs as far as it can, stops only at a signed question; decision of the policy's tier | 7, 9 (Q5) |
| Adoption: list, read, watch read-only (fresh) or take over (stale) | 7, 11, 12 (Q4) |
| v1 step-level driver; waits at a step boundary without a controller; adoption is "rerun the plan" | 7, 12 |
| Completion markers (snapshot, recursive chown) | 2 |
| Agent refuses a plan version it does not know; manifest needs no change for the later driver | 7 (`unknown_plan`, `Driver`) |
| `node.migrate-names`: stop, usermod/groupmod, re-render, chown only by name, start, verify | 8, 17 |
| The catalog accepts both names during the transition | 1, 9 (marker) |
| Jobs inbox, ⚑ count, bell on ⚑, `jobs` on fleet rows (was `null`) | 11, 14, 15 |
| What jobs do off Linux | 6 (`DefaultRunner`), 9 (`jobs_unsupported`), J9 |

---

**Execution:** Plan complete and saved to `docs/superpowers/plans/2026-10-09-durable-jobs.md`. The user's standing preference is subagent-driven execution. Use superpowers:subagent-driven-development, dispatching each wave's tasks in parallel.
