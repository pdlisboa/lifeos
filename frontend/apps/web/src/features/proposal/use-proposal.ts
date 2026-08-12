import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  acceptProposal,
  fetchProposals,
  rejectProposal,
  requestTrackRevision,
  type ProposalMilestone,
} from "./api";

export function trackProposalsQueryKey(goalId: string) {
  return ["proposals", "track", goalId] as const;
}

/** Só as propostas de trilha pendentes de uma meta — é tudo que a tela usa hoje. */
export function useTrackProposals(goalId: string) {
  return useQuery({
    queryKey: trackProposalsQueryKey(goalId),
    queryFn: () => fetchProposals({ status: "pending", goalId }),
    select: (proposals) => proposals.filter((p) => p.kind === "track"),
  });
}

/**
 * Aceitar/rejeitar muda a trilha e a lista de propostas — invalidar os dois
 * é o que evita a aba de proposta continuar aparecendo depois de resolvida
 * (mesmo racional de useCompleteAction em features/action).
 */
export function useAcceptProposal(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ proposalId, milestones }: { proposalId: string; milestones?: ProposalMilestone[] }) =>
      acceptProposal(proposalId, milestones ? { milestones } : undefined),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: trackProposalsQueryKey(goalId) });
      qc.invalidateQueries({ queryKey: ["track", goalId] });
    },
  });
}

export function useRejectProposal(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ proposalId, reason }: { proposalId: string; reason?: string }) => rejectProposal(proposalId, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: trackProposalsQueryKey(goalId) });
    },
  });
}

export function useRequestTrackRevision(goalId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => requestTrackRevision(goalId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: trackProposalsQueryKey(goalId) });
    },
  });
}
