import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { renderRoute } from "@/test/render";
import { makeUser } from "@/test/fixtures";
import { LoginRoute } from "./index";

function renderLogin() {
  return renderRoute(<LoginRoute />, {
    route: "/login",
    path: "/login",
    extraRoutes: [{ path: "/", element: <div>página protegida</div> }],
  });
}

describe("LoginRoute", () => {
  it("valida email e senha antes de submeter", async () => {
    const user = userEvent.setup();
    renderLogin();

    await user.click(await screen.findByRole("button", { name: "Entrar" }));

    expect(await screen.findByText("email inválido")).toBeInTheDocument();
    expect(screen.getByText("obrigatório")).toBeInTheDocument();
  });

  it("faz login com sucesso e navega pra rota de origem", async () => {
    server.use(
      http.post(`${API_BASE}/auth/login`, () =>
        HttpResponse.json({ user: makeUser(), token: null, expiresAt: "2026-09-01T00:00:00Z" }),
      ),
    );
    const user = userEvent.setup();
    renderLogin();

    await user.type(screen.getByLabelText("Email"), "dev@lifeos.local");
    await user.type(screen.getByLabelText("Senha"), "fatia1-dev-test");
    await user.click(screen.getByRole("button", { name: "Entrar" }));

    expect(await screen.findByText("página protegida")).toBeInTheDocument();
  });

  it("mostra mensagem amigável em credenciais inválidas (401)", async () => {
    server.use(
      http.post(`${API_BASE}/auth/login`, () =>
        HttpResponse.json({ title: "Credenciais inválidas", status: 401 }, { status: 401 }),
      ),
    );
    const user = userEvent.setup();
    renderLogin();

    await user.type(screen.getByLabelText("Email"), "dev@lifeos.local");
    await user.type(screen.getByLabelText("Senha"), "senha-errada");
    await user.click(screen.getByRole("button", { name: "Entrar" }));

    expect(await screen.findByText("Email ou senha incorretos.")).toBeInTheDocument();
  });

  it("mostra mensagem genérica em erro não-401", async () => {
    server.use(
      http.post(`${API_BASE}/auth/login`, () =>
        HttpResponse.json({ title: "Erro interno", status: 500 }, { status: 500 }),
      ),
    );
    const user = userEvent.setup();
    renderLogin();

    await user.type(screen.getByLabelText("Email"), "dev@lifeos.local");
    await user.type(screen.getByLabelText("Senha"), "fatia1-dev-test");
    await user.click(screen.getByRole("button", { name: "Entrar" }));

    expect(await screen.findByText("Não deu pra entrar agora. Tenta de novo.")).toBeInTheDocument();
  });

  it("já logado: pula direto pra rota de origem sem mostrar o formulário", async () => {
    server.use(http.get(`${API_BASE}/me`, () => HttpResponse.json(makeUser())));
    renderLogin();

    expect(await screen.findByText("página protegida")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Entrar" })).not.toBeInTheDocument();
  });
});
