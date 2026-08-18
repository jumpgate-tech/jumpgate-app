import { render, screen, cleanup, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import * as api from "../../api";
import { ApiError } from "../../api";
import { KeysSection } from "./KeysSection";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return { ...actual, listKeys: vi.fn(), createKey: vi.fn(), revokeKey: vi.fn() };
});

const listKeys = api.listKeys as unknown as ReturnType<typeof vi.fn>;
const createKey = api.createKey as unknown as ReturnType<typeof vi.fn>;
const revokeKey = api.revokeKey as unknown as ReturnType<typeof vi.fn>;

function open() {
  render(<KeysSection gid="g1" open onToggle={() => {}} />);
}

beforeEach(() => {
  vi.clearAllMocks();
  listKeys.mockResolvedValue([]);
  createKey.mockResolvedValue({ id: "k_new", key: "jg_freshsecret" });
  revokeKey.mockResolvedValue({ status: "revoked" });
});
afterEach(cleanup);

describe("KeysSection", () => {
  it("lists keys with their status", async () => {
    listKeys.mockResolvedValue([
      { id: "k1", label: "prod", disabled: false, allow_trace: false, credit_exempt: false, created_at: 1 },
      { id: "k2", label: "old", disabled: true, allow_trace: false, credit_exempt: false, created_at: 2 },
    ]);
    open();

    await waitFor(() => expect(screen.getByText("k1")).toBeTruthy());
    expect(screen.getByText("active")).toBeTruthy();
    expect(screen.getByText("revoked")).toBeTruthy();
  });

  // The raw key is shown once and never again, so the warning that says so is
  // as load-bearing as the value itself.
  it("shows a freshly minted key once, with a copy-it-now warning", async () => {
    open();
    await waitFor(() => expect(listKeys).toHaveBeenCalled());

    fireEvent.click(screen.getByText("Issue key"));

    await waitFor(() => expect(screen.getByText("jg_freshsecret")).toBeTruthy());
    expect(screen.getByText(/never shown again/i)).toBeTruthy();
  });

  // A revoked key must not offer a second Revoke button — the action is spent.
  it("offers Revoke only on an active key", async () => {
    listKeys.mockResolvedValue([
      { id: "k1", label: "prod", disabled: false, allow_trace: false, credit_exempt: false, created_at: 1 },
      { id: "k2", label: "old", disabled: true, allow_trace: false, credit_exempt: false, created_at: 2 },
    ]);
    open();

    await waitFor(() => expect(screen.getByText("k1")).toBeTruthy());
    expect(screen.getAllByText("Revoke")).toHaveLength(1);
  });

  it("revokes through the API and refreshes", async () => {
    listKeys.mockResolvedValue([
      { id: "k1", label: "prod", disabled: false, allow_trace: false, credit_exempt: false, created_at: 1 },
    ]);
    open();

    await waitFor(() => expect(screen.getByText("k1")).toBeTruthy());
    fireEvent.click(screen.getByText("Revoke"));

    await waitFor(() => expect(revokeKey).toHaveBeenCalledWith("g1", "k1"));
  });

  // A gateway that sells no keys has no store. That is configuration, not
  // failure, so it must not read as an error.
  it("reads a 501 as 'no key store', not as an error", async () => {
    listKeys.mockRejectedValue(new ApiError(501, "no key store"));
    open();

    await waitFor(() => expect(screen.getByText(/no key store/i)).toBeTruthy());
    expect(screen.queryByText(/Issue key/)).toBeNull();
  });

  // A real failure IS an error and must be shown, not swallowed into an empty
  // list that would read as "this gateway has no keys".
  it("surfaces a real failure", async () => {
    listKeys.mockRejectedValue(new ApiError(502, "the key store did not answer"));
    open();

    await waitFor(() => expect(screen.getByText(/did not answer/i)).toBeTruthy());
  });
});
