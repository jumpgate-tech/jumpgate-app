import { render, screen, cleanup, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import type { GatewayCapabilities, TlsVerification } from "../../api";
import { NetworkView } from "./NetworkView";
import { makeGateway } from "./fixtures";

afterEach(cleanup);

const handlers = {
  onBack: () => {},
  onOpenEndpoint: () => {},
  onAddEndpoint: () => {},
  onRemoveNetwork: () => {},
  onVerifyTls: () => Promise.resolve({} as TlsVerification),
  onRecheck: () => {},
};

describe("NetworkView", () => {
  it("renders the gateway URL, its endpoints, capabilities and health", () => {
    const caps: GatewayCapabilities = {
      at: "now",
      endpoints: [{ upstream: "public-1-1", chainId: 1, reachable: true, capabilities: [] }],
    };
    render(
      <NetworkView
        gw={makeGateway()}
        chainId={1}
        health={undefined}
        caps={caps}
        capsBusy={false}
        capsErr={null}
        busy={null}
        error={null}
        {...handlers}
      />,
    );
    expect(screen.getByText("https://valve.local/1")).toBeInTheDocument();
    expect(screen.getByText("publicnode")).toBeInTheDocument();
    expect(screen.getByText("Endpoints · 1")).toBeInTheDocument();
    // reachable=true folds to HTTP supported.
    expect(screen.getByText("HTTP").closest(".p-capitem")).toHaveClass("lit");
    expect(screen.getByText("Healthy")).toBeInTheDocument();
    expect(screen.getByText("Remove network")).toBeInTheDocument();
  });

  it("runs the live TLS check when the lock is clicked", async () => {
    const onVerifyTls = vi.fn(() => Promise.resolve({ ok: true, at: "2026-08-02T00:00:00Z" } as TlsVerification));
    render(
      <NetworkView
        gw={makeGateway()}
        chainId={1}
        health={undefined}
        caps={undefined}
        capsBusy={false}
        capsErr={null}
        busy={null}
        error={null}
        {...handlers}
        onVerifyTls={onVerifyTls}
      />,
    );
    const lock = document.querySelector('.p-ic[title="Verify HTTPS now"]') as HTMLElement;
    expect(lock).toBeTruthy();
    lock.click();
    await waitFor(() => expect(onVerifyTls).toHaveBeenCalled());
  });

  it("disables add-endpoint and remove-network while a lifecycle action is in flight", () => {
    const onAddEndpoint = vi.fn();
    const onRemoveNetwork = vi.fn();
    render(
      <NetworkView
        gw={makeGateway()}
        chainId={1}
        health={undefined}
        caps={undefined}
        capsBusy={false}
        capsErr={null}
        busy={"create"}
        error={null}
        {...handlers}
        onAddEndpoint={onAddEndpoint}
        onRemoveNetwork={onRemoveNetwork}
      />,
    );
    const add = screen.getByText("Add endpoint").closest(".p-row") as HTMLElement;
    const remove = screen.getByText("Remove network").closest(".p-remove") as HTMLElement;
    expect(add).toHaveClass("p-disabled");
    expect(remove).toHaveClass("p-disabled");
    add.click();
    remove.click();
    expect(onAddEndpoint).not.toHaveBeenCalled();
    expect(onRemoveNetwork).not.toHaveBeenCalled();
  });

  it("shows each endpoint's provider, its distinguishing URL, and a source icon for a local upstream", () => {
    const gw = makeGateway();
    gw.networks![0].upstreams = [
      { id: "mine", kind: "external", endpoint: "https://rpc.publicnode.com/pulsechain", label: "publicnode", local: true, recentOnly: false, actions: null },
    ];
    const caps: GatewayCapabilities = {
      at: "now",
      endpoints: [
        { upstream: "mine", chainId: 1, reachable: true, capabilities: [{ key: "archive", label: "Archive", status: "supported" }] },
      ],
    };
    render(
      <NetworkView gw={gw} chainId={1} health={undefined} caps={caps} capsBusy={false} capsErr={null} busy={null} error={null} {...handlers} />,
    );
    expect(screen.getByText("publicnode")).toBeInTheDocument();
    // scheme dropped, host + path kept
    expect(screen.getByText("rpc.publicnode.com/pulsechain")).toBeInTheDocument();
    // per-endpoint capability meter is drawn on the row
    expect(document.querySelector(".p-eprow .p-caps")).toBeTruthy();
    // a local ("yours") upstream carries the server source icon; a public one would not
    expect(document.querySelector('.p-eprow .p-src use[href="#p-server"]')).toBeTruthy();
  });

  it("shows 'no longer configured' when the chain has gone", () => {
    render(
      <NetworkView
        gw={makeGateway()}
        chainId={9999}
        health={undefined}
        caps={undefined}
        capsBusy={false}
        capsErr={null}
        busy={null}
        error={null}
        {...handlers}
      />,
    );
    expect(screen.getByText("This network is no longer configured.")).toBeInTheDocument();
  });
});
