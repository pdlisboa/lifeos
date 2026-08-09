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

	asc, err := svc.ListEvidence(context.Background(), userID, active.Goal.ID, true, 10)
	if err != nil {
		t.Fatalf("ListEvidence asc falhou: %v", err)
	}
	if len(asc) != 3 || *asc[0].Title != "primeira" || *asc[2].Title != "terceira" {
		t.Fatalf("ordem ascendente inesperada: %+v", asc)
	}

	desc, err := svc.ListEvidence(context.Background(), userID, active.Goal.ID, false, 10)
	if err != nil {
		t.Fatalf("ListEvidence desc falhou: %v", err)
	}
	if len(desc) != 3 || *desc[0].Title != "terceira" {
		t.Fatalf("ordem descendente inesperada: %+v", desc)
	}
}

func TestListEvidenceGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.ListEvidence(context.Background(), userID, "00000000-0000-0000-0000-000000000000", true, 10)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}
