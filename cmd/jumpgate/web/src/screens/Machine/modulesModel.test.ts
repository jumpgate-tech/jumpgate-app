import { describe, expect, test } from "vitest";
import type { Catalog, CatalogModule, ModuleStatus, Snapshot } from "../../api";
import {
  boxFields,
  droppedOnClientChange,
  gatewayModuleCards,
  moduleCards,
  publicGatewayRefs,
  publicOptions,
  resourcesLine,
  statusLine,
  type ModuleCtx,
} from "./modulesModel";

const GO_PULSE_REASON = "The message board needs reth (valve fork). This node runs go-pulse.";

const MSGBOARD: CatalogModule = {
  id: "msgboard",
  name: "Message board",
  summary: "A proof-of-work message board served by your node.",
  siteUrl: "https://msgboard.xyz",
  docsStep: "msgboard-enable",
  placement: "node",
  resources: { ramBytes: 256_000_000, diskBytes: 0 },
  applies: [
    { chainId: 369, client: "reth", ok: true, beta: false },
    { chainId: 369, client: "go-pulse", ok: false, beta: false, reason: GO_PULSE_REASON },
    { chainId: 1, client: "reth", ok: false, beta: false },
  ],
  methods: [{ name: "msgboard_addMessage", class: "write" }],
};

const CHECKPOINTZ: CatalogModule = {
  id: "checkpointz",
  name: "Checkpointz",
  summary: "Serves checkpoint sync to other beacons.",
  siteUrl: "",
  docsStep: "checkpointz-enable",
  placement: "box",
  resources: { ramBytes: 0, diskBytes: 0 },
  applies: [
    { chainId: 369, client: "", ok: true, beta: false },
    { chainId: 943, client: "", ok: true, beta: true },
  ],
  methods: [],
};

const WATCH: CatalogModule = {
  id: "watch",
  name: "Health watch",
  summary: "Watches endpoints.",
  siteUrl: "",
  docsStep: "watch-enable",
  placement: "box",
  resources: { ramBytes: 0, diskBytes: 0 },
  applies: [
    { chainId: 0, client: "", ok: true, beta: false },
    { chainId: 369, client: "", ok: true, beta: false },
  ],
  methods: [],
};

const CAT: Catalog = { networks: [], clients: [], modules: [MSGBOARD, CHECKPOINTZ, WATCH] };

function snap(over: Partial<Snapshot> = {}): Snapshot {
  return {
    at: "",
    execSyncing: false,
    execHead: 1,
    refHead: 1,
    beaconSlot: 1,
    beaconDistance: 0,
    execPeers: 5,
    beaconPeers: 5,
    diskUsedPct: 1,
    execActive: true,
    beaconActive: true,
    ...over,
  };
}

function st(over: Partial<ModuleStatus> = {}): ModuleStatus {
  return { id: "msgboard", served: true, active: true, accepting: true, count: 1234, size: 3_100_000, headBlock: 1, ...over };
}

function ctx(over: Partial<ModuleCtx> = {}): ModuleCtx {
  return { chainId: 369, client: "reth", enabled: [], statuses: [], gateStale: false, box: "box1", now: 1_000_000, ...over };
}

describe("moduleCards", () => {
  test("hidden on another chain", () => {
    const [c] = moduleCards(CAT, "node", ctx({ chainId: 1 }));
    expect(c.id).toBe("msgboard");
    expect(c.state).toBe("hidden");
  });

  test("disabled with the catalog reason", () => {
    const [c] = moduleCards(CAT, "node", ctx({ client: "go-pulse" }));
    expect(c.state).toBe("disabled");
    expect(c.reason).toBe(GO_PULSE_REASON);
  });

  test("off and on", () => {
    expect(moduleCards(CAT, "node", ctx())[0].state).toBe("off");
    expect(moduleCards(CAT, "node", ctx({ enabled: ["msgboard"] }))[0].state).toBe("on");
  });

  test("only the requested placement", () => {
    expect(moduleCards(CAT, "box", ctx()).map((c) => c.id)).toEqual(["checkpointz", "watch"]);
    expect(moduleCards(CAT, "gateway", ctx())).toEqual([]);
  });

  test("beta label", () => {
    const cards = moduleCards(CAT, "box", ctx({ chainId: 943 }));
    expect(cards.find((c) => c.id === "checkpointz")?.beta).toBe(true);
    expect(moduleCards(CAT, "box", ctx())[0].beta).toBe(false);
  });

  test("a chain-agnostic box module uses its chain-0 row", () => {
    const cards = moduleCards(CAT, "box", ctx({ chainId: 1 }));
    expect(cards.find((c) => c.id === "watch")?.state).toBe("off");
    expect(cards.find((c) => c.id === "checkpointz")?.state).toBe("hidden");
  });

  test("restart line and resources on node cards", () => {
    const [c] = moduleCards(CAT, "node", ctx());
    expect(c.restartLine).toBe("This restarts reth on box1. The beacon keeps running.");
    expect(c.resources).toBe("about 256 MB RAM");
    expect(c.siteUrl).toBe("https://msgboard.xyz");
    expect(c.docsStep).toBe("msgboard-enable");
  });

  test("a catalog without modules has no cards", () => {
    expect(moduleCards({ networks: [], clients: [] }, "node", ctx())).toEqual([]);
  });
});

describe("statusLine", () => {
  const on = (over: Partial<ModuleCtx> = {}) => ctx({ enabled: ["msgboard"], snapshot: snap(), ...over });

  test("status-line precedence", () => {
    // 1. Node stopped beats everything.
    expect(statusLine(MSGBOARD, st({ served: false }), on({ snapshot: snap({ execActive: false }) }))).toBe("Node stopped");
    // 2. on but not served: restarted under 60 s ago → Starting; after that the error.
    expect(statusLine(MSGBOARD, st({ served: false }), on({ restartedAt: 1_000_000 - 30_000 }))).toBe("Starting");
    expect(statusLine(MSGBOARD, st({ served: false, error: "msgboard is not served" }), on({ restartedAt: 1_000_000 - 90_000 }))).toBe(
      "msgboard is not served",
    );
    // 3. syncing
    expect(statusLine(MSGBOARD, st(), on({ snapshot: snap({ execSyncing: true, execPeers: 0 }) }))).toBe("Waiting for sync");
    // 4. no peers
    expect(statusLine(MSGBOARD, st({ accepting: false }), on({ snapshot: snap({ execPeers: 0 }) }))).toBe("No peers");
    // 5. not accepting
    expect(statusLine(MSGBOARD, st({ accepting: false }), on())).toBe("Starting");
    // 6. accepting
    expect(statusLine(MSGBOARD, st(), on())).toBe("Accepting · 1,234 messages · 3.1 MB");
    // 7. off but served
    expect(statusLine(MSGBOARD, st(), ctx({ snapshot: snap() }))).toBe("Served but not set by jumpgate");
    expect(statusLine(MSGBOARD, st({ served: false }), ctx({ snapshot: snap() }))).toBe("");
  });

  test("older gate", () => {
    const cards = moduleCards(CAT, "node", ctx({ enabled: ["msgboard"], gateStale: true }));
    expect(cards[0].status).toBe("Status needs a newer gate on this box");
    expect(cards[0].state).toBe("on");
  });

  test("box module status", () => {
    const c = ctx({ enabled: ["checkpointz"] });
    expect(statusLine(CHECKPOINTZ, st({ id: "checkpointz", active: true, identity: "enode://x" }), c)).toBe("Running");
    expect(statusLine(CHECKPOINTZ, st({ id: "checkpointz", active: true }), c)).toBe("Running");
    expect(statusLine(CHECKPOINTZ, st({ id: "checkpointz", active: false }), c)).toBe("Stopped");
    expect(statusLine(CHECKPOINTZ, st({ id: "checkpointz", error: "unit failed" }), c)).toBe("unit failed");
    expect(statusLine(CHECKPOINTZ, undefined, c)).toBe("Stopped");
    expect(statusLine(CHECKPOINTZ, undefined, ctx())).toBe("");
  });
});

describe("gatewayModuleCards", () => {
  const PRIVATE_SEND: CatalogModule = {
    id: "private-send",
    name: "Private send",
    summary: "Sends transactions privately.",
    siteUrl: "",
    docsStep: "private-send-enable",
    placement: "gateway",
    resources: { ramBytes: 0, diskBytes: 0 },
    applies: [
      { chainId: 1, client: "", ok: true, beta: false },
      { chainId: 369, client: "", ok: false, beta: false },
      { chainId: 943, client: "", ok: false, beta: false, reason: "Not on testnet yet." },
    ],
    methods: [],
  };
  const cat: Catalog = { networks: [], clients: [], modules: [MSGBOARD, PRIVATE_SEND] };

  test("applies when any of the gateway's chains does", () => {
    const [c] = gatewayModuleCards(cat, [369, 1], []);
    expect(c.id).toBe("private-send");
    expect(c.state).toBe("off");
    expect(gatewayModuleCards(cat, [369, 1], ["private-send"])[0].state).toBe("on");
  });

  test("a reason wins over hidden; nothing applies → hidden", () => {
    expect(gatewayModuleCards(cat, [369, 943], [])[0]).toMatchObject({ state: "disabled", reason: "Not on testnet yet." });
    expect(gatewayModuleCards(cat, [369], [])[0].state).toBe("hidden");
    expect(gatewayModuleCards(cat, [], [])[0].state).toBe("hidden");
  });
});

describe("droppedOnClientChange", () => {
  test("dropped on client change", () => {
    expect(droppedOnClientChange(CAT, 369, "go-pulse", ["msgboard"])).toEqual([
      { id: "msgboard", notice: "The message board will be turned off: go-pulse does not support it." },
    ]);
    expect(droppedOnClientChange(CAT, 369, "reth", ["msgboard"])).toEqual([]);
  });
});

describe("helpers", () => {
  test("resources line", () => {
    expect(resourcesLine({ ramBytes: 0, diskBytes: 0 })).toBe("");
    expect(resourcesLine({ ramBytes: 512_000_000, diskBytes: 2_000_000_000 })).toBe("about 512 MB RAM · 2 GB disk");
  });

  test("box fields only where declared", () => {
    expect(boxFields("el-bootnode")).toEqual(["port", "extIp"]);
    expect(boxFields("checkpointz")).toEqual(["port"]);
    expect(boxFields("watch")).toEqual([]);
  });

  test("public gateways referencing a node (server-provided tier)", () => {
    const gw = (id: string, tier: "local" | "private" | "public", target: string) => ({
      id,
      tier,
      config: {
        // TLS deliberately says nothing: the tier is the server's verdict.
        Networks: [{ ChainID: 369, Upstreams: [{ ID: "u", Kind: "managed-node" as const, TargetID: target, Endpoint: "", Local: true, RecentOnly: false }] }],
      },
    });
    expect(
      publicGatewayRefs([gw("a", "public", "t1"), gw("b", "private", "t1"), gw("c", "public", "t2"), gw("d", "local", "t1")], "t1"),
    ).toEqual(["a"]);
  });

  test("public options come from the catalog", () => {
    expect(publicOptions({ ...MSGBOARD, public: ["read", "post"] }).map((o) => o.value)).toEqual(["", "read", "post"]);
    expect(publicOptions({ ...MSGBOARD, public: ["read"] }).map((o) => o.value)).toEqual(["", "read"]);
    expect(publicOptions({ ...MSGBOARD, public: [] })).toEqual([]);
    expect(publicOptions(MSGBOARD)).toEqual([]);
  });
});

describe("fix round 1", () => {
  test("an on module whose row is now disabled can still be turned off", () => {
    const [c] = moduleCards(CAT, "node", ctx({ client: "go-pulse", enabled: ["msgboard"] }));
    expect(c.state).toBe("on");
    expect(c.reason).toBe(GO_PULSE_REASON);
    const [h] = moduleCards(CAT, "node", ctx({ chainId: 1, enabled: ["msgboard"] }));
    expect(h.state).toBe("on");
  });

  test("Node stopped only for a module that is on", () => {
    const stopped = snap({ execActive: false });
    expect(statusLine(MSGBOARD, st({ served: false }), ctx({ snapshot: stopped }))).toBe("");
    expect(statusLine(MSGBOARD, st({ served: false }), ctx({ snapshot: stopped, enabled: ["msgboard"] }))).toBe("Node stopped");
  });

  test("restart line names the box", () => {
    const [c] = moduleCards(CAT, "node", ctx({ box: "203.0.113.9" }));
    expect(c.restartLine).toBe("This restarts reth on 203.0.113.9. The beacon keeps running.");
  });
});
