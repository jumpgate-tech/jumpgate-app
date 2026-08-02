import { render, screen, cleanup } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import type { GatewayAnalytics } from "../../api";
import { EndpointView } from "./EndpointView";
import { makeGateway } from "./fixtures";

afterEach(cleanup);

const handlers = {
  onBack: () => {},
  onRename: () => {},
  onEditAddress: () => {},
  onRemove: () => {},
  onRecheck: () => {},
};

describe("EndpointView", () => {
  it("renders an external endpoint's address as editable", () => {
    const onEditAddress = vi.fn();
    render(
      <EndpointView
        gw={makeGateway()}
        chainId={1}
        upstreamId="public-1-1"
        health={undefined}
        caps={undefined}
        capsBusy={false}
        capsErr={null}
        error={null}
        {...handlers}
        onEditAddress={onEditAddress}
      />,
    );
    const url = screen.getByText("https://rpc.publicnode.com");
    expect(url).toBeInTheDocument();
    url.click();
    expect(onEditAddress).toHaveBeenCalled();
  });

  it("shows the chain-head lag line only when the endpoint was scored with a positive lag", () => {
    const health: GatewayAnalytics = {
      enabled: true,
      at: "now",
      since: "then",
      networks: [],
      endpoints: [
        {
          upstream: "public-1-1",
          chainId: 1,
          configured: true,
          requests: 10,
          errors: null,
          scored: true,
          score: 1,
          position: 0,
          primarySwitches: 0,
          excludedSeconds: 0,
          headLag: 3,
          finalizationLag: 0,
          latestBlock: 100,
        },
      ],
    };
    render(
      <EndpointView
        gw={makeGateway()}
        chainId={1}
        upstreamId="public-1-1"
        health={health}
        caps={undefined}
        capsBusy={false}
        capsErr={null}
        error={null}
        {...handlers}
      />,
    );
    expect(screen.getByText("Chain head")).toBeInTheDocument();
    expect(screen.getByText("behind 3 blocks")).toBeInTheDocument();
  });

  it("shows 'no longer configured' when the upstream has gone", () => {
    render(
      <EndpointView
        gw={makeGateway()}
        chainId={1}
        upstreamId="gone"
        health={undefined}
        caps={undefined}
        capsBusy={false}
        capsErr={null}
        error={null}
        {...handlers}
      />,
    );
    expect(screen.getByText("This endpoint is no longer configured.")).toBeInTheDocument();
  });
});
