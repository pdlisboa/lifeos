import { describe, expect, it } from "vitest";
import { ApiError } from "@lifeos/api-client";
import { queryClient } from "./query";

function retry(failureCount: number, error: unknown) {
  const fn = queryClient.getDefaultOptions().queries?.retry;
  if (typeof fn !== "function") throw new Error("retry não é função");
  return fn(failureCount, error as Error);
}

describe("queryClient default retry", () => {
  it("não retenta erro de estado (status < 500)", () => {
    const error = new ApiError({ type: "about:blank", title: "Não encontrado", status: 404 });
    expect(retry(0, error)).toBe(false);
  });

  it("retenta erro de servidor (status >= 500) até 2 vezes", () => {
    const error = new ApiError({ type: "about:blank", title: "Erro interno", status: 500 });
    expect(retry(0, error)).toBe(true);
    expect(retry(1, error)).toBe(true);
    expect(retry(2, error)).toBe(false);
  });

  it("retenta erros que não são ApiError (ex.: falha de rede) até 2 vezes", () => {
    const error = new TypeError("network error");
    expect(retry(0, error)).toBe(true);
    expect(retry(2, error)).toBe(false);
  });

  it("staleTime padrão é 10s", () => {
    expect(queryClient.getDefaultOptions().queries?.staleTime).toBe(10_000);
  });
});
