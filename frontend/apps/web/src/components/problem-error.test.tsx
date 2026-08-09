import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { ApiError } from "@lifeos/api-client";
import { ProblemError } from "./problem-error";

describe("ProblemError", () => {
  it("mostra título e detalhe de um ApiError", () => {
    const error = new ApiError({
      type: "https://lifeos/errors/max-active-goals",
      title: "Você já tem 3 metas ativas",
      status: 409,
      detail: "Qual você quer pausar?",
    });
    render(<ProblemError error={error} />);
    expect(screen.getByText("Você já tem 3 metas ativas")).toBeInTheDocument();
    expect(screen.getByText("Qual você quer pausar?")).toBeInTheDocument();
  });

  it("omite o detalhe quando ele não vem preenchido", () => {
    const error = new ApiError({ type: "about:blank", title: "Não encontrado", status: 404 });
    render(<ProblemError error={error} />);
    expect(screen.getByText("Não encontrado")).toBeInTheDocument();
  });

  it("cai no fallback genérico para erros que não são ApiError", () => {
    render(<ProblemError error={new Error("boom")} />);
    expect(screen.getByText("Algo deu errado. Tenta de novo.")).toBeInTheDocument();
  });
});
