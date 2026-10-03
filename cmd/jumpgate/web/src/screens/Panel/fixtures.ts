// Shared test fixtures for the panel's component and view tests. A running
// gateway fronting two chains, each with upstreams — enough to render every
// row, dot and meter without reaching for the real backend.
import type { GatewayView, ContainerStatus } from "../../api";

export const STATUS_RUNNING: ContainerStatus = {
  ID: "gw",
  ContainerName: "erpc-gw",
  State: "running",
  Image: "erpc:latest",
  ImageID: "sha256:abc",
  ExitCode: 0,
  Platform: "linux/amd64",
  EnginePlatform: "linux/amd64",
  Emulated: false,
  Detail: "",
};

export function makeGateway(overrides: Partial<GatewayView> = {}): GatewayView {
  return {
    id: "default",
    label: "Local gateway",
    containerName: "erpc-default",
    placement: { targetId: "local", backend: "docker" },
    status: STATUS_RUNNING,
    docker: { present: true, reachable: true, flavor: "docker" },
    baseUrl: "https://valve.local",
    tls: { enabled: true, status: STATUS_RUNNING },
    networks: [
      {
        chainId: 1,
        name: "Ethereum",
        url: "https://valve.local/1",
        path: "/1",
        knownSetSize: 3,
        serviceable: true,
        upstreams: [
          {
            id: "public-1-1",
            kind: "external",
            endpoint: "https://rpc.publicnode.com",
            label: "publicnode",
            local: false,
            recentOnly: false,
            actions: null,
          },
        ],
      },
      {
        chainId: 369,
        name: "PulseChain",
        url: "https://valve.local/369",
        path: "/369",
        knownSetSize: 2,
        serviceable: true,
        upstreams: [],
      },
    ],
    actions: ["stop", "restart", "recreate", "wipe"],
    wipeDiscards: "the container and its stored config",
    config: {
      ProjectID: "main",
      BindAddr: "127.0.0.1",
      Port: 4000,
      Networks: [
        {
          ChainID: 1,
          Upstreams: [
            { ID: "public-1-1", Kind: "external", Endpoint: "https://rpc.publicnode.com", Local: false, RecentOnly: false },
          ],
        },
        { ChainID: 369, Upstreams: [] },
      ],
    },
    ...overrides,
  };
}
