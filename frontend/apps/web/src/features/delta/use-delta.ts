import { useQuery } from "@tanstack/react-query";
import { fetchDelta, fetchProjection } from "./api";

export function useDelta(goalId: string) {
  return useQuery({ queryKey: ["delta", goalId], queryFn: () => fetchDelta(goalId) });
}

export function useProjection(goalId: string) {
  return useQuery({ queryKey: ["projection", goalId], queryFn: () => fetchProjection(goalId) });
}
