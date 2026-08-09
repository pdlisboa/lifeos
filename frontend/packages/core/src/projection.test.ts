import { describe, expect, it } from "vitest";
import { formatProjectionFooter } from "./projection";

describe("formatProjectionFooter", () => {
  it("usa o reason do servidor quando indisponível, sem reformatar", () => {
    const text = formatProjectionFooter({ available: false, reason: "ainda coletando ritmo (2 de 3 semanas)", minutesPerWeek: null });
    expect(text).toBe("ainda coletando ritmo (2 de 3 semanas)");
  });

  it("cai num texto honesto se vier indisponível sem reason", () => {
    const text = formatProjectionFooter({ available: false, reason: null, minutesPerWeek: null });
    expect(text).toBe("ainda coletando ritmo");
  });

  it("mostra o ritmo real em horas/semana quando disponível, sem prever chegada", () => {
    const text = formatProjectionFooter({ available: true, reason: null, minutesPerWeek: 192 });
    expect(text).toContain("~3,2h/semana");
    expect(text).not.toMatch(/semanas?$/);
    expect(text.toLowerCase()).not.toContain("marco sai em");
  });

  it("nunca inventa: available true sem minutesPerWeek cai no fallback honesto", () => {
    const text = formatProjectionFooter({ available: true, reason: null, minutesPerWeek: null });
    expect(text).toBe("ainda coletando ritmo");
  });
});
