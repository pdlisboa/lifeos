import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeProposal } from "@/test/fixtures";
import { TrackProposalView } from "./track-proposal";

const GOAL_ID = "goal-proposal";

function mockProposals(proposals: ReturnType<typeof makeProposal>[]) {
  server.use(http.get(`${API_BASE}/proposals`, () => HttpResponse.json(proposals)));
}

describe("TrackProposalView", () => {
  it("mostra carregando antes da resposta", () => {
    server.use(
      http.get(`${API_BASE}/proposals`, async () => {
        await new Promise((r) => setTimeout(r, 50));
        return HttpResponse.json([]);
      }),
    );
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);
    expect(screen.getByText("Carregando…")).toBeInTheDocument();
  });

  it("mostra erro quando a requisição falha", async () => {
    server.use(http.get(`${API_BASE}/proposals`, () => HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 })));
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);
    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
  });

  it("sem propostas pendentes, mostra mensagem própria", async () => {
    mockProposals([]);
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);
    expect(await screen.findByText("Nenhuma proposta de trilha pendente.")).toBeInTheDocument();
  });

  it("ignora propostas de outro kind", async () => {
    mockProposals([makeProposal({ kind: "level_change" })]);
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);
    expect(await screen.findByText("Nenhuma proposta de trilha pendente.")).toBeInTheDocument();
  });

  it("lista a proposta com rationale e os marcos do payload", async () => {
    mockProposals([
      makeProposal({
        rationale: "Você já sabe o básico de goroutines.",
        payload: {
          milestones: [
            {
              ordinal: 1,
              title: "Escreve um worker pool básico",
              completionCriteria: "worker pool sem vazar goroutine",
              competencyKeys: ["concurrency"],
              carriedOver: false,
              sourceLibraryTitle: null,
            },
            {
              ordinal: 2,
              title: "Marco já concluído",
              completionCriteria: "critério antigo",
              competencyKeys: [],
              carriedOver: true,
              sourceLibraryTitle: null,
            },
          ],
        },
      }),
    ]);
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);

    expect(await screen.findByText("Você já sabe o básico de goroutines.")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Escreve um worker pool básico")).toBeInTheDocument();
    expect(screen.getByDisplayValue("worker pool sem vazar goroutine")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Marco já concluído")).toBeInTheDocument();
    expect(screen.getByText("marco já concluído — copiado como está, intocável")).toBeInTheDocument();
  });

  it("aceita sem edições manda accept sem edits", async () => {
    const proposal = makeProposal({ id: "p-accept" });
    mockProposals([proposal]);
    let acceptedBody: unknown;
    server.use(
      http.post(`${API_BASE}/proposals/${proposal.id}/accept`, async ({ request }) => {
        acceptedBody = await request.json();
        return HttpResponse.json({ ...proposal, status: "accepted" });
      }),
    );
    const user = userEvent.setup();
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);

    await screen.findByText(proposal.rationale);
    await user.click(screen.getByRole("button", { name: "Aceitar" }));

    await vi.waitFor(() => expect(acceptedBody).toEqual({}));
  });

  it("editar um marco e aceitar manda os marcos editados em edits.milestones", async () => {
    const proposal = makeProposal({ id: "p-edit" });
    mockProposals([proposal]);
    let acceptedBody: { edits?: { milestones?: Array<{ title: string }> } } | undefined;
    server.use(
      http.post(`${API_BASE}/proposals/${proposal.id}/accept`, async ({ request }) => {
        acceptedBody = (await request.json()) as typeof acceptedBody;
        return HttpResponse.json({ ...proposal, status: "accepted" });
      }),
    );
    const user = userEvent.setup();
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);

    await screen.findByText(proposal.rationale);
    const titleInput = screen.getByDisplayValue("Escreve um worker pool básico com channels");
    await user.clear(titleInput);
    await user.type(titleInput, "Título editado por mim");

    expect(await screen.findByRole("button", { name: "Aceitar com edições" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Aceitar com edições" }));

    await vi.waitFor(() => expect(acceptedBody?.edits?.milestones?.[0]?.title).toBe("Título editado por mim"));
  });

  it("rejeitar abre o campo de motivo e envia ao confirmar", async () => {
    const proposal = makeProposal({ id: "p-reject" });
    mockProposals([proposal]);
    let rejectedBody: unknown;
    server.use(
      http.post(`${API_BASE}/proposals/${proposal.id}/reject`, async ({ request }) => {
        rejectedBody = await request.json();
        return HttpResponse.json({ ...proposal, status: "rejected" });
      }),
    );
    const user = userEvent.setup();
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);

    await screen.findByText(proposal.rationale);
    await user.click(screen.getByRole("button", { name: "rejeitar" }));

    const reasonField = screen.getByLabelText("Motivo da rejeição");
    await user.type(reasonField, "prefiro a trilha atual");
    await user.click(screen.getByRole("button", { name: "Confirmar rejeição" }));

    await vi.waitFor(() => expect(rejectedBody).toEqual({ reason: "prefiro a trilha atual" }));
  });

  it("mostra erro da mutação de aceite", async () => {
    const proposal = makeProposal({ id: "p-fail" });
    mockProposals([proposal]);
    server.use(
      http.post(`${API_BASE}/proposals/${proposal.id}/accept`, () =>
        HttpResponse.json({ title: "Proposta já foi resolvida", status: 409 }, { status: 409 }),
      ),
    );
    const user = userEvent.setup();
    renderRoute(<TrackProposalView goalId={GOAL_ID} />);

    await screen.findByText(proposal.rationale);
    await user.click(screen.getByRole("button", { name: "Aceitar" }));

    expect(await screen.findByText("Proposta já foi resolvida")).toBeInTheDocument();
  });
});
