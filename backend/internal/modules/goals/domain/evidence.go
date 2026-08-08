package domain

import (
	"fmt"
	"time"
)

type EvidenceKind string

const (
	EvidenceCodeSnippet      EvidenceKind = "code_snippet"
	EvidenceRepoLink         EvidenceKind = "repo_link"
	EvidenceCommitDiff       EvidenceKind = "commit_diff"
	EvidenceExerciseSolution EvidenceKind = "exercise_solution"
	EvidenceWrittenText      EvidenceKind = "written_text"
	EvidenceAudioRecording   EvidenceKind = "audio_recording"
	EvidenceListeningLog     EvidenceKind = "listening_log"
	EvidenceShadowingClip    EvidenceKind = "shadowing_clip"
	EvidenceFile             EvidenceKind = "file"
)

// ValidEvidenceKinds espelha o enum EvidenceKind de backend/api/openapi.yaml.
// É a autoridade que o carregador de packs usa para validar evidence_types
// no boot (04-agentes.md §3.3, §4.6) — packs importa domain, nunca o
// contrário, então o contrato continua tendo uma fonte única de verdade.
var ValidEvidenceKinds = map[EvidenceKind]bool{
	EvidenceCodeSnippet:      true,
	EvidenceRepoLink:         true,
	EvidenceCommitDiff:       true,
	EvidenceExerciseSolution: true,
	EvidenceWrittenText:      true,
	EvidenceAudioRecording:   true,
	EvidenceListeningLog:     true,
	EvidenceShadowingClip:    true,
	EvidenceFile:             true,
}

func (k EvidenceKind) Valid() bool { return ValidEvidenceKinds[k] }

type Evidence struct {
	ID       string
	GoalID   string
	UserID   string
	ActionID *string

	Kind        EvidenceKind
	Title       *string
	Body        *string
	BlobKey     *string
	ExternalURL *string
	Language    *string

	SupersedesID *string
	LocalOn      time.Time
	CreatedAt    time.Time
}

// NewEvidence garante evidence_has_content (02-modelo-de-dados.md §6.3):
// precisa de body, blobKey ou externalUrl — evidência sem conteúdo não é
// evidência de nada. Imutabilidade (RN-06) é garantida no banco (RULE ... DO
// INSTEAD NOTHING), não aqui: este tipo nunca ganha um método Update.
func NewEvidence(id, goalID, userID string, actionID *string, kind EvidenceKind, title, body, blobKey, externalURL, language, supersedesID *string, loc *time.Location, now time.Time) (*Evidence, error) {
	if !kind.Valid() {
		return nil, newRuleError("", fmt.Sprintf("tipo de evidência inválido: %s", kind))
	}
	body = trimToNil(body)
	blobKey = trimToNil(blobKey)
	externalURL = trimToNil(externalURL)
	if body == nil && blobKey == nil && externalURL == nil {
		return nil, newRuleError("", "evidência precisa de body, blobKey ou externalUrl")
	}
	return &Evidence{
		ID:           id,
		GoalID:       goalID,
		UserID:       userID,
		ActionID:     actionID,
		Kind:         kind,
		Title:        trimToNil(title),
		Body:         body,
		BlobKey:      blobKey,
		ExternalURL:  externalURL,
		Language:     trimToNil(language),
		SupersedesID: supersedesID,
		LocalOn:      DateOnly(now.In(loc)),
		CreatedAt:    now,
	}, nil
}
