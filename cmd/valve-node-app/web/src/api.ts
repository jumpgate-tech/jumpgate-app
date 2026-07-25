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
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

const JSON_HEADERS = { "Content-Type": "application/json" };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  if (!res.ok) {
    let message = res.statusText || `HTTP ${res.status}`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body && typeof body.error === "string" && body.error) {
        message = body.error;
      }
    } catch {
      // body wasn't JSON (or was empty) — fall back to statusText.
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  return text ? (JSON.parse(text) as T) : (undefined as T);
}
