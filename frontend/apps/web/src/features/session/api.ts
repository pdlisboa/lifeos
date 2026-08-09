import { api, unwrap } from "@/lib/api";

export function createSession(
  goalId: string,
  body: { startedAt: string; durationMin: number; note?: string; energy?: number; actionId?: string },
) {
  return unwrap(api.POST("/goals/{goalId}/sessions", { params: { path: { goalId } }, body }));
}
