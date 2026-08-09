import type { GoalArchetype } from "@/features/goal/api";

// Os únicos dois domain packs do produto (CLAUDE.md — "Packs de domínio: golang e
// english apenas"). archetype vem do pack (ambos "skill" hoje, backend/.../packs/*.yaml),
// nunca é escolha da pessoa (05-ux.md §4).
export const packs: { packId: string; label: string; archetype: GoalArchetype }[] = [
  { packId: "golang", label: "Go", archetype: "skill" },
  { packId: "english", label: "Inglês", archetype: "skill" },
];
