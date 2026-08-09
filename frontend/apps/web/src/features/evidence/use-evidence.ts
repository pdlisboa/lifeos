import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createEvidence, fetchEvidence, fetchEvidenceList } from "./api";
import type { EvidenceKind } from "./api";

/**
 * Sem otimismo (06-frontend.md §6.5): evidência é imutável (RN-06), e mostrar
 * sucesso antes da confirmação do servidor arrisca você achar que salvou algo
 * que se perdeu.
 */
export function useCreateEvidence(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      kind: EvidenceKind;
      title?: string;
      body?: string;
      externalUrl?: string;
      actionId?: string;
      competencyIds?: string[];
    }) => createEvidence(goalId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["evidence", goalId] });
      qc.invalidateQueries({ queryKey: ["delta", goalId] });
      qc.invalidateQueries({ queryKey: ["goal", goalId] });
      qc.invalidateQueries({ queryKey: ["today"] });
    },
  });
}

export function useEvidence(evidenceId: string | null) {
  return useQuery({
    queryKey: ["evidenceDetail", evidenceId],
    queryFn: () => fetchEvidence(evidenceId as string),
    enabled: evidenceId !== null,
  });
}

export function useEvidenceList(goalId: string, opts: { competencyId?: string; order?: "asc" | "desc" } = {}) {
  return useQuery({
    queryKey: ["evidence", goalId, opts.competencyId ?? null, opts.order ?? "asc"],
    queryFn: () => fetchEvidenceList(goalId, opts),
  });
}
