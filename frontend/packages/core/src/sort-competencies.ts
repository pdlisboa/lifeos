import type { CompetencyForPanel } from "./types";

function competencyDelta(c: CompetencyForPanel): number | null {
  if (c.delta !== undefined) return c.delta;
  if (c.level === null || c.baselineLevel === null) return null;
  return c.level - c.baselineLevel;
}

/**
 * Ordem do Painel de Delta: quem mais subiu primeiro (§5.1 da UX). Não medidas
 * (`level === null`) vão sempre para o fim, independente de qualquer delta.
 */
export function sortCompetenciesForPanel<T extends CompetencyForPanel>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    if (a.level === null && b.level === null) return 0;
    if (a.level === null) return 1;
    if (b.level === null) return -1;
    return (competencyDelta(b) ?? 0) - (competencyDelta(a) ?? 0);
  });
}
