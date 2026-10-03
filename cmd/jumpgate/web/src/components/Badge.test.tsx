import { render, cleanup } from "@testing-library/react";
import { describe, it, expect, afterEach } from "vitest";
import { Badge } from "./Badge";

afterEach(cleanup);

describe("Badge", () => {
  it("renders the text with a kind-specific class", () => {
    const { getByText } = render(<Badge text="pass" kind="ok" />);
    expect(getByText("pass")).toHaveClass("badge", "badge-ok");
  });
});
