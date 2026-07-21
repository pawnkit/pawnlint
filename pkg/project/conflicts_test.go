package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestConflictingIncludeSymbols(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.inc")
	second := filepath.Join(dir, "second.inc")
	if err := os.WriteFile(first, []byte("enum { Shared }\nstock Mixed() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("new Shared;\nnew Mixed;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"first.inc\"\n#include \"second.inc\"\nmain() {}\n")
	model, err := project.Build([]project.Source{{Path: root, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	conflicts := model.ConflictingIncludeSymbols()
	if len(conflicts) != 2 {
		t.Fatalf("conflicts = %+v", conflicts)
	}
	if conflicts[0].Owner.Path != root && conflicts[1].Owner.Path != root {
		t.Fatalf("conflict owners = %+v", conflicts)
	}
}

func TestConflictingIncludeSymbolsExcludeExistingDuplicateRules(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.inc")
	second := filepath.Join(dir, "second.inc")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("stock SameFunction() {}\nnew same_global;\nenum SameEnum { SameEntry }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"first.inc\"\n#include \"second.inc\"\nmain() {}\n")
	model, err := project.Build([]project.Source{{Path: root, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if conflicts := model.ConflictingIncludeSymbols(); len(conflicts) != 0 {
		t.Fatalf("conflicts = %+v", conflicts)
	}
	if len(model.DuplicateFunctions()) != 1 || len(model.DuplicateGlobals()) != 1 {
		t.Fatal("existing duplicate rules did not retain ownership")
	}
}

func TestDuplicateDefinitionsReportedOnceAcrossEntryPoints(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.inc")
	second := filepath.Join(dir, "second.inc")
	if err := os.WriteFile(first, []byte("stock Shared() {}\nnew shared_global;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("stock Shared() {}\nnew shared_global;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := []byte("#include \"first.inc\"\n#include \"second.inc\"\n")
	model, err := project.Build([]project.Source{
		{Path: filepath.Join(dir, "main.pwn"), Content: source},
		{Path: filepath.Join(dir, "other.pwn"), Content: source},
	}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(model.DuplicateFunctions()); got != 1 {
		t.Fatalf("duplicate functions = %d", got)
	}
	if got := len(model.DuplicateGlobals()); got != 1 {
		t.Fatalf("duplicate globals = %d", got)
	}
}

func TestConflictingIncludeSymbolsExposeKinds(t *testing.T) {
	dir := t.TempDir()
	include := filepath.Join(dir, "values.inc")
	if err := os.WriteFile(include, []byte("enum { Value }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"values.inc\"\nnew Value;\nmain() {}\n")
	model, err := project.Build([]project.Source{{Path: root, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	conflicts := model.ConflictingIncludeSymbols()
	if len(conflicts) != 1 || conflicts[0].First.Kind == conflicts[0].Second.Kind || conflicts[0].First.Kind != semantic.SymbolGlobal && conflicts[0].Second.Kind != semantic.SymbolGlobal {
		t.Fatalf("conflicts = %+v", conflicts)
	}
}
