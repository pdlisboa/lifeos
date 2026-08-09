import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeMilestone, makeTrack } from "@/test/fixtures";
import { TrackView } from "./track-view";

const GOAL_ID = "goal-track";

describe("TrackView", () => {
  it("mostra carregando antes da resposta", () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/track`, async () => {
        await new Promise((r) => setTimeout(r, 50));
        return HttpResponse.json(makeTrack());
      }),
    );
    renderRoute(<TrackView goalId={GOAL_ID} />);
    expect(screen.getByText("Carregando…")).toBeInTheDocument();
  });

  it("mostra erro quando a requisição falha", async () => {
    server.use(http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () => HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 })));
    renderRoute(<TrackView goalId={GOAL_ID} />);
    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
  });

  it("trilha vazia mostra mensagem própria", async () => {
    server.use(http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () => HttpResponse.json(makeTrack({ milestones: [] }))));
    renderRoute(<TrackView goalId={GOAL_ID} />);
    expect(await screen.findByText("Esta meta ainda não tem trilha definida.")).toBeInTheDocument();
  });

  it("lista marcos com título, critério e rótulo de status traduzido", async () => {
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/track`, () =>
        HttpResponse.json(
          makeTrack({
            milestones: [
              makeMilestone({ id: "m1", title: "CLI com testes", status: "current", completionCriteria: "roda com go run" }),
              makeMilestone({ id: "m2", title: "Interfaces", status: "locked", completionCriteria: "" }),
              makeMilestone({ id: "m3", title: "Servidor HTTP", status: "completed" }),
            ],
          }),
        ),
      ),
    );
    renderRoute(<TrackView goalId={GOAL_ID} />);

    expect(await screen.findByText("CLI com testes")).toBeInTheDocument();
    expect(screen.getByText("atual")).toBeInTheDocument();
    expect(screen.getByText("roda com go run")).toBeInTheDocument();

    expect(screen.getByText("Interfaces")).toBeInTheDocument();
    expect(screen.getByText("bloqueado")).toBeInTheDocument();

    expect(screen.getByText("Servidor HTTP")).toBeInTheDocument();
    expect(screen.getByText("concluído")).toBeInTheDocument();
  });
});
