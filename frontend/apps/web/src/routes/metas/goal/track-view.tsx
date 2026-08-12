import { useTrack } from "@/features/track/use-track";
import { useRequestTrackRevision } from "@/features/proposal/use-proposal";
import { ProblemError } from "@/components/problem-error";
import { Button } from "@/components/ui";
import { cn } from "@/lib/cn";

const statusLabel: Record<string, string> = {
  locked: "bloqueado",
  current: "atual",
  completed: "concluído",
  skipped: "pulado",
};

export function TrackView({ goalId }: { goalId: string }) {
  const { data: track, isPending, error } = useTrack(goalId);
  const requestRevision = useRequestTrackRevision(goalId);

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  const milestones = track.milestones ?? [];

  return (
    <div className="space-y-3">
      {milestones.length === 0 ? (
        <p className="text-sm text-fg-secondary">Esta meta ainda não tem trilha definida.</p>
      ) : (
        <ol className="space-y-3">
          {milestones.map((m) => (
            <li
              key={m.id}
              className={cn(
                "rounded-md border border-border-subtle px-4 py-3",
                m.status === "current" ? "bg-bg-overlay" : "bg-bg-raised",
              )}
            >
              <div className="flex items-baseline justify-between gap-4">
                <p className={cn("text-sm", m.status === "completed" ? "text-fg-secondary" : "text-fg-primary")}>
                  {m.title}
                </p>
                <span className="shrink-0 text-xs text-fg-muted">{statusLabel[m.status] ?? m.status}</span>
              </div>
              {m.completionCriteria && <p className="mt-1 text-xs text-fg-secondary">{m.completionCriteria}</p>}
            </li>
          ))}
        </ol>
      )}

      {requestRevision.error && <ProblemError error={requestRevision.error} />}
      {requestRevision.isSuccess ? (
        <p className="text-xs text-fg-muted">Pedido enviado ao planejador — a proposta aparece em alguns instantes.</p>
      ) : (
        <Button variant="ghost" disabled={requestRevision.isPending} onClick={() => requestRevision.mutate()}>
          {requestRevision.isPending ? "Pedindo…" : "Pedir revisão da trilha ao planejador"}
        </Button>
      )}
    </div>
  );
}
