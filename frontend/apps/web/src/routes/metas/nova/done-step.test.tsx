import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeGoal, makeGoalSummary } from "@/test/fixtures";
import { DoneStep } from "./done-step";

const GOAL_ID = "goal-done";

function renderDone() {
  return renderRoute(<DoneStep goalId={GOAL_ID} />, {
    extraRoutes: [{ path: `/metas/${GOAL_ID}`, element: <div>tela de detalhe da meta</div> }],
  });
}

describe("DoneStep", () => {
  it("'Depois eu escrevo' navega direto pro detalhe sem chamar a API", async () => {
    const user = userEvent.setup();
    renderDone();

    await user.click(screen.getByRole("button", { name: "Depois eu escrevo" }));

    expect(await screen.findByText("tela de detalhe da meta")).toBeInTheDocument();
  });

  it("ativa com definição de pronto: faz PATCH e depois activate, navega no sucesso", async () => {
    let patchBody: unknown;
    server.use(
      http.patch(`${API_BASE}/goals/${GOAL_ID}`, async ({ request }) => {
        patchBody = await request.json();
        return HttpResponse.json(makeGoal({ id: GOAL_ID }));
      }),
      http.post(`${API_BASE}/goals/${GOAL_ID}/activate`, () =>
        HttpResponse.json({ goal: makeGoal({ id: GOAL_ID, status: "active" }), action: null }),
      ),
    );
    const user = userEvent.setup();
    renderDone();

    await user.type(screen.getByRole("textbox"), "escrever e testar um serviço HTTP concorrente");
    await user.click(screen.getByRole("button", { name: "Ativar meta" }));

    expect(await screen.findByText("tela de detalhe da meta")).toBeInTheDocument();
    expect(patchBody).toEqual({ definitionOfDone: "escrever e testar um serviço HTTP concorrente" });
  });

  it("ativa sem digitar nada: não chama PATCH, só activate", async () => {
    let patchCalled = false;
    server.use(
      http.patch(`${API_BASE}/goals/${GOAL_ID}`, () => {
        patchCalled = true;
        return HttpResponse.json(makeGoal({ id: GOAL_ID }));
      }),
      http.post(`${API_BASE}/goals/${GOAL_ID}/activate`, () =>
        HttpResponse.json({ goal: makeGoal({ id: GOAL_ID, status: "active" }), action: null }),
      ),
    );
    const user = userEvent.setup();
    renderDone();

    await user.click(screen.getByRole("button", { name: "Ativar meta" }));

    await waitFor(() => expect(screen.getByText("tela de detalhe da meta")).toBeInTheDocument());
    expect(patchCalled).toBe(false);
  });

  it("RN-02: mostra seletor de metas ativas e reativa com pauseGoalId ao escolher uma", async () => {
    const activeGoals = [makeGoalSummary({ id: "outra-meta", title: "Meta velha" })];
    let activateCalls: unknown[] = [];
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/activate`, async ({ request }) => {
        const body = await request.json();
        activateCalls.push(body);
        if (activateCalls.length === 1) {
          return HttpResponse.json(
            { title: "Você já tem 3 metas ativas", status: 409, rule: "RN-02", activeGoals },
            { status: 409 },
          );
        }
        return HttpResponse.json({ goal: makeGoal({ id: GOAL_ID, status: "active" }), action: null });
      }),
    );
    const user = userEvent.setup();
    renderDone();

    await user.click(screen.getByRole("button", { name: "Ativar meta" }));

    expect(await screen.findByText("Você já tem 3 metas ativas. Qual pausar para ativar esta?")).toBeInTheDocument();
    const pauseButton = screen.getByRole("button", { name: 'pausar "Meta velha"' });

    await user.click(pauseButton);

    expect(await screen.findByText("tela de detalhe da meta")).toBeInTheDocument();
    expect(activateCalls).toEqual([{}, { pauseGoalId: "outra-meta" }]);
  });

  it("erro genérico (não RN-02) mostra ProblemError e não navega", async () => {
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/activate`, () =>
        HttpResponse.json({ title: "Falta a definição de pronto", status: 409, rule: "RN-01" }, { status: 409 }),
      ),
    );
    const user = userEvent.setup();
    renderDone();

    await user.click(screen.getByRole("button", { name: "Ativar meta" }));

    expect(await screen.findByText("Falta a definição de pronto")).toBeInTheDocument();
    expect(screen.queryByText("tela de detalhe da meta")).not.toBeInTheDocument();
  });
});
