// React Query hooks over the vault surface. useVault polls every 5 s so the
// banner follows a lock or unlock done elsewhere (the CLI, the TUI).
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";

export function useVault(): UseQueryResult<api.Vault> {
  return useQuery({
    queryKey: ["vault"],
    queryFn: api.getVault,
    refetchInterval: 5000,
    retry: false,
  });
}

// useUnlockVault and useLockVault write the answer straight into the cache.
export function useUnlockVault() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (passphrase: string) => api.unlockVault(passphrase),
    onSuccess: (v) => qc.setQueryData(["vault"], v),
  });
}

export function useLockVault() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.lockVault,
    onSuccess: (v) => qc.setQueryData(["vault"], v),
  });
}
