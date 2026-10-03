import { render, cleanup } from "@testing-library/react";
import { describe, it, expect, afterEach } from "vitest";
import { Footer } from "./Footer";
import { LEARN_ROOT } from "../ui";

afterEach(cleanup);

describe("Footer", () => {
  it("always renders the base learn link", () => {
    const { getByText } = render(<Footer />);
    expect(getByText(/Learn how this works/)).toHaveAttribute("href", LEARN_ROOT);
  });

  it("adds a context link when given a label and a distinct url", () => {
    const { getByText } = render(<Footer contextLabel="Ethereum" contextUrl="https://learn.valve.city/eth" />);
    expect(getByText("Ethereum")).toHaveAttribute("href", "https://learn.valve.city/eth");
  });

  it("omits the context link when the context url equals the base learn link", () => {
    const { queryByText } = render(<Footer contextLabel="Ethereum" contextUrl={LEARN_ROOT} />);
    expect(queryByText("Ethereum")).not.toBeInTheDocument();
  });
});
