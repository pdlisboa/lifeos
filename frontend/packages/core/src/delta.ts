/**
 * "O quanto andou", nunca "quanto falta" (P2). Retorna null quando não há
 * baseline ou nível atual — nesse caso não existe delta pra mostrar, e a UI
 * deve cair para outro estado (fase 1/2 do painel, ver deltaPhase), não para "0".
 */
export function formatDelta(
  baseline: number | null,
  current: number | null,
  windowDays: number | null,
): string | null {
  if (baseline === null || current === null) return null;

  const diff = current - baseline;
  if (diff === 0) return null;

  const arrow = diff > 0 ? "↑" : "↓";
  const magnitude = Math.abs(diff);

  if (windowDays === null) return `${arrow} ${magnitude}`;
  return `${arrow} ${magnitude} em ${windowDays} dias`;
}

/** "2 → 4" — o número é sempre de→para, nunca só o valor atual (§5.1 da UX). */
export function formatLevelRange(baseline: number | null, current: number | null): string {
  const b = baseline === null ? "?" : String(baseline);
  const c = current === null ? "?" : String(current);
  return `${b} → ${c}`;
}
