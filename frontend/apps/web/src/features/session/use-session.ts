import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createSession } from "./api";

export function useCreateSession(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { startedAt: string; durationMin: number; note?: string; energy?: number; actionId?: string }) =>
      createSession(goalId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["delta", goalId] });
      qc.invalidateQueries({ queryKey: ["goal", goalId] });
    },
  });
}
