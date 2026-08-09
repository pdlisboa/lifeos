import { Link } from "react-router-dom";
import { useGoals } from "@/features/goal/use-goals";
import { ProblemError } from "@/components/problem-error";
import { Button, Card, CardBody } from "@/components/ui";

const statusLabel: Record<string, string> = {
  draft: "rascunho",
  active: "ativa",
  at_risk: "em risco",
  stagnant: "parada",
  paused: "pausada",
  completed: "concluída",
  abandoned: "encerrada",
};

export function MetasListRoute() {
  const { data: goals, isPending, error } = useGoals();

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-medium text-fg-primary">Metas</h1>
        <Link to="/metas/nova">
          <Button>Nova meta</Button>
        </Link>
      </div>

      {goals.length === 0 ? (
        <div className="space-y-3 rounded-lg border border-dashed border-border px-4 py-8 text-center">
          <p className="text-sm text-fg-secondary">Nenhuma meta ainda.</p>
          <Link to="/metas/nova">
            <Button>Criar a primeira</Button>
          </Link>
        </div>
      ) : (
        <div className="space-y-2">
          {goals.map((goal) => (
            <Link key={goal.id} to={`/metas/${goal.id}`}>
              <Card className="transition-colors hover:border-border-subtle hover:bg-bg-overlay">
                <CardBody className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-fg-primary">{goal.title}</p>
                    <p className="text-xs text-fg-muted">{statusLabel[goal.status] ?? goal.status}</p>
                  </div>
                  {goal.status === "active" && <span className="text-xs text-fg-muted">{goal.daysActive} dias</span>}
                </CardBody>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
