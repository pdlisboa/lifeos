/** RFC 9457 — o corpo de erro que o backend devolve (03-api.md §4, openapi.yaml Problem). */
export interface ApiProblem {
  type: string;
  title: string;
  status: number;
  detail?: string;
  /** A regra de negócio violada, ex. "RN-02". A UI decide por isto, nunca pelo texto em português. */
  rule?: string | null;
  errors?: { field: string; message: string }[];
  /** Presente no erro de RN-02: metas ativas já existentes, pra UI perguntar qual pausar. */
  activeGoals?: unknown[];
}

export class ApiError extends Error {
  readonly problem: ApiProblem;

  constructor(problem: ApiProblem) {
    super(problem.title || `Erro ${problem.status}`);
    this.name = "ApiError";
    this.problem = problem;
  }
}
