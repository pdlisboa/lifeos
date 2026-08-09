import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeProbe, makeProbeTurn } from "@/test/fixtures";
import { ProbeStep } from "./probe-step";

const GOAL_ID = "goal-probe";

describe("ProbeStep", () => {
  it("mostra a pergunta atual e a contagem 'pergunta X de Y'", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/probe`, () =>
        HttpResponse.json(makeProbe({ askedCount: 1, currentQuestion: makeProbeTurn({ question: "Já mexeu com goroutines?" }) })),
      ),
    );
    renderRoute(<ProbeStep goalId={GOAL_ID} onDone={vi.fn()} />);

    expect(await screen.findByText("Já mexeu com goroutines?")).toBeInTheDocument();
    expect(screen.getByText("pergunta 2 de 5")).toBeInTheDocument();
  });

  it("mostra erro quando a busca da sondagem falha", async () => {
    server.use(http.get(`${API_BASE}/goals/${GOAL_ID}/probe`, () => HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 })));
    renderRoute(<ProbeStep goalId={GOAL_ID} onDone={vi.fn()} />);
    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
  });

  it("responder e receber outra pergunta: continua na sondagem, não chama onDone", async () => {
    const onDone = vi.fn();
    const turn1 = makeProbeTurn({ id: "turn-1", question: "Já mexeu com goroutines?" });
    const turn2 = makeProbeTurn({ id: "turn-2", question: "Já usou channels?" });
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/probe`, () => HttpResponse.json(makeProbe({ currentQuestion: turn1 }))),
      http.post(`${API_BASE}/goals/${GOAL_ID}/probe/answer`, () =>
        HttpResponse.json({ probe: makeProbe({ askedCount: 2, currentQuestion: turn2 }), nextQuestion: turn2 }),
      ),
    );
    const user = userEvent.setup();
    renderRoute(<ProbeStep goalId={GOAL_ID} onDone={onDone} />);

    await screen.findByText("Já mexeu com goroutines?");
    await user.click(screen.getByRole("button", { name: "Sim" }));

    expect(await screen.findByText("Já usou channels?")).toBeInTheDocument();
    expect(onDone).not.toHaveBeenCalled();
  });

  it("responder a última pergunta (nextQuestion null) chama onDone", async () => {
    const onDone = vi.fn();
    const turn1 = makeProbeTurn({ id: "turn-1", question: "Já mexeu com goroutines?" });
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/probe`, () => HttpResponse.json(makeProbe({ currentQuestion: turn1 }))),
      http.post(`${API_BASE}/goals/${GOAL_ID}/probe/answer`, () =>
        HttpResponse.json({ probe: makeProbe({ status: "completed", currentQuestion: null }), nextQuestion: null }),
      ),
    );
    const user = userEvent.setup();
    renderRoute(<ProbeStep goalId={GOAL_ID} onDone={onDone} />);

    await screen.findByText("Já mexeu com goroutines?");
    await user.click(screen.getByRole("button", { name: "Mais ou menos" }));

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
  });

  it("pular a sondagem chama skip e depois onDone", async () => {
    const onDone = vi.fn();
    let skipCalled = false;
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/probe`, () => HttpResponse.json(makeProbe())),
      http.post(`${API_BASE}/goals/${GOAL_ID}/probe/skip`, () => {
        skipCalled = true;
        return HttpResponse.json(makeProbe({ status: "skipped", currentQuestion: null }));
      }),
    );
    const user = userEvent.setup();
    renderRoute(<ProbeStep goalId={GOAL_ID} onDone={onDone} />);

    await screen.findByText(/pergunta/);
    await user.click(screen.getByRole("button", { name: "Já sei o suficiente, quero começar →" }));

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(skipCalled).toBe(true);
  });

  it("sem pergunta atual (sondagem já encerrada) chama onDone sem mostrar nada", async () => {
    const onDone = vi.fn();
    server.use(http.get(`${API_BASE}/goals/${GOAL_ID}/probe`, () => HttpResponse.json(makeProbe({ status: "completed", currentQuestion: null }))));
    renderRoute(<ProbeStep goalId={GOAL_ID} onDone={onDone} />);

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
  });
});
