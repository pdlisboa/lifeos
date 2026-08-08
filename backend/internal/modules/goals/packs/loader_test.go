package packs

import "testing"

func TestLoad(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatalf("Load() falhou: %v", err)
	}

	for _, id := range []string{"golang", "english"} {
		p, ok := reg.Get(id)
		if !ok {
			t.Fatalf("pack %q não carregado", id)
		}
		if len(p.Competencies) == 0 {
			t.Fatalf("pack %q sem competências", id)
		}
		if len(p.ProbeSeeds) == 0 {
			t.Fatalf("pack %q sem probe_seeds", id)
		}
	}

	if _, ok := reg.Get("nao-existe"); ok {
		t.Fatal("pack inexistente não deveria ser encontrado")
	}
}

func TestValidateRejectsWeightsNotSummingToOne(t *testing.T) {
	p := Pack{
		ID: "broken",
		Competencies: []Competency{
			{Key: "a", Weight: 0.5, Levels: fullLevels()},
			{Key: "b", Weight: 0.2, Levels: fullLevels()},
		},
	}
	if err := validate(p); err == nil {
		t.Fatal("esperava erro de pesos não somando 1.0")
	}
}

func TestValidateRejectsMissingLevel(t *testing.T) {
	levels := fullLevels()
	delete(levels, 3)
	p := Pack{
		ID: "broken",
		Competencies: []Competency{
			{Key: "a", Weight: 1.0, Levels: levels},
		},
	}
	if err := validate(p); err == nil {
		t.Fatal("esperava erro de nível faltando")
	}
}

func TestValidateRejectsBadEvidenceType(t *testing.T) {
	p := Pack{
		ID: "broken",
		Competencies: []Competency{
			{Key: "a", Weight: 1.0, Levels: fullLevels()},
		},
		EvidenceTypes: []string{"nao_existe_no_enum"},
	}
	if err := validate(p); err == nil {
		t.Fatal("esperava erro de evidence_type inválido")
	}
}

func fullLevels() map[int]string {
	return map[int]string{0: "a", 1: "b", 2: "c", 3: "d", 4: "e", 5: "f"}
}
