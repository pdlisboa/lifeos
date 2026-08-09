import { api, unwrap } from "@/lib/api";
import type { components } from "@lifeos/api-client";

export type EvidenceKind = components["schemas"]["EvidenceKind"];

// competencyIds é "Competências tocadas" (05-ux.md §7) — marcado por você,
// sem cerimônia (Fatia 2 fechou a lacuna de fatia-1-implementacao.md: o
// vínculo não depende mais do agente avaliador da Fatia 4).
export function createEvidence(
  goalId: string,
  body: {
    kind: EvidenceKind;
    title?: string;
    body?: string;
    externalUrl?: string;
    actionId?: string;
    competencyIds?: string[];
  },
) {
  return unwrap(api.POST("/goals/{goalId}/evidence", { params: { path: { goalId } }, body }));
}

export function fetchEvidence(evidenceId: string) {
  return unwrap(api.GET("/evidence/{evidenceId}", { params: { path: { evidenceId } } }));
}

/** Museu (§7.4) — ordem crescente por padrão, opcionalmente filtrado por competência tocada. */
export function fetchEvidenceList(
  goalId: string,
  opts: { competencyId?: string; order?: "asc" | "desc" } = {},
) {
  return unwrap(
    api.GET("/goals/{goalId}/evidence", {
      params: { path: { goalId }, query: { competencyId: opts.competencyId, order: opts.order } },
    }),
  );
}
