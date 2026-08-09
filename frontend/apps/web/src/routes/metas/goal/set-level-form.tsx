import { useState } from "react";
import { Button, Textarea } from "@/components/ui";
import { ProblemError } from "@/components/problem-error";
import { useSetLevel } from "@/features/competency/use-competency";

const LEVELS = [0, 1, 2, 3, 4, 5];

/**
 * RN-04 via aceite explícito (03-api.md): Fatia 1 não tem agente, então o único
 * jeito de sair de "ainda não medimos" é você mesmo declarar o nível — com uma
 * justificativa curta, pra o número não aparecer do nada (mesma régua do
 * `LevelEvent.rationale` do backend, que é obrigatório).
 */
export function SetLevelForm({
  goalId,
  competencyId,
  onDone,
}: {
  goalId: string;
  competencyId: string;
  onDone: () => void;
}) {
  const [level, setLevel] = useState(2);
  const [rationale, setRationale] = useState("");
  const setLevelMutation = useSetLevel(goalId);

  const canSubmit = rationale.trim().length >= 3;

  const submit = async () => {
    if (!canSubmit) return;
    try {
      await setLevelMutation.mutateAsync({ competencyId, level, rationale: rationale.trim() });
      onDone();
    } catch {
      // erro fica em setLevelMutation.error, renderizado abaixo
    }
  };

  return (
    <div className="space-y-2 rounded-md border border-border-subtle bg-bg-overlay p-3">
      <div className="flex flex-wrap gap-1.5">
        {LEVELS.map((l) => (
          <button
            key={l}
            type="button"
            onClick={() => setLevel(l)}
            aria-pressed={level === l}
            className={
              level === l
                ? "h-8 w-8 rounded-md bg-delta-positive text-sm font-medium text-bg-base"
                : "h-8 w-8 rounded-md border border-border text-sm text-fg-secondary hover:text-fg-primary"
            }
          >
            {l}
          </button>
        ))}
      </div>
      <Textarea
        rows={2}
        placeholder="por que esse nível? (ex.: já escrevo worker pool copiando exemplo)"
        value={rationale}
        onChange={(e) => setRationale(e.target.value)}
        aria-label="Por que esse nível?"
      />
      {setLevelMutation.error && <ProblemError error={setLevelMutation.error} />}
      <div className="flex gap-2">
        <Button
          variant="secondary"
          disabled={!canSubmit || setLevelMutation.isPending}
          onClick={submit}
        >
          {setLevelMutation.isPending ? "Salvando…" : "Confirmar nível"}
        </Button>
        <Button variant="ghost" onClick={onDone}>
          cancelar
        </Button>
      </div>
    </div>
  );
}
