import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import { AddNetworkPicker } from "./AddNetworkModal";

afterEach(cleanup);

// A stand-in for the full chainlist catalogue (the long tail behind the curated
// head). Kept small — the curated set comes from viem via the component itself.
const catalogue = [
  { chainId: 42161, name: "Arbitrum One" },
  { chainId: 10, name: "OP Mainnet" },
];

function setup(props: Partial<Parameters<typeof AddNetworkPicker>[0]> = {}) {
  const onPick = vi.fn();
  const onCancel = vi.fn();
  render(
    <AddNetworkPicker catalogue={catalogue} presentChainIds={[]} onPick={onPick} onCancel={onCancel} {...props} />,
  );
  return { onPick, onCancel };
}

const search = () => screen.getByPlaceholderText(/Search by name/);

describe("AddNetworkPicker", () => {
  it("shows the curated suggestions and hides the long tail until you search", () => {
    setup();
    expect(screen.getByText("Suggested")).toBeInTheDocument();
    expect(screen.getByText("Ethereum")).toBeInTheDocument();
    expect(screen.getByText("PulseChain")).toBeInTheDocument();
    expect(screen.getByText("Sepolia")).toBeInTheDocument();
    // both testnets carry the tag
    expect(screen.getAllByText("testnet")).toHaveLength(2);
    // a catalogue-only chain is not offered until the operator searches
    expect(screen.queryByText("Arbitrum One")).not.toBeInTheDocument();
  });

  it("pins caller-supplied extras after the curated head", () => {
    setup({ extraPinned: [{ chainId: 1337, name: "Devnet", testnet: true }] });
    expect(screen.getByText("Devnet")).toBeInTheDocument();
    const names = Array.from(document.querySelectorAll(".p-anm-name")).map((n) => n.textContent ?? "");
    expect(names[names.length - 1]).toContain("Devnet"); // last, after the curated chains
  });

  it("excludes chains already fronted", () => {
    setup({ presentChainIds: [1] });
    expect(screen.queryByText("Ethereum")).not.toBeInTheDocument();
    expect(screen.getByText("PulseChain")).toBeInTheDocument();
  });

  it("filters the full catalogue by name when searching", () => {
    setup();
    fireEvent.change(search(), { target: { value: "arb" } });
    expect(screen.getByText("Arbitrum One")).toBeInTheDocument();
    // curated non-matches drop out, and the "Suggested" heading is only for the
    // empty state
    expect(screen.queryByText("Ethereum")).not.toBeInTheDocument();
    expect(screen.queryByText("Suggested")).not.toBeInTheDocument();
  });

  it("matches a curated chain by its id, with no custom row when it's listed", () => {
    setup();
    fireEvent.change(search(), { target: { value: "369" } });
    expect(screen.getByText("PulseChain")).toBeInTheDocument();
    expect(screen.queryByText(/Add chain 369/)).not.toBeInTheDocument();
  });

  it("picks a chain id on row click", () => {
    const { onPick } = setup();
    fireEvent.click(screen.getByText("PulseChain"));
    expect(onPick).toHaveBeenCalledWith(369);
  });

  it("offers a custom add for a numeric id that matches nothing, and picks it", () => {
    const { onPick } = setup();
    fireEvent.change(search(), { target: { value: "424242" } });
    fireEvent.click(screen.getByText("Add chain 424242"));
    expect(onPick).toHaveBeenCalledWith(424242);
  });

  it("cancels", () => {
    const { onCancel } = setup();
    fireEvent.click(screen.getByText("Cancel"));
    expect(onCancel).toHaveBeenCalled();
  });

  it("shows a chain logo, and falls back to a placeholder glyph when it fails to load", () => {
    setup();
    const slot = document.querySelector(".p-anm-logo")!;
    const img = slot.querySelector("img") as HTMLImageElement;
    expect(img).toBeTruthy();
    expect(img.getAttribute("src")).toContain("gib.show/image/");
    // A 404 / offline load error swaps the image for the globe glyph, keeping
    // the slot (and therefore row alignment) intact.
    fireEvent.error(img);
    const after = document.querySelector(".p-anm-logo")!;
    expect(after.querySelector("img")).toBeNull();
    expect(after.querySelector("use")).toBeTruthy();
  });
});
