import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeGoalSummary, makeProbe } from "@/test/fixtures";
import { BasicsStep } from "./basics-step";
import { packs } from "./packs";

describe("BasicsStep", () => {
  it("valida título com pelo menos 3 caracteres", async () => {
    const onCreated = vi.fn();
    const user = userEvent.setup();
    renderRoute(<BasicsStep onCreated={onCreated} />);

    await user.type(screen.getByLabelText("O que você quer aprender?"), "Go");
    await user.click(screen.getByRole("button", { name: "Continuar" }));

    expect(await screen.findByText("pelo menos 3 caracteres")).toBeInTheDocument();
    expect(onCreated).not.toHaveBeenCalled();
  });

  it("golang é o pack padrão selecionado", () => {
    renderRoute(<BasicsStep onCreated={vi.fn()} />);
    const golangRadio = screen.getByRole("radio", { name: "Go" });
    expect(golangRadio).toBeChecked();
  });

  it("cria a meta com o pack escolhido e chama onCreated com o id", async () => {
    const onCreated = vi.fn();
    let requestBody: unknown;
    server.use(
      http.post(`${API_BASE}/goals`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json(
          { goal: makeGoalSummary({ id: "novo-goal", status: "draft" }), probe: makeProbe() },
          { status: 201 },
        );
      }),
    );
    const user = userEvent.setup();
    renderRoute(<BasicsStep onCreated={onCreated} />);

    await user.type(screen.getByLabelText("O que você quer aprender?"), "Inglês técnico");
    await user.click(screen.getByRole("radio", { name: "Inglês" }));
    await user.click(screen.getByRole("button", { name: "Continuar" }));

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith("novo-goal"));
    expect(requestBody).toEqual({
      title: "Inglês técnico",
      archetype: packs.find((p) => p.packId === "english")!.archetype,
      packId: "english",
    });
  });

  it("mostra erro da API sem chamar onCreated", async () => {
    const onCreated = vi.fn();
    server.use(
      http.post(`${API_BASE}/goals`, () => HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 })),
    );
    const user = userEvent.setup();
    renderRoute(<BasicsStep onCreated={onCreated} />);

    await user.type(screen.getByLabelText("O que você quer aprender?"), "Go backend");
    await user.click(screen.getByRole("button", { name: "Continuar" }));

    expect(await screen.findByText("Erro interno")).toBeInTheDocument();
    expect(onCreated).not.toHaveBeenCalled();
  });
});
