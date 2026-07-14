package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestCallGraphFindsCrossFileRecursion(t *testing.T) {
	dir := t.TempDir()
	include := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(include, []byte("stock Second() { First(); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nmain() { First(); }\nstock First() { Second(); }\n")
	model, err := project.Build([]project.Source{{Path: mainPath, Content: source}}, project.Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if model.CallGraph == nil || len(model.CallGraph.Calls) != 3 {
		t.Fatalf("call graph = %#v", model.CallGraph)
	}
	components := model.CallGraph.RecursiveComponents()
	if len(components) != 1 || len(components[0]) != 2 {
		t.Fatalf("recursive components = %#v", components)
	}
	if components[0][0].Name != "First" || components[0][1].Name != "Second" {
		t.Fatalf("component = %#v", components[0])
	}
}

func TestCallGraphFindsDirectRecursion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte("forward Recur();\nmain() { Recur(); }\nstock Recur() { Recur(); }\n")
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	components := model.CallGraph.RecursiveComponents()
	if len(components) != 1 || len(components[0]) != 1 || components[0][0].Name != "Recur" {
		t.Fatalf("recursive components = %#v", components)
	}
}
