import { useQuery } from "@tanstack/react-query";
import { fetchDelta } from "./api";

export function useDelta(goalId: string) {
  return useQuery({ queryKey: ["delta", goalId], queryFn: () => fetchDelta(goalId) });
}
