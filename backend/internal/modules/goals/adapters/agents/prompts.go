// Package agents é o lado do módulo Metas que conhece os 5 agentes:
// carrega os prompts versionados de prompts/*.md e os schemas de
// schemas/*.json (embutidos no binário, mesmo racional de packs — um
// arquivo ausente ou corrompido derruba o agente inteiro, e é melhor
// descobrir isso no boot do que em produção), monta o contexto de cada
// template e chama o Agent Gateway (01-arquitetura.md §3, §6).
package agents

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

//go:embed prompts/*.md
var promptFiles embed.FS

//go:embed schemas/*.json
var schemaFiles embed.FS

// promptSet é um prompt já pronto pra uso: o frontmatter (id/tier/schema) é
// só documentação do arquivo — cada agente já fixa Task/Tier/SchemaName no
// código Go que o chama (planner.go, practice.go), então não precisa ser
// parseado em runtime.
type promptSet struct {
	System   string
	UserTmpl *template.Template
}

func mustLoadPrompt(filename string) promptSet {
	p, err := loadPrompt(filename)
	if err != nil {
		panic(err)
	}
	return p
}

func mustLoadSchema(filename string) json.RawMessage {
	s, err := loadSchema(filename)
	if err != nil {
		panic(err)
	}
	return s
}

func loadPrompt(filename string) (promptSet, error) {
	raw, err := promptFiles.ReadFile("prompts/" + filename)
	if err != nil {
		return promptSet{}, fmt.Errorf("agents: ler prompt %s: %w", filename, err)
	}
	system, user, err := splitPromptSections(string(raw))
	if err != nil {
		return promptSet{}, fmt.Errorf("agents: %s: %w", filename, err)
	}
	tmpl, err := template.New(filename).Parse(user)
	if err != nil {
		return promptSet{}, fmt.Errorf("agents: parsear template de %s: %w", filename, err)
	}
	return promptSet{System: system, UserTmpl: tmpl}, nil
}

func loadSchema(filename string) (json.RawMessage, error) {
	raw, err := schemaFiles.ReadFile("schemas/" + filename)
	if err != nil {
		return nil, fmt.Errorf("agents: ler schema %s: %w", filename, err)
	}
	return json.RawMessage(raw), nil
}

// splitPromptSections separa um prompt em frontmatter / "# Sistema" /
// "# Usuário" pelas 3 linhas "---" que delimitam cada arquivo em
// prompts/*.md. text/template puro (não html/template) de propósito: os
// prompts carregam código e JSON de exemplo que não pode ser escapado.
func splitPromptSections(raw string) (system, user string, err error) {
	var segments []string
	var cur strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "---" {
			segments = append(segments, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteString(line)
		cur.WriteByte('\n')
	}
	segments = append(segments, cur.String())
	if len(segments) != 4 {
		return "", "", fmt.Errorf("esperava exatamente 3 delimitadores '---' (frontmatter/sistema/usuário), achei %d", len(segments)-1)
	}
	return strings.TrimSpace(segments[2]), strings.TrimSpace(segments[3]), nil
}
