import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeCompetency, makeDeltaCompetency, makeDeltaPanel, makeProjection } from "@/test/fixtures";
import { DeltaPanelView } from "./delta-panel";

const GOAL_ID = "goal-delta";

function mockDelta(panel: ReturnType<typeof makeDeltaPanel>) {
  server.use(
    http.get(`${API_BASE}/goals/${GOAL_ID}/delta`, () => HttpResponse.json(panel)),
    http.get(`${API_BASE}/goals/${GOAL_ID}/projection`, () => HttpResponse.json(makeProjection())),
  );
}

function mockProjection(projection: ReturnType<typeof makeProjection>) {
  server.use(http.get(`${API_BASE}/goals/${GOAL_ID}/projection`, () => HttpResponse.json(projection)));
}

function renderDelta() {
  return renderRoute(<DeltaPanelView goalId={GOAL_ID} />, {
    extraRoutes: [{ path: `/metas/${GOAL_ID}/evidencia`, element: <div>tela de evidência</div> }],
  });
}

describe("DeltaPanelView", () => {
  it("mostra carregando antes da resposta", () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/delta`, async () => {
        await new Promise((r) => setTimeout(r, 50));
        return HttpResponse.json(makeDeltaPanel());
      }),
    );
    renderDelta();
    expect(screen.getByText("Carregando…")).toBeInTheDocument();
  });

  it("mostra erro quando a requisição falha", async () => {
    server.use(http.get(`${API_BASE}/goals/${GOAL_ID}/delta`, () => HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 })));
    renderDelta();
    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
  });

  describe("fase baseline (sem evidência ainda)", () => {
    it("mostra o texto de ponto de partida, níveis 'você está aqui', e o link pra registrar a primeira evidência", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 2,
          totals: { evidenceCount: 0, milestonesDone: 0, sessionMinutes: 0 },
          competencies: [
            makeDeltaCompetency({ id: "c1", label: "Concorrência", level: 2 }),
            makeDeltaCompetency({ id: "c2", label: "Testes", level: null }),
          ],
        }),
      );
      renderDelta();

      expect(await screen.findByText(/Seu ponto de partida está registrado/)).toBeInTheDocument();
      expect(screen.getByText("Concorrência")).toBeInTheDocument();
      expect(screen.getByText("2 ← você está aqui")).toBeInTheDocument();
      expect(screen.getByText("Testes")).toBeInTheDocument();
      expect(screen.getByText("ainda não medimos")).toBeInTheDocument();
      expect(screen.getByRole("link", { name: "Registrar primeira evidência" })).toHaveAttribute(
        "href",
        `/metas/${GOAL_ID}/evidencia`,
      );
    });
  });

  describe("fase accumulating (evidência, mas < 3 semanas)", () => {
    it("mostra acúmulo (evidências, dias ativos, marcos) em vez de delta", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 10,
          totals: { evidenceCount: 4, milestonesDone: 1, sessionMinutes: 120 },
          competencies: [makeDeltaCompetency({ id: "c1", label: "Concorrência", level: 2 })],
        }),
      );
      renderDelta();

      expect(await screen.findByText(/Ainda sem delta confiável/)).toBeInTheDocument();
      expect(screen.getByText("4")).toBeInTheDocument();
      expect(screen.getByText("evidências")).toBeInTheDocument();
      expect(screen.getByText("10")).toBeInTheDocument();
      expect(screen.getByText("dias ativos")).toBeInTheDocument();
      expect(screen.getByText("1")).toBeInTheDocument();
      expect(screen.getByText("marcos concluídos")).toBeInTheDocument();
      // fase 2 mostra só o nível, sem seta/janela de delta
      expect(screen.getByText("2")).toBeInTheDocument();
    });
  });

  describe("fase delta (3+ semanas)", () => {
    it("mostra a frase-título (headline) quando presente", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 40,
          headline: "Há 60 dias você não usava context.Context.",
          totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 },
        }),
      );
      renderDelta();
      expect(await screen.findByText("Há 60 dias você não usava context.Context.")).toBeInTheDocument();
    });

    it("sem headline (Fatia 1, sem agente) não mostra bloco nenhum de frase", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 40,
          headline: null,
          totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 },
          competencies: [makeDeltaCompetency({ id: "c1", label: "Concorrência", level: 4, baselineLevel: 2, delta: 2, deltaWindowDays: 40 })],
        }),
      );
      renderDelta();
      await screen.findByText("Concorrência");
      expect(screen.queryByText(/^Há /)).not.toBeInTheDocument();
    });

    it("competência medida mostra faixa de → para e o delta com janela de dias", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 40,
          totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 },
          competencies: [
            makeDeltaCompetency({
              id: "c1",
              label: "Concorrência",
              level: 4,
              baselineLevel: 2,
              delta: 2,
              deltaWindowDays: 40,
              confidence: "high",
            }),
          ],
        }),
      );
      renderDelta();

      expect(await screen.findByText("2 → 4")).toBeInTheDocument();
      expect(screen.getByText("↑ 2 em 40 dias")).toBeInTheDocument();
    });

    it("competência não medida mostra 'ainda não medimos', sem barra de nível preenchida", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 40,
          totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 },
          competencies: [makeDeltaCompetency({ id: "c1", label: "Performance", level: null, confidence: "unknown" })],
        }),
      );
      renderDelta();

      expect(await screen.findByText("Performance")).toBeInTheDocument();
      // "ainda não medimos" aparece duas vezes nesse layout: no lugar do "de→para" e não há texto de delta
      expect(screen.getAllByText("ainda não medimos")).toHaveLength(1);
    });

    it("competência sem evidência recente (stale) mostra aviso de dias sem evidência, não o delta", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 40,
          totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 },
          competencies: [
            makeDeltaCompetency({
              id: "c1",
              label: "Interfaces e design",
              level: 3,
              baselineLevel: 3,
              confidence: "low",
              staleDays: 38,
            }),
          ],
        }),
      );
      renderDelta();

      expect(await screen.findByText("sem evidência há 38 dias ⟳")).toBeInTheDocument();
    });

    it("ordena por quem mais subiu primeiro (maior delta no topo)", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 40,
          totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 },
          competencies: [
            makeDeltaCompetency({ id: "c1", label: "Subiu pouco", level: 2, baselineLevel: 1, delta: 1 }),
            makeDeltaCompetency({ id: "c2", label: "Subiu muito", level: 4, baselineLevel: 1, delta: 3 }),
          ],
        }),
      );
      renderDelta();

      await screen.findByText("Subiu muito");
      const labels = screen.getAllByText(/Subiu/).map((el) => el.textContent);
      expect(labels).toEqual(["Subiu muito", "Subiu pouco"]);
    });

    it("rodapé mostra o reason quando a projeção está indisponível, sem inventar chegada", async () => {
      mockDelta(makeDeltaPanel({ daysActive: 40, totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 } }));
      mockProjection(makeProjection({ available: false, reason: "ainda coletando ritmo (2 de 3 semanas)" }));
      renderDelta();

      expect(await screen.findByText("ainda coletando ritmo (2 de 3 semanas)")).toBeInTheDocument();
    });

    it("rodapé mostra o ritmo real quando a projeção está disponível", async () => {
      mockDelta(makeDeltaPanel({ daysActive: 40, totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 } }));
      mockProjection(makeProjection({ available: true, reason: null, minutesPerWeek: 192 }));
      renderDelta();

      expect(await screen.findByText(/~3,2h\/semana/)).toBeInTheDocument();
    });

    it("clicar no nome de uma competência medida abre o gráfico temporal", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 40,
          totals: { evidenceCount: 12, milestonesDone: 2, sessionMinutes: 300 },
          competencies: [makeDeltaCompetency({ id: "c1", label: "Concorrência", level: 4, baselineLevel: 2, delta: 2 })],
        }),
      );
      server.use(http.get(`${API_BASE}/goals/${GOAL_ID}/competencies/c1/history`, () => HttpResponse.json([])));
      const user = userEvent.setup();
      renderDelta();

      const toggle = await screen.findByRole("button", { name: "Concorrência" });
      expect(toggle).toHaveAttribute("aria-expanded", "false");

      await user.click(toggle);
      expect(toggle).toHaveAttribute("aria-expanded", "true");
      expect(await screen.findByText("Nenhum evento de nível ainda.")).toBeInTheDocument();
    });
  });

  describe("definir nível manualmente (RN-04, competência ainda não medida)", () => {
    it("abre o formulário, confirma o nível, e o formulário fecha no sucesso", async () => {
      mockDelta(
        makeDeltaPanel({
          daysActive: 2,
          totals: { evidenceCount: 0, milestonesDone: 0, sessionMinutes: 0 },
          competencies: [makeDeltaCompetency({ id: "c1", label: "Performance", level: null })],
        }),
      );
      server.use(
        http.put(`${API_BASE}/goals/${GOAL_ID}/competencies/c1/level`, () =>
          HttpResponse.json(makeCompetency({ id: "c1", level: 3 })),
        ),
      );
      const user = userEvent.setup();
      renderDelta();

      await user.click(await screen.findByRole("button", { name: "definir nível" }));
      expect(screen.getByLabelText("Por que esse nível?")).toBeInTheDocument();

      await user.type(screen.getByLabelText("Por que esse nível?"), "consigo fazer sozinho, copiando exemplo");
      await user.click(screen.getByRole("button", { name: "Confirmar nível" }));

      await waitFor(() => expect(screen.queryByLabelText("Por que esse nível?")).not.toBeInTheDocument());
    });
  });
});
