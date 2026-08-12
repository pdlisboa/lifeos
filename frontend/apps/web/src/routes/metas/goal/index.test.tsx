import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeDeltaPanel, makeGoal, makeProposal, makeTrack } from "@/test/fixtures";
import { MetaDetailRoute } from "./index";

const GOAL_ID = "goal-detail";

function renderDetail(route = `/metas/${GOAL_ID}`) {
  return renderRoute(<MetaDetailRoute />, {
    route,
    path: "/metas/:goalId",
    extraRoutes: [
      { path: "/metas", element: <div>tela de lista de metas</div> },
      { path: `/metas/${GOAL_ID}/evidencia`, element: <div>tela de evidência</div> },
    ],
  });
}

describe("MetaDetailRoute", () => {
  it("mostra carregando antes da resposta", () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, async () => {
        await new Promise((r) => setTimeout(r, 50));
        return HttpResponse.json(makeGoal({ id: GOAL_ID }));
      }),
    );
    renderDetail();
    expect(screen.getByText("Carregando…")).toBeInTheDocument();
  });

  it("mostra erro quando a requisição falha", async () => {
    server.use(http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json({ title: "Não encontrado", status: 404 }, { status: 404 })));
    renderDetail();
    expect(await screen.findByText("Não encontrado")).toBeInTheDocument();
  });

  it("rascunho: mostra blockers e o passo de definição de pronto, sem abas", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, () =>
        HttpResponse.json(makeGoal({ id: GOAL_ID, status: "draft", blockers: ["falta a definição de pronto"] })),
      ),
    );
    renderDetail();

    expect(await screen.findByText(/falta a definição de pronto/)).toBeInTheDocument();
    expect(screen.getByText("Quando você vai saber que chegou lá?")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Trilha" })).not.toBeInTheDocument();
  });

  it("meta ativa: mostra abas com Trilha selecionada por padrão", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, status: "active", title: "Go backend" }))),
      http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () => HttpResponse.json(makeTrack({ milestones: [] }))),
    );
    renderDetail();

    expect(await screen.findByText("Esta meta ainda não tem trilha definida.")).toBeInTheDocument();
    // uppercase é só CSS (classe `uppercase`) — o texto real no DOM segue o fixture.
    expect(screen.getByText("Go backend")).toBeInTheDocument();
  });

  it("troca pra aba Delta ao clicar, e reflete na URL via ?aba=delta", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, status: "active" }))),
      http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () => HttpResponse.json(makeTrack({ milestones: [] }))),
      http.get(`${API_BASE}/goals/${GOAL_ID}/delta`, () => HttpResponse.json(makeDeltaPanel({ goalId: GOAL_ID, daysActive: 0, totals: { evidenceCount: 0, milestonesDone: 0, sessionMinutes: 0 } }))),
    );
    const user = userEvent.setup();
    renderDetail();

    await screen.findByText("Esta meta ainda não tem trilha definida.");
    await user.click(screen.getByRole("button", { name: "Delta" }));

    expect(await screen.findByText("Seu ponto de partida está registrado. Daqui a algumas semanas, esta tela vai mostrar o quanto você andou.")).toBeInTheDocument();
  });

  it("abre a URL já com ?aba=delta direto na aba Delta", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, status: "active" }))),
      http.get(`${API_BASE}/goals/${GOAL_ID}/delta`, () => HttpResponse.json(makeDeltaPanel({ totals: { evidenceCount: 0, milestonesDone: 0, sessionMinutes: 0 } }))),
    );
    renderDetail(`/metas/${GOAL_ID}?aba=delta`);

    expect(await screen.findByText(/Seu ponto de partida está registrado/)).toBeInTheDocument();
  });

  it("link 'Registrar evidência' aponta pra rota certa", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, status: "active" }))),
      http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () => HttpResponse.json(makeTrack({ milestones: [] }))),
    );
    renderDetail();
    await screen.findByText("Esta meta ainda não tem trilha definida.");

    expect(screen.getByRole("link", { name: "Registrar evidência" })).toHaveAttribute("href", `/metas/${GOAL_ID}/evidencia`);
  });

  it("sem proposta pendente, a aba Proposta não aparece", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, status: "active" }))),
      http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () => HttpResponse.json(makeTrack({ milestones: [] }))),
    );
    renderDetail();
    await screen.findByText("Esta meta ainda não tem trilha definida.");
    expect(screen.queryByRole("button", { name: "Proposta" })).not.toBeInTheDocument();
  });

  it("com proposta de trilha pendente, mostra a aba Proposta e permite navegar até ela", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, status: "active" }))),
      http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () => HttpResponse.json(makeTrack({ milestones: [] }))),
      http.get(`${API_BASE}/proposals`, () => HttpResponse.json([makeProposal({ goalId: GOAL_ID, rationale: "Motivo da proposta" })])),
    );
    const user = userEvent.setup();
    renderDetail();

    const tab = await screen.findByRole("button", { name: "Proposta" });
    await user.click(tab);

    expect(await screen.findByText("Motivo da proposta")).toBeInTheDocument();
  });

  it("'Registrar sessão' alterna a visibilidade do formulário de sessão", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, status: "active" }))),
      http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () => HttpResponse.json(makeTrack({ milestones: [] }))),
    );
    const user = userEvent.setup();
    renderDetail();
    await screen.findByText("Esta meta ainda não tem trilha definida.");

    expect(screen.queryByLabelText("Quantos minutos?")).not.toBeInTheDocument();

    // O toggle do cabeçalho é sempre o primeiro "Registrar sessão" no DOM —
    // o segundo só existe depois que o formulário abre (botão de submit dele).
    const toggle = () => screen.getAllByRole("button", { name: "Registrar sessão" })[0];

    await user.click(toggle());
    expect(screen.getByLabelText("Quantos minutos?")).toBeInTheDocument();

    await user.click(toggle());
    expect(screen.queryByLabelText("Quantos minutos?")).not.toBeInTheDocument();
  });
});
