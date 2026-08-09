import { useTrack } from "@/features/track/use-track";
import { ProblemError } from "@/components/problem-error";
import { cn } from "@/lib/cn";

const statusLabel: Record<string, string> = {
  locked: "bloqueado",
  current: "atual",
  completed: "concluído",
  skipped: "pulado",
};

export function TrackView({ goalId }: { goalId: string }) {
  const { data: track, isPending, error } = useTrack(goalId);

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  const milestones = track.milestones ?? [];

  if (milestones.length === 0) {
    return <p className="text-sm text-fg-secondary">Esta meta ainda não tem trilha definida.</p>;
  }

  return (
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
  );
}
