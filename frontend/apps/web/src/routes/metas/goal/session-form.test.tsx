import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeSession } from "@/test/fixtures";
import { SessionForm } from "./session-form";

const GOAL_ID = "goal-session";

describe("SessionForm", () => {
  it("registra sessão com a duração informada e chama onDone", async () => {
    const onDone = vi.fn();
    let requestBody: any;
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/sessions`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json(makeSession(), { status: 201 });
      }),
    );
    const user = userEvent.setup();
    renderRoute(<SessionForm goalId={GOAL_ID} onDone={onDone} />);

    const durationInput = screen.getByLabelText("Quantos minutos?");
    await user.clear(durationInput);
    await user.type(durationInput, "40");
    await user.type(screen.getByLabelText("Nota (opcional)"), "sessão de teste");
    await user.click(screen.getByRole("button", { name: "Registrar sessão" }));

    expect(onDone).toHaveBeenCalledTimes(1);
    expect(requestBody).toMatchObject({ durationMin: 40, note: "sessão de teste" });
    expect(requestBody.startedAt).toEqual(expect.any(String));
  });

  it("nota vazia vira undefined, não string vazia", async () => {
    let requestBody: any;
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/sessions`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json(makeSession(), { status: 201 });
      }),
    );
    const user = userEvent.setup();
    renderRoute(<SessionForm goalId={GOAL_ID} onDone={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Registrar sessão" }));

    expect(requestBody.note).toBeUndefined();
  });

  it("mostra erro e não chama onDone quando a API falha", async () => {
    const onDone = vi.fn();
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/sessions`, () => HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 })),
    );
    const user = userEvent.setup();
    renderRoute(<SessionForm goalId={GOAL_ID} onDone={onDone} />);

    await user.click(screen.getByRole("button", { name: "Registrar sessão" }));

    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
    expect(onDone).not.toHaveBeenCalled();
  });

  it("cancelar chama onDone sem registrar nada", async () => {
    const onDone = vi.fn();
    const user = userEvent.setup();
    renderRoute(<SessionForm goalId={GOAL_ID} onDone={onDone} />);

    await user.click(screen.getByRole("button", { name: "cancelar" }));

    expect(onDone).toHaveBeenCalledTimes(1);
  });
});
