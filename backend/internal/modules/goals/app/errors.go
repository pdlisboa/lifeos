package app

import "github.com/phablo/lifeos/internal/modules/goals/domain"

// MaxActiveGoalsError é RN-02. Carrega as metas ativas para a UI perguntar
// "qual você quer pausar?" em vez de só reclamar (03-api.md §5).
type MaxActiveGoalsError struct {
	ActiveGoals []domain.Goal
}

func (e *MaxActiveGoalsError) Error() string {
	return "RN-02: máximo de 3 metas ativas"
}
