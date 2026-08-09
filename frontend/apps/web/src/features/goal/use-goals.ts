import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as goalApi from "./api";
import type { GoalStatus } from "./api";

export function useGoals(status?: GoalStatus[]) {
  return useQuery({
    queryKey: ["goals", status ?? "all"],
    queryFn: () => goalApi.fetchGoals(status),
  });
}

export function useGoal(goalId: string) {
  return useQuery({
    queryKey: ["goal", goalId],
    queryFn: () => goalApi.fetchGoal(goalId),
  });
}

export function useCreateGoal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: goalApi.createGoal,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["goals"] });
    },
  });
}

export function usePatchGoal(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Parameters<typeof goalApi.patchGoal>[1]) => goalApi.patchGoal(goalId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["goal", goalId] });
    },
  });
}

export function useActivateGoal(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body?: { pauseGoalId?: string }) => goalApi.activateGoal(goalId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["goal", goalId] });
      qc.invalidateQueries({ queryKey: ["goals"] });
      qc.invalidateQueries({ queryKey: ["today"] });
    },
  });
}
