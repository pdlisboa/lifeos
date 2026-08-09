import { describe, expect, it } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeCompetency, makeEvidenceCard, makeGoal } from "@/test/fixtures";
import { MuseuView } from "./museu";

const GOAL_ID = "goal-museu";

function mockGoal(competencies: ReturnType<typeof makeCompetency>[]) {
  server.use(http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, competencies }))));
}

function mockEvidence(items: ReturnType<typeof makeEvidenceCard>[]) {
  server.use(
    http.get(`${API_BASE}/goals/${GOAL_ID}/evidence`, () => HttpResponse.json({ items, hasMore: false, nextCursor: null })),
  );
}

function renderMuseu() {
  return renderRoute(<MuseuView goalId={GOAL_ID} />);
}

describe("MuseuView", () => {
  it("mostra o estado vazio com a explicação, sem ilustração genérica", async () => {
    mockGoal([]);
    mockEvidence([]);
    renderMuseu();
    expect(
      await screen.findByText(/Quando você registrar evidências, elas aparecem aqui em ordem/),
    ).toBeInTheDocument();
  });

  it("lista evidências em ordem cronológica, mais antiga primeiro", async () => {
    mockGoal([makeCompetency({ id: "c1", label: "Concorrência" })]);
    mockEvidence([
      makeEvidenceCard({ id: "e1", title: "primeira", body: "v1", createdAt: "2026-06-12T10:00:00Z" }),
      makeEvidenceCard({ id: "e2", title: "segunda", body: "v2", createdAt: "2026-08-08T10:00:00Z" }),
    ]);
    renderMuseu();

    const items = await screen.findAllByRole("listitem");
    expect(items).toHaveLength(2);
    expect(within(items[0]).getByText("primeira")).toBeInTheDocument();
    expect(within(items[1]).getByText("segunda")).toBeInTheDocument();
  });

  it("botão comparar fica desabilitado sem competência selecionada", async () => {
    mockGoal([makeCompetency({ id: "c1", label: "Concorrência" })]);
    mockEvidence([makeEvidenceCard({ id: "e1" }), makeEvidenceCard({ id: "e2" })]);
    renderMuseu();

    const compareButton = await screen.findByRole("button", { name: "comparar ⇄" });
    expect(compareButton).toBeDisabled();
  });

  it("filtrar por competência habilita a comparação e chama a API com competencyId", async () => {
    mockGoal([makeCompetency({ id: "c1", label: "Concorrência" })]);
    let requestedUrl = "";
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/evidence`, ({ request }) => {
        requestedUrl = request.url;
        return HttpResponse.json({
          items: [
            makeEvidenceCard({ id: "e1", body: "v1", createdAt: "2026-06-12T10:00:00Z", levelsAtTime: { c1: 2 } }),
            makeEvidenceCard({ id: "e2", body: "v2", createdAt: "2026-08-08T10:00:00Z", levelsAtTime: { c1: 4 } }),
          ],
          hasMore: false,
          nextCursor: null,
        });
      }),
    );
    const user = userEvent.setup();
    renderMuseu();

    await screen.findAllByRole("listitem");
    await user.selectOptions(screen.getByLabelText(/Filtrar/), "c1");

    await waitFor(() => expect(requestedUrl).toContain("competencyId=c1"));
    expect(await screen.findByRole("button", { name: "comparar ⇄" })).toBeEnabled();
  });

  it("comparação lado a lado sugere a mais antiga contra a mais recente, com o nível da competência", async () => {
    mockGoal([makeCompetency({ id: "c1", label: "Concorrência" })]);
    mockEvidence([
      makeEvidenceCard({ id: "e1", body: "func main() { v1 }", createdAt: "2026-06-12T10:00:00Z", levelsAtTime: { c1: 2 } }),
      makeEvidenceCard({ id: "e2", body: "func main() { v2 }", createdAt: "2026-08-08T10:00:00Z", levelsAtTime: { c1: 4 } }),
    ]);
    const user = userEvent.setup();
    renderMuseu();

    await screen.findAllByRole("listitem");
    await user.selectOptions(screen.getByLabelText(/Filtrar/), "c1");
    await user.click(await screen.findByRole("button", { name: "comparar ⇄" }));

    expect(screen.getByText("func main() { v1 }")).toBeInTheDocument();
    expect(screen.getByText("func main() { v2 }")).toBeInTheDocument();
    expect(screen.getByText("Concorrência: nível 2")).toBeInTheDocument();
    expect(screen.getByText("Concorrência: nível 4")).toBeInTheDocument();
  });

  it("mostra 'nível não registrado' quando a evidência comparada não tem levelsAtTime pra essa competência", async () => {
    mockGoal([makeCompetency({ id: "c1", label: "Concorrência" })]);
    mockEvidence([
      makeEvidenceCard({ id: "e1", body: "v1", createdAt: "2026-06-12T10:00:00Z", levelsAtTime: {} }),
      makeEvidenceCard({ id: "e2", body: "v2", createdAt: "2026-08-08T10:00:00Z", levelsAtTime: {} }),
    ]);
    const user = userEvent.setup();
    renderMuseu();

    await screen.findAllByRole("listitem");
    await user.selectOptions(screen.getByLabelText(/Filtrar/), "c1");
    await user.click(await screen.findByRole("button", { name: "comparar ⇄" }));

    expect(screen.getAllByText(/nível não registrado nesse momento/)).toHaveLength(2);
  });

  it("mostra erro quando a listagem falha", async () => {
    mockGoal([]);
    server.use(
      http.get(`${API_BASE}/goals/${GOAL_ID}/evidence`, () =>
        HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 }),
      ),
    );
    renderMuseu();
    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
  });
});
