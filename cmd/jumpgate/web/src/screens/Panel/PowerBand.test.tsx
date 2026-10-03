import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import { masterState } from "../../panelModel";
import { PowerBand, primaryAction } from "./PowerBand";
import { makeGateway, STATUS_RUNNING } from "./fixtures";

afterEach(cleanup);

describe("primaryAction", () => {
  it("stops a running gateway, else starts, else creates, and never offers an unlisted action", () => {
    const running = makeGateway({ actions: ["stop", "restart"] });
    expect(primaryAction(running, masterState(running))).toBe("stop");

    const stopped = makeGateway({
      status: { ...STATUS_RUNNING, State: "created-but-stopped" },
      actions: ["start", "recreate"],
    });
    expect(primaryAction(stopped, masterState(stopped))).toBe("start");

    const fresh = makeGateway({
      status: { ...STATUS_RUNNING, State: "not-created" },
      actions: ["create"],
    });
    expect(primaryAction(fresh, masterState(fresh))).toBe("create");

    const blocked = makeGateway({ blocked: "Docker is not reachable", actions: [] });
    expect(primaryAction(blocked, masterState(blocked))).toBeNull();
  });
});

describe("PowerBand", () => {
  it("renders the master label and the chips for every action minus primary and wipe", () => {
    const gw = makeGateway({ actions: ["stop", "restart", "recreate", "wipe"] });
    render(
      <PowerBand gw={gw} master={masterState(gw)} busy={null} actionErr={null} onPower={() => {}} onChip={() => {}} />,
    );
    // Running → label "Running", sub "2 networks served".
    expect(screen.getByText("Running")).toBeInTheDocument();
    expect(screen.getByText("2 networks served")).toBeInTheDocument();
    // primary is "stop" and wipe is hidden → chips are Restart + Recreate only.
    expect(screen.getByRole("button", { name: /Restart/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Recreate/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Stop/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Wipe/ })).not.toBeInTheDocument();
  });

  it("fires onChip with the raw action name", () => {
    const gw = makeGateway({ actions: ["stop", "restart"] });
    const onChip = vi.fn();
    render(
      <PowerBand gw={gw} master={masterState(gw)} busy={null} actionErr={null} onPower={() => {}} onChip={onChip} />,
    );
    fireEvent.click(screen.getByRole("button", { name: /Restart/ }));
    expect(onChip).toHaveBeenCalledWith("restart");
  });

  it("disables chips and surfaces an action error while busy", () => {
    const gw = makeGateway({ actions: ["stop", "restart"] });
    render(
      <PowerBand
        gw={gw}
        master={masterState(gw)}
        busy={"restart"}
        actionErr={"restart failed: boom"}
        onPower={() => {}}
        onChip={() => {}}
      />,
    );
    expect(screen.getByRole("button", { name: /Restart/ })).toBeDisabled();
    expect(screen.getByText("restart failed: boom")).toBeInTheDocument();
  });
});
