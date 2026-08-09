import { Link } from "react-router-dom";
import {
  deltaPhase,
  formatDelta,
  formatLevelRange,
  levelState,
  sortCompetenciesForPanel,
} from "@lifeos/core";
import { useDelta } from "@/features/delta/use-delta";
import { ProblemError } from "@/components/problem-error";
import { Button } from "@/components/ui";
import { cn } from "@/lib/cn";

const MAX_LEVEL = 5;

function LevelBar({ baseline, current }: { baseline: number | null; current: number }) {
  if (baseline === null) {
    return (
      <div className="h-2 w-full rounded-full bg-bg-overlay">
        <div className="h-2 rounded-full bg-delta-positive" style={{ width: `${(current / MAX_LEVEL) * 100}%` }} />
      </div>
    );
  }
  const baselinePct = (Math.min(baseline, current) / MAX_LEVEL) * 100;
  const gainedPct = (Math.max(current - baseline, 0) / MAX_LEVEL) * 100;
  return (
    <div className="flex h-2 w-full overflow-hidden rounded-full bg-bg-overlay">
      <div className="h-2 bg-fg-muted/40" style={{ width: `${baselinePct}%` }} />
      <div className="h-2 bg-delta-positive" style={{ width: `${gainedPct}%` }} />
    </div>
  );
}

function NotMeasuredBar() {
  return (
    <div className="h-2 w-full rounded-full border border-dashed border-fg-muted/40" aria-hidden />
  );
}

export function DeltaPanelView({ goalId }: { goalId: string }) {
  const { data: panel, isPending, error } = useDelta(goalId);

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  // evidenceCount não tem semântica null=não-medido (é so um contador); 0 é o
  // valor correto quando o campo vem ausente, ao contrário de `level ?? 0`.
  const phase = deltaPhase(panel.totals.evidenceCount ?? 0, panel.daysActive);
  // openapi-typescript marca campos não-`required` como opcionais (`| undefined`);
  // o contrato sempre os manda como `null` quando ausentes. Normaliza aqui pra
  // bater com CompetencyForPanel — não é o `?? 0` proibido, é undefined→null.
  const sorted = sortCompetenciesForPanel(
    panel.competencies.map((c) => ({
      ...c,
      level: c.level ?? null,
      baselineLevel: c.baselineLevel ?? null,
      delta: c.delta ?? null,
    })),
  );

  if (phase === "baseline") {
    return (
      <div className="space-y-6">
        <div className="rounded-md border border-border-subtle bg-bg-overlay px-4 py-3 text-sm text-fg-secondary">
          Seu ponto de partida está registrado. Daqui a algumas semanas, esta tela vai mostrar o quanto você andou.
        </div>

        <ul className="space-y-2">
          {sorted.map((c) => (
            <li key={c.id} className="flex items-center justify-between text-sm">
              <span className="text-fg-primary">{c.label}</span>
              <span className="text-fg-secondary">
                {c.level === null ? "ainda não medimos" : `${c.level}  ← você está aqui`}
              </span>
            </li>
          ))}
        </ul>

        <Link to={`/metas/${goalId}/evidencia`}>
          <Button>Registrar primeira evidência</Button>
        </Link>
      </div>
    );
  }

  if (phase === "accumulating") {
    return (
      <div className="space-y-6">
        <div className="rounded-md border border-border-subtle bg-bg-overlay px-4 py-3 text-sm text-fg-secondary">
          Ainda sem delta confiável — faltam algumas semanas. Por enquanto, o que conta é aparecer.
        </div>

        <div className="flex gap-6 text-sm">
          <div>
            <p className="text-lg text-fg-primary">{panel.totals.evidenceCount}</p>
            <p className="text-xs text-fg-muted">evidências</p>
          </div>
          <div>
            <p className="text-lg text-fg-primary">{panel.daysActive}</p>
            <p className="text-xs text-fg-muted">dias ativos</p>
          </div>
          <div>
            <p className="text-lg text-fg-primary">{panel.totals.milestonesDone}</p>
            <p className="text-xs text-fg-muted">marcos concluídos</p>
          </div>
        </div>

        <ul className="space-y-2">
          {sorted.map((c) => (
            <li key={c.id} className="flex items-center justify-between text-sm">
              <span className="text-fg-primary">{c.label}</span>
              <span className="text-fg-secondary">{c.level === null ? "ainda não medimos" : c.level}</span>
            </li>
          ))}
        </ul>
      </div>
    );
  }

  // phase === "delta"
  return (
    <div className="space-y-6">
      {panel.headline && (
        <div className="rounded-md border border-border-subtle bg-bg-overlay px-4 py-3 text-sm text-fg-secondary">
          {panel.headline}
        </div>
      )}

      <div className="space-y-5">
        {sorted.map((c) => {
          const state = levelState(c.level, c.confidence);
          return (
            <div key={c.id} className="space-y-1.5">
              <div className="flex items-baseline justify-between">
                <span className="text-sm text-fg-primary">{c.label}</span>
                <span className="text-sm text-fg-secondary">
                  {c.level === null ? "ainda não medimos" : formatLevelRange(c.baselineLevel, c.level)}
                </span>
              </div>

              {c.level === null ? (
                <NotMeasuredBar />
              ) : (
                <LevelBar baseline={c.baselineLevel} current={c.level} />
              )}

              {c.level !== null && (
                <p className={cn("text-right text-xs", state === "stale" ? "text-stale-fg" : "text-fg-muted")}>
                  {state === "stale" && c.staleDays !== null
                    ? `sem evidência há ${c.staleDays} dias ⟳`
                    : formatDelta(c.baselineLevel, c.level, c.deltaWindowDays ?? null) ?? ""}
                </p>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
