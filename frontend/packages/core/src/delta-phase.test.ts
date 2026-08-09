import { describe, expect, it } from "vitest";
import { deltaPhase } from "./delta-phase";

describe("deltaPhase", () => {
  it("é baseline sem nenhuma evidência, mesmo depois de dias", () => {
    expect(deltaPhase(0, 0)).toBe("baseline");
    expect(deltaPhase(0, 10)).toBe("baseline");
  });

  it("é accumulating com evidência mas menos de 3 semanas ativa", () => {
    expect(deltaPhase(3, 1)).toBe("accumulating");
    expect(deltaPhase(3, 20)).toBe("accumulating");
  });

  it("é delta a partir de 21 dias ativos com evidência", () => {
    expect(deltaPhase(5, 21)).toBe("delta");
    expect(deltaPhase(12, 74)).toBe("delta");
  });
});
