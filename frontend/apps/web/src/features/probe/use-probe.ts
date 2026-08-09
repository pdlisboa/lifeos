import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as probeApi from "./api";

export function useProbe(goalId: string | null) {
  return useQuery({
    queryKey: ["probe", goalId],
    queryFn: () => probeApi.fetchProbe(goalId as string),
    enabled: goalId !== null,
  });
}

export function useAnswerProbe(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { turnId: string; answer: string }) => probeApi.answerProbe(goalId, body),
    onSuccess: (data) => {
      qc.setQueryData(["probe", goalId], data.probe);
    },
  });
}

export function useSkipProbe(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => probeApi.skipProbe(goalId),
    onSuccess: (probe) => {
      qc.setQueryData(["probe", goalId], probe);
    },
  });
}
