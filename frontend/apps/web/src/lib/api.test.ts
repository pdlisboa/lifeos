import { describe, expect, it } from "vitest";
import { unwrap } from "./api";

describe("unwrap", () => {
  it("devolve data quando presente", async () => {
    const result = await unwrap(Promise.resolve({ data: { ok: true } }));
    expect(result).toEqual({ ok: true });
  });

  it("lança quando data é undefined (toda resposta de erro já vira ApiError antes disso)", async () => {
    await expect(unwrap(Promise.resolve({ data: undefined }))).rejects.toThrow("resposta vazia inesperada");
  });
});
