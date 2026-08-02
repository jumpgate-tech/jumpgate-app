import { describe, it, expect } from "vitest";
import { endpointNameFromUrl, masterState } from "./panelModel";
import type { GatewayView } from "./api";

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
