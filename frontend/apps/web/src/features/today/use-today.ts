import { useQuery } from "@tanstack/react-query";
import { fetchToday } from "./api";

export const todayQueryKey = ["today"] as const;

export function useToday() {
  return useQuery({ queryKey: todayQueryKey, queryFn: fetchToday });
}
