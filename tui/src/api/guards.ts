// Type guards for every stream payload. A frame that fails its guard is
// dropped, so a bad or hostile frame cannot crash rendering. They check the
// fields the screens read, not every field Go sends.
import type * as T from "./types.ts";

type G<X> = (v: unknown) => v is X;
const obj = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);
const str = (v: unknown): v is string => typeof v === "string";
const num = (v: unknown): v is number => typeof v === "number" && Number.isFinite(v);
const bool = (v: unknown): v is boolean => typeof v === "boolean";
const opt = <X>(g: G<X>) => (v: unknown): v is X | undefined => v === undefined || g(v);
const nul = <X>(g: G<X>) => (v: unknown): v is X | null | undefined => v === undefined || v === null || g(v);
const arr = <X>(g: G<X>) => (v: unknown): v is X[] => Array.isArray(v) && v.every(g);
const shape = <X>(spec: Record<string, (v: unknown) => boolean>): G<X> => (v: unknown): v is X =>
  obj(v) && Object.entries(spec).every(([k, g]) => g(v[k]));

const errBody = shape<T.ApiErrorBody>({ error: str, code: opt(str), hint: opt(str) });
const client = shape<T.ClientStatus>({ active: bool, head: num, peers: num, state: str });
export const isNodeStatus = shape<T.NodeStatus>({ at: str, exec: client, beacon: client, overall: str, headLag: opt(num), diskUsedPct: opt(num) });
const disk = shape<T.DiskView>({ at: str, execBytes: num, beaconBytes: num, freeBytes: num, expectedExecBytes: num, expectedBeaconBytes: num, expectedLabel: str, fit: str, daysToFull: opt(num) });
const row = shape<T.FleetRow>({
  id: str,
  link: str,
  reachable: bool,
  lastSeen: opt(str),
  agent: opt(str),
  status: nul(isNodeStatus),
  disk: nul(disk),
  firewall: nul(shape({ grade: str })),
  jobs: nul(shape({ running: num, needsDecision: num })),
  error: nul(errBody),
  statusError: nul(errBody),
  diskError: nul(errBody),
  firewallError: nul(errBody),
  stale: shape({ status: bool, disk: bool, firewall: bool }),
  alertCount: num,
  alertMessage: opt(str),
});
export const isFleet = shape<T.Fleet>({ at: str, intervalSeconds: num, rows: arr(row), locked: opt(bool) });

const option = shape<T.JobOption>({ id: str, label: str, detail: opt(str) });
export const isJobView = shape<T.JobView>({
  target: str,
  id: str,
  kind: str,
  title: str,
  state: str,
  step: num,
  steps: arr(shape({ id: str, title: str, state: str })),
  question: opt(shape({ id: str, prompt: str, options: arr(option) })),
  lease: shape({ fresh: bool, mine: bool, label: opt(str) }),
  error: opt(str),
});
export const isJobsInbox = shape<T.JobsInbox>({ at: str, jobs: arr(isJobView), errors: arr(shape({ target: str, error: errBody })) });
export const isLogHit = shape<T.LogHit>({ unit: str, line: str, at: str, severity: str });
export const isLogHits = arr(isLogHit);
export const isJobOutput = arr(shape<T.JobOutputLine>({ unit: str, line: str, at: str }));
export const isSetupEvent = shape<T.SetupEvent>({ stepId: str, line: opt(str), done: opt(bool), err: opt(str) });
export const isPairEvent = shape<T.PairEvent>({ step: opt(str), line: opt(str), error: opt(str), code: opt(str), hint: opt(str), done: opt(bool), agent: opt(str) });
