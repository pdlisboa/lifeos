package app

import (
	"context"
	"errors"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

// TestCreateEvidenceIsTheCurrency é P1: registrar evidência é a moeda real do
// sistema, e toca a atividade da meta (consistência).
func TestCreateEvidenceIsTheCurrency(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	body := "func main() { fmt.Println(\"oi\") }"
	ev, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID,
		GoalID: active.Goal.ID,
		Kind:   domain.EvidenceCodeSnippet,
		Body:   &body,
	})
	if err != nil {
		t.Fatalf("CreateEvidence falhou: %v", err)
	}
	if ev.ID == "" {
		t.Fatal("evidência criada sem ID")
	}

	fetched, err := svc.GetEvidence(context.Background(), userID, ev.ID)
	if err != nil {
		t.Fatalf("GetEvidence falhou: %v", err)
	}
	if fetched.Body == nil || *fetched.Body != body {
		t.Fatalf("body persistido = %v, want %q", fetched.Body, body)
	}

	updated, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}
	if updated.Goal.LastActivityAt == nil {
		t.Fatal("registrar evidência deveria atualizar LastActivityAt da meta")
	}
}

func TestCreateEvidenceEmptyBodyFails(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	_, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID,
		GoalID: active.Goal.ID,
		Kind:   domain.EvidenceCodeSnippet,
	})
	if err == nil {
		t.Fatal("evidência sem body/blobKey/externalUrl deveria falhar")
	}
}

func TestCreateEvidenceGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	body := "conteúdo qualquer"
	_, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID,
		GoalID: "00000000-0000-0000-0000-000000000000",
		Kind:   domain.EvidenceCodeSnippet,
		Body:   &body,
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

// TestListEvidenceOrdering cobre a listagem crescente (padrão) e
// decrescente do museu (§7.4), com o limite passando direto.
func TestListEvidenceOrdering(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	titles := []string{"primeira", "segunda", "terceira"}
	for _, title := range titles {
		body := "conteúdo " + title
		if _, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
			UserID: userID, GoalID: active.Goal.ID, Kind: domain.EvidenceCodeSnippet, Title: &title, Body: &body,
		}); err != nil {
			t.Fatalf("CreateEvidence(%s) falhou: %v", title, err)
		}
	}

	asc, err := svc.ListEvidence(context.Background(), userID, active.Goal.ID, nil, true, 10)
	if err != nil {
		t.Fatalf("ListEvidence asc falhou: %v", err)
	}
	if len(asc) != 3 || *asc[0].Evidence.Title != "primeira" || *asc[2].Evidence.Title != "terceira" {
		t.Fatalf("ordem ascendente inesperada: %+v", asc)
	}

	desc, err := svc.ListEvidence(context.Background(), userID, active.Goal.ID, nil, false, 10)
	if err != nil {
		t.Fatalf("ListEvidence desc falhou: %v", err)
	}
	if len(desc) != 3 || *desc[0].Evidence.Title != "terceira" {
		t.Fatalf("ordem descendente inesperada: %+v", desc)
	}
}

func TestListEvidenceGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.ListEvidence(context.Background(), userID, "00000000-0000-0000-0000-000000000000", nil, true, 10)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

// TestCreateEvidenceTagsCompetencies cobre §7 da UX ("Competências
// tocadas") e a reconstrução point-in-time de levelsAtTime (§14.2 do modelo
// de dados) — a parte que a Fatia 1 deixava sempre vazia.
func TestCreateEvidenceTagsCompetencies(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	detail, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}
	comps := detail.Competencies
	if len(comps) < 2 {
		t.Fatalf("fixture esperava >= 2 competências, tem %d", len(comps))
	}
	touched := comps[0].ID

	_, err = svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID: userID, CompetencyID: touched, Level: 2, Rationale: "nível antes da evidência",
	})
	if err != nil {
		t.Fatalf("SetCompetencyLevel falhou: %v", err)
	}

	body := "func fetchAll(ctx context.Context) {}"
	ev, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID, GoalID: active.Goal.ID, Kind: domain.EvidenceCodeSnippet, Body: &body,
		CompetencyIDs: []string{touched},
	})
	if err != nil {
		t.Fatalf("CreateEvidence falhou: %v", err)
	}

	cards, err := svc.ListEvidence(context.Background(), userID, active.Goal.ID, nil, true, 10)
	if err != nil {
		t.Fatalf("ListEvidence falhou: %v", err)
	}
	var card *EvidenceCard
	for i := range cards {
		if cards[i].Evidence.ID == ev.ID {
			card = &cards[i]
		}
	}
	if card == nil {
		t.Fatal("evidência criada não apareceu na listagem")
	}
	if len(card.CompetencyIDs) != 1 || card.CompetencyIDs[0] != touched {
		t.Fatalf("CompetencyIDs = %v, want [%s]", card.CompetencyIDs, touched)
	}
	if lvl, ok := card.LevelsAtTime[touched]; !ok || lvl != 2 {
		t.Fatalf("LevelsAtTime[%s] = %v (ok=%v), want 2", touched, lvl, ok)
	}
	for _, c := range comps[1:] {
		if _, ok := card.LevelsAtTime[c.ID]; ok {
			t.Fatalf("competência %s nunca medida não deveria aparecer em LevelsAtTime (null nunca vira 0)", c.ID)
		}
	}

	filtered, err := svc.ListEvidence(context.Background(), userID, active.Goal.ID, &touched, true, 10)
	if err != nil {
		t.Fatalf("ListEvidence filtrado falhou: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Evidence.ID != ev.ID {
		t.Fatalf("filtro por competência = %+v, want só a evidência marcada", filtered)
	}

	other := comps[1].ID
	empty, err := svc.ListEvidence(context.Background(), userID, active.Goal.ID, &other, true, 10)
	if err != nil {
		t.Fatalf("ListEvidence filtrado (outra competência) falhou: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("filtro por competência não tocada deveria vir vazio, got %+v", empty)
	}
}

func TestCreateEvidenceRejectsCompetencyFromOtherGoal(t *testing.T) {
	svc, userID := newFixture(t)
	a := readyGoal(t, svc, userID, "Meta A")
	b := readyGoal(t, svc, userID, "Meta B")
	bDetail, err := svc.GetGoal(context.Background(), userID, b.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}

	body := "conteúdo"
	_, err = svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID, GoalID: a.Goal.ID, Kind: domain.EvidenceCodeSnippet, Body: &body,
		CompetencyIDs: []string{bDetail.Competencies[0].ID},
	})
	if err == nil {
		t.Fatal("competência de outra meta deveria ser rejeitada")
	}
}
