import { render } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { LegacyScreen } from "./LegacyScreen";

describe("LegacyScreen", () => {
  it("mounts the legacy render and runs cleanup on unmount", () => {
    const cleanup = vi.fn();
    const legacy = vi.fn((el: HTMLElement) => {
      el.textContent = "legacy";
      return cleanup;
    });
    const { unmount, container } = render(<LegacyScreen render={legacy} />);
    expect(legacy).toHaveBeenCalledTimes(1);
    expect(container.textContent).toContain("legacy");
    unmount();
    expect(cleanup).toHaveBeenCalledTimes(1);
  });
});
