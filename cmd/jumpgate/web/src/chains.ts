// The curated head of the network picker, from viem's typed chain definitions.
// These are the chains an operator here reaches for first; everything else in
// the picker comes from the full chainlist catalogue (api.allChains), searchable
// behind these. Named imports from viem/chains are tree-shaken, so only these
// definitions land in the bundle — not all ~200 of viem's chains.
import { mainnet, pulsechain, sepolia, pulsechainV4, base, bsc } from "viem/chains";
import type { ChainSummary } from "./api";

export interface CuratedChain extends ChainSummary {
  testnet: boolean;
  symbol?: string;
}

// Order is deliberate and product-owned: the two mainnets first, then their
// testnets, then the next tier down. It is NOT viem's default ordering.
export const CURATED: CuratedChain[] = [mainnet, pulsechain, sepolia, pulsechainV4, base, bsc].map((c) => ({
  chainId: c.id,
  name: c.name,
  testnet: Boolean(c.testnet),
  symbol: c.nativeCurrency?.symbol,
}));
