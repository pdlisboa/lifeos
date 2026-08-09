import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { render } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { createTestQueryClient } from "@/test/render";
import { makeUser } from "@/test/fixtures";
import { RequireAuth } from "./require-auth";

// Nota: usar MemoryRouter/Routes declarativo, não createMemoryRouter/RouterProvider
// (data router) — o data router do React Router constrói um Request interno pra
// navegação que colide com o AbortSignal do MSW+undici neste ambiente de teste
// (jsdom + Node). O app real usa createBrowserRouter (funciona no navegador de
// verdade); aqui só reproduzimos o comportamento declarativo, que é equivalente
// pro que RequireAuth precisa (Outlet + Navigate).
function renderWithRouter(initialPath: string) {
  const queryClient = createTestQueryClient();
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route element={<RequireAuth />}>
            <Route path="/" element={<div>conteúdo protegido</div>} />
          </Route>
          <Route path="/login" element={<div>tela de login</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("RequireAuth", () => {
  it("renderiza o conteúdo protegido quando /me resolve com usuário", async () => {
    server.use(http.get(`${API_BASE}/me`, () => HttpResponse.json(makeUser())));
    renderWithRouter("/");
    expect(await screen.findByText("conteúdo protegido")).toBeInTheDocument();
  });

  it("redireciona pro login quando /me devolve 401", async () => {
    server.use(http.get(`${API_BASE}/me`, () => new HttpResponse(null, { status: 401 })));
    renderWithRouter("/");
    expect(await screen.findByText("tela de login")).toBeInTheDocument();
  });

  it("não mostra nada nem o protegido nem o login enquanto /me está pendente", async () => {
    server.use(
      http.get(`${API_BASE}/me`, async () => {
        await new Promise((r) => setTimeout(r, 50));
        return HttpResponse.json(makeUser());
      }),
    );
    renderWithRouter("/");
    expect(screen.queryByText("conteúdo protegido")).not.toBeInTheDocument();
    expect(screen.queryByText("tela de login")).not.toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("conteúdo protegido")).toBeInTheDocument());
  });
});
