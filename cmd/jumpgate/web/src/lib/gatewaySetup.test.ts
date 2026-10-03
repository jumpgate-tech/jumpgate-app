import { describe, it, expect } from "vitest";
import { internalTLSConfig, SETUP_CHAINS } from "./gatewaySetup";
import type { GatewayUpstream } from "../api";

describe("internalTLSConfig", () => {
  it("builds an eRPC-on-4000 config fronted by Caddy's internal CA", () => {
    const cfg = internalTLSConfig([]);

    expect(cfg.ProjectID).toBe("main");
    expect(cfg.BindAddr).toBe("127.0.0.1");
    expect(cfg.Port).toBe(4000);
    expect(cfg.Networks).toEqual([]);
    expect(cfg.TLS).toEqual({
      Enabled: true,
      Hostname: "",
      CertSource: "internal",
      CertFile: "",
      KeyFile: "",
      HTTPSPort: 0,
      BindAddr: "",
      ImageRef: "",
    });
  });

  it("passes the given networks through untouched", () => {
    const upstream: GatewayUpstream = {
      ID: "public-1-1",
      Kind: "external",
      Endpoint: "https://rpc.example",
      Local: false,
      RecentOnly: false,
    };
    const cfg = internalTLSConfig([{ ChainID: 1, Upstreams: [upstream] }]);

    expect(cfg.Networks).toEqual([{ ChainID: 1, Upstreams: [upstream] }]);
  });

  it("hostname is left empty for the server to fill", () => {
    expect(internalTLSConfig([]).TLS?.Hostname).toBe("");
  });
});

describe("SETUP_CHAINS", () => {
  it("is Ethereum and PulseChain, and never the devnet", () => {
    expect(SETUP_CHAINS.map((c) => c.chainId)).toEqual([1, 369]);
    expect(SETUP_CHAINS.some((c) => c.chainId === 1337)).toBe(false);
  });
});
