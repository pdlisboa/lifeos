import { api, unwrap } from "@/lib/api";

export function setLevel(goalId: string, competencyId: string, body: { level: number; rationale: string }) {
  return unwrap(
    api.PUT("/goals/{goalId}/competencies/{competencyId}/level", {
      params: { path: { goalId, competencyId } },
      body,
    }),
  );
}

/** Série temporal de níveis (§5.3) — vem em ordem cronológica, cada evento com o rationale que sustentou a mudança. */
export function fetchCompetencyHistory(goalId: string, competencyId: string) {
  return unwrap(
    api.GET("/goals/{goalId}/competencies/{competencyId}/history", {
      params: { path: { goalId, competencyId } },
    }),
  );
}
