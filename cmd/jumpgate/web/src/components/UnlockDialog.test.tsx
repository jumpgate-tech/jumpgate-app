import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import { UnlockDialog } from "./UnlockDialog";

const fetchMock = vi.fn();

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function renderDialog() {
  const onClose = vi.fn();
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  render(<UnlockDialog onClose={onClose} />, { wrapper });
  return onClose;
}

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("UnlockDialog", () => {
  it("sends the passphrase base64 and clears the field", async () => {
    fetchMock.mockResolvedValue(json(200, { state: "unlocked", slots: ["passphrase"] }));
    const onClose = renderDialog();
    const input = screen.getByLabelText("Passphrase") as HTMLInputElement;
    expect(input.type).toBe("password");
    fireEvent.change(input, { target: { value: "päss" } });
    fireEvent.click(screen.getByRole("button", { name: "Unlock" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    const [path, init] = fetchMock.mock.calls[0];
    expect(path).toBe("/api/vault/unlock");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body)).toEqual({ passphrase: "cMOkc3M=" });
    expect(input.value).toBe("");
    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it("shows wrong passphrase without closing", async () => {
    fetchMock.mockResolvedValue(json(403, { error: "wrong passphrase", code: "wrong_passphrase" }));
    const onClose = renderDialog();
    const input = screen.getByLabelText("Passphrase") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "nope" } });
    fireEvent.click(screen.getByRole("button", { name: "Unlock" }));
    expect(await screen.findByText("Wrong passphrase")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(input.value).toBe("");
    expect(onClose).not.toHaveBeenCalled();
  });
});
