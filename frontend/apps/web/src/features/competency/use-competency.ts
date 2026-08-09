import { useMutation, useQueryClient } from "@tanstack/react-query";
import { setLevel } from "./api";

export function useSetLevel(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ competencyId, level, rationale }: { competencyId: string; level: number; rationale: string }) =>
      setLevel(goalId, competencyId, { level, rationale }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["delta", goalId] });
      qc.invalidateQueries({ queryKey: ["goal", goalId] });
    },
  });
}
