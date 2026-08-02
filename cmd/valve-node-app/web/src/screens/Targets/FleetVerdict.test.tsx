import { render, screen, cleanup } from "@testing-library/react";
import { describe, it, expect, afterEach } from "vitest";
import type { Target, Catalog } from "../../api";
import { FleetVerdict } from "./FleetVerdict";

const CATALOG: Catalog = {
  networks: [
    {
      ChainID: 1,
      Name: "Ethereum",
      CheckpointURL: "",
      ExecClients: [],
      BeaconClients: [],
      LearnURL: "",
      SnapshotSizeTB: 0,
      SyncLabel: "",
      GenesisSyncLabel: "",
    },
  ],
  clients: [],
};

afterEach(cleanup);

describe("FleetVerdict", () => {
  it("shows the ok pill and sentence, with no machine links, when everything is wired", () => {
    const targets: Target[] = [
      {
        id: "t1",
        mode: "local",
        wire: { ChainID: 1, ExecID: "reth", BeaconID: "lighthouse", DataDir: "", JWTPath: "", Archive: false },
      },
    ];
    render(<FleetVerdict targets={targets} catalog={CATALOG} />);

    expect(screen.getByText("OK")).toHaveClass("badge-ok");
    expect(screen.getByText(/All 1 machine healthy — Ethereum/)).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("shows the attention pill and a link per implicated machine when one needs setup", () => {
    const targets: Target[] = [{ id: "t1", mode: "local" }];
    render(<FleetVerdict targets={targets} catalog={CATALOG} />);

    expect(screen.getByText("Attention")).toHaveClass("badge-warn");
    expect(screen.getByText("1 machine still needs setup.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "t1" })).toHaveAttribute("href", "#/setup/t1");
  });

  it("shows 'No machines yet.' with no implicated links when the fleet is empty", () => {
    render(<FleetVerdict targets={[]} catalog={CATALOG} />);

    expect(screen.getByText("No machines yet.")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
