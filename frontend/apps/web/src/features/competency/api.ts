import { api, unwrap } from "@/lib/api";

export function setLevel(goalId: string, competencyId: string, body: { level: number; rationale: string }) {
  return unwrap(
    api.PUT("/goals/{goalId}/competencies/{competencyId}/level", {
      params: { path: { goalId, competencyId } },
      body,
    }),
  );
}
