import type { DeltaPhase } from "./types";

/** 3 semanas — abaixo disso o delta não é confiável o bastante para exibir (§5.2 da UX). */
const DELTA_RELIABLE_AFTER_DAYS = 21;

/**
 * Em que fase o Painel de Delta está (§5.2 da UX). A transição é por
 * quantidade de evidência e tempo, nunca por data fixa:
 *
 * - `baseline`: nenhuma evidência registrada ainda.
 * - `accumulating`: já há evidência, mas menos de 3 semanas de meta ativa —
 *   delta ainda não é confiável, a métrica é acúmulo.
 * - `delta`: 3+ semanas — o painel completo, com barra dupla e deltas.
 */
export function deltaPhase(evidenceCount: number, daysActive: number): DeltaPhase {
  if (evidenceCount === 0) return "baseline";
  if (daysActive < DELTA_RELIABLE_AFTER_DAYS) return "accumulating";
  return "delta";
}
