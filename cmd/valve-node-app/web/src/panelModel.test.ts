import { describe, it, expect } from "vitest";
import {
  endpointNameFromUrl,
  masterState,
  healthClass,
  capabilityCells,
  withNetwork,
  withoutNetwork,
  withUpstream,
  withoutUpstream,
} from "./panelModel";
import type { GatewayView, GatewayConfig, GatewayUpstream } from "./api";

describe("endpointNameFromUrl", () => {
  it("takes the registrable label from the host", () => {
    expect(endpointNameFromUrl("https://rpc.publicnode.com/pulsechain")).toBe("publicnode");
  });
  it("handles a bare host", () => {
    expect(endpointNameFromUrl("https://mainnet.infura.io")).toBe("infura");
  });
  it("falls back to host for localhost/ip", () => {
    expect(endpointNameFromUrl("http://127.0.0.1:8545")).toBe("127.0.0.1");
    expect(endpointNameFromUrl("http://localhost:8545")).toBe("localhost");
  });
  it("returns 'endpoint' for an unparseable string", () => {
    expect(endpointNameFromUrl("not a url")).toBe("endpoint");
  });
});

const gw = (over: Partial<GatewayView>): GatewayView => ({
  id: "default", label: "", containerName: "", placement: { targetId: "local", backend: "docker" },
  status: { State: "running" } as GatewayView["status"], docker: {} as GatewayView["docker"],
  baseUrl: "", tls: {} as GatewayView["tls"], networks: [], actions: [], wipeDiscards: "",
  config: {} as GatewayView["config"], ...over,
});

describe("masterState", () => {
  it("running gateway is green with a stop action", () => {
    const m = masterState(gw({ status: { State: "running" } as GatewayView["status"], actions: ["stop", "restart"] }));
    expect(m.tone).toBe("on"); expect(m.label).toBe("Running"); expect(m.actions).toContain("stop");
  });
  it("stopped gateway is red and offers start", () => {
    const m = masterState(gw({ status: { State: "created-but-stopped" } as GatewayView["status"], actions: ["start"] }));
    expect(m.tone).toBe("off"); expect(m.actions).toContain("start");
  });
  it("blocked gateway surfaces the reason and offers nothing", () => {
    const m = masterState(gw({ status: { State: "unknown" } as GatewayView["status"], actions: [], blocked: "Docker unreachable" }));
    expect(m.tone).toBe("blocked"); expect(m.blocked).toBe("Docker unreachable");
  });
  it("null gateway (nothing set up) is off", () => {
    expect(masterState(null).tone).toBe("off");
  });
});

describe("healthClass", () => {
  it("off when the gateway is not running", () => {
    expect(healthClass({ running: false, serviceable: true })).toBe("off");
  });
  it("frequent when running but not serviceable", () => {
    expect(healthClass({ running: true, serviceable: false })).toBe("frequent");
  });
  it("stable when serviceable and slow requests are rare/unknown", () => {
    expect(healthClass({ running: true, serviceable: true })).toBe("stable");
    expect(healthClass({ running: true, serviceable: true, slowRate: 0.02 })).toBe("stable");
  });
  it("occasional then frequent as the slow rate climbs", () => {
    expect(healthClass({ running: true, serviceable: true, slowRate: 0.15 })).toBe("occasional");
    expect(healthClass({ running: true, serviceable: true, slowRate: 0.6 })).toBe("frequent");
  });
});

describe("capabilityCells", () => {
  it("lights supported caps in fixed order, archive is the hot one", () => {
    const cells = capabilityCells({ http: "supported", ws: "supported", archive: "supported", trace: "unsupported" });
    expect(cells.map((c) => c.key)).toEqual(["http", "ws", "archive", "trace"]);
    expect(cells.find((c) => c.key === "archive")).toMatchObject({ lit: true, hot: true });
    expect(cells.find((c) => c.key === "trace")).toMatchObject({ lit: false });
  });
});

const upstream = (id: string, endpoint = "https://x.example"): GatewayUpstream => ({
  ID: id,
  Kind: "external",
  Endpoint: endpoint,
  Local: false,
  RecentOnly: false,
});

const baseConfig = (): GatewayConfig =>
  ({
    ProjectID: "default",
    BindAddr: "127.0.0.1",
    Port: 4000,
    Networks: [{ ChainID: 1, Upstreams: [upstream("a"), upstream("b")] }],
  }) as GatewayConfig;

describe("withNetwork", () => {
  it("appends a new network", () => {
    const cfg = baseConfig();
    const snapshot = JSON.parse(JSON.stringify(cfg));
    const next = withNetwork(cfg, 369, [upstream("c")]);
    expect(next.Networks).toHaveLength(2);
    expect(next.Networks!.find((n) => n.ChainID === 369)?.Upstreams.map((u) => u.ID)).toEqual(["c"]);
    expect(cfg).toEqual(snapshot);
  });

  it("replaces upstreams for a duplicate chainId", () => {
    const cfg = baseConfig();
    const snapshot = JSON.parse(JSON.stringify(cfg));
    const next = withNetwork(cfg, 1, [upstream("z")]);
    expect(next.Networks).toHaveLength(1);
    expect(next.Networks![0].Upstreams.map((u) => u.ID)).toEqual(["z"]);
    expect(cfg).toEqual(snapshot);
  });
});

describe("withoutNetwork", () => {
  it("drops the network", () => {
    const cfg = baseConfig();
    const snapshot = JSON.parse(JSON.stringify(cfg));
    const next = withoutNetwork(cfg, 1);
    expect(next.Networks).toEqual([]);
    expect(cfg).toEqual(snapshot);
  });

  it("is a no-op when the chainId isn't present", () => {
    const cfg = baseConfig();
    const next = withoutNetwork(cfg, 999);
    expect(next.Networks).toHaveLength(1);
  });
});

describe("withUpstream", () => {
  it("replaces an existing ID", () => {
    const cfg = baseConfig();
    const snapshot = JSON.parse(JSON.stringify(cfg));
    const next = withUpstream(cfg, 1, upstream("a", "https://replaced.example"));
    expect(next.Networks![0].Upstreams).toHaveLength(2);
    expect(next.Networks![0].Upstreams.find((u) => u.ID === "a")?.Endpoint).toBe("https://replaced.example");
    expect(cfg).toEqual(snapshot);
  });

  it("appends a new ID", () => {
    const cfg = baseConfig();
    const snapshot = JSON.parse(JSON.stringify(cfg));
    const next = withUpstream(cfg, 1, upstream("c"));
    expect(next.Networks![0].Upstreams.map((u) => u.ID)).toEqual(["a", "b", "c"]);
    expect(cfg).toEqual(snapshot);
  });

  it("creates the network when the chainId doesn't exist yet", () => {
    const cfg = baseConfig();
    const next = withUpstream(cfg, 369, upstream("x"));
    expect(next.Networks!.find((n) => n.ChainID === 369)?.Upstreams.map((u) => u.ID)).toEqual(["x"]);
  });
});

describe("withoutUpstream", () => {
  it("removes by ID and leaves the network present", () => {
    const cfg = baseConfig();
    const snapshot = JSON.parse(JSON.stringify(cfg));
    const next = withoutUpstream(cfg, 1, "a");
    expect(next.Networks).toHaveLength(1);
    expect(next.Networks![0].Upstreams.map((u) => u.ID)).toEqual(["b"]);
    expect(cfg).toEqual(snapshot);
  });
});
