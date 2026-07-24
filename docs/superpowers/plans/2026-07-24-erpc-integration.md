# eRPC Integration (Plan B)

> Inline TDD; commits per task on master (dev build, unpushed). Grounded in verified eRPC 0.1.1 facts.

**Goal:** Optionally run eRPC as a managed, de-rooted service in front of the node — local node primary (serves recent/everything it can), plus editable per-chain fallback/archive upstreams for historical state + failover. "Local full for everything, archive/backup for the rest."

**Verified eRPC facts:** binary `erpc --config <path>`; bare release assets `erpc_linux_x86_64` / `erpc_linux_arm64` at github.com/erpc/erpc (latest 0.1.1); config `projects[].upstreams[]` with `id`, `endpoint`, `evm.chainId`, `evm.blockAvailability.lower.latestBlockMinus:128` (full) or `lower: null` (archive); server `httpHostV4`/`httpPortV4` (default 0.0.0.0:4000); fallback via `tags:[tier:fallback]` + `routing.scoreMultipliers[].overall:0.2`. Verified default upstreams (eth_chainId): PulseChain `https://rpc.pulsechain.com` (0x171), v4 testnet `https://rpc.v4.testnet.pulsechain.com` (0x3af), Ethereum `https://ethereum-rpc.publicnode.com` (0x1).

## Global constraints
- eRPC is OFF unless `ERPCEnabled`. Default port 4000, default bind loopback (reuse the RPC-bind loopback-default model; its listen host is what gets exposed — the recommended thing to bind to Tailscale, not the raw node). Local-node upstream endpoint uses `w.RPCBind()` (so it works whether the node is loopback- or Tailscale-bound, on-box).
- De-rooted hardened unit like exec/beacon (`valve-node-erpc.service`, `User=valve-node`, ReadWritePaths=<DataDir>, NetBindCap when port <1024).
- Config at `<DataDir>/erpc.yaml`, chowned to the service user by the wire step's existing `chown -R`.
- Hand-render YAML via `text/template` (no new dep). `go build ./... && go test ./...`; UI via `npm run build`.

### Task 1: catalog — eRPC config + unit rendering + defaults
- `WireConfig`: `ERPCEnabled bool`, `ERPCPort int` (0=4000 via `ERPCHTTP()`), `ERPCBindAddr string` (""=127.0.0.1 via `ERPCBind()`), `ERPCUpstreams []string` (fallback/archive URLs).
- `networks.go`: `Network.DefaultUpstreams []string` — {1: publicnode, 369: rpc.pulsechain.com, 943: rpc.v4.testnet}. New helper `DefaultUpstreams(chainID) []string`.
- `RenderERPCConfig(w) (string, error)`: server host/port from bind/port; project `main`; upstream[0] = local node (`http://<RPCBind>:<ExecHTTP>`, `evm.chainId`, blockAvailability `latestBlockMinus:128` if `!w.Archive` else `lower: null`); then one upstream per `ERPCUpstreams` URL (chainId, `tier:fallback`, scoreMultipliers overall 0.2).
- `RenderERPCUnit(w) (string, error)`: hardened unit, ExecStart `erpc --config <DataDir>/erpc.yaml`, NetBindCap when `ERPCHTTP()<1024`.
- Tests: full-node config has latestBlockMinus:128 + fallback tags; archive config has lower:null; host/port reflect bind/port; unit has User=valve-node; disabled → callers skip (render still pure, gated by ERPCEnabled at the setup layer).

### Task 2: setup — install + wire eRPC (conditional)
- New `erpcInstallStep` (only when `ERPCEnabled`): download `erpc_linux_x86_64|arm64` from the pinned release to `/usr/local/bin/erpc`, chmod +x, verify `erpc --version`.
- Extend `wireStep`: when enabled, write `<DataDir>/erpc.yaml` + `valve-node-erpc.service`, `daemon-reload && enable --now`; content-diff restart like the others; the existing `chown -R` already covers the config. When disabled, stop+disable the unit if present (clean toggle-off).
- Handshake: when enabled, curl eRPC `http://<ERPCBind>:<port>` eth_chainId.
- Tests: install downloads the right asset; wire writes config+unit and enables; disabled path removes the unit.

### Task 3: server — validation + defaults surfacing
- Validate `ERPCPort` range, `ERPCBindAddr` IP, each `ERPCUpstreams` entry is an http(s) URL.
- Catalog API already serves `Network` (add `DefaultUpstreams` → auto-serialized) so the wizard can pre-fill.
- Tests: bad port/addr/upstream → 400.

### Task 4: ops — endpoints + firewall + diagnostics
- `Endpoints`: add the eRPC URL + reachability when enabled.
- Firewall `rpcNotPublicItem`: eRPC port graded like the others (exposing eRPC on Tailscale = pass-with-note; public = fail).
- Diagnostics: an `diag-erpc` rung (reachable + answers eth_chainId) when enabled.

### Task 5: UI — eRPC wizard step + endpoints card
- New wizard step (after mode): "RPC gateway (eRPC)" — enable toggle; when on: port, bind (reuse RPC-bind copy + unauthenticated caveat), and an upstream list pre-filled with `net.DefaultUpstreams` (editable, add/remove rows), framed as archive/backup for historical + failover with the decentralization-of-access note. Review shows the eRPC summary. Endpoints/dashboard show the eRPC URL as the primary front door when enabled.

### Task 6: docs — README v0.3 note.
