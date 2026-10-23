# The message board module, and the `Module` concept

Date: 2026-10-22
Status: draft for review
Research: `.claude/research/msgboard.md`
Sibling: `2026-10-17-docs-system-design.md` (where §8's steps are written)
Written against: `30a2e56` (jumpgate-app), `8230cc270` (valve-tech/reth)

## Goal

An operator running reth on PulseChain (369) or PulseChain Testnet v4 (943)
can turn on the **message board** (msgboard.xyz) from the app or the TUI and
see whether it works. It is local-only by default. Public reads and public
posting are separate, explicit choices.

The board is jumpgate's first **module**. A module is an optional capability
added to a node. More single-module configurations are planned for the same
release, txpool first (see `project-next-planning-session`). So the shape here
is a small, general `Module` definition, not a msgboard special case.

Success means:

- One toggle per node, in the app's node screen and in the TUI host view. It is
  shown only where it can work, and disabled with a reason on the same chain
  with another client.
- With the module on, `--http.api` names `msgboard` explicitly. It is never
  `all`, and a test proves that.
- With the module off, every rendered unit is byte-identical to today's, so
  upgrading jumpgate restarts no node.
- Status shows whether the board is served, whether it is accepting, and how
  many messages it holds.

## What exists today (grounding)

**The reth fork.**
- `crates/net/msgboard` serves the board, and `bin/reth/src/main.rs` wires it
  into both the ethereum and the pulsechain launch paths.
- The `msg/1` devp2p subprotocol is **always on**. There is no
  `--msgboard.enabled`.
- The board starts accepting once the node is not syncing and has at least one
  peer (`launch.rs:225`).
- The RPC namespace is installed with `merge_if_module_configured`, so it is
  served only on the transports whose `--*.api` names `msgboard`. `all`
  includes it, and that exposes `msgboard_addMessage` (`launch.rs:418-438`).
- Methods:
  - writes: `msgboard_addMessage`;
  - reads: `status`, `categories`, `content`, `contentPage`, `getMessage`;
  - subscription: `msgboard_subscribe`/`msgboard_unsubscribe`, with
    `msgboard_subscription` notifications.
- `msgboard_status` returns `{enabled, count, size, workMultiplier,
  workDivisor, headBlock}`, all as hex quantities.
- `msgboard_content` with no `limit` returns the whole board. That can be about
  167 MB of JSON (`rpc_api.rs` "Size"). The fork raises the default
  `--rpc.max-response-size` to 200 MiB for this (`msgboard_net.rs:69`).
- The fork validates `--http.api` with `ValveRpcModuleValidator`, which accepts
  `msgboard`. Upstream reth's `DefaultRpcModuleValidator` rejects it with
  `Unknown RPC module: 'msgboard'` (`module.rs:507-516`), so **an older or
  upstream binary given `msgboard` refuses to start**.
- The board's state: at most 10,000 messages of up to 8 KiB, about 80 MB in
  memory, flushed to a small MDBX database at `<datadir>/msgboard`. It needs
  only recent block hashes. A pruned node is enough.
- `msg/1` peer sessions are counted only by a Prometheus gauge,
  `msgboard.peer_sessions` (`metrics.rs:98`). No RPC namespace that jumpgate
  enables reports them.

**jumpgate.**
- The catalog already lists `reth` as `https://github.com/valve-tech/reth`,
  built from `main` (`internal/catalog/clients.go`). It is valid on chains 1,
  369 and 943 (`networks.go`).
- `execCommand` (`internal/catalog/units.go`) renders reth with **no
  `--http.api`**, so reth serves its standard set: `eth,net,web3`
  (`STANDARD_MODULES`, `module.rs:31`). Only the devnet passes `--http.api`
  (`devnet.go:90`, `eth,net,web3,txpool`).
- No `msgboard` string exists anywhere in `internal/` or `cmd/`.
- Node flags change in one way: by re-running setup with a new `WireConfig`.
  - Flow: `POST /api/targets/{id}/setup` → `setup.Plan` → the engine, which is
    idempotent (each step's Verify is also its "already done" probe).
  - The wire step byte-compares both units. On a change it stops both
    services, re-owns the data tree, rewrites the units and restarts both
    (`setup/steps.go:531-660`).
  - Setup runs over the legacy SSH executor for every target, including paired
    ones. "Setup over intents" is sub-project 5 (umbrella build order).
  - The umbrella spec makes `Setup` a routine **and** approval intent. The
    approval tier does not exist yet.
- The gateway: `RenderGatewayConfig` has no per-method routing. eRPC forwards
  any method to any upstream of the chain, fallbacks included.
- Gateway exposure tiers already exist as fields on `GatewayTLS`: Local (no
  TLS), Private (internal cert) and Public (ACME) (public-endpoint spec).
- The relay (`internal/relay`):
  - per-key `method_allow`/`method_block`;
  - an `allow_trace` gate over `trace_*`/`debug_*` (`policy.go`);
  - per-key rate limits (`ratelimit.go`);
  - per-call pricing (`billing.go` `PriceOf`).
  - Its WebSocket path synthesises only `eth_subscribe` (`ws.go:65`).
- Status: `monitor.Monitor.poll` runs loopback JSON-RPC probes. Paired boxes
  get the same `monitor.Snapshot` through `status.read`, which the gate runs
  (`agent/dispatch.go:101`).
- Node options in the app live in the Setup wizard (`ModeStep.tsx`). The
  Machine screen has Setup, Dashboard, Logs and Devnet sections.
- The TUI host view has six tabs and an `x` actions modal. Restart and stop use
  a typed confirmation. The TUI does not run the setup wizard (TUI spec, out of
  scope).
- Test harnesses:
  - `scripts/e2e-agent.sh` runs `-tags e2e` tests in `internal/bootstrap`
    against a systemd+sshd container with no client binaries.
  - `JUMPGATE_DOCKER_LIVE=1` runs `internal/setup/docker_live_test.go`
    against a real Docker engine with the devnet image,
    `ghcr.io/paradigmxyz/reth:latest` (upstream).

## Decisions

### D1. `Module` is catalog data, beside networks and clients

New file `internal/catalog/modules.go`:

```go
// Module is an optional capability added to a node's execution client.
type Module struct {
	ID      string // "msgboard"; also the docs reference name
	Name    string // "Message board"
	Summary string // one line, shown under the toggle
	SiteURL string // "https://msgboard.xyz"; "" for none

	Chains  []int    // chains it applies to: {369, 943}
	Clients []string // exec clients it applies to: {"reth"}

	Namespaces []string // added to the exec client's --http.api: {"msgboard"}
	Flags      []string // extra exec flags; msgboard adds none (fork defaults)

	Methods []ModuleMethod // every method the namespaces serve, classified (D7)

	Probe ModuleProbe // how to tell a binary and a live node support it (D9)

	NeedsSync bool      // serves nothing useful until the node is synced
	Resources Resources // extra RAM and disk, for display and preflight

	DocsStep string // the docs step that explains it (§8)
}

type MethodClass string

const (
	MethodRead      MethodClass = "read"      // bounded reads
	MethodUnbounded MethodClass = "unbounded" // reads with no response bound
	MethodWrite     MethodClass = "write"     // changes node state or gossips
	MethodStream    MethodClass = "stream"    // subscriptions
)

type ModuleMethod struct {
	Name  string
	Class MethodClass
}

type ModuleProbe struct {
	HelpContains string // a flag `<bin> node --help` must list: "--msgboard.block-range"
	StatusMethod string // a cheap read that must answer: "msgboard_status"
}

type Resources struct{ RAMBytes, DiskBytes uint64 }

func Modules() []Module
func ModuleByID(id string) (Module, bool)

// Applies reports whether m can run on chainID with execID. When it cannot,
// reason is one operator-facing sentence.
func (m Module) Applies(chainID int, execID string) (ok bool, reason string)
```

msgboard's entry:

| Field | Value |
|---|---|
| Chains | 369, 943 |
| Clients | `reth` |
| Namespaces | `msgboard` |
| Flags | none |
| Methods | `addMessage` write; `status`, `categories`, `contentPage`, `getMessage` read; `content` unbounded; `subscribe`, `unsubscribe` stream |
| Probe | `--msgboard.block-range` in `reth node --help`; `msgboard_status` answers |
| NeedsSync | true |
| Resources | RAM 256 MiB: the 80 MB board plus one 167 MB `content` build, rounded. Disk 128 MiB |

- **Why catalog data.** The catalog is already where networks, clients and
  unit templates live, and it does no I/O. The server's `/api/catalog`, the
  docs reference pages and the relay all read the same table.
- **Check against txpool.** A txpool module would be `Namespaces: {"txpool"}`,
  with every method read, on any chain, for `reth` and the geth family. It fits
  the same shape with no new field. A standalone box with no full node is
  something else (a base, not an add-on) and is left to its own spec.
- **Left out (YAGNI).** No module dependencies, no versions, no per-module
  tunables (the fork's `--msgboard.*` defaults are used), and no plugin
  loading.

### D2. A node records its modules on `WireConfig`

```go
type NodeModule struct {
	ID     string `json:"id"`
	Public string `json:"public,omitempty"` // "" | "read" | "post" (D6)
}

// on WireConfig:
Modules []NodeModule `json:"Modules,omitempty"`
```

- `omitempty` means every existing `config.json` and `node.json` reads as "no
  modules", unchanged.
- `catalog.ValidateModules(w)` refuses:
  - an unknown id;
  - a duplicate;
  - a module that does not apply to `w.ChainID`/`w.ExecID` (`Applies`'s reason
    becomes the error);
  - a `Public` value outside the three allowed values.
- `RenderUnits` and `setup.Plan` call it. So does the server's
  `validateWirePorts`, so a bad request is a 400 before anything runs.
- The server never drops an inapplicable module on its own. Dropping one is a
  visible step in the UI (D11).

### D3. `--http.api` is composed, rendered only when a module needs it, and never `all`

In `units.go`:

```go
// baseHTTPAPIs is each exec client's default --http.api set, the one it serves
// when the flag is absent. Only clients that some module applies to are listed.
var baseHTTPAPIs = map[string][]string{"reth": {"eth", "net", "web3"}}

// execHTTPAPI returns the --http.api value for w, or "" to render no flag.
func execHTTPAPI(w WireConfig) (string, error)
```

- **No modules: no flag.** The rendered unit stays byte-identical to today's.
  The wire step byte-compares units, so a jumpgate upgrade that rendered
  `--http.api eth,net,web3` everywhere would restart every node in the fleet.
- **With modules:**
  - The value is the client's base set, then each enabled module's namespaces
    in catalog order, with duplicates removed.
  - msgboard gives `--http.api eth,net,web3,msgboard`.
  - Module `Flags` follow, in the same order.
- **Refused tokens.** `execHTTPAPI` refuses any token in `{"all", "none"}`, and
  `Modules()`'s init-time check refuses a module that declares one. These are
  two separate guards for one rule.
- **The base is pinned in jumpgate**, not read from the binary. If the fork
  ever changes its default set, the composed flag keeps today's surface instead
  of following it.
- **No `--ws`.** The reth unit has no WebSocket listener today, and this module
  adds none. `msgboard_subscribe` therefore works only where a client already
  has a node WebSocket. That is open question 2.

### D4. Changing a module goes through setup: the existing path

The only code path that changes node flags is a setup re-run, so the toggle
uses it. There is no new job kind and no new intent.

- **New route.** `PUT /api/targets/{id}/modules/{module}`, body
  `{"enabled": bool, "public": "" | "read" | "post"}`.
  - It loads the stored wire (`setupExisting` for paired boxes, as setup does)
    and applies the change.
  - It validates the result (D2).
  - It then runs the same code `handleStartSetup` runs, factored into one
    function: claim the run, `refuseDuringJob`, `refuseLegacyOnMigratedBox`,
    persist the wire, `setup.Plan`, stream.
  - It returns the setup run, and the existing setup SSE stream reports
    progress.
- **Which steps act.**
  - preflight, account and toolchain verify and pass.
  - install-exec passes unless the binary lacks the module (D9).
  - The wire step's Verify fails because the exec unit differs, so it rewrites
    and restarts.
  - start and handshake verify as usual.
  - A new `modules` step after handshake checks the live probe (D9).
- **Fix in this work: restart only what changed.** The wire step now restarts
  both units when either differs. It changes to restart only the units whose
  content changed, so a module toggle restarts the execution client and leaves
  the beacon running.
  - The step's stop-before-chown applies only when a unit changed, which keeps
    the existing rationale.
  - The recursive chown still runs. Its cost on a multi-TB data dir is open
    question 6.
- **Gateways too.** When `Public` changes, or when the module is enabled or
  disabled, every gateway that references this node (`UpstreamManagedNode`) is
  re-rendered and re-provisioned through the existing gateway provision path,
  because its routing depends on the module (D6).
- **Signed intents later.** Paired boxes reach the server the same way setup
  already does, over SSH. When sub-project 5 makes `Setup` a durable job behind
  a signed intent (routine and approval tier), the module route calls it
  unchanged: module state is part of `WireConfig`, which is the payload that
  intent signs. Nothing here needs undoing.

| Change | Exec client | Beacon | Gateway | Rebuild |
|---|---|---|---|---|
| Turn on, binary already has msgboard | restart | untouched | re-provisioned | no |
| Turn on, binary predates msgboard | restart after rebuild | untouched | re-provisioned | yes, from `valve-tech/reth` `main` (tens of minutes) |
| Turn off | restart | untouched | re-provisioned | no |
| Change `Public` only | untouched (unit identical) | untouched | re-provisioned | no |

The toggle says which row applies before it runs: "This restarts reth on
box-1. The beacon keeps running." The rebuild row also gives the time.

### D5. Default exposure: the box and your own gateway

- **The node.** The namespace is served on the node's HTTP listener, which
  binds `RPCBind()`: loopback by default.
  - If the operator set `RPCBindAddr` to a routable address (for example their
    Tailscale IP), msgboard, including `addMessage`, is served there as well.
    reth cannot split namespaces per interface.
  - In that case the toggle shows "Reachable on 100.x.y.z as well, including
    posting", and turning it on needs one extra confirmation.
  - jumpgate does not try to fix this. That address is the operator's own
    network, by their own choice.
- **The gateway.** A gateway in the Local or Private tier counts as "yours".
  It routes every msgboard method except streams to the nodes that serve them.

### D6. Public exposure is an explicit per-node choice, enforced at the gateway

`NodeModule.Public` has three values:

| `Public` | Public-tier gateway routes to this node | Shown as |
|---|---|---|
| `""` (default) | no msgboard method | "Local only" |
| `"read"` | the module's `read` methods | "Public reads" |
| `"post"` | `read` methods and `write` methods | "Public reads and posting" |

- **Never public.** `unbounded` methods (`msgboard_content`) are never routed
  by a Public-tier gateway. One call can build 167 MB, and nothing can price or
  rate-limit a response by its size. `msgboard_contentPage` is the bounded way
  to read. `stream` methods are not routed by any gateway: the node has no
  WebSocket listener (D3).
- **How it is enforced.** Per upstream, in `RenderGatewayConfig`.
  - eRPC upstreams take `ignoreMethods` (wildcards allowed). The resolver
    derives a new `GatewayUpstream.IgnoreMethods` the same way it derives
    `Endpoint`: from the referenced target, never stored.
  - External and fallback upstreams on any chain get `msgboard_*` always. This
    stops `addMessage` from failing over to `rpc.pulsechain.com`, and stops a
    read from getting a confusing "method not found" from a fallback.
  - A managed node without the module gets `msgboard_*`.
  - A managed node with the module gets the module's methods minus what the
    gateway's tier and the node's `Public` allow.
  - With every upstream ignoring a method, eRPC answers it as unsupported. That
    is the gateway-level allow-list, with no second mechanism.
- **Hedging and retries.** eRPC may hedge a write to two msgboard-enabled
  upstreams. That is harmless: a message's id is its PoW hash, and the board
  rejects a duplicate.
- **Why public is opt-in.** `addMessage` is a write, even though PoW gates it.
  A public poster lends the operator's node and bandwidth to anyone's messages.
  The operator decides that, and posting implies reads.

The toggle shows `Public` only when the node is behind at least one
Public-tier gateway. Otherwise there is nothing public to choose.

### D7. The relay: posting needs an explicit key grant

The relay sits behind Caddy in front of a Public-tier gateway, so D6 already
bounds what reaches the node. The relay adds key policy on top of that:

- **Explicit only.** A method of class `write` in any catalog module is served
  only to a key whose `method_allow` names it exactly.
  - An empty allow list, which means "everything" for ordinary methods, does
    not cover it.
  - This is the same idea as `allow_trace`, but it needs no new key column.
  - The set comes from `catalog.Modules()`, so the next module's writes are
    covered without a relay change.
  - Refusal: 403 "msgboard_addMessage needs to be on this key's method allow
    list".
- **Rate limits.** The existing per-key per-second and per-day limits apply,
  one call per method, and a batch counts as its calls. There is no
  module-specific limiter.
- **Metering.**
  - msgboard methods are priced through the existing price book (`PriceOf`) and
    take the default price until the book names them.
  - The book gets entries for `msgboard_contentPage` and `msgboard_addMessage`
    as part of rollout (open question 8).
- **Subscriptions.** On a relay WebSocket, `msgboard_subscribe` is refused with
  "subscriptions other than eth_subscribe are not supported on this endpoint;
  poll msgboard_contentPage". That is what the relay's stream code does for
  anything it does not synthesise.

### D8. Status comes from observation, not from config

`monitor.Snapshot` gains:

```go
Modules []ModuleStatus `json:"modules,omitempty"`

type ModuleStatus struct {
	ID        string `json:"id"`
	Served    bool   `json:"served"`    // StatusMethod answered
	Accepting bool   `json:"accepting"` // msgboard_status.enabled
	Count     uint64 `json:"count"`
	Size      uint64 `json:"size"`
	HeadBlock uint64 `json:"headBlock"`
	Error     string `json:"error,omitempty"`
}
```

- **What is probed.** On each poll, for every module that `Applies` to the
  node's chain and client, the monitor calls the module's `StatusMethod` on
  loopback. msgboard is the only module, so the decoding is a small per-module
  function keyed by id.
- **Why observe.** Probing what is actually served, instead of reading config:
  - works on a paired box whose gate reads a `node.json` written at pairing,
    before any module existed;
  - catches a unit that says `msgboard` on a binary that is not serving it.
- **Cost.** One extra loopback call per 5 s poll, on reth nodes on 369/943
  only. Everything else is untouched.
- **Older gates.** A paired box whose gate predates this omits `modules`. The
  UI shows "Status needs a newer gate on this box" and keeps the toggle usable,
  because the toggle goes through setup, not the gate.
- **`msg/1` peers are not shown in v1.** They are observable only through the
  fork's Prometheus gauge (which needs a `--metrics` listener jumpgate does not
  run) or through `admin_peers` capabilities (which needs the admin namespace,
  and that will not be enabled for this). The status line shows the node's eth
  peer count, labelled as such. Open question 1.

**One status line**, decided in this order:

1. The exec client is not active: "Node stopped".
2. On in config but `Served` is false:
   - with the unit restarted under 60 s ago, "Starting";
   - after that, an error (D10).
3. `execSyncing`: "Waiting for sync". The board accepts nothing while the node
   syncs.
4. `execPeers == 0`: "No peers".
5. `Accepting` is false: "Starting".
6. Otherwise: "Accepting · 1,234 messages · 3.1 MB".
7. Off in config but `Served` is true: "Served but not set by jumpgate". This
   is a hand-edited unit, shown and not fought.

### D9. Detecting a reth that lacks msgboard happens before the unit is written

An upstream or pre-msgboard reth given `--http.api …,msgboard` exits at
startup, and `Restart=always` turns that into a crash loop. So detection
happens before the unit is written, at two points.

1. **Binary probe, in install-exec's Verify.**
   - When an enabled module has `Probe.HelpContains`, Verify also runs
     `/usr/local/bin/reth node --help` and looks for that flag.
   - If it is missing, Verify fails and Run rebuilds from `valve-tech/reth`
     `main`, the existing BuildCmd.
   - If the rebuilt binary still lacks it, the step fails with: "reth at
     /usr/local/bin/reth has no message board support (`reth node --help`
     lists no `--msgboard.*` flags), even after a rebuild from
     valve-tech/reth main." The unit is not rewritten.
   - Install runs before wire, so a box never gets a unit its binary rejects.
2. **Live probe, in a new `modules` step after handshake.**
   - It polls each enabled module's `StatusMethod` until the handshake timeout
     runs out.
   - `-32601` (method not found) fails the step with: "reth is running but not
     serving msgboard. The unit's --http.api may have been edited."
   - If the exec journal shows `Unknown RPC module`, that line is quoted.
   - Verify is the same probe, so a re-run is cheap.

Neither point reads `reth --version`. The fork builds from `main` with no tags
(`PinVersion: "main"`), so asking for the capability itself is the only
reliable test.

### D10. Errors

| Case | Where it is caught | What the operator sees |
|---|---|---|
| Non-reth client on 369/943 | `Applies`; UI before submit; server 400 | Toggle disabled: "The message board needs reth (valve fork). This node runs go-pulse." |
| Other chain (1, devnet, gateway-only) | `Applies` | Toggle hidden |
| Switching a msgboard node to another client in the wizard | Review step | "The message board will be turned off: go-pulse does not support it." The module is removed from the submitted wire. The server refuses a wire that still has it |
| Binary predates msgboard | install-exec Verify (D9) | "Rebuilding reth to add message board support (tens of minutes)", then the restart |
| Binary still lacks it after rebuild | install-exec Run | The D9 error; the unit is unchanged |
| Running but not serving | `modules` step; status `Served` false | The D9 live-probe error, with the journal line if there is one |
| Node not synced | Status | "Waiting for sync". Enabling is allowed; nothing fails |
| No peers | Status | "No peers" |
| A job runs on the box | `refuseDuringJob` | The existing 409: one job at a time |
| Paired box with root SSH off | `getShellExecutor` | The existing setup dial error. Until sub-project 5 a module change needs the same SSH access setup needs (open question 7) |
| Older gate | Status | "Status needs a newer gate on this box" |
| msgboard method on a gateway with no serving upstream | eRPC | eRPC's unsupported-method error. The gateway screen lists which chains route msgboard |

There is no automatic rollback. If the `modules` step fails, the error offers
"Turn the message board off", which is the same route with `enabled: false`.

### D11. UI

**App (`cmd/jumpgate/web`).**
- **A Modules section** in the Machine screen, after Dashboard. It appears only
  once the node is set up and some module applies or is disabled with a reason.
  One card per module:
  - the name and `Summary`;
  - a toggle;
  - the D8 status line;
  - `Resources` ("about 256 MB RAM");
  - an "Open msgboard.xyz" link (`SiteURL`, opens in a new tab);
  - a docs link (`DocsStep`).
- **Public choice.** Behind a Public-tier gateway, a radio: "Local only" /
  "Public reads" / "Public reads and posting". The posting option warns: "Anyone
  can post through your node. Posts cost the sender proof of work, not you."
- **Before it runs.** A dialog with D4's table row ("Restarts reth. Beacon keeps
  running."), then the setup run's progress inline.
- **New setups.** The wizard's ModeStep gets a "Modules" group with the same
  checkboxes, filtered by `Applies` for the chosen chain and client, and
  ReviewStep lists them. A new box gets the module in its first setup, with no
  second restart.
- **The catalog.** `/api/catalog` gains `modules`: the `Module` fields the UI
  needs, `Applies` evaluated per chain and client, and the reasons. No `Applies`
  logic is copied into TypeScript.

**TUI (`internal/tui`).**
- **The Overview tab** gets a Modules block: one line per applicable module,
  `Message board  on · Accepting · 1,234 msgs · local only`. A disabled module
  shows its reason. `msgboard.xyz` appears as text, since a terminal cannot
  open a browser on a remote jump host; the line is also the docs reference.
- **Keys.** `m` (Overview), and `x` actions, open "modules": a list with
  toggles, then a typed confirmation (`restart`), as restart and stop have.
  Then the setup run's progress, from the same setup SSE stream the server
  already serves. `Public` is a second line in that modal.
- **Scope.** This changes one line of the TUI spec's scope. The TUI still does
  not run the setup wizard, but it does start a module change, which the server
  carries out as a setup run. The server picks the transport, as TUI D1 says.
- **Fit.** 80×24, ASCII and NO_COLOR pass the existing width and palette tests.

User-facing text says **gate** for the per-box program (`jumpgate-agent.service`
and Go names are unchanged). Write "the box's gate" wherever "gateway" is near.

## Testing

**Unit: flag composition (`internal/catalog`).**
- A table over `execHTTPAPI` and `RenderUnits`:
  - no modules: no `--http.api`, and the unit is byte-identical to a golden
    taken at `30a2e56` for every network × client pair;
  - msgboard on 369 and on 943: `--http.api eth,net,web3,msgboard`;
  - duplicates collapse;
  - the order is stable.
- `ValidateModules`: chain 1, go-pulse, erigon-pulse, an unknown id, a
  duplicate and a bad `Public` each give their own error.

**The `all` guard.**
- `TestNoAPIListIsAll` renders every network × exec client × every subset of
  applicable modules, plus `DevnetCommand`. It splits every `--http.api` and
  `--ws.api` value on commas and fails on `all` or `none`.
- An init-time check refuses a `Module` whose `Namespaces` contain either.
- A test asserts that the check fires, using a fake table.

**Gateway (`gateway_test.go`).**
- Goldens for `ignoreMethods`:
  - an external fallback;
  - a managed node without the module;
  - a managed node with the module behind Local, Private and Public tiers, at
    each `Public` value.
- `msgboard_content` is ignored at every Public-tier setting, and
  `msgboard_subscribe` everywhere.

**Relay (`policy_test.go`, `ws_test.go`).**
- `msgboard_addMessage`:
  - refused with an empty allow list;
  - refused with a non-empty allow list that does not name it;
  - served when the allow list names it.
- `msgboard_status` follows ordinary key policy.
- A batch with one ungranted write is refused whole.
- `msgboard_subscribe` over WebSocket is refused with the stated message.

**Setup (`steps_test.go`, fake executor).**
- install-exec Verify fails on a `--help` with no `--msgboard.block-range` and
  passes on one with it.
- The rebuild-still-missing error is the D9 text, and wire never runs.
- When only the exec unit changed, the wire step restarts only the exec unit.
- The `modules` step maps `-32601` and `Unknown RPC module` to their messages.

**Monitor.**
- `msgboard_status` parsing (hex quantities, `enabled`).
- `-32601` → `Served: false`.
- A non-reth wire makes no extra call.

**Server.**
- The route:
  - an inapplicable module → 400;
  - a job running → 409;
  - a change of `Public` only → no unit write and a gateway re-provision;
  - `enabled: false` removes the flag.
- `/api/catalog` carries the modules and their reasons.

**Web and TUI.**
- Vitest for the modules model: hidden, disabled with a reason, the status-line
  precedence, and the dropped-on-client-change notice.
- A TUI golden of the Overview Modules block and the modal at 80×24, in ASCII.

**End to end.** Two harnesses already exist. Neither has a real msgboard node,
so:

- `TestMsgboardLive` in `internal/setup` runs behind `JUMPGATE_DOCKER_LIVE=1`,
  in the `docker_live_test.go` style.
  - **Negative, always:** it runs the upstream devnet image
    (`ghcr.io/paradigmxyz/reth:latest`) with `node --dev` and the composed
    `--http.api`. It asserts the container exits with `Unknown RPC module`,
    and that D9's live-probe mapping names it. This proves the guard is needed
    and works.
  - **Positive, when `JUMPGATE_MSGBOARD_IMAGE` names a valve-tech/reth image:**
    the same argv serves, `msgboard_status` answers, and the monitor parses it.
    `accepting` is false, because a `--dev` node has no peers, and the test
    asserts exactly that. Where that image comes from is open question 4.
- The `scripts/e2e-agent.sh` container gains a stub `/usr/local/bin/reth`: a
  shell script whose `node --help` lists `--msgboard.block-range`.
  `-tags e2e` then runs the module route's setup steps over real SSH and
  systemd: the unit is rewritten with the composed flag, only the exec unit
  restarts, and a second run is a no-op. The `modules` live probe is skipped
  there, since the stub serves no RPC.

## 8. Docs: the "Run the message board" pathway

An operator pathway in the docs system. Each step is atomic. Its `verified` map
holds the functions marked `// docs:covers <step-id>`, so changing any of them
fails the build until the step is re-checked.

```yaml
id: msgboard
title: Run the message board
audience: operator
after: [first-box]
steps: [modules-what, msgboard-enable, msgboard-status, msgboard-expose, msgboard-troubleshoot]
```

| # | Step id | Requires | `docs:covers` targets | Scene |
|---|---|---|---|---|
| 1 | `modules-what` | `first-sync` | `internal/catalog.Modules`, `internal/catalog.Module.Applies` | none |
| 2 | `msgboard-enable` | `modules-what`, `pair-box` | `internal/catalog.execHTTPAPI`, `internal/catalog.ValidateModules`, `internal/server.(*Server).handleSetModule`, `internal/tui.newModulesModal` | `app machine-modules`, `tui host-modules` |
| 3 | `msgboard-status` | `msgboard-enable` | `internal/monitor.probeModules`, `internal/monitor.parseMsgboardStatus` | `tui host-overview-msgboard` |
| 4 | `msgboard-expose` | `msgboard-enable` | `internal/catalog.moduleIgnoreMethods`, `internal/relay.checkOneMethod` | `app machine-modules-public` |
| 5 | `msgboard-troubleshoot` | `msgboard-status` | `internal/setup.installStep`, `internal/setup.modulesStep` | none |

- **A new typed reference.** `{{module msgboard}}` resolves against
  `catalog.Modules()` and renders the module's name, linked to the step.
- **The generated reference page.** "Networks and clients" gains a Modules
  table: id, name, chains, clients, namespaces, and per-method class. It is
  generated, so the D7 classification is documented by the same table the relay
  enforces.
- **Step 4 says it plainly.** Posting is a write, `msgboard_content` is never
  public, and posting through a relay key needs the method on its allow list.
- **Step 5** carries the D10 table in prose: "Waiting for sync", "No peers",
  the rebuild, and the not-served error.
- **Wording.** Steps say "gate" for the per-box program.
- **When steps land.** Each lands with the code it covers, as the staleness
  rule requires. If the docs package is not on `main` when this ships, the
  `docs:covers` markers are still added, and the steps are written as soon as
  the loader lands.

## Rollout

1. **Catalog.** `Module`, `Modules()`, `Applies`, `WireConfig.Modules`,
   `ValidateModules`, `execHTTPAPI`, and the `all` guard. No behaviour change:
   the goldens prove the units are identical.
2. **Setup.** The install-exec binary probe, the `modules` step, and the wire
   step restarting only changed units.
3. **Monitor.** The module probe, and the `status.read` field (released with
   the next gate).
4. **Server, app and gateway routing.** The module route (sharing the setup
   kickoff), `/api/catalog` modules, the Machine Modules section, the wizard
   group, and gateway `ignoreMethods` with `Public` fixed at `""`. With
   `Public` fixed, a Public-tier gateway routes no msgboard method, so the
   module is never public before step 6.
5. **TUI.** The Overview block and the modal.
6. **Public exposure.** `Public` offered in both UIs, the relay's
   explicit-only writes and its WebSocket refusal, and the price-book entries.
7. **Docs steps and scenes**, with steps 1–6 as they land.
8. **Dogfood.** On a 943 box, then a 369 box, before the release notes mention
   msgboard.xyz.

The release then carries the module concept for txpool to reuse.

## Out of scope

- `--msgboard.*` tunables in the UI.
- A WebSocket listener on the node.
- `msg/1` peer counts.
- Ethereum mainnet. The fork is chain-agnostic in code, but no msgboard network
  on chain 1 is known.
- erigon-pulse's own msgboard (`--msgboard.enabled`, RC8). Its PoW is the older
  construction, and reth rejects it (`docs/msgboard-parity-gaps.md` §21).
- Standalone single-module boxes with no full node.
- Running msgboard.xyz's UI, indexer or archive.

## Open questions

1. **`msg/1` peers.** Should the module add a loopback-only `--metrics
   127.0.0.1:<port>`, so the monitor can read `msgboard.peer_sessions`? It is
   the most useful "is gossip working" signal, at the cost of one more
   listener.
2. **Subscriptions.** Does msgboard.xyz need `msgboard_subscribe` against a
   self-hosted node? If it does, the module needs `--ws`, `--ws.api …,msgboard`
   and gateway WebSocket routing, which reopens D3 and D6 for streams.
3. **`msgboard_content` in public.** Does msgboard.xyz's UI call `content`
   with no `limit` against the endpoint a user configures? If it does, "Public
   reads" breaks that UI until it pages with `contentPage`. This needs a check
   of the UI bundle (`index-C0q4qPGw.js`).
4. **A fork image for the live test.** Is there a published `valve-tech/reth`
   container image? If not, CI either builds one (slow) or the positive half
   stays manual.
5. **Is the RAM figure right?** The 256 MiB in `Resources` is derived from
   defaults, not measured. Should preflight enforce it, or only display it?
6. **The chown cost.** The wire step's recursive chown on every Run, over a
   multi-TB reth data dir, could make a toggle slow. If it is measured as slow,
   should the step skip the walk when the owner marker verifies?
7. **Paired boxes without root SSH.** Until sub-project 5 ships, a module
   change needs setup's SSH access. Is that acceptable for this release, or
   should module changes be the first `Setup` job?
8. **Pricing.** What should `msgboard_contentPage` and `msgboard_addMessage`
   cost in the relay's price book? `contentPage` returns up to its maximum page
   size in one call.
9. **Pinning.** Turning the module on can trigger a rebuild from `main` at
   depth 1. Should the reth entry pin a known-good fork commit now that a
   feature depends on it?
