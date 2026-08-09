import { describe, expect, it } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { API_BASE } from "@/test/api-base";
import { createTestQueryClient } from "@/test/render";
import { RootLayout } from "./root-layout";

function renderLayout(initialPath = "/") {
  const queryClient = createTestQueryClient();
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route element={<RootLayout />}>
            <Route path="/" element={<div>conteúdo de hoje</div>} />
            <Route path="/metas" element={<div>conteúdo de metas</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("RootLayout", () => {
  it("mostra os links de navegação e o conteúdo da rota filha via Outlet", () => {
    renderLayout("/");
    expect(screen.getByRole("link", { name: "Hoje" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Metas" })).toBeInTheDocument();
    expect(screen.getByText("conteúdo de hoje")).toBeInTheDocument();
  });

  it("marca 'Hoje' como ativo em / e 'Metas' como ativo em /metas", () => {
    renderLayout("/metas");
    expect(screen.getByText("conteúdo de metas")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Metas" })).toHaveClass("bg-bg-overlay");
    expect(screen.getByRole("link", { name: "Hoje" })).not.toHaveClass("bg-bg-overlay");
  });

  it("'Sair' chama o logout", async () => {
    let logoutCalled = false;
    server.use(
      http.post(`${API_BASE}/auth/logout`, () => {
        logoutCalled = true;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    const user = userEvent.setup();
    renderLayout("/");

    await user.click(screen.getByRole("button", { name: "Sair" }));

    await waitFor(() => expect(logoutCalled).toBe(true));
  });
});
