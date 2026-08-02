// React Query hooks over api.ts's target/catalog surface — the pair every
// target-scoped screen (machine, security, diagnostics, ...) reads on mount
// to find its target's wire config and its network's footer link, mirroring
// the listTargets()+getCatalog() Promise.all every one of those legacy
// screens hand-rolled in its own init().
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
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

// useHost reports the OS/arch valve-node-app itself runs on — the Targets
// screen's third leg of its Promise.all(listTargets, getCatalog, getHost),
// used to decide whether local node setup is viable on THIS server.
export function useHost(): UseQueryResult<api.Host> {
  return useQuery({
    queryKey: ["host"],
    queryFn: () => api.getHost(),
  });
}

// useAddTarget mirrors targets.ts's addLocal()/addSSH(): both just call
// api.addTarget with a different body, then reload the target list — here,
// invalidate it so React Query refetches.
export function useAddTarget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: api.AddTargetRequest) => api.addTarget(body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["targets"] });
    },
  });
}

// useDeleteTarget mirrors targets.ts's deleteTarget(): removes a target,
// then invalidates the list so the card disappears.
export function useDeleteTarget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteTarget(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["targets"] });
    },
  });
}
