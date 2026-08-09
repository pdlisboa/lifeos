package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func TestInsertTrackWithMilestonesAndCompetencies(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	c1 := newCompetency(t, q, g, "idioms", "Idiomas")
	c2 := newCompetency(t, q, g, "testing", "Testes")

	track := &domain.Track{ID: mustID(t), GoalID: g.ID, UserID: userID, Version: 1, GeneratedBy: "user"}
	if err := InsertTrack(context.Background(), q, track); err != nil {
		t.Fatalf("InsertTrack: %v", err)
	}

	m1 := domain.Milestone{
		ID: mustID(t), TrackID: track.ID, GoalID: g.ID, UserID: userID,
		Ordinal: 1, Title: "Primeiro marco", CompletionCriteria: "critério 1",
		Status: domain.MilestoneCurrent, CompetencyIDs: []string{c1.ID, c2.ID},
	}
	if err := InsertMilestone(context.Background(), q, &m1); err != nil {
		t.Fatalf("InsertMilestone: %v", err)
	}
	for _, cid := range m1.CompetencyIDs {
		if err := InsertMilestoneCompetency(context.Background(), q, m1.ID, cid); err != nil {
			t.Fatalf("InsertMilestoneCompetency: %v", err)
		}
	}

	m2 := domain.Milestone{
		ID: mustID(t), TrackID: track.ID, GoalID: g.ID, UserID: userID,
		Ordinal: 2, Title: "Segundo marco", CompletionCriteria: "critério 2",
		Status: domain.MilestoneLocked,
	}
	if err := InsertMilestone(context.Background(), q, &m2); err != nil {
		t.Fatalf("InsertMilestone (2): %v", err)
	}

	fetched, err := GetCurrentTrack(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("GetCurrentTrack: %v", err)
	}
	if fetched.Version != 1 || len(fetched.Milestones) != 2 {
		t.Fatalf("track recuperada não bate: %+v", fetched)
	}
	if len(fetched.Milestones[0].CompetencyIDs) != 2 {
		t.Fatalf("esperava 2 competências mapeadas no primeiro marco, got %+v", fetched.Milestones[0].CompetencyIDs)
	}
	if len(fetched.Milestones[1].CompetencyIDs) != 0 {
		t.Fatalf("segundo marco não deveria ter competências mapeadas, got %+v", fetched.Milestones[1].CompetencyIDs)
	}

	fetchedMilestone, err := GetMilestone(context.Background(), q, userID, m1.ID)
	if err != nil {
		t.Fatalf("GetMilestone: %v", err)
	}
	if fetchedMilestone.Title != "Primeiro marco" {
		t.Fatalf("título = %q, want Primeiro marco", fetchedMilestone.Title)
	}

	if err := fetchedMilestone.Complete(time.Now()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if err := UpdateMilestone(context.Background(), q, fetchedMilestone); err != nil {
		t.Fatalf("UpdateMilestone: %v", err)
	}
	refetched, err := GetMilestone(context.Background(), q, userID, m1.ID)
	if err != nil {
		t.Fatalf("GetMilestone (2): %v", err)
	}
	if refetched.Status != domain.MilestoneCompleted || refetched.CompletedAt == nil {
		t.Fatalf("marco não persistiu como completed: %+v", refetched)
	}
}

func TestGetCurrentTrackNotFound(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta sem trilha")
	if _, err := GetCurrentTrack(context.Background(), q, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}

func TestGetMilestoneNotFound(t *testing.T) {
	q, userID := setup(t)
	if _, err := GetMilestone(context.Background(), q, userID, mustID(t)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}
