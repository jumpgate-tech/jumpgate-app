import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Settings as SettingsType } from "../../api";
import { Settings } from "./Settings";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    getSettings: vi.fn(),
    putSettings: vi.fn(),
  };
});

const BASE_SETTINGS: SettingsType = {
  aiProvider: "",
  aiKeySet: false,
  refRpcBase: "https://ref.example/rpc",
  providerKeysSet: [],
};

function renderScreen() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return render(<Settings />, { wrapper });
}

// The top-level ai-key <input> is the only <input type="password"> that's a
// direct child of a <label> whose own text starts with "API key" (as opposed
// to a provider-key row's <label><code>NAME</code>...</label>).
function aiKeyInput(): HTMLInputElement {
  return screen.getByText("API key").closest("label")!.querySelector("input")!;
}

function providerKeyInput(name: string): HTMLInputElement {
  return screen.getByText(name).closest(".pk-row")!.querySelector("input.provider-key")!;
}

function aiProviderSelect(): HTMLSelectElement {
  return screen.getByText("AI provider").closest("label")!.querySelector("select")!;
}

beforeEach(() => {
  vi.mocked(api.getSettings).mockReset();
  vi.mocked(api.putSettings).mockReset();
});
afterEach(cleanup);

describe("Settings", () => {
  it("shows Loading… while settings are in flight", () => {
    vi.mocked(api.getSettings).mockReturnValue(new Promise(() => {}));

    renderScreen();

    expect(screen.getByText("Loading…")).toBeInTheDocument();
  });

  it("shows a failed-to-load error (String(err), matching settings.ts's own catch) on rejection", async () => {
    vi.mocked(api.getSettings).mockRejectedValue(new Error("network down"));

    renderScreen();

    expect(await screen.findByText("Failed to load settings: Error: network down")).toBeInTheDocument();
  });

  it("renders the loaded form: provider select, ai-key placeholder, ref RPC base, footer", async () => {
    vi.mocked(api.getSettings).mockResolvedValue(BASE_SETTINGS);

    renderScreen();

    expect(await screen.findByText("AI provider")).toBeInTheDocument();
    expect(aiKeyInput()).toHaveAttribute("placeholder", "no key set");
    expect(screen.getByDisplayValue("https://ref.example/rpc")).toBeInTheDocument();
    // Always the VALVE_API_KEY row, even though no key is set for it yet.
    expect(screen.getByText("VALVE_API_KEY")).toBeInTheDocument();
    expect(screen.getByText(/Learn how this works/)).toBeInTheDocument();
    // No "Clear saved key" for a key that isn't set anywhere on the form.
    expect(screen.queryByText("Clear saved key")).not.toBeInTheDocument();
  });

  it("shows the AI-key clear button only when aiKeySet, and clearing marks it touched (sends aiKey: '')", async () => {
    vi.mocked(api.getSettings).mockResolvedValue({ ...BASE_SETTINGS, aiKeySet: true });
    vi.mocked(api.putSettings).mockResolvedValue({ ...BASE_SETTINGS, aiKeySet: false });

    renderScreen();
    await screen.findByText("AI provider");

    // Only the top-level ai-key row is set; VALVE_API_KEY isn't, so it gets none.
    const clearButtons = screen.getAllByText("Clear saved key");
    expect(clearButtons).toHaveLength(1);
    fireEvent.click(clearButtons[0]);

    fireEvent.click(screen.getByText("Save"));
    await waitFor(() => expect(api.putSettings).toHaveBeenCalledWith(expect.objectContaining({ aiKey: "" })));
  });

  it("orders provider-key rows with VALVE_API_KEY first, rest sorted", async () => {
    vi.mocked(api.getSettings).mockResolvedValue({
      ...BASE_SETTINGS,
      providerKeysSet: ["INFURA_API_KEY", "VALVE_API_KEY", "ALCHEMY_API_KEY"],
    });

    renderScreen();
    await screen.findByText("AI provider");

    const codes = screen.getAllByText(/^[A-Z_]+$/).map((el) => el.textContent);
    expect(codes).toEqual(["VALVE_API_KEY", "ALCHEMY_API_KEY", "INFURA_API_KEY"]);
  });

  it("saves only touched fields: provider + trimmed ref base always, aiKey/providerKeys only if touched", async () => {
    vi.mocked(api.getSettings).mockResolvedValue(BASE_SETTINGS);
    vi.mocked(api.putSettings).mockResolvedValue(BASE_SETTINGS);

    renderScreen();
    await screen.findByText("AI provider");

    fireEvent.click(screen.getByText("Save"));

    await waitFor(() =>
      expect(api.putSettings).toHaveBeenCalledWith({
        aiProvider: "",
        refRpcBase: "https://ref.example/rpc",
      }),
    );
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
  });

  it("sends providerKeys for a touched existing row and for the add-row, but not untouched rows", async () => {
    vi.mocked(api.getSettings).mockResolvedValue({
      ...BASE_SETTINGS,
      providerKeysSet: ["VALVE_API_KEY", "ALCHEMY_API_KEY"],
    });
    vi.mocked(api.putSettings).mockResolvedValue(BASE_SETTINGS);

    renderScreen();
    await screen.findByText("AI provider");

    // Touch the existing ALCHEMY_API_KEY row; leave VALVE_API_KEY untouched.
    fireEvent.change(providerKeyInput("ALCHEMY_API_KEY"), { target: { value: "new-alchemy-key" } });

    // Fill the add-row (name + its key).
    const newNameInput = screen.getByPlaceholderText("INFURA_API_KEY");
    const newValueInput = newNameInput.closest(".pk-new")!.querySelector('input[type="password"]')!;
    fireEvent.change(newNameInput, { target: { value: "INFURA_API_KEY" } });
    fireEvent.change(newValueInput, { target: { value: "infura-secret" } });

    fireEvent.click(screen.getByText("Save"));

    await waitFor(() =>
      expect(api.putSettings).toHaveBeenCalledWith(
        expect.objectContaining({
          providerKeys: { ALCHEMY_API_KEY: "new-alchemy-key", INFURA_API_KEY: "infura-secret" },
        }),
      ),
    );
  });

  it("picking a different AI provider does not blank an already-typed ai-key (controlled-input fix)", async () => {
    vi.mocked(api.getSettings).mockResolvedValue(BASE_SETTINGS);

    renderScreen();
    await screen.findByText("AI provider");

    fireEvent.change(aiKeyInput(), { target: { value: "typed-before-provider-change" } });
    fireEvent.change(aiProviderSelect(), { target: { value: "groq" } });

    expect(aiKeyInput()).toHaveValue("typed-before-provider-change");
  });

  it("shows a save error (err.message, distinct from the load path's String(err)) and keeps the form editable", async () => {
    vi.mocked(api.getSettings).mockResolvedValue(BASE_SETTINGS);
    vi.mocked(api.putSettings).mockRejectedValue(new Error("name must match ^[A-Za-z0-9_]+$"));

    renderScreen();
    await screen.findByText("AI provider");

    fireEvent.click(screen.getByText("Save"));

    expect(await screen.findByText("name must match ^[A-Za-z0-9_]+$")).toBeInTheDocument();
    expect(screen.queryByText("Saved.")).not.toBeInTheDocument();
  });

  it("clears touched state and the ai-key field after a successful save", async () => {
    vi.mocked(api.getSettings).mockResolvedValue(BASE_SETTINGS);
    vi.mocked(api.putSettings).mockResolvedValue({ ...BASE_SETTINGS, aiKeySet: true });

    renderScreen();
    await screen.findByText("AI provider");

    fireEvent.change(aiKeyInput(), { target: { value: "sekret" } });
    fireEvent.click(screen.getByText("Save"));

    await screen.findByText("Saved.");
    // The cache was seeded with aiKeySet:true, so the field now shows the
    // "leave blank to keep" placeholder, blank, with a Clear button — not the
    // just-typed value.
    expect(aiKeyInput()).toHaveValue("");
    expect(aiKeyInput()).toHaveAttribute("placeholder", "•••••••• (leave blank to keep)");
    expect(screen.getByText("Clear saved key")).toBeInTheDocument();
  });
});
