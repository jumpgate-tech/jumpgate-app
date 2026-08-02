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
    const labels = screen.getAllByText(/HTTP|WS|Archive|Trace/).map((n) => n.textContent);
    expect(labels).toEqual(["HTTP", "WS", "Archive", "Trace"]);
  });
});

describe("CapabilityDots", () => {
  it("renders one icon per cell (the dim list-row meter)", () => {
    const { container } = render(<CapabilityDots cells={capabilityCells({})} />);
    expect(container.querySelectorAll(".p-caps .p-i")).toHaveLength(4);
  });
});

describe("CapsBand", () => {
  it("shows 'probing…' only on the first fetch (busy, no data yet)", () => {
    render(<CapsBand statuses={{}} busy={true} err={null} hasData={false} />);
    expect(screen.getByText("probing…")).toBeInTheDocument();
  });

  it("shows the probe error when it failed with no data", () => {
    render(<CapsBand statuses={{}} busy={false} err={"socket refused"} hasData={false} />);
    expect(screen.getByText(/Couldn't check capabilities/)).toHaveTextContent("socket refused");
  });

  it("shows the meter (previous verdict) once there is data, even while refetching", () => {
    render(<CapsBand statuses={{ http: "supported" }} busy={true} err={null} hasData={true} />);
    expect(screen.queryByText("probing…")).not.toBeInTheDocument();
    expect(screen.getByText("HTTP").closest(".p-capitem")).toHaveClass("lit");
  });
});
