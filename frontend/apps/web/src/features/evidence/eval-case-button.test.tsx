import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { EvalCaseButton } from "./eval-case-button";

const GOAL_ID = "goal-eval";
const EVIDENCE_ID = "evidence-eval";

const competencies = [
  { id: "c1", label: "Concorrência" },
  { id: "c2", label: "Testes" },
];

describe("EvalCaseButton", () => {
  it("mostra o rótulo discreto de marcar quando não há caso ainda", () => {
    renderRoute(
      <EvalCaseButton goalId={GOAL_ID} evidenceId={EVIDENCE_ID} competencies={competencies} evalCase={null} />,
    );
    expect(screen.getByRole("button", { name: "usar como caso de eval" })).toBeInTheDocument();
  });

  it("mostra o rótulo de editar quando já existe um caso marcado", () => {
    renderRoute(
      <EvalCaseButton
        goalId={GOAL_ID}
        evidenceId={EVIDENCE_ID}
        competencies={competencies}
        evalCase={{ note: "nota existente", scores: [{ competencyId: "c1", level: 2 }] }}
      />,
    );
    expect(screen.getByRole("button", { name: "caso de eval ✓ — editar" })).toBeInTheDocument();
  });

  it("botão de salvar fica desabilitado sem nota ou sem nenhum nível escolhido", async () => {
    const user = userEvent.setup();
    renderRoute(
      <EvalCaseButton goalId={GOAL_ID} evidenceId={EVIDENCE_ID} competencies={competencies} evalCase={null} />,
    );
    await user.click(screen.getByRole("button", { name: "usar como caso de eval" }));

    expect(screen.getByRole("button", { name: "salvar caso de eval" })).toBeDisabled();

    await user.type(screen.getByLabelText("Por que esse caso importa"), "gabarito claro");
    expect(screen.getByRole("button", { name: "salvar caso de eval" })).toBeDisabled();

    await user.selectOptions(screen.getByLabelText("Nível para Concorrência"), "3");
    expect(screen.getByRole("button", { name: "salvar caso de eval" })).toBeEnabled();
  });

  it("salva a marcação enviando só as competências com nível escolhido", async () => {
    let requestBody: any;
    server.use(
      http.post(`${API_BASE}/evidence/${EVIDENCE_ID}/eval-case`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json({ id: EVIDENCE_ID, kind: "code_snippet", createdAt: "2026-08-12T12:00:00Z" });
      }),
    );
    const user = userEvent.setup();
    renderRoute(
      <EvalCaseButton goalId={GOAL_ID} evidenceId={EVIDENCE_ID} competencies={competencies} evalCase={null} />,
    );
    await user.click(screen.getByRole("button", { name: "usar como caso de eval" }));
    await user.type(screen.getByLabelText("Por que esse caso importa"), "gabarito claro");
    await user.selectOptions(screen.getByLabelText("Nível para Concorrência"), "3");
    await user.click(screen.getByRole("button", { name: "salvar caso de eval" }));

    await screen.findByRole("button", { name: "caso de eval ✓ — editar" });
    expect(requestBody).toEqual({ note: "gabarito claro", scores: [{ competencyId: "c1", level: 3 }] });
  });

  it("cancelar fecha o formulário sem chamar a API", async () => {
    const user = userEvent.setup();
    renderRoute(
      <EvalCaseButton goalId={GOAL_ID} evidenceId={EVIDENCE_ID} competencies={competencies} evalCase={null} />,
    );
    await user.click(screen.getByRole("button", { name: "usar como caso de eval" }));
    await user.click(screen.getByRole("button", { name: "cancelar" }));

    expect(screen.getByRole("button", { name: "usar como caso de eval" })).toBeInTheDocument();
  });

  it("desmarcar remove o caso e volta ao rótulo padrão", async () => {
    server.use(
      http.delete(`${API_BASE}/evidence/${EVIDENCE_ID}/eval-case`, () => new HttpResponse(null, { status: 204 })),
    );
    const user = userEvent.setup();
    renderRoute(
      <EvalCaseButton
        goalId={GOAL_ID}
        evidenceId={EVIDENCE_ID}
        competencies={competencies}
        evalCase={{ note: "nota existente", scores: [{ competencyId: "c1", level: 2 }] }}
      />,
    );
    await user.click(screen.getByRole("button", { name: "caso de eval ✓ — editar" }));
    await user.click(screen.getByRole("button", { name: "desmarcar" }));

    await screen.findByRole("button", { name: "usar como caso de eval" });
  });

  it("mostra erro da API sem fechar o formulário", async () => {
    server.use(
      http.post(`${API_BASE}/evidence/${EVIDENCE_ID}/eval-case`, () =>
        HttpResponse.json({ title: "algo deu errado", status: 400 }, { status: 400 }),
      ),
    );
    const user = userEvent.setup();
    renderRoute(
      <EvalCaseButton goalId={GOAL_ID} evidenceId={EVIDENCE_ID} competencies={competencies} evalCase={null} />,
    );
    await user.click(screen.getByRole("button", { name: "usar como caso de eval" }));
    await user.type(screen.getByLabelText("Por que esse caso importa"), "gabarito claro");
    await user.selectOptions(screen.getByLabelText("Nível para Concorrência"), "3");
    await user.click(screen.getByRole("button", { name: "salvar caso de eval" }));

    expect(await screen.findByText("algo deu errado")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "salvar caso de eval" })).toBeInTheDocument();
  });

  it("com highlightCompetencyIds, mostra as competências tocadas primeiro", async () => {
    const user = userEvent.setup();
    renderRoute(
      <EvalCaseButton
        goalId={GOAL_ID}
        evidenceId={EVIDENCE_ID}
        competencies={competencies}
        evalCase={null}
        highlightCompetencyIds={["c2"]}
      />,
    );
    await user.click(screen.getByRole("button", { name: "usar como caso de eval" }));

    const labels = screen.getAllByLabelText(/Nível para/).map((el) => el.getAttribute("aria-label"));
    expect(labels).toEqual(["Nível para Testes", "Nível para Concorrência"]);
  });
});
