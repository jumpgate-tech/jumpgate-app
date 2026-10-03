import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import * as theme from "../../theme";
import { SettingsSheet } from "./SettingsSheet";

vi.mock("../../theme", async () => {
  const actual = await vi.importActual<typeof import("../../theme")>("../../theme");
  return { ...actual, getThemePref: vi.fn(() => "system"), setThemePref: vi.fn() };
});

beforeEach(() => {
  vi.mocked(theme.setThemePref).mockReset();
});
afterEach(cleanup);

describe("SettingsSheet", () => {
  it("persists the picked appearance via setThemePref and moves the active pill", () => {
    render(<SettingsSheet actions={["stop"]} busy={null} onWipe={() => {}} onClose={() => {}} />);
    const dark = screen.getByRole("button", { name: "Dark" });
    fireEvent.click(dark);
    expect(theme.setThemePref).toHaveBeenCalledWith("dark");
    expect(dark).toHaveClass("active");
  });

  it("hides Wipe when the server does not list the wipe action", () => {
    render(<SettingsSheet actions={["stop", "restart"]} busy={null} onWipe={() => {}} onClose={() => {}} />);
    expect(screen.queryByRole("button", { name: /Wipe gateway/ })).not.toBeInTheDocument();
    expect(screen.queryByText("Danger zone")).not.toBeInTheDocument();
  });

  it("shows Wipe and fires onWipe when the server lists wipe", () => {
    const onWipe = vi.fn();
    render(<SettingsSheet actions={["stop", "wipe"]} busy={null} onWipe={onWipe} onClose={() => {}} />);
    const wipe = screen.getByRole("button", { name: /Wipe gateway/ });
    expect(wipe).toBeInTheDocument();
    wipe.click();
    expect(onWipe).toHaveBeenCalled();
  });

  it("disables Wipe while a lifecycle action is busy", () => {
    render(<SettingsSheet actions={["wipe"]} busy={"wipe"} onWipe={() => {}} onClose={() => {}} />);
    expect(screen.getByRole("button", { name: /Wipe gateway/ })).toBeDisabled();
  });
});
