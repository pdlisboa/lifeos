import { useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { useGoal } from "@/features/goal/use-goals";
import { useTrackProposals } from "@/features/proposal/use-proposal";
import { ProblemError } from "@/components/problem-error";
import { Button } from "@/components/ui";
import { cn } from "@/lib/cn";
import { DoneStep } from "@/routes/metas/nova/done-step";
import { TrackView } from "./track-view";
import { DeltaPanelView } from "./delta-panel";
import { MuseuView } from "./museu";
import { SessionForm } from "./session-form";
import { TrackProposalView } from "./track-proposal";

type Tab = "trilha" | "delta" | "museu" | "proposta";

export function MetaDetailRoute() {
  const { goalId } = useParams<{ goalId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const [showSessionForm, setShowSessionForm] = useState(false);

  const { data: goal, isPending, error } = useGoal(goalId!);
  // Hooks não podem ser condicionais — chamado aqui, antes de qualquer
  // return antecipado, mesmo que só use goal.id depois de saber que a meta
  // não está em rascunho (regra de Hooks do React).
  const { data: trackProposals } = useTrackProposals(goalId!);

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

  const hasPendingProposal = (trackProposals?.length ?? 0) > 0;

  const abaParam = searchParams.get("aba");
  const tab: Tab =
    abaParam === "delta" || abaParam === "museu" || (abaParam === "proposta" && hasPendingProposal)
      ? abaParam
      : "trilha";

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
          <button
            className={cn(
              "px-3 py-2 text-sm",
              tab === "museu" ? "border-b-2 border-delta-positive text-fg-primary" : "text-fg-muted",
            )}
            onClick={() => setSearchParams({ aba: "museu" })}
          >
            Museu
          </button>
          {hasPendingProposal && (
            <button
              className={cn(
                "px-3 py-2 text-sm",
                tab === "proposta" ? "border-b-2 border-delta-positive text-fg-primary" : "text-fg-muted",
              )}
              onClick={() => setSearchParams({ aba: "proposta" })}
            >
              Proposta
            </button>
          )}
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

      {tab === "trilha" && <TrackView goalId={goal.id} />}
      {tab === "delta" && <DeltaPanelView goalId={goal.id} />}
      {tab === "museu" && <MuseuView goalId={goal.id} />}
      {tab === "proposta" && <TrackProposalView goalId={goal.id} />}
    </div>
  );
}
