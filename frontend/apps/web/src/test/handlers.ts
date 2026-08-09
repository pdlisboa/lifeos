import { http, HttpResponse } from "msw";
import { API_BASE } from "./api-base";
import { makeConsistency } from "./fixtures";

/**
 * Handlers-padrão: **deslogado** por padrão (`/me` 401) — é o estado mais
 * comum pra testar telas isoladas sem precisar mockar autenticação toda hora.
 * Testes que precisam de sessão (RequireAuth feliz, LoginRoute já-logado)
 * sobrescrevem via `server.use(...)`.
 */
export const handlers = [
  http.get(`${API_BASE}/me`, () => new HttpResponse(null, { status: 401 })),
  http.get(`${API_BASE}/today`, () =>
    HttpResponse.json({
      goals: [],
      nudge: null,
      consistency: makeConsistency({ activeDays: 0, todayDone: false, label: "0 dos últimos 30 dias" }),
      pendingProposals: 0,
    }),
  ),
];
