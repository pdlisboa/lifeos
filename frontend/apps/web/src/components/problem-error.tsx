import { ApiError } from "@lifeos/api-client";

/**
 * Fallback genérico de erro (06-frontend.md §4). Toda tela que sabe reagir a um
 * `rule` específico (ex. RN-02) trata antes de chegar aqui — isto é só o que
 * sobra quando não há tratamento dedicado.
 */
export function ProblemError({ error }: { error: unknown }) {
  if (error instanceof ApiError) {
    return (
      <div className="rounded-md border border-danger-fg/30 bg-danger-bg px-4 py-3 text-sm text-danger-fg">
        <p className="font-medium">{error.problem.title}</p>
        {error.problem.detail && <p className="mt-1 text-danger-fg/80">{error.problem.detail}</p>}
      </div>
    );
  }
  return (
    <div className="rounded-md border border-danger-fg/30 bg-danger-bg px-4 py-3 text-sm text-danger-fg">
      Algo deu errado. Tenta de novo.
    </div>
  );
}
