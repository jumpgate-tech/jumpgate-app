# Wizard Storage + Consensus Cluster (Plan A)

> Executed inline TDD; UI verified by build. Commits per task on master (dev build, unpushed).

**Goal:** In the setup wizard, let the operator choose the data location and see free disk there, auto-downgrade archive→full when the location can't hold archive (re-checked on location change), and toggle checkpoint sync (with an editable checkpoint URL).

**Constraints:** Reuse `ops.diskFreeBytes` (nearest-existing-ancestor `df` walk). Default `CheckpointURL` empty = per-network default; `NoCheckpoint` omits `--checkpoint-sync-url` (sync from genesis). 10% safety margin on fit checks, matching preflight. `go build ./... && go test ./...`; UI via `npm run build`.

### Task 1: `ops.FreeBytesAt` + `GET /api/targets/{id}/disk?path=`
- ops: exported `FreeBytesAt(ctx, e, path) (uint64, error)` wrapping `diskFreeBytes`.
- server: `handleDiskFree` — target must exist (wire not required, since the probe runs pre-setup); read `path` query (400 if empty); `getExecutor`; return `{path, freeBytes}` (502 on probe error).
- Tests: happy path (scripted df) + missing-path 400 + unknown target 404.

### Task 2: catalog checkpoint config
- `WireConfig.CheckpointURL string` (override; empty = network default), `WireConfig.NoCheckpoint bool`.
- `beaconCommand`: `url := w.CheckpointURL; if "" { url = net.CheckpointURL }`; render `--checkpoint-sync-url url` only when `!w.NoCheckpoint`; always render `--genesis-beacon-api-url url`.
- Server validation: non-empty `CheckpointURL` must parse as http(s) URL.
- Tests: override renders; NoCheckpoint drops the flag; default unchanged.

### Task 3: UI storage — location + free disk + auto-downgrade
- api.ts: `getDiskFree(id, path)`.
- wizard mode step: promote the data-location field out of Advanced; on step entry and on location `change`, probe free disk; show "Free at <path>: X · archive needs ~A / full needs ~F" with fit marks; if archive selected and free < archive×1.1 but ≥ full×1.1, auto-switch to full with a reversible note; if neither fits, warn.

### Task 4: UI checkpoint toggle + editable URL
- wizard: a consensus block — checkpoint-sync toggle (default on) + URL input (default `net.CheckpointURL`, shown when on); thread `CheckpointURL`/`NoCheckpoint` into `StartSetupRequest`; review step shows checkpoint status.

### Task 5: docs
- README note under v0.3 (unreleased) once the cluster lands.
