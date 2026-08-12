package http

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes monta as rotas da Fatia 1 do módulo Metas
// (backend/api/openapi.yaml): goals, probe, competencies, track, action,
// sessions, evidence, delta, consistency. Assume que `r` já está atrás do
// middleware de autenticação — quem monta isso é cmd/api (via module.go).
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/today", h.GetToday)

	r.Route("/goals", func(r chi.Router) {
		r.Get("/", h.ListGoals)
		r.Post("/", h.CreateGoal)

		r.Route("/{goalId}", func(r chi.Router) {
			r.Get("/", h.GetGoal)
			r.Patch("/", h.PatchGoal)
			r.Post("/activate", h.ActivateGoal)

			r.Get("/probe", h.GetProbe)
			r.Post("/probe/answer", h.AnswerProbe)
			r.Post("/probe/skip", h.SkipProbe)

			r.Get("/delta", h.GetDelta)
			r.Get("/projection", h.GetProjection)
			r.Get("/consistency", h.GetConsistency)
			r.Get("/track", h.GetTrack)
			r.Post("/track", h.RequestTrackRevision)

			r.Get("/action", h.GetPendingAction)
			r.Post("/action", h.CreateUserAction)

			r.Get("/sessions", h.ListSessions)
			r.Post("/sessions", h.CreateSession)

			r.Get("/evidence", h.ListEvidence)
			r.Post("/evidence", h.CreateEvidence)

			r.Put("/competencies/{competencyId}/level", h.SetCompetencyLevel)
			r.Get("/competencies/{competencyId}/history", h.CompetencyHistory)
		})
	})

	r.Route("/actions/{actionId}", func(r chi.Router) {
		r.Post("/complete", h.CompleteAction)
		r.Post("/skip", h.SkipAction)
	})

	r.Get("/evidence/{evidenceId}", h.GetEvidence)

	r.Route("/proposals", func(r chi.Router) {
		r.Get("/", h.ListProposals)
		r.Post("/{proposalId}/accept", h.AcceptProposal)
		r.Post("/{proposalId}/reject", h.RejectProposal)
	})
}
