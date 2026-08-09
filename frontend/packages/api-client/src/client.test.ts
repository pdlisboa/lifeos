import { afterEach, describe, expect, it, vi } from "vitest";
import { createApiClient } from "./client";
import { ApiError } from "./problem";

function jsonResponse(body: unknown, status: number) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("createApiClient", () => {
  it("resolve normalmente quando a resposta é ok", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ status: "ok" }, 200));
    vi.stubGlobal("fetch", fetchMock);

    const api = createApiClient({ baseUrl: "http://api.test" });
    const { data, error } = await api.GET("/healthz");

    expect(error).toBeUndefined();
    expect(data).toEqual({ status: "ok" });
  });

  it("lança ApiError com o problem+json parseado quando o status é >= 400", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ type: "about:blank", title: "Não autenticado", status: 401, detail: "sem cookie" }, 401),
    );
    vi.stubGlobal("fetch", fetchMock);

    const api = createApiClient({ baseUrl: "http://api.test" });

    await expect(api.GET("/me")).rejects.toMatchObject({
      name: "ApiError",
      problem: { title: "Não autenticado", status: 401, detail: "sem cookie" },
    });
  });

  it("cai no fallback (about:blank + statusText) quando o corpo do erro não é JSON válido", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("<html>not json</html>", { status: 500, statusText: "Internal Server Error" }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const api = createApiClient({ baseUrl: "http://api.test" });

    try {
      await api.GET("/healthz");
      expect.unreachable("deveria ter lançado");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).problem).toEqual({
        type: "about:blank",
        title: "Internal Server Error",
        status: 500,
      });
    }
  });

  it("manda o header vindo de getAuthHeader (mobile)", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({}, 200));
    vi.stubGlobal("fetch", fetchMock);

    const api = createApiClient({ baseUrl: "http://api.test", getAuthHeader: () => "Bearer token-123" });
    await api.GET("/healthz");

    const request = fetchMock.mock.calls[0][0] as Request;
    expect(request.headers.get("Authorization")).toBe("Bearer token-123");
  });

  it("não manda Authorization quando getAuthHeader não é passado (web usa cookie)", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({}, 200));
    vi.stubGlobal("fetch", fetchMock);

    const api = createApiClient({ baseUrl: "http://api.test" });
    await api.GET("/healthz");

    const request = fetchMock.mock.calls[0][0] as Request;
    expect(request.headers.get("Authorization")).toBeNull();
  });

  it("não manda Authorization quando getAuthHeader devolve null", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({}, 200));
    vi.stubGlobal("fetch", fetchMock);

    const api = createApiClient({ baseUrl: "http://api.test", getAuthHeader: () => null });
    await api.GET("/healthz");

    const request = fetchMock.mock.calls[0][0] as Request;
    expect(request.headers.get("Authorization")).toBeNull();
  });

  it("busca globalThis.fetch a cada chamada, não só na criação do client (regressão: MSW/patch tardio)", async () => {
    const originalFetch = vi.fn().mockResolvedValue(jsonResponse({ from: "original" }, 200));
    vi.stubGlobal("fetch", originalFetch);

    const api = createApiClient({ baseUrl: "http://api.test" });

    // troca o fetch global DEPOIS do client já criado — como o MSW faz em
    // server.listen() depois que o módulo já foi importado.
    const patchedFetch = vi.fn().mockResolvedValue(jsonResponse({ from: "patched" }, 200));
    vi.stubGlobal("fetch", patchedFetch);

    const { data } = await api.GET("/healthz");

    expect(originalFetch).not.toHaveBeenCalled();
    expect(patchedFetch).toHaveBeenCalledTimes(1);
    expect(data).toEqual({ from: "patched" });
  });
});
