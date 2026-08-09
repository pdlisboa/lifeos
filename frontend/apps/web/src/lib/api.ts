import { createApiClient } from "@lifeos/api-client";

// Web autentica por cookie HttpOnly (credentials: include no client), nunca por
// Bearer — getAuthHeader fica de fora de propósito (06-frontend.md §3, 03-api.md §8).
export const api = createApiClient({
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1",
});

/** Toda resposta de erro já foi lançada como ApiError pelo client — aqui só sobra o caminho feliz. */
export async function unwrap<T>(promise: Promise<{ data?: T }>): Promise<T> {
  const { data } = await promise;
  if (data === undefined) throw new Error("resposta vazia inesperada");
  return data;
}
