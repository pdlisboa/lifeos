import { useEffect, useState } from "react";
import { Button, Label, Textarea } from "@/components/ui";
import { ProblemError } from "@/components/problem-error";
import { useMarkEvalCase, useUnmarkEvalCase } from "./use-evidence";

type Competency = { id: string; label: string };
type EvalCase = { note: string; scores: { competencyId: string; level: number }[] } | null | undefined;

const NOT_SET = "";

/**
 * Botão discreto pra capturar material de eval do A3 (04-agentes.md §6.1):
 * nota livre + o nível que você daria por competência (gabarito humano).
 * Puro dado — não chama agente nenhum. Usado no museu e na tela de
 * evidência (06-frontend.md).
 */
export function EvalCaseButton({
  goalId,
  evidenceId,
  competencies,
  evalCase,
  highlightCompetencyIds = [],
}: {
  goalId: string;
  evidenceId: string;
  competencies: Competency[];
  evalCase: EvalCase;
  highlightCompetencyIds?: string[];
}) {
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState(evalCase?.note ?? "");
  const [levels, setLevels] = useState<Record<string, string>>(() => initialLevels(competencies, evalCase));
  // Espelha a prop, mas some na frente da atualização otimista de rótulo
  // assim que salvar/desmarcar responde — sem esperar o refetch da query do
  // pai (museu/tela de evidência) pra não piscar o rótulo errado.
  const [displayEvalCase, setDisplayEvalCase] = useState(evalCase);
  useEffect(() => setDisplayEvalCase(evalCase), [evalCase]);

  const mark = useMarkEvalCase(goalId);
  const unmark = useUnmarkEvalCase(goalId);
  const ordered = orderedCompetencies(competencies, highlightCompetencyIds);

  const startEditing = () => {
    setNote(displayEvalCase?.note ?? "");
    setLevels(initialLevels(competencies, displayEvalCase));
    setOpen(true);
  };

  const setLevel = (competencyId: string, value: string) => {
    setLevels((prev) => ({ ...prev, [competencyId]: value }));
  };

  const scores = Object.entries(levels)
    .filter(([, v]) => v !== NOT_SET)
    .map(([competencyId, v]) => ({ competencyId, level: Number(v) }));

  const canSave = note.trim().length > 0 && scores.length > 0;

  const save = async () => {
    try {
      await mark.mutateAsync({ evidenceId, note: note.trim(), scores });
      setDisplayEvalCase({ note: note.trim(), scores });
      setOpen(false);
    } catch {
      // erro fica em mark.error, renderizado abaixo
    }
  };

  const remove = async () => {
    try {
      await unmark.mutateAsync(evidenceId);
      setDisplayEvalCase(null);
      setOpen(false);
    } catch {
      // erro fica em unmark.error, renderizado abaixo
    }
  };

  if (!open) {
    return (
      <Button variant="ghost" className="h-auto px-2 py-1 text-xs" onClick={startEditing}>
        {displayEvalCase ? "caso de eval ✓ — editar" : "usar como caso de eval"}
      </Button>
    );
  }

  return (
    <div className="space-y-3 rounded-md border border-border-subtle bg-bg-overlay p-3">
      <div>
        <Label htmlFor={`eval-note-${evidenceId}`}>Por que esse caso importa</Label>
        <Textarea
          id={`eval-note-${evidenceId}`}
          rows={2}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="ex.: exemplo claro de goroutine com race condition"
        />
      </div>

      <div>
        <Label>Nível que você daria (gabarito)</Label>
        <div className="mt-1 space-y-1.5">
          {ordered.map((c) => (
            <div key={c.id} className="flex items-center justify-between gap-2 text-sm text-fg-secondary">
              <span>{c.label}</span>
              <select
                aria-label={`Nível para ${c.label}`}
                value={levels[c.id] ?? NOT_SET}
                onChange={(e) => setLevel(c.id, e.target.value)}
                className="rounded border border-border bg-bg-raised px-1.5 py-0.5 text-xs text-fg-primary"
              >
                <option value={NOT_SET}>—</option>
                {[0, 1, 2, 3, 4, 5].map((lvl) => (
                  <option key={lvl} value={lvl}>
                    {lvl}
                  </option>
                ))}
              </select>
            </div>
          ))}
        </div>
      </div>

      {mark.error && <ProblemError error={mark.error} />}
      {unmark.error && <ProblemError error={unmark.error} />}

      <div className="flex items-center justify-between gap-2">
        <div className="flex gap-2">
          {displayEvalCase && (
            <Button variant="danger" className="h-auto px-2 py-1 text-xs" disabled={unmark.isPending} onClick={remove}>
              {unmark.isPending ? "desmarcando…" : "desmarcar"}
            </Button>
          )}
          <Button variant="ghost" className="h-auto px-2 py-1 text-xs" onClick={() => setOpen(false)}>
            cancelar
          </Button>
        </div>
        <Button className="h-auto px-3 py-1 text-xs" disabled={!canSave || mark.isPending} onClick={save}>
          {mark.isPending ? "salvando…" : "salvar caso de eval"}
        </Button>
      </div>
    </div>
  );
}

function initialLevels(competencies: Competency[], evalCase: EvalCase): Record<string, string> {
  const byCompetency = new Map(evalCase?.scores.map((s) => [s.competencyId, String(s.level)]));
  const out: Record<string, string> = {};
  for (const c of competencies) {
    out[c.id] = byCompetency.get(c.id) ?? NOT_SET;
  }
  return out;
}

// Evidências marcam "competências tocadas" na criação (§7 da UX) — pôr essas
// primeiro poupa o usuário de garimpar a lista pra achar o que a evidência
// já indicou que toca.
function orderedCompetencies(competencies: Competency[], highlightCompetencyIds: string[]): Competency[] {
  if (highlightCompetencyIds.length === 0) return competencies;
  const highlighted = new Set(highlightCompetencyIds);
  const touched = competencies.filter((c) => highlighted.has(c.id));
  const rest = competencies.filter((c) => !highlighted.has(c.id));
  return [...touched, ...rest];
}
