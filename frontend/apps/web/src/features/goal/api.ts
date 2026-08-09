import { api, unwrap } from "@/lib/api";
import type { components } from "@lifeos/api-client";

export type GoalStatus = components["schemas"]["GoalStatus"];
export type GoalArchetype = components["schemas"]["GoalArchetype"];

export function fetchGoals(status?: GoalStatus[]) {
  return unwrap(api.GET("/goals", { params: { query: status ? { status } : {} } }));
}

export function fetchGoal(goalId: string) {
  return unwrap(api.GET("/goals/{goalId}", { params: { path: { goalId } } }));
}

export function createGoal(body: { title: string; archetype: GoalArchetype; packId: string; why?: string }) {
  return unwrap(api.POST("/goals", { body }));
}

export function patchGoal(
  goalId: string,
  body: { title?: string; why?: string; definitionOfDone?: string; horizonOn?: string },
) {
  return unwrap(api.PATCH("/goals/{goalId}", { params: { path: { goalId } }, body }));
}

export function activateGoal(goalId: string, body?: { pauseGoalId?: string }) {
  return unwrap(api.POST("/goals/{goalId}/activate", { params: { path: { goalId } }, body: body ?? {} }));
}
