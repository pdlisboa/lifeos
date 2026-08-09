import type { ReactElement } from "react";
import { render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: 0 },
      mutations: { retry: false },
    },
  });
}

interface RenderRouteOptions {
  /** URL inicial do MemoryRouter — o que `useParams`/`useSearchParams` do componente veem. */
  route?: string;
  /** Padrão de rota que casa com `route` (ex.: "/metas/:goalId"). */
  path?: string;
  /** Rotas extras pra onde o componente pode navegar (Link, useNavigate) durante o teste. */
  extraRoutes?: Array<{ path: string; element: ReactElement }>;
  queryClient?: QueryClient;
}

/**
 * Render de rota com os providers reais (TanStack Query + Router) — as telas
 * usam `useNavigate`/`useParams`/`useSearchParams` de verdade, então testá-las
 * sem esses providers exigiria mockar tudo isso à mão.
 */
export function renderRoute(ui: ReactElement, options: RenderRouteOptions = {}) {
  const { route = "/", path = "/", extraRoutes = [], queryClient = createTestQueryClient() } = options;

  const result = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path={path} element={ui} />
          {extraRoutes.map((r) => (
            <Route key={r.path} path={r.path} element={r.element} />
          ))}
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { ...result, queryClient };
}
