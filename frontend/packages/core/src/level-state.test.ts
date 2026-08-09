import { describe, expect, it } from "vitest";
import { levelState } from "./level-state";

describe("levelState", () => {
  it("é unknown quando o nível é null, mesmo com confidence preenchida", () => {
    expect(levelState(null, "high")).toBe("unknown");
  });

  it("é unknown quando confidence é unknown", () => {
    expect(levelState(3, "unknown")).toBe("unknown");
  });

  it("é stale quando confidence é low (auto-declarado ou vencido por desuso)", () => {
    expect(levelState(2, "low")).toBe("stale");
  });

  it("é measured para medium e high", () => {
    expect(levelState(3, "medium")).toBe("measured");
    expect(levelState(4, "high")).toBe("measured");
  });

  it("nunca trata null como nível 0", () => {
    expect(levelState(0, "high")).toBe("measured");
    expect(levelState(null, "high")).not.toBe("measured");
  });
});
