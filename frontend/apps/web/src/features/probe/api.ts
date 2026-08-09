import { api, unwrap } from "@/lib/api";

export function fetchProbe(goalId: string) {
  return unwrap(api.GET("/goals/{goalId}/probe", { params: { path: { goalId } } }));
}

export function answerProbe(goalId: string, body: { turnId: string; answer: string }) {
  return unwrap(api.POST("/goals/{goalId}/probe/answer", { params: { path: { goalId } }, body }));
}

export function skipProbe(goalId: string) {
  return unwrap(api.POST("/goals/{goalId}/probe/skip", { params: { path: { goalId } } }));
}
