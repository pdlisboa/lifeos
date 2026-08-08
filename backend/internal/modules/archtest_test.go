// Teste de arquitetura (01-arquitetura.md §2.1, §11): um módulo de domínio
// nunca importa outro módulo de domínio — comunicação só por eventos.
// Módulos podem importar platform; platform nunca importa módulo de
// domínio. Se este teste quebrar, o problema é o import, não o teste.
package modules

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePrefix = "github.com/phablo/lifeos/internal/modules/"

func TestModulesDoNotImportEachOther(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ler %s: %v", root, err)
	}

	var mods []string
	for _, e := range entries {
		if e.IsDir() {
			mods = append(mods, e.Name())
		}
	}
	if len(mods) == 0 {
		t.Fatal("nenhum módulo de domínio encontrado — o teste não está olhando o diretório certo")
	}

	fset := token.NewFileSet()
	for _, mod := range mods {
		modDir := filepath.Join(root, mod)
		err := filepath.WalkDir(modDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if !strings.HasPrefix(importPath, modulePrefix) {
					continue
				}
				rest := strings.TrimPrefix(importPath, modulePrefix)
				otherMod := strings.SplitN(rest, "/", 2)[0]
				if otherMod != mod {
					rel, _ := filepath.Rel(root, path)
					t.Errorf("%s: módulo %q importa módulo %q (%s) — comunicação entre módulos de domínio só por eventos",
						rel, mod, otherMod, importPath)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestPlatformDoesNotImportModules garante o outro lado da regra: platform
// nunca importa módulo de domínio.
func TestPlatformDoesNotImportModules(t *testing.T) {
	modulesDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	platformDir := filepath.Join(filepath.Dir(modulesDir), "platform")
	if _, err := os.Stat(platformDir); os.IsNotExist(err) {
		t.Skip("internal/platform não existe")
	}

	fset := token.NewFileSet()
	err = filepath.WalkDir(platformDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(importPath, modulePrefix) {
				rel, _ := filepath.Rel(filepath.Dir(modulesDir), path)
				t.Errorf("%s: platform importa módulo de domínio (%s) — a dependência é sempre no outro sentido", rel, importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
