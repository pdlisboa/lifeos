import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeCompetency } from "@/test/fixtures";
import { SetLevelForm } from "./set-level-form";

const GOAL_ID = "goal-setlevel";
const COMPETENCY_ID = "comp-1";

describe("SetLevelForm", () => {
  it("nível 2 vem selecionado por padrão", () => {
    renderRoute(<SetLevelForm goalId={GOAL_ID} competencyId={COMPETENCY_ID} onDone={vi.fn()} />);
    expect(screen.getByRole("button", { name: "2" })).toHaveAttribute("aria-pressed", "true");
  });

  it("desabilita 'Confirmar nível' até a justificativa ter pelo menos 3 caracteres", async () => {
    const user = userEvent.setup();
    renderRoute(<SetLevelForm goalId={GOAL_ID} competencyId={COMPETENCY_ID} onDone={vi.fn()} />);

    expect(screen.getByRole("button", { name: "Confirmar nível" })).toBeDisabled();

    await user.type(screen.getByLabelText("Por que esse nível?"), "ok");
    expect(screen.getByRole("button", { name: "Confirmar nível" })).toBeDisabled();

    await user.type(screen.getByLabelText("Por que esse nível?"), "ok mais um pouco");
    expect(screen.getByRole("button", { name: "Confirmar nível" })).toBeEnabled();
  });

  it("confirma com o nível escolhido e a justificativa, e chama onDone no sucesso", async () => {
    const onDone = vi.fn();
    let requestBody: any;
    server.use(
      http.put(`${API_BASE}/goals/${GOAL_ID}/competencies/${COMPETENCY_ID}/level`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json(makeCompetency({ id: COMPETENCY_ID, level: 4 }));
      }),
    );
    const user = userEvent.setup();
    renderRoute(<SetLevelForm goalId={GOAL_ID} competencyId={COMPETENCY_ID} onDone={onDone} />);

    await user.click(screen.getByRole("button", { name: "4" }));
    await user.type(screen.getByLabelText("Por que esse nível?"), "escrevo worker pool copiando exemplo");
    await user.click(screen.getByRole("button", { name: "Confirmar nível" }));

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(requestBody).toEqual({ level: 4, rationale: "escrevo worker pool copiando exemplo" });
  });

  it("mostra erro e não chama onDone quando a API rejeita", async () => {
    const onDone = vi.fn();
    server.use(
      http.put(`${API_BASE}/goals/${GOAL_ID}/competencies/${COMPETENCY_ID}/level`, () =>
        HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 }),
      ),
    );
    const user = userEvent.setup();
    renderRoute(<SetLevelForm goalId={GOAL_ID} competencyId={COMPETENCY_ID} onDone={onDone} />);

    await user.type(screen.getByLabelText("Por que esse nível?"), "justificativa válida");
    await user.click(screen.getByRole("button", { name: "Confirmar nível" }));

    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
    expect(onDone).not.toHaveBeenCalled();
  });

  it("cancelar chama onDone sem enviar nada", async () => {
    const onDone = vi.fn();
    const user = userEvent.setup();
    renderRoute(<SetLevelForm goalId={GOAL_ID} competencyId={COMPETENCY_ID} onDone={onDone} />);

    await user.click(screen.getByRole("button", { name: "cancelar" }));

    expect(onDone).toHaveBeenCalledTimes(1);
  });
});
