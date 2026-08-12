package agents

import (
	"strings"
	"testing"
)

const practiceSchema = `{
	"type": "object",
	"required": ["title", "estimatedMin", "minimalVariant", "expectedEvidence"],
	"additionalProperties": false,
	"properties": {
		"title": {"type": "string", "minLength": 5},
		"estimatedMin": {"type": "integer", "minimum": 5, "maximum": 30},
		"minimalVariant": {"type": "string", "minLength": 15},
		"difficultyHint": {"type": "string", "enum": ["easier", "same", "harder"]},
		"expectedEvidence": {
			"type": "object",
			"required": ["kind", "description"],
			"additionalProperties": false,
			"properties": {
				"kind": {"type": "string"},
				"description": {"type": "string", "minLength": 10}
			}
		},
		"tags": {
			"type": "array",
			"minItems": 1,
			"maxItems": 3,
			"items": {"type": "string"}
		}
	}
}`

func TestValidate_AcceptsWellFormedOutput(t *testing.T) {
	data := `{
		"title": "Worker pool com channels",
		"estimatedMin": 20,
		"minimalVariant": "Escreva só a assinatura da função e os canais",
		"difficultyHint": "same",
		"expectedEvidence": {"kind": "code_snippet", "description": "cole o worker pool"},
		"tags": ["go", "concurrency"]
	}`
	if errs := validate([]byte(practiceSchema), []byte(data)); len(errs) != 0 {
		t.Fatalf("esperava saída válida, teve erros: %v", errs)
	}
}

func TestValidate_RejectsMissingRequiredField(t *testing.T) {
	data := `{
		"title": "Worker pool com channels",
		"estimatedMin": 20,
		"expectedEvidence": {"kind": "code_snippet", "description": "cole o worker pool"}
	}`
	errs := validate([]byte(practiceSchema), []byte(data))
	if !containsSubstr(errs, "minimalVariant") {
		t.Fatalf("esperava erro citando minimalVariant ausente, teve: %v", errs)
	}
}

func TestValidate_RejectsStringBelowMinLength(t *testing.T) {
	data := `{
		"title": "Go",
		"estimatedMin": 20,
		"minimalVariant": "Escreva só a assinatura da função e os canais",
		"expectedEvidence": {"kind": "code_snippet", "description": "cole o worker pool"}
	}`
	errs := validate([]byte(practiceSchema), []byte(data))
	if !containsSubstr(errs, "title") {
		t.Fatalf("esperava erro de minLength em title, teve: %v", errs)
	}
}

func TestValidate_RejectsNumberAboveMaximum(t *testing.T) {
	data := `{
		"title": "Worker pool com channels",
		"estimatedMin": 120,
		"minimalVariant": "Escreva só a assinatura da função e os canais",
		"expectedEvidence": {"kind": "code_snippet", "description": "cole o worker pool"}
	}`
	errs := validate([]byte(practiceSchema), []byte(data))
	if !containsSubstr(errs, "estimatedMin") {
		t.Fatalf("esperava erro de maximum em estimatedMin (P4: ação pequena), teve: %v", errs)
	}
}

func TestValidate_RejectsValueOutsideEnum(t *testing.T) {
	data := `{
		"title": "Worker pool com channels",
		"estimatedMin": 20,
		"minimalVariant": "Escreva só a assinatura da função e os canais",
		"difficultyHint": "impossible",
		"expectedEvidence": {"kind": "code_snippet", "description": "cole o worker pool"}
	}`
	errs := validate([]byte(practiceSchema), []byte(data))
	if !containsSubstr(errs, "difficultyHint") {
		t.Fatalf("esperava erro de enum em difficultyHint, teve: %v", errs)
	}
}

func TestValidate_RejectsAdditionalProperty(t *testing.T) {
	data := `{
		"title": "Worker pool com channels",
		"estimatedMin": 20,
		"minimalVariant": "Escreva só a assinatura da função e os canais",
		"expectedEvidence": {"kind": "code_snippet", "description": "cole o worker pool"},
		"surpresa": "campo que o schema não declara"
	}`
	errs := validate([]byte(practiceSchema), []byte(data))
	if !containsSubstr(errs, "surpresa") {
		t.Fatalf("esperava erro de propriedade não declarada, teve: %v", errs)
	}
}

func TestValidate_RejectsArrayBelowMinItems(t *testing.T) {
	data := `{
		"title": "Worker pool com channels",
		"estimatedMin": 20,
		"minimalVariant": "Escreva só a assinatura da função e os canais",
		"expectedEvidence": {"kind": "code_snippet", "description": "cole o worker pool"},
		"tags": []
	}`
	errs := validate([]byte(practiceSchema), []byte(data))
	if !containsSubstr(errs, "tags") {
		t.Fatalf("esperava erro de minItems em tags, teve: %v", errs)
	}
}

func TestValidate_RejectsNestedObjectViolations(t *testing.T) {
	data := `{
		"title": "Worker pool com channels",
		"estimatedMin": 20,
		"minimalVariant": "Escreva só a assinatura da função e os canais",
		"expectedEvidence": {"kind": "code_snippet", "description": "curta"}
	}`
	errs := validate([]byte(practiceSchema), []byte(data))
	if !containsSubstr(errs, "description") {
		t.Fatalf("esperava erro de minLength em expectedEvidence.description, teve: %v", errs)
	}
}

func TestValidate_RejectsInvalidJSON(t *testing.T) {
	errs := validate([]byte(practiceSchema), []byte(`{"title": "sem fechar aspas}`))
	if len(errs) == 0 {
		t.Fatal("esperava erro pra JSON malformado")
	}
}

const nullableSchema = `{
	"type": "object",
	"required": ["title"],
	"additionalProperties": false,
	"properties": {
		"title": {"type": "string", "minLength": 5},
		"milestoneId": {"type": ["string", "null"]},
		"rationale": {"type": ["string", "null"], "maxLength": 300}
	}
}`

func TestValidate_AcceptsNullForNullableType(t *testing.T) {
	data := `{"title": "Worker pool com channels", "milestoneId": null, "rationale": null}`
	if errs := validate([]byte(nullableSchema), []byte(data)); len(errs) != 0 {
		t.Fatalf("esperava saída válida com null em campo nullable, teve erros: %v", errs)
	}
}

func TestValidate_AcceptsStringForNullableType(t *testing.T) {
	data := `{"title": "Worker pool com channels", "milestoneId": "abc-123", "rationale": "porque sim"}`
	if errs := validate([]byte(nullableSchema), []byte(data)); len(errs) != 0 {
		t.Fatalf("esperava saída válida, teve erros: %v", errs)
	}
}

func TestValidate_RejectsNullForNonNullableType(t *testing.T) {
	data := `{"title": null}`
	errs := validate([]byte(nullableSchema), []byte(data))
	if !containsSubstr(errs, "title") {
		t.Fatalf("esperava erro citando title (não é nullable), teve: %v", errs)
	}
}

func containsSubstr(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
