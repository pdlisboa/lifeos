import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeGoalSummary } from "@/test/fixtures";
import { MetasListRoute } from "./index";

function renderMetas() {
  return renderRoute(<MetasListRoute />, {
    extraRoutes: [
      { path: "/metas/nova", element: <div>tela de nova meta</div> },
      { path: "/metas/:goalId", element: <div>tela de detalhe</div> },
    ],
  });
}

describe("MetasListRoute", () => {
  it("mostra carregando antes da resposta", () => {
    server.use(
      http.get(`${API_BASE}/goals`, async () => {
        await new Promise((r) => setTimeout(r, 50));
        return HttpResponse.json([]);
      }),
    );
    renderMetas();
    expect(screen.getByText("Carregando…")).toBeInTheDocument();
  });

  it("mostra erro quando a requisição falha", async () => {
    server.use(http.get(`${API_BASE}/goals`, () => HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 })));
    renderMetas();
    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
  });

  it("estado vazio: mostra CTA pra criar a primeira meta", async () => {
    server.use(http.get(`${API_BASE}/goals`, () => HttpResponse.json([])));
    renderMetas();
    expect(await screen.findByText("Nenhuma meta ainda.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Criar a primeira" })).toBeInTheDocument();
  });

  it("lista metas com o rótulo de status certo, e dias ativos só pra meta ativa", async () => {
    server.use(
      http.get(`${API_BASE}/goals`, () =>
        HttpResponse.json([
          makeGoalSummary({ id: "g1", title: "Go backend", status: "active", daysActive: 12 }),
          makeGoalSummary({ id: "g2", title: "Inglês técnico", status: "draft", daysActive: 0 }),
          makeGoalSummary({ id: "g3", title: "Meta pausada", status: "paused", daysActive: 3 }),
        ]),
      ),
    );
    renderMetas();

    expect(await screen.findByText("Go backend")).toBeInTheDocument();
    expect(screen.getByText("ativa")).toBeInTheDocument();
    expect(screen.getByText("12 dias")).toBeInTheDocument();

    expect(screen.getByText("Inglês técnico")).toBeInTheDocument();
    expect(screen.getByText("rascunho")).toBeInTheDocument();

    expect(screen.getByText("Meta pausada")).toBeInTheDocument();
    expect(screen.getByText("pausada")).toBeInTheDocument();
    // "3 dias" só apareceria pra status active — meta pausada não mostra contagem
    expect(screen.queryByText("3 dias")).not.toBeInTheDocument();
  });
});
