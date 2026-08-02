import { describe, it, expect } from "vitest";
import { endpointNameFromUrl } from "./panelModel";

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
