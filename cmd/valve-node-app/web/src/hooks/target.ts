// React Query hooks over api.ts's target/catalog surface — the pair every
// target-scoped screen (machine, security, diagnostics, ...) reads on mount
// to find its target's wire config and its network's footer link, mirroring
// the listTargets()+getCatalog() Promise.all every one of those legacy
// screens hand-rolled in its own init().
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";

export function useTargets(): UseQueryResult<api.Target[]> {
  return useQuery({
    queryKey: ["targets"],
    queryFn: () => api.listTargets(),
  });
}

export function useCatalog(): UseQueryResult<api.Catalog> {
  return useQuery({
    queryKey: ["catalog"],
    queryFn: () => api.getCatalog(),
  });
}
