import { describe, it, expect } from "vitest";
import { checkKind } from "./securityModel";

describe("checkKind", () => {
  it("maps pass -> ok", () => {
    expect(checkKind("pass")).toBe("ok");
  });
  it("maps fail -> bad", () => {
    expect(checkKind("fail")).toBe("bad");
  });
  it("maps warn -> warn", () => {
    expect(checkKind("warn")).toBe("warn");
  });
  it("maps unknown -> neutral", () => {
    expect(checkKind("unknown")).toBe("neutral");
  });
});
