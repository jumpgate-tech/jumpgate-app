// Thin typed wrapper around valve-node-app's JSON/SSE API (internal/server/api.go).
// Every type here mirrors the *actual wire shape* of the corresponding Go
// struct, not a guessed camelCase version of it: several response structs
// (catalog.Network, executor.SSHConfig, catalog.WireConfig) carry no `json`
// tags, so encoding/json marshals them using the bare (capitalized) Go field
// name. Where a struct DOES carry explicit tags (catalogClient, setup.Event,
// monitor.Snapshot, logwatch.Hit, settings), this file uses the tagged
// lowerCamelCase names. Verified against the real server via a scratch
// marshal/unmarshal test, not just by reading the struct definitions.
//
// The browser sends the session cookie automatically on same-origin fetch
// and EventSource calls, so nothing here needs an Authorization header.

// ---------------------------------------------------------------------
// catalog
// ---------------------------------------------------------------------

export interface Network {
  ChainID: number;
  Name: string;
  CheckpointURL: string;
  ExecClients: string[];
  BeaconClients: string[];
  LearnURL: string;
  // SnapshotSizeTB is the size, in decimal TB, of *Valve's reth snapshot
  // artifact* for this chain — the `snapshot.sizeTB` figure from
  // learn.valve.city's network data, and the only published size that
  // exists. It is reth-specific and tied to one block height: it is not a
  // per-client node size (nothing is published for go-pulse, erigon-pulse
  // or geth) and not a full-vs-archive figure.
  //
  // The wizard uses it as the archive-tier baseline and scales it by
  // FULL_TIER_FRACTION for the full tier, mirroring Go's ExpectedBytes —
  // but that fraction is an unsourced placeholder, so anything shown to
  // the operator must be labelled as an estimate, never as measured fact.
  // See catalog.Network.SnapshotSizeTB for the authoritative caveat.
  SnapshotSizeTB: number;
  // Human sync-time estimates: SyncLabel for a snapshot-assisted sync
  // (a full/pruned node), GenesisSyncLabel for a from-genesis sync (what an
  // archive node generally needs). Baseline figures — real time scales with
  // CPU and disk speed.
  SyncLabel: string;
  GenesisSyncLabel: string;
}

export interface CatalogClient {
  id: string;
  kind: string;
  repo: string;
  pinVersion: string;
  toolchain: string;
  learnUrl: string;
  // snapshotSupported reports whether this execution client can be
  // fast-synced from Valve's execution snapshot (an alternative to
  // consensus checkpoint sync, which only speeds up the beacon side).
  snapshotSupported: boolean;
}

export interface Catalog {
  networks: Network[];
  clients: CatalogClient[];
}

export interface Host {
  os: string;
  arch: string;
}

// getHost reports the OS/arch valve-node-app itself runs on — used to decide
// whether local node setup is viable (needs a Linux host).
export function getHost(): Promise<Host> {
  return request<Host>("/api/host");
}

export function getCatalog(): Promise<Catalog> {
  return request<Catalog>("/api/catalog");
}

// ---------------------------------------------------------------------
// targets
// ---------------------------------------------------------------------

export interface SSHConfig {
  Host: string;
  User: string;
  KeyPath: string;
  HostKeyFile?: string;
  Port?: number;
}

export interface WireConfig {
  ChainID: number;
  ExecID: string;
  BeaconID: string;
  DataDir: string;
  JWTPath: string;
  Archive: boolean;
  // ExecHTTPPort/BeaconHTTPPort/ExecP2PPort are omitted (undefined) when a
  // target used the server's defaults (8545/5052/30303) — the server
  // zero-values these to the same defaults, so there's no wire distinction
  // between "not set" and "set to default" once persisted.
  ExecHTTPPort?: number;
  BeaconHTTPPort?: number;
  ExecP2PPort?: number;
  // RPCBindAddr is the host the exec/beacon HTTP RPC binds to. Omitted when
  // loopback (the default); set to a routable address (e.g. a Tailscale IP)
  // to reach the node's RPC from another machine.
  RPCBindAddr?: string;
}

export type TargetMode = "local" | "ssh";

export interface Target {
  id: string;
  mode: TargetMode;
  ssh?: SSHConfig;
  wire?: WireConfig;
}

export function listTargets(): Promise<Target[]> {
  return request<Target[]>("/api/targets");
}

export interface AddTargetRequest {
  id: string;
  mode: TargetMode;
  ssh?: SSHConfig;
}

export function addTarget(body: AddTargetRequest): Promise<Target> {
  return request<Target>("/api/targets", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(body),
  });
}

export function deleteTarget(id: string): Promise<void> {
  return request<void>(`/api/targets/${encodeURIComponent(id)}`, { method: "DELETE" });
}

// ---------------------------------------------------------------------
// setup wizard
// ---------------------------------------------------------------------

// StartSetupRequest is catalog.WireConfig's request shape. DataDir/JWTPath
// are optional — the server fills in `/var/lib/valve-node-app/<chainId>` and
// `<dataDir>/jwt.hex` when omitted (see handleStartSetup).
export interface StartSetupRequest {
  ChainID: number;
  ExecID: string;
  BeaconID: string;
  DataDir?: string;
  JWTPath?: string;
  Archive: boolean;
  // Only send these when the operator changed a port from its default in
  // the wizard's Advanced section — see WireConfig's comment above.
  ExecHTTPPort?: number;
  BeaconHTTPPort?: number;
  ExecP2PPort?: number;
  // Only send when the operator set a non-loopback RPC bind address.
  RPCBindAddr?: string;
  // Consensus checkpoint sync: CheckpointURL overrides the network default;
  // NoCheckpoint disables checkpoint sync (sync from genesis).
  CheckpointURL?: string;
  NoCheckpoint?: boolean;
  // Execution-client snapshot restore: only sent when the selected exec
  // client supports it (catalog client's snapshotSupported) and the
  // operator opted in. SnapshotKey is the free key issued at valve.city.
  ExecSnapshot?: boolean;
  SnapshotKey?: string;
}

export interface DiskFree {
  path: string;
  freeBytes: number;
}

export function getDiskFree(id: string, path: string): Promise<DiskFree> {
  return request<DiskFree>(
    `/api/targets/${encodeURIComponent(id)}/disk?path=${encodeURIComponent(path)}`,
  );
}

export function startSetup(id: string, wire: StartSetupRequest): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/targets/${encodeURIComponent(id)}/setup`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(wire),
  });
}

export interface SetupEvent {
  stepId: string;
  line?: string;
  done?: boolean;
  err?: string;
}

export function streamSetup(id: string, onEvent: (ev: SetupEvent) => void): () => void {
  const es = new EventSource(`/api/targets/${encodeURIComponent(id)}/setup/stream`);
  es.onmessage = (msg) => {
    try {
      onEvent(JSON.parse(msg.data) as SetupEvent);
    } catch {
      // Malformed frame — drop it rather than crash the stream handler.
    }
  };
  return () => es.close();
}

// ---------------------------------------------------------------------
// monitor
// ---------------------------------------------------------------------

export interface Snapshot {
  at: string;
  execSyncing: boolean;
  execHead: number;
  refHead: number;
  beaconSlot: number;
  beaconDistance: number;
  execPeers: number;
  beaconPeers: number;
  diskUsedPct: number;
  execActive: boolean;
  beaconActive: boolean;
}

export function streamMonitor(id: string, onSnapshot: (s: Snapshot) => void): () => void {
  const es = new EventSource(`/api/targets/${encodeURIComponent(id)}/monitor/stream`);
  es.onmessage = (msg) => {
    try {
      onSnapshot(JSON.parse(msg.data) as Snapshot);
    } catch {
      // ignore malformed frame
    }
  };
  return () => es.close();
}

// ---------------------------------------------------------------------
// logs
// ---------------------------------------------------------------------

export interface Hit {
  unit: string;
  line: string;
  at: string;
  signature: string;
  severity: string; // info|warn|error|critical
  explain: string;
  learnUrl?: string;
}

export function getLogs(id: string, n = 200): Promise<Hit[]> {
  return request<Hit[]>(`/api/targets/${encodeURIComponent(id)}/logs?n=${n}`);
}

export function streamLogs(id: string, onHit: (h: Hit) => void): () => void {
  const es = new EventSource(`/api/targets/${encodeURIComponent(id)}/logs/stream`);
  es.onmessage = (msg) => {
    try {
      onHit(JSON.parse(msg.data) as Hit);
    } catch {
      // ignore malformed frame
    }
  };
  return () => es.close();
}

// ---------------------------------------------------------------------
// explain
// ---------------------------------------------------------------------

export interface ExplainResponse {
  text: string;
  sentExcerpt: string[];
}

// explain, when `lines` is omitted (undefined), lets the server auto-select
// the target's recent error/critical log lines. Pass an explicit array
// (even []) to control exactly what gets sent — logs.ts uses this so the
// consent modal can show the operator the exact excerpt before it goes out.
export function explain(id: string, lines?: string[]): Promise<ExplainResponse> {
  const body = lines === undefined ? {} : { lines };
  return request<ExplainResponse>(`/api/targets/${encodeURIComponent(id)}/explain`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(body),
  });
}

// ---------------------------------------------------------------------
// service control / clear / disk usage / endpoints / firewall
// (internal/ops — day-2 operator actions on a wired target)
// ---------------------------------------------------------------------

export type ServiceID = "exec" | "beacon";
export type ServiceActionKind = "start" | "stop" | "restart";

// serviceAction's response mirrors serviceActionResponse in api.go, which
// deliberately carries no json tag and so encodes as PascalCase {"Active":...}.
export interface ServiceActionResult {
  Active: boolean;
}

export function serviceAction(
  id: string,
  svc: ServiceID,
  action: ServiceActionKind,
): Promise<ServiceActionResult> {
  return request<ServiceActionResult>(
    `/api/targets/${encodeURIComponent(id)}/services/${svc}/${action}`,
    { method: "POST" },
  );
}

// clearService always confirms with the service id itself — the UI's own
// modal is the "type the service name" confirmation gate; this call is only
// reachable after that gate has passed.
export function clearService(id: string, svc: ServiceID): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/targets/${encodeURIComponent(id)}/services/${svc}/clear`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ Confirm: svc }),
  });
}

// DiskUsage mirrors ops.DU, another untagged struct — PascalCase fields.
export interface DiskUsage {
  ExecBytes: number;
  BeaconBytes: number;
  DiskFreeBytes: number;
  ExpectedExecBytes: number;
  ExpectedBeaconBytes: number;
  SyncLabel: string;
  GenesisSyncLabel: string;
}

export function getDiskUsage(id: string): Promise<DiskUsage> {
  return request<DiskUsage>(`/api/targets/${encodeURIComponent(id)}/du`);
}

// EndpointInfo mirrors ops.EndpointInfo.
export interface EndpointInfo {
  ExecHTTP: string;
  BeaconHTTP: string;
  ExecReachable: boolean;
  BeaconReachable: boolean;
  ChainIDMatches: boolean;
  Access: "local" | "ssh";
  TunnelHint: string;
}

export function getEndpoints(id: string): Promise<EndpointInfo> {
  return request<EndpointInfo>(`/api/targets/${encodeURIComponent(id)}/endpoints`);
}

// CheckItem mirrors ops.CheckItem.
export interface CheckItem {
  ID: string;
  Title: string;
  Why: string;
  Status: "pass" | "fail" | "warn" | "unknown";
  Detail: string;
  Fix: string;
}

export function getFirewallChecklist(id: string): Promise<CheckItem[]> {
  return request<CheckItem[]>(`/api/targets/${encodeURIComponent(id)}/firewall`);
}

// DiagReport mirrors server.DiagReport: one diagnostics-ladder run — items
// in order, stopping at the first failure — plus when it ran and what
// triggered it ("manual", "journal: <signature>", "monitor: <condition>").
export interface DiagReport {
  at: string;
  trigger: string;
  items: CheckItem[];
  failedId?: string;
}

export function runNetworkDiagnostics(id: string): Promise<DiagReport> {
  return request<DiagReport>(`/api/targets/${encodeURIComponent(id)}/diagnostics`);
}

export function getLatestDiagnostics(id: string): Promise<DiagReport | null> {
  return request<DiagReport | null>(`/api/targets/${encodeURIComponent(id)}/diagnostics/latest`);
}

// ---------------------------------------------------------------------
// container-backed services: the local devnet and the eRPC gateway
// (internal/server/containers.go)
//
// Two naming conventions meet in these types, and both are the real wire
// shape rather than a preference: this file's own response structs carry
// json tags and are lowerCamelCase, while the values lifted straight out of
// Go (ops.ContainerStatus, ops.WipeReport, catalog.DevnetConfig,
// catalog.GatewayConfig) carry no tags and so arrive PascalCase.
// ---------------------------------------------------------------------

// Only the devnet is addressed under a target now. The eRPC gateway is a
// fleet-wide LAYER with its own top-level surface (/api/gateways) — see the
// gateway section below.
export type ContainerServiceID = "devnet";
export type ContainerActionKind = "start" | "stop" | "restart";

// ContainerState mirrors ops' State* constants. "not-created" and
// "created-but-stopped" are deliberately distinct: one needs provisioning,
// the other is one start away.
export type ContainerState = "not-created" | "created-but-stopped" | "running" | "unknown";

export interface ContainerStatus {
  ID: string;
  ContainerName: string;
  State: ContainerState;
  // Image is what the container was actually CREATED from, which is not
  // necessarily what the app would build or run today — that difference is
  // how "this container predates your config change" becomes visible.
  Image: string;
  ImageID: string;
  ExitCode: number;
  // Platform is the image the container is ACTUALLY running; EnginePlatform is
  // what the engine runs natively. Emulated is true when they differ, meaning
  // every instruction is being translated by QEMU.
  //
  // It matters because an emulated container reports State "running" like any
  // other: in this app's own testing an emulated node reported running and
  // answered nothing, so a screen showing State without Emulated is showing a
  // healthy service that does not work.
  Platform: string;
  EnginePlatform: string;
  Emulated: boolean;
  Detail: string;
}

export interface DevnetConfig {
  ChainID: number;
  BlockTime: string;
  ImageRef: string;
  ContainerName: string;
  BindAddr: string;
  HTTPPort: number;
  WSPort: number;
  Platform: string;
}

export interface ServiceEndpoint {
  label: string;
  url: string;
}

// ContainerView is one service's card. Every list-shaped field can arrive as
// null (a nil Go slice), so callers must normalize with `?? []` rather than
// trusting the array type.
export interface ContainerView {
  id: ContainerServiceID;
  label: string;
  containerName: string;
  configured: boolean;
  status: ContainerStatus;
  endpoints: ServiceEndpoint[] | null;
  // actions are the ones this state permits, decided server-side so the UI
  // can never offer one that ops would reject.
  actions: string[] | null;
  blocked?: string;
  wipeDiscards: string;
  restartsOnWipe: string[] | null;
  warnings?: string[] | null;
  devnet?: DevnetConfig;
  error?: string;
  hint?: string;
  code?: string;
}

export interface DockerView {
  present: boolean;
  reachable: boolean;
  flavor: string;
  serverVersion?: string;
  detail?: string;
  hint?: string;
}

export interface ContainersResponse {
  docker: DockerView;
  services: ContainerView[];
}

export function getContainers(id: string): Promise<ContainersResponse> {
  return request<ContainersResponse>(`/api/targets/${encodeURIComponent(id)}/containers`);
}

export function containerAction(
  id: string,
  svc: ContainerServiceID,
  action: ContainerActionKind,
): Promise<{ status: ContainerStatus }> {
  return request<{ status: ContainerStatus }>(
    `/api/targets/${encodeURIComponent(id)}/containers/${svc}/${action}`,
    { method: "POST" },
  );
}

// WipeReport mirrors ops.WipeReport — an untagged struct, so PascalCase.
// Cascaded is the whole reason a wipe is not just a delete: it lists the
// services restarted because this one's data was thrown away.
export interface WipeReport {
  ID: string;
  ContainerName: string;
  ContainerRemoved: boolean;
  VolumesRemoved: string[] | null;
  VolumesAbsent: string[] | null;
  Recreated: boolean;
  Cascaded: string[] | null;
  CascadeSkipped: string[] | null;
}

export interface WipeResult {
  report: WipeReport;
  status: ContainerStatus;
  // error/hint are set on a PARTIAL failure — most importantly a cascade
  // that could not restart a service in front of the wiped one, which means
  // the wipe DID happen and something is now serving a stale head.
  error?: string;
  hint?: string;
  code?: string;
}

// wipeContainer resolves with the report even when the server answers with an
// error status, because a failed cascade still destroyed and rebuilt the
// service — reporting only "wipe failed" there would describe a wipe that
// succeeded. It throws only when there is no report to show.
export async function wipeContainer(id: string, svc: ContainerServiceID): Promise<WipeResult> {
  const res = await fetch(`/api/targets/${encodeURIComponent(id)}/containers/${svc}/wipe`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ Confirm: svc }),
  });
  const text = await res.text();
  let body: unknown = null;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    // Not JSON — fall through to the throw below.
  }
  if (body && typeof body === "object" && "report" in body) {
    return body as WipeResult;
  }
  const message =
    body && typeof body === "object" && typeof (body as { error?: unknown }).error === "string"
      ? (body as { error: string }).error
      : res.statusText || `HTTP ${res.status}`;
  throw new ApiError(res.status, message);
}

// provisionContainer creates (or re-creates) a service by running its setup
// plan. It returns as soon as the run is accepted — progress arrives on the
// target's existing setup stream (streamSetup), which is shared with the node
// wizard, so only one provisioning run per target can be in flight.
export function provisionContainer(id: string, svc: ContainerServiceID): Promise<{ status: string }> {
  return request<{ status: string }>(
    `/api/targets/${encodeURIComponent(id)}/containers/${svc}/provision`,
    { method: "POST" },
  );
}

// resetDevnet throws a devnet's chain away and brings it back from genesis.
//
// It is the same operation as wipeContainer — ops.WipeService, so the fronting
// gateways are restarted and the report says which — with no typed
// confirmation, because a devnet is a scratch chain and resetting it is
// routine. It resolves with the report even on a partial failure, for the same
// reason wipeContainer does: a cascade that failed still reset the chain.
export async function resetDevnet(id: string): Promise<WipeResult> {
  const res = await fetch(`/api/targets/${encodeURIComponent(id)}/containers/devnet/reset`, {
    method: "POST",
    headers: JSON_HEADERS,
  });
  const text = await res.text();
  let body: unknown = null;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    // Not JSON — fall through to the throw below.
  }
  if (body && typeof body === "object" && "report" in body) {
    return body as WipeResult;
  }
  const message =
    body && typeof body === "object" && typeof (body as { error?: unknown }).error === "string"
      ? (body as { error: string }).error
      : res.statusText || `HTTP ${res.status}`;
  throw new ApiError(res.status, message);
}

export interface ContainerConfig {
  id: ContainerServiceID;
  configured: boolean;
  devnet?: DevnetConfig;
}

export function getContainerConfig(id: string, svc: ContainerServiceID): Promise<ContainerConfig> {
  return request<ContainerConfig>(
    `/api/targets/${encodeURIComponent(id)}/containers/${svc}/config`,
  );
}

// putContainerConfig stores the DESIRED configuration. It never touches a
// running container: ports, mounts and the command line are all fixed when a
// container is created, so applying a change is provisionContainer's job.
export function putContainerConfig(
  id: string,
  svc: ContainerServiceID,
  config: DevnetConfig | GatewayConfig,
): Promise<ContainerConfig> {
  return request<ContainerConfig>(
    `/api/targets/${encodeURIComponent(id)}/containers/${svc}/config`,
    { method: "PUT", headers: JSON_HEADERS, body: JSON.stringify(config) },
  );
}

// ---------------------------------------------------------------------
// gateways: eRPC as a LAYER over the fleet (internal/server/gateways.go)
//
// A gateway is not a machine's service. It fronts N chains across M
// endpoints, and those endpoints can be a devnet on this laptop, a node on a
// fleet box, or a public endpoint — so it NAMES the machine it runs on
// (placement) rather than belonging to it, and there can be several.
//
// The naming convention split from the container routes carries over: this
// file's own response structs are tagged and lowerCamelCase, while values
// lifted straight out of Go (catalog.GatewayConfig, ops.ContainerStatus)
// carry no tags and arrive PascalCase.
// ---------------------------------------------------------------------

// UpstreamKind is what an endpoint IS. The two managed kinds store a
// reference (kind + machine) rather than a URL, and the server derives the
// URL on every read — which is what stops a gateway pointing at a node's old
// address after someone changed it on the node's own screen.
export type UpstreamKind = "managed-node" | "managed-devnet" | "external";

export interface GatewayUpstream {
  ID: string;
  Kind?: UpstreamKind;
  // TargetID is the machine a managed upstream refers to.
  TargetID?: string;
  // Endpoint is stored only for "external"; for managed kinds it is derived
  // and whatever is stored here is ignored.
  Endpoint: string;
  Local: boolean;
  RecentOnly: boolean;
}

export interface GatewayNetwork {
  ChainID: number;
  Upstreams: GatewayUpstream[];
}

// GatewayTLS is the HTTPS front's stored settings. Null means no front.
export interface GatewayTLS {
  Enabled: boolean;
  Hostname: string;
  // CertSource: "internal" uses Caddy's own CA (no domain, no network, one
  // trust-store install); "files" uses a certificate already on disk.
  CertSource: string;
  CertFile: string;
  KeyFile: string;
  HTTPSPort: number;
  // BindAddr defaults to 0.0.0.0, unlike every other bind here: a TLS front on
  // loopback serves only the machine that never needed TLS.
  BindAddr: string;
  ImageRef: string;
}

export interface GatewayConfig {
  ProjectID: string;
  BindAddr: string;
  Port: number;
  // Networks is null (not []) for a gateway with nothing configured — an
  // untagged Go nil slice.
  Networks: GatewayNetwork[] | null;
  // MetricsOff is the negative on purpose, mirroring the Go field: absent or
  // false means the gateway IS counting its requests, which is the default and
  // what every configuration written before this existed means.
  MetricsOff?: boolean;
  MetricsPort?: number;
  TLS?: GatewayTLS | null;
}

export interface GatewayPlacement {
  targetId: string;
  backend: string; // "docker" | "systemd"
}

// UpstreamView is one endpoint row. endpoint is RESOLVED (what eRPC will
// dial); label says what it is in the operator's terms; problem says why it
// cannot be used, on the row, before you click anything.
export interface UpstreamView {
  id: string;
  kind: UpstreamKind;
  targetId?: string;
  endpoint: string;
  label: string;
  local: boolean;
  recentOnly: boolean;
  problem?: string;
  // actions is keyed off the KIND: only a devnet is offered "reset".
  actions: string[] | null;
}

export interface NetworkView {
  chainId: number;
  name: string;
  url?: string;
  path: string;
  upstreams: UpstreamView[] | null;
  // knownSetSize is how many upstreams "Add valve's set…" would put on this
  // chain — the ENTRY count, which is what the redundancy bar's denominator
  // has to be: its numerator counts configured upstreams, so denominating it
  // in providers instead would make the page's own primary action overshoot
  // the target every time. 0 means valve has measured no set for this chain,
  // and the bar must then show no denominator rather than a target of zero.
  knownSetSize: number;
  // serviceable is false when nothing on this chain can be dialed — eRPC
  // would accept the config and fail every call on this path.
  serviceable: boolean;
  warnings?: string[] | null;
}

// TlsView reports the EFFECTIVE certificate source beside the configured one.
// They differ exactly when a certificate on disk was unusable and the app fell
// back to its own CA rather than serving nothing — which is a thing the
// operator has to be told, not a thing to discover in a browser warning.
export interface TlsView {
  enabled: boolean;
  hostname?: string;
  url?: string;
  certSource?: string;
  effectiveCertSource?: string;
  fallback?: string;
  fallbackReason?: string;
  containerName?: string;
  status: ContainerStatus;
  // rootCaPath is the file to install in a trust store to stop the browser
  // warning. Naming it is the difference between a solvable warning and a
  // mystery.
  rootCaPath?: string;
  // suggestedHostname is a name under a domain whose wildcard already resolves
  // to loopback, so HTTPS can be turned on without owning a domain first.
  suggestedHostname?: string;
  // verification is the LAST live check (verifyGatewayTls), not a fresh one:
  // it opens real connections and waits for a block, so it is never run on a
  // poll. `at` says how old it is.
  verification?: TlsVerification | null;
  error?: string;
}

// TlsAssertionStatus: "unavailable" is not a fourth spelling of "fail". A
// gateway whose upstream is http:// serves HTTPS perfectly and cannot serve
// subscriptions — a missing capability, reported as missing.
export type TlsAssertionStatus = "pass" | "fail" | "skip" | "unavailable";

export interface TlsAssertion {
  id: string;
  title: string;
  // why says what this assertion catches that the others do not.
  why: string;
  status: TlsAssertionStatus;
  detail: string;
}

// TlsVerification is one live "is HTTPS actually serving?" run: a handshake,
// the name on the certificate, the chain against the expected root, an RPC
// call through it, and a subscription over wss.
export interface TlsVerification {
  at: string;
  url: string;
  hostname: string;
  // address is what was actually dialed — the name is pinned to it, so DNS is
  // not what is under test.
  address: string;
  chainId?: number;
  path?: string;
  certSource?: string;
  trustSource?: string;
  subject?: string;
  issuer?: string;
  notBefore?: string;
  notAfter?: string;
  expiresIn?: string;
  // expiryWarning is set only when the expiry is something to act on — an
  // internal-CA leaf lives 12 hours by design and is renewed in process.
  expiryWarning?: string;
  assertions: TlsAssertion[] | null;
  ok: boolean;
  subscriptionsOk: boolean;
  summary: string;
}

export interface GatewayView {
  id: string;
  label: string;
  containerName: string;
  placement: GatewayPlacement;
  status: ContainerStatus;
  // docker is the engine reading for THIS gateway's machine — two gateways
  // can sit on two different boxes.
  docker: DockerView;
  baseUrl: string;
  tls: TlsView;
  networks: NetworkView[] | null;
  actions: string[] | null;
  blocked?: string;
  wipeDiscards: string;
  warnings?: string[] | null;
  // config is the STORED configuration, references intact. Editors must
  // round-trip this, never the resolved endpoints.
  config: GatewayConfig;
  error?: string;
  hint?: string;
  code?: string;
}

// UpstreamSource is a real thing in the fleet a new upstream can point at.
export interface UpstreamSource {
  kind: UpstreamKind;
  targetId: string;
  chainId: number;
  label: string;
  endpoint: string;
}

// NetworkPreset is one option in the add-a-chain picker, sourced from the
// catalog so it cannot drift from what the app supports.
export interface NetworkPreset {
  chainId: number;
  name: string;
  // devnet marks the one preset that can provision its own upstream.
  devnet: boolean;
}

export interface TargetSummary {
  id: string;
  mode: string;
  hasDevnet: boolean;
  hasNode: boolean;
}

// OrphanedContainer is a container a merge stopped managing but did NOT
// stop — see config.Config.Orphans. It keeps serving stale config with
// nothing pointing at it until the operator wipes it themselves and
// dismisses the record.
export interface OrphanedContainer {
  containerName: string;
  targetId: string;
  mergedInto: string;
}

export interface GatewaysResponse {
  gateways: GatewayView[] | null;
  targets: TargetSummary[] | null;
  sources: UpstreamSource[] | null;
  presets: NetworkPreset[] | null;
  orphans?: OrphanedContainer[];
}

export function getGateways(): Promise<GatewaysResponse> {
  return request<GatewaysResponse>("/api/gateways");
}

// dismissOrphan forgets a leftover-container record only. It never touches
// the container: this app never stops a container it did not just start.
export async function dismissOrphan(name: string): Promise<void> {
  await request(`/api/orphans/${encodeURIComponent(name)}`, { method: "DELETE" });
}

export interface CreateGatewayRequest {
  id: string;
  placement: GatewayPlacement;
  config?: GatewayConfig;
}

export function createGateway(body: CreateGatewayRequest): Promise<GatewayView> {
  return request<GatewayView>("/api/gateways", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(body),
  });
}

// verifyGatewayTls runs the live HTTPS check against a gateway's front and
// returns the evidence. It is a GET because it writes nothing, and it is slow
// by design — it waits for a real block on a real subscription.
export function verifyGatewayTls(gid: string): Promise<TlsVerification> {
  return request<TlsVerification>(`/api/gateways/${encodeURIComponent(gid)}/tls/verify`);
}

// ---- traffic: who is actually carrying the load -------------------------

// UpstreamShare is one endpoint's measured share of a chain's answered
// requests, against what the routing configuration intended.
//
// Both the fraction and the raw count are here on purpose: 100% of four
// requests and 100% of four million draw the same bar and are very different
// facts, so the UI can qualify one and not the other.
export interface UpstreamShare {
  upstream: string;
  succeeded: number;
  // actual and intended are fractions in 0..1.
  actual: number;
  intended: number;
  diverged: boolean;
  // unconfigured marks an endpoint eRPC is still counting that this gateway's
  // saved configuration no longer lists — the state between editing a config
  // and re-creating the container, which is exactly when someone needs telling
  // that their change has not been applied yet.
  unconfigured?: boolean;
}

export interface NetworkTraffic {
  chainId: number;
  // received is what clients asked for; attributed is what some endpoint
  // answered. The gap is failure, which is what keeps a chain failing every
  // call from looking identical to one nobody has called.
  received: number;
  attributed: number;
  // unattributed is what the gateway answered from its own cache, without
  // calling any endpoint. Kept apart from attributed so the received gap
  // means failures and nothing else.
  unattributed: number;
  upstreams: UpstreamShare[] | null;
}

export interface GatewayTraffic {
  // enabled is false when the operator turned the counters off. It has to be
  // distinguishable from "no traffic yet", which looks identical in the
  // numbers and is fixed by a completely different action.
  enabled: boolean;
  at: string;
  // since is when the gateway process started. These counters are cumulative
  // from then, and a share with no window on it invites being read as "now".
  since: string;
  networks: NetworkTraffic[] | null;
  error?: string;
}

// getGatewayTraffic reads one gateway's own request counters. It is a separate
// call from getGateways deliberately: it runs a command on the gateway's
// machine, and folding that into the list would put a per-gateway way to hang
// behind the screen an operator opens precisely when a gateway is misbehaving.
export function getGatewayTraffic(gid: string): Promise<GatewayTraffic> {
  return request<GatewayTraffic>(`/api/gateways/${encodeURIComponent(gid)}/traffic`);
}

// ---- analytics: how is it doing, and why --------------------------------
//
// The share bars answer detection. These answer the question you ask next,
// and they come in two halves that must never be averaged together: what
// CLIENTS experienced (networks) and what the GATEWAY sees of its endpoints
// (endpoints). Only the first is client traffic — every number in the second
// counts eRPC's own state poller too, because eRPC publishes no label that
// separates them.

// Bucket is one cumulative histogram bucket: every request that finished
// within `le` seconds. le is a string because the last one is "+Inf", which
// JSON cannot carry as a number.
export interface Bucket {
  le: string;
  count: number;
}

export interface Latency {
  count: number;
  // mean is null when nothing has been counted. 0 would be a claim about
  // speed; "nobody has called this" is not one.
  mean: number | null;
  buckets: Bucket[] | null;
}

export interface MethodLatency extends Latency {
  method: string;
}

export interface EndpointLatency extends Latency {
  upstream: string;
}

export interface NetworkAnalytics {
  chainId: number;
  name: string;
  // received is what clients asked for; answered is what endpoints returned;
  // unattributed is what the gateway answered from its own cache. failed is
  // the remainder, computed server-side so there is one definition of it.
  received: number;
  answered: number;
  unattributed: number;
  failed: number;
  methods: MethodLatency[] | null;
  endpoints: EndpointLatency[] | null;
  // cached is how long the requests answered from the gateway's own cache
  // took. It is not an endpoint row because a cache hit called no endpoint —
  // rendering it as one would put a server on the screen that does not exist.
  cached: Latency;
  // failedLatency is how long the failed requests took. Failing fast and
  // timing out after thirty seconds are different problems, and the failed
  // count alone cannot tell them apart.
  failedLatency: Latency;
}

export interface ErrorClass {
  class: string;
  severity: string;
  method: string;
  count: number;
}

// EndpointHealth is the gateway's own view of one endpoint. `requests`
// includes the state poller and is usually mostly the state poller — it is
// what the gateway asked of this endpoint, not what your clients did.
export interface EndpointHealth {
  upstream: string;
  chainId: number;
  configured: boolean;
  requests: number;
  errors: ErrorClass[] | null;
  // scored is false when eRPC has never formed an opinion about this
  // endpoint. Without it, position 0 ("preferred") is indistinguishable from
  // an unset number, and an endpoint nothing can reach reads as the chosen
  // one.
  scored: boolean;
  score: number;
  position: number;
  primarySwitches: number;
  excludedSeconds: number;
  headLag: number;
  finalizationLag: number;
  latestBlock: number;
}

export interface GatewayAnalytics {
  enabled: boolean;
  at: string;
  since: string;
  networks: NetworkAnalytics[] | null;
  endpoints: EndpointHealth[] | null;
  error?: string;
}

// getGatewayAnalytics reads one gateway's counters and returns both folds of
// the same scrape. One request, one curl on the gateway's machine.
export function getGatewayAnalytics(gid: string): Promise<GatewayAnalytics> {
  return request<GatewayAnalytics>(`/api/gateways/${encodeURIComponent(gid)}/analytics`);
}

// ---- capabilities: what an endpoint can actually DO ----------------------

export type CapabilityStatus = "supported" | "unsupported" | "inconclusive" | "inconsistent";

// Capability is one verdict for one endpoint. detail carries the evidence in
// words, so an operator can argue with the verdict — which is the point of
// inferring capability from behaviour rather than from a provider's claims.
export interface Capability {
  key: string;
  label: string;
  status: CapabilityStatus;
  detail?: string;
  method?: string;
}

export interface EndpointCapabilities {
  // upstream joins this row to the traffic row of the same id.
  upstream: string;
  chainId: number;
  probedUrl?: string;
  reachable: boolean;
  reachDetail?: string;
  // unprobeable says why this endpoint could not be probed FROM HERE — an
  // address that only resolves inside a docker network, or a loopback bind on
  // a machine reached over SSH. A stated reason is a different thing from a
  // blank cell.
  unprobeable?: string;
  capabilities: Capability[] | null;
}

export interface GatewayCapabilities {
  at: string;
  endpoints: EndpointCapabilities[] | null;
}

// getGatewayCapabilities returns the cached probe results, re-probing only
// when they have aged out or refresh is asked for. Probing opens real sockets
// against real endpoints, so it must never ride a screen's poll cadence.
export function getGatewayCapabilities(gid: string, refresh = false): Promise<GatewayCapabilities> {
  const q = refresh ? "?refresh=1" : "";
  return request<GatewayCapabilities>(`/api/gateways/${encodeURIComponent(gid)}/capabilities${q}`);
}

export function deleteGateway(gid: string): Promise<{ status: string; note: string }> {
  return request<{ status: string; note: string }>(`/api/gateways/${encodeURIComponent(gid)}`, {
    method: "DELETE",
  });
}

// putGatewayConfig stores the DESIRED configuration. It never touches a
// running container — a container's published port and mounts are fixed at
// creation — so applying a change is provisionGateway's job.
export function putGatewayConfig(gid: string, config: GatewayConfig): Promise<GatewayView> {
  return request<GatewayView>(`/api/gateways/${encodeURIComponent(gid)}/config`, {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify(config),
  });
}

export function gatewayAction(
  gid: string,
  action: ContainerActionKind,
): Promise<{ status: ContainerStatus }> {
  return request<{ status: ContainerStatus }>(
    `/api/gateways/${encodeURIComponent(gid)}/${action}`,
    { method: "POST" },
  );
}

// TrustCertResult is the outcome of installing a gateway's internal-CA root
// into the trust store of the machine it runs on. ok is true only when the
// install actually ran and succeeded; when it is false, ranCommand (if present)
// is the exact command to run by hand — the only option when the device that
// opens the URL is not the machine the gateway runs on.
export interface TrustCertResult {
  ok: boolean;
  ranCommand?: string;
  message: string;
}

// trustGatewayCert installs THIS gateway's own exported internal-CA root into
// the trust store of the machine it runs on. It only ever installs that one
// derived path — never an arbitrary file — and only when the gateway is served
// by Caddy's own authority; the server refuses otherwise.
export function trustGatewayCert(gid: string): Promise<TrustCertResult> {
  return request<TrustCertResult>(`/api/gateways/${encodeURIComponent(gid)}/trust-cert`, {
    method: "POST",
  });
}

// provisionGateway returns as soon as the run is accepted. Progress arrives
// on the PLACEMENT machine's setup stream — the same one the node wizard and
// the devnet use — and the response says which machine that is, so the caller
// does not have to know the placement rule.
export function provisionGateway(gid: string): Promise<{ status: string; targetId: string }> {
  return request<{ status: string; targetId: string }>(
    `/api/gateways/${encodeURIComponent(gid)}/provision`,
    { method: "POST" },
  );
}

// wipeGateway resolves with the report even on an error status, for the same
// reason wipeContainer does.
export async function wipeGateway(gid: string): Promise<WipeResult> {
  const res = await fetch(`/api/gateways/${encodeURIComponent(gid)}/wipe`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ Confirm: gid }),
  });
  const text = await res.text();
  let body: unknown = null;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    // Not JSON — fall through to the throw below.
  }
  if (body && typeof body === "object" && "report" in body) {
    return body as WipeResult;
  }
  const message =
    body && typeof body === "object" && typeof (body as { error?: unknown }).error === "string"
      ? (body as { error: string }).error
      : res.statusText || `HTTP ${res.status}`;
  throw new ApiError(res.status, message);
}

// ---------------------------------------------------------------------
// chainlist: public endpoint discovery (internal/chainlist)
// ---------------------------------------------------------------------

export interface ChainlistEndpoint {
  url: string;
  kind: "http" | "ws";
  status: "live" | "unprobed" | "rejected";
  chainId?: number;
  latencyMs?: number;
  reason?: string;
}

export interface ChainlistResult {
  chainId: number;
  // source is "vendored" when the live feed was unreachable and the built-in
  // snapshot stood in — not a failure, but the operator should know.
  source: "feed" | "vendored";
  fetchError?: string;
  endpoints: ChainlistEndpoint[] | null;
  live: number;
}

// discoverEndpoints fetches chainid.network, drops the ${API_KEY} provider
// slots, and probes what is left with eth_chainId — so what comes back is
// endpoints that are answering for this chain RIGHT NOW, not what a feed
// claims. Rejected ones come back too, each with its reason.
export function discoverEndpoints(chainId: number): Promise<ChainlistResult> {
  return request<ChainlistResult>(`/api/chainlist/${chainId}`);
}

// ---------------------------------------------------------------------
// known set: valve's own vetted, measured, ordered endpoints per chain
// (internal/server/knownset.go)
// ---------------------------------------------------------------------

export interface KnownSetEndpoint {
  url: string;
  provider: string;
  websocket: boolean;
  archive: boolean;
  // alreadyAdded is true when this URL is already an upstream on this
  // gateway's chain — the row is still shown (the count offered has to be
  // the count that lands), just not offered again.
  alreadyAdded: boolean;
}

// KnownSetResponse.endpoints is JSON null (not []) for a chain valve has not
// measured — callers must coalesce with `?? []` rather than assume an array.
export interface KnownSetResponse {
  endpoints: KnownSetEndpoint[] | null;
  // usingDefaultKey says the set resolved with the key that ships with the app
  // rather than one the operator stored. The key itself is deliberately NOT
  // here: the UI needs to know which key is in play, never what it is.
  usingDefaultKey: boolean;
}

// knownSet is the vetted, ordered set — measured capability, not a live
// feed — offered before the chainlist probe because it is the vetted
// default and the feed is the escape hatch.
export function knownSet(gid: string, chainId: number): Promise<KnownSetResponse> {
  return request<KnownSetResponse>(`/api/gateways/${encodeURIComponent(gid)}/knownset/${chainId}`);
}

// ---------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------

export type AIProvider = "" | "gemini" | "groq" | "ollama";

export interface Settings {
  aiProvider: AIProvider;
  aiKeySet: boolean;
  refRpcBase: string;
  // providerKeysSet NAMES the ${...} placeholders that have a key stored —
  // never the values, same rule as aiKeySet. The server promises a non-nil
  // array; callers still coalesce, because an older binary behind a newer
  // bundle would send nothing at all.
  providerKeysSet: string[];
}

export function getSettings(): Promise<Settings> {
  return request<Settings>("/api/settings");
}

export interface PutSettingsRequest {
  aiProvider?: AIProvider;
  // aiKey omitted => leave the stored key unchanged. "" => explicitly clear
  // it. Never send a field the user hasn't touched.
  aiKey?: string;
  refRpcBase?: string;
  // providerKeys is a PATCH by placeholder name, not a replacement: a name
  // carrying a value stores it, a name carrying "" forgets it, and a name that
  // is absent is left alone. Send only what the operator touched — GET never
  // echoes the values back, so re-sending what was last read would wipe every
  // key. A name must match ^[A-Za-z0-9_]+$; the server rejects the WHOLE
  // request on one bad name, so the error has to reach the operator.
  providerKeys?: Record<string, string>;
}

export function putSettings(body: PutSettingsRequest): Promise<Settings> {
  return request<Settings>("/api/settings", {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify(body),
  });
}

// ---------------------------------------------------------------------
// fetch plumbing
// ---------------------------------------------------------------------

export class ApiError extends Error {
  status: number;
  // hint is the server's operator-facing guidance (e.g. ops' "start Docker
  // Desktop / OrbStack / colima" for an unreachable engine), and code is the
  // machine-readable kind ("docker-absent", "docker-unreachable",
  // "service-not-created", "not-configured"). Both are optional: only the
  // container routes populate them.
  hint?: string;
  code?: string;
  constructor(status: number, message: string, hint?: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.hint = hint;
    this.code = code;
  }
}

const JSON_HEADERS = { "Content-Type": "application/json" };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  if (!res.ok) {
    let message = res.statusText || `HTTP ${res.status}`;
    let hint: string | undefined;
    let code: string | undefined;
    try {
      const body = (await res.json()) as { error?: string; hint?: string; code?: string };
      if (body && typeof body.error === "string" && body.error) {
        message = body.error;
      }
      if (body && typeof body.hint === "string" && body.hint) hint = body.hint;
      if (body && typeof body.code === "string" && body.code) code = body.code;
    } catch {
      // body wasn't JSON (or was empty) — fall back to statusText.
    }
    throw new ApiError(res.status, message, hint, code);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  return text ? (JSON.parse(text) as T) : (undefined as T);
}
