// One sample per fixture, typed as the TS interface, so a field rename on
// either side shows up in routes.test.ts.
import type * as T from "../src/api/types.ts";

const at = "2026-10-28T12:00:00Z";
const err: T.ApiErrorBody = { error: "x", code: "x", hint: "x" };
const client: T.ClientStatus = { active: true, syncing: false, head: 1, distance: 1, peers: 1, state: "synced" };
const nodestatus: T.NodeStatus = { at, exec: client, beacon: client, refHead: 1, headLag: 1, overall: "synced", diskUsedPct: 1 };
const diskview: T.DiskView = {
  at, execBytes: 1, beaconBytes: 1, freeBytes: 1, expectedExecBytes: 1, expectedBeaconBytes: 1, expectedLabel: "x", fit: "ok", daysToFull: 1,
};
const job: T.JobView = {
  target: "a", id: "j", kind: "k", title: "t", state: "s", step: 0,
  steps: [{ id: "s", title: "t", state: "done" }],
  question: { id: "q", prompt: "p", options: [{ id: "o", label: "l", detail: "d" }], tier: "t", signable: true },
  lease: { label: "l", fresh: true, mine: true }, updatedAt: at, driving: true,
};

export const samples: Record<string, unknown> = {
  fleet: {
    at, intervalSeconds: 1, locked: true,
    rows: [{
      id: "a", link: "agent", thisMachine: true, reachable: true, lastSeen: at, chainId: 1, network: "n", agent: "v",
      status: nodestatus, disk: diskview, firewall: { pass: 1, warn: 1, fail: 1, unknown: 1, grade: "A" },
      jobs: { running: 1, needsDecision: 1 }, error: err, statusError: { ...err, at }, diskError: { ...err, at }, firewallError: { ...err, at },
      stale: { status: true, disk: true, firewall: true }, alertCount: 1, alertMessage: "m",
    }],
  } satisfies T.Fleet,
  nodestatus,
  diskview,
  vaultview: { state: "locked", slots: ["x"], plain: ["x"], reason: "x" } satisfies T.VaultView,
  jobview: job,
  jobsinbox: { at, jobs: [job], errors: [{ target: "a", error: { error: "x", code: "x" } }] } satisfies T.JobsInbox,
  jobdetail: { job, log: [{ index: 0, kind: "k", at, text: "t" }] } satisfies T.JobDetail,
  jobdecision: { decision: "approve", option: "o" } satisfies T.JobDecision,
  runningjobs: { at, jobs: [{ target: "a", id: "j", kind: "k", state: "s", step: 1, total: 1, needsDecision: true }] } satisfies T.RunningJobs,
  joboutput: [{ unit: "u", line: "l", at }] satisfies T.JobOutputLine[],
  settings: { aiProvider: "x", aiKeySet: true, vaultIdleLockMinutes: 1 } satisfies T.Settings,
  uiprefs: { fleetColumns: ["x"], hostTabs: ["x"], chains: [1], refreshSeconds: 1, units: "GB", drawer: "auto" } satisfies T.UIPrefs,
  controllerview: { recorded: "x", address: "x", store: "x", state: "x" } satisfies T.ControllerView,
  targets: [{
    id: "a", mode: "ssh", ssh: { Host: "h", User: "u", Port: 22 },
    agent: { address: "x", transport: "ssh", pairedAt: at }, link: "agent", thisMachine: false,
  }] satisfies T.TargetView[],
  endpoints: {
    ExecHTTP: "x", BeaconHTTP: "x", ExecReachable: true, BeaconReachable: true, ChainIDMatches: true, Access: "ssh", TunnelHint: "x",
  } satisfies T.Endpoints,
  firewall: [{ ID: "x", Title: "x", Status: "pass", Detail: "x", Fix: "x" }] satisfies T.CheckItem[],
  loghits: [{ unit: "u", line: "l", at, severity: "warn" }] satisfies T.LogHit[],
  serviceresult: { Active: true } satisfies T.ServiceResult,
  gateways: [{ id: "g", label: "l", baseUrl: "u", tier: "t" }] satisfies T.GatewaySummary[],
  modulesview: {
    modules: [{ id: "x", instance: "1", served: true, active: true, accepting: true, count: 1, note: "n", error: "e" }], gateStale: false, message: "",
  } satisfies T.ModulesView,
  catalog: { modules: [{ id: "w", name: "n", summary: "s", placement: "box", applies: [{ chainId: 1, client: "c", ok: true }] }] } satisfies { modules: T.CatalogModule[] },
  setmodule: { enabled: true, public: "read", chainId: 1, port: 1, extIp: "x", upstreams: ["a=b"] } satisfies T.SetModule,
  setmoduleresult: { status: "s", gateways: ["g"], failed: [{ id: "g", error: "e" }] } satisfies T.SetModuleResult,
  setupevent: { stepId: "s", line: "l", done: true, err: "e" } satisfies T.SetupEvent,
  agentcheck: { agent: "x", version: "x", chainId: 1, setUp: true } satisfies T.AgentCheck,
  sshcommand: { argv: ["ssh"], display: "x" } satisfies T.SSHCommand,
  error: err,
  hostkeyprobe: { hops: [{ hostPort: "h", fingerprint: "f", keyType: "k", state: "unknown", probeId: "p", error: "e" }], allConfirmed: false } satisfies T.HostKeyProbe,
  recordedhostkeys: { hostPort: "h", keys: [{ fingerprint: "f", keyType: "k", store: "jumpgate" }] } satisfies T.RecordedHostKeys,
  pairevent: { step: "s", line: "l", error: "e", code: "c", hint: "h", done: true, agent: "a" } satisfies T.PairEvent,
  pairinginfo: { controller: "c", transportKey: "k", system: false, scope: "self" } satisfies T.PairingInfo,
  agentupgrade: { upgraded: true, from: "a", to: "b" } satisfies T.GateUpgrade,
};
