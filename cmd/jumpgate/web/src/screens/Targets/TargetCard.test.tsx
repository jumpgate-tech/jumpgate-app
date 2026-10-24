import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { Catalog, FleetRow, Target } from "../../api";
import { TargetCard } from "./TargetCard";

const CATALOG: Catalog = { networks: [], clients: [] };
const BOX: Target = { id: "box-1", mode: "ssh", ssh: { Host: "203.0.113.10", User: "root", KeyPath: "/k" } };

function renderCard(fleetRow?: FleetRow) {
  return render(<TargetCard target={BOX} catalog={CATALOG} canRunNode hostOS="linux" onRemove={() => {}} fleetRow={fleetRow} />);
}

afterEach(cleanup);

describe("TargetCard alerts", () => {
  it("shows the alert count and the first message", () => {
    renderCard({ id: "box-1", reachable: true, alertCount: 2, alertMessage: "Disk is 93% full." });
    expect(screen.getByText("2 alerts")).toBeTruthy();
    expect(screen.getByText(/Disk is 93% full\./)).toBeTruthy();
  });

  it("says 1 alert, singular", () => {
    renderCard({ id: "box-1", reachable: true, alertCount: 1, alerts: [{ key: "no-peers", message: "No peers.", since: "" }] });
    expect(screen.getByText("1 alert")).toBeTruthy();
    expect(screen.getByText(/No peers\./)).toBeTruthy();
  });

  it("shows nothing without alerts", () => {
    renderCard({ id: "box-1", reachable: true, alertCount: 0 });
    expect(screen.queryByText(/alert/)).toBeNull();
  });
});
