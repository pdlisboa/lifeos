package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/httpx"
)

// writeAppError mapeia os erros que os casos de uso devolvem para o formato
// RFC 9457 do contrato (03-api.md §5): `rule` carrega a regra de negócio
// violada, e o 409 de RN-02 traz `activeGoals` para a UI perguntar "qual
// pausar" em vez de só reclamar.
func writeAppError(w http.ResponseWriter, err error) {
	var maxActive *app.MaxActiveGoalsError
	if errors.As(err, &maxActive) {
		writeMaxActiveGoals(w, maxActive)
		return
	}

	if errors.Is(err, postgres.ErrNotFound) {
		httpx.WriteProblem(w, http.StatusNotFound, "Não encontrado", "", "")
		return
	}

	var ruleErr *domain.RuleError
	if errors.As(err, &ruleErr) {
		status := http.StatusBadRequest
		if ruleErr.Rule != "" {
			status = http.StatusConflict
		}
		httpx.WriteProblem(w, status, ruleErr.Msg, "", ruleErr.Rule)
		return
	}

	httpx.WriteProblem(w, http.StatusInternalServerError, "Erro interno", err.Error(), "")
}

func writeMaxActiveGoals(w http.ResponseWriter, e *app.MaxActiveGoalsError) {
	body := struct {
		Type        string           `json:"type,omitempty"`
		Title       string           `json:"title"`
		Status      int              `json:"status"`
		Detail      string           `json:"detail,omitempty"`
		Rule        string           `json:"rule,omitempty"`
		ActiveGoals []goalSummaryDTO `json:"activeGoals"`
	}{
		Title:       "Você já tem 3 metas ativas",
		Status:      http.StatusConflict,
		Detail:      "Dispersão é uma das causas de abandono. Qual você quer pausar?",
		Rule:        "RN-02",
		ActiveGoals: toGoalSummaryDTOs(e.ActiveGoals),
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(body)
}
