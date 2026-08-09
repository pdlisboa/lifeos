import { useProbe, useAnswerProbe, useSkipProbe } from "@/features/probe/use-probe";
import { ProblemError } from "@/components/problem-error";
import { Button } from "@/components/ui";

const answers = ["Sim", "Mais ou menos", "Não"];

export function ProbeStep({ goalId, onDone }: { goalId: string; onDone: () => void }) {
  const { data: probe, isPending, error } = useProbe(goalId);
  const answer = useAnswerProbe(goalId);
  const skip = useSkipProbe(goalId);

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  if (!probe.currentQuestion || probe.status !== "open") {
    onDone();
    return null;
  }

  const question = probe.currentQuestion;
  const busy = answer.isPending || skip.isPending;

  const submitAnswer = async (text: string) => {
    const result = await answer.mutateAsync({ turnId: question.id, answer: text });
    if (!result.nextQuestion) onDone();
  };

  const doSkip = async () => {
    await skip.mutateAsync();
    onDone();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-baseline justify-between">
        <h2 className="text-base font-medium text-fg-primary">Vamos achar seu ponto de partida</h2>
        <span className="text-xs text-fg-muted">
          pergunta {probe.askedCount + 1} de {probe.maxQuestions}
        </span>
      </div>

      <p className="text-sm text-fg-primary">{question.question}</p>

      <div className="flex flex-wrap gap-2">
        {answers.map((a) => (
          <Button key={a} variant="secondary" disabled={busy} onClick={() => submitAnswer(a)}>
            {a}
          </Button>
        ))}
      </div>

      <div className="border-t border-border-subtle pt-4">
        <button
          className="text-sm text-fg-secondary hover:text-fg-primary"
          disabled={busy}
          onClick={doSkip}
        >
          Já sei o suficiente, quero começar →
        </button>
      </div>
    </div>
  );
}
