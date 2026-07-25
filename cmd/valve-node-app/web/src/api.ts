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

export type ContainerServiceID = "devnet" | "erpc";
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

export interface GatewayUpstream {
  ID: string;
  Endpoint: string;
  Local: boolean;
  RecentOnly: boolean;
}

export interface GatewayNetwork {
  ChainID: number;
  Upstreams: GatewayUpstream[];
}

export interface GatewayConfig {
  ProjectID: string;
  BindAddr: string;
  Port: number;
  // Networks is null (not []) for a gateway with nothing configured — an
  // untagged Go nil slice.
  Networks: GatewayNetwork[] | null;
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
  gateway?: GatewayConfig;
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

export interface ContainerConfig {
  id: ContainerServiceID;
  configured: boolean;
  devnet?: DevnetConfig;
  gateway?: GatewayConfig;
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
// settings
// ---------------------------------------------------------------------

export type AIProvider = "" | "gemini" | "groq" | "ollama";

export interface Settings {
  aiProvider: AIProvider;
  aiKeySet: boolean;
  refRpcBase: string;
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
