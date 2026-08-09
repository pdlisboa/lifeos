import { useState } from "react";
import { CartesianGrid, Line, LineChart, ResponsiveContainer, XAxis, YAxis } from "recharts";
import { formatLevelRange } from "@lifeos/core";
import { useCompetencyHistory } from "@/features/competency/use-competency";
import { useEvidence } from "@/features/evidence/use-evidence";

const MONTHS = ["jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"];

function formatShortDate(iso: string): string {
  const d = new Date(iso);
  return `${String(d.getDate()).padStart(2, "0")}/${MONTHS[d.getMonth()]}`;
}

/**
 * Série temporal de uma competência (05-ux.md §5.3). O gráfico é só o
 * traço — cada ponto que importa vira uma linha clicável logo abaixo, com o
 * rationale e (quando existe) o link pra evidência que sustentou a mudança.
 * Nível que muda sem explicação é nível em que você não confia.
 */
export function CompetencyHistoryChart({ goalId, competencyId }: { goalId: string; competencyId: string }) {
  const { data: events, isPending, error } = useCompetencyHistory(goalId, competencyId);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  if (isPending) return <p className="text-xs text-fg-muted">Carregando histórico…</p>;
  if (error) return <p className="text-xs text-fg-muted">Não deu pra carregar o histórico agora.</p>;
  if (events.length === 0) return <p className="text-xs text-fg-muted">Nenhum evento de nível ainda.</p>;

  // openapi-typescript marca campo não-`required` como `| undefined`; o contrato
  // sempre manda `null` quando ausente (06-frontend.md §6.2) — normaliza aqui.
  const points = events.map((e) => ({
    id: e.id,
    occurredAt: e.occurredAt,
    fromLevel: e.fromLevel ?? null,
    toLevel: e.toLevel,
    rationale: e.rationale,
    evidenceId: e.evidenceId ?? null,
  }));
  const chartData = points.map((p) => ({ date: p.occurredAt, nível: p.toLevel }));

  return (
    <div className="space-y-3">
      <div style={{ width: "100%", height: 140 }}>
        <ResponsiveContainer>
          <LineChart data={chartData} margin={{ top: 8, right: 12, left: -24, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#262b33" />
            <XAxis dataKey="date" tickFormatter={formatShortDate} stroke="#6b7280" fontSize={11} />
            <YAxis domain={[0, 5]} allowDecimals={false} stroke="#6b7280" fontSize={11} width={28} />
            <Line type="monotone" dataKey="nível" stroke="#4ade80" strokeWidth={2} dot={{ r: 3, fill: "#4ade80" }} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <ul className="space-y-1">
        {points.map((p) => (
          <HistoryPointRow
            key={p.id}
            event={p}
            expanded={expandedId === p.id}
            onToggle={() => setExpandedId(expandedId === p.id ? null : p.id)}
          />
        ))}
      </ul>
    </div>
  );
}

function HistoryPointRow({
  event,
  expanded,
  onToggle,
}: {
  event: { id: string; occurredAt: string; fromLevel: number | null; toLevel: number; rationale: string; evidenceId: string | null };
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <li className="rounded-md border border-border-subtle">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={expanded}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-fg-secondary hover:text-fg-primary"
      >
        <span aria-hidden className="text-delta-positive">
          ●
        </span>
        <span>{formatShortDate(event.occurredAt)}</span>
        <span className="text-fg-primary">{formatLevelRange(event.fromLevel, event.toLevel)}</span>
      </button>
      {expanded && (
        <div className="border-t border-border-subtle px-3 py-2 text-sm">
          <p className="text-fg-secondary">{event.rationale}</p>
          {event.evidenceId && <EvidenceLink evidenceId={event.evidenceId} />}
        </div>
      )}
    </li>
  );
}

function EvidenceLink({ evidenceId }: { evidenceId: string }) {
  const [show, setShow] = useState(false);
  const { data: evidence, isPending } = useEvidence(show ? evidenceId : null);

  if (!show) {
    return (
      <button
        type="button"
        className="mt-2 text-xs text-fg-muted underline hover:text-fg-secondary"
        onClick={() => setShow(true)}
      >
        ver evidência
      </button>
    );
  }

  return (
    <div className="mt-2 space-y-1">
      {isPending && <p className="text-xs text-fg-muted">Carregando evidência…</p>}
      {evidence && (
        <pre className="max-h-40 overflow-auto whitespace-pre-wrap rounded bg-bg-base p-2 font-mono text-xs text-fg-secondary">
          {evidence.body ?? evidence.externalUrl ?? "(sem conteúdo textual)"}
        </pre>
      )}
    </div>
  );
}
