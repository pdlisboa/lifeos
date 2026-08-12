import { api, unwrap } from "@/lib/api";
import type { components } from "@lifeos/api-client";

export type Proposal = components["schemas"]["Proposal"];
export type ProposalStatus = "pending" | "accepted" | "rejected" | "expired";

/** O formato de `payload` quando `kind === "track"` — o que HandlePlanTrack grava (04-agentes.md §4.1). */
export interface ProposalMilestone {
  ordinal: number;
  title: string;
  completionCriteria: string;
  competencyKeys: string[];
  carriedOver: boolean;
  sourceLibraryTitle: string | null;
}

export function fetchProposals(params: { status?: ProposalStatus; goalId?: string } = {}) {
  return unwrap(api.GET("/proposals", { params: { query: params } }));
}

export function acceptProposal(proposalId: string, edits?: { milestones: ProposalMilestone[] }) {
  return unwrap(
    api.POST("/proposals/{proposalId}/accept", {
      params: { path: { proposalId } },
      body: edits ? { edits } : {},
    }),
  );
}

export function rejectProposal(proposalId: string, reason?: string) {
  return unwrap(
    api.POST("/proposals/{proposalId}/reject", {
      params: { path: { proposalId } },
      body: reason ? { reason } : {},
    }),
  );
}

export function requestTrackRevision(goalId: string) {
  return unwrap(api.POST("/goals/{goalId}/track", { params: { path: { goalId } } }));
}
