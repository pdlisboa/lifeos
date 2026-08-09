import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeAction, makeEvidence, makeGoal } from "@/test/fixtures";
import { EvidenciaRoute } from "./index";

const GOAL_ID = "goal-evidencia";

function setupGoalAndAction(overrides: { hasAction?: boolean } = {}) {
  server.use(
    http.get(`${API_BASE}/goals/${GOAL_ID}`, () => HttpResponse.json(makeGoal({ id: GOAL_ID, title: "Go backend" }))),
    http.get(`${API_BASE}/goals/${GOAL_ID}/action`, () =>
      overrides.hasAction === false
        ? HttpResponse.json({ title: "Não encontrado", status: 404 }, { status: 404 })
        : HttpResponse.json(makeAction({ title: "Escreva um worker pool" })),
    ),
  );
}

function renderEvidencia() {
  return renderRoute(<EvidenciaRoute />, {
    route: `/metas/${GOAL_ID}/evidencia`,
    path: "/metas/:goalId/evidencia",
    extraRoutes: [{ path: `/metas/${GOAL_ID}`, element: <div>tela de detalhe da meta</div> }],
  });
}

describe("EvidenciaRoute", () => {
  it("mostra o título da meta no cabeçalho", async () => {
    setupGoalAndAction();
    renderEvidencia();
    expect(await screen.findByText("Nova evidência · Go backend")).toBeInTheDocument();
  });

  it("kind padrão é Código, com textarea rotulada 'Código'", async () => {
    setupGoalAndAction();
    renderEvidencia();
    await screen.findByText("Nova evidência · Go backend");
    expect(screen.getByLabelText("Código")).toBeInTheDocument();
  });

  it("trocar pra Texto muda o rótulo do campo", async () => {
    setupGoalAndAction();
    const user = userEvent.setup();
    renderEvidencia();
    await screen.findByText("Nova evidência · Go backend");

    await user.click(screen.getByRole("button", { name: "Texto" }));

    expect(screen.getByLabelText("Texto")).toBeInTheDocument();
    expect(screen.queryByLabelText("Código")).not.toBeInTheDocument();
  });

  it("trocar pra Link mostra campo de URL em vez do textarea", async () => {
    setupGoalAndAction();
    const user = userEvent.setup();
    renderEvidencia();
    await screen.findByText("Nova evidência · Go backend");

    await user.click(screen.getByRole("button", { name: "Link" }));

    expect(screen.getByLabelText("Link")).toBeInTheDocument();
    expect(screen.queryByLabelText("Código")).not.toBeInTheDocument();
  });

  it("mostra o checkbox de vínculo com a ação quando existe ação pendente, marcado por padrão", async () => {
    setupGoalAndAction({ hasAction: true });
    renderEvidencia();
    const checkbox = await screen.findByRole("checkbox", { name: /Escreva um worker pool/ });
    expect(checkbox).toBeChecked();
  });

  it("não mostra o checkbox quando não há ação pendente", async () => {
    setupGoalAndAction({ hasAction: false });
    renderEvidencia();
    await screen.findByText("Nova evidência · Go backend");
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  });

  it("registra evidência de código com actionId quando o checkbox está marcado, e navega no sucesso", async () => {
    setupGoalAndAction({ hasAction: true });
    let requestBody: any;
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/evidence`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json({ evidence: makeEvidence(), job: null }, { status: 202 });
      }),
    );
    const user = userEvent.setup();
    renderEvidencia();

    await screen.findByText("Nova evidência · Go backend");
    await user.type(screen.getByLabelText("Código"), "func main, sem chaves nesse texto de teste");
    await user.click(screen.getByRole("button", { name: "Registrar" }));

    await waitFor(() => expect(screen.getByText("tela de detalhe da meta")).toBeInTheDocument());
    expect(requestBody).toMatchObject({ kind: "code_snippet", body: "func main, sem chaves nesse texto de teste" });
    expect(requestBody.actionId).toEqual(expect.any(String));
  });

  it("desmarcar o checkbox omite actionId do envio", async () => {
    setupGoalAndAction({ hasAction: true });
    let requestBody: any;
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/evidence`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json({ evidence: makeEvidence(), job: null }, { status: 202 });
      }),
    );
    const user = userEvent.setup();
    renderEvidencia();

    await screen.findByText("Nova evidência · Go backend");
    await user.type(screen.getByLabelText("Código"), "func main, sem chaves nesse texto de teste");
    await user.click(screen.getByRole("checkbox"));
    await user.click(screen.getByRole("button", { name: "Registrar" }));

    await waitFor(() => expect(requestBody).toBeDefined());
    expect(requestBody.actionId).toBeUndefined();
  });

  it("registra evidência de link com externalUrl", async () => {
    setupGoalAndAction({ hasAction: false });
    let requestBody: any;
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/evidence`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json({ evidence: makeEvidence(), job: null }, { status: 202 });
      }),
    );
    const user = userEvent.setup();
    renderEvidencia();

    await screen.findByText("Nova evidência · Go backend");
    await user.click(screen.getByRole("button", { name: "Link" }));
    await user.type(screen.getByLabelText("Link"), "https://github.com/foo/bar");
    await user.click(screen.getByRole("button", { name: "Registrar" }));

    await waitFor(() => expect(requestBody).toBeDefined());
    expect(requestBody).toMatchObject({ kind: "repo_link", externalUrl: "https://github.com/foo/bar" });
    expect(requestBody.body).toBeUndefined();
  });

  it("mostra erro e não navega quando a API rejeita a evidência vazia", async () => {
    setupGoalAndAction();
    server.use(
      http.post(`${API_BASE}/goals/${GOAL_ID}/evidence`, () =>
        HttpResponse.json(
          { title: "essa evidência está vazia — cole um código, um texto ou um link antes de registrar", status: 400 },
          { status: 400 },
        ),
      ),
    );
    const user = userEvent.setup();
    renderEvidencia();

    await screen.findByText("Nova evidência · Go backend");
    await user.click(screen.getByRole("button", { name: "Registrar" }));

    expect(await screen.findByText(/essa evidência está vazia/)).toBeInTheDocument();
    expect(screen.queryByText("tela de detalhe da meta")).not.toBeInTheDocument();
  });
});
