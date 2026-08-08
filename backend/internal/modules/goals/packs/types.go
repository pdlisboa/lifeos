// Package packs carrega os Domain Packs (00-negocio.md §8): o conhecimento
// declarativo que diferencia uma meta de Go de uma de inglês sem nenhum
// `if goalType == "golang"` no núcleo do módulo (04-agentes.md §2).
package packs

// Competency é uma competência do pack, com os descritores 0-5 usados tanto
// pela avaliação (Fatia 4+) quanto pela UI (04-agentes.md §3.1: descritores
// precisam ser observáveis).
type Competency struct {
	Key             string         `yaml:"key"`
	Label           string         `yaml:"label"`
	Weight          float64        `yaml:"weight"`
	EvidenceSignals []string       `yaml:"evidence_signals"`
	Levels          map[int]string `yaml:"levels"`
}

// PracticeFormat é um dos moldes que o Gerador de Prática (A2) usa — e que o
// fallback sem agente da Fatia 1 usa diretamente (04-agentes.md §5).
type PracticeFormat struct {
	Key     string   `yaml:"key"`
	Label   string   `yaml:"label"`
	Minutes [2]int   `yaml:"minutes"`
	GoodFor []string `yaml:"good_for"`
	Shape   string   `yaml:"shape"`
	Example string   `yaml:"example"`
}

// MilestoneSpec é um marco da biblioteca de referência — ponto de partida
// do Planejador (A1) e, sem agente, a trilha inteira da Fatia 1.
type MilestoneSpec struct {
	Title        string   `yaml:"title"`
	Competencies []string `yaml:"competencies"`
	Criteria     string   `yaml:"criteria"`
	TypicalLevel int      `yaml:"typical_level"`
}

type Curation struct {
	PreferredDomains []string `yaml:"preferred_domains"`
	Avoid            []string `yaml:"avoid"`
	Seeds            []string `yaml:"seeds"`
}

// ProbeSeed é uma pergunta escrita à mão para a sondagem estática da Fatia 1
// (04-agentes.md §4.7) — o A1 substitui isto por um diálogo adaptativo só a
// partir da Fatia 3, sem mudar o contrato da API.
type ProbeSeed struct {
	Competency string            `yaml:"competency"`
	Question   string            `yaml:"question"`
	Followups  map[string]string `yaml:"followups"`
}

type Pack struct {
	ID        string `yaml:"id"`
	Version   int    `yaml:"version"`
	Label     string `yaml:"label"`
	Archetype string `yaml:"archetype"`
	Language  string `yaml:"language"`

	Competencies     []Competency     `yaml:"competencies"`
	EvidenceTypes    []string         `yaml:"evidence_types"`
	PracticeFormats  []PracticeFormat `yaml:"practice_formats"`
	MilestoneLibrary []MilestoneSpec  `yaml:"milestone_library"`
	Curation         Curation         `yaml:"curation"`
	ProgressSignals  []string         `yaml:"progress_signals"`
	ProbeSeeds       []ProbeSeed      `yaml:"probe_seeds"`
}

// CompetencyByKey facilita lookups repetidos em app/ sem varrer o slice.
func (p Pack) CompetencyByKey(key string) (Competency, bool) {
	for _, c := range p.Competencies {
		if c.Key == key {
			return c, true
		}
	}
	return Competency{}, false
}

// PracticeFormatFor devolve o primeiro formato de prática compatível com a
// competência dada — é o segundo passo do fallback de A2 (04-agentes.md §5).
func (p Pack) PracticeFormatFor(competencyKey string) (PracticeFormat, bool) {
	for _, f := range p.PracticeFormats {
		for _, key := range f.GoodFor {
			if key == competencyKey {
				return f, true
			}
		}
	}
	return PracticeFormat{}, false
}
