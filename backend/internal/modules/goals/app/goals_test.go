package app

import (
	"context"
	"errors"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

// createDraftGoal cria uma meta de Go em rascunho, com as 6 competências do
// pack já materializadas (CreateGoal), pronta para receber DoD e ativar.
func createDraftGoal(t *testing.T, svc *Service, userID, title string) *CreateGoalResult {
	t.Helper()
	res, err := svc.CreateGoal(context.Background(), CreateGoalInput{
		UserID:    userID,
		Title:     title,
		Archetype: domain.ArchetypeSkill,
		PackID:    "golang",
	})
	if err != nil {
		t.Fatalf("CreateGoal falhou: %v", err)
	}
	return res
}

// readyGoal cria e ativa uma meta (RN-01 satisfeita: DoD setado, 5
// competências do pack golang). Devolve a meta ativa e a primeira ação
// gerada pelo fallback.
func readyGoal(t *testing.T, svc *Service, userID, title string) *ActivateGoalResult {
	t.Helper()
	created := createDraftGoal(t, svc, userID, title)
	dod := "Escrever um servidor HTTP concorrente com testes, sem tutorial"
	if _, err := svc.PatchGoal(context.Background(), userID, created.Goal.ID, PatchGoalInput{DefinitionOfDone: &dod}); err != nil {
		t.Fatalf("PatchGoal (DoD) falhou: %v", err)
	}
	result, err := svc.ActivateGoal(context.Background(), ActivateGoalInput{UserID: userID, GoalID: created.Goal.ID})
	if err != nil {
		t.Fatalf("ActivateGoal falhou: %v", err)
	}
	return result
}

func TestCreateGoalMaterializesCompetenciesAndProbe(t *testing.T) {
	svc, userID := newFixture(t)

	res := createDraftGoal(t, svc, userID, "Aprender Go de verdade")

	if res.Goal.Status != domain.GoalDraft {
		t.Fatalf("status = %s, want draft", res.Goal.Status)
	}
	if len(res.Competencies) != 6 {
		t.Fatalf("esperava 6 competências do pack golang, got %d", len(res.Competencies))
	}
	for _, c := range res.Competencies {
		if c.CurrentLevel != nil {
			t.Fatalf("competência %s nasceu com nível %v, want nil (desconhecido)", c.PackKey, *c.CurrentLevel)
		}
	}
	if res.Probe == nil || len(res.Probe.Turns) != 1 {
		t.Fatalf("esperava sondagem aberta com a primeira pergunta, got %+v", res.Probe)
	}

	// persistiu de verdade — não só em memória.
	detail, err := svc.GetGoal(context.Background(), userID, res.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}
	if detail.Goal.Title != "Aprender Go de verdade" || len(detail.Competencies) != 6 {
		t.Fatalf("goal persistida não bate: %+v", detail)
	}
}

func TestCreateGoalUnknownPack(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.CreateGoal(context.Background(), CreateGoalInput{
		UserID: userID, Title: "Meta qualquer", Archetype: domain.ArchetypeSkill, PackID: "esperanto",
	})
	if err == nil {
		t.Fatal("esperava erro para pack desconhecido")
	}
	var ruleErr *domain.RuleError
	if !errors.As(err, &ruleErr) {
		t.Fatalf("esperava RuleError, got %T: %v", err, err)
	}
}

func TestGetGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.GetGoal(context.Background(), userID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}

func TestGetGoalWrongUserIsNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	other := pgtest.NewUser(t)
	created := createDraftGoal(t, svc, userID, "Meta do dono")

	if _, err := svc.GetGoal(context.Background(), other, created.Goal.ID); !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("meta de outro usuário deveria ser invisível (ErrNotFound), got %v", err)
	}
}

func TestListGoalsFiltersByStatus(t *testing.T) {
	svc, userID := newFixture(t)
	createDraftGoal(t, svc, userID, "Rascunho 1")
	readyGoal(t, svc, userID, "Ativa 1")

	all, err := svc.ListGoals(context.Background(), userID, nil)
	if err != nil {
		t.Fatalf("ListGoals(nil) falhou: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("esperava 2 metas no total, got %d", len(all))
	}

	active, err := svc.ListGoals(context.Background(), userID, []string{"active"})
	if err != nil {
		t.Fatalf("ListGoals(active) falhou: %v", err)
	}
	if len(active) != 1 || active[0].Status != domain.GoalActive {
		t.Fatalf("esperava 1 meta ativa, got %+v", active)
	}
}

func TestPatchGoalValidatesTitle(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta original")

	shortTitle := "Go"
	_, err := svc.PatchGoal(context.Background(), userID, created.Goal.ID, PatchGoalInput{Title: &shortTitle})
	if err == nil {
		t.Fatal("esperava erro para título curto demais")
	}

	newTitle := "Meta com título válido"
	updated, err := svc.PatchGoal(context.Background(), userID, created.Goal.ID, PatchGoalInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("PatchGoal válido falhou: %v", err)
	}
	if updated.Title != newTitle {
		t.Fatalf("título = %q, want %q", updated.Title, newTitle)
	}
}

func TestActivateGoalRequiresRN01(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Sem definição de pronto")

	_, err := svc.ActivateGoal(context.Background(), ActivateGoalInput{UserID: userID, GoalID: created.Goal.ID})
	if err == nil {
		t.Fatal("esperava falha de RN-01 (sem definição de pronto)")
	}
	var ruleErr *domain.RuleError
	if !errors.As(err, &ruleErr) || ruleErr.Rule != "RN-01" {
		t.Fatalf("esperava RuleError RN-01, got %v", err)
	}
}

// TestActivateGoalGeneratesFallbackTrackAndAction cobre RN-03: ativar uma
// meta nunca a deixa sem próxima ação. Também documenta o comportamento
// atual do fallback (fatia-1-implementacao.md §3/§4): sempre mira o primeiro
// marco (typical_level mais baixo) e a primeira competência mapeada nele.
func TestActivateGoalGeneratesFallbackTrackAndAction(t *testing.T) {
	svc, userID := newFixture(t)
	result := readyGoal(t, svc, userID, "Meta com trilha fallback")

	if result.Goal.Status != domain.GoalActive {
		t.Fatalf("status = %s, want active", result.Goal.Status)
	}
	if result.Action == nil {
		t.Fatal("RN-03: ActivateGoal deveria devolver uma próxima ação")
	}
	if result.Action.GeneratedBy != domain.GeneratedByFallback {
		t.Fatalf("generatedBy = %s, want fallback", result.Action.GeneratedBy)
	}
	if result.Action.PracticeFormat == nil || *result.Action.PracticeFormat != "kata" {
		t.Fatalf("practiceFormat = %v, want kata (primeiro formato compatível com idioms)", result.Action.PracticeFormat)
	}

	track, err := svc.GetTrack(context.Background(), userID, result.Goal.ID)
	if err != nil {
		t.Fatalf("GetTrack falhou: %v", err)
	}
	if len(track.Milestones) != 7 {
		t.Fatalf("esperava 7 marcos (milestone_library do pack golang), got %d", len(track.Milestones))
	}
	if track.Milestones[0].Status != domain.MilestoneCurrent {
		t.Fatalf("primeiro marco deveria ser current, got %s", track.Milestones[0].Status)
	}
	for _, m := range track.Milestones[1:] {
		if m.Status != domain.MilestoneLocked {
			t.Fatalf("marco %q deveria ser locked, got %s", m.Title, m.Status)
		}
	}

	pending, err := svc.GetPendingAction(context.Background(), userID, result.Goal.ID)
	if err != nil {
		t.Fatalf("GetPendingAction falhou: %v", err)
	}
	if pending.ID != result.Action.ID {
		t.Fatalf("ação pendente persistida difere da devolvida por ActivateGoal")
	}
}

func TestActivateGoalRN02MaxThreeActive(t *testing.T) {
	svc, userID := newFixture(t)
	readyGoal(t, svc, userID, "Ativa 1")
	readyGoal(t, svc, userID, "Ativa 2")
	readyGoal(t, svc, userID, "Ativa 3")

	fourth := createDraftGoal(t, svc, userID, "Quarta meta")
	dod := "Critério observável qualquer"
	if _, err := svc.PatchGoal(context.Background(), userID, fourth.Goal.ID, PatchGoalInput{DefinitionOfDone: &dod}); err != nil {
		t.Fatalf("PatchGoal falhou: %v", err)
	}

	_, err := svc.ActivateGoal(context.Background(), ActivateGoalInput{UserID: userID, GoalID: fourth.Goal.ID})
	if err == nil {
		t.Fatal("esperava RN-02 (máx 3 metas ativas)")
	}
	var maxErr *MaxActiveGoalsError
	if !errors.As(err, &maxErr) {
		t.Fatalf("esperava MaxActiveGoalsError, got %T: %v", err, err)
	}
	if len(maxErr.ActiveGoals) != 3 {
		t.Fatalf("esperava 3 metas ativas na lista do erro, got %d", len(maxErr.ActiveGoals))
	}
}

// TestActivateGoalWithPauseGoalIDFreesSlot cobre a saída de RN-02: pausar
// outra meta na mesma transação libera o slot para a nova ativação.
func TestActivateGoalWithPauseGoalIDFreesSlot(t *testing.T) {
	svc, userID := newFixture(t)
	first := readyGoal(t, svc, userID, "Ativa 1")
	readyGoal(t, svc, userID, "Ativa 2")
	readyGoal(t, svc, userID, "Ativa 3")

	fourth := createDraftGoal(t, svc, userID, "Quarta meta, via pausa")
	dod := "Critério observável qualquer"
	if _, err := svc.PatchGoal(context.Background(), userID, fourth.Goal.ID, PatchGoalInput{DefinitionOfDone: &dod}); err != nil {
		t.Fatalf("PatchGoal falhou: %v", err)
	}

	pauseID := first.Goal.ID
	result, err := svc.ActivateGoal(context.Background(), ActivateGoalInput{
		UserID: userID, GoalID: fourth.Goal.ID, PauseGoalID: &pauseID,
	})
	if err != nil {
		t.Fatalf("ActivateGoal com pauseGoalId deveria funcionar: %v", err)
	}
	if result.Goal.Status != domain.GoalActive {
		t.Fatalf("quarta meta deveria estar ativa, status = %s", result.Goal.Status)
	}

	pausedDetail, err := svc.GetGoal(context.Background(), userID, first.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal da meta pausada falhou: %v", err)
	}
	if pausedDetail.Goal.Status != domain.GoalPaused {
		t.Fatalf("primeira meta deveria estar paused, status = %s", pausedDetail.Goal.Status)
	}

	active, err := svc.ListGoals(context.Background(), userID, []string{"active", "at_risk", "stagnant"})
	if err != nil {
		t.Fatalf("ListGoals falhou: %v", err)
	}
	if len(active) != 3 {
		t.Fatalf("esperava voltar a 3 metas ativas após pausar uma e ativar outra, got %d", len(active))
	}
}

func TestActivateGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.ActivateGoal(context.Background(), ActivateGoalInput{
		UserID: userID, GoalID: "00000000-0000-0000-0000-000000000000",
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

func TestCreateGoalInvalidTitle(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.CreateGoal(context.Background(), CreateGoalInput{
		UserID: userID, Title: "Go", Archetype: domain.ArchetypeSkill, PackID: "golang",
	})
	if err == nil {
		t.Fatal("esperava erro para título curto demais (domain.NewGoal)")
	}
	var ruleErr *domain.RuleError
	if !errors.As(err, &ruleErr) {
		t.Fatalf("esperava RuleError, got %T: %v", err, err)
	}
}

func TestActivateGoalPauseGoalIDNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta a ativar")
	dod := "Critério observável qualquer"
	if _, err := svc.PatchGoal(context.Background(), userID, created.Goal.ID, PatchGoalInput{DefinitionOfDone: &dod}); err != nil {
		t.Fatalf("PatchGoal falhou: %v", err)
	}

	bogus := "00000000-0000-0000-0000-000000000000"
	_, err := svc.ActivateGoal(context.Background(), ActivateGoalInput{UserID: userID, GoalID: created.Goal.ID, PauseGoalID: &bogus})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound ao pausar meta inexistente, got %v", err)
	}
}
