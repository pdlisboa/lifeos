import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchCompetencyHistory, setLevel } from "./api";

export function useSetLevel(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ competencyId, level, rationale }: { competencyId: string; level: number; rationale: string }) =>
      setLevel(goalId, competencyId, { level, rationale }),
    onSuccess: (_data, { competencyId }) => {
      qc.invalidateQueries({ queryKey: ["delta", goalId] });
      qc.invalidateQueries({ queryKey: ["goal", goalId] });
      qc.invalidateQueries({ queryKey: ["competencyHistory", goalId, competencyId] });
    },
  });
}

export function useCompetencyHistory(goalId: string, competencyId: string, enabled = true) {
  return useQuery({
    queryKey: ["competencyHistory", goalId, competencyId],
    queryFn: () => fetchCompetencyHistory(goalId, competencyId),
    enabled,
  });
}
