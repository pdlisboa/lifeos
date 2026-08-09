import { describe, expect, it } from "vitest";
import { formatConsistency } from "./consistency";

describe("formatConsistency", () => {
  it("formata dias ativos na janela", () => {
    expect(formatConsistency(18, 30)).toBe("18 dos últimos 30 dias");
  });

  it("funciona com zero dias ativos, sem tratamento especial de streak", () => {
    expect(formatConsistency(0, 30)).toBe("0 dos últimos 30 dias");
  });
});
