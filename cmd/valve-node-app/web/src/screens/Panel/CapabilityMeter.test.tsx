import { render, screen, cleanup } from "@testing-library/react";
import { describe, it, expect, afterEach } from "vitest";
import { capabilityCells } from "../../panelModel";
import { CapabilityDots, CapabilityMeter, CapsBand } from "./CapabilityMeter";

afterEach(cleanup);

describe("CapabilityMeter", () => {
  it("lights the supported cells and marks archive hot when lit", () => {
    render(<CapabilityMeter statuses={{ http: "supported", archive: "supported" }} />);
    const http = screen.getByText("HTTP").closest(".p-capitem");
    const archive = screen.getByText("Archive").closest(".p-capitem");
    const ws = screen.getByText("WS").closest(".p-capitem");
    expect(http).toHaveClass("lit");
    expect(archive).toHaveClass("lit");
    expect(ws).not.toHaveClass("lit");
  });

  it("renders all four cells in fixed order", () => {
    render(<CapabilityMeter statuses={{}} />);
    // The cell's visible label is its direct text node; the state word lives in
    // a separate visually-hidden <span>, so read the label node specifically.
    const labels = screen
      .getAllByText(/^(HTTP|WS|Archive|Trace)$/)
      .map((n) => Array.from(n.childNodes).find((c) => c.nodeType === Node.TEXT_NODE)?.textContent);
    expect(labels).toEqual(["HTTP", "WS", "Archive", "Trace"]);
  });

  // a11y: lit/unlit is otherwise colour-only, so a visually-hidden state word
  // carries it to a screen reader.
  it("adds a visually-hidden state word to each cell", () => {
    render(<CapabilityMeter statuses={{ http: "supported" }} />);
    expect(screen.getByText(": supported")).toBeInTheDocument();
    expect(screen.getAllByText(": unavailable")).toHaveLength(3);
  });
});

describe("CapabilityDots", () => {
  it("renders one icon per cell (the dim list-row meter)", () => {
    const { container } = render(<CapabilityDots cells={capabilityCells({})} />);
    expect(container.querySelectorAll(".p-caps .p-i")).toHaveLength(4);
  });

  // a11y: the glyphs are icon + colour only, so each carries a name + state
  // accessible name a screen reader can read.
  it("labels each cell with its name and state", () => {
    render(<CapabilityDots cells={capabilityCells({ http: "supported" })} />);
    expect(screen.getByRole("img", { name: "HTTP: supported" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Trace: unavailable" })).toBeInTheDocument();
  });
});

describe("CapsBand", () => {
  it("tells the operator to start the gateway when it is stopped", () => {
    render(<CapsBand statuses={{}} busy={false} err={"socket refused"} hasData={false} running={false} />);
    expect(screen.getByText(/Start the gateway/)).toBeInTheDocument();
    // The raw probe error is NOT shown for a stopped gateway.
    expect(screen.queryByText(/Couldn't check capabilities/)).not.toBeInTheDocument();
  });

  it("shows 'probing…' only on the first fetch (busy, no data yet)", () => {
    render(<CapsBand statuses={{}} busy={true} err={null} hasData={false} running={true} />);
    expect(screen.getByText("probing…")).toBeInTheDocument();
  });

  it("shows the probe error when it failed with no data", () => {
    render(<CapsBand statuses={{}} busy={false} err={"socket refused"} hasData={false} running={true} />);
    expect(screen.getByText(/Couldn't check capabilities/)).toHaveTextContent("socket refused");
  });

  it("shows the meter (previous verdict) once there is data, even while refetching", () => {
    render(<CapsBand statuses={{ http: "supported" }} busy={true} err={null} hasData={true} running={true} />);
    expect(screen.queryByText("probing…")).not.toBeInTheDocument();
    expect(screen.getByText("HTTP").closest(".p-capitem")).toHaveClass("lit");
  });
});
