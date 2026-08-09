import { useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { useGoal } from "@/features/goal/use-goals";
import { ProblemError } from "@/components/problem-error";
import { Button } from "@/components/ui";
import { cn } from "@/lib/cn";
import { DoneStep } from "@/routes/metas/nova/done-step";
import { TrackView } from "./track-view";
import { DeltaPanelView } from "./delta-panel";
import { SessionForm } from "./session-form";

type Tab = "trilha" | "delta";

export function MetaDetailRoute() {
  const { goalId } = useParams<{ goalId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const [showSessionForm, setShowSessionForm] = useState(false);

  const { data: goal, isPending, error } = useGoal(goalId!);

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  if (goal.status === "draft") {
    return (
      <div className="space-y-6">
        <Link to="/metas" className="text-sm text-fg-muted hover:text-fg-secondary">
          ← Metas
        </Link>
        <div>
          <h1 className="text-lg font-medium uppercase tracking-wide text-fg-primary">{goal.title}</h1>
          {(goal.blockers ?? []).length > 0 && (
            <ul className="mt-2 space-y-1 text-sm text-fg-secondary">
              {(goal.blockers ?? []).map((b) => (
                <li key={b}>· {b}</li>
              ))}
            </ul>
          )}
        </div>
        <DoneStep goalId={goal.id} />
      </div>
    );
  }

  const tab: Tab = searchParams.get("aba") === "delta" ? "delta" : "trilha";

  return (
    <div className="space-y-6">
      <Link to="/metas" className="text-sm text-fg-muted hover:text-fg-secondary">
        ← Metas
      </Link>

      <div className="flex items-baseline justify-between">
        <h1 className="text-lg font-medium uppercase tracking-wide text-fg-primary">{goal.title}</h1>
        <span className="text-sm text-fg-muted">{goal.daysActive} dias</span>
      </div>

      <div className="flex items-center justify-between border-b border-border-subtle">
        <div className="flex gap-1">
          <button
            className={cn(
              "px-3 py-2 text-sm",
              tab === "trilha" ? "border-b-2 border-delta-positive text-fg-primary" : "text-fg-muted",
            )}
            onClick={() => setSearchParams({ aba: "trilha" })}
          >
            Trilha
          </button>
          <button
            className={cn(
              "px-3 py-2 text-sm",
              tab === "delta" ? "border-b-2 border-delta-positive text-fg-primary" : "text-fg-muted",
            )}
            onClick={() => setSearchParams({ aba: "delta" })}
          >
            Delta
          </button>
        </div>
        <div className="flex gap-2 pb-2">
          <Button variant="ghost" onClick={() => setShowSessionForm((v) => !v)}>
            Registrar sessão
          </Button>
          <Link to={`/metas/${goal.id}/evidencia`}>
            <Button variant="secondary">Registrar evidência</Button>
          </Link>
        </div>
      </div>

      {showSessionForm && <SessionForm goalId={goal.id} onDone={() => setShowSessionForm(false)} />}

      {tab === "trilha" ? <TrackView goalId={goal.id} /> : <DeltaPanelView goalId={goal.id} />}
    </div>
  );
}
