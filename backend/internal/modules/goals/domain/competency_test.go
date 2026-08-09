package domain

import (
	"testing"
	"time"
)

func TestApplyLevelEventFreezesBaselineOnFirstEvent(t *testing.T) {
	c := &Competency{ID: "c1"}
	now := time.Now()

	ev, err := NewLevelEvent("e1", "c1", "u1", nil, 2, ConfidenceLow, SourceProbe, nil, "sondagem inicial", now)
	if err != nil {
		t.Fatalf("NewLevelEvent falhou: %v", err)
	}
	if err := c.ApplyLevelEvent(*ev); err != nil {
		t.Fatalf("ApplyLevelEvent falhou: %v", err)
	}
	if c.CurrentLevel == nil || *c.CurrentLevel != 2 {
		t.Fatalf("CurrentLevel = %v, want 2", c.CurrentLevel)
	}
	if c.BaselineLevel == nil || *c.BaselineLevel != 2 {
		t.Fatalf("BaselineLevel = %v, want 2 (congelado no primeiro evento)", c.BaselineLevel)
	}

	from := 2
	evidenceID := "ev-1"
	ev2, err := NewLevelEvent("e2", "c1", "u1", &from, 4, ConfidenceMedium, SourceSelf, &evidenceID, "evidência nova", now)
	if err != nil {
		t.Fatalf("NewLevelEvent falhou: %v", err)
	}
	if ev2.EvidenceID == nil || *ev2.EvidenceID != evidenceID {
		t.Fatalf("EvidenceID = %v, want %q", ev2.EvidenceID, evidenceID)
	}
	if err := c.ApplyLevelEvent(*ev2); err != nil {
		t.Fatalf("ApplyLevelEvent falhou: %v", err)
	}
	if *c.CurrentLevel != 4 {
		t.Fatalf("CurrentLevel = %d, want 4", *c.CurrentLevel)
	}
	// baseline não deve mudar no segundo evento — é o "de" fixo do delta.
	if *c.BaselineLevel != 2 {
		t.Fatalf("BaselineLevel mudou para %d, deveria continuar 2", *c.BaselineLevel)
	}
}

func TestCompetencyDeltaNilWhenUnknown(t *testing.T) {
	c := &Competency{}
	if d := c.Delta(); d != nil {
		t.Fatalf("Delta() = %v, want nil quando nível é desconhecido (nunca 0)", d)
	}

	baseline, current := 1, 4
	c.BaselineLevel = &baseline
	c.CurrentLevel = &current
	d := c.Delta()
	if d == nil || *d != 3 {
		t.Fatalf("Delta() = %v, want 3", d)
	}
}

func TestNewLevelEventValidation(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name       string
		toLevel    int
		confidence Confidence
		source     LevelSource
		rationale  string
		wantErr    bool
	}{
		{"válido", 3, ConfidenceMedium, SourceSelf, "evidência clara", false},
		{"nível fora de faixa", 9, ConfidenceMedium, SourceSelf, "x", true},
		{"confiança unknown não é permitida em evento", 3, ConfidenceUnknown, SourceSelf, "x", true},
		{"origem inválida", 3, ConfidenceMedium, LevelSource("bogus"), "x", true},
		{"rationale vazio", 3, ConfidenceMedium, SourceSelf, "  ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewLevelEvent("id", "c1", "u1", nil, tc.toLevel, tc.confidence, tc.source, nil, tc.rationale, now)
			if (err != nil) != tc.wantErr {
				t.Fatalf("erro = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestLevelAtTime(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []LevelEvent{
		{ToLevel: 2, OccurredAt: base},
		{ToLevel: 4, OccurredAt: base.AddDate(0, 0, 10)},
		{ToLevel: 5, OccurredAt: base.AddDate(0, 0, 20)},
	}

	if lvl := LevelAtTime(events, base.AddDate(0, 0, -1)); lvl != nil {
		t.Fatalf("LevelAtTime antes do primeiro evento = %v, want nil (não medido, nunca 0)", lvl)
	}
	if lvl := LevelAtTime(events, base); lvl == nil || *lvl != 2 {
		t.Fatalf("LevelAtTime no instante do primeiro evento = %v, want 2", lvl)
	}
	if lvl := LevelAtTime(events, base.AddDate(0, 0, 15)); lvl == nil || *lvl != 4 {
		t.Fatalf("LevelAtTime entre o 2º e o 3º evento = %v, want 4", lvl)
	}
	if lvl := LevelAtTime(events, base.AddDate(0, 0, 100)); lvl == nil || *lvl != 5 {
		t.Fatalf("LevelAtTime depois do último evento = %v, want 5", lvl)
	}
	if lvl := LevelAtTime(nil, base); lvl != nil {
		t.Fatalf("LevelAtTime sem eventos = %v, want nil", lvl)
	}
}

func TestStaleDays(t *testing.T) {
	c := &Competency{}
	if d := c.StaleDays(time.Now()); d != nil {
		t.Fatalf("StaleDays sem evidência deveria ser nil, got %v", d)
	}

	last := time.Now().AddDate(0, 0, -46)
	c.LastEvidenceAt = &last
	d := c.StaleDays(time.Now())
	if d == nil || *d < 45 {
		t.Fatalf("StaleDays = %v, want >= 45 (limiar RN-05)", d)
	}
}
