package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/agents"
	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/platform/httpx"
)

func (h *Handler) ListProposals(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	var goalID *string
	if g := r.URL.Query().Get("goalId"); g != "" {
		goalID = &g
	}

	proposals, err := h.Service.ListProposals(r.Context(), uid, status, goalID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]proposalDTO, len(proposals))
	for i, p := range proposals {
		items[i] = toProposalDTO(p)
	}
	httpx.WriteJSON(w, http.StatusOK, items)
}

type acceptProposalRequest struct {
	Edits map[string]json.RawMessage `json:"edits"`
}

// AcceptProposal só sabe aplicar `kind: track` nesta rodada — os outros
// tipos ainda não têm agente nenhum produzindo-os. `edits.milestones`,
// quando presente, substitui o payload original (P8: o agente propõe, você
// decide).
func (h *Handler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	proposalID := chi.URLParam(r, "proposalId")

	var req acceptProposalRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // corpo é opcional
	}

	var milestones []agents.PlannerMilestoneOutput
	if raw, ok := req.Edits["milestones"]; ok {
		if err := json.Unmarshal(raw, &milestones); err != nil {
			httpx.WriteProblem(w, http.StatusBadRequest, "edits.milestones inválido", err.Error(), "")
			return
		}
	}

	p, err := h.Service.AcceptProposal(r.Context(), app.AcceptProposalInput{
		UserID:     uid,
		ProposalID: proposalID,
		Milestones: milestones,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProposalDTO(*p))
}

type rejectProposalRequest struct {
	Reason *string `json:"reason"`
}

func (h *Handler) RejectProposal(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	proposalID := chi.URLParam(r, "proposalId")

	var req rejectProposalRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // corpo é opcional
	}

	p, err := h.Service.RejectProposal(r.Context(), uid, proposalID, req.Reason)
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProposalDTO(*p))
}
