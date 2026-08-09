import { useEffect } from "react";
import { useProbe, useAnswerProbe, useSkipProbe } from "@/features/probe/use-probe";
import { ProblemError } from "@/components/problem-error";
import { Button } from "@/components/ui";

const answers = ["Sim", "Mais ou menos", "Não"];

export function ProbeStep({ goalId, onDone }: { goalId: string; onDone: () => void }) {
  const { data: probe, isPending, error } = useProbe(goalId);
  const answer = useAnswerProbe(goalId);
  const skip = useSkipProbe(goalId);

  const probeDone = !!probe && (!probe.currentQuestion || probe.status !== "open");

  // Efeito colateral (onDone troca o passo do wizard) precisa ficar fora do
  // corpo do render — chamar direto ali dispara de novo a cada re-render
  // enquanto probeDone continuar true, não só na primeira vez que fica true.
  useEffect(() => {
    if (probeDone) onDone();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [probeDone]);

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;
  if (probeDone) return null;

  const question = probe.currentQuestion!;
  const busy = answer.isPending || skip.isPending;

  // Não chama onDone aqui: a mutation atualiza o cache do probe, probeDone
  // recalcula no próximo render, e o useEffect acima dispara onDone uma única
  // vez — chamar nos dois lugares duplicava a chamada (probeDone virava true
  // E o retorno da mutation também mandava avançar).
  const submitAnswer = async (text: string) => {
    try {
      await answer.mutateAsync({ turnId: question.id, answer: text });
    } catch {
      // erro fica em answer.error, renderizado abaixo
    }
  };

  const doSkip = async () => {
    try {
      await skip.mutateAsync();
    } catch {
      // erro fica em skip.error, renderizado abaixo
    }
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

      {(answer.error || skip.error) && <ProblemError error={answer.error ?? skip.error} />}

      <div className="border-t border-border-subtle pt-4">
        <button className="text-sm text-fg-secondary hover:text-fg-primary" disabled={busy} onClick={doSkip}>
          Já sei o suficiente, quero começar →
        </button>
      </div>
    </div>
  );
}
