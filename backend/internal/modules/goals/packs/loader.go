package packs

import (
	"embed"
	"fmt"
	"math"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

//go:embed golang.yaml english.yaml
var files embed.FS

// Registry é o registro em memória populado no boot (04-agentes.md §4.6):
// map[packID]Pack, montado uma vez a partir dos YAML embutidos no binário.
// Embutido — não lido do disco — porque um pack ausente ou corrompido
// derrubaria todos os agentes daquele domínio, e embutido não existe a
// classe de erro "funciona local, falha no container".
type Registry struct {
	byID map[string]Pack
}

// Load lê e valida todos os packs embutidos. Falha ruidosa: se um pack não
// passar na validação, o erro sobe e quem chama (cmd/api) deve encerrar o
// processo — subir com um pack inválido seria pior que não subir, porque o
// problema só apareceria semanas depois, com dados gravados em cima de uma
// rubrica quebrada.
func Load() (*Registry, error) {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("listar packs embutidos: %w", err)
	}

	reg := &Registry{byID: make(map[string]Pack)}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := files.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("ler %s: %w", entry.Name(), err)
		}
		var p Pack
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parsear %s: %w", entry.Name(), err)
		}
		if err := validate(p); err != nil {
			return nil, fmt.Errorf("pack %s (%s) inválido: %w", p.ID, entry.Name(), err)
		}
		if _, dup := reg.byID[p.ID]; dup {
			return nil, fmt.Errorf("pack id duplicado: %s", p.ID)
		}
		reg.byID[p.ID] = p
	}
	return reg, nil
}

func (r *Registry) Get(id string) (Pack, bool) {
	p, ok := r.byID[id]
	return p, ok
}

func (r *Registry) IDs() []string {
	ids := make([]string, 0, len(r.byID))
	for id := range r.byID {
		ids = append(ids, id)
	}
	return ids
}

// validate roda as checagens de 04-agentes.md §3.3 / §4.6.
func validate(p Pack) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("id vazio")
	}
	if len(p.Competencies) == 0 {
		return fmt.Errorf("nenhuma competência declarada")
	}

	seenKeys := make(map[string]bool, len(p.Competencies))
	var weightSum float64
	for _, c := range p.Competencies {
		if c.Key == "" {
			return fmt.Errorf("competência sem key")
		}
		if seenKeys[c.Key] {
			return fmt.Errorf("chave de competência duplicada: %s", c.Key)
		}
		seenKeys[c.Key] = true
		weightSum += c.Weight

		for level := 0; level <= 5; level++ {
			if strings.TrimSpace(c.Levels[level]) == "" {
				return fmt.Errorf("competência %s: falta o nível %d", c.Key, level)
			}
		}
	}
	// pesos somam exatamente 1.0 (com folga para ponto flutuante)
	if math.Abs(weightSum-1.0) > 0.01 {
		return fmt.Errorf("pesos das competências somam %.4f, esperado 1.0", weightSum)
	}

	for _, f := range p.PracticeFormats {
		if f.Minutes[0] < 5 || f.Minutes[1] > 30 || f.Minutes[0] > f.Minutes[1] {
			return fmt.Errorf("formato de prática %s: janela de minutos [%d,%d] fora de 5-30 (P4)", f.Key, f.Minutes[0], f.Minutes[1])
		}
		for _, key := range f.GoodFor {
			if !seenKeys[key] {
				return fmt.Errorf("formato de prática %s: good_for aponta para competência inexistente: %s", f.Key, key)
			}
		}
	}

	for _, m := range p.MilestoneLibrary {
		for _, key := range m.Competencies {
			if !seenKeys[key] {
				return fmt.Errorf("marco %q: competência inexistente: %s", m.Title, key)
			}
		}
	}

	for _, et := range p.EvidenceTypes {
		if !domain.ValidEvidenceKinds[domain.EvidenceKind(et)] {
			return fmt.Errorf("evidence_type %q não existe no enum EvidenceKind da API", et)
		}
	}

	for _, seed := range p.ProbeSeeds {
		if seed.Competency != "" && !seenKeys[seed.Competency] {
			return fmt.Errorf("probe_seed aponta para competência inexistente: %s", seed.Competency)
		}
	}

	return nil
}
