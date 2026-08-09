package app

import (
	"context"
	"errors"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
)

func TestGetTrackBeforeActivationNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta ainda em rascunho")

	_, err := svc.GetTrack(context.Background(), userID, created.Goal.ID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound antes da ativação (sem trilha ainda), got %v", err)
	}
}

func TestGetTrackAfterActivation(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	track, err := svc.GetTrack(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetTrack falhou: %v", err)
	}
	if track.Version != 1 {
		t.Fatalf("version = %d, want 1", track.Version)
	}
	if len(track.Milestones) == 0 {
		t.Fatal("esperava marcos vindos da milestone_library do pack")
	}
}

func TestGetTrackGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.GetTrack(context.Background(), userID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}
