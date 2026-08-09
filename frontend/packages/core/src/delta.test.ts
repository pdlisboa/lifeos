import { describe, expect, it } from "vitest";
import { formatDelta, formatLevelRange } from "./delta";

describe("formatDelta", () => {
  it("formata subida com janela de dias", () => {
    expect(formatDelta(2, 4, 40)).toBe("↑ 2 em 40 dias");
  });

  it("formata queda", () => {
    expect(formatDelta(4, 2, 10)).toBe("↓ 2 em 10 dias");
  });

  it("omite a janela quando days é null", () => {
    expect(formatDelta(2, 4, null)).toBe("↑ 2");
  });

  it("retorna null quando não há delta (baseline igual ao atual)", () => {
    expect(formatDelta(3, 3, 51)).toBeNull();
  });

  it("retorna null quando baseline é desconhecido", () => {
    expect(formatDelta(null, 4, 10)).toBeNull();
  });

  it("retorna null quando o nível atual é desconhecido", () => {
    expect(formatDelta(2, null, 10)).toBeNull();
  });
});

describe("formatLevelRange", () => {
  it("formata de → para", () => {
    expect(formatLevelRange(2, 4)).toBe("2 → 4");
  });

  it("usa ? para lados desconhecidos", () => {
    expect(formatLevelRange(null, 4)).toBe("? → 4");
    expect(formatLevelRange(2, null)).toBe("2 → ?");
  });
});
