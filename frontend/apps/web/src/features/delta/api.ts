import { api, unwrap } from "@/lib/api";

export function fetchDelta(goalId: string) {
  return unwrap(api.GET("/goals/{goalId}/delta", { params: { path: { goalId } } }));
}

export function fetchProjection(goalId: string) {
  return unwrap(api.GET("/goals/{goalId}/projection", { params: { path: { goalId } } }));
}
