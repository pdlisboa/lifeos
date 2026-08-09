export type Confidence = "unknown" | "low" | "medium" | "high";

export type LevelState = "unknown" | "measured" | "stale";

export type DeltaPhase = "baseline" | "accumulating" | "delta";

/** Formato mínimo aceito por sortCompetenciesForPanel — qualquer objeto com esses campos serve. */
export interface CompetencyForPanel {
  level: number | null;
  baselineLevel: number | null;
  /** Se já vier calculado pela API (DeltaPanel), usa direto em vez de recalcular. */
  delta?: number | null;
}
