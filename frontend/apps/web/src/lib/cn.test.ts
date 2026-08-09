import { describe, expect, it } from "vitest";
import { cn } from "./cn";

describe("cn", () => {
  it("junta classes simples", () => {
    expect(cn("a", "b")).toBe("a b");
  });

  it("ignora valores falsy", () => {
    expect(cn("a", false, undefined, null, "b")).toBe("a b");
  });

  it("resolve conflito do Tailwind mantendo a última classe", () => {
    expect(cn("px-2", "px-4")).toBe("px-4");
  });

  it("aplica classes condicionais via objeto", () => {
    expect(cn("base", { ativo: true, inativo: false })).toBe("base ativo");
  });
});
