import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { todayQueryKey } from "@/features/today/use-today";
import { completeAction, fetchAction, skipAction, type SkipReason } from "./api";

export function useGoalAction(goalId: string) {
  return useQuery({
    queryKey: ["goal-action", goalId],
    queryFn: () => fetchAction(goalId),
    retry: false,
  });
}

/**
 * Concluir/pular ação muda /today e o delta da meta (06-frontend.md §6.4).
 * Invalidar os dois é o que evita a tela "Hoje" mentir sobre o que já foi feito.
 */
export function useCompleteAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      actionId,
      goalId,
      ...body
    }: {
      actionId: string;
      goalId: string;
      usedMinimalVariant?: boolean;
      durationMin?: number;
      note?: string;
    }) => completeAction(actionId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: todayQueryKey });
      qc.invalidateQueries({ queryKey: ["delta", variables.goalId] });
      qc.invalidateQueries({ queryKey: ["goal", variables.goalId] });
    },
  });
}

export function useSkipAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ actionId, reason, note }: { actionId: string; goalId: string; reason: SkipReason; note?: string }) =>
      skipAction(actionId, { reason, note }),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: todayQueryKey });
      qc.invalidateQueries({ queryKey: ["goal", variables.goalId] });
    },
  });
}
