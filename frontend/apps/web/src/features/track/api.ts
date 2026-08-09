import { api, unwrap } from "@/lib/api";

export function fetchTrack(goalId: string) {
  return unwrap(api.GET("/goals/{goalId}/track", { params: { path: { goalId } } }));
}
