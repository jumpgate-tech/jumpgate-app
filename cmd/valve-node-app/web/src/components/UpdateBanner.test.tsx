import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { Update } from "../api";
import { UpdateBanner } from "./UpdateBanner";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    getUpdate: vi.fn(),
  };
});

const AVAILABLE: Update = {
  current: "v0.3.5",
  latest: "v0.4.0",
  updateAvailable: true,
  releaseUrl: "https://example.test/releases/v0.4.0",
  notifyEnabled: true,
};

function renderBanner() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return render(<UpdateBanner />, { wrapper });
}

beforeEach(() => {
  vi.mocked(api.getUpdate).mockReset();
});
afterEach(cleanup);

describe("UpdateBanner", () => {
  it("shows nothing when no update is available", async () => {
    vi.mocked(api.getUpdate).mockResolvedValue({ ...AVAILABLE, updateAvailable: false });
    renderBanner();
    await waitFor(() => expect(api.getUpdate).toHaveBeenCalled());
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("stays hidden when notices are off, even if an update exists", async () => {
    vi.mocked(api.getUpdate).mockResolvedValue({ ...AVAILABLE, notifyEnabled: false });
    renderBanner();
    await waitFor(() => expect(api.getUpdate).toHaveBeenCalled());
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("shows the version and a release link when an update is available", async () => {
    vi.mocked(api.getUpdate).mockResolvedValue(AVAILABLE);
    renderBanner();
    await screen.findByRole("status");
    expect(screen.getByText("v0.4.0")).toBeInTheDocument();
    const link = screen.getByText("View release").closest("a")!;
    expect(link).toHaveAttribute("href", "https://example.test/releases/v0.4.0");
  });

  it("dismisses for the session", async () => {
    vi.mocked(api.getUpdate).mockResolvedValue(AVAILABLE);
    renderBanner();
    await screen.findByRole("status");
    fireEvent.click(screen.getByText("Dismiss"));
    expect(screen.queryByRole("status")).toBeNull();
  });
});
