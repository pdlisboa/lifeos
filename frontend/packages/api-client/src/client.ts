import createClient from "openapi-fetch";
import type { paths } from "./schema";
import { ApiError, type ApiProblem } from "./problem";

export interface CreateApiClientOptions {
  baseUrl: string;
  /** Mobile manda Bearer token em vez de cookie (03-api.md §8). */
  getAuthHeader?: () => string | null;
}

/**
 * Cliente tipado sobre o contrato gerado (openapi.yaml). Regras de 06-frontend.md §3:
 * cookie de sessão incluído, `application/problem+json` vira ApiProblem tipado,
 * e qualquer status >= 400 lança — quem chama não precisa checar `error` manualmente,
 * o TanStack Query trata o throw.
 */
export function createApiClient({ baseUrl, getAuthHeader }: CreateApiClientOptions) {
  // openapi-fetch usa `fetch = globalThis.fetch` como valor padrão de parâmetro,
  // capturado uma vez na criação do client — se algo troca `globalThis.fetch`
  // depois (ex.: MSW em teste, um service worker), o client fica preso na
  // referência antiga. Passar essa função em vez do fetch direto garante busca
  // em `globalThis.fetch` a cada chamada.
  const client = createClient<paths>({
    baseUrl,
    credentials: "include",
    fetch: (...args: Parameters<typeof fetch>) => globalThis.fetch(...args),
  });

  client.use({
    onRequest({ request }) {
      const authHeader = getAuthHeader?.();
      if (authHeader) request.headers.set("Authorization", authHeader);
      return request;
    },
    async onResponse({ response }) {
      if (response.ok) return response;

      let problem: ApiProblem;
      try {
        problem = await response.clone().json();
      } catch {
        problem = { type: "about:blank", title: response.statusText, status: response.status };
      }
      throw new ApiError(problem);
    },
  });

  return client;
}

export type ApiClient = ReturnType<typeof createApiClient>;
export { ApiError };
export type { ApiProblem };
export type { paths } from "./schema";
export type { components } from "./schema";
