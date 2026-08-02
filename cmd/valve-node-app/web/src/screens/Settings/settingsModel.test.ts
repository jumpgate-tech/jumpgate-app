import { describe, it, expect } from "vitest";
import { VALVE_KEY, keyNames } from "./settingsModel";

describe("keyNames", () => {
  it("always leads with VALVE_API_KEY even when unset", () => {
    expect(keyNames([])).toEqual([VALVE_KEY]);
  });

  it("appends the rest sorted, without duplicating VALVE_API_KEY", () => {
    expect(keyNames(["INFURA_API_KEY", "ALCHEMY_API_KEY", VALVE_KEY])).toEqual([
      VALVE_KEY,
      "ALCHEMY_API_KEY",
      "INFURA_API_KEY",
    ]);
  });

  it("coalesces a null/undefined providerKeysSet to just VALVE_API_KEY", () => {
    expect(keyNames(null)).toEqual([VALVE_KEY]);
    expect(keyNames(undefined)).toEqual([VALVE_KEY]);
  });
});
