import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeAction, makeTodayGoal } from "@/test/fixtures";
import { GoalTodayCard } from "./goal-card";

function renderCard(item = makeTodayGoal()) {
  return { ...renderRoute(<GoalTodayCard item={item} />), item };
}

describe("GoalTodayCard", () => {
  it("mostra título da meta, ação, estimativa e recentWin", () => {
    const item = makeTodayGoal({
      recentWin: "Concorrência subiu de 2 para 4.",
      action: makeAction({ title: "Escreva um worker pool", estimatedMin: 25, detail: "detalhe da ação" }),
    });
    renderCard(item);

    expect(screen.getByText(item.goal.title)).toBeInTheDocument();
    expect(screen.getByText("Escreva um worker pool")).toBeInTheDocument();
    expect(screen.getByText("~25 min")).toBeInTheDocument();
    expect(screen.getByText("detalhe da ação")).toBeInTheDocument();
    expect(screen.getByText("Concorrência subiu de 2 para 4.")).toBeInTheDocument();
  });

  it("não mostra o bloco de recentWin quando ele é null", () => {
    const item = makeTodayGoal({ recentWin: null });
    renderCard(item);
    expect(screen.queryByText(/subiu/)).not.toBeInTheDocument();
  });

  it("Feita: conclui a ação com sucesso", async () => {
    const item = makeTodayGoal();
    let requestBody: unknown;
    server.use(
      http.post(`${API_BASE}/actions/${item.action.id}/complete`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json({ completed: item.action, next: makeAction(), session: null });
      }),
    );
    const user = userEvent.setup();
    renderCard(item);

    await user.click(screen.getByRole("button", { name: "Feita" }));

    await waitFor(() => expect(requestBody).toEqual({}));
  });

  it("Feita: mostra erro se a conclusão falhar", async () => {
    const item = makeTodayGoal();
    server.use(
      http.post(`${API_BASE}/actions/${item.action.id}/complete`, () =>
        HttpResponse.json({ title: "Ação já concluída", status: 409 }, { status: 409 }),
      ),
    );
    const user = userEvent.setup();
    renderCard(item);

    await user.click(screen.getByRole("button", { name: "Feita" }));

    expect(await screen.findByText("Ação já concluída")).toBeInTheDocument();
  });

  it("Pulei: revela os motivos, e depois de escolher um volta pro estado normal (regressão do bug de 2026-08-09)", async () => {
    const item = makeTodayGoal();
    server.use(
      http.post(`${API_BASE}/actions/${item.action.id}/skip`, () => HttpResponse.json({ next: makeAction() })),
    );
    const user = userEvent.setup();
    renderCard(item);

    await user.click(screen.getByRole("button", { name: "Pulei" }));
    expect(screen.getByText("por quê?")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "sem tempo" }));

    // Bug original: showSkipReasons nunca resetava depois do sucesso, e a
    // tela ficava presa em "por quê?" pra sempre mesmo com a mutation ok.
    await waitFor(() => expect(screen.queryByText("por quê?")).not.toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Feita" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Pulei" })).toBeInTheDocument();
  });

  it("Pulei: cancelar volta pro estado normal sem chamar a API", async () => {
    const item = makeTodayGoal();
    const user = userEvent.setup();
    renderCard(item);

    await user.click(screen.getByRole("button", { name: "Pulei" }));
    await user.click(screen.getByRole("button", { name: "cancelar" }));

    expect(screen.queryByText("por quê?")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Feita" })).toBeInTheDocument();
  });

  it("só tenho 5 min: revela a variante mínima, e completar reseta o estado", async () => {
    const item = makeTodayGoal({ action: makeAction({ minimalVariant: "Releia e escreva o primeiro passo." }) });
    let requestBody: unknown;
    server.use(
      http.post(`${API_BASE}/actions/${item.action.id}/complete`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json({ completed: item.action, next: makeAction(), session: null });
      }),
    );
    const user = userEvent.setup();
    renderCard(item);

    await user.click(screen.getByRole("button", { name: "só tenho 5 min ▾" }));
    expect(screen.getByText("Releia e escreva o primeiro passo.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Feita (mínimo)" }));

    await waitFor(() => expect(requestBody).toEqual({ usedMinimalVariant: true }));
    await waitFor(() => expect(screen.queryByText("Releia e escreva o primeiro passo.")).not.toBeInTheDocument());
  });

  it("não mostra o link de 5 min quando a ação não tem variante mínima", () => {
    const item = makeTodayGoal({ action: makeAction({ minimalVariant: null }) });
    renderCard(item);
    expect(screen.queryByRole("button", { name: /5 min/ })).not.toBeInTheDocument();
  });
});
