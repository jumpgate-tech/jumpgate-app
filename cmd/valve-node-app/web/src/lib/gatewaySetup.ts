// Shared building blocks for the zero-machine one-click eRPC setup: the
// chains it comes up serving and the gateway config it creates. Both were
// originally defined in the (now-removed) capability-detected home screen
// and are kept here so the Easy-Button panel's one-click setup builds the
// exact same config rather than a second, driftable version of it.
import type * as api from "../api";

// SETUP_CHAINS are the chains the zero-machine one-click endpoint comes up
// serving: Ethereum and PulseChain, by valve's own measured known set. The
// devnet (1337) is deliberately absent — this flow builds a real public
// endpoint, not a scratch chain.
export const SETUP_CHAINS = [
  { chainId: 1, name: "Ethereum" },
  { chainId: 369, name: "PulseChain" },
];

// internalTLSConfig is the gateway config the one-click setup flow creates and
// updates: an eRPC on 4000 fronted by Caddy's internal CA (HTTPSPort 0 → 443).
// Hostname is left empty on purpose — the server fills a name whose wildcard
// already resolves to loopback.
export function internalTLSConfig(networks: api.GatewayNetwork[]): api.GatewayConfig {
  return {
    ProjectID: "main",
    BindAddr: "127.0.0.1",
    Port: 4000,
    Networks: networks,
    TLS: {
      Enabled: true,
      Hostname: "",
      CertSource: "internal",
      CertFile: "",
      KeyFile: "",
      HTTPSPort: 0,
      BindAddr: "",
      ImageRef: "",
    },
  };
}
