import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

function Hello() {
  return <p>hello react</p>;
}

describe("toolchain", () => {
  it("renders a component under jsdom", () => {
    render(<Hello />);
    expect(screen.getByText("hello react")).toBeInTheDocument();
  });
});
