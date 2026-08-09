import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeEvidence, makeLevelEvent } from "@/test/fixtures";
import { CompetencyHistoryChart } from "./competency-history-chart";

const GOAL_ID = "goal-1";
const COMP_ID = "comp-1";

function mockHistory(events: ReturnType<typeof makeLevelEvent>[]) {
  server.use(
    http.get(`${API_BASE}/goals/${GOAL_ID}/competencies/${COMP_ID}/history`, () => HttpResponse.json(events)),
  );
}

function renderChart() {
  return renderRoute(<CompetencyHistoryChart goalId={GOAL_ID} competencyId={COMP_ID} />);
}

describe("CompetencyHistoryChart", () => {
  it("mostra estado vazio quando não há evento nenhum", async () => {
    mockHistory([]);
    renderChart();
    expect(await screen.findByText("Nenhum evento de nível ainda.")).toBeInTheDocument();
  });

  it("mostra erro quando a requisição falha", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/competencies/${COMP_ID}/history`, () =>
        HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 }),
      ),
    );
    renderChart();
    expect(await screen.findByText("Não deu pra carregar o histórico agora.")).toBeInTheDocument();
  });

  it("lista cada evento como uma linha clicável com a faixa de → para", async () => {
    mockHistory([
      makeLevelEvent({ id: "e1", fromLevel: 2, toLevel: 3, occurredAt: "2026-06-12T10:00:00Z", rationale: "primeira subida" }),
      makeLevelEvent({ id: "e2", fromLevel: 3, toLevel: 4, occurredAt: "2026-08-08T10:00:00Z", rationale: "segunda subida" }),
    ]);
    renderChart();

    expect(await screen.findByText("2 → 3")).toBeInTheDocument();
    expect(screen.getByText("3 → 4")).toBeInTheDocument();
    expect(screen.getByText("12/jun")).toBeInTheDocument();
    expect(screen.getByText("08/ago")).toBeInTheDocument();
  });

  it("clicar num ponto expande o rationale, sem link de evidência quando não há", async () => {
    mockHistory([makeLevelEvent({ id: "e1", toLevel: 2, rationale: "sondagem inicial", evidenceId: null })]);
    const user = userEvent.setup();
    renderChart();

    const row = await screen.findByRole("button", { expanded: false });
    expect(screen.queryByText("sondagem inicial")).not.toBeInTheDocument();

    await user.click(row);
    expect(screen.getByText("sondagem inicial")).toBeInTheDocument();
    expect(screen.queryByText("ver evidência")).not.toBeInTheDocument();
  });

  it("com evidenceId, mostra 'ver evidência' e carrega o conteúdo ao clicar", async () => {
    mockHistory([
      makeLevelEvent({
        id: "e1",
        fromLevel: 2,
        toLevel: 4,
        rationale: "usei errgroup com cancelamento correto",
        evidenceId: "ev-1",
      }),
    ]);
    server.use(
      http.get(`${API_BASE}/evidence/ev-1`, () =>
        HttpResponse.json(makeEvidence({ id: "ev-1", body: "func fetchAll(ctx context.Context) error { ... }" })),
      ),
    );
    const user = userEvent.setup();
    renderChart();

    await user.click(await screen.findByRole("button", { expanded: false }));
    const verEvidencia = screen.getByRole("button", { name: "ver evidência" });
    await user.click(verEvidencia);

    await waitFor(() =>
      expect(screen.getByText("func fetchAll(ctx context.Context) error { ... }")).toBeInTheDocument(),
    );
  });
});
