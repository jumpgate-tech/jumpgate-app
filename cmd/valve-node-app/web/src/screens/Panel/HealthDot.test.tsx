import { render, screen, cleanup } from "@testing-library/react";
import { describe, it, expect, afterEach } from "vitest";
import { HealthDot } from "./HealthDot";

afterEach(cleanup);

// HealthDot's whole job is to turn panelModel's healthClass into the dot's
// class — stillness is health, faster motion a worse slow rate.
describe("HealthDot", () => {
  it("is 'off' when the gateway is not running", () => {
    const { container } = render(<HealthDot running={false} serviceable={true} />);
    expect(container.querySelector(".p-dot")).toHaveClass("off");
  });

  it("is 'stable' when running, serviceable and rarely slow", () => {
    const { container } = render(<HealthDot running={true} serviceable={true} slowRate={0.05} />);
    expect(container.querySelector(".p-dot")).toHaveClass("stable");
  });

  it("is 'occasional' at a 10–40% slow rate", () => {
    const { container } = render(<HealthDot running={true} serviceable={true} slowRate={0.2} />);
    expect(container.querySelector(".p-dot")).toHaveClass("occasional");
  });

  it("is 'frequent' above 40% slow, or when unserviceable while running", () => {
    const { container: a } = render(<HealthDot running={true} serviceable={true} slowRate={0.6} />);
    expect(a.querySelector(".p-dot")).toHaveClass("frequent");
    const { container: b } = render(<HealthDot running={true} serviceable={false} />);
    expect(b.querySelector(".p-dot")).toHaveClass("frequent");
  });

  // a11y: the dot is colour + motion only, so it must carry a text state word
  // (role="img" + aria-label) that a screen reader can announce.
  it("exposes its state as an accessible name, not colour/motion alone", () => {
    render(<HealthDot running={true} serviceable={true} slowRate={0.05} />);
    expect(screen.getByRole("img", { name: "Healthy" })).toBeInTheDocument();
    cleanup();
    render(<HealthDot running={true} serviceable={true} slowRate={0.6} />);
    expect(screen.getByRole("img", { name: "Degraded" })).toBeInTheDocument();
    cleanup();
    render(<HealthDot running={false} serviceable={true} />);
    expect(screen.getByRole("img", { name: "Stopped" })).toBeInTheDocument();
    cleanup();
    render(<HealthDot running={true} serviceable={false} />);
    expect(screen.getByRole("img", { name: "Unserviceable" })).toBeInTheDocument();
  });
});
