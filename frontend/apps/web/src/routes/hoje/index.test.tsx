import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeConsistency, makeTodayGoal } from "@/test/fixtures";
import { HojeRoute } from "./index";

function renderHoje() {
  return renderRoute(<HojeRoute />, {
    extraRoutes: [{ path: "/metas/nova", element: <div>tela de criar meta</div> }],
  });
}

describe("HojeRoute", () => {
  it("mostra estado de carregamento antes da resposta", () => {
    server.use(
      http.get(`${API_BASE}/today`, async () => {
        await new Promise((r) => setTimeout(r, 50));
        return HttpResponse.json({ goals: [], nudge: null, consistency: makeConsistency(), pendingProposals: 0 });
      }),
    );
    renderHoje();
    expect(screen.getByText("Carregando…")).toBeInTheDocument();
  });

  it("mostra erro quando a requisição falha", async () => {
    server.use(http.get(`${API_BASE}/today`, () => HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 })));
    renderHoje();
    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
  });

  it("estado vazio: sem metas mostra CTA pra criar a primeira", async () => {
    server.use(
      http.get(`${API_BASE}/today`, () =>
        HttpResponse.json({ goals: [], nudge: null, consistency: makeConsistency({ activeDays: 0, label: "0 dos últimos 30 dias" }), pendingProposals: 0 }),
      ),
    );
    renderHoje();

    expect(await screen.findByText("Nada aqui ainda. Que tal começar por uma coisa só?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Criar meta" })).toBeInTheDocument();
  });

  it("mostra o rótulo de consistência e um card por meta ativa", async () => {
    const goals = [makeTodayGoal({ goal: { ...makeTodayGoal().goal, title: "Go backend" } }), makeTodayGoal({ goal: { ...makeTodayGoal().goal, title: "Inglês técnico" } })];
    server.use(
      http.get(`${API_BASE}/today`, () =>
        HttpResponse.json({ goals, nudge: null, consistency: makeConsistency({ activeDays: 7, windowDays: 30, label: "7 dos últimos 30 dias" }), pendingProposals: 0 }),
      ),
    );
    renderHoje();

    expect(await screen.findByText("7 dos últimos 30 dias")).toBeInTheDocument();
    expect(screen.getByText("Go backend")).toBeInTheDocument();
    expect(screen.getByText("Inglês técnico")).toBeInTheDocument();
    expect(screen.queryByText("Nada aqui ainda. Que tal começar por uma coisa só?")).not.toBeInTheDocument();
  });
});
