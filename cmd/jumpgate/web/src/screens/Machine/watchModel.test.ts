import { describe, expect, test } from "vitest";
import type { CatalogModule } from "../../api";
import { chainAgnostic, endpointLines, watchRequest } from "./watchModel";

const mod = (chains: number[]): CatalogModule => ({
  id: "x",
  name: "X",
  summary: "",
  siteUrl: "",
  docsStep: "",
  placement: "box",
  resources: { ramBytes: 0, diskBytes: 0 },
  methods: [],
  applies: chains.map((chainId) => ({ chainId, client: "", ok: true, beta: false })),
});

describe("watch settings", () => {
  test("endpoint lines are trimmed name=url entries", () => {
    expect(endpointLines(" gw=https://rpc.example \n\n cl=beacon+http://10.0.0.2:5052\n")).toEqual([
      "gw=https://rpc.example",
      "cl=beacon+http://10.0.0.2:5052",
    ]);
  });

  test("an empty webhook keeps the saved one; clear removes it; a new one replaces it", () => {
    expect(watchRequest("gw=https://a", "", false)).toEqual({ enabled: true, upstreams: ["gw=https://a"] });
    expect(watchRequest("", "  ", true)).toEqual({ enabled: true, upstreams: [], clearWebhook: true });
    expect(watchRequest("", "https://ntfy.sh/t", false)).toEqual({
      enabled: true,
      upstreams: [],
      webhook: "https://ntfy.sh/t",
    });
  });

  test("a webhook must be https", () => {
    expect(() => watchRequest("", "http://ntfy.sh/t", false)).toThrow("The webhook must be an https:// URL.");
  });

  test("a module listed only for chain 0 takes no chain", () => {
    expect(chainAgnostic(mod([0]))).toBe(true);
    expect(chainAgnostic(mod([0, 369]))).toBe(false);
    expect(chainAgnostic(undefined)).toBe(false);
  });
});
