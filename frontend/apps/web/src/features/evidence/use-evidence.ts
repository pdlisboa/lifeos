import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createEvidence, fetchEvidence, fetchEvidenceList, markEvalCase, unmarkEvalCase } from "./api";
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

// Captura de material de eval pro A3 (04-agentes.md §6.1) — invalida a
// listagem (museu) e o detalhe pra refletir a marcação sem precisar de F5.
export function useMarkEvalCase(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      evidenceId,
      note,
      scores,
    }: {
      evidenceId: string;
      note: string;
      scores: { competencyId: string; level: number }[];
    }) => markEvalCase(evidenceId, { note, scores }),
    onSuccess: (_data, { evidenceId }) => {
      qc.invalidateQueries({ queryKey: ["evidence", goalId] });
      qc.invalidateQueries({ queryKey: ["evidenceDetail", evidenceId] });
    },
  });
}

export function useUnmarkEvalCase(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (evidenceId: string) => unmarkEvalCase(evidenceId),
    onSuccess: (_data, evidenceId) => {
      qc.invalidateQueries({ queryKey: ["evidence", goalId] });
      qc.invalidateQueries({ queryKey: ["evidenceDetail", evidenceId] });
    },
  });
}
