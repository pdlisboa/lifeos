import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError } from "@lifeos/api-client";
import { Button, Textarea } from "@/components/ui";
import { ProblemError } from "@/components/problem-error";
import { usePatchGoal, useActivateGoal } from "@/features/goal/use-goals";
import type { components } from "@lifeos/api-client";

type GoalSummary = components["schemas"]["GoalSummary"];

export function DoneStep({ goalId }: { goalId: string }) {
  const navigate = useNavigate();
  const [definitionOfDone, setDefinitionOfDone] = useState("");
  const [activeGoalsToChoose, setActiveGoalsToChoose] = useState<GoalSummary[] | null>(null);
  const patchGoal = usePatchGoal(goalId);
  const activateGoal = useActivateGoal(goalId);

  const activate = async (pauseGoalId?: string) => {
    try {
      if (definitionOfDone.trim()) {
        await patchGoal.mutateAsync({ definitionOfDone });
      }
      await activateGoal.mutateAsync(pauseGoalId ? { pauseGoalId } : undefined);
      navigate(`/metas/${goalId}`);
    } catch (err) {
      if (err instanceof ApiError && err.problem.rule === "RN-02" && err.problem.activeGoals) {
        setActiveGoalsToChoose(err.problem.activeGoals as GoalSummary[]);
      }
    }
  };

  const busy = patchGoal.isPending || activateGoal.isPending;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-medium text-fg-primary">Quando você vai saber que chegou lá?</h2>
        <p className="mt-1 text-sm text-fg-secondary">
          Não vale "saber Go". Vale algo que dá pra observar: "escrever e testar um serviço HTTP concorrente sem
          seguir tutorial".
        </p>
      </div>

      <Textarea
        rows={3}
        value={definitionOfDone}
        onChange={(e) => setDefinitionOfDone(e.target.value)}
        placeholder="escrever e testar..."
      />

      {activeGoalsToChoose && (
        <div className="space-y-2 rounded-md border border-border-subtle bg-bg-overlay p-4">
          <p className="text-sm text-fg-primary">Você já tem 3 metas ativas. Qual pausar para ativar esta?</p>
          <div className="space-y-2">
            {activeGoalsToChoose.map((g) => (
              <Button key={g.id} variant="secondary" className="w-full justify-start" onClick={() => activate(g.id)}>
                pausar "{g.title}"
              </Button>
            ))}
          </div>
        </div>
      )}

      {(patchGoal.error || activateGoal.error) && !activeGoalsToChoose && (
        <ProblemError error={activateGoal.error ?? patchGoal.error} />
      )}

      <div className="flex justify-between">
        <Button variant="ghost" disabled={busy} onClick={() => navigate(`/metas/${goalId}`)}>
          Depois eu escrevo
        </Button>
        <Button disabled={busy} onClick={() => activate()}>
          {busy ? "Ativando…" : "Ativar meta"}
        </Button>
      </div>
    </div>
  );
}
