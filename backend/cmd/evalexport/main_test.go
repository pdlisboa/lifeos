package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Nível claro de goroutine com race condition": "nivel-claro-de-goroutine-com-race",
		"   ":           "caso",
		"É-só-um-caso!": "e-so-um-caso",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteCasesNumbersInOrderAndDisambiguatesSlugs(t *testing.T) {
	dir := t.TempDir()
	cases := []evalCaseFile{
		{ID: "ev1", Reason: "goroutine vazando"},
		{ID: "ev2", Reason: "goroutine vazando"},
	}
	if err := writeCases(dir, cases); err != nil {
		t.Fatalf("writeCases: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("esperava 2 arquivos, got %d", len(entries))
	}
	names := []string{entries[0].Name(), entries[1].Name()}
	if names[0] != "001-goroutine-vazando.json" || names[1] != "002-goroutine-vazando-2.json" {
		t.Fatalf("nomes = %v, want [001-goroutine-vazando.json 002-goroutine-vazando-2.json]", names)
	}

	var decoded evalCaseFile
	raw, err := os.ReadFile(filepath.Join(dir, names[0]))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.ID != "ev1" {
		t.Fatalf("decoded.ID = %q, want ev1", decoded.ID)
	}
}

// TestWriteCasesReplacesStaleFiles cobre a semântica de export: rodar de
// novo com um conjunto menor não deixa arquivo órfão de um caso desmarcado.
func TestWriteCasesReplacesStaleFiles(t *testing.T) {
	dir := t.TempDir()
	if err := writeCases(dir, []evalCaseFile{
		{ID: "ev1", Reason: "primeiro"},
		{ID: "ev2", Reason: "segundo"},
	}); err != nil {
		t.Fatalf("writeCases (primeira leva): %v", err)
	}

	if err := writeCases(dir, []evalCaseFile{
		{ID: "ev1", Reason: "primeiro"},
	}); err != nil {
		t.Fatalf("writeCases (segunda leva): %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("esperava 1 arquivo depois da segunda leva, got %d: %v", len(entries), entries)
	}
}

func TestWriteCasesEmptySetClearsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := writeCases(dir, []evalCaseFile{{ID: "ev1", Reason: "único"}}); err != nil {
		t.Fatalf("writeCases: %v", err)
	}
	if err := writeCases(dir, nil); err != nil {
		t.Fatalf("writeCases (vazio): %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("esperava diretório vazio depois de export sem casos, got %v", entries)
	}
}
