import { render, screen, cleanup } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import { HashRouter } from "react-router-dom";
import { activeNav, App } from "./App";

vi.mock("./screens/Panel/Panel", () => ({
  Panel: () => <div>panel-screen</div>,
}));
vi.mock("./rpc", () => ({
  renderRPC: vi.fn((el: HTMLElement) => {
    el.textContent = "rpc-screen";
  }),
}));
vi.mock("./targets", () => ({
  renderTargets: vi.fn((el: HTMLElement) => {
    el.textContent = "targets-screen";
  }),
}));
vi.mock("./screens/Settings/Settings", () => ({
  Settings: () => <div>settings-screen</div>,
}));
vi.mock("./machine", () => ({
  renderMachine: vi.fn((el: HTMLElement, id: string) => {
    el.textContent = `machine-screen:${id}`;
  }),
}));
vi.mock("./screens/Security/Security", async () => {
  const { useParams } = await import("react-router-dom");
  return {
    Security: () => {
      const { id } = useParams<{ id: string }>();
      return <div>security-screen:{id}</div>;
    },
  };
});
vi.mock("./screens/Diagnostics/Diagnostics", async () => {
  const { useParams } = await import("react-router-dom");
  return {
    Diagnostics: () => {
      const { id } = useParams<{ id: string }>();
      return <div>diag-screen:{id}</div>;
    },
  };
});
vi.mock("./analytics", () => ({
  renderAnalytics: vi.fn((el: HTMLElement, id: string) => {
    el.textContent = `analytics-screen:${id}`;
  }),
}));

describe("activeNav", () => {
  it("maps machine→targets and home/panel→rpc, else identity", () => {
    expect(activeNav("machine")).toBe("targets");
    expect(activeNav("home")).toBe("rpc");
    expect(activeNav("panel")).toBe("rpc");
    expect(activeNav("settings")).toBe("settings");
  });
});

describe("App routing", () => {
  afterEach(() => {
    cleanup();
    window.location.hash = "";
  });

  function renderAt(hash: string) {
    window.location.hash = hash;
    return render(
      <HashRouter>
        <App />
      </HashRouter>,
    );
  }

  it("renders the nav labels", () => {
    renderAt("#/rpc");
    expect(screen.getByText("RPC")).toBeInTheDocument();
    expect(screen.getByText("Machines")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });

  it("empty hash renders the panel screen with RPC active", async () => {
    renderAt("");
    expect(await screen.findByText("panel-screen")).toBeInTheDocument();
    expect(screen.getByText("RPC")).toHaveClass("active");
  });

  it("#/rpc renders the RPC screen with RPC active", async () => {
    renderAt("#/rpc");
    expect(await screen.findByText("rpc-screen")).toBeInTheDocument();
    expect(screen.getByText("RPC")).toHaveClass("active");
  });

  it("#/targets renders the targets screen with Machines active", async () => {
    renderAt("#/targets");
    expect(await screen.findByText("targets-screen")).toBeInTheDocument();
    expect(screen.getByText("Machines")).toHaveClass("active");
  });

  it("#/settings renders the settings screen with Settings active", async () => {
    renderAt("#/settings");
    expect(await screen.findByText("settings-screen")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toHaveClass("active");
  });

  it("#/machine/:id renders the machine screen with Machines active", async () => {
    renderAt("#/machine/abc");
    expect(await screen.findByText("machine-screen:abc")).toBeInTheDocument();
    expect(screen.getByText("Machines")).toHaveClass("active");
  });

  it("#/machine with no id redirects to targets", async () => {
    renderAt("#/machine");
    expect(await screen.findByText("targets-screen")).toBeInTheDocument();
  });

  it("#/security/:id renders the security screen", async () => {
    renderAt("#/security/abc");
    expect(await screen.findByText("security-screen:abc")).toBeInTheDocument();
  });

  it("#/diag/:id renders the diagnostics screen", async () => {
    renderAt("#/diag/abc");
    expect(await screen.findByText("diag-screen:abc")).toBeInTheDocument();
  });

  it("#/analytics/:id renders the analytics screen", async () => {
    renderAt("#/analytics/gid1");
    expect(await screen.findByText("analytics-screen:gid1")).toBeInTheDocument();
  });

  it("#/analytics with no id redirects to rpc", async () => {
    renderAt("#/analytics");
    expect(await screen.findByText("rpc-screen")).toBeInTheDocument();
  });

  it("#/security with no id redirects to targets", async () => {
    renderAt("#/security");
    expect(await screen.findByText("targets-screen")).toBeInTheDocument();
  });

  it("#/diag with no id redirects to targets", async () => {
    renderAt("#/diag");
    expect(await screen.findByText("targets-screen")).toBeInTheDocument();
  });

  it.each(["setup", "dash", "logs", "services"])("#/%s/:id redirects to machine/:id", async (p) => {
    renderAt(`#/${p}/xyz`);
    expect(await screen.findByText("machine-screen:xyz")).toBeInTheDocument();
  });

  it.each(["setup", "dash", "logs", "services"])("#/%s with no id redirects to targets", async (p) => {
    renderAt(`#/${p}`);
    expect(await screen.findByText("targets-screen")).toBeInTheDocument();
  });

  it("unknown route falls back to the panel screen", async () => {
    renderAt("#/nope");
    expect(await screen.findByText("panel-screen")).toBeInTheDocument();
  });
});
