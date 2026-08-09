import { api, unwrap } from "@/lib/api";
import type { components } from "@lifeos/api-client";

export type EvidenceKind = components["schemas"]["EvidenceKind"];

// Sem competencyIds de propósito: o contrato (openapi.yaml, fonte da verdade) não
// aceita esse campo na criação — o vínculo evidência↔competência só existe a partir
// da avaliação do agente (Fatia 4; ver fatia-1-implementacao.md sobre EvidenceCard).
export function createEvidence(
  goalId: string,
  body: {
    kind: EvidenceKind;
    title?: string;
    body?: string;
    externalUrl?: string;
    actionId?: string;
  },
) {
  return unwrap(api.POST("/goals/{goalId}/evidence", { params: { path: { goalId } }, body }));
}
