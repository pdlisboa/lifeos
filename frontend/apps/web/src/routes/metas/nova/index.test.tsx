import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeGoal, makeGoalSummary, makeProbe, makeProbeTurn } from "@/test/fixtures";
import { NovaMetaRoute } from "./index";

describe("NovaMetaRoute (wizard completo)", () => {
  it("cria meta → responde sondagem até o fim → ativa", async () => {
    const GOAL_ID = "goal-wizard";
    const turn = makeProbeTurn({ id: "turn-1", question: "Já mexeu com goroutines?" });

    server.use(
      http.post(`${API_BASE}/goals`, () =>
        HttpResponse.json(
          { goal: makeGoalSummary({ id: GOAL_ID, status: "draft" }), probe: makeProbe({ currentQuestion: turn }) },
          { status: 201 },
        ),
      ),
      http.get(`${API_BASE}/goals/${GOAL_ID}/probe`, () => HttpResponse.json(makeProbe({ currentQuestion: turn }))),
      http.post(`${API_BASE}/goals/${GOAL_ID}/probe/answer`, () =>
        HttpResponse.json({ probe: makeProbe({ status: "completed", currentQuestion: null }), nextQuestion: null }),
      ),
      http.patch(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID }))),
      http.post(`${API_BASE}/goals/${GOAL_ID}/activate`, () =>
        HttpResponse.json({ goal: makeGoal({ id: GOAL_ID, status: "active" }), action: null }),
      ),
    );

    const user = userEvent.setup();
    renderRoute(<NovaMetaRoute />, {
      extraRoutes: [{ path: `/metas/${GOAL_ID}`, element: <div>tela de detalhe da meta</div> }],
    });

    // Passo 1: básico
    await user.type(screen.getByLabelText("O que você quer aprender?"), "Go backend");
    await user.click(screen.getByRole("button", { name: "Continuar" }));

    // Passo 2: sondagem
    expect(await screen.findByText("Já mexeu com goroutines?")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Sim" }));

    // Passo 3: definição de pronto + ativação
    expect(await screen.findByText("Quando você vai saber que chegou lá?")).toBeInTheDocument();
    await user.type(screen.getByRole("textbox"), "escrever e testar um serviço HTTP concorrente");
    await user.click(screen.getByRole("button", { name: "Ativar meta" }));

    await waitFor(() => expect(screen.getByText("tela de detalhe da meta")).toBeInTheDocument());
  });

  it("cria meta → pula sondagem → deixa definição pra depois", async () => {
    const GOAL_ID = "goal-wizard-skip";
    server.use(
      http.post(`${API_BASE}/goals`, () =>
        HttpResponse.json({ goal: makeGoalSummary({ id: GOAL_ID, status: "draft" }), probe: makeProbe() }, { status: 201 }),
      ),
      http.get(`${API_BASE}/goals/${GOAL_ID}/probe`, () => HttpResponse.json(makeProbe())),
      http.post(`${API_BASE}/goals/${GOAL_ID}/probe/skip`, () => HttpResponse.json(makeProbe({ status: "skipped", currentQuestion: null }))),
    );

    const user = userEvent.setup();
    renderRoute(<NovaMetaRoute />, {
      extraRoutes: [{ path: `/metas/${GOAL_ID}`, element: <div>tela de detalhe da meta</div> }],
    });

    await user.type(screen.getByLabelText("O que você quer aprender?"), "Go backend");
    await user.click(screen.getByRole("button", { name: "Continuar" }));

    await screen.findByText(/pergunta/);
    await user.click(screen.getByRole("button", { name: "Já sei o suficiente, quero começar →" }));

    expect(await screen.findByText("Quando você vai saber que chegou lá?")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Depois eu escrevo" }));

    await waitFor(() => expect(screen.getByText("tela de detalhe da meta")).toBeInTheDocument());
  });
});
