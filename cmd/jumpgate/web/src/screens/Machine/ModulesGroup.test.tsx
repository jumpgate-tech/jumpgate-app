import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import type { Catalog } from "../../api";
import { ModulesGroup } from "./ModeStep";
import { ReviewStep } from "./ReviewStep";

const REASON = "The message board needs reth (valve fork). This node runs go-pulse.";

const CAT: Catalog = {
  networks: [],
  clients: [],
  modules: [
    {
      id: "msgboard",
      name: "Message board",
      summary: "A board.",
      siteUrl: "",
      docsStep: "msgboard-enable",
      placement: "node",
      resources: { ramBytes: 0, diskBytes: 0 },
      applies: [
        { chainId: 369, client: "reth", ok: true, beta: false },
        { chainId: 369, client: "go-pulse", ok: false, beta: false, reason: REASON },
        { chainId: 1, client: "reth", ok: false, beta: false },
      ],
      methods: [],
    },
  ],
};

afterEach(cleanup);

describe("ModulesGroup (wizard)", () => {
  it("offers a checkbox where the module applies", () => {
    const onChange = vi.fn();
    render(<ModulesGroup catalog={CAT} chainId={369} execId="reth" selected={[]} onChange={onChange} />);
    fireEvent.click(screen.getByLabelText(/Message board/));
    expect(onChange).toHaveBeenCalledWith([{ id: "msgboard" }]);
  });

  it("disables with the reason verbatim", () => {
    render(<ModulesGroup catalog={CAT} chainId={369} execId="go-pulse" selected={[]} onChange={() => {}} />);
    expect((screen.getByLabelText(/Message board/) as HTMLInputElement).disabled).toBe(true);
    expect(screen.getByText(REASON)).toBeTruthy();
  });

  it("review lists the modules and the dropped notice", () => {
    render(
      <ReviewStep
        targetId="t1"
        net={undefined}
        chainId={369}
        execId="go-pulse"
        beaconId="lighthouse"
        archive={false}
        dataDir="/d"
        jwtPath="/d/jwt"
        checkpoint={true}
        checkpointUrl=""
        execHTTPPort=""
        beaconHTTPPort=""
        execP2PPort=""
        rpcBindAddr=""
        modules={{ names: ["Health"], notices: ["The message board will be turned off: go-pulse does not support it."] }}
        startError={null}
        starting={false}
        onBack={() => {}}
        onStart={() => {}}
      />,
    );
    expect(screen.getByText("Health")).toBeTruthy();
    expect(screen.getByText("The message board will be turned off: go-pulse does not support it.")).toBeTruthy();
  });

  it("renders nothing when no module applies", () => {
    const { container } = render(<ModulesGroup catalog={CAT} chainId={1} execId="reth" selected={[]} onChange={() => {}} />);
    expect(container.textContent).toBe("");
  });
});
