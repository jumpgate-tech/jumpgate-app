import { describe, expect, it } from "vitest";
import { isTerminal, setupFinished, setupLine } from "./setupEvents";

describe("setupEvents", () => {
  it("formats a step's events as before", () => {
    expect(setupLine({ stepId: "preflight", line: "checking" })).toBe("preflight: checking");
    expect(setupLine({ stepId: "run", done: true })).toBe("run: done");
    expect(setupLine({ stepId: "config", err: "boom" })).toBe("config: boom");
  });

  it("gives the terminal frame no line, or its error once with a label", () => {
    expect(isTerminal({ stepId: "", done: true })).toBe(true);
    expect(isTerminal({ stepId: "run", done: true })).toBe(false);
    expect(setupLine({ stepId: "", done: true })).toBeNull();
    expect(setupLine({ stepId: "", done: true, err: "context canceled" })).toBe("setup: context canceled");
  });

  it("finishes on an error, the final step's done, or the terminal frame", () => {
    expect(setupFinished({ stepId: "preflight", err: "x" })).toBe(true);
    expect(setupFinished({ stepId: "run", done: true })).toBe(true);
    expect(setupFinished({ stepId: "", done: true })).toBe(true);
    expect(setupFinished({ stepId: "preflight", done: true })).toBe(false);
    expect(setupFinished({ stepId: "run", done: false })).toBe(false);
    expect(setupFinished({ stepId: "" })).toBe(false);
  });
});
