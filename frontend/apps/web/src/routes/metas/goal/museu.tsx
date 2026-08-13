import { useState } from "react";
import type { components } from "@lifeos/api-client";
import { useGoal } from "@/features/goal/use-goals";
import { useEvidenceList } from "@/features/evidence/use-evidence";
import { EvalCaseButton } from "@/features/evidence/eval-case-button";
import { ProblemError } from "@/components/problem-error";
import { Button } from "@/components/ui";
import { cn } from "@/lib/cn";

type EvidenceCard = components["schemas"]["EvidenceCard"];

function formatFullDate(iso: string): string {
  return new Date(iso).toLocaleDateString("pt-BR", { day: "numeric", month: "long" });
}

function evidenceContent(e: { body?: string | null; externalUrl?: string | null }): string {
  return e.body ?? e.externalUrl ?? "(sem conteúdo textual)";
}

/**
 * Museu de evidências (05-ux.md §6): ordem crescente por padrão — o mais
 * antigo primeiro, porque essa tela é feita para comparar o começo com o
 * agora. A comparação lado a lado é a funcionalidade mais poderosa do
 * produto e não depende de agente nenhum.
 */
export function MuseuView({ goalId }: { goalId: string }) {
  const { data: goal } = useGoal(goalId);
  const [competencyId, setCompetencyId] = useState<string>("all");
  const [compareMode, setCompareMode] = useState(false);
  const [leftId, setLeftId] = useState<string | null>(null);
  const [rightId, setRightId] = useState<string | null>(null);

  const { data: page, isPending, error } = useEvidenceList(goalId, {
    competencyId: competencyId === "all" ? undefined : competencyId,
    order: "asc",
  });

  const competencies = goal?.competencies ?? [];

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  const items = page.items ?? [];
  const canCompare = competencyId !== "all" && items.length >= 2;

  const toggleCompare = () => {
    if (!compareMode && items.length >= 2) {
      setLeftId(items[0].id);
      setRightId(items[items.length - 1].id);
    }
    setCompareMode((v) => !v);
  };

  const changeCompetency = (value: string) => {
    setCompetencyId(value);
    setCompareMode(false);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <label className="flex items-center gap-2 text-sm text-fg-secondary">
          Filtrar:
          <select
            value={competencyId}
            onChange={(e) => changeCompetency(e.target.value)}
            className="rounded-md border border-border bg-bg-raised px-2 py-1 text-sm text-fg-primary"
          >
            <option value="all">todas as competências</option>
            {competencies.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
              </option>
            ))}
          </select>
        </label>
        <Button variant={compareMode ? "secondary" : "ghost"} disabled={!canCompare} onClick={toggleCompare}>
          comparar ⇄
        </Button>
      </div>

      {items.length === 0 ? (
        <p className="text-sm text-fg-secondary">
          Quando você registrar evidências, elas aparecem aqui em ordem — e você vai poder comparar a primeira com a
          última.
        </p>
      ) : compareMode ? (
        <CompareView
          items={items}
          leftId={leftId ?? items[0].id}
          rightId={rightId ?? items[items.length - 1].id}
          onLeftChange={setLeftId}
          onRightChange={setRightId}
          competencyId={competencyId}
          competencyLabel={competencies.find((c) => c.id === competencyId)?.label ?? ""}
        />
      ) : (
        <ul className="space-y-3">
          {items.map((e) => (
            <li key={e.id} className="rounded-md border border-border-subtle p-3">
              <p className="text-xs text-fg-muted">{formatFullDate(e.createdAt)}</p>
              {e.title && <p className="mt-0.5 text-sm text-fg-primary">{e.title}</p>}
              <pre className="mt-1 max-h-32 overflow-auto whitespace-pre-wrap rounded bg-bg-base p-2 font-mono text-xs text-fg-secondary">
                {evidenceContent(e)}
              </pre>
              <div className="mt-2">
                <EvalCaseButton
                  goalId={goalId}
                  evidenceId={e.id}
                  competencies={competencies}
                  evalCase={e.evalCase}
                  highlightCompetencyIds={e.competencyIds}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function CompareView({
  items,
  leftId,
  rightId,
  onLeftChange,
  onRightChange,
  competencyId,
  competencyLabel,
}: {
  items: EvidenceCard[];
  leftId: string;
  rightId: string;
  onLeftChange: (id: string) => void;
  onRightChange: (id: string) => void;
  competencyId: string;
  competencyLabel: string;
}) {
  const left = items.find((i) => i.id === leftId) ?? items[0];
  const right = items.find((i) => i.id === rightId) ?? items[items.length - 1];

  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
      <CompareCard evidence={left} items={items} otherId={rightId} onChange={onLeftChange} competencyId={competencyId} competencyLabel={competencyLabel} />
      <CompareCard evidence={right} items={items} otherId={leftId} onChange={onRightChange} competencyId={competencyId} competencyLabel={competencyLabel} />
    </div>
  );
}

function CompareCard({
  evidence,
  items,
  otherId,
  onChange,
  competencyId,
  competencyLabel,
}: {
  evidence: EvidenceCard;
  items: EvidenceCard[];
  otherId: string;
  onChange: (id: string) => void;
  competencyId: string;
  competencyLabel: string;
}) {
  const level = evidence.levelsAtTime?.[competencyId];
  return (
    <div className="rounded-md border border-border-subtle">
      <div className="flex items-center justify-between gap-2 border-b border-border-subtle bg-bg-overlay px-3 py-2">
        <select
          value={evidence.id}
          onChange={(e) => onChange(e.target.value)}
          className="rounded border border-border bg-bg-raised px-1.5 py-0.5 text-xs text-fg-primary"
        >
          {items.map((i) => (
            <option key={i.id} value={i.id} disabled={i.id === otherId}>
              {formatFullDate(i.createdAt)}
            </option>
          ))}
        </select>
        <span className="text-xs text-fg-secondary">
          {competencyLabel}: {level === undefined ? "nível não registrado nesse momento" : `nível ${level}`}
        </span>
      </div>
      <pre
        className={cn(
          "max-h-64 overflow-auto whitespace-pre-wrap p-3 font-mono text-xs text-fg-secondary",
        )}
      >
        {evidenceContent(evidence)}
      </pre>
    </div>
  );
}
