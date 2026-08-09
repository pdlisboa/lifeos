import { useQuery } from "@tanstack/react-query";
import { fetchTrack } from "./api";

export function useTrack(goalId: string) {
  return useQuery({ queryKey: ["track", goalId], queryFn: () => fetchTrack(goalId) });
}
