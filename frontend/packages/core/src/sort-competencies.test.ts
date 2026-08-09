import { describe, expect, it } from "vitest";
import { sortCompetenciesForPanel } from "./sort-competencies";
import type { CompetencyForPanel } from "./types";

interface Item extends CompetencyForPanel {
  label: string;
}

describe("sortCompetenciesForPanel", () => {
  it("ordena por maior subida primeiro", () => {
    const items: Item[] = [
      { label: "Testes", level: 2, baselineLevel: 1 },
      { label: "Concorrência", level: 4, baselineLevel: 2 },
      { label: "Interfaces", level: 3, baselineLevel: 3 },
    ];
    expect(sortCompetenciesForPanel(items).map((i) => i.label)).toEqual([
      "Concorrência",
      "Testes",
      "Interfaces",
    ]);
  });

  it("manda não medidas para o fim, sem depender do delta", () => {
    const items: Item[] = [
      { label: "Performance", level: null, baselineLevel: null },
      { label: "Concorrência", level: 4, baselineLevel: 2 },
    ];
    expect(sortCompetenciesForPanel(items).map((i) => i.label)).toEqual([
      "Concorrência",
      "Performance",
    ]);
  });

  it("usa o campo delta pré-calculado da API quando presente", () => {
    const items: Item[] = [
      { label: "A", level: 3, baselineLevel: 3, delta: 1 },
      { label: "B", level: 4, baselineLevel: 2, delta: 2 },
    ];
    expect(sortCompetenciesForPanel(items).map((i) => i.label)).toEqual(["B", "A"]);
  });
});
