import { render, screen, cleanup } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import { ListView } from "./ListView";
import { makeGateway } from "./fixtures";

afterEach(cleanup);

const noop = () => {};
const handlers = {
  onSetup: noop,
  onPower: noop,
  onChip: noop,
  onOpenSettings: noop,
  onOpenNetwork: noop,
  onAddNetwork: noop,
};

describe("ListView", () => {
  it("renders one row per configured network plus the add-a-network row", () => {
    render(<ListView gw={makeGateway()} health={undefined} busy={null} actionErr={null} setupLog={[]} {...handlers} />);
    expect(screen.getByText("Ethereum")).toBeInTheDocument();
    expect(screen.getByText("PulseChain")).toBeInTheDocument();
    expect(screen.getByText("Add a network")).toBeInTheDocument();
    // Running gateway → the summary reads "2 networks served" (shown in both
    // the header and the power band's sub-line).
    expect(screen.getAllByText("2 networks served").length).toBeGreaterThan(0);
  });

  it("drills into a network on row click", () => {
    const onOpenNetwork = vi.fn();
    render(
      <ListView
        gw={makeGateway()}
        health={undefined}
        busy={null}
        actionErr={null}
        setupLog={[]}
        {...handlers}
        onOpenNetwork={onOpenNetwork}
      />,
    );
    screen.getByText("Ethereum").click();
    expect(onOpenNetwork).toHaveBeenCalledWith(1);
  });

  it("shows the one-click 'Set up my endpoint' hero when there is no gateway", () => {
    render(<ListView gw={null} health={undefined} busy={null} actionErr={null} setupLog={[]} {...handlers} />);
    expect(screen.getByText("Set up my endpoint")).toBeInTheDocument();
    expect(screen.queryByText("Add a network")).not.toBeInTheDocument();
  });

  it("fires onSetup from the empty-state power button, and narrates the setup log", () => {
    const onSetup = vi.fn();
    const { container } = render(
      <ListView
        gw={null}
        health={undefined}
        busy={null}
        actionErr={null}
        setupLog={["Preparing your endpoint…", "Creating the gateway…"]}
        {...handlers}
        onSetup={onSetup}
      />,
    );
    (container.querySelector(".p-emptybtn") as HTMLButtonElement).click();
    expect(onSetup).toHaveBeenCalled();
    expect(screen.getByText("Creating the gateway…")).toBeInTheDocument();
  });
});
