import { useState } from "react";
import { Link } from "react-router-dom";
import { Card, CardBody, Button } from "@/components/ui";
import { ProblemError } from "@/components/problem-error";
import { useCompleteAction, useSkipAction } from "@/features/action/use-actions";
import type { SkipReason } from "@/features/action/api";
import type { components } from "@lifeos/api-client";

type TodayGoal = components["schemas"]["TodayGoal"];

const skipReasons: { value: SkipReason; label: string }[] = [
  { value: "too_hard", label: "difícil demais" },
  { value: "too_easy", label: "fácil demais" },
  { value: "no_time", label: "sem tempo" },
  { value: "not_relevant", label: "não faz sentido" },
];

export function GoalTodayCard({ item }: { item: TodayGoal }) {
  const [showMinimal, setShowMinimal] = useState(false);
  const [showSkipReasons, setShowSkipReasons] = useState(false);
  const complete = useCompleteAction();
  const skip = useSkipAction();

  const { goal, action, recentWin } = item;

  const busy = complete.isPending || skip.isPending;

  return (
    <div className="space-y-2">
      <div className="flex items-baseline justify-between">
        <Link to={`/metas/${goal.id}`} className="text-sm font-semibold uppercase tracking-wide text-fg-primary hover:underline">
          {goal.title}
        </Link>
        <span className="text-xs text-fg-muted">{goal.daysActive} dias</span>
      </div>

      <Card>
        {recentWin && (
          <CardBody className="border-b border-border-subtle text-sm text-fg-secondary">{recentWin}</CardBody>
        )}
        <CardBody className="space-y-3">
          <div>
            <div className="flex items-start justify-between gap-4">
              <p className="text-sm text-fg-primary">{action.title}</p>
              <span className="shrink-0 text-xs text-fg-muted">~{action.estimatedMin} min</span>
            </div>
            {action.detail && <p className="mt-1 text-xs text-fg-secondary">{action.detail}</p>}
          </div>

          {!showSkipReasons ? (
            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant="primary"
                disabled={busy}
                onClick={() =>
                  complete.mutate(
                    { actionId: action.id, goalId: goal.id },
                    { onSuccess: () => setShowMinimal(false) },
                  )
                }
              >
                Feita
              </Button>
              <Button variant="secondary" disabled={busy} onClick={() => setShowSkipReasons(true)}>
                Pulei
              </Button>
              {action.minimalVariant && !showMinimal && (
                <button
                  className="text-xs text-fg-muted hover:text-fg-secondary"
                  onClick={() => setShowMinimal(true)}
                >
                  só tenho 5 min ▾
                </button>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              <p className="text-xs text-fg-secondary">por quê?</p>
              <div className="flex flex-wrap gap-2">
                {skipReasons.map((r) => (
                  <Button
                    key={r.value}
                    variant="secondary"
                    disabled={busy}
                    onClick={() =>
                      skip.mutate(
                        { actionId: action.id, goalId: goal.id, reason: r.value },
                        { onSuccess: () => setShowSkipReasons(false) },
                      )
                    }
                  >
                    {r.label}
                  </Button>
                ))}
                <Button variant="ghost" onClick={() => setShowSkipReasons(false)}>
                  cancelar
                </Button>
              </div>
            </div>
          )}

          {(complete.error || skip.error) && <ProblemError error={complete.error ?? skip.error} />}

          {showMinimal && action.minimalVariant && (
            <div className="rounded-md border border-border-subtle bg-bg-overlay px-3 py-2">
              <p className="text-xs text-fg-secondary">{action.minimalVariant}</p>
              <Button
                variant="secondary"
                className="mt-2"
                disabled={busy}
                onClick={() =>
                  complete.mutate(
                    { actionId: action.id, goalId: goal.id, usedMinimalVariant: true },
                    { onSuccess: () => setShowMinimal(false) },
                  )
                }
              >
                Feita (mínimo)
              </Button>
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
