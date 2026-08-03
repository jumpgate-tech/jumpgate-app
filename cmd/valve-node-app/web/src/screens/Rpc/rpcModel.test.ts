import { describe, it, expect } from "vitest";
import type { EndpointCapabilities, GatewayTraffic, GatewayView, NetworkView, UpstreamView } from "../../api";
import {
  DEVNET_CHAIN_ID,
  attentionLines,
  canAddGatewayOn,
  capTagClass,
  chainLacksEntirely,
  chainVerdict,
  configuredSchemes,
  defaultSelection,
  internalCaPath,
  manualTrustCommand,
  mergePendingNetworks,
  orderedNetworks,
  pruneEmptyNetworks,
  redundancy,
  shareCellModel,
  statusOf,
  storedConfig,
  tlsAttention,
  withDevnetUpstream,
  withExternalUpstreams,
  withManagedUpstream,
  withoutChain,
  withoutEndpoint,
} from "./rpcModel";

function up(over: Partial<UpstreamView> = {}): UpstreamView {
  return { id: "u1", kind: "external", endpoint: "https://a.example", label: "a", local: false, recentOnly: false, actions: [], ...over };
}
function net(over: Partial<NetworkView> = {}): NetworkView {
  return { chainId: 1, name: "Ethereum", path: "/main/evm/1", upstreams: [], knownSetSize: 0, serviceable: true, ...over };
}

describe("chainVerdict", () => {
  it("no endpoint → bad/no-endpoint", () => {
    expect(chainVerdict(net({ upstreams: [] }))).toMatchObject({ tone: "bad", kind: "no-endpoint" });
  });
  it("not serviceable → bad/not-serviceable", () => {
    expect(chainVerdict(net({ upstreams: [up({ endpoint: "wss://x" })], serviceable: false }))).toMatchObject({
      tone: "bad",
      kind: "not-serviceable",
    });
  });
  it("no ws upstream → warn/no-websocket naming the schemes", () => {
    const v = chainVerdict(net({ upstreams: [up({ endpoint: "https://a" }), up({ endpoint: "http://b" })] }));
    expect(v).toMatchObject({ tone: "warn", kind: "no-websocket" });
    expect(v.schemes).toEqual(["http", "https"]);
    expect(v.why).toContain("eth_subscribe");
  });
  it("single ws upstream → warn/single", () => {
    expect(chainVerdict(net({ upstreams: [up({ endpoint: "wss://a" })] }))).toMatchObject({ tone: "warn", kind: "single" });
  });
  it("multiple but none local → warn/no-local", () => {
    const v = chainVerdict(net({ upstreams: [up({ endpoint: "wss://a" }), up({ id: "u2", endpoint: "https://b" })] }));
    expect(v).toMatchObject({ tone: "warn", kind: "no-local" });
  });
  it("a broken one among many → warn/some-unusable with counts", () => {
    const v = chainVerdict(
      net({
        upstreams: [
          up({ endpoint: "wss://a", local: true }),
          up({ id: "u2", endpoint: "https://b", problem: "dead" }),
        ],
      }),
    );
    expect(v).toMatchObject({ tone: "warn", kind: "some-unusable", broken: 1, total: 2, usable: 1 });
  });
  it("healthy → ok with total", () => {
    const v = chainVerdict(
      net({ upstreams: [up({ endpoint: "wss://a", local: true }), up({ id: "u2", endpoint: "https://b" })] }),
    );
    expect(v).toMatchObject({ tone: "ok", kind: "ok", total: 2 });
  });
});

describe("configuredSchemes", () => {
  it("dedupes and sorts, skipping unresolved endpoints", () => {
    expect(configuredSchemes([up({ endpoint: "https://a" }), up({ endpoint: "WSS://b" }), up({ endpoint: "" })])).toEqual([
      "https",
      "wss",
    ]);
  });
});

describe("orderedNetworks", () => {
  it("keeps real chains' order and puts devnet last", () => {
    const nets = [net({ chainId: DEVNET_CHAIN_ID, name: "d" }), net({ chainId: 1 }), net({ chainId: 369 })];
    expect(orderedNetworks(nets).map((n) => n.chainId)).toEqual([1, 369, DEVNET_CHAIN_ID]);
  });
});

describe("redundancy", () => {
  it("counts against the set when one exists", () => {
    expect(redundancy(4, 7)).toMatchObject({ total: 7, filled: 4, label: "4 of 7" });
  });
  it("states the count alone with no set", () => {
    expect(redundancy(2, 0)).toMatchObject({ total: 2, filled: 2, label: "2" });
  });
  it("states the overshoot in the open past the set size", () => {
    expect(redundancy(8, 7).label).toBe("8 (set is 7)");
  });
});

describe("statusOf + chainLacksEntirely + capTagClass", () => {
  const reach: EndpointCapabilities = { upstream: "u1", chainId: 1, reachable: true, capabilities: [] };
  const unreach: EndpointCapabilities = { upstream: "u2", chainId: 1, reachable: false, capabilities: [] };
  it("synthesises http from reachability", () => {
    expect(statusOf(reach, "http")).toBe("supported");
    expect(statusOf(unreach, "http")).toBe("unsupported");
  });
  it("reads a real capability from the list", () => {
    const e: EndpointCapabilities = { upstream: "u1", chainId: 1, reachable: true, capabilities: [{ key: "ws", label: "WebSocket", status: "supported" }] };
    expect(statusOf(e, "ws")).toBe("supported");
    expect(statusOf(e, "trace")).toBeUndefined();
  });
  it("chainLacksEntirely only when every probed endpoint is unsupported", () => {
    expect(chainLacksEntirely([unreach], "http")).toBe(true);
    expect(chainLacksEntirely([reach, unreach], "http")).toBe(false);
    expect(chainLacksEntirely([], "http")).toBe(false);
  });
  it("capTagClass distinguishes missing chain-wide from off here", () => {
    expect(capTagClass("unsupported", true)).toBe("cap missing");
    expect(capTagClass("unsupported", false)).toBe("cap off");
    expect(capTagClass("inconclusive", false)).toBe("cap unknown");
    expect(capTagClass("inconsistent", false)).toBe("cap mixed");
    expect(capTagClass("supported", false)).toBe("cap");
  });
});

describe("shareCellModel", () => {
  const traffic: GatewayTraffic = {
    enabled: true,
    at: "now",
    since: "then",
    networks: [{ chainId: 1, received: 10, attributed: 10, unattributed: 0, upstreams: [{ upstream: "u1", succeeded: 5, actual: 0.5, intended: 0.5, diverged: false }] }],
  };
  it("reading while loading", () => {
    expect(shareCellModel(undefined, true, 1, "u1", false)).toEqual({ kind: "reading" });
  });
  it("unreadable when null", () => {
    expect(shareCellModel(null, false, 1, "u1", false)).toEqual({ kind: "unreadable" });
  });
  it("off when counters disabled", () => {
    expect(shareCellModel({ ...traffic, enabled: false }, false, 1, "u1", false)).toEqual({ kind: "off" });
  });
  it("none when no attributed traffic", () => {
    expect(shareCellModel({ ...traffic, networks: [{ chainId: 1, received: 0, attributed: 0, unattributed: 0, upstreams: [] }] }, false, 1, "u1", false)).toEqual({ kind: "none" });
  });
  it("a bar with pct + tick", () => {
    expect(shareCellModel(traffic, false, 1, "u1", false)).toMatchObject({ kind: "bar", pct: 50, tickPct: 50, fill: "ok" });
  });
});

describe("tls attention", () => {
  function gw(over: Partial<GatewayView> = {}): GatewayView {
    return {
      id: "default",
      label: "default",
      containerName: "erpc-default",
      placement: { targetId: "local", backend: "docker" },
      status: { ID: "", ContainerName: "", State: "running", Image: "", ImageID: "", ExitCode: 0, Platform: "", EnginePlatform: "", Emulated: false, Detail: "" },
      docker: { present: true, reachable: true, flavor: "docker" },
      baseUrl: "https://valve.local",
      tls: { enabled: false, status: { ID: "", ContainerName: "", State: "running", Image: "", ImageID: "", ExitCode: 0, Platform: "", EnginePlatform: "", Emulated: false, Detail: "" } },
      networks: [],
      actions: [],
      wipeDiscards: "the container",
      config: { ProjectID: "main", BindAddr: "127.0.0.1", Port: 4000, Networks: [] },
      ...over,
    };
  }
  it("no lines when TLS is off", () => {
    expect(tlsAttention(gw(), null)).toEqual([]);
  });
  it("warns when the front is not running, with a start command", () => {
    const g = gw({ tls: { enabled: true, url: "https://x", containerName: "caddy-default", status: { ID: "", ContainerName: "", State: "created-but-stopped", Image: "", ImageID: "", ExitCode: 0, Platform: "", EnginePlatform: "", Emulated: false, Detail: "" } } });
    const lines = tlsAttention(g, null);
    expect(lines[0]).toMatchObject({ tone: "warn", cmd: "docker start caddy-default" });
  });
  it("attentionLines folds read error + block + action error", () => {
    const lines = attentionLines(gw({ error: "boom", hint: "start docker", blocked: "held" }), null, "restart failed");
    expect(lines.map((l) => l.text)).toEqual([
      "This gateway could not be read: boom — start docker",
      "held",
      "restart failed",
    ]);
  });
});

describe("config transforms", () => {
  function gwCfg(): GatewayView {
    return {
      id: "default",
      config: { ProjectID: "main", BindAddr: "127.0.0.1", Port: 4000, Networks: [{ ChainID: 1, Upstreams: [{ ID: "public-1-1", Kind: "external", Endpoint: "https://a", Local: false, RecentOnly: false }] }] },
    } as unknown as GatewayView;
  }
  it("storedConfig deep-copies networks", () => {
    const gw = gwCfg();
    const cfg = storedConfig(gw);
    cfg.Networks![0].Upstreams[0].Endpoint = "changed";
    expect(gw.config.Networks![0].Upstreams[0].Endpoint).toBe("https://a");
  });
  it("withExternalUpstreams continues the id scheme and skips dupes", () => {
    const cfg = withExternalUpstreams(storedConfig(gwCfg()), 1, ["https://a", "https://b"]);
    const ids = cfg.Networks!.find((n) => n.ChainID === 1)!.Upstreams.map((u) => u.ID);
    expect(ids).toEqual(["public-1-1", "public-1-2"]);
  });
  it("withManagedUpstream / withDevnetUpstream create a network when absent", () => {
    const a = withManagedUpstream(storedConfig(gwCfg()), 369, "managed-node", "box");
    expect(a.Networks!.find((n) => n.ChainID === 369)!.Upstreams[0]).toMatchObject({ ID: "node-box", Kind: "managed-node", Local: true });
    const d = withDevnetUpstream(storedConfig(gwCfg()), DEVNET_CHAIN_ID, "local");
    expect(d.Networks!.find((n) => n.ChainID === DEVNET_CHAIN_ID)!.Upstreams[0]).toMatchObject({ ID: "devnet", Kind: "managed-devnet" });
  });
  it("withoutChain / withoutEndpoint / pruneEmptyNetworks", () => {
    expect(withoutChain(storedConfig(gwCfg()), 1).Networks).toEqual([]);
    expect(withoutEndpoint(storedConfig(gwCfg()), 1, "public-1-1").Networks![0].Upstreams).toEqual([]);
    expect(pruneEmptyNetworks({ ProjectID: "main", BindAddr: "", Port: 1, Networks: [{ ChainID: 2, Upstreams: [] }] }).Networks).toEqual([]);
  });
});

describe("mergePendingNetworks", () => {
  it("adds placeholders not yet echoed, drops ones the server returned", () => {
    const merged = mergePendingNetworks([net({ chainId: 1 })], [1, 42], [{ chainId: 42, name: "Chain 42" }], "main");
    expect(merged.map((n) => n.chainId)).toEqual([1, 42]);
    expect(merged[1]).toMatchObject({ serviceable: false, path: "/main/evm/42" });
  });
});

describe("misc", () => {
  it("defaultSelection ticks the fastest three and guarantees a ws slot", () => {
    const sel = defaultSelection([
      { url: "https://a", kind: "http", status: "live", latencyMs: 10 },
      { url: "https://b", kind: "http", status: "live", latencyMs: 20 },
      { url: "https://c", kind: "http", status: "live", latencyMs: 30 },
      { url: "wss://d", kind: "ws", status: "unprobed", latencyMs: 999 },
    ]);
    expect(sel.has("wss://d")).toBe(true);
    expect(sel.size).toBe(3);
  });
  it("manualTrustCommand per OS", () => {
    expect(manualTrustCommand("darwin", "/x", "g")).toContain("security add-trusted-cert");
    expect(manualTrustCommand("windows", "/x", "g")).toContain("certutil");
    expect(manualTrustCommand("linux", "/x", "g")).toContain("update-ca-certificates");
  });
  it("internalCaPath only for internal-CA fronts", () => {
    const base = { enabled: true, rootCaPath: "/root.crt", effectiveCertSource: "internal" };
    expect(internalCaPath({ tls: base } as unknown as GatewayView)).toBe("/root.crt");
    expect(internalCaPath({ tls: { ...base, effectiveCertSource: "files" } } as unknown as GatewayView)).toBeNull();
  });
  it("canAddGatewayOn is false where a gateway already sits", () => {
    const gws = [{ placement: { targetId: "local" } }] as unknown as GatewayView[];
    expect(canAddGatewayOn("local", gws)).toBe(false);
    expect(canAddGatewayOn("box2", gws)).toBe(true);
  });
});
