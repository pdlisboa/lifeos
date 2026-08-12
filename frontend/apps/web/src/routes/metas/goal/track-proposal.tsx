import { useState } from "react";
import { Button, Input, Textarea } from "@/components/ui";
import { ProblemError } from "@/components/problem-error";
import { useAcceptProposal, useRejectProposal, useTrackProposals } from "@/features/proposal/use-proposal";
import type { Proposal, ProposalMilestone } from "@/features/proposal/api";

function parseMilestones(payload: Proposal["payload"]): ProposalMilestone[] {
  if (!payload || typeof payload !== "object") return [];
  const raw = (payload as { milestones?: unknown }).milestones;
  return Array.isArray(raw) ? (raw as ProposalMilestone[]) : [];
}

function ProposalCard({ proposal, goalId }: { proposal: Proposal; goalId: string }) {
  const original = parseMilestones(proposal.payload);
  const [milestones, setMilestones] = useState(original);
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState("");
  const accept = useAcceptProposal(goalId);
  const reject = useRejectProposal(goalId);

  const edited = JSON.stringify(milestones) !== JSON.stringify(original);

  const updateMilestone = (index: number, patch: Partial<ProposalMilestone>) => {
    setMilestones((prev) => prev.map((m, i) => (i === index ? { ...m, ...patch } : m)));
  };

  const handleAccept = async () => {
    try {
      await accept.mutateAsync({ proposalId: proposal.id, milestones: edited ? milestones : undefined });
    } catch {
      // erro fica em accept.error, renderizado abaixo
    }
  };

  const handleReject = async () => {
    try {
      await reject.mutateAsync({ proposalId: proposal.id, reason: reason.trim() || undefined });
    } catch {
      // erro fica em reject.error
    }
  };

  return (
    <div className="space-y-3 rounded-md border border-border-subtle bg-bg-raised p-4">
      <p className="text-sm text-fg-secondary">{proposal.rationale}</p>

      <ol className="space-y-2">
        {milestones.map((m, i) => (
          <li key={i} className="space-y-1.5 rounded-md border border-border-subtle bg-bg-overlay p-3">
            <div className="flex items-baseline gap-2">
              <span className="shrink-0 text-xs text-fg-muted">{i + 1}.</span>
              <Input
                value={m.title}
                onChange={(e) => updateMilestone(i, { title: e.target.value })}
                aria-label={`Título do marco ${i + 1}`}
              />
            </div>
            <Textarea
              rows={2}
              value={m.completionCriteria}
              onChange={(e) => updateMilestone(i, { completionCriteria: e.target.value })}
              aria-label={`Critério do marco ${i + 1}`}
            />
            {m.carriedOver && (
              <p className="text-xs text-fg-muted">marco já concluído — copiado como está, intocável</p>
            )}
          </li>
        ))}
      </ol>

      {accept.error && <ProblemError error={accept.error} />}
      {reject.error && <ProblemError error={reject.error} />}

      {rejecting ? (
        <div className="space-y-2">
          <Textarea
            rows={2}
            placeholder="por que você prefere não aceitar? (opcional)"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            aria-label="Motivo da rejeição"
          />
          <div className="flex gap-2">
            <Button variant="danger" disabled={reject.isPending} onClick={handleReject}>
              {reject.isPending ? "Rejeitando…" : "Confirmar rejeição"}
            </Button>
            <Button variant="ghost" onClick={() => setRejecting(false)}>
              cancelar
            </Button>
          </div>
        </div>
      ) : (
        <div className="flex gap-2">
          <Button disabled={accept.isPending} onClick={handleAccept}>
            {accept.isPending ? "Aplicando…" : edited ? "Aceitar com edições" : "Aceitar"}
          </Button>
          <Button variant="ghost" disabled={accept.isPending} onClick={() => setRejecting(true)}>
            rejeitar
          </Button>
        </div>
      )}
    </div>
  );
}

/**
 * Revisão de trilha do A1 (04-agentes.md §4.1): o agente propõe, você decide
 * (P8) — aceitar aplica os marcos como nova versão da trilha, editar troca
 * título/critério antes de aplicar, rejeitar não muda nada (RN-07).
 */
export function TrackProposalView({ goalId }: { goalId: string }) {
  const { data: proposals, isPending, error } = useTrackProposals(goalId);

  if (isPending) return <p className="text-sm text-fg-muted">Carregando…</p>;
  if (error) return <ProblemError error={error} />;

  if (proposals.length === 0) {
    return <p className="text-sm text-fg-secondary">Nenhuma proposta de trilha pendente.</p>;
  }

  return (
    <div className="space-y-4">
      {proposals.map((p) => (
        <ProposalCard key={p.id} proposal={p} goalId={goalId} />
      ))}
    </div>
  );
}
