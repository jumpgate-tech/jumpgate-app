import { describe, it, expect } from "vitest";
import type { DiagReport } from "../../api";
import { failedCheckTitle } from "./diagModel";

const ITEMS: DiagReport["items"] = [
  { ID: "c1", Title: "DNS resolves", Why: "needed", Status: "pass", Detail: "ok", Fix: "" },
  { ID: "c2", Title: "Port 30303 reachable", Why: "peers", Status: "fail", Detail: "closed", Fix: "sudo ufw allow 30303" },
];

describe("failedCheckTitle", () => {
  it("returns the title of the item matching failedId", () => {
    const report: DiagReport = { at: "now", trigger: "manual", items: ITEMS, failedId: "c2" };
    expect(failedCheckTitle(report)).toBe("Port 30303 reachable");
  });

  it("falls back to the raw id when no item matches", () => {
    const report: DiagReport = { at: "now", trigger: "manual", items: ITEMS, failedId: "missing" };
    expect(failedCheckTitle(report)).toBe("missing");
  });

  it("falls back to an empty string when failedId is unset", () => {
    const report: DiagReport = { at: "now", trigger: "manual", items: ITEMS };
    expect(failedCheckTitle(report)).toBe("");
  });
});
