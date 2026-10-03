// React Query hooks over api.ts's settings surface — the Settings screen's
// data layer (mirrors settings.ts's own load()/save() bookkeeping).
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";

export function useSettings(): UseQueryResult<api.Settings> {
  return useQuery({
    queryKey: ["settings"],
    queryFn: () => api.getSettings(),
  });
}

// usePutSettings mirrors settings.ts's save(): putSettings() already returns
// the fresh, canonical Settings (the same object save() assigned to `current`
// on success), so the cache is seeded directly with setQueryData rather than
// invalidated — invalidating would cost a redundant GET round-trip for data
// the response already carried.
export function usePutSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: api.PutSettingsRequest) => api.putSettings(body),
    onSuccess: (updated) => {
      qc.setQueryData(["settings"], updated);
    },
  });
}
