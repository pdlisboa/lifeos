import { Link } from "react-router-dom";
import { formatConsistency } from "@lifeos/core";
import { useToday } from "@/features/today/use-today";
import { ProblemError } from "@/components/problem-error";
import { Button } from "@/components/ui";
import { GoalTodayCard } from "./goal-card";

export function HojeRoute() {
  const { data, isPending, error } = useToday();

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  const { goals, consistency } = data;

  return (
    <div className="space-y-8">
      <div className="flex items-baseline justify-between">
        <h1 className="text-lg font-medium text-fg-primary">Hoje</h1>
        <span className="text-sm text-fg-secondary">
          {formatConsistency(consistency.activeDays, consistency.windowDays)}
        </span>
      </div>

      {goals.length === 0 ? (
        <div className="space-y-3 rounded-lg border border-dashed border-border px-4 py-8 text-center">
          <p className="text-sm text-fg-secondary">Nada aqui ainda. Que tal começar por uma coisa só?</p>
          <Link to="/metas/nova">
            <Button>Criar meta</Button>
          </Link>
        </div>
      ) : (
        <div className="space-y-8">
          {goals.map((item) => (
            <GoalTodayCard key={item.goal.id} item={item} />
          ))}
        </div>
      )}
    </div>
  );
}
