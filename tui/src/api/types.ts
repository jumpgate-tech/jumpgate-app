// Hand-written mirrors of the internal/api structs the dashboard reads. Only
// the fields it uses are listed; test/routes.test.ts checks each against a
// fixture the Go side writes (internal/api/tsfixtures_test.go).

export interface ApiErrorBody {
  error: string;
  code: string;
  hint?: string;
}

export interface ProbeError extends ApiErrorBody {
  at: string;
}

export type SyncState = "synced" | "syncing" | "stopped" | "no_data" | "unavailable";

export interface ClientStatus {
  active: boolean;
  syncing: boolean;
  head: number;
  distance?: number;
  peers: number;
  state: SyncState;
}

export interface NodeStatus {
  at: string;
  exec: ClientStatus;
  beacon: ClientStatus;
  refHead?: number;
  headLag?: number;
  overall: SyncState;
  diskUsedPct?: number;
}

export interface DiskView {
  at: string;
  execBytes: number;
  beaconBytes: number;
  freeBytes: number;
  expectedExecBytes: number;
  expectedBeaconBytes: number;
  expectedLabel: string;
  fit: "ok" | "tight" | "short" | "unknown";
  daysToFull?: number;
}

export interface FirewallSummary {
  pass: number;
  warn: number;
  fail: number;
  unknown: number;
  grade: string;
}

export interface FleetRow {
  id: string;
  link: string;
  thisMachine: boolean;
  reachable: boolean;
  lastSeen?: string;
  chainId?: number;
  network?: string;
  /** The gate's version on the box. */
  agent?: string;
  status: NodeStatus | null;
  disk: DiskView | null;
  firewall: FirewallSummary | null;
  jobs: { running: number; needsDecision: number } | null;
  error: ApiErrorBody | null;
  statusError: ProbeError | null;
  diskError: ProbeError | null;
  firewallError?: ProbeError | null;
  stale: { status: boolean; disk: boolean; firewall: boolean };
  alertCount: number;
  alertMessage?: string;
}

export interface Fleet {
  at: string;
  intervalSeconds: number;
  rows: FleetRow[];
  locked?: boolean;
}

export interface VaultView {
  state: string;
  slots: string[];
  plain?: string[];
  reason?: string;
}

export interface JobStepView {
  id: string;
  title: string;
  state: string;
}

export interface JobOption {
  id: string;
  label: string;
  detail?: string;
}

export interface JobQuestion {
  id: string;
  prompt: string;
  options: JobOption[];
  tier: string;
  signable: boolean;
}

export interface JobView {
  target: string;
  id: string;
  kind: string;
  title: string;
  state: string;
  step: number;
  steps: JobStepView[];
  question?: JobQuestion;
  lease: { label?: string; fresh: boolean; mine: boolean };
  updatedAt: string;
  error?: string;
  driving: boolean;
}

export interface JobsInbox {
  at: string;
  jobs: JobView[];
  errors: { target: string; error: ApiErrorBody }[];
}

export interface JobDetail {
  job: JobView;
  log: { index: number; kind: string; at: string; text: string }[];
}

export interface JobDecision {
  decision: string;
  option: string;
}

export interface RunningJob {
  target: string;
  id: string;
  kind: string;
  state: string;
  step: number;
  total: number;
  needsDecision: boolean;
}

export interface RunningJobs {
  at: string;
  jobs: RunningJob[];
}

export interface JobOutputLine {
  unit: string;
  line: string;
  at: string;
}

export interface Settings {
  aiProvider: string;
  aiKeySet: boolean;
  vaultIdleLockMinutes: number;
}

export type DrawerPref = "auto" | "open" | "min";

export interface UIPrefs {
  fleetColumns: string[];
  hostTabs: string[];
  chains: number[];
  refreshSeconds: number;
  units: string;
  drawer: DrawerPref;
  [k: string]: unknown;
}

export interface ControllerView {
  recorded: string;
  address: string;
  store: string;
  state: string;
  reason?: string;
}

export interface TargetView {
  id: string;
  mode: string;
  ssh?: { Host: string; User: string; Port: number };
  agent: { address: string; transport: string; pairedAt: string } | null;
  link: string;
  thisMachine: boolean;
}

export interface Endpoints {
  ExecHTTP: string;
  BeaconHTTP: string;
  ExecReachable: boolean;
  BeaconReachable: boolean;
  ChainIDMatches: boolean;
  Access: string;
  TunnelHint: string;
}

export interface CheckItem {
  ID: string;
  Title: string;
  Status: string;
  Detail: string;
  Fix: string;
}

export interface LogHit {
  unit: string;
  line: string;
  at: string;
  severity: string;
}

export interface ServiceResult {
  Active: boolean;
}

export interface GatewaySummary {
  id: string;
  label: string;
  baseUrl: string;
  tier: string;
}

export interface ModulesView {
  modules: ModuleStatus[];
  gateStale: boolean;
  message: string;
}

export interface AgentCheck {
  agent: string;
  version: string;
  chainId: number;
  setUp: boolean;
}

export interface SSHCommand {
  argv: string[];
  display: string;
}

export interface SettingsUpdate {
  aiProvider?: string;
  aiKey?: string;
  vaultIdleLockMinutes?: number;
}

export interface SSHView {
  Host: string;
  User: string;
  Port: number;
  jump?: SSHView;
}

export interface HostKeyHop {
  hostPort: string;
  fingerprint?: string;
  keyType?: string;
  state: "confirmed" | "unknown" | "mismatch" | "unreachable";
  probeId?: string;
  error?: string;
}

export interface HostKeyProbe {
  hops: HostKeyHop[];
  allConfirmed: boolean;
}

export interface RecordedHostKeys {
  hostPort: string;
  keys: { fingerprint: string; keyType: string; store: "jumpgate" | "openssh" }[];
}

export interface PairEvent {
  step?: string;
  line?: string;
  error?: string;
  code?: string;
  hint?: string;
  done?: boolean;
  /** The paired gate's address. */
  agent?: string;
}

export interface PairingInfo {
  controller: string;
  transportKey: string;
  system: boolean;
  scope?: string;
}

export interface GateUpgrade {
  upgraded: boolean;
  from: string;
  to: string;
}

export interface CatalogModule {
  id: string;
  name: string;
  summary: string;
  placement: string;
  applies: { chainId: number; client: string; ok: boolean }[];
}

export interface ModuleStatus {
  id: string;
  instance?: string;
  served: boolean;
  active: boolean;
  accepting: boolean;
  count: number;
  note?: string;
  error?: string;
}

export interface SetModule {
  enabled: boolean;
  public?: string;
  chainId?: number;
  port?: number;
  extIp?: string;
  upstreams?: string[];
}

export interface SetModuleResult {
  status: string;
  gateways?: string[];
  failed?: { id: string; error: string }[];
}

export interface SetupEvent {
  stepId: string;
  line?: string;
  done?: boolean;
  err?: string;
}

export interface VPNView {
  id: string;
  provider: string;
  interface: string;
  targetId: string;
  autostart: boolean;
  configured: boolean;
}

export interface VPNServerView {
  id: string;
  targetId: string;
  interface: string;
  address: string;
  listenPort: number;
  publicKey: string;
  endpoint: string;
  peers: { name: string; publicKey: string; allowedIp: string }[];
}
