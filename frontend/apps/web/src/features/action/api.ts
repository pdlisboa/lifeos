import { api, unwrap } from "@/lib/api";
import type { components } from "@lifeos/api-client";

export type SkipReason = components["schemas"]["SkipReason"];

export function fetchAction(goalId: string) {
  return unwrap(api.GET("/goals/{goalId}/action", { params: { path: { goalId } } }));
}

export function completeAction(
  actionId: string,
  body: { usedMinimalVariant?: boolean; durationMin?: number; note?: string },
) {
  return unwrap(api.POST("/actions/{actionId}/complete", { params: { path: { actionId } }, body }));
}

export function skipAction(actionId: string, body: { reason: SkipReason; note?: string }) {
  return unwrap(api.POST("/actions/{actionId}/skip", { params: { path: { actionId } }, body }));
}
