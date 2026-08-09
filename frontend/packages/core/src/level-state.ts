import type { Confidence, LevelState } from "./types";

/**
 * Decide o estado visual de uma competência. `null` nunca vira "medido com
 * nível 0" — é a distinção mais cara do produto (06-frontend.md §6.2).
 *
 * `confidence: 'low'` cobre tanto "auto-declarado" quanto "vencido por
 * desuso" (ver descrição de Confidence no contrato); em ambos os casos o
 * número existe mas não sustenta uma barra cheia de confiança, então o
 * tratamento visual é o mesmo: 'stale'.
 */
export function levelState(level: number | null, confidence: Confidence): LevelState {
  if (level === null || confidence === "unknown") return "unknown";
  if (confidence === "low") return "stale";
  return "measured";
}
