package http

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

// TestWriteAppErrorMapping cobre o mapeamento RFC 9457 (03-api.md §5) sem
// precisar de banco: RuleError com Rule vira 409, sem Rule vira 400,
// ErrNotFound vira 404, e qualquer outro erro vira 500.
func TestWriteAppErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantRule   string
	}{
		{"RuleError com regra vira 409", domain.NewRuleError("RN-01", "faltam competências"), 409, "RN-01"},
		{"RuleError sem regra vira 400", domain.NewRuleError("", "campo inválido"), 400, ""},
		{"ErrNotFound vira 404", postgres.ErrNotFound, 404, ""},
		{"ErrNotFound envolvido (%w) ainda vira 404", errIsWrapped(), 404, ""},
		{"erro genérico vira 500", errors.New("falha inesperada"), 500, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeAppError(rec, tc.err)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Fatalf("Content-Type = %q, want application/problem+json", ct)
			}
			var body struct {
				Rule string `json:"rule"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decodificar corpo: %v", err)
			}
			if body.Rule != tc.wantRule {
				t.Fatalf("rule = %q, want %q", body.Rule, tc.wantRule)
			}
		})
	}
}

func errIsWrapped() error {
	return &wrappedErr{postgres.ErrNotFound}
}

type wrappedErr struct{ inner error }

func (e *wrappedErr) Error() string { return "meta a pausar: " + e.inner.Error() }
func (e *wrappedErr) Unwrap() error { return e.inner }

// TestWriteAppErrorMaxActiveGoals cobre a saída especial de RN-02: 409 com
// o campo activeGoals preenchido, para a UI perguntar "qual pausar".
func TestWriteAppErrorMaxActiveGoals(t *testing.T) {
	g1, err := domain.NewGoal("g1", "u1", "Meta 1", domain.ArchetypeSkill, "golang", nil, time.Now())
	if err != nil {
		t.Fatalf("NewGoal: %v", err)
	}
	rec := httptest.NewRecorder()
	writeAppError(rec, &app.MaxActiveGoalsError{ActiveGoals: []domain.Goal{*g1}})

	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	var body struct {
		Rule        string           `json:"rule"`
		ActiveGoals []goalSummaryDTO `json:"activeGoals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decodificar corpo: %v", err)
	}
	if body.Rule != "RN-02" {
		t.Fatalf("rule = %q, want RN-02", body.Rule)
	}
	if len(body.ActiveGoals) != 1 || body.ActiveGoals[0].ID != "g1" {
		t.Fatalf("activeGoals = %+v", body.ActiveGoals)
	}
}
